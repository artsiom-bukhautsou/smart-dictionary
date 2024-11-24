package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type translationRepository struct {
	conn *pgxpool.Pool
}

// NewTranslationRepository creates a new instance of translationRepository
func NewTranslationRepository(conn *pgxpool.Pool) *translationRepository {
	return &translationRepository{conn: conn}
}

// AddTranslation inserts a translation into the database
func (t *translationRepository) AddTranslation(ctx context.Context, translation domain.Translation, translatedFrom, translatedTo string) (int, error) {
	var id int
	err := t.conn.QueryRow(ctx, `
		INSERT INTO translations(lexical_item, meaning, examples, translated_from, translated_to, translated_lexical_item, translated_meaning, translated_examples)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`,
		translation.OriginalLexicalItem,
		translation.OriginalMeaning,
		translation.OriginalExamples,
		translatedFrom,
		translatedTo,
		translation.TranslatedLexicalItem,
		translation.TranslatedMeaning,
		translation.TranslatedExamples,
	).Scan(&id)

	if err != nil {
		return 0, err
	}
	return id, nil
}

// GetAllTranslations retrieves all translations from the database
func (t *translationRepository) GetAllTranslations(ctx context.Context) ([]domain.Translation, error) {
	var translations []domain.Translation

	rows, err := t.conn.Query(ctx, `
		SELECT lexical_item, meaning, examples, translated_from, translated_to, translated_lexical_item, translated_meaning, translated_examples
		FROM translations
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var translation domain.Translation
		err := rows.Scan(
			&translation.OriginalLexicalItem,
			&translation.OriginalMeaning,
			&translation.OriginalExamples,
			&translation.TranslatedFrom,
			&translation.TranslatedTo,
			&translation.TranslatedLexicalItem,
			&translation.TranslatedMeaning,
			&translation.TranslatedExamples,
		)
		if err != nil {
			return nil, err
		}
		translations = append(translations, translation)
	}

	return translations, nil
}

// GetTranslation retrieves a translation based on the lexical item and the languages
func (t *translationRepository) GetTranslation(ctx context.Context, lexicalItem, translateFrom, translateTo string) (*domain.Translation, error) {
	lexicalItem = strings.ToLower(lexicalItem)
	rows, err := t.conn.Query(ctx, `
		SELECT lexical_item, meaning, examples, translated_from, translated_to, translated_lexical_item, translated_meaning, translated_examples
		FROM translations
		WHERE lexical_item = $1 AND translated_from = $2 AND translated_to = $3
		LIMIT 1;
	`, lexicalItem, translateFrom, translateTo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var translation domain.Translation
		err := rows.Scan(
			&translation.OriginalLexicalItem,
			&translation.OriginalMeaning,
			&translation.OriginalExamples,
			&translation.TranslatedFrom,
			&translation.TranslatedTo,
			&translation.TranslatedLexicalItem,
			&translation.TranslatedMeaning,
			&translation.TranslatedExamples,
		)
		if err != nil {
			return nil, err
		}
		return &translation, nil
	}

	return nil, nil
}

func (t *translationRepository) CreateDeck(ctx context.Context, userID int, deckName string) (int, error) {
	_, err := t.conn.Exec(
		ctx,
		"INSERT INTO decks (user_id, deck_name) VALUES ($1, $2)",
		userID,
		deckName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create a deck: %w", err)
	}
	return 0, nil
}

func (t *translationRepository) SaveToDeckLexicalItem(ctx context.Context, deckID, translationID int) (int, error) {
	_, err := t.conn.Exec(
		ctx,
		"INSERT INTO deck_translations (deck_id, translation_id) VALUES ($1, $2)",
		deckID,
		translationID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to assosiate translation with deck: %w", err)
	}
	return 0, nil
}

func (t *translationRepository) GetDecksByUserID(ctx context.Context, userID int) ([]domain.Deck, error) {
	var decks []domain.Deck

	query := "SELECT id, deck_name, user_id FROM decks WHERE user_id = $1"

	rows, err := t.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve decks for user_id %d: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var deck domain.Deck
		if err := rows.Scan(&deck.ID, &deck.Name, &deck.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan deck row: %w", err)
		}
		decks = append(decks, deck)
	}

	return decks, nil
}

func (t *translationRepository) GetDeckTranslations(ctx context.Context, deckID int, translationIDs []int, userID int) ([]domain.DeckTranslation, error) {
	var translations []domain.DeckTranslation

	query := `
		SELECT 
		    dt.id AS deck_translation_id, 
		    dt.deck_id, 
		    dt.translation_id, 
		    d.deck_name, 
		    d.user_id, 
		    t.lexical_item, 
		    t.meaning, 
		    t.examples, 
		    t.translated_from, 
		    t.translated_to, 
		    t.translated_lexical_item, 
		    t.translated_meaning, 
		    t.translated_examples
		FROM 
		    deck_translations dt
		JOIN 
		    decks d ON dt.deck_id = d.id
		JOIN 
		    translations t ON dt.translation_id = t.id
		WHERE 
		    d.id = $1
		AND
			d.user_id = $2
	`
	args := []interface{}{deckID, userID}

	if len(translationIDs) > 0 {
		query += " AND t.id = ANY($3)"
		args = append(args, translationIDs)
	}

	rows, err := t.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve deck translations for deck_id %d: %w", deckID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var dt domain.DeckTranslation
		var deck domain.Deck
		var translation domain.Translation

		if err := rows.Scan(
			&dt.ID,
			&deck.ID,
			&translation.ID,
			&deck.Name,
			&deck.UserID,
			&translation.OriginalLexicalItem,
			&translation.OriginalMeaning,
			&translation.OriginalExamples,
			&translation.TranslatedFrom,
			&translation.TranslatedTo,
			&translation.TranslatedLexicalItem,
			&translation.TranslatedMeaning,
			&translation.TranslatedExamples,
		); err != nil {
			return nil, fmt.Errorf("failed to scan deck_translation row: %w", err)
		}

		dt.Deck = deck
		dt.Translation = translation

		translations = append(translations, dt)
	}

	return translations, nil
}
