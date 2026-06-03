package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
)

func TestHumanRendererStatusOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "directory:/tmp/code", Type: "directory", State: engine.StateSatisfied, Action: engine.ActionNoop, Message: "directory exists"},
			{ResourceID: "brew:git", Type: "brew", State: engine.StateMissing, Action: engine.ActionApply, Message: "formula is missing"},
			{ResourceID: "brew:go", Type: "brew", State: engine.StateChanged, Action: engine.ActionApply, Message: "formula is outdated"},
		},
		Summary: engine.PlanSummary{
			Total:     3,
			Satisfied: 1,
			Missing:   1,
			Changed:   1,
			ToApply:   2,
		},
	})

	for _, fragment := range []string{
		"Config: /tmp/kitout.yaml",
		"directory:/tmp/code directory exists",
		"need      brew:git",
		"outdated: brew:go",
		"3 total, 1 satisfied, 1 missing, 1 changed",
		"2 changes needed",
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(fragment)) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) < 4 {
		t.Fatalf("stdout = %q, want status lines", stdout.String())
	}
	resourceColumn := strings.Index(lines[1], "directory:/tmp/code")
	for _, line := range lines[2:4] {
		if got := strings.Index(line, "brew:"); got != resourceColumn {
			t.Fatalf("resource column = %d in %q, want %d", got, line, resourceColumn)
		}
	}
	messageColumn := strings.Index(lines[1], "directory exists")
	for _, line := range []string{lines[2], lines[3]} {
		if got := strings.Index(line, "formula"); got != messageColumn {
			t.Fatalf("message column = %d in %q, want %d", got, line, messageColumn)
		}
	}
}

func TestHumanRendererAlignsMessageColumnsAcrossLongResourceIDs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "brew:git", Type: "brew", State: engine.StateChanged, Message: "formula is outdated"},
			{ResourceID: "directory:/Users/nix/.config", Type: "directory", State: engine.StateSatisfied, Message: "directory exists"},
		},
	})

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) < 3 {
		t.Fatalf("stdout = %q, want status lines", stdout.String())
	}

	want := strings.Index(lines[1], "formula is outdated")
	if got := strings.Index(lines[2], "directory exists"); got != want {
		t.Fatalf("message column = %d in %q, want %d from %q", got, lines[2], want, lines[1])
	}
	if !strings.Contains(stdout.String(), "outdated: brew:git                     formula is outdated") {
		t.Fatalf("stdout = %q, want brew row padded to message column", stdout.String())
	}
}

func TestHumanRendererAlignsDryRunAndApplyMessageColumns(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "cask:ghostty", Action: engine.ActionApply, Message: "cask is missing"},
			{ResourceID: "symlink:/Users/nix/.zshrc", Action: engine.ActionApply, Message: "symlink points elsewhere"},
		},
	})
	renderer.renderApplyReport("/tmp/kitout.yaml", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "brew:git", Changed: true, Message: "installed formula"},
			{ResourceID: "directory:/Users/nix/.config", Action: "noop", Message: "directory exists"},
		},
	})

	lines := strings.Split(stdout.String(), "\n")
	dryRunShort := findLineContaining(t, lines, "cask:ghostty")
	dryRunLong := findLineContaining(t, lines, "symlink:/Users/nix/.zshrc")
	if want := strings.Index(dryRunShort, "cask is missing"); strings.Index(dryRunLong, "symlink points elsewhere") != want {
		t.Fatalf("dry-run lines are not message-aligned:\n%q\n%q", dryRunShort, dryRunLong)
	}

	applyShort := findLineContaining(t, lines, "brew:git")
	applyLong := findLineContaining(t, lines, "directory:/Users/nix/.config")
	if want := strings.Index(applyShort, "installed formula"); strings.Index(applyLong, "directory exists") != want {
		t.Fatalf("apply lines are not message-aligned:\n%q\n%q", applyShort, applyLong)
	}
}

func TestHumanRendererQuietSuppressesStatusOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{quiet: true})

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{})

	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHumanRendererColorsHumanMarkersWhenEnabled(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})
	renderer.color = true

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "directory:/tmp/code", Type: "directory", State: engine.StateSatisfied, Message: "directory exists"},
			{ResourceID: "brew:git", Type: "brew", State: engine.StateMissing, Message: "formula is missing"},
			{ResourceID: "brew:go", Type: "brew", State: engine.StateChanged, Message: "formula is outdated"},
			{ResourceID: "shell:setup", Type: "shell", State: engine.StateFailed, Message: "command failed"},
			{ResourceID: "repo:kitout", Type: "repo", State: engine.StateSkipped, Message: "skipped"},
		},
	})
	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "directory:/tmp/code", Action: engine.ActionApply, Message: "directory is missing"},
			{ResourceID: "shell:broken", Action: engine.ActionFail, Message: "status failed"},
			{ResourceID: "repo:kitout", Action: engine.ActionSkip, Message: "skipped"},
		},
	})
	renderer.renderApplyReport("/tmp/kitout.yaml", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "directory:/tmp/code", Changed: true, Message: "created directory"},
			{ResourceID: "brew:git", Action: "noop", Message: "formula is installed"},
			{ResourceID: "repo:kitout", Action: "skip", Message: "skipped"},
			{ResourceID: "shell:setup", Error: "command failed", Message: "failed"},
		},
	})
	renderer.renderDoctorReport(doctorReport{
		ConfigPath: "/tmp/kitout.yaml",
		Items: []doctorItem{
			{Name: "macOS", State: doctorOK, Message: "running on macOS"},
			{Name: "CPU architecture", State: doctorWarn, Message: "Intel architecture"},
			{Name: "Homebrew", State: doctorFail, Message: "Homebrew is not available"},
		},
	})

	for _, fragment := range []string{
		ansiGreen + "ok       " + ansiReset,
		ansiYellow + "need     " + ansiReset,
		ansiYellow + "outdated:" + ansiReset,
		ansiRed + "fail     " + ansiReset,
		ansiCyan + "skip     " + ansiReset,
		ansiYellow + "apply" + ansiReset,
		ansiGreen + "done" + ansiReset,
		ansiYellow + "warn" + ansiReset,
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want color fragment %q", stdout.String(), fragment)
		}
	}
}

func TestHumanRendererInvalidConfigOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderInvalidConfigDetails(config.ValidationErrors{
		{Field: "version", Message: "is required"},
	})
	renderer.renderInvalidConfig(config.ParseError{
		Path: "/tmp/kitout.yaml",
		Err:  errors.New("unknown field"),
	})

	want := "Invalid config: version is required\nInvalid config: parse config /tmp/kitout.yaml: unknown field\n"
	if stderr.String() != want {
		t.Fatalf("stderr = %q, want %q", stderr.String(), want)
	}
	if stdout.String() != "" {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func findLineContaining(t *testing.T, lines []string, fragment string) string {
	t.Helper()

	for _, line := range lines {
		if strings.Contains(line, fragment) {
			return line
		}
	}
	t.Fatalf("line containing %q was not found in %#v", fragment, lines)
	return ""
}
