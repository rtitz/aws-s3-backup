# Example of a backup

**[Back](../README.md)**

  * In this example you are in the directoy: /Users/rtitz/
  * The following files are in this directory:
```text
├── aws-s3-backup_macos-arm64 <- This is the aws-s3-backup executable
├── dir <- This is a directory containing files and sub directories
│   ├── dir1 <- This is an empty directory
│   ├── dir2 <- This is a directory containing files
│   │   └── new-file
│   ├── file1
│   ├── file2
│   ├── file3
├── fileUtils <- This is a directory containing files 
│   ├── fileInfo.go
│   ├── fileSizeUnit.go
│   ├── splitArchives.go
│   └── tar.go
├── my-input.json <- This is the input.json for aws-s3-backup
├── test-directory <- This is a directory containing files and sub directories
│   ├── NEW
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── notifications
│   │   └── mail.go
│   └── test.file
├── test.file <- This is a file
├─── test2.file <- This is another file
└── tmp <- This is an empty directory
```

  * Here is how your my-input.json could look like:
```json
{
    "tasks": [
        {
            "S3Bucket": "my-s3-backup-bucket",
            "S3Prefix": "backup",
            "StorageClass": "DEEP_ARCHIVE",
            "ArchiveSplitEachMB": "250",
            "TmpStorageToBuildArchives": "/Users/rtitz/tmp",
            "CleanupTmpStorage": "true",
            "Content": [
                "/Users/rtitz/test-directory",
                "/Users/rtitz/test.file",
                "/Users/rtitz/test2.file",
                "/Users/rtitz/dir/dir2/new-file"
            ]
        }
    ]
}
```

  * Here is how to execute this:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"

aws-s3-backup_macos-arm64 -json my-input.json 
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.1.0

Authentication via AWS environment variables... Successful!

MODE: BACKUP

2024/01/31 19:46:52 Creating archive...
2024/01/31 19:46:52 Add test-directory/NEW to archive...
2024/01/31 19:46:52 Add test-directory/go.mod to archive...
2024/01/31 19:46:52 Add test-directory/go.sum to archive...
2024/01/31 19:46:52 Add test-directory/main.go to archive...
2024/01/31 19:46:52 Add test-directory/notifications/mail.go to archive...
2024/01/31 19:46:52 Add test-directory/test.file to archive...
2024/01/31 19:46:52 Archive created successfully
2024/01/31 19:46:52 S3 path: s3://my-s3-backup-bucket/backup/Users/rtitz/test-directory.tar.gz
2024/01/31 19:46:52 Local path: /Users/rtitz/tmp/test-directory.tar.gz
2024/01/31 19:46:52 Size: 2.30 KB
2024/01/31 19:46:52 StorageClass: STANDARD
2024/01/31 19:46:52 Upload...
2024/01/31 19:46:53 Checksum eKAudS57/6mBVMtqKKmeocy18rd4sea8prI3jiSX3Q8= : OK
2024/01/31 19:46:53  100.00 % (1/1) UPLOADED - DONE!
============================================================================

2024/01/31 19:46:53 Creating archive...
2024/01/31 19:46:53 Add test.file to archive...
2024/01/31 19:46:53 Archive created successfully
2024/01/31 19:46:53 S3 path: s3://my-s3-backup-bucket/backup/Users/rtitz/test.file.tar.gz
2024/01/31 19:46:53 Local path: /Users/rtitz/tmp/test.file.tar.gz
2024/01/31 19:46:53 Size: 586.00 B
2024/01/31 19:46:53 StorageClass: STANDARD
2024/01/31 19:46:53 Upload...
2024/01/31 19:46:53 Checksum qMxdNnoj0VT97SAAEFd/kcvzeFg9wAjRMKlJd/b+QZk= : OK
2024/01/31 19:46:53  100.00 % (1/1) UPLOADED - DONE!
============================================================================

2024/01/31 19:46:53 Creating archive...
2024/01/31 19:46:53 Add test2.file to archive...
2024/01/31 19:46:53 Archive created successfully
2024/01/31 19:46:53 S3 path: s3://my-s3-backup-bucket/backup/Users/rtitz/test2.file.tar.gz
2024/01/31 19:46:53 Local path: /Users/rtitz/tmp/test2.file.tar.gz
2024/01/31 19:46:53 Size: 500.00 B
2024/01/31 19:46:53 StorageClass: STANDARD
2024/01/31 19:46:53 Upload...
2024/01/31 19:46:54 Checksum y3YpCD2t5MXFQDiMx/ccVo+7Dm1abQbwpDu9sOkoS3k= : OK
2024/01/31 19:46:54  100.00 % (1/1) UPLOADED - DONE!
============================================================================

2024/01/31 19:46:54 Creating archive...
2024/01/31 19:46:54 Add new-file to archive...
2024/01/31 19:46:54 Archive created successfully
2024/01/31 19:46:54 S3 path: s3://my-s3-backup-bucket/backup/Users/rtitz/dir/dir2/new-file.tar.gz
2024/01/31 19:46:54 Local path: /Users/rtitz/tmp/new-file.tar.gz
2024/01/31 19:46:54 Size: 113.00 B
2024/01/31 19:46:54 StorageClass: STANDARD
2024/01/31 19:46:54 Upload...
2024/01/31 19:46:54 Checksum NXnzWWlpXyS5PCyWZ3UGiL/E1p7eDCDO9ZpszcMT1cY= : OK
2024/01/31 19:46:54  100.00 % (1/1) UPLOADED - DONE!
============================================================================

2024/01/31 19:46:54 Upload of additional JSON files...
2024/01/31 19:46:55 Checksum cBYlHIN2TMmJpriF+yXHWdWQyvCHQtzFZ3eAZrkQr4o= : OK
2024/01/31 19:46:55 Checksum 5zkLtNqJQOfrhHXg/7Hbya2QaiA+IQYOIvRSGuWDXsE= : OK
2024/01/31 19:46:55 Upload of additional JSON files: OK
```

  * Everything in 'Content' part of the my-input.json is uploaded to S3 bucket "my-s3-backup-bucket" and placed with their full paths into folder "backup" in this bucket.

**[Back](../README.md)**