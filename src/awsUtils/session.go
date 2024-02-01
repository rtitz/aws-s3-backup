package awsUtils

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rtitz/aws-s3-backup/variables"
)

func testAwsSession(ctx context.Context, cfg aws.Config) error {
	clientS3 := s3.NewFromConfig(cfg)
	_, err := clientS3.ListBuckets(ctx, &s3.ListBucketsInput{})
	return err
}

func CreateAwsSession(ctx context.Context, awsProfile, awsRegion string) (aws.Config, error) {

	var cfg aws.Config
	var err error
	var sessionOk = false

	// Load AWS session from environment variables
	if awsProfile == variables.AwsCliProfileDefault {
		fmt.Printf("Authentication via AWS environment variables... ")
		cfg, _ = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
		)
		if err := testAwsSession(ctx, cfg); err == nil {
			sessionOk = true
		}
		if sessionOk {
			fmt.Printf("Successful!\n")
		} else {
			fmt.Printf("Failed, trying next method...\n")
		}
	}

	// Load AWS session from config file
	if variables.AwsAuthCredentialsFrom == "files" && !sessionOk {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithSharedCredentialsFiles(
				[]string{variables.AwsCredentialsFile},
			),
			config.WithSharedConfigFiles(
				[]string{variables.AwsConfigFile},
			),
		)
		if err != nil {
			log.Fatalf("ERROR! Failed to load AWS credentials from file '%s' (%v)\n", variables.AwsCredentialsFile, err)
		}

		// Load AWS session from AWS cli config
	} else if variables.AwsAuthCredentialsFrom == "awsCliProfile" && !sessionOk {
		fmt.Printf("Authentication via AWS CLI profile... ")
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(awsRegion),
			config.WithSharedConfigProfile(awsProfile),
		)
		if err != nil {
			fmt.Printf("\n")
			log.Fatalf("ERROR! Failed to load AWS CLI Profile '%s' (%v)\n", awsProfile, err)
		}
	}

	if !sessionOk {
		err = testAwsSession(ctx, cfg)
		if err != nil {
			fmt.Printf("\n")
			log.Fatalf("ERROR! Failed to test AWS credentials with ListBuckets operation. Verify that credentials are fine and 's3:ListAllMyBuckets' is allowed for your cedentials! (%v)", err)
		} else {
			fmt.Printf("Successful!\n")
		}
	}

	return cfg, nil
}
