package awsUtils

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// HeadObject retrieves metadata from an object without returning the object itself.
func HeadObject(ctx context.Context, cfg aws.Config, bucket, object string) (*s3.HeadObjectOutput, error) {

	clientS3 := s3.NewFromConfig(cfg)

	output, err := clientS3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &bucket, Key: &object})
	return output, err
}
