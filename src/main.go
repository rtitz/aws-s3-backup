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

func checkIfPathAlreadyProcessed(path string, write bool) (bool, error) {
	if write {
		// Create processed file
		out, err := os.OpenFile(variables.ProcessedTrackingFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalln("Error writing 'processed' file:", err)
		}
		defer out.Close()
		w := bufio.NewWriter(out)
		w.WriteString(path + "\n")
		w.Flush()
		out.Close()
		return true, nil
	} else { // Check if done
		processed := false
		if _, err := os.Stat(variables.ProcessedTrackingFile); errors.Is(err, os.ErrNotExist) {
			//fmt.Println("file not exist")
			return processed, nil
		}
		readFile, err := os.Open(variables.ProcessedTrackingFile)
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
	// End of: Define and check parameters

	fmt.Printf("%s %s\n\n", variables.AppName, variables.AppVersion)

	// Create new session
	ctx := context.TODO()
	cfg := awsUtils.CreateAwsSession(ctx, *awsProfile, *awsRegion)

	currentWorkingDirectory, _ := os.Getwd()

	os.Chdir(currentWorkingDirectory)
	tasks, err := getInput(*inputFile)
	if err != nil {
		log.Fatalf("FAILED TO PARSE %s : %v", *inputFile, err)
	}

	for _, task := range tasks {
		path := task.Source
		s3Bucket := task.S3Bucket
		s3Prefix := filepath.Clean(task.S3Prefix)
		archiveSplitEachMB, _ := strconv.Atoi(task.ArchiveSplitEachMB)
		tmpStorageToBuildArchives := task.TmpStorageToBuildArchives

		// Check if path is aleady processed
		os.Chdir(currentWorkingDirectory)
		processed, _ := checkIfPathAlreadyProcessed(path, false)
		if processed {
			log.Printf("SKIP - Path: '%s' already in '%s'!", path, variables.ProcessedTrackingFile)
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
			log.Println("S3 path: s3://" + s3Bucket + "/" + s3Prefix + archivePath + filepath.Base(part))
			log.Println("Local path: " + part)
			log.Println("StorageClass: " + storageClass)

			if partNumber > numberOfParts { // HowToFile
				log.Printf("Upload (HowToFile) ...")
			} else if numberOfParts > 1 {
				log.Printf("Upload (%d/%d) ...", partNumber, numberOfParts)
			} else {
				log.Printf("Upload...")
			}
			if err := awsUtils.PutObject(ctx, cfg, part, s3Bucket, s3Prefix+archivePath+filepath.Base(part), storageClass); err != nil {
				//time.Sleep(time.Millisecond * 500)
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

			if variables.CleanupAfterUpload {
				os.Remove(part)
			}
		}

		// Write path to file that records the processed paths
		os.Chdir(currentWorkingDirectory)
		checkIfPathAlreadyProcessed(path, true)
	}

}
