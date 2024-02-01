package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rtitz/aws-s3-backup/awsUtils"
	"github.com/rtitz/aws-s3-backup/fileUtils"
	"github.com/rtitz/aws-s3-backup/variables"
)

// Used for backup
func getInputBackup(inputJson string) ([]variables.InputData, error) {
	var input []variables.InputData
	jsonFile, err := os.Open(inputJson)
	if err != nil {
		return input, err
	}
	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)
	var tasks variables.Tasks
	json.Unmarshal(byteValue, &tasks)

	for i := 0; i < len(tasks.Tasks); i++ {
		for _, c := range tasks.Tasks[i].Content {

			splitSize, err := strconv.Atoi(tasks.Tasks[i].ArchiveSplitEachMB)
			if ((int64(splitSize) >= variables.SplitUploadsEachXMegaBytes) && variables.UploadMethod == "TransferManager") || err != nil {
				errMessage := fmt.Sprintf("ArchiveSplitEachMB must be smaller than %v MB", variables.SplitUploadsEachXMegaBytes)
				return input, errors.New(errMessage)
			}

			newEntry := variables.InputData{
				Source:     c,
				LocalPath:  []string{},
				RemotePath: []string{},
				S3Prefix:   tasks.Tasks[i].S3Prefix,
				//TrimBeginningOfPathInS3:   tasks.Tasks[i].TrimBeginningOfPathInS3,
				S3Bucket:                  tasks.Tasks[i].S3Bucket,
				StorageClass:              tasks.Tasks[i].StorageClass,
				ArchiveSplitEachMB:        tasks.Tasks[i].ArchiveSplitEachMB,
				TmpStorageToBuildArchives: tasks.Tasks[i].TmpStorageToBuildArchives,
				CleanupTmpStorage:         tasks.Tasks[i].CleanupTmpStorage,
				Sha256CheckSum:            "0",
			}
			input = append(input, newEntry)
		}
	}
	return input, nil
}

// Used for backup
func buildArchive(files []string, archiveFile string) (string, error) {
	archiveFile = archiveFile + "." + variables.ArchiveExtension
	archiveFile = strings.ReplaceAll(archiveFile, " ", "-") // REPLACE SPACE WITH -

	log.Println("Creating archive...")
	// Create output file
	out, err := os.Create(archiveFile)
	if err != nil {
		log.Fatalln("Error writing archive:", err)
	}
	defer out.Close()

	// Create the archive and write the output to the "out" Writer
	var keepArchiveFile bool
	keepArchiveFile, err = fileUtils.CreateArchive(files, out)
	if err != nil {
		out.Close()
		os.Remove(archiveFile)
		log.Fatalln("Error creating archive:", err)
	}
	if keepArchiveFile {
		log.Println("Archive created successfully")
	} else { // Archive not created since it is already an archive
		os.Remove(archiveFile)
		archiveFile = strings.TrimSuffix(archiveFile, "."+variables.ArchiveExtension)

		file := files[0]
		source, err := os.Open(file) //open the source file
		if err != nil {
			panic(err)
		}
		defer source.Close()

		destination, err := os.Create(archiveFile) //create the destination file
		if err != nil {
			panic(err)
		}
		defer destination.Close()
		_, err = io.Copy(destination, source) //copy the contents of source to destination file
		if err != nil {
			panic(err)
		}
		log.Println("Existing archive copied successfully")
	}
	return archiveFile, nil
}

// Used for backup
func createTxtHowToCombineSplittedArchive(archive string, listOfParts []string) (string, error) {
	var parts string
	var path string
	for i, part := range listOfParts {
		path = filepath.Clean(filepath.Dir(part))
		part = filepath.Base(part)
		if i == 0 { // First iteration in this loop; do not add a space in the beginning
			parts = parts + part
		} else {
			parts = parts + " " + part
		}
	}
	if !strings.HasSuffix(archive, "."+variables.ArchiveExtension) {
		archive = archive + "." + variables.ArchiveExtension
	}

	// Content is a cat command that makes cat on all files and redirects the output into a single new file
	contentOfHowToFile := fmt.Sprintf("cat %s > %s && rm -f %s %s\n", parts, archive, parts, archive+variables.HowToBuildFileSuffix)

	// Create HowToFile file
	howToFile := path + "/" + archive + variables.HowToBuildFileSuffix
	out, err := os.Create(howToFile)
	if err != nil {
		log.Fatalln("Error writing how-to-file:", err)
	}
	defer out.Close()
	w := bufio.NewWriter(out)
	w.WriteString(contentOfHowToFile)
	w.Flush()

	return howToFile, nil
}

