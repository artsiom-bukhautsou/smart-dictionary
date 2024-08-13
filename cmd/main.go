package main

import (
	"context"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/api"
	"github.com/bukhavtsov/artems-dictionary/internal/repository"
	"github.com/bukhavtsov/artems-dictionary/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log/slog"
	"os"
)

var (
	chatGPTAPIURL     = os.Getenv("CHAT_GPT_API_URL")
	apiKey            = os.Getenv("OPEN_AI_API_KEY")
	PostgresUserName  = os.Getenv("POSTGRES_USERNAME")
	PostgresPassword  = os.Getenv("POSTGRES_PASSWORD")
	PostgresPort      = os.Getenv("POSTGRES_PORT")
	PostgresHost      = os.Getenv("POSTGRES_HOST")
	PostgresDBName    = os.Getenv("POSTGRES_DBNAME")
	MochiCardsBaseURL = os.Getenv("MOCHI_CARDS_BASE_URL")
	MochiToken        = os.Getenv("MOCHI_TOKEN")
)

func main() {
	e := echo.New()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	connString := "postgres://" + PostgresUserName + ":" + PostgresPassword + "@" + PostgresHost + ":" + PostgresPort + "/" + PostgresDBName
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		fmt.Println("Unable to connect to the database:", err)
		return
	}
	defer conn.Close(context.Background())

	userRepository := repository.NewUserRepository(conn)
	authService := service.NewAuthService(userRepository)
	flashCardsRepository := repository.NewMochiCardRepository(MochiCardsBaseURL, MochiToken)
	translationRepository := repository.NewTranslationRepository(conn)
	translatorServer := api.NewTranslatorServer(*authService, translationRepository, flashCardsRepository, *logger, chatGPTAPIURL, apiKey)

	apiGroup := e.Group("/api")
	apiGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Deck-Id"},
			AllowCredentials: true,
		}),
		middleware.BasicAuth(authService.BasicAuth),
	)
	apiGroup.POST("/translations", translatorServer.Translate)

	e.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Deck-Id"},
			AllowCredentials: true,
		}),
	)
	e.POST("/auth/signin", translatorServer.SignIn)
	e.POST("/auth/signup", translatorServer.SignUp)
	slog.Error("server has failed", slog.Any("err", e.Start(":8080")))
}
