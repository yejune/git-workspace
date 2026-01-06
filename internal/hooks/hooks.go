// Package hooks handles git hooks installation
package hooks

import (
	"os"
	"path/filepath"
)

const postCheckoutHook = `#!/bin/sh
# git-subclone post-checkout hook
# Automatically syncs subclones after checkout

if command -v git-subclone >/dev/null 2>&1; then
    git-subclone sync --recursive
fi
`

// Install installs git hooks in the repository
func Install(repoRoot string) error {
	hooksDir := filepath.Join(repoRoot, ".git", "hooks")

	// Ensure hooks directory exists
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return err
	}

	// Install post-checkout hook
	hookPath := filepath.Join(hooksDir, "post-checkout")
	return os.WriteFile(hookPath, []byte(postCheckoutHook), 0755)
}

// Uninstall removes git hooks from the repository
func Uninstall(repoRoot string) error {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")

	// Read current hook
	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	// Only remove if it's our hook
	if string(content) == postCheckoutHook {
		return os.Remove(hookPath)
	}

	return nil
}

// IsInstalled checks if the hook is installed
func IsInstalled(repoRoot string) bool {
	hookPath := filepath.Join(repoRoot, ".git", "hooks", "post-checkout")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		return false
	}
	return string(content) == postCheckoutHook
}
