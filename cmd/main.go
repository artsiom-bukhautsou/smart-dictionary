package main

import (
	"context"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	"github.com/bukhavtsov/artems-dictionary/internal/server"
	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	chatGPTAPIURL = os.Getenv("CHAT_GPT_API_URL")
	apiKey        = os.Getenv("OPEN_AI_API_KEY")

	postgresUserName = os.Getenv("POSTGRES_USERNAME")
	postgresPassword = os.Getenv("POSTGRES_PASSWORD")
	postgresPort     = os.Getenv("POSTGRES_PORT")
	postgresHost     = os.Getenv("POSTGRES_HOST")
	postgresDBName   = os.Getenv("POSTGRES_DBNAME")

	jwtSecretKeyAccess     = os.Getenv("JWT_SECRET_KEY_ACCESS")
	jwtSecretKeyRefresh    = os.Getenv("JWT_SECRET_KEY_REFRESH")
	jwtIss                 = os.Getenv("JWT_ISS")
	jwtRefreshTokenExpTime = os.Getenv("JWT_REFRESH_TOKEN_EXP_TIME")
	jwtAccessTokenExpTime  = os.Getenv("JWT_ACCESS_TOKEN_EXP_TIME")

	allowOrigins = os.Getenv("ALLOW_ORIGINS")
	tlsCertFile  = os.Getenv("TLS_CERT_FILE")
	tlsKeyFile   = os.Getenv("TLS_KEY_FILE")
)

func main() {
	e := echo.New()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	connString := "postgres://" + postgresUserName + ":" + postgresPassword + "@" + postgresHost + ":" + postgresPort + "/" + postgresDBName
	conn, err := pgxpool.New(context.Background(), connString)
	if err != nil {
		logger.Error("Unable to connect to the database", slog.Any("err", err))
		return
	}

	jwtRefreshTokenExpTimeDuration, err := time.ParseDuration(jwtRefreshTokenExpTime)
	if err != nil {
		logger.Error("Unable to parse JWT_REFRESH_TOKEN_EXP_TIME", slog.Any("err", err))
		return
	}
	jwtAccessTokenExpTimeDuration, err := time.ParseDuration(jwtAccessTokenExpTime)
	if err != nil {
		logger.Error("Unable to parse JWT_ACCESS_TOKEN_EXP_TIME", slog.Any("err", err))
		return
	}
	originsList := strings.Split(allowOrigins, ",")
	fmt.Println(originsList)

	authRepository := infrastructure.NewAuthRepository(conn)
	jwtAuth := usecase.NewJWTAuth(
		*authRepository,
		jwtSecretKeyAccess,
		jwtSecretKeyRefresh,
		jwtIss,
		jwtRefreshTokenExpTimeDuration,
		jwtAccessTokenExpTimeDuration,
	)
	authService := usecase.NewAuthService(*authRepository, *jwtAuth)
	translationRepository := infrastructure.NewTranslationRepository(conn)
	translatorServer := server.NewTranslatorServer(
		*authService,
		*jwtAuth,
		jwtAccessTokenExpTimeDuration,
		jwtRefreshTokenExpTimeDuration,
		translationRepository,
		*logger,
		chatGPTAPIURL,
		apiKey,
	)

	apiGroup := e.Group("/api")
	apiGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     originsList,
			AllowCredentials: true,
		}),
		ValidateAccessToken(*jwtAuth),
	)
	apiGroup.POST("/translations", translatorServer.Translate)
	apiGroup.DELETE("/accounts", translatorServer.DeleteUsersAccount)

	authGroup := e.Group("/auth")
	authGroup.Use(
		middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins:     originsList,
			AllowCredentials: true,
		}),
	)
	authGroup.POST("/signin", translatorServer.SignIn)
	authGroup.POST("/signup", translatorServer.SignUp)
	authGroup.POST("/refresh", translatorServer.RefreshRefreshToken)
	slog.Error("server has failed", slog.Any("err", e.StartTLS(":8080", tlsCertFile, tlsKeyFile)))
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
