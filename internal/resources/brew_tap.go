package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const brewTapType = "brew_tap"

// BrewTapResource ensures a Homebrew tap is available.
type BrewTapResource struct {
	name      string
	runner    platform.Runner
	installed brewTapInstalledChecker
}

var _ engine.Resource = BrewTapResource{}

// NewBrewTap returns a resource for one Homebrew tap.
func NewBrewTap(name string, runner platform.Runner) BrewTapResource {
	return newBrewTap(name, runner, newDirectBrewTapInstalledChecker(runner))
}

func newBrewTap(name string, runner platform.Runner, installed brewTapInstalledChecker) BrewTapResource {
	return BrewTapResource{name: name, runner: runner, installed: installed}
}

func (resource BrewTapResource) ID() string {
	return brewTapType + ":" + resource.name
}

func (resource BrewTapResource) Type() string {
	return brewTapType
}

func (resource BrewTapResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	installed, err := resource.installed.Contains(ctx, resource.name)
	if err == nil && installed {
		return resource.status(engine.StateSatisfied, "tap is installed"), nil
	}
	if err == nil {
		return resource.status(engine.StateMissing, "tap is missing"), nil
	}

	return resource.status(engine.StateFailed, "could not inspect tap"), err
}

func (resource BrewTapResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}

	switch status.State {
	case engine.StateSatisfied:
		return resource.applyResult("noop", false, "tap already installed"), nil
	case engine.StateMissing:
		if _, err := resource.runner.Run(ctx, "brew", "tap", resource.name); err != nil {
			return resource.applyResult("tap", false, "could not add tap"), err
		}
		return resource.applyResult("tap", true, "added tap"), nil
	default:
		err := fmt.Errorf("cannot apply tap %s from state %s", resource.name, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}
}

func (resource BrewTapResource) validate() error {
	if resource.name == "" {
		return errors.New("brew tap name is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	if resource.installed == nil {
		return errors.New("brew tap installed checker is required")
	}
	return nil
}

func (resource BrewTapResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource BrewTapResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource BrewTapResource) details() map[string]string {
	return map[string]string{"name": resource.name}
}

type brewTapInstalledChecker interface {
	Contains(ctx context.Context, name string) (bool, error)
}

type directBrewTapInstalledChecker struct {
	runner platform.Runner
}

func newDirectBrewTapInstalledChecker(runner platform.Runner) directBrewTapInstalledChecker {
	return directBrewTapInstalledChecker{runner: runner}
}

func (checker directBrewTapInstalledChecker) Contains(ctx context.Context, name string) (bool, error) {
	result, err := checker.runner.Run(ctx, "brew", "tap")
	if err != nil {
		return false, err
	}
	return containsBrewTap(result.Stdout, name), nil
}

type brewTapInstalledCache struct {
	runner  platform.Runner
	loaded  bool
	names   map[string]struct{}
	loadErr error
}

func newBrewTapInstalledCache(runner platform.Runner) *brewTapInstalledCache {
	return &brewTapInstalledCache{runner: runner}
}

func (cache *brewTapInstalledCache) Contains(ctx context.Context, name string) (bool, error) {
	if !cache.loaded {
		cache.load(ctx)
	}

	_, ok := cache.names[name]
	return ok, cache.loadErr
}

func (cache *brewTapInstalledCache) load(ctx context.Context) {
	cache.loaded = true
	cache.names = make(map[string]struct{})

	result, err := cache.runner.Run(ctx, "brew", "tap")
	for _, field := range strings.Fields(result.Stdout) {
		cache.names[field] = struct{}{}
	}
	if err != nil {
		cache.loadErr = err
	}
}

func containsBrewTap(output, name string) bool {
	for _, field := range strings.Fields(output) {
		if field == name {
			return true
		}
	}
	return false
}
