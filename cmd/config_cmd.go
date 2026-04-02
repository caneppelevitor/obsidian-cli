package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
)

var (
	showConfig bool
	editConfig bool
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show current configuration or create/edit config file",
	RunE: func(cmd *cobra.Command, args []string) error {
		if editConfig {
			return runConfigEdit()
		}
		if showConfig {
			return runConfigShow()
		}
		return runConfigStatus()
	},
}

func init() {
	configCmd.Flags().BoolVar(&showConfig, "show", false, "Show current configuration")
	configCmd.Flags().BoolVar(&editConfig, "edit", false, "Open config file in default editor")
	rootCmd.AddCommand(configCmd)
}

func runConfigShow() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("formatting config: %w", err)
	}

	fmt.Println("Current Configuration:")
	fmt.Println(string(data))
	return nil
}

func runConfigEdit() error {
	// Ensure config file exists
	if _, err := os.Stat(config.ConfigFile); os.IsNotExist(err) {
		cfg, loadErr := config.Load()
		if loadErr != nil {
			cfg = config.DefaultConfig()
		}
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("creating config file: %w", err)
		}
		fmt.Printf("Created config file: %s\n", config.ConfigFile)
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	cmd := exec.Command(editor, config.ConfigFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runConfigStatus() error {
	vp, _ := config.GetVaultPath()

	fmt.Printf("Config file: %s\n", config.ConfigFile)
	if vp != "" {
		fmt.Printf("Current vault: %s\n", vp)
	} else {
		fmt.Println("No vault configured")
	}
	fmt.Println("Use --show to see full config, --edit to modify")
	return nil
}
