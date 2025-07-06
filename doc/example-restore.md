# 📥 Example of a restore

**[Back](../README.md)**

**⚠️ IMPORTANT: THERE IS NOTHING SPECIAL ABOUT THE WAY HOW THIS TOOL STORES DATA IN S3!**\
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

  * 📁 Here is how to list buckets:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"

aws-s3-backup_macos-arm64 -mode restore
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.3.1

MODE: RESTORE

Available buckets:
  cdk-hnb659fds-assets-123456789012-us-east-1
  my-s3-backup-bucket
  s3-20230312
```

  * 📄 Here is how to generate restore JSON file:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX" # Only needed if not already done in previous command
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX" # Only needed if not already done in previous command

aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination restore/
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.3.1

MODE: RESTORE

Generated: 'generated-restore-input.json'

Downloading: backup/Users/rtitz/dir/dir2/new-file.tar.gz
Downloading: backup/Users/rtitz/test-directory.tar.gz
Downloading: backup/Users/rtitz/test.file.tar.gz
Downloading: backup/Users/rtitz/test2.file.tar.gz
Downloading: backup/my-input.json
Downloading: backup/my-input.json-Processed.txt

Scanning for split archives to combine...
Archive combination complete
Restore complete!
```

  * ▶️ Here is how to restore with JSON file:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX" # Only needed if not already done in previous command
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX" # Only needed if not already done in previous command

aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination restore/ -json generated-restore-input.json
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.3.1

MODE: RESTORE

Downloading: backup/Users/rtitz/dir/dir2/new-file.tar.gz
Downloading: backup/Users/rtitz/test-directory.tar.gz
Downloading: backup/Users/rtitz/test.file.tar.gz
Downloading: backup/Users/rtitz/test2.file.tar.gz
Downloading: backup/my-input.json
Downloading: backup/my-input.json-Processed.txt

Scanning for split archives to combine...
Archive combination complete
Restore complete!
```

  * 📁 Files now locate in your current directory in folder restore/
  * ✅ **Split archives are automatically combined** - no manual extraction needed
  * 🧊 If objects are in DEEP_ARCHIVE storage class, it will guide you through the restore process

**[Back](../README.md)**