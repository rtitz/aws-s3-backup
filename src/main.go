package main

import (
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

func buildArchive(files []string, archiveFile string) error {
	// Files which to include in the tar.gz archive
	//files := []string{"example.txt", "test/test.txt"}
	//files := []string{"build.sh", "go.mod", "go.sum", "main.go"}
	//archiveFile := "output.tar.gz"

	log.Println("Creating archive...")
	// Create output file
	out, err := os.Create(archiveFile)
	if err != nil {
		log.Fatalln("Error writing archive:", err)
	}
	defer out.Close()

	// Create the archive and write the output to the "out" Writer
	err = fileUtils.CreateArchive(files, out)
	if err != nil {
		os.Remove(archiveFile)
		log.Fatalln("Error creating archive:", err)
	}

	fmt.Println("Archive created successfully")
	// END OF: TAR.GZ
	return nil
}

func createTxtHowToCombineSplittedArchive(archive string, listOfParts []string) error {
	var parts string
	for i, part := range listOfParts {
		part = filepath.Base(part)
		if i == 0 { // First iteration in this loop; do not add a space in the beginning
			parts = parts + part
		} else {
			parts = parts + " " + part
		}
	}
	fmt.Printf("cat %s > %s\n", parts, archive)
	// TODO: Write to a file
	return nil
}

func main() {

	// Define and check parameters
	inputFile := flag.String("json", "", "JSON file that contains the input parameters")
	// TODO: PARSE AWS PROFILE
	// TODO: PARSE AWS REGION
	flag.Parse()

	if *inputFile == "" {
		fmt.Printf("Parameter missing / wrong! Try again and specify the following parameters.\n\nParameter list:\n\n")
		flag.PrintDefaults()
		fmt.Printf("\n")
		os.Exit(999)
	}
	// End of: Define and check parameters

	// Create new session
	ctx := context.TODO()
	cfg := awsUtils.CreateAwsSession(ctx)

	currentWorkingDirectory, _ := os.Getwd()

	/*fileContent, err := getInputOld()
	if err != nil {
		log.Fatalf("FAILED TO GET INPUT: %v", err)
	}*/

	os.Chdir(currentWorkingDirectory)
	tasks, err := getInput(*inputFile)
	if err != nil {
		log.Fatalf("FAILED TO PARSE %s : %v", *inputFile, err)
	}
	/*for _, task := range tasks {
		fmt.Println(task.Source)
		fmt.Println(task.LocalPath)
		fmt.Println(task.RemotePath)
		fmt.Println(task.S3Prefix)
		fmt.Println(task.S3Bucket)
		fmt.Println(task.Sha256CheckSum)
		fmt.Println()
	}*/

	//for _, item := range fileContent {
	for _, task := range tasks {
		path := task.Source
		s3Bucket := task.S3Bucket
		s3Prefix := filepath.Clean(task.S3Prefix)
		archiveSplitEachMB, _ := strconv.Atoi(task.ArchiveSplitEachMB)
		tmpStorageToBuildArchives := task.TmpStorageToBuildArchives
		var storageClass types.StorageClass

		switch task.StorageClass {
		case "STANDARD":
			storageClass = types.StorageClassStandard
		case "ONEZONE_IA":
			storageClass = types.StorageClassOnezoneIa
		case "DEEP_ARCHIVE":
			storageClass = types.StorageClassDeepArchive
		case "REDUCED_REDUNDANCY":
			storageClass = types.StorageClassReducedRedundancy
		default:
			storageClass = types.StorageClassStandard
		}
		storageClass = types.StorageClassStandard

		archiveTmp := filepath.Clean(tmpStorageToBuildArchives)
		os.MkdirAll(archiveTmp, os.ModePerm)

		c := []string{path}

		archivePath := filepath.Clean(filepath.Dir(path)) + "/"
		archive := filepath.Base(path) + ".tar.gz"
		archive = strings.ReplaceAll(archive, " ", "-") // REPLACE SPACE WITH -
		fullArchivePath := archiveTmp + "/" + archive

		buildArchive(c, fullArchivePath)
		listOfParts, err := fileUtils.SplitArchive(fullArchivePath, int64(archiveSplitEachMB))
		if err != nil {
			log.Fatalf("FAILED TO SPLIT: %v", err)
		}

		if len(listOfParts) > 1 { // Remove original (unsplitted) archive, if it has been splitted
			os.Remove(fullArchivePath)
			createTxtHowToCombineSplittedArchive(archive, listOfParts)
		}

		for _, part := range listOfParts {
			fmt.Println("S3 path: " + s3Prefix + archivePath + filepath.Base(part))
			fmt.Println("Local path: " + part)
			fmt.Println("StorageClass: " + storageClass)
			log.Printf("Upload...")
			if err := awsUtils.PutObject(ctx, cfg, part, s3Bucket, s3Prefix+archivePath+filepath.Base(part), storageClass); err != nil {
				log.Fatalf("UPLOAD FAILED! %v", err)
			}
			log.Printf(" DONE!\n")
			if variables.CleanupAfterUpload {
				os.Remove(part)
			}
		}

	}

}
