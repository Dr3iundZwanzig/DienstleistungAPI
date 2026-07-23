package main

import (
	"net/http"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/auth"
	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	user, err := cfg.db.GetUserByRefreshToken(refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", err)
		return
	}
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't get user for refresh token", nil)
		return
	}

	// Hash des alten token für rotation
	oldTokenHash, err := auth.HashRefreshToken(refreshToken)
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

	accessToken, err := auth.MakeJWT(
		user.ID,
		cfg.jwtSecret,
		cfg.refreshedAccessTokenTTL,
		cfg.jwtIssuer,
		cfg.jwtAudience,
		"user", // hardcoded scope später ändern
	)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate token", err)
		return
	}

	respondWithJSON(w, http.StatusOK, response{
		Token:        accessToken,
		RefreshToken: newRefreshToken,
	})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't find token", err)
		return
	}

	tokenHash, err := auth.HashRefreshToken(refreshToken)
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
