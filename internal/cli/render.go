package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

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

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	}
	if path != "" || len(plan.Items) == 0 {
		fmt.Fprintln(r.stdout, "Results:")
	} else {
		fmt.Fprintln(r.stdout, "\nResults:")
	}
	resourceWidth := planStatusLabelWidth(plan.Items)
	for _, item := range plan.Items {
		label := displayResourceLabel(item.Type, item.ResourceID, item.Details)
		fmt.Fprintf(r.stdout, "%s %-*s %s\n", r.statusMarker(item), paddedWidth(resourceWidth, label), label, statusMessage(item))
		if item.Error != "" {
			renderIndentedDetail(r.stdout, strings.Repeat(" ", statusLeftWidth), "error", item.Error)
		}
	}
	fmt.Fprintf(r.stdout, "\nSummary: %s\n", statusSummary(plan.Summary))
	if attention := statusAttentionCount(plan.Summary); attention > 0 {
		fmt.Fprintf(r.stdout, "%s\n", statusAttentionMessage(attention))
	}
}

func (r humanRenderer) renderStatusStart(path string) {
	if r.quiet {
		return
	}

	fmt.Fprintln(r.stdout, "Kitout is checking your Mac setup...")
	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
}

func (r humanRenderer) renderDryRunPlan(path string, plan engine.Plan) {
	if r.quiet {
		return
	}

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	}
	fmt.Fprintf(r.stdout, "%s Previewing planned changes:\n", r.dryRunBadge())
	for _, item := range plan.Items {
		switch item.Action {
		case engine.ActionApply:
			fmt.Fprintf(r.stdout, "%s %s\n", r.dryRunMarker(), r.colorize(dryRunMessage(item), ansiYellow))
		case engine.ActionFail:
			fmt.Fprintf(r.stdout, "%s Cannot apply %s: %s\n", r.failSymbol(), displayResourceLabel(item.Type, item.ResourceID, item.Details), item.Message)
			if item.Error != "" {
				renderIndentedDetail(r.stdout, "    ", "error", item.Error)
			}
		case engine.ActionSkip:
			fmt.Fprintf(r.stdout, "%s Skipping %s: %s\n", r.skipSymbol(), displayResourceLabel(item.Type, item.ResourceID, item.Details), item.Message)
		}
	}
	if plan.Summary.ToApply == 0 {
		fmt.Fprintln(r.stdout, "No changes.")
	}
	fmt.Fprintf(r.stdout, "\n%s No changes made because --dry-run was used.\n", r.dryRunBadge())
	fmt.Fprintln(r.stdout, "No shell commands will run without explicit approval.")
}

func (r humanRenderer) renderApplyPlanStart(path string, dryRun bool) {
	if r.quiet {
		return
	}

	if dryRun {
		fmt.Fprintf(r.stdout, "%s Kitout is running in dry-run mode. No changes will be made.\n", r.dryRunBadge())
	} else {
		fmt.Fprintln(r.stdout, "Kitout is planning changes for your Mac setup...")
	}
	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
}

func (r humanRenderer) renderApplyStart(path string) {
	if r.quiet {
		return
	}

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	}
	fmt.Fprintln(r.stdout, "Applying changes:")
}

func (r humanRenderer) renderApplyReport(path string, report engine.ApplyReport) {
	if r.quiet {
		return
	}

	if path != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
	} else {
		fmt.Fprintln(r.stdout, "\nResults:")
	}
	resourceWidth := applyLabelWidth(report.Items)
	for _, item := range report.Items {
		label := displayResourceLabel(item.Type, item.ResourceID, item.Details)
		fmt.Fprintf(r.stdout, "%s %-*s %s\n", r.applyMarker(item), paddedWidth(resourceWidth, label), label, item.Message)
		if item.Error != "" {
			renderIndentedDetail(r.stdout, "    ", "error", item.Error)
		}
	}
	fmt.Fprintf(r.stdout, "\nSummary: %s\n", applySummary(report.Summary))
}

