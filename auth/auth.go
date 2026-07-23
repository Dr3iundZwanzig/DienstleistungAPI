package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenType string

const (
	TokenTypeAccess TokenType = "dienstleistung-access"
)

var ErrNoAuthHeaderIncluded = errors.New("no auth header included in request")

// erweitert RegisteredClaims mit CustomClaims
type CustomClaims struct {
	Scope          string `json:"scope"`
	SessionVersion int    `json:"session_version"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, argon2id.DefaultParams)
	if err != nil {
		return "", err
	}
	return hash, nil
}

func CheckPasswordHash(password, hash string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hash)
	if err != nil {
		return false, err
	}
	return match, nil
}

func MakeJWT(
	userID uuid.UUID,
	tokenSecret string,
	expiresIn time.Duration,
	issuer string,
	audience string,
	scope string,
) (string, error) {
	return MakeJWTWithSessionVersion(userID, tokenSecret, expiresIn, issuer, audience, scope, 1)
}

func MakeJWTWithSessionVersion(
	userID uuid.UUID,
	tokenSecret string,
	expiresIn time.Duration,
	issuer string,
	audience string,
	scope string,
	sessionVersion int,
) (string, error) {
	if sessionVersion <= 0 {
		sessionVersion = 1
	}

	signingKey := []byte(tokenSecret)
	claims := CustomClaims{
		Scope:          scope,
		SessionVersion: sessionVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(expiresIn)),
			Subject:   userID.String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(signingKey)
}

// testing
func ValidateJWT(tokenString, tokenSecret, expectedIssuer, expectedAudience string) (uuid.UUID, error) {
	userID, _, err := ValidateJWTClaims(tokenString, tokenSecret, expectedIssuer, expectedAudience)
	if err != nil {
		return uuid.Nil, err
	}

	return userID, nil
}

func ValidateJWTClaims(tokenString, tokenSecret, expectedIssuer, expectedAudience string) (uuid.UUID, CustomClaims, error) {
	claimsStruct := CustomClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		return uuid.Nil, CustomClaims{}, err
	}
	if !token.Valid {
		return uuid.Nil, CustomClaims{}, errors.New("invalid token")
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, CustomClaims{}, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, CustomClaims{}, err
	}
	if issuer != expectedIssuer {
		return uuid.Nil, CustomClaims{}, errors.New("invalid issuer")
	}

	audience, err := token.Claims.GetAudience()
	if err != nil {
		return uuid.Nil, CustomClaims{}, err
	}
	if len(audience) == 0 || audience[0] != expectedAudience {
		return uuid.Nil, CustomClaims{}, errors.New("invalid audience")
	}

	if claimsStruct.SessionVersion <= 0 {
		return uuid.Nil, CustomClaims{}, errors.New("invalid session version")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, CustomClaims{}, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, claimsStruct, nil
}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "Bearer" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}

func MakeRefreshToken() (string, error) {
	token := make([]byte, 32)
	_, err := rand.Read(token)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(token), nil
}

// hashes refresh token mit SHA-256
func HashRefreshToken(token string) (string, error) {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash), nil
}

// vergleicht token mit hash in der datenbank
func VerifyRefreshToken(token, hash string) (bool, error) {
	computedHash, err := HashRefreshToken(token)
	if err != nil {
		return false, err
	}
	return computedHash == hash, nil
}

func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeaderIncluded
	}
	splitAuth := strings.Split(authHeader, " ")
	if len(splitAuth) < 2 || splitAuth[0] != "ApiKey" {
		return "", errors.New("malformed authorization header")
	}

	return splitAuth[1], nil
}
