package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/hooks"
)

var initUninstall bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install git hooks for automatic subclone sync",
	Long: `Install a post-checkout hook that automatically syncs subclones
when switching branches or cloning the repository.

This ensures subclones are always up-to-date after checkout operations.

Use --uninstall to remove the hooks.

Examples:
  git-subclone init             # Install hooks
  git-subclone init --uninstall # Remove hooks`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initUninstall, "uninstall", false, "Remove git hooks")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	if initUninstall {
		if err := hooks.Uninstall(repoRoot); err != nil {
			return fmt.Errorf("failed to uninstall hooks: %w", err)
		}
		fmt.Println("✓ Git hooks uninstalled")
		return nil
	}

	if hooks.IsInstalled(repoRoot) {
		fmt.Println("Git hooks already installed.")
		return nil
	}

	if err := hooks.Install(repoRoot); err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Println("✓ Git hooks installed")
	fmt.Println("  Subclones will be synced automatically after checkout.")
	return nil
}
