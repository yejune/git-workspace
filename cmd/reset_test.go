package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yejune/git-workspace/internal/manifest"
)

// ============ Basic Tests (3개) ============

func TestRunReset_BasicMotherRepo(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create manifest with mother repo keep files
	m := &manifest.Manifest{
		Keep: []string{"config.json", ".env"},
	}
	manifest.Save(dir, m)

	// Create keep files
	configPath := filepath.Join(dir, "config.json")
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(configPath, []byte(`{"port": 3000}`), 0644)
	os.WriteFile(envPath, []byte("DEBUG=true"), 0644)

	// Commit files
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Add config files").Run()

	// Apply skip-worktree
	exec.Command("git", "-C", dir, "update-index", "--skip-worktree", "config.json").Run()
	exec.Command("git", "-C", dir, "update-index", "--skip-worktree", ".env").Run()

	t.Run("reset mother repo keep files", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err != nil {
			t.Fatalf("runReset failed: %v", err)
		}

		// Check keep files are unskipped
		output, _ := exec.Command("git", "-C", dir, "ls-files", "-v").Output()
		if strings.Contains(string(output), "S config.json") {
			t.Error("config.json should be unskipped")
		}
		if strings.Contains(string(output), "S .env") {
			t.Error(".env should be unskipped")
		}

		// Check manifest keep list is cleared
		m, _ := manifest.Load(dir)
		if len(m.Keep) != 0 {
			t.Errorf("manifest keep should be empty, got %d items", len(m.Keep))
		}

		// Check backup created
		backupDir := filepath.Join(dir, ".workspaces", "backup")
		if _, err := os.Stat(backupDir); os.IsNotExist(err) {
			t.Error("backup directory should exist")
		}
	})
}

func TestRunReset_BasicWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Clone workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "apps/admin"})

	// Add keep files to workspace
	wsPath := filepath.Join(dir, "apps/admin")
	configPath := filepath.Join(wsPath, "config.json")
	os.WriteFile(configPath, []byte(`{"api": "localhost"}`), 0644)
	exec.Command("git", "-C", wsPath, "add", ".").Run()
	exec.Command("git", "-C", wsPath, "commit", "-m", "Add config").Run()

	// Update manifest with keep
	m, _ := manifest.Load(dir)
	for i := range m.Workspaces {
		if m.Workspaces[i].Path == "apps/admin" {
			m.Workspaces[i].Keep = []string{"config.json"}
		}
	}
	manifest.Save(dir, m)

	// Apply skip-worktree
	exec.Command("git", "-C", wsPath, "update-index", "--skip-worktree", "config.json").Run()

	t.Run("reset workspace keep files", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err != nil {
			t.Fatalf("runReset failed: %v", err)
		}

		// Check keep files are unskipped
		output, _ := exec.Command("git", "-C", wsPath, "ls-files", "-v").Output()
		if strings.Contains(string(output), "S config.json") {
			t.Error("config.json should be unskipped")
		}

		// Check workspace keep list is cleared
		m, _ := manifest.Load(dir)
		ws := m.Find("apps/admin")
		if ws != nil && len(ws.Keep) != 0 {
			t.Errorf("workspace keep should be empty, got %d items", len(ws.Keep))
		}
	})
}

func TestRunReset_BasicIgnorePatterns(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create manifest with ignore patterns
	m := &manifest.Manifest{
		Ignore: []string{"*.log", "tmp/"},
	}
	manifest.Save(dir, m)

	// Sync to apply ignore patterns to .gitignore
	runSync(syncCmd, []string{})

	// Verify patterns are in .gitignore
	gitignoreContent, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if !strings.Contains(string(gitignoreContent), "*.log") {
		t.Error(".gitignore should contain *.log before reset")
	}

	t.Run("reset removes ignore patterns", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err != nil {
			t.Fatalf("runReset failed: %v", err)
		}

		// Check ignore patterns removed from .gitignore
		gitignoreContent, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		content := string(gitignoreContent)
		if strings.Contains(content, "*.log") || strings.Contains(content, "tmp/") {
			t.Error(".gitignore should not contain ignore patterns after reset")
		}

		// Check manifest ignore list is cleared
		m, _ := manifest.Load(dir)
		if len(m.Ignore) != 0 {
			t.Errorf("manifest ignore should be empty, got %d items", len(m.Ignore))
		}
	})
}

