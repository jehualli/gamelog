// Package config loads all app configuration from environment variables.
// LEARN: Cloud-native apps never hardcode secrets in source code.
// Instead they read config from the environment — this works the same
// whether running locally (via .env file) or deployed to AWS/GCP/Heroku.
package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

// App is the global config instance. All packages import this.
var App Config

// Config holds every setting the app needs to run.
type Config struct {
	JWTSecret           string // signs JWT tokens — keep secret
	CloudinaryCloudName string
	CloudinaryAPIKey    string
	CloudinaryAPISecret string
	DBPath              string // SQLite file path
	Port                string
}

// Load reads .env then environment variables, populating App.
func Load() {
	// godotenv.Load() reads key=value pairs from .env and calls os.Setenv for each.
	// It fails silently if .env doesn't exist — in production you'd set env vars directly.
	if err := godotenv.Load(); err != nil {
		log.Println("config: no .env file found, reading from environment")
	}

	App = Config{
		JWTSecret:           getEnv("JWT_SECRET", "dev-secret-change-in-production"),
		CloudinaryCloudName: getEnv("CLOUDINARY_CLOUD_NAME", ""),
		CloudinaryAPIKey:    getEnv("CLOUDINARY_API_KEY", ""),
		CloudinaryAPISecret: getEnv("CLOUDINARY_API_SECRET", ""),
		DBPath:              getEnv("DB_PATH", "gamelog.db"),
		Port:                getEnv("PORT", "8080"),
	}

	log.Printf("config: loaded (port=%s, db=%s, cloudinary=%v)",
		App.Port, App.DBPath, App.CloudinaryCloudName != "")
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
