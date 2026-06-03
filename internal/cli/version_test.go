package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/buildinfo"
)

func TestVersionPrintsMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{"kitout dev", "commit unknown", "built unknown"} {
		if !strings.Contains(output, want) {
			t.Fatalf("version output = %q, want %q", output, want)
		}
	}
}

func TestVersionPrintsInjectedMetadata(t *testing.T) {
	oldVersion := buildinfo.Version
	oldCommit := buildinfo.Commit
	oldBuildDate := buildinfo.BuildDate
	t.Cleanup(func() {
		buildinfo.Version = oldVersion
		buildinfo.Commit = oldCommit
		buildinfo.BuildDate = oldBuildDate
	})

	buildinfo.Version = "1.2.3"
	buildinfo.Commit = "abc1234"
	buildinfo.BuildDate = "2026-06-03T13:14:15Z"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	want := "kitout 1.2.3\ncommit abc1234\nbuilt 2026-06-03T13:14:15Z\n"
	if got := stdout.String(); got != want {
		t.Fatalf("version output = %q, want %q", got, want)
	}
}
