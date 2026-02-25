package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/lib/pq"
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
		INSERT INTO users (email, name, last_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	name := sql.NullString{String: u.Name, Valid: u.Name != ""}
	lastName := sql.NullString{String: u.LastName, Valid: u.LastName != ""}
	return r.DB.QueryRowContext(ctx, query, u.Email, name, lastName, u.CreatedAt, u.UpdatedAt).Scan(&u.ID)
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, name, last_name, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	u := &domain.User{}
	var name, lastName sql.NullString
	err := r.DB.QueryRowContext(ctx, query, email).Scan(&u.ID, &u.Email, &name, &lastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Name = name.String
	u.LastName = lastName.String
	return u, nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, name, last_name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	u := &domain.User{}
	var name, lastName sql.NullString
	err := r.DB.QueryRowContext(ctx, query, id).Scan(&u.ID, &u.Email, &name, &lastName, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.Name = name.String
	u.LastName = lastName.String
	return u, nil
}

func (r *userRepository) Update(ctx context.Context, u *domain.User) error {
	query := `
		UPDATE users
		SET name = $1, last_name = $2, email = $3, updated_at = $4
		WHERE id = $5
	`
	result, err := r.DB.ExecContext(ctx, query, u.Name, u.LastName, u.Email, u.UpdatedAt, u.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return domain.ErrDuplicateEmail
		}
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return domain.ErrUserNotFound
	}
	return nil
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
