package engine

// ResourceState describes the observed state of a resource.
type ResourceState string

const (
	StateSatisfied ResourceState = "satisfied"
	StateMissing   ResourceState = "missing"
	StateChanged   ResourceState = "changed"
	StateFailed    ResourceState = "failed"
	StateSkipped   ResourceState = "skipped"
	StateUnknown   ResourceState = "unknown"
)

// StatusResult is the structured result of checking one resource.
type StatusResult struct {
	ResourceID string
	Type       string
	State      ResourceState
	Message    string
	Details    map[string]string
}

// ApplyResult is the structured result of applying one resource.
type ApplyResult struct {
	ResourceID string
	Type       string
	Action     string
	Changed    bool
	Message    string
	Details    map[string]string
}
