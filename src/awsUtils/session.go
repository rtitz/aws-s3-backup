package awsUtils

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/variables"
)

func CreateAwsSession(ctx context.Context, awsProfile, awsRegion string) aws.Config {

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
			config.WithRegion(awsRegion),
			config.WithSharedConfigProfile(awsProfile),
		)
	}

	if err != nil {
		log.Fatalf("ERROR! Failed to load AWS CLI Profile '%s' (%v)\n", awsProfile, err)
	}

	// Test credentials
	clientS3 := s3.NewFromConfig(cfg)
	_, errTest := clientS3.ListBuckets(ctx, &s3.ListBucketsInput{})
	if errTest != nil {
		log.Fatalf("CREDENTIAL TEST (ListBuckets) FAILED!\nERROR: %v\n", errTest)
	}

	return cfg
}
