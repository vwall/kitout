package scripts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type releaseWorkflow struct {
	Jobs map[string]releaseWorkflowJob `yaml:"jobs"`
}

type releaseWorkflowJob struct {
	Steps []releaseWorkflowStep `yaml:"steps"`
}

type releaseWorkflowStep struct {
	Name  string `yaml:"name"`
	ID    string `yaml:"id"`
	Shell string `yaml:"shell"`
	Run   string `yaml:"run"`
}

func TestReleaseWorkflowValidatesTagBeforeExportingMetadata(t *testing.T) {
	workflow := loadReleaseWorkflow(t)
	metadataStep := requireReleaseWorkflowStep(t, workflow, "Derive release metadata")

	if metadataStep.Shell != "bash" {
		t.Fatalf("metadata shell = %q, want bash", metadataStep.Shell)
	}

	run := metadataStep.Run
	validator := `version="$(scripts/validate-release-version.sh "${GITHUB_REF_NAME}")"`
	validatorIndex := strings.Index(run, validator)
	if validatorIndex < 0 {
		t.Fatalf("metadata step does not derive VERSION through the release tag validator:\n%s", run)
	}
	if strings.Contains(run, `version="${GITHUB_REF_NAME#v}"`) {
		t.Fatalf("metadata step still strips the tag prefix without validation:\n%s", run)
	}

	versionExportIndex := strings.Index(run, `echo "VERSION=${version}"`)
	if versionExportIndex < 0 {
		t.Fatalf("metadata step does not export VERSION from the validated version:\n%s", run)
	}
	if validatorIndex > versionExportIndex {
		t.Fatalf("metadata step exports VERSION before validating GITHUB_REF_NAME:\n%s", run)
	}

	envExportIndex := strings.Index(run, `>> "${GITHUB_ENV}"`)
	if envExportIndex < 0 {
		t.Fatalf("metadata step does not export metadata through GITHUB_ENV:\n%s", run)
	}
	if validatorIndex > envExportIndex {
		t.Fatalf("metadata step writes to GITHUB_ENV before validating GITHUB_REF_NAME:\n%s", run)
	}
}

func TestReleaseWorkflowRunsValidatedGateBeforePackagingAndPublishing(t *testing.T) {
	workflow := loadReleaseWorkflow(t)
	steps := workflow.Jobs["release"].Steps

	gateIndex := requireReleaseWorkflowStepIndex(t, steps, "Run release gate")
	packageIndex := requireReleaseWorkflowStepIndex(t, steps, "Package release artifacts")
	publishIndex := requireReleaseWorkflowStepIndex(t, steps, "Publish GitHub release")

	gate := steps[gateIndex]
	wantGate := `make release-check VERSION="${VERSION}" COMMIT="${COMMIT}" BUILD_DATE="${BUILD_DATE}"`
	if strings.TrimSpace(gate.Run) != wantGate {
		t.Fatalf("release gate command = %q, want %q", strings.TrimSpace(gate.Run), wantGate)
	}
	if gateIndex > packageIndex {
		t.Fatalf("release gate runs after packaging; gate index %d, package index %d", gateIndex, packageIndex)
	}
	if gateIndex > publishIndex {
		t.Fatalf("release gate runs after publishing; gate index %d, publish index %d", gateIndex, publishIndex)
	}
}

func loadReleaseWorkflow(t *testing.T) releaseWorkflow {
	t.Helper()

	repoRoot := filepath.Clean(filepath.Join(".."))
	workflowPath := filepath.Join(repoRoot, ".github", "workflows", "release.yml")
	body, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatal(err)
	}

	var workflow releaseWorkflow
	if err := yaml.Unmarshal(body, &workflow); err != nil {
		t.Fatalf("parse release workflow: %v", err)
	}
	if _, ok := workflow.Jobs["release"]; !ok {
		t.Fatalf("release workflow does not define a release job")
	}
	return workflow
}

func requireReleaseWorkflowStep(t *testing.T, workflow releaseWorkflow, name string) releaseWorkflowStep {
	t.Helper()

	steps := workflow.Jobs["release"].Steps
	index := requireReleaseWorkflowStepIndex(t, steps, name)
	return steps[index]
}

func requireReleaseWorkflowStepIndex(t *testing.T, steps []releaseWorkflowStep, name string) int {
	t.Helper()

	for i, step := range steps {
		if step.Name == name {
			return i
		}
	}
	t.Fatalf("release workflow step %q not found", name)
	return -1
}
