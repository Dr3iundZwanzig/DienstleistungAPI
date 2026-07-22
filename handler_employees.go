package main

import (
	"encoding/json"
	"net/http"

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
