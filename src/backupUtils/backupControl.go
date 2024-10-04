package backupUtils

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/awsUtils"
	"github.com/rtitz/aws-s3-backup/cryptUtils"
	"github.com/rtitz/aws-s3-backup/fileUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Control function for backup
func ControlBackup(ctx context.Context, cfg aws.Config, inputFile string) error {
	fmt.Printf("\nMODE: BACKUP\n\n")

	// ControlBackup parameter check
	processedTrackingFile := inputFile + variables.ProcessedTrackingSuffix
	currentWorkingDirectory, _ := os.Getwd()
	filesUploaded := false
	skipInputFileUpload := false

	// Checksum mode
	var checksumMode string
	if variables.ChecksumMode == "sha256" {
		checksumMode = "sha256"
	} else if variables.ChecksumMode == "md5" {
		checksumMode = "md5"
	}

	os.Chdir(currentWorkingDirectory) // Ensure that working directory is current dir
	tasks, err := getInputBackup(inputFile)
	if err != nil {
		log.Fatalf("FAILED TO PARSE %s : %v", inputFile, err)
	}
	if len(tasks) < 1 {
		log.Fatalf("NO TASKS FOUND! Please check that the json in your parameters is in correct format!")
	}
	// END OF: ControlBackup parameter check

	for _, task := range tasks { // Loop through tasks defined in input json file
		path := task.Source
		s3Bucket := task.S3Bucket
		s3Prefix := filepath.Clean(task.S3Prefix)
		trimBeginningOfPathInS3 := task.TrimBeginningOfPathInS3
		encryptionSecret := task.EncryptionSecret
		archiveSplitEachMB, _ := strconv.Atoi(task.ArchiveSplitEachMB)
		tmpStorageToBuildArchives := task.TmpStorageToBuildArchives
		cleanupTmpStorage := task.CleanupTmpStorage

		var cleanupTmpStorageBool bool
		switch strings.ToLower(cleanupTmpStorage) {
		case "true":
			cleanupTmpStorageBool = true
		case "yes":
			cleanupTmpStorageBool = true
		case "false":
			cleanupTmpStorageBool = false
		case "no":
			cleanupTmpStorageBool = false
		default:
			cleanupTmpStorageBool = variables.CleanupAfterUploadDefault
		}

		// Check if path is aleady processed
		os.Chdir(currentWorkingDirectory)
		processed, _ := checkIfPathAlreadyProcessed(processedTrackingFile, path, []string{}, false)
		if processed {
			log.Printf("SKIP - Path: '%s' already in '%s'!", path, processedTrackingFile)
			continue
		}

		// StorageClasses
		var storageClass types.StorageClass
		switch task.StorageClass {
		case "STANDARD":
			storageClass = types.StorageClassStandard
		case "STANDARD_IA": // Min storage duration in days: 30 and 128kB
			storageClass = types.StorageClassStandardIa
		case "DEEP_ARCHIVE": // Min storage duration in days: 180 and 40kB
			storageClass = types.StorageClassDeepArchive
		case "GLACIER_IR": // Min storage duration in days: 90 and 128kB
			storageClass = types.StorageClassGlacierIr
		case "GLACIER": // Min storage duration in days: 90 and 128kB
			storageClass = types.StorageClassGlacier
		case "REDUCED_REDUNDANCY": // Only 99.99% durability! Not recommended!
			storageClass = types.StorageClassReducedRedundancy
		default:
			log.Printf("WARNING: StorageClass '%s' not supported! Using 'STANDARD' instead!", task.StorageClass)
			storageClass = types.StorageClassStandard
		}

		archiveTmp := filepath.Clean(tmpStorageToBuildArchives) // This is the temp location to build the archives (directory)
		os.MkdirAll(archiveTmp, os.ModePerm)

		c := []string{path} // This is the content for the backup (c is the "Content" section of the input json file)

		archivePath := filepath.Clean(filepath.Dir(path)) + "/"                 // This is the path to the archive file
		archive := filepath.Base(path)                                          // This is the archive file name
		fullArchivePath, _ := fileUtils.BuildArchive(c, archiveTmp+"/"+archive) // Create the archive out of the defined content

		var encryptionEnabled bool = false
		if encryptionSecret != "" { // Only if EncryptionSecret is set
			encryptionEnabled = true
			skipInputFileUpload = true
			//fullArchivePath, _ = fileUtils.CryptFile(true, fullArchivePath, "default", encryptionSecret)
			//archive = filepath.Base(fullArchivePath) // This is the archive file name
		}

		// Give the archive to the SplitArchive function. If needed (depending on size), it will split the archive.
		listOfParts, err := fileUtils.SplitFileIntoSmallerParts(fullArchivePath, int64(archiveSplitEachMB))
		if err != nil {
			log.Fatalf("FAILED TO SPLIT: %v", err)
		}

		numberOfParts := len(listOfParts)
		if numberOfParts > 1 { // Remove original (unsplitted) archive, if it has been splitted
			os.Remove(fullArchivePath)
			HowToFile, err := fileUtils.CreateTxtHowToCombineSplittedFile(archive, listOfParts) // Create HowToCombineSplittedArchive text file and add it to the list of files need to be uploaded
			if err != nil {
				log.Fatalf("FAILED TO CREATE HOW-TO-FILE: %v", err)
			}
			listOfParts = append(listOfParts, HowToFile)
		}

		// Encryption
		if encryptionEnabled {
			//log.Printf("Encryption enabled!")
			var listOfPartsEncrypted []string
			for i, part := range listOfParts { // Iterate through the list of files to be uploaded
				partNumber := i + 1

				if partNumber > numberOfParts { // This is the HowToFile, since it is not counted as one of the archive parts
					log.Printf("Encrypting (HowToFile) ...")
				} else if numberOfParts > 1 { // Splitted archive file being uploaded
					log.Printf("Encrypting (%d/%d) ...", partNumber, numberOfParts)
				} else { // Only one (unsplitted) archive file being uploaded
					log.Printf("Encrypting...")
				}
				outputFileEnc, errEnc := cryptUtils.CryptFile(true, part, "default", encryptionSecret) // True Encrypt ; False Decrypt
				if errEnc != nil {
					log.Fatalf("failed to encrypt: %v", errEnc)
				}
				os.Remove(part)
				listOfPartsEncrypted = append(listOfPartsEncrypted, outputFileEnc)
			}
			listOfParts = listOfPartsEncrypted
			// TODO: Remove EncryptionSecret value from input.json and re-enable upload of inputfile (at the moment skipped if encryption enabled)
		}
		// END OF: Encryption

		// Path in S3 Bucket
		trimmedS3Path := strings.TrimPrefix(archivePath, trimBeginningOfPathInS3)
		if !strings.HasPrefix(trimmedS3Path, "/") {
			trimmedS3Path = "/" + trimmedS3Path
		}
		s3PathToFile := s3Prefix + trimmedS3Path // This is the full path where the object in the S3 Bucket will be located

		for i, part := range listOfParts { // Iterate through the list of files to be uploaded
			partNumber := i + 1

			// Get file info
			_, sizeRaw, size, unit, checksum, err := fileUtils.GetFileInfo(part, checksumMode)
			_ = sizeRaw
			if err != nil {
				log.Fatalf("ERROR GETTING FILE INFO: %v\n", err)
			}

			log.Println("S3 path: s3://" + s3Bucket + "/" + s3PathToFile + filepath.Base(part))
			log.Println("Local path: " + part)
			log.Printf("Size: %.2f %s\n", size, unit)
			log.Println("StorageClass: " + storageClass)

			if partNumber > numberOfParts { // This is the HowToFile, since it is not counted as one of the archive parts
				log.Printf("Upload (HowToFile) ...")
			} else if numberOfParts > 1 { // Splitted archive file being uploaded
				log.Printf("Upload (%d/%d) ...", partNumber, numberOfParts)
			} else { // Only one (unsplitted) archive file being uploaded
				log.Printf("Upload...")
			}

			// Start the upload function
			if err := awsUtils.PutObject(ctx, cfg, checksumMode, checksum, part, s3Bucket, s3PathToFile+filepath.Base(part), storageClass); err != nil {
				time.Sleep(time.Millisecond * 200)
				errCleanup := awsUtils.DeleteObj(ctx, cfg, s3Bucket, s3PathToFile+filepath.Base(part))
				if errCleanup != nil {
					log.Printf("FAILED TO CLEANUP BROKEN UPLOAD: s3://%s/%s ERROR: %v", s3Bucket, s3PathToFile+filepath.Base(part), errCleanup)
				}
				log.Fatalf("UPLOAD FAILED! %v", err)
			}

			// Calculate how many percentage (of splitted archive) are uploaded
			var percentage float64 = (float64(partNumber) / float64(numberOfParts)) * float64(100)
			if percentage == 100 {
				log.Printf(" %.2f %% (%d/%d) UPLOADED - DONE!\n", percentage, partNumber, numberOfParts)
				fmt.Printf("%s\n\n", variables.OutputSeperator)
			} else if partNumber > numberOfParts { // HowToFile
				log.Printf(" HowToFile UPLOADED\n")
				fmt.Printf("%s\n\n", variables.OutputSeperator)
			} else {
				log.Printf(" %.2f %% (%d/%d) UPLOADED\n", percentage, partNumber, numberOfParts)
			}

			// Cleanup the temp storage that has been used to build the archive if CleanupTmpStorage is true
			if cleanupTmpStorageBool {
				os.Remove(part)
			}
			filesUploaded = true
		}

		// Write path to file that records the already processed files / paths
		os.Chdir(currentWorkingDirectory)
		checkIfPathAlreadyProcessed(processedTrackingFile, path, listOfParts, true)
	}
	// Put additional data about backup in S3 Bucket
	if filesUploaded {
		additionalUploadOk := true
		listOfAdditionalFiles := []string{}
		// Also upload the input json file and the tracking file about processed uploads
		if !skipInputFileUpload {
			listOfAdditionalFiles = append(listOfAdditionalFiles, inputFile)
		}
		listOfAdditionalFiles = append(listOfAdditionalFiles, processedTrackingFile)

		s3Bucket := tasks[0].S3Bucket
		s3Prefix := filepath.Clean(tasks[0].S3Prefix) + "/"
		log.Println("Upload of additional JSON files...")
		for _, part := range listOfAdditionalFiles {
			_, _, _, _, checksum, _ := fileUtils.GetFileInfo(part, checksumMode) // Get file info
			if err := awsUtils.PutObject(ctx, cfg, checksumMode, checksum, part, s3Bucket, s3Prefix+filepath.Base(part), types.StorageClassStandard); err != nil {
				time.Sleep(time.Millisecond * 200)
				awsUtils.DeleteObj(ctx, cfg, s3Bucket, s3Prefix+filepath.Base(part))
				log.Printf("UPLOAD OF ADDITIONAL JSON FILES FAILED! %v", err)
				additionalUploadOk = false
			}
		}
		if additionalUploadOk {
			log.Println("Upload of additional JSON files: OK")
		}
	}
	return nil
}
