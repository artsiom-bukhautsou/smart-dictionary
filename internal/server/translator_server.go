package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type TranslatorServer struct {
	logger slog.Logger

	authService          usecase.AuthService
	jwtService           usecase.JWTAuth
	refreshTokenDuration time.Duration
	accessTokenDuration  time.Duration

	chatGPTAPIURL string
	apiKey        string

	translatorRepository domain.TranslatorRepository
}

func NewTranslatorServer(
	authService usecase.AuthService,
	jwtService usecase.JWTAuth,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	translatorRepository domain.TranslatorRepository,
	logger slog.Logger, chatGPTAPIURL string,
	apiKey string) *TranslatorServer {
	return &TranslatorServer{
		authService:          authService,
		jwtService:           jwtService,
		accessTokenDuration:  accessTokenDuration,
		refreshTokenDuration: refreshTokenDuration,
		translatorRepository: translatorRepository,
		logger:               logger,
		chatGPTAPIURL:        chatGPTAPIURL,
		apiKey:               apiKey,
	}
}

func (t TranslatorServer) SignIn(c echo.Context) error {
	var creds domain.AuthCredentials
	err := c.Bind(&creds)
	if err != nil {
		t.logger.Error("signin - failed to convert", slog.Any("err", err.Error()))
		return c.String(http.StatusBadRequest, "invalid input")
	}
	token, err := t.authService.SignIn(creds.Username, creds.Password)
	if err != nil {
		t.logger.Error("failed to signin", slog.Any("err", err.Error()))
		return c.String(http.StatusUnauthorized, "unauthorized")
	}
	t.enrichAuthToken(c, token)
	return c.String(http.StatusOK, "successfully authenticated")
}

func (t TranslatorServer) SignUp(c echo.Context) error {
	var creds domain.AuthCredentials
	err := c.Bind(&creds)
	if err != nil {
		t.logger.Error("signup - failed to convert", slog.Any("err", err.Error()))
		return c.String(http.StatusBadRequest, "invalid input")
	}
	token, err := t.authService.SignUp(creds)
	if err != nil {
		t.logger.Error("failed to signup", slog.Any("err", err.Error()))
		return c.String(http.StatusInternalServerError, "failed to Sign-Up")
	}
	t.enrichAuthToken(c, token)
	return c.String(http.StatusOK, "successfully sign up")
}

func (t TranslatorServer) RefreshRefreshToken(c echo.Context) error {
	req := c.Request()
	refreshTokenCookie, err := req.Cookie("refresh_token")
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			log.Println("refresh token cookie not found")
			return c.JSON(http.StatusUnauthorized, map[string]string{"message": "refresh token not found"})
		}
		log.Println("error retrieving refresh token cookie:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
	}
	refreshToken := refreshTokenCookie.Value
	updatedTokens, err := t.jwtService.RefreshRefreshToken(refreshToken)
	if err != nil {
		log.Println("error refreshing refresh token:", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "failed to refresh refresh token"})
	}

	t.enrichAuthToken(c, updatedTokens)
	return c.String(http.StatusOK, "successfully refreshed tokens")
}

func (t TranslatorServer) Translate(c echo.Context) error {
	var req domain.RequestMessage
	err := c.Bind(&req)
	if err != nil {
		t.logger.Error("translate - failed to convert", slog.Any("err", err.Error()))
		return c.String(http.StatusBadRequest, "invalid input")
	}
	req.Word = strings.ToLower(req.Word)
	translation, err := t.translatorRepository.GetWordTranslation(c.Request().Context(), req.Word)
	if err != nil {
		t.logger.Error("failed to get translation", slog.Any("err", err.Error()))
		return c.String(http.StatusInternalServerError, "server error try again later")
	}
	if translation != nil {
		return c.JSON(http.StatusOK, translation)
	}
	message, err := t.callChatGPTAPI("Translate the word, provide response in the following json format: word(string), meaning (string), examples (string array size 2), russianTranslation (string), meaningRussian (string) examplesRussian (string array size 2). Word to translate:" + req.Word)
	if err != nil {
		t.logger.Error("failed to make a call to chatgpt", slog.Any("err", err.Error()))
		return c.String(http.StatusInternalServerError, "server error try again later")
	}
	go func() {
		err = t.translatorRepository.AddWordTranslation(context.Background(), *message)
		if err != nil {
			t.logger.Error(err.Error())
		}
	}()
	return c.JSON(http.StatusOK, message)
}

func (t TranslatorServer) callChatGPTAPI(prompt string) (*domain.WordTranslation, error) {
	requestBody := fmt.Sprintf(`{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "%s"}]}`, prompt)

	req, err := http.NewRequest("POST", t.chatGPTAPIURL, bytes.NewBuffer([]byte(requestBody)))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+t.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var chatGPTResp domain.ChatGPTResponse
	err = json.Unmarshal(body, &chatGPTResp)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall response: %w", err)
	}
	if len(chatGPTResp.Choices) != 1 {
		return nil, fmt.Errorf("expected number of choices is 1, actual %d", len(chatGPTResp.Choices))
	}

	var wordTranslation domain.WordTranslation
	err = json.Unmarshal([]byte(chatGPTResp.Choices[0].Message.Content), &wordTranslation)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w, received string is: %s", err, chatGPTResp.Choices[0].Message.Content)
	}

	return &wordTranslation, nil
}

func (t TranslatorServer) enrichAuthToken(c echo.Context, token *domain.Token) {
	c.SetCookie(&http.Cookie{
		Name:     "access_token",
		Value:    token.Access,
		Path:     "/",
		Domain:   "",                                    // Set to your domain if needed
		Expires:  time.Now().Add(t.accessTokenDuration), // Set expiration as per your requirements
		Secure:   false,                                 // Set to true if using HTTPS
		HttpOnly: false,                                 // to be able to take cookies by frontend
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    token.Refresh,
		Path:     "/",
		Domain:   "",                                     // Set to your domain if needed
		Expires:  time.Now().Add(t.refreshTokenDuration), // Set expiration as per your requirements
		Secure:   false,                                  // Set to true if using HTTPS
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}
