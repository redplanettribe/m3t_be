package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"multitrackticketing/internal/domain"
)

type jwtClaims struct {
	jwt.RegisteredClaims
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type jwtIssuer struct {
	secret []byte
}

// NewJWTIssuer returns a TokenIssuer that signs JWTs with HS256 using the given secret.
// The expiry passed to Issue is used for each token.
func NewJWTIssuer(secret string, _ time.Duration) domain.TokenIssuer {
	return &jwtIssuer{secret: []byte(secret)}
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
