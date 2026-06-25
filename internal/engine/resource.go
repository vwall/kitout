package engine

import "context"

// Resource checks and satisfies one unit of desired machine state.
type Resource interface {
	ID() string
	Type() string
	Status(ctx context.Context) (StatusResult, error)
	Apply(ctx context.Context) (ApplyResult, error)
}

// ApplyBlocker marks a resource whose unmet state should stop later apply work.
type ApplyBlocker interface {
	BlocksApply() bool
}
