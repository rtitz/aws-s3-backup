# AWS S3 Backup
This is a tool create in Go to backup data to AWS S3.\
It will create tar.gz archives out of the paths you input in input.json (see 'example-input.json')\
Then it will split these archives into smaller chunks, if they are large (configurable, see 'example-input.json')\
It will upload the archives to the S3 bucket speciefied in input.json file and store it in the StorageClass choosen by you.\
Checksums are caclulated and verifed to ensure the integrity of uploaded data to S3.\

**There are charges from AWS for the S3 usage!**
This [link](https://calculator.aws/) can help you calculating the chages


## Requirements for AWS S3 Backup
 * Ensure that the AWS CLI is installed and configured! (https://docs.aws.amazon.com/cli/)
 * AWS CLI configuration (https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)
 * An existing S3 Bucket within your AWS account
 * An IAM user with access key (and secret access key)
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
                "s3:ListBucketVersions",
                "s3:ListAllMyBuckets"
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


## Usage in general
  * See 'example-input.json' and build your own input.json
  * See the help
```
aws-s3-backup_macos-arm64 -help
```

## Backup your data
  * Execute with **your** input.json
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json
```

  * Execute with **your** input.json and specify an AWS cli profile and AWS region
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json -profile test -region eu-central-1
```
**NOTE:** Default AWS CLI profile is: 'default' and default AWS region is 'us-east-1'.

## Restore your backup
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

## Command line parameters

### json
 * Specify the path to input.json (see below)

### profile
  * Specify the name of the AWS CLI profile
  * See: [AWS Documentation about AWS CLI Configuration](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html)

### region
  * The AWS region must match the region in which your destination S3 bucket lives
  * See [AWS Documentation about S3 Buckets](https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingBucket.html)


## input.json

### StorageClasses

The following values for 'StorageClass' in JSON input file are supported:
  * STANDARD
  * STANDARD_IA
  * DEEP_ARCHIVE
  * GLACIER_IR
  * GLACIER
  * REDUCED_REDUNDANCY (**NOT** recommended! Do not use it!)

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

---
## [Build it on your own from source](doc/build.md)
