package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type jwtIssuer struct {
	secret []byte
}

// NewJWTIssuer returns a TokenIssuer and TokenVerifier that sign/verify JWTs with HS256 using the given secret.
// The expiry passed to Issue is used for each token. The same value implements domain.TokenVerifier for protected routes.
func NewJWTIssuer(secret string, _ time.Duration) *jwtIssuer {
	return &jwtIssuer{secret: []byte(secret)}
}

// Verify parses and validates the JWT and returns the subject (user ID). Implements domain.TokenVerifier.
func (i *jwtIssuer) Verify(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return i.secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("invalid or expired token: %w", err)
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}
	return claims.Subject, nil
}

func (i *jwtIssuer) Issue(userID, email string, roles []string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
		Email: email,
		Roles: roles,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(i.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}
