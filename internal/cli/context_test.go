package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextShowsReadOnlyAgentSummary(t *testing.T) {
	dir := t.TempDir()
	missingDir := filepath.Join(dir, "code")
	copyTarget := filepath.Join(dir, "home", ".codex", "skills", "nuxt-practices")
	marker := filepath.Join(dir, "marker")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`

copies:
  - source: ./codex/skills/nuxt-practices
    target: `+copyTarget+`
    replace: false

shell:
  - name: Create marker
    command: touch `+marker+`
    when: always
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"context", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	output := stdout.String()
	for _, fragment := range []string{
		"Kitout agent context",
		"Config: " + configPath,
		"Safe read-only commands:",
		"Requires explicit user approval:",
		"directory:" + missingDir,
		"copy:" + copyTarget,
		"shell:Create marker",
		"Edit files in the setup repo",
	} {
		if !strings.Contains(output, fragment) {
			t.Fatalf("stdout = %q, want fragment %q", output, fragment)
		}
	}
	if _, err := os.Stat(missingDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want directory to remain missing", missingDir, err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("Stat(%q) error = %v, want shell command marker to remain missing", marker, err)
	}
}

func TestContextRejectsImplicitConfigWhenLocalAndHomeBothExist(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(dir, "kitout.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	writeHomeCLIConfigFile(t, home, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"context"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "pass --config to choose one") {
		t.Fatalf("stderr = %q, want --config guidance", stderr.String())
	}
}

func TestContextJSONReportsDeclaredResources(t *testing.T) {
	dir := t.TempDir()
	copyTarget := filepath.Join(dir, "home", ".zshrc")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+filepath.Join(dir, "code")+`

copies:
  - source: ./home/zshrc
    target: `+copyTarget+`
    replace: false
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"context", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeContextJSON(t, stdout.String())
	if response.Command != "context" || !response.OK {
		t.Fatalf("response = %+v, want ok context response", response)
	}
	if response.Config == nil || response.Config.Path != configPath || !response.Config.Valid {
		t.Fatalf("config = %+v, want valid config path %q", response.Config, configPath)
	}
	if response.Context == nil {
		t.Fatalf("context = nil, want context payload")
	}
	if response.Plan != nil || response.Apply != nil {
		t.Fatalf("plan/apply = %s/%s, want absent mutation reports", rawMessageString(response.Plan), rawMessageString(response.Apply))
	}
	if response.Context.SchemaVersion != 1 {
		t.Fatalf("schema_version = %d, want 1", response.Context.SchemaVersion)
	}
	if len(response.Context.SafeCommands) == 0 || len(response.Context.RequiresApproval) == 0 {
		t.Fatalf("context commands = %+v / %+v, want safe and approval commands", response.Context.SafeCommands, response.Context.RequiresApproval)
	}
	resource := contextResourceByID(t, response.Context.Resources, "copy:"+copyTarget)
	if resource.Type != "copy" {
		t.Fatalf("type = %q, want copy", resource.Type)
	}
	if resource.Details["target"] != copyTarget || resource.Details["replace"] != "false" {
		t.Fatalf("details = %+v, want copy target and replace=false", resource.Details)
	}
}

type contextJSONResponse struct {
	Command string              `json:"command"`
	OK      bool                `json:"ok"`
	Config  *statusJSONConfig   `json:"config"`
	Context *contextJSONPayload `json:"context"`
	Plan    *json.RawMessage    `json:"plan"`
	Apply   *json.RawMessage    `json:"apply"`
	Error   *statusJSONError    `json:"error"`
}

type contextJSONPayload struct {
	SchemaVersion    int                   `json:"schema_version"`
	SafeCommands     []contextJSONCommand  `json:"safe_commands"`
	RequiresApproval []contextJSONCommand  `json:"requires_approval"`
	Resources        []contextJSONResource `json:"resources"`
	Guidance         []string              `json:"guidance"`
}

type contextJSONCommand struct {
	Command string `json:"command"`
	Reason  string `json:"reason"`
}

type contextJSONResource struct {
	ResourceID string            `json:"resource_id"`
	Type       string            `json:"type"`
	Label      string            `json:"label"`
	Details    map[string]string `json:"details"`
}

func decodeContextJSON(t *testing.T, output string) contextJSONResponse {
	t.Helper()

	var response contextJSONResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}
	return response
}

func contextResourceByID(t *testing.T, resources []contextJSONResource, id string) contextJSONResource {
	t.Helper()

	for _, resource := range resources {
		if resource.ResourceID == id {
			return resource
		}
	}
	t.Fatalf("resources = %+v, want %s", resources, id)
	return contextJSONResource{}
}

func rawMessageString(message *json.RawMessage) string {
	if message == nil {
		return ""
	}
	return string(*message)
}