// Used for backup
func checkIfPathAlreadyProcessed(processedTrackingFile, path string, listOfParts []string, write bool) (bool, error) {
	if write {
		// Create processed file
		out, err := os.OpenFile(processedTrackingFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln("Error writing 'processed' file:", err)
		}
		defer out.Close()
		w := bufio.NewWriter(out)
		dt := time.Now()
		timestampStr := dt.Format(time.RFC1123)
		timestampUnixStr := strconv.Itoa(int(dt.Unix()))

		if len(listOfParts) > 1 { // Splitted means HowTo exists as additional element in the listOfParts
			listOfParts = listOfParts[:len(listOfParts)-1]
		}
		numberOfParts := len(listOfParts)

		stringToWrite := path + "\n * Timestamp of upload: " + timestampUnixStr + " (" + timestampStr + ")\n * Number of file parts: " + strconv.Itoa(numberOfParts)
		w.WriteString(stringToWrite + "\n\n")
		w.Flush()
		out.Close()
		return true, nil
	} else { // Check if done
		processed := false
		if _, err := os.Stat(processedTrackingFile); errors.Is(err, os.ErrNotExist) {
			//fmt.Println("file not exist")
			return processed, nil
		}
		readFile, err := os.Open(processedTrackingFile)
		if err != nil {
			fmt.Println(err)
		}
		defer readFile.Close()
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		for fileScanner.Scan() {
			//fmt.Println(fileScanner.Text())
			if fileScanner.Text() == path {
				processed = true
				break
			}
		}
		readFile.Close()
		return processed, nil
	}
}

// Control function for backup
func controlBackup(ctx context.Context, cfg aws.Config, inputFile string) error {
	fmt.Printf("\nMODE: BACKUP\n\n")

	// ControlBackup parameter check
	processedTrackingFile := inputFile + variables.ProcessedTrackingSuffix
	currentWorkingDirectory, _ := os.Getwd()
	filesUploaded := false

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
		//trimBeginningOfPathInS3 := task.TrimBeginningOfPathInS3
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

		archivePath := filepath.Clean(filepath.Dir(path)) + "/"       // This is the path to the archive file
		archive := filepath.Base(path)                                // This is the archive file name
		fullArchivePath, _ := buildArchive(c, archiveTmp+"/"+archive) // Create the archive out of the defined content

		// Path in S3 Bucket
		/*fmt.Println("TRIM: ", trimBeginningOfPathInS3)
		fmt.Println("ARCHIVEPATH:", archivePath)
		archivePath = strings.TrimLeft(archivePath, trimBeginningOfPathInS3)
		if !strings.HasPrefix(archivePath, "/") {
			archivePath = "/" + archivePath
		}
		fmt.Println("ARCHIVEPATH:", archivePath)*/
		s3PathToFile := s3Prefix + archivePath // This is the full path where the object in the S3 Bucket will be located
		//fmt.Println("ARCHIVEPATH FULL :", s3PathToFile)

		// Give the archive to the SplitArchive function. If needed (depending on size), it will split the archive.
		listOfParts, err := fileUtils.SplitArchive(fullArchivePath, int64(archiveSplitEachMB))
		if err != nil {
			log.Fatalf("FAILED TO SPLIT: %v", err)
		}

		numberOfParts := len(listOfParts)
		if numberOfParts > 1 { // Remove original (unsplitted) archive, if it has been splitted
			os.Remove(fullArchivePath)
			HowToFile, err := createTxtHowToCombineSplittedArchive(archive, listOfParts) // Create HowToCombineSplittedArchive text file and add it to the list of files need to be uploaded
			if err != nil {
				log.Fatalf("FAILED TO CREATE HOW-TO-FILE: %v", err)
			}
			listOfParts = append(listOfParts, HowToFile)
		}

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
		listOfAdditionalFiles := []string{ // Also upload the input json file and the tracking file about processed uploads
			inputFile,
			processedTrackingFile,
		}
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

// =================================================================================================================

// Used for restore
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

// Used for restore
func restoreObjects(ctx context.Context, cfg aws.Config, bucket, inputJson, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool, autoRetryDownloadMinutes, restoreExpiresAfterDays int64) ([]variables.InputData, bool, error) {
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
				c = askForConfirmation(" Request restore of this object?", true, true)
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
			downloaded, err := awsUtils.GetObject(ctx, cfg, bucket, contents.Contents[i].Key, downloadLocation)
			if err != nil {
				fmt.Println("Download failed! ", err.Error())
			} else if err == nil && downloaded {
				fmt.Println("Download: OK")
			} else if err == nil && !downloaded {
				fmt.Println("Download: SKIPPED (Already downloaded!)")
			}
		}
		fmt.Printf("\n")
	}
	return input, restoresPending, nil
}

