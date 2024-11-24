package domain

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
