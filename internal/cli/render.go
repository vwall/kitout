package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/vwall/kitout/internal/engine"
)

type humanRenderer struct {
	stdout io.Writer
	stderr io.Writer
	quiet  bool
	color  bool
}

func newHumanRenderer(stdout, stderr io.Writer, opts globalOptions) humanRenderer {
	return humanRenderer{
		stdout: stdout,
		stderr: stderr,
		quiet:  opts.quiet,
		color:  humanColorEnabled(stdout, opts),
	}
}

func (r humanRenderer) renderStatusPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	resourceWidth := planResourceIDWidth(plan.Items)
	for _, item := range plan.Items {
		fmt.Fprintf(r.stdout, "%s %-*s %s\n", r.statusMarker(item), resourceWidth, item.ResourceID, item.Message)
		if item.Error != "" {
			fmt.Fprintf(r.stdout, "%-*s error: %s\n", statusMarkerWidth, "", item.Error)
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
	if attention := statusAttentionCount(plan.Summary); attention > 0 {
		fmt.Fprintf(r.stdout, "%s\n", statusAttentionMessage(attention))
	}
}

func (r humanRenderer) renderDryRunPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	fmt.Fprintln(r.stdout, "Plan:")
	resourceWidth := planResourceIDWidth(plan.Items)
	for _, item := range plan.Items {
		switch item.Action {
		case engine.ActionApply:
			fmt.Fprintf(r.stdout, "  %s %-*s %s\n", r.marker("apply:", ansiYellow, actionMarkerWidth), resourceWidth, item.ResourceID, item.Message)
		case engine.ActionFail:
			fmt.Fprintf(r.stdout, "  %s %-*s %s\n", r.marker("fail:", ansiRed, actionMarkerWidth), resourceWidth, item.ResourceID, item.Message)
			if item.Error != "" {
				fmt.Fprintf(r.stdout, "        error: %s\n", item.Error)
			}
		case engine.ActionSkip:
			fmt.Fprintf(r.stdout, "  %s %-*s %s\n", r.marker("skip:", ansiCyan, actionMarkerWidth), resourceWidth, item.ResourceID, item.Message)
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

	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	resourceWidth := applyResourceIDWidth(report.Items)
	for _, item := range report.Items {
		fmt.Fprintf(r.stdout, "%s %-*s %s\n", r.applyMarker(item), resourceWidth, item.ResourceID, item.Message)
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

	fmt.Fprintf(r.stdout, "Config: %s\n\n", report.ConfigPath)
	fmt.Fprintln(r.stdout, "Doctor:")
	for _, item := range report.Items {
		fmt.Fprintf(r.stdout, "%s %-26s %s\n", r.doctorMarker(item), item.Name, item.Message)
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

const (
	minResourceIDWidth = 18
	statusMarkerWidth  = len("missing")
	actionMarkerWidth  = len("apply:")
	shortMarkerWidth   = len("done:")

	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
	ansiReset  = "\x1b[0m"
)

func humanColorEnabled(stdout io.Writer, opts globalOptions) bool {
	if opts.noColor {
		return false
	}

	file, ok := stdout.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (r humanRenderer) colorize(text, color string) string {
	if !r.color || color == "" {
		return text
	}
	return color + text + ansiReset
}

func (r humanRenderer) statusMarker(item engine.PlanItem) string {
	return r.marker(statusMarker(item), statusMarkerColor(item), statusMarkerWidth)
}

func (r humanRenderer) applyMarker(item engine.ApplyItem) string {
	return r.marker(applyMarker(item), applyMarkerColor(item), shortMarkerWidth)
}

func (r humanRenderer) doctorMarker(item doctorItem) string {
	return r.marker(doctorMarker(item), doctorMarkerColor(item), shortMarkerWidth)
}

func (r humanRenderer) marker(text, color string, width int) string {
	return r.colorize(fmt.Sprintf("%-*s", width, text), color)
}

func planResourceIDWidth(items []engine.PlanItem) int {
	width := minResourceIDWidth
	for _, item := range items {
		if len(item.ResourceID) > width {
			width = len(item.ResourceID)
		}
	}
	return width
}

func applyResourceIDWidth(items []engine.ApplyItem) int {
	width := minResourceIDWidth
	for _, item := range items {
		if len(item.ResourceID) > width {
			width = len(item.ResourceID)
		}
	}
	return width
}

func statusMarker(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return "ok"
	case engine.StateMissing:
		return "missing"
	case engine.StateChanged:
		return "changed"
	case engine.StateSkipped:
		return "skip"
	case engine.StateFailed, engine.StateUnknown:
		return "fail"
	default:
		return "fail"
	}
}

func statusAttentionCount(summary engine.PlanSummary) int {
	return summary.Missing + summary.Changed + summary.Failed + summary.Unknown
}

func statusAttentionMessage(count int) string {
	if count == 1 {
		return "1 resource needs attention"
	}
	return fmt.Sprintf("%d resources need attention", count)
}

func statusMarkerColor(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return ansiGreen
	case engine.StateMissing, engine.StateChanged:
		return ansiYellow
	case engine.StateSkipped:
		return ansiCyan
	case engine.StateFailed, engine.StateUnknown:
		return ansiRed
	default:
		return ansiRed
	}
}

func applyMarker(item engine.ApplyItem) string {
	if item.Error != "" {
		return "fail:"
	}
	switch item.Action {
	case "noop":
		return "ok:"
	case "skip":
		return "skip:"
	default:
		if item.Changed {
			return "done:"
		}
		return "ok:"
	}
}

func applyMarkerColor(item engine.ApplyItem) string {
	if item.Error != "" {
		return ansiRed
	}
	switch item.Action {
	case "noop":
		return ansiGreen
	case "skip":
		return ansiCyan
	default:
		if item.Changed {
			return ansiGreen
		}
		return ansiGreen
	}
}

func doctorMarker(item doctorItem) string {
	switch item.State {
	case doctorOK:
		return "ok:"
	case doctorWarn:
		return "warn:"
	case doctorFail:
		return "fail:"
	default:
		return "fail:"
	}
}

func doctorMarkerColor(item doctorItem) string {
	switch item.State {
	case doctorOK:
		return ansiGreen
	case doctorWarn:
		return ansiYellow
	case doctorFail:
		return ansiRed
	default:
		return ansiRed
	}
}
