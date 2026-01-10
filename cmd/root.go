// Package cmd implements the CLI commands for git-multirepo
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is set at build time via -ldflags
	Version = "0.2.11"
	// Root command flags
	rootBranch string
	rootPath   string
)

// Deprecated: Use 'clone' command instead
var rootCmd = &cobra.Command{
	Use:   "git-multirepo [url] [path]",
	Short: "Manage nested git repositories with independent push capability",
	Long: `git-multirepo manages nested git repositories within a parent project.

Each workspace maintains its own .git directory and can push to its own remote,
while the parent project tracks the source files (but not .git).

Commands:
  clone    Clone a new repository
  sync     Clone or pull all repositories
  list     List all registered repositories
  remove   Remove a repository
  status   Show repository status
  pull     Pull repository changes
  reset    Reset repository state
  branch   Manage repository branches
  selfupdate Update git-multirepo to latest version`,
	Version: Version,
	Args:    cobra.MaximumNArgs(2),
	RunE:    runRoot,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.Flags().StringVarP(&rootBranch, "branch", "b", "", "Branch to clone")
	rootCmd.Flags().StringVarP(&rootPath, "path", "p", "", "Destination path")
}

func runRoot(cmd *cobra.Command, args []string) error {
	// No args = show help
	if len(args) == 0 {
		return cmd.Help()
	}

	// Show deprecation warning
	fmt.Println("⚠️  'git multirepo <url>' is deprecated")
	fmt.Println("Use 'git multirepo clone <url>' instead")
	fmt.Println()

	// Delegate to cloneCmd
	// Transfer flags from root to clone
	cloneBranch = rootBranch
	clonePath = rootPath

	return cloneCmd.RunE(cmd, args)
}

// osExit is a variable that can be overridden in tests
var osExit = os.Exit

// Execute runs the root command and exits with code 1 on error
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		osExit(1)
	}
}
