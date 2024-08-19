package usecase

import (
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
)

type AuthService struct {
	authRepository infrastructure.AuthRepository
	jwtAuth        JWTAuth
}

func NewAuthService(authRepository infrastructure.AuthRepository, jwtAuth JWTAuth) *AuthService {
	return &AuthService{authRepository: authRepository, jwtAuth: jwtAuth}
}

func (s AuthService) SignIn(login, password string) (*domain.Token, error) {
	err := s.authRepository.SignIn(login, password)
	if err != nil {
		return nil, fmt.Errorf("singIn: %w", err)
	}
	refresh, err := s.jwtAuth.GenerateRefresh(login)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	err = s.authRepository.UpdateRefreshToken(login, refresh)
	if err != nil {
		return nil, fmt.Errorf("failed to update refresh token: %w", err)
	}
	access, err := s.jwtAuth.GenerateAccess(login)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: access, Refresh: refresh}, nil
}

func (s AuthService) SignUp(credentials domain.AuthCredentials) (*domain.Token, error) {
	isUsernameExist, err := s.authRepository.IsUsernameExist(credentials.Username)
	if err == nil {
		return nil, fmt.Errorf("signup: %w", err)
	}
	if isUsernameExist {
		return nil, fmt.Errorf("signup: username exists")
	}
	err = s.authRepository.SignUp(credentials)
	if err != nil {
		return nil, fmt.Errorf("signup failed: %w", err)
	}
	credentials.RefreshToken, err = s.jwtAuth.GenerateRefresh(credentials.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	access, err := s.jwtAuth.GenerateAccess(credentials.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: access, Refresh: credentials.RefreshToken}, nil
}
