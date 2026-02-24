package postgres

import (
	"context"
	"database/sql"
	"time"

	"multitrackticketing/internal/domain"
)

type loginCodeRepository struct {
	DB *sql.DB
}

// NewLoginCodeRepository returns a domain.LoginCodeRepository implemented with Postgres.
func NewLoginCodeRepository(db *sql.DB) domain.LoginCodeRepository {
	return &loginCodeRepository{DB: db}
}

func (r *loginCodeRepository) Create(ctx context.Context, email, codeHash string, expiresAt time.Time) error {
	query := `
		INSERT INTO login_codes (email, code_hash, expires_at)
		VALUES ($1, $2, $3)
	`
	_, err := r.DB.ExecContext(ctx, query, email, codeHash, expiresAt)
	return err
}

func (r *loginCodeRepository) Consume(ctx context.Context, email, codeHash string) (consumed bool, err error) {
	var id string
	query := `
		SELECT id FROM login_codes
		WHERE email = $1 AND code_hash = $2 AND expires_at > NOW()
		LIMIT 1
	`
	err = r.DB.QueryRowContext(ctx, query, email, codeHash).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	deleteQuery := `DELETE FROM login_codes WHERE id = $1`
	_, err = r.DB.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return false, err
	}
	return true, nil
}
