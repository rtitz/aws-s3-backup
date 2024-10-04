package awsUtils

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Restores an object from S3 storage class (e.g. from DEEP-ARCHIVE)
func RestoreObject(ctx context.Context, cfg aws.Config, bucket, object, retrievalMode string, restoreExpiresAfterDays int64) error {
	clientS3 := s3.NewFromConfig(cfg)

	var retrievalModeTier types.Tier = types.TierBulk
	if strings.ToLower(retrievalMode) == "bulk" {
		retrievalModeTier = types.TierBulk
	} else if strings.ToLower(retrievalMode) == "standard" {
		retrievalModeTier = types.TierStandard
	}

	_, err := clientS3.RestoreObject(ctx, &s3.RestoreObjectInput{Bucket: &bucket, Key: &object, RestoreRequest: &types.RestoreRequest{
		Days:                 aws.Int32(int32(restoreExpiresAfterDays)),
		GlacierJobParameters: &types.GlacierJobParameters{Tier: retrievalModeTier},
	}})

	return err
}
