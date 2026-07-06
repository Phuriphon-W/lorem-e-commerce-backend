package utils

import (
	"strings"
)

// SanitizeOrder ensures that the provided order string is safe to use in an ORDER BY clause.
// It checks if the column is in the allowedColumns list and enforces ASC/DESC direction.
// If the input is invalid or empty, it returns the defaultOrder.
func SanitizeOrder(order string, allowedColumns []string, defaultOrder string) string {
	if order == "" {
		return defaultOrder
	}

	clauses := strings.Split(order, ",")
	var sanitizedClauses []string

	for _, clause := range clauses {
		clause = strings.TrimSpace(clause)
		if clause == "" {
			continue
		}

		parts := strings.Fields(clause)
		if len(parts) == 0 {
			return defaultOrder
		}

		column := strings.ToLower(parts[0])
		direction := "ASC"

		if len(parts) > 1 {
			dir := strings.ToUpper(parts[1])
			if dir == "DESC" || dir == "ASC" {
				direction = dir
			} else {
				// If direction is invalid, fallback to default order
				return defaultOrder
			}
		}

		if len(parts) > 2 {
			// If there are more parts, it is invalid (e.g. SQL injection attempts)
			return defaultOrder
		}

		// Check if column is allowed
		isAllowed := false
		for _, col := range allowedColumns {
			if column == col {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return defaultOrder
		}

		sanitizedClauses = append(sanitizedClauses, column+" "+direction)
	}

	if len(sanitizedClauses) == 0 {
		return defaultOrder
	}

	return strings.Join(sanitizedClauses, ", ")
}
