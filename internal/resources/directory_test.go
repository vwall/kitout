package resources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/engine"
)

func TestDirectoryStatusSatisfiedWhenPathIsDirectory(t *testing.T) {
	path := t.TempDir()
	resource := NewDirectory(path)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	assertStatus(t, result, resource.ID(), engine.StateSatisfied, "directory exists")
	if result.Details["path"] != path {
		t.Fatalf("Details[path] = %q, want %q", result.Details["path"], path)
	}
}

func TestDirectoryStatusMissingWhenPathDoesNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "code", "kitout")
	resource := NewDirectory(path)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	assertStatus(t, result, resource.ID(), engine.StateMissing, "directory is missing")
}

func TestDirectoryStatusChangedWhenPathIsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kitout")
	if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	resource := NewDirectory(path)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	assertStatus(t, result, resource.ID(), engine.StateChanged, "path exists but is not a directory")
}

func TestDirectoryApplyCreatesMissingDirectoryAndParents(t *testing.T) {
	path := filepath.Join(t.TempDir(), "code", "kitout")
	resource := NewDirectory(path)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	assertApply(t, result, resource.ID(), "create", true, "created directory")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%q) returned error: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("Stat(%q).IsDir() = false, want true", path)
	}
}

func TestDirectoryApplyIsIdempotentWhenDirectoryExists(t *testing.T) {
	path := t.TempDir()
	resource := NewDirectory(path)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	assertApply(t, result, resource.ID(), "noop", false, "directory already exists")
}

func TestDirectoryApplyFailsWhenPathIsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "kitout")
	if err := os.WriteFile(path, []byte("not a directory"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	resource := NewDirectory(path)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want conflict error")
	}

	assertApply(t, result, resource.ID(), "fail", false, "")
	if !strings.Contains(err.Error(), "path exists and is not a directory") {
		t.Fatalf("error = %q, want conflict guidance", err.Error())
	}
	if result.Message != err.Error() {
		t.Fatalf("Message = %q, want %q", result.Message, err.Error())
	}
}

func TestDirectoryApplyFailureForEmptyPath(t *testing.T) {
	resource := NewDirectory("")

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want path error")
	}

	assertApply(t, result, resource.ID(), "fail", false, "directory path is required")
}

func TestDirectoryDryRunPlanDoesNotCreateDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "code")
	resource := NewDirectory(path)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if got, want := len(plan.Items), 1; got != want {
		t.Fatalf("len(plan.Items) = %d, want %d", got, want)
	}
	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want path to remain missing", path, err)
	}
}

func assertStatus(t *testing.T, result engine.StatusResult, id string, state engine.ResourceState, message string) {
	t.Helper()

	if result.ResourceID != id {
		t.Fatalf("ResourceID = %q, want %q", result.ResourceID, id)
	}
	if result.Type != directoryType {
		t.Fatalf("Type = %q, want %q", result.Type, directoryType)
	}
	if result.State != state {
		t.Fatalf("State = %q, want %q", result.State, state)
	}
	if message != "" && result.Message != message {
		t.Fatalf("Message = %q, want %q", result.Message, message)
	}
}

func assertApply(t *testing.T, result engine.ApplyResult, id, action string, changed bool, message string) {
	t.Helper()

	if result.ResourceID != id {
		t.Fatalf("ResourceID = %q, want %q", result.ResourceID, id)
	}
	if result.Type != directoryType {
		t.Fatalf("Type = %q, want %q", result.Type, directoryType)
	}
	if result.Action != action {
		t.Fatalf("Action = %q, want %q", result.Action, action)
	}
	if result.Changed != changed {
		t.Fatalf("Changed = %v, want %v", result.Changed, changed)
	}
	if message != "" && result.Message != message {
		t.Fatalf("Message = %q, want %q", result.Message, message)
	}
}
