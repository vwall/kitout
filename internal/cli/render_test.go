package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
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
			{ResourceID: "symlink:/tmp/link", Type: "symlink", State: engine.StateChanged, Action: engine.ActionApply, Message: "symlink points elsewhere", Details: map[string]string{"target": "/tmp/link"}},
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
		"Results:\n",
		"✓ satisfied directory: /tmp/code satisfied",
		"! missing   brew: git",
		"! changed   symlink: /tmp/link",
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
	changedSymlinkLine := findLineContaining(t, lines, "symlink: /tmp/link")
	resourceColumn := visualColumn(directoryLine, "directory:")
	for _, line := range []string{missingBrewLine, changedSymlinkLine} {
		target := "brew:"
		if strings.Contains(line, "symlink:") {
			target = "symlink:"
		}
		if got := visualColumn(line, target); got != resourceColumn {
			t.Fatalf("resource column = %d in %q, want %d", got, line, resourceColumn)
		}
	}
	messageColumn := visualLastColumn(directoryLine, "satisfied")
	for _, line := range []string{missingBrewLine, changedSymlinkLine} {
		var got int
		if strings.Contains(line, "missing") {
			got = visualLastColumn(line, "missing")
		} else {
			got = visualColumn(line, "symlink points elsewhere")
		}
		if got != messageColumn {
			t.Fatalf("message column = %d in %q, want %d", got, line, messageColumn)
		}
	}
}

func TestHumanRendererStatusShowsAdvisoriesWithoutAttention(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "brew:git",
				Type:       "brew",
				State:      engine.StateSatisfied,
				Action:     engine.ActionNoop,
				Message:    "formula is installed",
				Details:    map[string]string{"name": "git"},
				Advisories: []engine.Advisory{{
					Code:     "homebrew_formula_outdated",
					Severity: engine.AdvisoryNotice,
					Message:  "formula update available for git",
					Fix:      "Run `brew upgrade git` when you want to update it.",
				}},
			},
		},
		Summary: engine.PlanSummary{Total: 1, Satisfied: 1, Advisories: 1},
	})

	for _, fragment := range []string{
		"✓ satisfied brew: git",
		"i brew: git: formula update available for git",
		"fix: Run `brew upgrade git` when you want to update it.",
		"Summary: 1 satisfied",
		"1 advisory",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if strings.Contains(stdout.String(), "needs attention") {
		t.Fatalf("stdout = %q, want advisories outside attention count", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestJSONRendererIncludesPlanAdvisories(t *testing.T) {
	var stdout bytes.Buffer
	renderer := newJSONRenderer(&stdout)

	err := renderer.renderPlan("status", "/tmp/kitout.yaml", nil, engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "cask:ghostty",
				Type:       "cask",
				State:      engine.StateSatisfied,
				Action:     engine.ActionNoop,
				Message:    "cask is installed",
				Advisories: []engine.Advisory{{
					Code:     "homebrew_cask_outdated",
					Severity: engine.AdvisoryNotice,
					Message:  "cask update available for ghostty",
					Fix:      "Run `brew upgrade --cask ghostty` when you want to update it.",
					Details:  map[string]string{"name": "ghostty"},
				}},
			},
		},
		Summary: engine.PlanSummary{Total: 1, Satisfied: 1, Advisories: 1},
	}, false)
	if err != nil {
		t.Fatalf("renderPlan returned error: %v", err)
	}

	var response statusJSONResponse
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil {
		t.Fatalf("decode JSON output %q: %v", stdout.String(), err)
	}
	if response.Plan.Summary.Advisories != 1 {
		t.Fatalf("advisories = %d, want 1", response.Plan.Summary.Advisories)
	}
	if got := response.Plan.Items[0].Advisories[0].Code; got != "homebrew_cask_outdated" {
		t.Fatalf("advisory code = %q, want homebrew_cask_outdated", got)
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
		"dry-run Would install asdf version ruby 4.0.5",
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
			{ResourceID: "copy:/tmp/kitout-target", Type: "copy", State: engine.StateChanged, Message: "copy target differs", Details: map[string]string{"target": "/tmp/kitout-target"}},
			{ResourceID: "directory:/tmp/kitout-long/.config", Type: "directory", State: engine.StateSatisfied, Message: "directory exists"},
		},
	})

	lines := strings.Split(stdout.String(), "\n")
	if len(lines) < 3 {
		t.Fatalf("stdout = %q, want status lines", stdout.String())
	}

	changedLine := findLineContaining(t, lines, "copy target differs")
	directoryLine := findLineContaining(t, lines, "directory: /tmp/kitout-long/.config")
	want := visualColumn(changedLine, "copy target differs")
	if got := visualLastColumn(directoryLine, "satisfied"); got != want {
		t.Fatalf("message column = %d in %q, want %d from %q", got, directoryLine, want, changedLine)
	}
	if !strings.Contains(stdout.String(), "! changed   copy: /tmp/kitout-target") {
		t.Fatalf("stdout = %q, want copy row padded to message column", stdout.String())
	}
}

