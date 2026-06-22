package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// Cipher provides AES-256-GCM encryption for tokens at rest.
type Cipher struct {
	aead cipher.AEAD
}

// NewCipher builds a Cipher from a 32-byte key.
func NewCipher(key []byte) (*Cipher, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("token encryption key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// ResolveEncKey returns a 32-byte key from the configured base64 TOKEN_ENC_KEY.
// If tokenEncKey is empty it derives one from sessionSecret via SHA-256 and
// reports derived=true so the caller can warn (dev-only convenience).
func ResolveEncKey(tokenEncKey, sessionSecret string) (key []byte, derived bool, err error) {
	if tokenEncKey != "" {
		raw, decErr := base64.StdEncoding.DecodeString(tokenEncKey)
		if decErr != nil {
			return nil, false, fmt.Errorf("decode TOKEN_ENC_KEY (base64): %w", decErr)
		}
		if len(raw) != 32 {
			return nil, false, fmt.Errorf("TOKEN_ENC_KEY must decode to 32 bytes, got %d", len(raw))
		}
		return raw, false, nil
	}
	// Derive from the session secret. Always yields 32 bytes.
	sum := sha256.Sum256([]byte("mlm-token-enc:" + sessionSecret))
	return sum[:], true, nil
}

// Encrypt returns a base64 string of nonce||ciphertext. Empty input yields "".
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Decrypt reverses Encrypt. Empty input yields "".
func (c *Cipher) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

// RandomID returns a URL-safe base64 opaque identifier with n bytes of entropy.
func RandomID(nBytes int) (string, error) {
	if nBytes <= 0 {
		nBytes = 32
	}
	b := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
