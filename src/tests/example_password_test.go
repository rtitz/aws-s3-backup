package tests

import (
	"strings"
	"testing"

	"github.com/rtitz/aws-s3-backup/utils"
)

func TestExamplePasswordBlocked(t *testing.T) {
	// Test that the example password is blocked
	err := utils.ValidateEncryptionPassword("MyS3cureB@ckup2024!")
	if err == nil {
		t.Error("Expected error for example password, got nil")
	}
	
	if !strings.Contains(err.Error(), "example password") {
		t.Errorf("Error should mention example password, got: %v", err)
	}
	
	if !strings.Contains(err.Error(), "❌") {
		t.Errorf("Error should start with ❌, got: %v", err)
	}
}