package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/platform"
)

type doctorState string

const (
	doctorOK   doctorState = "ok"
	doctorWarn doctorState = "warn"
	doctorFail doctorState = "fail"
)

type doctorReport struct {
	ConfigPath     string
	ConfigWarnings []config.ConfigWarning
	Items          []doctorItem
	Summary        doctorSummary
}

type doctorItem struct {
	Name    string
	State   doctorState
	Message string
	Fix     string
	Details map[string]string
}

type doctorSummary struct {
	Total int `json:"total"`
	OK    int `json:"ok"`
	Warn  int `json:"warn"`
	Fail  int `json:"fail"`
}

type doctorSystemInfo struct {
	OS    string
	Arch  string
	Shell string
	Path  string
}

type doctorChecker struct {
	runner platform.Runner
	info   doctorSystemInfo
}

const homebrewMetadataStaleAfter = 14 * 24 * time.Hour

func runDoctor(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	configPath, err := config.SelectPath(opts.configPath)
	if err != nil {
		renderer := newHumanRenderer(stdout, stderr, opts)
		jsonRenderer := newJSONRenderer(stdout)
		return renderConfigError("doctor", err, opts, renderer, jsonRenderer, stderr)
	}

	renderer := newHumanRenderer(stdout, stderr, opts)
	if !opts.json {
		renderer.renderDoctorStart(configPath)
	}

	checker := newDoctorChecker(newCLIExecRunner(), doctorSystemInfo{
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
		Shell: os.Getenv("SHELL"),
		Path:  os.Getenv("PATH"),
	})
	report := checker.Check(context.Background(), configPath)

	if opts.json {
		if err := newJSONRenderer(stdout).renderDoctorReport(report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return doctorExitCode(report)
	}

	humanReport := report
	humanReport.ConfigPath = ""
	renderer.renderDoctorReport(humanReport)
	return doctorExitCode(report)
}

func newDoctorChecker(runner platform.Runner, info doctorSystemInfo) doctorChecker {
	return doctorChecker{
		runner: runner,
		info:   info,
	}
}

func (checker doctorChecker) Check(ctx context.Context, configPath string) doctorReport {
	configItem, loaded, configOK := checker.checkConfig(configPath)
	xcodeItem := checker.checkCommand(ctx, "Xcode Command Line Tools", "xcode-select", []string{"-p"}, "Install them with `xcode-select --install`.")
	homebrewItem := checker.checkCommand(ctx, "Homebrew", "brew", []string{"--version"}, "Install Homebrew, then rerun `kitout doctor`.")
	homebrewPathItem := checker.checkHomebrewPath(ctx, homebrewItem)
	homebrewFreshnessItem := checker.checkHomebrewFreshness(ctx, homebrewItem)
	gitItem := checker.checkCommand(ctx, "Git", "git", []string{"--version"}, "Install Git or the Xcode Command Line Tools, then rerun `kitout doctor`.")
	shellItem := checker.checkShellEnvironment()
	report := doctorReport{
		ConfigPath:     configPath,
		ConfigWarnings: loaded.Warnings,
		Items: []doctorItem{
			checker.checkOS(),
			checker.checkArchitecture(),
			xcodeItem,
			homebrewItem,
			homebrewPathItem,
			homebrewFreshnessItem,
			gitItem,
			shellItem,
			configItem,
		},
	}
	if configOK {
		report.Items = append(report.Items, checker.checkPathPermissions(loaded.Config))
	}

	for _, item := range report.Items {
		report.Summary.add(item)
	}

	return report
}

func (checker doctorChecker) checkHomebrewPath(ctx context.Context, homebrewItem doctorItem) doctorItem {
	if homebrewItem.State != doctorOK {
		return doctorItem{
			Name:    "Homebrew path",
			State:   doctorWarn,
			Message: "skipped because Homebrew is not available",
			Fix:     "Install Homebrew, then rerun `kitout doctor`.",
		}
	}

	result, err := checker.runner.Run(ctx, "brew", "--prefix")
	if err != nil {
		return doctorItem{
			Name:    "Homebrew path",
			State:   doctorFail,
			Message: "Homebrew prefix could not be detected",
			Fix:     "Run `brew doctor`, then rerun `kitout doctor`.",
			Details: map[string]string{
				"command": "brew --prefix",
				"error":   err.Error(),
			},
		}
	}

	prefix := strings.TrimSpace(result.Stdout)
	if prefix == "" {
		return doctorItem{
			Name:    "Homebrew path",
			State:   doctorWarn,
			Message: "Homebrew prefix was empty",
			Fix:     "Run `brew doctor`, then rerun `kitout doctor`.",
			Details: map[string]string{"command": "brew --prefix"},
		}
	}

	expected := expectedHomebrewPrefix(checker.info.Arch)
	if expected != "" && prefix != expected {
		return doctorItem{
			Name:    "Homebrew path",
			State:   doctorWarn,
			Message: fmt.Sprintf("Homebrew prefix is %s; expected %s for this architecture", prefix, expected),
			Fix:     fmt.Sprintf("Use the Homebrew installation at %s or update PATH before running Kitout.", expected),
			Details: map[string]string{
				"command":  "brew --prefix",
				"prefix":   prefix,
				"expected": expected,
			},
		}
	}

	return doctorItem{
		Name:    "Homebrew path",
		State:   doctorOK,
		Message: fmt.Sprintf("Homebrew prefix is %s", prefix),
		Details: map[string]string{
			"command": "brew --prefix",
			"prefix":  prefix,
		},
	}
}

func (checker doctorChecker) checkHomebrewFreshness(ctx context.Context, homebrewItem doctorItem) doctorItem {
	if homebrewItem.State != doctorOK {
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "skipped because Homebrew is not available",
			Fix:     "Install Homebrew, then rerun `kitout doctor`.",
		}
	}

	repositoryResult, err := checker.runner.Run(ctx, "brew", "--repository")
	details := map[string]string{"repository_command": "brew --repository"}
	if err != nil {
		details["error"] = err.Error()
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "Homebrew repository could not be detected",
			Fix:     "Run `brew doctor`, then rerun `kitout doctor`.",
			Details: details,
		}
	}

	repository := strings.TrimSpace(repositoryResult.Stdout)
	if repository == "" {
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "Homebrew repository path was empty",
			Fix:     "Run `brew doctor`, then rerun `kitout doctor`.",
			Details: details,
		}
	}
	details["repository"] = repository

	gitArgs := []string{"-C", repository, "log", "-1", "--format=%ct"}
	details["command"] = commandString("git", gitArgs)
	commitResult, err := checker.runner.Run(ctx, "git", gitArgs...)
	if err != nil {
		details["error"] = err.Error()
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "Homebrew metadata age could not be checked",
			Fix:     "Run `brew update` when you want to refresh Homebrew and taps.",
			Details: details,
		}
	}

	updatedAt, err := parseUnixTimestamp(strings.TrimSpace(commitResult.Stdout))
	if err != nil {
		details["error"] = err.Error()
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "Homebrew metadata age could not be parsed",
			Fix:     "Run `brew update` when you want to refresh Homebrew and taps.",
			Details: details,
		}
	}

	age := time.Since(updatedAt)
	if age < 0 {
		age = 0
	}
	details["last_updated_at"] = updatedAt.Format(time.RFC3339)
	details["age"] = describeHomebrewMetadataAge(age)

	if age > homebrewMetadataStaleAfter {
		return doctorItem{
			Name:    "Homebrew freshness",
			State:   doctorWarn,
			Message: "Homebrew metadata appears stale; last updated " + describeHomebrewMetadataAge(age),
			Fix:     "Run `brew update` when you want to refresh Homebrew and taps.",
			Details: details,
		}
	}

	return doctorItem{
		Name:    "Homebrew freshness",
		State:   doctorOK,
		Message: "Homebrew metadata looks current; last updated " + describeHomebrewMetadataAge(age),
		Details: details,
	}
}

