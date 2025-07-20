package domain

import "time"

type Collection struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	UserID int    `json:"userId"`
}

type CollectionTranslation struct {
	ID          int         `json:"id"`
	Collection  Collection  `json:"collection"`
	Translation Translation `json:"translation"`
	Due         *time.Time  `json:"due,omitempty"`
}
