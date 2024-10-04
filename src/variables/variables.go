package variables

import (
	"time"
)

// App name and version
var AppName string = "AWS-S3-BACKUP"
var AppVersion string = "1.3.0"

// AWS AUTHENTICATION
var AwsAuthCredentialsFrom string = "awsCliProfile" // "files" or "awsCliProfile"
var AwsCredentialsFile string = "aws-credentials"   // Used if AwsAuthCredentialsFrom is "files"
var AwsConfigFile string = "aws-config"             // Used if AwsAuthCredentialsFrom is "files"

var AwsCliProfileDefault string = "default"  // Used if AwsAuthCredentialsFrom is "awsCliProfile"
var AwsCliRegionDefault string = "us-east-1" // Used if AwsAuthCredentialsFrom is "awsCliProfile"
// END OF: AWS AUTHENTICATION

var UploadMethod string = "PutObject" // PutObject or TransferManager or Disabled
// var UploadMethod string = "Disabled"       // DISABLED IS ONLY USED FOR DEBUG AND DEVELOPMENT
var SplitUploadsEachXMegaBytes int64 = 500 // If TransferManager is used
var CleanupAfterUploadDefault bool = true
var HowToBuildFileSuffix string = "-HowToBuild.txt"
var ProcessedTrackingSuffix string = "-Processed.txt"
var ArchiveExtension string = "tar.gz"
var EncryptionExtension string = "enc"
var ChecksumMode = "sha256" // sha256 or md5 / If md5 then the S3 ETag is used

var OutputSeperator = "============================================================================"

type Tasks struct {
	Tasks []Task `json:"tasks"`
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

type InputData struct {
	Source                    string
	LocalPath                 []string
	RemotePath                []string
	S3Prefix                  string
	TrimBeginningOfPathInS3   string
	S3Bucket                  string
	StorageClass              string
	ArchiveSplitEachMB        string
	TmpStorageToBuildArchives string
	CleanupTmpStorage         string
	EncryptionSecret          string
	Sha256CheckSum            string
}

func (InputDataF *InputData) AddLocalPath(path string) []string {
	InputDataF.LocalPath = append(InputDataF.LocalPath, path)
	return InputDataF.LocalPath
}

// =========================================================
// Restore specific

type Contents struct {
	Contents       []Content `json:"Contents"`
	RequestCharged string    `json:"RequestCharged"`
}

type Content struct {
	Key          string    `json:"Key"`
	Size         int64     `json:"Size"`
	StorageClass string    `json:"StorageClass"`
	LastModified time.Time `json:"LastModified"`
}

var JsonOutputFile string = "generated-restore-input.json"
var DefaultDaysRestoreIsAvailable int = 3
var DefaultRetrievalMode string = "bulk" // bulk is the cheapest one (https://docs.aws.amazon.com/AmazonS3/latest/userguide/restoring-objects-retrieval-options.html)

var RestoreNotNeededMessage = "Not needed for this Storage Class"
var RestoreNotInitiatedMessage = "Not initiated"
var RestoreOngoingMessageBulk = "ongoing [ Typically done within 48 hours. (Mode: bulk) ]"
var RestoreOngoingMessageStandard = "ongoing [ Typically done within 12 hours. (Mode: standard) ]"
var RestoreDoneMessage = "restored"

var StorageClassesNeedRestore []string
var FilesNeedingDecryption []string

func init() {
	StorageClassesNeedRestore = []string{
		"DEEP_ARCHIVE",
		"GLACIER",
	}
}
