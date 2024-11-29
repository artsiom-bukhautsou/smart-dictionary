package domain

type Collection struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID int    `json:"userId"`
}

// TODO: think about making it easier collection []Translations
type CollectionTranslation struct {
	ID          int         `json:"id"`
	Collection  Collection  `json:"collection"`
	Translation Translation `json:"translation"`
}
