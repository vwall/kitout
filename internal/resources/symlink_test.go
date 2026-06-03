package resources

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestSymlinkStatusSatisfiedWhenTargetPointsToSource(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewSymlink(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), symlinkType, engine.StateSatisfied, "symlink is correct")
}

func TestSymlinkStatusMissingWhenTargetDoesNotExist(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	resource := NewSymlink(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), symlinkType, engine.StateMissing, "symlink is missing")
}

func TestSymlinkStatusChangedWhenTargetPointsElsewhere(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	other := filepath.Join(dir, "other")
	target := filepath.Join(dir, "target")
	if err := os.Symlink(other, target); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewSymlink(source, target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), symlinkType, engine.StateChanged, "symlink points elsewhere")
}

func TestSymlinkStatusFailedWhenTargetIsFileAndReplaceFalse(t *testing.T) {
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	resource := NewSymlink("source", target, false)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), symlinkType, engine.StateFailed, "target exists and replacement is not allowed")
}

func TestSymlinkApplyCreatesMissingTarget(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	resource := NewSymlink(source, target, false)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), symlinkType, "create", true, "created symlink")
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink returned error: %v", err)
	}
	if link != source {
		t.Fatalf("Readlink = %q, want %q", link, source)
	}
}

func TestSymlinkApplyIsIdempotentWhenTargetIsCorrect(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	if err := os.Symlink(source, target); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewSymlink(source, target, false)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), symlinkType, "noop", false, "symlink already correct")
}

func TestSymlinkApplyFailsWithoutReplace(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	other := filepath.Join(dir, "other")
	target := filepath.Join(dir, "target")
	if err := os.Symlink(other, target); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewSymlink(source, target, false)

	result, err := resource.Apply(context.Background())
	if !containsError(err, "replacement is not allowed") {
		t.Fatalf("Apply error = %v, want replacement guidance", err)
	}

	expectApply(t, result, resource.ID(), symlinkType, "fail", false, err.Error())
}

func TestSymlinkApplyReplacesTargetWhenAllowed(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	other := filepath.Join(dir, "other")
	target := filepath.Join(dir, "target")
	if err := os.Symlink(other, target); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewSymlink(source, target, true)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), symlinkType, "replace", true, "replaced symlink")
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("Readlink returned error: %v", err)
	}
	if link != source {
		t.Fatalf("Readlink = %q, want %q", link, source)
	}
}

func TestSymlinkDryRunPlanDoesNotCreateTarget(t *testing.T) {
	source := filepath.Join(t.TempDir(), "source")
	target := filepath.Join(t.TempDir(), "target")
	resource := NewSymlink(source, target, false)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want path to remain missing", target, err)
	}
}
