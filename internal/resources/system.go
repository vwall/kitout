package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const systemType = "system"

// XcodeCommandLineToolsResource ensures Apple's Command Line Tools are installed.
type XcodeCommandLineToolsResource struct {
	runner platform.Runner
}

var _ engine.Resource = XcodeCommandLineToolsResource{}

// NewXcodeCommandLineToolsRequirement returns a resource for Xcode Command Line Tools.
func NewXcodeCommandLineToolsRequirement(runner platform.Runner) XcodeCommandLineToolsResource {
	return XcodeCommandLineToolsResource{runner: runner}
}

func (resource XcodeCommandLineToolsResource) ID() string {
	return systemType + ":xcode_command_line_tools"
}

func (resource XcodeCommandLineToolsResource) Type() string {
	return systemType
}

func (resource XcodeCommandLineToolsResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if resource.runner == nil {
		err := errors.New("command runner is required")
		return resource.status(engine.StateFailed, err.Error(), ""), err
	}

	result, err := resource.runner.Run(ctx, "xcode-select", "-p")
	path := strings.TrimSpace(result.Stdout)
	if err == nil && path != "" {
		return resource.status(engine.StateSatisfied, "Command Line Tools are installed", path), nil
	}
	if err == nil {
		return resource.status(engine.StateMissing, "Command Line Tools are missing", ""), nil
	}
	if isExitCode(err, 1) || isExitCode(err, 2) {
		return resource.status(engine.StateMissing, "Command Line Tools are missing", ""), nil
	}

	return resource.status(engine.StateFailed, "could not inspect Command Line Tools", path), err
}

func (resource XcodeCommandLineToolsResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message, status.Details["path"]), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "Command Line Tools already installed", status.Details["path"]), nil
	case engine.StateMissing:
		if _, err := resource.runner.Run(ctx, "xcode-select", "--install"); err != nil {
			return resource.applyResult("install", false, "could not start Command Line Tools installer", ""), err
		}
		return resource.applyResult("install", true, "started Command Line Tools installer", ""), nil
	default:
		err := fmt.Errorf("cannot apply Command Line Tools from state %s", status.State)
		return resource.applyResult("fail", false, err.Error(), status.Details["path"]), err
	}
}

func (resource XcodeCommandLineToolsResource) status(state engine.ResourceState, message, path string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details(path))
}

func (resource XcodeCommandLineToolsResource) applyResult(action string, changed bool, message, path string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details(path))
}

func (resource XcodeCommandLineToolsResource) details(path string) map[string]string {
	details := map[string]string{
		"name":     "xcode_command_line_tools",
		"required": "true",
	}
	if path != "" {
		details["path"] = path
	}
	return details
}

// RosettaResource ensures Rosetta is installed on Apple Silicon Macs.
type RosettaResource struct {
	runner platform.Runner
}

var _ engine.Resource = RosettaResource{}

// NewRosettaRequirement returns a resource for Rosetta.
func NewRosettaRequirement(runner platform.Runner) RosettaResource {
	return RosettaResource{runner: runner}
}

func (resource RosettaResource) ID() string {
	return systemType + ":rosetta"
}

func (resource RosettaResource) Type() string {
	return systemType
}

func (resource RosettaResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if resource.runner == nil {
		err := errors.New("command runner is required")
		return resource.status(engine.StateFailed, err.Error(), ""), err
	}

	archResult, err := resource.runner.Run(ctx, "uname", "-m")
	architecture := strings.TrimSpace(archResult.Stdout)
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect CPU architecture", architecture), err
	}
	if architecture != "arm64" {
		return resource.status(engine.StateSatisfied, "Rosetta is not required on this architecture", architecture), nil
	}

	_, err = resource.runner.Run(ctx, "pkgutil", "--pkg-info", "com.apple.pkg.RosettaUpdateAuto")
	if err == nil {
		return resource.status(engine.StateSatisfied, "Rosetta is installed", architecture), nil
	}
	if isExitCode(err, 1) {
		return resource.status(engine.StateMissing, "Rosetta is missing", architecture), nil
	}

	return resource.status(engine.StateFailed, "could not inspect Rosetta", architecture), err
}

func (resource RosettaResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message, status.Details["architecture"]), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "Rosetta already installed or not required", status.Details["architecture"]), nil
	case engine.StateMissing:
		if _, err := resource.runner.Run(ctx, "softwareupdate", "--install-rosetta", "--agree-to-license"); err != nil {
			return resource.applyResult("install", false, "could not install Rosetta", status.Details["architecture"]), err
		}
		return resource.applyResult("install", true, "installed Rosetta", status.Details["architecture"]), nil
	default:
		err := fmt.Errorf("cannot apply Rosetta from state %s", status.State)
		return resource.applyResult("fail", false, err.Error(), status.Details["architecture"]), err
	}
}

func (resource RosettaResource) status(state engine.ResourceState, message, architecture string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details(architecture))
}

func (resource RosettaResource) applyResult(action string, changed bool, message, architecture string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details(architecture))
}

func (resource RosettaResource) details(architecture string) map[string]string {
	details := map[string]string{
		"name":     "rosetta",
		"required": "true",
	}
	if architecture != "" {
		details["architecture"] = architecture
	}
	return details
}
