package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rtitz/aws-s3-backup/config"
	"github.com/rtitz/aws-s3-backup/services"
	"github.com/rtitz/aws-s3-backup/utils"
	"github.com/rtitz/aws-s3-backup/version"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	flags := parseFlags()

	if flags.version {
		fmt.Printf("%s %s\n", version.AppName, version.AppVersion)
		return nil
	}

	cfg := &config.Config{
		Mode:                       flags.mode,
		InputFile:                  flags.inputFile,
		AWSProfile:                 flags.awsProfile,
		AWSRegion:                  flags.awsRegion,
		Bucket:                     flags.bucket,
		Prefix:                     flags.prefix,
		DownloadLocation:           flags.downloadLocation,
		RetrievalMode:              flags.retrievalMode,
		RestoreWithoutConfirmation: flags.restoreWithoutConfirmation,
		AutoRetryDownloadMinutes:   flags.autoRetryDownloadMinutes,
		RestoreExpiresAfterDays:    flags.restoreExpiresAfterDays,
		DryRun:                     flags.dryRun,
	}

	if err := cfg.Validate(); err != nil {
		fmt.Printf("%s %s\n\n", version.AppName, version.AppVersion)
		fmt.Printf("❌ Error: %v\n\nParameter list:\n\n", err)
		flag.PrintDefaults()
		fmt.Printf("\nFor help visit: https://github.com/rtitz/aws-s3-backup\n\n")
		return err
	}

	fmt.Printf("%s %s\n\n", version.AppName, version.AppVersion)

	ctx := context.Background()
	
	// Skip AWS authentication for dry-run backup mode
	if cfg.Mode == "backup" && cfg.DryRun {
		log.Println("⚠️  [DRY-RUN] Skipping AWS authentication - no S3 operations will be performed")
		log.Println("⚠️  [DRY-RUN] No bucket validation or AWS connectivity checks performed")
		log.Println("⚠️  [DRY-RUN] Ensure bucket exists and credentials work before real backup")
		backupService := services.NewBackupService(aws.Config{})
		return backupService.ProcessBackup(ctx, cfg.InputFile, cfg.DryRun)
	}
	
	awsCfg, err := utils.CreateAWSSession(ctx, cfg.AWSProfile, cfg.AWSRegion)
	if err != nil {
		fmt.Printf("\n❌ AWS Authentication Failed\n\n")
		fmt.Printf("Error: %v\n\n", err)
		fmt.Printf("🔧 How to Fix Authentication:\n\n")
		fmt.Printf("Option 1 - Environment Variables (Recommended):\n")
		fmt.Printf("  export AWS_ACCESS_KEY_ID=\"YOUR_ACCESS_KEY\"\n")
		fmt.Printf("  export AWS_SECRET_ACCESS_KEY=\"YOUR_SECRET_KEY\"\n")
		fmt.Printf("  # Optional: export AWS_SESSION_TOKEN=\"YOUR_TOKEN\" (for temporary credentials)\n\n")
		fmt.Printf("Option 2 - AWS CLI Profile:\n")
		fmt.Printf("  aws configure --profile %s\n", cfg.AWSProfile)
		fmt.Printf("  # Then run: aws-s3-backup -profile %s ...\n\n", cfg.AWSProfile)
		fmt.Printf("Option 3 - AWS IAM Identity Center:\n")
		fmt.Printf("  1. Sign in to AWS console\n")
		fmt.Printf("  2. Click 'Command line or programmatic access'\n")
		fmt.Printf("  3. Copy and run the export commands\n\n")
		fmt.Printf("💡 Tips:\n")
		fmt.Printf("  • Test with dry-run first: --dry-run (no AWS credentials needed)\n")
		fmt.Printf("  • Check region matches your S3 bucket: -region %s\n", cfg.AWSRegion)
		fmt.Printf("  • Verify IAM permissions for S3 operations\n\n")
		fmt.Printf("📖 More help: https://github.com/rtitz/aws-s3-backup#authentication\n\n")
		return fmt.Errorf("❌ authentication failed")
	}

	switch cfg.Mode {
	case "backup":
		backupService := services.NewBackupService(awsCfg)
		return backupService.ProcessBackup(ctx, cfg.InputFile, cfg.DryRun)
	case "restore":
		restoreService := services.NewRestoreService(awsCfg)
		return restoreService.ProcessRestore(ctx, cfg.Bucket, cfg.Prefix, cfg.InputFile, cfg.DownloadLocation)
	default:
		return fmt.Errorf("❌ invalid mode: %s", cfg.Mode)
	}
}

type appFlags struct {
	mode                       string
	bucket                     string
	prefix                     string
	inputFile                  string
	downloadLocation           string
	retrievalMode              string
	restoreWithoutConfirmation bool
	autoRetryDownloadMinutes   int64
	restoreExpiresAfterDays    int64
	awsProfile                 string
	awsRegion                  string
	version                    bool
	dryRun                     bool
}

func parseFlags() *appFlags {
	flags := &appFlags{}
	flag.StringVar(&flags.mode, "mode", config.DefaultMode, "Operation mode (backup or restore)")
	flag.StringVar(&flags.bucket, "bucket", "", "S3 bucket name for restore mode")
	flag.StringVar(&flags.prefix, "prefix", "", "S3 object prefix filter for restore mode")
	flag.StringVar(&flags.inputFile, "json", "", "JSON file with input parameters")
	flag.StringVar(&flags.downloadLocation, "destination", "", "Download location for restore mode")
	flag.StringVar(&flags.retrievalMode, "retrievalMode", config.DefaultRetrievalMode, "Retrieval mode (bulk or standard) for Glacier objects")
	flag.BoolVar(&flags.restoreWithoutConfirmation, "restoreWithoutConfirmation", false, "Skip confirmation for Glacier restores")
	flag.Int64Var(&flags.autoRetryDownloadMinutes, "autoRetryDownloadMinutes", 0, "Auto-retry download interval in minutes (min 60)")
	flag.Int64Var(&flags.restoreExpiresAfterDays, "restoreExpiresAfterDays", config.DefaultRestoreExpiresAfterDays, "Days restore is available in Standard storage")
	flag.StringVar(&flags.awsProfile, "profile", config.DefaultAWSProfile, "AWS CLI profile name")
	flag.StringVar(&flags.awsRegion, "region", config.DefaultAWSRegion, "AWS region")
	flag.BoolVar(&flags.version, "version", false, "Print version")
	flag.BoolVar(&flags.dryRun, "dryrun", false, "Test mode - skip S3 uploads")
	flag.Parse()

	flags.mode = strings.ToLower(flags.mode)
	flags.retrievalMode = strings.ToLower(flags.retrievalMode)

	return flags
}