func parseUnixTimestamp(value string) (time.Time, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse unix timestamp %q: %w", value, err)
	}
	return time.Unix(seconds, 0), nil
}

func describeHomebrewMetadataAge(age time.Duration) string {
	days := int(age.Hours() / 24)
	switch days {
	case 0:
		return "today"
	case 1:
		return "1 day ago"
	default:
		return fmt.Sprintf("%d days ago", days)
	}
}

func expectedHomebrewPrefix(arch string) string {
	switch arch {
	case "arm64":
		return "/opt/homebrew"
	case "amd64":
		return "/usr/local"
	default:
		return ""
	}
}

func (checker doctorChecker) checkShellEnvironment() doctorItem {
	shell := strings.TrimSpace(checker.info.Shell)
	pathValue := strings.TrimSpace(checker.info.Path)
	details := map[string]string{
		"shell": shell,
		"path":  pathValue,
	}

	if shell == "" {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: "SHELL is not set",
			Fix:     "Set SHELL to your login shell before running Kitout.",
			Details: details,
		}
	}
	info, err := os.Stat(shell)
	if err != nil {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: fmt.Sprintf("configured shell %s is not available", shell),
			Fix:     "Set SHELL to an existing shell path before running Kitout.",
			Details: details,
		}
	}
	if info.IsDir() {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: fmt.Sprintf("configured shell %s is a directory", shell),
			Fix:     "Set SHELL to an executable shell path before running Kitout.",
			Details: details,
		}
	}
	if info.Mode().Perm()&0o111 == 0 {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: fmt.Sprintf("configured shell %s is not executable", shell),
			Fix:     "Set SHELL to an executable shell path before running Kitout.",
			Details: details,
		}
	}
	if pathValue == "" {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: "PATH is not set",
			Fix:     "Set PATH before running Kitout so external tools can be found.",
			Details: details,
		}
	}

	expectedBrewBin := expectedHomebrewBin(checker.info.Arch)
	if expectedBrewBin != "" && !pathContains(pathValue, expectedBrewBin) {
		return doctorItem{
			Name:    "Shell environment",
			State:   doctorWarn,
			Message: fmt.Sprintf("PATH does not include %s", expectedBrewBin),
			Fix:     fmt.Sprintf("Add %s to PATH before running Kitout.", expectedBrewBin),
			Details: details,
		}
	}

	return doctorItem{
		Name:    "Shell environment",
		State:   doctorOK,
		Message: "SHELL and PATH look usable",
		Details: details,
	}
}

