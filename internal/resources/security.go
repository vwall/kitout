package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const securityType = "security"

const socketfilterfwPath = "/usr/libexec/ApplicationFirewall/socketfilterfw"

// FileVaultResource requires FileVault to be enabled.
type FileVaultResource struct {
	runner platform.Runner
}

var _ engine.Resource = FileVaultResource{}

// NewFileVaultRequirement returns a resource that verifies FileVault is enabled.
func NewFileVaultRequirement(runner platform.Runner) FileVaultResource {
	return FileVaultResource{runner: runner}
}

func (resource FileVaultResource) ID() string {
	return securityType + ":filevault"
}

func (resource FileVaultResource) Type() string {
	return securityType
}

func (resource FileVaultResource) BlocksApply() bool {
	return true
}

func (resource FileVaultResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if resource.runner == nil {
		err := errors.New("command runner is required")
		return resource.status(engine.StateFailed, err.Error()), err
	}

	result, err := resource.runner.Run(ctx, "fdesetup", "status")
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect FileVault"), err
	}

	enabled, ok := parseFileVaultEnabled(result.Stdout)
	if !ok {
		err := errors.New("could not parse FileVault status")
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if enabled {
		return resource.status(engine.StateSatisfied, "FileVault is enabled"), nil
	}

	return resource.status(engine.StateMissing, "FileVault must be enabled"), nil
}

func (resource FileVaultResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "FileVault already enabled"), nil
	case engine.StateMissing:
		if _, openErr := resource.runner.Run(ctx, "open", "x-apple.systempreferences:com.apple.preference.security"); openErr != nil {
			return resource.applyResult("manual", false, "could not open System Settings for FileVault"), openErr
		}
		err := errors.New("enable FileVault manually in System Settings, then rerun Kitout")
		return resource.applyResult("manual", false, err.Error()), err
	default:
		err := fmt.Errorf("cannot apply FileVault requirement from state %s", status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource FileVaultResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource FileVaultResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource FileVaultResource) details() map[string]string {
	return map[string]string{
		"name":     "filevault",
		"required": "true",
	}
}

func parseFileVaultEnabled(stdout string) (bool, bool) {
	status := strings.ToLower(stdout)
	switch {
	case strings.Contains(status, "filevault is on"):
		return true, true
	case strings.Contains(status, "filevault is off"):
		return false, true
	default:
		return false, false
	}
}

// FirewallResource ensures the macOS application firewall is enabled or disabled.
type FirewallResource struct {
	enabled bool
	runner  platform.Runner
}

var _ engine.Resource = FirewallResource{}

// NewFirewall returns a resource for the macOS application firewall state.
func NewFirewall(enabled bool, runner platform.Runner) FirewallResource {
	return FirewallResource{enabled: enabled, runner: runner}
}

func (resource FirewallResource) ID() string {
	return securityType + ":firewall"
}

func (resource FirewallResource) Type() string {
	return securityType
}

func (resource FirewallResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if resource.runner == nil {
		err := errors.New("command runner is required")
		return resource.status(engine.StateFailed, err.Error()), err
	}

	result, err := resource.runner.Run(ctx, socketfilterfwPath, "--getglobalstate")
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect firewall"), err
	}

	enabled, ok := parseFirewallEnabled(result.Stdout)
	if !ok {
		err := errors.New("could not parse firewall status")
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if enabled == resource.enabled {
		return resource.status(engine.StateSatisfied, firewallStateMessage("firewall is", resource.enabled)), nil
	}

	return resource.status(engine.StateChanged, "firewall state differs"), nil
}

func (resource FirewallResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, firewallStateMessage("firewall already", resource.enabled)), nil
	case engine.StateChanged:
		if _, err := resource.runner.Run(ctx, "sudo", socketfilterfwPath, "--setglobalstate", onOff(resource.enabled)); err != nil {
			return resource.applyResult("set", false, "could not update firewall"), err
		}
		return resource.applyResult("set", true, firewallStateMessage("updated firewall to", resource.enabled)), nil
	default:
		err := fmt.Errorf("cannot apply firewall from state %s", status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource FirewallResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource FirewallResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource FirewallResource) details() map[string]string {
	return map[string]string{
		"name":    "firewall",
		"enabled": fmt.Sprintf("%t", resource.enabled),
	}
}

func parseFirewallEnabled(stdout string) (bool, bool) {
	status := strings.ToLower(stdout)
	switch {
	case strings.Contains(status, "state = 1") || strings.Contains(status, "firewall is enabled"):
		return true, true
	case strings.Contains(status, "state = 0") || strings.Contains(status, "firewall is disabled"):
		return false, true
	default:
		return false, false
	}
}

// FirewallStealthModeResource ensures macOS firewall stealth mode has the configured state.
type FirewallStealthModeResource struct {
	enabled bool
	runner  platform.Runner
}

var _ engine.Resource = FirewallStealthModeResource{}

// NewFirewallStealthMode returns a resource for macOS firewall stealth mode.
func NewFirewallStealthMode(enabled bool, runner platform.Runner) FirewallStealthModeResource {
	return FirewallStealthModeResource{enabled: enabled, runner: runner}
}

func (resource FirewallStealthModeResource) ID() string {
	return securityType + ":firewall_stealth_mode"
}

func (resource FirewallStealthModeResource) Type() string {
	return securityType
}

func (resource FirewallStealthModeResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if resource.runner == nil {
		err := errors.New("command runner is required")
		return resource.status(engine.StateFailed, err.Error()), err
	}

	result, err := resource.runner.Run(ctx, socketfilterfwPath, "--getstealthmode")
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect firewall stealth mode"), err
	}

	enabled, ok := parseStealthModeEnabled(result.Stdout)
	if !ok {
		err := errors.New("could not parse firewall stealth mode status")
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if enabled == resource.enabled {
		return resource.status(engine.StateSatisfied, firewallStateMessage("firewall stealth mode is", resource.enabled)), nil
	}

	return resource.status(engine.StateChanged, "firewall stealth mode differs"), nil
}

func (resource FirewallStealthModeResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, firewallStateMessage("firewall stealth mode already", resource.enabled)), nil
	case engine.StateChanged:
		if _, err := resource.runner.Run(ctx, "sudo", socketfilterfwPath, "--setstealthmode", onOff(resource.enabled)); err != nil {
			return resource.applyResult("set", false, "could not update firewall stealth mode"), err
		}
		return resource.applyResult("set", true, firewallStateMessage("updated firewall stealth mode to", resource.enabled)), nil
	default:
		err := fmt.Errorf("cannot apply firewall stealth mode from state %s", status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource FirewallStealthModeResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource FirewallStealthModeResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource FirewallStealthModeResource) details() map[string]string {
	return map[string]string{
		"name":    "firewall_stealth_mode",
		"enabled": fmt.Sprintf("%t", resource.enabled),
	}
}

func parseStealthModeEnabled(stdout string) (bool, bool) {
	status := strings.ToLower(stdout)
	switch {
	case strings.Contains(status, "stealth mode enabled"):
		return true, true
	case strings.Contains(status, "stealth mode disabled"):
		return false, true
	default:
		return false, false
	}
}

func firewallStateMessage(prefix string, enabled bool) string {
	if enabled {
		return prefix + " enabled"
	}
	return prefix + " disabled"
}

func onOff(enabled bool) string {
	if enabled {
		return "on"
	}
	return "off"
}
