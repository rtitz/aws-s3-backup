package restoreUtils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rtitz/aws-s3-backup/awsUtils"
	"github.com/rtitz/aws-s3-backup/fileUtils"
	"github.com/rtitz/aws-s3-backup/generalUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Restore an S3 object (Download the object, if needed trigger the restore process from storage class, wait until finished and download)
func restoreObjects(ctx context.Context, cfg aws.Config, bucket, inputJson, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool, restoreExpiresAfterDays int64) ([]variables.InputData, bool, error) {
	var input []variables.InputData
	var restoresPending bool = false
	jsonFile, err := os.Open(inputJson)
	if err != nil {
		return input, restoresPending, err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var contents variables.Contents
	json.Unmarshal(byteValue, &contents)

	for i := 0; i < len(contents.Contents); i++ {
		var restoreStatus string = variables.RestoreNotNeededMessage
		if slices.Contains(variables.StorageClassesNeedRestore, contents.Contents[i].StorageClass) {
			objectInfo, err := awsUtils.HeadObject(ctx, cfg, bucket, contents.Contents[i].Key)
			if err != nil {
				log.Fatalf("ERROR! Failed to get object info! / Check authentication! (%v)\n", err)
			}
			restoreStatus = variables.RestoreNotInitiatedMessage
			if objectInfo.Restore != nil { // Restore initiated
				restoreStatus = *objectInfo.Restore
				if strings.Contains(restoreStatus, "ongoing-request=\"true\"") {
					var ongoingMessage string = variables.RestoreOngoingMessageBulk
					if strings.ToLower(retrievalMode) == "bulk" {
						ongoingMessage = variables.RestoreOngoingMessageBulk
					} else if strings.ToLower(retrievalMode) == "standard" {
						ongoingMessage = variables.RestoreOngoingMessageStandard
					}
					restoreStatus = ongoingMessage + " (Details: " + *objectInfo.Restore + ")"
				} else if strings.Contains(restoreStatus, "ongoing-request=\"false\"") {
					restoreStatus = variables.RestoreDoneMessage + " (Details: " + *objectInfo.Restore + ")"
				}
			}
		}
		_, size, unit := fileUtils.FileSizeUnitCalculation(float64(contents.Contents[i].Size))

		fmt.Printf("%s \n * Size: %.2f %s\n * StorageClass: %s\n * RestoreStatus: %s\n", contents.Contents[i].Key, size, unit, contents.Contents[i].StorageClass, restoreStatus)
		if restoreStatus == "Not initiated" {
			var c bool
			if restoreWithoutConfirmation {
				c = true
			} else {
				c = generalUtils.AskForConfirmation(" Request restore of this object?", true, true)
			}
			if c {
				if err := awsUtils.RestoreObject(ctx, cfg, bucket, contents.Contents[i].Key, retrievalMode, restoreExpiresAfterDays); err != nil {
					fmt.Println("Failed to restore object: ", err.Error())
				} else {
					fmt.Println("Restore requested!")
					restoresPending = true
				}
			}
		}

		if strings.Contains(restoreStatus, "ongoing-request=\"true\"") {
			restoresPending = true
		}

		if strings.Contains(restoreStatus, "ongoing-request=\"false\"") || (restoreStatus == variables.RestoreNotNeededMessage) {
			fmt.Println(" Downloading ...")
			encrypted, fileName, downloaded, err := awsUtils.GetObject(ctx, cfg, bucket, contents.Contents[i].Key, downloadLocation)
			if err != nil {
				fmt.Println("Download failed! ", err.Error())
			} else {
				if downloaded {
					fmt.Println("Download: OK")
				} else if !downloaded {
					fmt.Println("Download: SKIPPED (Already downloaded!)")
				}
				if encrypted {
					variables.FilesNeedingDecryption = append(variables.FilesNeedingDecryption, fileName)
				}
			}
		}
		fmt.Printf("\n")
	}
	return input, restoresPending, nil
}
