package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	CloudinaryURL  string
	JWTSecret      string
	ServerPort     string
	Environment    string
}

var AppConfig *Config

func Load() error {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		// .env file is optional, continue without it
	}

	AppConfig = &Config{
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://khalil:44441318@127.0.0.1/fmbq?sslmode=disable"),
		CloudinaryURL: getEnv("CLOUDINARY_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		ServerPort:    getEnv("PORT", "8080"),
		Environment:   getEnv("ENVIRONMENT", "development"),
	}

	// Debug: Print the database URL being used
	println("Using DATABASE_URL:", AppConfig.DatabaseURL)

	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
