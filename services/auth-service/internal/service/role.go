package service

import (
	"context"

	"auth-service/internal/repository"
)

type AuthorizationService struct {
	roles repository.RoleRepository
}

func NewAuthorizationService(roles repository.RoleRepository) *AuthorizationService {
	return &AuthorizationService{roles: roles}
}

func (s *AuthorizationService) CanDelete(ctx context.Context, role string) (bool, error) {
	r, err := s.roles.Get(ctx, role)
	if err != nil {
		return false, err
	}
	return r.CanDelete, nil
}
