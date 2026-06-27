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

// AdvisorySeverity describes how prominently an advisory should be shown.
type AdvisorySeverity string

const (
	AdvisoryNotice  AdvisorySeverity = "notice"
	AdvisoryWarning AdvisorySeverity = "warning"
)

// Advisory describes non-blocking information discovered while checking a
// resource. Advisories do not make a satisfied resource drift from config.
type Advisory struct {
	Code     string
	Severity AdvisorySeverity
	Message  string
	Fix      string
	Details  map[string]string
}

// StatusResult is the structured result of checking one resource.
type StatusResult struct {
	ResourceID string
	Type       string
	State      ResourceState
	Message    string
	Details    map[string]string
	Advisories []Advisory
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
