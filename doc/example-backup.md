# 💾 Example of a backup

**[Back](../README.md)**

  * In this example you are in the directoy: /Users/rtitz/
  * The following files are in this directory:
```text
├── ./aws-s3-backup_macos-arm64 <- This is the aws-s3-backup executable
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

  * 📄 Here is how your my-input.json could look like:
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

  * ▶️ Here is how to execute this:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX"

# Normal backup
./aws-s3-backup_macos-arm64 -json my-input.json

# Or test with dry-run (no S3 upload)
./aws-s3-backup_macos-arm64 -json my-input.json -dryrun 
```

  * 📝 Output will be similar to:
```
AWS-S3-BACKUP 1.4.1

MODE: BACKUP
REGION: us-east-1

📦 Creating archive: test-directory.tar.gz
➡️ Adding to archive: test-directory/NEW
➡️ Adding to archive: test-directory/go.mod
✅ Archive created successfully: test-directory.tar.gz
⬆️ Uploading (1/4): test-directory.tar.gz (2.30 KB)
✅ Upload successful: test-directory.tar.gz

==================================================
📊 BACKUP SUMMARY
==================================================
✅ Successfully uploaded: 4
📁 Total files processed: 5
💾 Total data processed: 3.67 KB
⏱️ Preparation time: 1.2s
⏱️ Upload time: 2.3s
⏱️ Total time: 3.5s

🎉 Backup completed successfully!
==================================================
```

  * 📋 Dry-run output would show:
```
AWS-S3-BACKUP 1.4.1

MODE: BACKUP
REGION: DRY-RUN

📦 Creating archive: test-directory.tar.gz
➡️ Adding to archive: test-directory/NEW
✅ Archive created successfully: test-directory.tar.gz
⬆️ [DRY-RUN] Would upload (1/4): test-directory.tar.gz (2.30 KB) to s3://my-s3-backup-bucket/backup/test-directory/test-directory.tar.gz
📁 [DRY-RUN] Sorted locally: /Users/rtitz/tmp/my-s3-backup-bucket/backup/test-directory/test-directory.tar.gz
⬆️ [DRY-RUN] Would upload additional file: my-input.json
🧽 [DRY-RUN] Files moved to bucket directory in /Users/rtitz/tmp - no cleanup needed

==================================================
📋 BACKUP SUMMARY (DRY-RUN)
==================================================
✅ Files that would be uploaded: 5
📁 Total files processed: 5
💾 Total data processed: 3.67 KB
⏱️ Preparation time: 1.2s
⏱️ Upload time (simulated): 45ms
⏱️ Total time: 1.3s

🎉 Dry-run completed successfully!
==================================================

```

  * 📋 Dry-run output would show:
```
AWS-S3-BACKUP 1.4.1

MODE: BACKUP

[DRY-RUN] Would upload (1/4): test-directory.tar.gz (2.30 MB) to s3://my-s3-backup-bucket/backup/Users/rtitz/test-directory.tar.gz
[DRY-RUN] Would upload (2/4): test.file.tar.gz (0.57 MB) to s3://my-s3-backup-bucket/backup/Users/rtitz/test.file.tar.gz
[DRY-RUN] Would upload (3/4): test2.file.tar.gz (0.49 MB) to s3://my-s3-backup-bucket/backup/Users/rtitz/test2.file.tar.gz
[DRY-RUN] Would upload (4/4): new-file.tar.gz (0.11 MB) to s3://my-s3-backup-bucket/backup/Users/rtitz/dir/dir2/new-file.tar.gz
[DRY-RUN] Would upload additional file: my-input.json to s3://my-s3-backup-bucket/backup/my-input.json
```

  * ✅ Everything in 'Content' part of the my-input.json is uploaded to S3 bucket "my-s3-backup-bucket" and organized into subdirectories based on the last directory name of each content path within the "backup" folder.
  * 🔄 **Smart uploads**: If you run the same backup again, existing files will be skipped automatically
  * ⏭️ **Skip existing files**: Shows `⏭️ Skipping: filename (already exists in S3)` for files that don't need re-upload
  * 🌐 **Network resilience**: Automatically retries network failures for up to 12 hours
  * 🛡️ **Safe operations**: Never overwrites existing data, aborts upload if safety cannot be verified
  * 🕰️ **Timestamp preservation**: Original file and directory timestamps stored in archives
  * 📊 **Enhanced summary**: Displays count of uploaded, skipped, and failed files

**[Back](../README.md)**
## 🔄 Smart Upload Example

When running the same backup multiple times:

