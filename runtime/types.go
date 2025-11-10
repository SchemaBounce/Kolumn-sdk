package runtime

import "context"

// Runtime defines the minimal contract Kolumn core expects from any provider implementation.
type Runtime interface {
	Init(ctx context.Context, req InitRequest) error
	Capabilities(ctx context.Context) (Capabilities, error)
	Plan(ctx context.Context, req PlanRequest) (PlanResponse, error)
	Apply(ctx context.Context, req ApplyRequest) (ApplyResult, error)
	Inspect(ctx context.Context, req InspectRequest) (InspectResult, error)
	Close(ctx context.Context) error
}

// InitRequest contains connection and configuration data required to initialise a runtime.
type InitRequest struct {
	Provider   string                 `json:"provider"`
	Connection map[string]any         `json:"connection"`
	Settings   map[string]any         `json:"settings,omitempty"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
}

// Capabilities describes high level abilities the runtime exposes.
type Capabilities struct {
	Provider     string                 `json:"provider"`
	Version      string                 `json:"version"`
	Protocol     string                 `json:"protocol"`
	Features     map[string]bool        `json:"features,omitempty"`
	ResourceKinds []ResourceKind        `json:"resource_kinds,omitempty"`
	Details      map[string]any         `json:"details,omitempty"`
}

// ResourceKind captures metadata about a resource the runtime can manage.
type ResourceKind struct {
	Type        string         `json:"type"`
	Description string         `json:"description,omitempty"`
	Operations  []string       `json:"operations,omitempty"`
	Config      map[string]any `json:"config_schema,omitempty"`
	State       map[string]any `json:"state_schema,omitempty"`
}

// PlanRequest contains desired state and options for planning.
type PlanRequest struct {
	DesiredState map[string]any         `json:"desired_state"`
	CurrentState map[string]any         `json:"current_state,omitempty"`
	Options      map[string]any         `json:"options,omitempty"`
}

// PlanResponse contains a generic change list produced by the runtime.
type PlanResponse struct {
	Provider   string                 `json:"provider"`
	Operations []Operation            `json:"operations"`
	Summary    map[string]any         `json:"summary,omitempty"`
	Metadata   map[string]any         `json:"metadata,omitempty"`
}

// Operation represents a single planned change.
type Operation struct {
	ID          string                 `json:"id"`
	Action      string                 `json:"action"`
	Resource    ResourceRef            `json:"resource"`
	Risk        string                 `json:"risk,omitempty"`
	Statements  []string               `json:"statements,omitempty"`
	Rollback    []string               `json:"rollback,omitempty"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

// ResourceRef is a provider-agnostic handle to a resource instance.
type ResourceRef struct {
	Type string `json:"type"`
	Name string `json:"name"`
	ID   string `json:"id,omitempty"`
}

// ApplyRequest passes a previously generated plan back to the runtime.
type ApplyRequest struct {
	Plan     PlanResponse       `json:"plan"`
	Options  map[string]any     `json:"options,omitempty"`
}

// ApplyResult contains the outcome of executing a plan.
type ApplyResult struct {
	Success bool                   `json:"success"`
	Errors  []string               `json:"errors,omitempty"`
	Outputs map[string]any         `json:"outputs,omitempty"`
	Metadata map[string]any        `json:"metadata,omitempty"`
}

// InspectRequest asks the runtime for current state details.
type InspectRequest struct {
	Scope    ResourceRef          `json:"scope,omitempty"`
	Options  map[string]any       `json:"options,omitempty"`
}

// InspectResult contains discovered state information.
type InspectResult struct {
	State    map[string]any       `json:"state"`
	Metadata map[string]any       `json:"metadata,omitempty"`
}
