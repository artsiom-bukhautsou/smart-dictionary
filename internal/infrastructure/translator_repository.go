package infrastructure

import (
	"context"
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
func (t *translationRepository) AddTranslation(ctx context.Context, translation domain.Translation, translatedFrom, translatedTo string) error {
	_, err := t.conn.Exec(ctx, `
		INSERT INTO translations(lexical_item, meaning, examples, translated_from, translated_to, translated_lexical_item, translated_meaning, translated_examples)
		VALUES($1, $2, $3, $4, $5, $6, $7, $8)
	`,
		translation.OriginalLexicalItem,
		translation.OriginalMeaning,
		translation.OriginalExamples,
		translatedFrom,
		translatedTo,
		translation.TranslatedLexicalItem,
		translation.TranslatedMeaning,
		translation.TranslatedExamples,
	)

	return err
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
