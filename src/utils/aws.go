package utils

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	config_app "github.com/rtitz/aws-s3-backup/config"
)

// AWS session and configuration management

// CreateAWSSession creates and validates AWS configuration
func CreateAWSSession(ctx context.Context, profile, region string) (aws.Config, error) {
	cfg, err := loadAWSConfig(ctx, profile, region)
	if err != nil {
		return aws.Config{}, err
	}

	if err := validateAWSCredentials(ctx, cfg); err != nil {
		return aws.Config{}, err
	}

	return cfg, nil
}

// loadAWSConfig loads AWS configuration with profile and region
func loadAWSConfig(ctx context.Context, profile, region string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("❌ failed to load AWS config: %w", err)
	}
	return cfg, nil
}

// validateAWSCredentials tests AWS credentials by listing buckets
func validateAWSCredentials(ctx context.Context, cfg aws.Config) error {
	client := s3.NewFromConfig(cfg)
	_, err := client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return fmt.Errorf("❌ AWS credentials validation failed: %w", err)
	}
	return nil
}

// S3 file operations

// UploadFile uploads a file to S3 with specified storage class
func UploadFile(ctx context.Context, cfg aws.Config, filePath, bucket, key string, storageClass types.StorageClass) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer file.Close()

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       &bucket,
		Key:          &key,
		Body:         file,
		StorageClass: storageClass,
	})
	
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}
	
	return nil
}

// DownloadFile downloads a file from S3
func DownloadFile(ctx context.Context, cfg aws.Config, bucket, key, filePath string) error {
	regionCfg, err := getRegionSpecificConfig(ctx, cfg, bucket)
	if err != nil {
		regionCfg = cfg // Fallback to original config
	}

	s3Object, err := getS3Object(ctx, regionCfg, bucket, key)
	if err != nil {
		return err
	}
	defer s3Object.Body.Close()

	return saveObjectToFile(s3Object.Body, filePath)
}

// getRegionSpecificConfig gets AWS config for bucket's region
func getRegionSpecificConfig(ctx context.Context, cfg aws.Config, bucket string) (aws.Config, error) {
	_, regionCfg, err := GetBucketRegionWithConfig(ctx, cfg, bucket)
	return regionCfg, err
}

// getS3Object retrieves an object from S3
func getS3Object(ctx context.Context, cfg aws.Config, bucket, key string) (*s3.GetObjectOutput, error) {
	client := s3.NewFromConfig(cfg)
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get S3 object: %w", err)
	}
	return result, nil
}

// saveObjectToFile saves S3 object body to local file
func saveObjectToFile(body io.ReadCloser, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, body)
	if err != nil {
		return fmt.Errorf("failed to save object to file: %w", err)
	}
	
	return nil
}

// CheckObjectExists checks if an object exists in S3
func CheckObjectExists(ctx context.Context, cfg aws.Config, bucket, key string) (bool, error) {
	regionCfg, err := getRegionSpecificConfig(ctx, cfg, bucket)
	if err != nil {
		regionCfg = cfg
	}

	client := s3.NewFromConfig(regionCfg)
	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		if isNotFoundError(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence: %w", err)
	}

	return true, nil
}

// isNotFoundError checks if error indicates object not found
func isNotFoundError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "NotFound") || strings.Contains(errStr, "404")
}

// Glacier restore operations

