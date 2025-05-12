package domain

type TTSRequest struct {
	Text     string `json:"text"`
	Language string `json:"language"`
}

const (
	LanguageEnglish = "english"
	LanguagePolish  = "polish"
	LanguageRussian = "russian"
)
