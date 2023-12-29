package awsUtils

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DeleteObj(ctx context.Context, cfg aws.Config, bucket, object string) error {

	log.Printf("Delete %s in bucket: %s ... \n", object, bucket)
	clientS3 := s3.NewFromConfig(cfg)

	clientS3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &bucket, Key: &object})
	return nil

}
