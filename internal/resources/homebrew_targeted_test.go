package resources

import (
	"context"
	"testing"

	"github.com/vwall/kitout/internal/platform"
)

func TestDirectHomebrewOutdatedChecksAreTargetedAndFresh(t *testing.T) {
	for _, kind := range []string{"--formula", "--cask"} {
		t.Run(kind, func(t *testing.T) {
			runner := &fakeRunner{responses: []fakeResponse{
				{result: platform.CommandResult{Name: "brew", Stdout: "first\n"}},
				{},
				{result: platform.CommandResult{Name: "brew", Stdout: "second\n"}},
			}}
			checker := targetedOutdatedChecker(kind, runner)
			for i, test := range []struct {
				name     string
				outdated bool
			}{{"first", true}, {"first", false}, {"second", true}} {
				got, err := checker.Contains(context.Background(), test.name)
				if err != nil || got != test.outdated {
					t.Fatalf("check %d = %v, %v; want %v", i, got, err, test.outdated)
				}
			}
			expectCalls(t, runner.calls, []commandCall{
				{name: "brew", args: []string{"outdated", kind, "--quiet", "first"}},
				{name: "brew", args: []string{"outdated", kind, "--quiet", "first"}},
				{name: "brew", args: []string{"outdated", kind, "--quiet", "second"}},
			})
		})
	}
}

func TestDirectHomebrewOutdatedPreservesErrorsWithOutput(t *testing.T) {
	for _, kind := range []string{"--formula", "--cask"} {
		for _, test := range []struct {
			name, output        string
			err                 error
			outdated, wantError bool
		}{
			{name: "current"},
			{name: "outdated", output: "pkg\n", outdated: true},
			{name: "outdated exit one", output: "pkg\n", err: commandError("brew", nil, 1), outdated: true},
			{name: "empty exit one", err: commandError("brew", nil, 1), wantError: true},
			{name: "failure with partial output", output: "pkg\n", err: commandError("brew", nil, 2), wantError: true},
			{name: "canceled with partial output", output: "pkg\n", err: context.Canceled, wantError: true},
		} {
			t.Run(kind+"/"+test.name, func(t *testing.T) {
				runner := &fakeRunner{responses: []fakeResponse{{result: platform.CommandResult{Name: "brew", Stdout: test.output}, err: test.err}}}
				got, err := targetedOutdatedChecker(kind, runner).Contains(context.Background(), "pkg")
				if got != test.outdated || (err != nil) != test.wantError {
					t.Fatalf("Contains = %v, %v; want %v, error=%v", got, err, test.outdated, test.wantError)
				}
				if test.wantError && err.Error() != test.err.Error() {
					t.Fatalf("error = %v, want original %v", err, test.err)
				}
			})
		}
	}
}

func targetedOutdatedChecker(kind string, runner platform.Runner) brewOutdatedChecker {
	if kind == "--cask" {
		return newDirectCaskOutdatedChecker(runner)
	}
	return newDirectBrewOutdatedChecker(runner)
}
