package infrastructure

import (
	"context"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/jackc/pgx/v5"
	"strings"
)

type translationRepository struct {
	conn *pgx.Conn
}

// NewTranslationRepository creates a new instance of translationRepository
func NewTranslationRepository(conn *pgx.Conn) *translationRepository {
	return &translationRepository{conn: conn}
}

// AddWordTranslation inserts a message into the database
func (t *translationRepository) AddWordTranslation(ctx context.Context, translation domain.WordTranslation) error {
	_, err := t.conn.Exec(ctx, `
		INSERT INTO translations(word, meaning, examples, russian_translation, meaning_russian, examples_russian)
		VALUES($1, $2, $3, $4, $5, $6)
	`, translation.Word, translation.Meaning, translation.Examples, translation.RussianTranslation, translation.MeaningRussian, translation.ExamplesRussian)

	return err
}

func (t *translationRepository) GetAllWordTranslations(ctx context.Context) ([]domain.WordTranslation, error) {
	var messages []domain.WordTranslation

	rows, err := t.conn.Query(ctx, `
		SELECT word, meaning, examples, russian_translation, meaning_russian, examples_russian
		FROM translations
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var translation domain.WordTranslation
		err := rows.Scan(&translation.Word, &translation.Meaning, &translation.Examples, &translation.RussianTranslation, &translation.MeaningRussian, &translation.ExamplesRussian)
		if err != nil {
			return nil, err
		}
		messages = append(messages, translation)
	}

	return messages, nil
}

func (t *translationRepository) GetWordTranslation(ctx context.Context, word string) (*domain.WordTranslation, error) {
	word = strings.ToLower(word)
	rows, err := t.conn.Query(ctx, `
		SELECT word, meaning, examples, russian_translation, meaning_russian, examples_russian
		FROM translations
		WHERE word = $1
		LIMIT 1;
	`, word)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var translation domain.WordTranslation
		err := rows.Scan(&translation.Word, &translation.Meaning, &translation.Examples, &translation.RussianTranslation, &translation.MeaningRussian, &translation.ExamplesRussian)
		if err != nil {
			return nil, err
		}
		return &translation, nil
	}

	return nil, nil
}
