package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerAppointmentsCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Date         string   `json:"date"`
		StartTime    string   `json:"start_time"`
		EndTime      string   `json:"end_time"`
		EmployeeName string   `json:"employee_name"`
		EmployeeID   string   `json:"employee_id,omitempty"`
		ServiceIDs   []string `json:"service_ids"`
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
	if len(params.ServiceIDs) == 0 {
		respondWithError(w, http.StatusBadRequest, "service_ids is required", nil)
		return
	}

	selectedServices, err := cfg.db.GetActiveLeafServicesByIDs(params.ServiceIDs)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid service_ids provided", err)
		return
	}

	serviceNames := make([]string, 0, len(selectedServices))
	totalDuration := 0
	totalPrice := 0.0
	for _, service := range selectedServices {
		serviceNames = append(serviceNames, service.Name)
		totalDuration += service.DurationMinutes
		totalPrice += service.Price
	}

	if params.EmployeeID != "" {
		employee, err := cfg.db.GetEmployee(params.EmployeeID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't validate employee", err)
			return
		}
		if employee == nil {
			respondWithError(w, http.StatusBadRequest, "Invalid employee_id provided", nil)
			return
		}
	}

	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
		return
	}

	appointment, err := cfg.db.CreateAppointment(database.CreateAppointmentParams{
		Date:                 params.Date,
		StartTime:            params.StartTime,
		EndTime:              params.EndTime,
		EmployeeName:         params.EmployeeName,
		EmployeeID:           params.EmployeeID,
		UserID:               userID.String(),
		Services:             serviceNames,
		TotalDurationMinutes: totalDuration,
		TotalPrice:           totalPrice,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save appointment", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, appointment)
}

func (cfg *apiConfig) handlerAppointmentsList(w http.ResponseWriter, r *http.Request) {
	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
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
	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
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
	userID, ok := cfg.authenticateExistingUser(w, r)
	if !ok {
		return
	}

	err := cfg.db.CancelAllAppointmentsFromUser(userID.String())
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
