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
	s.downloadLocation = downloadLocation // Store for later use
	fmt.Printf("\nMODE: RESTORE\n")
	fmt.Printf("REGION: %s\n\n", s.cfg.Region)

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

	// Check if any files to be downloaded are encrypted and get password upfront
	var password string
	if s.hasEncryptedFiles(objects) {
		password, err = s.getDecryptionPassword()
		if err != nil {
			return err
		}
	}

	for _, obj := range objects {
		s.summary.TotalFiles++
		if err := s.downloadObject(ctx, bucket, obj.Key, downloadLocation); err != nil {
			log.Printf("❌ Failed to download %s: %v", obj.Key, err)
			s.summary.FailedDownloads++
		}
		// Note: SuccessfulDownloads and SkippedFiles are incremented in downloadObject
	}

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
		// Get bucket region and info
		region, err := utils.ValidateBucketExistsWithRegion(ctx, s.cfg, *bucket.Name)
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

func (s *RestoreService) downloadObject(ctx context.Context, bucket, key, downloadDir string) error {
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

	log.Printf("⬇️ Downloading: %s", key)
	if err := utils.DownloadFile(ctx, s.cfg, bucket, key, localPath); err != nil {
		return err
	}
	s.summary.SuccessfulDownloads++
	return nil
}
