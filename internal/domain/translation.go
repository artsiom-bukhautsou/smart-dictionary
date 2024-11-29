package domain

import (
	"fmt"
	"strings"
)

type Translation struct {
	ID                    int      `json:"id"`
	OriginalLexicalItem   string   `json:"originalLexicalItem"`
	OriginalMeaning       string   `json:"originalMeaning"`
	OriginalExamples      []string `json:"originalExamples"`
	TranslatedFrom        string   `json:"translatedFrom"`
	TranslatedTo          string   `json:"translatedTo"`
	TranslatedLexicalItem string   `json:"translatedLexicalItem"`
	TranslatedMeaning     string   `json:"translatedMeaning"`
	TranslatedExamples    []string `json:"translatedExamples"`
}

func IsTranslationNilOrEmpty(t *Translation) bool {
	if t == nil {
		return false
	}
	if t.OriginalLexicalItem == "" {
		return true
	}
	if t.OriginalMeaning == "" {
		return true
	}
	if len(t.OriginalExamples) == 0 {
		return true
	}
	if t.TranslatedLexicalItem == "" {
		return true
	}
	if t.TranslatedMeaning == "" {
		return true
	}
	if len(t.TranslatedExamples) == 0 {
		return true
	}
	return false
}

func ConvertTranslationToQuizletString(ts []Translation) string {
	var result strings.Builder
	for _, t := range ts {
		result.WriteString(t.OriginalLexicalItem)
		result.WriteString(";originalMeaning: " + t.OriginalMeaning + "\n")
		result.WriteString("originalExamples:\n")
		for i, example := range t.OriginalExamples {
			result.WriteString(fmt.Sprintf("%d) %s\n", i+1, example))
		}
		result.WriteString("translatedLexicalItem: " + t.TranslatedLexicalItem + "\n")
		result.WriteString("translatedMeaning: " + t.TranslatedMeaning + "\n")
		result.WriteString("translatedExamples:\n")
		for i, example := range t.TranslatedExamples {
			result.WriteString(fmt.Sprintf("%d) %s\n", i+1, example))
		}
		result.WriteString("\n\n")
	}
	return result.String()
}
