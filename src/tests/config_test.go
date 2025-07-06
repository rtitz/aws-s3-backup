package tests

import (
	"testing"

	"github.com/rtitz/aws-s3-backup/config"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  config.Config
		wantErr bool
	}{
		{
			name: "valid backup config",
			config: config.Config{
				Mode:                    "backup",
				InputFile:               "test.json",
				RestoreExpiresAfterDays: 3,
			},
			wantErr: false,
		},
		{
			name: "invalid mode",
			config: config.Config{
				Mode: "invalid",
			},
			wantErr: true,
		},
		{
			name: "backup without input file",
			config: config.Config{
				Mode: "backup",
			},
			wantErr: true,
		},
		{
			name: "invalid retry minutes",
			config: config.Config{
				Mode:                     "restore",
				AutoRetryDownloadMinutes: 30,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParseStorageClass(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"STANDARD", "STANDARD"},
		{"DEEP_ARCHIVE", "DEEP_ARCHIVE"},
		{"INVALID", "STANDARD"}, // Should default to STANDARD
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := config.ParseStorageClass(tt.input)
			if string(result) != tt.expected {
				t.Errorf("ParseStorageClass(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseArchiveSplitMB(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{"250", 250, false},
		{"", 250, false}, // Default value
		{"0", 0, true},   // Invalid
		{"abc", 0, true}, // Invalid
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := config.ParseArchiveSplitMB(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseArchiveSplitMB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseArchiveSplitMB() = %v, want %v", got, tt.want)
			}
		})
	}
}