// ============ Error Tests (3개) ============

func TestRunReset_ErrorNotInGitRepo(t *testing.T) {
	dir := t.TempDir()
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir(dir)

	t.Run("reset outside git repo", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err == nil {
			t.Error("should error when not in a git repository")
		}
		if !strings.Contains(err.Error(), "not in a git repository") {
			t.Errorf("expected 'not in a git repository' error, got: %v", err)
		}
	})
}

func TestRunReset_ErrorManifestLoad(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create invalid manifest
	manifestPath := filepath.Join(dir, ".git.workspaces")
	os.WriteFile(manifestPath, []byte("invalid: yaml: [[["), 0644)

	t.Run("reset with invalid manifest", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err == nil {
			t.Error("should error with invalid manifest")
		}
		if !strings.Contains(err.Error(), "failed to load manifest") {
			t.Errorf("expected 'failed to load manifest' error, got: %v", err)
		}
	})
}

func TestRunReset_ErrorBackupCreation(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create manifest with keep file
	m := &manifest.Manifest{
		Keep: []string{"config.json"},
	}
	manifest.Save(dir, m)

	// Create keep file
	configPath := filepath.Join(dir, "config.json")
	os.WriteFile(configPath, []byte(`{"test": true}`), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Add config").Run()

	// Make backup directory unwritable
	backupDir := filepath.Join(dir, ".workspaces", "backup")
	os.MkdirAll(backupDir, 0755)
	os.Chmod(backupDir, 0444)
	defer os.Chmod(backupDir, 0755)

	t.Run("reset with backup error", func(t *testing.T) {
		err := runReset(resetCmd, []string{})
		if err == nil {
			t.Error("should error when backup fails")
		}
		if !strings.Contains(err.Error(), "failed to backup") {
			t.Errorf("expected 'failed to backup' error, got: %v", err)
		}
	})
}

// ============ Integration Tests (2개) ============

func TestRunReset_IntegrationMultipleWorkspaces(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo1 := setupRemoteRepo(t)
	remoteRepo2 := setupRemoteRepo(t)

	// Clone multiple workspaces
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo1, "apps/api"})
	runClone(cloneCmd, []string{remoteRepo2, "apps/web"})

	// Add keep files to both workspaces
	apiPath := filepath.Join(dir, "apps/api")
	webPath := filepath.Join(dir, "apps/web")

	os.WriteFile(filepath.Join(apiPath, "api.config"), []byte("api"), 0644)
	exec.Command("git", "-C", apiPath, "add", ".").Run()
	exec.Command("git", "-C", apiPath, "commit", "-m", "Add api config").Run()

	os.WriteFile(filepath.Join(webPath, "web.config"), []byte("web"), 0644)
	exec.Command("git", "-C", webPath, "add", ".").Run()
	exec.Command("git", "-C", webPath, "commit", "-m", "Add web config").Run()

	// Update manifest
	m, _ := manifest.Load(dir)
	for i := range m.Workspaces {
		if m.Workspaces[i].Path == "apps/api" {
			m.Workspaces[i].Keep = []string{"api.config"}
		}
		if m.Workspaces[i].Path == "apps/web" {
			m.Workspaces[i].Keep = []string{"web.config"}
		}
	}
	manifest.Save(dir, m)

	// Apply skip-worktree
	exec.Command("git", "-C", apiPath, "update-index", "--skip-worktree", "api.config").Run()
	exec.Command("git", "-C", webPath, "update-index", "--skip-worktree", "web.config").Run()

	t.Run("reset multiple workspaces", func(t *testing.T) {
		output := captureOutput(func() {
			err := runReset(resetCmd, []string{})
			if err != nil {
				t.Fatalf("runReset failed: %v", err)
			}
		})

		// Check both workspaces are mentioned in output
		if !strings.Contains(output, "apps/api") {
			t.Error("output should mention apps/api")
		}
		if !strings.Contains(output, "apps/web") {
			t.Error("output should mention apps/web")
		}

		// Check all keep files unskipped
		apiOutput, _ := exec.Command("git", "-C", apiPath, "ls-files", "-v").Output()
		if strings.Contains(string(apiOutput), "S api.config") {
			t.Error("api.config should be unskipped")
		}

		webOutput, _ := exec.Command("git", "-C", webPath, "ls-files", "-v").Output()
		if strings.Contains(string(webOutput), "S web.config") {
			t.Error("web.config should be unskipped")
		}

		// Check manifest cleared
		m, _ := manifest.Load(dir)
		for _, ws := range m.Workspaces {
			if len(ws.Keep) != 0 {
				t.Errorf("workspace %s keep should be empty", ws.Path)
			}
		}
	})
}

