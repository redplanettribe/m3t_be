package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"multitrackticketing/internal/domain"
)

const (
	minPasswordLen = 8
	defaultRole    = "attendee"
)

var emailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type authService struct {
	userRepo       domain.UserRepository
	roleRepo       domain.RoleRepository
	passwordHasher domain.PasswordHasher
	tokenIssuer    domain.TokenIssuer
	tokenExpiry    time.Duration
}

// NewAuthService creates an AuthService with the given repositories and auth ports.
func NewAuthService(userRepo domain.UserRepository, roleRepo domain.RoleRepository, passwordHasher domain.PasswordHasher, tokenIssuer domain.TokenIssuer, tokenExpiry time.Duration) domain.AuthService {
	return &authService{
		userRepo:       userRepo,
		roleRepo:       roleRepo,
		passwordHasher: passwordHasher,
		tokenIssuer:    tokenIssuer,
		tokenExpiry:    tokenExpiry,
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

	salt, err := s.passwordHasher.GenerateSalt()
	if err != nil {
		return nil, err
	}
	hash, err := s.passwordHasher.Hash(salt, password)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := domain.NewUser(email, hash, salt, strings.TrimSpace(name), now, now)
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

func (s *authService) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.userRepo.GetByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return "", fmt.Errorf("invalid credentials")
	}
	if err := s.passwordHasher.Compare(user.PasswordHash, user.Salt, password); err != nil {
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

	token, err := s.tokenIssuer.Issue(user.ID, user.Email, roleCodes, s.tokenExpiry)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}
	return token, nil
}
