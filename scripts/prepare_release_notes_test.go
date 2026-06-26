package scripts_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPrepareReleaseNotesPrefersVersionWithoutLeadingV(t *testing.T) {
	root := newReleaseNotesWorkspace(t)
	writeFile(t, filepath.Join(root, "docs", "release", "1.2.3.md"), "without v\n")
	writeFile(t, filepath.Join(root, "docs", "release", "v1.2.3.md"), "with v\n")

	output, source := runPrepareReleaseNotes(t, root, "v1.2.3", "1.2.3")

	if source != "docs/release/1.2.3.md" {
		t.Fatalf("source = %q, want docs/release/1.2.3.md", source)
	}
	if output != "without v\n" {
		t.Fatalf("output = %q, want version notes body", output)
	}
}

func TestPrepareReleaseNotesUsesTaggedFileWhenVersionFileIsMissing(t *testing.T) {
	root := newReleaseNotesWorkspace(t)
	writeFile(t, filepath.Join(root, "docs", "release", "v1.2.3.md"), "tagged body\n")

	output, source := runPrepareReleaseNotes(t, root, "v1.2.3", "1.2.3")

	if source != "docs/release/v1.2.3.md" {
		t.Fatalf("source = %q, want docs/release/v1.2.3.md", source)
	}
	if output != "tagged body\n" {
		t.Fatalf("output = %q, want tagged notes body", output)
	}
}

func TestPrepareReleaseNotesWritesFallbackWhenNoNotesFileExists(t *testing.T) {
	root := newReleaseNotesWorkspace(t)

	output, source := runPrepareReleaseNotes(t, root, "v1.2.3", "1.2.3")

	if source != "generated fallback" {
		t.Fatalf("source = %q, want generated fallback", source)
	}
	for _, want := range []string{
		"# Kitout 1.2.3",
		"No release notes file was found for v1.2.3.",
		"kitout_1.2.3_darwin_arm64.tar.gz",
		"kitout_1.2.3_darwin_amd64.tar.gz",
		"kitout_1.2.3_checksums.txt",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("fallback output missing %q:\n%s", want, output)
		}
	}
}

func TestValidateReleaseVersionAcceptsSemanticVersionTag(t *testing.T) {
	output, stderr, err := runValidateReleaseVersion("v1.2.3")
	if err != nil {
		t.Fatalf("validate-release-version failed: %v\nstderr:\n%s", err, stderr)
	}
	if output != "1.2.3\n" {
		t.Fatalf("output = %q, want 1.2.3", output)
	}
}

func TestValidateReleaseVersionRejectsUnsafeTag(t *testing.T) {
	output, stderr, err := runValidateReleaseVersion(`v1.2.3";printf${IFS}KITOUT_VALIDATION_MARKER;#`)
	if err == nil {
		t.Fatalf("validate-release-version succeeded with output %q, want failure", output)
	}
	if !strings.Contains(stderr, "release tag must be vX.Y.Z") {
		t.Fatalf("stderr = %q, want semantic-version rejection", stderr)
	}
}

func TestReleaseCheckRejectsUnsafeVersionBeforeRecipeExpansion(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is not available")
	}

	tempDir := t.TempDir()
	markerPath := filepath.Join(tempDir, "pwned")
	payload := `1.2.3"; printf KITOUT_PWNED > ` + markerPath + `; #`

	cmd := exec.Command(makePath, "release-check", "VERSION="+payload)
	cmd.Dir = filepath.Clean(filepath.Join(".."))
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("make release-check succeeded with unsafe VERSION, want failure\noutput:\n%s", output)
	}
	if !strings.Contains(string(output), "release tag must be vX.Y.Z") {
		t.Fatalf("output = %q, want release-version rejection", output)
	}
	if _, statErr := os.Stat(markerPath); !os.IsNotExist(statErr) {
		t.Fatalf("marker path state = %v, want no injected command output", statErr)
	}
}

func TestBuildKeepsVersionOutOfRecipeShell(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is not available")
	}

	payload := `1.2.3"; printf KITOUT_PWNED; #`
	cmd := exec.Command(makePath, "-n", "build", "VERSION="+payload)
	cmd.Dir = filepath.Clean(filepath.Join(".."))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("make -n build failed: %v\noutput:\n%s", err, output)
	}
	if strings.Contains(string(output), "KITOUT_PWNED") {
		t.Fatalf("build recipe expanded unsafe VERSION into shell text:\n%s", output)
	}
	if !strings.Contains(string(output), `-ldflags "$LDFLAGS"`) {
		t.Fatalf("build recipe does not use shell LDFLAGS environment:\n%s", output)
	}
}

func newReleaseNotesWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs", "release"), 0755); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func runPrepareReleaseNotes(t *testing.T, root, tag, version string) (string, string) {
	t.Helper()

	repoRoot := filepath.Clean(filepath.Join(".."))
	scriptPath, err := filepath.Abs(filepath.Join(repoRoot, "scripts", "prepare-release-notes.sh"))
	if err != nil {
		t.Fatal(err)
	}
	outputPath := filepath.Join(root, "release-notes.md")
	cmd := exec.Command(scriptPath, tag, version, outputPath)
	cmd.Dir = root

	sourceBytes, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("prepare-release-notes failed: %v\nstderr:\n%s", err, exitErr.Stderr)
		}
		t.Fatal(err)
	}

	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}

	return string(outputBytes), strings.TrimSpace(string(sourceBytes))
}

func runValidateReleaseVersion(tag string) (string, string, error) {
	repoRoot := filepath.Clean(filepath.Join(".."))
	scriptPath, err := filepath.Abs(filepath.Join(repoRoot, "scripts", "validate-release-version.sh"))
	if err != nil {
		return "", "", err
	}
	cmd := exec.Command(scriptPath, tag)
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(output), string(exitErr.Stderr), err
		}
		return string(output), "", err
	}
	return string(output), "", nil
}
