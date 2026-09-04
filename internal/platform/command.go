package platform

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Runner executes external commands for resources and platform checks.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

// CommandResult captures the observable output from an external command.
type CommandResult struct {
	Name            string
	Args            []string
	Stdout          string
	Stderr          string
	ExitCode        int
	StdoutTruncated bool
	StderrTruncated bool
}

// ExecRunner runs commands through os/exec.
type ExecRunner struct {
	stdout                io.Writer
	stderr                io.Writer
	renderCommands        bool
	boundedOutput         bool
	trustCommandPath      bool
	preserveUserPath      bool
	waitDelay             time.Duration
	processGroupIsolation *bool
}

const defaultCommandWaitDelay = time.Second

type commandCancellationState struct {
	mu             sync.Mutex
	err            error
	afterStart     func() error
	finish         func() error
	processGroupID int
}

func (state *commandCancellationState) capture(err error) error {
	if err == nil || err == os.ErrProcessDone {
		return err
	}
	state.mu.Lock()
	if state.err == nil {
		state.err = err
	} else {
		state.err = errors.Join(state.err, err)
	}
	state.mu.Unlock()
	return err
}

func (state *commandCancellationState) load() error {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.err
}

func (state *commandCancellationState) finishCommand() {
	if state.finish != nil {
		state.capture(state.finish())
	}
}

func (state *commandCancellationState) commandStarted() error {
	if state.afterStart == nil {
		return nil
	}
	return state.afterStart()
}

func (state *commandCancellationState) ownedProcessGroupID(process *os.Process) int {
	if state.processGroupID != 0 {
		return state.processGroupID
	}
	if process == nil {
		return 0
	}
	return process.Pid
}

var trustedCommandDirectories = []string{
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
	"/opt/homebrew/bin",
	"/usr/local/bin",
}

var trustedAbsoluteCommandDirectories = []string{
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
	"/usr/libexec",
	"/opt/homebrew/bin",
	"/usr/local/bin",
}

const pathEnvironmentPrefix = "PATH="

// NewExecRunner returns the default external command runner.
func NewExecRunner() ExecRunner {
	return ExecRunner{}
}

// NewVerboseExecRunner returns a runner that captures command output while also
// streaming subprocess output to the provided writers.
func NewVerboseExecRunner(stdout, stderr io.Writer) ExecRunner {
	return ExecRunner{
		stdout:         stdout,
		stderr:         stderr,
		renderCommands: true,
	}
}

// WithTrustedCommandPath returns a runner variant that resolves top-level
// commands from trusted system locations and hides the caller's PATH from child
// processes unless WithUserPath is also applied.
func WithTrustedCommandPath(runner Runner) Runner {
	switch execRunner := runner.(type) {
	case ExecRunner:
		execRunner.trustCommandPath = true
		return execRunner
	case *ExecRunner:
		copy := *execRunner
		copy.trustCommandPath = true
		return copy
	default:
		return runner
	}
}

// WithUserPath returns a runner variant that preserves the caller's PATH for
// explicitly approved shell commands.
func WithUserPath(runner Runner) Runner {
	switch execRunner := runner.(type) {
	case ExecRunner:
		execRunner.preserveUserPath = true
		return execRunner
	case *ExecRunner:
		copy := *execRunner
		copy.preserveUserPath = true
		return copy
	default:
		return runner
	}
}

