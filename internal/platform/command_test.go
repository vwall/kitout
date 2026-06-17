package platform

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"
)

func TestExecRunnerCapturesSuccessfulCommandOutput(t *testing.T) {
	result, err := NewExecRunner().Run(context.Background(), os.Args[0], helperProcessArgs("success")...)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if result.Name != os.Args[0] {
		t.Fatalf("Name = %q, want test binary", result.Name)
	}
	if !reflect.DeepEqual(result.Args, helperProcessArgs("success")) {
		t.Fatalf("Args = %#v, want helper process args", result.Args)
	}
	if strings.TrimSpace(result.Stdout) != "hello stdout" {
		t.Fatalf("Stdout = %q, want helper stdout", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "hello stderr" {
		t.Fatalf("Stderr = %q, want helper stderr", result.Stderr)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
}

func TestExecRunnerStreamsOutputWhenVerbose(t *testing.T) {
	var stdout strings.Builder
	var stderr strings.Builder

	result, err := NewVerboseExecRunner(&stdout, &stderr).Run(context.Background(), os.Args[0], helperProcessArgs("success")...)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	if !strings.HasPrefix(stdout.String(), "$ "+os.Args[0]+" -test.run=TestHelperProcess -- success\n") {
		t.Fatalf("stdout = %q, want rendered command prefix", stdout.String())
	}
	if !strings.Contains(stdout.String(), "hello stdout\n") {
		t.Fatalf("stdout = %q, want streamed stdout", stdout.String())
	}
	if stderr.String() != "hello stderr\n" {
		t.Fatalf("stderr = %q, want streamed stderr", stderr.String())
	}
	if strings.TrimSpace(result.Stdout) != "hello stdout" {
		t.Fatalf("result.Stdout = %q, want captured stdout", result.Stdout)
	}
	if strings.TrimSpace(result.Stderr) != "hello stderr" {
		t.Fatalf("result.Stderr = %q, want captured stderr", result.Stderr)
	}
}

func TestRenderCommandQuotesUnsafeArgs(t *testing.T) {
	got := renderCommand("brew", []string{"install", "visual studio code"})
	want := "brew install 'visual studio code'"
	if got != want {
		t.Fatalf("renderCommand = %q, want %q", got, want)
	}
}

func TestExecRunnerCopiesArgs(t *testing.T) {
	args := helperProcessArgs("success")

	result, err := NewExecRunner().Run(context.Background(), os.Args[0], args...)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	args[0] = "mutated"
	if result.Args[0] != "-test.run=TestHelperProcess" {
		t.Fatalf("Args[0] = %q, want copied helper flag", result.Args[0])
	}
}

func TestExecRunnerReportsNonZeroExitWithOutput(t *testing.T) {
	result, err := NewExecRunner().Run(context.Background(), os.Args[0], helperProcessArgs("failure")...)
	if err == nil {
		t.Fatal("Run returned nil error, want command error")
	}

	var commandError CommandError
	if !errors.As(err, &commandError) {
		t.Fatalf("Run error = %T %[1]v, want CommandError", err)
	}
	if result.ExitCode == 0 {
		t.Fatalf("ExitCode = %d, want non-zero", result.ExitCode)
	}
	if commandError.Result.ExitCode != result.ExitCode {
		t.Fatalf("CommandError exit = %d, want %d", commandError.Result.ExitCode, result.ExitCode)
	}
	if strings.TrimSpace(result.Stderr) != "helper failed" {
		t.Fatalf("Stderr = %q, want helper failure output", result.Stderr)
	}
	if !strings.Contains(err.Error(), renderCommand(os.Args[0], helperProcessArgs("failure"))+" exited with status") {
		t.Fatalf("error = %q, want command status", err.Error())
	}
	if strings.Contains(err.Error(), ": exit status") {
		t.Fatalf("error = %q, want no duplicated exit status text", err.Error())
	}
	if !strings.Contains(err.Error(), "stderr:\nhelper failed") {
		t.Fatalf("error = %q, want stderr summary", err.Error())
	}
}

func TestCommandErrorSummarizesOutputTail(t *testing.T) {
	result := CommandResult{
		Name:     "asdf",
		Args:     []string{"install", "ruby", "3.3.6"},
		Stdout:   "downloaded ruby source\n",
		Stderr:   "one\ntwo\nthree\nfour\nfive\nsix\nseven\neight\nnine\nten\neleven\n",
		ExitCode: 1,
	}
	err := CommandError{Result: result, Err: errors.New("exit status 1")}

	got := err.Error()
	for _, fragment := range []string{
		"asdf install ruby 3.3.6 failed after exit status 1: exit status 1",
		"stderr:\n... 1 earlier lines omitted\ntwo",
		"eleven",
		"stdout:\ndownloaded ruby source",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("error = %q, want fragment %q", got, fragment)
		}
	}
	if strings.Contains(got, "\none\n") {
		t.Fatalf("error = %q, want earliest stderr line omitted", got)
	}
}

func TestCommandErrorPreservesNonExitErrorsWithProcessState(t *testing.T) {
	err := CommandError{
		Result: CommandResult{
			Name:     "kitout-helper",
			Args:     []string{"run"},
			ExitCode: 0,
		},
		Err: errors.New("copy stdout: broken pipe"),
	}

	got := err.Error()
	for _, fragment := range []string{
		"kitout-helper run failed after exit status 0",
		"copy stdout: broken pipe",
	} {
		if !strings.Contains(got, fragment) {
			t.Fatalf("error = %q, want fragment %q", got, fragment)
		}
	}
}

func TestExecRunnerReportsMissingCommand(t *testing.T) {
	_, err := NewExecRunner().Run(context.Background(), "kitout-command-that-does-not-exist")
	if err == nil {
		t.Fatal("Run returned nil error, want missing command error")
	}
	if !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("errors.Is(err, exec.ErrNotFound) = false for %v", err)
	}
}

func TestExecRunnerRespectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := NewExecRunner().Run(ctx, os.Args[0], helperProcessArgs("success")...)
	if err == nil {
		t.Fatal("Run returned nil error, want canceled context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("errors.Is(err, context.Canceled) = false for %v", err)
	}
	if result.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestExecRunnerRejectsEmptyCommandName(t *testing.T) {
	result, err := NewExecRunner().Run(context.Background(), "")
	if err == nil {
		t.Fatal("Run returned nil error, want validation error")
	}
	if !strings.Contains(err.Error(), "command name is required") {
		t.Fatalf("error = %q, want command name guidance", err.Error())
	}
	if result.ExitCode != -1 {
		t.Fatalf("ExitCode = %d, want -1", result.ExitCode)
	}
}

func TestHelperProcess(t *testing.T) {
	args := os.Args
	for i, arg := range args {
		if arg == "--" && i+1 < len(args) {
			switch args[i+1] {
			case "success":
				fmt.Fprintln(os.Stdout, "hello stdout")
				fmt.Fprintln(os.Stderr, "hello stderr")
				os.Exit(0)
			case "failure":
				fmt.Fprintln(os.Stderr, "helper failed")
				os.Exit(17)
			}
		}
	}
}

func helperProcessArgs(scenario string) []string {
	return []string{"-test.run=TestHelperProcess", "--", scenario}
}
