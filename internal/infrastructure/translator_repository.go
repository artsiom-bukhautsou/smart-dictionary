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

func (t *translationRepository) CreateCollection(ctx context.Context, userID int, collectionName string) (int, error) {
	_, err := t.conn.Exec(
		ctx,
		"INSERT INTO collections (user_id, collection_name) VALUES ($1, $2)",
		userID,
		collectionName,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create a collection: %w", err)
	}
	return 0, nil
}

func (t *translationRepository) SaveToCollectionLexicalItem(ctx context.Context, collectionID, translationID int) (int, error) {
	_, err := t.conn.Exec(
		ctx,
		"INSERT INTO collection_translations (collection_id, translation_id) VALUES ($1, $2)",
		collectionID,
		translationID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to assosiate translation with collection: %w", err)
	}
	return 0, nil
}

func (t *translationRepository) GetCollectionsByUserID(ctx context.Context, userID int) ([]domain.Collection, error) {
	var collections []domain.Collection

	query := "SELECT id, collection_name, user_id FROM collections WHERE user_id = $1"

	rows, err := t.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve collections for user_id %d: %w", userID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var collection domain.Collection
		if err := rows.Scan(&collection.ID, &collection.Name, &collection.UserID); err != nil {
			return nil, fmt.Errorf("failed to scan collection row: %w", err)
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

func (t *translationRepository) GetCollectionTranslations(ctx context.Context, collectionID int, translationIDs []int, userID int) ([]domain.CollectionTranslation, error) {
	var translations []domain.CollectionTranslation

	query := `
		SELECT 
		    ct.id AS collection_translation_id, 
		    ct.collection_id, 
		    ct.translation_id, 
		    c.collection_name, 
		    c.user_id, 
		    t.lexical_item, 
		    t.meaning, 
		    t.examples, 
		    t.translated_from, 
		    t.translated_to, 
		    t.translated_lexical_item, 
		    t.translated_meaning, 
		    t.translated_examples
		FROM 
		    collection_translations ct
		JOIN 
		    collections c ON ct.collection_id = c.id
		JOIN 
		    translations t ON ct.translation_id = t.id
		WHERE 
		    c.id = $1
		AND
			c.user_id = $2
	`
	args := []interface{}{collectionID, userID}

	if len(translationIDs) > 0 {
		query += " AND t.id = ANY($3)"
		args = append(args, translationIDs)
	}

	rows, err := t.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve collection translations for collection_id %d: %w", collectionID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var ct domain.CollectionTranslation
		var collection domain.Collection
		var translation domain.Translation

		if err := rows.Scan(
			&ct.ID,
			&collection.ID,
			&translation.ID,
			&collection.Name,
			&collection.UserID,
			&translation.OriginalLexicalItem,
			&translation.OriginalMeaning,
			&translation.OriginalExamples,
			&translation.TranslatedFrom,
			&translation.TranslatedTo,
			&translation.TranslatedLexicalItem,
			&translation.TranslatedMeaning,
			&translation.TranslatedExamples,
		); err != nil {
			return nil, fmt.Errorf("failed to scan collection_translation row: %w", err)
		}

		ct.Collection = collection
		ct.Translation = translation

		translations = append(translations, ct)
	}

	return translations, nil
}

func (t *translationRepository) DeleteCollectionTranslations(ctx context.Context, translationIDs []int, collectionID int, userID int) error {
	query := `
		DELETE FROM public.collection_translations
		WHERE translation_id = ANY($1)
		  AND collection_id = (
			SELECT id FROM public.collections
			WHERE id = $2 AND user_id = $3
		  );
	`
	_, err := t.conn.Exec(ctx, query, translationIDs, collectionID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete translations: %w", err)
	}
	return nil
}