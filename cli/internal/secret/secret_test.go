package secret

import (
	"path/filepath"
	"testing"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	useTempConfig(t)

	plaintext := "Bearer super-secret-token"
	enc, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if enc == plaintext {
		t.Fatalf("ciphertext equals plaintext")
	}

	got, err := Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

func TestEncryptUniqueNonce(t *testing.T) {
	useTempConfig(t)

	a, err := Encrypt("same")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Encrypt("same")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("expected different ciphertexts for repeated plaintext (nonce reuse)")
	}
}

func TestDecryptTampered(t *testing.T) {
	useTempConfig(t)

	enc, err := Encrypt("value")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(enc + "AA"); err == nil {
		t.Fatalf("expected error decrypting tampered ciphertext")
	}
}

// useTempConfig points the config dir at a temp location and resets the key
// cache so each test generates its own key.
func useTempConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)                                  // darwin + linux fallback
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "cfg")) // linux
	mu.Lock()
	cachedKey = nil
	mu.Unlock()
}
