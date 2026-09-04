package resources

import (
	"bytes"
	"context"
	"fmt"
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

func TestCopyStatusFailedWhenTargetAncestorIsSymlink(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	linkedAncestor := filepath.Join(dir, "linked-target")
	outside := filepath.Join(dir, "outside")
	target := filepath.Join(linkedAncestor, "target")
	writeFile(t, source, "contents")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewCopy(source, target, false)

	result, err := resource.Status(context.Background())
	if !containsError(err, "copy target ancestor") {
		t.Fatalf("Status error = %v, want target ancestor symlink error", err)
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

func TestCopyApplyRejectsMissingFileTargetWithSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	linkedAncestor := filepath.Join(dir, "linked-target")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, "target")
	target := filepath.Join(linkedAncestor, "target")
	writeFile(t, source, "contents")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if !containsError(err, "copy target ancestor") {
		t.Fatalf("Apply error = %v, want target ancestor symlink error", err)
	}

	expectApply(t, result, resource.ID(), copyType, "fail", false, err.Error())
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
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

func TestCopyApplyRejectsMissingDirectoryTargetWithSymlinkAncestor(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "source")
	linkedAncestor := filepath.Join(dir, "linked-target")
	outside := filepath.Join(dir, "outside")
	outsideTarget := filepath.Join(outside, "profile", "SKILL.md")
	target := filepath.Join(linkedAncestor, "profile")
	writeFile(t, filepath.Join(source, "SKILL.md"), "skill")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.Symlink(outside, linkedAncestor); err != nil {
		t.Fatalf("Symlink returned error: %v", err)
	}
	resource := NewCopy(source, target, false)

	result, err := resource.Apply(context.Background())
	if !containsError(err, "copy target ancestor") {
		t.Fatalf("Apply error = %v, want target ancestor symlink error", err)
	}

	expectApply(t, result, resource.ID(), copyType, "fail", false, err.Error())
	if _, err := os.Lstat(outsideTarget); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want outside target to remain missing", outsideTarget, err)
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

func TestCopyApplyRejectsOverlappingPathsWithoutChangingSource(t *testing.T) {
	for _, scenario := range []string{"source-inside-target", "target-inside-source", "same-path", "source-parent-alias"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			container := filepath.Join(dir, "container")
			source := filepath.Join(container, "source")
			writeFile(t, source, "original")
			target := container
			switch scenario {
			case "target-inside-source":
				source = container
				target = filepath.Join(container, "missing", "copy")
			case "same-path":
				target = source
			case "source-parent-alias":
				alias := filepath.Join(dir, "alias")
				if err := os.Symlink(container, alias); err != nil {
					t.Fatal(err)
				}
				source = filepath.Join(alias, "source")
			}
			result, err := NewCopy(source, target, true).Apply(context.Background())
			if !containsError(err, "must not overlap") || result.Changed {
				t.Fatalf("Apply = %+v, %v; want overlap rejection without changes", result, err)
			}
			if got := readFile(t, filepath.Join(container, "source")); got != "original" {
				t.Fatalf("source contents = %q", got)
			}
			entries, err := os.ReadDir(container)
			if err != nil || len(entries) != 1 {
				t.Fatalf("container entries = %v, %v; want only original source", entries, err)
			}
		})
	}
}

func TestCopyApplyReplacesIncompatibleNestedTarget(t *testing.T) {
	for _, kind := range []string{"file", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
			writeFile(t, filepath.Join(source, "nested", "a"), "desired")
			writeFile(t, filepath.Join(target, "nested"), "old")
			if kind == "symlink" {
				if err := os.Remove(filepath.Join(target, "nested")); err != nil {
					t.Fatal(err)
				}
				outside := filepath.Join(dir, "outside")
				writeFile(t, outside, "untouched")
				if err := os.Symlink(outside, filepath.Join(target, "nested")); err != nil {
					t.Fatal(err)
				}
			}
			resource := NewCopy(source, target, true)
			result, err := resource.Apply(context.Background())
			if err != nil || !result.Changed {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			if got := readFile(t, filepath.Join(target, "nested", "a")); got != "desired" {
				t.Fatalf("copied contents = %q", got)
			}
			if kind == "symlink" && readFile(t, filepath.Join(dir, "outside")) != "untouched" {
				t.Fatal("modified symlink referent")
			}
			status, err := resource.Status(context.Background())
			if err != nil || status.State != engine.StateSatisfied {
				t.Fatalf("Status after apply = %+v, %v", status, err)
			}
		})
	}
}

func TestCopyApplyRejectsCaseInsensitiveOverlap(t *testing.T) {
	for _, direction := range []string{"source-inside-target", "target-inside-source"} {
		t.Run(direction, func(t *testing.T) {
			dir := t.TempDir()
			container := filepath.Join(dir, "Container")
			original := filepath.Join(container, "source")
			writeFile(t, original, "original")
			alias := filepath.Join(dir, "container")
			if _, err := os.Stat(alias); os.IsNotExist(err) {
				t.Skip("filesystem is case-sensitive")
			} else if err != nil {
				t.Fatal(err)
			}
			source, target := original, alias
			if direction == "target-inside-source" {
				source, target = container, filepath.Join(alias, "missing", "copy")
			}
			result, err := NewCopy(source, target, true).Apply(context.Background())
			if !containsError(err, "must not overlap") || result.Changed {
				t.Fatalf("Apply = %+v, %v; want overlap rejection without changes", result, err)
			}
			if readFile(t, original) != "original" {
				t.Fatal("source was modified")
			}
			entries, err := os.ReadDir(container)
			if err != nil || len(entries) != 1 {
				t.Fatalf("container entries = %v, %v; want only original source", entries, err)
			}
		})
	}
}

func TestCopyStreamingFiles(t *testing.T) {
	for _, size := range []int{0, copyBufferSize - 1, copyBufferSize, copyBufferSize + 1, 3*copyBufferSize + 17} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			dir := t.TempDir()
			source, target := filepath.Join(dir, "source"), filepath.Join(dir, "target")
			contents := bytes.Repeat([]byte("x"), size)
			if err := os.WriteFile(source, contents, 0o700); err != nil {
				t.Fatal(err)
			}
			resource := NewCopy(source, target, true)
			result, err := resource.Apply(context.Background())
			if err != nil || !result.Changed {
				t.Fatalf("Apply = %+v, %v", result, err)
			}
			got, err := os.ReadFile(target)
			if err != nil || !bytes.Equal(got, contents) {
				t.Fatalf("copied contents differ: %v", err)
			}
			info, err := os.Stat(target)
			if err != nil || info.Mode().Perm() != 0o700 {
				t.Fatalf("copied mode = %v, %v", info, err)
			}
			status, err := resource.Status(context.Background())
			if err != nil || status.State != engine.StateSatisfied {
				t.Fatalf("Status = %+v, %v", status, err)
			}
			if size > 0 {
				for _, offset := range []int{0, size / 2, size - 1} {
					changed := bytes.Clone(contents)
					changed[offset] = 'y'
					if err := os.WriteFile(target, changed, 0o700); err != nil {
						t.Fatal(err)
					}
					status, err = resource.Status(context.Background())
					if err != nil || status.State != engine.StateChanged {
						t.Fatalf("mismatch at %d: Status = %+v, %v", offset, status, err)
					}
				}
			}
			if err := os.WriteFile(target, append(contents, 'z'), 0o700); err != nil {
				t.Fatal(err)
			}
			status, err = resource.Status(context.Background())
			if err != nil || status.State != engine.StateChanged {
				t.Fatalf("size mismatch: Status = %+v, %v", status, err)
			}
		})
	}
}
