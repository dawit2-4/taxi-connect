package config

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// DBConfig holds database connection parameters
  type DBConfig struct {
        Host     string
        Port     string
        User     string
        Password string
        DBName   string
        SSLMode  string
  }

  // LoadDBConfig reads configuration from environment variables with sensible defaults
  func LoadDBConfig() DBConfig {
        return DBConfig{
                Host:     getEnv("DB_HOST", "localhost"),
                Port:     getEnv("DB_PORT", "5433"),
                User:     getEnv("DB_USER", "postgres"),
                Password: getEnv("DB_PASSWORD", "postgres"),
                DBName:   getEnv("DB_NAME", "taxi_connect"),
                SSLMode:  getEnv("DB_SSLMODE", "disable"),
        }
  }

  // getEnv helper reads env var or returns fallback
  func getEnv(key, fallback string) string {
        if value, ok := os.LookupEnv(key); ok {
                return value
        }
        return fallback
  }

  // ConnectDB opens a connection to the PostgreSQL database
  func ConnectDB(config DBConfig) (*sql.DB, error) {
        // Build connection string
        connStr := fmt.Sprintf(
                "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
                config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
        )

        // Configure pgx config
        pgxConfig, err := pgx.ParseConfig(connStr)
        if err != nil {
                return nil, fmt.Errorf("failed to parse postgres config: %w", err)
        }

        // Use stdlib to get *sql.DB from pgx config
        sqlDB := stdlib.OpenDB(*pgxConfig)

        // Test the connection
        if err := sqlDB.Ping(); err != nil {
                return nil, fmt.Errorf("failed to connect to database: %w", err)
        }

        log.Println("Successfully connected to PostgreSQL")
        return sqlDB, nil
  }