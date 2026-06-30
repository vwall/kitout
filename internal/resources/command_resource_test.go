package resources

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

func TestBrewTapStatusSatisfiedWhenTapIsInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\nvwall/kitout\n")},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewTapType, engine.StateSatisfied, "tap is installed")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"tap"}}})
}

func TestBrewTapStatusMissingWhenTapIsNotInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\n")},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewTapType, engine.StateMissing, "tap is missing")
}

func TestBrewTapStatusFailsWhenTapListFails(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"tap"}, 1)},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Status(context.Background())
	if err == nil {
		t.Fatal("Status returned nil error, want tap list failure")
	}

	expectStatus(t, result, resource.ID(), brewTapType, engine.StateFailed, "could not inspect tap")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"tap"}}})
}

func TestBrewTapApplyAddsMissingTap(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\n")},
		{result: commandResult("brew", []string{"tap", "vwall/kitout"}, 0)},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewTapType, "tap", true, "added tap")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"tap"}},
		{name: "brew", args: []string{"tap", "vwall/kitout"}},
	})
}

func TestBrewTapApplyIsIdempotentWhenInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "vwall/kitout\n")},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewTapType, "noop", false, "tap already installed")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"tap"}}})
}

func TestBrewTapApplyReportsTapFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\n")},
		{err: commandError("brew", []string{"tap", "vwall/kitout"}, 1)},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want tap failure")
	}

	expectApply(t, result, resource.ID(), brewTapType, "tap", false, "could not add tap")
}

func TestBrewTapDryRunPlanDoesNotTap(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\n")},
	}}
	resource := NewBrewTap("vwall/kitout", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"tap"}}})
}

func TestBrewTapDryRunBatchesInstalledCheckForBuiltResources(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\nvwall/kitout\n")},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Taps: []string{"vwall/kitout", "homebrew/cask-fonts"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("vwall/kitout state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("homebrew/cask-fonts state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"tap"}}})
}

func TestBrewTapUncachedBuildUsesDirectInstalledChecks(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\nvwall/kitout\n")},
		{result: resultWithStdout("brew", []string{"tap"}, "homebrew/core\n")},
	}}
	resources := BuildUncached(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Taps: []string{"vwall/kitout", "homebrew/cask-fonts"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("vwall/kitout state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("homebrew/cask-fonts state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"tap"}},
		{name: "brew", args: []string{"tap"}},
	})
}

func TestBrewPackageStatusSatisfiedWhenFormulaIsInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 0)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewType, engine.StateSatisfied, "formula is installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageStatusSatisfiedWhenOutdatedCheckExitsOneWithNoOutput(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{
			result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 1),
			err:    commandError("brew", []string{"outdated", "--formula", "--quiet"}, 1),
		},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewType, engine.StateSatisfied, "formula is installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageStatusAdvisesWhenFormulaIsOutdated(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewType, engine.StateSatisfied, "formula is installed")
	expectAdvisory(t, result.Advisories, "homebrew_formula_outdated", "formula update available for git", "brew upgrade git")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageStatusMissingWhenFormulaIsNotInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("brew", []string{"list", "--formula", "git"}, 1)}}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewType, engine.StateMissing, "formula is missing")
}

func TestBrewPackageStatusWarnsWhenOutdatedCheckFails(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{err: commandError("brew", []string{"outdated", "--formula", "--quiet"}, 2)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), brewType, engine.StateSatisfied, "formula is installed")
	expectAdvisoryWithSeverity(t, result.Advisories, "homebrew_formula_update_check_failed", engine.AdvisoryWarning, "could not inspect formula updates for git", "brew doctor")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageApplyInstallsMissingFormula(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--formula", "git"}, 1)},
		{result: commandResult("brew", []string{"install", "git"}, 0)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "install", true, "installed formula")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"install", "git"}},
	})
}

func TestBrewPackageApplyIsIdempotentWhenInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{
			result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 1),
			err:    commandError("brew", []string{"outdated", "--formula", "--quiet"}, 1),
		},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "noop", false, "formula already installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageApplyDoesNotUpgradeOutdatedFormula(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "noop", false, "formula already installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageUpgradeUpdatesOutdatedFormula(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
		{result: commandResult("brew", []string{"upgrade", "git"}, 0)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "upgrade", true, "upgraded formula")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"upgrade", "git"}},
	})
}

func TestBrewPackageUpgradeNoopsWhenFormulaIsCurrent(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 0)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "noop", false, "formula already current")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageUpgradeSkipsMissingFormula(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--formula", "git"}, 1)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "skip", false, "formula is missing; run `kitout apply` first")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"list", "--formula", "git"}}})
}

