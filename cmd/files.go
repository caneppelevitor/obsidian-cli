package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

var filesCmd = &cobra.Command{
	Use:   "files",
	Short: "List all markdown files in vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		vp, err := resolveVaultPath()
		if err != nil {
			return err
		}
		return runFiles(vp)
	},
}

func init() {
	rootCmd.AddCommand(filesCmd)
}

func runFiles(vaultPath string) error {
	infos, err := vault.ListMarkdownFilesWithInfo(vaultPath)
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}

	if len(infos) == 0 {
		fmt.Println("No markdown files found in vault")
		return nil
	}

	fmt.Println("\nMarkdown files in vault:")
	fmt.Println("──────────────────────────────────────────────────")

	for i, info := range infos {
		fmt.Printf("%2d. %s (%s)\n", i+1, info.RelPath, info.ModTime)
	}

	fmt.Println("──────────────────────────────────────────────────")
	return nil
}
