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
	"log"
	"log/slog"
	"net/http"
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

	authRepository := infrastructure.NewAuthRepository(conn)
	jwtAuth := usecase.NewJWTAuth(*authRepository)
	authService := usecase.NewAuthService(*authRepository, *jwtAuth)
	flashCardsRepository := infrastructure.NewMochiCardRepository(MochiCardsBaseURL, MochiToken)
	translationRepository := infrastructure.NewTranslationRepository(conn)
	translatorServer := server.NewTranslatorServer(*authService, translationRepository, flashCardsRepository, *logger, chatGPTAPIURL, apiKey)

	apiGroup := e.Group("/server")
	apiGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     []string{"*"},
			AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "Deck-Id"},
			AllowCredentials: true,
		}),
		authMiddleware(*jwtAuth),
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
	slog.Error("server has failed", slog.Any("err", e.Start(":8080")))
}

func authMiddleware(jwtAuth usecase.JWTAuth) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			log.Println(req.RequestURI)

			signUpReq := req.RequestURI == "/api/v1/signup" && req.Method == http.MethodPost
			signInReq := req.RequestURI == "/api/v1/signin" && req.Method == http.MethodPost
			if !signUpReq && !signInReq {
				accessTokenCookie, err := req.Cookie("accessToken")
				if err != nil {
					if err == http.ErrNoCookie {
						log.Println("access token cookie not found")
						return c.JSON(http.StatusUnauthorized, map[string]string{"message": "access token not found"})
					}
					log.Println("error retrieving access token cookie:", err)
					return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
				}

				accessToken := accessTokenCookie.Value

				// Retrieve the refresh token from cookies
				refreshTokenCookie, err := req.Cookie("refresh_token")
				if err != nil {
					if err == http.ErrNoCookie {
						log.Println("refresh token cookie not found")
						return c.JSON(http.StatusUnauthorized, map[string]string{"message": "refresh token not found"})
					}
					log.Println("error retrieving refresh token cookie:", err)
					return c.JSON(http.StatusInternalServerError, map[string]string{"message": "internal server error"})
				}

				refreshToken := refreshTokenCookie.Value

				err = jwtAuth.Validate(&accessToken, &refreshToken)
				if err != nil {
					log.Println("invalid tokens:", err)
					return c.JSON(http.StatusUnauthorized, map[string]string{"message": "invalid tokens"})
				}

				c.Response().Header().Set("access_token", accessToken)
			}

			return next(c)
		}
	}
}