func renderIndentedDetail(writer io.Writer, indent, label, text string) {
	lines := strings.Split(text, "\n")
	fmt.Fprintf(writer, "%s%s: %s\n", indent, label, lines[0])

	continuationIndent := indent + strings.Repeat(" ", len(label)+2)
	for _, line := range lines[1:] {
		fmt.Fprintf(writer, "%s%s\n", continuationIndent, line)
	}
}

func (r humanRenderer) BeforeStatus(resource engine.Resource) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "%s Checking %s...\n", r.progressMarker(), displayResourceLabel(resource.Type(), resource.ID(), nil))
}

func (r humanRenderer) BeforeApply(item engine.PlanItem) {
	if r.quiet {
		return
	}

	fmt.Fprintf(r.stdout, "%s %s\n", r.progressMarker(), applyProgressMessage(item))
}

func (r humanRenderer) renderDoctorStart(path string) {
	if r.quiet {
		return
	}

	fmt.Fprintln(r.stdout, "Kitout is checking local prerequisites...")
	fmt.Fprintf(r.stdout, "Config: %s\n\n", path)
}

func (r humanRenderer) renderDoctorReport(report doctorReport) {
	if r.quiet {
		return
	}

	if report.ConfigPath != "" {
		fmt.Fprintf(r.stdout, "Config: %s\n\n", report.ConfigPath)
	}
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
	minResourceLabelWidth = 18
	statusMarkerWidth     = len("satisfied")
	statusLeftWidth       = 12
	shortMarkerWidth      = len("done:")

	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiBlue   = "\x1b[34m"
	ansiCyan   = "\x1b[36m"
	ansiReset  = "\x1b[0m"
)

