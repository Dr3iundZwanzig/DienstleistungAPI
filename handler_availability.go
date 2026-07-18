package main

import (
	"encoding/json"
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerAvailabilityGet(w http.ResponseWriter, r *http.Request) {
	employeeID := r.URL.Query().Get("employee_id")
	if employeeID == "" {
		respondWithError(w, http.StatusBadRequest, "employee_id query parameter is required", nil)
		return
	}

	availability, err := cfg.db.GetAvailabilityByEmployeeID(employeeID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't load availability", err)
		return
	}
	if availability == nil {
		seededAvailability, seedErr := cfg.db.CreateAvailability(database.CreateAvailabilityParams{
			EmployeeID: employeeID,
			Dates:      buildSeedAvailabilityDates(),
		})
		if seedErr != nil {
			respondWithError(w, http.StatusInternalServerError, "Couldn't seed availability", seedErr)
			return
		}
		respondWithJSON(w, http.StatusOK, seededAvailability)
		return
	}

	respondWithJSON(w, http.StatusOK, availability)
}

func (cfg *apiConfig) handlerAvailabilityCreate(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		EmployeeID string                      `json:"employee_id"`
		Dates      []database.AvailabilityDate `json:"dates"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	if err := decoder.Decode(&params); err != nil {
		respondWithError(w, http.StatusBadRequest, "Couldn't decode availability data", err)
		return
	}

	if params.EmployeeID == "" || len(params.Dates) == 0 {
		respondWithError(w, http.StatusBadRequest, "employee_id and dates are required", nil)
		return
	}

	availability, err := cfg.db.CreateAvailability(database.CreateAvailabilityParams{
		EmployeeID: params.EmployeeID,
		Dates:      params.Dates,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't save availability", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]interface{}{
		"message":      "Availability saved",
		"availability": availability,
	})
}
