package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dropalltables/canvas/internal/api"
	"github.com/dropalltables/canvas/internal/config"
	"github.com/dropalltables/canvas/internal/ui"
	"github.com/spf13/cobra"
)

var (
	flagFormat  string
	flagPast    int
	flagFuture  int
	flagAll     bool
	flagCourse  string
	flagType    string
)

var assignmentsCmd = &cobra.Command{
	Use:     "assignments",
	Aliases: []string{"a", "ls"},
	Short:   "List assignments",
	Long:    "List Canvas assignments with due dates and status",
	RunE:    runAssignmentsList,
}

var doneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark assignment as done",
	Args:  cobra.ExactArgs(1),
	RunE:  runAssignmentsDone,
}

var rawCmd = &cobra.Command{
	Use:    "raw",
	Short:  "Show raw API response (debug)",
	Hidden: true,
	RunE:   runAssignmentsRaw,
}

func init() {
	assignmentsCmd.Flags().StringVarP(&flagFormat, "format", "f", "table", "Output format: table, json, compact")
	assignmentsCmd.Flags().IntVarP(&flagPast, "past", "p", 7, "Days in the past to include")
	assignmentsCmd.Flags().IntVarP(&flagFuture, "future", "F", 30, "Days in the future to include")
	assignmentsCmd.Flags().BoolVarP(&flagAll, "all", "a", false, "Include completed assignments")
	assignmentsCmd.Flags().StringVarP(&flagCourse, "course", "c", "", "Filter by course name (substring match)")
	assignmentsCmd.Flags().StringVarP(&flagType, "type", "t", "", "Filter by type: assignment, quiz, discussion")

	rawCmd.Flags().IntVarP(&flagPast, "past", "p", 14, "Days in the past")
	rawCmd.Flags().IntVarP(&flagFuture, "future", "F", 30, "Days in the future")

	assignmentsCmd.AddCommand(doneCmd)
	assignmentsCmd.AddCommand(rawCmd)
}

func getClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		if err == config.ErrNotConfigured {
			ui.Error("Not logged in. Run 'canvas auth login' to authenticate.")
			return nil, err
		}
		return nil, err
	}
	return api.NewClient(cfg), nil
}

func runAssignmentsList(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	assignments, err := client.GetAssignments(flagPast, flagFuture)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.IsUnauthorized() {
			ui.Error("Unauthorized. Run 'canvas auth login' to authenticate.")
			return err
		}
		ui.Error("Failed to fetch: %v", err)
		return err
	}

	filtered := filterAssignments(assignments)

	if flagFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(filtered)
	}

	if len(filtered) == 0 {
		fmt.Println("No assignments found.")
		return nil
	}

	switch flagFormat {
	case "compact":
		printCompact(filtered)
	default:
		printTable(filtered)
	}

	return nil
}

func filterAssignments(assignments []api.Assignment) []api.Assignment {
	var result []api.Assignment
	for _, a := range assignments {
		if !flagAll && a.Completed {
			continue
		}
		if flagCourse != "" && !strings.Contains(strings.ToLower(a.ContextName), strings.ToLower(flagCourse)) {
			continue
		}
		if flagType != "" {
			typeMap := map[string]string{
				"assignment": "assignment",
				"quiz":       "quiz",
				"discussion": "discussion_topic",
			}
			if mapped, ok := typeMap[flagType]; ok {
				if a.PlannableType != mapped {
					continue
				}
			}
		}
		result = append(result, a)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].DueAt == nil {
			return false
		}
		if result[j].DueAt == nil {
			return true
		}
		return result[i].DueAt.Before(*result[j].DueAt)
	})

	return result
}

