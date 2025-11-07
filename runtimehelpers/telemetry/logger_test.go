package telemetry

import (
	"context"
	"errors"
	"testing"
)

type record struct {
	level  Level
	msg    string
	fields Fields
	err    error
}

type recordingLogger struct {
	entries []record
}

func (r *recordingLogger) Debug(ctx context.Context, msg string, fields Fields) {
	r.entries = append(r.entries, record{level: LevelDebug, msg: msg, fields: fields})
}

func (r *recordingLogger) Info(ctx context.Context, msg string, fields Fields) {
	r.entries = append(r.entries, record{level: LevelInfo, msg: msg, fields: fields})
}

func (r *recordingLogger) Warn(ctx context.Context, msg string, fields Fields) {
	r.entries = append(r.entries, record{level: LevelWarn, msg: msg, fields: fields})
}

func (r *recordingLogger) Error(ctx context.Context, msg string, err error, fields Fields) {
	r.entries = append(r.entries, record{level: LevelError, msg: msg, fields: fields, err: err})
}

func (r *recordingLogger) WithComponent(string) Logger { return r }

func TestMergeFields(t *testing.T) {
	base := Fields{"one": 1}
	extra := Fields{"two": 2}
	merged := MergeFields(base, extra)

	if len(merged) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(merged))
	}
	if base["two"] != nil {
		t.Fatalf("expected base map to remain untouched")
	}
	if merged["two"].(int) != 2 {
		t.Fatalf("expected merged value 2, got %v", merged["two"])
	}
}

func TestTrackOperationSuccess(t *testing.T) {
	rec := &recordingLogger{}
	err := TrackOperation(context.Background(), rec, "plan", func(context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(rec.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rec.entries))
	}
	if rec.entries[0].level != LevelInfo || rec.entries[1].level != LevelInfo {
		t.Fatalf("expected both entries to be info level")
	}
}

func TestTrackOperationError(t *testing.T) {
	rec := &recordingLogger{}
	boom := errors.New("boom")
	err := TrackOperation(context.Background(), rec, "apply", func(context.Context) error {
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom error, got %v", err)
	}
	if len(rec.entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(rec.entries))
	}
	if rec.entries[1].level != LevelError {
		t.Fatalf("expected error level on failure")
	}
	if rec.entries[1].err == nil {
		t.Fatalf("expected error to be recorded")
	}
}

func TestNewLoggerUsesFactoryOverride(t *testing.T) {
	t.Cleanup(func() {
		ResetLoggerFactory()
	})

	calls := 0
	SetLoggerFactory(FactoryFunc(func(component string) Logger {
		calls++
		if component != "custom.component" {
			t.Fatalf("unexpected component %q", component)
		}
		return NoopLogger{}
	}))

	logger := NewLogger("custom.component")
	if _, ok := logger.(NoopLogger); !ok {
		t.Fatalf("expected NoopLogger, got %T", logger)
	}
	if calls != 1 {
		t.Fatalf("expected factory to be invoked once, got %d", calls)
	}
}

func TestNewLoggerFallsBackWhenFactoryReturnsNil(t *testing.T) {
	t.Cleanup(func() {
		ResetLoggerFactory()
	})

	SetLoggerFactory(FactoryFunc(func(string) Logger {
		return nil
	}))

	logger := NewLogger("fallback")
	if logger == nil {
		t.Fatal("expected fallback logger, got nil")
	}
}
