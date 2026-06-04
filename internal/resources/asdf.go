package resources

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

const (
	asdfPluginType       = "asdf_plugin"
	asdfToolVersionsType = "asdf_tool_versions"
)

// ASDFPluginResource ensures an asdf plugin and exact tool versions are installed.
type ASDFPluginResource struct {
	name                string
	url                 string
	updateBeforeInstall bool
	versions            []string
	runner              platform.Runner
}

var _ engine.Resource = ASDFPluginResource{}

// ASDFPluginOptions controls optional asdf plugin apply behavior.
type ASDFPluginOptions struct {
	UpdateBeforeInstall bool
}

// NewASDFPlugin returns a resource for one asdf plugin.
func NewASDFPlugin(name, url string, versions []string, runner platform.Runner) ASDFPluginResource {
	return NewASDFPluginWithOptions(name, url, versions, ASDFPluginOptions{}, runner)
}

// NewASDFPluginWithOptions returns a resource for one asdf plugin with options.
func NewASDFPluginWithOptions(name, url string, versions []string, options ASDFPluginOptions, runner platform.Runner) ASDFPluginResource {
	return ASDFPluginResource{
		name:                name,
		url:                 url,
		updateBeforeInstall: options.UpdateBeforeInstall,
		versions:            append([]string(nil), versions...),
		runner:              runner,
	}
}

func (resource ASDFPluginResource) ID() string {
	return asdfPluginType + ":" + resource.name
}

func (resource ASDFPluginResource) Type() string {
	return asdfPluginType
}

func (resource ASDFPluginResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}
	if err := resource.checkASDF(ctx); err != nil {
		return resource.status(engine.StateFailed, "asdf is required before checking plugins"), err
	}

	pluginURL, ok, err := resource.installedPluginURL(ctx)
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect asdf plugins"), err
	}
	if !ok {
		return resource.status(engine.StateMissing, "asdf plugin is missing"), nil
	}
	if pluginURL != resource.url {
		return resource.status(engine.StateChanged, "asdf plugin URL does not match config"), nil
	}

	missing, err := resource.missingVersions(ctx)
	if err != nil {
		return resource.status(engine.StateFailed, "could not inspect asdf versions"), err
	}
	if len(missing) > 0 {
		return resource.missingVersionStatus(missing), nil
	}

	return resource.status(engine.StateSatisfied, "asdf plugin and versions are installed"), nil
}

func (resource ASDFPluginResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	if err := resource.validate(); err != nil {
		return resource.applyResult("fail", false, err.Error()), err
	}
	if err := resource.checkASDF(ctx); err != nil {
		return resource.applyResult("fail", false, "asdf is required before applying plugins"), err
	}

	pluginURL, ok, err := resource.installedPluginURL(ctx)
	if err != nil {
		return resource.applyResult("fail", false, "could not inspect asdf plugins"), err
	}
	if ok && pluginURL != resource.url {
		err := fmt.Errorf("cannot replace asdf plugin %s: configured URL does not match installed URL", resource.name)
		return resource.applyResult("fail", false, err.Error()), err
	}

	changed := false
	pluginWasPresent := ok
	if !ok {
		if _, err := resource.runner.Run(ctx, "asdf", "plugin", "add", resource.name, resource.url); err != nil {
			return resource.applyResult("add", false, "could not add asdf plugin"), err
		}
		changed = true
	}

	missing, err := resource.missingVersions(ctx)
	if err != nil {
		return resource.applyResult("fail", changed, "could not inspect asdf versions"), err
	}
	updated := false
	if len(missing) > 0 && resource.updateBeforeInstall && pluginWasPresent {
		if _, err := resource.runner.Run(ctx, "asdf", "plugin", "update", resource.name); err != nil {
			return resource.applyResult("update", changed, "could not update asdf plugin"), err
		}
		changed = true
		updated = true
	}
	for _, version := range missing {
		result, err := resource.runner.Run(ctx, "asdf", "install", resource.name, version)
		if err != nil {
			message := asdfInstallFailureMessage(resource.name, version, result, err)
			return resource.applyResult("install", changed, message), err
		}
		changed = true
	}

	if !changed {
		return resource.applyResult("noop", false, "asdf plugin and versions already installed"), nil
	}
	if updated {
		return resource.applyResult("install", true, "updated asdf plugin and installed versions"), nil
	}
	return resource.applyResult("install", true, "installed asdf plugin or versions"), nil
}

