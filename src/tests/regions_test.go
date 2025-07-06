package tests

import (
	"testing"

	"github.com/rtitz/aws-s3-backup/utils"
)

func TestGetRegionInfo(t *testing.T) {
	tests := []struct {
		region        string
		expectedFlag  string
		expectedGDPR  bool
		expectedCountry string
	}{
		{
			region:        "us-east-1",
			expectedFlag:  "🇺🇸",
			expectedGDPR:  false,
			expectedCountry: "United States",
		},
		{
			region:        "eu-west-1",
			expectedFlag:  "🇮🇪",
			expectedGDPR:  true,
			expectedCountry: "Ireland",
		},
		{
			region:        "eu-central-1",
			expectedFlag:  "🇩🇪",
			expectedGDPR:  true,
			expectedCountry: "Germany",
		},
		{
			region:        "ap-southeast-1",
			expectedFlag:  "🇸🇬",
			expectedGDPR:  false,
			expectedCountry: "Singapore",
		},
		{
			region:        "unknown-region",
			expectedFlag:  "🌍",
			expectedGDPR:  false,
			expectedCountry: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			info := utils.GetRegionInfo(tt.region)
			
			if info.Flag != tt.expectedFlag {
				t.Errorf("Expected flag %s, got %s", tt.expectedFlag, info.Flag)
			}
			
			if info.GDPRCompliant != tt.expectedGDPR {
				t.Errorf("Expected GDPR compliance %v, got %v", tt.expectedGDPR, info.GDPRCompliant)
			}
			
			if info.Country != tt.expectedCountry {
				t.Errorf("Expected country %s, got %s", tt.expectedCountry, info.Country)
			}
		})
	}
}