func TestBrewPackageUpgradeReportsOutdatedCheckFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{err: commandError("brew", []string{"outdated", "--formula", "--quiet"}, 2)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Upgrade(context.Background())
	if err == nil {
		t.Fatal("Upgrade returned nil error, want update check failure")
	}

	expectApply(t, result, resource.ID(), brewType, "fail", false, "could not inspect formula updates for git")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageUpgradeReportsUpgradeFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
		{err: commandError("brew", []string{"upgrade", "git"}, 2)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Upgrade(context.Background())
	if err == nil {
		t.Fatal("Upgrade returned nil error, want upgrade failure")
	}

	expectApply(t, result, resource.ID(), brewType, "upgrade", false, "could not upgrade formula")
}

func TestBrewPackageApplyIgnoresOutdatedFormulaUpgradeFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), brewType, "noop", false, "formula already installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageApplyReportsInstallFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--formula", "git"}, 1)},
		{err: commandError("brew", []string{"install", "git"}, 2)},
	}}
	resource := NewBrewPackage("git", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want install failure")
	}

	expectApply(t, result, resource.ID(), brewType, "install", false, "could not install formula")
}

func TestBrewPackageDryRunPlanDoesNotInstall(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("brew", []string{"list", "--formula", "git"}, 1)}}}
	resource := NewBrewPackage("git", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"list", "--formula", "git"}}})
}

func TestBrewPackageDryRunPlanDoesNotUpgrade(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "git\n")},
	}}
	resource := NewBrewPackage("git", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("State = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[0].Action != engine.ActionNoop {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionNoop)
	}
	expectAdvisory(t, plan.Items[0].Advisories, "homebrew_formula_outdated", "formula update available for git", "brew upgrade git")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageDryRunBatchesOutdatedCheckForBuiltResources(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\ngo\n")},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "go\n")},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"git", "go"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("git state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateSatisfied {
		t.Fatalf("go state = %q, want %q", plan.Items[1].State, engine.StateSatisfied)
	}
	expectAdvisory(t, plan.Items[1].Advisories, "homebrew_formula_outdated", "formula update available for go", "brew upgrade go")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageDryRunBatchesInstalledCheckForBuiltResources(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"list", "--formula", "--quiet"}, "git\n")},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 0)},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"git", "go"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("git state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("go state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestBrewPackageDryRunUsesDirectChecksForFullyQualifiedBuiltFormula(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "owner/repo/example"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet", "owner/repo/example"}, 0)},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"owner/repo/example"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("owner/repo/example state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "owner/repo/example"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet", "owner/repo/example"}},
	})
}

func TestBrewPackageDryRunChecksFullyQualifiedBuiltFormulaOutdatedDirectly(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "owner/repo/example"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet", "owner/repo/example"}, "example\n")},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"owner/repo/example"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("owner/repo/example state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	expectAdvisory(t, plan.Items[0].Advisories, "homebrew_formula_outdated", "formula update available for owner/repo/example", "brew upgrade owner/repo/example")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "owner/repo/example"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet", "owner/repo/example"}},
	})
}

func TestBrewPackageUncachedBuildUsesDirectInstalledChecks(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 0)},
		{err: commandError("brew", []string{"list", "--formula", "go"}, 1)},
	}}
	resources := BuildUncached(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"git", "go"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("git state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("go state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"list", "--formula", "go"}},
	})
}

func TestBrewPackageUncachedBuildDoesNotShareOutdatedChecks(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--formula", "git"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--formula", "--quiet"}, 0)},
		{result: commandResult("brew", []string{"list", "--formula", "go"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--formula", "--quiet"}, "go\n")},
	}}
	resources := BuildUncached(config.Config{
		Version: config.CurrentVersion,
		Brew: config.Brew{
			Packages: []string{"git", "go"},
		},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("git state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateSatisfied {
		t.Fatalf("go state = %q, want %q", plan.Items[1].State, engine.StateSatisfied)
	}
	expectAdvisory(t, plan.Items[1].Advisories, "homebrew_formula_outdated", "formula update available for go", "brew upgrade go")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--formula", "git"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
		{name: "brew", args: []string{"list", "--formula", "go"}},
		{name: "brew", args: []string{"outdated", "--formula", "--quiet"}},
	})
}

func TestCaskStatusSatisfiedWhenInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--cask", "--quiet"}, 0)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), caskType, engine.StateSatisfied, "cask is installed")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
	})
}

