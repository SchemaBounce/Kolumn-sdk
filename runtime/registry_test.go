package runtime

import (
	"context"
	"errors"
	"testing"
)

type noopRuntime struct{}

func (noopRuntime) Init(context.Context, InitRequest) error { return nil }
func (noopRuntime) Capabilities(context.Context) (Capabilities, error) {
	return Capabilities{Provider: "noop"}, nil
}
func (noopRuntime) Plan(context.Context, PlanRequest) (PlanResponse, error) {
	return PlanResponse{Provider: "noop"}, nil
}
func (noopRuntime) Apply(context.Context, ApplyRequest) (ApplyResult, error) {
	return ApplyResult{Success: true}, nil
}
func (noopRuntime) Inspect(context.Context, InspectRequest) (InspectResult, error) {
	return InspectResult{State: map[string]any{}}, nil
}
func (noopRuntime) Close(context.Context) error { return nil }

func TestRegistryRegisterAndLookup(t *testing.T) {
	defer Clear()

	err := Register("noop", func() (Runtime, error) {
		return noopRuntime{}, nil
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	r, err := Lookup(context.Background(), "noop")
	if err != nil {
		t.Fatalf("lookup failed: %v", err)
	}

	if _, err := r.Capabilities(context.Background()); err != nil {
		t.Fatalf("capabilities failed: %v", err)
	}
}

func TestRegistryDuplicate(t *testing.T) {
	defer Clear()
	factory := func() (Runtime, error) { return noopRuntime{}, nil }

	if err := Register("dup", factory); err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	if err := Register("dup", factory); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
}

func TestRegistryLookupMissing(t *testing.T) {
	defer Clear()
	if _, err := Lookup(context.Background(), "missing"); err == nil {
		t.Fatalf("expected missing provider error")
	}
}

func TestRegistryFactoryError(t *testing.T) {
	defer Clear()

	errFactory := errors.New("boom")
	Register("bad", func() (Runtime, error) { return nil, errFactory })

	_, err := Lookup(context.Background(), "bad")
	if err == nil || !errors.Is(err, errFactory) {
		t.Fatalf("expected factory error, got %v", err)
	}
}

func TestListAndClear(t *testing.T) {
	defer Clear()

	Register("foo", func() (Runtime, error) { return noopRuntime{}, nil })
	Register("bar", func() (Runtime, error) { return noopRuntime{}, nil })

	providers := List()
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}

	Clear()
	if len(List()) != 0 {
		t.Fatalf("expected empty registry after clear")
	}
}
