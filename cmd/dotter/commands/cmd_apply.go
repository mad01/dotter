package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mad01/dotter/internal/config"
	"github.com/mad01/dotter/internal/dotfile"
	"github.com/mad01/dotter/internal/hooks"
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
			color.Cyan("\n*** DRY RUN MODE ENABLED ***")
			color.Cyan("No actual changes will be made.")
			color.Cyan("****************************\n")
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintln(os.Stderr, color.RedString("Error loading configuration: %v", err))
			os.Exit(1)
		}

		symlinkAction := dotfile.SymlinkActionBackup // Default action
		if overwriteExisting {
			symlinkAction = dotfile.SymlinkActionOverwrite
			fmt.Println("Symlink action: Overwrite existing files.")
		} else if skipExisting {
			symlinkAction = dotfile.SymlinkActionSkip
			fmt.Println("Symlink action: Skip existing files.")
		} else {
			fmt.Println("Symlink action: Backup existing files.")
		}

		// Execute pre-apply hooks
		if len(cfg.Hooks.PreApply) > 0 {
			preContext := &hooks.HookContext{
				DryRun: dryRun,
			}
			if err := hooks.RunHooks(cfg.Hooks.PreApply, hooks.PreApply, preContext, dryRun); err != nil {
				fmt.Fprintln(os.Stderr, color.RedString("Error executing pre-apply hooks: %v", err))
				os.Exit(1)
			}
		}

		fmt.Println("\nProcessing dotfiles...")
		dotfilesApplied := 0
		dotfilesSkippedOrFailed := 0

		for name, df := range cfg.Dotfiles {
			fmt.Printf("  Applying dotfile: %s (Source: %s, Target: %s)\n", name, df.Source, df.Target)

			// Execute pre-link hooks for this specific dotfile
			if preHooks, exists := cfg.Hooks.PreLink[name]; exists && len(preHooks) > 0 {
				linkContext := &hooks.HookContext{
					DotfileName: name,
					SourcePath:  filepath.Join(cfg.DotfilesRepoPath, df.Source),
					TargetPath:  df.Target,
					DryRun:      dryRun,
				}
				if err := hooks.RunHooks(preHooks, hooks.PreLink, linkContext, dryRun); err != nil {
					fmt.Fprintln(os.Stderr, color.RedString("Error executing pre-link hooks for %s: %v", name, err))
					dotfilesSkippedOrFailed++
					continue
				}
			}

			templateData := make(map[string]interface{})

			var symlinkErr error
			currentSourcePath := filepath.Join(cfg.DotfilesRepoPath, df.Source)
			dotfileToSymlink := df
			repoPathForSymlink := cfg.DotfilesRepoPath

			if df.IsTemplate {
				fmt.Printf("    Processing as template: %s\n", df.Source)
				var processedPath string
				var templateErr error
				if dryRun {
					processedPath, templateErr = dotfile.WriteProcessedTemplateToFile(currentSourcePath, cfg, templateData, true)
					if templateErr == nil && processedPath == "" { // dry run specific path
						processedPath = "/tmp/fake_processed_template_for_dry_run" // ensure it has a value for dry run
					}
				} else {
					processedPath, templateErr = dotfile.WriteProcessedTemplateToFile(currentSourcePath, cfg, templateData, false)
				}

				if templateErr != nil {
					fmt.Fprintln(os.Stderr, color.YellowString("    - Warning: Error processing template for %s: %v", name, templateErr))
					dotfilesSkippedOrFailed++
					continue
				}
				dotfileToSymlink.Source = processedPath
				repoPathForSymlink = "" // Processed template is an absolute path
			}

			symlinkErr = dotfile.CreateSymlink(dotfileToSymlink, repoPathForSymlink, symlinkAction, dryRun)

			// Cleanup for templated files
			if df.IsTemplate && repoPathForSymlink == "" && !dryRun && dotfileToSymlink.Source != "/tmp/fake_processed_template_for_dry_run" {
				// Check if the source is in a temp-like directory before removing
				// This is a basic check; for more robust checks, consider if WriteProcessedTemplateToFile returns if it's a temp file.
				if strings.HasPrefix(dotfileToSymlink.Source, os.TempDir()) || strings.Contains(dotfileToSymlink.Source, "dotter-temp-") {
					if removeErr := os.Remove(dotfileToSymlink.Source); removeErr != nil {
						fmt.Fprintln(os.Stderr, color.YellowString("    - Warning: failed to remove temporary processed file %s: %v", dotfileToSymlink.Source, removeErr))
					} else {
						// fmt.Printf("    Successfully removed temporary processed file %s\n", dotfileToSymlink.Source)
					}
				}
			}

			if symlinkErr != nil {
				fmt.Fprintln(os.Stderr, color.RedString("    - Error applying dotfile %s: %v", name, symlinkErr))
				dotfilesSkippedOrFailed++
			} else {
				if !dryRun { // only count as applied if not dry run
					dotfilesApplied++
				}

				// Execute post-link hooks for this specific dotfile if symlink was created successfully
				if postHooks, exists := cfg.Hooks.PostLink[name]; exists && len(postHooks) > 0 {
					linkContext := &hooks.HookContext{
						DotfileName: name,
						SourcePath:  filepath.Join(cfg.DotfilesRepoPath, df.Source),
						TargetPath:  df.Target,
						DryRun:      dryRun,
					}
					if err := hooks.RunHooks(postHooks, hooks.PostLink, linkContext, dryRun); err != nil {
						fmt.Fprintln(os.Stderr, color.YellowString("Warning: post-link hook for %s failed: %v", name, err))
					}
				}
			}
		}
		if dryRun {
			color.Cyan("  Dotfiles processing (dry run): Inspect messages above for intended actions.")
		} else {
			fmt.Printf("  Dotfiles processed: %s applied, %s skipped/failed.\n", color.GreenString("%d", dotfilesApplied), color.YellowString("%d", dotfilesSkippedOrFailed))
		}

		fmt.Println("\nProcessing shell configurations...")
		currentShell := shell.AutoDetectShell()
		if currentShell == "" {
			fmt.Fprintln(os.Stderr, color.YellowString("Could not auto-detect current shell. Skipping shell configuration."))
		} else {
			fmt.Printf("  Detected shell: %s\n", currentShell)
			var aliasFile, funcFile string
			var genErr error

			aliasFile, funcFile, genErr = shell.GenerateShellConfigs(cfg, currentShell, dryRun)

			if genErr != nil {
				fmt.Fprintln(os.Stderr, color.RedString("  Error generating shell configs for %s: %v", currentShell, genErr))
			} else {
				linesToSource := []string{}
				if aliasFile != "" && (len(cfg.Shell.Aliases) > 0 || (dryRun && aliasFile != "")) {
					linesToSource = append(linesToSource, fmt.Sprintf("source %s", aliasFile))
				}
				if funcFile != "" && (len(cfg.Shell.Functions) > 0 || (dryRun && funcFile != "")) {
					linesToSource = append(linesToSource, fmt.Sprintf("source %s", funcFile))
				}

				if len(linesToSource) > 0 {
					fmt.Printf("  Injecting source lines into %s rc file...\n", currentShell)
					if err := shell.InjectSourceLines(currentShell, linesToSource, dryRun); err != nil {
						fmt.Fprintln(os.Stderr, color.RedString("  Error injecting source lines into %s rc file: %v", currentShell, err))
					}
				} else {
					fmt.Println("  No shell aliases or functions configured to source.")
				}
			}
		}

		// Tool management in apply (TODO based on config)
		if len(cfg.Tools) > 0 {
			fmt.Println("\nChecking tool configurations (installation not performed by apply):")
			for _, t := range cfg.Tools {
				var statusColor func(format string, a ...interface{}) string
				status := "Not installed"
				if tool.CheckStatus(t.CheckCommand) {
					status = "Installed"
					statusColor = color.GreenString
				} else {
					statusColor = color.YellowString
				}
				fmt.Printf("  - Tool '%s': %s. Install hint: %s\n", t.Name, statusColor(status), t.InstallHint)
				// TODO: Process tool.ConfigFiles if any, similar to main dotfiles (symlinking, templating)
				// This would need to respect dryRun as well.
			}
		}

		// Execute post-apply hooks
		if len(cfg.Hooks.PostApply) > 0 {
			postContext := &hooks.HookContext{
				DryRun: dryRun,
			}
			if err := hooks.RunHooks(cfg.Hooks.PostApply, hooks.PostApply, postContext, dryRun); err != nil {
				fmt.Fprintln(os.Stderr, color.YellowString("Warning: post-apply hooks failed: %v", err))
			}
		}

		fmt.Println("") // Add a newline for spacing
		if dryRun {
			color.Cyan("DRY RUN: Dotter apply finished. No actual changes were made.")
		} else {
			color.Green("Dotter apply complete.")
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
