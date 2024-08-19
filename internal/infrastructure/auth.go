package infrastructure

import (
	"context"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthRepository struct {
	conn *pgx.Conn
}

func NewAuthRepository(conn *pgx.Conn) *AuthRepository {
	return &AuthRepository{conn: conn}
}

func (ur *AuthRepository) SignIn(userName string, password string) error {
	var dbPassword string
	err := ur.conn.QueryRow(context.Background(), "SELECT password FROM users WHERE user_name = $1", userName).Scan(&dbPassword)
	if err != nil {
		return err
	}
	err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	return nil
}

func (ur *AuthRepository) UpdateRefreshToken(username, refreshToken string) error {
	result, err := ur.conn.Exec(context.Background(), "UPDATE users SET refresh_token = $1 WHERE username = $2", refreshToken, username)
	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows updated, user not found")
	}
	return nil
}

func (ur *AuthRepository) IsUsernameExist(userName string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE user_name = $1)"
	err := ur.conn.QueryRow(context.Background(), query, userName).Scan(&exists)

	if err != nil {
		return false, fmt.Errorf("could not check user existence: %w", err)
	}
	return exists, nil
}

func (ur *AuthRepository) SignUp(creds domain.AuthCredentials) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	_, err = ur.conn.Exec(
		context.Background(),
		"INSERT INTO users (user_name, password, refresh_token) VALUES ($1, $2, $3)",
		creds.Username,
		string(hashedPassword),
		creds.RefreshToken,
	)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
