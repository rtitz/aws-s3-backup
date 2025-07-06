# AWS S3 Backup
This is a robust tool created in Go to backup data to [Amazon S3](https://docs.aws.amazon.com/AmazonS3/latest/userguide/).\
It creates tar.gz archives from specified paths, splits large archives into configurable chunks, and uploads them to S3 with your chosen storage class.\
Features comprehensive error handling, input validation, and checksum verification to ensure data integrity.\

## ✨ Key Features
- 🛡️ **Reliable Error Handling**: Graceful error recovery instead of crashes
- ✅ **Input Validation**: Comprehensive configuration validation
- 🏗️ **Modular Architecture**: Clean separation of concerns for maintainability
- 🧪 **Unit Testing**: Test coverage for critical components
- ⚙️ **Flexible Configuration**: Support for multiple storage classes and encryption
- 🔗 **Automatic Archive Combination**: Split archives are automatically combined during restore
- 🚀 **Multi-Core Compression**: Uses parallel gzip compression for faster archive creation
- 📋 **Dry-Run Mode**: Test backups locally without uploading to S3
- 📊 **Summary Reports**: Detailed operation summaries with success/failure counts

  * See: [Example of a backup](doc/example-backup.md)
  * See: [Example of a restore](doc/example-restore.md)

**There are charges from AWS for the S3 usage!**
This [link](https://calculator.aws/) can help you calculating the chages


## 📋 Requirements for AWS S3 Backup
 * There are CloudFormation templates in directory "cloudformation", that can be used to deploy the requirements

Here are the requirements in detail, in case you do not want to use the provided CloudFormation templates:
 * An existing S3 Bucket within your AWS account
 * An IAM user with access key (and secret access key) OR configured AWS CLI profile OR programmatic access credentials from AWS IAM Identity Center
 * Your IAM user should have the following permissions (can be a separate attached IAM policy):
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:DeleteObject",
                "s3:DeleteObjectVersion",
                "s3:PutObject",
                "s3:GetObject",
                "s3:GetObjectVersion",
                "s3:RestoreObject",
                "s3:ListBucket",
                "s3:ListBucketVersions"
            ],
            "Resource": [
              "arn:aws:s3:::NAME-OF-YOUR-S3-BUCKET",
              "arn:aws:s3:::NAME-OF-YOUR-S3-BUCKET/*"
            ]
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListAllMyBuckets"
            ],
            "Resource": [
              "*"
            ]
        }
    ]
}
```


## 🚀 Usage in general
  * In the directory 'bin/' you will find pre-compiled executable binaries for different operating systems. You can just execute them in a terminal.
  * See 'example-input.json' and build your own input.json
  * See: [Example of a backup](doc/example-backup.md)
  * See: [Example of a restore](doc/example-restore.md)
  * See the help

```
aws-s3-backup_macos-arm64 -help
```

## 🛠️ Development

### 🔨 Building from Source
```bash
# Build the application
cd src && go build -o ../bin/aws-s3-backup .

# Run tests
cd src && go test ./...

# Run tests with coverage
cd src && go test -coverprofile=coverage.out ./...
cd src && go tool cover -html=coverage.out -o coverage.html

# Build for all platforms
cd src && ./build.sh
```

### 📁 Code Structure
- `config/` - Configuration management and validation
- `services/` - Business logic services (backup, restore)
- `utils/` - Consolidated utilities (AWS, files, crypto, archive)
- `tests/` - Unit tests

## 💾 Backup your data
  * Execute with **your** input.json
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json
```

  * Execute with **your** input.json and specify an AWS cli profile and AWS region
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json -profile test -region eu-central-1
```

  * Test backup locally without uploading to S3 (dry-run mode)
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json -dryrun
```
**NOTE:** Default AWS CLI profile is: 'default' and default AWS region is 'us-east-1'.

## 📥 Restore your backup
  * List your buckets
```
aws-s3-backup_macos-arm64 -mode restore 
```

  * Restore everything out of an bucket (and generate an input json for restore called 'generated-restore-input.json' in your current directory)
```
aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination Downloads/restore/
``` 

```
aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -json generated-restore-input.json
```

**Note**: Split archives are automatically detected and combined back into single files during restore. No manual intervention required.

## ⚙️ Command line parameters

### json
 * Specify the path to input.json (see below)

