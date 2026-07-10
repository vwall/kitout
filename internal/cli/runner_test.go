package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/platform"
)

func runWithCLIExecRunners(
	t *testing.T,
	args []string,
	stdin io.Reader,
	stdout, stderr io.Writer,
	runners ...platform.Runner,
) int {
	t.Helper()

	next := 0
	app := newApplication(context.Background(), stdin, stdout, stderr)
	app.runners.newRunner = func() platform.Runner {
		if next >= len(runners) {
			t.Fatalf("newRunner called %d times, want %d", next+1, len(runners))
		}
		runner := runners[next]
		next++
		return runner
	}

	code := app.run(args)
	if next != len(runners) {
		t.Fatalf("newRunner called %d times, want %d", next, len(runners))
	}
	return code
}

func TestApplicationDoesNotDispatchCanceledDoctorChecks(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	runner := &contextProbeRunner{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newApplication(ctx, nil, &stdout, &stderr)
	app.runners.newRunner = func() platform.Runner { return runner }

	code := app.run([]string{"doctor", "--config", configPath})

	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if runner.calls != 0 {
		t.Fatalf("runner calls = %d, want 0", runner.calls)
	}
	if !strings.Contains(stdout.String(), "Execution stopped: context canceled") {
		t.Fatalf("stdout = %q, want cancellation result", stdout.String())
	}
	if strings.Contains(stdout.String(), "is not available") {
		t.Fatalf("stdout = %q, want no false prerequisite failures", stdout.String())
	}
}

func TestApplicationStopsDoctorAfterRunnerCancelsContext(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	ctx, cancel := context.WithCancel(context.Background())
	runner := &contextProbeRunner{cancel: cancel}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newApplication(ctx, nil, &stdout, &stderr)
	app.runners.newRunner = func() platform.Runner { return runner }

	code := app.run([]string{"doctor", "--config", configPath})

	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
	if !strings.Contains(stdout.String(), "Execution stopped: context canceled") {
		t.Fatalf("stdout = %q, want cancellation result", stdout.String())
	}
	if strings.Contains(stdout.String(), "is not available") {
		t.Fatalf("stdout = %q, want canceled check to be discarded", stdout.String())
	}
}

func TestApplicationStopsDoctorWhenRunnerReturnsCancellation(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	runner := &contextProbeRunner{err: context.Canceled}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := newApplication(context.Background(), nil, &stdout, &stderr)
	app.runners.newRunner = func() platform.Runner { return runner }

	code := app.run([]string{"doctor", "--config", configPath})

	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if runner.calls != 1 {
		t.Fatalf("runner calls = %d, want 1", runner.calls)
	}
	if !strings.Contains(stdout.String(), "Execution stopped: context canceled") {
		t.Fatalf("stdout = %q, want cancellation result", stdout.String())
	}
	if strings.Contains(stdout.String(), "is not available") {
		t.Fatalf("stdout = %q, want canceled check to be discarded", stdout.String())
	}
}

type contextProbeRunner struct {
	calls  int
	cancel context.CancelFunc
	err    error
}

func (runner *contextProbeRunner) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	runner.calls++
	if runner.cancel != nil {
		runner.cancel()
	}
	result := platform.CommandResult{
		Name: name,
		Args: append([]string(nil), args...),
	}
	if runner.err != nil {
		return result, runner.err
	}
	return result, ctx.Err()
}
