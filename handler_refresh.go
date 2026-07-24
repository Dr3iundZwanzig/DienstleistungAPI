package main

import (
	"net/http"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/auth"
	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	refreshTokenValue := ""
	if bearerToken, err := auth.GetBearerToken(r.Header); err == nil {
		refreshTokenValue = bearerToken
	} else {
		cookie, cookieErr := r.Cookie("refresh_token")
		if cookieErr != nil || cookie.Value == "" {
			respondWithError(w, http.StatusBadRequest, "Couldn't find token", cookieErr)
			return
		}
		refreshTokenValue = cookie.Value
	}

	user, err := cfg.db.GetUserByRefreshToken(refreshTokenValue)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", nil)
		return
	}

	// Hash des alten token für rotation
	oldTokenHash, err := auth.HashRefreshToken(refreshTokenValue)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash refresh token", err)
		return
	}

	newRefreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create refresh token", err)
		return
	}

	newTokenHash, err := auth.HashRefreshToken(newRefreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash new refresh token", err)
		return
	}
	//token rotation refreshed auch ttl
	_, err = cfg.db.RotateRefreshToken(oldTokenHash, database.CreateRefreshTokenParams{
		Token:     newTokenHash,
		UserID:    user.ID,
		ExpiresAt: time.Now().UTC().Add(cfg.refreshTokenTTL),
	})
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't rotate refresh token", err)
		return
	}

	accessToken, err := auth.MakeJWTWithSessionVersion(
		user.ID,
		cfg.jwtSecret,
		cfg.refreshedAccessTokenTTL,
		cfg.jwtIssuer,
		cfg.jwtAudience,
		"user", // hardcoded scope später ändern
		user.SessionVersion,
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefreshToken,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		MaxAge:   int(cfg.refreshTokenTTL.Seconds()),
	})

	respondWithJSON(w, http.StatusOK, response{
		Token: accessToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	clearRefreshTokenCookie(w)

	refreshTokenValue := ""
	if bearerToken, err := auth.GetBearerToken(r.Header); err == nil {
		refreshTokenValue = bearerToken
	} else {
		cookie, cookieErr := r.Cookie("refresh_token")
		if cookieErr != nil || cookie.Value == "" {
			respondWithError(w, http.StatusBadRequest, "Couldn't find token", cookieErr)
			return
		}
		refreshTokenValue = cookie.Value
	}

	tokenHash, err := auth.HashRefreshToken(refreshTokenValue)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash token", err)
		return
	}

	err = cfg.db.RevokeRefreshToken(tokenHash)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't revoke session", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// logout für user
func (cfg *apiConfig) handlerLogoutAll(w http.ResponseWriter, r *http.Request) {
	clearRefreshTokenCookie(w)

	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
		return
	}

	err := cfg.db.InvalidateUserSessions(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't invalidate sessions", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
