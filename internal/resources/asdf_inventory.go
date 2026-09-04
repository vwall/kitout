package resources

import (
	"context"
	"errors"
	"sync"

	"github.com/vwall/kitout/internal/platform"
)

// asdfPlanningInventory shares read-only prerequisites across one planning build.
// Per-plugin version queries and all apply commands use the underlying live runner.
type asdfPlanningInventory struct {
	runner  platform.Runner
	mu      sync.Mutex
	version asdfInventoryResult
	plugins asdfInventoryResult
}

type asdfInventoryResult struct {
	loaded bool
	result platform.CommandResult
	err    error
}

func newASDFPlanningInventory(runner platform.Runner) *asdfPlanningInventory {
	return &asdfPlanningInventory{runner: runner}
}

func (inventory *asdfPlanningInventory) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	if err := ctx.Err(); err != nil {
		return platform.CommandResult{}, err
	}
	var cached *asdfInventoryResult
	if name == "asdf" {
		if len(args) == 1 && args[0] == "--version" {
			cached = &inventory.version
		} else if len(args) == 3 && args[0] == "plugin" && args[1] == "list" && args[2] == "--urls" {
			cached = &inventory.plugins
		}
	}
	if cached == nil {
		return inventory.runner.Run(ctx, name, args...)
	}
	inventory.mu.Lock()
	defer inventory.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return platform.CommandResult{}, err
	}
	if cached.loaded {
		return cached.result, cached.err
	}
	result, err := inventory.runner.Run(ctx, name, args...)
	if ctx.Err() == nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		*cached = asdfInventoryResult{loaded: true, result: result, err: err}
	}
	return result, err
}
