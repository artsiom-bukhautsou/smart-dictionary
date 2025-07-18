package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

type Card struct {
	ID       int       `json:"id"`
	Question string    `json:"question"`
	Answer   string    `json:"answer"`
	Due      time.Time `json:"due"`
}

type Deck struct {
	Cards []Card `json:"cards"`
}

const dataFile = "flashcards.json"

type FleshCardsServer struct {
	// Add fields here if you want server-level config, db, etc.
}

// --- Constructor ---
func NewFleshCardsServer() *FleshCardsServer {
	return &FleshCardsServer{}
}

// --- Persistence helpers (now methods) ---
func (s *FleshCardsServer) loadDeck() Deck {
	f, err := os.Open(dataFile)
	if err != nil {
		return Deck{}
	}
	defer f.Close()
	var deck Deck
	json.NewDecoder(f).Decode(&deck)
	return deck
}

func (s *FleshCardsServer) saveDeck(deck Deck) {
	f, err := os.Create(dataFile)
	if err != nil {
		fmt.Println("Failed to save:", err)
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.Encode(deck)
}

// --- Helper: Find card index by ID ---
func findCardIndexByID(cards []Card, id int) int {
	for i, c := range cards {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// --- HTTP Handlers (now methods) ---

func (s *FleshCardsServer) AddCardHandler(c echo.Context) error {
	var card Card
	if err := c.Bind(&card); err != nil {
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	deck := s.loadDeck()
	// Generate new ID: max existing ID + 1
	maxID := 0
	for _, c := range deck.Cards {
		if c.ID > maxID {
			maxID = c.ID
		}
	}
	card.ID = maxID + 1
	card.Due = time.Now()
	deck.Cards = append(deck.Cards, card)
	s.saveDeck(deck)
	return c.JSON(http.StatusCreated, card)
}

func (s *FleshCardsServer) ListCardsHandler(c echo.Context) error {
	deck := s.loadDeck()
	return c.JSON(http.StatusOK, deck.Cards)
}

func (s *FleshCardsServer) DeleteCardHandler(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid card id")
	}
	deck := s.loadDeck()
	idx := findCardIndexByID(deck.Cards, id)
	if idx == -1 {
		return c.String(http.StatusNotFound, "Card not found")
	}
	deck.Cards = append(deck.Cards[:idx], deck.Cards[idx+1:]...)
	s.saveDeck(deck)
	return c.NoContent(http.StatusNoContent)
}

func (s *FleshCardsServer) GetDueCardHandler(c echo.Context) error {
	deck := s.loadDeck()
	now := time.Now()
	dueIndexes := []int{}
	for i, card := range deck.Cards {
		if card.Due.Before(now) || card.Due.Equal(now) {
			dueIndexes = append(dueIndexes, i)
		}
	}
	if len(dueIndexes) == 0 {
		return c.NoContent(http.StatusNoContent)
	}
	idx := dueIndexes[rand.Intn(len(dueIndexes))]
	resp := map[string]interface{}{
		"id":   deck.Cards[idx].ID, // Return card ID (not index)
		"card": deck.Cards[idx],
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *FleshCardsServer) RateCardHandler(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.String(http.StatusBadRequest, "Invalid card id")
	}
	var input struct{ Rating int `json:"rating"` }
	if err := c.Bind(&input); err != nil {
		return c.String(http.StatusBadRequest, "Invalid input")
	}
	deck := s.loadDeck()
	idx := findCardIndexByID(deck.Cards, id)
	if idx == -1 {
		return c.String(http.StatusNotFound, "Card not found")
	}
	now := time.Now()
	switch input.Rating {
	case 1:
		deck.Cards[idx].Due = now // again
	case 2:
		deck.Cards[idx].Due = now.AddDate(0, 0, 7) // 7 days
	case 3:
		deck.Cards[idx].Due = now.AddDate(0, 1, 0) // 1 month
	case 4:
		deck.Cards[idx].Due = now.AddDate(0, 2, 0) // 2 months
	default:
		deck.Cards[idx].Due = now
	}
	s.saveDeck(deck)
	return c.JSON(http.StatusOK, deck.Cards[idx])
}
