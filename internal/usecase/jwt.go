// Implemented according to this guide https://dekh.medium.com/the-complete-guide-to-json-web-tokens-jwt-and-token-based-authentication-32501cb5125c
package usecase

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/domain"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	"github.com/golang-jwt/jwt"
	"log"
	"strconv"
	"time"
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
	username, err := j.getUsernameFromToken(refreshToken, j.secretKeyRefresh)
	if err != nil {
		return nil, fmt.Errorf("failed to get username from refresh token: %w", err)
	}
	updatedRefresh, err := j.GenerateRefresh(username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	updatedAccess, err := j.GenerateAccess(username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}
	return &domain.Token{Access: updatedAccess, Refresh: updatedRefresh}, nil
}

func (j JWTAuth) IsAccessTokenValid(access string) (bool, error) {
	if !j.isValidTime(access, j.secretKeyAccess) {
		return false, fmt.Errorf("access token time is over")
	}
	token, err := parse(access, j.secretKeyAccess)
	if err != nil {
		return false, fmt.Errorf("failed to parse access token: %w", err)
	}
	jti, err := j.GetJTI(token)
	if err != nil {
		return false, fmt.Errorf("failed to get JTI: %w", err)
	}
	isExist, err := j.authRepo.IsUsernameExist(jti.Username)
	if err != nil {
		return false, fmt.Errorf("user hasn't been found: %w", err)
	}
	return isExist, nil
}

func (j JWTAuth) IsRefreshTokenValid(refresh string) (bool, error) {
	if !j.isValidTime(refresh, j.secretKeyRefresh) {
		return false, errors.New("refresh token time is over")
	}
	username, err := j.getUsernameFromToken(refresh, j.secretKeyRefresh)
	if err != nil {
		return false, fmt.Errorf("failed to find user: %w", err)
	}
	return username != "", nil
}

func (j JWTAuth) GetUsernameFromAccessToken(token string) (string, error) {
	return j.getUsernameFromToken(token, j.secretKeyAccess)
}

func (j JWTAuth) getUsernameFromToken(tokenString, secretKey string) (string, error) {
	token, err := parse(tokenString, secretKey)
	if err != nil {
		return "", fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	jti, err := j.GetJTI(token)
	if err != nil {
		return "", fmt.Errorf("couldn't get jti: %w", err)
	}
	isExist, err := j.authRepo.IsUsernameExist(jti.Username)
	if err != nil {
		return "", fmt.Errorf("couldn't check in database: %w", err)
	}
	if !isExist {
		return "", fmt.Errorf("couldn't find the username")
	}
	return jti.Username, nil
}

func (j JWTAuth) isValidTime(tokenString, secretKey string) bool {
	token, err := parse(tokenString, secretKey)
	if err != nil {
		log.Printf("failed to parse token: %v", err)
		return false
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expString := fmt.Sprintf("%f", claims["exp"])
		exp, err := strconv.ParseFloat(expString, 64)
		if err != nil {
			log.Println(err)
			return false
		}
		now := float64(time.Now().Unix())
		if exp > now {
			return true
		}
	}
	return false
}

func (j JWTAuth) GetJTI(token *jwt.Token) (*JTI, error) {
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		jtiJson := fmt.Sprintf("%v", claims["jti"])
		var jti JTI
		err := json.Unmarshal([]byte(jtiJson), &jti)
		if err != nil {
			return nil, err
		}
		return &jti, nil
	}
	return nil, fmt.Errorf("user hasn't been found")
}

func (j JWTAuth) GenerateAccess(username string) (tokenString string, err error) {
	jti, err := json.Marshal(&JTI{username})
	if err != nil {
		return "", fmt.Errorf("failed to marshal access JTI: %w", err)
	}
	claims := jwt.StandardClaims{
		Issuer:    j.iss,
		Id:        string(jti),
		ExpiresAt: time.Now().Add(j.accessTokenExpTime).Unix(),
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(j.secretKeyAccess))
	if err != nil {
		return "", err
	}
	return token, nil
}

func (j JWTAuth) GenerateRefresh(username string) (tokenString string, err error) {
	jti, err := json.Marshal(&JTI{username})
	if err != nil {
		return "", fmt.Errorf("failed to marshal refresh JTI: %w", err)
	}
	claims := jwt.StandardClaims{
		Issuer:    j.iss,
		Id:        string(jti),
		ExpiresAt: time.Now().Add(j.refreshTokenExpTime).Unix(),
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(j.secretKeyRefresh))
	if err != nil {
		return "", fmt.Errorf("GenerateRefresh error: %w", err)
	}
	return token, nil
}
