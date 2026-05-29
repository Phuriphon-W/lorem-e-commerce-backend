package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSMTPEmailService(t *testing.T) {
	service := NewSMTPEmailService("localhost", 587, "user", "pass", "no-reply@example.com")
	assert.NotNil(t, service)
}

func TestSendResetPasswordEmail_ConnectionError(t *testing.T) {
	// Use an invalid port to simulate connection failure after template processing
	service := NewSMTPEmailService("127.0.0.1", 12345, "user", "pass", "no-reply@example.com")

	err := service.SendResetPasswordEmail("test@example.com", "TestUser", "http://example.com/reset")

	// We expect an error because it cannot connect to 127.0.0.1:12345
	// But it shouldn't be a template error.
	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to send email") || strings.Contains(err.Error(), "connection refused"), "Expected connection error, got: %v", err)
}

func TestSendResetPasswordEmail_WithoutAuth(t *testing.T) {
	// Test without username and password to cover the if s.user != "" branch
	service := NewSMTPEmailService("127.0.0.1", 12345, "", "", "no-reply@example.com")

	err := service.SendResetPasswordEmail("test@example.com", "TestUser", "http://example.com/reset")

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "failed to send email") || strings.Contains(err.Error(), "connection refused"), "Expected connection error, got: %v", err)
}
