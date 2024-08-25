package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type AuthRepository struct {
	conn *pgxpool.Pool
}

func NewAuthRepository(conn *pgxpool.Pool) *AuthRepository {
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
	result, err := ur.conn.Exec(context.Background(), "UPDATE users SET refresh_token = $1 WHERE user_name = $2", refreshToken, username)
	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows updated, user not found")
	}
	return nil
}

func (ur *AuthRepository) IsUsernameExist(userName string) (bool, error) {
	var username string
	query := "SELECT user_name FROM users WHERE user_name = $1"
	err := ur.conn.QueryRow(context.Background(), query, userName).Scan(&username)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not check user existence: %w", err)
	}
	exists := username != ""
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

func (ur *AuthRepository) RemoveUser(username string) error {
	result, err := ur.conn.Exec(
		context.Background(), "DELETE FROM users WHERE user_name = $1", username)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with username: %s", username)
	}
	return nil
}
