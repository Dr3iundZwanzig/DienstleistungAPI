package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Employee struct {
	ID          string    `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Name        string    `json:"name"`
	Title       string    `json:"title"`
	Specialties []string  `json:"specialties"`
	IsActive    bool      `json:"is_active"`
}

type CreateEmployeeParams struct {
	ID          string
	Name        string
	Title       string
	Specialties []string
	IsActive    bool
}

func (c Client) GetEmployees() ([]Employee, error) {
	query := `
		SELECT
			id,
			created_at,
			updated_at,
			name,
			title,
			specialties,
			is_active
		FROM employees
		ORDER BY name
	`

	rows, err := c.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	employees := []Employee{}
	for rows.Next() {
		var employee Employee
		var specialtiesJSON string
		if err := rows.Scan(&employee.ID, &employee.CreatedAt, &employee.UpdatedAt, &employee.Name, &employee.Title, &specialtiesJSON, &employee.IsActive); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(specialtiesJSON), &employee.Specialties); err != nil {
			return nil, err
		}
		employees = append(employees, employee)
	}

	return employees, nil
}

func (c Client) CreateEmployee(params CreateEmployeeParams) (*Employee, error) {
	specialtiesJSON, err := json.Marshal(params.Specialties)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO employees
			(id, created_at, updated_at, name, title, specialties, is_active)
		VALUES
			(?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?, ?)
	`
	_, err = c.db.Exec(query, params.ID, params.Name, params.Title, string(specialtiesJSON), params.IsActive)
	if err != nil {
		return nil, err
	}

	return c.GetEmployee(params.ID)
}

func (c Client) GetEmployee(id string) (*Employee, error) {
	query := `
		SELECT id, created_at, updated_at, name, title, specialties, is_active
		FROM employees
		WHERE id = ?
	`
	var employee Employee
	var specialtiesJSON string
	err := c.db.QueryRow(query, id).Scan(&employee.ID, &employee.CreatedAt, &employee.UpdatedAt, &employee.Name, &employee.Title, &specialtiesJSON, &employee.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(specialtiesJSON), &employee.Specialties); err != nil {
		return nil, err
	}
	return &employee, nil
}

func (c Client) ResolveEmployeeForServices(serviceNames []string) (string, error) {
	employees, err := c.GetEmployees()
	if err != nil {
		return "", err
	}

	selectedServiceNames := make([]string, 0, len(serviceNames))
	for _, name := range serviceNames {
		trimmed := strings.ToLower(strings.TrimSpace(name))
		if trimmed != "" {
			selectedServiceNames = append(selectedServiceNames, trimmed)
		}
	}
	if len(selectedServiceNames) == 0 {
		return "", nil
	}

	var bestEmployee *Employee
	bestScore := -1

	for _, employee := range employees {
		if !employee.IsActive {
			continue
		}

		var score int
		for _, serviceName := range selectedServiceNames {
			for _, specialty := range employee.Specialties {
				trimmedSpecialty := strings.ToLower(strings.TrimSpace(specialty))
				if trimmedSpecialty == "" {
					continue
				}
				if strings.Contains(trimmedSpecialty, serviceName) || strings.Contains(serviceName, trimmedSpecialty) {
					score += 2
					break
				}
			}
		}

		if score > bestScore {
			bestScore = score
			bestEmployee = &employee
		}
	}

	if bestEmployee == nil {
		return "", nil
	}

	return bestEmployee.ID, nil
}
