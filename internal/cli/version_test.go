package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionPrintsMetadata(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"version"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	output := stdout.String()
	for _, want := range []string{"kitout 0.1.0", "commit unknown", "built unknown"} {
		if !strings.Contains(output, want) {
			t.Fatalf("version output = %q, want %q", output, want)
		}
	}
}
