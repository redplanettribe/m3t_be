package auth

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBcryptHasher_GenerateSalt(t *testing.T) {
	h := NewBcryptHasher(10)
	hexRe := regexp.MustCompile(`^[0-9a-f]{64}$`)

	for i := 0; i < 5; i++ {
		salt, err := h.GenerateSalt()
		require.NoError(t, err)
		assert.Regexp(t, hexRe, salt, "salt should be 64 hex characters")
	}
}

func TestBcryptHasher_Hash_and_Compare(t *testing.T) {
	h := NewBcryptHasher(10)
	salt, err := h.GenerateSalt()
	require.NoError(t, err)
	password := "my-secret-password"

	hash, err := h.Hash(salt, password)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	err = h.Compare(hash, salt, password)
	require.NoError(t, err)
}

func TestBcryptHasher_Compare_wrong_password(t *testing.T) {
	h := NewBcryptHasher(10)
	salt, err := h.GenerateSalt()
	require.NoError(t, err)
	hash, err := h.Hash(salt, "correct")
	require.NoError(t, err)

	err = h.Compare(hash, salt, "wrong")
	assert.Error(t, err)
}

func TestBcryptHasher_Compare_wrong_salt(t *testing.T) {
	h := NewBcryptHasher(10)
	salt1, _ := h.GenerateSalt()
	salt2, _ := h.GenerateSalt()
	hash, err := h.Hash(salt1, "password")
	require.NoError(t, err)

	err = h.Compare(hash, salt2, "password")
	assert.Error(t, err)
}
