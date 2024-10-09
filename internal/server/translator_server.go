package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/labstack/echo/v4"
)

type TranslatorServer struct {
	logger slog.Logger

	authService          usecase.AuthService
	jwtService           usecase.JWTAuth
	refreshTokenDuration time.Duration
	accessTokenDuration  time.Duration

	chatGPTAPIURL string
	apiKey        string

	translatorRepository TranslatorRepository
}

func NewTranslatorServer(
	authService usecase.AuthService,
	jwtService usecase.JWTAuth,
	accessTokenDuration time.Duration,
	refreshTokenDuration time.Duration,
	translatorRepository TranslatorRepository,
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

type TranslatorRepository interface {
	AddTranslation(ctx context.Context, translation domain.Translation, translatedFrom, translatedTo string) error
	GetAllTranslations(ctx context.Context) ([]domain.Translation, error)
	GetTranslation(ctx context.Context, lexicalItem, translateFrom, translateTo string) (*domain.Translation, error)
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
	var req domain.TranslationRequest
	err := c.Bind(&req)
	if err != nil {
		t.logger.Error("translate - failed to convert", slog.Any("err", err.Error()))
		return c.String(http.StatusBadRequest, "invalid input")
	}
	if _, ok := domain.SupportedLanguages[req.TranslateFrom]; !ok {
		return c.String(http.StatusBadRequest, "original language is not supported")
	}
	if _, ok := domain.SupportedLanguages[req.TranslateTo]; !ok {
		return c.String(http.StatusBadRequest, "target language is not supported")
	}
	const maxLexicalItemLength = 80
	if len(req.LexicalItem) > maxLexicalItemLength {
		return c.String(http.StatusBadRequest, fmt.Sprintf("max lexical item size is %d", maxLexicalItemLength))
	}
	req.LexicalItem = strings.ToLower(req.LexicalItem)
	translation, err := t.translatorRepository.GetTranslation(c.Request().Context(), req.LexicalItem, req.TranslateFrom, req.TranslateTo)
	if err != nil {
		t.logger.Error("failed to get translation", slog.Any("err", err.Error()))
		return c.String(http.StatusInternalServerError, "server error try again later")
	}
	if translation != nil {
		return c.JSON(http.StatusOK, translation)
	}
	promptTemplate := "Translate the lexical item: '%s', from '%s' to '%s'. Provide response in JSON format as follows: translatedFrom: string; translatedTo: string; originalLexicalItem: string; originalMeaning: string; originalExamples: [string, string]; translatedLexicalItem: string; translatedMeaning: string; translatedExamples: [string, string];. Ensure that 'originalMeaning' is in the original language ('translatedFrom')."
	prompt := fmt.Sprintf(promptTemplate, req.LexicalItem, req.TranslateFrom, req.TranslateTo)
	lexicalItem, err := t.callChatGPTAPI(prompt)
	if err != nil {
		t.logger.Error("failed to make a call to chatgpt", slog.Any("err", err.Error()))
		return c.String(http.StatusInternalServerError, "server error try again later")
	}
	if domain.IsTranslationNilOrEmpty(lexicalItem) {
		return c.String(http.StatusBadRequest, "couldn't translate")
	}
	go func() {
		err = t.translatorRepository.AddTranslation(context.Background(), *lexicalItem, req.TranslateFrom, req.TranslateTo)
		if err != nil {
			t.logger.Error(err.Error())
		}
	}()
	return c.JSON(http.StatusOK, lexicalItem)
}

func (t TranslatorServer) callChatGPTAPI(prompt string) (*domain.Translation, error) {
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

	var wordTranslation domain.Translation
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
		Secure:   true,                                  // Set to true if using HTTPS
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:     "refresh_token",
		Value:    token.Refresh,
		Path:     "/",
		Domain:   "",                                     // Set to your domain if needed
		Expires:  time.Now().Add(t.refreshTokenDuration), // Set expiration as per your requirements
		Secure:   true,                                   // Set to true if using HTTPS
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (t TranslatorServer) DeleteUsersAccount(c echo.Context) error {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
	}
	// Token usually comes as "Bearer <token>", so we split to get the actual token part
	token := strings.TrimSpace(strings.Replace(auth, "Bearer", "", 1))
	if token == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
	}
	sub, err := t.jwtService.GetSubFromAccessToken(token)
	if err != nil {
		t.logger.Error("failed to get sub from access token", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to delete user"})
	}
	err = t.authService.DeleteUser(sub)
	if err != nil {
		t.logger.Error("failed to delete user from the database", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to delete user"})
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("user %s deleted", sub))
}
