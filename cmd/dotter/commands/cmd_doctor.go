package commands

import (
	"fmt"
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
		color.Cyan("🩺 Running dotter doctor checks...")
		healthy := true

		// 1. Verify config.toml readability and validity
		fmt.Print(color.New(color.FgWhite, color.Bold).Sprint("\nChecking configuration file... "))
		cfg, err := config.LoadConfig() // This already does validation
		if err != nil {
			color.Red("Error: %v", err)
			healthy = false
		} else {
			color.Green("OK")
		}

		if cfg == nil { // If config failed to load, cannot proceed with other checks
			fmt.Fprintln(os.Stderr, color.RedString("Cannot perform further checks due to configuration load failure."))
			os.Exit(1)
		}

		// 2. Check for broken symlinks for managed dotfiles
		fmt.Println(color.New(color.FgWhite, color.Bold).Sprint("\nChecking managed dotfile symlinks:"))
		foundIssuesInSymlinks := false
		if len(cfg.Dotfiles) == 0 {
			color.Yellow("  No dotfiles configured to check.")
		} else {
			for name, df := range cfg.Dotfiles {
				templateMarker := ""
				if df.IsTemplate {
					templateMarker = color.CyanString(" (template)")
				}
				fmt.Printf("  - %s%s (Target: %s): ", color.New(color.Bold).Sprint(name), templateMarker, df.Target)
				absoluteTarget, expandErr := config.ExpandPath(df.Target)
				if expandErr != nil {
					color.Red("Error expanding target path: %v", expandErr)
					healthy = false
					foundIssuesInSymlinks = true
					continue
				}

				targetInfo, statErr := os.Lstat(absoluteTarget)
				if os.IsNotExist(statErr) {
					color.Yellow("Not linked (target does not exist)")
				} else if statErr != nil {
					color.Red("Error checking target: %v", statErr)
					healthy = false
					foundIssuesInSymlinks = true
				} else {
					if targetInfo.Mode()&os.ModeSymlink == 0 {
						color.Yellow("Exists but is NOT a symlink")
						foundIssuesInSymlinks = true // This is an issue if we expect a symlink
					} else {
						linkDest, readlinkErr := os.Readlink(absoluteTarget)
						if readlinkErr != nil {
							color.Red("Symlink (error reading destination: %v)", readlinkErr)
							healthy = false
							foundIssuesInSymlinks = true
						} else {
							var actualSourcePath string
							if df.IsTemplate {
								actualSourcePath = linkDest // For templates, linkDest is the absolute path to the processed file.
							} else {
								expandedRepoSource, _ := config.ExpandPath(filepath.Join(cfg.DotfilesRepoPath, df.Source))
								actualSourcePath = expandedRepoSource
								if linkDest != actualSourcePath {
									color.Yellow("WARN: Symlink points to '%s', but config expects '%s'. Checking existence of actual '%s'... ", linkDest, actualSourcePath, linkDest)
									actualSourcePath = linkDest // For broken check, use what it *actually* points to
								}
							}

							if _, err := os.Stat(actualSourcePath); os.IsNotExist(err) {
								color.Red("BROKEN SYMLINK (source '%s' does not exist)", actualSourcePath)
								healthy = false
								foundIssuesInSymlinks = true
							} else if err != nil {
								color.Red("Error stating symlink source '%s': %v", actualSourcePath, err)
								healthy = false
								foundIssuesInSymlinks = true
							} else {
								color.Green("OK (links to '%s')", linkDest)
							}
						}
					}
				}
			}
			if !foundIssuesInSymlinks && len(cfg.Dotfiles) > 0 {
				color.Green("  All checked symlinks appear valid or target does not exist yet.")
			}
		}

		// Add tool status checks to doctor command
		fmt.Println(color.New(color.FgWhite, color.Bold).Sprint("\nChecking configured tools:"))
		if len(cfg.Tools) == 0 {
			color.Yellow("  No tools configured to check.")
		} else {
			for _, t := range cfg.Tools {
				fmt.Printf("  - Tool '%s' (check: '%s'): ", color.New(color.Bold).Sprint(t.Name), t.CheckCommand)
				if tool.CheckStatus(t.CheckCommand) {
					color.Green("Installed")
				} else {
					color.Yellow("Not Installed (or check failed)")
					fmt.Printf("      Install hint: %s\n", t.InstallHint)
					// Not marking as unhealthy, as this is informational in doctor
				}
			}
		}

		// 3. Verify if rc file snippets are correctly sourced
		fmt.Println(color.New(color.FgWhite, color.Bold).Sprint("\nChecking RC file sourcing:"))
		shellsToTest := shell.GetSupportedShells() // Use the function to get all supported shells
		foundRCIssues := false
		for _, s := range shellsToTest {
			fmt.Printf("  Shell '%s': ", color.New(color.Bold).Sprint(s))
			rcPath, err := shell.GetRCFilePath(s)
			if err != nil {
				color.Yellow("Could not get RC file path: %v", err)
				// Not necessarily unhealthy for this shell if path not found for some reason, but worth noting
				continue
			}
			if _, err := os.Stat(rcPath); os.IsNotExist(err) {
				color.Yellow("RC file '%s' does not exist. Dotter block not present.", rcPath)
				continue // Not an error for doctor if RC file itself is missing
			}
			content, err := os.ReadFile(rcPath)
			if err != nil {
				color.Red("Could not read RC file '%s': %v", rcPath, err)
				healthy = false
				foundRCIssues = true
				continue
			}
			if strings.Contains(string(content), shell.DotterBlockBeginMarker) && strings.Contains(string(content), shell.DotterBlockEndMarker) {
				color.Green("Dotter managed block found.")
				blockStartIndex := strings.Index(string(content), shell.DotterBlockBeginMarker)
				blockEndIndex := strings.Index(string(content), shell.DotterBlockEndMarker)
				// Ensured blockStartIndex < blockEndIndex in previous implementation. It is implicitly handled by Index returning -1 if not found.
				// And the outer if checks both exist.
				blockContent := string(content)[blockStartIndex+len(shell.DotterBlockBeginMarker) : blockEndIndex]
				blockLines := strings.Split(blockContent, "\n")
				foundMissingSourceFiles := false
				sourcedFilesExpected := (len(cfg.Shell.Aliases) > 0 || len(cfg.Shell.Functions) > 0)
				sourcedFilesFoundInBlock := 0

				for _, line := range blockLines {
					trimmedLine := strings.TrimSpace(line)
					var sourcedFile string
					if strings.HasPrefix(trimmedLine, "source ") {
						sourcedFile = strings.TrimPrefix(trimmedLine, "source ")
					} else if strings.HasPrefix(trimmedLine, ". ") { // POSIX sh alternate for source
						sourcedFile = strings.TrimPrefix(trimmedLine, ". ")
					}

					if sourcedFile != "" {
						sourcedFilesFoundInBlock++
						expandedSourcedFile, expErr := config.ExpandPath(sourcedFile)
						if expErr != nil {
							color.Yellow("    -> Could not expand path for sourced file '%s': %v", sourcedFile, expErr)
							foundMissingSourceFiles = true
							healthy = false // Path expansion failure is an issue
							continue
						}
						if _, statErr := os.Stat(expandedSourcedFile); os.IsNotExist(statErr) {
							color.Red("    -> Sourced file '%s' (expanded: '%s') does NOT exist.", sourcedFile, expandedSourcedFile)
							foundMissingSourceFiles = true
							healthy = false
						} else if statErr != nil {
							color.Red("    -> Error checking sourced file '%s': %v", expandedSourcedFile, statErr)
							foundMissingSourceFiles = true
							healthy = false
						} else {
							color.Green("    -> Sourced file '%s' exists.", expandedSourcedFile)
						}
					}
				}
				if !foundMissingSourceFiles && sourcedFilesFoundInBlock > 0 {
					color.Green("    All detected source commands in block point to existing files.")
				} else if sourcedFilesExpected && sourcedFilesFoundInBlock == 0 {
					color.Yellow("    Dotter block found, but no source commands for generated files detected, yet shell items are configured.")
					// healthy = false // Could make this unhealthy
					foundRCIssues = true
				} else if !sourcedFilesExpected && sourcedFilesFoundInBlock == 0 {
					color.Green("    Dotter block found, and no shell items are configured (no source commands expected).")
				}
				if foundMissingSourceFiles {
					foundRCIssues = true
				}

			} else {
				color.Yellow("Dotter managed block NOT found.")
				if len(cfg.Shell.Aliases) > 0 || len(cfg.Shell.Functions) > 0 {
					color.Yellow("    Warning: Aliases/functions are configured but dotter block is missing in %s.", rcPath)
					healthy = false // If shell items are configured, missing block is an issue
					foundRCIssues = true
				}
			}
		}
		if !foundRCIssues && len(shellsToTest) > 0 {
			// This message might be too broad if some shells were skipped due to no RC file
			// color.Green("  RC file checks passed for tested shells.")
		}

		fmt.Println("\n" + color.CyanString("Doctor checks complete."))
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
