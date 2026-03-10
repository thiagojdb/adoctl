package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"adoctl/pkg/models"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

// ansiEscapePattern matches ANSI escape sequences (color codes)
var ansiEscapePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences from a string
func stripANSI(s string) string {
	return ansiEscapePattern.ReplaceAllString(s, "")
}

// displayWidth returns the visual width of a string (handles Unicode and ANSI)
func displayWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

// formatAndOutputIterations formats iteration list based on --format flag
func formatAndOutputIterations(cmd *cobra.Command, iterations []models.Iteration) error {
	format := GetFormat(cmd)
	writer := NewOutputWriter(format)

	if writer.IsStructured() {
		return writer.Write(iterations)
	}

	// Table output
	output := renderIterationsTable(iterations, format == string(FormatModern))
	fmt.Println(output)

	if ShouldCopyOutput(cmd) {
		return CopyToClipboard(output)
	}

	return nil
}

// getTerminalWidth returns the terminal width, defaulting to 120 if unavailable
func getTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 80 {
		return 120
	}
	return width
}

// truncate truncates a string to max display width, adding ellipsis if truncated
func truncate(s string, maxWidth int) string {
	if maxWidth <= 3 {
		return s[:maxWidth]
	}
	w := displayWidth(s)
	if w <= maxWidth {
		return s
	}
	// Need to truncate - find the position to cut
	stripped := stripANSI(s)
	runes := []rune(stripped)
	result := ""
	currentWidth := 0
	for _, r := range runes {
		rw := runewidth.RuneWidth(r)
		if currentWidth+rw+3 > maxWidth { // +3 for "..."
			break
		}
		result += string(r)
		currentWidth += rw
	}
	return result + "..."
}

// padRight pads a string to the right with spaces to reach display width
func padRight(s string, width int) string {
	w := displayWidth(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

// shortenWorkItemType abbreviates long work item types
func shortenWorkItemType(wiType string) string {
	switch wiType {
	case "Product Backlog Item":
		return "PBI"
	default:
		return wiType
	}
}

// renderIterationsTable renders iterations as a table
func renderIterationsTable(iterations []models.Iteration, modern bool) string {
	if len(iterations) == 0 {
		return "No sprints found."
	}

	if modern {
		return renderIterationsModern(iterations)
	}
	return renderIterationsClassic(iterations)
}

func renderIterationsModern(iterations []models.Iteration) string {
	var sb strings.Builder
	headerStyle := color.New(color.Bold, color.Underline)
	sb.WriteString(headerStyle.Sprint("Sprints/Iterations"))
	sb.WriteString("\n\n")

	for _, iter := range iterations {
		prefix := "  "
		if iter.IsCurrent {
			prefix = "▶ "
		}

		nameStyle := color.New()
		if iter.IsCurrent {
			nameStyle = color.New(color.Bold, color.FgGreen)
		}

		dateRange := formatDateRange(iter.StartDate, iter.FinishDate)
		currentMarker := ""
		if iter.IsCurrent {
			currentMarker = " [CURRENT]"
		}

		sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, nameStyle.Sprint(iter.Name), currentMarker))
		if dateRange != "" {
			sb.WriteString(fmt.Sprintf("   Path: %s\n", iter.Path))
			sb.WriteString(fmt.Sprintf("   Dates: %s\n", dateRange))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func renderIterationsClassic(iterations []models.Iteration) string {
	var sb strings.Builder
	width := getTerminalWidth()

	// Define column widths (proportional to terminal width)
	// Account for spaces between columns (4 spaces)
	spacing := 4
	nameWidth := int(float64(width) * 0.20)
	pathWidth := int(float64(width) * 0.42)
	startWidth := 12
	endWidth := 12
	statusWidth := 8

	totalWidth := nameWidth + pathWidth + startWidth + endWidth + statusWidth + (spacing * 4)
	if totalWidth > width {
		// Adjust path width to fit
		pathWidth -= totalWidth - width
	}

	// Ensure minimum widths
	if nameWidth < 15 {
		nameWidth = 15
	}
	if pathWidth < 20 {
		pathWidth = 20
	}

	// Build header
	headerStyle := color.New(color.Bold)
	header1 := padRight("NAME", nameWidth)
	header2 := padRight("PATH", pathWidth)
	header3 := padRight("START DATE", startWidth)
	header4 := padRight("END DATE", endWidth)
	header5 := "STATUS"

	sb.WriteString(headerStyle.Sprint(header1))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(header2))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(header3))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(header4))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(header5))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Build rows
	for _, iter := range iterations {
		status := ""
		statusColor := color.New()
		if iter.IsCurrent {
			status = "CURRENT"
			statusColor = color.New(color.FgGreen, color.Bold)
		}

		startDate := ""
		if iter.StartDate != nil {
			startDate = iter.StartDate.Format("2006-01-02")
		}

		finishDate := ""
		if iter.FinishDate != nil {
			finishDate = iter.FinishDate.Format("2006-01-02")
		}

		// Color for current sprint
		nameColor := color.New()
		pathColor := color.New(color.FgHiBlack)
		dateColor := color.New()
		if iter.IsCurrent {
			nameColor = color.New(color.Bold, color.FgGreen)
			dateColor = color.New(color.Bold)
		}

		sb.WriteString(padRight(truncate(nameColor.Sprint(iter.Name), nameWidth), nameWidth))
		sb.WriteString(strings.Repeat(" ", spacing))
		sb.WriteString(padRight(truncate(pathColor.Sprint(iter.Path), pathWidth), pathWidth))
		sb.WriteString(strings.Repeat(" ", spacing))
		sb.WriteString(padRight(truncate(dateColor.Sprint(startDate), startWidth), startWidth))
		sb.WriteString(strings.Repeat(" ", spacing))
		sb.WriteString(padRight(truncate(dateColor.Sprint(finishDate), endWidth), endWidth))
		sb.WriteString(strings.Repeat(" ", spacing))
		sb.WriteString(truncate(statusColor.Sprint(status), statusWidth))
		sb.WriteString("\n")
	}

	return sb.String()
}

