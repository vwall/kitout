package resources

import (
	"context"
	"errors"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestLoginShellStatusSatisfiedWithExplicitPath(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": "/bin/zsh\n" + path + "\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: "+path+"\n")},
	}}
	resource := newLoginShell(path, true, runner, system, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateSatisfied, "login shell is current")
	if got := result.Details["resolved_path"]; got != path {
		t.Fatalf("resolved_path = %q, want %q", got, path)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "id", args: []string{"-un"}},
		{name: "dscl", args: []string{".", "-read", "/Users/nix", "UserShell"}},
	})
}

func TestLoginShellStatusResolvesHomebrewPath(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": path + "\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"--prefix"}, "/opt/homebrew\n")},
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: "+path+"\n")},
	}}
	resource := newLoginShell("homebrew:fish", true, runner, system, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateSatisfied, "login shell is current")
	if got := result.Details["resolved_path"]; got != path {
		t.Fatalf("resolved_path = %q, want %q", got, path)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"--prefix"}},
		{name: "id", args: []string{"-un"}},
		{name: "dscl", args: []string{".", "-read", "/Users/nix", "UserShell"}},
	})
}

func TestLoginShellStatusFailsWhenHomebrewCannotResolve(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"--prefix"}, 127)},
	}}
	resource := newLoginShell("homebrew:fish", true, runner, fakeLoginShellSystem{}, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want Homebrew failure")
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateFailed, "could not resolve login shell path")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"--prefix"}}})
}

func TestLoginShellStatusMissingWhenShellPathDoesNotExist(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	runner := &fakeRunner{}
	resource := newLoginShell(path, true, runner, fakeLoginShellSystem{files: map[string]bool{path: false}}, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateMissing, "shell path is missing")
	expectCalls(t, runner.calls, nil)
}

func TestLoginShellStatusMissingWhenEtcShellsNeedsUpdate(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": "/bin/zsh\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: "+path+"\n")},
	}}
	resource := newLoginShell(path, true, runner, system, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateMissing, "shell path is not listed in /etc/shells")
}

func TestLoginShellStatusFailsWhenEtcShellsNeedsUpdateButConfigDisallowsIt(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": "/bin/zsh\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: "+path+"\n")},
	}}
	resource := newLoginShell(path, false, runner, system, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateFailed, "shell path is not listed in /etc/shells")
}

func TestLoginShellStatusChangedWhenCurrentShellDiffers(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": path + "\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: /bin/zsh\n")},
	}}
	resource := newLoginShell(path, true, runner, system, "/etc/shells")

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), loginShellType, engine.StateChanged, "login shell differs")
}

func TestLoginShellApplyAppendsEtcShellsAndRunsChsh(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/tmp/shells": "/bin/zsh\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: /bin/zsh\n")},
		{result: commandResult("sudo", []string{"sh", "-c", "printf '%s\\n' \"$1\" >> \"$2\"", "kitout", path, "/tmp/shells"}, 0)},
		{result: commandResult("chsh", []string{"-s", path}, 0)},
	}}
	resource := newLoginShell(path, true, runner, system, "/tmp/shells")

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), loginShellType, "set", true, "updated login shell")
	expectCalls(t, runner.calls, []commandCall{
		{name: "id", args: []string{"-un"}},
		{name: "dscl", args: []string{".", "-read", "/Users/nix", "UserShell"}},
		{name: "sudo", args: []string{"sh", "-c", "printf '%s\\n' \"$1\" >> \"$2\"", "kitout", path, "/tmp/shells"}},
		{name: "chsh", args: []string{"-s", path}},
	})
}

func TestLoginShellApplyRunsOnlyChshWhenShellIsAllowed(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": path + "\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: /bin/zsh\n")},
		{result: commandResult("chsh", []string{"-s", path}, 0)},
	}}
	resource := newLoginShell(path, true, runner, system, "/etc/shells")

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), loginShellType, "set", true, "updated login shell")
	expectCalls(t, runner.calls, []commandCall{
		{name: "id", args: []string{"-un"}},
		{name: "dscl", args: []string{".", "-read", "/Users/nix", "UserShell"}},
		{name: "chsh", args: []string{"-s", path}},
	})
}

func TestLoginShellApplyFailsWhenShellPathIsStillMissing(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	resource := newLoginShell(path, true, &fakeRunner{}, fakeLoginShellSystem{files: map[string]bool{path: false}}, "/etc/shells")

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want missing shell error")
	}

	expectApply(t, result, resource.ID(), loginShellType, "fail", false, "shell path is missing")
}

func TestLoginShellDryRunPlanDoesNotMutate(t *testing.T) {
	path := "/opt/homebrew/bin/fish"
	system := fakeLoginShellSystem{
		files: map[string]bool{path: true},
		contents: map[string]string{
			"/etc/shells": path + "\n",
		},
	}
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("id", []string{"-un"}, "nix\n")},
		{result: resultWithStdout("dscl", []string{".", "-read", "/Users/nix", "UserShell"}, "UserShell: /bin/zsh\n")},
	}}
	resource := newLoginShell(path, true, runner, system, "/etc/shells")

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "id", args: []string{"-un"}},
		{name: "dscl", args: []string{".", "-read", "/Users/nix", "UserShell"}},
	})
}

type fakeLoginShellSystem struct {
	files    map[string]bool
	contents map[string]string
	readErr  error
}

func (system fakeLoginShellSystem) fileExists(path string) (bool, error) {
	if system.files == nil {
		return false, nil
	}
	return system.files[path], nil
}

func (system fakeLoginShellSystem) readFile(path string) ([]byte, error) {
	if system.readErr != nil {
		return nil, system.readErr
	}
	if system.contents == nil {
		return nil, errors.New("missing fake file contents")
	}
	return []byte(system.contents[path]), nil
}
