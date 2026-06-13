package googlehealth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// Encrypt encrypts plaintext with AES-256-GCM using key (which must be 32
// bytes), returning the ciphertext and the randomly generated nonce used to
// produce it. The nonce must be stored alongside the ciphertext for Decrypt.
func Encrypt(key, plaintext []byte) (ciphertext, nonce []byte, err error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}

	nonce = make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("generate nonce: %w", err)
	}

	return gcm.Seal(nil, nonce, plaintext, nil), nonce, nil
}

// Decrypt decrypts ciphertext with AES-256-GCM using key and nonce, both
// produced by a prior call to Encrypt.
func Decrypt(key, nonce, ciphertext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decrypt: authentication failed")
	}

	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	return gcm, nil
}
