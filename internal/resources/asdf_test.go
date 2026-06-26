package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestASDFPluginStatusFailsWhenASDFIsMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("asdf", []string{"--version"}, 127)},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want missing asdf prerequisite")
	}

	expectStatus(t, result, resource.ID(), asdfPluginType, engine.StateFailed, "asdf is required before checking plugins")
	expectCalls(t, runner.calls, []commandCall{{name: "asdf", args: []string{"--version"}}})
}

func TestASDFPluginStatusSatisfiedWhenPluginAndVersionsAreInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "  3.3.6\n* 3.4.1\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfPluginType, engine.StateSatisfied, "asdf plugin and versions are installed")
}

func TestASDFPluginStatusMissingWhenPluginIsMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "nodejs https://github.com/asdf-vm/asdf-nodejs.git\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfPluginType, engine.StateMissing, "asdf plugin is missing")
}

func TestASDFPluginStatusChangedWhenPluginURLDiffers(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://example.invalid/ruby.git\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", nil, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfPluginType, engine.StateChanged, "asdf plugin URL does not match config")
}

func TestASDFPluginStatusMissingWhenVersionIsMissing(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "3.2.0\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfPluginType, engine.StateMissing, "asdf version is missing")
	if result.Details["missing_versions"] != "3.3.6" {
		t.Fatalf("Details[missing_versions] = %q, want %q", result.Details["missing_versions"], "3.3.6")
	}
}

func TestASDFPluginApplyAddsMissingPluginAndInstallsVersions(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "")},
		{result: commandResult("asdf", []string{"plugin", "add", "ruby", "https://github.com/asdf-vm/asdf-ruby.git"}, 0)},
		{err: commandError("asdf", []string{"list", "ruby"}, 1)},
		{result: commandResult("asdf", []string{"install", "ruby", "3.3.6"}, 0)},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), asdfPluginType, "install", true, "installed asdf plugin or versions")
	expectCalls(t, runner.calls, []commandCall{
		{name: "asdf", args: []string{"--version"}},
		{name: "asdf", args: []string{"plugin", "list", "--urls"}},
		{name: "asdf", args: []string{"plugin", "add", "ruby", "https://github.com/asdf-vm/asdf-ruby.git"}},
		{name: "asdf", args: []string{"list", "ruby"}},
		{name: "asdf", args: []string{"install", "ruby", "3.3.6"}},
	})
}

func TestASDFPluginApplyUpdatesPluginBeforeInstallingMissingVersionsWhenConfigured(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "3.2.0\n")},
		{result: commandResult("asdf", []string{"plugin", "update", "ruby"}, 0)},
		{result: commandResult("asdf", []string{"install", "ruby", "3.3.6"}, 0)},
	}}
	resource := NewASDFPluginWithOptions(
		"ruby",
		"https://github.com/asdf-vm/asdf-ruby.git",
		[]string{"3.3.6"},
		ASDFPluginOptions{UpdateBeforeInstall: true},
		runner,
	)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), asdfPluginType, "install", true, "updated asdf plugin and installed versions")
	expectCalls(t, runner.calls, []commandCall{
		{name: "asdf", args: []string{"--version"}},
		{name: "asdf", args: []string{"plugin", "list", "--urls"}},
		{name: "asdf", args: []string{"list", "ruby"}},
		{name: "asdf", args: []string{"plugin", "update", "ruby"}},
		{name: "asdf", args: []string{"install", "ruby", "3.3.6"}},
	})
}

func TestASDFPluginApplyDoesNotUpdateNewlyAddedPluginBeforeInstallingVersions(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "")},
		{result: commandResult("asdf", []string{"plugin", "add", "ruby", "https://github.com/asdf-vm/asdf-ruby.git"}, 0)},
		{err: commandError("asdf", []string{"list", "ruby"}, 1)},
		{result: commandResult("asdf", []string{"install", "ruby", "3.3.6"}, 0)},
	}}
	resource := NewASDFPluginWithOptions(
		"ruby",
		"https://github.com/asdf-vm/asdf-ruby.git",
		[]string{"3.3.6"},
		ASDFPluginOptions{UpdateBeforeInstall: true},
		runner,
	)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), asdfPluginType, "install", true, "installed asdf plugin or versions")
	expectCalls(t, runner.calls, []commandCall{
		{name: "asdf", args: []string{"--version"}},
		{name: "asdf", args: []string{"plugin", "list", "--urls"}},
		{name: "asdf", args: []string{"plugin", "add", "ruby", "https://github.com/asdf-vm/asdf-ruby.git"}},
		{name: "asdf", args: []string{"list", "ruby"}},
		{name: "asdf", args: []string{"install", "ruby", "3.3.6"}},
	})
}

func TestASDFPluginApplyReportsActionableVersionNotFoundFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "3.2.0\n")},
		{err: commandErrorWithStderr("asdf", []string{"install", "ruby", "3.3.6"}, 1, "Version not found\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want install failure")
	}

	expectApply(
		t,
		result,
		resource.ID(),
		asdfPluginType,
		"install",
		false,
		"asdf version ruby 3.3.6 was not found; run `asdf plugin update ruby` and retry, or set `update_before_install: true` for this plugin",
	)
	expectCalls(t, runner.calls, []commandCall{
		{name: "asdf", args: []string{"--version"}},
		{name: "asdf", args: []string{"plugin", "list", "--urls"}},
		{name: "asdf", args: []string{"list", "ruby"}},
		{name: "asdf", args: []string{"install", "ruby", "3.3.6"}},
	})
}

