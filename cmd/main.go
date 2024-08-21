package main

import (
	"context"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	"github.com/bukhavtsov/artems-dictionary/internal/server"
	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log/slog"
	"net/http"
	"os"
	"strings"
)

var (
	chatGPTAPIURL    = os.Getenv("CHAT_GPT_API_URL")
	apiKey           = os.Getenv("OPEN_AI_API_KEY")
	PostgresUserName = os.Getenv("POSTGRES_USERNAME")
	PostgresPassword = os.Getenv("POSTGRES_PASSWORD")
	PostgresPort     = os.Getenv("POSTGRES_PORT")
	PostgresHost     = os.Getenv("POSTGRES_HOST")
	PostgresDBName   = os.Getenv("POSTGRES_DBNAME")
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

	authRepository := infrastructure.NewAuthRepository(conn)
	jwtAuth := usecase.NewJWTAuth(*authRepository)
	authService := usecase.NewAuthService(*authRepository, *jwtAuth)
	translationRepository := infrastructure.NewTranslationRepository(conn)
	translatorServer := server.NewTranslatorServer(*authService, *jwtAuth, translationRepository, *logger, chatGPTAPIURL, apiKey)

	apiGroup := e.Group("/server")
	apiGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Deck-Id"},
			AllowCredentials: true,
		}),
		ValidateAccessToken(*jwtAuth),
	)
	apiGroup.POST("/translations", translatorServer.Translate)

	authGroup := e.Group("/auth")
	authGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
			AllowCredentials: true,
		}),
	)
	authGroup.POST("/signin", translatorServer.SignIn)
	authGroup.POST("/signup", translatorServer.SignUp)
	authGroup.POST("/refresh", translatorServer.RefreshRefreshToken)
	slog.Error("server has failed", slog.Any("err", e.Start(":8080")))
}

func ValidateAccessToken(jwtAuth usecase.JWTAuth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			auth := req.Header.Get("Authorization")
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
			}
			// Token usually comes as "Bearer <token>", so we split to get the actual token part
			token := strings.TrimSpace(strings.Replace(auth, "Bearer", "", 1))
			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "missing or malformed token"})
			}
			// Validate the token using the JWTAuth use case
			isValid, err := jwtAuth.IsAccessTokenValid(token)
			if !isValid || err != nil {
				if err != nil {
					fmt.Printf("failed to validate token: %v", err)
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid or expired token"})
			}
			return next(c)
		}
	}
}
