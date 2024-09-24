// Implemented according to this guide https://dekh.medium.com/the-complete-guide-to-json-web-tokens-jwt-and-token-based-authentication-32501cb5125c
package usecase

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	"github.com/golang-jwt/jwt"
)

type JTI struct {
	Username string `json:"username"`
}

type JWTAuth struct {
	authRepo            infrastructure.AuthRepository
	secretKeyAccess     string
	secretKeyRefresh    string
	iss                 string
	refreshTokenExpTime time.Duration
	accessTokenExpTime  time.Duration
}

func NewJWTAuth(
	authRepo infrastructure.AuthRepository,
	secretKeyAccess string,
	secretKeyRefresh string,
	iss string,
	refreshTokenExpTime time.Duration,
	accessTokenExpTime time.Duration,
) *JWTAuth {
	return &JWTAuth{
		authRepo:            authRepo,
		secretKeyAccess:     secretKeyAccess,
		secretKeyRefresh:    secretKeyRefresh,
		iss:                 iss,
		refreshTokenExpTime: refreshTokenExpTime,
		accessTokenExpTime:  accessTokenExpTime,
	}
}

func parse(tokenString, secretKey string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (j JWTAuth) RefreshRefreshToken(refreshToken string) (*domain.Token, error) {
	isValid, err := j.IsRefreshTokenValid(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}
	if !isValid {
		return nil, errors.New("invalid refresh token")
	}
	token, err := parse(refreshToken, j.secretKeyRefresh)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	sub, err := j.getSubjectFromToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to get sub from refresh token: %w", err)
	}
	updatedRefresh, err := j.GenerateRefresh(sub)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	updatedAccess, err := j.GenerateAccess(sub)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: updatedAccess, Refresh: updatedRefresh}, nil
}

func (j JWTAuth) IsAccessTokenValid(access string) (bool, error) {
	token, err := parse(access, j.secretKeyAccess)
	if err != nil {
		return false, fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	if !j.isValidTime(token) {
		return false, fmt.Errorf("access token time is over")
	}
	return true, nil
}

func (j JWTAuth) IsRefreshTokenValid(refresh string) (bool, error) {
	token, err := parse(refresh, j.secretKeyRefresh)
	if err != nil {
		return false, fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	if !j.isValidTime(token) {
		return false, errors.New("refresh token time is over")
	}
	sub, err := j.getSubjectFromToken(token)
	if err != nil {
		return false, fmt.Errorf("failed to get sub: %w", err)
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		return false, fmt.Errorf("access token has sub with invalid format: %w", err)
	}
	exist, err := j.authRepo.DoesUserIDExist(userID)
	if err != nil {
		return false, fmt.Errorf("user isn't found: %w", err)
	}
	return exist, nil
}

func (j JWTAuth) GetSubFromAccessToken(tokenStr string) (string, error) {
	token, err := parse(tokenStr, j.secretKeyAccess)
	if err != nil {
		return "", fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	return j.getSubjectFromToken(token)
}

func (j JWTAuth) getSubjectFromToken(token *jwt.Token) (string, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if sub, ok := claims["sub"].(string); ok {
			return sub, nil
		}
		return "", fmt.Errorf("`sub` claim is not present in token")
	}
	return "", fmt.Errorf("invalid token")
}

func (j JWTAuth) isValidTime(token *jwt.Token) bool {
	// TODO: refactor
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expString := fmt.Sprintf("%f", claims["exp"])
		fmt.Println("expString", expString)
		exp, err := strconv.ParseFloat(expString, 64)
		if err != nil {
			log.Println(err)
			return false
		}
		now := float64(time.Now().Unix())
		if exp > now {
			return true
		}
		fmt.Println("now", now)
	}
	return false
}

func (j JWTAuth) GenerateAccess(sub string) (tokenString string, err error) {
	claims := jwt.StandardClaims{
		Issuer:    j.iss,
		Subject:   sub,
		ExpiresAt: time.Now().Add(j.accessTokenExpTime).Unix(),
	}
	// TODO: add authorisation also
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(j.secretKeyAccess))
	if err != nil {
		return "", fmt.Errorf("couldn't generate access token")
	}
	return token, nil
}

func (j JWTAuth) GenerateRefresh(sub string) (tokenString string, err error) {
	claims := jwt.StandardClaims{
		Issuer:    j.iss,
		Subject:   sub,
		ExpiresAt: time.Now().Add(j.refreshTokenExpTime).Unix(),
		// TODO: as far as I have GTI in db, it's probably makes sense to generate a unique token for it
		// maybe it makes sense to add a authorisation also by user role. That would be awesome
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(j.secretKeyRefresh))
	if err != nil {
		return "", fmt.Errorf("couldn't generate refresh token")
	}
	userID, err := strconv.Atoi(sub)
	if err != nil {
		return "", fmt.Errorf("sub has unsupported format: %w", err)
	}
	err = j.authRepo.UpdateRefreshToken(userID, token)
	if err != nil {
		return "", fmt.Errorf("failed to update refresh token: %w", err)
	}
	return token, nil
}
