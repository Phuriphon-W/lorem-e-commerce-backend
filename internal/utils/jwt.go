package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func GenerateJWT(userId uuid.UUID, secret string, duration time.Duration) (string, error) {
	// Create the Claims (Payload)
	claims := jwt.MapClaims{
		"id":  userId.String(),
		"exp": time.Now().Add(duration).Unix(),
		"iat": time.Now().Unix(), // Issued at
	}

	// Create the token using the signing method (HS256 is the standard)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with secret key
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
