package cli

import (
	"bytes"
	"errors"
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
		},
		Summary: engine.PlanSummary{
			Total:     2,
			Satisfied: 1,
			Missing:   1,
			ToApply:   1,
		},
	})

	for _, fragment := range []string{
		"Config: /tmp/kitout.yaml",
		"directory:/tmp/code directory exists",
		"need brew:git",
		"2 total, 1 satisfied, 1 missing",
		"1 changes needed",
	} {
		if !bytes.Contains(stdout.Bytes(), []byte(fragment)) {
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
