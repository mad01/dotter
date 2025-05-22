package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mad01/dotter/internal/config"
	"github.com/mad01/dotter/internal/dotfile"
	"github.com/mad01/dotter/internal/shell"
	"github.com/mad01/dotter/internal/tool"
	"github.com/spf13/cobra"
)

var (
	overwriteExisting bool
	skipExisting      bool
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply dotter configurations",
	Long:  `Applies the configurations defined in your dotter config file. This includes symlinking dotfiles, setting up shell environments, etc.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Applying dotter configurations...")

		if dryRun {
			fmt.Println("\n*** DRY RUN MODE ENABLED ***")
			fmt.Println("No actual changes will be made.")
			fmt.Println("****************************\n")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Error loading configuration: %v\n", err)
		}

		symlinkAction := dotfile.SymlinkActionBackup // Default action
		if overwriteExisting {
			symlinkAction = dotfile.SymlinkActionOverwrite
		} else if skipExisting {
			symlinkAction = dotfile.SymlinkActionSkip
		}

		fmt.Println("Processing dotfiles...")
		for name, df := range cfg.Dotfiles {
			fmt.Printf("  Applying dotfile: %s (Source: %s, Target: %s)\n", name, df.Source, df.Target)

			templateData := make(map[string]interface{})

			var symlinkErr error
			currentSourcePath := filepath.Join(cfg.DotfilesRepoPath, df.Source)
			dotfileToSymlink := df
			repoPathForSymlink := cfg.DotfilesRepoPath

			if df.IsTemplate {
				fmt.Printf("    Processing as template: %s\n", df.Source)
				if dryRun {
					// WriteProcessedTemplateToFile will now print its own dry run message
					processedPath, templateErr := dotfile.WriteProcessedTemplateToFile(currentSourcePath, cfg, templateData, true)
					if templateErr != nil { // Still check for processing errors
						log.Printf("    Error processing template (dry run) for %s: %v\n", name, templateErr)
						continue
					}
					dotfileToSymlink.Source = processedPath // Use the fake path for symlink dry run
					repoPathForSymlink = ""
				} else {
					processedPath, templateErr := dotfile.WriteProcessedTemplateToFile(currentSourcePath, cfg, templateData, false)
					if templateErr != nil {
						log.Printf("    Error processing template for %s: %v\n", name, templateErr)
						continue
					}
					dotfileToSymlink.Source = processedPath
					repoPathForSymlink = ""
				}
			}

			// Pass dryRun to CreateSymlink if it's adapted to handle it
			// For now, CreateSymlink doesn't know about dryRun, so it will attempt FS operations.
			// We gate the actual call here for dryRun.
			if dryRun {
				// CreateSymlink will now print its own dry run messages if called with dryRun=true
				symlinkErr = dotfile.CreateSymlink(dotfileToSymlink, repoPathForSymlink, symlinkAction, true)
			} else {
				symlinkErr = dotfile.CreateSymlink(dotfileToSymlink, repoPathForSymlink, symlinkAction, false)
			}

			// Cleanup for templated files, outside dry run (dryRun is false here implicitly by the outer if)
			if df.IsTemplate && repoPathForSymlink == "" && !dryRun && dotfileToSymlink.Source != "/tmp/fake_processed_template_for_dry_run" {
				if removeErr := os.Remove(dotfileToSymlink.Source); removeErr != nil {
					log.Printf("    Warning: failed to remove temporary processed file %s: %v\n", dotfileToSymlink.Source, removeErr)
				}
			}

			if symlinkErr != nil {
				log.Printf("    Error applying dotfile %s: %v\n", name, symlinkErr)
			}
		}

		fmt.Println("Processing shell configurations...")
		currentShell := shell.AutoDetectShell()
		if currentShell == "" {
			log.Println("Could not auto-detect current shell. Skipping shell configuration.")
		} else {
			var aliasFile, funcFile string
			var genErr error
			if dryRun {
				fmt.Printf("  [DRY RUN] Would generate shell config files for %s.\n", currentShell)
				// Create dummy paths for sourcing logic in dry run
				genDir, _ := shell.GetDotterGeneratedDir()
				aliasFile = filepath.Join(genDir, shell.GeneratedAliasesFilename)
				funcFile = filepath.Join(genDir, shell.GeneratedFunctionsFilename)
				// GenerateShellConfigs will now print its own dry run messages
				aliasFile, funcFile, genErr = shell.GenerateShellConfigs(cfg, currentShell, true)
				// No actual files created, but paths are needed for subsequent dry run of InjectSourceLines
			} else {
				aliasFile, funcFile, genErr = shell.GenerateShellConfigs(cfg, currentShell, false)
			}

			if genErr != nil {
				log.Printf("Error generating shell configs for %s: %v\n", currentShell, genErr)
			} else {
				linesToSource := []string{}
				if aliasFile != "" && (len(cfg.Shell.Aliases) > 0 || dryRun) { // check len for non-dry run
					linesToSource = append(linesToSource, fmt.Sprintf("source %s", aliasFile))
				}
				if funcFile != "" && (len(cfg.Shell.Functions) > 0 || dryRun) {
					linesToSource = append(linesToSource, fmt.Sprintf("source %s", funcFile))
				}

				if len(linesToSource) > 0 {
					if dryRun {
						// InjectSourceLines will now print its own dry run messages
						if err := shell.InjectSourceLines(currentShell, linesToSource, true); err != nil {
							// Log error even in dry run if the dry run logic itself fails
							log.Printf("Error during dry run of injecting source lines for %s: %v\n", currentShell, err)
						}
					} else {
						if err := shell.InjectSourceLines(currentShell, linesToSource, false); err != nil {
							log.Printf("Error injecting source lines into %s rc file: %v\n", currentShell, err)
						}
					}
				}
			}
		}

		// Tool management in apply (TODO based on config)
		if len(cfg.Tools) > 0 {
			fmt.Println("\nChecking tool configurations (installation not performed by apply):")
			for _, t := range cfg.Tools {
				status := "Not installed"
				if tool.CheckStatus(t.CheckCommand) {
					status = "Installed"
				}
				fmt.Printf("  - Tool '%s': %s. Install hint: %s\n", t.Name, status, t.InstallHint)
				// TODO: Process tool.ConfigFiles if any, similar to main dotfiles (symlinking, templating)
				// This would need to respect dryRun as well.
			}
		}

		fmt.Println("\nDotter apply complete.")
		if dryRun {
			fmt.Println("DRY RUN: No changes were made.")
		}
	},
}

func init() {
	rootCmd.AddCommand(applyCmd)
	applyCmd.Flags().BoolVar(&overwriteExisting, "overwrite", false, "Overwrite existing files at target locations for symlinks")
	applyCmd.Flags().BoolVar(&skipExisting, "skip", false, "Skip symlinking if target file already exists")
	// Note: --overwrite and --skip are mutually exclusive in behavior.
	// Cobra doesn't enforce this directly, would need custom validation or be handled by logic choosing one if both true.
	// Current logic: if overwrite is true, it takes precedence over skip.
}
