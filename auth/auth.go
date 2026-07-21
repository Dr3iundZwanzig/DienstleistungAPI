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

// CustomClaims extends jwt.RegisteredClaims with standardized claims
type CustomClaims struct {
	Scope string `json:"scope"`
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
	signingKey := []byte(tokenSecret)
	claims := CustomClaims{
		Scope: scope,
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

// kann erweitert werden um auch CustomClaims als return zu haben für witere überprüfung des scopes
func ValidateJWT(tokenString, tokenSecret, expectedIssuer, expectedAudience string) (uuid.UUID, error) {
	claimsStruct := CustomClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		&claimsStruct,
		func(token *jwt.Token) (interface{}, error) { return []byte(tokenSecret), nil },
	)
	if err != nil {
		return uuid.Nil, err
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return uuid.Nil, err
	}

	issuer, err := token.Claims.GetIssuer()
	if err != nil {
		return uuid.Nil, err
	}
	if issuer != expectedIssuer {
		return uuid.Nil, errors.New("invalid issuer")
	}

	audience, err := token.Claims.GetAudience()
	if err != nil {
		return uuid.Nil, err
	}
	if len(audience) == 0 || audience[0] != expectedAudience {
		return uuid.Nil, errors.New("invalid audience")
	}

	id, err := uuid.Parse(userIDString)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user ID: %w", err)
	}
	return id, nil
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

// HashRefreshToken hashes a refresh token using SHA-256 for deterministic storage
// Unlike passwords, tokens need deterministic hashing to enable database queries
func HashRefreshToken(token string) (string, error) {
	hash := sha256.Sum256([]byte(token))
	return fmt.Sprintf("%x", hash), nil
}

// VerifyRefreshToken compares a plaintext refresh token with its stored SHA-256 hash
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
