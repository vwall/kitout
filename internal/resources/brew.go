package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const brewType = "brew"

// BrewPackageResource ensures a Homebrew formula is installed.
type BrewPackageResource struct {
	name   string
	runner platform.Runner
}

var _ engine.Resource = BrewPackageResource{}

// NewBrewPackage returns a resource for one Homebrew formula.
func NewBrewPackage(name string, runner platform.Runner) BrewPackageResource {
	return BrewPackageResource{name: name, runner: runner}
}

func (resource BrewPackageResource) ID() string {
	return brewType + ":" + resource.name
}

func (resource BrewPackageResource) Type() string {
	return brewType
}

func (resource BrewPackageResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	_, err := resource.runner.Run(ctx, "brew", "list", "--formula", resource.name)
	if err == nil {
		outdated, err := resource.runner.Run(ctx, "brew", "outdated", "--formula", "--quiet", resource.name)
		if containsLine(outdated.Stdout, resource.name) {
			return resource.status(engine.StateChanged, "formula is outdated"), nil
		}
		if err != nil {
			if isExitCode(err, 1) {
				return resource.status(engine.StateSatisfied, "formula is installed"), nil
			}
			return resource.status(engine.StateFailed, "could not inspect formula updates"), err
		}
		return resource.status(engine.StateSatisfied, "formula is installed"), nil
	}
	if isExitCode(err, 1) {
		return resource.status(engine.StateMissing, "formula is missing"), nil
	}

	return resource.status(engine.StateFailed, "could not inspect formula"), err
}

func (resource BrewPackageResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "formula already installed"), nil
	case engine.StateMissing:
		if _, err := resource.runner.Run(ctx, "brew", "install", resource.name); err != nil {
			return resource.applyResult("install", false, "could not install formula"), err
		}
		return resource.applyResult("install", true, "installed formula"), nil
	case engine.StateChanged:
		if _, err := resource.runner.Run(ctx, "brew", "upgrade", resource.name); err != nil {
			return resource.applyResult("upgrade", false, "could not upgrade formula"), err
		}
		return resource.applyResult("upgrade", true, "upgraded formula"), nil
	default:
		err := fmt.Errorf("cannot apply formula %s from state %s", resource.name, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource BrewPackageResource) validate() error {
	if resource.name == "" {
		return errors.New("brew package name is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	return nil
}

func (resource BrewPackageResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource BrewPackageResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource BrewPackageResource) details() map[string]string {
	return map[string]string{"name": resource.name}
}

func isExitCode(err error, code int) bool {
	var commandError platform.CommandError
	return errors.As(err, &commandError) && commandError.Result.ExitCode == code
}

func containsLine(output, want string) bool {
	for _, line := range strings.Fields(output) {
		if line == want {
			return true
		}
	}
	return false
}
