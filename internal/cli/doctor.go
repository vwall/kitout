package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
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
	report := doctorReport{
		ConfigPath: configPath,
		Items: []doctorItem{
			checker.checkOS(),
			checker.checkArchitecture(),
			checker.checkCommand(ctx, "Xcode Command Line Tools", "xcode-select", []string{"-p"}, "Install them with `xcode-select --install`."),
			checker.checkCommand(ctx, "Homebrew", "brew", []string{"--version"}, "Install Homebrew, then rerun `kitout doctor`."),
			checker.checkCommand(ctx, "Git", "git", []string{"--version"}, "Install Git or the Xcode Command Line Tools, then rerun `kitout doctor`."),
			checker.checkConfig(configPath),
		},
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

func (checker doctorChecker) checkConfig(path string) doctorItem {
	resolvedPath, err := config.ResolvePath(path)
	if err != nil {
		return doctorItem{
			Name:    "Config",
			State:   doctorFail,
			Message: "config path could not be resolved",
			Fix:     "Pass a readable config path with `--config`.",
			Details: map[string]string{"error": err.Error()},
		}
	}

	_, err = config.LoadFile(resolvedPath)
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
		}
	}

	return doctorItem{
		Name:    "Config",
		State:   doctorOK,
		Message: "config is valid",
		Details: map[string]string{"path": resolvedPath},
	}
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