// Run executes a command and captures stdout, stderr, and exit status.
func (runner ExecRunner) Run(ctx context.Context, name string, args ...string) (CommandResult, error) {
	result := CommandResult{
		Name:     name,
		Args:     append([]string(nil), args...),
		ExitCode: -1,
	}

	if name == "" {
		return result, errors.New("command name is required")
	}

	if runner.renderCommands && runner.stdout != nil {
		fmt.Fprintf(runner.stdout, "$ %s\n", renderCommand(name, args))
	}

	executable := name
	if runner.trustCommandPath {
		trustedPath, err := resolveTrustedCommandPath(name)
		if err != nil {
			return result, CommandError{Result: result, Err: err}
		}
		executable = trustedPath
	}

	cmd := exec.CommandContext(ctx, executable, args...)
	waitDelay := runner.waitDelay
	if waitDelay <= 0 {
		waitDelay = defaultCommandWaitDelay
	}
	isolateProcessGroup := !processHasControllingTerminal()
	if runner.processGroupIsolation != nil {
		isolateProcessGroup = *runner.processGroupIsolation
	}
	cancellationState := configureCommandCancellation(cmd, waitDelay, isolateProcessGroup)
	originalCancel := cmd.Cancel
	var cancelOnce sync.Once
	var cancelErr error
	cmd.Cancel = func() error {
		cancelOnce.Do(func() {
			if originalCancel != nil {
				cancelErr = originalCancel()
			}
		})
		return cancelErr
	}
	if runner.trustCommandPath && !runner.preserveUserPath {
		cmd.Env = trustedCommandEnvironment()
	}
	stdout := commandOutputCapture{bounded: runner.boundedOutput}
	stderr := commandOutputCapture{bounded: runner.boundedOutput}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if runner.stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, runner.stdout)
	}
	if runner.stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, runner.stderr)
	}

	err := cmd.Start()
	if err == nil {
		if startErr := cancellationState.commandStarted(); startErr != nil {
			_ = cmd.Cancel()
			err = errors.Join(startErr, cmd.Wait())
		} else {
			waitDone := make(chan struct{})
			watcherDone := make(chan struct{})
			go func() {
				defer close(watcherDone)
				select {
				case <-ctx.Done():
					_ = cmd.Cancel()
				case <-waitDone:
				}
			}()
			err = cmd.Wait()
			close(waitDone)
			<-watcherDone
		}
	}
	cancellationState.finishCommand()
	if ctxErr := ctx.Err(); ctxErr != nil && !errors.Is(err, ctxErr) {
		err = errors.Join(err, ctxErr)
	} else if ctxErr == nil && commandProcessWasInterrupted(cmd.ProcessState) {
		err = errors.Join(err, context.Canceled)
	}
	if cancellationErr := cancellationState.load(); cancellationErr != nil && !errors.Is(err, cancellationErr) {
		err = errors.Join(err, fmt.Errorf("command cancellation cleanup: %w", cancellationErr))
	}
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()
	result.StdoutTruncated = stdout.truncated
	result.StderrTruncated = stderr.truncated

	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}

	if err != nil {
		return result, CommandError{Result: result, Err: err}
	}

	return result, nil
}

func resolveTrustedCommandPath(name string) (string, error) {
	if filepath.Base(name) != name {
		if !filepath.IsAbs(name) {
			return "", fmt.Errorf("command path %q must be absolute: %w", name, exec.ErrNotFound)
		}
		if !isTrustedAbsoluteCommandPath(name) {
			return "", fmt.Errorf("command path %q is outside trusted command paths: %w", name, exec.ErrNotFound)
		}
		if err := requireExecutableCommand(name); err != nil {
			return "", err
		}
		return filepath.Clean(name), nil
	}

	for _, dir := range trustedCommandDirectories {
		path := filepath.Join(dir, name)
		if err := requireExecutableCommand(path); err == nil {
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) && !errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("could not inspect trusted command %s: %w", path, err)
		}
	}

	return "", fmt.Errorf("%q not found in trusted command paths: %w", name, exec.ErrNotFound)
}

