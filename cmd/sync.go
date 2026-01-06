package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/yejune/git-subclone/internal/git"
	"github.com/yejune/git-subclone/internal/manifest"
)

var syncRecursive bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Clone or pull all registered subclones",
	Long: `Synchronize all subclones registered in .subclones.yaml.

For each subclone:
  - If not cloned yet: clone it
  - If already exists: pull latest changes

Use --recursive to also sync subclones within subclones.

Examples:
  git-subclone sync
  git-subclone sync --recursive`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().BoolVarP(&syncRecursive, "recursive", "r", false, "Recursively sync subclones within subclones")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	return syncDir(repoRoot, syncRecursive, 0)
}

func syncDir(dir string, recursive bool, depth int) error {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	m, err := manifest.Load(dir)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	if len(m.Subclones) == 0 {
		if depth == 0 {
			fmt.Println("No subclones registered. Use 'git-subclone add' to add one.")
		}
		return nil
	}

	for _, sc := range m.Subclones {
		fullPath := filepath.Join(dir, sc.Path)

		if git.IsRepo(fullPath) {
			// Already cloned, pull
			fmt.Printf("%s↻ Pulling %s...\n", indent, sc.Path)
			if err := git.Pull(fullPath); err != nil {
				fmt.Printf("%s  ✗ Failed to pull: %v\n", indent, err)
				continue
			}
			fmt.Printf("%s  ✓ Updated\n", indent)
		} else {
			// Not cloned, clone it
			fmt.Printf("%s↓ Cloning %s...\n", indent, sc.Path)

			// Create parent directory if needed
			parentDir := filepath.Dir(fullPath)
			if err := os.MkdirAll(parentDir, 0755); err != nil {
				fmt.Printf("%s  ✗ Failed to create directory: %v\n", indent, err)
				continue
			}

			if err := git.Clone(sc.Repo, fullPath, sc.Branch); err != nil {
				fmt.Printf("%s  ✗ Failed to clone: %v\n", indent, err)
				continue
			}

			// Add .git to parent's .gitignore
			if err := git.AddToGitignore(dir, sc.Path); err != nil {
				fmt.Printf("%s  ⚠ Failed to update .gitignore: %v\n", indent, err)
			}

			fmt.Printf("%s  ✓ Cloned\n", indent)
		}

		// Recursive sync
		if recursive {
			subManifest := filepath.Join(fullPath, manifest.FileName)
			if _, err := os.Stat(subManifest); err == nil {
				if err := syncDir(fullPath, recursive, depth+1); err != nil {
					fmt.Printf("%s  ⚠ Recursive sync warning: %v\n", indent, err)
				}
			}
		}
	}

	return nil
}
