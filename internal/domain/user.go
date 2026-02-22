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
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Salt         string    `json:"-"` // per-user salt used when hashing password
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// NewUser returns a new User with the given fields. ID is typically set by the repository on create.
func NewUser(email, passwordHash, salt, name string, createdAt, updatedAt time.Time) *User {
	return &User{
		Email:        email,
		PasswordHash: passwordHash,
		Salt:         salt,
		Name:         name,
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
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

// UserService defines the business logic for user profile operations.
type UserService interface {
	GetByID(ctx context.Context, id string) (*User, error)
	Update(ctx context.Context, user *User) error
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