func (resource ASDFPluginResource) validate() error {
	if resource.name == "" {
		return errors.New("asdf plugin name is required")
	}
	if resource.url == "" {
		return errors.New("asdf plugin URL is required")
	}
	for _, version := range resource.versions {
		if strings.TrimSpace(version) == "" {
			return errors.New("asdf version is required")
		}
		if strings.TrimSpace(version) == "latest" {
			return errors.New("asdf version must be exact, not latest")
		}
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	return nil
}

func (resource ASDFPluginResource) checkASDF(ctx context.Context) error {
	_, err := resource.runner.Run(ctx, "asdf", "--version")
	return err
}

func (resource ASDFPluginResource) installedPluginURL(ctx context.Context) (string, bool, error) {
	result, err := resource.runner.Run(ctx, "asdf", "plugin", "list", "--urls")
	if err != nil {
		return "", false, err
	}

	for _, line := range strings.Split(result.Stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == resource.name {
			return fields[1], true, nil
		}
	}

	return "", false, nil
}

func (resource ASDFPluginResource) missingVersions(ctx context.Context) ([]string, error) {
	result, err := resource.runner.Run(ctx, "asdf", "list", resource.name)
	if err != nil {
		if isExitCode(err, 1) || isExitCode(err, 2) {
			return append([]string(nil), resource.versions...), nil
		}
		return nil, err
	}

	installed := parseASDFVersionList(result.Stdout)
	missing := make([]string, 0)
	for _, version := range resource.versions {
		if !installed[version] {
			missing = append(missing, version)
		}
	}
	return missing, nil
}

func parseASDFVersionList(output string) map[string]bool {
	versions := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		value := strings.TrimSpace(line)
		value = strings.TrimPrefix(value, "*")
		value = strings.TrimSpace(value)
		if value != "" {
			versions[value] = true
		}
	}
	return versions
}

func asdfInstallFailureMessage(plugin, version string, result platform.CommandResult, err error) string {
	if isASDFVersionNotFound(result, err) {
		return fmt.Sprintf("asdf version %s %s was not found; run `asdf plugin update %s` and retry, or set `update_before_install: true` for this plugin", plugin, version, plugin)
	}
	return "could not install asdf version"
}

func isASDFVersionNotFound(result platform.CommandResult, err error) bool {
	var commandErr platform.CommandError
	if errors.As(err, &commandErr) {
		result = commandErr.Result
	}

	output := strings.ToLower(result.Stdout + "\n" + result.Stderr)
	return strings.Contains(output, "version not found")
}

func (resource ASDFPluginResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource ASDFPluginResource) missingVersionStatus(missing []string) engine.StatusResult {
	message := "asdf versions are missing"
	if len(missing) == 1 {
		message = "asdf version is missing"
	}
	details := resource.details()
	details["missing_versions"] = strings.Join(missing, ",")
	return statusResult(resource.ID(), resource.Type(), engine.StateMissing, message, details)
}

func (resource ASDFPluginResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource ASDFPluginResource) details() map[string]string {
	details := map[string]string{
		"name": resource.name,
		"url":  resource.url,
	}
	if len(resource.versions) > 0 {
		details["versions"] = strings.Join(resource.versions, ",")
	}
	if resource.updateBeforeInstall {
		details["update_before_install"] = "true"
	}
	return details
}

// ASDFToolVersionsResource ensures configured tool entries exist in one .tool-versions file.
type ASDFToolVersionsResource struct {
	path  string
	tools map[string]string
}

var _ engine.Resource = ASDFToolVersionsResource{}

// NewASDFToolVersions returns a resource for configured .tool-versions entries.
func NewASDFToolVersions(path string, tools map[string]string) ASDFToolVersionsResource {
	copied := make(map[string]string, len(tools))
	for tool, version := range tools {
		copied[tool] = version
	}
	return ASDFToolVersionsResource{path: path, tools: copied}
}

func (resource ASDFToolVersionsResource) ID() string {
	return asdfToolVersionsType + ":" + resource.path
}

func (resource ASDFToolVersionsResource) Type() string {
	return asdfToolVersionsType
}

