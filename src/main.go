package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/rtitz/aws-s3-backup/config"
	"github.com/rtitz/aws-s3-backup/services"
	"github.com/rtitz/aws-s3-backup/utils"
)

// main is the application entry point
func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

// run orchestrates the main application flow
func run() error {
	flags := parseFlags()

	if flags.version {
		fmt.Printf("%s %s\n", config.AppName, config.AppVersion)
		return nil
	}

	cfg := buildConfig(flags)
	if err := validateAndShowHelp(cfg); err != nil {
		return err
	}

	fmt.Printf("%s %s\n\n", config.AppName, config.AppVersion)
	ctx := context.Background()

	return executeMode(ctx, cfg, flags)
}

// buildConfig creates a Config struct from command line flags
func buildConfig(flags *appFlags) *config.Config {
	return &config.Config{
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
}

// validateAndShowHelp validates configuration and shows help on error
func validateAndShowHelp(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		fmt.Printf("%s %s\n\n", config.AppName, config.AppVersion)
		fmt.Printf("❌ Error: %v\n\nParameter list:\n\n", err)
		flag.PrintDefaults()
		fmt.Printf("\nFor help visit: https://github.com/rtitz/aws-s3-backup\n\n")
		return err
	}
	return nil
}

// executeMode runs the appropriate operation mode (backup or restore)
func executeMode(ctx context.Context, cfg *config.Config, flags *appFlags) error {

	// Get AWS authentication configuration
	var awsCfg aws.Config
	var err error
	if !cfg.DryRun { // No dry-run
		fmt.Printf("🔑 Authenticating with AWS...\n")
		awsCfg, err = getAWSConfig(ctx, cfg)
		if err != nil {
			return err
		}
	} else {
		awsCfg = aws.Config{} // Empty AWS Config as all communication to AWS is skipped
		switch cfg.Mode {
		case "backup":
			log.Println("⚠️  [DRY-RUN] Skipping AWS authentication - no S3 operations will be performed")
			log.Println("⚠️  [DRY-RUN] No bucket validation or AWS connectivity checks performed")
			log.Println("⚠️  [DRY-RUN] Ensure bucket exists and credentials work before real backup")
		case "restore":
			log.Println("⚠️  [DRY-RUN] Skipping AWS authentication - using local directory as bucket")
		default:
			return fmt.Errorf("❌ invalid mode: %s", cfg.Mode)
		}
		fmt.Printf("\nPress 'ENTER' to confirm Dry-Run: ")
		var confirm string
		fmt.Scanln(&confirm)
	}

	// Execute the appropriate mode
	switch cfg.Mode {
	case "backup":
		return executeBackup(ctx, awsCfg, cfg)
	case "restore":
		return executeRestore(ctx, awsCfg, cfg, flags)
	default:
		return fmt.Errorf("❌ invalid mode: %s", cfg.Mode)
	}
}

// getAWSConfig creates and validates AWS configuration
func getAWSConfig(ctx context.Context, cfg *config.Config) (aws.Config, error) {
	// Check for environment variables for access key and secret access key
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")

	// If environment variables are set, use them
	if accessKey != "" && secretKey != "" {
		fmt.Printf("🔑 Using AWS credentials from environment variables...\n")
		awsCfg, err := utils.CreateAWSSessionWithEnvironmentVariables(ctx, accessKey, secretKey, cfg.AWSRegion)
		if err != nil {
			showAuthenticationHelp(cfg, err)
			return aws.Config{}, fmt.Errorf("❌ authentication failed")
		}
		return awsCfg, nil
	} else {
		// Get the current operating system
		os := runtime.GOOS
		if os == "windows" {
			fmt.Printf("🔑 Environment variables $Env:AWS_ACCESS_KEY_ID and $Env:AWS_SECRET_ACCESS_KEY not found.\n")
		} else {
			fmt.Printf("🔑 Environment variables AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY not found.\n")
		}
		fmt.Printf("🔑 Using AWS credentials from AWS CLI profile...\n")
		awsCfg, err := utils.CreateAWSSession(ctx, cfg.AWSProfile, cfg.AWSRegion)
		if err != nil {
			showAuthenticationHelp(cfg, err)
			return aws.Config{}, fmt.Errorf("❌ authentication failed")
		}
		return awsCfg, nil
	}
}

// showAuthenticationHelp displays detailed AWS authentication troubleshooting
func showAuthenticationHelp(cfg *config.Config, err error) {
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
}

// executeBackup runs the backup operation
func executeBackup(ctx context.Context, awsCfg aws.Config, cfg *config.Config) error {
	backupService := services.NewBackupService(awsCfg)
	return backupService.ProcessBackup(ctx, cfg.InputFile, cfg.DryRun)
}

// executeRestore runs the restore operation
func executeRestore(ctx context.Context, awsCfg aws.Config, cfg *config.Config, flags *appFlags) error {
	restoreService := services.NewRestoreService(awsCfg)
	return restoreService.ProcessRestore(ctx, cfg.Bucket, cfg.Prefix, cfg.InputFile,
		cfg.DownloadLocation, cfg.DryRun, flags.skipDecompression, cfg.RetrievalMode,
		int32(cfg.RestoreExpiresAfterDays), int(cfg.AutoRetryDownloadMinutes), cfg.RestoreWithoutConfirmation)
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
	skipDecompression          bool
}

// parseFlags parses command line arguments and returns application flags
func parseFlags() *appFlags {
	flags := &appFlags{}
	flag.StringVar(&flags.mode, "mode", config.DefaultMode, "Operation mode (backup or restore)")
	flag.StringVar(&flags.bucket, "bucket", "", "S3 bucket name for restore mode")
	flag.StringVar(&flags.prefix, "prefix", "", "S3 object prefix filter for restore mode")
	flag.StringVar(&flags.inputFile, "json", "", "JSON file with input parameters")
	flag.StringVar(&flags.downloadLocation, "destination", "", "Download location for restore mode")
	flag.StringVar(&flags.retrievalMode, "retrievalMode", config.DefaultRetrievalMode, "Retrieval mode (bulk, standard, or expedited) for Glacier objects")
	flag.BoolVar(&flags.restoreWithoutConfirmation, "restoreWithoutConfirmation", false, "Skip confirmation for Glacier restores")
	flag.Int64Var(&flags.autoRetryDownloadMinutes, "autoRetryDownloadMinutes", 0, "Auto-retry download interval in minutes (min 5)")
	flag.Int64Var(&flags.restoreExpiresAfterDays, "restoreExpiresAfterDays", config.DefaultRestoreExpiresAfterDays, "Days restore is available in Standard storage")
	flag.StringVar(&flags.awsProfile, "profile", config.DefaultAWSProfile, "AWS CLI profile name")
	flag.StringVar(&flags.awsRegion, "region", config.DefaultAWSRegion, "AWS region")
	flag.BoolVar(&flags.version, "version", false, "Print version")
	flag.BoolVar(&flags.dryRun, "dryrun", false, "Test mode - skip S3 uploads")
	flag.BoolVar(&flags.skipDecompression, "skipDecompression", false, "Skip archive decompression during restore")
	flag.Parse()

	flags.mode = strings.ToLower(flags.mode)
	flags.retrievalMode = strings.ToLower(flags.retrievalMode)

	return flags
}
