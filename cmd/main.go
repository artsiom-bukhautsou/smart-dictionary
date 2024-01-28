package main

import (
	"context"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/api"
	"github.com/bukhavtsov/artems-dictionary/internal/repository"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log/slog"
	"os"
)

var (
	chatGPTAPIURL    = os.Getenv("CHAT_GPT_API_URL")
	apiKey           = os.Getenv("OPEN_AI_API_KEY")
	PostgresUserName = os.Getenv("POSTGRES_USERNAME")
	PostgresPassword = os.Getenv("POSTGRES_PASSWORD")
	PostgresPort     = os.Getenv("POSTGRES_PORT")
	PostgresHost     = os.Getenv("POSTGRES_HOST")
	PostgresDBName   = os.Getenv("POSTGRES_DBNAME")
	Username         = os.Getenv("USER_NAME")
	Password         = os.Getenv("PASSWORD")
)

func main() {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept},
	}))
	e.Use(middleware.BasicAuth(basicAuth))
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	connString := "postgres://" + PostgresUserName + ":" + PostgresPassword + "@" + PostgresHost + ":" + PostgresPort + "/" + PostgresDBName
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		fmt.Println("Unable to connect to the database:", err)
		return
	}
	defer conn.Close(context.Background())

	translationRepository := repository.NewTranslationRepository(conn)
	translatorServer := api.NewTranslatorServer(translationRepository, *logger, chatGPTAPIURL, apiKey)
	e.POST("/translations", translatorServer.Translate)
	e.GET("/translations/download", translatorServer.DownloadTranslations)
	slog.Error("server has failed", slog.Any("err", e.Start(":8080")))
}

func basicAuth(username, password string, c echo.Context) (bool, error) {
	if username == Username && password == Password {
		return true, nil
	}
	return false, nil
}
