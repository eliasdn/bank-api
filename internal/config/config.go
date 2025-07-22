package config

import (
	"log"
	"os"
	"strconv"
	"time"
)

type AppConfig struct {
	// Server configuration
	Server struct {
		Port         string
		ReadTimeout  time.Duration
		WriteTimeout time.Duration
		IdleTimeout  time.Duration
	}

	// Database configuration
	Database struct {
		Driver          string
		Host            string
		Port            string
		User            string
		Password        string
		Name            string
		SSLMode         string
		MaxOpenConns    int
		MaxIdleConns    int
		ConnMaxLifetime time.Duration
		ConnMaxIdleTime time.Duration
	}

	// JWT configuration
	JWT struct {
		Secret     string
		Expiration time.Duration
	}

	// Security configuration
	Security struct {
		BcryptCost int
	}

	// Rate limiting
	RateLimit struct {
		RequestsPerSecond int
		BurstSize         int
	}

	// Logging
	Logging struct {
		Level string
	}

	// Environment
	Environment string
}

func LoadConfig() *AppConfig {
	cfg := &AppConfig{}

	// Environment
	cfg.Environment = getEnv("ENVIRONMENT", "development")

	// Server configuration
	cfg.Server.Port = getEnv("PORT", "8080")
	cfg.Server.ReadTimeout = getDuration("SERVER_READ_TIMEOUT", 15*time.Second)
	cfg.Server.WriteTimeout = getDuration("SERVER_WRITE_TIMEOUT", 15*time.Second)
	cfg.Server.IdleTimeout = getDuration("SERVER_IDLE_TIMEOUT", 60*time.Second)

	// Database configuration
	cfg.Database.Driver = getEnv("DB_DRIVER", "sqlite")
	cfg.Database.Host = getEnv("DB_HOST", "localhost")
	cfg.Database.Port = getEnv("DB_PORT", "5432")
	cfg.Database.User = getEnv("DB_USER", "postgres")
	cfg.Database.Password = getEnv("DB_PASSWORD", "postgres")
	cfg.Database.Name = getEnv("DB_NAME", "bank")
	cfg.Database.SSLMode = getEnv("DB_SSL_MODE", "disable")
	cfg.Database.MaxOpenConns = getInt("DB_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = getInt("DB_MAX_IDLE_CONNS", 5)
	cfg.Database.ConnMaxLifetime = getDuration("DB_CONN_MAX_LIFETIME", 3600*time.Second)
	cfg.Database.ConnMaxIdleTime = getDuration("DB_CONN_MAX_IDLE_TIME", 300*time.Second)

	// JWT configuration
	cfg.JWT.Secret = getEnv("JWT_SECRET", "your-secret-key-change-this-in-production")
	cfg.JWT.Expiration = getDuration("JWT_EXPIRATION", 24*time.Hour)

	// Security configuration
	cfg.Security.BcryptCost = getInt("BCRYPT_COST", 10)

	// Rate limiting
	cfg.RateLimit.RequestsPerSecond = getInt("RATE_LIMIT_RPS", 100)
	cfg.RateLimit.BurstSize = getInt("RATE_LIMIT_BURST", 200)

	// Logging
	cfg.Logging.Level = getEnv("LOG_LEVEL", "info")

	// Validate required configuration
	if cfg.JWT.Secret == "your-secret-key-change-this-in-production" && cfg.Environment == "production" {
		log.Fatal("JWT_SECRET must be set in production environment")
	}

	return cfg
}

func LoadTestConfig() *AppConfig {
	cfg := &AppConfig{}

	cfg.Environment = "test"
	cfg.Server.Port = "8080"
	cfg.Server.ReadTimeout = 5 * time.Second
	cfg.Server.WriteTimeout = 5 * time.Second
	cfg.Server.IdleTimeout = 5 * time.Second

	cfg.Database.Driver = "sqlite"
	cfg.Database.Host = "localhost"
	cfg.Database.Port = "5432"
	cfg.Database.User = "test"
	cfg.Database.Password = "test"
	cfg.Database.Name = "bank_test"
	cfg.Database.SSLMode = "disable"

	cfg.JWT.Secret = "test-secret"
	cfg.JWT.Expiration = 1 * time.Hour

	cfg.Security.BcryptCost = 4 // Lower for tests

	cfg.RateLimit.RequestsPerSecond = 1000
	cfg.RateLimit.BurstSize = 2000

	cfg.Logging.Level = "debug"

	return cfg
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func (c *AppConfig) IsDevelopment() bool {
	return c.Environment == "development"
}

func (c *AppConfig) IsProduction() bool {
	return c.Environment == "production"
}

func (c *AppConfig) IsTest() bool {
	return c.Environment == "test"
}
