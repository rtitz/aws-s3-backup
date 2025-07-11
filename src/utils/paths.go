package utils

import (
	"path/filepath"
	"strings"
)

// NormalizePath converts any path to use forward slashes (Unix-style)
// This ensures consistent path handling across different operating systems
func NormalizePath(path string) string {
	cleanPath := filepath.Clean(path)
	return strings.ReplaceAll(cleanPath, "\\", "/")
}

// TrimPathPrefix removes a prefix from a path in a cross-platform way
// It handles different path separators and ensures consistent results
func TrimPathPrefix(fullPath, prefix string) string {
	if prefix == "" {
		return NormalizePath(fullPath)
	}

	// Normalize both paths to use forward slashes
	normalizedFull := NormalizePath(fullPath)
	normalizedPrefix := NormalizePath(prefix)

	// Ensure prefix ends with slash for proper trimming
	normalizedPrefix = ensureTrailingSlash(normalizedPrefix)

	// Remove prefix if it matches
	if strings.HasPrefix(normalizedFull, normalizedPrefix) {
		trimmed := strings.TrimPrefix(normalizedFull, normalizedPrefix)
		return strings.TrimPrefix(trimmed, "/")
	}

	return normalizedFull
}

// ensureTrailingSlash adds a trailing slash if not present
func ensureTrailingSlash(path string) string {
	if !strings.HasSuffix(path, "/") {
		return path + "/"
	}
	return path
}
