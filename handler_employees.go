package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

type employeeSeedPayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Specialties []string `json:"specialties"`
	IsActive    bool     `json:"is_active"`
}

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

func (cfg *apiConfig) handlerTestResetAndSeed(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" && cfg.platform != "test" {
		respondWithError(w, http.StatusForbidden, "Test reset is only available in dev or test", nil)
		return
	}

	if err := cfg.db.Reset(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset database", err)
		return
	}

	seedData := defaultSeedEmployees()
	if err := cfg.seedEmployees(seedData); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seed default test data", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]any{
		"message":          "Database reset and test data seeded",
		"seeded_employees": len(seedData),
	})
}

func (cfg *apiConfig) seedEmployees(employees []employeeSeedPayload) error {
	for _, employee := range employees {
		_, err := cfg.db.CreateEmployee(database.CreateEmployeeParams{
			ID:          employee.ID,
			Name:        employee.Name,
			Title:       employee.Title,
			Specialties: employee.Specialties,
			IsActive:    employee.IsActive,
		})
		if err != nil {
			return err
		}

		_, err = cfg.db.CreateAvailability(database.CreateAvailabilityParams{
			EmployeeID: employee.ID,
			Dates:      buildSeedAvailabilityDates(),
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func defaultSeedEmployees() []employeeSeedPayload {
	return []employeeSeedPayload{
		{
			ID:          "emp_anna",
			Name:        "Anna Weber",
			Title:       "Senior Stylist",
			Specialties: []string{"Haarschnitt", "Farbe"},
			IsActive:    true,
		},
		{
			ID:          "emp_ben",
			Name:        "Ben Kruger",
			Title:       "Barber",
			Specialties: []string{"Fade", "Bart"},
			IsActive:    true,
		},
		{
			ID:          "emp_carla",
			Name:        "Carla Neumann",
			Title:       "Color Specialist",
			Specialties: []string{"Balayage", "Farbberatung"},
			IsActive:    true,
		},
	}
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
