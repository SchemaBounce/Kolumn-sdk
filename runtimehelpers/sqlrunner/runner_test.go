package sqlrunner

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"sync"
	"testing"

	"github.com/schemabounce/kolumn/sdk/runtimehelpers/telemetry"
)

func TestNewRunnerRequiresConfig(t *testing.T) {
	if _, err := NewRunner(Config{}); err == nil {
		t.Fatalf("expected error when no connection info provided")
	}
}

func TestExecRetriesOnTransientError(t *testing.T) {
	state := &stubState{execErrors: []error{sql.ErrConnDone, nil}}
	db := sql.OpenDB(&stubConnector{state: state})

	runner, err := NewRunner(Config{
		ExistingDB: db,
		Logger:     telemetry.NoopLogger{},
		Retry: RetryPolicy{
			Attempts: 2,
		},
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	ctx := context.Background()
	if _, err := runner.Exec(ctx, "INSERT INTO test"); err != nil {
		t.Fatalf("exec should succeed after retry: %v", err)
	}

	if len(state.execQueries) != 2 {
		t.Fatalf("expected 2 exec attempts, got %d", len(state.execQueries))
	}
}

func TestWithTransactionRollbackOnError(t *testing.T) {
	state := &stubState{}
	db := sql.OpenDB(&stubConnector{state: state})

	runner, err := NewRunner(Config{ExistingDB: db, Logger: telemetry.NoopLogger{}})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	wantErr := errors.New("boom")
	err = runner.WithTransaction(context.Background(), nil, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "UPDATE items SET value=1"); err != nil {
			return err
		}
		return wantErr
	})

	if !errors.Is(err, wantErr) {
		t.Fatalf("expected boom error, got %v", err)
	}
	if state.rollbackCount != 1 {
		t.Fatalf("expected rollback to be called once, got %d", state.rollbackCount)
	}
	if state.commitCount != 0 {
		t.Fatalf("did not expect commit")
	}
}

func TestQueryTemplateRenders(t *testing.T) {
	state := &stubState{
		queryColumns: []string{"id"},
		queryRows:    [][]driver.Value{{int64(42)}},
	}
	db := sql.OpenDB(&stubConnector{state: state})

	runner, err := NewRunner(Config{
		ExistingDB: db,
		Logger:     telemetry.NoopLogger{},
	})
	if err != nil {
		t.Fatalf("new runner: %v", err)
	}

	rows, err := runner.QueryTemplate(context.Background(), "SELECT id FROM {{.table}} WHERE id = {{.id}}", map[string]any{
		"table": "users",
		"id":    42,
	})
	if err != nil {
		t.Fatalf("query template: %v", err)
	}
	defer rows.Close()

	if !rows.Next() {
		t.Fatalf("expected a row")
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if id != 42 {
		t.Fatalf("expected id 42, got %d", id)
	}
	if state.lastQuery != "SELECT id FROM users WHERE id = 42" {
		t.Fatalf("unexpected query: %s", state.lastQuery)
	}
}

// --- Stub driver implementation for tests ---

type stubConnector struct {
	state *stubState
}

func (c *stubConnector) Connect(context.Context) (driver.Conn, error) {
	return &stubConn{state: c.state}, nil
}

func (c *stubConnector) Driver() driver.Driver { return stubDriver{} }

type stubDriver struct{}

func (stubDriver) Open(name string) (driver.Conn, error) {
	return nil, errors.New("stub driver requires connector")
}

type stubState struct {
	mu            sync.Mutex
	execErrors    []error
	execQueries   []string
	lastQuery     string
	queryColumns  []string
	queryRows     [][]driver.Value
	beginErr      error
	commitErr     error
	rollbackErr   error
	commitCount   int
	rollbackCount int
}

type stubConn struct {
	state *stubState
}

func (c *stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *stubConn) Close() error                        { return nil }
func (c *stubConn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *stubConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	if c.state.beginErr != nil {
		return nil, c.state.beginErr
	}
	return &stubTx{state: c.state}, nil
}

func (c *stubConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.execQueries = append(c.state.execQueries, query)
	if len(c.state.execErrors) > 0 {
		err := c.state.execErrors[0]
		c.state.execErrors = c.state.execErrors[1:]
		if err != nil {
			return nil, err
		}
	}
	return driver.RowsAffected(1), nil
}

func (c *stubConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	c.state.mu.Lock()
	defer c.state.mu.Unlock()
	c.state.lastQuery = query
	rows := make([][]driver.Value, len(c.state.queryRows))
	for i := range c.state.queryRows {
		row := make([]driver.Value, len(c.state.queryRows[i]))
		copy(row, c.state.queryRows[i])
		rows[i] = row
	}
	return &stubRows{columns: append([]string(nil), c.state.queryColumns...), rows: rows}, nil
}

func (c *stubConn) Ping(ctx context.Context) error { return nil }

// Ensure interfaces satisfied
var _ driver.ExecerContext = (*stubConn)(nil)
var _ driver.QueryerContext = (*stubConn)(nil)
var _ driver.ConnBeginTx = (*stubConn)(nil)
var _ driver.Pinger = (*stubConn)(nil)

type stubTx struct {
	state *stubState
}

func (tx *stubTx) Commit() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.commitCount++
	return tx.state.commitErr
}

func (tx *stubTx) Rollback() error {
	tx.state.mu.Lock()
	defer tx.state.mu.Unlock()
	tx.state.rollbackCount++
	return tx.state.rollbackErr
}

type stubRows struct {
	columns []string
	rows    [][]driver.Value
	idx     int
}

func (r *stubRows) Columns() []string { return r.columns }
func (r *stubRows) Close() error      { return nil }

func (r *stubRows) Next(dest []driver.Value) error {
	if r.idx >= len(r.rows) {
		return io.EOF
	}
	copy(dest, r.rows[r.idx])
	r.idx++
	return nil
}

var _ driver.Rows = (*stubRows)(nil)