func TestCaskStatusAdvisesWhenOutdated(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--cask", "--quiet"}, "ghostty\n")},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), caskType, engine.StateSatisfied, "cask is installed")
	expectAdvisory(t, result.Advisories, "homebrew_cask_outdated", "cask update available for ghostty", "brew upgrade --cask ghostty")
}

func TestCaskStatusWarnsWhenOutdatedCheckFails(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{err: commandError("brew", []string{"outdated", "--cask", "--quiet"}, 2)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), caskType, engine.StateSatisfied, "cask is installed")
	expectAdvisoryWithSeverity(t, result.Advisories, "homebrew_cask_update_check_failed", engine.AdvisoryWarning, "could not inspect cask updates for ghostty", "brew doctor")
}

func TestCaskStatusMissingWhenNotInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("brew", []string{"list", "--cask", "ghostty"}, 1)}}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Status(context.Background())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}

	expectStatus(t, result, resource.ID(), caskType, engine.StateMissing, "cask is missing")
}

func TestCaskApplyInstallsMissingCask(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--cask", "ghostty"}, 1)},
		{result: commandResult("brew", []string{"install", "--cask", "ghostty"}, 0)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), caskType, "install", true, "installed cask")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"install", "--cask", "ghostty"}},
	})
}

func TestCaskApplyIsIdempotentWhenInstalled(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--cask", "--quiet"}, 0)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Apply(context.Background())
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), caskType, "noop", false, "cask already installed")
}

func TestCaskApplyReportsInstallFailure(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--cask", "ghostty"}, 1)},
		{err: commandError("brew", []string{"install", "--cask", "ghostty"}, 2)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Apply(context.Background())
	if err == nil {
		t.Fatal("Apply returned nil error, want install failure")
	}

	expectApply(t, result, resource.ID(), caskType, "install", false, "could not install cask")
}

func TestCaskUpgradeUpdatesOutdatedCask(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: resultWithStdout("brew", []string{"outdated", "--cask", "--quiet"}, "ghostty\n")},
		{result: commandResult("brew", []string{"upgrade", "--cask", "ghostty"}, 0)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), caskType, "upgrade", true, "upgraded cask")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
		{name: "brew", args: []string{"upgrade", "--cask", "ghostty"}},
	})
}

func TestCaskUpgradeNoopsWhenCaskIsCurrent(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--cask", "--quiet"}, 0)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), caskType, "noop", false, "cask already current")
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
	})
}

func TestCaskUpgradeSkipsMissingCask(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{err: commandError("brew", []string{"list", "--cask", "ghostty"}, 1)},
	}}
	resource := NewCask("ghostty", runner)

	result, err := resource.Upgrade(context.Background())
	if err != nil {
		t.Fatalf("Upgrade returned error: %v", err)
	}

	expectApply(t, result, resource.ID(), caskType, "skip", false, "cask is missing; run `kitout apply` first")
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"list", "--cask", "ghostty"}}})
}

func TestCaskDryRunPlanDoesNotInstall(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{{err: commandError("brew", []string{"list", "--cask", "ghostty"}, 1)}}}
	resource := NewCask("ghostty", runner)

	plan := engine.NewPlanner().Build(context.Background(), []engine.Resource{resource})

	if plan.Items[0].Action != engine.ActionApply {
		t.Fatalf("Action = %q, want %q", plan.Items[0].Action, engine.ActionApply)
	}
	expectCalls(t, runner.calls, []commandCall{{name: "brew", args: []string{"list", "--cask", "ghostty"}}})
}

func TestCaskDryRunBatchesInstalledCheckForBuiltResources(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: resultWithStdout("brew", []string{"list", "--cask", "--quiet"}, "ghostty\n")},
		{result: commandResult("brew", []string{"outdated", "--cask", "--quiet"}, 0)},
	}}
	resources := Build(config.Config{
		Version: config.CurrentVersion,
		Brew:    config.Brew{Casks: []string{"ghostty", "rectangle"}},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("ghostty state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("rectangle state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "--quiet"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
	})
}

func TestCaskUncachedBuildUsesDirectInstalledChecks(t *testing.T) {
	runner := &fakeRunner{responses: []fakeResponse{
		{result: commandResult("brew", []string{"list", "--cask", "ghostty"}, 0)},
		{result: commandResult("brew", []string{"outdated", "--cask", "--quiet"}, 0)},
		{err: commandError("brew", []string{"list", "--cask", "rectangle"}, 1)},
	}}
	resources := BuildUncached(config.Config{
		Version: config.CurrentVersion,
		Brew:    config.Brew{Casks: []string{"ghostty", "rectangle"}},
	}, runner)

	plan := engine.NewPlanner().Build(context.Background(), resources)

	if plan.Items[0].State != engine.StateSatisfied {
		t.Fatalf("ghostty state = %q, want %q", plan.Items[0].State, engine.StateSatisfied)
	}
	if plan.Items[1].State != engine.StateMissing {
		t.Fatalf("rectangle state = %q, want %q", plan.Items[1].State, engine.StateMissing)
	}
	expectCalls(t, runner.calls, []commandCall{
		{name: "brew", args: []string{"list", "--cask", "ghostty"}},
		{name: "brew", args: []string{"outdated", "--cask", "--quiet"}},
		{name: "brew", args: []string{"list", "--cask", "rectangle"}},
	})
}

