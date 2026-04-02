package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
)

var vaultPath string

var rootCmd = &cobra.Command{
	Use:   "obsidian",
	Short: "CLI tool for managing Obsidian vault with interactive interface",
	Long:  "A CLI tool for managing daily notes in Obsidian vaults with section-based organization and automatic task logging.",
	RunE: func(cmd *cobra.Command, args []string) error {
		vp, err := resolveVaultPath()
		if err != nil {
			return err
		}
		return runDaily(vp)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&vaultPath, "vault", "v", "", "Path to Obsidian vault")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// resolveVaultPath returns the vault path from flag or config.
func resolveVaultPath() (string, error) {
	if vaultPath != "" {
		return vaultPath, nil
	}
	vp, err := config.GetVaultPath()
	if err != nil {
		return "", fmt.Errorf("reading config: %w", err)
	}
	if vp == "" {
		return "", fmt.Errorf("no vault path configured. Run \"obsidian init\" first")
	}
	return vp, nil
}
