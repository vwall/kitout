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

const (
	// HomebrewFormulaOutdatedAdvisory identifies an installed formula with an available update.
	HomebrewFormulaOutdatedAdvisory = "homebrew_formula_outdated"
	// HomebrewFormulaUpdateCheckFailedAdvisory identifies a failed formula update check.
	HomebrewFormulaUpdateCheckFailedAdvisory = "homebrew_formula_update_check_failed"
)

// BrewPackageResource ensures a Homebrew formula is installed.
type BrewPackageResource struct {
	name      string
	runner    platform.Runner
	installed brewInstalledChecker
	outdated  brewOutdatedChecker
}

var _ engine.Resource = BrewPackageResource{}

// NewBrewPackage returns a resource for one Homebrew formula.
func NewBrewPackage(name string, runner platform.Runner) BrewPackageResource {
	return newBrewPackage(name, runner, newDirectBrewInstalledChecker(runner), newBrewOutdatedCache(runner))
}

func newBrewPackage(name string, runner platform.Runner, installed brewInstalledChecker, outdated brewOutdatedChecker) BrewPackageResource {
	return BrewPackageResource{name: name, runner: runner, installed: installed, outdated: outdated}
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

	installed, err := resource.installed.Contains(ctx, resource.name)
	if err == nil && installed {
		advisories := resource.updateAdvisories(ctx)
		return resource.statusWithAdvisories(engine.StateSatisfied, "formula is installed", advisories), nil
	}
	if err == nil {
		return resource.status(engine.StateMissing, "formula is missing"), nil
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
	default:
		err := fmt.Errorf("cannot apply formula %s from state %s", resource.name, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

// Upgrade updates an installed Homebrew formula when Homebrew reports an
// available update. Missing formulae are left for apply.
func (resource BrewPackageResource) Upgrade(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		if advisory, ok := advisoryWithCode(status.Advisories, HomebrewFormulaUpdateCheckFailedAdvisory); ok {
			err := fmt.Errorf("could not inspect formula updates for %s", resource.name)
			if detail := advisory.Details["error"]; detail != "" {
				err = fmt.Errorf("%s: %s", err.Error(), detail)
			}
			return resource.applyResult("fail", false, advisory.Message), err
		}
		if _, ok := advisoryWithCode(status.Advisories, HomebrewFormulaOutdatedAdvisory); !ok {
			return resource.applyResult("noop", false, "formula already current"), nil
		}
		if _, err := resource.runner.Run(ctx, "brew", "upgrade", resource.name); err != nil {
			return resource.applyResult("upgrade", false, "could not upgrade formula"), err
		}
		return resource.applyResult("upgrade", true, "upgraded formula"), nil
	case engine.StateMissing:
		return resource.applyResult("skip", false, "formula is missing; run `kitout apply` first"), nil
	default:
		err := fmt.Errorf("cannot upgrade formula %s from state %s", resource.name, status.State)
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
	if resource.installed == nil {
		return errors.New("brew installed checker is required")
	}
	if resource.outdated == nil {
		return errors.New("brew outdated checker is required")
	}
	return nil
}

func (resource BrewPackageResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource BrewPackageResource) statusWithAdvisories(state engine.ResourceState, message string, advisories []engine.Advisory) engine.StatusResult {
	return statusResultWithAdvisories(resource.ID(), resource.Type(), state, message, resource.details(), advisories)
}

func (resource BrewPackageResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource BrewPackageResource) details() map[string]string {
	return map[string]string{"name": resource.name}
}

func (resource BrewPackageResource) updateAdvisories(ctx context.Context) []engine.Advisory {
	isOutdated, err := resource.outdated.Contains(ctx, resource.name)
	if err != nil {
		return []engine.Advisory{{
			Code:     HomebrewFormulaUpdateCheckFailedAdvisory,
			Severity: engine.AdvisoryWarning,
			Message:  fmt.Sprintf("could not inspect formula updates for %s", resource.name),
			Fix:      "Run `brew doctor`, then try the update check again.",
			Details: map[string]string{
				"name":  resource.name,
				"error": err.Error(),
			},
		}}
	}
	if !isOutdated {
		return nil
	}

	return []engine.Advisory{{
		Code:     HomebrewFormulaOutdatedAdvisory,
		Severity: engine.AdvisoryNotice,
		Message:  fmt.Sprintf("formula update available for %s", resource.name),
		Fix:      fmt.Sprintf("Run `kitout upgrade %s` or `brew upgrade %s` when you want to update it.", resource.ID(), resource.name),
		Details:  map[string]string{"name": resource.name},
	}}
}

func isExitCode(err error, code int) bool {
	var commandError platform.CommandError
	return errors.As(err, &commandError) && commandError.Result.ExitCode == code
}

func advisoryWithCode(advisories []engine.Advisory, code string) (engine.Advisory, bool) {
	for _, advisory := range advisories {
		if advisory.Code == code {
			return advisory, true
		}
	}
	return engine.Advisory{}, false
}

type brewInstalledChecker interface {
	Contains(ctx context.Context, name string) (bool, error)
}

type directBrewInstalledChecker struct {
	runner platform.Runner
}

func newDirectBrewInstalledChecker(runner platform.Runner) directBrewInstalledChecker {
	return directBrewInstalledChecker{runner: runner}
}

func (checker directBrewInstalledChecker) Contains(ctx context.Context, name string) (bool, error) {
	_, err := checker.runner.Run(ctx, "brew", "list", "--formula", name)
	if err == nil {
		return true, nil
	}
	if isExitCode(err, 1) {
		return false, nil
	}
	return false, err
}

type brewInstalledCache struct {
	runner  platform.Runner
	loaded  bool
	names   map[string]struct{}
	loadErr error
}

func newBrewInstalledCache(runner platform.Runner) *brewInstalledCache {
	return &brewInstalledCache{runner: runner}
}

func (cache *brewInstalledCache) Contains(ctx context.Context, name string) (bool, error) {
	if !cache.loaded {
		cache.load(ctx)
	}

	_, ok := cache.names[name]
	return ok, cache.loadErr
}

func (cache *brewInstalledCache) load(ctx context.Context) {
	cache.loaded = true
	cache.names = make(map[string]struct{})

	result, err := cache.runner.Run(ctx, "brew", "list", "--formula", "--quiet")
	for _, field := range strings.Fields(result.Stdout) {
		cache.names[field] = struct{}{}
	}
	if err != nil {
		cache.loadErr = err
	}
}

type brewOutdatedCache struct {
	runner  platform.Runner
	loaded  bool
	names   map[string]struct{}
	loadErr error
}

type brewOutdatedChecker interface {
	Contains(ctx context.Context, name string) (bool, error)
}

func newBrewOutdatedCache(runner platform.Runner) *brewOutdatedCache {
	return &brewOutdatedCache{runner: runner}
}

func (cache *brewOutdatedCache) Contains(ctx context.Context, name string) (bool, error) {
	if !cache.loaded {
		cache.load(ctx)
	}

	_, ok := cache.names[name]
	return ok, cache.loadErr
}

func (cache *brewOutdatedCache) load(ctx context.Context) {
	cache.loaded = true
	cache.names = make(map[string]struct{})

	result, err := cache.runner.Run(ctx, "brew", "outdated", "--formula", "--quiet")
	for _, field := range strings.Fields(result.Stdout) {
		cache.names[field] = struct{}{}
	}
	if err != nil && !isExitCode(err, 1) {
		cache.loadErr = err
	}
}

type directBrewOutdatedChecker struct {
	runner platform.Runner
}

func newDirectBrewOutdatedChecker(runner platform.Runner) directBrewOutdatedChecker {
	return directBrewOutdatedChecker{runner: runner}
}

func (checker directBrewOutdatedChecker) Contains(ctx context.Context, name string) (bool, error) {
	result, err := checker.runner.Run(ctx, "brew", "outdated", "--formula", "--quiet", name)
	if strings.TrimSpace(result.Stdout) != "" {
		return true, nil
	}
	if err == nil || isExitCode(err, 1) {
		return false, nil
	}
	return false, err
}
