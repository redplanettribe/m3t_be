package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"multitrackticketing/internal/domain"
)

const (
	bcryptCost     = 10
	minPasswordLen = 8
	defaultRole    = "attendee"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type authService struct {
	userRepo   domain.UserRepository
	roleRepo   domain.RoleRepository
	jwtSecret  []byte
	jwtExpiry  time.Duration
}

// NewAuthService creates an AuthService with the given repositories and JWT config
func NewAuthService(userRepo domain.UserRepository, roleRepo domain.RoleRepository, jwtSecret string, jwtExpiry time.Duration) domain.AuthService {
	return &authService{
		userRepo:  userRepo,
		roleRepo:  roleRepo,
		jwtSecret: []byte(jwtSecret),
		jwtExpiry: jwtExpiry,
	}
}

func (s *authService) SignUp(ctx context.Context, email, password, name, role string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRegexp.MatchString(email) {
		return nil, fmt.Errorf("invalid email format")
	}
	if len(password) < minPasswordLen {
		return nil, fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}

	roleCode := strings.TrimSpace(strings.ToLower(role))
	if roleCode != "admin" && roleCode != "attendee" {
		roleCode = defaultRole
	}

	saltBytes := make([]byte, 32)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)

	saltedInput := salt + password
	sum := sha256.Sum256([]byte(saltedInput))
	bcryptInput := hex.EncodeToString(sum[:])
	hash, err := bcrypt.GenerateFromPassword([]byte(bcryptInput), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	now := time.Now()
	user := &domain.User{
		Email:        email,
		PasswordHash: string(hash),
		Salt:         salt,
		Name:         strings.TrimSpace(name),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	roleRecord, err := s.roleRepo.GetByCode(ctx, roleCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get role %q: %w", roleCode, err)
	}
	if err := s.userRepo.AssignRole(ctx, user.ID, roleRecord.ID); err != nil {
		return nil, fmt.Errorf("failed to assign role: %w", err)
	}

	return user, nil
}

type jwtClaims struct {
	jwt.RegisteredClaims
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	saltedInput := user.Salt + password
	sum := sha256.Sum256([]byte(saltedInput))
	bcryptInput := hex.EncodeToString(sum[:])
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(bcryptInput)); err != nil {
		return "", fmt.Errorf("invalid credentials")
	}

	roles, err := s.roleRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to load roles: %w", err)
	}
	roleCodes := make([]string, len(roles))
	for i, r := range roles {
		roleCodes[i] = r.Code
	}

	now := time.Now()
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiry)),
		},
		Email: user.Email,
		Roles: roleCodes,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return tokenString, nil
}
