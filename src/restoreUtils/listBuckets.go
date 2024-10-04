package restoreUtils

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rtitz/aws-s3-backup/awsUtils"
)

// Prints a list of availbe S3 buckets
func listBuckets(ctx context.Context, cfg aws.Config) error {
	listOfBuckets, err := awsUtils.ListBuckets(ctx, cfg)
	if err != nil {
		log.Fatalln("Error listing bucekts:", err)
	}

	for _, bucket := range listOfBuckets {
		fmt.Printf("%s\n", bucket)
	}
	return nil
}
