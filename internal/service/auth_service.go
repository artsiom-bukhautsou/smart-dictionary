package service

import (
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/repository"
	"github.com/labstack/echo/v4"
	"log/slog"
)

type AuthService struct {
	userRepository *repository.UserRepository
}

// temporal cache, was made locally because we don't have many users at the moment.
var userNameWithPasswordCache = make(map[string]string)

func NewAuthService(userRepository *repository.UserRepository) *AuthService {
	return &AuthService{
		userRepository: userRepository,
	}
}

func (a AuthService) BasicAuth(username, password string, c echo.Context) (bool, error) {
	cachePassword, found := userNameWithPasswordCache[username]
	if found && cachePassword == password {
		return true, nil
	}
	err := a.userRepository.GetUser(username, password)
	if err != nil {
		slog.Error(err.Error())
		return false, nil
	}
	userNameWithPasswordCache[username] = password
	return true, nil
}

func (a AuthService) CreateUser(username string, password string) error {
	err := a.userRepository.CreateUser(username, password)
	if err != nil {
		slog.Error(err.Error())
		return fmt.Errorf("failed to craete a user")
	}
	return nil
}