**First run:**
```
⬆️ Uploading (1/3): test-directory.tar.gz (2.30 KB)
✅ Upload successful: test-directory.tar.gz
⬆️ Uploading (2/3): test.file.tar.gz (0.57 KB)
✅ Upload successful: test.file.tar.gz
⬆️ Uploading additional file: my-input.json
✅ Additional file uploaded: my-input.json

📊 BACKUP SUMMARY
✅ Successfully uploaded: 3
📁 Total files processed: 3
```

**Second run (same files):**
```
⏭️ Skipping (1/3): test-directory.tar.gz (already exists in S3)
⏭️ Skipping (2/3): test.file.tar.gz (already exists in S3)
⏭️ Skipping additional file: my-input.json (already exists in S3)

📊 BACKUP SUMMARY
⏭️ Skipped (already exists): 3
📁 Total files processed: 3
```

This prevents unnecessary uploads and reduces costs significantly.
## 📁 Dry-Run Local File Sorting

During dry-run mode, files are organized locally exactly as they would appear in S3:

**Directory Structure Created:**
```
/Users/rtitz/tmp/
└── my-s3-backup-bucket/          # Bucket directory
    └── backup/                    # S3 prefix
        ├── test-directory/        # Content subdirectory
        │   └── test-directory.tar.gz
        ├── test.file/             # Content subdirectory
        │   └── test.file.tar.gz
        ├── test2.file/            # Content subdirectory
        │   └── test2.file.tar.gz
        └── my-input.json
```

**Smart Directory Handling:**
- **First run**: Creates `/tmp/my-s3-backup-bucket/`
- **Subsequent runs**: Reuses the same directory
- **File conflicts**: If a file named `my-s3-backup-bucket` exists, creates `my-s3-backup-bucket-1/`
- **Efficient storage**: Files are moved (not copied) to avoid duplication

**Benefits:**
- ✅ **Preview S3 structure** - See exactly how files will be organized
- ✅ **Test configurations** - Verify `TrimBeginningOfPathInS3` and `S3Prefix` settings
- ✅ **No duplication** - Files stored only once in final structure
- ✅ **Consistent paths** - All content lines use the same bucket directory

## 📁 Content Organization

Files are organized into subdirectories based on the last directory name of each content path:

**Content Path Examples:**
- `/home/user/documents` → S3: `backup/documents/documents.tar.gz`
- `/data/photos` → S3: `backup/photos/photos.tar.gz`  
- `/tmp/logs` → S3: `backup/logs/logs.tar.gz`

**Benefits:**
- ✅ **Clean organization** - Each content item gets its own subdirectory
- ✅ **Easy identification** - Subdirectory name matches the source directory
- ✅ **Conflict prevention** - Multiple items with same name stay separated
## 🌐 Network Interruption Handling

If network connection is lost during backup:

```
⬆️ Uploading (3/10): file003.tar.gz (1.8 MB)
⚠️ Upload file003.tar.gz failed (attempt 1): dial tcp: network is unreachable
🔄 Retrying in 1s... (Press Ctrl+C to cancel, will retry for up to 12 hours)
⚠️ Upload file003.tar.gz failed (attempt 2): dial tcp: network is unreachable  
🔄 Retrying in 2s... (Press Ctrl+C to cancel, will retry for up to 12 hours)
[Network restored]
✅ Upload file003.tar.gz succeeded after 3 attempts
⬆️ Uploading (4/10): file004.tar.gz (2.1 MB)
```

**Benefits:**
- No manual intervention needed for temporary network issues
- Completed uploads are never lost or repeated
- User can cancel anytime with Ctrl+C
- Automatic recovery when network returns
## 🔐 Encrypted Backup Example

Using encryption in your input.json:

```json
{
    "tasks": [
        {
            "S3Bucket": "my-secure-backup-bucket",
            "S3Prefix": "encrypted-backup",
            "StorageClass": "STANDARD_IA",
            "ArchiveSplitEachMB": "100",
            "TmpStorageToBuildArchives": "/tmp/backup",
            "CleanupTmpStorage": "true",
            "EncryptionSecret": "MyS3cureP@ssw0rd2024!",
            "Content": [
                "/home/user/important-documents",
                "/home/user/photos"
            ]
        }
    ]
}
```

**Output with encryption:**
```
📦 Creating archive: important-documents.tar.gz
✅ Archive created successfully: important-documents.tar.gz
🔒 Encrypting file: important-documents.tar.gz
✅ File encrypted successfully: important-documents.tar.gz.enc
⬆️ Uploading (1/2): important-documents.tar.gz.enc (15.2 MB)
✅ Upload successful: important-documents.tar.gz.enc
```

**⚠️ CRITICAL WARNINGS:**
- **Password required for restore** - Keep it safe and accessible
- **Manual decryption scripts provided** - `decrypt_manual.py` and `decrypt_openssl.sh`
- **Test your setup** - Verify you can restore before relying on encryption
- **No password recovery** - If you lose the password, your data is permanently inaccessible