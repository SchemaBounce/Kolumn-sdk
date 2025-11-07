# Kolumn Provider Runtime Contract

Phase one defines a provider-neutral runtime interface that core Kolumn and providers can both rely on. The goal is to remove RPC-specific assumptions while keeping extensibility for future transport layers.

## Runtime Lifecycle

```
runtime, _ := provider.Lookup(ctx, "postgres")

defer runtime.Close(ctx)

if err := runtime.Init(ctx, runtime.InitRequest{Provider: "postgres", Connection: conn}); err != nil {
    // surface configuration errors early
}

caps, _ := runtime.Capabilities(ctx)
plan, _ := runtime.Plan(ctx, planReq)
applyResult, _ := runtime.Apply(ctx, runtime.ApplyRequest{Plan: plan})
state, _ := runtime.Inspect(ctx, runtime.InspectRequest{})
```

### InitRequest
- `Provider`: logical provider identifier (e.g. `postgres`, `snowflake`).
- `Connection`: opaque map of connection settings supplied from Kolumn HCL.
- `Settings`: optional provider-specific behaviour flags.
- `Metadata`: diagnostic context (workspace ID, run ID, etc.).

### Capabilities
Providers describe the resource kinds they can manage, version/build info, and feature toggles. Core surfaces this to UX layers and validation.

### Plan / Apply
- `PlanRequest` contains desired and current state as provider-neutral maps and optional planner options.
- `PlanResponse` returns a list of `Operation` objects. Each operation includes an `Action`, a `ResourceRef`, and optional SQL statements / metadata.
- `ApplyRequest` wraps a previously generated plan and execution options.
- `ApplyResult` reports success, errors, and arbitrary outputs.

### Inspect
Allows discovery of provider state (e.g. fetch current schema, health metrics). Implementations can scope inspection with `InspectRequest.Scope`.

## Registry
- Providers register factories with `runtime.Register("postgres", NewRuntime)`. Factories must be idempotent and return a fresh runtime.
- Core calls `runtime.Lookup(ctx, providerID)` to obtain a runtime without knowing if it is in-process or RPC-backed.
- `runtime.Clear()` exists for tests; production code should only call `Register` during init.

### Helper Packages
To keep adapters consistent, runtimes should lean on the shared helpers bundled with the SDK:

- `runtimehelpers/telemetry` exposes a slim `Logger` interface and `TrackOperation` helper so runtimes inherit the standard structured logging fields (`provider`, `request_id`, etc.).
- `runtimehelpers/sqlrunner` wraps `database/sql` with retry/backoff, templated query support, and telemetry instrumentation. Use it whenever a runtime needs to execute SQL directly.
- `runtimehelpers/testkit` provides a `Harness`, JSON fixture loader, and `FakeRuntime` so providers can smoke-test `Init → Plan → Apply → Inspect` flows without RPC infrastructure.

Using these helpers avoids bespoke logging/DB code in each provider and keeps tests deterministic.

## Error Handling
- Factories should return typed errors where possible (`ErrInvalidConfig`, `ErrConnectionFailed`, etc.). Core will wrap but preserve underlying causes.
- Runtime methods receive a `context.Context`; they must honour cancellation for long-running operations.

## Next Steps
- Providers migrate from the legacy 4-method RPC interface to the runtime contract via lightweight adapters.
- Core Kolumn replaces the emergency direct-mode codepath with the registry lookup so all execution flows via the new interface.
