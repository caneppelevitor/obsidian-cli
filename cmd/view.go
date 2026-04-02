package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/caneppelevitor/obsidian-cli/internal/vault"
)

var viewCmd = &cobra.Command{
	Use:   "view",
	Short: "View mode to browse files in vault",
	RunE: func(cmd *cobra.Command, args []string) error {
		vp, err := resolveVaultPath()
		if err != nil {
			return err
		}
		return runView(vp)
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
}

func runView(vaultPath string) error {
	files, err := vault.ListMarkdownFiles(vaultPath)
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No markdown files found in vault")
		return nil
	}

	// Build options for selection
	options := make([]huh.Option[string], len(files))
	for i, f := range files {
		options[i] = huh.NewOption(fmt.Sprintf("%2d. %s", i+1, f), f)
	}

	var selected string
	err = huh.NewSelect[string]().
		Title("Select a file to view:").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil // User cancelled
	}

	return displayFile(vaultPath, selected)
}

func displayFile(vaultPath, filename string) error {
	filePath := filepath.Join(vaultPath, filename)
	fileContent, err := vault.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file %s: %w", filename, err)
	}

	if fileContent == "" {
		fmt.Println("File is empty")
		return nil
	}

	fmt.Println("\n" + strings.Repeat("═", 80))
	fmt.Printf("Viewing: %s\n", filename)
	fmt.Println(strings.Repeat("═", 80))

	// Render markdown with Glamour
	rendered, err := glamour.Render(fileContent, "dark")
	if err != nil {
		// Fallback to plain text with line numbers
		lines := strings.Split(fileContent, "\n")
		for i, line := range lines {
			fmt.Printf("%3d │ %s\n", i+1, line)
		}
	} else {
		fmt.Print(rendered)
	}

	fmt.Println(strings.Repeat("═", 80))
	return nil
}

// resolveFileByNameOrNumber resolves a file reference that could be a name or number.
func resolveFileByNameOrNumber(input string, files []string) (string, bool) {
	num, err := strconv.Atoi(input)
	if err == nil && num > 0 && num <= len(files) {
		return files[num-1], true
	}
	for _, f := range files {
		if f == input {
			return f, true
		}
	}
	return "", false
}
