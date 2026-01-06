package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	m, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(m.Subclones) != 0 {
		t.Errorf("expected 0 subclones, got %d", len(m.Subclones))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	m := &Manifest{
		Subclones: []Subclone{
			{Path: "packages/sub-a", Repo: "https://github.com/test/sub-a.git", Branch: "main"},
			{Path: "libs/sub-b", Repo: "https://github.com/test/sub-b.git"},
		},
	}

	if err := Save(dir, m); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Check file exists
	path := filepath.Join(dir, FileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("manifest file not created")
	}

	// Load and verify
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Subclones) != 2 {
		t.Errorf("expected 2 subclones, got %d", len(loaded.Subclones))
	}

	if loaded.Subclones[0].Path != "packages/sub-a" {
		t.Errorf("expected path packages/sub-a, got %s", loaded.Subclones[0].Path)
	}

	if loaded.Subclones[0].Branch != "main" {
		t.Errorf("expected branch main, got %s", loaded.Subclones[0].Branch)
	}
}

func TestAddAndRemove(t *testing.T) {
	m := &Manifest{Subclones: []Subclone{}}

	m.Add("test/path", "https://github.com/test/repo.git", "develop")

	if len(m.Subclones) != 1 {
		t.Errorf("expected 1 subclone, got %d", len(m.Subclones))
	}

	if !m.Exists("test/path") {
		t.Error("expected subclone to exist")
	}

	if !m.Remove("test/path") {
		t.Error("expected Remove to return true")
	}

	if m.Exists("test/path") {
		t.Error("expected subclone to not exist")
	}

	if m.Remove("nonexistent") {
		t.Error("expected Remove to return false for nonexistent path")
	}
}

func TestFind(t *testing.T) {
	m := &Manifest{
		Subclones: []Subclone{
			{Path: "a", Repo: "repo-a"},
			{Path: "b", Repo: "repo-b"},
		},
	}

	sc := m.Find("a")
	if sc == nil {
		t.Fatal("expected to find subclone")
	}
	if sc.Repo != "repo-a" {
		t.Errorf("expected repo-a, got %s", sc.Repo)
	}

	if m.Find("nonexistent") != nil {
		t.Error("expected nil for nonexistent path")
	}
}
