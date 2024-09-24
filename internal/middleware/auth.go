package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/bukhavtsov/artems-dictionary/internal/usecase"
	"github.com/labstack/echo/v4"
)

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
				return c.JSON(http.StatusForbidden, map[string]string{"message": "missing or malformed token"})
			}
			// Validate the token using the JWTAuth use case
			isValid, err := jwtAuth.IsAccessTokenValid(token)
			if !isValid || err != nil {
				if err != nil {
					fmt.Printf("failed to validate token: %v", err)
				}
				return c.JSON(http.StatusForbidden, map[string]string{"message": "invalid or expired token"})
			}
			return next(c)
		}
	}
}
