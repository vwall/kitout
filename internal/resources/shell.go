package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const shellType = "shell"

// ShellCommandResource runs an explicitly configured shell command when its condition asks for it.
type ShellCommandResource struct {
	name        string
	command     string
	when        string
	runner      platform.Runner
	applyRunner platform.Runner
}

var _ engine.Resource = ShellCommandResource{}

// NewShellCommand returns a resource for one explicit shell command.
func NewShellCommand(name, command, when string, runner platform.Runner) ShellCommandResource {
	return NewShellCommandWithApplyRunner(name, command, when, runner, runner)
}

// NewShellCommandWithApplyRunner returns a resource with separate condition and apply runners.
func NewShellCommandWithApplyRunner(name, command, when string, runner, applyRunner platform.Runner) ShellCommandResource {
	return ShellCommandResource{name: name, command: command, when: when, runner: runner, applyRunner: applyRunner}
}

func (resource ShellCommandResource) ID() string {
	return shellType + ":" + resource.name
}

func (resource ShellCommandResource) Type() string {
	return shellType
}

func (resource ShellCommandResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	shouldRun, err := resource.shouldRun(ctx)
	if err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if shouldRun {
		return resource.status(engine.StateMissing, "command should run"), nil
	}

	return resource.status(engine.StateSatisfied, "condition already satisfied"), nil
}

func (resource ShellCommandResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "command does not need to run"), nil
	case engine.StateMissing:
		if _, err := platform.WithBoundedOutput(resource.applyRunner).Run(ctx, "sh", "-c", resource.command); err != nil {
			return resource.applyResult("run", false, "command failed"), err
		}
		return resource.applyResult("run", true, "command completed"), nil
	default:
		err := fmt.Errorf("cannot apply shell command %s from state %s", resource.name, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource ShellCommandResource) validate() error {
	if resource.name == "" {
		return errors.New("shell command name is required")
	}
	if resource.command == "" {
		return errors.New("shell command is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	if resource.applyRunner == nil {
		return errors.New("command apply runner is required")
	}
	return nil
}

func (resource ShellCommandResource) shouldRun(ctx context.Context) (bool, error) {
	condition := strings.TrimSpace(resource.when)
	if condition == "" || condition == "always" {
		return true, nil
	}

	name, ok := strings.CutPrefix(condition, "missing-command:")
	if ok {
		name = strings.TrimSpace(name)
		if name == "" {
			return false, errors.New("missing-command condition requires a command name")
		}
		_, err := resource.runner.Run(ctx, "sh", "-c", "command -v \"$1\"", "kitout", name)
		if err == nil {
			return false, nil
		}
		if isExitCode(err, 1) || isExitCode(err, 127) {
			return true, nil
		}
		return false, err
	}

	path, ok := strings.CutPrefix(condition, "exists:")
	if ok {
		path = strings.TrimSpace(path)
		if path == "" {
			return false, errors.New("exists condition requires a path")
		}
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	path, ok = strings.CutPrefix(condition, "missing:")
	if ok {
		path = strings.TrimSpace(path)
		if path == "" {
			return false, errors.New("missing condition requires a path")
		}
		_, err := os.Stat(path)
		if err == nil {
			return false, nil
		}
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	return false, fmt.Errorf("unsupported shell condition %q", resource.when)
}

func (resource ShellCommandResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource ShellCommandResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource ShellCommandResource) details() map[string]string {
	details := map[string]string{
		"name":    resource.name,
		"command": resource.command,
	}
	if resource.when != "" {
		details["when"] = resource.when
	}
	return details
}
