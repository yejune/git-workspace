package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of all subclones",
	Long: `Display detailed status information for all registered subclones.

Shows clone status, current branch, uncommitted changes, and sync status.

Examples:
  git-subclone status`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Subclones) == 0 {
		fmt.Println("No subclones registered.")
		return nil
	}

	fmt.Printf("Subclones (%d):\n\n", len(m.Subclones))

	for _, sc := range m.Subclones {
		fullPath := filepath.Join(repoRoot, sc.Path)

		fmt.Printf("  %s\n", sc.Path)
		fmt.Printf("    Remote: %s\n", sc.Repo)

		if sc.Branch != "" {
			fmt.Printf("    Configured branch: %s\n", sc.Branch)
		}

		if !git.IsRepo(fullPath) {
			fmt.Printf("    Status: ○ not cloned\n")
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			fmt.Printf("    Current branch: unknown\n")
		} else {
			fmt.Printf("    Current branch: %s\n", branch)
		}

		// Check for changes
		hasChanges, err := git.HasChanges(fullPath)
		if err != nil {
			fmt.Printf("    Status: ✗ error checking status\n")
		} else if hasChanges {
			fmt.Printf("    Status: ● has uncommitted changes\n")
		} else {
			fmt.Printf("    Status: ✓ clean\n")
		}

		fmt.Println()
	}

	return nil
}
