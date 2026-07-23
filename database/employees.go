package database

import (
	"database/sql"
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
	ServiceIDs  []string
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
	employeeIDs := make([]string, 0)
	for rows.Next() {
		var employee Employee
		if err := rows.Scan(&employee.ID, &employee.CreatedAt, &employee.UpdatedAt, &employee.Name, &employee.Title, &employee.IsActive); err != nil {
			return nil, err
		}
		employees = append(employees, employee)
		employeeIDs = append(employeeIDs, employee.ID)
	}

	specialtiesByEmployeeID, err := c.getSpecialtiesByEmployeeIDs(employeeIDs)
	if err != nil {
		return nil, err
	}

	for idx := range employees {
		employees[idx].Specialties = specialtiesByEmployeeID[employees[idx].ID]
	}

	return employees, nil
}

func (c Client) CreateEmployee(params CreateEmployeeParams) (*Employee, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO employees
			(id, created_at, updated_at, name, title, is_active)
		VALUES
			(?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?)
	`
	_, err = tx.Exec(query, params.ID, params.Name, params.Title, params.IsActive)
	if err != nil {
		return nil, err
	}

	serviceIDs, err := c.resolveEmployeeServiceIDs(params.ServiceIDs, params.Specialties)
	if err != nil {
		return nil, err
	}

	if err := linkEmployeeServicesTx(tx, params.ID, serviceIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return c.GetEmployee(params.ID)
}

func (c Client) GetEmployee(id string) (*Employee, error) {
	query := `
		SELECT id, created_at, updated_at, name, title, is_active
		FROM employees
		WHERE id = ?
	`
	var employee Employee
	err := c.db.QueryRow(query, id).Scan(&employee.ID, &employee.CreatedAt, &employee.UpdatedAt, &employee.Name, &employee.Title, &employee.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	specialtiesByEmployeeID, err := c.getSpecialtiesByEmployeeIDs([]string{id})
	if err != nil {
		return nil, err
	}
	employee.Specialties = specialtiesByEmployeeID[id]
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

func (c Client) getSpecialtiesByEmployeeIDs(employeeIDs []string) (map[string][]string, error) {
	result := make(map[string][]string, len(employeeIDs))
	if len(employeeIDs) == 0 {
		return result, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(employeeIDs)), ",")
	args := make([]any, 0, len(employeeIDs))
	for _, id := range employeeIDs {
		args = append(args, id)
		result[id] = []string{}
	}

	query := `
		SELECT es.employee_id, s.name
		FROM employee_services es
		JOIN services s ON s.id = es.service_id
		WHERE es.employee_id IN (` + placeholders + `)
		ORDER BY s.name
	`

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var employeeID string
		var serviceName string
		if err := rows.Scan(&employeeID, &serviceName); err != nil {
			return nil, err
		}
		result[employeeID] = append(result[employeeID], serviceName)
	}

	return result, rows.Err()
}

func (c Client) resolveEmployeeServiceIDs(serviceIDs []string, serviceNames []string) ([]string, error) {
	if len(serviceIDs) > 0 {
		selected, err := c.GetActiveLeafServicesByIDs(serviceIDs)
		if err != nil {
			return nil, err
		}
		resolved := make([]string, 0, len(selected))
		for _, service := range selected {
			resolved = append(resolved, service.ID)
		}
		return resolved, nil
	}

	normalized := make([]string, 0, len(serviceNames))
	for _, name := range serviceNames {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			normalized = append(normalized, strings.ToLower(trimmed))
		}
	}
	if len(normalized) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(normalized)), ",")
	args := make([]any, 0, len(normalized))
	for _, name := range normalized {
		args = append(args, name)
	}

	query := `
		SELECT s.id
		FROM services s
		WHERE LOWER(s.name) IN (` + placeholders + `)
		AND s.is_active = 1
		AND NOT EXISTS (SELECT 1 FROM services c WHERE c.parent_id = s.id)
		ORDER BY s.id
	`

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return nil, nil
	}

	return ids, nil
}

// employee->service join rows
func linkEmployeeServicesTx(tx *sql.Tx, employeeID string, serviceIDs []string) error {
	if len(serviceIDs) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(serviceIDs))
	for _, serviceID := range serviceIDs {
		if _, ok := seen[serviceID]; ok {
			continue
		}
		seen[serviceID] = struct{}{}
		if _, err := tx.Exec(`INSERT INTO employee_services (employee_id, service_id) VALUES (?, ?)`, employeeID, serviceID); err != nil {
			return err
		}
	}

	return nil
}
