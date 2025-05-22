package commands

import (
	"bytes" // For replacing content in embedded file
	"fmt"

	// "io/fs" // Unused import
	"log"
	"os"
	"path/filepath"

	// For replacing content
	// For replacing content
	"github.com/AlecAivazis/survey/v2"
	"github.com/mad01/dotter/internal/config"
	"github.com/spf13/cobra"
)

//go:embed ../../configs/examples/default.config.toml
var defaultConfigContentBytes []byte

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dotter configuration",
	Long:  `Initializes a new dotter configuration file and provides guidance on next steps.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initializing dotter...")

		defaultConfigPath, err := config.GetDefaultConfigPath()
		if err != nil {
			log.Fatalf("Error: Could not determine default config path: %v", err)
		}

		// Check if config file already exists
		if _, err := os.Stat(defaultConfigPath); err == nil {
			overwrite := false
			prompt := &survey.Confirm{
				Message: fmt.Sprintf("Configuration file already exists at %s. Overwrite?", defaultConfigPath),
			}
			survey.AskOne(prompt, &overwrite)
			if !overwrite {
				fmt.Println("Initialization cancelled.")
				return
			}
		} else if !os.IsNotExist(err) {
			log.Fatalf("Error checking for config file at %s: %v", defaultConfigPath, err)
		}

		// Get dotfiles repository path from user
		dotfilesRepoPathInput := ""
		// Try to read the current value from the embedded config if it exists and is valid TOML
		// This is a bit advanced; for now, we'll use a simpler default.
		defaultRepoPathSuggestion, _ := config.ExpandPath("~/.dotfiles") // Suggest a default

		promptRepo := &survey.Input{
			Message: "Enter the path to your dotfiles source repository (e.g., ~/.dotfiles_src):",
			Default: defaultRepoPathSuggestion,
		}
		survey.AskOne(promptRepo, &dotfilesRepoPathInput, survey.WithValidator(survey.Required))

		expandedRepoPath, err := config.ExpandPath(dotfilesRepoPathInput)
		if err != nil {
			log.Fatalf("Error expanding repository path '%s': %v", dotfilesRepoPathInput, err)
		}

		// Use embedded default config content and replace the dotfiles_repo_path
		finalConfigContent := bytes.ReplaceAll(defaultConfigContentBytes,
			[]byte("dotfiles_repo_path = \"~/.dotfiles\""), // The placeholder in the embedded file
			[]byte(fmt.Sprintf("dotfiles_repo_path = \"%s\"", expandedRepoPath)),
		)

		configDir := filepath.Dir(defaultConfigPath)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			log.Fatalf("Error creating config directory %s: %v", configDir, err)
		}

		if err := os.WriteFile(defaultConfigPath, finalConfigContent, 0644); err != nil {
			log.Fatalf("Error writing default configuration to %s: %v", defaultConfigPath, err)
		}
		fmt.Printf("Default configuration file created at %s\n", defaultConfigPath)

		fmt.Println("\nNext Steps:")
		fmt.Printf("1. Populate your dotfiles repository at '%s'.\n", expandedRepoPath)
		fmt.Printf("2. Customize your '%s' with the dotfiles, tools, and shell settings you want to manage.\n", defaultConfigPath)
		fmt.Println("3. Run 'dotter apply' to apply your configurations.")
		fmt.Println("\nImportant: It is highly recommended to commit your dotfiles source repository (and potentially this config file if you symlink it there) to version control (e.g., Git).")

		fmt.Println("\nTip: Consider version controlling your dotter config file by placing it in your dotfiles repository and symlinking it to the default location.")

	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// Helper to read embedded or relative file - for future use if default.config.toml is properly packaged
// This function is no longer needed as we are using go:embed directly for defaultConfigContentBytes
/*
func getDefaultConfigTemplateContent() ([]byte, error) {
	// This is a placeholder. Ideally, use go:embed for the default config.
	// For now, trying to read from a relative path that might not always work.
	// A more robust solution for development could be to find it relative to this source file.
	// For release, go:embed is the way.
	// relPath := "../../configs/examples/default.config.toml" // Path relative to this file - Unused
	// To make this slightly more robust for now, try to get CWD and build path from there
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("could not get current working directory: %w", err)
	}
	// This assumes `configs` is at the project root where `go run` or the binary might be executed.
	// This is fragile.
	absPath := filepath.Join(cwd, "configs", "examples", "default.config.toml")
	fmt.Printf("Attempting to load default config template from: %s\n", absPath) // Debugging line

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("could not read default config template at '%s' (ensure it exists or use go:embed): %w", absPath, err)
	}
	return content, nil
}
*/
