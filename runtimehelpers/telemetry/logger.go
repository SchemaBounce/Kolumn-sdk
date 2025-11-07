package telemetry

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/schemabounce/kolumn/sdk/helpers/logging"
)

// Fields represents structured telemetry fields.
type Fields map[string]any

// Level models log/metric verbosity.
type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

// Logger captures structured telemetry for runtimes.
type Logger interface {
	Debug(ctx context.Context, msg string, fields Fields)
	Info(ctx context.Context, msg string, fields Fields)
	Warn(ctx context.Context, msg string, fields Fields)
	Error(ctx context.Context, msg string, err error, fields Fields)
	WithComponent(component string) Logger
}

// StructuredLogger adapts helpers/logging to the runtime telemetry interface.
type StructuredLogger struct {
	component string
	base      *logging.Logger
}

// NewLogger builds a structured logger scoped to a component name.
func NewLogger(component string) Logger {
	if factory := currentFactory(); factory != nil {
		if logger := factory.New(component); logger != nil {
			return logger
		}
	}
	return newStructuredLogger(component)
}

func newStructuredLogger(component string) *StructuredLogger {
	if component == "" {
		component = "runtime"
	}
	return &StructuredLogger{
		component: component,
		base:      logging.NewLogger(component),
	}
}

// WithComponent clones the logger with a new component.
func (l *StructuredLogger) WithComponent(component string) Logger {
	return newStructuredLogger(component)
}

// Debug emits a debug event with fields.
func (l *StructuredLogger) Debug(_ context.Context, msg string, fields Fields) {
	l.base.DebugWithFields(msg, flatten(fields)...)
}

// Info emits an info event with fields.
func (l *StructuredLogger) Info(_ context.Context, msg string, fields Fields) {
	l.base.InfoWithFields(msg, flatten(fields)...)
}

// Warn emits a warning event with fields.
func (l *StructuredLogger) Warn(_ context.Context, msg string, fields Fields) {
	l.base.WarnWithFields(msg, flatten(fields)...)
}

// Error emits an error event with fields and the error message attached.
func (l *StructuredLogger) Error(_ context.Context, msg string, err error, fields Fields) {
	merged := cloneFields(fields)
	if err != nil {
		merged["error"] = err.Error()
	}
	l.base.ErrorWithFields(msg, flatten(merged)...)
}

// NoopLogger drops all telemetry and is safe for tests.
type NoopLogger struct{}

func (NoopLogger) Debug(context.Context, string, Fields)        {}
func (NoopLogger) Info(context.Context, string, Fields)         {}
func (NoopLogger) Warn(context.Context, string, Fields)         {}
func (NoopLogger) Error(context.Context, string, error, Fields) {}
func (NoopLogger) WithComponent(string) Logger                  { return NoopLogger{} }

// TrackOperation logs the lifecycle of a named operation.
func TrackOperation(ctx context.Context, logger Logger, name string, fn func(context.Context) error) error {
	if logger == nil {
		logger = NoopLogger{}
	}
	start := time.Now()
	logger.Info(ctx, fmt.Sprintf("%s.start", name), Fields{"ts": start.Format(time.RFC3339Nano)})

	err := fn(ctx)

	dur := time.Since(start)
	fields := Fields{"duration_ms": dur.Seconds() * 1000}
	if err != nil {
		logger.Error(ctx, fmt.Sprintf("%s.fail", name), err, fields)
		return err
	}

	logger.Info(ctx, fmt.Sprintf("%s.success", name), fields)
	return nil
}

func flatten(fields Fields) []any {
	if len(fields) == 0 {
		return nil
	}
	cloned := cloneFields(fields)
	keys := make([]string, 0, len(cloned))
	for k := range cloned {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	out := make([]any, 0, len(keys)*2)
	for _, k := range keys {
		out = append(out, k, cloned[k])
	}
	return out
}

func cloneFields(fields Fields) Fields {
	if len(fields) == 0 {
		return Fields{}
	}
	cp := make(Fields, len(fields))
	for k, v := range fields {
		cp[k] = v
	}
	return cp
}

// MergeFields merges field maps into a single map without mutating inputs.
func MergeFields(base Fields, others ...Fields) Fields {
	merged := cloneFields(base)
	for _, set := range others {
		for k, v := range set {
			merged[k] = v
		}
	}
	return merged
}
