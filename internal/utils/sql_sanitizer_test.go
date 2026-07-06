package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeOrder(t *testing.T) {
	allowed := []string{"id", "created_at", "updated_at", "username", "first_name", "last_name", "email"}
	defaultOrder := "created_at DESC"

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Empty order",
			input:    "",
			expected: defaultOrder,
		},
		{
			name:     "Valid simple asc",
			input:    "username ASC",
			expected: "username ASC",
		},
		{
			name:     "Valid simple desc",
			input:    "created_at DESC",
			expected: "created_at DESC",
		},
		{
			name:     "Valid case-insensitive column and direction",
			input:    "UserName desc",
			expected: "username DESC",
		},
		{
			name:     "Valid default direction (omitted)",
			input:    "first_name",
			expected: "first_name ASC",
		},
		{
			name:     "Valid multi-column sorting",
			input:    "first_name ASC, last_name DESC",
			expected: "first_name ASC, last_name DESC",
		},
		{
			name:     "Valid multi-column sorting with extra whitespace",
			input:    "  first_name   ASC  ,   last_name   DESC  ",
			expected: "first_name ASC, last_name DESC",
		},
		{
			name:     "Valid multi-column with some missing directions",
			input:    "first_name, last_name DESC",
			expected: "first_name ASC, last_name DESC",
		},
		{
			name:     "Invalid column name fallback",
			input:    "password ASC",
			expected: defaultOrder,
		},
		{
			name:     "Invalid direction fallback",
			input:    "first_name invalid",
			expected: defaultOrder,
		},
		{
			name:     "Invalid multi-column because one column is invalid",
			input:    "first_name ASC, password DESC",
			expected: defaultOrder,
		},
		{
			name:     "Invalid multi-column because one direction is invalid",
			input:    "first_name ASC, last_name invalid",
			expected: defaultOrder,
		},
		{
			name:     "SQL Injection prevention (extra tokens in clause)",
			input:    "first_name ASC; DROP TABLE users",
			expected: defaultOrder,
		},
		{
			name:     "SQL Injection prevention (extra tokens in multi-column)",
			input:    "first_name ASC, last_name DESC; DELETE FROM users",
			expected: defaultOrder,
		},
		{
			name:     "Trailing commas are cleaned up",
			input:    "first_name ASC, ",
			expected: "first_name ASC",
		},
		{
			name:     "Leading commas are cleaned up",
			input:    ", first_name ASC",
			expected: "first_name ASC",
		},
		{
			name:     "Only commas and spaces fallback",
			input:    " , , ",
			expected: defaultOrder,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeOrder(tc.input, allowed, defaultOrder)
			assert.Equal(t, tc.expected, got)
		})
	}
}
