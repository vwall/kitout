package resources

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

func TestRepoStatusSatisfiedWhenRepoOriginMatches(t *testing.T) {
	path := t.TempDir()
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("git", []string{"-C", path, "rev-parse", "--show-toplevel"}, path+"\n")},
		{result: resultWithStdout("git", []string{"-C", path, "remote", "get-url", "origin"}, "git@example.com:a/repo.git\n")},
	}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "main", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), repoType, engine.StateSatisfied, "repository exists")
	expectCalls(t, runner.calls, []commandCall{
		{name: "git", args: []string{"-C", path, "rev-parse", "--show-toplevel"}},
		{name: "git", args: []string{"-C", path, "remote", "get-url", "origin"}},
	})
}

func TestRepoStatusMissingWhenPathDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo")
	resource := NewRepo(path, "git@example.com:a/repo.git", "", &fakeRunner{})

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), repoType, engine.StateMissing, "repository is missing")
}

func TestRepoStatusChangedWhenPathIsNotRepo(t *testing.T) {
	path := t.TempDir()
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("git", []string{"-C", path, "rev-parse", "--show-toplevel"}, 128)}}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), repoType, engine.StateChanged, "path exists but is not a Git repository")
}

func TestRepoStatusChangedWhenOriginDiffers(t *testing.T) {
	path := t.TempDir()
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("git", []string{"-C", path, "rev-parse", "--show-toplevel"}, path+"\n")},
		{result: resultWithStdout("git", []string{"-C", path, "remote", "get-url", "origin"}, "git@example.com:other/repo.git\n")},
	}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), repoType, engine.StateChanged, "repository origin does not match config")
}

func TestRepoApplyClonesMissingRepoWithBranch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo")
	runner := &fakeRunner{responses: []fakeResponse{
		{},
	}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "main", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), repoType, "clone", true, "cloned repository")
	expectCalls(t, runner.calls, []commandCall{
		{name: "git", args: []string{"clone", "--branch", "main", "git@example.com:a/repo.git", path}},
	})
}

func TestRepoApplyIsIdempotentWhenRepoExists(t *testing.T) {
	path := t.TempDir()
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("git", []string{"-C", path, "rev-parse", "--show-toplevel"}, path+"\n")},
		{result: resultWithStdout("git", []string{"-C", path, "remote", "get-url", "origin"}, "git@example.com:a/repo.git\n")},
	}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), repoType, "noop", false, "repository already exists")
}

func TestRepoApplyFailsWhenCloneFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo")
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("git", []string{"clone", "git@example.com:a/repo.git", path}, 128)}}}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want clone failure")
	}

	expectApply(t, result, resource.ID(), repoType, "clone", false, "could not clone repository")
}

func TestRepoApplyFailsWhenPathIsDifferentContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo")
	if err := os.WriteFile(path, []byte("not a repo"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", &fakeRunner{})

	result, err := resource.Apply(context.Background())
	if !containsError(err, "path already exists with different contents") {
		t.Fatalf("Apply error = %v, want changed path guidance", err)
	}

	expectApply(t, result, resource.ID(), repoType, "fail", false, err.Error())
}

func TestRepoDryRunPlanDoesNotClone(t *testing.T) {
	path := filepath.Join(t.TempDir(), "repo")
	runner := &fakeRunner{}
	resource := NewRepo(path, "git@example.com:a/repo.git", "", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, nil)
}

func TestRepoStatusRequiresCheckoutRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "checkout")
	const origin = "https://example.com/repo.git"
	for _, args := range [][]string{{"init", root}, {"-C", root, "remote", "add", "origin", origin}} {
		if output, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
	}
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name, path string
		state      engine.ResourceState
	}{
		{"root", root, engine.StateSatisfied},
		{"ordinary child", child, engine.StateChanged},
		{"symlink to root", alias, engine.StateSatisfied},
		{"child through symlink", filepath.Join(alias, "child"), engine.StateChanged},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resource := NewRepo(tt.path, origin, "", platform.NewExecRunner())
			status, err := resource.Status(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if status.State != tt.state {
				t.Fatalf("status = %+v, want %s", status, tt.state)
			}
			if tt.state == engine.StateChanged {
				result, err := resource.Apply(context.Background())
				if err == nil || result.Changed {
					t.Fatalf("Apply = %+v, %v; want refusal", result, err)
				}
			}
		})
	}
}
