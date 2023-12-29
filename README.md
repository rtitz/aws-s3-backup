# AWS S3 Backup
This is a tool create in Go to backup data to AWS S3.\
It will create tar.gz archives out of the paths you input in input.json (see 'example-input.json')\
Then it will split these archives into smaller chunks, if they are large (configurable, see 'example-input.json')\
It will upload the archives to the S3 bucket speciefied in input.json file and store it in the StorageClass choosen by you.

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
            "Action": [
                "s3:Put*",
            ],
            "Resource": "arn:aws:s3:::NAME-OF-YOUR-S3-BUCKET/*",
            "Effect": "Allow"
        }
    ]
}
```


## Usage
  * See 'example-input.json' and build your own input.json
  * See the help
```
aws-s3-backup_macos-arm64 -help
```

  * Execute with **your** input.json
```
aws-s3-backup_macos-arm64 -json ~/tmp/input.json
```

## StorageClasses (for input.json)

The following values for 'StorageClass' in JSON input file are supported:
  * STANDARD
  * STANDARD_IA
  * DEEP_ARCHIVE
  * GLACIER_IR
  * GLACIER
  * REDUCED_REDUNDANCY (**NOT** recommended! Do not use it!)


**For the different StorageClasses different pricing and different minimum storage duration applies!\
Depending on the StorageClass you choose, there is maybe a retrieval charge for your data etc.**

For more info about the different StorageClasses see:
 * [AWS Documentation: Amazon S3 Storage Classes](https://aws.amazon.com/s3/storage-classes/)
 * [AWS Documentation: Using Amazon S3 storage classes](https://docs.aws.amazon.com/AmazonS3/latest/userguide/storage-class-intro.html)

---
## [Build it on your own from source](doc/build.md)
