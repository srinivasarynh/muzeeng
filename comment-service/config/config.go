package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	DBName       string
	SSLMode      string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  time.Duration
}

// LoadDatabaseConfig loads database configuration from environment variables
func LoadDatabaseConfig(prefix string) (*DatabaseConfig, error) {
	cfg := &DatabaseConfig{
		Host:         getEnv(prefix+"DB_HOST", "postgres"),
		User:         getEnv(prefix+"DB_USER", "postgres"),
		Password:     getEnv(prefix+"DB_PASSWORD", "postgres"),
		DBName:       getEnv(prefix+"DB_NAME", "comment_service_db"),
		SSLMode:      getEnv(prefix+"DB_SSLMODE", "disable"),
		MaxOpenConns: getEnvAsInt(prefix+"DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns: getEnvAsInt(prefix+"DB_MAX_IDLE_CONNS", 5),
		MaxLifetime:  getEnvAsDuration(prefix+"DB_MAX_LIFETIME", 5*time.Minute),
	}

	var err error
	cfg.Port, err = strconv.Atoi(getEnv(prefix+"DB_PORT", "5432"))
	if err != nil {
		return nil, fmt.Errorf("invalid database port: %w", err)
	}

	if cfg.DBName == "" {
		return nil, fmt.Errorf("database name is required (set %sDB_NAME)", prefix)
	}

	return cfg, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvAsInt gets an environment variable as int or returns a default value
func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

// getEnvAsDuration gets an environment variable as duration or returns a default value
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
