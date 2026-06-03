package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
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
type ExecRunner struct{}

// NewExecRunner returns the default external command runner.
func NewExecRunner() ExecRunner {
	return ExecRunner{}
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

	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

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
