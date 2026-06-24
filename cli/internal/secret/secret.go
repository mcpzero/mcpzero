// Package secret encrypts small local secrets (e.g. upstream MCP auth headers)
// at rest using AES-256-GCM with a locally-generated key.
//
// Threat model: the key lives in the same config dir as the encrypted data
// (`<config>/mcpzero/state.key`, mode 0600). This protects against accidental
// sharing/backup of a tunnel state file, but not against an attacker who can
// already read the whole config directory.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

const keyLen = 32 // AES-256

var (
	mu        sync.Mutex
	cachedKey []byte
)

func keyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(dir, "mcpzero", "state.key"), nil
}

// loadOrCreateKey returns the local encryption key, generating and persisting
// one (0600) on first use.
func loadOrCreateKey() ([]byte, error) {
	mu.Lock()
	defer mu.Unlock()
	if cachedKey != nil {
		return cachedKey, nil
	}

	path, err := keyPath()
	if err != nil {
		return nil, err
	}

	if data, err := os.ReadFile(path); err == nil {
		if len(data) != keyLen {
			return nil, fmt.Errorf("invalid state key length %d (expected %d)", len(data), keyLen)
		}
		cachedKey = data
		return cachedKey, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read state key: %w", err)
	}

	key := make([]byte, keyLen)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate state key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(path, key, 0o600); err != nil {
		return nil, fmt.Errorf("write state key: %w", err)
	}
	cachedKey = key
	return cachedKey, nil
}

// Encrypt returns base64(nonce || ciphertext) for the given plaintext.
func Encrypt(plaintext string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt.
func Decrypt(encoded string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(key)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode secret: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plaintext), nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	return gcm, nil
}
