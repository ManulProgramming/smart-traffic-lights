package repository

import (
	"context"
	"crypto/sha256"
	"time"

	"auth-service/internal/model"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type SessionRepository interface {
	Create(context.Context, pgx.Tx, *model.Session) error
	GetActive(context.Context, uuid.UUID) (*model.Session, error)
	Revoke(context.Context, uuid.UUID) error
	RevokeAllForUser(context.Context, pgx.Tx, int64) error
}

type sessionRepository struct {
	db interface {
		QueryRow(context.Context, string, ...any) pgx.Row
		Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	}
}

func NewSessionRepository(db interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(ctx context.Context, tx pgx.Tx, session *model.Session) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`, session.ID, session.UserID, session.TokenHash, session.CreatedAt, session.ExpiresAt)
	return err
}

func (r *sessionRepository) GetActive(ctx context.Context, id uuid.UUID) (*model.Session, error) {
	var s model.Session
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, token_hash, created_at, expires_at, revoked_at
		FROM sessions
		WHERE id = $1 AND revoked_at IS NULL AND expires_at > $2
	`, id, time.Now().UTC()).Scan(
		&s.ID, &s.UserID, &s.TokenHash, &s.CreatedAt, &s.ExpiresAt, &s.RevokedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *sessionRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE sessions SET revoked_at = COALESCE(revoked_at, NOW()) WHERE id = $1`, id)
	return err
}

func (r *sessionRepository) RevokeAllForUser(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx, `UPDATE sessions SET revoked_at = COALESCE(revoked_at, NOW()) WHERE user_id = $1 AND revoked_at IS NULL`, userID)
	return err
}

func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
