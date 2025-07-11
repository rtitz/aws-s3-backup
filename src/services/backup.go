package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/config"
	"github.com/rtitz/aws-s3-backup/utils"
)

type BackupService struct {
	cfg     aws.Config
	summary *BackupSummary
}

type BackupSummary struct {
	SuccessfulUploads int
	FailedUploads     int
	SkippedFiles      int
	Warnings          int
	TotalFiles        int
	TotalBytes        int64
	PreparationTime   time.Duration
	UploadTime        time.Duration
	TotalTime         time.Duration
}

func NewBackupService(cfg aws.Config) *BackupService {
	return &BackupService{
		cfg:     cfg,
		summary: &BackupSummary{},
	}
}

func (s *BackupService) ProcessBackup(ctx context.Context, inputFile string, dryRun bool) error {
	startTime := time.Now()
	
	fmt.Printf("\nMODE: BACKUP\n")
	if dryRun {
		fmt.Printf("REGION: DRY-RUN\n\n")
	} else {
		fmt.Printf("REGION: %s\n\n", s.cfg.Region)
	}

	tasks, err := config.LoadTasks(inputFile)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Validate buckets exist before processing (skip in dry-run)
	if !dryRun {
		if err := s.validateBuckets(ctx, tasks); err != nil {
			return err
		}
	}

	for _, task := range tasks {
		if err := s.processTask(ctx, task, dryRun); err != nil {
			return fmt.Errorf("❌ failed to process task: %w", err)
		}
	}

	if err := s.uploadAdditionalFiles(ctx, tasks, inputFile, dryRun); err != nil {
		return err
	}

	s.summary.TotalTime = time.Since(startTime)
	s.printSummary(dryRun)
	return nil
}

func (s *BackupService) processTask(ctx context.Context, task config.Task, dryRun bool) error {
	// Validate encryption password if provided
	if err := utils.ValidateEncryptionPassword(task.EncryptionSecret); err != nil {
		return err
	}

	splitMB, err := config.ParseArchiveSplitMB(task.ArchiveSplitEachMB)
	if err != nil {
		return err
	}

	storageClass := config.ParseStorageClass(task.StorageClass)
	cleanupTmp := config.ParseCleanupFlag(task.CleanupTmpStorage)

	for _, contentPath := range task.Content {
		if err := s.processContent(ctx, task, contentPath, splitMB, storageClass, cleanupTmp, dryRun); err != nil {
			s.summary.FailedUploads++
			return fmt.Errorf("failed to process content %s: %w", contentPath, err)
		}
	}

	return nil
}

