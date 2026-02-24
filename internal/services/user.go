package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"multitrackticketing/internal/domain"
)

const (
	defaultRole         = "attendee"
	loginCodeDigits     = 6
	loginCodeExpiryMins = 15
)

var (
	emailRegexp    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	loginCodeRegex = regexp.MustCompile(`^\d{6}$`)
)

type userService struct {
	userRepo      domain.UserRepository
	roleRepo      domain.RoleRepository
	loginCodeRepo domain.LoginCodeRepository
	tokenIssuer   domain.TokenIssuer
	tokenExpiry   time.Duration
	emailService  domain.EmailService
}

// NewUserService creates a UserService with the given repositories and auth ports.
func NewUserService(userRepo domain.UserRepository, roleRepo domain.RoleRepository, loginCodeRepo domain.LoginCodeRepository, tokenIssuer domain.TokenIssuer, tokenExpiry time.Duration, emailService domain.EmailService) domain.UserService {
	return &userService{
		userRepo:      userRepo,
		roleRepo:      roleRepo,
		loginCodeRepo: loginCodeRepo,
		tokenIssuer:   tokenIssuer,
		tokenExpiry:   tokenExpiry,
		emailService:  emailService,
	}
}

func (s *userService) RequestLoginCode(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRegexp.MatchString(email) {
		return fmt.Errorf("invalid email format")
	}
	code, err := generateLoginCode(loginCodeDigits)
	if err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}
	codeHash := hashLoginCode(code)
	expiresAt := time.Now().Add(loginCodeExpiryMins * time.Minute)
	if err := s.loginCodeRepo.Create(ctx, email, codeHash, expiresAt); err != nil {
		return fmt.Errorf("failed to store login code: %w", err)
	}
	if s.emailService != nil {
		data := &domain.LoginCodeEmailData{
			Email:            email,
			Code:             code,
			ExpiresInMinutes: loginCodeExpiryMins,
		}
		if err := s.emailService.SendLoginCode(ctx, data); err != nil {
			return fmt.Errorf("failed to send login code email: %w", err)
		}
	}
	return nil
}

func (s *userService) VerifyLoginCode(ctx context.Context, email, code string) (string, *domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if !emailRegexp.MatchString(email) {
		return "", nil, fmt.Errorf("invalid email format")
	}
	code = strings.TrimSpace(code)
	if !loginCodeRegex.MatchString(code) {
		return "", nil, fmt.Errorf("invalid or expired code")
	}
	codeHash := hashLoginCode(code)
	consumed, err := s.loginCodeRepo.Consume(ctx, email, codeHash)
	if err != nil {
		return "", nil, fmt.Errorf("failed to verify code: %w", err)
	}
	if !consumed {
		return "", nil, fmt.Errorf("invalid or expired code")
	}
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return "", nil, fmt.Errorf("failed to get user: %w", err)
		}
		// New user: create with no password, assign attendee role
		roleRecord, err := s.roleRepo.GetByCode(ctx, defaultRole)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get role %q: %w", defaultRole, err)
		}
		now := time.Now()
		user = domain.NewUser(email, "", "", "", "", now, now)
		if err := s.userRepo.Create(ctx, user); err != nil {
			return "", nil, fmt.Errorf("failed to create user: %w", err)
		}
		if err := s.userRepo.AssignRole(ctx, user.ID, roleRecord.ID); err != nil {
			return "", nil, fmt.Errorf("failed to assign role: %w", err)
		}
	}
	roles, err := s.roleRepo.ListByUserID(ctx, user.ID)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load roles: %w", err)
	}
	roleCodes := make([]string, len(roles))
	for i, r := range roles {
		roleCodes[i] = r.Code
	}
	token, err := s.tokenIssuer.Issue(user.ID, user.Email, roleCodes, s.tokenExpiry)
	if err != nil {
		return "", nil, fmt.Errorf("failed to sign token: %w", err)
	}
	return token, user, nil
}

func generateLoginCode(digits int) (string, error) {
	const digitspace = "0123456789"
	b := make([]byte, digits)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = digitspace[int(b[i])%len(digitspace)]
	}
	return string(b), nil
}

func hashLoginCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}

func (s *userService) GetByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

func (s *userService) Update(ctx context.Context, user *domain.User) error {
	user.Name = strings.TrimSpace(user.Name)
	user.LastName = strings.TrimSpace(user.LastName)
	if user.Email != "" && !emailRegexp.MatchString(user.Email) {
		return fmt.Errorf("invalid email format")
	}
	user.UpdatedAt = time.Now()
	if err := s.userRepo.Update(ctx, user); err != nil {
		if errors.Is(err, domain.ErrDuplicateEmail) {
			return err
		}
		if errors.Is(err, domain.ErrUserNotFound) {
			return err
		}
		return fmt.Errorf("failed to update user: %w", err)
	}
	return nil
}
