package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const caskType = "cask"

// CaskResource ensures a Homebrew cask is installed.
type CaskResource struct {
	name   string
	runner platform.Runner
}

var _ engine.Resource = CaskResource{}

// NewCask returns a resource for one Homebrew cask.
func NewCask(name string, runner platform.Runner) CaskResource {
	return CaskResource{name: name, runner: runner}
}

func (resource CaskResource) ID() string {
	return caskType + ":" + resource.name
}

func (resource CaskResource) Type() string {
	return caskType
}

func (resource CaskResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	_, err := resource.runner.Run(ctx, "brew", "list", "--cask", resource.name)
	if err == nil {
		return resource.status(engine.StateSatisfied, "cask is installed"), nil
	}
	if isExitCode(err, 1) {
		return resource.status(engine.StateMissing, "cask is missing"), nil
	}

	return resource.status(engine.StateFailed, "could not inspect cask"), err
}

func (resource CaskResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "cask already installed"), nil
	case engine.StateMissing:
		if _, err := resource.runner.Run(ctx, "brew", "install", "--cask", resource.name); err != nil {
			return resource.applyResult("install", false, "could not install cask"), err
		}
		return resource.applyResult("install", true, "installed cask"), nil
	default:
		err := fmt.Errorf("cannot apply cask %s from state %s", resource.name, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource CaskResource) validate() error {
	if resource.name == "" {
		return errors.New("cask name is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	return nil
}

func (resource CaskResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource CaskResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource CaskResource) details() map[string]string {
	return map[string]string{"name": resource.name}
}
