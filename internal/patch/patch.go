package patch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Create creates a patch file from the diff between HEAD and working tree
// in unified diff format. If file is empty, diffs all changes.
func Create(repoPath, file, patchPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repoPath cannot be empty")
	}
	if patchPath == "" {
		return fmt.Errorf("patchPath cannot be empty")
	}

	// Ensure patch directory exists
	patchDir := filepath.Dir(patchPath)
	if err := os.MkdirAll(patchDir, 0755); err != nil {
		return fmt.Errorf("failed to create patch directory: %w", err)
	}

	// Build git diff command
	args := []string{"diff", "HEAD"}
	if file != "" {
		args = append(args, "--", file)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	// Capture output
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("git diff failed: %w\nstderr: %s", err, exitErr.Stderr)
		}
		return fmt.Errorf("git diff failed: %w", err)
	}

	// Write patch file
	if err := os.WriteFile(patchPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write patch file: %w", err)
	}

	return nil
}

// Apply applies a patch file to the working tree using the patch command.
// The patch must be in unified diff format.
func Apply(repoPath, patchPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repoPath cannot be empty")
	}
	if patchPath == "" {
		return fmt.Errorf("patchPath cannot be empty")
	}

	// Check if patch file exists
	if _, err := os.Stat(patchPath); err != nil {
		return fmt.Errorf("patch file not found: %w", err)
	}

	// Open patch file
	patchFile, err := os.Open(patchPath)
	if err != nil {
		return fmt.Errorf("failed to open patch file: %w", err)
	}
	defer patchFile.Close()

	// Apply patch with -p1 to strip leading path component
	cmd := exec.Command("patch", "-p1")
	cmd.Dir = repoPath
	cmd.Stdin = patchFile

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("patch apply failed: %w\noutput: %s", err, output)
	}

	return nil
}

// Check checks if applying the patch would cause conflicts using --dry-run.
// Returns true if conflicts are detected, false otherwise.
func Check(repoPath, patchPath string) (hasConflicts bool, err error) {
	if repoPath == "" {
		return false, fmt.Errorf("repoPath cannot be empty")
	}
	if patchPath == "" {
		return false, fmt.Errorf("patchPath cannot be empty")
	}

	// Check if patch file exists
	if _, err := os.Stat(patchPath); err != nil {
		return false, fmt.Errorf("patch file not found: %w", err)
	}

	// Open patch file
	patchFile, err := os.Open(patchPath)
	if err != nil {
		return false, fmt.Errorf("failed to open patch file: %w", err)
	}
	defer patchFile.Close()

	// Run patch with --dry-run to check for conflicts
	cmd := exec.Command("patch", "--dry-run", "-p1")
	cmd.Dir = repoPath
	cmd.Stdin = patchFile

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		outputStr := string(output)

		// Check if it's actually a conflict (patch can apply but would conflict)
		// Common conflict indicators in patch output
		if strings.Contains(outputStr, "FAILED") ||
			strings.Contains(outputStr, "rejected") ||
			strings.Contains(outputStr, "Hunk") && strings.Contains(outputStr, "FAILED") {
			return true, nil // Has conflicts, no error
		}

		// Other patch errors (malformed patch, file not found, etc.)
		return false, fmt.Errorf("patch check failed: %s", outputStr)
	}

	return false, nil
}
