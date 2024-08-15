package domain

import "context"

type WordTranslation struct {
	Word               string   `json:"word"`
	Meaning            string   `json:"meaning"`
	Examples           []string `json:"examples"`
	RussianTranslation string   `json:"russianTranslation"`
	MeaningRussian     string   `json:"meaningRussian"`
	ExamplesRussian    []string `json:"examplesRussian"`
}

type TranslatorRepository interface {
	AddWordTranslation(ctx context.Context, message WordTranslation) error
	GetAllWordTranslations(ctx context.Context) ([]WordTranslation, error)
	GetWordTranslation(ctx context.Context, word string) (*WordTranslation, error)
}

// CardRepository defines the interface for creating a card
type CardRepository interface {
	CreateCard(deckID, content string) error
}
