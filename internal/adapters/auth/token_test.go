package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTIssuer_Issue(t *testing.T) {
	secret := "test-secret"
	expiry := 24 * time.Hour
	issuer := NewJWTIssuer(secret, expiry)

	token, err := issuer.Issue("user-123", "u@example.com", []string{"admin", "attendee"}, expiry)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Parse and verify claims
	parsed, err := jwt.ParseWithClaims(token, &jwtClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	require.NoError(t, err)
	require.True(t, parsed.Valid)
	claims, ok := parsed.Claims.(*jwtClaims)
	require.True(t, ok)
	assert.Equal(t, "user-123", claims.Subject)
	assert.Equal(t, "u@example.com", claims.Email)
	assert.Equal(t, []string{"admin", "attendee"}, claims.Roles)
}
