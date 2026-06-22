package resources

import (
	"context"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

// Build converts a validated config into executable resources in stable order.
func Build(cfg config.Config, runner platform.Runner) []engine.Resource {
	return build(cfg, runner, true)
}

// BuildUncached converts a validated config into resources that inspect live state
// independently. Use it for apply execution so planning caches cannot go stale.
func BuildUncached(cfg config.Config, runner platform.Runner) []engine.Resource {
	return build(cfg, runner, false)
}

func build(cfg config.Config, runner platform.Runner, batchHomebrew bool) []engine.Resource {
	resources := make([]engine.Resource, 0, resourceCount(cfg))
	brewOutdated := newBrewOutdatedCache(runner)
	var brewTapInstalled brewTapInstalledChecker = newDirectBrewTapInstalledChecker(runner)
	var brewInstalled brewInstalledChecker = newDirectBrewInstalledChecker(runner)
	var caskInstalled caskInstalledChecker = newDirectCaskInstalledChecker(runner)
	if batchHomebrew {
		brewTapInstalled = newBrewTapInstalledCache(runner)
		brewInstalled = newBrewInstalledCache(runner)
		caskInstalled = newCaskInstalledCache(runner)
	}

	for _, name := range cfg.Brew.Taps {
		resources = append(resources, newBrewTap(name, runner, brewTapInstalled))
	}
	for _, name := range cfg.Brew.Packages {
		outdated := brewOutdated
		if !batchHomebrew {
			outdated = newBrewOutdatedCache(runner)
		}
		resources = append(resources, newBrewPackage(name, runner, brewInstalled, outdated))
	}
	for _, plugin := range cfg.ASDF.Plugins {
		resources = append(resources, NewASDFPluginWithOptions(
			plugin.Name,
			plugin.URL,
			plugin.Versions,
			ASDFPluginOptions{UpdateBeforeInstall: plugin.UpdateBeforeInstall},
			runner,
		))
	}
	for _, item := range cfg.ASDF.ToolVersions {
		resources = append(resources, NewASDFToolVersions(item.Path, item.Tools))
	}
	for _, name := range cfg.Casks {
		resources = append(resources, newCask(name, runner, caskInstalled))
	}
	for _, path := range cfg.Directories {
		resources = append(resources, NewDirectory(path))
	}
	for _, repo := range cfg.Repos {
		resources = append(resources, NewRepo(repo.Path, repo.URL, repo.Branch, runner))
	}
	for _, copy := range cfg.Copies {
		resources = append(resources, NewCopy(copy.Source, copy.Target, copy.Replace))
	}
	for _, symlink := range cfg.ExpandedSymlinks() {
		resources = append(resources, NewSymlink(symlink.Source, symlink.Target, symlink.Replace))
	}
	for _, item := range cfg.MacOSDefaults {
		resources = append(resources, NewMacOSDefault(item.Domain, item.Key, item.Type, item.Value, runner))
	}
	if cfg.LoginShell != nil {
		resources = append(resources, NewLoginShell(cfg.LoginShell.Path, cfg.LoginShell.AddToEtcShells, runner))
	}
	for _, command := range cfg.Shell {
		resources = append(resources, NewShellCommand(command.Name, command.Command, command.When, runner))
	}

	return resources
}

func resourceCount(cfg config.Config) int {
	return len(cfg.Brew.Taps) +
		len(cfg.Brew.Packages) +
		len(cfg.ASDF.Plugins) +
		len(cfg.ASDF.ToolVersions) +
		len(cfg.Casks) +
		len(cfg.Directories) +
		len(cfg.Repos) +
		len(cfg.Copies) +
		len(cfg.ExpandedSymlinks()) +
		len(cfg.MacOSDefaults) +
		configuredLoginShellCount(cfg) +
		len(cfg.Shell)
}

func configuredLoginShellCount(cfg config.Config) int {
	if cfg.LoginShell == nil {
		return 0
	}
	return 1
}

// UnsupportedResource reports known config sections that do not have apply support yet.
type UnsupportedResource struct {
	typ     string
	id      string
	message string
	details map[string]string
}

var _ engine.Resource = UnsupportedResource{}

// NewUnsupportedResource returns a resource that is visible in status but skipped.
func NewUnsupportedResource(typ, id, message string, details map[string]string) UnsupportedResource {
	copied := make(map[string]string, len(details))
	for key, value := range details {
		copied[key] = value
	}
	return UnsupportedResource{
		typ:     typ,
		id:      id,
		message: message,
		details: copied,
	}
}

func (resource UnsupportedResource) ID() string {
	return resource.id
}

func (resource UnsupportedResource) Type() string {
	return resource.typ
}

func (resource UnsupportedResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return statusResult(resource.ID(), resource.Type(), engine.StateFailed, "context canceled while checking resource", resource.details), err
	}

	return statusResult(resource.ID(), resource.Type(), engine.StateSkipped, resource.message, resource.details), nil
}

func (resource UnsupportedResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return applyResult(resource.ID(), resource.Type(), "fail", false, "context canceled while applying resource", resource.details), err
	}

	return applyResult(resource.ID(), resource.Type(), "skip", false, resource.message, resource.details), nil
}