// formatDateRange formats a date range for display
func formatDateRange(start, finish *time.Time) string {
	if start == nil || finish == nil {
		return ""
	}
	return fmt.Sprintf("%s to %s", start.Format("Jan 2"), finish.Format("Jan 2, 2006"))
}

// formatAndOutputWorkItems formats hierarchical work items
func formatAndOutputWorkItems(cmd *cobra.Command, items []models.WorkItemHierarchy, filter string) error {
	format := GetFormat(cmd)
	writer := NewOutputWriter(format)

	if writer.IsStructured() {
		return writer.Write(items)
	}

	// Table/hierarchical output
	output := renderHierarchicalTable(items, filter, format == string(FormatModern))
	fmt.Println(output)

	if ShouldCopyOutput(cmd) {
		return CopyToClipboard(output)
	}

	return nil
}

// renderHierarchicalTable renders work items with indentation for children
func renderHierarchicalTable(items []models.WorkItemHierarchy, filter string, modern bool) string {
	if len(items) == 0 {
		if filter != "" {
			return fmt.Sprintf("No work items found assigned to: %s", filter)
		}
		return "No work items found in this sprint."
	}

	if modern {
		return renderWorkItemsModern(items, filter)
	}
	return renderWorkItemsClassic(items, filter)
}

func renderWorkItemsModern(items []models.WorkItemHierarchy, filter string) string {
	var sb strings.Builder
	headerStyle := color.New(color.Bold, color.Underline)
	if filter != "" {
		sb.WriteString(headerStyle.Sprintf("Work Items (filtered by: %s)", filter))
	} else {
		sb.WriteString(headerStyle.Sprint("Work Items"))
	}
	sb.WriteString("\n\n")

	for _, item := range items {
		renderWorkItemModern(&sb, item, 0, filter)
	}
	return sb.String()
}

func renderWorkItemsClassic(items []models.WorkItemHierarchy, filter string) string {
	var sb strings.Builder
	width := getTerminalWidth()

	// Define column widths with spacing - reduced type width
	spacing := 2
	idWidth := 12
	typeWidth := 6 // Reduced from 16 - "PBI" is only 3 chars
	stateWidth := 12
	assignedWidth := 18
	titleWidth := width - idWidth - typeWidth - stateWidth - assignedWidth - (spacing * 4) - 2

	if titleWidth < 25 {
		titleWidth = 25
	}

	// Build header
	headerStyle := color.New(color.Bold)
	sb.WriteString(headerStyle.Sprint(padRight("ID", idWidth)))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(padRight("TYPE", typeWidth)))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(padRight("TITLE", titleWidth)))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint(padRight("STATE", stateWidth)))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(headerStyle.Sprint("ASSIGNED TO"))
	sb.WriteString("\n")
	sb.WriteString(strings.Repeat("─", width))
	sb.WriteString("\n")

	// Track tree state for proper branch rendering
	treeState := make([]bool, 10) // tracks if parent has more siblings at each depth

	// Render items
	for i, item := range items {
		// Check if this is the last item at root level
		isLast := i == len(items)-1
		renderWorkItemTreeRow(&sb, item, 0, filter, idWidth, typeWidth, titleWidth, stateWidth, assignedWidth, spacing, treeState, isLast)
	}

	return sb.String()
}

