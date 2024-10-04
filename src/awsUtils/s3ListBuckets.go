package awsUtils

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// List all available S3 buckets
func ListBuckets(ctx context.Context, cfg aws.Config) ([]string, error) {
	var listOfBuckets []string
	clientS3 := s3.NewFromConfig(cfg)

	output, err := clientS3.ListBuckets(ctx, &s3.ListBucketsInput{})
	for _, bucket := range output.Buckets {
		listOfBuckets = append(listOfBuckets, *bucket.Name)
	}
	return listOfBuckets, err
}
