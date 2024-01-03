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
	"strconv"
	"strings"
	"time"

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

func checkIfPathAlreadyProcessed(processedTrackingFile, path string, write bool) (bool, error) {
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
		w.WriteString(path + "\n * Timestamp of upload: " + timestampUnixStr + " (" + timestampStr + ")\n\n")
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

// START
func main() {

	// Define and check parameters
	inputFile := flag.String("json", "", "JSON file that contains the input parameters")
	awsProfile := flag.String("profile", variables.AwsCliProfileDefault, "Specify the AWS CLI profile, for example: 'default'")
	awsRegion := flag.String("region", variables.AwsCliRegionDefault, "Specify the AWS CLI profile, for example: 'default'")
	flag.Parse()

	if *inputFile == "" {
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		os.Exit(999)
	}
	processedTrackingFile := *inputFile + variables.ProcessedTrackingSuffix
	// End of: Define and check parameters

	fmt.Printf("%s %s\n\n", variables.AppName, variables.AppVersion)

	// Create new session
	ctx := context.TODO()
	cfg := awsUtils.CreateAwsSession(ctx, *awsProfile, *awsRegion)

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
	tasks, err := getInput(*inputFile)
	if err != nil {
		log.Fatalf("FAILED TO PARSE %s : %v", *inputFile, err)
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
		fmt.Println(cleanupTmpStorageBool)

		// Check if path is aleady processed
		os.Chdir(currentWorkingDirectory)
		processed, _ := checkIfPathAlreadyProcessed(processedTrackingFile, path, false)
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
		checkIfPathAlreadyProcessed(processedTrackingFile, path, true)
	}

	// Put additional data about backup in S3 Bucket
	if filesUploaded {
		additionalUploadOk := true
		listOfAdditionalFiles := []string{
			*inputFile,
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

}
