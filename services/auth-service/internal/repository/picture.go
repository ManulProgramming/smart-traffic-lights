package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type PictureRepository interface {
	Create(context.Context, pgx.Tx, int64, []byte) error
	DeleteByUserID(context.Context, pgx.Tx, int64) error
}

type pictureRepository struct{}

func NewPictureRepository() PictureRepository {
	return &pictureRepository{}
}

func (r *pictureRepository) Create(ctx context.Context, tx pgx.Tx, userID int64, image []byte) error {
	_, err := tx.Exec(ctx, `INSERT INTO pictures (user_id, image) VALUES ($1, $2)`, userID, image)
	return err
}

func (r *pictureRepository) DeleteByUserID(ctx context.Context, tx pgx.Tx, userID int64) error {
	_, err := tx.Exec(ctx, `DELETE FROM pictures WHERE user_id = $1`, userID)
	return err
}
