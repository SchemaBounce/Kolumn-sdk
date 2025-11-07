package sqlrunner

import (
	"bytes"
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"net"
	"text/template"
	"time"

	"github.com/schemabounce/kolumn/sdk/runtimehelpers/telemetry"
)

// Config describes how to create a Runner.
type Config struct {
	Driver          string
	DSN             string
	ExistingDB      *sql.DB
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	Retry           RetryPolicy
	Template        TemplateConfig
	Logger          telemetry.Logger
}

// RetryPolicy captures retry behavior for transient errors.
type RetryPolicy struct {
	Attempts    int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	ShouldRetry func(error) bool
}

// TemplateConfig configures SQL templating helpers.
type TemplateConfig struct {
	LeftDelim  string
	RightDelim string
	Funcs      template.FuncMap
}

// Runner wraps sql.DB with retry, telemetry, and templating conveniences.
type Runner struct {
	db      *sql.DB
	closeDB bool
	cfg     Config
	logger  telemetry.Logger
	policy  RetryPolicy
}

// NewRunner constructs a Runner using the provided configuration.
func NewRunner(cfg Config) (*Runner, error) {
	if cfg.ExistingDB == nil && (cfg.Driver == "" || cfg.DSN == "") {
		return nil, fmt.Errorf("sqlrunner: either ExistingDB or Driver+DSN must be provided")
	}

	var (
		db      *sql.DB
		closeDB bool
		err     error
	)

	if cfg.ExistingDB != nil {
		db = cfg.ExistingDB
	} else {
		db, err = sql.Open(cfg.Driver, cfg.DSN)
		if err != nil {
			return nil, fmt.Errorf("sqlrunner: open connection: %w", err)
		}
		closeDB = true
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	logger := cfg.Logger
	if logger == nil {
		logger = telemetry.NoopLogger{}
	}

	policy := cfg.Retry
	if policy.Attempts <= 0 {
		policy.Attempts = 3
	}
	if policy.BaseDelay <= 0 {
		policy.BaseDelay = 50 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 500 * time.Millisecond
	}
	if policy.ShouldRetry == nil {
		policy.ShouldRetry = defaultShouldRetry
	}

	return &Runner{
		db:      db,
		closeDB: closeDB,
		cfg:     cfg,
		logger:  logger,
		policy:  policy,
	}, nil
}

// DB exposes the underlying *sql.DB.
func (r *Runner) DB() *sql.DB {
	return r.db
}

// Close releases the underlying connection if the Runner created it.
func (r *Runner) Close() error {
	if r == nil || r.db == nil || !r.closeDB {
		return nil
	}
	return r.db.Close()
}

// Exec executes a statement with retry + telemetry.
func (r *Runner) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return runWithRetry(ctx, r, "exec", query, func(ctx context.Context) (sql.Result, error) {
		return r.db.ExecContext(ctx, query, args...)
	})
}

// ExecTemplate renders a template and executes the resulting statement.
func (r *Runner) ExecTemplate(ctx context.Context, queryTemplate string, data any, args ...any) (sql.Result, error) {
	rendered, err := r.renderTemplate("exec", queryTemplate, data)
	if err != nil {
		return nil, err
	}
	return r.Exec(ctx, rendered, args...)
}

// Query runs a query and returns the resulting rows with retry.
func (r *Runner) Query(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return runWithRetry(ctx, r, "query", query, func(ctx context.Context) (*sql.Rows, error) {
		return r.db.QueryContext(ctx, query, args...)
	})
}

// QueryTemplate renders a template and runs the query.
func (r *Runner) QueryTemplate(ctx context.Context, queryTemplate string, data any, args ...any) (*sql.Rows, error) {
	rendered, err := r.renderTemplate("query", queryTemplate, data)
	if err != nil {
		return nil, err
	}
	return r.Query(ctx, rendered, args...)
}

// QueryRow runs a query expected to return a single row.
func (r *Runner) QueryRow(ctx context.Context, query string, args ...any) *sql.Row {
	return r.db.QueryRowContext(ctx, query, args...)
}

// WithTransaction executes fn inside a transaction, retrying on transient errors.
func (r *Runner) WithTransaction(ctx context.Context, opts *sql.TxOptions, fn func(context.Context, *sql.Tx) error) error {
	_, err := runWithRetry(ctx, r, "tx", "", func(ctx context.Context) (struct{}, error) {
		tx, err := r.db.BeginTx(ctx, opts)
		if err != nil {
			return struct{}{}, err
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback()
			}
		}()

		if err := fn(ctx, tx); err != nil {
			return struct{}{}, err
		}
		if err := tx.Commit(); err != nil {
			return struct{}{}, err
		}
		committed = true
		return struct{}{}, nil
	})
	return err
}

func (r *Runner) renderTemplate(name, queryTemplate string, data any) (string, error) {
	tpl := template.New(name)
	if r.cfg.Template.LeftDelim != "" || r.cfg.Template.RightDelim != "" {
		left := r.cfg.Template.LeftDelim
		right := r.cfg.Template.RightDelim
		if left == "" {
			left = "{{"
		}
		if right == "" {
			right = "}}"
		}
		tpl = tpl.Delims(left, right)
	}
	if len(r.cfg.Template.Funcs) > 0 {
		tpl = tpl.Funcs(r.cfg.Template.Funcs)
	}

	parsed, err := tpl.Parse(queryTemplate)
	if err != nil {
		return "", fmt.Errorf("sqlrunner: parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := parsed.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("sqlrunner: execute template: %w", err)
	}
	return buf.String(), nil
}

func runWithRetry[T any](ctx context.Context, r *Runner, operation, statement string, fn func(context.Context) (T, error)) (T, error) {
	var zero T
	delay := r.policy.BaseDelay

	for attempt := 1; attempt <= r.policy.Attempts; attempt++ {
		start := time.Now()
		result, err := fn(ctx)
		duration := time.Since(start)

		if err == nil {
			r.logger.Debug(ctx, "sqlrunner.success", telemetry.Fields{
				"operation":   operation,
				"query":       statement,
				"attempt":     attempt,
				"duration_ms": duration.Seconds() * 1000,
			})
			return result, nil
		}

		if ctx.Err() != nil {
			return zero, ctx.Err()
		}

		fields := telemetry.Fields{
			"operation": operation,
			"query":     statement,
			"attempt":   attempt,
		}

		if attempt == r.policy.Attempts || !r.policy.ShouldRetry(err) {
			r.logger.Error(ctx, "sqlrunner.error", err, fields)
			return zero, err
		}

		r.logger.Warn(ctx, "sqlrunner.retry", telemetry.MergeFields(fields, telemetry.Fields{
			"next_delay_ms": delay.Seconds() * 1000,
		}))

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return zero, ctx.Err()
		case <-timer.C:
			timer.Stop()
		}

		delay = nextDelay(delay, r.policy.MaxDelay)
	}

	return zero, fmt.Errorf("sqlrunner: failed after %d attempts", r.policy.Attempts)
}

func nextDelay(current, max time.Duration) time.Duration {
	next := current * 2
	if next > max {
		return max
	}
	return next
}

func defaultShouldRetry(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	if errors.Is(err, driver.ErrBadConn) || errors.Is(err, sql.ErrConnDone) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() || netErr.Temporary() {
			return true
		}
	}

	return false
}
