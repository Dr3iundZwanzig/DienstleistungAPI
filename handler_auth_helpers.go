package main

import (
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/auth"
	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
	"github.com/google/uuid"
)

// handler für auth von usern
func (cfg *apiConfig) authenticateExistingUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return uuid.Nil, false
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret, cfg.jwtIssuer, cfg.jwtAudience)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
		return uuid.Nil, false
	}

	user, err := cfg.db.GetUser(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't validate user", err)
		return uuid.Nil, false
	}
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Session no longer valid. Please log in again.", nil)
		return uuid.Nil, false
	}

	return userID, true
}

// role check für staff/admin endpoints später auch mit claims scope check
func (cfg *apiConfig) requireStaffOrAdmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
		return uuid.Nil, false
	}

	user, err := cfg.db.GetUser(userID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't validate user role", err)
		return uuid.Nil, false
	}
	if user == nil {
		respondWithError(w, http.StatusUnauthorized, "Session no longer valid. Please log in again.", nil)
		return uuid.Nil, false
	}

	if user.Role != database.UserRoleStaff && user.Role != database.UserRoleAdmin {
		respondWithError(w, http.StatusForbidden, "Staff or admin role required", nil)
		return uuid.Nil, false
	}

	return userID, true
}