func (s *BackupService) processContent(ctx context.Context, task config.Task, contentPath string, splitMB int64, storageClass types.StorageClass, cleanupTmp bool, dryRun bool) error {
	prepStart := time.Now()
	
	if err := os.MkdirAll(task.TmpStorageToBuildArchives, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	archiveName := filepath.Base(contentPath)
	archivePath := filepath.Join(task.TmpStorageToBuildArchives, archiveName)

	fullArchivePath := archivePath + "." + config.ArchiveExtension
	if err := utils.CreateArchive([]string{contentPath}, fullArchivePath); err != nil {
		return fmt.Errorf("failed to build archive: %w", err)
	}

	parts, err := s.prepareParts(fullArchivePath, splitMB, task.EncryptionSecret)
	if err != nil {
		return fmt.Errorf("failed to prepare parts: %w", err)
	}

	s3Path := s.buildS3Path(task, contentPath)
	
	// Track preparation time
	s.summary.PreparationTime += time.Since(prepStart)
	
	if err := s.uploadParts(ctx, parts, task.S3Bucket, s3Path, storageClass, dryRun); err != nil {
		return fmt.Errorf("failed to upload parts: %w", err)
	}

	if dryRun {
		if cleanupTmp {
			log.Printf("🧽 [DRY-RUN] Skipping cleanup of temporary files - files kept for inspection")
		}
	} else if cleanupTmp {
		s.cleanupFiles(parts)
	}

	return nil
}

func (s *BackupService) prepareParts(archivePath string, splitMB int64, encryptionSecret string) ([]string, error) {
	parts, err := utils.SplitFile(archivePath, splitMB)
	if err != nil {
		return nil, err
	}

	if len(parts) > 1 {
		os.Remove(archivePath)
		// Create simple how-to file
		howToFile := archivePath + "-HowToBuild.txt"
		err := os.WriteFile(howToFile, []byte("Use 'cat parts > combined' to rebuild"), 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to create how-to file: %w", err)
		}
		parts = append(parts, howToFile)
	}

	if encryptionSecret != "" {
		return s.encryptParts(parts, encryptionSecret)
	}

	return parts, nil
}

func (s *BackupService) encryptParts(parts []string, secret string) ([]string, error) {
	var encryptedParts []string
	for _, part := range parts {
		encryptedFile, err := utils.EncryptFile(part, secret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt %s: %w", part, err)
		}
		os.Remove(part)
		encryptedParts = append(encryptedParts, encryptedFile)
	}
	return encryptedParts, nil
}

func (s *BackupService) buildS3Path(task config.Task, contentPath string) string {
	archivePath := filepath.Dir(contentPath)
	trimmedPath := utils.TrimPathPrefix(archivePath, task.TrimBeginningOfPathInS3)
	
	if task.S3Prefix == "" {
		return trimmedPath
	}
	return utils.NormalizePath(task.S3Prefix) + "/" + trimmedPath
}

func (s *BackupService) uploadParts(ctx context.Context, parts []string, bucket, s3Path string, storageClass types.StorageClass, dryRun bool) error {
	uploadStart := time.Now()
	defer func() {
		s.summary.UploadTime += time.Since(uploadStart)
	}()
	
	for i, part := range parts {
		_, err := utils.GetFileChecksum(part)
		if err != nil {
			return fmt.Errorf("failed to get checksum: %w", err)
		}
		size, err := utils.GetFileSize(part)
		if err != nil {
			return fmt.Errorf("failed to get size: %w", err)
		}
		sizeFloat := float64(size) / (1024 * 1024) // MB
		unit := "MB"

		s3Key := s3Path + filepath.Base(part)
		s.summary.TotalFiles++
		
		// Check if object already exists in S3 - REQUIRED for safety
		if !dryRun {
			var exists bool
			err := utils.RetryWithBackoff(ctx, func() error {
				var checkErr error
				exists, checkErr = utils.CheckObjectExists(ctx, s.cfg, bucket, s3Key)
				return checkErr
			}, fmt.Sprintf("Check existence of %s", s3Key))
			
			if err != nil {
				s.summary.FailedUploads++
				return fmt.Errorf("❌ Cannot verify object existence for %s: %w. Upload aborted to prevent overwriting existing data", s3Key, err)
			}
			if exists {
				log.Printf("⏭️ Skipping (%d/%d): %s (already exists in S3)", i+1, len(parts), filepath.Base(part))
				s.summary.SkippedFiles++
				continue
			}
		}
		
		s.summary.TotalBytes += size
		if dryRun {
			log.Printf("⬆️  [DRY-RUN] Would upload (%d/%d): %s (%.2f %s) to s3://%s/%s", i+1, len(parts), part, sizeFloat, unit, bucket, s3Key)
			s.summary.SuccessfulUploads++
		} else {
			log.Printf("⬆️ Uploading (%d/%d): %s (%.2f %s)", i+1, len(parts), part, sizeFloat, unit)
			
			// Retry upload with exponential backoff for network errors
			err := utils.RetryWithBackoff(ctx, func() error {
				return utils.UploadFile(ctx, s.cfg, part, bucket, s3Key, storageClass)
			}, fmt.Sprintf("Upload %s", filepath.Base(part)))
			
			if err != nil {
				s.summary.FailedUploads++
				return fmt.Errorf("❌ failed to upload %s: %w", part, err)
			}
			log.Printf("✅ Upload successful: %s", filepath.Base(part))
			s.summary.SuccessfulUploads++
		}
	}
	return nil
}

func (s *BackupService) uploadAdditionalFiles(ctx context.Context, tasks []config.Task, inputFile string, dryRun bool) error {
	if len(tasks) == 0 {
		return nil
	}

	bucket := tasks[0].S3Bucket
	var prefix string
	if tasks[0].S3Prefix == "" {
		prefix = ""
	} else {
		prefix = filepath.Clean(tasks[0].S3Prefix) + "/"
	}

	// Create sanitized version of input.json without encryption secrets
	sanitizedFile, err := s.createSanitizedInputFile(inputFile, tasks)
	if err != nil {
		return fmt.Errorf("failed to create sanitized input file: %w", err)
	}
	defer os.Remove(sanitizedFile)

	files := []string{sanitizedFile}

	for _, file := range files {
		_, err := utils.GetFileChecksum(file)
		if err != nil {
			return fmt.Errorf("failed to get file info for %s: %w", file, err)
		}

		size, err := utils.GetFileSize(file)
		if err != nil {
			return fmt.Errorf("failed to get file size for %s: %w", file, err)
		}

		s3Key := prefix + filepath.Base(inputFile) // Use original filename for S3 key
		s.summary.TotalFiles++
		
		// Check if additional file already exists in S3 - REQUIRED for safety
		if !dryRun {
			var exists bool
			err := utils.RetryWithBackoff(ctx, func() error {
				var checkErr error
				exists, checkErr = utils.CheckObjectExists(ctx, s.cfg, bucket, s3Key)
				return checkErr
			}, fmt.Sprintf("Check existence of %s", s3Key))
			
			if err != nil {
				s.summary.FailedUploads++
				return fmt.Errorf("❌ Cannot verify object existence for %s: %w. Upload aborted to prevent overwriting existing data", s3Key, err)
			}
			if exists {
				log.Printf("⏭️ Skipping additional file: %s (already exists in S3)", filepath.Base(inputFile))
				s.summary.SkippedFiles++
				continue
			}
		}
		
		s.summary.TotalBytes += size
		if dryRun {
			log.Printf("⬆️  [DRY-RUN] Would upload additional file: %s to s3://%s/%s", filepath.Base(inputFile), bucket, s3Key)
			s.summary.SuccessfulUploads++
		} else {
			log.Printf("⬆️ Uploading additional file: %s", filepath.Base(inputFile))
			
			// Retry upload with exponential backoff for network errors
			err := utils.RetryWithBackoff(ctx, func() error {
				return utils.UploadFile(ctx, s.cfg, file, bucket, s3Key, types.StorageClassStandard)
			}, fmt.Sprintf("Upload additional file %s", filepath.Base(inputFile)))
			
			if err != nil {
				s.summary.FailedUploads++
				return fmt.Errorf("❌ failed to upload additional file %s: %w", filepath.Base(inputFile), err)
			}
			log.Printf("✅ Additional file uploaded: %s", filepath.Base(inputFile))
			s.summary.SuccessfulUploads++
		}
	}

	log.Println("Additional files uploaded successfully")
	return nil
}

func (s *BackupService) cleanupFiles(files []string) {
	for _, file := range files {
		os.Remove(file)
	}
}

func (s *BackupService) validateBuckets(ctx context.Context, tasks []config.Task) error {
	buckets := make(map[string]bool)
	for _, task := range tasks {
		buckets[task.S3Bucket] = true
	}

	for bucket := range buckets {
		region, updatedCfg, err := utils.ValidateBucketExistsWithRegion(ctx, s.cfg, bucket)
		if err != nil {
			return fmt.Errorf("❌ S3 bucket '%s' does not exist or is not accessible: %w", bucket, err)
		}
		// Update service config with correct region
		s.cfg = updatedCfg
		regionInfo := utils.GetRegionInfo(region)
		gdprStatus := "⚠️  Non-GDPR"
		if regionInfo.GDPRCompliant {
			gdprStatus = "🔒 GDPR"
		}
		log.Printf("✅ S3 bucket validated: %s (region: %s %s %s %s)", bucket, region, regionInfo.Flag, regionInfo.Country, gdprStatus)
	}
	return nil
}

func (s *BackupService) createSanitizedInputFile(inputFile string, tasks []config.Task) (string, error) {
	// Create sanitized tasks with empty encryption secrets
	sanitizedTasks := make([]config.Task, len(tasks))
	for i, task := range tasks {
		sanitizedTasks[i] = task
		sanitizedTasks[i].EncryptionSecret = "" // Remove encryption secret
	}

	// Create temporary file
	tempFile := inputFile + ".sanitized.tmp"
	data, err := json.MarshalIndent(map[string][]config.Task{"tasks": sanitizedTasks}, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}

func (s *BackupService) printSummary(dryRun bool) {
	fmt.Printf("%s", "\n"+strings.Repeat("=", 50)+"\n")
	if dryRun {
		fmt.Printf("📋 BACKUP SUMMARY (DRY-RUN)\n")
	} else {
		fmt.Printf("📊 BACKUP SUMMARY\n")
	}
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")

	if s.summary.SuccessfulUploads > 0 {
		if dryRun {
			fmt.Printf("✅ Files that would be uploaded: %d\n", s.summary.SuccessfulUploads)
		} else {
			fmt.Printf("✅ Successfully uploaded: %d\n", s.summary.SuccessfulUploads)
		}
	}

	if s.summary.SkippedFiles > 0 {
		fmt.Printf("⏭️ Skipped (already exists): %d\n", s.summary.SkippedFiles)
	}

	if s.summary.FailedUploads > 0 {
		fmt.Printf("❌ Failed uploads: %d\n", s.summary.FailedUploads)
	}

	if s.summary.Warnings > 0 {
		fmt.Printf("⚠️  Warnings: %d\n", s.summary.Warnings)
	}

	fmt.Printf("📁 Total files processed: %d\n", s.summary.TotalFiles)
	fmt.Printf("💾 Total data processed: %s\n", utils.FormatBytes(s.summary.TotalBytes))
	fmt.Printf("⏱️  Preparation time: %v\n", s.summary.PreparationTime.Round(time.Millisecond))
	if dryRun {
		fmt.Printf("⏱️  Upload time (simulated): %v\n", s.summary.UploadTime.Round(time.Millisecond))
	} else {
		fmt.Printf("⏱️  Upload time: %v\n", s.summary.UploadTime.Round(time.Millisecond))
	}
	fmt.Printf("⏱️  Total time: %v\n", s.summary.TotalTime.Round(time.Millisecond))

	if s.summary.FailedUploads == 0 {
		if dryRun {
			fmt.Printf("\n🎉 Dry-run completed successfully!\n")
		} else {
			fmt.Printf("\n🎉 Backup completed successfully!\n")
		}
	} else {
		fmt.Printf("\n⚠️  Backup completed with errors!\n")
	}
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")
}
