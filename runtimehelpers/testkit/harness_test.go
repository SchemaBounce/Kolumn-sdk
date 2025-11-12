package testkit

import (
	"context"
	"strings"
	"testing"

	sdkRuntime "github.com/schemabounce/kolumn/sdk/runtime"
	"github.com/schemabounce/kolumn/sdk/runtimehelpers/telemetry"
)

func TestHarnessRunWithFakeRuntime(t *testing.T) {
	fake := &FakeRuntime{}
	fake.CapabilitiesFunc = func(context.Context) (sdkRuntime.Capabilities, error) {
		return sdkRuntime.Capabilities{Provider: "fake"}, nil
	}
	fake.PlanFunc = func(ctx context.Context, req sdkRuntime.PlanRequest) (sdkRuntime.PlanResponse, error) {
		if req.DesiredState["resource"] != "example" {
			t.Fatalf("unexpected desired state: %v", req.DesiredState)
		}
		return sdkRuntime.PlanResponse{Provider: "fake", Summary: map[string]any{"count": 1}}, nil
	}
	fake.ApplyFunc = func(ctx context.Context, req sdkRuntime.ApplyRequest) (sdkRuntime.ApplyResult, error) {
		if req.Plan.Provider != "fake" {
			t.Fatalf("plan provider lost during apply")
		}
		return sdkRuntime.ApplyResult{Success: true}, nil
	}
	fake.InspectFunc = func(ctx context.Context, req sdkRuntime.InspectRequest) (sdkRuntime.InspectResult, error) {
		state := map[string]any{"name": req.Scope.Name}
		return sdkRuntime.InspectResult{State: state}, nil
	}

	harness := Harness{
		Provider: "fake",
		Factory: func(context.Context) (sdkRuntime.Runtime, error) {
			return fake, nil
		},
		Logger: telemetry.NoopLogger{},
	}

	fixture := Fixture{
		Init:  sdkRuntime.InitRequest{Provider: "fake"},
		Plan:  sdkRuntime.PlanRequest{DesiredState: map[string]any{"resource": "example"}},
		Apply: map[string]any{"mode": "check"},
		Inspect: &sdkRuntime.InspectRequest{
			Scope: sdkRuntime.ResourceRef{Name: "example"},
		},
	}

	result, err := harness.Run(context.Background(), fixture)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}

	if result.Capabilities.Provider != "fake" {
		t.Fatalf("expected fake capabilities, got %s", result.Capabilities.Provider)
	}
	if result.Plan.Summary["count"].(int) != 1 {
		t.Fatalf("expected plan summary count=1")
	}
	if !result.Apply.Success {
		t.Fatalf("expected apply success")
	}
	if result.Inspect == nil || result.Inspect.State["name"] != "example" {
		t.Fatalf("inspect state missing")
	}

	calls := fake.Calls()
	expectedOrder := []string{"Init", "Capabilities", "Plan", "Apply", "Inspect", "Close"}
	if len(calls) != len(expectedOrder) {
		t.Fatalf("expected %d calls, got %d", len(expectedOrder), len(calls))
	}
	for i, name := range expectedOrder {
		if calls[i].Name != name {
			t.Fatalf("expected call %d to be %s, got %s", i, name, calls[i].Name)
		}
	}
}

func TestLoadFixture(t *testing.T) {
	raw := `{
        "init": {"provider": "pg"},
        "plan": {"desired_state": {"foo": "bar"}},
        "apply_options": {"dry": true}
    }`

	fx, err := LoadFixture(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	if fx.Init.Provider != "pg" {
		t.Fatalf("unexpected provider %s", fx.Init.Provider)
	}
	if fx.Apply["dry"].(bool) != true {
		t.Fatalf("expected dry=true")
	}
}
