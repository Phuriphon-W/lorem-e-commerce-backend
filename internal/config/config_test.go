package config

import (
	"os"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_SMTPEnvVars(t *testing.T) {
	// Reset Viper to ensure a clean state
	viper.Reset()

	// Backup existing env vars and restore them later
	backupEnv := func(key string) {
		val, exists := os.LookupEnv(key)
		t.Cleanup(func() {
			if exists {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		})
	}

	backupEnv("SMTP_HOST")
	backupEnv("SMTP_PORT")
	backupEnv("SMTP_USER")
	backupEnv("SMTP_PASSWORD")
	backupEnv("SMTP_FROM")

	// Set test environment variables
	t.Setenv("SMTP_HOST", "smtp.test.com")
	t.Setenv("SMTP_PORT", "25")
	t.Setenv("SMTP_USER", "testuser")
	t.Setenv("SMTP_PASSWORD", "testpass")
	t.Setenv("SMTP_FROM", "test@test.com")

	// Temporarily rename .env if it exists so we force loading from env
	if _, err := os.Stat(".env"); err == nil {
		err = os.Rename(".env", ".env.tmp")
		assert.NoError(t, err)
		t.Cleanup(func() {
			os.Rename(".env.tmp", ".env")
		})
	}

	// Load configuration
	LoadConfig()

	// Assert SMTP config values
	assert.Equal(t, "smtp.test.com", GlobalConfig.SmtpHost, "SMTP_HOST should be loaded from env")
	assert.Equal(t, 25, GlobalConfig.SmtpPort, "SMTP_PORT should be loaded from env")
	assert.Equal(t, "testuser", GlobalConfig.SmtpUser, "SMTP_USER should be loaded from env")
	assert.Equal(t, "testpass", GlobalConfig.SmtpPassword, "SMTP_PASSWORD should be loaded from env")
	assert.Equal(t, "test@test.com", GlobalConfig.SmtpFrom, "SMTP_FROM should be loaded from env")
}
