package config

import (
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://neondb_owner:npg_kGErW7FMByH2@ep-muddy-poetry-adb64k0i-pooler.c-2.us-east-1.aws.neon.tech/neondb?sslmode=require&channel_binding=require"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
