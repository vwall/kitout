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
	"strings"

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
	ConfigPath string
	Items      []doctorItem
	Summary    doctorSummary
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
	OS   string
	Arch string
}

type doctorChecker struct {
	runner platform.Runner
	info   doctorSystemInfo
}

func runDoctor(args []string, opts globalOptions, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(stderr)
	addGlobalFlags(fs, &opts)

	if err := fs.Parse(args); err != nil {
		return exitValidation
	}

	configPath := opts.configPath
	if configPath == "" {
		configPath = config.DefaultPath
	}

	checker := newDoctorChecker(platform.NewExecRunner(), doctorSystemInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	})
	report := checker.Check(context.Background(), configPath)

	if opts.json {
		if err := newJSONRenderer(stdout).renderDoctorReport(report); err != nil {
			fmt.Fprintf(stderr, "Failed to render JSON: %v\n", err)
			return exitRuntimeError
		}
		return doctorExitCode(report)
	}

	newHumanRenderer(stdout, stderr, opts).renderDoctorReport(report)
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
	report := doctorReport{
		ConfigPath: configPath,
		Items: []doctorItem{
			checker.checkOS(),
			checker.checkArchitecture(),
			checker.checkCommand(ctx, "Xcode Command Line Tools", "xcode-select", []string{"-p"}, "Install them with `xcode-select --install`."),
			checker.checkCommand(ctx, "Homebrew", "brew", []string{"--version"}, "Install Homebrew, then rerun `kitout doctor`."),
			checker.checkCommand(ctx, "Git", "git", []string{"--version"}, "Install Git or the Xcode Command Line Tools, then rerun `kitout doctor`."),
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
		len(cfg.Directories)+len(cfg.ASDF.ToolVersions)+len(cfg.Repos)+len(cfg.Symlinks))

	for _, path := range cfg.Directories {
		targets = append(targets, pathPermissionTarget{Kind: "directory", Path: path})
	}
	for _, item := range cfg.ASDF.ToolVersions {
		targets = append(targets, pathPermissionTarget{Kind: ".tool-versions", Path: item.Path})
	}
	for _, repo := range cfg.Repos {
		targets = append(targets, pathPermissionTarget{Kind: "repo", Path: repo.Path})
	}
	for _, symlink := range cfg.Symlinks {
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
