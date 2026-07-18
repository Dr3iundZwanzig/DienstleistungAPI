package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerEmployeesList(w http.ResponseWriter, r *http.Request) {
	employees, err := cfg.db.GetEmployees()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't load employees", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string][]database.Employee{"data": employees})
}

func (cfg *apiConfig) handlerEmployeesResolve(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Services []string `json:"services"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode employee resolution data", err)
		return
	}

	employeeID, err := cfg.db.ResolveEmployeeForServices(params.Services)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't resolve employee", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"employee_id": employeeID})
}

func (cfg *apiConfig) handlerEmployeesSeed(w http.ResponseWriter, r *http.Request) {
	type employeePayload struct {
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Title       string   `json:"title"`
		Specialties []string `json:"specialties"`
		IsActive    bool     `json:"is_active"`
	}

	var payload struct {
		Data []employeePayload `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode employee data", err)
		return
	}

	for _, employee := range payload.Data {
		_, err := cfg.db.CreateEmployee(database.CreateEmployeeParams{
			ID:          employee.ID,
			Name:        employee.Name,
			Title:       employee.Title,
			Specialties: employee.Specialties,
			IsActive:    employee.IsActive,
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't save employee", err)
			return
		}

		_, err = cfg.db.CreateAvailability(database.CreateAvailabilityParams{
			EmployeeID: employee.ID,
			Dates:      buildSeedAvailabilityDates(),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't save availability for employee", err)
			return
		}
	}
	fmt.Print("employee data seeded")
	respondWithJSON(w, http.StatusCreated, map[string]string{"message": "Employees seeded"})
}

func buildSeedAvailabilityDates() []database.AvailabilityDate {
	start := time.Now().AddDate(0, 0, 2)
	baseDate := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)

	dates := make([]database.AvailabilityDate, 0, 4)
	for offset := 0; offset < 4; offset++ {
		current := baseDate.AddDate(0, 0, offset*2)
		dateStr := current.Format("2006-01-02")
		slots := []database.AvailabilitySlot{
			{StartTime: "09:00", EndTime: "09:30", IsAvailable: true},
			{StartTime: "09:30", EndTime: "10:00", IsAvailable: true},
			{StartTime: "10:00", EndTime: "10:30", IsAvailable: false},
			{StartTime: "10:30", EndTime: "11:00", IsAvailable: false},
			{StartTime: "11:00", EndTime: "11:30", IsAvailable: true},
			{StartTime: "11:30", EndTime: "12:00", IsAvailable: true},
			{StartTime: "14:00", EndTime: "14:30", IsAvailable: true},
			{StartTime: "14:30", EndTime: "15:00", IsAvailable: true},
		}
		dates = append(dates, database.AvailabilityDate{Date: dateStr, Slots: slots})
	}

	return dates
}
