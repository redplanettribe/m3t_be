package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"multitrackticketing/internal/domain"
)

type bcryptHasher struct {
	cost int
}

// NewBcryptHasher returns a PasswordHasher that uses bcrypt with salt+password
// derived via SHA256 (same algorithm as the previous auth service).
func NewBcryptHasher(cost int) domain.PasswordHasher {
	return &bcryptHasher{cost: cost}
}

func (h *bcryptHasher) GenerateSalt() (string, error) {
	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	return hex.EncodeToString(saltBytes), nil
}

func (h *bcryptHasher) Hash(salt, password string) (string, error) {
	saltedInput := salt + password
	sum := sha256.Sum256([]byte(saltedInput))
	bcryptInput := hex.EncodeToString(sum[:])
	hash, err := bcrypt.GenerateFromPassword([]byte(bcryptInput), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func (h *bcryptHasher) Compare(hash, salt, password string) error {
	saltedInput := salt + password
	sum := sha256.Sum256([]byte(saltedInput))
	bcryptInput := hex.EncodeToString(sum[:])
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(bcryptInput))
}
