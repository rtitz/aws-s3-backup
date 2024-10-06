package cryptUtils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"

	"golang.org/x/crypto/scrypt"
)

// SOURCE: https://bruinsslot.jp/post/golang-crypto/

// Aes256GcmEncrypt encrypts data using AES-256-GCM.
func Aes256GcmEncrypt(key, data []byte) ([]byte, error) {
	key, salt, err := deriveKey(key, nil)
	if err != nil {
		return nil, err
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	ciphertext = append(ciphertext, salt...)

	return ciphertext, nil
}

// Aes256GcmDecrypt decrypts data from AES-256-GCM encryption.
func Aes256GcmDecrypt(key, data []byte) ([]byte, error) {
	salt, data := data[len(data)-32:], data[:len(data)-32]

	key, _, err := deriveKey(key, salt)
	if err != nil {
		return nil, err
	}

	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(blockCipher)
	if err != nil {
		return nil, err
	}

	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// deriveKey derives a key from a password and salt using scrypt.
// The salt is randomly generated if not provided.
// The cost parameter is set to 2^20 (1048576) and the block size is set to 8.
// The key length is set to 32 bytes.
// The salt is appended to the end of the key for storage purposes.
func deriveKey(password, salt []byte) ([]byte, []byte, error) {
	if salt == nil {
		salt = make([]byte, 32)
		if _, err := rand.Read(salt); err != nil {
			return nil, nil, err
		}
	}

	costParameter := 32768                                          // 2^20 = 1048576 ; 2^15 = 32768
	key, err := scrypt.Key(password, salt, costParameter, 8, 1, 32) // 2^15
	if err != nil {
		return nil, nil, err
	}

	return key, salt, nil
}

// generateKey generates a random 32-byte key.
/*func generateKey() ([]byte, error) {
	key := make([]byte, 32)

	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}*/
