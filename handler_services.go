package main

import (
	"net/http"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

func (cfg *apiConfig) handlerServicesTree(w http.ResponseWriter, r *http.Request) {
	services, err := cfg.db.GetServicesTree()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't load services", err)
		return
	}

	respondWithJSON(w, http.StatusOK, map[string][]database.ServiceNode{"data": services})
}
