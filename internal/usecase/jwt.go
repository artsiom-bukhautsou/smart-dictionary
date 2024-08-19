package usecase

import (
	"encoding/json"
	"fmt"
	"github.com/bukhavtsov/artems-dictionary/internal/infrastructure"
	"github.com/golang-jwt/jwt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// TODO: Rewrite to the env variables
const (
	secretKeyAccess     = "eXamp1eK3yACceS$"
	secretKeyRefresh    = "r3Fr3S4eXamp1eK3y"
	iss                 = "smart_dicti"
	refreshTokenName    = "refresh_token"
	accessTokenName     = "access_token"
	refreshTokenExpTime = time.Hour * 24
	accessTokenExpTime  = time.Hour
)

type JTI struct {
	Username string `json:"username"`
}

type JWTAuth struct {
	authRepo infrastructure.AuthRepository
}

func NewJWTAuth(authRepo infrastructure.AuthRepository) *JWTAuth {
	return &JWTAuth{authRepo: authRepo}
}

func parse(tokenString, secretKey string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}

func (j JWTAuth) VerifyPermission(endPoint func(w http.ResponseWriter, r *http.Request)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accessCookie, err := r.Cookie(accessTokenName)
		if err == nil && j.isVerifiedAccess(accessCookie.Value) {
			endPoint(w, r)
			return
		}
		refreshCookie, err := r.Cookie(refreshTokenName)
		if err == nil && j.isVerifiedRefresh(refreshCookie.Value) {
			username, err := j.getUsernameFromToken(refreshCookie.Value, secretKeyRefresh)
			if err != nil {
				log.Println(err)
				return
			}
			updatedAccess, err := j.getUpdatedAccess(username)
			if err != nil {
				log.Println("accessCookie token :", err)
				return
			}
			http.SetCookie(w, &http.Cookie{Name: accessTokenName, Value: updatedAccess})
			endPoint(w, r)
			return
		}
		w.WriteHeader(http.StatusGone)
	})
}

func (j JWTAuth) Validate(accessToken, refreshToken *string) error {
	if j.isVerifiedAccess(*accessToken) {
		return nil
	}
	if j.isVerifiedRefresh(*refreshToken) {
		username, err := j.getUsernameFromToken(*refreshToken, secretKeyRefresh)
		if err != nil {
			log.Println(err)
			return fmt.Errorf("failed to get user: %w", err)
		}
		updatedAccess, err := j.getUpdatedAccess(username)
		if err != nil {
			return fmt.Errorf("failed to update access token:%w", err)
		}
		accessToken = &updatedAccess
		return nil
	}
	return jwt.NewValidationError("access and refresh tokens are not valid", 413)
}

func (j JWTAuth) isVerifiedAccess(access string) bool {
	if !j.isValidTime(access, secretKeyAccess) {
		log.Println("access token time is over")
		return false
	}
	token, err := parse(access, secretKeyAccess)
	if err != nil {
		log.Printf("failed to parse access token: %v", err)
		return false
	}
	jti, err := j.GetJTI(token)
	if err != nil {
		log.Printf("failed to get JTI: %v", err)
		return false
	}
	isExist, err := j.authRepo.IsUsernameExist(jti.Username)
	if err != nil {
		log.Println("user hasn't been found:", err)
		return false
	}
	return isExist
}

func (j JWTAuth) getUpdatedAccess(username string) (access string, err error) {
	access, err = j.GenerateAccess(username)
	if err != nil {
		log.Printf("failed to generate access: %v", err)
	}
	return
}

func (j JWTAuth) isVerifiedRefresh(refresh string) bool {
	if !j.isValidTime(refresh, secretKeyRefresh) {
		log.Println("refresh token time is over")
		return false
	}
	username, err := j.getUsernameFromToken(refresh, secretKeyRefresh)
	if err != nil {
		log.Printf("failed to find user: %v", err)
		return false
	}
	return username != ""
}

func (j JWTAuth) getUsernameFromToken(tokenString, secretKeyAccess string) (string, error) {
	token, err := parse(tokenString, secretKeyAccess)
	if err != nil {
		return "", fmt.Errorf("couldn't parse token string and secret: %w", err)
	}
	jti, err := j.GetJTI(token)
	if err != nil {
		return "", fmt.Errorf("could't get jti: %w", err)
	}
	isExist, err := j.authRepo.IsUsernameExist(jti.Username)
	if err != nil {
		return "", fmt.Errorf("coudn't check in database: %w", err)
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
		Issuer:    iss,
		Id:        string(jti),
		ExpiresAt: time.Now().Add(accessTokenExpTime).Unix(),
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(secretKeyAccess))
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
		Issuer:    iss,
		Id:        string(jti),
		ExpiresAt: time.Now().Add(refreshTokenExpTime).Unix(),
	}
	rawToken := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	token, err := rawToken.SignedString([]byte(secretKeyRefresh))
	if err != nil {
		return "", fmt.Errorf("GenerateRefresh error: %w", err)
	}
	return token, nil
}
