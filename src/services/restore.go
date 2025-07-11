package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/config"
	"github.com/rtitz/aws-s3-backup/utils"
)

type RestoreService struct {
	cfg              aws.Config
	summary          *RestoreSummary
	downloadLocation string
}

type RestoreSummary struct {
	SuccessfulDownloads int
	FailedDownloads     int
	SkippedFiles        int
	Warnings            int
	TotalFiles          int
	TotalBytes          int64
	RestoreWaitTime     time.Duration
	ActualDownloadTime  time.Duration
	ProcessingTime      time.Duration
	TotalTime           time.Duration
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

func (s *RestoreService) ProcessRestore(ctx context.Context, bucket, prefix, inputFile, downloadLocation string, dryRun, skipDecompression bool, retrievalMode string, restoreExpiresAfterDays int32, autoRetryDownloadMinutes int, restoreWithoutConfirmation bool) error {
	startTime := time.Now()

	s.downloadLocation = downloadLocation // Store for later use
	fmt.Printf("\nMODE: RESTORE\n")
	if dryRun {
		fmt.Printf("REGION: DRY-RUN\n\n")
	} else {
		fmt.Printf("REGION: %s\n\n", s.cfg.Region)
	}

	if bucket == "" {
		if dryRun {
			return fmt.Errorf("❌ bucket parameter required for dry-run restore (use local directory path)")
		}
		return s.listBuckets(ctx)
	}

	if downloadLocation == "" {
		return fmt.Errorf("❌ download location not specified (-destination)")
	}

	var objects []S3Object
	var err error
	if dryRun {
		// In dry-run mode, treat bucket as local directory and scan for files
		objects, err = s.scanLocalDirectory(bucket, prefix)
	} else {
		objects, err = s.getObjectList(ctx, bucket, prefix, inputFile)
	}
	if err != nil {
		return err
	}

	if objects == nil {
		return nil // User cancelled restore
	}

	// Filter out objects that already have decompressed files
	filteredObjects := s.filterObjectsWithDecompressedFiles(objects, downloadLocation)
	if len(filteredObjects) < len(objects) {
		skippedCount := len(objects) - len(filteredObjects)
		log.Printf("⏭️ Skipping %d objects (decompressed files already exist)", skippedCount)
		s.summary.SkippedFiles += skippedCount
	}

	// Check if any files to be downloaded are encrypted and get password upfront
	var password string
	if s.hasEncryptedFiles(filteredObjects) {
		password, err = s.getDecryptionPassword()
		if err != nil {
			return err
		}
	}

	// Check for Glacier objects and handle restore if needed
	if !dryRun {
		if err := s.handleGlacierRestore(ctx, bucket, filteredObjects, retrievalMode, restoreExpiresAfterDays, restoreWithoutConfirmation); err != nil {
			return fmt.Errorf("❌ Glacier restore failed: %w", err)
		}
		
		// Auto-retry logic if specified
		if autoRetryDownloadMinutes > 0 {
			if err := s.waitForGlacierRestore(ctx, bucket, filteredObjects, int(autoRetryDownloadMinutes)); err != nil {
				return fmt.Errorf("❌ Auto-retry failed: %w", err)
			}
		}
	}

	for _, obj := range filteredObjects {
		s.summary.TotalFiles++
		if dryRun {
			log.Printf("⬇️ [DRY-RUN] Would download: %s (%s)", obj.Key, utils.FormatBytes(obj.Size))
			s.summary.SuccessfulDownloads++
			s.summary.TotalBytes += obj.Size
		} else {
			if err := s.downloadObject(ctx, bucket, obj.Key, downloadLocation, obj.Size); err != nil {
				log.Printf("❌ Failed to download %s: %v", obj.Key, err)
				s.summary.FailedDownloads++
			}
			// Note: SuccessfulDownloads and SkippedFiles are incremented in downloadObject
		}
	}

	// Track processing time (decryption + combination)
	processingStart := time.Now()

	// Decrypt encrypted files first (before combining)
	if password != "" || s.hasEncryptedFilesInDir(downloadLocation) {
		// If we don't have password yet (files existed locally), get it now
		if password == "" {
			password, err = s.getDecryptionPassword()
			if err != nil {
				return err
			}
		}
		if err := s.decryptFiles(downloadLocation, password, objects); err != nil {
			log.Printf("⚠️ Warning: Some files could not be decrypted: %v", err)
			s.summary.Warnings++
		}
	}

	// Combine split files (including decrypted ones)
	if err := utils.CombineFiles(downloadLocation); err != nil {
		log.Printf("⚠️ Warning: Failed to combine files: %v", err)
		s.summary.Warnings++
	}

	// Decompress tar.gz archives (unless skipped)
	if !skipDecompression {
		if err := s.decompressArchives(downloadLocation); err != nil {
			log.Printf("⚠️ Warning: Failed to decompress archives: %v", err)
			s.summary.Warnings++
		}
	} else {
		log.Printf("⏭️ Skipping archive decompression (--skipDecompression flag set)")
	}

	s.summary.ProcessingTime = time.Since(processingStart)
	s.summary.TotalTime = time.Since(startTime)
	s.printSummary(dryRun)
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
		// Get bucket region and info
		region, err := utils.GetBucketRegion(ctx, s.cfg, *bucket.Name)
		if err != nil {
			// If we can't get region info, show basic info
			fmt.Printf("  %s (region: unknown)\n", *bucket.Name)
			continue
		}

		regionInfo := utils.GetRegionInfo(region)
		gdprStatus := "⚠️  Non-GDPR"
		if regionInfo.GDPRCompliant {
			gdprStatus = "🔒 GDPR"
		}
		fmt.Printf("  %s (region: %s %s %s %s)\n", *bucket.Name, region, regionInfo.Flag, regionInfo.Country, gdprStatus)
	}

	fmt.Printf("\n💡 To restore from a bucket, specify it with the -bucket parameter:\n")
	fmt.Printf("   aws-s3-backup -mode restore -bucket BUCKET_NAME -destination /path/to/restore/\n\n")
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
	// Get bucket region and create region-specific config
	region, regionCfg, err := utils.GetBucketRegionWithConfig(ctx, s.cfg, bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to get bucket region: %w", err)
	}

	// Update service config for subsequent operations
	s.cfg = regionCfg

	client := s3.NewFromConfig(regionCfg)
	log.Printf("📁 Listing objects in bucket: %s (region: %s)", bucket, region)

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
		fmt.Printf("aws-s3-backup -mode restore -bucket %s -json generated-restore-input.json -destination %s\n", bucket, s.downloadLocation)
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

func (s *RestoreService) printSummary(dryRun bool) {
	fmt.Printf("%s", "\n"+strings.Repeat("=", 50)+"\n")
	if dryRun {
		fmt.Printf("📋 RESTORE SUMMARY (DRY-RUN)\n")
	} else {
		fmt.Printf("📊 RESTORE SUMMARY\n")
	}
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")

	if s.summary.SuccessfulDownloads > 0 {
		if dryRun {
			fmt.Printf("✅ Files that would be downloaded: %d\n", s.summary.SuccessfulDownloads)
		} else {
			fmt.Printf("✅ Successfully downloaded: %d\n", s.summary.SuccessfulDownloads)
		}
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
	fmt.Printf("💾 Total data processed: %s\n", utils.FormatBytes(s.summary.TotalBytes))

	if s.summary.RestoreWaitTime > 0 {
		if dryRun {
			fmt.Printf("⏱️  Restore wait time (simulated): %v\n", s.summary.RestoreWaitTime.Round(time.Millisecond))
		} else {
			fmt.Printf("⏱️  Restore wait time: %v\n", s.summary.RestoreWaitTime.Round(time.Millisecond))
		}
	}

	if dryRun {
		fmt.Printf("⏱️  Download time (simulated): %v\n", s.summary.ActualDownloadTime.Round(time.Millisecond))
	} else {
		fmt.Printf("⏱️  Download time: %v\n", s.summary.ActualDownloadTime.Round(time.Millisecond))
	}
	fmt.Printf("⏱️  Processing time: %v\n", s.summary.ProcessingTime.Round(time.Millisecond))
	fmt.Printf("⏱️  Total time: %v\n", s.summary.TotalTime.Round(time.Millisecond))

	if s.summary.FailedDownloads == 0 {
		if dryRun {
			fmt.Printf("\n🎉 Dry-run restore completed successfully!\n")
		} else {
			fmt.Printf("\n🎉 Restore completed successfully!\n")
		}
	} else {
		fmt.Printf("\n⚠️  Restore completed with errors!\n")
	}
	fmt.Printf("%s", strings.Repeat("=", 50)+"\n")
}

func (s *RestoreService) hasEncryptedFiles(objects []S3Object) bool {
	for _, obj := range objects {
		if strings.HasSuffix(obj.Key, "."+config.EncryptionExt) {
			return true
		}
	}
	return false
}

func (s *RestoreService) hasEncryptedFilesInDir(downloadDir string) bool {
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "."+config.EncryptionExt) {
			return true
		}
	}
	return false
}

