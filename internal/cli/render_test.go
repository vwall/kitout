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
		"Config: /tmp/kitout.yaml\n\n",
		"✓ satisfied directory: /tmp/code satisfied",
		"! missing   brew: git",
		"! changed   brew: go",
		"Summary: 1 satisfied, 1 missing, 1 changed",
		"2 resources need attention",
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
	directoryLine := findLineContaining(t, lines, "directory: /tmp/code")
	missingBrewLine := findLineContaining(t, lines, "brew: git")
	changedBrewLine := findLineContaining(t, lines, "brew: go")
	resourceColumn := visualColumn(directoryLine, "directory:")
	for _, line := range []string{missingBrewLine, changedBrewLine} {
		if got := visualColumn(line, "brew:"); got != resourceColumn {
			t.Fatalf("resource column = %d in %q, want %d", got, line, resourceColumn)
		}
	}
	messageColumn := visualLastColumn(directoryLine, "satisfied")
	for _, line := range []string{missingBrewLine, changedBrewLine} {
		var got int
		if strings.Contains(line, "missing") {
			got = visualLastColumn(line, "missing")
		} else {
			got = visualColumn(line, "formula")
		}
		if got != messageColumn {
			t.Fatalf("message column = %d in %q, want %d", got, line, messageColumn)
		}
	}
}

func TestHumanRendererShowsASDFMissingVersions(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	item := engine.PlanItem{
		ResourceID: "asdf_plugin:ruby",
		Type:       "asdf_plugin",
		State:      engine.StateMissing,
		Action:     engine.ActionApply,
		Message:    "asdf version is missing",
		Details: map[string]string{
			"name":             "ruby",
			"missing_versions": "4.0.5",
		},
	}
	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items:   []engine.PlanItem{item},
		Summary: engine.PlanSummary{Total: 1, Missing: 1, ToApply: 1},
	})
	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items:   []engine.PlanItem{item},
		Summary: engine.PlanSummary{Total: 1, Missing: 1, ToApply: 1},
	})
	renderer.BeforeApply(item)

	for _, fragment := range []string{
		"! missing   asdf_plugin: ruby 4.0.5",
		"i Would install asdf version ruby 4.0.5",
		"> Installing asdf version ruby 4.0.5...",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHumanRendererAlignsMessageColumnsAcrossLongResourceIDs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "brew:git", Type: "brew", State: engine.StateChanged, Message: "formula is outdated"},
			{ResourceID: "directory:/tmp/kitout-long/.config", Type: "directory", State: engine.StateSatisfied, Message: "directory exists"},
		},
	})

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) < 3 {
		t.Fatalf("stdout = %q, want status lines", stdout.String())
	}

	changedLine := findLineContaining(t, lines, "formula is outdated")
	directoryLine := findLineContaining(t, lines, "directory: /tmp/kitout-long/.config")
	want := visualColumn(changedLine, "formula is outdated")
	if got := visualLastColumn(directoryLine, "satisfied"); got != want {
		t.Fatalf("message column = %d in %q, want %d from %q", got, directoryLine, want, changedLine)
	}
	if !strings.Contains(stdout.String(), "! changed   brew: git") {
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
			{ResourceID: "symlink:/Users/example/.zshrc", Action: engine.ActionApply, Message: "symlink points elsewhere"},
		},
		Summary: engine.PlanSummary{Total: 2, Missing: 2, ToApply: 2},
	})
	renderer.renderApplyReport("/tmp/kitout.yaml", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "brew:git", Changed: true, Message: "installed formula"},
			{ResourceID: "directory:/Users/example/.config", Action: "noop", Message: "directory exists"},
		},
		Summary: engine.ApplySummary{Total: 2, Changed: 1, Noop: 1},
	})
	if !strings.Contains(stdout.String(), "Config: /tmp/kitout.yaml\n\ni Would install cask ghostty") {
		t.Fatalf("stdout = %q, want compact dry-run plan", stdout.String())
	}
	if !strings.Contains(stdout.String(), "i Would link ~/.zshrc") {
		t.Fatalf("stdout = %q, want symlink dry-run sentence", stdout.String())
	}
	if !strings.Contains(stdout.String(), "No shell commands will run without explicit approval.") {
		t.Fatalf("stdout = %q, want dry-run safety message", stdout.String())
	}

	lines := strings.Split(stdout.String(), "\n")
	applyShort := findLineContaining(t, lines, "brew: git")
	applyLong := findLineContaining(t, lines, "directory: ~/.config")
	if want := visualColumn(applyShort, "installed formula"); visualColumn(applyLong, "directory exists") != want {
		t.Fatalf("apply lines are not message-aligned:\n%q\n%q", applyShort, applyLong)
	}
}

func TestHumanRendererApplyProgressOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderApplyStart("/tmp/kitout.yaml")
	renderer.BeforeApply(engine.PlanItem{
		ResourceID: "brew:go",
		Type:       "brew",
		State:      engine.StateChanged,
		Action:     engine.ActionApply,
		Details:    map[string]string{"name": "go"},
	})
	renderer.renderApplyReport("", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "brew:go", Type: "brew", Action: "upgrade", Changed: true, Message: "upgraded formula", Details: map[string]string{"name": "go"}},
		},
		Summary: engine.ApplySummary{Total: 1, Changed: 1},
	})

	for _, fragment := range []string{
		"Config: /tmp/kitout.yaml\n\nApplying changes:",
		"> Upgrading formula go...",
		"\nResults:\n",
		"✓ done  brew: go",
		"Summary: 1 changed",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
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
		ansiGreen + "✓ satisfied" + ansiReset,
		ansiYellow + "! missing  " + ansiReset,
		ansiYellow + "! changed  " + ansiReset,
		ansiRed + "× fail     " + ansiReset,
		ansiCyan + "- skip     " + ansiReset,
		ansiBlue + "i" + ansiReset,
		ansiGreen + "✓ done " + ansiReset,
		ansiYellow + "warn:" + ansiReset,
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

func visualColumn(line, fragment string) int {
	index := strings.Index(line, fragment)
	if index < 0 {
		return -1
	}
	return displayWidth(line[:index])
}

func visualLastColumn(line, fragment string) int {
	index := strings.LastIndex(line, fragment)
	if index < 0 {
		return -1
	}
	return displayWidth(line[:index])
}