func (resource ASDFToolVersionsResource) Status(ctx context.Context) (engine.StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return resource.status(engine.StateFailed, "context canceled while checking .tool-versions"), err
	}
	if err := resource.validate(); err != nil {
		return resource.status(engine.StateFailed, err.Error()), err
	}

	contents, err := os.ReadFile(resource.path)
	if errors.Is(err, os.ErrNotExist) {
		return resource.status(engine.StateMissing, ".tool-versions file is missing"), nil
	}
	if err != nil {
		return resource.status(engine.StateFailed, "could not read .tool-versions file"), err
	}

	current := parseToolVersions(string(contents))
	for tool, version := range resource.tools {
		currentVersion, ok := current[tool]
		if !ok {
			return resource.status(engine.StateMissing, ".tool-versions entry is missing"), nil
		}
		if currentVersion != version {
			return resource.status(engine.StateChanged, ".tool-versions entry differs"), nil
		}
	}

	return resource.status(engine.StateSatisfied, ".tool-versions entries are correct"), nil
}

func (resource ASDFToolVersionsResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	status, err := resource.Status(ctx)
	if err != nil {
		return resource.applyResult("fail", false, status.Message), err
	}
	if status.State == engine.StateSatisfied {
		return resource.applyResult("noop", false, ".tool-versions entries already correct"), nil
	}
	if status.State != engine.StateMissing && status.State != engine.StateChanged {
		err := fmt.Errorf("cannot apply .tool-versions %s from state %s", resource.path, status.State)
		return resource.applyResult("fail", false, err.Error()), err
	}

	if err := os.MkdirAll(filepath.Dir(resource.path), 0o755); err != nil {
		return resource.applyResult("write", false, "could not create .tool-versions parent directory"), err
	}

	contents, err := os.ReadFile(resource.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return resource.applyResult("write", false, "could not read .tool-versions file"), err
	}

	updated := updateToolVersions(string(contents), resource.tools)
	if err := os.WriteFile(resource.path, []byte(updated), 0o644); err != nil {
		return resource.applyResult("write", false, "could not write .tool-versions file"), err
	}

	return resource.applyResult("write", true, "updated .tool-versions entries"), nil
}

func (resource ASDFToolVersionsResource) validate() error {
	if resource.path == "" {
		return errors.New(".tool-versions path is required")
	}
	if len(resource.tools) == 0 {
		return errors.New(".tool-versions tools are required")
	}
	for tool, version := range resource.tools {
		if strings.TrimSpace(tool) == "" {
			return errors.New(".tool-versions tool name is required")
		}
		if strings.TrimSpace(version) == "" {
			return errors.New(".tool-versions version is required")
		}
		if strings.TrimSpace(version) == "latest" {
			return errors.New(".tool-versions version must be exact, not latest")
		}
	}
	return nil
}

func parseToolVersions(contents string) map[string]string {
	tools := make(map[string]string)
	for _, line := range strings.Split(contents, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 {
			tools[fields[0]] = fields[1]
		}
	}
	return tools
}

func updateToolVersions(contents string, desired map[string]string) string {
	remaining := make(map[string]string, len(desired))
	for tool, version := range desired {
		remaining[tool] = version
	}

	lines := strings.Split(contents, "\n")
	hadTrailingNewline := strings.HasSuffix(contents, "\n")
	updated := make([]string, 0, len(lines)+len(desired))

	for i, line := range lines {
		if contents == "" && i == 0 && line == "" {
			continue
		}
		if i == len(lines)-1 && line == "" && hadTrailingNewline {
			continue
		}

		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 1 && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			if version, ok := remaining[fields[0]]; ok {
				updated = append(updated, fields[0]+" "+version)
				delete(remaining, fields[0])
				continue
			}
		}
		updated = append(updated, line)
	}

	names := make([]string, 0, len(remaining))
	for tool := range remaining {
		names = append(names, tool)
	}
	sort.Strings(names)
	for _, tool := range names {
		updated = append(updated, tool+" "+remaining[tool])
	}

	return strings.Join(updated, "\n") + "\n"
}

func (resource ASDFToolVersionsResource) status(state engine.ResourceState, message string) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details())
}

func (resource ASDFToolVersionsResource) applyResult(action string, changed bool, message string) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details())
}

func (resource ASDFToolVersionsResource) details() map[string]string {
	details := map[string]string{"path": resource.path}
	names := make([]string, 0, len(resource.tools))
	for tool := range resource.tools {
		names = append(names, tool)
	}
	sort.Strings(names)
	for _, tool := range names {
		details["tool."+tool] = resource.tools[tool]
	}
	return details
}
