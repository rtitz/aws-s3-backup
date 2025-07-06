package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rtitz/aws-s3-backup/utils"
)

func TestPgzipCompression(t *testing.T) {
	// Create temporary directories
	tmpDir, err := os.MkdirTemp("", "test_pgzip")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testDir := filepath.Join(tmpDir, "testdata")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(testDir, "test.txt")
	testData := []byte("Hello World! This is test data for pgzip compression.")
	if err := os.WriteFile(testFile, testData, 0644); err != nil {
		t.Fatal(err)
	}

	// Test archive creation with pgzip
	archivePath := filepath.Join(tmpDir, "test.tar.gz")
	err = utils.CreateArchive([]string{testDir}, archivePath)
	if err != nil {
		t.Fatalf("CreateArchive failed: %v", err)
	}

	// Verify archive was created
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatal("Archive was not created")
	}

	// Verify archive is a valid gzip file by checking file size > 0
	info, err := os.Stat(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("Archive file is empty")
	}
}