package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type ServerConfig struct {
	Port         string
	Environment  string
	AllowOrigins string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
	MaxConns int
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret     string
	RefreshSecret    string
	AccessExpiresIn  time.Duration
	RefreshExpiresIn time.Duration
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			Environment:  getEnv("ENVIRONMENT", "development"),
			AllowOrigins: getEnv("ALLOW_ORIGINS", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "edusaas"),
			Password: getEnv("DB_PASSWORD", "edusaas_dev"),
			DBName:   getEnv("DB_NAME", "edusaas"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
			MaxConns: getEnvInt("DB_MAX_CONNS", 20),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:     getEnv("JWT_ACCESS_SECRET", "change-me-access-secret-dev-only"),
			RefreshSecret:    getEnv("JWT_REFRESH_SECRET", "change-me-refresh-secret-dev-only"),
			AccessExpiresIn:  time.Duration(getEnvInt("JWT_ACCESS_EXPIRES_MINUTES", 15)) * time.Minute,
			RefreshExpiresIn: time.Duration(getEnvInt("JWT_REFRESH_EXPIRES_DAYS", 7)) * 24 * time.Hour,
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}
