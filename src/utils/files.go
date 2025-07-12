package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
)

// File splitting constants
const (
	BytesPerMB      = 1024 * 1024
	PartNumFormat   = "%s-part%05d"
	PartNumDigits   = 5
	DefaultFilePerm = 0644
	DefaultDirPerm  = 0755
)

// OpenFile in OS default editor
func OpenFile(filePath string) error {
	var cmd *exec.Cmd

	// Determine the operating system
	switch runtime.GOOS {
	case "darwin": // macOS
		cmd = exec.Command("open", filePath)
	case "linux":
		cmd = exec.Command("xdg-open", filePath)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", filePath)
	default:
		return fmt.Errorf("unsupported platform")
	}

	// Run the command
	return cmd.Start()
}

// SplitFile splits a file into smaller chunks if it exceeds the specified size
func SplitFile(filePath string, chunkSizeMB int64) ([]string, error) {
	log.Printf("🔍 Checking if file needs splitting: %s", filepath.Base(filePath))

	fileInfo, err := getFileInfo(filePath)
	if err != nil {
		return nil, err
	}

	chunkSize := chunkSizeMB * BytesPerMB
	if !needsSplitting(fileInfo.Size(), chunkSize) {
		log.Printf("✅ File does not need splitting: %s (%s)",
			filepath.Base(filePath), FormatBytes(fileInfo.Size()))
		return []string{filePath}, nil
	}

	return performFileSplit(filePath, fileInfo.Size(), chunkSize)
}

// CombineFiles combines split files back together
func CombineFiles(downloadDir string) error {
	log.Printf("🔍 Scanning for split files to combine in: %s", downloadDir)

	splitGroups, err := findSplitFileGroups(downloadDir)
	if err != nil {
		return err
	}

	if len(splitGroups) == 0 {
		log.Printf("ℹ️ No split files found to combine")
		return nil
	}

	return combineSplitGroups(splitGroups, downloadDir)
}

// GetFileChecksum calculates SHA256 checksum of a file
func GetFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetFileSize returns the size of a file
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}
	return info.Size(), nil
}

// FormatBytes formats bytes into human readable format
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Helper functions for file splitting
// getFileInfo gets file information for splitting
func getFileInfo(filePath string) (os.FileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return info, nil
}

// needsSplitting checks if file exceeds chunk size
func needsSplitting(fileSize, chunkSize int64) bool {
	return fileSize > chunkSize
}

// performFileSplit splits file into chunks
func performFileSplit(filePath string, fileSize, chunkSize int64) ([]string, error) {
	numParts := calculateNumParts(fileSize, chunkSize)
	log.Printf("✂️ Splitting into %d MB chunks: %s (%s) -> %d parts",
		chunkSize/BytesPerMB, filepath.Base(filePath), FormatBytes(fileSize), numParts)

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file for splitting: %w", err)
	}
	defer file.Close()

	parts, err := createFileParts(file, filePath, chunkSize, numParts)
	if err != nil {
		return nil, err
	}

	log.Printf("✅ File split into %d parts", len(parts))
	return parts, nil
}

// calculateNumParts computes number of parts needed
func calculateNumParts(fileSize, chunkSize int64) int64 {
	return (fileSize + chunkSize - 1) / chunkSize
}

// createFileParts creates individual part files
func createFileParts(file *os.File, filePath string, chunkSize int64, numParts int64) ([]string, error) {
	var parts []string
	buffer := make([]byte, chunkSize)

	for partNum := 1; int64(partNum) <= numParts; partNum++ {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read file chunk: %w", err)
		}

		partPath := fmt.Sprintf(PartNumFormat, filePath, partNum)
		log.Printf("📦 Creating part %d/%d: %s", partNum, numParts, filepath.Base(partPath))

		if err := os.WriteFile(partPath, buffer[:n], DefaultFilePerm); err != nil {
			return nil, fmt.Errorf("failed to write part file: %w", err)
		}

		parts = append(parts, partPath)
	}

	return parts, nil
}

// Helper functions for file combining
// findSplitFileGroups finds and groups split files for combining
func findSplitFileGroups(downloadDir string) (map[string][]string, error) {
	splitGroups := make(map[string][]string)
	partPattern := regexp.MustCompile(`^(.+)-part(\d{5})$`)

	err := filepath.Walk(downloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		return processPotentialSplitFile(path, info, downloadDir, partPattern, splitGroups)
	})

	return splitGroups, err
}