func TestHumanRendererAlignsDryRunAndApplyMessageColumns(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})
	home := mustUserHome(t)

	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{ResourceID: "cask:ghostty", Action: engine.ActionApply, Message: "cask is missing"},
			{ResourceID: "symlink:" + home + "/.zshrc", Action: engine.ActionApply, Message: "symlink points elsewhere"},
		},
		Summary: engine.PlanSummary{Total: 2, Missing: 2, ToApply: 2},
	})
	renderer.renderApplyReport("/tmp/kitout.yaml", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "brew:git", Changed: true, Message: "installed formula"},
			{ResourceID: "directory:" + home + "/.config", Action: "noop", Message: "directory exists"},
		},
		Summary: engine.ApplySummary{Total: 2, Changed: 1, Noop: 1},
	})
	if !strings.Contains(stdout.String(), "Config: /tmp/kitout.yaml\n\n[dry-run] Previewing planned changes:\ndry-run Would install cask ghostty") {
		t.Fatalf("stdout = %q, want compact dry-run plan", stdout.String())
	}
	if !strings.Contains(stdout.String(), "dry-run Would link ~/.zshrc") {
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

func TestHumanRendererShowsLoginShellDryRunMessage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "login_shell:homebrew:fish",
				Type:       "login_shell",
				State:      engine.StateMissing,
				Action:     engine.ActionApply,
				Message:    "shell path is not listed in /etc/shells",
				Details: map[string]string{
					"path":                 "homebrew:fish",
					"resolved_path":        "/opt/homebrew/bin/fish",
					"add_to_etc_shells":    "true",
					"listed_in_etc_shells": "false",
				},
			},
		},
		Summary: engine.PlanSummary{Total: 1, Missing: 1, ToApply: 1},
	})

	if !strings.Contains(stdout.String(), "dry-run Would allow login shell /opt/homebrew/bin/fish and set it for the current user") {
		t.Fatalf("stdout = %q, want login shell dry-run message", stdout.String())
	}
}

func TestHumanRendererShowsSecuritySystemAndSSHKeyDryRunMessages(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})
	home := mustUserHome(t)

	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "security:filevault",
				Type:       "security",
				State:      engine.StateMissing,
				Action:     engine.ActionApply,
				Details:    map[string]string{"name": "filevault", "required": "true"},
			},
			{
				ResourceID: "security:firewall",
				Type:       "security",
				State:      engine.StateChanged,
				Action:     engine.ActionApply,
				Details:    map[string]string{"name": "firewall", "enabled": "true"},
			},
			{
				ResourceID: "system:rosetta",
				Type:       "system",
				State:      engine.StateMissing,
				Action:     engine.ActionApply,
				Details:    map[string]string{"name": "rosetta", "required": "true"},
			},
			{
				ResourceID: "ssh_key:" + home + "/.ssh/id_ed25519",
				Type:       "ssh_key",
				State:      engine.StateMissing,
				Action:     engine.ActionApply,
				Details:    map[string]string{"path": home + "/.ssh/id_ed25519", "type": "ed25519"},
			},
		},
		Summary: engine.PlanSummary{Total: 4, Missing: 3, Changed: 1, ToApply: 4},
	})

	for _, fragment := range []string{
		"dry-run Requires FileVault to be enabled manually",
		"dry-run Would enable firewall",
		"dry-run Would install Rosetta",
		"dry-run Would generate SSH keypair ~/.ssh/id_ed25519",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
}

