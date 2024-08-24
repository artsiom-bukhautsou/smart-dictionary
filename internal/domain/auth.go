package domain

type AuthCredentials struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	RefreshToken string `json:"refresh_token"`
}
