package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExplainShowsRequestedResource(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "code")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"explain", "--config", configPath, "directory:" + missingDir}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	output := stdout.String()
	for _, fragment := range []string{
		"Resource: directory:" + missingDir,
		"Type: directory",
		"state: missing",
		"action: apply",
		"path: " + missingDir,
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("stdout = %q, want fragment %q", output, fragment)
		}
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want directory to remain missing", missingDir, err)
	}
}

func TestExplainUnknownResourceIDReturnsValidationError(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"explain", "--config", configPath, "directory:/missing"}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), `resource "directory:/missing" is not configured`) {
		t.Fatalf("stderr = %q, want resource not configured guidance", stderr.String())
	}
}

func TestExplainOnlyInspectsRequestedResource(t *testing.T) {
	dir := t.TempDir()
	missingDir := filepath.Join(dir, "code")
	configPath := writeCLIConfigFile(t, `version: 1

brew:
  packages:
    - ripgrep

directories:
  - `+missingDir+`
`)
	runner := &fakeApplyRunner{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithCLIExecRunners(
		t,
		[]string{"explain", "--config", configPath, "directory:" + missingDir},
		nil,
		&stdout,
		&stderr,
		runner,
	)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if strings.Contains(stdout.String(), "brew:ripgrep") {
		t.Fatalf("stdout = %q, want only requested directory resource", stdout.String())
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %#v, want no external commands for unrelated brew resource", runner.calls)
	}
}

func TestExplainShellResourceDoesNotRunCommandAndMarksRisky(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+marker+`
    when: always
`)
	runner := &fakeApplyRunner{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithCLIExecRunners(
		t,
		[]string{"explain", "--config", configPath, "shell:Create marker"},
		nil,
		&stdout,
		&stderr,
		runner,
	)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	output := stdout.String()
	if !strings.Contains(output, "requires_approval: true") || !strings.Contains(output, "shell resources run explicit configured commands") {
		t.Fatalf("stdout = %q, want shell approval guidance", output)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want shell command marker to remain missing", marker, err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %#v, want shell command not to run during explain", runner.calls)
	}
}

func TestExplainJSONReportsSafety(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "marker")
	configPath := writeCLIConfigFile(t, `version: 1

shell:
  - name: Create marker
    command: touch `+marker+`
    when: always
`)
	runner := &fakeApplyRunner{}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithCLIExecRunners(
		t,
		[]string{"explain", "--config", configPath, "--json", "shell:Create marker"},
		nil,
		&stdout,
		&stderr,
		runner,
	)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}

	response := decodeExplainJSON(t, stdout.String())
	if response.Command != "explain" || !response.OK {
		t.Fatalf("response = %+v, want ok explain response", response)
	}
	if response.Explain == nil {
		t.Fatalf("explain = nil, want explain payload")
	}
	if response.Explain.Resource.ResourceID != "shell:Create marker" {
		t.Fatalf("resource = %+v, want shell resource", response.Explain.Resource)
	}
	if !response.Explain.Safety.WouldApply || !response.Explain.Safety.RequiresApproval {
		t.Fatalf("safety = %+v, want risky would-apply shell resource", response.Explain.Safety)
	}
	if response.Explain.Status.Details["command"] != "touch "+marker {
		t.Fatalf("details = %+v, want shell command detail", response.Explain.Status.Details)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want shell command marker to remain missing", marker, err)
	}
}

type explainJSONResponse struct {
	Command string              `json:"command"`
	OK      bool                `json:"ok"`
	Config  *statusJSONConfig   `json:"config"`
	Explain *explainJSONPayload `json:"explain"`
	Error   *statusJSONError    `json:"error"`
}

type explainJSONPayload struct {
	Resource contextJSONResource `json:"resource"`
	Status   explainJSONStatus   `json:"status"`
	Safety   explainJSONSafety   `json:"safety"`
}

type explainJSONStatus struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	State      string            `json:"state"`
	Action     string            `json:"action"`
	Message    string            `json:"message"`
	Details    map[string]string `json:"details"`
}

type explainJSONSafety struct {
	WouldApply       bool   `json:"would_apply"`
	RequiresApproval bool   `json:"requires_approval"`
	Reason           string `json:"reason"`
}

func decodeExplainJSON(t *testing.T, output string) explainJSONResponse {
	t.Helper()

	var response explainJSONResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	return response
}
