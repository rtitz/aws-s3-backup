package awsUtils

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/variables"
)

func RestoreObject(ctx context.Context, cfg aws.Config, bucket, object string) error {

	defaultDaysRestoreIsAvailable := variables.DefaultDaysRestoreIsAvailable
	clientS3 := s3.NewFromConfig(cfg)

	_, err := clientS3.RestoreObject(ctx, &s3.RestoreObjectInput{Bucket: &bucket, Key: &object, RestoreRequest: &types.RestoreRequest{
		Days:                 aws.Int32(int32(defaultDaysRestoreIsAvailable)),
		GlacierJobParameters: &types.GlacierJobParameters{Tier: variables.DefaultRestoreObjectTier},
	}})

	return err
}
