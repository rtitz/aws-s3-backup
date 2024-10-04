package awsUtils

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/variables"
)

// List objects inside a S3 bucket and write them to an JSON output file
func ListObjects(ctx context.Context, cfg aws.Config, bucket, prefix, jsonOutputFile string) error {
	var objects []variables.Content
	clientS3 := s3.NewFromConfig(cfg)

	output, err := clientS3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &bucket, Prefix: &prefix})
	if err != nil {
		log.Fatalf("ERROR! Failed to list objects! Is the bucket name correct? (%v)\n", err)
	}
	objectCounter := 0
	for _, content := range output.Contents {
		item := variables.Content{
			Key:          *content.Key,
			Size:         *content.Size,
			StorageClass: string(content.StorageClass),
			LastModified: *content.LastModified,
		}

		if *content.Size != 0 { // Exclude folders
			objectCounter++
			fmt.Println(*content.Key)
			objects = append(objects, item)
		}
	}
	contents := variables.Contents{
		Contents: objects,
	}
	file, _ := json.MarshalIndent(contents, "", " ")
	_ = os.WriteFile(jsonOutputFile, file, 0644)

	fmt.Println()
	fmt.Printf("Number of objects returned: %d\n", objectCounter)
	fmt.Fprintf(os.Stdout, "More objects available than returned: %t\n\n", *output.IsTruncated)
	return err
}
