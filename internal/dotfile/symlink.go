package dotfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mad01/dotter/internal/config"
)

// SymlinkAction defines the action to take if a file already exists at the target location.
type SymlinkAction int

const (
	// SymlinkActionBackup backups the existing file.
	SymlinkActionBackup SymlinkAction = iota
	// SymlinkActionOverwrite overwrites the existing file.
	SymlinkActionOverwrite
	// SymlinkActionSkip skips symlinking if the target exists.
	SymlinkActionSkip
)

// CreateSymlink creates a symbolic link from source to target.
// It handles path expansion for both source (relative to repoPath) and target.
// If repoPath is empty, dotfileCfg.Source is assumed to be an absolute path already.
// It also manages existing files at the target location based on the specified action.
// If dryRun is true, it will only print the actions it would take.
func CreateSymlink(dotfileCfg config.Dotfile, dotfilesRepoPath string, action SymlinkAction, dryRun bool) error {
	var absoluteSource string
	var err error

	if dotfilesRepoPath == "" { // Source is already absolute (e.g., a processed template file)
		absoluteSource = dotfileCfg.Source
	} else {
		absoluteSource, err = config.ExpandPath(filepath.Join(dotfilesRepoPath, dotfileCfg.Source))
		if err != nil {
			return fmt.Errorf("failed to expand source path '%s' relative to '%s': %w", dotfileCfg.Source, dotfilesRepoPath, err)
		}
	}

	absoluteTarget, err := config.ExpandPath(dotfileCfg.Target)
	if err != nil {
		return fmt.Errorf("failed to expand target path '%s': %w", dotfileCfg.Target, err)
	}

	// Ensure the source file exists (unless it's a dry run for a templated file that wouldn't exist yet)
	if !dryRun || (dryRun && dotfilesRepoPath != "") { // if it's a template dry run, source might be faked
		if _, err := os.Stat(absoluteSource); os.IsNotExist(err) {
			return fmt.Errorf("source file '%s' (expanded: '%s') does not exist", dotfileCfg.Source, absoluteSource)
		}
	}

	targetInfo, err := os.Lstat(absoluteTarget)
	if err == nil {
		switch action {
		case SymlinkActionBackup:
			backupPath := absoluteTarget + ".bak"
			fmt.Printf("Target '%s' exists. ", absoluteTarget)
			if dryRun {
				fmt.Printf("[DRY RUN] Would back up to '%s'\n", backupPath)
			} else {
				fmt.Printf("Backing up to '%s'\n", backupPath)
				if err := os.Rename(absoluteTarget, backupPath); err != nil {
					return fmt.Errorf("failed to backup '%s' to '%s': %w", absoluteTarget, backupPath, err)
				}
			}
		case SymlinkActionOverwrite:
			fmt.Printf("Target '%s' exists. ", absoluteTarget)
			if dryRun {
				fmt.Printf("[DRY RUN] Would overwrite.\n")
			} else {
				fmt.Printf("Overwriting.\n")
				if err := os.Remove(absoluteTarget); err != nil {
					return fmt.Errorf("failed to remove existing target '%s' for overwrite: %w", absoluteTarget, err)
				}
			}
		case SymlinkActionSkip:
			if targetInfo.Mode()&os.ModeSymlink != 0 {
				linkTarget, readErr := os.Readlink(absoluteTarget)
				if readErr == nil && linkTarget == absoluteSource {
					fmt.Printf("Target '%s' is already correctly symlinked to '%s'. Skipping.\n", absoluteTarget, absoluteSource)
					return nil
				}
			}
			fmt.Printf("Target '%s' exists and is not the correct symlink (or not a symlink). Skipping as per action.\n", absoluteTarget)
			return nil
		default:
			return fmt.Errorf("unknown action for existing target '%s'", absoluteTarget)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat target '%s': %w", absoluteTarget, err)
	}

	targetDir := filepath.Dir(absoluteTarget)
	if dryRun {
		fmt.Printf("[DRY RUN] Would ensure target directory '%s' exists.\n", targetDir)
		fmt.Printf("[DRY RUN] Would create symlink: '%s' -> '%s'\n", absoluteTarget, absoluteSource)
	} else {
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory '%s': %w", targetDir, err)
		}
		fmt.Printf("Creating symlink: '%s' -> '%s'\n", absoluteTarget, absoluteSource)
		if err := os.Symlink(absoluteSource, absoluteTarget); err != nil {
			return fmt.Errorf("failed to create symlink from '%s' to '%s': %w", absoluteSource, absoluteTarget, err)
		}
	}

	return nil
}
