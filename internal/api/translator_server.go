package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"
)

type TranslatorServer struct {
	logger               slog.Logger
	chatGPTAPIURL        string
	apiKey               string
	translatorRepository domain.TranslatorRepository
}

func NewTranslatorServer(translatorRepository domain.TranslatorRepository, logger slog.Logger, chatGPTAPIURL string, apiKey string) *TranslatorServer {
	return &TranslatorServer{
		translatorRepository: translatorRepository,
		logger:               logger,
		chatGPTAPIURL:        chatGPTAPIURL,
		apiKey:               apiKey,
	}
}

func (t TranslatorServer) Translate(c echo.Context) error {
	var req domain.RequestMessage
	err := c.Bind(&req)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	t.logger.Info("received request", slog.String("word", req.Word))
	translation, err := t.translatorRepository.GetWordTranslation(c.Request().Context(), req.Word)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if translation != nil {
		return c.JSON(http.StatusOK, translation)
	}
	message, err := t.callChatGPTAPI("Translate the word, provide response in the following json format: word(string), meaning (string), examples (string array size 2), russianTranslation (string), meaningRussian (string) examplesRussian (string array size 2). Word to translate:" + req.Word)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	err = t.translatorRepository.AddWordTranslation(c.Request().Context(), *message)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, message)
}

func (t TranslatorServer) DownloadTranslations(c echo.Context) error {
	// Retrieve all messages
	messages, err := t.translatorRepository.GetAllWordTranslations(c.Request().Context())
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to retrieve messages")
	}

	// Create a CSV writer
	c.Response().Header().Set("Content-Type", "text/txt")
	c.Response().Header().Set("Content-Disposition", "attachment;filename=messages.txt")

	var buffer bytes.Buffer

	// Write messages to CSV
	for _, translation := range messages {
		// Format line for in-memory storage
		line := fmt.Sprintf("%s\n;*meaning*: %s\n*examples*: %s\n*russian*: %s\n*russian meaning*: %s\n*examples*: %s\n\n",
			translation.Word,
			translation.Meaning,
			strings.Join(translation.Examples, ". "),
			translation.RussianTranslation,
			translation.MeaningRussian,
			strings.Join(translation.ExamplesRussian, ". "),
		)

		// Write line to the in-memory buffer
		if _, err := buffer.WriteString(line); err != nil {
			return err
		}
	}
	return c.String(http.StatusOK, buffer.String())
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
