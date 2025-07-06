package tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rtitz/aws-s3-backup/utils"
)

func TestCombineSplitFiles(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "test_combine")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test split files
	testData := []byte("Hello World Test Data")
	part1Data := testData[:10]
	part2Data := testData[10:]

	// Write part files
	part1Path := filepath.Join(tmpDir, "test.tar.gz-part00001")
	part2Path := filepath.Join(tmpDir, "test.tar.gz-part00002")
	
	if err := os.WriteFile(part1Path, part1Data, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(part2Path, part2Data, 0644); err != nil {
		t.Fatal(err)
	}

	// Test combination
	err = utils.CombineFiles(tmpDir)
	if err != nil {
		t.Fatalf("CombineSplitFiles failed: %v", err)
	}

	// Verify combined file exists
	combinedPath := filepath.Join(tmpDir, "test.tar.gz")
	if _, err := os.Stat(combinedPath); os.IsNotExist(err) {
		t.Fatal("Combined file was not created")
	}

	// Verify content
	combinedData, err := os.ReadFile(combinedPath)
	if err != nil {
		t.Fatal(err)
	}

	if string(combinedData) != string(testData) {
		t.Fatalf("Combined data mismatch: got %q, want %q", string(combinedData), string(testData))
	}

	// Verify part files were cleaned up
	if _, err := os.Stat(part1Path); !os.IsNotExist(err) {
		t.Fatal("Part file 1 was not cleaned up")
	}
	if _, err := os.Stat(part2Path); !os.IsNotExist(err) {
		t.Fatal("Part file 2 was not cleaned up")
	}
}