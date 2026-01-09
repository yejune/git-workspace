package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CreatePatchBackup backs up a patch file with timestamp to the backup directory
// Backup structure: backup/patched/yyyy/mm/dd/sub-path/file.yyyymmdd_hhmmss.patch
func CreatePatchBackup(patchPath, backupDir string) error {
	// Check if patch file exists
	if _, err := os.Stat(patchPath); os.IsNotExist(err) {
		return nil // No patch to backup
	}

	// Generate timestamp
	now := time.Now()
	timestamp := now.Format("20060102_150405")

	// Extract relative path from .workspaces/patches/ (handle both absolute and relative paths)
	relPath := patchPath
	// Try to find .workspaces/patches/ in the path
	if idx := strings.Index(patchPath, ".workspaces/patches/"); idx != -1 {
		relPath = patchPath[idx+len(".workspaces/patches/"):]
	} else if idx := strings.Index(patchPath, ".workspaces-patches/"); idx != -1 {
		// Fallback for old naming
		relPath = patchPath[idx+len(".workspaces-patches/"):]
	}

	// Build backup path: backup/patched/yyyy/mm/dd/...
	backupPath := filepath.Join(
		backupDir,
		"patched",
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
		relPath,
	)

	// Add timestamp to filename
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	backupPath = filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy patch file to backup location
	return copyFile(patchPath, backupPath)
}

// CreateFileBackup backs up the entire file with timestamp
// Backup structure: backup/modified/yyyy/mm/dd/sub-path/file.yyyymmdd_hhmmss.ext
func CreateFileBackup(filePath, backupDir, repoRoot string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // No file to backup
	}

	// Generate timestamp
	now := time.Now()
	timestamp := now.Format("20060102_150405")

	// Extract relative path from repoRoot (handle both absolute and relative paths)
	relPath := filePath
	// If it's an absolute path, make it relative to repoRoot
	if filepath.IsAbs(filePath) {
		var err error
		relPath, err = filepath.Rel(repoRoot, filePath)
		if err != nil {
			// Fallback: use the original path if we can't make it relative
			relPath = filePath
		}
	}

	// Build backup path: backup/modified/yyyy/mm/dd/...
	backupPath := filepath.Join(
		backupDir,
		"modified",
		now.Format("2006"),
		now.Format("01"),
		now.Format("02"),
		relPath,
	)

	// Add timestamp to filename
	dir := filepath.Dir(backupPath)
	base := filepath.Base(backupPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	backupPath = filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, timestamp, ext))

	// Create directory if not exists
	if err := os.MkdirAll(filepath.Dir(backupPath), 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Copy file to backup location
	return copyFile(filePath, backupPath)
}

// Cleanup removes backups older than specified days
func Cleanup(backupDir string, days int) error {
	if days <= 0 {
		return fmt.Errorf("days must be positive")
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	return filepath.Walk(backupDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file is older than cutoff
		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove old backup %s: %w", path, err)
			}
		}

		return nil
	})
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Sync to ensure data is written to disk
	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}
