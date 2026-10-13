package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/pkg/errors"
)

const (
	// KeyPrefix identifies opsway API keys, which makes leaked keys easy to detect.
	KeyPrefix = "opsw_"
	// displayPrefixLength is the number of characters of the key stored in plaintext for identification.
	displayPrefixLength = len(KeyPrefix) + 6
	secretBytes         = 32
)

// GenerateKey returns a new random plaintext API key.
func GenerateKey() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", errors.Wrap(err, "failed to read random bytes")
	}

	return KeyPrefix + hex.EncodeToString(b), nil
}

// HashKey returns the hex encoded SHA-256 hash of a plaintext key. API keys are high entropy random
// values, so a fast unsalted hash is sufficient and allows lookups by hash.
func HashKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))

	return hex.EncodeToString(sum[:])
}

// DisplayPrefix returns the non-secret leading part of a key.
func DisplayPrefix(plaintext string) string {
	if len(plaintext) <= displayPrefixLength {
		return plaintext
	}

	return plaintext[:displayPrefixLength]
}

// LooksLikeKey reports whether a token has the API key format.
func LooksLikeKey(token string) bool {
	return strings.HasPrefix(token, KeyPrefix) && len(token) == len(KeyPrefix)+secretBytes*2
}

// Matches compares a plaintext key with a stored hash in constant time.
func Matches(plaintext, hash string) bool {
	return subtle.ConstantTimeCompare([]byte(HashKey(plaintext)), []byte(hash)) == 1
}
