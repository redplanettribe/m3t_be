package domain

import (
	"context"
	"time"
)

// User represents a registered user
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"` // per-user salt used when hashing password
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Role represents an application role (e.g. admin, attendee)
type Role struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

// UserRepository defines the interface for user storage
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	AssignRole(ctx context.Context, userID, roleID string) error
}

// RoleRepository defines the interface for role storage
type RoleRepository interface {
	GetByCode(ctx context.Context, code string) (*Role, error)
	ListByUserID(ctx context.Context, userID string) ([]*Role, error)
}

// AuthService defines the business logic for authentication
type AuthService interface {
	SignUp(ctx context.Context, email, password, name, role string) (*User, error)
	Login(ctx context.Context, email, password string) (token string, err error)
}
