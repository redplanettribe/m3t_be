package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"multitrackticketing/internal/domain"
)

var userEmailRegexp = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

type userService struct {
	userRepo domain.UserRepository
}

// NewUserService creates a UserService with the given repository.
func NewUserService(userRepo domain.UserRepository) domain.UserService {
	return &userService{userRepo: userRepo}
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
	if user.Email != "" && !userEmailRegexp.MatchString(user.Email) {
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
