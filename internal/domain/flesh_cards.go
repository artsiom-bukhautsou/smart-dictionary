package domain

type RatingType int

const (
	RatingAgain    RatingType = 1
	RatingWeek     RatingType = 2
	RatingMonth    RatingType = 3
	RatingTwoMonth RatingType = 4
)

type RateTranslationInput struct {
	Rating RatingType `json:"rating"`
}