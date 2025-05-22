package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DotterBlockBeginMarker = "# BEGIN DOTTER MANAGED BLOCK"
	DotterBlockEndMarker   = "# END DOTTER MANAGED BLOCK"
)

// SupportedShell represents a shell type that dotter can manage.
type SupportedShell string

const (
	Bash SupportedShell = "bash"
	Zsh  SupportedShell = "zsh"
	Fish SupportedShell = "fish"
	// Add other shells as needed (e.g., Powershell)
)

// GetRCFilePath returns the typical path for the RC file of a given shell.
func GetRCFilePath(shell SupportedShell) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get user home directory: %w", err)
	}

	switch shell {
	case Bash:
		return filepath.Join(homeDir, ".bashrc"), nil
	case Zsh:
		// Zsh can also use .zprofile for login shells or .zshenv for all invocations.
		// .zshrc is typically for interactive shells. This is usually what we want.
		// Check if ZDOTDIR is set, otherwise default to ~/.zshrc
		if zdotdir := os.Getenv("ZDOTDIR"); zdotdir != "" {
			return filepath.Join(zdotdir, ".zshrc"), nil
		}
		return filepath.Join(homeDir, ".zshrc"), nil
	case Fish:
		// Fish typically uses ~/.config/fish/config.fish
		configDir := filepath.Join(homeDir, ".config", "fish")
		return filepath.Join(configDir, "config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s", shell)
	}
}

// InjectSourceLines ensures that the specified sourceLine (e.g., "source ~/.config/dotter/generated.sh")
// is present in the dotter managed block of the given shell rc file.
// If the block doesn't exist, it's created.
// If the line already exists in the block, it's not added again.
// additionalLines are other lines to ensure are within the block.
// If dryRun is true, it prints what it would do instead of modifying the file.
func InjectSourceLines(shell SupportedShell, additionalLines []string, dryRun bool) error {
	rcFilePath, err := GetRCFilePath(shell)
	if err != nil {
		return fmt.Errorf("cannot get RC file path for %s: %w", shell, err)
	}

	rcDir := filepath.Dir(rcFilePath)
	if !dryRun {
		if err := os.MkdirAll(rcDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for rc file %s: %w", rcFilePath, err)
		}
	} else {
		// Check if dir exists in dry run for more accurate messaging
		if _, statErr := os.Stat(rcDir); os.IsNotExist(statErr) {
			fmt.Printf("[DRY RUN] Would create directory for rc file %s\n", rcDir)
		}
	}

	fileContent, err := os.ReadFile(rcFilePath)
	if os.IsNotExist(err) {
		fileContent = []byte{}
	} else if err != nil {
		return fmt.Errorf("failed to read rc file %s: %w", rcFilePath, err)
	}

	lines := strings.Split(string(fileContent), "\n")
	newLines, modified := ensureDotterBlock(lines, additionalLines)

	if modified {
		output := strings.Join(newLines, "\n")
		if output != "" && !strings.HasSuffix(output, "\n") {
			output += "\n"
		}
		if dryRun {
			fmt.Printf("[DRY RUN] Would update rc file: %s\n", rcFilePath)
			fmt.Println("[DRY RUN] New content would be:")
			fmt.Println(output) // Potentially long, consider summarizing or showing diff
		} else {
			fmt.Printf("Updating rc file: %s\n", rcFilePath)
			if err := os.WriteFile(rcFilePath, []byte(output), 0644); err != nil {
				return fmt.Errorf("failed to write updated rc file %s: %w", rcFilePath, err)
			}
		}
	} else {
		fmt.Printf("RC file %s is already up to date.\n", rcFilePath)
	}
	return nil
}

