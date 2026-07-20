package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"time"
)

type Availability struct {
	EmployeeID string             `json:"employee_id"`
	CreatedAt  time.Time          `json:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Dates      []AvailabilityDate `json:"dates"`
}

type AvailabilityDate struct {
	Date  string             `json:"date"`
	Slots []AvailabilitySlot `json:"slots"`
}

type AvailabilitySlot struct {
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	IsAvailable bool   `json:"is_available"`
}

type CreateAvailabilityParams struct {
	EmployeeID string
	Dates      []AvailabilityDate
}

func (c Client) CreateAvailability(params CreateAvailabilityParams) (*Availability, error) {
	dataJSON, err := json.Marshal(params.Dates)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO availability
			(employee_id, created_at, updated_at, data)
		VALUES
			(?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(employee_id) DO UPDATE SET
			updated_at = CURRENT_TIMESTAMP,
			data = excluded.data
	`
	_, err = c.db.Exec(query, params.EmployeeID, string(dataJSON))
	if err != nil {
		return nil, err
	}

	return c.GetAvailabilityByEmployeeID(params.EmployeeID)
}

func (c Client) GetAvailabilityByEmployeeID(employeeID string) (*Availability, error) {
	query := `
		SELECT employee_id, created_at, updated_at, data
		FROM availability
		WHERE employee_id = ?
	`
	var availability Availability
	var dataJSON string
	err := c.db.QueryRow(query, employeeID).Scan(&availability.EmployeeID, &availability.CreatedAt, &availability.UpdatedAt, &dataJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(dataJSON), &availability.Dates); err != nil {
		return nil, err
	}
	return &availability, nil
}

func (c Client) IsAvailabilitySlotAvailable(employeeID, date, startTime, endTime string) (bool, error) {
	availability, err := c.GetAvailabilityByEmployeeID(employeeID)
	if err != nil {
		return false, err
	}
	if availability == nil {
		return false, nil
	}

	for _, day := range availability.Dates {
		if day.Date != date {
			continue
		}
		for _, slot := range day.Slots {
			if slot.StartTime == startTime && slot.EndTime == endTime {
				return slot.IsAvailable, nil
			}
		}
	}

	return false, nil
}

func (c Client) CloseAvailabilitySlot(employeeID, date, startTime, endTime string) error {
	return c.setAvailabilitySlotAvailability(employeeID, date, startTime, endTime, false)
}

func (c Client) OpenAvailabilitySlot(employeeID, date, startTime, endTime string) error {
	return c.setAvailabilitySlotAvailability(employeeID, date, startTime, endTime, true)
}

func (c Client) setAvailabilitySlotAvailability(employeeID, date, startTime, endTime string, isAvailable bool) error {
	availability, err := c.GetAvailabilityByEmployeeID(employeeID)
	if err != nil {
		return err
	}
	if availability == nil {
		return nil
	}

	updated := false
	for _, day := range availability.Dates {
		if day.Date != date {
			continue
		}
		for i := range day.Slots {
			if day.Slots[i].StartTime == startTime && day.Slots[i].EndTime == endTime {
				day.Slots[i].IsAvailable = isAvailable
				updated = true
			}
		}
	}

	if !updated {
		return nil
	}

	dataJSON, err := json.Marshal(availability.Dates)
	if err != nil {
		return err
	}

	query := `
		UPDATE availability
		SET updated_at = CURRENT_TIMESTAMP, data = ?
		WHERE employee_id = ?
	`
	_, err = c.db.Exec(query, string(dataJSON), employeeID)
	return err
}

func updateAvailabilitySlot(execQuerier interface {
	QueryRow(string, ...interface{}) *sql.Row
	Exec(string, ...interface{}) (sql.Result, error)
}, employeeID, date, startTime, endTime string, isAvailable bool) error {
	query := `
		SELECT data
		FROM availability
		WHERE employee_id = ?
	`

	var dataJSON string
	err := execQuerier.QueryRow(query, employeeID).Scan(&dataJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	var dates []AvailabilityDate
	if err := json.Unmarshal([]byte(dataJSON), &dates); err != nil {
		return err
	}

	updated := false
	for _, day := range dates {
		if day.Date != date {
			continue
		}
		for i := range day.Slots {
			if day.Slots[i].StartTime == startTime && day.Slots[i].EndTime == endTime {
				day.Slots[i].IsAvailable = isAvailable
				updated = true
			}
		}
	}

	if !updated {
		return nil
	}

	updatedJSON, err := json.Marshal(dates)
	if err != nil {
		return err
	}

	_, err = execQuerier.Exec(`
		UPDATE availability
		SET updated_at = CURRENT_TIMESTAMP, data = ?
		WHERE employee_id = ?
	`, string(updatedJSON), employeeID)
	return err
}
