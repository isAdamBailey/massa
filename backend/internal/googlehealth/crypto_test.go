package googlehealth_test

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/isAdamBailey/massa/backend/internal/googlehealth"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	plaintext := []byte("super-secret-refresh-token")

	ciphertext, nonce, err := googlehealth.Encrypt(key, plaintext)
	require.NoError(t, err)
	assert.NotEqual(t, plaintext, ciphertext)

	got, err := googlehealth.Decrypt(key, nonce, ciphertext)
	require.NoError(t, err)
	assert.Equal(t, plaintext, got)
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	ciphertext, nonce, err := googlehealth.Encrypt(key, []byte("data"))
	require.NoError(t, err)

	wrongKey := make([]byte, 32)
	_, err = rand.Read(wrongKey)
	require.NoError(t, err)

	_, err = googlehealth.Decrypt(wrongKey, nonce, ciphertext)
	require.Error(t, err)
}

func TestEncrypt_InvalidKeySize(t *testing.T) {
	_, _, err := googlehealth.Encrypt([]byte("too-short"), []byte("data"))
	require.Error(t, err)
}
