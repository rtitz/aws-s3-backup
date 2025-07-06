package utils

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// CreateAWSSession creates and validates AWS configuration
func CreateAWSSession(ctx context.Context, profile, region string) (aws.Config, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
		config.WithSharedConfigProfile(profile),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("❌ failed to load AWS config: %w", err)
	}

	// Test connection
	client := s3.NewFromConfig(cfg)
	_, err = client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return aws.Config{}, fmt.Errorf("❌ AWS credentials validation failed: %w", err)
	}

	return cfg, nil
}

// UploadFile uploads a file to S3
func UploadFile(ctx context.Context, cfg aws.Config, filePath, bucket, key string, storageClass types.StorageClass) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	uploader := manager.NewUploader(s3.NewFromConfig(cfg))
	_, err = uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:       &bucket,
		Key:          &key,
		Body:         file,
		StorageClass: storageClass,
	})
	return err
}

// ValidateBucketExistsWithRegion checks if bucket exists and returns its region
func ValidateBucketExistsWithRegion(ctx context.Context, cfg aws.Config, bucket string) (string, error) {
	client := s3.NewFromConfig(cfg)

	// First validate bucket exists
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	if err != nil {
		return "", err
	}

	// Get bucket location
	result, err := client.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &bucket,
	})
	if err != nil {
		return "", err
	}

	// Handle default region (us-east-1 returns empty string)
	region := string(result.LocationConstraint)
	if region == "" {
		region = "us-east-1"
	}

	return region, nil
}

// DownloadFile downloads a file from S3
func DownloadFile(ctx context.Context, cfg aws.Config, bucket, key, filePath string) error {
	client := s3.NewFromConfig(cfg)
	result, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})
	if err != nil {
		return err
	}
	defer result.Body.Close()

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, result.Body)
	return err
}
