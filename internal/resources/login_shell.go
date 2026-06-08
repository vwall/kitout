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

const loginShellType = "login_shell"

// LoginShellResource ensures the current user's macOS login shell matches config.
type LoginShellResource struct {
	path           string
	addToEtcShells bool
	runner         platform.Runner
	system         loginShellSystem
	etcShellsPath  string
}

var _ engine.Resource = LoginShellResource{}

// NewLoginShell returns a resource for the current user's desired login shell.
func NewLoginShell(path string, addToEtcShells bool, runner platform.Runner) LoginShellResource {
	return newLoginShell(path, addToEtcShells, runner, osLoginShellSystem{}, "/etc/shells")
}

func newLoginShell(path string, addToEtcShells bool, runner platform.Runner, system loginShellSystem, etcShellsPath string) LoginShellResource {
	return LoginShellResource{
		path:           path,
		addToEtcShells: addToEtcShells,
		runner:         runner,
		system:         system,
		etcShellsPath:  etcShellsPath,
	}
}

func (resource LoginShellResource) ID() string {
	return loginShellType + ":" + resource.path
}

func (resource LoginShellResource) Type() string {
	return loginShellType
}

func (resource LoginShellResource) Status(ctx context.Context) (engine.StatusResult, error) {
	state, err := resource.inspect(ctx)
	if err != nil {
		return resource.status(engine.StateFailed, state.message, state), err
	}

	switch {
	case !state.shellExists:
		return resource.status(engine.StateMissing, "shell path is missing", state), nil
	case !state.listedInEtcShells && !resource.addToEtcShells:
		return resource.status(engine.StateFailed, "shell path is not listed in /etc/shells", state), nil
	case !state.listedInEtcShells:
		return resource.status(engine.StateMissing, "shell path is not listed in /etc/shells", state), nil
	case state.currentShell != state.resolvedPath:
		return resource.status(engine.StateChanged, "login shell differs", state), nil
	default:
		return resource.status(engine.StateSatisfied, "login shell is current", state), nil
	}
}

func (resource LoginShellResource) Apply(ctx context.Context) (engine.ApplyResult, error) {
	state, err := resource.inspect(ctx)
	if err != nil {
		return resource.applyResult("fail", false, state.message, state), err
	}

	if !state.shellExists {
		err := fmt.Errorf("login shell path %s does not exist", state.resolvedPath)
		return resource.applyResult("fail", false, "shell path is missing", state), err
	}

	changed := false
	if !state.listedInEtcShells {
		if !resource.addToEtcShells {
			err := fmt.Errorf("login shell path %s is not listed in /etc/shells", state.resolvedPath)
			return resource.applyResult("fail", false, "shell path is not listed in /etc/shells", state), err
		}
		if err := resource.appendEtcShells(ctx, state.resolvedPath); err != nil {
			return resource.applyResult("append_etc_shells", false, "could not update /etc/shells", state), err
		}
		changed = true
		state.listedInEtcShells = true
	}

	if state.currentShell != state.resolvedPath {
		if _, err := resource.runner.Run(ctx, "chsh", "-s", state.resolvedPath); err != nil {
			return resource.applyResult("chsh", changed, "could not set login shell", state), err
		}
		changed = true
		state.currentShell = state.resolvedPath
	}

	if !changed {
		return resource.applyResult("noop", false, "login shell already current", state), nil
	}
	return resource.applyResult("set", true, "updated login shell", state), nil
}

func (resource LoginShellResource) inspect(ctx context.Context) (loginShellState, error) {
	state := loginShellState{
		etcShellsPath: resource.etcShellsPath,
	}

	if err := resource.validate(); err != nil {
		state.message = err.Error()
		return state, err
	}

	resolved, err := resource.resolvePath(ctx)
	state.resolvedPath = resolved
	if err != nil {
		state.message = "could not resolve login shell path"
		return state, err
	}

	exists, err := resource.system.fileExists(resolved)
	if err != nil {
		state.message = "could not inspect login shell path"
		return state, err
	}
	state.shellExists = exists
	if !exists {
		return state, nil
	}

	contents, err := resource.system.readFile(resource.etcShellsPath)
	if err != nil {
		state.message = "could not read /etc/shells"
		return state, err
	}
	state.listedInEtcShells = lineSetContains(string(contents), resolved)

	userResult, err := resource.runner.Run(ctx, "id", "-un")
	if err != nil {
		state.message = "could not determine current user"
		return state, err
	}
	state.user = strings.TrimSpace(userResult.Stdout)
	if state.user == "" {
		err := errors.New("current user name is empty")
		state.message = err.Error()
		return state, err
	}

	dsclPath := "/Users/" + state.user
	shellResult, err := resource.runner.Run(ctx, "dscl", ".", "-read", dsclPath, "UserShell")
	if err != nil {
		state.message = "could not read current login shell"
		return state, err
	}
	currentShell, err := parseDSCLUserShell(shellResult.Stdout)
	if err != nil {
		state.message = err.Error()
		return state, err
	}
	state.currentShell = currentShell

	return state, nil
}