// getWorkItemTypeColor returns color style for work item type
func getWorkItemTypeColor(wiType string) *color.Color {
	switch wiType {
	case "Feature":
		return color.New(color.FgMagenta, color.Bold)
	case "Epic":
		return color.New(color.FgCyan, color.Bold)
	case "Product Backlog Item", "PBI", "User Story":
		return color.New(color.FgBlue, color.Bold)
	case "Bug":
		return color.New(color.FgRed)
	case "Task":
		return color.New(color.FgGreen)
	case "Test Case":
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgWhite)
	}
}

// getStateColor returns color style for work item state
func getStateColor(state string) *color.Color {
	switch state {
	case "Done", "Closed", "Resolved":
		return color.New(color.FgGreen)
	case "Active", "In Progress":
		return color.New(color.FgBlue)
	case "New", "To Do", "Approved":
		return color.New(color.FgHiBlack)
	case "Committed":
		return color.New(color.FgYellow)
	case "Removed":
		return color.New(color.FgRed)
	default:
		return color.New(color.FgWhite)
	}
}

// getIDColor returns color style for work item ID
func getIDColor(wiType string) *color.Color {
	switch wiType {
	case "Feature", "Epic":
		return color.New(color.FgHiWhite, color.Bold)
	case "Product Backlog Item", "PBI", "User Story":
		return color.New(color.FgHiBlue)
	case "Bug":
		return color.New(color.FgHiRed)
	case "Task":
		return color.New(color.FgHiGreen)
	default:
		return color.New(color.FgHiBlack)
	}
}

// renderWorkItemTreeRow renders a row with tree-style connectors in the ID column
func renderWorkItemTreeRow(sb *strings.Builder, item models.WorkItemHierarchy, depth int, filter string,
	idWidth, typeWidth, titleWidth, stateWidth, assignedWidth, spacing int, treeState []bool, isLast bool) {

	// Build tree prefix for ID column
	prefix := buildTreePrefix(depth, treeState, isLast)

	// Color for tree prefix
	treeColor := color.New(color.FgHiBlack)

	title := item.Title
	// Handle context marker for filtered views
	contextPrefix := ""
	if !item.IsAssignedToUser && filter != "" {
		contextPrefix = "[ctx] "
	}
	titleWithPrefix := contextPrefix + title

	assigned := item.AssignedTo
	if assigned != "" {
		assigned = truncateEmail(assigned)
	}

	// Shorten work item type for display
	shortType := shortenWorkItemType(item.Type)

	// Get colors
	idColor := getIDColor(item.Type)
	typeColor := getWorkItemTypeColor(item.Type)
	stateColor := getStateColor(item.State)
	assignedColor := color.New(color.FgHiBlack)

	// Build ID with color (keeping tree prefix dim)
	idWithColor := fmt.Sprintf("%s%s", treeColor.Sprint(prefix), idColor.Sprintf("#%d", item.ID))

	sb.WriteString(padRight(truncate(idWithColor, idWidth), idWidth))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(padRight(truncate(typeColor.Sprint(shortType), typeWidth), typeWidth))
	sb.WriteString(strings.Repeat(" ", spacing))

	// Title - dim if not assigned in filtered view
	titleColor := color.New()
	if !item.IsAssignedToUser && filter != "" {
		titleColor = color.New(color.FgHiBlack)
	}
	sb.WriteString(padRight(truncate(titleColor.Sprint(titleWithPrefix), titleWidth), titleWidth))
	sb.WriteString(strings.Repeat(" ", spacing))

	sb.WriteString(padRight(truncate(stateColor.Sprint(item.State), stateWidth), stateWidth))
	sb.WriteString(strings.Repeat(" ", spacing))
	sb.WriteString(truncate(assignedColor.Sprint(assigned), assignedWidth))
	sb.WriteString("\n")

	// Render children with updated tree state
	if len(item.Children) > 0 {
		// Mark that the current depth has children (for tree connectors)
		treeState[depth] = !isLast
		for i, child := range item.Children {
			childIsLast := i == len(item.Children)-1
			renderWorkItemTreeRow(sb, child, depth+1, filter, idWidth, typeWidth, titleWidth, stateWidth, assignedWidth, spacing, treeState, childIsLast)
		}
	}
}

