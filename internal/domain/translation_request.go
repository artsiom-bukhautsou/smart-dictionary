package domain

type TranslationRequest struct {
	LexicalItem   string `json:"lexicalItem"`
	TranslateFrom string `json:"translateFrom"`
	TranslateTo   string `json:"translateTo"`
}
