package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mad01/dotter/internal/config"
	"github.com/mad01/dotter/internal/shell"
	"github.com/mad01/dotter/internal/tool"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the health of the dotter setup",
	Long:  `Performs a series of checks to ensure dotter is configured correctly and all managed items are in a healthy state.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running dotter doctor checks...")
		healthy := true

		// 1. Verify config.toml readability and validity
		fmt.Print("Checking configuration file... ")
		cfg, err := config.LoadConfig() // This already does validation
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			healthy = false
		} else {
			fmt.Println("OK")
		}

		if cfg == nil { // If config failed to load, cannot proceed with other checks
			log.Println("Cannot perform further checks due to configuration load failure.")
			os.Exit(1)
		}

		// 2. Check for broken symlinks for managed dotfiles
		fmt.Println("\nChecking managed dotfile symlinks:")
		foundBrokenSymlinks := false
		for name, df := range cfg.Dotfiles {
			fmt.Printf("  - %s (Target: %s): ", name, df.Target)
			absoluteTarget, expandErr := config.ExpandPath(df.Target)
			if expandErr != nil {
				fmt.Printf("Error expanding target path: %v\n", expandErr)
				healthy = false
				foundBrokenSymlinks = true
				continue
			}

			targetInfo, statErr := os.Lstat(absoluteTarget)
			if os.IsNotExist(statErr) {
				fmt.Println("Not linked (target does not exist).")
				// This isn't necessarily "broken" for doctor, just not applied yet.
				// But if it *was* a symlink and now missing, that's an issue.
				// For now, we just note it.
			} else if statErr != nil {
				fmt.Printf("Error checking target: %v\n", statErr)
				healthy = false
				foundBrokenSymlinks = true
			} else {
				if targetInfo.Mode()&os.ModeSymlink == 0 {
					fmt.Println("Exists but is NOT a symlink.")
				} else {
					linkDest, readlinkErr := os.Readlink(absoluteTarget)
					if readlinkErr != nil {
						fmt.Printf("Symlink (error reading destination: %v)\n", readlinkErr)
						healthy = false
						foundBrokenSymlinks = true
					} else {
						// Check if the symlink destination (source file) exists
						var actualSourcePath string
						if df.IsTemplate {
							// For templates, the linkDest is an absolute path to the processed file.
							actualSourcePath = linkDest
						} else {
							// For non-templates, linkDest should match the expanded source from config
							expandedSource, _ := config.ExpandPath(filepath.Join(cfg.DotfilesRepoPath, df.Source))
							actualSourcePath = expandedSource
							// We also directly check linkDest if it's different, as it *is* what it points to
							if linkDest != actualSourcePath && !df.IsTemplate { // df.IsTemplate condition is redundant here but for clarity
								fmt.Printf("Symlink points to '%s', but config expects '%s'. Checking existence of '%s'... ", linkDest, actualSourcePath, linkDest)
								actualSourcePath = linkDest // Check what it *actually* points to for broken status
							}
						}

						if _, err := os.Stat(actualSourcePath); os.IsNotExist(err) {
							fmt.Printf("BROKEN SYMLINK (source '%s' does not exist)\n", actualSourcePath)
							healthy = false
							foundBrokenSymlinks = true
						} else if err != nil {
							fmt.Printf("Error stating symlink source '%s': %v\n", actualSourcePath, err)
							healthy = false
							foundBrokenSymlinks = true
						} else {
							fmt.Printf("OK (links to '%s')\n", linkDest)
						}
					}
				}
			}
		}
		if !foundBrokenSymlinks {
			fmt.Println("  All checked symlinks appear valid or target does not exist yet.")
		}

		// Add tool status checks to doctor command
		fmt.Println("\nChecking configured tools:")
		if len(cfg.Tools) == 0 {
			fmt.Println("  No tools configured to check.")
		} else {
			for _, t := range cfg.Tools {
				fmt.Printf("  - Tool '%s' (check: '%s'): ", t.Name, t.CheckCommand)
				if tool.CheckStatus(t.CheckCommand) {
					color.Green("Installed")
				} else {
					color.Yellow("Not Installed (or check failed)")
					fmt.Printf("    Install hint: %s\n", t.InstallHint)
					// Consider if this should make healthy=false. For now, it's a warning/info.
				}
			}
		}

		// 3. Verify if rc file snippets are correctly sourced
		fmt.Println("\nChecking RC file sourcing:")
		// This is a more complex check. For now, we'll just check if the dotter block exists.
		// A deeper check would involve seeing if the sourced files exist and are in PATH etc.
		shellsToTest := []shell.SupportedShell{shell.Bash, shell.Zsh, shell.Fish} // Could also use AutoDetectShell or get from config
		for _, s := range shellsToTest {
			fmt.Printf("  Shell '%s': ", s)
			rcPath, err := shell.GetRCFilePath(s)
			if err != nil {
				fmt.Printf("Could not get RC file path: %v\n", err)
				continue
			}
			if _, err := os.Stat(rcPath); os.IsNotExist(err) {
				fmt.Printf("RC file '%s' does not exist. Dotter block not present.\n", rcPath)
				continue // Not necessarily an error for doctor, just not set up
			}
			content, err := os.ReadFile(rcPath)
			if err != nil {
				fmt.Printf("Could not read RC file '%s': %v\n", rcPath, err)
				healthy = false
				continue
			}
			if strings.Contains(string(content), shell.DotterBlockBeginMarker) && strings.Contains(string(content), shell.DotterBlockEndMarker) {
				fmt.Println("Dotter managed block found.")
				// TODO: Check if the source lines within the block point to existing, valid files.
			} else {
				fmt.Println("Dotter managed block NOT found.")
				// This might be okay if no shell aliases/functions are defined.
				if len(cfg.Shell.Aliases) > 0 || len(cfg.Shell.Functions) > 0 {
					fmt.Printf("    Warning: Aliases/functions are configured but dotter block is missing in %s.\n", rcPath)
					// healthy = false // Decide if this makes the setup unhealthy
				}
			}
		}

		fmt.Println("\nDoctor checks complete.")
		if healthy {
			color.Green("Dotter setup appears to be healthy! ✅")
		} else {
			color.Red("Dotter setup has some issues. ❌ Please review the messages above.")
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
