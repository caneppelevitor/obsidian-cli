package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/config"
)

var sampleConfig bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize Obsidian CLI with YAML configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		if sampleConfig {
			return createSampleConfig()
		}
		return runInit()
	},
}

func init() {
	initCmd.Flags().BoolVar(&sampleConfig, "sample-config", false, "Create a sample YAML config file in current directory")
	rootCmd.AddCommand(initCmd)
}

func runInit() error {
	vp := vaultPath

	dirValidator := func(s string) error {
		if s == "" {
			return nil // allow empty for optional fields
		}
		info, err := os.Stat(s)
		if err != nil {
			return fmt.Errorf("path does not exist or is not accessible")
		}
		if !info.IsDir() {
			return fmt.Errorf("path must be a directory")
		}
		return nil
	}

	if vp == "" {
		var inputPath string
		err := huh.NewInput().
			Title("Where are your daily notes stored?").
			Description("Path to the folder where daily notes will be created").
			Value(&inputPath).
			Validate(func(s string) error {
				if s == "" {
					return fmt.Errorf("path is required")
				}
				return dirValidator(s)
			}).
			Run()
		if err != nil {
			return fmt.Errorf("input cancelled: %w", err)
		}
		vp = inputPath
	}

	// Ask for vault root path (for file browser)
	var rootPath string
	err := huh.NewInput().
		Title("Where is your Obsidian vault root?").
		Description("Root directory for the file browser (leave empty to use daily notes path)").
		Value(&rootPath).
		Validate(dirValidator).
		Run()
	if err != nil {
		return fmt.Errorf("input cancelled: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	cfg.Vault.DefaultPath = vp
	if rootPath != "" {
		cfg.Vault.RootPath = rootPath
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("\nConfiguration created: %s\n", config.ConfigFile)
	fmt.Printf("  Daily notes path: %s\n", vp)
	if rootPath != "" {
		fmt.Printf("  Vault root path:  %s\n", rootPath)
	}
	fmt.Println("\nUse \"obsidian config --edit\" to customize further")
	fmt.Println("Run \"obsidian daily\" to get started!")

	return nil
}

func createSampleConfig() error {
	samplePath := filepath.Join(".", "obsidian-cli.config.yaml")
	cfg := config.DefaultConfig()
	if err := writeSampleConfig(samplePath, cfg); err != nil {
		return fmt.Errorf("creating sample config: %w", err)
	}
	fmt.Printf("Sample config created: %s\n", samplePath)
	fmt.Println("Edit this file and copy it to ~/.obsidian-cli/config.yaml")
	return nil
}

func writeSampleConfig(path string, cfg config.AppConfig) error {
	return os.WriteFile(path, []byte(sampleConfigYAML), 0o644)
}

const sampleConfigYAML = `# Obsidian CLI Configuration
# Copy this file to ~/.obsidian-cli/config.yaml

vault:
  defaultPath: "/path/to/your/vault/daily-notes"
  rootPath: "/path/to/your/vault"  # Root for file browser (defaults to defaultPath)

logging:
  tasks:
    logFile: "tasks-log.md"
    autoLog: true
  ideas:
    logFile: "ideas-log.md"
    autoLog: true
  questions:
    logFile: "questions-log.md"
    autoLog: true
  insights:
    logFile: "insights-log.md"
    autoLog: true
  timestampFormat: "simple"

dailyNotes:
  sections:
    - "Daily Log"
    - "Tasks"
    - "Ideas"
    - "Questions"
    - "Insights"
    - "Links to Expand"
  tags:
    - "#daily"
    - "#inbox"
  titleFormat: "YYYY-MM-DD"

interface:
  theme:
    border: "cyan"
    title: "white"
    content: "white"
    input: "yellow"
    highlight: "green"
  autoScroll: true
  showLineNumbers: true
  eisenhowerTags:
    "#do": "131"
    "#delegate": "180"
    "#schedule": "66"
    "#eliminate": "145"

organization:
  sectionPrefixes:
    "[]": "Tasks"
    "-": "Ideas"
    "?": "Questions"
    "!": "Insights"

advanced:
  backup:
    enabled: false
    directory: ".obsidian-cli-backups"
    maxBackups: 5
  performance:
    maxFileSize: 10
    watchFiles: false
`