func (s *RestoreService) getDecryptionPassword() (string, error) {
	fmt.Printf("🔐 Encrypted files detected. Enter decryption password: ")
	var password string
	fmt.Scanln(&password)
	if password == "" {
		return "", fmt.Errorf("❌ password required for encrypted files")
	}
	return password, nil
}

func (s *RestoreService) decryptFiles(downloadDir, password string, objects []S3Object) error {
	// Filter encrypted files from objects list, excluding those already decrypted
	var encryptedFiles []S3Object
	for _, obj := range objects {
		if strings.HasSuffix(obj.Key, "."+config.EncryptionExt) {
			// Check if decrypted version already exists
			decryptedKey := strings.TrimSuffix(obj.Key, "."+config.EncryptionExt)
			decryptedPath := filepath.Join(downloadDir, decryptedKey)
			if _, err := os.Stat(decryptedPath); err == nil {
				log.Printf("⏭️ Skipping decryption of %s (decrypted version already exists: %s)", obj.Key, decryptedKey)
				continue
			}
			encryptedFiles = append(encryptedFiles, obj)
		}
	}

	if len(encryptedFiles) == 0 {
		log.Printf("ℹ️ No encrypted files found in objects list")
		return nil
	}

	log.Printf("🔓 Found %d encrypted files to decrypt from objects list", len(encryptedFiles))

	for _, obj := range encryptedFiles {
		// Build local file path
		localPath := filepath.Join(downloadDir, obj.Key)

		// Calculate decrypted filename by removing .enc extension
		decryptedName := strings.TrimSuffix(obj.Key, "."+config.EncryptionExt)
		decryptedPath := filepath.Join(downloadDir, decryptedName)

		// Skip if decrypted version already exists
		if _, err := os.Stat(decryptedPath); err == nil {
			log.Printf("⏭️ Skipping decryption of %s (decrypted file already exists: %s)", obj.Key, decryptedName)
			continue
		}

		// Check if encrypted file exists locally
		if _, err := os.Stat(localPath); os.IsNotExist(err) {
			log.Printf("⚠️ Encrypted file not found locally: %s", obj.Key)
			continue
		}

		log.Printf("🔓 Decrypting: %s -> %s", obj.Key, decryptedName)

		for {
			_, err := utils.DecryptFile(localPath, password)
			if err == nil {
				// Decryption successful, remove encrypted file
				if err := os.Remove(localPath); err != nil {
					log.Printf("⚠️ Warning: Could not remove encrypted file %s: %v", obj.Key, err)
				}
				log.Printf("✅ Successfully decrypted: %s", decryptedName)
				break
			}

			// Decryption failed, ask for password or skip
			log.Printf("❌ Failed to decrypt %s: %v", obj.Key, err)
			fmt.Printf("Enter password for %s (or 'skip' to skip this file): ", obj.Key)
			var input string
			fmt.Scanln(&input)

			if strings.ToLower(input) == "skip" {
				log.Printf("⏭️ Skipping decryption of: %s", obj.Key)
				break
			}

			if input == "" {
				log.Printf("⏭️ Skipping decryption of: %s (empty password)", obj.Key)
				break
			}

			// Update password for all subsequent files
			password = input
		}
	}
	return nil
}

