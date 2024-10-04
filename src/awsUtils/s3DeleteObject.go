package awsUtils

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Delete Object from S3 bucket
func DeleteObj(ctx context.Context, cfg aws.Config, bucket, object string) error {

	log.Printf("Delete: s3://%s/%s ... \n", bucket, object)
	clientS3 := s3.NewFromConfig(cfg)

	_, err := clientS3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &bucket, Key: &object})
	return err
}