func TestRunReset_IntegrationMixedMotherAndWorkspace(t *testing.T) {
	dir, cleanup := setupTestEnv(t)
	defer cleanup()

	remoteRepo := setupRemoteRepo(t)

	// Create manifest with both mother repo keep and workspace keep
	m := &manifest.Manifest{
		Keep:   []string{".env"},
		Ignore: []string{"*.log"},
	}
	manifest.Save(dir, m)

	// Create mother repo keep file
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("SECRET=123"), 0644)
	exec.Command("git", "-C", dir, "add", ".").Run()
	exec.Command("git", "-C", dir, "commit", "-m", "Add env").Run()
	exec.Command("git", "-C", dir, "update-index", "--skip-worktree", ".env").Run()

	// Clone workspace
	cloneBranch = ""
	runClone(cloneCmd, []string{remoteRepo, "libs/utils"})

	// Add keep file to workspace
	wsPath := filepath.Join(dir, "libs/utils")
	wsConfig := filepath.Join(wsPath, "utils.config")
	os.WriteFile(wsConfig, []byte("config"), 0644)
	exec.Command("git", "-C", wsPath, "add", ".").Run()
	exec.Command("git", "-C", wsPath, "commit", "-m", "Add config").Run()

	// Update manifest
	m, _ = manifest.Load(dir)
	for i := range m.Workspaces {
		if m.Workspaces[i].Path == "libs/utils" {
			m.Workspaces[i].Keep = []string{"utils.config"}
		}
	}
	manifest.Save(dir, m)

	exec.Command("git", "-C", wsPath, "update-index", "--skip-worktree", "utils.config").Run()

	// Apply ignore patterns
	runSync(syncCmd, []string{})

	t.Run("reset both mother repo and workspace", func(t *testing.T) {
		output := captureOutput(func() {
			err := runReset(resetCmd, []string{})
			if err != nil {
				t.Fatalf("runReset failed: %v", err)
			}
		})

		// Check output mentions both
		if !strings.Contains(output, "Mother repo") {
			t.Error("output should mention Mother repo")
		}
		if !strings.Contains(output, "libs/utils") {
			t.Error("output should mention workspace")
		}

		// Check mother repo keep unskipped
		motherOutput, _ := exec.Command("git", "-C", dir, "ls-files", "-v").Output()
		if strings.Contains(string(motherOutput), "S .env") {
			t.Error(".env should be unskipped")
		}

		// Check workspace keep unskipped
		wsOutput, _ := exec.Command("git", "-C", wsPath, "ls-files", "-v").Output()
		if strings.Contains(string(wsOutput), "S utils.config") {
			t.Error("utils.config should be unskipped")
		}

		// Check ignore patterns removed
		gitignoreContent, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		if strings.Contains(string(gitignoreContent), "*.log") {
			t.Error(".gitignore should not contain *.log after reset")
		}

		// Check manifest completely cleared
		m, _ := manifest.Load(dir)
		if len(m.Keep) != 0 {
			t.Error("mother repo keep should be empty")
		}
		if len(m.Ignore) != 0 {
			t.Error("mother repo ignore should be empty")
		}
		for _, ws := range m.Workspaces {
			if len(ws.Keep) != 0 {
				t.Errorf("workspace %s keep should be empty", ws.Path)
			}
		}

		// Check patches preserved message
		if !strings.Contains(output, "Patches preserved") {
			t.Error("output should mention patches are preserved")
		}
	})
}
