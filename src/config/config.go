package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Application constants
const (
	AppName    = "AWS-S3-BACKUP"
	AppVersion = "1.3.4"
)

// Default configuration values
const (
	DefaultMode                    = "backup"
	DefaultAWSProfile              = "default"
	DefaultAWSRegion               = "us-east-1"
	DefaultRetrievalMode           = "bulk"
	DefaultRestoreExpiresAfterDays = 3
	DefaultArchiveSplitMB          = 250
	DefaultCleanupTmpStorage       = true
)

// File extensions
const (
	ArchiveExtension = "tar.gz"
	EncryptionExt    = "enc"
)

// S3 lifecycle defaults
const (
	DefaultAbortIncompleteMultipartUploadDays = 2
	DefaultNoncurrentVersionExpirationDays    = 1
)

// Config holds the application configuration
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

// Task represents a single backup task from JSON input
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

// Tasks wraps multiple Task objects for JSON parsing
type Tasks struct {
	Tasks []Task `json:"tasks"`
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if err := c.validateMode(); err != nil {
		return err
	}
	if err := c.validateBackupRequirements(); err != nil {
		return err
	}
	if err := c.validateRetrySettings(); err != nil {
		return err
	}
	return c.validateRestoreSettings()
}

// validateMode checks if the operation mode is valid
func (c *Config) validateMode() error {
	if c.Mode != "backup" && c.Mode != "restore" {
		return fmt.Errorf("❌ invalid mode '%s', must be 'backup' or 'restore'", c.Mode)
	}
	return nil
}

// validateBackupRequirements checks backup-specific configuration
func (c *Config) validateBackupRequirements() error {
	if c.Mode == "backup" && c.InputFile == "" {
		return fmt.Errorf("❌ json parameter required for backup mode")
	}
	return nil
}

// validateRetrySettings checks auto-retry configuration
func (c *Config) validateRetrySettings() error {
	if c.AutoRetryDownloadMinutes > 0 && c.AutoRetryDownloadMinutes < 5 {
		return fmt.Errorf("❌ autoRetryDownloadMinutes must be 5 or higher")
	}
	return nil
}

// validateRestoreSettings checks restore-specific configuration
func (c *Config) validateRestoreSettings() error {
	if c.RestoreExpiresAfterDays < 1 {
		return fmt.Errorf("❌ restoreExpiresAfterDays must be 1 or higher")
	}
	return nil
}

// LoadTasks reads and parses tasks from a JSON file
func LoadTasks(inputFile string) ([]Task, error) {
	data, err := readInputFile(inputFile)
	if err != nil {
		return nil, err
	}

	tasks, err := parseTasksJSON(data)
	if err != nil {
		return nil, err
	}

	if len(tasks.Tasks) == 0 {
		return nil, fmt.Errorf("❌ no tasks found in input file")
	}

	return tasks.Tasks, nil
}

// readInputFile reads the JSON input file
func readInputFile(inputFile string) ([]byte, error) {
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return nil, fmt.Errorf("❌ failed to read input file: %w", err)
	}
	return data, nil
}

// parseTasksJSON parses JSON data into Tasks struct
func parseTasksJSON(data []byte) (*Tasks, error) {
	var tasks Tasks
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("❌ failed to parse JSON: %w", err)
	}
	return &tasks, nil
}

// ParseStorageClass converts string to AWS S3 storage class type
func ParseStorageClass(storageClass string) types.StorageClass {
	storageClassMap := map[string]types.StorageClass{
		"STANDARD":                   types.StorageClassStandard,
		"STANDARD_IA":                types.StorageClassStandardIa,
		"ONEZONE_IA":                 types.StorageClassOnezoneIa,
		"DEEP_ARCHIVE":               types.StorageClassDeepArchive,
		"GLACIER_IR":                 types.StorageClassGlacierIr,
		"GLACIER":                    types.StorageClassGlacier,
		"GLACIER_FLEXIBLE_RETRIEVAL": types.StorageClassGlacier,
		"EXPRESS_ONEZONE":            types.StorageClassExpressOnezone,
		"REDUCED_REDUNDANCY":         types.StorageClassReducedRedundancy,
	}

	if class, exists := storageClassMap[storageClass]; exists {
		return class
	}
	return types.StorageClassStandard
}

// ParseCleanupFlag converts string to boolean for cleanup setting
func ParseCleanupFlag(cleanup string) bool {
	switch strings.ToLower(cleanup) {
	case "true", "yes":
		return true
	case "false", "no":
		return false
	default:
		return DefaultCleanupTmpStorage
	}
}

// ParseArchiveSplitMB converts string to int64 for archive split size
func ParseArchiveSplitMB(splitMB string) (int64, error) {
	if splitMB == "" {
		return DefaultArchiveSplitMB, nil
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
