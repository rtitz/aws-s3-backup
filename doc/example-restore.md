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
AWS-S3-BACKUP 1.3.4

MODE: RESTORE
REGION: us-east-1

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
AWS-S3-BACKUP 1.3.4

MODE: RESTORE
REGION: us-east-1

Generated: 'generated-restore-input.json'

Do you want to continue with restore, without editing generated input JSON? [y/N]: y

⬇️ Downloading: backup/Users/rtitz/test-directory.tar.gz (2.30 KB)
✅ Successfully downloaded: test-directory.tar.gz
⬇️ Downloading: backup/my-input.json (1.2 KB)
✅ Successfully downloaded: my-input.json

🔓 Decrypting encrypted files...
🔗 Combining split archives...

==================================================
📊 RESTORE SUMMARY
==================================================
✅ Successfully downloaded: 5
📁 Total files processed: 5
💾 Total data processed: 3.67 KB
⏱️ Download time: 2.1s
⏱️ Processing time: 0.8s
⏱️ Total time: 2.9s

🎉 Restore completed successfully!
==================================================
```

  * ▶️ Here is how to restore with JSON file:
```bash
export AWS_ACCESS_KEY_ID="AKXXXXXXXXX" # Only needed if not already done in previous command
export AWS_SECRET_ACCESS_KEY="XXXXXXXXXXXXXXXXX" # Only needed if not already done in previous command

aws-s3-backup_macos-arm64 -mode restore -bucket my-s3-backup-bucket -destination restore/ -json generated-restore-input.json
```

  * Output will be similar to:
```
AWS-S3-BACKUP 1.3.4

MODE: RESTORE
REGION: us-east-1

⬇️ Downloading: backup/Users/rtitz/test-directory.tar.gz
✅ Successfully downloaded: test-directory.tar.gz
⬇️ Downloading: backup/my-input.json
✅ Successfully downloaded: my-input.json

🔓 Decrypting encrypted files...
🔗 Combining split archives...

==================================================
📊 RESTORE SUMMARY
==================================================
✅ Successfully downloaded: 5
📁 Total files processed: 5
💾 Total data processed: 3.67 KB
⏱️ Download time: 2.1s
⏱️ Processing time: 0.8s
⏱️ Total time: 2.9s

🎉 Restore completed successfully!
==================================================
```

  * 📋 Here is how to test restore with dry-run:
```bash
# Test restore using local directory as bucket source
aws-s3-backup_macos-arm64 -mode restore -bucket /path/to/local/backup -destination restore/ -dryrun
```

  * Output for dry-run would show:
```
AWS-S3-BACKUP 1.3.4

MODE: RESTORE
REGION: DRY-RUN

📁 [DRY-RUN] Found 5 files in local directory: /path/to/local/backup
⬇️ [DRY-RUN] Would download: test-directory.tar.gz (2.30 KB)
⬇️ [DRY-RUN] Would download: my-input.json (1.2 KB)

🔓 Decrypting encrypted files...
🔗 Combining split archives...

==================================================
📋 RESTORE SUMMARY (DRY-RUN)
==================================================
✅ Files that would be downloaded: 5
📁 Total files processed: 5
💾 Total data processed: 3.67 KB
⏱️ Download time (simulated): 45ms
⏱️ Processing time: 0.8s
⏱️ Total time: 0.9s

🎉 Dry-run restore completed successfully!
==================================================
```

  * 📁 Files now locate in your current directory in folder restore/
  * ✅ **Split archives are automatically combined** - no manual extraction needed
  * 🧊 **Glacier objects are automatically detected and restored** - tool will prompt for confirmation
  * ⚡ **Fast Glacier access** - Use `-retrievalMode expedited` for 1-5 minute restores from GLACIER_FLEXIBLE_RETRIEVAL
  * 🔄 **Auto-retry for Glacier** - Use `-autoRetryDownloadMinutes 5` to automatically retry every 5 minutes
  * 🌐 **Network resilience** - Automatically retries download failures for up to 12 hours
  * 🕰️ **Timestamp preservation** - Original file and directory timestamps are maintained
  * 📊 **Enhanced progress** - Shows file sizes during downloads
  * ⏱️ **Enhanced timing breakdown** - See preparation, download/upload, and processing times separately

**[Back](../README.md)**
## 🧊 Glacier Restore Examples

### Restore with Glacier Objects (Interactive)
```bash
# Tool will detect Glacier objects and ask for confirmation
aws-s3-backup_macos-arm64 -mode restore -bucket my-glacier-bucket -destination restore/
```

Output will show:
```
🧊 Found 3 objects in Glacier storage classes that may need restore
🔄 3 objects need to be restored from Glacier
  - backup/file1.tar.gz (GLACIER_FLEXIBLE_RETRIEVAL, 2.3 MB)
  - backup/file2.tar.gz (DEEP_ARCHIVE, 1.8 MB)
  - backup/file3.tar.gz (GLACIER, 950 KB)