func expectedHomebrewBin(arch string) string {
	prefix := expectedHomebrewPrefix(arch)
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "bin")
}

func pathContains(pathValue, dir string) bool {
	for _, entry := range filepath.SplitList(pathValue) {
		if entry == dir {
			return true
		}
	}
	return false
}

func (checker doctorChecker) checkOS() doctorItem {
	if checker.info.OS == "darwin" {
		return doctorItem{
			Name:    "macOS",
			State:   doctorOK,
			Message: "running on macOS",
			Details: map[string]string{"os": checker.info.OS},
		}
	}

	return doctorItem{
		Name:    "macOS",
		State:   doctorFail,
		Message: fmt.Sprintf("unsupported OS %q", checker.info.OS),
		Fix:     "Run Kitout on macOS.",
		Details: map[string]string{"os": checker.info.OS},
	}
}

func (checker doctorChecker) checkArchitecture() doctorItem {
	switch checker.info.Arch {
	case "arm64":
		return doctorItem{
			Name:    "CPU architecture",
			State:   doctorOK,
			Message: "running on Apple Silicon",
			Details: map[string]string{"arch": checker.info.Arch},
		}
	case "amd64":
		return doctorItem{
			Name:    "CPU architecture",
			State:   doctorWarn,
			Message: "running on Intel architecture; MVP support is focused on Apple Silicon",
			Details: map[string]string{"arch": checker.info.Arch},
		}
	default:
		return doctorItem{
			Name:    "CPU architecture",
			State:   doctorWarn,
			Message: fmt.Sprintf("untested architecture %q", checker.info.Arch),
			Details: map[string]string{"arch": checker.info.Arch},
		}
	}
}

func (checker doctorChecker) checkCommand(ctx context.Context, label, name string, args []string, fix string) doctorItem {
	result, err := checker.runner.Run(ctx, name, args...)
	if err != nil {
		return doctorItem{
			Name:    label,
			State:   doctorFail,
			Message: fmt.Sprintf("%s is not available", label),
			Fix:     fix,
			Details: map[string]string{
				"command": commandString(name, args),
				"error":   err.Error(),
			},
		}
	}

	return doctorItem{
		Name:    label,
		State:   doctorOK,
		Message: commandSuccessMessage(label, result.Stdout),
		Details: map[string]string{
			"command": commandString(name, args),
		},
	}
}

func (checker doctorChecker) checkConfig(path string) (doctorItem, config.LoadedConfig, bool) {
	resolvedPath, err := config.ResolvePath(path)
	if err != nil {
		return doctorItem{
			Name:    "Config",
			State:   doctorFail,
			Message: "config path could not be resolved",
			Fix:     "Pass a readable config path with `--config`.",
			Details: map[string]string{"error": err.Error()},
		}, config.LoadedConfig{}, false
	}

	loaded, err := config.LoadFile(resolvedPath)
	if err != nil {
		return doctorItem{
			Name:    "Config",
			State:   doctorFail,
			Message: "config is not valid",
			Fix:     "Fix the config file or create one with `kitout init`.",
			Details: map[string]string{
				"path":  resolvedPath,
				"error": err.Error(),
			},
		}, config.LoadedConfig{}, false
	}

	if len(loaded.Warnings) > 0 {
		return doctorItem{
			Name:    "Config",
			State:   doctorWarn,
			Message: "config is valid, but " + loaded.Warnings[0].Message,
			Fix:     "Update the config to remove this warning.",
			Details: map[string]string{
				"path":    resolvedPath,
				"warning": loaded.Warnings[0].Message,
			},
		}, loaded, true
	}

	return doctorItem{
		Name:    "Config",
		State:   doctorOK,
		Message: "config is valid",
		Details: map[string]string{"path": resolvedPath},
	}, loaded, true
}