func TestASDFPluginApplyPreservesGenericInstallFailureMessage(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "3.2.0\n")},
		{err: commandErrorWithStderr("asdf", []string{"install", "ruby", "3.3.6"}, 1, "network unavailable\n")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want install failure")
	}

	expectApply(t, result, resource.ID(), asdfPluginType, "install", false, "could not install asdf version ruby 3.3.6")
	if !containsError(err, "stderr:\nnetwork unavailable") {
		t.Fatalf("Apply error = %q, want asdf stderr summary", err.Error())
	}
}

func TestASDFPluginDryRunPlanDoesNotInstall(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("asdf", []string{"--version"}, 0)},
		{result: resultWithStdout("asdf", []string{"plugin", "list", "--urls"}, "ruby https://github.com/asdf-vm/asdf-ruby.git\n")},
		{result: resultWithStdout("asdf", []string{"list", "ruby"}, "")},
	}}
	resource := NewASDFPlugin("ruby", "https://github.com/asdf-vm/asdf-ruby.git", []string{"3.3.6"}, runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "asdf", args: []string{"--version"}},
		{name: "asdf", args: []string{"plugin", "list", "--urls"}},
		{name: "asdf", args: []string{"list", "ruby"}},
	})
}

func TestASDFToolVersionsStatusSatisfiedWhenEntriesMatch(t *testing.T) {
	path := writeToolVersions(t, "ruby 3.3.6\nnodejs 22.12.0\n")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfToolVersionsType, engine.StateSatisfied, ".tool-versions entries are correct")
}

func TestASDFToolVersionsStatusMissingWhenFileIsMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".tool-versions")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfToolVersionsType, engine.StateMissing, ".tool-versions file is missing")
}

func TestASDFToolVersionsStatusChangedWhenEntryDiffers(t *testing.T) {
	path := writeToolVersions(t, "ruby 3.2.0\n")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), asdfToolVersionsType, engine.StateChanged, ".tool-versions entry differs")
}

func TestASDFToolVersionsStatusFailedWhenAncestorIsSymlink(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	path := filepath.Join(linkedAncestor, ".tool-versions")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Status(context.Background())
	if !containsError(err, ".tool-versions ancestor") {
		t.Fatalf("Status error = %v, want ancestor symlink error", err)
	}

	expectStatus(t, result, resource.ID(), asdfToolVersionsType, engine.StateFailed, err.Error())
}

func TestASDFToolVersionsApplyPreservesUnrelatedEntries(t *testing.T) {
	path := writeToolVersions(t, "# managed nearby\nnodejs 22.12.0\nruby 3.2.0\n")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6", "python": "3.12.8"})

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), asdfToolVersionsType, "write", true, "updated .tool-versions entries")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated .tool-versions: %v", err)
	}
	want := "# managed nearby\nnodejs 22.12.0\nruby 3.3.6\npython 3.12.8\n"
	if string(contents) != want {
		t.Fatalf(".tool-versions = %q, want %q", string(contents), want)
	}
}

func TestASDFToolVersionsApplyCreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", ".tool-versions")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), asdfToolVersionsType, "write", true, "updated .tool-versions entries")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .tool-versions: %v", err)
	}
	if string(contents) != "ruby 3.3.6\n" {
		t.Fatalf(".tool-versions = %q, want ruby entry", string(contents))
	}
}

func TestASDFToolVersionsApplyRejectsMissingFileWithSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	linkedAncestor := filepath.Join(dir, "linked")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, ".tool-versions")
	path := filepath.Join(linkedAncestor, ".tool-versions")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Apply(context.Background())
	if !containsError(err, ".tool-versions ancestor") {
		t.Fatalf("Apply error = %v, want ancestor symlink error", err)
	}

	expectApply(t, result, resource.ID(), asdfToolVersionsType, "fail", false, err.Error())
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
	}
}

func TestASDFToolVersionsApplyRejectsSymlinkPath(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "shell-profile")
	original := []byte("export PATH=/usr/local/bin:$PATH\n")
	if err := os.WriteFile(target, original, 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}
	path := filepath.Join(dir, ".tool-versions")
	if err := os.Symlink(target, path); err != nil {
		t.Fatalf("create symlink: %v", err)
	}
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want symlink rejection")
	}

	expectApply(t, result, resource.ID(), asdfToolVersionsType, "fail", false, ".tool-versions path must be a regular file, not a symlink")
	contents, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatalf("read target: %v", readErr)
	}
	if string(contents) != string(original) {
		t.Fatalf("symlink target was modified: %q", string(contents))
	}
}

func TestASDFToolVersionsDryRunPlanDoesNotWrite(t *testing.T) {
	path := writeToolVersions(t, "ruby 3.2.0\n")
	resource := NewASDFToolVersions(path, map[string]string{"ruby": "3.3.6"})

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .tool-versions: %v", err)
	}
	if string(contents) != "ruby 3.2.0\n" {
		t.Fatalf(".tool-versions changed during dry-run plan: %q", string(contents))
	}
}

func writeToolVersions(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".tool-versions")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write .tool-versions: %v", err)
	}
	return path
}
