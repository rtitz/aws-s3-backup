# Example of a restore

**[Back](../README.md)**

**IMPORTANT: THERE IS NOTHING SPECIAL ABOUT THE WAY HOW THIS TOOL STORES DATA IN S3!**\
You can always restore without this tool. Even via the S3 web console a restore via a browser is possible. (It is just less manual work to use this tool, if there are many files.)\
  * See [S3 UserGuide](https://docs.aws.amazon.com/AmazonS3/latest/userguide/)
  * See [S3 UserGuide - Download Objects](https://docs.aws.amazon.com/AmazonS3/latest/userguide/download-objects.html)
  * See [S3 UserGuide - Restoring an archived object](https://docs.aws.amazon.com/AmazonS3/latest/userguide/restoring-objects.html)

---
The following example describes the restore, if you want to use this tool also for a restore.

  * In this example you are in the directoy: /Users/rtitz/
  * The following files are in this directory:
```text
└── aws-s3-backup_macos-arm64 <- This is the aws-s3-backup executable, nothing else is required here.
```

  * Here is how to list  buckets:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"

aws-s3-backup_macos-arm64 -mode restore
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.1.0

Authentication via AWS environment variables... Successful!

MODE: RESTORE

No bucket specified
Here is the list of buckets you can specify with parameter -bucket

cdk-hnb659fds-assets-123456789012-us-east-1
my-s3-backup-bucket
s3-20230312
```

  * Here is how to generate restore JSON file:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX" # Only needed if not already done in previous command
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX" # Only needed if not already done in previous command

aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination restore/
# Other way to generade the restore JSON file, if AWS CLI is installed:
# aws s3api list-objects-v2 --bucket my-s3-backup-bucket > generated-restore-input.json
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.1.0

Authentication via AWS environment variables... Successful!

MODE: RESTORE

backup/Users/rtitz/dir/dir2/new-file.tar.gz
backup/Users/rtitz/test-directory.tar.gz
backup/Users/rtitz/test.file.tar.gz
backup/Users/rtitz/test2.file.tar.gz
backup/my-input.json
backup/my-input.json-Processed.txt

Number of objects returned: 6
More objects available than returned: false

Generated: 'generated-restore-input.json'

Do you want to continue with restore, without editing generated input JSON? [y/N]: n
```

  * You can, if needed, remove objects to be restored from the 'generated-restore-input.json', which exists now in your current directory.
  * Here is how to restore with JSON file:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX" # Only needed if not already done in previous command
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX" # Only needed if not already done in previous command

aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination restore/ -json generated-restore-input.json
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.1.0

Authentication via AWS environment variables... Successful!

MODE: RESTORE

backup/Users/rtitz/dir/dir2/new-file.tar.gz 
 * Size: 113.00 B
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

backup/Users/rtitz/test-directory.tar.gz 
 * Size: 2.30 KB
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

backup/Users/rtitz/test.file.tar.gz 
 * Size: 586.00 B
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

backup/Users/rtitz/test2.file.tar.gz 
 * Size: 500.00 B
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

backup/my-input.json 
 * Size: 557.00 B
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

backup/my-input.json-Processed.txt 
 * Size: 518.00 B
 * StorageClass: STANDARD
 * RestoreStatus: Not needed for this Storage Class
 Downloading ...
Download: OK

2024/01/31 20:17:28 Done! 
```

  * Files now locate in your current directory in folder restore/
  * If objects are in DEEP_ARCHIVE storage class, it will guide you through the restore process. Output will be slightly different.

**[Back](../README.md)**