package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Runner executes external commands for resources and platform checks.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (CommandResult, error)
}

// CommandResult captures the observable output from an external command.
type CommandResult struct {
	Name     string
	Args     []string
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecRunner runs commands through os/exec.
type ExecRunner struct {
	stdout           io.Writer
	stderr           io.Writer
	renderCommands   bool
	trustCommandPath bool
	preserveUserPath bool
}

var trustedCommandDirectories = []string{
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
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
	if runner.trustCommandPath && !runner.preserveUserPath {
		cmd.Env = trustedCommandEnvironment()
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if runner.stdout != nil {
		cmd.Stdout = io.MultiWriter(&stdout, runner.stdout)
	}
	if runner.stderr != nil {
		cmd.Stderr = io.MultiWriter(&stderr, runner.stderr)
	}

	err := cmd.Run()
	result.Stdout = stdout.String()
	result.Stderr = stderr.String()

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
		return name, nil
	}

	for _, dir := range trustedCommandDirectories {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err == nil {
			if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
				continue
			}
			return path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("could not inspect trusted command %s: %w", path, err)
		}
	}

	return "", fmt.Errorf("%q not found in trusted command paths: %w", name, exec.ErrNotFound)
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
	if section := commandOutputSection("stderr", result.Stderr); section != "" {
		sections = append(sections, section)
	}
	if section := commandOutputSection("stdout", result.Stdout); section != "" {
		sections = append(sections, section)
	}
	return strings.Join(sections, "\n")
}

func commandOutputSection(label, output string) string {
	lines := commandOutputLines(output)
	if len(lines) == 0 {
		return ""
	}

	omitted := len(lines) - commandOutputMaxLines
	if omitted > 0 {
		lines = append([]string{fmt.Sprintf("... %d earlier lines omitted", omitted)}, lines[omitted:]...)
	}

	return label + ":\n" + strings.Join(lines, "\n")
}

func commandOutputLines(output string) []string {
	output = strings.TrimRight(output, "\r\n")
	if strings.TrimSpace(output) == "" {
		return nil
	}

	rawLines := strings.Split(output, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, truncateCommandOutputLine(line))
	}
	return lines
}

func truncateCommandOutputLine(line string) string {
	runes := []rune(line)
	if len(runes) <= commandOutputMaxLineLength {
		return line
	}
	return string(runes[:commandOutputMaxLineLength]) + "..."
}