### profile (if AWS CLI is installed)
  * Specify the name of the AWS CLI profile
  * Default profiles beeing used: default
  * If AWS CLI is not installed, you can use environment variables for authentication
  * You can list your profiles with: 'aws configure list-profiles' command
  * See: [AWS Documentation about AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

### region
  * The AWS region must match the region in which your destination S3 bucket lives
  * Default is us-east-1
  * See [AWS Documentation about S3 Buckets](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingBucket.html)

### mode
  * Operation mode (backup or restore)
  * Default is backup

### bucket (only used for restore)
  * If mode is 'restore' you have to specify the bucket, in which your data is stored.
  * Without this parameter you will get a list of Buckets printed.

### prefix (only used for restore)
  * Specify a prefix to limit object list to objects in a specific 'folder' in the S3 bucket.
  * Example: 'archive'

### destination (only used for restore)
  * Path / directory the restore should be downloaded to. Download location.
  * Example: 'restore/'

### retrievalMode (only used for restore)
  * Mode of retrieval (bulk or standard)
  * Used for objects stored Glacier / archive storage classes.
  * bulk takes up to 48 hours / standard takes up to 12 hours, but is more expensive than bulk
  * Default is bulk

### restoreExpiresAfterDays (only used for restore)
  * Days that a restore from DeepArchive storage classes is available in (more expensive) Standard storage class
  * Default is 3 (days)

### autoRetryDownloadMinutes (only used for restore)
  * If a restore from Glacier / archive storage classes to standard storage class is needed and this is for example 60 it will retry the download every 60 minutes.
  * If this parameter is specified, restores will be done without confirmation! (See: restoreWithoutConfirmation)

### restoreWithoutConfirmation (only used for restore)
  * Restore objects from Glacier / archive storage classes to standard storage class has to be confirmed per object. If this parameter is specified, restores will be done without confirmation!
  * By default this parameter is not specified

### dryrun
  * Test mode that performs all backup operations except S3 uploads
  * Creates archives, splits files, encrypts data, but skips uploading to S3
  * Useful for testing configurations and measuring local performance
  * No AWS credentials required in dry-run mode
  * Shows what would be uploaded with detailed logging


## 📄 The 'input.json' file for backups

### 🗄️ StorageClasses

The following values for 'StorageClass' in JSON input file are supported:
  * STANDARD
  * STANDARD_IA
  * DEEP_ARCHIVE
  * GLACIER_IR
  * GLACIER
  * REDUCED_REDUNDANCY (**NOT recommended!** Do not use it!)

**For the different StorageClasses different pricing and different minimum storage duration applies!\
Depending on the StorageClass you choose, there is maybe a retrieval charge for your data etc.\
For download (restore) from S3 there is always a fee for traffic out of AWS to the Internet. (As of 2024 this is USD 0.09 per GB + Tax)**

For more info about the different StorageClasses and AWS S3 pricing in general see:
 * [AWS Documentation: Amazon S3 Storage Classes](https://aws.amazon.com/s3/storage-classes/)
 * [AWS Documentation: Using Amazon S3 storage classes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/storage-class-intro.html)
 * [AWS S3 Pricing](https://aws.amazon.com/s3/pricing/)


### TmpStorageToBuildArchives variable
  * This path will be used to build the tar.gz archives and (if needed) to split them into smaller chunks
  * The files stored here are the exact objects uploaded to S3 (verified during upload with checksums to ensure the data integrity)
  * Ensure that you have enough free space here

### CleanupTmpStorage variable
 * Default value (also if unset!) is: True
 * If you set this to False, it will keep all data in TmpStorageToBuildArchives path after uploading to S3. (see above)
 * You could for example verify you can successfully extract the archives or rebuild them if splitted
 * Technically there is no need to disable the CleanupTmpStorage since this data is stored in your S3 bucket

### TrimBeginningOfPathInS3 variable
  * Default value (also if unset!) is: "" (empty)
  * If your content path is for example: "/home/rtitz/tmp/pico/" and TrimBeginningOfPathInS3 is for example: "/home/rtitz/" then "/home/rtitz/" will be removed (trimmed) from the S3 path. (Result in S3 will be: s3://my-s3-backup-bucket/backup/tmp/pico.tar.gz)

### EncryptionSecret variable
  * Default value (also if unset!) is: "" (Encryption disabled / Nothing will be encrypted)
  * If you set a value, this is going to be your secret used to encrypt the archive (or archive parts) before upload. (AES-256-GCM)
  * During restore you will be asked for the secret to decrypt the file(s)
  * 🔒 **Password Requirements for Security:**
    * Minimum 12 characters (16+ recommended)
    * At least one uppercase letter (A-Z)
    * At least one lowercase letter (a-z)
    * At least one number (0-9)
    * At least one special character (!@#$%^&*)
    * No common dictionary words or patterns
    * Example passwords from documentation are blocked for security

## 🔐 Authentication via environment variables (instead of AWS CLI)
  * Do not specify the parameter -profile
  * If you sign in via the AWS IAM Identity Center, you will find the button 'Command line or programmatic access', you can copy the AWS environment variable commands from here and execute aws-s3-backup tool afterwards.
  * Before executing aws-s3-backup tool, set environment variables as follows (If not copied from AWS IAM Identity Center sign in page):

Linux and macOS:
```
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"
./aws-s3-backup
```

Windows PowerShell:
```
$Env:AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
$Env:AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"
aws-s3-backup.exe
```

  * HINT: You can create an IAM User in the AWS IAM Console and an Access Key for this user to get AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY. (There is also a CloudFormation template to create the user.)
  * **NEVER SHARE THESE KEYS WITH OTHERS!**

## 🛡️ Error Handling
The application now features robust error handling:
- **Graceful failures**: No more unexpected crashes
- **Detailed error messages**: Clear indication of what went wrong with ❌ prefix
- **Input validation**: Configuration errors caught early
- **Password validation**: Strong encryption password requirements enforced
- **Retry logic**: Network failures handled appropriately
- **Operation summaries**: Comprehensive reports showing what succeeded, failed, or was skipped

## 🧪 Testing
Run the test suite to verify functionality:
```bash
cd src
go test ./...
```

For coverage reports:
```bash
cd src && go test -coverprofile=coverage.out ./...
cd src && go tool cover -html=coverage.out -o coverage.html
```

---
## [Build it on your own from source](doc/build.md)