func (s *RestoreService) downloadObject(ctx context.Context, bucket, key, downloadDir string, size int64) error {
	// Skip if already exists
	localPath := fmt.Sprintf("%s/%s", strings.TrimRight(downloadDir, "/"), key)
	if _, err := os.Stat(localPath); err == nil {
		log.Printf("⏭️ Skipping %s (already exists)", key)
		s.summary.SkippedFiles++
		return nil
	}

	// Skip encrypted files if decrypted version already exists
	if strings.HasSuffix(key, "."+config.EncryptionExt) {
		decryptedKey := strings.TrimSuffix(key, "."+config.EncryptionExt)
		decryptedPath := fmt.Sprintf("%s/%s", strings.TrimRight(downloadDir, "/"), decryptedKey)
		if _, err := os.Stat(decryptedPath); err == nil {
			log.Printf("⏭️ Skipping %s (decrypted version already exists: %s)", key, decryptedKey)
			s.summary.SkippedFiles++
			return nil
		}
	}

	// Skip split files if combined file already exists
	partPattern := regexp.MustCompile(`^(.+)-part\d{5}(\.` + config.EncryptionExt + `)?$`)
	if matches := partPattern.FindStringSubmatch(filepath.Base(key)); len(matches) >= 2 {
		baseName := matches[1]
		// Build combined file path maintaining directory structure
		combinedKey := filepath.Join(filepath.Dir(key), baseName)
		combinedPath := fmt.Sprintf("%s/%s", strings.TrimRight(downloadDir, "/"), combinedKey)
		if _, err := os.Stat(combinedPath); err == nil {
			log.Printf("⏭️ Skipping %s (combined file already exists: %s)", key, baseName)
			s.summary.SkippedFiles++
			return nil
		}
	}

	// Skip HowToBuild.txt files if combined file already exists
	howToBuildPattern := regexp.MustCompile(`^(.+)-HowToBuild\.txt(\.` + config.EncryptionExt + `)?$`)
	if matches := howToBuildPattern.FindStringSubmatch(filepath.Base(key)); len(matches) >= 2 {
		baseName := matches[1]
		// Build combined file path maintaining directory structure
		combinedKey := filepath.Join(filepath.Dir(key), baseName)
		combinedPath := fmt.Sprintf("%s/%s", strings.TrimRight(downloadDir, "/"), combinedKey)
		if _, err := os.Stat(combinedPath); err == nil {
			log.Printf("⏭️ Skipping %s (combined file already exists: %s)", key, baseName)
			s.summary.SkippedFiles++
			return nil
		}
	}

	// Create directory if needed
	dirPath := filepath.Dir(localPath)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return err
	}

	// Check if file already exists locally (skip re-download)
	if _, err := os.Stat(localPath); err == nil {
		log.Printf("⏭️ Skipping download: %s (already exists locally)", key)
		return nil
	}
	
	log.Printf("⬇️ Downloading: %s (%s)", key, utils.FormatBytes(size))

	// Track actual download time
	downloadStart := time.Now()
	
	// Retry download with exponential backoff for network errors
	err := utils.RetryWithBackoff(ctx, func() error {
		return utils.DownloadFile(ctx, s.cfg, bucket, key, localPath)
	}, fmt.Sprintf("Download %s", key))
	
	if err != nil {
		return err
	}
	s.trackActualDownloadTime(time.Since(downloadStart))

	// Track downloaded bytes
	if size, err := utils.GetFileSize(localPath); err == nil {
		s.summary.TotalBytes += size
	}

	s.summary.SuccessfulDownloads++
	return nil
}

