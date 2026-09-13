package repository

import (
	"context"
	"errors"
	"strings"

	"auth-service/internal/model"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type UserRepository interface {
	Create(context.Context, pgx.Tx, *model.User) error
	GetByID(context.Context, int64) (*model.User, error)
	GetByLogin(context.Context, string) (*model.User, error)
	List(context.Context, int, int) ([]*model.User, int, error)
	Update(context.Context, pgx.Tx, *model.User) error
	Delete(context.Context, pgx.Tx, int64) error
}

type userRepository struct {
	db interface {
		QueryRow(context.Context, string, ...any) pgx.Row
		Query(context.Context, string, ...any) (pgx.Rows, error)
	}
}

func NewUserRepository(db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, tx pgx.Tx, user *model.User) error {
	err := tx.QueryRow(ctx, `
		INSERT INTO users (name, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`, user.Name, user.Email, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.CreatedAt)
	return mapDBError(err)
}

func (r *userRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, created_at, role
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) GetByLogin(ctx context.Context, login string) (*model.User, error) {
	var u model.User
	err := r.db.QueryRow(ctx, `
		SELECT id, name, email, password_hash, created_at, role
		FROM users
		WHERE name = $1 OR lower(email) = lower($1)
	`, login).Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, email, password_hash, created_at, role
		FROM users
		ORDER BY id
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt, &u.Role); err != nil {
			return nil, 0, err
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *userRepository) Update(ctx context.Context, tx pgx.Tx, user *model.User) error {
	err := tx.QueryRow(ctx, `
		UPDATE users
		SET name = $1, email = $2, password_hash = $3, role = $4
		WHERE id = $5
		RETURNING created_at
	`, user.Name, user.Email, user.PasswordHash, user.Role, user.ID).Scan(&user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return mapDBError(err)
}

func (r *userRepository) Delete(ctx context.Context, tx pgx.Tx, id int64) error {
	tag, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		if strings.Contains(pgErr.ConstraintName, "users_") {
			return ErrAlreadyExists
		}
	}
	return err
}
