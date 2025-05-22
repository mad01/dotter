package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dotter",
	Short: "dotter is a tool for managing dotfiles and shell configurations.",
	Long: `dotter helps you manage your dotfiles, shell tools, rc files, and helper functions seamlessly.
Inspired by tools like Starship, it uses a TOML configuration file to define how your environment is set up.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Default action when dotter is run without subcommands
		fmt.Println("Use 'dotter --help' for more information.")
	},
}

var dryRun bool // Global variable for the dry-run flag

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() { // This init is for the package, not a specific command
	rootCmd.PersistentFlags().BoolVarP(&dryRun, "dry-run", "n", false, "Show what changes would be made without actually making them")
}
