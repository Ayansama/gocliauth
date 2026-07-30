package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	DBPath           string
	LockoutThreshold int
	LockoutDuration  time.Duration
	SessionTimeout   time.Duration
}

// LoadConfig loads configuration from environment variables with fallback defaults.
func LoadConfig() *Config {
	return &Config{
		DBPath:   getEnv("DB_PATH", "app.db"),
		LockoutThreshold: getEnvAsInt("LOCKOUT_THRESHOLD", 5),
		LockoutDuration: time.Duration(getEnvAsInt("LOCKOUT_DURATION_MINS", 15))* time.Minute,
		SessionTimeout:   time.Duration(getEnvAsInt("SESSION_TIMEOUT_MINS", 15)) * time.Minute,
	}
}

func getEnv(key, defaultValue string) string{
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valStr := getEnv(key, "")
	if valStr == ""{
		return defaultValue
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return defaultValue
	}
	return val
}