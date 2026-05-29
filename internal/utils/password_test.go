package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHashAndVerifyPassword(t *testing.T) {
	password := "mysecurepassword"

	// Hash the password
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify with correct password
	isValid := VerifyPassword(password, hash)
	assert.True(t, isValid)

	// Verify with incorrect password
	isValid = VerifyPassword("wrongpassword", hash)
	assert.False(t, isValid)
}

func TestHashPassword_Empty(t *testing.T) {
	password := ""

	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)

	isValid := VerifyPassword(password, hash)
	assert.True(t, isValid)
}

func TestHashPassword_TooLong(t *testing.T) {
	// bcrypt max password length is 72 bytes
	password := ""
	for i := 0; i < 73; i++ {
		password += "a"
	}

	_, err := HashPassword(password)
	assert.Error(t, err)
}
