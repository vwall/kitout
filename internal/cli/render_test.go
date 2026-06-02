package cli

import (
	"bytes"
	"errors"
	"testing"

	"github.com/vwall/kitout/internal/config"
)

func TestHumanRendererStatusOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{})

	renderer.renderStatusConfigValid("/tmp/kitout.yaml")
	renderer.renderStatusChecksNotImplemented()

	want := "Config valid: /tmp/kitout.yaml\nStatus checks are not implemented yet.\n"
	if stdout.String() != want {
		t.Fatalf("stdout = %q, want %q", stdout.String(), want)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestHumanRendererQuietSuppressesStatusOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	renderer := newHumanRenderer(&stdout, &stderr, globalOptions{quiet: true})

	renderer.renderStatusConfigValid("/tmp/kitout.yaml")
	renderer.renderStatusChecksNotImplemented()

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
