package postgres

import (
	"context"
	"database/sql"
	"multitrackticketing/internal/domain"
)

type roleRepository struct {
	DB *sql.DB
}

func NewRoleRepository(db *sql.DB) domain.RoleRepository {
	return &roleRepository{DB: db}
}

func (r *roleRepository) GetByCode(ctx context.Context, code string) (*domain.Role, error) {
	query := `
		SELECT id, code
		FROM roles
		WHERE code = $1
	`
	role := &domain.Role{}
	err := r.DB.QueryRowContext(ctx, query, code).Scan(&role.ID, &role.Code)
	if err != nil {
		return nil, err
	}
	return role, nil
}

func (r *roleRepository) ListByUserID(ctx context.Context, userID string) ([]*domain.Role, error) {
	query := `
		SELECT r.id, r.code
		FROM roles r
		INNER JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`
	rows, err := r.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*domain.Role
	for rows.Next() {
		role := &domain.Role{}
		if err := rows.Scan(&role.ID, &role.Code); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}