// buildTreePrefix creates tree-style connectors for hierarchical display
// Uses: ├── for intermediate children, └── for last child, │ for continuing parent lines
func buildTreePrefix(depth int, treeState []bool, isLast bool) string {
	if depth == 0 {
		return ""
	}

	var sb strings.Builder
	// Build prefix for each level
	for i := 0; i < depth-1; i++ {
		if treeState[i] {
			sb.WriteString("│  ") // Parent has more siblings, draw vertical line
		} else {
			sb.WriteString("   ") // Parent is last, no vertical line
		}
	}

	// Add the branch connector for current level
	if isLast {
		sb.WriteString("└─ ")
	} else {
		sb.WriteString("├─ ")
	}

	return sb.String()
}

// renderWorkItemModern renders a work item in modern format with proper indentation
func renderWorkItemModern(sb *strings.Builder, item models.WorkItemHierarchy, depth int, filter string) {
	indent := strings.Repeat("  ", depth)
	icon := getWorkItemIcon(item.Type)

	nameStyle := color.New()
	if !item.IsAssignedToUser && filter != "" {
		// Parent item not assigned to user (shown for context)
		nameStyle = color.New(color.Faint)
	}

	stateStyle := color.New()
	switch item.State {
	case "Done", "Closed", "Resolved":
		stateStyle = color.New(color.FgGreen)
	case "Active", "In Progress":
		stateStyle = color.New(color.FgBlue)
	case "New", "To Do":
		stateStyle = color.New(color.Faint)
	default:
		stateStyle = color.New(color.FgYellow)
	}

	title := item.Title
	if len(title) > 60 {
		title = title[:57] + "..."
	}

	assigned := ""
	if item.AssignedTo != "" {
		assigned = fmt.Sprintf(" (@%s)", truncateEmail(item.AssignedTo))
	}

	sb.WriteString(fmt.Sprintf("%s%s %s [%s] %s%s\n",
		indent,
		icon,
		nameStyle.Sprintf("#%d", item.ID),
		stateStyle.Sprint(item.State),
		title,
		assigned,
	))

	for _, child := range item.Children {
		renderWorkItemModern(sb, child, depth+1, filter)
	}
}

// getWorkItemIcon returns an emoji/icon for a work item type
func getWorkItemIcon(workItemType string) string {
	switch workItemType {
	case "Feature":
		return "⭐"
	case "Epic":
		return "🚀"
	case "Product Backlog Item", "PBI", "User Story":
		return "📋"
	case "Bug":
		return "🐛"
	case "Task":
		return "✅"
	case "Test Case":
		return "🧪"
	case "Issue":
		return "⚠️"
	default:
		return "📄"
	}
}

// truncateEmail extracts just the name part of an email or shortens it
func truncateEmail(email string) string {
	if idx := strings.Index(email, "@"); idx > 0 {
		return email[:idx]
	}
	if idx := strings.Index(email, " <"); idx > 0 {
		return email[:idx]
	}
	return email
}

// GetFormat extracts the format from command flags
func GetFormat(cmd *cobra.Command) string {
	format, _ := cmd.Flags().GetString("format")
	if format == "" {
		format = "table"
	}
	return format
}

// renderWorkItemsJSON renders work items as JSON
func renderWorkItemsJSON(items []models.WorkItemHierarchy) ([]byte, error) {
	return json.MarshalIndent(items, "", "  ")
}

// renderWorkItemsYAML renders work items as YAML
func renderWorkItemsYAML(items []models.WorkItemHierarchy) ([]byte, error) {
	return yaml.Marshal(items)
}

// renderIterationsJSON renders iterations as JSON
func renderIterationsJSON(iterations []models.Iteration) ([]byte, error) {
	return json.MarshalIndent(iterations, "", "  ")
}

// renderIterationsYAML renders iterations as YAML
func renderIterationsYAML(iterations []models.Iteration) ([]byte, error) {
	return yaml.Marshal(iterations)
}
