package config

import (
	"log"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Port                 int           `mapstructure:"PORT"`
	DBHost               string        `mapstructure:"DB_HOST"`
	DBUser               string        `mapstructure:"DB_USER"`
	DBPassword           string        `mapstructure:"DB_PASSWORD"`
	DBName               string        `mapstructure:"DB_NAME"`
	DBPort               int           `mapstructure:"DB_PORT"`
	JWTSecret            string        `mapstructure:"JWT_SECRET"`
	JWTExpire            string        `mapstructure:"JWT_EXPIRE"`
	FrontendURL          string        `mapstructure:"FRONTEND_URL"`
	S3Endpoint           string        `mapstructure:"S3_URL"`
	BucketName           string        `mapstructure:"BUCKET_NAME"`
	AwsAccessKey         string        `mapstructure:"AWS_ACCESS_KEY"`
	AwsSecretKey         string        `mapstructure:"AWS_SECRET_KEY"`
	AwsRegion            string        `mapstructure:"AWS_REGION"`
	StripeSecretKey      string        `mapstructure:"STRIPE_SECRET_KEY"`
	StripeWebhookSecret  string        `mapstructure:"STRIPE_WEBHOOK_SECRET"`
	StripeSessionExpire  time.Duration `mapstructure:"STRIPE_SESSION_EXPIRE"`
	SmtpHost             string        `mapstructure:"SMTP_HOST"`
	SmtpPort             int           `mapstructure:"SMTP_PORT"`
	SmtpUser             string        `mapstructure:"SMTP_USER"`
	SmtpPassword         string        `mapstructure:"SMTP_PASSWORD"`
	SmtpFrom             string        `mapstructure:"SMTP_FROM"`
	RateLimitLimit       int           `mapstructure:"RATE_LIMIT_LIMIT"`
	RateLimitPeriodSec   int           `mapstructure:"RATE_LIMIT_PERIOD_SEC"`
	RedisHost            string        `mapstructure:"REDIS_HOST"`
	RedisPort            string        `mapstructure:"REDIS_PORT"`
	RedisPassword        string        `mapstructure:"REDIS_PASSWORD"`
	DBMaxOpenConns       int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	DBMaxIdleConns       int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	DBConnMaxLifetimeMin int           `mapstructure:"DB_CONN_MAX_LIFETIME_MIN"`
	DBConnMaxIdleTimeMin int           `mapstructure:"DB_CONN_MAX_IDLE_TIME_MIN"`
}

var GlobalConfig *Config

func LoadConfig() {
	// Setup Viper to read the .env file
	viper.SetConfigFile(".env")
	viper.AutomaticEnv() // Read from system environment variables if they exist

	// Set Default Values (Used if not found in .env)
	viper.SetDefault("PORT", 5000)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_USER", "admin")
	viper.SetDefault("DB_PASSWORD", "password")
	viper.SetDefault("DB_NAME", "lorem")
	viper.SetDefault("DB_PORT", 5433)
	viper.SetDefault("JWT_EXPIRE", "24h")
	viper.SetDefault("FRONTEND_URL", "http://localhost:3000")
	viper.SetDefault("AWS_REGION", "ap-southeast-1")
	viper.SetDefault("STRIPE_SESSION_EXPIRE", "30m")
	viper.SetDefault("RATE_LIMIT_LIMIT", 5)
	viper.SetDefault("RATE_LIMIT_PERIOD_SEC", 60)
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", "6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("DB_MAX_OPEN_CONNS", 50)
	viper.SetDefault("DB_MAX_IDLE_CONNS", 25)
	viper.SetDefault("DB_CONN_MAX_LIFETIME_MIN", 30)
	viper.SetDefault("DB_CONN_MAX_IDLE_TIME_MIN", 5)

	// Set empty defaults for required secrets so Viper unmarshals them from ENV
	viper.SetDefault("JWT_SECRET", "")
	viper.SetDefault("S3_URL", "")
	viper.SetDefault("BUCKET_NAME", "")
	viper.SetDefault("AWS_ACCESS_KEY", "")
	viper.SetDefault("AWS_SECRET_KEY", "")
	viper.SetDefault("STRIPE_SECRET_KEY", "")
	viper.SetDefault("STRIPE_WEBHOOK_SECRET", "")
	viper.SetDefault("SMTP_HOST", "")
	viper.SetDefault("SMTP_PORT", 0)
	viper.SetDefault("SMTP_USER", "")
	viper.SetDefault("SMTP_PASSWORD", "")
	viper.SetDefault("SMTP_FROM", "")

	// Read the file
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Warning: No .env file found, using defaults and system env")
	}

	// Unmarshal the config into our struct
	err := viper.Unmarshal(&GlobalConfig)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	log.Println("Config loaded successfully")
}