func ensureDotterBlock(lines []string, contentLines []string) ([]string, bool) {
	var newLines []string
	modified := false
	blockFound := false
	alreadyHasContent := make(map[string]bool)

	// First pass: find existing block and its content
	startIndex, endIndex := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == DotterBlockBeginMarker {
			startIndex = i
			blockFound = true
		}
		if strings.TrimSpace(line) == DotterBlockEndMarker && blockFound {
			endIndex = i
			break
		}
		if blockFound && startIndex != -1 && endIndex == -1 && i > startIndex {
			trimmedLine := strings.TrimSpace(line)
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") { // Ignore comments and empty lines within block for checking
				alreadyHasContent[trimmedLine] = true
			}
		}
	}

	// Determine if new content lines are actually new
	newContentToAdd := []string{}
	for _, cl := range contentLines {
		if !alreadyHasContent[strings.TrimSpace(cl)] {
			newContentToAdd = append(newContentToAdd, cl)
			modified = true // If any new line is to be added, we are modifying
		}
	}

	if !blockFound { // No block, append a new one
		newLines = append(newLines, lines...)
		// Remove trailing empty lines before adding new block
		for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
			newLines = newLines[:len(newLines)-1]
		}
		if len(newLines) > 0 {
			newLines = append(newLines, "") // Add a blank line before our block if file not empty
		}
		newLines = append(newLines, DotterBlockBeginMarker)
		newLines = append(newLines, contentLines...)
		newLines = append(newLines, DotterBlockEndMarker)
		return newLines, true // Definitely modified
	}

	// Block found, check if we need to add any new lines within it.
	if !modified && endIndex != -1 && startIndex != -1 { // No new lines to add, and block is well-formed
		// Also check if the number of lines to ensure matches what's there (excluding comments/blanks)
		currentBlockContentCount := 0
		for i := startIndex + 1; i < endIndex; i++ {
			trimmedLine := strings.TrimSpace(lines[i])
			if trimmedLine != "" && !strings.HasPrefix(trimmedLine, "#") {
				currentBlockContentCount++
			}
		}
		if currentBlockContentCount == len(contentLines) {
			return lines, false // No changes needed
		}
		// If counts differ even if all lines are present, it implies some reordering or extra lines are in contentLines
		// or some lines were removed from contentLines. In this case, we should rewrite the block.
		modified = true
	}

	// Reconstruct lines: either block not found, or needs modification/rewrite
	// This logic will effectively rewrite the block if it exists, or add it if it doesn't.
	// We iterate through the original lines and either copy them or, when we encounter
	// our block, we discard the old block and insert the new one.

	finalLines := []string{}
	if startIndex != -1 && endIndex != -1 { // Existing well-formed block, overwrite its content
		finalLines = append(finalLines, lines[:startIndex]...)
		finalLines = append(finalLines, DotterBlockBeginMarker)
		finalLines = append(finalLines, contentLines...)
		finalLines = append(finalLines, DotterBlockEndMarker)
		finalLines = append(finalLines, lines[endIndex+1:]...)
		return finalLines, true // modified is true if newContentToAdd had items or counts differed
	} else { // Block not found, or malformed (e.t. no end marker). Append to end.
		// Remove any partial/malformed block before appending
		cleanedLines := []string{}
		inPotentialBlock := false
		for _, line := range lines {
			if strings.TrimSpace(line) == DotterBlockBeginMarker {
				inPotentialBlock = true
				modified = true // Found a start, implies we want to rewrite
				continue
			}
			if strings.TrimSpace(line) == DotterBlockEndMarker && inPotentialBlock {
				inPotentialBlock = false
				continue
			}
			if !inPotentialBlock {
				cleanedLines = append(cleanedLines, line)
			}
		}

		// Remove trailing empty lines before adding new block
		for len(cleanedLines) > 0 && strings.TrimSpace(cleanedLines[len(cleanedLines)-1]) == "" {
			cleanedLines = cleanedLines[:len(cleanedLines)-1]
		}
		if len(cleanedLines) > 0 {
			cleanedLines = append(cleanedLines, "")
		}
		cleanedLines = append(cleanedLines, DotterBlockBeginMarker)
		cleanedLines = append(cleanedLines, contentLines...)
		cleanedLines = append(cleanedLines, DotterBlockEndMarker)
		return cleanedLines, true
	}
}

// GetSupportedShells returns a slice of shells dotter explicitly supports for RC file management.
func GetSupportedShells() []SupportedShell {
	return []SupportedShell{Bash, Zsh, Fish}
}

// AutoDetectShell attempts to determine the current shell from environment variables.
// This is a basic detection and might not be exhaustive.
func AutoDetectShell() SupportedShell {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		// On Windows, SHELL might not be set. ComProc might be cmd.exe.
		// For now, we focus on Unix-like shells.
		return "" // Cannot determine
	}

	shellName := filepath.Base(shellPath)
	switch shellName {
	case "bash":
		return Bash
	case "zsh":
		return Zsh
	case "fish":
		return Fish
	default:
		fmt.Printf("Warning: Unrecognized shell %s, cannot auto-configure rc file.\n", shellName)
		return "" // Or a generic/unknown type
	}
}

var (
	// GetDotterGeneratedDir defines the function to get the dotter generated scripts directory.
	// This is a variable to allow for easier testing.
	GetDotterGeneratedDir = getDotterGeneratedDirInternal
)

// getDotterGeneratedDirInternal returns the directory path where dotter stores its generated scripts.
// e.g. ~/.config/dotter/generated or $XDG_CONFIG_HOME/dotter/generated
func getDotterGeneratedDirInternal() (string, error) {
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get user home directory: %w", err)
		}
		configHome = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(configHome, "dotter", "generated"), nil
}
