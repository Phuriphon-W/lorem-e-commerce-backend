package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGenerateAndVerifyJWT_Success(t *testing.T) {
	userId := uuid.New()
	secret := "supersecret"
	duration := time.Hour

	// Generate
	tokenString, err := GenerateJWT(userId, secret, duration)
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// Verify
	claims, err := VerifyJWT(tokenString, secret)
	assert.NoError(t, err)
	assert.NotNil(t, claims)

	// Check claims
	assert.Equal(t, userId.String(), claims["id"])
	assert.WithinDuration(t, time.Now().Add(duration), time.Unix(int64(claims["exp"].(float64)), 0), time.Minute)
}

func TestVerifyJWT_InvalidToken(t *testing.T) {
	_, err := VerifyJWT("invalid.token.string", "supersecret")
	assert.Error(t, err)
}

func TestVerifyJWT_WrongSecret(t *testing.T) {
	userId := uuid.New()
	duration := time.Hour

	tokenString, _ := GenerateJWT(userId, "rightsecret", duration)

	_, err := VerifyJWT(tokenString, "wrongsecret")
	assert.Error(t, err)
	assert.ErrorIs(t, err, jwt.ErrSignatureInvalid)
}

func TestVerifyJWT_ExpiredToken(t *testing.T) {
	userId := uuid.New()
	duration := -1 * time.Hour // Expired

	tokenString, _ := GenerateJWT(userId, "supersecret", duration)

	_, err := VerifyJWT(tokenString, "supersecret")
	assert.Error(t, err)
	assert.ErrorIs(t, err, jwt.ErrTokenExpired)
}
