package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/platform"
)

func TestDoctorCheckerReportsHealthySystem(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	runner := &fakeDoctorRunner{responses: []fakeDoctorResponse{
		{result: doctorCommandResult("xcode-select", []string{"-p"}, "/Library/Developer/CommandLineTools\n")},
		{result: doctorCommandResult("brew", []string{"--version"}, "Homebrew 4.0.0\n")},
		{result: doctorCommandResult("brew", []string{"--prefix"}, "/opt/homebrew\n")},
		{result: doctorCommandResult("git", []string{"--version"}, "git version 2.45.0\n")},
	}}
	checker := newDoctorChecker(runner, healthyDoctorInfo(t, "arm64"))

	report := checker.Check(context.Background(), configPath)

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %+v", report)
	}
	if report.Summary != (doctorSummary{Total: 9, OK: 9}) {
		t.Fatalf("summary = %+v, want nine ok checks", report.Summary)
	}
	expectDoctorCalls(t, runner.calls, []doctorCommandCall{
		{name: "xcode-select", args: []string{"-p"}},
		{name: "brew", args: []string{"--version"}},
		{name: "brew", args: []string{"--prefix"}},
		{name: "git", args: []string{"--version"}},
	})
	assertDoctorItem(t, report, "Homebrew", doctorOK, "Homebrew 4.0.0")
	assertDoctorItem(t, report, "Homebrew path", doctorOK, "Homebrew prefix is /opt/homebrew")
	assertDoctorItem(t, report, "Shell environment", doctorOK, "SHELL and PATH look usable")
	assertDoctorItem(t, report, "Config", doctorOK, "config is valid")
	assertDoctorItem(t, report, "Path permissions", doctorOK, "no configured filesystem write targets")
}

func TestDoctorCheckerReportsPrerequisiteFailures(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	runner := &fakeDoctorRunner{responses: []fakeDoctorResponse{
		{err: doctorCommandError("xcode-select", []string{"-p"})},
		{err: doctorCommandError("brew", []string{"--version"})},
		{result: doctorCommandResult("git", []string{"--version"}, "git version 2.45.0\n")},
	}}
	checker := newDoctorChecker(runner, doctorSystemInfo{
		OS:    "linux",
		Arch:  "amd64",
		Shell: executablePath(t),
		Path:  "/usr/local/bin:/usr/bin:/bin",
	})

	report := checker.Check(context.Background(), configPath)

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
	if report.Summary.Fail != 3 {
		t.Fatalf("failures = %d, want 3", report.Summary.Fail)
	}
	if report.Summary.Warn != 2 {
		t.Fatalf("warnings = %d, want 2", report.Summary.Warn)
	}
	assertDoctorItem(t, report, "macOS", doctorFail, "unsupported OS")
	assertDoctorItem(t, report, "CPU architecture", doctorWarn, "Intel architecture")
	assertDoctorItem(t, report, "Homebrew", doctorFail, "Homebrew is not available")
	assertDoctorItem(t, report, "Homebrew path", doctorWarn, "skipped because Homebrew is not available")
}

func TestDoctorCheckerReportsInvalidConfig(t *testing.T) {
	configPath := writeCLIConfigFile(t, `version: 1

repos:
  - path: ~/code/kitout
`)
	runner := &fakeDoctorRunner{}
	checker := newDoctorChecker(runner, healthyDoctorInfo(t, "arm64"))

	report := checker.Check(context.Background(), configPath)

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
	assertDoctorItem(t, report, "Config", doctorFail, "config is not valid")
	configItem := doctorItemByName(t, report, "Config")
	if !strings.Contains(configItem.Details["error"], "repos[0].url is required") {
		t.Fatalf("config error = %q, want validation detail", configItem.Details["error"])
	}
}

