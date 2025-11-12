package testkit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	sdkRuntime "github.com/schemabounce/kolumn/sdk/runtime"
	"github.com/schemabounce/kolumn/sdk/runtimehelpers/telemetry"
)

// Fixture represents a minimal round-trip provider scenario serialized to JSON.
type Fixture struct {
	Init    sdkRuntime.InitRequest     `json:"init"`
	Plan    sdkRuntime.PlanRequest     `json:"plan"`
	Apply   map[string]any             `json:"apply_options,omitempty"`
	Inspect *sdkRuntime.InspectRequest `json:"inspect,omitempty"`
}

// Result captures the outputs of running a fixture through a runtime.
type Result struct {
	Capabilities sdkRuntime.Capabilities
	Plan         sdkRuntime.PlanResponse
	Apply        sdkRuntime.ApplyResult
	Inspect      *sdkRuntime.InspectResult
}

// Harness coordinates runtime creation and fixture execution.
type Harness struct {
	Provider string
	Factory  func(context.Context) (sdkRuntime.Runtime, error)
	Logger   telemetry.Logger
}

// Run executes the provided fixture through the runtime pipeline.
func (h Harness) Run(ctx context.Context, fx Fixture) (Result, error) {
	logger := h.Logger
	if logger == nil {
		logger = telemetry.NoopLogger{}
	}

	runtimeInstance, err := h.runtime(ctx)
	if err != nil {
		return Result{}, err
	}
	defer runtimeInstance.Close(ctx)

	log := logger.WithComponent(h.Provider)

	if err := telemetry.TrackOperation(ctx, log, "runtime.init", func(ctx context.Context) error {
		return runtimeInstance.Init(ctx, fx.Init)
	}); err != nil {
		return Result{}, fmt.Errorf("testkit: init: %w", err)
	}

	var caps sdkRuntime.Capabilities
	if err := telemetry.TrackOperation(ctx, log, "runtime.capabilities", func(ctx context.Context) error {
		var err error
		caps, err = runtimeInstance.Capabilities(ctx)
		return err
	}); err != nil {
		return Result{}, fmt.Errorf("testkit: capabilities: %w", err)
	}

	var planResp sdkRuntime.PlanResponse
	if err := telemetry.TrackOperation(ctx, log, "runtime.plan", func(ctx context.Context) error {
		var err error
		planResp, err = runtimeInstance.Plan(ctx, fx.Plan)
		return err
	}); err != nil {
		return Result{}, fmt.Errorf("testkit: plan: %w", err)
	}

	applyReq := sdkRuntime.ApplyRequest{Plan: planResp}
	if len(fx.Apply) > 0 {
		applyReq.Options = fx.Apply
	}

	var applyResult sdkRuntime.ApplyResult
	if err := telemetry.TrackOperation(ctx, log, "runtime.apply", func(ctx context.Context) error {
		var err error
		applyResult, err = runtimeInstance.Apply(ctx, applyReq)
		return err
	}); err != nil {
		return Result{}, fmt.Errorf("testkit: apply: %w", err)
	}

	var inspectResult *sdkRuntime.InspectResult
	if fx.Inspect != nil {
		var ir sdkRuntime.InspectResult
		if err := telemetry.TrackOperation(ctx, log, "runtime.inspect", func(ctx context.Context) error {
			var err error
			ir, err = runtimeInstance.Inspect(ctx, *fx.Inspect)
			return err
		}); err != nil {
			return Result{}, fmt.Errorf("testkit: inspect: %w", err)
		}
		inspectResult = &ir
	}

	return Result{
		Capabilities: caps,
		Plan:         planResp,
		Apply:        applyResult,
		Inspect:      inspectResult,
	}, nil
}

func (h Harness) runtime(ctx context.Context) (sdkRuntime.Runtime, error) {
	if h.Factory != nil {
		return h.Factory(ctx)
	}
	if h.Provider == "" {
		return nil, fmt.Errorf("testkit: provider id required when factory is nil")
	}
	return sdkRuntime.Lookup(ctx, h.Provider)
}

