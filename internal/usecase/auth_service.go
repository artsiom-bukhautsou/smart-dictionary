package usecase

import (
	"fmt"
	"strconv"

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
	userID, err := s.authRepository.SignIn(login, password)
	if err != nil {
		return nil, fmt.Errorf("singIn: %w", err)
	}
	refresh, err := s.jwtAuth.GenerateRefresh(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	access, err := s.jwtAuth.GenerateAccess(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: access, Refresh: refresh}, nil
}

func (s AuthService) SignUp(credentials domain.AuthCredentials) (*domain.Token, error) {
	isUsernameExist, err := s.authRepository.DoesUsernameExist(credentials.Username)
	if err != nil {
		return nil, fmt.Errorf("signup: %w", err)
	}
	if isUsernameExist {
		return nil, fmt.Errorf("signup: username exists")
	}
	userID, err := s.authRepository.SignUp(credentials)
	if err != nil {
		return nil, fmt.Errorf("signup failed: %w", err)
	}
	refresh, err := s.jwtAuth.GenerateRefresh(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	access, err := s.jwtAuth.GenerateAccess(fmt.Sprintf("%d", userID))
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: access, Refresh: refresh}, nil
}

func (s AuthService) DeleteUser(userID string) error {
	id, err := strconv.Atoi(userID)
	if err != nil {
		return fmt.Errorf("userID has unsupported format: %w", err)
	}
	err = s.authRepository.RemoveUser(id)
	if err != nil {
		return fmt.Errorf("failed to remove user: %w", err)
	}
	return nil
}
