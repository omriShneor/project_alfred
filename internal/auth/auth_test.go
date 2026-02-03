package auth

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptor(t *testing.T) {
	// Set a test encryption key
	os.Setenv("ALFRED_ENCRYPTION_KEY", "test-encryption-key-for-tests")
	defer os.Unsetenv("ALFRED_ENCRYPTION_KEY")

	encryptor, err := NewEncryptor(nil)
	require.NoError(t, err)
	require.NotNil(t, encryptor)

	t.Run("encrypt and decrypt bytes", func(t *testing.T) {
		plaintext := []byte("Hello, World! This is a test message.")

		ciphertext, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext)

		decrypted, err := encryptor.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("encrypt and decrypt string", func(t *testing.T) {
		plaintext := "This is a secret token"

		encrypted, err := encryptor.EncryptString(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, encrypted)

		decrypted, err := encryptor.DecryptString(encrypted)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted)
	})

	t.Run("decrypt invalid ciphertext fails", func(t *testing.T) {
		_, err := encryptor.Decrypt([]byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("decrypt invalid base64 fails", func(t *testing.T) {
		_, err := encryptor.DecryptString("not-valid-base64!!!")
		assert.Error(t, err)
	})

	t.Run("different encryptions produce different ciphertexts", func(t *testing.T) {
		plaintext := []byte("same message")

		ct1, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)

		ct2, err := encryptor.Encrypt(plaintext)
		require.NoError(t, err)

		// Due to random nonce, ciphertexts should be different
		assert.NotEqual(t, ct1, ct2)

		// But both should decrypt to the same plaintext
		dec1, _ := encryptor.Decrypt(ct1)
		dec2, _ := encryptor.Decrypt(ct2)
		assert.Equal(t, dec1, dec2)
	})
}

func TestEncryptorWithCustomKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	encryptor, err := NewEncryptor(key)
	require.NoError(t, err)

	plaintext := "test message"
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestGenerateKey(t *testing.T) {
	key1, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key1, 32)

	key2, err := GenerateKey()
	require.NoError(t, err)
	assert.Len(t, key2, 32)

	// Keys should be random, so different
	assert.NotEqual(t, key1, key2)
}

func TestEncryptorWithDerivedKey(t *testing.T) {
	// Test that encryptor can derive key from ANTHROPIC_API_KEY
	os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	os.Setenv("ANTHROPIC_API_KEY", "sk-test-key-12345")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	encryptor, err := NewEncryptor(nil)
	require.NoError(t, err)

	plaintext := "test message"
	encrypted, err := encryptor.EncryptString(plaintext)
	require.NoError(t, err)

	decrypted, err := encryptor.DecryptString(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestEncryptorNoKeyAvailable(t *testing.T) {
	// Save and clear environment variables
	savedEncKey := os.Getenv("ALFRED_ENCRYPTION_KEY")
	savedAPIKey := os.Getenv("ANTHROPIC_API_KEY")
	os.Unsetenv("ALFRED_ENCRYPTION_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	defer func() {
		// Restore environment
		if savedEncKey != "" {
			os.Setenv("ALFRED_ENCRYPTION_KEY", savedEncKey)
		}
		if savedAPIKey != "" {
			os.Setenv("ANTHROPIC_API_KEY", savedAPIKey)
		}
	}()

	_, err := NewEncryptor(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no encryption key available")
}
