package awsUtils

import (
	"context"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/variables"
)

func RestoreObject(ctx context.Context, cfg aws.Config, bucket, object, retrievalMode string) error {

	defaultDaysRestoreIsAvailable := variables.DefaultDaysRestoreIsAvailable
	clientS3 := s3.NewFromConfig(cfg)

	var retrievalModeTier types.Tier = types.TierBulk
	if strings.ToLower(retrievalMode) == "bulk" {
		retrievalModeTier = types.TierBulk
	} else if strings.ToLower(retrievalMode) == "standard" {
		retrievalModeTier = types.TierStandard
	}

	_, err := clientS3.RestoreObject(ctx, &s3.RestoreObjectInput{Bucket: &bucket, Key: &object, RestoreRequest: &types.RestoreRequest{
		Days:                 aws.Int32(int32(defaultDaysRestoreIsAvailable)),
		GlacierJobParameters: &types.GlacierJobParameters{Tier: retrievalModeTier},
	}})

	return err
}