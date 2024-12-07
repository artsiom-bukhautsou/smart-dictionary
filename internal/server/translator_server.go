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
	"strconv"
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
	AddTranslation(ctx context.Context, translation domain.Translation, translatedFrom, translatedTo string) (int, error)
	GetAllTranslations(ctx context.Context) ([]domain.Translation, error)
	GetTranslation(ctx context.Context, lexicalItem, translateFrom, translateTo string) (*domain.Translation, error)
	GetCollectionsByUserID(ctx context.Context, userID int) ([]domain.Collection, error)
	CreateCollectionByUserID(ctx context.Context, userID int, collectionName string) (int, error)
	DeleteCollectionByUserID(ctx context.Context, userID int, collectionID int) error
	GetCollectionTranslations(ctx context.Context, collectionID int, translationIDs []int, userID int) ([]domain.CollectionTranslation, error)
	DeleteCollectionTranslations(ctx context.Context, translationIDs []int, collectionID int, userID int) error
	CreateCollection(ctx context.Context, userID int, collectionName string) (int, error)
	SaveToCollectionLexicalItem(ctx context.Context, collectionID, translationID int) (int, error)
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
	sub, failed, err := t.GetSubFromToken(c)
	if failed {
		t.logger.Error("failed to get sub from token", slog.Any("err", err.Error()))
	}
	var req domain.TranslationRequest
	err = c.Bind(&req)
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
	if req.SavingEnabled {
		go func() {
			ctx := context.Background()
			translationID, err := t.translatorRepository.AddTranslation(ctx, *lexicalItem, req.TranslateFrom, req.TranslateTo)
			if err != nil {
				t.logger.Error(err.Error())
				return
			}
			t.addTranslationToCollection(ctx, sub, translationID, req.CollectionID)

		}()
	}
	return c.JSON(http.StatusOK, lexicalItem)
}

func (t TranslatorServer) addTranslationToCollection(ctx context.Context, sub string, translationID int, collectionID int) {
	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return
	}

	if collectionID == 0 {
		// set default collection
		collections, err := t.translatorRepository.GetCollectionsByUserID(ctx, userID)
		if err != nil {
			t.logger.Error("GetCollectionsByUserID failed", slog.Any("err", err.Error()))
			return
		}
		var collectionID int
		if len(collections) == 0 {
			collectionID, err = t.translatorRepository.CreateCollection(ctx, userID, "default")
			if err != nil {
				t.logger.Error("CreateCollection failed", slog.Any("err", err.Error()))
				return
			}
		} else {
			collectionID = collections[0].ID
		}
		_, err = t.translatorRepository.SaveToCollectionLexicalItem(ctx, collectionID, translationID)
		if err != nil {
			t.logger.Error("SaveToCollectionLexicalItem failed", slog.Any("err", err.Error()))
			return
		}
		return
	}
	_, err = t.translatorRepository.SaveToCollectionLexicalItem(ctx, collectionID, translationID)
	if err != nil {
		t.logger.Error("SaveToCollectionLexicalItem failed", slog.Any("err", err.Error()))
		return
	}

}

func (t TranslatorServer) callChatGPTAPI(prompt string) (*domain.Translation, error) {
	requestBody := fmt.Sprintf(`{
		"model": "gpt-3.5-turbo",
		"messages": [
			{
				"role": "user",
				"content": %q
			}
		]
	}`, prompt)
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
	translationJSON := chatGPTResp.Choices[0].Message.Content

	var wordTranslation domain.Translation
	err = json.Unmarshal([]byte(translationJSON), &wordTranslation)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w, received string is: %s", err, chatGPTResp.Choices[0].Message.Content)
	}
	return &wordTranslation, nil
}

