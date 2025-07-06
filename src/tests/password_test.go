package tests

import (
	"strings"
	"testing"

	"github.com/rtitz/aws-s3-backup/utils"
)

func TestPasswordValidation(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "empty password (no encryption)",
			password: "",
			wantErr:  false,
		},
		{
			name:     "example password blocked",
			password: "MyS3cureB@ckup2024!",
			wantErr:  true,
		},
		{
			name:     "too short",
			password: "Short1!",
			wantErr:  true,
		},
		{
			name:     "no uppercase",
			password: "mypassword123!",
			wantErr:  true,
		},
		{
			name:     "no lowercase",
			password: "MYPASSWORD123!",
			wantErr:  true,
		},
		{
			name:     "no numbers",
			password: "MyPassword!@#",
			wantErr:  true,
		},
		{
			name:     "no special chars",
			password: "MyPassword123",
			wantErr:  true,
		},
		{
			name:     "common password",
			password: "Password123!",
			wantErr:  true,
		},
		{
			name:     "minimum length met",
			password: "Tr0ub4dor&3_Test",
			wantErr:  false,
		},
		{
			name:     "sequential numbers",
			password: "MyPass123word!",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := utils.ValidateEncryptionPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEncryptionPassword() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "❌") {
				t.Errorf("Error message should start with ❌, got: %v", err)
			}
		})
	}
}