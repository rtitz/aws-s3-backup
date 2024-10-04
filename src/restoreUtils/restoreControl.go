package restoreUtils

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rtitz/aws-s3-backup/awsUtils"
	"github.com/rtitz/aws-s3-backup/cryptUtils"
	"github.com/rtitz/aws-s3-backup/generalUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Control function for restore
func ControlRestore(ctx context.Context, cfg aws.Config, bucket, prefix, inputFile, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool, autoRetryDownloadMinutes, restoreExpiresAfterDays int64) error {
	fmt.Printf("\nMODE: RESTORE\n\n")

	// ControlRestore parameter check
	if restoreWithoutConfirmation {
		fmt.Printf("\nATTENTION!\nRestore of objects from Glacier / archive storage classes to standard storage class will be done WITHOUT any confirmation, because you have specified the 'restoreWithoutConfirmation' parameter!\n\n")
		c := generalUtils.AskForConfirmation("Do you want to continue, without confirming restore requests from from Glacier / archive storage classes?", true, false)
		if !c {
			fmt.Println("Abort by user!")
			os.Exit(9)
		}
		fmt.Println()
	}

	if bucket == "" {
		fmt.Println("No bucket specified")
		fmt.Println("Here is the list of buckets you can specify with parameter -bucket")
		fmt.Println()
		err := listBuckets(ctx, cfg) // Print a list of all existing buckets if no bucket is specified
		fmt.Println()
		return err
	}

	if downloadLocation == "" {
		fmt.Println("Download location not specified (-destination)")
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		fmt.Println("ERROR: Download location not specified (-destination)")
		os.Exit(8)
	}

	if inputFile == "" && prefix == "" { // Build an input file if not given, matching the JSON output of commend: aws s3api list-objects-v2 --bucket s3-bucket
		outputFile := variables.JsonOutputFile
		if err := awsUtils.ListObjects(ctx, cfg, bucket, prefix, outputFile); err != nil {
			log.Fatalf("ERROR! Failed to list objects! (%v)\n", err)
		}
		fmt.Printf("Generated: '%s'\n\n", outputFile)
		c := generalUtils.AskForConfirmation("Do you want to continue with restore, without editing generated input JSON?", true, false)
		if !c {
			return nil
		}
		inputFile = outputFile
	}

	if prefix != "" {
		var outputFile string
		if _, err := os.Stat(inputFile); err == nil { // If inputFile exists
			fmt.Printf("\nERROR: You filtered by prefix parameter and specified json parameter, but the JSON file exists! (%s)\nChoose a path for json parameter that does not exist or remove prefix parameter to take an existing json as input for restore!\n", inputFile)
			os.Exit(2)
		} else { // inputFile does not exist
			outputFile = inputFile
		}
		if inputFile == "" {
			outputFile = variables.JsonOutputFile
		}
		if err := awsUtils.ListObjects(ctx, cfg, bucket, prefix, outputFile); err != nil { // Build an input file if not given, matching the JSON output of commend: aws s3api list-objects-v2 --bucket s3-bucket
			log.Fatalf("ERROR! Failed to list objects! (%v)\n", err)
		}
		fmt.Printf("Generated: '%s'\n\n", outputFile)
		c := generalUtils.AskForConfirmation("Do you want to continue with restore, without editing generated input JSON?", true, false)
		if !c {
			return nil // If answer is 'false' return. Allow the user to adjust the generated restore json file.
		}
		inputFile = outputFile
	}
	// END OF: ControlRestore parameter check

	if _, err := os.Stat(inputFile); err != nil { // If inputFile does not exists
		fmt.Printf("ERROR! Input file '%s' does not exist!\n", inputFile)
		os.Exit(3)
	} else { // Restore objects as described in input json file
		for {
			_, pendingRestore, err := restoreObjects(ctx, cfg, bucket, inputFile, downloadLocation, retrievalMode, restoreWithoutConfirmation, restoreExpiresAfterDays)
			if err != nil {
				fmt.Printf("ERROR: Restore failed! (%v)\n", err)
			}

			if pendingRestore && autoRetryDownloadMinutes <= 0 { // End (break) the for loop, if there are pending restores and 'auto retry download' is off
				fmt.Println("Not everything is downloaded yet. Restores are ongoing.\n'DEEP_ARCHIVE' restores can take up to 48 hours.\nJust execute this command again in a few hours, it will only download new (not already downloaded) objects.")
				break
			} else if !pendingRestore { // End (break) the for loop if there are no pending restores
				log.Println("Done! ")
				break
			}

			// There are pending restores and 'auto retry download' is enabled. Sleep for the configured duration, before retrying the download
			log.Printf("Restores are ongoing. 'DEEP_ARCHIVE' restores can take up to 48 hours. Retry download in %d minutes ... (You can cancel with CTRL + C and execute this command again)\n\n", autoRetryDownloadMinutes)
			time.Sleep(time.Minute * time.Duration(autoRetryDownloadMinutes))
		}

		// If files needing decryption, decrypt them
		numberOfParts := len(variables.FilesNeedingDecryption)
		if numberOfParts > 0 {
			for {
				err := cryptUtils.DecryptFiles(numberOfParts)
				if err == nil {
					break
				} else {
					fmt.Printf("ERROR: Decryption failed! (%v)\n", err)
				}
			}
			fmt.Println("\nDONE!")
		}
	}

	return nil
}
