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

func getInput(inputJson string) ([]variables.InputData, error) {
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
				Source:                    c,
				LocalPath:                 []string{},
				RemotePath:                []string{},
				S3Prefix:                  tasks.Tasks[i].S3Prefix,
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
	// END OF: TAR.GZ
	return archiveFile, nil
}

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

func controlBackup(ctx context.Context, cfg aws.Config, inputFile string) error {
	fmt.Println("MODE: BACKUP")
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

	os.Chdir(currentWorkingDirectory)
	tasks, err := getInput(inputFile)
	if err != nil {
		log.Fatalf("FAILED TO PARSE %s : %v", inputFile, err)
	}
	if len(tasks) < 1 {
		log.Fatalf("NO TASKS FOUND! Please check that the json in your parameters is in correct format!")
	}

	for _, task := range tasks {
		path := task.Source
		s3Bucket := task.S3Bucket
		s3Prefix := filepath.Clean(task.S3Prefix)
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

		archiveTmp := filepath.Clean(tmpStorageToBuildArchives)
		os.MkdirAll(archiveTmp, os.ModePerm)

		c := []string{path} // This is the content for the backup

		archivePath := filepath.Clean(filepath.Dir(path)) + "/"
		archive := filepath.Base(path)
		//archive = archive + "." + variables.ArchiveExtension
		//archive = strings.ReplaceAll(archive, " ", "-") // REPLACE SPACE WITH -
		//fullArchivePath :=
		fullArchivePath, _ := buildArchive(c, archiveTmp+"/"+archive)
		listOfParts, err := fileUtils.SplitArchive(fullArchivePath, int64(archiveSplitEachMB))
		if err != nil {
			log.Fatalf("FAILED TO SPLIT: %v", err)
		}

		numberOfParts := len(listOfParts)
		if numberOfParts > 1 { // Remove original (unsplitted) archive, if it has been splitted
			os.Remove(fullArchivePath)
			HowToFile, err := createTxtHowToCombineSplittedArchive(archive, listOfParts)
			if err != nil {
				log.Fatalf("FAILED TO CREATE HOW-TO-FILE: %v", err)
			}
			listOfParts = append(listOfParts, HowToFile)
		}

		for i, part := range listOfParts {
			partNumber := i + 1

			// Get file info
			_, sizeRaw, size, unit, checksum, err := fileUtils.GetFileInfo(part, checksumMode)
			_ = sizeRaw
			if err != nil {
				log.Fatalf("ERROR GETTING FILE INFO: %v\n", err)
			}

			log.Println("S3 path: s3://" + s3Bucket + "/" + s3Prefix + archivePath + filepath.Base(part))
			log.Println("Local path: " + part)
			log.Printf("Size: %.2f %s\n", size, unit)
			log.Println("StorageClass: " + storageClass)

			if partNumber > numberOfParts { // HowToFile
				log.Printf("Upload (HowToFile) ...")
			} else if numberOfParts > 1 {
				log.Printf("Upload (%d/%d) ...", partNumber, numberOfParts)
			} else {
				log.Printf("Upload...")
			}
			if err := awsUtils.PutObject(ctx, cfg, checksumMode, checksum, part, s3Bucket, s3Prefix+archivePath+filepath.Base(part), storageClass); err != nil {
				time.Sleep(time.Millisecond * 200)
				errCleanup := awsUtils.DeleteObj(ctx, cfg, s3Bucket, s3Prefix+archivePath+filepath.Base(part))
				if errCleanup != nil {
					log.Printf("FAILED TO CLEANUP BROKEN UPLOAD: s3://%s/%s ERROR: %v", s3Bucket, s3Prefix+archivePath+filepath.Base(part), errCleanup)
				}
				log.Fatalf("UPLOAD FAILED! %v", err)
			}

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

			if cleanupTmpStorageBool {
				os.Remove(part)
			}
			filesUploaded = true
		}

		// Write path to file that records the processed paths
		os.Chdir(currentWorkingDirectory)
		checkIfPathAlreadyProcessed(processedTrackingFile, path, listOfParts, true)
	}

	// Put additional data about backup in S3 Bucket
	if filesUploaded {
		additionalUploadOk := true
		listOfAdditionalFiles := []string{
			inputFile,
			processedTrackingFile,
		}
		s3Bucket := tasks[0].S3Bucket
		s3Prefix := filepath.Clean(tasks[0].S3Prefix) + "/"
		log.Println("Upload of additional JSON files...")
		for _, part := range listOfAdditionalFiles {
			// Get file info
			_, _, _, _, checksum, _ := fileUtils.GetFileInfo(part, checksumMode)
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

func restoreObjects(ctx context.Context, cfg aws.Config, bucket, inputJson, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool) ([]variables.InputData, error) {
	var input []variables.InputData
	jsonFile, err := os.Open(inputJson)
	if err != nil {
		return input, err
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
					restoreStatus = variables.RestoreOngoingMessage + " (Details: " + *objectInfo.Restore + ")"
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
				if err := awsUtils.RestoreObject(ctx, cfg, bucket, contents.Contents[i].Key, retrievalMode); err != nil {
					fmt.Println("Failed to restore object: ", err.Error())
				} else {
					fmt.Println("Restore requested!")
				}
			}
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
	return input, nil
}

func controlRestore(ctx context.Context, cfg aws.Config, bucket, prefix, inputFile, downloadLocation, retrievalMode string, restoreWithoutConfirmation bool) error {
	fmt.Println("MODE: RESTORE")

	if restoreWithoutConfirmation {
		fmt.Printf("\nATTENTION!\nRestore of objects from Glacier / archive storage classes to standard storage class will be done WITHOUT any confirmation, because you have specified the '%s' parameter!\n\n", variables.RestoreWithoutConfirmationParameter)
		c := askForConfirmation("Do you want to continue, without confirming restore requests from from Glacier / archive storage classes?", true, false)
		if !c {
			fmt.Println("Abort by user!")
			os.Exit(9)
		}
	}

	if bucket == "" {
		fmt.Println("No bucket specified")
		fmt.Println("Here is the list of buckets you can specify with parameter -bucket")
		fmt.Println()
		err := listBuckets(ctx, cfg)
		fmt.Println()
		return err
	}

	if downloadLocation == "" {
		fmt.Println("Download location not specified")
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		os.Exit(999)
	}

	if inputFile == "" && prefix == "" { // Build an input file is not given, matching the JSON output of commend: aws s3api list-objects-v2 --bucket s3-bucket
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

	if _, err := os.Stat(inputFile); err == nil { // If inputFile exists
		restoreObjects(ctx, cfg, bucket, inputFile, downloadLocation, retrievalMode, restoreWithoutConfirmation)
		fmt.Println("Done! ")
	} else {
		fmt.Printf("ERROR! Input file '%s' does not exist!\n", inputFile)
		os.Exit(3)
	}

	return nil
}

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
	retrievalMode := flag.String("retrievalMode", variables.DefaultRetrievalMode, "Mode of retrieval (bulk or standard)")
	restoreWithoutConfirmation := flag.Bool(variables.RestoreWithoutConfirmationParameter, false, "Restore objects from Glacier / archive storage classes to standard storage class has to be confirmed per object. If this parameter is specified, restores will be done without confirmation!")
	//combineDownloadedParts := flag.Bool("combineDownloadedParts", false, "Automatically combine downloaded splittet file parts to one single file after download")
	awsProfile := flag.String("profile", variables.AwsCliProfileDefault, "Specify the AWS CLI profile, for example: 'default'")
	awsRegion := flag.String("region", variables.AwsCliRegionDefault, "Specify the AWS CLI profile, for example: 'us-east-1'")
	flag.Parse()
	*retrievalMode = strings.ToLower(*retrievalMode)
	*mode = strings.ToLower(*mode)

	if (*mode == "backup" && *inputFile == "") || (*mode != "backup" && *mode != "restore") {
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		os.Exit(999)
	}
	// End of: Define and check parameters

	fmt.Printf("%s %s\n\n", variables.AppName, variables.AppVersion)

	// Create new session
	ctx := context.TODO()
	cfg := awsUtils.CreateAwsSession(ctx, *awsProfile, *awsRegion)

	if *mode == "backup" {
		controlBackup(ctx, cfg, *inputFile)
	} else if *mode == "restore" {
		controlRestore(ctx, cfg, *bucket, *prefix, *inputFile, *downloadLocation, *retrievalMode, *restoreWithoutConfirmation)
	}

}
