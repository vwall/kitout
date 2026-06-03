package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatusLoadsValidConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+dir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config: "+configPath) {
		t.Fatalf("stdout = %q, want config path", stdout.String())
	}
	if !strings.Contains(stdout.String(), "directory:"+dir) {
		t.Fatalf("stdout = %q, want directory status", stdout.String())
	}
	if !strings.Contains(stdout.String(), "directory exists") {
		t.Fatalf("stdout = %q, want satisfied directory message", stdout.String())
	}
}

func TestStatusReturnsChangesWhenResourceIsMissing(t *testing.T) {
	missingDir := filepath.Join(t.TempDir(), "missing")
	configPath := writeCLIConfigFile(t, `version: 1

directories:
  - `+missingDir+`
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitChanges {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitChanges, stderr.String())
	}
	if !strings.Contains(stdout.String(), "directory is missing") {
		t.Fatalf("stdout = %q, want missing directory status", stdout.String())
	}
	if !strings.Contains(stdout.String(), "1 changes needed") {
		t.Fatalf("stdout = %q, want changes summary", stdout.String())
	}
}

func TestStatusReportsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

symlinks:
  - source: ~/dotfiles/zshrc
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Invalid config: symlinks[0].target is required") {
		t.Fatalf("stderr = %q, want structured validation error", stderr.String())
	}
}

func TestStatusReportsUnknownFieldsAsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1
unknown: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if !strings.Contains(stderr.String(), "Invalid config: parse config "+configPath) {
		t.Fatalf("stderr = %q, want invalid config parse guidance", stderr.String())
	}
	if !strings.Contains(stderr.String(), "unknown") {
		t.Fatalf("stderr = %q, want unknown field name", stderr.String())
	}
}

func TestStatusReportsMissingConfigAsRuntimeError(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if !strings.Contains(stderr.String(), "Failed to load config: read config "+configPath) {
		t.Fatalf("stderr = %q, want missing config guidance", stderr.String())
	}
}

func TestStatusAcceptsGlobalConfigFlagBeforeCommand(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"--config", configPath, "status"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
}

func TestStatusQuietSuppressesSuccessOutput(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath, "--quiet"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestStatusJSONReportsValidConfig(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitOK, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeStatusJSON(t, stdout.String())
	if response.Command != "status" {
		t.Fatalf("command = %q, want status", response.Command)
	}
	if !response.OK {
		t.Fatalf("ok = false, want true")
	}
	if response.Config == nil || response.Config.Path != configPath || !response.Config.Valid {
		t.Fatalf("config = %+v, want valid config path %q", response.Config, configPath)
	}
	if response.Plan == nil {
		t.Fatalf("plan = nil, want empty plan")
	}
	if response.Plan.Summary.Total != 0 {
		t.Fatalf("total = %d, want 0", response.Plan.Summary.Total)
	}
	if response.Error != nil {
		t.Fatalf("error = %+v, want nil", response.Error)
	}
}

func TestStatusJSONReportsValidationErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

symlinks:
  - source: ~/dotfiles/zshrc
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeStatusJSON(t, stdout.String())
	if response.OK {
		t.Fatalf("ok = true, want false")
	}
	if response.Config == nil || response.Config.Valid {
		t.Fatalf("config = %+v, want invalid config status", response.Config)
	}
	if response.Error == nil || response.Error.Type != "validation" {
		t.Fatalf("error = %+v, want validation error", response.Error)
	}
	if len(response.Error.Details) != 1 {
		t.Fatalf("details = %+v, want one validation detail", response.Error.Details)
	}
	if response.Error.Details[0].Field != "symlinks[0].target" {
		t.Fatalf("detail field = %q, want symlinks[0].target", response.Error.Details[0].Field)
	}
}

func TestStatusJSONReportsParseErrors(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1
unknown: true
`)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitValidation {
		t.Fatalf("exit code = %d, want %d", code, exitValidation)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeStatusJSON(t, stdout.String())
	if response.OK {
		t.Fatalf("ok = true, want false")
	}
	if response.Config == nil || response.Config.Path != configPath || response.Config.Valid {
		t.Fatalf("config = %+v, want invalid config path %q", response.Config, configPath)
	}
	if response.Error == nil || response.Error.Type != "parse" {
		t.Fatalf("error = %+v, want parse error", response.Error)
	}
	if !strings.Contains(response.Error.Message, "unknown") {
		t.Fatalf("message = %q, want unknown field", response.Error.Message)
	}
}

func TestStatusJSONReportsMissingConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "missing.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"status", "--config", configPath, "--json"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	response := decodeStatusJSON(t, stdout.String())
	if response.OK {
		t.Fatalf("ok = true, want false")
	}
	if response.Error == nil || response.Error.Type != "runtime" {
		t.Fatalf("error = %+v, want runtime error", response.Error)
	}
	if !strings.Contains(response.Error.Message, "Failed to load config: read config "+configPath) {
		t.Fatalf("message = %q, want missing config guidance", response.Error.Message)
	}
}

func writeCLIConfigFile(t *testing.T, contents string) string {
	t.Helper()

	configPath := filepath.Join(t.TempDir(), "kitout.yaml")
	if err := os.WriteFile(configPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return configPath
}

type statusJSONResponse struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Config  *statusJSONConfig `json:"config"`
	Plan    *statusJSONPlan   `json:"plan"`
	Error   *statusJSONError  `json:"error"`
}

type statusJSONConfig struct {
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
}

type statusJSONPlan struct {
	Summary statusJSONPlanSummary `json:"summary"`
	Items   []statusJSONPlanItem  `json:"items"`
	DryRun  bool                  `json:"dry_run"`
}

type statusJSONPlanSummary struct {
	Total     int `json:"total"`
	Satisfied int `json:"satisfied"`
	Missing   int `json:"missing"`
	Changed   int `json:"changed"`
	Failed    int `json:"failed"`
	Skipped   int `json:"skipped"`
	Unknown   int `json:"unknown"`
	ToApply   int `json:"to_apply"`
}

type statusJSONPlanItem struct {
	ResourceID string `json:"resource_id"`
	Type       string `json:"type"`
	State      string `json:"state"`
	Action     string `json:"action"`
	Message    string `json:"message"`
	Error      string `json:"error"`
}

type statusJSONError struct {
	Type    string                  `json:"type"`
	Message string                  `json:"message"`
	Details []statusJSONErrorDetail `json:"details"`
}

type statusJSONErrorDetail struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func decodeStatusJSON(t *testing.T, output string) statusJSONResponse {
	t.Helper()

	var response statusJSONResponse
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Fatalf("decode JSON output %q: %v", output, err)
	}

	return response
}
