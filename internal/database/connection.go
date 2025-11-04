package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// DB represents the database connection
type DB struct {
	*sql.DB
}

// Config represents database configuration
type Config struct {
	Path          string        `json:"path"`
	MaxOpenConns  int           `json:"max_open_conns"`
	MaxIdleConns  int           `json:"max_idle_conns"`
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime"`
}

// DefaultConfig returns default database configuration
func DefaultConfig() Config {
	return Config{
		Path:          "./data/reservas.db",
		MaxOpenConns:  10,
		MaxIdleConns:  5,
		ConnMaxLifetime: time.Hour,
	}
}

// NewConnection creates a new database connection
func NewConnection(config Config) (*DB, error) {
	// SQLite connection string with optimizations
	dsn := fmt.Sprintf("%s?cache=shared&mode=rwc&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=10000&_temp_store=memory&_mmap_size=268435456",
		config.Path)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set WAL mode for better performance
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Printf("Warning: failed to set WAL mode: %v", err)
	}

	// Set synchronous mode to NORMAL for balance between safety and performance
	if _, err := db.Exec("PRAGMA synchronous = NORMAL"); err != nil {
		log.Printf("Warning: failed to set synchronous mode: %v", err)
	}

	// Enable query optimizations
	if _, err := db.Exec("PRAGMA optimize"); err != nil {
		log.Printf("Warning: failed to optimize database: %v", err)
	}

	log.Printf("Database connection established: %s", config.Path)
	return &DB{DB: db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}

// Begin starts a new transaction
func (db *DB) Begin() (*sql.Tx, error) {
	return db.DB.Begin()
}

// GetStats returns database statistics
func (db *DB) GetStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get page count
	var pageCount int64
	err := db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return nil, err
	}
	stats["page_count"] = pageCount

	// Get page size
	var pageSize int64
	err = db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return nil, err
	}
	stats["page_size"] = pageSize

	// Calculate database size
	stats["database_size_bytes"] = pageCount * pageSize
	stats["database_size_mb"] = float64(pageCount*pageSize) / (1024 * 1024)

	// Get cache size
	var cacheSize int64
	err = db.QueryRow("PRAGMA cache_size").Scan(&cacheSize)
	if err != nil {
		return nil, err
	}
	stats["cache_size_pages"] = cacheSize
	stats["cache_size_kb"] = cacheSize * pageSize / 1024

	// Get journal mode
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		return nil, err
	}
	stats["journal_mode"] = journalMode

	// Get foreign keys status
	var foreignKeys int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys)
	if err != nil {
		return nil, err
	}
	stats["foreign_keys_enabled"] = foreignKeys == 1

	return stats, nil
}

// Optimize runs database optimization commands
func (db *DB) Optimize() error {
	optimizations := []string{
		"PRAGMA optimize",
		"ANALYZE",
		"VACUUM",
	}

	for _, sql := range optimizations {
		if _, err := db.Exec(sql); err != nil {
			log.Printf("Warning: failed to run optimization '%s': %v", sql, err)
		}
	}

	return nil
}

// Backup creates a backup of the database
func (db *DB) Backup(backupPath string) error {
	if _, err := db.Exec("VACUUM INTO " + backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	
	log.Printf("Database backup created: %s", backupPath)
	return nil
}

// RunTransaction executes a function within a transaction
func (db *DB) RunTransaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// HealthCheck performs a basic health check
func (db *DB) HealthCheck() error {
	// Test basic query
	var version string
	err := db.QueryRow("SELECT sqlite_version()").Scan(&version)
	if err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	// Test table existence (will be added after schema creation)
	// For now, just verify we can query
	log.Printf("Database health check passed. SQLite version: %s", version)
	return nil
}