func TestDoctorCheckerReportsWritableConfiguredPaths(t *testing.T) {
	dir := t.TempDir()
	toolVersionsPath := filepath.Join(dir, ".tool-versions")
	if err := os.WriteFile(toolVersionsPath, []byte("ruby 3.3.6\n"), 0o644); err != nil {
		t.Fatalf("write .tool-versions: %v", err)
	}
	configPath := writeCLIConfigFile(t, "version: 1\n"+
		"asdf:\n"+
		"  tool_versions:\n"+
		"    - path: "+toolVersionsPath+"\n"+
		"      tools:\n"+
		"        ruby: 3.3.6\n"+
		"directories:\n"+
		"  - "+filepath.Join(dir, "code")+"\n"+
		"repos:\n"+
		"  - path: "+filepath.Join(dir, "repo")+"\n"+
		"    url: git@example.com:me/repo.git\n"+
		"symlinks:\n"+
		"  - source: "+filepath.Join(dir, "dotfile")+"\n"+
		"    target: "+filepath.Join(dir, "link")+"\n"+
		"symlink_groups:\n"+
		"  - source_root: "+filepath.Join(dir, "dotfiles", "home")+"\n"+
		"    target_root: "+filepath.Join(dir, "home")+"\n"+
		"    paths:\n"+
		"      - .gitconfig\n")
	runner := &fakeDoctorRunner{}
	checker := newDoctorChecker(runner, healthyDoctorInfo(t, "arm64"))

	report := checker.Check(context.Background(), configPath)

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %+v", report)
	}
	assertDoctorItem(t, report, "Path permissions", doctorOK, "5 configured write target(s) look writable")
}

func TestDoctorCheckerReportsUnwritableConfiguredPaths(t *testing.T) {
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o555); err != nil {
		t.Fatalf("create locked dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(locked, 0o755); err != nil {
			t.Fatalf("restore locked dir permissions: %v", err)
		}
	})
	configPath := writeCLIConfigFile(t, "version: 1\n"+
		"directories:\n"+
		"  - "+filepath.Join(locked, "code")+"\n")
	runner := &fakeDoctorRunner{}
	checker := newDoctorChecker(runner, healthyDoctorInfo(t, "arm64"))

	report := checker.Check(context.Background(), configPath)

	if !report.HasFailures() {
		t.Fatalf("HasFailures() = false, want true")
	}
	assertDoctorItem(t, report, "Path permissions", doctorFail, "1 configured write target(s) may not be writable")
	item := doctorItemByName(t, report, "Path permissions")
	if !strings.Contains(item.Details["failures"], locked+" is not writable") {
		t.Fatalf("failures = %q, want locked parent detail", item.Details["failures"])
	}
}

func TestDoctorCheckerWarnsForUnexpectedAppleSiliconHomebrewPath(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	runner := &fakeDoctorRunner{responses: []fakeDoctorResponse{
		{result: doctorCommandResult("xcode-select", []string{"-p"}, "/Library/Developer/CommandLineTools\n")},
		{result: doctorCommandResult("brew", []string{"--version"}, "Homebrew 4.0.0\n")},
		{result: doctorCommandResult("brew", []string{"--prefix"}, "/usr/local\n")},
		{result: doctorCommandResult("git", []string{"--version"}, "git version 2.45.0\n")},
	}}
	checker := newDoctorChecker(runner, healthyDoctorInfo(t, "arm64"))

	report := checker.Check(context.Background(), configPath)

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %+v", report)
	}
	assertDoctorItem(t, report, "Homebrew path", doctorWarn, "expected /opt/homebrew")
}

func TestDoctorCheckerWarnsWhenShellPathMissesHomebrewBin(t *testing.T) {
	configPath := writeCLIConfigFile(t, "version: 1\n")
	runner := &fakeDoctorRunner{responses: []fakeDoctorResponse{
		{result: doctorCommandResult("xcode-select", []string{"-p"}, "/Library/Developer/CommandLineTools\n")},
		{result: doctorCommandResult("brew", []string{"--version"}, "Homebrew 4.0.0\n")},
		{result: doctorCommandResult("brew", []string{"--prefix"}, "/opt/homebrew\n")},
		{result: doctorCommandResult("git", []string{"--version"}, "git version 2.45.0\n")},
	}}
	info := healthyDoctorInfo(t, "arm64")
	info.Path = "/usr/bin:/bin"
	checker := newDoctorChecker(runner, info)

	report := checker.Check(context.Background(), configPath)

	if report.HasFailures() {
		t.Fatalf("HasFailures() = true, want false: %+v", report)
	}
	assertDoctorItem(t, report, "Shell environment", doctorWarn, "PATH does not include /opt/homebrew/bin")
}