func (checker doctorChecker) checkPathPermissions(cfg config.Config) doctorItem {
	targets := configuredWriteTargets(cfg)
	if len(targets) == 0 {
		return doctorItem{
			Name:    "Path permissions",
			State:   doctorOK,
			Message: "no configured filesystem write targets",
		}
	}

	failures := make([]string, 0)
	for _, target := range targets {
		if err := checkWritableTarget(target.Path, target.Kind); err != nil {
			failures = append(failures, fmt.Sprintf("%s %s: %v", target.Kind, target.Path, err))
		}
	}

	if len(failures) > 0 {
		return doctorItem{
			Name:    "Path permissions",
			State:   doctorFail,
			Message: fmt.Sprintf("%d configured write target(s) may not be writable", len(failures)),
			Fix:     "Fix ownership or permissions for the listed paths, then rerun `kitout doctor`.",
			Details: map[string]string{
				"targets":  strings.Join(targetDescriptions(targets), "\n"),
				"failures": strings.Join(failures, "\n"),
			},
		}
	}

	return doctorItem{
		Name:    "Path permissions",
		State:   doctorOK,
		Message: fmt.Sprintf("%d configured write target(s) look writable", len(targets)),
		Details: map[string]string{
			"targets": strings.Join(targetDescriptions(targets), "\n"),
		},
	}
}

type pathPermissionTarget struct {
	Kind string
	Path string
}

func configuredWriteTargets(cfg config.Config) []pathPermissionTarget {
	targets := make([]pathPermissionTarget, 0,
		len(cfg.Directories)+len(cfg.ASDF.ToolVersions)+len(cfg.Repos)+len(cfg.Copies)+len(cfg.ExpandedSymlinks()))

	for _, path := range cfg.Directories {
		targets = append(targets, pathPermissionTarget{Kind: "directory", Path: path})
	}
	for _, item := range cfg.ASDF.ToolVersions {
		targets = append(targets, pathPermissionTarget{Kind: ".tool-versions", Path: item.Path})
	}
	for _, repo := range cfg.Repos {
		targets = append(targets, pathPermissionTarget{Kind: "repo", Path: repo.Path})
	}
	for _, copy := range cfg.Copies {
		targets = append(targets, pathPermissionTarget{Kind: "copy", Path: copy.Target})
	}
	for _, symlink := range cfg.ExpandedSymlinks() {
		targets = append(targets, pathPermissionTarget{Kind: "symlink", Path: symlink.Target})
	}

	return targets
}

func targetDescriptions(targets []pathPermissionTarget) []string {
	descriptions := make([]string, 0, len(targets))
	for _, target := range targets {
		descriptions = append(descriptions, target.Kind+" "+target.Path)
	}
	return descriptions
}

func checkWritableTarget(path, kind string) error {
	if path == "" {
		return errors.New("path is required")
	}

	info, err := os.Lstat(path)
	if err == nil {
		switch {
		case kind == ".tool-versions" && info.Mode().IsRegular():
			return requireWritable(path)
		case kind == "directory" && info.IsDir():
			return requireWritable(path)
		case kind == "repo" && info.IsDir():
			return requireWritable(path)
		case kind == "symlink":
			return requireWritable(filepath.Dir(path))
		default:
			return requireWritable(filepath.Dir(path))
		}
	}
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	parent, err := nearestExistingParent(path)
	if err != nil {
		return err
	}
	return requireWritable(parent)
}

func nearestExistingParent(path string) (string, error) {
	parent := filepath.Dir(path)
	for {
		if parent == "" || parent == "." {
			return "", fmt.Errorf("no existing parent found for %s", path)
		}

		info, err := os.Stat(parent)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("parent %s is not a directory", parent)
			}
			return parent, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		next := filepath.Dir(parent)
		if next == parent {
			return "", fmt.Errorf("no existing parent found for %s", path)
		}
		parent = next
	}
}

func requireWritable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o222 == 0 {
		return fmt.Errorf("%s is not writable", path)
	}
	return nil
}

func commandSuccessMessage(label, stdout string) string {
	firstLine := strings.TrimSpace(strings.Split(strings.TrimSpace(stdout), "\n")[0])
	if firstLine == "" {
		return fmt.Sprintf("%s is available", label)
	}

	return firstLine
}

func commandString(name string, args []string) string {
	parts := append([]string{name}, args...)
	return strings.Join(parts, " ")
}

func (summary *doctorSummary) add(item doctorItem) {
	summary.Total++

	switch item.State {
	case doctorOK:
		summary.OK++
	case doctorWarn:
		summary.Warn++
	case doctorFail:
		summary.Fail++
	}
}

func (report doctorReport) HasFailures() bool {
	return report.Summary.Fail > 0
}

func doctorExitCode(report doctorReport) int {
	if report.HasFailures() {
		return exitRuntimeError
	}
	return exitOK
}
