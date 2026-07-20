package main

import (
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/auth"
	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
	"github.com/google/uuid"
)

// authenticateExistingUser is the shared guard for handlers protected by access JWTs.
func (cfg *apiConfig) authenticateExistingUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return uuid.Nil, false
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
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

// requireStaffOrAdmin is the authorization seam for endpoints that should later be restricted
// to staff/admin roles once role-based access control is added to the backend.
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