func TestHumanRendererShowsCopyDryRunAndApplyMessages(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})
	home := mustUserHome(t)

	item := engine.PlanItem{
		ResourceID: "copy:" + home + "/.codex/skills/nuxt-practices",
		Type:       "copy",
		State:      engine.StateChanged,
		Action:     engine.ActionApply,
		Details: map[string]string{
			"source": home + "/dotfiles/codex/skills/nuxt-practices",
			"target": home + "/.codex/skills/nuxt-practices",
		},
	}
	renderer.renderDryRunPlan("/tmp/kitout.yaml", engine.Plan{
		Items:   []engine.PlanItem{item},
		Summary: engine.PlanSummary{Total: 1, Changed: 1, ToApply: 1},
	})
	renderer.BeforeApply(item)

	for _, fragment := range []string{
		"dry-run Would replace copy target ~/.codex/skills/nuxt-practices",
		"> Replacing copy target ~/.codex/skills/nuxt-practices...",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func mustUserHome(t *testing.T) string {
	t.Helper()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("user home: %v", err)
	}
	return home
}

func TestHumanRendererApplyProgressOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderApplyStart("/tmp/kitout.yaml")
	renderer.BeforeApply(engine.PlanItem{
		ResourceID: "brew:go",
		Type:       "brew",
		State:      engine.StateMissing,
		Action:     engine.ActionApply,
		Details:    map[string]string{"name": "go"},
	})
	renderer.renderApplyReport("", engine.ApplyReport{
		Items: []engine.ApplyItem{
			{ResourceID: "brew:go", Type: "brew", Action: "install", Changed: true, Message: "installed formula", Details: map[string]string{"name": "go"}},
		},
		Summary: engine.ApplySummary{Total: 1, Changed: 1},
	})

	for _, fragment := range []string{
		"Config: /tmp/kitout.yaml\n\nApplying changes:",
		"> Installing formula go...",
		"\nResults:\n",
		"✓ done  brew: go",
		"Summary: 1 changed",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
}

func TestHumanRendererBrewTapMessages(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderDryRunPlan("", engine.Plan{
		Items: []engine.PlanItem{
			{
				ResourceID: "brew_tap:vwall/kitout",
				Type:       "brew_tap",
				State:      engine.StateMissing,
				Action:     engine.ActionApply,
				Details:    map[string]string{"name": "vwall/kitout"},
			},
		},
		Summary: engine.PlanSummary{Total: 1, Missing: 1, ToApply: 1},
	})
	renderer.BeforeApply(engine.PlanItem{
		ResourceID: "brew_tap:vwall/kitout",
		Type:       "brew_tap",
		State:      engine.StateMissing,
		Action:     engine.ActionApply,
		Details:    map[string]string{"name": "vwall/kitout"},
	})

	for _, fragment := range []string{
		"Would add Homebrew tap vwall/kitout",
		"> Adding Homebrew tap vwall/kitout...",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHumanRendererStartupOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusStart("/tmp/status.yaml")
	renderer.renderApplyPlanStart("/tmp/apply.yaml", false)
	renderer.renderApplyPlanStart("/tmp/dry-run.yaml", true)
	renderer.renderDoctorStart("/tmp/doctor.yaml")

	for _, fragment := range []string{
		"Kitout is checking your Mac setup...\nConfig: /tmp/status.yaml",
		"Kitout is planning changes for your Mac setup...\nConfig: /tmp/apply.yaml",
		"[dry-run] Kitout is running in dry-run mode. No changes will be made.\nConfig: /tmp/dry-run.yaml",
		"Kitout is checking local prerequisites...\nConfig: /tmp/doctor.yaml",
	} {
		if !strings.Contains(stdout.String(), fragment) {
			t.Fatalf("stdout = %q, want fragment %q", stdout.String(), fragment)
		}
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
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
			{ResourceID: "symlink:/tmp/link", Type: "symlink", State: engine.StateChanged, Message: "symlink points elsewhere"},
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
		ansiBlue + "[dry-run]" + ansiReset,
		ansiBlue + "dry-run" + ansiReset,
		ansiYellow + "Would create directory /tmp/code" + ansiReset,
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

func TestHumanRendererConfigWarnings(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderConfigWarnings([]config.ConfigWarning{
		{Field: "example", Message: "example warning"},
	})

	want := "warning: example warning\n"
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