Do you want to restore these 3 objects from Glacier storage? [y/N]: y

🔄 Initiating restore for 3 objects (mode: bulk, expires after: 3 days)
🔄 Restoring: backup/file1.tar.gz
✅ Restore initiated for: backup/file1.tar.gz
⏰ Bulk retrieval typically takes 5-12 hours for Glacier, up to 48 hours for Deep Archive

❌ Glacier restore failed: restore requests initiated. Please wait for completion before downloading
```

### Expedited Restore (Fast)
```bash
# Use expedited mode for GLACIER_FLEXIBLE_RETRIEVAL (1-5 minutes)
aws-s3-backup_macos-arm64 -mode restore -bucket my-glacier-bucket -destination restore/ -retrievalMode expedited -restoreWithoutConfirmation
```

### Auto-Retry Restore
```bash
# Automatically retry every 5 minutes until objects are available
aws-s3-backup_macos-arm64 -mode restore -bucket my-glacier-bucket -destination restore/ -autoRetryDownloadMinutes 5 -restoreWithoutConfirmation
```

Output shows progress:
```
🔄 Auto-retry enabled: checking restore status every 5 minutes
📊 Restore progress: 1/3 objects ready (2 still waiting)
⏰ Waiting 5 minutes before next check...
📊 Restore progress: 3/3 objects ready (0 still waiting)
🎉 All Glacier objects are now restored and available for download
```
## 🌐 Network Interruption During Restore

If network connection is lost during restore:

```
⬇️ Downloading: backup/file005.tar.gz
⚠️ Download backup/file005.tar.gz failed (attempt 1): read tcp: connection reset by peer
🔄 Retrying in 1s... (Press Ctrl+C to cancel, will retry for up to 12 hours)
⚠️ Download backup/file005.tar.gz failed (attempt 2): dial tcp: no route to host
🔄 Retrying in 2s... (Press Ctrl+C to cancel, will retry for up to 12 hours)
[Network restored]
✅ Download backup/file005.tar.gz succeeded after 3 attempts
✅ Successfully downloaded: file005.tar.gz
```

**Recovery Process:**
- Automatic retry with exponential backoff
- No data corruption or partial downloads
- Completed downloads are preserved
- User can cancel and resume later if needed
## 🔐 Encrypted File Restore

When restoring encrypted files, you'll be prompted for the password:

```
⬇️ Downloading: backup/encrypted-file.tar.gz.enc (5.2 MB)
✅ Successfully downloaded: encrypted-file.tar.gz.enc

🔐 Encrypted files detected. Enter decryption password: [password hidden]
🔓 Decrypting: encrypted-file.tar.gz.enc -> encrypted-file.tar.gz
✅ Successfully decrypted: encrypted-file.tar.gz
📎 Decompressing: encrypted-file.tar.gz
✅ Successfully decompressed and removed: encrypted-file.tar.gz
```

**⚠️ Important Notes:**
- **Keep your password safe** - without it, encrypted files cannot be recovered
- **Manual decryption available** - `decrypt_manual.py` script provided as backup
- **Test encryption setup** before relying on it for important data

## 🕒 Timestamp Preservation

All original timestamps are preserved during restore:

```
📎 Decompressing: backup.tar.gz
✅ Successfully decompressed and removed: backup.tar.gz
⚠️ Warning: Could not set timestamps for /readonly/dir: permission denied
```

**Features:**
- ✅ File modification and access times preserved
- ✅ Directory timestamps maintained  
- ✅ File permissions restored
- ⚠️ Non-fatal warnings if timestamps can't be set (read-only filesystems)