func humanColorEnabled(stdout io.Writer, opts globalOptions) bool {
	if opts.noColor {
		return false
	}
	if opts.color {
		return true
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
	return r.colorize(fmt.Sprintf("%s %-*s", statusSymbol(item), statusMarkerWidth, statusMarker(item)), statusMarkerColor(item))
}

func (r humanRenderer) applyMarker(item engine.ApplyItem) string {
	return r.colorize(fmt.Sprintf("%s %-*s", applySymbol(item), shortMarkerWidth, applyMarker(item)), applyMarkerColor(item))
}

func (r humanRenderer) doctorMarker(item doctorItem) string {
	return r.marker(doctorMarker(item), doctorMarkerColor(item), shortMarkerWidth)
}

func (r humanRenderer) marker(text, color string, width int) string {
	return r.colorize(fmt.Sprintf("%-*s", width, text), color)
}

func (r humanRenderer) dryRunMarker() string {
	return r.colorize("dry-run", ansiBlue)
}

func (r humanRenderer) dryRunBadge() string {
	return r.colorize("[dry-run]", ansiBlue)
}

func (r humanRenderer) progressMarker() string {
	return r.colorize(">", ansiBlue)
}

func (r humanRenderer) failSymbol() string {
	return r.colorize("×", ansiRed)
}

func (r humanRenderer) skipSymbol() string {
	return r.colorize("-", ansiCyan)
}

func planStatusLabelWidth(items []engine.PlanItem) int {
	width := minResourceLabelWidth
	for _, item := range items {
		label := displayResourceLabel(item.Type, item.ResourceID, item.Details)
		if displayWidth(label) > width {
			width = displayWidth(label)
		}
	}
	return width
}

func applyLabelWidth(items []engine.ApplyItem) int {
	width := minResourceLabelWidth
	for _, item := range items {
		label := displayResourceLabel(item.Type, item.ResourceID, item.Details)
		if displayWidth(label) > width {
			width = displayWidth(label)
		}
	}
	return width
}

func statusMarker(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return "satisfied"
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

func statusSymbol(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return "✓"
	case engine.StateMissing, engine.StateChanged:
		return "!"
	case engine.StateSkipped:
		return "-"
	case engine.StateFailed, engine.StateUnknown:
		return "×"
	default:
		return "×"
	}
}

func statusMessage(item engine.PlanItem) string {
	switch item.State {
	case engine.StateSatisfied:
		return "satisfied"
	case engine.StateMissing:
		return "missing"
	case engine.StateChanged:
		if item.Message == "" {
			return "changed"
		}
		return item.Message
	case engine.StateSkipped:
		if item.Message == "" {
			return "skipped"
		}
		return item.Message
	case engine.StateFailed, engine.StateUnknown:
		if item.Message == "" {
			return "failed"
		}
		return "failed: " + item.Message
	default:
		if item.Message == "" {
			return "failed"
		}
		return "failed: " + item.Message
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
		return "fail"
	}
	switch item.Action {
	case "noop":
		return "ok"
	case "skip":
		return "skip"
	default:
		if item.Changed {
			return "done"
		}
		return "ok"
	}
}

func applySymbol(item engine.ApplyItem) string {
	if item.Error != "" {
		return "×"
	}
	switch item.Action {
	case "skip":
		return "-"
	default:
		return "✓"
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

func displayResourceLabel(typ, id string, details map[string]string) string {
	if typ == "" {
		typ, _ = splitResourceID(id)
	}
	target := displayResourceTarget(typ, id, details)
	if typ == "" || target == "" {
		return id
	}
	return typ + ": " + target
}

func displayResourceTarget(typ, id string, details map[string]string) string {
	switch typ {
	case "brew", "cask", "shell":
		if value := details["name"]; value != "" {
			return value
		}
	case "asdf_plugin":
		if value := details["name"]; value != "" {
			if missing := displayList(details["missing_versions"]); missing != "" {
				return value + " " + missing
			}
			return value
		}
	case "directory", "repo", "asdf_tool_versions":
		if value := details["path"]; value != "" {
			return compactPath(value)
		}
	case "login_shell":
		if value := details["resolved_path"]; value != "" {
			return compactPath(value)
		}
		if value := details["path"]; value != "" {
			return compactPath(value)
		}
	case "copy", "symlink":
		if value := details["target"]; value != "" {
			return compactPath(value)
		}
	case "macos_default":
		domain := details["domain"]
		key := details["key"]
		if domain != "" && key != "" {
			return domain + "/" + key
		}
	}

	_, target := splitResourceID(id)
	return compactPath(target)
}

func splitResourceID(id string) (string, string) {
	typ, target, ok := strings.Cut(id, ":")
	if !ok {
		return "", id
	}
	return typ, target
}

func compactPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+"/") {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}

func statusSummary(summary engine.PlanSummary) string {
	parts := make([]string, 0, 5)
	if summary.Satisfied > 0 {
		parts = append(parts, fmt.Sprintf("%d satisfied", summary.Satisfied))
	}
	if summary.Missing > 0 {
		parts = append(parts, fmt.Sprintf("%d missing", summary.Missing))
	}
	if summary.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", summary.Changed))
	}
	if failed := summary.Failed + summary.Unknown; failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", failed))
	}
	if summary.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", summary.Skipped))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d total", summary.Total)
	}
	return strings.Join(parts, ", ")
}

func applySummary(summary engine.ApplySummary) string {
	parts := make([]string, 0, 4)
	if summary.Changed > 0 {
		parts = append(parts, fmt.Sprintf("%d changed", summary.Changed))
	}
	if summary.Noop > 0 {
		parts = append(parts, fmt.Sprintf("%d unchanged", summary.Noop))
	}
	if summary.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", summary.Failed))
	}
	if summary.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("%d skipped", summary.Skipped))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d total", summary.Total)
	}
	return strings.Join(parts, ", ")
}

