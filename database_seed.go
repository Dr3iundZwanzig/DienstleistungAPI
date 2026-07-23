package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Dr3iundZwanzig/DienstleistungAPI/database"
)

//test daten für die datenbank

type employeeSeedPayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Specialties []string `json:"specialties"`
	IsActive    bool     `json:"is_active"`
}

func (cfg *apiConfig) seedServices(services []database.ServiceNode) error {
	return cfg.db.ReplaceServicesTree(services)
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

func defaultSeedEmployees() []employeeSeedPayload {
	return []employeeSeedPayload{
		{
			ID:          "emp_001",
			Name:        "Anna Müller",
			Title:       "Friseurmeisterin",
			Specialties: []string{"Haarschnitt", "Farbbehandlung"},
			IsActive:    true,
		},
		{
			ID:          "emp_002",
			Name:        "Thomas Schmidt",
			Title:       "Friseur",
			Specialties: []string{"Haarschnitt", "Bartpflege"},
			IsActive:    true,
		},
		{
			ID:          "emp_003",
			Name:        "Sarah Weber",
			Title:       "Kosmetikerin",
			Specialties: []string{"Maniküre", "Gesichtsbehandlung"},
			IsActive:    true,
		},
		{
			ID:          "emp_004",
			Name:        "Michael Klein",
			Title:       "Barbier",
			Specialties: []string{"Bartpflege", "Haarschnitt"},
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

func countServiceNodes(nodes []database.ServiceNode) int {
	total := 0
	for _, node := range nodes {
		total++
		total += countServiceNodes(node.Children)
	}
	return total
}

// datenbank nur für tests zurücksetzen und wieder mit neuen test daten füllen user daten werden auch gelöcht
func (cfg *apiConfig) handlerTestResetAndSeed(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" && cfg.platform != "test" {
		respondWithError(w, http.StatusForbidden, "Test reset is only available in dev or test", nil)
		return
	}

	if err := cfg.db.Reset(); err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset database", err)
		return
	}

	serviceSeedData, seedData, err := cfg.seedDatabase()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't seed default test data", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]any{
		"message":          "Database reset and test data seeded",
		"seeded_employees": len(seedData),
		"seeded_services":  countServiceNodes(serviceSeedData),
	})
}

func (cfg *apiConfig) seedDatabase() ([]database.ServiceNode, []employeeSeedPayload, error) {
	serviceSeedData := defaultSeedServicesTree()
	if err := cfg.seedServices(serviceSeedData); err != nil {
		return []database.ServiceNode{}, []employeeSeedPayload{}, err
	}

	seedData := defaultSeedEmployees()
	if err := cfg.seedEmployees(seedData); err != nil {
		return []database.ServiceNode{}, []employeeSeedPayload{}, err
	}
	return serviceSeedData, seedData, nil
}

func (cfg *apiConfig) ensureDatabaseSeeded() error {
	services, err := cfg.db.GetServicesTree()
	if err != nil {
		return err
	}

	employees, err := cfg.db.GetEmployees()
	if err != nil {
		return err
	}

	servicesEmpty := len(services) == 0
	employeesEmpty := len(employees) == 0

	if servicesEmpty && employeesEmpty {
		_, _, err := cfg.seedDatabase()
		return err
	}

	if !servicesEmpty && !employeesEmpty {
		return nil
	}

	return fmt.Errorf("database is partially seeded (services: %d, employees: %d); run test reset-and-seed endpoint or reset manually", len(services), len(employees))
}
