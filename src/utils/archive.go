package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/klauspost/pgzip"
)

// CreateArchive creates a tar.gz archive with multi-core compression
func CreateArchive(files []string, outputPath string) error {
	log.Printf("📦 Creating archive: %s", filepath.Base(outputPath))
	
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Limit to 75% of cores, minimum 1, maximum 8
	cores := runtime.NumCPU()
	maxCores := max(1, min(8, cores*3/4))
	
	gw, err := pgzip.NewWriterLevel(out, pgzip.BestSpeed)
	if err != nil {
		return err
	}
	gw.SetConcurrency(1<<20, maxCores) // 1MB blocks, limited cores
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	for _, file := range files {
		if err := addToArchive(tw, file); err != nil {
			return err
		}
	}
	
	log.Printf("✅ Archive created successfully: %s", filepath.Base(outputPath))
	return nil
}

// ExtractArchive extracts a tar.gz archive to the specified directory
func ExtractArchive(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	// Store directory timestamps to set them after all files are extracted
	dirTimestamps := make(map[string]*tar.Header)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Convert tar path (always forward slashes) to OS-specific path
		osPath := filepath.FromSlash(header.Name)
		target := filepath.Join(destDir, osPath)
		
		// Ensure the target directory exists
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
			// Store directory timestamp for later
			dirTimestamps[target] = header
		case tar.TypeReg:
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			
			// Preserve file timestamps immediately
			if err := os.Chtimes(target, header.AccessTime, header.ModTime); err != nil {
				// Don't fail extraction if timestamp setting fails
				log.Printf("⚠️ Warning: Could not set timestamps for %s: %v", target, err)
			}
		}
	}

	// Set directory timestamps after all files are extracted
	for dirPath, header := range dirTimestamps {
		if err := os.Chtimes(dirPath, header.AccessTime, header.ModTime); err != nil {
			// Don't fail extraction if timestamp setting fails
			log.Printf("⚠️ Warning: Could not set timestamps for directory %s: %v", dirPath, err)
		}
	}

	return nil
}

// addToArchive recursively adds files to tar archive
func addToArchive(tw *tar.Writer, filePath string) error {
	return filepath.Walk(filePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		log.Printf("➕ Adding to archive: %s", path)
		
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		// Use relative path from the base directory to avoid TAR path length limits
		relPath, err := filepath.Rel(filepath.Dir(filePath), path)
		if err != nil {
			// Fallback to just the filename if relative path fails
			relPath = filepath.Base(path)
		}
		// Normalize path for cross-platform compatibility (always use forward slashes in tar)
		header.Name = NormalizePath(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		return err
	})
}
