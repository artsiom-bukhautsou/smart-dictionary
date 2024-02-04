package repository

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type CardRequest struct {
	Content string `json:"content"`
	DeckID  string `json:"deck-id"`
	// TODO: ADD images support later
}

// MochiCardRepository implements the CardRepository interface
type MochiCardRepository struct {
	BaseURL string
	Token   string
	DeckID  string
}

// NewMochiCardRepository creates a new instance of MochiCardRepository with the provided configuration
func NewMochiCardRepository(baseURL, token, deckID string) *MochiCardRepository {
	return &MochiCardRepository{
		BaseURL: baseURL,
		Token:   token,
		DeckID:  deckID,
	}
}

// CreateCard sends a request to create a card using the provided data
func (m MochiCardRepository) CreateCard(content string) error {
	url := m.BaseURL + "/api/cards/"
	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": m.generateBasicAuthHeader(),
	}
	req := CardRequest{Content: content, DeckID: m.DeckID}

	requestBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %v", err)
	}

	request, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, response body: %s", response.StatusCode, responseBody)
	}

	return nil
}

// generateBasicAuthHeader generates the Basic Auth header using the token
func (m MochiCardRepository) generateBasicAuthHeader() string {
	auth := m.Token + ":"
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}
