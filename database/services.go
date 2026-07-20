package database

import (
	"database/sql"
	"fmt"
	"strings"
)

type ServiceNode struct {
	ID              string        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	DurationMinutes int           `json:"duration_minutes,omitempty"`
	Price           float64       `json:"price,omitempty"`
	Currency        string        `json:"currency,omitempty"`
	IsActive        bool          `json:"is_active"`
	Children        []ServiceNode `json:"children,omitempty"`
}

type ServiceSelection struct {
	ID              string
	Name            string
	DurationMinutes int
	Price           float64
	Currency        string
}

func (c Client) SeedServicesIfEmpty(nodes []ServiceNode) error {
	var count int
	if err := c.db.QueryRow(`SELECT COUNT(1) FROM services`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return c.ReplaceServicesTree(nodes)
}

func (c Client) ReplaceServicesTree(nodes []ServiceNode) error {
	tx, err := c.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM services`); err != nil {
		return err
	}

	insertQuery := `
		INSERT INTO services
			(id, parent_id, name, description, duration_minutes, price, currency, is_active, sort_order)
		VALUES
			(?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	var insertNode func(parentID *string, node ServiceNode, sortOrder int) error
	insertNode = func(parentID *string, node ServiceNode, sortOrder int) error {
		if strings.TrimSpace(node.ID) == "" {
			return fmt.Errorf("service id is required")
		}
		if strings.TrimSpace(node.Name) == "" {
			return fmt.Errorf("service name is required for id %s", node.ID)
		}

		var dbParentID any = nil
		if parentID != nil {
			dbParentID = *parentID
		}

		var duration any = nil
		var price any = nil
		var currency any = nil
		if len(node.Children) == 0 {
			duration = node.DurationMinutes
			price = node.Price
			currency = node.Currency
		}

		_, err := tx.Exec(insertQuery,
			node.ID,
			dbParentID,
			node.Name,
			node.Description,
			duration,
			price,
			currency,
			node.IsActive,
			sortOrder,
		)
		if err != nil {
			return err
		}

		for idx, child := range node.Children {
			parent := node.ID
			if err := insertNode(&parent, child, idx); err != nil {
				return err
			}
		}
		return nil
	}

	for idx, node := range nodes {
		if err := insertNode(nil, node, idx); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (c Client) GetServicesTree() ([]ServiceNode, error) {
	rows, err := c.db.Query(`
		SELECT id, parent_id, name, description, duration_minutes, price, currency, is_active
		FROM services
		ORDER BY COALESCE(parent_id, ''), sort_order, name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodesByID := make(map[string]*ServiceNode)
	childrenByParent := make(map[string][]*ServiceNode)
	roots := make([]*ServiceNode, 0)

	for rows.Next() {
		var node ServiceNode
		var parentID sql.NullString
		var description sql.NullString
		var duration sql.NullInt64
		var price sql.NullFloat64
		var currency sql.NullString

		if err := rows.Scan(&node.ID, &parentID, &node.Name, &description, &duration, &price, &currency, &node.IsActive); err != nil {
			return nil, err
		}

		if description.Valid {
			node.Description = description.String
		}
		if duration.Valid {
			node.DurationMinutes = int(duration.Int64)
		}
		if price.Valid {
			node.Price = price.Float64
		}
		if currency.Valid {
			node.Currency = currency.String
		}

		nodeCopy := node
		nodesByID[node.ID] = &nodeCopy
		if parentID.Valid {
			childrenByParent[parentID.String] = append(childrenByParent[parentID.String], &nodeCopy)
		} else {
			roots = append(roots, &nodeCopy)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var attachChildren func(node *ServiceNode)
	attachChildren = func(node *ServiceNode) {
		children := childrenByParent[node.ID]
		if len(children) == 0 {
			return
		}
		node.Children = make([]ServiceNode, 0, len(children))
		for _, child := range children {
			attachChildren(child)
			node.Children = append(node.Children, *child)
		}
	}

	result := make([]ServiceNode, 0, len(roots))
	for _, root := range roots {
		if _, ok := nodesByID[root.ID]; !ok {
			continue
		}
		attachChildren(root)
		result = append(result, *root)
	}

	return result, nil
}

func (c Client) GetActiveLeafServicesByIDs(serviceIDs []string) ([]ServiceSelection, error) {
	orderedIDs := uniqueNonEmpty(serviceIDs)
	if len(orderedIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(orderedIDs)), ",")
	args := make([]any, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		args = append(args, id)
	}

	query := `
		SELECT
			s.id,
			s.name,
			COALESCE(s.duration_minutes, 0),
			COALESCE(s.price, 0),
			COALESCE(s.currency, ''),
			s.is_active,
			(SELECT COUNT(1) FROM services c WHERE c.parent_id = s.id) AS child_count
		FROM services s
		WHERE s.id IN (` + placeholders + `)
	`

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type record struct {
		selection  ServiceSelection
		isActive   bool
		childCount int
	}
	byID := make(map[string]record)

	for rows.Next() {
		var rec record
		if err := rows.Scan(
			&rec.selection.ID,
			&rec.selection.Name,
			&rec.selection.DurationMinutes,
			&rec.selection.Price,
			&rec.selection.Currency,
			&rec.isActive,
			&rec.childCount,
		); err != nil {
			return nil, err
		}
		byID[rec.selection.ID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	selected := make([]ServiceSelection, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		rec, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("service %s not found", id)
		}
		if !rec.isActive {
			return nil, fmt.Errorf("service %s is not active", id)
		}
		if rec.childCount > 0 {
			return nil, fmt.Errorf("service %s is not bookable", id)
		}
		selected = append(selected, rec.selection)
	}

	return selected, nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
