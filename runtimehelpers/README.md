# Runtime Helpers

Reusable building blocks for Kolumn provider runtimes. These packages remove boilerplate from adapters and keep logging/testing behavior consistent across repositories.

## Packages

### `telemetry`
Structured logging utilities that wrap `helpers/logging` and expose a tiny interface for runtimes. Highlights:
- `telemetry.Logger` interface with `Debug/Info/Warn/Error` plus `WithComponent`.
- `telemetry.StructuredLogger` to bridge existing logging configuration.
- `telemetry.TrackOperation` helper that emits `*.start`, `*.success`, and `*.fail` events with durations, used by the runtime harness and SQL runner.
- `telemetry.MergeFields` for composing structured field maps without mutating inputs.
- `telemetry.SetLoggerFactory` lets embedders (e.g. Kolumn core) inject an observability-backed logger factory while third-party providers keep the default helper-based logger.

Usage:
```go
log := telemetry.NewLogger("postgres.runtime").WithComponent("plan")
telemetry.TrackOperation(ctx, log, "plan", func(ctx context.Context) error {
    // plan logic here
    return nil
})
```

Custom factories:
```go
telemetry.SetLoggerFactory(telemetry.FactoryFunc(func(component string) telemetry.Logger {
    return observability.NewRuntimeLogger(component)
}))
defer telemetry.ResetLoggerFactory()
```

### `sqlrunner`
A thin wrapper around `database/sql` that provides:
- Connection management (inject an existing `*sql.DB` or create one via driver+DSN).
- Retry policy with exponential backoff and transient error detection.
- Structured telemetry per operation (`exec`, `query`, `tx`).
- Optional templated queries (Go `text/template`).
- Transaction helper that reuses the retry policy and guarantees rollback if commit fails.

Usage:
```go
runner, _ := sqlrunner.NewRunner(sqlrunner.Config{
    Driver: "postgres",
    DSN:    connString,
    Logger: telemetry.NewLogger("postgres.db"),
})
rows, err := runner.Query(ctx, "SELECT * FROM foo WHERE id=$1", id)
```

### `testkit`
Test harness for exercising runtimes without RPC:
- JSON fixture loader (`LoadFixture`, `LoadFixtureFile`).
- `Harness` that runs `Init → Capabilities → Plan → Apply → Inspect` with shared telemetry.
- `FakeRuntime` for unit tests requiring instrumentation or error injection.

`Harness.Run` returns a `Result` struct so tests can assert on plan operations, apply outputs, and inspect state.

## Testing

All helper packages ship with unit tests. Execute from the repo root:
```
go test ./runtimehelpers/...
```
(When the Go module proxy is unreachable, set `GOMODCACHE`/`GOCACHE` to writable locations or vendor dependencies so the command can succeed.)