// Control function for restore
func controlRestore(ctx context.Context, cfg aws.Config, bucket, prefix, inputFile, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool, autoRetryDownloadMinutes, restoreExpiresAfterDays int64) error {
	fmt.Printf("\nMODE: RESTORE\n\n")

	// ControlRestore parameter check
	if restoreWithoutConfirmation {
		fmt.Printf("\nATTENTION!\nRestore of objects from Glacier / archive storage classes to standard storage class will be done WITHOUT any confirmation, because you have specified the 'restoreWithoutConfirmation' parameter!\n\n")
		c := askForConfirmation("Do you want to continue, without confirming restore requests from from Glacier / archive storage classes?", true, false)
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
		os.Exit(8)
	}

	if inputFile == "" && prefix == "" { // Build an input file if not given, matching the JSON output of commend: aws s3api list-objects-v2 --bucket s3-bucket
		outputFile := variables.JsonOutputFile
		if err := awsUtils.ListObjects(ctx, cfg, bucket, prefix, outputFile); err != nil {
			log.Fatalf("ERROR! Failed to list objects! (%v)\n", err)
		}
		fmt.Printf("Generated: '%s'\n\n", outputFile)
		c := askForConfirmation("Do you want to continue with restore, without editing generated input JSON?", true, false)
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
		c := askForConfirmation("Do you want to continue with restore, without editing generated input JSON?", true, false)
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
			_, pendingRestore, err := restoreObjects(ctx, cfg, bucket, inputFile, downloadLocation, retrievalMode, restoreWithoutConfirmation, autoRetryDownloadMinutes, restoreExpiresAfterDays)
			if err != nil {
				fmt.Printf("ERROR: Restore failed! (%v)\n", err)
			}

			if pendingRestore && autoRetryDownloadMinutes <= 0 { // End (break) the for loop if there are pending restores and 'auto retry download' is off
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
	}

	return nil
}

// General function to ask for user confirmations
func askForConfirmation(s string, handleDefault, defaultValue bool) bool {
	reader := bufio.NewReader(os.Stdin)
	answers := "[y/n]"
	if handleDefault && defaultValue {
		answers = "[Y/n]"
	} else if handleDefault && !defaultValue {
		answers = "[y/N]"
	}

	for {
		fmt.Printf("%s %s: ", s, answers)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}
		response = strings.ToLower(strings.TrimSpace(response))
		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		} else if handleDefault && response == "" {
			return defaultValue
		}
	}
}

// START
func main() {
	// Define and check parameters
	mode := flag.String("mode", "backup", "Operation mode (backup or restore)")
	bucket := flag.String("bucket", "", "If mode is 'restore' you have to specify the bucket, in which your data is stored. Without this parameter you will get a list of Buckets printed.")
	prefix := flag.String("prefix", "", "Specify a prefix to limit object list to objects in a specific 'folder' in the S3 bucket. (Example: 'archive')")
	inputFile := flag.String("json", "", "JSON file that contains the input parameters")
	downloadLocation := flag.String("destination", "", "Path / directory the restore should be downloaded to. Download location. (Example: 'restore/')")
	retrievalMode := flag.String("retrievalMode", variables.DefaultRetrievalMode, "Mode of retrieval (bulk or standard) for objects stored Glacier / archive storage classes. (bulk takes up to 48 hours / standard takes up to 12 hours, but is more expensive than bulk)")
	restoreWithoutConfirmation := flag.Bool("restoreWithoutConfirmation", false, "Restore objects from Glacier / archive storage classes to standard storage class has to be confirmed per object. If this parameter is specified, restores will be done without confirmation!")
	autoRetryDownloadMinutes := flag.Int64("autoRetryDownloadMinutes", 0, "If a restore from Glacier / archive storage classes to standard storage class is needed and this is for example 60 it will retry the download every 60 minutes. If this parameter is specified, restores will be done without confirmation!")
	restoreExpiresAfterDays := flag.Int64("restoreExpiresAfterDays", int64(variables.DefaultDaysRestoreIsAvailable), "Days that a restore from DeepArchive storage classes is available in (more expensive) Standard storage class")
	//combineDownloadedParts := flag.Bool("combineDownloadedParts", false, "Automatically combine downloaded splittet file parts to one single file after download, if this parameter is specified.")
	awsProfile := flag.String("profile", variables.AwsCliProfileDefault, "Specify the AWS CLI profile, for example: 'default'")
	awsRegion := flag.String("region", variables.AwsCliRegionDefault, "Specify the AWS CLI profile, for example: 'us-east-1'")
	flag.Parse()
	*retrievalMode = strings.ToLower(*retrievalMode)
	*mode = strings.ToLower(*mode)

	if (*mode == "backup" && *inputFile == "") || (*mode != "backup" && *mode != "restore") {
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
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
		log.Fatalf("FAILED TO AUTHENTICATE! (%v)", err)
	}

	// Depending on the mode start controlBackup or controlRestore
	if *mode == "backup" {
		controlBackup(ctx, cfg, *inputFile)
	} else if *mode == "restore" {
		controlRestore(ctx, cfg, *bucket, *prefix, *inputFile, *downloadLocation, *retrievalMode, *restoreWithoutConfirmation, *autoRetryDownloadMinutes, *restoreExpiresAfterDays)
	}

}
