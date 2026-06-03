package cli

import (
	"fmt"
	"io"

	"github.com/vwall/kitout/internal/engine"
)

type humanRenderer struct {
	stdout io.Writer
	stderr io.Writer
	quiet  bool
}

func newHumanRenderer(stdout, stderr io.Writer, opts globalOptions) humanRenderer {
	return humanRenderer{
		stdout: stdout,
		stderr: stderr,
		quiet:  opts.quiet,
	}
}

func (r humanRenderer) renderStatusPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n", path)
	for _, item := range plan.Items {
		fmt.Fprintf(r.stdout, "%s %-18s %s\n", statusMarker(item), item.ResourceID, item.Message)
		if item.Error != "" {
			fmt.Fprintf(r.stdout, "    error: %s\n", item.Error)
		}
	}
	fmt.Fprintf(r.stdout, "\n%d total, %d satisfied, %d missing, %d changed, %d failed, %d skipped\n",
		plan.Summary.Total,
		plan.Summary.Satisfied,
		plan.Summary.Missing,
		plan.Summary.Changed,
		plan.Summary.Failed+plan.Summary.Unknown,
		plan.Summary.Skipped,
	)
	if plan.Summary.ToApply > 0 {
		fmt.Fprintf(r.stdout, "%d changes needed\n", plan.Summary.ToApply)
	}
}

func (r humanRenderer) renderDryRunPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n", path)
	fmt.Fprintln(r.stdout, "Plan:")
	for _, item := range plan.Items {
		switch item.Action {
		case engine.ActionApply:
			fmt.Fprintf(r.stdout, "  apply %-18s %s\n", item.ResourceID, item.Message)
		case engine.ActionFail:
			fmt.Fprintf(r.stdout, "  fail  %-18s %s\n", item.ResourceID, item.Message)
			if item.Error != "" {
				fmt.Fprintf(r.stdout, "        error: %s\n", item.Error)
			}
		case engine.ActionSkip:
			fmt.Fprintf(r.stdout, "  skip  %-18s %s\n", item.ResourceID, item.Message)
		}
	}
	if plan.Summary.ToApply == 0 {
		fmt.Fprintln(r.stdout, "  no changes")
	}
	fmt.Fprintln(r.stdout, "\nNo changes made because --dry-run was used.")
}

func (r humanRenderer) renderApplyReport(path string, report engine.ApplyReport) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n", path)
	for _, item := range report.Items {
		fmt.Fprintf(r.stdout, "%s %-18s %s\n", applyMarker(item), item.ResourceID, item.Message)
		if item.Error != "" {
			fmt.Fprintf(r.stdout, "    error: %s\n", item.Error)
		}
	}
	fmt.Fprintf(r.stdout, "\n%d total, %d changed, %d unchanged, %d failed, %d skipped\n",
		report.Summary.Total,
		report.Summary.Changed,
		report.Summary.Noop,
		report.Summary.Failed,
		report.Summary.Skipped,
	)
}

func (r humanRenderer) renderDoctorReport(report doctorReport) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n", report.ConfigPath)
	fmt.Fprintln(r.stdout, "Doctor:")
	for _, item := range report.Items {
		fmt.Fprintf(r.stdout, "%s %-26s %s\n", doctorMarker(item), item.Name, item.Message)
		if item.Fix != "" {
			fmt.Fprintf(r.stdout, "    fix: %s\n", item.Fix)
		}
	}
	fmt.Fprintf(r.stdout, "\n%d total, %d ok, %d warnings, %d failed\n",
		report.Summary.Total,
		report.Summary.OK,
		report.Summary.Warn,
		report.Summary.Fail,
	)
}

func (r humanRenderer) renderInvalidConfigDetails(err error) {
	fmt.Fprintln(r.stderr, err.Error())
}

func (r humanRenderer) renderInvalidConfig(err error) {
	fmt.Fprintf(r.stderr, "Invalid config: %v\n", err)
}

func (r humanRenderer) renderConfigLoadFailure(err error) {
	fmt.Fprintf(r.stderr, "Failed to load config: %v\n", err)
}

func statusMarker(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return "ok  "
	case engine.StateMissing, engine.StateChanged:
		return "need"
	case engine.StateSkipped:
		return "skip"
	case engine.StateFailed, engine.StateUnknown:
		return "fail"
	default:
		return "fail"
	}
}

func applyMarker(item engine.ApplyItem) string {
	if item.Error != "" {
		return "fail"
	}
	switch item.Action {
	case "noop":
		return "ok  "
	case "skip":
		return "skip"
	default:
		if item.Changed {
			return "done"
		}
		return "ok  "
	}
}

func doctorMarker(item doctorItem) string {
	switch item.State {
	case doctorOK:
		return "ok  "
	case doctorWarn:
		return "warn"
	case doctorFail:
		return "fail"
	default:
		return "fail"
	}
}
