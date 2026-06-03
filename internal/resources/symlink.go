package resources

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/vwall/kitout/internal/engine"
)

const symlinkType = "symlink"

// SymlinkResource ensures a target path points to a source path.
type SymlinkResource struct {
	source  string
	target  string
	replace bool
}

var _ engine.Resource = SymlinkResource{}

// NewSymlink returns a resource for one symbolic link.
func NewSymlink(source, target string, replace bool) SymlinkResource {
	return SymlinkResource{source: source, target: target, replace: replace}
}

func (resource SymlinkResource) ID() string {
	return symlinkType + ":" + resource.target
}

func (resource SymlinkResource) Type() string {
	return symlinkType
}

func (resource SymlinkResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return resource.status(engine.StateFailed, "context canceled while checking symlink"), err
	}
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	info, err := os.Lstat(resource.target)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, "symlink is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect symlink target"), err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if resource.replace {
			return resource.status(engine.StateChanged, "target exists and is not a symlink"), nil
		}
		return resource.status(engine.StateFailed, "target exists and replacement is not allowed"), nil
	}

	currentSource, err := os.Readlink(resource.target)
	if err != nil {
		return resource.status(engine.StateFailed, "could not read symlink"), err
	}
	if currentSource != resource.source {
		return resource.status(engine.StateChanged, "symlink points elsewhere"), nil
	}

	return resource.status(engine.StateSatisfied, "symlink is correct"), nil
}

func (resource SymlinkResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "symlink already correct"), nil
	case engine.StateMissing:
		if err := os.Symlink(resource.source, resource.target); err != nil {
			return resource.applyResult("create", false, "could not create symlink"), err
		}
		return resource.applyResult("create", true, "created symlink"), nil
	case engine.StateChanged:
		if !resource.replace {
			err := fmt.Errorf("cannot replace symlink %s: replacement is not allowed", resource.target)
			return resource.applyResult("fail", false, err.Error()), err
		}
		if err := os.Remove(resource.target); err != nil {
			return resource.applyResult("replace", false, "could not remove existing target"), err
		}
		if err := os.Symlink(resource.source, resource.target); err != nil {
			return resource.applyResult("replace", false, "could not create replacement symlink"), err
		}
		return resource.applyResult("replace", true, "replaced symlink"), nil
	case engine.StateFailed:
		err := errors.New(status.Message)
		return resource.applyResult("fail", false, status.Message), err
	default:
		err := fmt.Errorf("cannot apply symlink %s from state %s", resource.target, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource SymlinkResource) validate() error {
	if resource.source == "" {
		return errors.New("symlink source is required")
	}
	if resource.target == "" {
		return errors.New("symlink target is required")
	}
	return nil
}

func (resource SymlinkResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource SymlinkResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource SymlinkResource) details() map[string]string {
	replace := "false"
	if resource.replace {
		replace = "true"
	}
	return map[string]string{
		"source":  resource.source,
		"target":  resource.target,
		"replace": replace,
	}
}