func printTable(assignments []api.Assignment) {
	now := time.Now()

	maxTitleLen := 50
	maxCourseLen := 25

	for _, a := range assignments {
		dueStr := formatDue(a.DueAt, now)
		status := formatStatus(a)
		title := truncate(a.Plannable.Title, maxTitleLen)
		course := truncate(a.ContextName, maxCourseLen)
		typeIcon := typeIcon(a.PlannableType)

		var line string
		if a.Completed {
			line = fmt.Sprintf("  %s %s  %-12s  %-*s  %s",
				status, typeIcon, dueStr, maxCourseLen, course, title)
			fmt.Println(ui.Faint(line))
		} else if a.DueAt != nil && a.DueAt.Before(now) {
			line = fmt.Sprintf("  %s %s  %s  %-*s  %s",
				status, typeIcon, ui.Red(fmt.Sprintf("%-12s", dueStr)), maxCourseLen, course, title)
			fmt.Println(line)
		} else if a.DueAt != nil && a.DueAt.Before(now.Add(24*time.Hour)) {
			line = fmt.Sprintf("  %s %s  %s  %-*s  %s",
				status, typeIcon, ui.Yellow(fmt.Sprintf("%-12s", dueStr)), maxCourseLen, course, title)
			fmt.Println(line)
		} else {
			line = fmt.Sprintf("  %s %s  %-12s  %-*s  %s",
				status, typeIcon, dueStr, maxCourseLen, course, title)
			fmt.Println(line)
		}
	}

	fmt.Printf("\n%s assignments\n", ui.Faint(fmt.Sprintf("%d", len(assignments))))
}

func printCompact(assignments []api.Assignment) {
	now := time.Now()
	for _, a := range assignments {
		dueStr := formatDue(a.DueAt, now)
		status := "[ ]"
		if a.Completed {
			status = "[x]"
		}
		fmt.Printf("%s %-12s %s\n", status, dueStr, a.Plannable.Title)
	}
}

func formatDue(due *time.Time, now time.Time) string {
	if due == nil {
		return "No due date"
	}

	days := int(due.Sub(now).Hours() / 24)
	if days < 0 {
		return fmt.Sprintf("%dd overdue", -days)
	}
	if days == 0 {
		return due.Format("Today 15:04")
	}
	if days == 1 {
		return due.Format("Tomorrow 15:04")
	}
	if days < 7 {
		return due.Format("Mon 15:04")
	}
	return due.Format("Jan 02")
}

func formatStatus(a api.Assignment) string {
	if a.Completed {
		return ui.Green("[x]")
	}
	return "[ ]"
}

func typeIcon(t string) string {
	switch t {
	case "assignment":
		return "A"
	case "quiz":
		return "Q"
	case "discussion_topic":
		return "D"
	default:
		return "?"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "~"
}

func runAssignmentsDone(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	var id int
	if _, err := fmt.Sscanf(args[0], "%d", &id); err != nil {
		ui.Error("Invalid ID: %s", args[0])
		return err
	}

	assignments, err := client.GetAssignments(30, 30)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.IsUnauthorized() {
			ui.Error("Unauthorized. Run 'canvas auth login' to authenticate.")
			return err
		}
		ui.Error("Failed to fetch: %v", err)
		return err
	}

	var target *api.Assignment
	for _, a := range assignments {
		if a.PlannableID == id {
			target = &a
			break
		}
	}

	if target == nil {
		ui.Error("Assignment %d not found", id)
		return fmt.Errorf("not found")
	}

	if err := client.MarkDone(target.PlannableType, target.PlannableID); err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.IsUnauthorized() {
			ui.Error("Unauthorized. Run 'canvas auth login' to authenticate.")
			return err
		}
		ui.Error("Failed: %v", err)
		return err
	}

	fmt.Printf("Marked '%s' as done\n", target.Plannable.Title)
	return nil
}

func runAssignmentsRaw(cmd *cobra.Command, args []string) error {
	client, err := getClient()
	if err != nil {
		return err
	}

	data, err := client.GetRawPlannerItems(flagPast, flagFuture)
	if err != nil {
		if apiErr, ok := err.(*api.APIError); ok && apiErr.IsUnauthorized() {
			ui.Error("Unauthorized. Run 'canvas auth login' to authenticate.")
			return err
		}
		ui.Error("Failed: %v", err)
		return err
	}

	var pretty json.RawMessage
	if err := json.Unmarshal(data, &pretty); err != nil {
		fmt.Println(string(data))
		return nil
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(pretty)
}