func dryRunMessage(item engine.PlanItem) string {
	typ := item.Type
	if typ == "" {
		typ, _ = splitResourceID(item.ResourceID)
	}
	target := displayResourceTarget(typ, item.ResourceID, item.Details)
	switch typ {
	case "brew":
		if item.State == engine.StateChanged {
			return "Would upgrade formula " + target
		}
		return "Would install formula " + target
	case "cask":
		return "Would install cask " + target
	case "directory":
		return "Would create directory " + target
	case "copy":
		if item.State == engine.StateChanged {
			return "Would replace copy target " + target
		}
		return "Would copy to " + target
	case "symlink":
		if item.State == engine.StateChanged {
			return "Would replace symlink " + target
		}
		return "Would link " + target
	case "repo":
		return "Would clone repository " + target
	case "macos_default":
		return "Would set macOS default " + target
	case "login_shell":
		if item.Details["listed_in_etc_shells"] == "false" && item.Details["add_to_etc_shells"] == "true" {
			return "Would allow login shell " + target + " and set it for the current user"
		}
		return "Would set login shell to " + target
	case "shell":
		if command := item.Details["command"]; command != "" {
			name := item.Details["name"]
			if name == "" {
				return "Would run shell command " + command
			}
			return "Would run shell command " + name + ": " + command
		}
		return "Would run shell command " + target
	case "asdf_plugin":
		if item.State == engine.StateChanged {
			return "Would update asdf plugin " + target
		}
		if missing := displayList(item.Details["missing_versions"]); missing != "" {
			return "Would install asdf " + versionNoun(missing) + " " + target
		}
		return "Would install asdf plugin " + target
	case "asdf_tool_versions":
		return "Would update .tool-versions " + target
	default:
		return "Would apply " + displayResourceLabel(item.Type, item.ResourceID, item.Details)
	}
}

func applyProgressMessage(item engine.PlanItem) string {
	typ := item.Type
	if typ == "" {
		typ, _ = splitResourceID(item.ResourceID)
	}
	target := displayResourceTarget(typ, item.ResourceID, item.Details)
	switch typ {
	case "brew":
		if item.State == engine.StateChanged {
			return "Upgrading formula " + target + "..."
		}
		return "Installing formula " + target + "..."
	case "cask":
		return "Installing cask " + target + "..."
	case "directory":
		return "Creating directory " + target + "..."
	case "copy":
		if item.State == engine.StateChanged {
			return "Replacing copy target " + target + "..."
		}
		return "Copying to " + target + "..."
	case "symlink":
		if item.State == engine.StateChanged {
			return "Replacing symlink " + target + "..."
		}
		return "Linking " + target + "..."
	case "repo":
		return "Cloning repository " + target + "..."
	case "macos_default":
		return "Setting macOS default " + target + "..."
	case "login_shell":
		return "Setting login shell to " + target + "..."
	case "shell":
		if command := item.Details["command"]; command != "" {
			name := item.Details["name"]
			if name == "" {
				return "Running shell command " + command + "..."
			}
			return "Running shell command " + name + "..."
		}
		return "Running shell command " + target + "..."
	case "asdf_plugin":
		if item.State == engine.StateChanged {
			return "Updating asdf plugin " + target + "..."
		}
		if missing := displayList(item.Details["missing_versions"]); missing != "" {
			return "Installing asdf " + versionNoun(missing) + " " + target + "..."
		}
		return "Installing asdf plugin " + target + "..."
	case "asdf_tool_versions":
		return "Updating .tool-versions " + target + "..."
	default:
		return "Applying " + displayResourceLabel(item.Type, item.ResourceID, item.Details) + "..."
	}
}

func paddedWidth(width int, label string) int {
	return width + len(label) - displayWidth(label)
}

func displayWidth(text string) int {
	return utf8.RuneCountInString(text)
}

func displayList(value string) string {
	parts := make([]string, 0)
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ", ")
}

func versionNoun(versions string) string {
	if strings.Contains(versions, ",") {
		return "versions"
	}
	return "version"
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
