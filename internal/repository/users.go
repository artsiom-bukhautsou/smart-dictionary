package repository

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type UserRepository struct {
	conn *pgx.Conn
}

func NewUserRepository(conn *pgx.Conn) *UserRepository {
	return &UserRepository{conn: conn}
}

func (ur *UserRepository) GetUser(userName string, password string) error {
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

func (ur *UserRepository) CreateUser(username string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	_, err = ur.conn.Exec(context.Background(), "INSERT INTO users (user_name, password) VALUES ($1, $2)", username, string(hashedPassword))
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}