func TestDoctorUsesLocalConfigByDefaultWhenHomeConfigIsMissing(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	localPath := filepath.Join(dir, "kitout.yaml")
	if err := os.WriteFile(localPath, []byte("version: 1\n\nrepos:\n  - path: local-repo\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	wantLocalPath, err := filepath.Abs("kitout.yaml")
	if err != nil {
		t.Fatalf("resolve absolute local path: %v", err)
	}

	explicitPath := filepath.Join(dir, "explicit.yaml")
	if err := os.WriteFile(explicitPath, []byte("version: 1\n\nrepos:\n  - path: explicit-repo\n"), 0o644); err != nil {
		t.Fatalf("write explicit config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitRuntimeError, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config: "+wantLocalPath) {
		t.Fatalf("stdout = %q, want local config path %q", stdout.String(), wantLocalPath)
	}
	if !strings.Contains(stdout.String(), "config is not valid") {
		t.Fatalf("stdout = %q, want invalid selected config", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()

	code = Run([]string{"doctor", "--config", explicitPath}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d; stderr: %s", code, exitRuntimeError, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Config: "+explicitPath) {
		t.Fatalf("stdout = %q, want explicit config path %q", stdout.String(), explicitPath)
	}
	if strings.Contains(stdout.String(), wantLocalPath) {
		t.Fatalf("stdout = %q, want explicit config to override local config", stdout.String())
	}
}

func TestDoctorRejectsImplicitConfigWhenLocalAndHomeBothExist(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Chdir(dir)
	t.Setenv("HOME", home)

	if err := os.WriteFile(filepath.Join(dir, "kitout.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write local config: %v", err)
	}
	homePath := writeHomeCLIConfigFile(t, home, "version: 1\n")
	wantLocalPath, err := filepath.Abs("kitout.yaml")
	if err != nil {
		t.Fatalf("resolve absolute local path: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"doctor"}, nil, &stdout, &stderr)
	if code != exitRuntimeError {
		t.Fatalf("exit code = %d, want %d", code, exitRuntimeError)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if !strings.Contains(stderr.String(), "both local config "+wantLocalPath+" and home config "+homePath+" exist") {
		t.Fatalf("stderr = %q, want ambiguous config guidance", stderr.String())
	}
	if !strings.Contains(stderr.String(), "pass --config to choose one") {
		t.Fatalf("stderr = %q, want --config guidance", stderr.String())
	}
}

func TestDoctorJSONRejectsImplicitConfigWhenLocalAndHomeBothExist(t *testing.T) {
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

	code := Run([]string{"doctor", "--json"}, nil, &stdout, &stderr)
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
	if !strings.Contains(response.Error.Message, "pass --config to choose one") {
		t.Fatalf("message = %q, want --config guidance", response.Error.Message)
	}
}

func TestDoctorExitCodeAllowsWarnings(t *testing.T) {
	report := doctorReport{
		Summary: doctorSummary{Total: 1, Warn: 1},
	}

	if got := doctorExitCode(report); got != exitOK {
		t.Fatalf("exit code = %d, want %d", got, exitOK)
	}
}

func TestHumanRendererDoctorOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderDoctorReport(doctorReport{
		ConfigPath: "/tmp/kitout.yaml",
		Items: []doctorItem{
			{Name: "macOS", State: doctorOK, Message: "running on macOS"},
			{Name: "Homebrew", State: doctorFail, Message: "Homebrew is not available", Fix: "Install Homebrew."},
		},
		Summary: doctorSummary{Total: 2, OK: 1, Fail: 1},
	})

	for _, fragment := range []string{
		"Config: /tmp/kitout.yaml",
		"Doctor:",
		"ok:   macOS",
		"fail: Homebrew",
		"fix: Install Homebrew.",
		"2 total, 1 ok, 0 warnings, 1 failed",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestJSONRendererDoctorOutput(t *testing.T) {
	var stdout bytes.Buffer
	renderer := newJSONRenderer(&stdout)

	err := renderer.renderDoctorReport(doctorReport{
		ConfigPath: "/tmp/kitout.yaml",
		Items: []doctorItem{
			{Name: "Config", State: doctorOK, Message: "config is valid", Details: map[string]string{"path": "/tmp/kitout.yaml"}},
		},
		Summary: doctorSummary{Total: 1, OK: 1},
	})
	if err != nil {
		t.Fatalf("renderDoctorReport returned error: %v", err)
	}

	var response doctorJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode JSON output %q: %v", stdout.String(), err)
	}
	if response.Command != "doctor" || !response.OK {
		t.Fatalf("response = %+v, want ok doctor response", response)
	}
	if response.Config == nil || response.Config.Path != "/tmp/kitout.yaml" || !response.Config.Valid {
		t.Fatalf("config = %+v, want valid config path", response.Config)
	}
	if response.Doctor == nil || response.Doctor.Summary.OK != 1 {
		t.Fatalf("doctor = %+v, want one ok check", response.Doctor)
	}
}

func healthyDoctorInfo(t *testing.T, arch string) doctorSystemInfo {
	t.Helper()

	pathValue := "/usr/bin:/bin"
	switch arch {
	case "arm64":
		pathValue = "/opt/homebrew/bin:" + pathValue
	case "amd64":
		pathValue = "/usr/local/bin:" + pathValue
	}

	return doctorSystemInfo{
		OS:    "darwin",
		Arch:  arch,
		Shell: executablePath(t),
		Path:  pathValue,
	}
}

func executablePath(t *testing.T) string {
	t.Helper()

	path, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable: %v", err)
	}
	return path
}

func assertDoctorItem(t *testing.T, report doctorReport, name string, state doctorState, messageFragment string) {
	t.Helper()

	item := doctorItemByName(t, report, name)
	if item.State != state {
		t.Fatalf("%s state = %q, want %q", name, item.State, state)
	}
	if !strings.Contains(item.Message, messageFragment) {
		t.Fatalf("%s message = %q, want fragment %q", name, item.Message, messageFragment)
	}
}

func doctorItemByName(t *testing.T, report doctorReport, name string) doctorItem {
	t.Helper()

	for _, item := range report.Items {
		if item.Name == name {
			return item
		}
	}
	t.Fatalf("doctor item %q was not found in %+v", name, report.Items)
	return doctorItem{}
}

type fakeDoctorRunner struct {
	calls     []doctorCommandCall
	responses []fakeDoctorResponse
}

type doctorCommandCall struct {
	name string
	args []string
}

type fakeDoctorResponse struct {
	result platform.CommandResult
	err    error
}

func (runner *fakeDoctorRunner) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	runner.calls = append(runner.calls, doctorCommandCall{
		name: name,
		args: append([]string(nil), args...),
	})

	if len(runner.responses) == 0 {
		return doctorCommandResult(name, args, ""), nil
	}

	response := runner.responses[0]
	runner.responses = runner.responses[1:]
	if response.result.Name == "" {
		response.result = doctorCommandResult(name, args, "")
	}
	return response.result, response.err
}

func expectDoctorCalls(t *testing.T, got, want []doctorCommandCall) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func doctorCommandResult(name string, args []string, stdout string) platform.CommandResult {
	return platform.CommandResult{
		Name:     name,
		Args:     append([]string(nil), args...),
		Stdout:   stdout,
		ExitCode: 0,
	}
}

func doctorCommandError(name string, args []string) platform.CommandError {
	return platform.CommandError{
		Result: platform.CommandResult{
			Name:     name,
			Args:     append([]string(nil), args...),
			ExitCode: 1,
		},
		Err: errors.New("command failed"),
	}
}

type doctorJSONResponse struct {
	Command string            `json:"command"`
	OK      bool              `json:"ok"`
	Config  *statusJSONConfig `json:"config"`
	Doctor  *doctorJSONReport `json:"doctor"`
	Error   *statusJSONError  `json:"error"`
}

type doctorJSONReport struct {
	Summary doctorSummary    `json:"summary"`
	Items   []doctorJSONItem `json:"items"`
}

type doctorJSONItem struct {
	Name    string            `json:"name"`
	State   string            `json:"state"`
	Message string            `json:"message"`
	Fix     string            `json:"fix"`
	Details map[string]string `json:"details"`
}
