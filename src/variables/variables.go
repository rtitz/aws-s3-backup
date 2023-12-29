package variables

// AWS AUTHENTICATION
var AwsAuthCredentialsFrom string = "awsCliProfile" // "files" or "awsCliProfile"

var AwsCredentialsFile string = "aws-credentials" // Used if AwsAuthCredentialsFrom is "files"
var AwsConfigFile string = "aws-config"           // Used if AwsAuthCredentialsFrom is "files"

var AwsCliProfileDefault string = "default"  // Used if AwsAuthCredentialsFrom is "awsCliProfile"
var AwsCliRegionDefault string = "us-east-1" // Used if AwsAuthCredentialsFrom is "awsCliProfile"
// END OF: AWS AUTHENTICATION

var UploadMethod string = "Disabled"       // PutObject or TransferManager or Disabled
var SplitUploadsEachXMegaBytes int64 = 500 // If TransferManager is used
var CleanupAfterUpload bool = true
var HowToBuildFileSuffix string = "-HowToBuild.txt"

type Tasks struct {
	Tasks []Task `json:"tasks"`
}

type Task struct {
	S3Bucket                  string   `json:"S3Bucket"`
	S3Prefix                  string   `json:"S3Prefix"`
	StorageClass              string   `json:"StorageClass"`
	ArchiveSplitEachMB        string   `json:"ArchiveSplitEachMB"`
	TmpStorageToBuildArchives string   `json:"TmpStorageToBuildArchives"`
	Content                   []string `json:"Content"`
}

type InputData struct {
	Source                    string
	LocalPath                 []string
	RemotePath                []string
	S3Prefix                  string
	S3Bucket                  string
	StorageClass              string
	ArchiveSplitEachMB        string
	TmpStorageToBuildArchives string
	Sha256CheckSum            string
}

func (InputDataF *InputData) AddLocalPath(path string) []string {
	InputDataF.LocalPath = append(InputDataF.LocalPath, path)
	return InputDataF.LocalPath
}