func isTrustedAbsoluteCommandPath(path string) bool {
	cleanPath := filepath.Clean(path)
	for _, dir := range trustedAbsoluteCommandDirectories {
		cleanDir := filepath.Clean(dir)
		rel, err := filepath.Rel(cleanDir, cleanPath)
		if err != nil {
			continue
		}
		if rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

func requireExecutableCommand(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("command path %q is not executable: %w", path, exec.ErrNotFound)
	}
	return nil
}

func trustedCommandEnvironment() []string {
	return trustedCommandEnvironmentFrom(os.Environ())
}

func trustedCommandEnvironmentFrom(env []string) []string {
	filtered := make([]string, 0, len(env)+1)
	for _, value := range env {
		if strings.HasPrefix(value, pathEnvironmentPrefix) {
			continue
		}
		filtered = append(filtered, value)
	}
	return append(filtered, pathEnvironmentPrefix+trustedCommandPath())
}

func trustedCommandPath() string {
	return strings.Join(trustedCommandDirectories, string(os.PathListSeparator))
}

func renderCommand(name string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, shellQuote(name))
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}
	for _, char := range arg {
		if !shellSafe(char) {
			return "'" + strings.ReplaceAll(arg, "'", "'\\''") + "'"
		}
	}
	return arg
}

func shellSafe(char rune) bool {
	return char >= 'a' && char <= 'z' ||
		char >= 'A' && char <= 'Z' ||
		char >= '0' && char <= '9' ||
		strings.ContainsRune("@%_+=:,./-", char)
}

// CommandError wraps a failed external command while preserving its result.
type CommandError struct {
	Result CommandResult
	Err    error
}

func (err CommandError) Error() string {
	message := err.message()
	if output := commandOutputSummary(err.Result); output != "" {
		return message + "\n" + output
	}
	return message
}

func (err CommandError) message() string {
	command := renderCommand(err.Result.Name, err.Result.Args)
	var exitError *exec.ExitError
	if err.Result.ExitCode >= 0 && errors.As(err.Err, &exitError) {
		return fmt.Sprintf("%s exited with status %d", command, err.Result.ExitCode)
	}
	if err.Result.ExitCode >= 0 {
		return fmt.Sprintf("%s failed after exit status %d: %v", command, err.Result.ExitCode, err.Err)
	}

	return fmt.Sprintf("%s failed: %v", command, err.Err)
}

func (err CommandError) Unwrap() error {
	return err.Err
}

const (
	commandOutputMaxLines      = 10
	commandOutputMaxLineLength = 240
)

func commandOutputSummary(result CommandResult) string {
	sections := make([]string, 0, 2)
	if section := commandOutputSection("stderr", result.Stderr, result.StderrTruncated); section != "" {
		sections = append(sections, section)
	}
	if section := commandOutputSection("stdout", result.Stdout, result.StdoutTruncated); section != "" {
		sections = append(sections, section)
	}
	return strings.Join(sections, "\n")
}

func commandOutputSection(label, output string, truncated bool) string {
	lines := make([]string, 0, commandOutputMaxLines)
	remaining := output
	for len(remaining) > 0 && len(lines) < commandOutputMaxLines {
		boundary := strings.LastIndexByte(remaining, '\n')
		line := strings.TrimRight(remaining[boundary+1:], "\r")
		if boundary < 0 {
			remaining = ""
		} else {
			remaining = remaining[:boundary]
		}
		if strings.TrimSpace(line) != "" {
			lines = append(lines, truncateCommandOutputLine(line))
		}
	}
	if len(lines) == 0 && !truncated {
		return ""
	}
	var section strings.Builder
	section.WriteString(label + ":\n")
	if truncated {
		section.WriteString("... output truncated; showing captured tail\n")
	}
	if len(remaining) > 0 {
		section.WriteString("... earlier lines omitted\n")
	}
	for i := len(lines) - 1; i >= 0; i-- {
		section.WriteString(lines[i])
		if i > 0 {
			section.WriteByte('\n')
		}
	}
	return strings.TrimSuffix(section.String(), "\n")
}

func truncateCommandOutputLine(line string) string {
	count := 0
	for offset := range line {
		if count == commandOutputMaxLineLength {
			return line[:offset] + "..."
		}
		count++
	}
	return line
}
