package domain

type Deck struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID int    `json:"userId"`
}

type DeckTranslation struct {
	ID          int         `json:"id"`
	Deck        Deck        `json:"deck"`
	Translation Translation `json:"translation"`
}
