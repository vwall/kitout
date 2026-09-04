package resources

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/vwall/kitout/internal/config"
	"github.com/vwall/kitout/internal/engine"
	"github.com/vwall/kitout/internal/platform"
)

type inspectionCountRunner struct {
	calls     int
	commands  []commandCall
	inventory string
}

func (runner *inspectionCountRunner) Run(ctx context.Context, name string, args ...string) (platform.CommandResult, error) {
	runner.calls++
	runner.commands = append(runner.commands, commandCall{name: name, args: append([]string(nil), args...)})
	if err := ctx.Err(); err != nil {
		return platform.CommandResult{}, err
	}
	result := platform.CommandResult{}
	if name == "asdf" && strings.Join(args, " ") == "plugin list --urls" {
		result.Stdout = runner.inventory
	}
	if name == "asdf" && len(args) == 2 && args[0] == "list" {
		result.Stdout = "1.0\n"
	}
	return result, nil
}

func TestBuildScopesASDFInventoryToPlanningOperation(t *testing.T) {
	cfg := config.Config{Version: 1}
	runner := &inspectionCountRunner{}
	for i := 0; i < 30; i++ {
		name := fmt.Sprintf("tool%d", i)
		url := "https://example.com/" + name
		cfg.ASDF.Plugins = append(cfg.ASDF.Plugins, config.ASDFPlugin{Name: name, URL: url, Versions: []string{"1.0"}})
		runner.inventory += name + " " + url + "\n"
	}
	for pass := 1; pass <= 2; pass++ {
		plan := engine.NewPlanner().Build(context.Background(), Build(cfg, runner))
		if plan.Summary.Satisfied != 30 || runner.calls != pass*32 {
			t.Fatalf("pass %d: summary=%+v calls=%d; want 30 satisfied and %d calls", pass, plan.Summary, runner.calls, pass*32)
		}
	}
	runner.calls = 0
	plan := engine.NewPlanner().Build(context.Background(), BuildUncached(cfg, runner))
	if plan.Summary.Satisfied != 30 || runner.calls != 90 {
		t.Fatalf("uncached: summary=%+v calls=%d; want 30 satisfied and 90 fresh calls", plan.Summary, runner.calls)
	}
}

func TestBuildUncachedHomebrewUpgradeUsesOnlyTargetedInspections(t *testing.T) {
	for _, kind := range []string{"--formula", "--cask"} {
		t.Run(kind, func(t *testing.T) {
			cfg := config.Config{Version: 1}
			var names []string
			for i := 0; i < 30; i++ {
				names = append(names, fmt.Sprintf("package%d", i))
			}
			if kind == "--formula" {
				cfg.Brew.Packages = names
			} else {
				cfg.Brew.Casks = names
			}
			runner := &inspectionCountRunner{}
			for _, resource := range BuildUncached(cfg, runner) {
				upgrader := resource.(interface {
					Upgrade(context.Context) (engine.ApplyResult, error)
				})
				result, err := upgrader.Upgrade(context.Background())
				if err != nil || result.Changed || result.Action != "noop" {
					t.Fatalf("Upgrade = %+v, %v; want unchanged current package", result, err)
				}
			}
			if runner.calls != 60 {
				t.Fatalf("commands = %d, want 60 fresh targeted inspections", runner.calls)
			}
			for i, name := range names {
				expectCalls(t, runner.commands[i*2:i*2+2], []commandCall{
					{name: "brew", args: []string{"list", kind, name}},
					{name: "brew", args: []string{"outdated", kind, "--quiet", name}},
				})
			}
		})
	}
}
