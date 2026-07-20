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

func (cfg *apiConfig) seedServices(services []database.ServiceNode) error {
	return cfg.db.ReplaceServicesTree(services)
}

func defaultSeedServicesTree() []database.ServiceNode {
	return []database.ServiceNode{
		{
			ID:       "cat_01",
			Name:     "Friseur",
			IsActive: true,
			Children: []database.ServiceNode{
				{
					ID:       "sub_01",
					Name:     "Herren",
					IsActive: true,
					Children: []database.ServiceNode{
						{
							ID:              "srv_001",
							Name:            "Haarschnitt",
							Description:     "Waschen, Schneiden, Föhnen",
							DurationMinutes: 45,
							Price:           39.90,
							Currency:        "EUR",
							IsActive:        true,
						},
						{
							ID:              "srv_002",
							Name:            "Bartpflege",
							Description:     "Trimmen und Konturen nachrasieren",
							DurationMinutes: 30,
							Price:           19.90,
							Currency:        "EUR",
							IsActive:        true,
						},
					},
				},
				{
					ID:       "sub_02",
					Name:     "Damen",
					IsActive: true,
					Children: []database.ServiceNode{
						{
							ID:              "srv_003",
							Name:            "Farbbehandlung",
							Description:     "Färben oder Tönen inkl. Beratung",
							DurationMinutes: 90,
							Price:           79.00,
							Currency:        "EUR",
							IsActive:        true,
						},
					},
				},
			},
		},
		{
			ID:       "cat_02",
			Name:     "Kosmetik",
			IsActive: true,
			Children: []database.ServiceNode{
				{
					ID:              "srv_004",
					Name:            "Maniküre",
					Description:     "Nagelpflege inkl. Lackieren",
					DurationMinutes: 40,
					Price:           29.90,
					Currency:        "EUR",
					IsActive:        true,
				},
				{
					ID:              "srv_005",
					Name:            "Gesichtsbehandlung",
					Description:     "Reinigung, Pflege und Abschlusspflege",
					DurationMinutes: 60,
					Price:           49.90,
					Currency:        "EUR",
					IsActive:        true,
				},
			},
		},
	}
}

func countServiceNodes(nodes []database.ServiceNode) int {
	total := 0
	for _, node := range nodes {
		total++
		total += countServiceNodes(node.Children)
	}
	return total
}
