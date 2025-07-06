package utils

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// SplitFile splits a file into smaller chunks
func SplitFile(filePath string, chunkSizeMB int64) ([]string, error) {
	log.Printf("🔍 Checking if file needs splitting: %s", filepath.Base(filePath))
	
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	chunkSize := chunkSizeMB * 1024 * 1024
	if info.Size() <= chunkSize {
		log.Printf("✅ File does not need splitting: %s", filepath.Base(filePath))
		return []string{filePath}, nil
	}
	
	log.Printf("✂️ Splitting file into %d MB chunks: %s", chunkSizeMB, filepath.Base(filePath))

	var parts []string
	partNum := 1
	buffer := make([]byte, chunkSize)

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		partPath := fmt.Sprintf("%s-part%05d", filePath, partNum)
		log.Printf("📦 Creating part %d: %s", partNum, filepath.Base(partPath))
		if err := os.WriteFile(partPath, buffer[:n], 0644); err != nil {
			return nil, err
		}
		parts = append(parts, partPath)
		partNum++
	}

	log.Printf("✅ File split into %d parts", len(parts))
	return parts, nil
}

// CombineFiles combines split files back together
func CombineFiles(downloadDir string) error {
	log.Printf("🔍 Scanning for split files to combine in: %s", downloadDir)
	
	splitGroups := make(map[string][]string)
	partPattern := regexp.MustCompile(`^(.+)-part(\d{5})$`)

	// Walk through all directories recursively
	err := filepath.Walk(downloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		
		filename := info.Name()
		matches := partPattern.FindStringSubmatch(filename)
		if len(matches) == 3 {
			baseName := matches[1]
			// Use relative path from downloadDir as the key
			relPath, _ := filepath.Rel(downloadDir, filepath.Dir(path))
			if relPath == "." {
				relPath = ""
			}
			key := filepath.Join(relPath, baseName)
			log.Printf("🔍 Found split file: %s (group: %s)", filename, key)
			splitGroups[key] = append(splitGroups[key], path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if len(splitGroups) == 0 {
		log.Printf("ℹ️ No split files found to combine")
		return nil
	}
	
	for baseName, parts := range splitGroups {
		if len(parts) > 1 {
			log.Printf("🔗 Combining %d parts for %s", len(parts), baseName)
			sort.Slice(parts, func(i, j int) bool {
				return extractPartNumber(parts[i]) < extractPartNumber(parts[j])
			})
			if err := combineFileParts(parts, baseName, downloadDir); err != nil {
				return err
			}
			// Remove HowToBuild.txt file if it exists
			howToFile := filepath.Join(downloadDir, baseName+"-HowToBuild.txt")
			if _, err := os.Stat(howToFile); err == nil {
				os.Remove(howToFile)
				log.Printf("🗑️ Removed instruction file: %s-HowToBuild.txt", baseName)
			}
			log.Printf("✅ Successfully combined %s", baseName)
		}
	}
	return nil
}

func extractPartNumber(filename string) int {
	partPattern := regexp.MustCompile(`-part(\d{5})$`)
	matches := partPattern.FindStringSubmatch(filepath.Base(filename))
	if len(matches) == 2 {
		num, _ := strconv.Atoi(matches[1])
		return num
	}
	return 0
}

func combineFileParts(parts []string, baseName, downloadDir string) error {
	outputPath := filepath.Join(downloadDir, baseName)
	// Ensure output directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	output, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer output.Close()

	for i, partPath := range parts {
		log.Printf("🔗 Combining part %d/%d: %s", i+1, len(parts), filepath.Base(partPath))
		part, err := os.Open(partPath)
		if err != nil {
			return err
		}
		io.Copy(output, part)
		part.Close()
		os.Remove(partPath)
	}
	return nil
}

// GetFileChecksum calculates SHA256 checksum of a file
func GetFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetFileSize returns the size of a file
func GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}