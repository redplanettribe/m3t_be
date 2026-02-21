package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"
)

type userRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) domain.UserRepository {
	return &userRepository{DB: db}
}

func (r *userRepository) Create(ctx context.Context, u *domain.User) error {
	query := `
		INSERT INTO users (email, password_hash, salt, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	return r.DB.QueryRowContext(ctx, query, u.Email, u.PasswordHash, u.Salt, u.Name, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, salt, name, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	u := &domain.User{}
	err := r.DB.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Salt, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, salt, name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	u := &domain.User{}
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Salt, &u.Name, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepository) AssignRole(ctx context.Context, userID, roleID string) error {
	query := `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`
	_, err := r.DB.ExecContext(ctx, query, userID, roleID)
	return err
}
