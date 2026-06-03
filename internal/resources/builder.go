package resources

import (
	"context"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

// Build converts a validated config into executable resources in stable order.
func Build(cfg config.Config, runner platform.Runner) []engine.Resource {
	resources := make([]engine.Resource, 0, resourceCount(cfg))

	for _, name := range cfg.Brew.Packages {
		resources = append(resources, NewBrewPackage(name, runner))
	}
	for _, plugin := range cfg.ASDF.Plugins {
		resources = append(resources, NewASDFPlugin(plugin.Name, plugin.URL, plugin.Versions, runner))
	}
	for _, item := range cfg.ASDF.ToolVersions {
		resources = append(resources, NewASDFToolVersions(item.Path, item.Tools))
	}
	for _, name := range cfg.Casks {
		resources = append(resources, NewCask(name, runner))
	}
	for _, path := range cfg.Directories {
		resources = append(resources, NewDirectory(path))
	}
	for _, repo := range cfg.Repos {
		resources = append(resources, NewRepo(repo.Path, repo.URL, repo.Branch, runner))
	}
	for _, symlink := range cfg.Symlinks {
		resources = append(resources, NewSymlink(symlink.Source, symlink.Target, symlink.Replace))
	}
	for _, item := range cfg.MacOSDefaults {
		resources = append(resources, NewMacOSDefault(item.Domain, item.Key, item.Type, item.Value, runner))
	}
	for _, command := range cfg.Shell {
		resources = append(resources, NewShellCommand(command.Name, command.Command, command.When, runner))
	}

	return resources
}

func resourceCount(cfg config.Config) int {
	return len(cfg.Brew.Packages) +
		len(cfg.ASDF.Plugins) +
		len(cfg.ASDF.ToolVersions) +
		len(cfg.Casks) +
		len(cfg.Directories) +
		len(cfg.Repos) +
		len(cfg.Symlinks) +
		len(cfg.MacOSDefaults) +
		len(cfg.Shell)
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