func (resource LoginShellResource) validate() error {
	if strings.TrimSpace(resource.path) == "" {
		return errors.New("login shell path is required")
	}
	if resource.runner == nil {
		return errors.New("command runner is required")
	}
	if resource.system == nil {
		return errors.New("login shell system is required")
	}
	if resource.etcShellsPath == "" {
		return errors.New("/etc/shells path is required")
	}
	return nil
}

func (resource LoginShellResource) resolvePath(ctx context.Context) (string, error) {
	path := strings.TrimSpace(resource.path)
	binary, ok := strings.CutPrefix(path, "homebrew:")
	if !ok {
		return path, nil
	}

	result, err := resource.runner.Run(ctx, "brew", "--prefix")
	if err != nil {
		return "", err
	}
	prefix := strings.TrimSpace(result.Stdout)
	if prefix == "" {
		return "", errors.New("brew --prefix returned an empty path")
	}
	return strings.TrimRight(prefix, "/") + "/bin/" + binary, nil
}

func (resource LoginShellResource) appendEtcShells(ctx context.Context, path string) error {
	_, err := resource.runner.Run(ctx, "sudo", "sh", "-c", "printf '%s\\n' \"$1\" >> \"$2\"", "kitout", path, resource.etcShellsPath)
	return err
}

func (resource LoginShellResource) status(state engine.ResourceState, message string, inspected loginShellState) engine.StatusResult {
	return statusResult(resource.ID(), resource.Type(), state, message, resource.details(inspected))
}

func (resource LoginShellResource) applyResult(action string, changed bool, message string, inspected loginShellState) engine.ApplyResult {
	return applyResult(resource.ID(), resource.Type(), action, changed, message, resource.details(inspected))
}

func (resource LoginShellResource) details(inspected loginShellState) map[string]string {
	details := map[string]string{
		"path":              resource.path,
		"add_to_etc_shells": fmt.Sprintf("%t", resource.addToEtcShells),
	}
	if inspected.resolvedPath != "" {
		details["resolved_path"] = inspected.resolvedPath
	}
	if inspected.currentShell != "" {
		details["current_shell"] = inspected.currentShell
	}
	if inspected.user != "" {
		details["user"] = inspected.user
	}
	if inspected.etcShellsPath != "" {
		details["etc_shells_path"] = inspected.etcShellsPath
	}
	if inspected.resolvedPath != "" {
		details["shell_exists"] = fmt.Sprintf("%t", inspected.shellExists)
		details["listed_in_etc_shells"] = fmt.Sprintf("%t", inspected.listedInEtcShells)
	}
	return details
}

type loginShellState struct {
	resolvedPath      string
	etcShellsPath     string
	shellExists       bool
	listedInEtcShells bool
	user              string
	currentShell      string
	message           string
}

type loginShellSystem interface {
	fileExists(path string) (bool, error)
	readFile(path string) ([]byte, error)
}

type osLoginShellSystem struct{}

func (osLoginShellSystem) fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return !info.IsDir() && info.Mode()&0o111 != 0, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (osLoginShellSystem) readFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func lineSetContains(contents, value string) bool {
	for _, line := range strings.Split(contents, "\n") {
		if strings.TrimSpace(line) == value {
			return true
		}
	}
	return false
}

func parseDSCLUserShell(stdout string) (string, error) {
	for _, line := range strings.Split(stdout, "\n") {
		value, ok := strings.CutPrefix(strings.TrimSpace(line), "UserShell:")
		if ok {
			value = strings.TrimSpace(value)
			if value == "" {
				return "", errors.New("current login shell is empty")
			}
			return value, nil
		}
	}
	return "", errors.New("could not parse current login shell")
}
