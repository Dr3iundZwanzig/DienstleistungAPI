package database

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type Client struct {
	db *sql.DB
}

func NewClient(pathToDB string) (Client, error) {
	db, err := sql.Open("sqlite3", pathToDB)
	if err != nil {
		return Client{}, err
	}
	if _, err = db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return Client{}, err
	}
	c := Client{db}
	err = c.autoMigrate()
	if err != nil {
		return Client{}, err
	}
	return c, nil

}

func (c *Client) autoMigrate() error {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		password TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL
	);
	`
	_, err := c.db.Exec(userTable)
	if err != nil {
		return err
	}
	refreshTokenTable := `
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		token TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		revoked_at TIMESTAMP,
		user_id TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = c.db.Exec(refreshTokenTable)
	if err != nil {
		return err
	}

	employeeTable := `
	CREATE TABLE IF NOT EXISTS employees (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		name TEXT NOT NULL,
		title TEXT NOT NULL,
		specialties TEXT NOT NULL, -- wird als JSON array, e.g. ["Haarschnitt","Farbbehandlung"] gespeichert kann später auch besser normalisiert werden
		is_active BOOLEAN NOT NULL DEFAULT 1
	);
	`
	_, err = c.db.Exec(employeeTable)
	if err != nil {
		return err
	}

	availabilityTable := `
	CREATE TABLE IF NOT EXISTS availability (
		employee_id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		data TEXT NOT NULL,
		FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE
	);
	`
	_, err = c.db.Exec(availabilityTable)
	if err != nil {
		return err
	}

	appointmentTable := `
	CREATE TABLE IF NOT EXISTS appointments (
		id TEXT PRIMARY KEY,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		date TEXT NOT NULL,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		employee_name TEXT NOT NULL,
		employee_id TEXT,
		user_id TEXT NOT NULL,
		services TEXT NOT NULL,
		total_duration_minutes INTEGER DEFAULT 0,
		total_price REAL DEFAULT 0.0,
		FOREIGN KEY(employee_id) REFERENCES employees(id) ON DELETE CASCADE,
		FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
	);
	`
	_, err = c.db.Exec(appointmentTable)
	if err != nil {
		return err
	}
	return nil
}

func (c Client) Reset() error {
	if _, err := c.db.Exec("DELETE FROM appointments"); err != nil {
		return fmt.Errorf("failed to reset table appointments: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM availability"); err != nil {
		return fmt.Errorf("failed to reset table availability: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM refresh_tokens"); err != nil {
		return fmt.Errorf("failed to reset table refresh_tokens: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM employees"); err != nil {
		return fmt.Errorf("failed to reset table employees: %w", err)
	}
	if _, err := c.db.Exec("DELETE FROM users"); err != nil {
		return fmt.Errorf("failed to reset table users: %w", err)
	}
	return nil
}

func (c Client) Close() error {
	return c.db.Close()
}
