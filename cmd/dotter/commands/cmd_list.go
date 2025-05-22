package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/mad01/dotter/internal/config"
	// "github.com/mad01/dotter/internal/dotfile" // For symlink status check - removing to clear linter
	"github.com/mad01/dotter/internal/tool" // Added for tool status check
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed dotfiles, tools, and shell configurations",
	Long:  `Displays a list of all items currently managed by dotter, along with their status.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Listing managed items...")

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading configuration: %v\nConsider running 'dotter init' if you haven't already.", err)
		}

		fmt.Println("\nManaged Dotfiles:")
		if len(cfg.Dotfiles) == 0 {
			fmt.Println("  No dotfiles configured.")
		} else {
			for name, df := range cfg.Dotfiles {
				status := "Unknown"
				absoluteTarget, expandErr := config.ExpandPath(df.Target)
				if expandErr != nil {
					status = fmt.Sprintf("Error expanding target path: %v", expandErr)
				} else {
					// Check symlink status
					targetInfo, statErr := os.Lstat(absoluteTarget)
					if os.IsNotExist(statErr) {
						status = "Not linked (target does not exist)"
					} else if statErr != nil {
						status = fmt.Sprintf("Error checking target: %v", statErr)
					} else {
						if targetInfo.Mode()&os.ModeSymlink != 0 {
							linkDest, readlinkErr := os.Readlink(absoluteTarget)
							if readlinkErr != nil {
								status = "Symlink (error reading destination)"
							} else {
								absoluteSource, _ := config.ExpandPath(filepath.Join(cfg.DotfilesRepoPath, df.Source))
								if df.IsTemplate {
									// For templates, the symlink points to a temporary processed file.
									// We can check if it points to *a* file, and if it's in our temp dir structure.
									if strings.Contains(linkDest, filepath.Join(os.TempDir(), "dotter", "processed_templates")) {
										status = fmt.Sprintf("Correctly linked (templated) to: %s", linkDest)
									} else {
										status = fmt.Sprintf("Symlinked (templated) to an unexpected path: %s", linkDest)
									}
								} else if linkDest == absoluteSource {
									status = fmt.Sprintf("Correctly linked to: %s", linkDest)
								} else {
									status = fmt.Sprintf("Symlinked to WRONG source: %s (expected %s)", linkDest, absoluteSource)
								}
							}
						} else {
							status = "Exists but is NOT a symlink"
						}
					}
				}
				templateMarker := ""
				if df.IsTemplate {
					templateMarker = " (template)"
				}
				fmt.Printf("  - %s%s:\n      Source: %s\n      Target: %s\n      Status: %s\n", name, templateMarker, df.Source, df.Target, status)
			}
		}

		fmt.Println("\nConfigured Tools:")
		if len(cfg.Tools) == 0 {
			fmt.Println("  No tools configured.")
		} else {
			for _, t := range cfg.Tools {
				status := "Not installed"
				if tool.CheckStatus(t.CheckCommand) {
					status = "Installed"
				}
				fmt.Printf("  - %s (Check: '%s', Hint: '%s'): %s\n", t.Name, t.CheckCommand, t.InstallHint, status)
			}
		}

		fmt.Println("\nDefined Shell Aliases:")
		if len(cfg.Shell.Aliases) == 0 {
			fmt.Println("  No shell aliases defined.")
		} else {
			for name, command := range cfg.Shell.Aliases {
				fmt.Printf("  - %s: %s\n", name, command)
			}
		}

		fmt.Println("\nDefined Shell Functions:")
		if len(cfg.Shell.Functions) == 0 {
			fmt.Println("  No shell functions defined.")
		} else {
			for name := range cfg.Shell.Functions {
				fmt.Printf("  - %s\n", name)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(listCmd)
}
