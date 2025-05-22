package shell

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mad01/dotter/internal/config"
)

const (
	GeneratedAliasesFilename   = "generated_aliases.sh"
	GeneratedFunctionsFilename = "generated_functions.sh"
)

// GenerateShellConfigs generates script files for aliases and functions
// and returns the paths to the generated files and any errors.
// If dryRun is true, it prints what it would do and returns the prospective paths,
// but does not write any files.
func GenerateShellConfigs(cfg *config.Config, shellType SupportedShell, dryRun bool) (aliasFilePath string, funcFilePath string, err error) {
	generatedDir, err := GetDotterGeneratedDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get dotter generated scripts directory: %w", err)
	}
	if !dryRun {
		if err := os.MkdirAll(generatedDir, 0755); err != nil {
			return "", "", fmt.Errorf("failed to create directory for generated shell scripts '%s': %w", generatedDir, err)
		}
	} else {
		if _, statErr := os.Stat(generatedDir); os.IsNotExist(statErr) {
			fmt.Printf("[DRY RUN] Would create directory for generated shell scripts: %s\n", generatedDir)
		}
	}

	aliasFilePath = filepath.Join(generatedDir, GeneratedAliasesFilename)
	funcFilePath = filepath.Join(generatedDir, GeneratedFunctionsFilename)

	// Generate Aliases
	if len(cfg.Shell.Aliases) > 0 {
		var aliasContent strings.Builder
		aliasContent.WriteString("#!/bin/sh\n")
		aliasContent.WriteString("# Dotter generated aliases - DO NOT EDIT MANUALLY\n\n")
		for name, command := range cfg.Shell.Aliases {
			// Basic sanitization for alias name and command could be added here if necessary
			aliasContent.WriteString(fmt.Sprintf("alias %s='%s'\n", name, strings.ReplaceAll(command, "'", "'\\''")))
		}
		if dryRun {
			fmt.Printf("[DRY RUN] Would write generated aliases to: %s\n", aliasFilePath)
		} else {
			if err := os.WriteFile(aliasFilePath, []byte(aliasContent.String()), 0644); err != nil {
				return aliasFilePath, "", fmt.Errorf("failed to write generated aliases file '%s': %w", aliasFilePath, err)
			}
			fmt.Printf("Generated aliases at: %s\n", aliasFilePath)
		}
	} else {
		if !dryRun { // Only attempt removal if not in dry run
			if _, err := os.Stat(aliasFilePath); err == nil { // Check if file exists before removing
				if err := os.Remove(aliasFilePath); err != nil {
					log.Printf("Warning: could not remove existing empty alias file %s: %v\n", aliasFilePath, err)
				}
			}
		}
		aliasFilePath = "" // Indicate no file generated
	}

	// Generate Functions
	if len(cfg.Shell.Functions) > 0 {
		var funcContent strings.Builder
		funcContent.WriteString("#!/bin/sh\n") // Or make this dependent on shellType for more complex functions
		funcContent.WriteString("# Dotter generated functions - DO NOT EDIT MANUALLY\n\n")
		for name, function := range cfg.Shell.Functions {
			// For POSIX shells, function syntax is: func_name() { body }
			// Fish shell syntax is different: function func_name; body; end;
			// For now, sticking to POSIX sh compatible.
			if shellType == Fish {
				funcContent.WriteString(fmt.Sprintf("function %s\n  %s\nend\n\n", name, strings.TrimSpace(function.Body)))
			} else {
				funcContent.WriteString(fmt.Sprintf("%s() {\n%s\n}\n\n", name, strings.TrimSpace(function.Body)))
			}
		}
		if dryRun {
			fmt.Printf("[DRY RUN] Would write generated functions to: %s\n", funcFilePath)
		} else {
			if err := os.WriteFile(funcFilePath, []byte(funcContent.String()), 0644); err != nil {
				return aliasFilePath, funcFilePath, fmt.Errorf("failed to write generated functions file '%s': %w", funcFilePath, err)
			}
			fmt.Printf("Generated functions at: %s\n", funcFilePath)
		}
	} else {
		if !dryRun { // Only attempt removal if not in dry run
			if _, err := os.Stat(funcFilePath); err == nil { // Check if file exists before removing
				if err := os.Remove(funcFilePath); err != nil {
					log.Printf("Warning: could not remove existing empty function file %s: %v\n", funcFilePath, err)
				}
			}
		}
		funcFilePath = "" // Indicate no file generated
	}

	return aliasFilePath, funcFilePath, nil
}
