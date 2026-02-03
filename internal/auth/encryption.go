package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

// Encryptor handles AES-256-GCM encryption for sensitive data like OAuth tokens
type Encryptor struct {
	key []byte
}

// NewEncryptor creates a new encryptor using the provided key or ALFRED_ENCRYPTION_KEY env var
// If no key is provided, generates a deterministic key from ANTHROPIC_API_KEY (for simplicity)
func NewEncryptor(key []byte) (*Encryptor, error) {
	if len(key) == 0 {
		// Try to get from environment
		if envKey := os.Getenv("ALFRED_ENCRYPTION_KEY"); envKey != "" {
			decoded, err := base64.StdEncoding.DecodeString(envKey)
			if err == nil && len(decoded) == 32 {
				key = decoded
			} else {
				// Use as raw key material, hash to get 32 bytes
				hash := sha256.Sum256([]byte(envKey))
				key = hash[:]
			}
		} else if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
			// Derive a key from the API key (not ideal but works for development)
			hash := sha256.Sum256([]byte("alfred-encryption-" + apiKey))
			key = hash[:]
		} else {
			return nil, fmt.Errorf("no encryption key available: set ALFRED_ENCRYPTION_KEY or ANTHROPIC_API_KEY")
		}
	}

	if len(key) != 32 {
		// Hash to get exactly 32 bytes for AES-256
		hash := sha256.Sum256(key)
		key = hash[:]
	}

	return &Encryptor{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Nonce is prepended to the ciphertext
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt
func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// EncryptString encrypts a string and returns base64-encoded ciphertext
func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString decrypts base64-encoded ciphertext to a string
func (e *Encryptor) DecryptString(encoded string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	plaintext, err := e.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// GenerateKey generates a random 32-byte key for AES-256
func GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}
