package repository

import (
	"context"
	"errors"

	"auth-service/internal/model"
	"github.com/jackc/pgx/v5"
)

type RoleRepository interface {
	Get(context.Context, string) (*model.Role, error)
}

type roleRepository struct {
	db interface {
		QueryRow(context.Context, string, ...any) pgx.Row
	}
}

func NewRoleRepository(db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}) RoleRepository {
	return &roleRepository{db: db}
}

func (r *roleRepository) Get(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.db.QueryRow(ctx, `
		SELECT name, is_admin, show_history, can_read, can_delete
		FROM roles WHERE name = $1
	`, name).Scan(&role.Name, &role.IsAdmin, &role.ShowHistory, &role.CanRead, &role.CanDelete)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}
