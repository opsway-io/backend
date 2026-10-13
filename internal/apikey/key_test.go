package apikey

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateKey(t *testing.T) {
	a, err := GenerateKey()
	require.NoError(t, err)
	b, err := GenerateKey()
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
	assert.True(t, LooksLikeKey(a))
	assert.False(t, LooksLikeKey("opsw_short"))
	assert.False(t, LooksLikeKey("eyJhbGciOiJIUzI1NiJ9.e30.x"))
}

func TestHashKey(t *testing.T) {
	key, err := GenerateKey()
	require.NoError(t, err)

	hash := HashKey(key)
	assert.Len(t, hash, 64)
	assert.Equal(t, hash, HashKey(key), "hash must be deterministic")
	assert.NotContains(t, hash, key[len(KeyPrefix):])
	assert.True(t, Matches(key, hash))
	assert.False(t, Matches(key+"x", hash))

	// Known SHA-256 vector
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", HashKey(""))
}

func TestDisplayPrefix(t *testing.T) {
	assert.Equal(t, "opsw_abcdef", DisplayPrefix("opsw_abcdef0123456789"))
	assert.Equal(t, "opsw_", DisplayPrefix("opsw_"))
}
