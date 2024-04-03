package service

import (
	"github.com/bukhavtsov/artems-dictionary/internal/repository"
	"github.com/labstack/echo/v4"
	"log/slog"
)

type AuthService struct {
	userRepository *repository.UserRepository
}

func NewAuthService(userRepository *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepository: userRepository,
	}
}

func (a AuthService) BasicAuth(username, password string, c echo.Context) (bool, error) {
	err := a.userRepository.GetUser(username, password)
	if err != nil {
		slog.Error(err.Error())
		return false, nil
	}
	return true, nil
}
