package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/tui"
	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

var dailyCmd = &cobra.Command{
	Use:   "daily",
	Short: "Open or create today's daily note (interactive mode)",
	RunE: func(cmd *cobra.Command, args []string) error {
		vp, err := resolveVaultPath()
		if err != nil {
			return err
		}
		return runDaily(vp)
	},
}

func init() {
	rootCmd.AddCommand(dailyCmd)
}

func runDaily(vaultPath string) error {
	filePath, content, created, err := vault.EnsureDailyNote(vaultPath)
	if err != nil {
		return fmt.Errorf("opening daily note: %w", err)
	}

	if created {
		fmt.Printf("Created new daily note: %s\n", vault.DailyNoteFilename())
	} else {
		fmt.Printf("Opening existing daily note: %s\n", vault.DailyNoteFilename())
	}

	return tui.Run(vaultPath, filePath, content)
}
