package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type Config struct {
	Mode                       string
	InputFile                  string
	AWSProfile                 string
	AWSRegion                  string
	Bucket                     string
	Prefix                     string
	DownloadLocation           string
	RetrievalMode              string
	RestoreWithoutConfirmation bool
	AutoRetryDownloadMinutes   int64
	RestoreExpiresAfterDays    int64
	DryRun                     bool
}

type Task struct {
	S3Bucket                  string   `json:"S3Bucket"`
	S3Prefix                  string   `json:"S3Prefix"`
	TrimBeginningOfPathInS3   string   `json:"TrimBeginningOfPathInS3"`
	StorageClass              string   `json:"StorageClass"`
	ArchiveSplitEachMB        string   `json:"ArchiveSplitEachMB"`
	TmpStorageToBuildArchives string   `json:"TmpStorageToBuildArchives"`
	CleanupTmpStorage         string   `json:"CleanupTmpStorage"`
	EncryptionSecret          string   `json:"EncryptionSecret"`
	Content                   []string `json:"Content"`
}

type Tasks struct {
	Tasks []Task `json:"tasks"`
}

func (c *Config) Validate() error {
	if c.Mode != "backup" && c.Mode != "restore" {
		return fmt.Errorf("❌ invalid mode '%s', must be 'backup' or 'restore'", c.Mode)
	}
	if c.Mode == "backup" && c.InputFile == "" {
		return fmt.Errorf("❌ json parameter required for backup mode")
	}
	if c.AutoRetryDownloadMinutes > 0 && c.AutoRetryDownloadMinutes < 60 {
		return fmt.Errorf("❌ autoRetryDownloadMinutes must be 60 or higher")
	}
	if c.RestoreExpiresAfterDays < 1 {
		return fmt.Errorf("❌ restoreExpiresAfterDays must be 1 or higher")
	}
	return nil
}

func LoadTasks(inputFile string) ([]Task, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to read input file: %w", err)
	}

	var tasks Tasks
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("❌ failed to parse JSON: %w", err)
	}

	if len(tasks.Tasks) == 0 {
		return nil, fmt.Errorf("❌ no tasks found in input file")
	}

	return tasks.Tasks, nil
}

func ParseStorageClass(storageClass string) types.StorageClass {
	switch storageClass {
	case "STANDARD":
		return types.StorageClassStandard
	case "STANDARD_IA":
		return types.StorageClassStandardIa
	case "DEEP_ARCHIVE":
		return types.StorageClassDeepArchive
	case "GLACIER_IR":
		return types.StorageClassGlacierIr
	case "GLACIER":
		return types.StorageClassGlacier
	case "REDUCED_REDUNDANCY":
		return types.StorageClassReducedRedundancy
	default:
		return types.StorageClassStandard
	}
}

func ParseCleanupFlag(cleanup string) bool {
	switch strings.ToLower(cleanup) {
	case "true", "yes":
		return true
	case "false", "no":
		return false
	default:
		return true
	}
}

func ParseArchiveSplitMB(splitMB string) (int64, error) {
	if splitMB == "" {
		return 250, nil
	}
	mb, err := strconv.ParseInt(splitMB, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("❌ invalid ArchiveSplitEachMB value: %w", err)
	}
	if mb <= 0 {
		return 0, fmt.Errorf("❌ ArchiveSplitEachMB must be positive")
	}
	return mb, nil
}