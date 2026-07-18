package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

type Appointment struct {
	ID                   string    `json:"id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	Date                 string    `json:"date"`
	StartTime            string    `json:"start_time"`
	EndTime              string    `json:"end_time"`
	EmployeeName         string    `json:"employee_name"`
	EmployeeID           string    `json:"employee_id,omitempty"`
	UserID               string    `json:"user_id"`
	Services             []string  `json:"services"`
	TotalDurationMinutes int       `json:"total_duration_minutes"`
	TotalPrice           float64   `json:"total_price"`
}

type CreateAppointmentParams struct {
	Date                 string
	StartTime            string
	EndTime              string
	EmployeeName         string
	EmployeeID           string
	UserID               string
	Services             []string
	TotalDurationMinutes int
	TotalPrice           float64
}

func (c Client) CreateAppointment(params CreateAppointmentParams) (*Appointment, error) {
	servicesJSON, err := json.Marshal(params.Services)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO appointments
			(id, created_at, updated_at, date, start_time, end_time, employee_name, employee_id, user_id, services, total_duration_minutes, total_price)
		VALUES
			(?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	id := NewID()
	_, err = c.db.Exec(query, id, params.Date, params.StartTime, params.EndTime, params.EmployeeName, params.EmployeeID, params.UserID, string(servicesJSON), params.TotalDurationMinutes, params.TotalPrice)
	if err != nil {
		return nil, err
	}

	if params.EmployeeID != "" {
		if err := c.CloseAvailabilitySlot(params.EmployeeID, params.Date, params.StartTime, params.EndTime); err != nil {
			return nil, err
		}
	}

	return c.GetAppointment(id)
}

func (c Client) GetAppointment(id string) (*Appointment, error) {
	query := `
		SELECT id, created_at, updated_at, date, start_time, end_time, employee_name, employee_id, user_id, services, total_duration_minutes, total_price
		FROM appointments
		WHERE id = ?
	`
	var appointment Appointment
	var servicesJSON string
	err := c.db.QueryRow(query, id).Scan(&appointment.ID, &appointment.CreatedAt, &appointment.UpdatedAt, &appointment.Date, &appointment.StartTime, &appointment.EndTime, &appointment.EmployeeName, &appointment.EmployeeID, &appointment.UserID, &servicesJSON, &appointment.TotalDurationMinutes, &appointment.TotalPrice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(servicesJSON), &appointment.Services); err != nil {
		return nil, err
	}
	return &appointment, nil
}

func (c Client) GetAppointmentsByUserID(userID string) ([]Appointment, error) {
	query := `
		SELECT id, created_at, updated_at, date, start_time, end_time, employee_name, employee_id, user_id, services, total_duration_minutes, total_price
		FROM appointments
		WHERE user_id = ?
		ORDER BY date ASC, start_time ASC, created_at ASC
	`

	rows, err := c.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appointments := []Appointment{}
	for rows.Next() {
		var appointment Appointment
		var servicesJSON string
		if err := rows.Scan(&appointment.ID, &appointment.CreatedAt, &appointment.UpdatedAt, &appointment.Date, &appointment.StartTime, &appointment.EndTime, &appointment.EmployeeName, &appointment.EmployeeID, &appointment.UserID, &servicesJSON, &appointment.TotalDurationMinutes, &appointment.TotalPrice); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(servicesJSON), &appointment.Services); err != nil {
			return nil, err
		}
		appointments = append(appointments, appointment)
	}

	return appointments, rows.Err()
}

func (c Client) CancelAppointmentByIDAndUserID(appointmentID, userID string) (*Appointment, error) {
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	appointment, err := getAppointmentByIDAndUserID(tx, appointmentID, userID)
	if err != nil {
		return nil, err
	}
	if appointment == nil {
		return nil, nil
	}

	if appointment.EmployeeID != "" {
		if err := updateAvailabilitySlot(tx, appointment.EmployeeID, appointment.Date, appointment.StartTime, appointment.EndTime, true); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(`DELETE FROM appointments WHERE id = ? AND user_id = ?`, appointmentID, userID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return appointment, nil
}

func (c Client) CancelAllAppointmentsFromUser(userID string) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	apointments, err := c.GetAppointmentsByUserID(userID)
	if err != nil {
		return err
	}
	for _, appointment := range apointments {
		if appointment.EmployeeID != "" {
			if err := updateAvailabilitySlot(tx, appointment.EmployeeID, appointment.Date, appointment.StartTime, appointment.EndTime, true); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DELETE FROM appointments WHERE id = ? AND user_id = ?`, appointment.ID, userID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func getAppointmentByIDAndUserID(queryRow interface {
	QueryRow(string, ...interface{}) *sql.Row
}, appointmentID, userID string) (*Appointment, error) {
	query := `
		SELECT id, created_at, updated_at, date, start_time, end_time, employee_name, employee_id, user_id, services, total_duration_minutes, total_price
		FROM appointments
		WHERE id = ? AND user_id = ?
	`

	var appointment Appointment
	var servicesJSON string
	err := queryRow.QueryRow(query, appointmentID, userID).Scan(&appointment.ID, &appointment.CreatedAt, &appointment.UpdatedAt, &appointment.Date, &appointment.StartTime, &appointment.EndTime, &appointment.EmployeeName, &appointment.EmployeeID, &appointment.UserID, &servicesJSON, &appointment.TotalDurationMinutes, &appointment.TotalPrice)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if err := json.Unmarshal([]byte(servicesJSON), &appointment.Services); err != nil {
		return nil, err
	}

	return &appointment, nil
}

func NewID() string {
	now := time.Now().UTC()
	return fmt.Sprintf("%s-%d", now.Format("20060102150405"), now.Nanosecond())
}