func expectStatus(t *testing.T, result engine.StatusResult, id, typ string, state engine.ResourceState, message string) {
	t.Helper()

	if result.ResourceID != id {
		t.Fatalf("ResourceID = %q, want %q", result.ResourceID, id)
	}
	if result.Type != typ {
		t.Fatalf("Type = %q, want %q", result.Type, typ)
	}
	if result.State != state {
		t.Fatalf("State = %q, want %q", result.State, state)
	}
	if result.Message != message {
		t.Fatalf("Message = %q, want %q", result.Message, message)
	}
}

func expectAdvisory(t *testing.T, advisories []engine.Advisory, code, message, fixFragment string) {
	t.Helper()

	expectAdvisoryWithSeverity(t, advisories, code, engine.AdvisoryNotice, message, fixFragment)
}

func expectAdvisoryWithSeverity(t *testing.T, advisories []engine.Advisory, code string, severity engine.AdvisorySeverity, message, fixFragment string) {
	t.Helper()

	if len(advisories) != 1 {
		t.Fatalf("advisories = %+v, want one advisory", advisories)
	}
	advisory := advisories[0]
	if advisory.Code != code {
		t.Fatalf("advisory code = %q, want %q", advisory.Code, code)
	}
	if advisory.Severity != severity {
		t.Fatalf("advisory severity = %q, want %q", advisory.Severity, severity)
	}
	if advisory.Message != message {
		t.Fatalf("advisory message = %q, want %q", advisory.Message, message)
	}
	if !strings.Contains(advisory.Fix, fixFragment) {
		t.Fatalf("advisory fix = %q, want fragment %q", advisory.Fix, fixFragment)
	}
}

func expectApply(t *testing.T, result engine.ApplyResult, id, typ, action string, changed bool, message string) {
	t.Helper()

	if result.ResourceID != id {
		t.Fatalf("ResourceID = %q, want %q", result.ResourceID, id)
	}
	if result.Type != typ {
		t.Fatalf("Type = %q, want %q", result.Type, typ)
	}
	if result.Action != action {
		t.Fatalf("Action = %q, want %q", result.Action, action)
	}
	if result.Changed != changed {
		t.Fatalf("Changed = %v, want %v", result.Changed, changed)
	}
	if result.Message != message {
		t.Fatalf("Message = %q, want %q", result.Message, message)
	}
}

type commandCall struct {
	name string
	args []string
}

type fakeResponse struct {
	result platform.CommandResult
	err    error
}

type fakeRunner struct {
	calls     []commandCall
	responses []fakeResponse
}

func (runner *fakeRunner) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	runner.calls = append(runner.calls, commandCall{
		name: name,
		args: append([]string(nil), args...),
	})

	if len(runner.responses) == 0 {
		return commandResult(name, args, 0), nil
	}

	response := runner.responses[0]
	runner.responses = runner.responses[1:]
	if response.result.Name == "" {
		response.result = commandResult(name, args, 0)
	}
	return response.result, response.err
}

func expectCalls(t *testing.T, got, want []commandCall) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("calls = %#v, want %#v", got, want)
	}
}

func commandResult(name string, args []string, exitCode int) platform.CommandResult {
	return platform.CommandResult{
		Name:     name,
		Args:     append([]string(nil), args...),
		Stdout:   "",
		Stderr:   "",
		ExitCode: exitCode,
	}
}

func commandError(name string, args []string, exitCode int) platform.CommandError {
	return platform.CommandError{
		Result: commandResult(name, args, exitCode),
		Err:    errors.New("command failed"),
	}
}

func commandErrorWithStderr(name string, args []string, exitCode int, stderr string) platform.CommandError {
	err := commandError(name, args, exitCode)
	err.Result.Stderr = stderr
	return err
}

func resultWithStdout(name string, args []string, stdout string) platform.CommandResult {
	result := commandResult(name, args, 0)
	result.Stdout = stdout
	return result
}

func containsError(err error, fragment string) bool {
	return err != nil && strings.Contains(err.Error(), fragment)
}
