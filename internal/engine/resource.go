package engine

import "context"

// Resource checks and satisfies one unit of desired machine state.
type Resource interface {
	ID() string
	Type() string
	Status(ctx context.Context) (StatusResult, error)
	Apply(ctx context.Context) (ApplyResult, error)
}
