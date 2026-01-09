package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yejune/git-workspace/internal/git"
	"github.com/yejune/git-workspace/internal/i18n"
	"github.com/yejune/git-workspace/internal/manifest"
)

var statusCmd = &cobra.Command{
	Use:   "status [path]",
	Short: "Show detailed status of workspaces",
	Long: `Display comprehensive status information for each workspace:

Examples:
  git workspace status              # Show status for all workspaces
  git workspace status apps/admin   # Show status for specific workspace

For each workspace, shows:
  1. Local Status (modified, untracked, staged files)
  2. Remote Status (commits behind/ahead)
  3. How to resolve (step-by-step commands)`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	// Define color printers
	// Use Fprintf to always print to the correct stdout
	var (
		printCyan   = func(format string, a ...interface{}) { color.New(color.FgCyan, color.Bold).Fprintf(os.Stdout, format, a...) }
		printBlue   = func(format string, a ...interface{}) { color.New(color.FgBlue, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGreen  = func(format string, a ...interface{}) { color.New(color.FgGreen).Fprintf(os.Stdout, format, a...) }
		printYellow = func(format string, a ...interface{}) { color.New(color.FgYellow).Fprintf(os.Stdout, format, a...) }
		printRed    = func(format string, a ...interface{}) { color.New(color.FgRed, color.Bold).Fprintf(os.Stdout, format, a...) }
		printGray   = func(format string, a ...interface{}) { color.New(color.Faint).Fprintf(os.Stdout, format, a...) }
	)

	repoRoot, err := git.GetRepoRoot()
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	m, err := manifest.Load(repoRoot)
	if err != nil {
		return fmt.Errorf("failed to load manifest: %w", err)
	}

	// Set language from manifest
	i18n.SetLanguage(m.GetLanguage())

	if len(m.Workspaces) == 0 {
		fmt.Println(i18n.T("no_subs_registered"))
		return nil
	}

	// Filter workspaces if path argument provided
	var workspacesToProcess []manifest.WorkspaceEntry
	if len(args) > 0 {
		targetPath := args[0]
		found := false
		for _, workspace := range m.Workspaces {
			if workspace.Path == targetPath {
				workspacesToProcess = []manifest.WorkspaceEntry{workspace}
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf(i18n.T("sub_not_found", targetPath))
		}
	} else {
		workspacesToProcess = m.Workspaces
	}

	for idx, ws := range workspacesToProcess {
		if idx > 0 {
			// Add separator between workspaces
			printGray("%s\n", strings.Repeat("─", 80))
			fmt.Println()
		}

		fullPath := filepath.Join(repoRoot, ws.Path)

		// Workspace header
		printCyan("%s", ws.Path)

		if !git.IsRepo(fullPath) {
			printRed(" %s\n", i18n.T("not_cloned"))
			fmt.Println()
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			printGray("    git workspace sync\n")
			fmt.Println()
			continue
		}

		// Get current branch
		branch, err := git.GetCurrentBranch(fullPath)
		if err != nil {
			branch = "unknown"
		}
		printGray(" (%s)\n", branch)
		fmt.Println()

		// Section 1: Local Status
		printBlue("  %s\n", i18n.T("local_status"))

		modifiedFiles, _ := git.GetModifiedFiles(fullPath)
		untrackedFiles, _ := git.GetUntrackedFiles(fullPath)
		stagedFiles, _ := git.GetStagedFiles(fullPath)

		hasLocalChanges := false

		if len(modifiedFiles) > 0 {
			hasLocalChanges = true
			printYellow("    %s\n", i18n.T("files_modified", len(modifiedFiles)))
			for _, file := range modifiedFiles {
				printGray("      - %s\n", file)
			}
		}

		if len(untrackedFiles) > 0 {
			hasLocalChanges = true
			printYellow("    %s\n", i18n.T("files_untracked", len(untrackedFiles)))
			for _, file := range untrackedFiles {
				printGray("      - %s\n", file)
			}
		}

		if len(stagedFiles) > 0 {
			hasLocalChanges = true
			printYellow("    %s\n", i18n.T("files_staged", len(stagedFiles)))
			for _, file := range stagedFiles {
				printGray("      - %s\n", file)
			}
		}

		if !hasLocalChanges {
			printGreen("    %s\n", i18n.T("clean_working_tree"))
		}
		fmt.Println()

		// Section 2: Remote Status
		printBlue("  %s\n", i18n.T("remote_status"))

		// Fetch from remote (suppress errors)
		_ = git.Fetch(fullPath)

		behindCount, _ := git.GetBehindCount(fullPath, branch)
		aheadCount, _ := git.GetAheadCount(fullPath, branch)

		if behindCount > 0 {
			printYellow("    %s\n", i18n.T("commits_behind", behindCount, branch))
		}

		if aheadCount > 0 {
			printYellow("    %s\n", i18n.T("commits_ahead", aheadCount))
		}

		if behindCount == 0 && aheadCount == 0 {
			printGreen("    %s\n", i18n.T("up_to_date"))
		}

		// Check if remote branch exists
		if behindCount == 0 && aheadCount == 0 {
			// Try to verify remote branch exists
			if err := git.Fetch(fullPath); err != nil {
				printRed("    %s\n", i18n.T("cannot_fetch"))
			}
		}
		fmt.Println()

		// Section 3: How to resolve
		needsResolution := hasLocalChanges || behindCount > 0 || aheadCount > 0

		if needsResolution {
			printBlue("  %s\n", i18n.T("how_to_resolve"))
			fmt.Println()

			if hasLocalChanges {
				printYellow("    %s\n", i18n.T("resolve_commit"))
				printGray("       cd %s\n", ws.Path)
				if len(stagedFiles) > 0 || len(modifiedFiles) > 0 {
					printGray("       git add .\n")
					printGray("       git commit -m \"your message\"\n")
				}
				if len(untrackedFiles) > 0 {
					printGray("       %s\n", i18n.T("resolve_or_gitignore"))
				}
				fmt.Println()
			}

			if behindCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_pull"))
				printGray("       git workspace pull %s\n", ws.Path)
				fmt.Println()
			}

			if aheadCount > 0 {
				printYellow("    %s\n", i18n.T("resolve_push"))
				printGray("       cd %s\n", ws.Path)
				printGray("       git push\n")
				fmt.Println()
			}
		} else {
			printGreen("  %s\n", i18n.T("no_action_needed"))
			fmt.Println()
		}
	}

	return nil
}
