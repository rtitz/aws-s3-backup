package backupUtils

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/rtitz/aws-s3-backup/variables"
)

// Reads the input.json to get the backup input paramters
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
				Source:                    c,
				LocalPath:                 []string{},
				RemotePath:                []string{},
				S3Prefix:                  tasks.Tasks[i].S3Prefix,
				TrimBeginningOfPathInS3:   tasks.Tasks[i].TrimBeginningOfPathInS3,
				S3Bucket:                  tasks.Tasks[i].S3Bucket,
				StorageClass:              tasks.Tasks[i].StorageClass,
				ArchiveSplitEachMB:        tasks.Tasks[i].ArchiveSplitEachMB,
				TmpStorageToBuildArchives: tasks.Tasks[i].TmpStorageToBuildArchives,
				CleanupTmpStorage:         tasks.Tasks[i].CleanupTmpStorage,
				EncryptionSecret:          tasks.Tasks[i].EncryptionSecret,
				Sha256CheckSum:            "0",
			}
			input = append(input, newEntry)
		}
	}
	return input, nil
}