// processPotentialSplitFile processes a file that might be a split part
func processPotentialSplitFile(path string, info os.FileInfo, downloadDir string,
	partPattern *regexp.Regexp, splitGroups map[string][]string) error {

	filename := info.Name()
	matches := partPattern.FindStringSubmatch(filename)

	if len(matches) == 3 {
		baseName := matches[1]
		key := buildSplitGroupKey(path, downloadDir, baseName)
		log.Printf("🔍 Found split file: %s (group: %s)", filename, key)
		splitGroups[key] = append(splitGroups[key], path)
	}

	return nil
}

// buildSplitGroupKey creates a key for grouping split files
func buildSplitGroupKey(path, downloadDir, baseName string) string {
	relPath, _ := filepath.Rel(downloadDir, filepath.Dir(path))
	if relPath == "." {
		relPath = ""
	}
	return filepath.Join(relPath, baseName)
}

// combineSplitGroups combines all groups of split files
func combineSplitGroups(splitGroups map[string][]string, downloadDir string) error {
	for baseName, parts := range splitGroups {
		if len(parts) > 1 {
			if err := combineSingleGroup(baseName, parts, downloadDir); err != nil {
				return err
			}
		}
	}
	return nil
}

// combineSingleGroup combines parts of a single split file
func combineSingleGroup(baseName string, parts []string, downloadDir string) error {
	totalSize := calculateTotalSize(parts)
	log.Printf("🔗 Combining %d parts: %s (%s) -> 1 file",
		len(parts), filepath.Base(baseName), FormatBytes(totalSize))

	sortPartsByNumber(parts)

	if err := combineFileParts(parts, baseName, downloadDir); err != nil {
		return err
	}

	cleanupAfterCombine(baseName, downloadDir)
	log.Printf("✅ Successfully combined: %s (%s)", filepath.Base(baseName), FormatBytes(totalSize))

	return nil
}

// calculateTotalSize computes total size of all parts
func calculateTotalSize(parts []string) int64 {
	var totalSize int64
	for _, part := range parts {
		if info, err := os.Stat(part); err == nil {
			totalSize += info.Size()
		}
	}
	return totalSize
}

// sortPartsByNumber sorts parts by their part number
func sortPartsByNumber(parts []string) {
	sort.Slice(parts, func(i, j int) bool {
		return extractPartNumber(parts[i]) < extractPartNumber(parts[j])
	})
}

// extractPartNumber extracts part number from filename
func extractPartNumber(filename string) int {
	partPattern := regexp.MustCompile(`-part(\d{5})$`)
	matches := partPattern.FindStringSubmatch(filepath.Base(filename))
	if len(matches) == 2 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}

// combineFileParts combines parts into single file
func combineFileParts(parts []string, baseName, downloadDir string) error {
	outputPath := filepath.Join(downloadDir, baseName)

	if err := os.MkdirAll(filepath.Dir(outputPath), DefaultDirPerm); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create combined file: %w", err)
	}
	defer output.Close()

	return copyAndCleanupParts(parts, output)
}

// copyAndCleanupParts copies parts to output and removes them
func copyAndCleanupParts(parts []string, output *os.File) error {
	for i, partPath := range parts {
		log.Printf("🔗 Processing part %d/%d: %s", i+1, len(parts), filepath.Base(partPath))

		if err := copyPartToOutput(partPath, output); err != nil {
			return err
		}

		if err := os.Remove(partPath); err != nil {
			log.Printf("⚠️ Warning: Could not remove part file %s: %v", partPath, err)
		}
	}
	return nil
}

// copyPartToOutput copies single part to output file
func copyPartToOutput(partPath string, output *os.File) error {
	part, err := os.Open(partPath)
	if err != nil {
		return fmt.Errorf("failed to open part file: %w", err)
	}
	defer part.Close()

	if _, err := io.Copy(output, part); err != nil {
		return fmt.Errorf("failed to copy part data: %w", err)
	}

	return nil
}

// cleanupAfterCombine removes instruction files after combining
func cleanupAfterCombine(baseName, downloadDir string) {
	howToFile := filepath.Join(downloadDir, baseName+"-HowToBuild.txt")
	if _, err := os.Stat(howToFile); err == nil {
		os.Remove(howToFile)
		log.Printf("🗑️ Removed instruction file: %s-HowToBuild.txt", filepath.Base(baseName))
	}
}
