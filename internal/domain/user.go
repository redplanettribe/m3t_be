package domain

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors for user operations.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrDuplicateEmail  = errors.New("email already in use")
)

// User represents a registered user
// swagger:model User
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewUser returns a new User with the given fields. ID is typically set by the repository on create.
func NewUser(email, name, lastName string, createdAt, updatedAt time.Time) *User {
	return &User{
		Email:     email,
		Name:      name,
		LastName:  lastName,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

// Role represents an application role (e.g. admin, attendee)
type Role struct {
	ID   string `json:"id"`
	Code string `json:"code"`
}

// NewRole returns a new Role with the given id and code.
func NewRole(id, code string) *Role {
	return &Role{ID: id, Code: code}
}

// PasswordHasher handles salt generation, hashing, and verification.
// Implementations may use bcrypt, argon2, etc.
type PasswordHasher interface {
	GenerateSalt() (string, error)
	Hash(salt, password string) (hash string, err error)
	Compare(hash, salt, password string) error
}

// TokenIssuer issues tokens (e.g. JWT) for an authenticated user.
type TokenIssuer interface {
	Issue(userID, email string, roles []string, expiry time.Duration) (string, error)
}

// TokenVerifier verifies a token and returns the authenticated user ID.
type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

// UserRepository defines the interface for user storage
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error
	AssignRole(ctx context.Context, userID, roleID string) error
}

// LoginCodeRepository defines the interface for one-time login code storage.
type LoginCodeRepository interface {
	Create(ctx context.Context, email, codeHash string, expiresAt time.Time) error
	Consume(ctx context.Context, email, codeHash string) (consumed bool, err error)
}

// UserService defines the business logic for user profile and authentication.
type UserService interface {
	RequestLoginCode(ctx context.Context, email string) error
	VerifyLoginCode(ctx context.Context, email, code string) (token string, user *User, err error)
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error
}

// RoleRepository defines the interface for role storage
type RoleRepository interface {
	GetByCode(ctx context.Context, code string) (*Role, error)
	ListByUserID(ctx context.Context, userID string) ([]*Role, error)
}
