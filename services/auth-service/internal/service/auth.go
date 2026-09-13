package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"auth-service/internal/model"
	"auth-service/internal/repository"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type RegisterRequest struct {
	Name       string
	Email      string
	Password   string
	Picture    []byte
	HasPicture bool
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	User      *model.User
}

type Principal struct {
	UserID    int64
	SessionID uuid.UUID
	Role      string
}

type AuthService struct {
	users      repository.UserRepository
	sessions   repository.SessionRepository
	roles      repository.RoleRepository
	pictures   repository.PictureRepository
	passwords  *PasswordService
	jwt        *JWTService
	sessionTTL time.Duration
	txManager  interface {
		Begin(context.Context) (pgx.Tx, error)
	}
}

func NewAuthService(
	users repository.UserRepository,
	sessions repository.SessionRepository,
	roles repository.RoleRepository,
	pictures repository.PictureRepository,
	passwords *PasswordService,
	jwt *JWTService,
	sessionTTL time.Duration,
	txManager interface {
		Begin(context.Context) (pgx.Tx, error)
	},
) *AuthService {
	return &AuthService{users: users, sessions: sessions, roles: roles, pictures: pictures, passwords: passwords, jwt: jwt, sessionTTL: sessionTTL, txManager: txManager}
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*AuthResult, error) {
	hash, err := s.passwords.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := &model.User{Name: req.Name, Email: req.Email, PasswordHash: hash, Role: "USER"}

	tx, err := s.beginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if err := s.users.Create(ctx, tx, user); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, ErrUserAlreadyExists
		}
		return nil, err
	}

	if req.HasPicture {
		if err := s.pictures.Create(ctx, tx, user.ID, req.Picture); err != nil {
			return nil, err
		}
	}

	now := time.Now().UTC()
	sessionID := uuid.New()
	token, expiresAt, err := s.jwt.Generate(user.ID, user.Role, sessionID, now)
	if err != nil {
		return nil, err
	}

	session := &model.Session{ID: sessionID, UserID: user.ID, TokenHash: repository.HashToken(token), CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL)}
	if err := s.sessions.Create(ctx, tx, session); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit registration: %w", err)
	}
	return &AuthResult{Token: token, ExpiresAt: expiresAt, User: user}, nil
}

func (s *AuthService) Login(ctx context.Context, login, password string) (*AuthResult, error) {
	user, err := s.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}
	ok, err := s.passwords.Verify(password, user.PasswordHash)
	if err != nil {
		return nil, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return nil, ErrInvalidCredentials
	}

	tx, err := s.beginTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	sessionID := uuid.New()
	token, expiresAt, err := s.jwt.Generate(user.ID, user.Role, sessionID, now)
	if err != nil {
		return nil, err
	}
	session := &model.Session{ID: sessionID, UserID: user.ID, TokenHash: repository.HashToken(token), CreatedAt: now, ExpiresAt: now.Add(s.sessionTTL)}
	if err := s.sessions.Create(ctx, tx, session); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit login: %w", err)
	}
	return &AuthResult{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) Validate(ctx context.Context, token string) (*Principal, error) {
	claims, err := s.jwt.Parse(token)
	if err != nil {
		return nil, ErrInvalidToken
	}
	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return nil, ErrInvalidToken
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	session, err := s.sessions.GetActive(ctx, sessionID)
	if err != nil || session.UserID != userID {
		return nil, ErrInvalidToken
	}
	if !constantTimeEqual(repository.HashToken(token), session.TokenHash) {
		return nil, ErrInvalidToken
	}
	return &Principal{UserID: userID, SessionID: sessionID, Role: claims.Role}, nil
}

func (s *AuthService) Logout(ctx context.Context, principal *Principal) error {
	return s.sessions.Revoke(ctx, principal.SessionID)
}

func (s *AuthService) beginTx(ctx context.Context) (pgx.Tx, error) {
	if s.txManager == nil {
		return nil, errors.New("transaction manager is not configured")
	}
	return s.txManager.Begin(ctx)
}

func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range a {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}
