package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const caskType = "cask"

const (
	// HomebrewCaskOutdatedAdvisory identifies an installed cask with an available update.
	HomebrewCaskOutdatedAdvisory = "homebrew_cask_outdated"
	// HomebrewCaskUpdateCheckFailedAdvisory identifies a failed cask update check.
	HomebrewCaskUpdateCheckFailedAdvisory = "homebrew_cask_update_check_failed"
)

// CaskResource ensures a Homebrew cask is installed.
type CaskResource struct {
	name      string
	runner    platform.Runner
	installed caskInstalledChecker
	outdated  caskOutdatedChecker
}

var _ engine.Resource = CaskResource{}

// NewCask returns a resource for one Homebrew cask.
func NewCask(name string, runner platform.Runner) CaskResource {
	return newCask(name, runner, newDirectCaskInstalledChecker(runner), newCaskOutdatedCache(runner))
}

func newCask(name string, runner platform.Runner, installed caskInstalledChecker, outdated caskOutdatedChecker) CaskResource {
	return CaskResource{name: name, runner: runner, installed: installed, outdated: outdated}
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

	installed, err := resource.installed.Contains(ctx, resource.name)
	if err == nil && installed {
		advisories := resource.updateAdvisories(ctx)
		return resource.statusWithAdvisories(engine.StateSatisfied, "cask is installed", advisories), nil
	}
	if err == nil || isExitCode(err, 1) {
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

// Upgrade updates an installed Homebrew cask when Homebrew reports an available
// update. Missing casks are left for apply.
func (resource CaskResource) Upgrade(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		if advisory, ok := advisoryWithCode(status.Advisories, HomebrewCaskUpdateCheckFailedAdvisory); ok {
			err := fmt.Errorf("could not inspect cask updates for %s", resource.name)
			if detail := advisory.Details["error"]; detail != "" {
				err = fmt.Errorf("%s: %s", err.Error(), detail)
			}
			return resource.applyResult("fail", false, advisory.Message), err
		}
		if _, ok := advisoryWithCode(status.Advisories, HomebrewCaskOutdatedAdvisory); !ok {
			return resource.applyResult("noop", false, "cask already current"), nil
		}
		if _, err := resource.runner.Run(ctx, "brew", "upgrade", "--cask", resource.name); err != nil {
			return resource.applyResult("upgrade", false, "could not upgrade cask"), err
		}
		return resource.applyResult("upgrade", true, "upgraded cask"), nil
	case engine.StateMissing:
		return resource.applyResult("skip", false, "cask is missing; run `kitout apply` first"), nil
	default:
		err := fmt.Errorf("cannot upgrade cask %s from state %s", resource.name, status.State)
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
	if resource.installed == nil {
		return errors.New("cask installed checker is required")
	}
	if resource.outdated == nil {
		return errors.New("cask outdated checker is required")
	}
	return nil
}

func (resource CaskResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource CaskResource) statusWithAdvisories(state engine.ResourceState, message string, advisories []engine.Advisory) engine.StatusResult {
	return statusResultWithAdvisories(resource.ID(), resource.Type(), state, message, resource.details(), advisories)
}

func (resource CaskResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource CaskResource) details() map[string]string {
	return map[string]string{"name": resource.name}
}

func (resource CaskResource) updateAdvisories(ctx context.Context) []engine.Advisory {
	isOutdated, err := resource.outdated.Contains(ctx, resource.name)
	if err != nil {
		return []engine.Advisory{{
			Code:     HomebrewCaskUpdateCheckFailedAdvisory,
			Severity: engine.AdvisoryWarning,
			Message:  fmt.Sprintf("could not inspect cask updates for %s", resource.name),
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
		Code:     HomebrewCaskOutdatedAdvisory,
		Severity: engine.AdvisoryNotice,
		Message:  fmt.Sprintf("cask update available for %s", resource.name),
		Fix:      fmt.Sprintf("Run `brew upgrade --cask %s` when you want to update it.", resource.name),
		Details:  map[string]string{"name": resource.name},
	}}
}

type caskInstalledChecker interface {
	Contains(ctx context.Context, name string) (bool, error)
}

type directCaskInstalledChecker struct {
	runner platform.Runner
}

func newDirectCaskInstalledChecker(runner platform.Runner) directCaskInstalledChecker {
	return directCaskInstalledChecker{runner: runner}
}

func (checker directCaskInstalledChecker) Contains(ctx context.Context, name string) (bool, error) {
	_, err := checker.runner.Run(ctx, "brew", "list", "--cask", name)
	if err == nil {
		return true, nil
	}
	if isExitCode(err, 1) {
		return false, nil
	}
	return false, err
}

type caskInstalledCache struct {
	runner  platform.Runner
	loaded  bool
	names   map[string]struct{}
	loadErr error
}

func newCaskInstalledCache(runner platform.Runner) *caskInstalledCache {
	return &caskInstalledCache{runner: runner}
}

func (cache *caskInstalledCache) Contains(ctx context.Context, name string) (bool, error) {
	if !cache.loaded {
		cache.load(ctx)
	}

	_, ok := cache.names[name]
	return ok, cache.loadErr
}

func (cache *caskInstalledCache) load(ctx context.Context) {
	cache.loaded = true
	cache.names = make(map[string]struct{})

	result, err := cache.runner.Run(ctx, "brew", "list", "--cask", "--quiet")
	for _, field := range strings.Fields(result.Stdout) {
		cache.names[field] = struct{}{}
	}
	if err != nil {
		cache.loadErr = err
	}
}

type caskOutdatedCache struct {
	runner  platform.Runner
	loaded  bool
	names   map[string]struct{}
	loadErr error
}

type caskOutdatedChecker interface {
	Contains(ctx context.Context, name string) (bool, error)
}

func newCaskOutdatedCache(runner platform.Runner) *caskOutdatedCache {
	return &caskOutdatedCache{runner: runner}
}

func (cache *caskOutdatedCache) Contains(ctx context.Context, name string) (bool, error) {
	if !cache.loaded {
		cache.load(ctx)
	}

	_, ok := cache.names[name]
	return ok, cache.loadErr
}

func (cache *caskOutdatedCache) load(ctx context.Context) {
	cache.loaded = true
	cache.names = make(map[string]struct{})

	result, err := cache.runner.Run(ctx, "brew", "outdated", "--cask", "--quiet")
	for _, field := range strings.Fields(result.Stdout) {
		cache.names[field] = struct{}{}
	}
	if err != nil && !isExitCode(err, 1) {
		cache.loadErr = err
	}
}
