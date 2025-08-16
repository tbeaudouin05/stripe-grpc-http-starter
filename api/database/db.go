package database

import (
    "database/sql"
    "fmt"
    "strings"
    "time"

    _ "github.com/lib/pq"
    config "github.com/tbeaudouin05/stripe-trellai/api/config"
)

var db *sql.DB

// Initialize connects to the Neon database and verifies the connection
func Initialize() error {
    var err error
    dsn := withDisablePreparedStatements(config.AppConfig.DatabaseURL)
    db, err = sql.Open("postgres", dsn)
    if err != nil {
        return fmt.Errorf("failed to connect to Neon: %w", err)
    }
    // Verify connection
    err = db.Ping()
    if err != nil {
        return fmt.Errorf("failed to ping database: %w", err)
    }

    // Configure connection pool
    // Use a single connection to avoid prepared statement issues with PgBouncer/Neon in tests.
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(5 * time.Minute)

    return nil
}

// withDisablePreparedStatements appends disable_prepared_statements=true and binary_parameters=yes to the DSN if not present.
// This nudges lib/pq to avoid server-side prepared statements and binary mode, which can break with PgBouncer transaction pooling.
func withDisablePreparedStatements(dsn string) string {
    lower := strings.ToLower(dsn)
    if strings.Contains(lower, "disable_prepared_statements=") || strings.Contains(lower, "prefer_simple_protocol=") {
        return dsn
    }
    sep := "?"
    if strings.Contains(dsn, "?") {
        sep = "&"
    }
    extras := []string{"disable_prepared_statements=true"}
    if !strings.Contains(lower, "binary_parameters=") {
        extras = append(extras, "binary_parameters=yes")
    }
    return dsn + sep + strings.Join(extras, "&")
}

// GetDB returns the database connection
func GetDB() *sql.DB {
    return db
}


