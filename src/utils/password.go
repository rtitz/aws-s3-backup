package utils

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// ValidateEncryptionPassword validates password strength for AES-256-GCM encryption
func ValidateEncryptionPassword(password string) error {
	if password == "" {
		return nil // Empty password means no encryption
	}

	// Prevent using the example password
	examplePassword := "MyS3cureB@ckup2024!"
	if password == examplePassword {
		return fmt.Errorf("❌ The example password 'MyS3cureB@ckup2024!' is not allowed for security reasons.\n\n🔒 Please create your own unique password following the requirements.")
	}

	var issues []string

	// Check minimum length
	if len(password) < 12 {
		issues = append(issues, fmt.Sprintf("• Password too short (%d chars) - minimum 12 characters required", len(password)))
	}

	// Check character requirements
	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		issues = append(issues, "• Missing uppercase letters (A-Z)")
	}
	if !hasLower {
		issues = append(issues, "• Missing lowercase letters (a-z)")
	}
	if !hasDigit {
		issues = append(issues, "• Missing numbers (0-9)")
	}
	if !hasSpecial {
		issues = append(issues, "• Missing special characters (!@#$%^&*)")
	}

	// Check for common weak patterns
	if isCommonPassword(password) {
		issues = append(issues, "• Password contains common patterns or dictionary words")
	}

	if len(issues) > 0 {
		return fmt.Errorf("❌ Encryption password does not meet security requirements:\n\n%s\n\n🔒 Requirements:\n• Minimum 12 characters (16+ recommended)\n• At least one uppercase letter (A-Z)\n• At least one lowercase letter (a-z)\n• At least one number (0-9)\n• At least one special character (!@#$%%^&*)\n• No common dictionary words or patterns\n\n💡 Example: %s", strings.Join(issues, "\n"), examplePassword)
	}

	return nil
}

// isCommonPassword checks for common weak password patterns
func isCommonPassword(password string) bool {
	lower := strings.ToLower(password)

	// Common weak patterns
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

	// Check for simple patterns like "123", "abc", "aaa"
	if matched, _ := regexp.MatchString(`(.)\1{3,}`, password); matched { // 4+ repeated chars
		return true
	}
	if matched, _ := regexp.MatchString(`(012|123|234|345|456|567|678|789|890)`, password); matched { // Sequential numbers
		return true
	}
	if matched, _ := regexp.MatchString(`(abc|bcd|cde|def|efg|fgh|ghi|hij|ijk|jkl|klm|lmn|mno|nop|opq|pqr|qrs|rst|stu|tuv|uvw|vwx|wxy|xyz)`, lower); matched { // Sequential letters
		return true
	}

	return false
}
