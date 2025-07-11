package utils

// RegionInfo contains information about an AWS region
type RegionInfo struct {
	Flag        string
	Country     string
	GDPRCompliant bool
}

// GetRegionInfo returns flag and GDPR compliance information for AWS regions
func GetRegionInfo(region string) RegionInfo {
	regionMap := map[string]RegionInfo{
		// US Regions
		"us-east-1":      {"🇺🇸", "United States", false},
		"us-east-2":      {"🇺🇸", "United States", false},
		"us-west-1":      {"🇺🇸", "United States", false},
		"us-west-2":      {"🇺🇸", "United States", false},
		"us-gov-east-1": {"🇺🇸", "United States", false},
		"us-gov-west-1": {"🇺🇸", "United States", false},

		// Europe Regions (GDPR Compliant)
		"eu-central-1":   {"🇩🇪", "Germany", true},
		"eu-central-2":   {"🇨🇭", "Switzerland", true},
		"eu-west-1":      {"🇮🇪", "Ireland", true},
		"eu-west-2":      {"🇬🇧", "United Kingdom", true},
		"eu-west-3":      {"🇫🇷", "France", true},
		"eu-north-1":     {"🇸🇪", "Sweden", true},
		"eu-south-1":     {"🇮🇹", "Italy", true},
		"eu-south-2":     {"🇪🇸", "Spain", true},

		// Asia Pacific Regions
		"ap-east-1":      {"🇭🇰", "Hong Kong", false},
		"ap-south-1":     {"🇮🇳", "India", false},
		"ap-south-2":     {"🇮🇳", "India", false},
		"ap-southeast-1": {"🇸🇬", "Singapore", false},
		"ap-southeast-2": {"🇦🇺", "Australia", false},
		"ap-southeast-3": {"🇮🇩", "Indonesia", false},
		"ap-southeast-4": {"🇦🇺", "Australia", false},
		"ap-northeast-1": {"🇯🇵", "Japan", false},
		"ap-northeast-2": {"🇰🇷", "South Korea", false},
		"ap-northeast-3": {"🇯🇵", "Japan", false},

		// Canada
		"ca-central-1": {"🇨🇦", "Canada", false},
		"ca-west-1":    {"🇨🇦", "Canada", false},

		// South America
		"sa-east-1": {"🇧🇷", "Brazil", false},

		// Africa
		"af-south-1": {"🇿🇦", "South Africa", false},

		// Middle East
		"me-south-1":   {"🇧🇭", "Bahrain", false},
		"me-central-1": {"🇦🇪", "UAE", false},

		// Israel
		"il-central-1": {"🇮🇱", "Israel", false},

		// China (Special regions)
		"cn-north-1":     {"🇨🇳", "China", false},
		"cn-northwest-1": {"🇨🇳", "China", false},
	}

	if info, exists := regionMap[region]; exists {
		return info
	}

	// Unknown region
	return RegionInfo{"🌍", "Unknown", false}
}

// GetAllRegions returns a list of all available AWS region codes
func GetAllRegions() []string {
	return []string{
		"us-east-1",
		"us-east-2",
		"us-west-1",
		"us-west-2",
		"eu-central-1",
		"eu-central-2",
		"eu-west-1",
		"eu-west-2",
		"eu-west-3",
		"eu-north-1",
		"eu-south-1",
		"eu-south-2",
		"ap-east-1",
		"ap-south-1",
		"ap-south-2",
		"ap-southeast-1",
		"ap-southeast-2",
		"ap-southeast-3",
		"ap-southeast-4",
		"ap-northeast-1",
		"ap-northeast-2",
		"ap-northeast-3",
		"ca-central-1",
		"ca-west-1",
		"sa-east-1",
		"af-south-1",
		"me-south-1",
		"me-central-1",
		"il-central-1",
	}
}