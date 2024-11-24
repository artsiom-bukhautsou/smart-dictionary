package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	middlewareInternal "github.com/bukhavtsov/artems-dictionary/internal/middleware"
	"github.com/bukhavtsov/artems-dictionary/internal/server"
	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	allowOrigins  = os.Getenv("ALLOW_ORIGINS")
	tlsCertFile   = os.Getenv("TLS_CERT_FILE")
	tlsKeyFile    = os.Getenv("TLS_KEY_FILE")
	disableTLSEnv = os.Getenv("DISABLE_TLS")
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
		middlewareInternal.ValidateAccessToken(*jwtAuth),
	)
	apiGroup.POST("/translations", translatorServer.Translate)
	apiGroup.GET("/collections", translatorServer.GetCollections)
	apiGroup.GET("/collections/:collectionID/translations", translatorServer.GetCollectionsTranslations)
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

	var disableTLS bool
	if disableTLSEnv != "" {
		disableTLS, err = strconv.ParseBool(disableTLSEnv)
		if err != nil {
			slog.Error(err.Error())
		}
	}
	fmt.Println("disable TLS: ", disableTLS)
	if disableTLS {
		slog.Error("server has failed", slog.Any("err", e.Start(":8080")))
	} else {
		slog.Error("server has failed", slog.Any("err", e.StartTLS(":8080", tlsCertFile, tlsKeyFile)))
	}
}
