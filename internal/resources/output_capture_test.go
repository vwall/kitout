package resources

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/platform"
)

func TestShellApplyBoundsCapturedFailureOutput(t *testing.T) {
	resource := NewShellCommand("large logs", "printf '%070000d\\n' 0; printf 'stdout tail\\n'; printf '%070000d\\n' 0 >&2; printf 'stderr tail\\n' >&2; exit 7", "always", platform.NewExecRunner())
	result, err := resource.Apply(context.Background())
	var commandErr platform.CommandError
	if !errors.As(err, &commandErr) || result.Changed || commandErr.Result.ExitCode != 7 {
		t.Fatalf("Apply=%+v, %v; want exit7 failure", result, err)
	}
	captured := commandErr.Result
	if !captured.StdoutTruncated || !captured.StderrTruncated {
		t.Fatal("shell mutation must use bounded capture for both streams")
	}
	if len(captured.Stdout) > 64*1024 || len(captured.Stderr) > 64*1024 {
		t.Fatalf("captured bytes=%d/%d", len(captured.Stdout), len(captured.Stderr))
	}
	for _, marker := range []string{"stdout tail", "stderr tail", "truncated"} {
		if !strings.Contains(err.Error(), marker) {
			t.Fatalf("error omitted diagnostic %q", marker)
		}
	}
}

func TestASDFInstallPreservesFullOutputForFailureGuidance(t *testing.T) {
	dir := t.TempDir()
	script := `#!/bin/sh
case "$1 $2" in
  '--version ') printf 'asdf test\n' ;;
  'plugin list') printf 'ruby https://example.com/ruby.git\n' ;;
  'list ruby') exit 0 ;;
  'install ruby') printf 'VERSION NOT FOUND\n'; printf '%070000d\n' 0; exit 1 ;;
  *) exit 2 ;;
esac
`
	if err := os.WriteFile(filepath.Join(dir, "asdf"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	resource := NewASDFPlugin("ruby", "https://example.com/ruby.git", []string{"1.0"}, platform.NewExecRunner())
	result, err := resource.Apply(context.Background())
	var commandErr platform.CommandError
	if !errors.As(err, &commandErr) {
		t.Fatalf("Apply error=%v; want install command failure", err)
	}
	if commandErr.Result.StdoutTruncated || len(commandErr.Result.Stdout) < 70000 {
		t.Fatal("ASDF parser input must remain complete")
	}
	if !strings.Contains(result.Message, "asdf plugin update ruby") {
		t.Fatalf("lost failure guidance: %s", result.Message)
	}
}
