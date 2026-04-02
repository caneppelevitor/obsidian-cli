package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
	"github.com/caneppelevitor/obsidian-cli/internal/tasks"
)

var (
	pendingOnly   bool
	recentDays    string
	completeIndex string
)

var tasksCmd = &cobra.Command{
	Use:   "tasks",
	Short: "View and manage tasks from the centralized task log",
	RunE: func(cmd *cobra.Command, args []string) error {
		vp, err := resolveVaultPath()
		if err != nil {
			return err
		}
		return runTasks(vp)
	},
}

func init() {
	tasksCmd.Flags().BoolVar(&pendingOnly, "pending", false, "Show only unchecked tasks")
	tasksCmd.Flags().StringVar(&recentDays, "recent", "", "Show tasks from last N days (default: 7)")
	tasksCmd.Flags().StringVar(&completeIndex, "complete", "", "Mark task as complete by index number")
	rootCmd.AddCommand(tasksCmd)
}

func runTasks(vaultPath string) error {
	taskLogFile, err := config.GetTaskLogFile()
	if err != nil {
		return fmt.Errorf("getting task log file: %w", err)
	}
	taskLogPath := filepath.Join(vaultPath, taskLogFile)

	allTasks, err := tasks.ReadTaskLog(taskLogPath)
	if err != nil {
		return fmt.Errorf("reading task log: %w", err)
	}

	if allTasks == nil {
		fmt.Println("No task log found. Create some tasks first!")
		return nil
	}

	// Complete a task
	if completeIndex != "" {
		idx, err := strconv.Atoi(completeIndex)
		if err != nil {
			return fmt.Errorf("invalid task index: %s", completeIndex)
		}
		if err := tasks.CompleteTask(taskLogPath, idx-1, allTasks); err != nil {
			return err
		}
		fmt.Printf("Task completed: %s\n", allTasks[idx-1].Content)
		return nil
	}

	// Filter
	filtered := allTasks
	if pendingOnly {
		var pending []tasks.Task
		for _, t := range filtered {
			if !t.Completed {
				pending = append(pending, t)
			}
		}
		filtered = pending
	}

	if recentDays != "" {
		days, err := strconv.Atoi(recentDays)
		if err != nil {
			days = 7
		}
		filtered = tasks.FilterRecent(filtered, days, vaultPath)
	}

	// Display
	if len(filtered) == 0 {
		fmt.Println("No tasks found matching your criteria.")
		return nil
	}

	title := "All Tasks"
	if pendingOnly {
		title = "Pending Tasks"
	} else if recentDays != "" {
		title = fmt.Sprintf("Tasks from last %s days", recentDays)
	}

	fmt.Printf("\n%s:\n", title)
	fmt.Println("──────────────────────────────────────────────────")

	for i, task := range filtered {
		fmt.Println(tasks.FormatTaskDisplay(task, i))
	}

	if !pendingOnly && completeIndex == "" {
		pendingCount := 0
		completedCount := 0
		for _, t := range filtered {
			if t.Completed {
				completedCount++
			} else {
				pendingCount++
			}
		}
		fmt.Println("──────────────────────────────────────────────────")
		fmt.Printf("Total: %d | Pending: %d | Completed: %d\n", len(filtered), pendingCount, completedCount)
	}

	fmt.Println("\nTip: Use --complete <number> to mark a task as done")
	return nil
}