func (t TranslatorServer) enrichAuthToken(c echo.Context, token *domain.Token) {
	c.SetCookie(&http.Cookie{
		Name:    "access_token",
		Value:   token.Access,
		Path:    "/",
		Domain:  "",                                    // Set to your domain if needed
		Expires: time.Now().Add(t.accessTokenDuration), // Set expiration as per your requirements
		Secure:   true,                                  // Set to true if using HTTPS
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	c.SetCookie(&http.Cookie{
		Name:    "refresh_token",
		Value:   token.Refresh,
		Path:    "/",
		Domain:  "",                                     // Set to your domain if needed
		Expires: time.Now().Add(t.refreshTokenDuration), // Set expiration as per your requirements
		Secure:   true,                                   // Set to true if using HTTPS
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}

func (t TranslatorServer) DeleteUsersAccount(c echo.Context) error {
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}
	err := t.authService.DeleteUser(sub)
	if err != nil {
		t.logger.Error("failed to delete user from the database", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to delete user"})
	}
	return c.JSON(http.StatusOK, fmt.Sprintf("user %s deleted", sub))
}

func (t TranslatorServer) GetCollections(c echo.Context) error {
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	collections, err := t.translatorRepository.GetCollectionsByUserID(c.Request().Context(), userID)
	if err != nil {
		t.logger.Error("failed get decs for the user", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get collections for the user"})
	}
	return c.JSON(http.StatusOK, collections)
}

func (t TranslatorServer) CreateCollection(c echo.Context) error {
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	var req domain.CollectionCreateRequest
	err = c.Bind(&req)
	if err != nil {
		t.logger.Error("collectoin create request - failed to convert", slog.Any("err", err.Error()))
		return c.String(http.StatusBadRequest, "invalid input")
	}
	collectionID, err := t.translatorRepository.CreateCollectionByUserID(c.Request().Context(), userID, req.CollectionName)
	if err != nil {
		t.logger.Error("failed to create collection for the user", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to create collection for the user"})
	}
	resp := domain.CollectionCreateResponse{ID: collectionID}
	return c.JSON(http.StatusOK, resp)
}

func (t TranslatorServer) DeleteCollection(c echo.Context) error {
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	collectionIDParam := c.Param("collectionID")
	collectionID, err := strconv.Atoi(collectionIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid CollectionID",
		})
	}
	err = t.translatorRepository.DeleteCollectionByUserID(c.Request().Context(), userID, collectionID)
	if err != nil {
		t.logger.Error("failed to delete collection", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to delete collection"})
	}
	return c.NoContent(http.StatusNoContent)
}

func (t TranslatorServer) GetCollectionsTranslations(c echo.Context) error {
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}

	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	collectionIDParam := c.Param("collectionID")
	collectionID, err := strconv.Atoi(collectionIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid CollectionID",
		})
	}

	// Get TranslationIDs from path parameter
	var translationIDs []int
	translationIDsParam := c.Param("translationIDs")
	if translationIDsParam != "" {
		translationIDStrings := strings.Split(translationIDsParam, ",")
		for _, idStr := range translationIDStrings {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "Invalid TranslationIDs",
				})
			}
			translationIDs = append(translationIDs, id)
		}
	}

	collectionsTranslations, err := t.translatorRepository.GetCollectionTranslations(c.Request().Context(), collectionID, translationIDs, userID)
	if err != nil {
		t.logger.Error("failed to get collection's translations", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get collection's translations"})
	}
	return c.JSON(http.StatusOK, collectionsTranslations)
}

func (t TranslatorServer) DeleteCollectionsTranslations(c echo.Context) error {
	collectionIDParam := c.Param("collectionID")
	collectionID, err := strconv.Atoi(collectionIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid CollectionID"})
	}
	var translationIDs []int
	params := c.QueryParams()
	translationIDsParams, ok := params["translationIds"]
	if ok {
		for _, idStr := range translationIDsParams {
			id, err := strconv.Atoi(idStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid TranslationIDs"})
			}
			translationIDs = append(translationIDs, id)
		}
	}

	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}

	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	err = t.translatorRepository.DeleteCollectionTranslations(c.Request().Context(), translationIDs, collectionID, userID)
	if err != nil {
		t.logger.Error("failed to delete collection's translations", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to delete collection's translations"})
	}
	return c.NoContent(http.StatusNoContent)
}

func (t TranslatorServer) ExportCollectionsTranslations(c echo.Context) error {
	collectionIDParam := c.Param("collectionID")
	collectionID, err := strconv.Atoi(collectionIDParam)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid CollectionID",
		})
	}
	product := c.QueryParam("product")
	if product == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "product is not specified"})
	}
	sub, failed, status := t.GetSubFromToken(c)
	if failed {
		return status
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		t.logger.Error("failed to convert sub string to userID int", slog.Any("err", err.Error()))
		return c.JSON(http.StatusBadRequest, map[string]string{"message": "invalid userID"})
	}
	collectionsTranslations, err := t.translatorRepository.GetCollectionTranslations(c.Request().Context(), collectionID, []int{}, userID)
	if err != nil {
		t.logger.Error("failed to get collection's translations", slog.Any("err", err.Error()))
		return c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get collection's translations"})
	}
	var translations []domain.Translation
	for _, collectionTranslation := range collectionsTranslations {
		translations = append(translations, collectionTranslation.Translation)
	}
	quizletString := domain.ConvertTranslationToQuizletString(translations)

	filename := fmt.Sprintf("flesh-cards-%s-%d.txt", product, collectionID)
	c.Response().Header().Set(echo.HeaderContentDisposition, "attachment; filename="+filename)
	c.Response().Header().Set(echo.HeaderContentType, "text/plain")

	return c.Blob(http.StatusOK, "text/plain", []byte(quizletString))
}

func (t TranslatorServer) GetSubFromToken(c echo.Context) (string, bool, error) {
	auth := c.Request().Header.Get("Authorization")
	if auth == "" {
		return "", true, c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
	}

	token := strings.TrimSpace(strings.Replace(auth, "Bearer", "", 1))
	if token == "" {
		return "", true, c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
	}
	sub, err := t.jwtService.GetSubFromAccessToken(token)
	if err != nil {
		t.logger.Error("failed to get sub from access token", slog.Any("err", err.Error()))
		return "", true, c.JSON(http.StatusInternalServerError, map[string]string{"message": "failed to get users collections"})
	}
	return sub, false, nil
}
