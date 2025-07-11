package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Password validation constants
const (
	MinPasswordLength = 12
	ExamplePassword   = "MyS3cureB@ckup2024!"
)

// Password requirement flags
type passwordRequirements struct {
	hasUpper   bool
	hasLower   bool
	hasDigit   bool
	hasSpecial bool
}

// ValidateEncryptionPassword validates password strength for AES-256-GCM encryption
func ValidateEncryptionPassword(password string) error {
	if password == "" {
		return nil // Empty password means no encryption
	}

	if err := checkExamplePassword(password); err != nil {
		return err
	}

	issues := validatePasswordStrength(password)
	if len(issues) > 0 {
		return buildPasswordError(issues)
	}

	return nil
}

// checkExamplePassword prevents using the documentation example password
func checkExamplePassword(password string) error {
	if password == ExamplePassword {
		return fmt.Errorf("❌ The example password '%s' is not allowed for security reasons.\n\n🔒 Please create your own unique password following the requirements.", ExamplePassword)
	}
	return nil
}

// validatePasswordStrength checks all password requirements
func validatePasswordStrength(password string) []string {
	var issues []string

	// Check minimum length
	if len(password) < MinPasswordLength {
		issues = append(issues, fmt.Sprintf("• Password too short (%d chars) - minimum %d characters required", len(password), MinPasswordLength))
	}

	// Check character requirements
	reqs := analyzePasswordCharacters(password)
	issues = append(issues, checkCharacterRequirements(reqs)...)

	// Check for common weak patterns
	if isCommonPassword(password) {
		issues = append(issues, "• Password contains common patterns or dictionary words")
	}

	return issues
}

// analyzePasswordCharacters categorizes characters in the password
func analyzePasswordCharacters(password string) passwordRequirements {
	var reqs passwordRequirements

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			reqs.hasUpper = true
		case unicode.IsLower(char):
			reqs.hasLower = true
		case unicode.IsDigit(char):
			reqs.hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			reqs.hasSpecial = true
		}
	}

	return reqs
}

// checkCharacterRequirements validates character type requirements
func checkCharacterRequirements(reqs passwordRequirements) []string {
	var issues []string

	if !reqs.hasUpper {
		issues = append(issues, "• Missing uppercase letters (A-Z)")
	}
	if !reqs.hasLower {
		issues = append(issues, "• Missing lowercase letters (a-z)")
	}
	if !reqs.hasDigit {
		issues = append(issues, "• Missing numbers (0-9)")
	}
	if !reqs.hasSpecial {
		issues = append(issues, "• Missing special characters (!@#$%^&*)")
	}

	return issues
}

// buildPasswordError creates a comprehensive error message
func buildPasswordError(issues []string) error {
	return fmt.Errorf("❌ Encryption password does not meet security requirements:\n\n%s\n\n🔒 Requirements:\n• Minimum %d characters (16+ recommended)\n• At least one uppercase letter (A-Z)\n• At least one lowercase letter (a-z)\n• At least one number (0-9)\n• At least one special character (!@#$%%^&*)\n• No common dictionary words or patterns\n\n💡 Example: %s",
		strings.Join(issues, "\n"), MinPasswordLength, ExamplePassword)
}

// isCommonPassword checks for common weak password patterns
func isCommonPassword(password string) bool {
	return containsWeakPatterns(password) || containsSimplePatterns(password)
}

// containsWeakPatterns checks for dictionary words and common passwords
func containsWeakPatterns(password string) bool {
	lower := strings.ToLower(password)

	weakPatterns := []string{
		"password", "123456", "qwerty", "admin", "login",
		"welcome", "monkey", "dragon", "master", "shadow",
		"letmein", "football", "baseball", "superman", "batman",
		"trustno1", "hello", "world", "computer", "internet",
	}

	for _, pattern := range weakPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}

	return false
}

// containsSimplePatterns checks for repetitive and sequential patterns
func containsSimplePatterns(password string) bool {
	patterns := []struct {
		regex       string
		description string
	}{
		{`(.)\\1{3,}`, "4+ repeated characters"},
		{`(012|123|234|345|456|567|678|789|890)`, "sequential numbers"},
		{`(?i)(abc|bcd|cde|def|efg|fgh|ghi|hij|ijk|jkl|klm|lmn|mno|nop|opq|pqr|qrs|rst|stu|tuv|uvw|vwx|wxy|xyz)`, "sequential letters"},
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern.regex, password); matched {
			return true
		}
	}

	return false
}