// scanLocalDirectory scans a local directory for files (dry-run mode)
func (s *RestoreService) scanLocalDirectory(localDir, prefix string) ([]S3Object, error) {
	var objects []S3Object

	if _, err := os.Stat(localDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("❌ local directory does not exist: %s", localDir)
	}

	err := filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Create relative path from local directory
		relPath, err := filepath.Rel(localDir, path)
		if err != nil {
			return err
		}

		// Apply prefix filter if specified
		if prefix != "" && !strings.HasPrefix(relPath, prefix) {
			return nil
		}

		objects = append(objects, S3Object{
			Key:          relPath,
			Size:         info.Size(),
			StorageClass: "STANDARD", // Default for dry-run
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan local directory: %w", err)
	}

	log.Printf("📁 [DRY-RUN] Found %d files in local directory: %s", len(objects), localDir)
	return objects, nil
}

// trackActualDownloadTime adds time spent on actual data transfer
func (s *RestoreService) trackActualDownloadTime(duration time.Duration) {
	s.summary.ActualDownloadTime += duration
}

// decompressArchives decompresses tar.gz files in the download directory
func (s *RestoreService) decompressArchives(downloadDir string) error {
	log.Printf("📎 Scanning for archives to decompress...")

	err := filepath.Walk(downloadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Check if file is a tar.gz archive
		if strings.HasSuffix(info.Name(), ".tar.gz") {
			// Check if decompressed version already exists
			baseName := strings.TrimSuffix(info.Name(), ".tar.gz")
			decompressedPath := filepath.Join(filepath.Dir(path), baseName)
			if _, err := os.Stat(decompressedPath); err == nil {
				log.Printf("⏭️ Skipping decompression of %s (already exists: %s)", info.Name(), baseName)
				return nil
			}

			log.Printf("📎 Decompressing: %s", info.Name())

			// Extract to same directory
			if err := utils.ExtractArchive(path, filepath.Dir(path)); err != nil {
				log.Printf("❌ Failed to decompress %s: %v", info.Name(), err)
				return nil // Continue with other files
			}

			// Remove the archive after successful extraction
			if err := os.Remove(path); err != nil {
				log.Printf("⚠️ Warning: Could not remove archive %s: %v", info.Name(), err)
			} else {
				log.Printf("✅ Successfully decompressed and removed: %s", info.Name())
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory for archives: %w", err)
	}

	log.Printf("✅ Archive decompression completed")
	return nil
}

// filterObjectsWithDecompressedFiles removes objects that already have final processed files
func (s *RestoreService) filterObjectsWithDecompressedFiles(objects []S3Object, downloadDir string) []S3Object {
	var filtered []S3Object
	for _, obj := range objects {
		if !s.finalFileExists(obj.Key, downloadDir) {
			filtered = append(filtered, obj)
		}
	}
	return filtered
}

// finalFileExists checks if the final processed version of a file already exists
func (s *RestoreService) finalFileExists(key, downloadDir string) bool {
	// Check for HowToBuild.txt files
	howToBuildPattern := regexp.MustCompile(`^(.+)-HowToBuild\.txt(\.enc)?$`)
	if matches := howToBuildPattern.FindStringSubmatch(filepath.Base(key)); len(matches) >= 2 {
		baseName := matches[1]
		
		// For HowToBuild files of tar.gz archives, check if decompressed version exists
		if strings.HasSuffix(baseName, ".tar.gz") {
			decompressedName := strings.TrimSuffix(baseName, ".tar.gz")
			decompressedPath := filepath.Join(downloadDir, filepath.Dir(key), decompressedName)
			if _, err := os.Stat(decompressedPath); err == nil {
				return true
			}
		}
		
		// Check if combined file exists
		combinedPath := filepath.Join(downloadDir, filepath.Dir(key), baseName)
		if _, err := os.Stat(combinedPath); err == nil {
			return true
		}
	}
	
	// Check for split parts (e.g., file-part00001, file.tar.gz-part00001)
	partPattern := regexp.MustCompile(`^(.+)-part\d{5}(\.enc)?$`)
	if matches := partPattern.FindStringSubmatch(filepath.Base(key)); len(matches) >= 2 {
		baseName := matches[1]
		
		// For split tar.gz files, check if decompressed version exists
		if strings.HasSuffix(baseName, ".tar.gz") {
			decompressedName := strings.TrimSuffix(baseName, ".tar.gz")
			decompressedPath := filepath.Join(downloadDir, filepath.Dir(key), decompressedName)
			if _, err := os.Stat(decompressedPath); err == nil {
				return true
			}
		}
		
		// For other split files, check if combined file exists
		combinedPath := filepath.Join(downloadDir, filepath.Dir(key), baseName)
		if _, err := os.Stat(combinedPath); err == nil {
			return true
		}
	}
	
	// For tar.gz files, check if decompressed version exists
	if strings.HasSuffix(key, ".tar.gz") {
		baseName := strings.TrimSuffix(filepath.Base(key), ".tar.gz")
		decompressedPath := filepath.Join(downloadDir, filepath.Dir(key), baseName)
		if _, err := os.Stat(decompressedPath); err == nil {
			return true
		}
		// Also check if the archive exists but hasn't been decompressed yet
		archivePath := filepath.Join(downloadDir, key)
		if _, err := os.Stat(archivePath); err == nil {
			// Archive exists but not decompressed - don't skip download
			return false
		}
	}
	
	// For encrypted tar.gz files, check if decompressed version exists
	if strings.HasSuffix(key, ".tar.gz.enc") {
		baseName := strings.TrimSuffix(filepath.Base(key), ".tar.gz.enc")
		decompressedPath := filepath.Join(downloadDir, filepath.Dir(key), baseName)
		if _, err := os.Stat(decompressedPath); err == nil {
			return true
		}
	}
	
	// For regular files, check if the file itself exists
	localPath := filepath.Join(downloadDir, key)
	if _, err := os.Stat(localPath); err == nil {
		return true
	}
	
	return false
}
// handleGlacierRestore handles restore requests for Glacier objects
func (s *RestoreService) handleGlacierRestore(ctx context.Context, bucket string, objects []S3Object, retrievalMode string, restoreExpiresAfterDays int32, restoreWithoutConfirmation bool) error {
	var glacierObjects []S3Object
	
	// Find objects that need restore
	for _, obj := range objects {
		if s.isGlacierStorageClass(obj.StorageClass) {
			glacierObjects = append(glacierObjects, obj)
		}
	}
	
	if len(glacierObjects) == 0 {
		return nil // No Glacier objects
	}
	
	log.Printf("🧊 Found %d objects in Glacier storage classes that may need restore", len(glacierObjects))
	
	// Check restore status for each object
	var needsRestore []S3Object
	var available []S3Object
	
	for _, obj := range glacierObjects {
		restored, err := utils.CheckObjectRestoreStatus(ctx, s.cfg, bucket, obj.Key)
		if err != nil {
			log.Printf("⚠️ Could not check restore status for %s: %v", obj.Key, err)
			needsRestore = append(needsRestore, obj)
			continue
		}
		
		if restored {
			available = append(available, obj)
		} else {
			needsRestore = append(needsRestore, obj)
		}
	}
	
	if len(available) > 0 {
		log.Printf("✅ %d objects already restored and available", len(available))
	}
	
	if len(needsRestore) == 0 {
		return nil // All objects are already restored
	}
	
	log.Printf("🔄 %d objects need to be restored from Glacier", len(needsRestore))
	
	// Show objects that need restore
	for _, obj := range needsRestore {
		log.Printf("  - %s (%s, %s)", obj.Key, obj.StorageClass, utils.FormatBytes(obj.Size))
	}
	
	// Ask for confirmation unless restoreWithoutConfirmation is set
	if !restoreWithoutConfirmation {
		fmt.Printf("\nDo you want to restore these %d objects from Glacier storage? [y/N]: ", len(needsRestore))
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
			return fmt.Errorf("restore cancelled by user")
		}
	}
	
	// Initiate restore for objects that need it
	log.Printf("🔄 Initiating restore for %d objects (mode: %s, expires after: %d days)", len(needsRestore), retrievalMode, restoreExpiresAfterDays)
	
	for _, obj := range needsRestore {
		log.Printf("🔄 Restoring: %s", obj.Key)
		if err := utils.RestoreObject(ctx, s.cfg, bucket, obj.Key, retrievalMode, restoreExpiresAfterDays); err != nil {
			// Check if restore is already in progress
			if strings.Contains(err.Error(), "RestoreAlreadyInProgress") {
				log.Printf("ℹ️ Restore already in progress for: %s", obj.Key)
			} else {
				log.Printf("❌ Failed to initiate restore for %s: %v", obj.Key, err)
			}
		} else {
			log.Printf("✅ Restore initiated for: %s", obj.Key)
		}
	}
	
	// Inform user about wait time
	switch retrievalMode {
	case "expedited":
		log.Printf("⏰ Expedited retrieval typically takes 1-5 minutes for Glacier Flexible Retrieval")
	case "bulk":
		log.Printf("⏰ Bulk retrieval typically takes 5-12 hours for Glacier, up to 48 hours for Deep Archive")
	default:
		log.Printf("⏰ Standard retrieval typically takes 3-5 hours for Glacier, up to 12 hours for Deep Archive")
	}
	
	return nil // Don't return error, let the process continue
}

// isGlacierStorageClass checks if storage class requires restore
func (s *RestoreService) isGlacierStorageClass(storageClass string) bool {
	switch storageClass {
	case "GLACIER", "GLACIER_FLEXIBLE_RETRIEVAL", "DEEP_ARCHIVE", "GLACIER_IR":
		return true
	default:
		return false
	}
}
// waitForGlacierRestore waits for Glacier objects to be restored with auto-retry
func (s *RestoreService) waitForGlacierRestore(ctx context.Context, bucket string, objects []S3Object, retryMinutes int) error {
	var glacierObjects []S3Object
	
	// Find Glacier objects that need checking
	for _, obj := range objects {
		if s.isGlacierStorageClass(obj.StorageClass) {
			glacierObjects = append(glacierObjects, obj)
		}
	}
	
	if len(glacierObjects) == 0 {
		return nil
	}
	
	log.Printf("🔄 Auto-retry enabled: checking restore status every %d minutes", retryMinutes)
	
	retryInterval := time.Duration(retryMinutes) * time.Minute
	startTime := time.Now()
	
	for {
		var stillWaiting []S3Object
		
		// Check status of all Glacier objects
		for _, obj := range glacierObjects {
			restored, err := utils.CheckObjectRestoreStatus(ctx, s.cfg, bucket, obj.Key)
			if err != nil {
				log.Printf("⚠️ Could not check restore status for %s: %v", obj.Key, err)
				stillWaiting = append(stillWaiting, obj)
				continue
			}
			
			if !restored {
				stillWaiting = append(stillWaiting, obj)
			} else {
				log.Printf("✅ Object restored and available: %s", obj.Key)
			}
		}
		
		// If all objects are restored, break the loop
		if len(stillWaiting) == 0 {
			log.Printf("🎉 All Glacier objects are now restored and available for download")
			s.summary.RestoreWaitTime = time.Since(startTime)
			return nil
		}
		
		// Show progress
		restored := len(glacierObjects) - len(stillWaiting)
		log.Printf("📊 Restore progress: %d/%d objects ready (%d still waiting)", restored, len(glacierObjects), len(stillWaiting))
		
		// Wait before next check
		log.Printf("⏰ Waiting %d minutes before next check...", retryMinutes)
		time.Sleep(retryInterval)
		
		// Update the list for next iteration
		glacierObjects = stillWaiting
	}
}