package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/rtitz/aws-s3-backup/awsUtils"
	"github.com/rtitz/aws-s3-backup/backupUtils"
	"github.com/rtitz/aws-s3-backup/restoreUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// START
func main() {
	//debugUtils.Test()
	//os.Exit(0)

	// TODO
	/*
	 - Automaitically combine downloaded files if splitted before upload (unsplitArchive.go detect filename -part and numbers)
	*/

	// Define and check parameters
	mode := flag.String("mode", "backup", "Operation mode (backup or restore)")
	bucket := flag.String("bucket", "", "Only used for mode 'restore'! You have to specify the bucket, in which your data is stored. Without this parameter you will get a list of Buckets printed.")
	prefix := flag.String("prefix", "", "Only used for mode 'restore'! You can specify a prefix to limit object list to objects in a specific 'folder' in the S3 bucket. (Example: 'archive')")
	inputFile := flag.String("json", "", "JSON file that contains the input parameters")
	downloadLocation := flag.String("destination", "", "Only used for mode 'restore'! Path / directory the restore should be downloaded to. Download location. (Example: 'restore/')")
	retrievalMode := flag.String("retrievalMode", variables.DefaultRetrievalMode, "Only used for mode 'restore'! Mode of retrieval (bulk or standard) for objects stored Glacier / archive storage classes. (bulk takes up to 48 hours / standard takes up to 12 hours, but is more expensive than bulk)")
	restoreWithoutConfirmation := flag.Bool("restoreWithoutConfirmation", false, "Only used for mode 'restore'! Restore objects from Glacier / archive storage classes to standard storage class has to be confirmed per object. If this parameter is specified, restores will be done without confirmation!")
	autoRetryDownloadMinutes := flag.Int64("autoRetryDownloadMinutes", 0, "Only used for mode 'restore'! If a restore from Glacier / archive storage classes to standard storage class is needed and this is for example 60 it will retry the download every 60 minutes. If this parameter is specified, restores will be done without confirmation!")
	restoreExpiresAfterDays := flag.Int64("restoreExpiresAfterDays", int64(variables.DefaultDaysRestoreIsAvailable), "Only used for mode 'restore'! Days that a restore from DeepArchive storage classes is available in (more expensive) Standard storage class")
	awsProfile := flag.String("profile", variables.AwsCliProfileDefault, "Specify the AWS CLI profile, for example: 'default'")
	awsRegion := flag.String("region", variables.AwsCliRegionDefault, "Specify the AWS CLI profile, for example: 'us-east-1'")
	flag.Parse()
	*retrievalMode = strings.ToLower(*retrievalMode)
	*mode = strings.ToLower(*mode)

	if (*mode == "backup" && *inputFile == "") || (*mode != "backup" && *mode != "restore") {
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		fmt.Printf("\nFor help visit: https://github.com/rtitz/aws-s3-backup\n\n")
		os.Exit(11)
	}

	if *autoRetryDownloadMinutes > 0 {
		*restoreWithoutConfirmation = true
		if *autoRetryDownloadMinutes < 60 {
			fmt.Printf("Parameter -autoRetryDownloadMinutes must be 60 or higher.\n")
			os.Exit(14)
		}
	}

	if *restoreExpiresAfterDays < 1 {
		fmt.Printf("Parameter -restoreExpiresAfterDays must be 1 or higher.\n")
		os.Exit(15)
	}
	// End of: Define and check parameters

	fmt.Printf("%s %s\n\n", variables.AppName, variables.AppVersion)

	// Create new session
	ctx := context.TODO()
	cfg, err := awsUtils.CreateAwsSession(ctx, *awsProfile, *awsRegion)
	if err != nil {
		log.Fatalf("FAILED TO AUTHENTICATE! (%v)\n\nSee 'Authentication' section of README !\n", err)
	}

	// Depending on the mode start controlBackup or controlRestore
	if *mode == "backup" {
		backupUtils.ControlBackup(ctx, cfg, *inputFile)
	} else if *mode == "restore" {
		restoreUtils.ControlRestore(ctx, cfg, *bucket, *prefix, *inputFile, *downloadLocation, *retrievalMode, *restoreWithoutConfirmation, *autoRetryDownloadMinutes, *restoreExpiresAfterDays)
	}

}
