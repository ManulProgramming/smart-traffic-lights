package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"auth-service/internal/model"
	"auth-service/internal/repository"

	"github.com/jackc/pgx/v5"
)

type UpdateUserRequest struct {
	Name            *string
	Email           *string
	NewPassword     *string
	CurrentPassword string
	Picture         []byte
	HasPicture      bool
}

type UserService struct {
	users     repository.UserRepository
	sessions  repository.SessionRepository
	pictures  repository.PictureRepository
	passwords *PasswordService
	txManager interface {
		Begin(context.Context) (pgx.Tx, error)
	}
}

func NewUserService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	pictures repository.PictureRepository,
	passwords *PasswordService,
	txManager interface {
		Begin(context.Context) (pgx.Tx, error)
	},
) *UserService {
	return &UserService{users: users, sessions: sessions, pictures: pictures, passwords: passwords, txManager: txManager}
}

func (s *UserService) Get(ctx context.Context, id int64) (*model.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context, limit, offset int) ([]*model.User, int, error) {
	return s.users.List(ctx, limit, offset)
}

func (s *UserService) Update(ctx context.Context, id int64, req UpdateUserRequest) (*model.User, error) {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if ok, err := s.passwords.Verify(req.CurrentPassword, user.PasswordHash); err != nil || !ok {
		return nil, ErrInvalidCredentials
	}

	if req.Name != nil {
		user.Name = strings.TrimSpace(*req.Name)
	}
	if req.Email != nil {
		user.Email = strings.ToLower(strings.TrimSpace(*req.Email))
	}
	if req.NewPassword != nil {
		hash, err := s.passwords.Hash(*req.NewPassword)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = hash
	}

	tx, err := s.txManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.users.Update(ctx, tx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	if req.HasPicture {
		if err := s.pictures.DeleteByUserID(ctx, tx, id); err != nil {
			return nil, err
		}
		if err := s.pictures.Create(ctx, tx, id, req.Picture); err != nil {
			return nil, err
		}
	}

	if req.NewPassword != nil {
		if err := s.sessions.RevokeAllForUser(ctx, tx, id); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit user update: %w", err)
	}
	return user, nil
}

func (s *UserService) Delete(ctx context.Context, id int64, password string) error {
	user, err := s.users.GetByID(ctx, id)
	if err != nil {
		return err
	}
	ok, err := s.passwords.Verify(password, user.PasswordHash)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}

	tx, err := s.txManager.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.users.Delete(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
