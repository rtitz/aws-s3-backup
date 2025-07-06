package utils

import (
	"archive/tar"
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
		header.Name = path

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		return err
	})
}
