package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// EncryptFile encrypts a file with AES-256-GCM
func EncryptFile(inputPath, password string) (string, error) {
	log.Printf("🔒 Encrypting file: %s", filepath.Base(inputPath))
	
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	encrypted, err := encrypt(data, []byte(password))
	if err != nil {
		return "", err
	}

	outputPath := inputPath + "." + EncryptionExt
	err = os.WriteFile(outputPath, encrypted, 0644)
	
	if err == nil {
		log.Printf("✅ File encrypted successfully: %s", filepath.Base(outputPath))
	}
	return outputPath, err
}

// DecryptFile decrypts a file encrypted with AES-256-GCM
func DecryptFile(inputPath, password string) (string, error) {
	log.Printf("🔓 Decrypting file: %s", filepath.Base(inputPath))
	
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return "", err
	}

	decrypted, err := decrypt(data, []byte(password))
	if err != nil {
		return "", err
	}

	outputPath := strings.TrimSuffix(inputPath, "."+EncryptionExt)
	err = os.WriteFile(outputPath, decrypted, 0644)
	
	if err == nil {
		log.Printf("✅ File decrypted successfully: %s", filepath.Base(outputPath))
	}
	return outputPath, err
}

func encrypt(data, password []byte) ([]byte, error) {
	key, salt, err := deriveKey(password, nil)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return append(ciphertext, salt...), nil
}

func decrypt(data, password []byte) ([]byte, error) {
	salt, data := data[len(data)-32:], data[:len(data)-32]
	key, _, err := deriveKey(password, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func deriveKey(password, salt []byte) ([]byte, []byte, error) {
	// Use 50% of cores, minimum 1, maximum 6
	cores := runtime.NumCPU()
	p := max(1, min(6, cores/2))

	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, err
		}
	}
	key, err := scrypt.Key(password, salt, 32768, 8, p, 32)
	return key, salt, err
}
