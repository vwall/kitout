package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestCopyStatusSatisfiedWhenFileMatchesSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "same")
	writeFile(t, target, "same")
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateSatisfied, "copy target matches source")
}

func TestCopyStatusMissingWhenTargetDoesNotExist(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	writeFile(t, source, "contents")
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateMissing, "copy target is missing")
}

func TestCopyStatusFailedWhenSourceIsMissing(t *testing.T) {
	dir := t.TempDir()
	resource := NewCopy(filepath.Join(dir, "missing"), filepath.Join(dir, "target"), false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateFailed, "copy source is missing")
}

func TestCopyStatusChangedWhenFileDiffersAndReplaceTrue(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "source")
	writeFile(t, target, "target")
	resource := NewCopy(source, target, true)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateChanged, "copy target differs from source")
}

func TestCopyStatusFailedWhenFileDiffersAndReplaceFalse(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "source")
	writeFile(t, target, "target")
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateFailed, "copy target differs and replacement is not allowed")
}

func TestCopyStatusSatisfiedWhenDirectoryMatchesSource(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, filepath.Join(source, "SKILL.md"), "skill")
	writeFile(t, filepath.Join(source, "references", "nuxt.md"), "nuxt")
	writeFile(t, filepath.Join(target, "SKILL.md"), "skill")
	writeFile(t, filepath.Join(target, "references", "nuxt.md"), "nuxt")
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateSatisfied, "copy target matches source")
}

func TestCopyStatusChangedWhenDirectoryHasExtraFileAndReplaceTrue(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, filepath.Join(source, "SKILL.md"), "skill")
	writeFile(t, filepath.Join(target, "SKILL.md"), "skill")
	writeFile(t, filepath.Join(target, "extra.md"), "extra")
	resource := NewCopy(source, target, true)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateChanged, "copy target differs from source")
}

func TestCopyStatusFailedWhenSourceContainsSymlink(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(filepath.Join(source, "SKILL.md"), filepath.Join(source, "link")); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if !containsError(err, "copy source contains symlink") {
		t.Fatalf("Status error = %v, want symlink error", err)
	}

	expectStatus(t, result, resource.ID(), copyType, engine.StateFailed, err.Error())
}

func TestCopyApplyCreatesMissingFileTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "nested", "target")
	writeFile(t, source, "contents")
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), copyType, "create", true, "copied source to target")
	if got := readFile(t, target); got != "contents" {
		t.Fatalf("target contents = %q, want contents", got)
	}
}

func TestCopyApplyCreatesMissingDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, filepath.Join(source, "SKILL.md"), "skill")
	writeFile(t, filepath.Join(source, "references", "rails.md"), "rails")
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), copyType, "create", true, "copied source to target")
	if got := readFile(t, filepath.Join(target, "SKILL.md")); got != "skill" {
		t.Fatalf("target SKILL.md = %q, want skill", got)
	}
	if got := readFile(t, filepath.Join(target, "references", "rails.md")); got != "rails" {
		t.Fatalf("target references/rails.md = %q, want rails", got)
	}
}

func TestCopyApplyIsIdempotentWhenTargetMatches(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "same")
	writeFile(t, target, "same")
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), copyType, "noop", false, "copy target already matches source")
}

func TestCopyApplyFailsWithoutReplace(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "source")
	writeFile(t, target, "target")
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if !containsError(err, "replacement is not allowed") {
		t.Fatalf("Apply error = %v, want replacement guidance", err)
	}

	expectApply(t, result, resource.ID(), copyType, "fail", false, "copy target differs and replacement is not allowed")
}

func TestCopyApplyReplacesTargetWhenAllowed(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "source")
	writeFile(t, target, "target")
	resource := NewCopy(source, target, true)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), copyType, "replace", true, "replaced copy target")
	if got := readFile(t, target); got != "source" {
		t.Fatalf("target contents = %q, want source", got)
	}
}

func TestCopyDryRunPlanDoesNotCreateTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	target := filepath.Join(dir, "target")
	writeFile(t, source, "contents")
	resource := NewCopy(source, target, false)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want path to remain missing", target, err)
	}
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	return string(contents)
}
