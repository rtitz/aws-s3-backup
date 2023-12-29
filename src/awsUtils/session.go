package awsUtils

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/variables"
)

func CreateAwsSession(ctx context.Context) aws.Config {

	var cfg aws.Config
	var err error

	if variables.AwsAuthCredentialsFrom == "files" {

		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithSharedCredentialsFiles(
				[]string{variables.AwsCredentialsFile},
			),
			config.WithSharedConfigFiles(
				[]string{variables.AwsConfigFile},
			),
		)
	} else if variables.AwsAuthCredentialsFrom == "awsCliProfile" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(variables.AwsCliRegion),
			config.WithSharedConfigProfile(variables.AwsCliProfile),
		)
	}

	if err != nil {
		log.Fatalf("ERROR: %v\n", err)
	}

	// Test credentials
	clientS3 := s3.NewFromConfig(cfg)
	_, errTest := clientS3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if errTest != nil {
		log.Fatalf("CREDENTIAL TEST (ListBuckets) FAILED!\nERROR: %v\n", errTest)
	}

	return cfg
}
