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

func (ur *AuthRepository) SignIn(userName string, password string) (int, error) {
	var dbPassword string
	var userID int
	err := ur.conn.QueryRow(
		context.Background(),
		"SELECT id, password FROM users WHERE user_name = $1",
		userName,
	).Scan(&userID, &dbPassword)
	if err != nil {
		return 0, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(dbPassword), []byte(password))
	if err != nil {
		return 0, fmt.Errorf("authentication failed: %w", err)
	}
	return userID, nil
}

func (ur *AuthRepository) UpdateRefreshToken(userID int, refreshToken string) error {
	result, err := ur.conn.Exec(context.Background(), "UPDATE users SET refresh_token = $1 WHERE id = $2", refreshToken, userID)
	if err != nil {
		return fmt.Errorf("failed to update refresh token: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("no rows updated, user not found")
	}
	return nil
}

func (ur *AuthRepository) DoesUserIDExist(userID int) (bool, error) {
	var id int
	query := "SELECT id FROM users WHERE id = $1"
	err := ur.conn.QueryRow(context.Background(), query, id).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not check user_id existence: %w", err)
	}
	exists := id != 0
	return exists, nil
}

func (ur *AuthRepository) DoesUsernameExist(username string) (bool, error) {
	var usernameStr string
	query := "SELECT user_name FROM users WHERE user_name = $1"
	err := ur.conn.QueryRow(context.Background(), query, username).Scan(&usernameStr)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("could not check username existence: %w", err)
	}
	exists := usernameStr != ""
	return exists, nil
}

func (ur *AuthRepository) SignUp(creds domain.AuthCredentials) (int, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(creds.Password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("failed to hash password: %w", err)
	}
	var userID int
	err = ur.conn.QueryRow(
		context.Background(),
		"INSERT INTO users (user_name, password, refresh_token) VALUES ($1, $2, $3) RETURNING id",
		creds.Username,
		string(hashedPassword),
		creds.RefreshToken,
	).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %w", err)
	}
	return userID, nil
}

func (ur *AuthRepository) RemoveUser(id int) error {
	result, err := ur.conn.Exec(
		context.Background(), "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no user found with id: %d", id)
	}
	return nil
}
