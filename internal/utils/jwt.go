package utils

import (
	"fmt"
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

func VerifyJWT(tokenString string, secret string) (jwt.MapClaims, error) {
	// Parse and verify token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, err
	}

	// Type insertion and check token valid
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}