// RestoreObject initiates restore from Glacier storage classes
func RestoreObject(ctx context.Context, cfg aws.Config, bucket, key string, retrievalMode string, restoreExpiresAfterDays int32) error {
	regionCfg, err := getRegionSpecificConfig(ctx, cfg, bucket)
	if err != nil {
		regionCfg = cfg
	}

	client := s3.NewFromConfig(regionCfg)
	tier := mapRetrievalModeToTier(retrievalMode)

	_, err = client.RestoreObject(ctx, &s3.RestoreObjectInput{
		Bucket: &bucket,
		Key:    &key,
		RestoreRequest: &types.RestoreRequest{
			Days: &restoreExpiresAfterDays,
			GlacierJobParameters: &types.GlacierJobParameters{
				Tier: tier,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to initiate restore: %w", err)
	}
	
	return nil
}

// mapRetrievalModeToTier converts string to AWS Tier type
func mapRetrievalModeToTier(retrievalMode string) types.Tier {
	switch strings.ToLower(retrievalMode) {
	case "standard":
		return types.TierStandard
	case "expedited":
		return types.TierExpedited
	case "bulk":
		return types.TierBulk
	default:
		return types.TierBulk
	}
}

// CheckObjectRestoreStatus checks if an object is restored and available for download
func CheckObjectRestoreStatus(ctx context.Context, cfg aws.Config, bucket, key string) (bool, error) {
	regionCfg, err := getRegionSpecificConfig(ctx, cfg, bucket)
	if err != nil {
		regionCfg = cfg
	}

	client := s3.NewFromConfig(regionCfg)
	result, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check object status: %w", err)
	}

	return isObjectRestored(result), nil
}

// isObjectRestored checks if Glacier object is restored
func isObjectRestored(result *s3.HeadObjectOutput) bool {
	// Check if object is in Glacier storage class
	if isGlacierStorageClass(result.StorageClass) {
		return checkGlacierRestoreStatus(result.Restore)
	}
	
	return true // Object is in Standard storage or already restored
}

// isGlacierStorageClass checks if storage class is Glacier type
func isGlacierStorageClass(storageClass types.StorageClass) bool {
	glacierClasses := []types.StorageClass{
		types.StorageClassGlacier,
		types.StorageClassDeepArchive,
		types.StorageClassGlacierIr,
	}
	
	for _, class := range glacierClasses {
		if storageClass == class {
			return true
		}
	}
	return false
}

// checkGlacierRestoreStatus parses restore status string
func checkGlacierRestoreStatus(restore *string) bool {
	if restore == nil {
		return false // Not restored
	}

	restoreStatus := *restore
	if strings.Contains(restoreStatus, `ongoing-request="true"`) {
		return false // Restore in progress
	}
	if strings.Contains(restoreStatus, `ongoing-request="false"`) {
		return true // Restore completed
	}
	
	return false
}

// Bucket management operations

// ValidateBucketExistsWithRegion checks if bucket exists and returns its region and updated config
func ValidateBucketExistsWithRegion(ctx context.Context, cfg aws.Config, bucket string) (string, aws.Config, error) {
	client := s3.NewFromConfig(cfg)

	// First validate bucket exists
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	
	if err != nil {
		return handleBucketCreation(ctx, cfg, bucket)
	}

	// Get existing bucket region
	return getBucketRegionAndConfig(ctx, cfg, bucket)
}

// handleBucketCreation manages bucket creation workflow
func handleBucketCreation(ctx context.Context, cfg aws.Config, bucket string) (string, aws.Config, error) {
	fmt.Printf("\n❌ S3 bucket '%s' does not exist.\n", bucket)
	
	if !confirmBucketCreation() {
		return "", aws.Config{}, fmt.Errorf("bucket '%s' does not exist", bucket)
	}

	selectedRegion := selectBucketRegion()
	if !isValidRegion(selectedRegion) {
		return "", aws.Config{}, fmt.Errorf("invalid region: %s", selectedRegion)
	}

	fmt.Printf("\n🏗️ Creating bucket '%s' in region %s...\n", bucket, selectedRegion)
	if err := CreateBucket(ctx, cfg, bucket, selectedRegion); err != nil {
		handleBucketCreationError(bucket, err)
		return "", aws.Config{}, fmt.Errorf("bucket creation failed: %w", err)
	}

	fmt.Printf("✅ Bucket '%s' created successfully in region %s\n", bucket, selectedRegion)
	
	updatedCfg := cfg.Copy()
	updatedCfg.Region = selectedRegion
	
	return selectedRegion, updatedCfg, nil
}

// confirmBucketCreation asks user to confirm bucket creation
func confirmBucketCreation() bool {
	fmt.Printf("Do you want to create it? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

// selectBucketRegion prompts user to select region for new bucket
func selectBucketRegion() string {
	showAvailableRegions()
	
	fmt.Printf("\nEnter region code for bucket creation [%s]: ", config_app.DefaultAWSRegion)
	var selectedRegion string
	fmt.Scanln(&selectedRegion)
	
	if selectedRegion == "" {
		selectedRegion = config_app.DefaultAWSRegion
	}
	
	return selectedRegion
}

// showAvailableRegions displays all available AWS regions
func showAvailableRegions() {
	fmt.Printf("\nAvailable regions:\n")
	regions := GetAllRegions()
	for _, region := range regions {
		regionInfo := GetRegionInfo(region)
		gdprStatus := "⚠️  Non-GDPR"
		if regionInfo.GDPRCompliant {
			gdprStatus = "🔒 GDPR"
		}
		fmt.Printf("  %s (%s %s %s)\n", region, regionInfo.Flag, regionInfo.Country, gdprStatus)
	}
}

// handleBucketCreationError provides helpful error messages
func handleBucketCreationError(bucket string, err error) {
	fmt.Printf("❌ Bucket creation failed: %v\n", err)
	
	errorMessages := map[string]string{
		"BucketAlreadyExists": fmt.Sprintf("💡 The bucket name '%s' is already taken globally. Try a different name.", bucket),
		"InvalidBucketName":   "💡 Invalid bucket name. Use lowercase letters, numbers, and hyphens only.",
		"AccessDenied":        "💡 Insufficient permissions. Check your IAM policy includes s3:CreateBucket.",
		"TooManyBuckets":      "💡 Account bucket limit reached (100 buckets max). Delete unused buckets.",
	}
	
	for errorType, message := range errorMessages {
		if strings.Contains(err.Error(), errorType) {
			fmt.Printf("%s\n", message)
			break
		}
	}
}

// getBucketRegionAndConfig gets existing bucket's region and config
func getBucketRegionAndConfig(ctx context.Context, cfg aws.Config, bucket string) (string, aws.Config, error) {
	client := s3.NewFromConfig(cfg)
	result, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return "", aws.Config{}, fmt.Errorf("failed to get bucket location: %w", err)
	}

	// Handle default region (us-east-1 returns empty string)
	region := string(result.LocationConstraint)
	if region == "" {
		region = config_app.DefaultAWSRegion
	}

	return region, cfg, nil
}

// CreateBucket creates an S3 bucket with security best practices
func CreateBucket(ctx context.Context, cfg aws.Config, bucketName, region string) error {
	client := s3.NewFromConfig(cfg)

	if err := createS3Bucket(ctx, client, bucketName, region); err != nil {
		return err
	}

	// Use region-specific client for subsequent operations
	if region != cfg.Region {
		regionCfg := cfg.Copy()
		regionCfg.Region = region
		client = s3.NewFromConfig(regionCfg)
	}

	return configureBucketSecurity(ctx, client, bucketName)
}

// createS3Bucket creates the S3 bucket in specified region
func createS3Bucket(ctx context.Context, client *s3.Client, bucketName, region string) error {
	createInput := &s3.CreateBucketInput{
		Bucket: &bucketName,
	}

	// Add location constraint for regions other than us-east-1
	if region != config_app.DefaultAWSRegion {
		createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraint(region),
		}
	}

	_, err := client.CreateBucket(ctx, createInput)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}
	
	fmt.Printf("✅ Bucket created successfully\n")
	return nil
}

// configureBucketSecurity applies security settings to bucket
func configureBucketSecurity(ctx context.Context, client *s3.Client, bucketName string) error {
	if err := enableBucketVersioning(ctx, client, bucketName); err != nil {
		return err
	}
	
	if err := blockPublicAccess(ctx, client, bucketName); err != nil {
		return err
	}
	
	if err := enableBucketEncryption(ctx, client, bucketName); err != nil {
		return err
	}
	
	if err := configureBucketLifecycle(ctx, client, bucketName); err != nil {
		return err
	}
	
	fmt.Printf("🎉 Bucket configuration completed successfully!\n")
	return nil
}

// enableBucketVersioning enables versioning on the bucket
func enableBucketVersioning(ctx context.Context, client *s3.Client, bucketName string) error {
	fmt.Printf("🔄 Enabling versioning...\n")
	_, err := client.PutBucketVersioning(ctx, &s3.PutBucketVersioningInput{
		Bucket: &bucketName,
		VersioningConfiguration: &types.VersioningConfiguration{
			Status: types.BucketVersioningStatusEnabled,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to enable versioning: %w", err)
	}
	fmt.Printf("✅ Versioning enabled\n")
	return nil
}

// blockPublicAccess blocks all public access to the bucket
func blockPublicAccess(ctx context.Context, client *s3.Client, bucketName string) error {
	fmt.Printf("🔒 Blocking public access...\n")
	_, err := client.PutPublicAccessBlock(ctx, &s3.PutPublicAccessBlockInput{
		Bucket: &bucketName,
		PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to configure public access block: %w", err)
	}
	fmt.Printf("✅ Public access blocked\n")
	return nil
}

// enableBucketEncryption enables AES-256 server-side encryption
func enableBucketEncryption(ctx context.Context, client *s3.Client, bucketName string) error {
	fmt.Printf("🔐 Enabling server-side encryption (AES-256)...\n")
	_, err := client.PutBucketEncryption(ctx, &s3.PutBucketEncryptionInput{
		Bucket: &bucketName,
		ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
			Rules: []types.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
						SSEAlgorithm: types.ServerSideEncryptionAes256,
					},
					BucketKeyEnabled: aws.Bool(true),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to configure encryption: %w", err)
	}
	fmt.Printf("✅ Server-side encryption enabled\n")
	return nil
}

// configureBucketLifecycle sets up lifecycle policy for cleanup
func configureBucketLifecycle(ctx context.Context, client *s3.Client, bucketName string) error {
	fmt.Printf("♻️ Configuring lifecycle policy...\n")
	_, err := client.PutBucketLifecycleConfiguration(ctx, &s3.PutBucketLifecycleConfigurationInput{
		Bucket: &bucketName,
		LifecycleConfiguration: &types.BucketLifecycleConfiguration{
			Rules: []types.LifecycleRule{
				{
					ID:     aws.String("cleanup"),
					Status: types.ExpirationStatusEnabled,
					Filter: &types.LifecycleRuleFilter{
						Prefix: aws.String(""),
					},
					AbortIncompleteMultipartUpload: &types.AbortIncompleteMultipartUpload{
						DaysAfterInitiation: aws.Int32(config_app.DefaultAbortIncompleteMultipartUploadDays),
					},
					Expiration: &types.LifecycleExpiration{
						ExpiredObjectDeleteMarker: aws.Bool(true),
					},
					NoncurrentVersionExpiration: &types.NoncurrentVersionExpiration{
						NoncurrentDays: aws.Int32(config_app.DefaultNoncurrentVersionExpirationDays),
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to configure lifecycle: %w", err)
	}
	fmt.Printf("✅ Lifecycle policy configured\n")
	return nil
}

// Utility functions for bucket operations

// GetBucketRegion gets the region of an existing bucket without offering to create it
func GetBucketRegion(ctx context.Context, cfg aws.Config, bucket string) (string, error) {
	client := s3.NewFromConfig(cfg)

	result, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get bucket location: %w", err)
	}

	// Handle default region (us-east-1 returns empty string)
	region := string(result.LocationConstraint)
	if region == "" {
		region = config_app.DefaultAWSRegion
	}

	return region, nil
}

// GetBucketRegionWithConfig gets the region of an existing bucket and returns updated config
func GetBucketRegionWithConfig(ctx context.Context, cfg aws.Config, bucket string) (string, aws.Config, error) {
	region, err := GetBucketRegion(ctx, cfg, bucket)
	if err != nil {
		return "", aws.Config{}, err
	}

	// Create region-specific config if needed
	if region != cfg.Region {
		updatedCfg := cfg.Copy()
		updatedCfg.Region = region
		return region, updatedCfg, nil
	}

	return region, cfg, nil
}

// isValidRegion checks if the provided region code is valid
// isValidRegion checks if the provided region code is valid
func isValidRegion(region string) bool {
	regions := GetAllRegions()
	for _, r := range regions {
		if r == region {
			return true
		}
	}
	return false
}