package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ArchiveOldBackups archives previous month backups to tar.gz and removes originals
// Archives are saved as: archived/YYYY-MM-{modified|patched}.tar.gz
// Only previous months are archived, current month is preserved
func ArchiveOldBackups(backupDir string) error {
	now := time.Now()
	currentYear := now.Format("2006")
	currentMonth := now.Format("01")

	fmt.Println("\n[Archive] Checking for old backups to archive...")

	// Process modified backups
	if err := archiveBackupType(backupDir, "modified", currentYear, currentMonth); err != nil {
		return fmt.Errorf("failed to archive modified backups: %w", err)
	}

	// Process patched backups
	if err := archiveBackupType(backupDir, "patched", currentYear, currentMonth); err != nil {
		return fmt.Errorf("failed to archive patched backups: %w", err)
	}

	fmt.Println("[Archive] Completed")
	return nil
}

// archiveBackupType archives a specific backup type (modified or patched)
func archiveBackupType(backupDir, backupType, currentYear, currentMonth string) error {
	typeDir := filepath.Join(backupDir, backupType)

	// Check if directory exists
	if _, err := os.Stat(typeDir); os.IsNotExist(err) {
		return nil // No backups to archive
	}

	// Get all year directories
	years, err := os.ReadDir(typeDir)
	if err != nil {
		return fmt.Errorf("failed to read %s directory: %w", backupType, err)
	}

	archivedCount := 0

	for _, yearEntry := range years {
		if !yearEntry.IsDir() {
			continue
		}

		year := yearEntry.Name()
		yearPath := filepath.Join(typeDir, year)

		// Get all month directories
		months, err := os.ReadDir(yearPath)
		if err != nil {
			fmt.Printf("  [Archive] Warning: failed to read year %s: %v\n", year, err)
			continue
		}

		for _, monthEntry := range months {
			if !monthEntry.IsDir() {
				continue
			}

			month := monthEntry.Name()
			monthPath := filepath.Join(yearPath, month)

			// Skip current month
			if year == currentYear && month == currentMonth {
				fmt.Printf("  [Archive] Skipping current month: %s/%s\n", year, month)
				continue
			}

			// Archive this month
			archiveName := fmt.Sprintf("%s-%s-%s.tar.gz", year, month, backupType)
			archivePath := filepath.Join(backupDir, "archived", archiveName)

			// Check if archive already exists
			if _, err := os.Stat(archivePath); err == nil {
				fmt.Printf("  [Archive] Already exists: %s\n", archiveName)
				continue
			}

			fmt.Printf("  [Archive] Archiving %s/%s/%s -> %s\n", backupType, year, month, archiveName)

			// Create archived directory if not exists
			archivedDir := filepath.Join(backupDir, "archived")
			if err := os.MkdirAll(archivedDir, 0755); err != nil {
				return fmt.Errorf("failed to create archived directory: %w", err)
			}

			// Create tar.gz archive
			// Use relative path from typeDir to preserve directory structure
			if err := createTarGz(typeDir, archivePath, year, month); err != nil {
				return fmt.Errorf("failed to create archive %s: %w", archiveName, err)
			}

			// Verify archive
			if err := verifyTarGz(archivePath); err != nil {
				// Remove corrupted archive
				os.Remove(archivePath)
				return fmt.Errorf("archive verification failed for %s: %w", archiveName, err)
			}

			fmt.Printf("  [Archive] Verified: %s\n", archiveName)

			// Remove original directory only after successful archive and verification
			if err := os.RemoveAll(monthPath); err != nil {
				return fmt.Errorf("failed to remove original directory %s: %w", monthPath, err)
			}

			fmt.Printf("  [Archive] Removed original: %s/%s/%s\n", backupType, year, month)
			archivedCount++

			// Clean up empty year directory
			remaining, err := os.ReadDir(yearPath)
			if err == nil && len(remaining) == 0 {
				os.Remove(yearPath)
			}
		}
	}

	if archivedCount > 0 {
		fmt.Printf("  [Archive] Archived %d month(s) for %s\n", archivedCount, backupType)
	}

	return nil
}

// createTarGz creates a tar.gz archive from a directory using native Go
func createTarGz(baseDir, archivePath, year, month string) error {
	srcDir := filepath.Join(baseDir, year, month)

	// Create archive file
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(archiveFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk directory and add files to tar
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from baseDir (preserves YYYY/MM structure)
		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header for %s: %w", path, err)
		}

		// Use relative path as name
		header.Name = filepath.ToSlash(relPath)

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// Write file contents if not directory
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file %s to tar: %w", path, err)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	return nil
}

// verifyTarGz verifies the integrity of a tar.gz archive using native Go
func verifyTarGz(archivePath string) error {
	// Open archive file
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer archiveFile.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Read all entries to verify archive structure
	fileCount := 0
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		fileCount++

		// Try to read file contents to verify integrity
		if !header.FileInfo().IsDir() {
			// Just consume the data without storing it
			if _, err := io.Copy(io.Discard, tarReader); err != nil {
				return fmt.Errorf("failed to verify file %s: %w", header.Name, err)
			}
		}
	}

	if fileCount == 0 {
		return fmt.Errorf("archive is empty")
	}

	return nil
}

// ShouldRunArchive checks if archiving should run (24 hours since last check)
func ShouldRunArchive(workspacesDir string) bool {
	checkFile := filepath.Join(workspacesDir, ".last-archive-check")

	// Check if file exists
	info, err := os.Stat(checkFile)
	if err != nil {
		// File doesn't exist, should run
		return true
	}

	// Check if 24 hours have passed
	lastCheck := info.ModTime()
	elapsed := time.Since(lastCheck)

	return elapsed >= 24*time.Hour
}

// UpdateArchiveCheck updates the last archive check timestamp
func UpdateArchiveCheck(workspacesDir string) error {
	checkFile := filepath.Join(workspacesDir, ".last-archive-check")

	// Create or update the file
	file, err := os.Create(checkFile)
	if err != nil {
		return fmt.Errorf("failed to create check file: %w", err)
	}
	defer file.Close()

	// Write current timestamp
	timestamp := time.Now().Format(time.RFC3339)
	if _, err := file.WriteString(timestamp + "\n"); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}

	return nil
}
