package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rtitz/aws-s3-backup/config"
	"golang.org/x/crypto/scrypt"
)

// Encryption constants
const (
	SaltSize         = 32
	MinEncryptedSize = SaltSize + 12 + 16 // salt + nonce + tag
	NewScryptN       = 131072             // N=128K (stronger)
	LegacyScryptN    = 32768              // N=32K (backward compatibility)
	ScryptR          = 8
	KeySize          = 32
)

// EncryptFile encrypts a file with AES-256-GCM and saves it with .enc extension
func EncryptFile(inputPath, password string) (string, error) {
	log.Printf("🔒 Encrypting file: %s", filepath.Base(inputPath))

	data, err := readFileForEncryption(inputPath)
	if err != nil {
		return "", err
	}

	encrypted, err := encryptData(data, []byte(password))
	if err != nil {
		return "", err
	}

	outputPath := buildEncryptedFilePath(inputPath)
	if err := writeEncryptedFile(outputPath, encrypted); err != nil {
		return "", err
	}

	log.Printf("✅ File encrypted successfully: %s", filepath.Base(outputPath))
	return outputPath, nil
}

// DecryptFile decrypts a file encrypted with AES-256-GCM
func DecryptFile(inputPath, password string) (string, error) {
	data, err := readEncryptedFile(inputPath)
	if err != nil {
		return "", err
	}

	decrypted, err := decryptData(data, []byte(password))
	if err != nil {
		return "", err
	}

	outputPath := buildDecryptedFilePath(inputPath)
	if err := writeDecryptedFile(outputPath, decrypted); err != nil {
		return "", err
	}

	return outputPath, nil
}

// File I/O helpers
// readFileForEncryption reads a file for encryption
func readFileForEncryption(inputPath string) ([]byte, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file for encryption: %w", err)
	}
	return data, nil
}

// readEncryptedFile reads an encrypted file
func readEncryptedFile(inputPath string) ([]byte, error) {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}
	return data, nil
}

// writeEncryptedFile writes encrypted data to file
func writeEncryptedFile(outputPath string, data []byte) error {
	return os.WriteFile(outputPath, data, 0644)
}

// writeDecryptedFile writes decrypted data to file
func writeDecryptedFile(outputPath string, data []byte) error {
	return os.WriteFile(outputPath, data, 0644)
}

// buildEncryptedFilePath creates output path with .enc extension
func buildEncryptedFilePath(inputPath string) string {
	return inputPath + "." + config.EncryptionExt
}

// buildDecryptedFilePath removes .enc extension from path
func buildDecryptedFilePath(inputPath string) string {
	return strings.TrimSuffix(inputPath, "."+config.EncryptionExt)
}

// Core encryption/decryption functions
// encryptData encrypts data using AES-256-GCM
func encryptData(data, password []byte) ([]byte, error) {
	if err := validateEncryptionInput(data, password); err != nil {
		return nil, err
	}

	key, salt, err := deriveEncryptionKey(password, nil)
	if err != nil {
		return nil, err
	}

	gcm, err := createGCMCipher(key)
	if err != nil {
		return nil, err
	}

	nonce, err := generateNonce(gcm.NonceSize())
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return append(ciphertext, salt...), nil
}

// decryptData decrypts AES-256-GCM encrypted data
func decryptData(data, password []byte) ([]byte, error) {
	if err := validateDecryptionInput(data); err != nil {
		return nil, err
	}

	// Check for versioned format (future-proofing)
	if isVersionedFormat(data) {
		return decryptVersionedFormat(data, password)
	}

	// Current format (v1)
	return decryptCurrentFormat(data, password)
}

// Validation helpers
// validateEncryptionInput validates data and password for encryption
func validateEncryptionInput(data, password []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot encrypt empty data")
	}
	if len(password) == 0 {
		return fmt.Errorf("password cannot be empty")
	}
	return nil
}

// validateDecryptionInput validates encrypted data format
func validateDecryptionInput(data []byte) error {
	if len(data) < MinEncryptedSize {
		return fmt.Errorf("invalid encrypted data: too short (%d bytes)", len(data))
	}
	return nil
}

// Cryptographic helpers
// deriveEncryptionKey derives encryption key using scrypt
func deriveEncryptionKey(password, salt []byte) ([]byte, []byte, error) {
	if len(password) == 0 {
		return nil, nil, fmt.Errorf("password cannot be empty")
	}

	if salt == nil {
		var err error
		salt, err = generateSalt()
		if err != nil {
			return nil, nil, err
		}
	} else if len(salt) != SaltSize {
		return nil, nil, fmt.Errorf("invalid salt length: expected %d, got %d", SaltSize, len(salt))
	}

	p := calculateScryptP()
	key, err := scrypt.Key(password, salt, NewScryptN, ScryptR, p, KeySize)
	if err != nil {
		return nil, nil, fmt.Errorf("key derivation failed: %w", err)
	}

	return key, salt, nil
}

// createGCMCipher creates AES-GCM cipher from key
func createGCMCipher(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	return gcm, nil
}

// generateNonce generates random nonce of specified size
func generateNonce(size int) ([]byte, error) {
	nonce := make([]byte, size)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	return nonce, nil
}

// generateSalt generates random salt for key derivation
func generateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// calculateScryptP calculates scrypt parallelization parameter
func calculateScryptP() int {
	cores := runtime.NumCPU()
	return max(1, min(6, cores/2))
}

// Decryption format handlers
// isVersionedFormat checks if data uses versioned encryption format
func isVersionedFormat(data []byte) bool {
	return len(data) >= 4 && string(data[:4]) == "ENC2"
}

// decryptVersionedFormat decrypts future versioned format (not implemented)
func decryptVersionedFormat(data, password []byte) ([]byte, error) {
	// Future implementation for enhanced security
	_ = data
	_ = password
	return nil, fmt.Errorf("v2 encryption format not yet implemented")
}

// decryptCurrentFormat decrypts current v1 format
func decryptCurrentFormat(data, password []byte) ([]byte, error) {
	salt, cipherData := extractSaltAndData(data)

	// Try new parameters first (N=128K)
	if result, err := tryDecryptWithScryptParams(cipherData, password, salt, NewScryptN); err == nil {
		return result, nil
	}

	// Fall back to legacy parameters (N=32K) for backward compatibility
	if result, err := tryDecryptWithScryptParams(cipherData, password, salt, LegacyScryptN); err == nil {
		return result, nil
	}

	return nil, fmt.Errorf("decryption failed with both parameter sets")
}

// extractSaltAndData separates salt from encrypted data
func extractSaltAndData(data []byte) ([]byte, []byte) {
	salt := data[len(data)-SaltSize:]
	cipherData := data[:len(data)-SaltSize]
	return salt, cipherData
}

// tryDecryptWithScryptParams attempts decryption with specific scrypt parameters
func tryDecryptWithScryptParams(data, password, salt []byte, N int) ([]byte, error) {
	p := calculateScryptP()
	key, err := scrypt.Key(password, salt, N, ScryptR, p, KeySize)
	if err != nil {
		return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
	}

	gcm, err := createGCMCipher(key)
	if err != nil {
		return nil, err
	}

	if len(data) < gcm.NonceSize() {
		return nil, fmt.Errorf("invalid encrypted data: nonce too short")
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("GCM decryption failed: %w", err)
	}

	return plaintext, nil
}
