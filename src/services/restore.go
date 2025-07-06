package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/utils"
)

type RestoreService struct {
	cfg     aws.Config
	summary *RestoreSummary
}

type RestoreSummary struct {
	SuccessfulDownloads int
	FailedDownloads     int
	SkippedFiles        int
	Warnings            int
	TotalFiles          int
}

type S3Object struct {
	Key          string `json:"Key"`
	Size         int64  `json:"Size"`
	StorageClass string `json:"StorageClass"`
}

type S3Contents struct {
	Contents []S3Object `json:"Contents"`
}

func NewRestoreService(cfg aws.Config) *RestoreService {
	return &RestoreService{
		cfg:     cfg,
		summary: &RestoreSummary{},
	}
}

func (s *RestoreService) ProcessRestore(ctx context.Context, bucket, prefix, inputFile, downloadLocation string) error {
	fmt.Printf("\nMODE: RESTORE\n\n")

	if bucket == "" {
		return s.listBuckets(ctx)
	}

	if downloadLocation == "" {
		return fmt.Errorf("❌ download location not specified (-destination)")
	}

	objects, err := s.getObjectList(ctx, bucket, prefix, inputFile)
	if err != nil {
		return err
	}

	if objects == nil {
		return nil // User cancelled restore
	}

	for _, obj := range objects {
		s.summary.TotalFiles++
		if err := s.downloadObject(ctx, bucket, obj.Key, downloadLocation); err != nil {
			log.Printf("❌ Failed to download %s: %v", obj.Key, err)
			s.summary.FailedDownloads++
		}
		// Note: SuccessfulDownloads and SkippedFiles are incremented in downloadObject
	}

	// Combine split files
	if err := utils.CombineFiles(downloadLocation); err != nil {
		log.Printf("⚠️ Warning: Failed to combine files: %v", err)
		s.summary.Warnings++
	}

	s.printSummary()
	return nil
}

func (s *RestoreService) listBuckets(ctx context.Context) error {
	client := s3.NewFromConfig(s.cfg)
	result, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return err
	}

	fmt.Println("Available buckets:")
	for _, bucket := range result.Buckets {
		fmt.Printf("  %s\n", *bucket.Name)
	}
	return nil
}

func (s *RestoreService) getObjectList(ctx context.Context, bucket, prefix, inputFile string) ([]S3Object, error) {
	if inputFile != "" {
		return s.loadObjectsFromFile(inputFile)
	}
	return s.listObjects(ctx, bucket, prefix)
}

func (s *RestoreService) loadObjectsFromFile(inputFile string) ([]S3Object, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, err
	}

	var contents S3Contents
	if err := json.Unmarshal(data, &contents); err != nil {
		return nil, err
	}

	return contents.Contents, nil
}

func (s *RestoreService) listObjects(ctx context.Context, bucket, prefix string) ([]S3Object, error) {
	client := s3.NewFromConfig(s.cfg)

	input := &s3.ListObjectsV2Input{
		Bucket: &bucket,
	}
	if prefix != "" {
		input.Prefix = &prefix
	}

	result, err := client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, err
	}

	var objects []S3Object
	for _, obj := range result.Contents {
		objects = append(objects, S3Object{
			Key:          *obj.Key,
			Size:         *obj.Size,
			StorageClass: string(obj.StorageClass),
		})
	}

	// Save to file for future use
	if err := s.saveObjectsToFile(objects, "generated-restore-input.json"); err != nil {
		return nil, fmt.Errorf("❌ failed to save restore input file: %w", err)
	}

	fmt.Printf("\nGenerated: 'generated-restore-input.json'\n\n")
	fmt.Printf("Do you want to continue with restore, without editing generated input JSON? [y/N]: ")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Printf("Restore cancelled. Edit 'generated-restore-input.json' and run:\n")
		fmt.Printf("aws-s3-backup -mode restore -bucket %s -json generated-restore-input.json\n", bucket)
		return nil, nil
	}

	return objects, nil
}

func (s *RestoreService) saveObjectsToFile(objects []S3Object, filename string) error {
	contents := S3Contents{Contents: objects}
	data, err := json.MarshalIndent(contents, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (s *RestoreService) printSummary() {
	fmt.Printf("%s", "\n"+strings.Repeat("=", 50)+"\n")
	fmt.Printf("📊 RESTORE SUMMARY\n")
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")

	if s.summary.SuccessfulDownloads > 0 {
		fmt.Printf("✅ Successfully downloaded: %d\n", s.summary.SuccessfulDownloads)
	}

	if s.summary.SkippedFiles > 0 {
		fmt.Printf("⏭️ Skipped (already exists): %d\n", s.summary.SkippedFiles)
	}

	if s.summary.FailedDownloads > 0 {
		fmt.Printf("❌ Failed downloads: %d\n", s.summary.FailedDownloads)
	}

	if s.summary.Warnings > 0 {
		fmt.Printf("⚠️  Warnings: %d\n", s.summary.Warnings)
	}

	fmt.Printf("📁 Total files processed: %d\n", s.summary.TotalFiles)

	if s.summary.FailedDownloads == 0 {
		fmt.Printf("\n🎉 Restore completed successfully!\n")
	} else {
		fmt.Printf("\n⚠️  Restore completed with errors!\n")
	}
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")
}

func (s *RestoreService) downloadObject(ctx context.Context, bucket, key, downloadDir string) error {
	// Skip if already exists
	localPath := fmt.Sprintf("%s/%s", strings.TrimRight(downloadDir, "/"), key)
	if _, err := os.Stat(localPath); err == nil {
		log.Printf("⏭️ Skipping %s (already exists)", key)
		s.summary.SkippedFiles++
		return nil
	}

	// Create directory if needed
	dirPath := filepath.Dir(localPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	log.Printf("⬇️ Downloading: %s", key)
	if err := utils.DownloadFile(ctx, s.cfg, bucket, key, localPath); err != nil {
		return err
	}
	s.summary.SuccessfulDownloads++
	return nil
}
