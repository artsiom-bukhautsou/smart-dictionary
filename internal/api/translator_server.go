package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/service"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"
)

type TranslatorServer struct {
	logger               slog.Logger
	authService          service.AuthService
	chatGPTAPIURL        string
	apiKey               string
	translatorRepository domain.TranslatorRepository
	cardsRepository      domain.CardRepository
}

func NewTranslatorServer(
	authService service.AuthService,
	translatorRepository domain.TranslatorRepository,
	cardsRepository domain.CardRepository,
	logger slog.Logger, chatGPTAPIURL string,
	apiKey string) *TranslatorServer {
	return &TranslatorServer{
		authService:          authService,
		translatorRepository: translatorRepository,
		cardsRepository:      cardsRepository,
		logger:               logger,
		chatGPTAPIURL:        chatGPTAPIURL,
		apiKey:               apiKey,
	}
}

func (t TranslatorServer) SignIn(c echo.Context) error {
	var creds domain.AuthCredentials
	err := c.Bind(&creds)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	areCredsValid, err := t.authService.BasicAuth(creds.Username, creds.Password, c)
	if !areCredsValid {
		return c.String(http.StatusForbidden, "Received invalid credentials")
	}
	if err != nil {
		return c.String(http.StatusUnauthorized, err.Error())
	}
	return c.String(http.StatusOK, "successfully authenticated")
}

func (t TranslatorServer) SignUp(c echo.Context) error {
	var creds domain.AuthCredentials
	err := c.Bind(&creds)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	err = t.authService.CreateUser(creds.Username, creds.Password)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.String(http.StatusOK, "successfully sign up")
}

func (t TranslatorServer) Translate(c echo.Context) error {
	deckID := c.Request().Header.Get("Deck-Id")
	if deckID == "" {
		return c.String(http.StatusBadRequest, "Deck-Id wasn't provided")
	}
	var req domain.RequestMessage
	err := c.Bind(&req)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	req.Word = strings.ToLower(req.Word)
	translation, err := t.translatorRepository.GetWordTranslation(c.Request().Context(), req.Word)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if translation != nil {
		go func() {
			err = t.cardsRepository.CreateCard(deckID, wordTranslationToMarkdown(*translation))
			if err != nil {
				t.logger.Error(err.Error())
			}
		}()
		return c.JSON(http.StatusOK, translation)
	}
	message, err := t.callChatGPTAPI("Translate the word, provide response in the following json format: word(string), meaning (string), examples (string array size 2), russianTranslation (string), meaningRussian (string) examplesRussian (string array size 2). Word to translate:" + req.Word)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	go func() {
		err = t.translatorRepository.AddWordTranslation(context.Background(), *message)
		if err != nil {
			t.logger.Error(err.Error())
		}
		err = t.cardsRepository.CreateCard(deckID, wordTranslationToMarkdown(*message))
		if err != nil {
			if err != nil {
				t.logger.Error(err.Error())
			}
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

func wordTranslationToMarkdown(wt domain.WordTranslation) string {
	var markdownBuilder strings.Builder

	// Word and horizontal line
	markdownBuilder.WriteString(fmt.Sprintf("%s\n\n---\n\n", wt.Word))

	// Bold formatting for headers
	bold := func(s string) string {
		return fmt.Sprintf("**%s**", s)
	}

	// Heading with the word
	markdownBuilder.WriteString(fmt.Sprintf("%s\n\n", bold(wt.Word)))

	// Meaning section
	markdownBuilder.WriteString(fmt.Sprintf("%s\n", bold("Meaning")))
	markdownBuilder.WriteString(fmt.Sprintf("%s\n\n", wt.Meaning))

	// Examples section
	if len(wt.Examples) > 0 {
		markdownBuilder.WriteString(fmt.Sprintf("%s\n", bold("Examples")))
		for _, example := range wt.Examples {
			markdownBuilder.WriteString(fmt.Sprintf("- %s\n", example))
		}
		markdownBuilder.WriteString("\n")
	}

	// Russian Translation section
	markdownBuilder.WriteString(fmt.Sprintf("%s\n", bold("Russian Translation")))
	markdownBuilder.WriteString(fmt.Sprintf("%s\n\n", wt.RussianTranslation))

	// Meaning in Russian section
	markdownBuilder.WriteString(fmt.Sprintf("%s\n", bold("Meaning in Russian")))
	markdownBuilder.WriteString(fmt.Sprintf("%s\n\n", wt.MeaningRussian))

	// Examples in Russian section
	if len(wt.ExamplesRussian) > 0 {
		markdownBuilder.WriteString(fmt.Sprintf("%s\n", bold("Examples in Russian")))
		for _, exampleRussian := range wt.ExamplesRussian {
			markdownBuilder.WriteString(fmt.Sprintf("- %s\n", exampleRussian))
		}
		markdownBuilder.WriteString("\n")
	}

	return markdownBuilder.String()
}
