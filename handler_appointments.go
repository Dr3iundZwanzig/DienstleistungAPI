package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/auth"
	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerAppointmentsCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Date                 string   `json:"date"`
		StartTime            string   `json:"start_time"`
		EndTime              string   `json:"end_time"`
		EmployeeName         string   `json:"employee_name"`
		EmployeeID           string   `json:"employee_id,omitempty"`
		Services             []string `json:"services"`
		TotalDurationMinutes int      `json:"total_duration_minutes"`
		TotalPrice           float64  `json:"total_price"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode appointment data", err)
		return
	}

	if params.Date == "" || params.StartTime == "" || params.EndTime == "" {
		respondWithError(w, http.StatusBadRequest, "date, start_time and end_time are required", nil)
		return
	}

	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	appointment, err := cfg.db.CreateAppointment(database.CreateAppointmentParams{
		Date:                 params.Date,
		StartTime:            params.StartTime,
		EndTime:              params.EndTime,
		EmployeeName:         params.EmployeeName,
		EmployeeID:           params.EmployeeID,
		UserID:               userID.String(),
		Services:             params.Services,
		TotalDurationMinutes: params.TotalDurationMinutes,
		TotalPrice:           params.TotalPrice,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save appointment", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, appointment)
}

func (cfg *apiConfig) handlerAppointmentsList(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	appointments, err := cfg.db.GetAppointmentsByUserID(userID.String())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't load appointments", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string][]database.Appointment{"data": appointments})
}

func (cfg *apiConfig) handlerAppointmentsCancel(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	appointmentID := r.PathValue("id")
	if appointmentID == "" {
		respondWithError(w, http.StatusBadRequest, "appointment id is required", nil)
		return
	}

	appointment, err := cfg.db.CancelAppointmentByIDAndUserID(appointmentID, userID.String())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't cancel appointment", err)
		return
	}
	if appointment == nil {
		respondWithError(w, http.StatusNotFound, "Appointment not found", nil)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Appointment cancelled",
		"id":      appointment.ID,
	})
}
func (cfg *apiConfig) handlerAppointmentsCancelAll(w http.ResponseWriter, r *http.Request) {
	bearerToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Authorization token required", err)
		return
	}

	userID, err := auth.ValidateJWT(bearerToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	err = cfg.db.CancelAllAppointmentsFromUser(userID.String())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't cancel appointment", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{
		"message": "Appointment cancelled",
	})
}

func resolveAppointmentEmployeeName(providedName, employeeID string, lookup func(string) (*database.Employee, error)) (string, error) {
	if employeeID == "" {
		return providedName, nil
	}

	if providedName != "" && providedName != "Keine Präferenz" {
		return providedName, nil
	}

	employee, err := lookup(employeeID)
	if err != nil {
		return "", fmt.Errorf("lookup employee %s: %w", employeeID, err)
	}
	if employee == nil || employee.Name == "" {
		return providedName, nil
	}

	return employee.Name, nil
}