// LoadFixture reads a fixture from an io.Reader.
func LoadFixture(r io.Reader) (Fixture, error) {
	var fx Fixture
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fx); err != nil {
		return Fixture{}, fmt.Errorf("testkit: decode fixture: %w", err)
	}
	return fx, nil
}

// LoadFixtureFile reads a fixture from disk.
func LoadFixtureFile(path string) (Fixture, error) {
	f, err := os.Open(path)
	if err != nil {
		return Fixture{}, fmt.Errorf("testkit: open fixture: %w", err)
	}
	defer f.Close()
	return LoadFixture(f)
}

// FakeRuntime is a configurable runtime implementation for tests.
type FakeRuntime struct {
	InitFunc         func(context.Context, sdkRuntime.InitRequest) error
	CapabilitiesFunc func(context.Context) (sdkRuntime.Capabilities, error)
	PlanFunc         func(context.Context, sdkRuntime.PlanRequest) (sdkRuntime.PlanResponse, error)
	ApplyFunc        func(context.Context, sdkRuntime.ApplyRequest) (sdkRuntime.ApplyResult, error)
	InspectFunc      func(context.Context, sdkRuntime.InspectRequest) (sdkRuntime.InspectResult, error)
	CloseFunc        func(context.Context) error

	mu    sync.Mutex
	calls []CallRecord
}

// CallRecord captures executed method names + inputs for assertions.
type CallRecord struct {
	Name    string
	Payload any
}

// Calls returns a snapshot of recorded calls.
func (f *FakeRuntime) Calls() []CallRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	cp := make([]CallRecord, len(f.calls))
	copy(cp, f.calls)
	return cp
}

func (f *FakeRuntime) record(name string, payload any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, CallRecord{Name: name, Payload: payload})
}

// Ensure FakeRuntime implements sdkRuntime.Runtime.
var _ sdkRuntime.Runtime = (*FakeRuntime)(nil)

func (f *FakeRuntime) Init(ctx context.Context, req sdkRuntime.InitRequest) error {
	f.record("Init", req)
	if f.InitFunc != nil {
		return f.InitFunc(ctx, req)
	}
	return nil
}

func (f *FakeRuntime) Capabilities(ctx context.Context) (sdkRuntime.Capabilities, error) {
	f.record("Capabilities", nil)
	if f.CapabilitiesFunc != nil {
		return f.CapabilitiesFunc(ctx)
	}
	return sdkRuntime.Capabilities{Provider: "fake"}, nil
}

func (f *FakeRuntime) Plan(ctx context.Context, req sdkRuntime.PlanRequest) (sdkRuntime.PlanResponse, error) {
	f.record("Plan", req)
	if f.PlanFunc != nil {
		return f.PlanFunc(ctx, req)
	}
	return sdkRuntime.PlanResponse{Provider: "fake"}, nil
}

func (f *FakeRuntime) Apply(ctx context.Context, req sdkRuntime.ApplyRequest) (sdkRuntime.ApplyResult, error) {
	f.record("Apply", req)
	if f.ApplyFunc != nil {
		return f.ApplyFunc(ctx, req)
	}
	return sdkRuntime.ApplyResult{Success: true}, nil
}

func (f *FakeRuntime) Inspect(ctx context.Context, req sdkRuntime.InspectRequest) (sdkRuntime.InspectResult, error) {
	f.record("Inspect", req)
	if f.InspectFunc != nil {
		return f.InspectFunc(ctx, req)
	}
	return sdkRuntime.InspectResult{State: map[string]any{}}, nil
}

func (f *FakeRuntime) Close(ctx context.Context) error {
	f.record("Close", nil)
	if f.CloseFunc != nil {
		return f.CloseFunc(ctx)
	}
	return nil
}
