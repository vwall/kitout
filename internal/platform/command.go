package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
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
	stdout         io.Writer
	stderr         io.Writer
	renderCommands bool
}

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

	cmd := exec.CommandContext(ctx, name, args...)
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
	if err.Result.ExitCode >= 0 {
		return fmt.Sprintf("%s exited with status %d: %v", err.Result.Name, err.Result.ExitCode, err.Err)
	}

	return fmt.Sprintf("%s failed: %v", err.Result.Name, err.Err)
}

func (err CommandError) Unwrap() error {
	return err.Err
}
