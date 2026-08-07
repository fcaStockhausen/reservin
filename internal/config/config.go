package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"reservas/internal/database"
)

// Config represents application configuration
type Config struct {
	Database    database.Config   `json:"database"`
	Server      ServerConfig      `json:"server"`
	Logging     LoggingConfig     `json:"logging"`
	Calculation CalculationConfig `json:"calculation"`
	Data        DataConfig        `json:"data"`
}

// ServerConfig represents HTTP server configuration
type ServerConfig struct {
	Host         string        `json:"host"`
	Port         int           `json:"port"`
	ReadTimeout  time.Duration `json:"read_timeout"`
	WriteTimeout time.Duration `json:"write_timeout"`
	IdleTimeout  time.Duration `json:"idle_timeout"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Level      string `json:"level"`
	Format     string `json:"format"` // json, text
	Output     string `json:"output"` // stdout, stderr, file
	Filename   string `json:"filename"`
	MaxSize    int    `json:"max_size"` // MB
	MaxBackups int    `json:"max_backups"`
	MaxAge     int    `json:"max_age"` // days
}

// CalculationConfig represents calculation engine configuration
type CalculationConfig struct {
	MaxWorkers     int           `json:"max_workers"`
	Timeout        time.Duration `json:"timeout"`
	Precision      int           `json:"precision"`
	DecimalPlaces  int           `json:"decimal_places"`
	BatchSize      int           `json:"batch_size"`
	ParallelPolicy int           `json:"parallel_policy"`
}

// DataConfig represents data source configuration
type DataConfig struct {
	MortalityTables DataConfigSource `json:"mortality_tables"`
	Circular491     DataConfigSource `json:"circular_491"`  // 1985 legacy tables (RV-85/B-85)
	VTDData         DataConfigSource `json:"vtd_data"`      // current VTD year
	VTDHistorico    DataConfigSource `json:"vtd_historico"` // historical VTD curves (2020+)
	TMRates         DataConfigSource `json:"tm_rates"`
	PolicyData      DataConfigSource `json:"policy_data"`
	BackupPath      string           `json:"backup_path"`
}

// DataConfigSource represents a data source configuration
type DataConfigSource struct {
	Path     string `json:"path"`
	Format   string `json:"format"`   // xlsx, csv, json
	Encoding string `json:"encoding"` // utf-8, latin1
	Enabled  bool   `json:"enabled"`
}

// DefaultConfig returns default application configuration
func DefaultConfig() Config {
	return Config{
		Database: database.DefaultConfig(),
		Server: ServerConfig{
			Host:         "localhost",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Filename:   "reservas.log",
			MaxSize:    100, // MB
			MaxBackups: 5,
			MaxAge:     30, // days
		},
		Calculation: CalculationConfig{
			MaxWorkers:     4,
			Timeout:        5 * time.Minute,
			Precision:      8,
			DecimalPlaces:  2,
			BatchSize:      1000,
			ParallelPolicy: 100,
		},
		Data: DataConfig{
			MortalityTables: DataConfigSource{
				Path:     "./data/normativo/articles-20210_tablas_mort_hist.xlsx",
				Format:   "xlsx",
				Encoding: "utf-8",
				Enabled:  true,
			},
			Circular491: DataConfigSource{
				Path:     "./docs/normativo/Tablas_Mortalidad_Circular_491_extr_1.xlsx",
				Format:   "xlsx",
				Encoding: "utf-8",
				Enabled:  true,
			},
			VTDData: DataConfigSource{
				Path:     "./data/normativo/VTD_2025_.xlsx",
				Format:   "xlsx",
				Encoding: "utf-8",
				Enabled:  true,
			},
			VTDHistorico: DataConfigSource{
				Path:     "./data/vtd/articles-51926_recurso_1.xlsx",
				Format:   "xlsx",
				Encoding: "utf-8",
				Enabled:  true,
			},
			TMRates: DataConfigSource{
				Path:     "./data/normativo/tm_rates.csv",
				Format:   "csv",
				Encoding: "utf-8",
				Enabled:  false,
			},
			PolicyData: DataConfigSource{
				Path:     "./data/policies.xlsx",
				Format:   "xlsx",
				Encoding: "utf-8",
				Enabled:  false,
			},
			BackupPath: "./data/backups",
		},
	}
}

// Load loads configuration from file
func Load(configPath string) (Config, error) {
	// Start with default configuration
	cfg := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default configuration file
		if err := Save(cfg, configPath); err != nil {
			return cfg, fmt.Errorf("failed to create default config: %w", err)
		}
		fmt.Printf("Created default configuration file: %s\n", configPath)
		return cfg, nil
	}

	// Read configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return cfg, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := Validate(cfg); err != nil {
		return cfg, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Save saves configuration to file
func Save(cfg Config, configPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal configuration: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates configuration values
func Validate(cfg Config) error {
	// Validate database configuration
	if cfg.Database.Path == "" {
		return fmt.Errorf("database path cannot be empty")
	}
	if cfg.Database.MaxOpenConns <= 0 {
		return fmt.Errorf("database max open connections must be positive")
	}
	if cfg.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database max idle connections cannot be negative")
	}

	// Validate server configuration
	if cfg.Server.Port < 1 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}
	if cfg.Server.ReadTimeout < 0 {
		return fmt.Errorf("server read timeout cannot be negative")
	}
	if cfg.Server.WriteTimeout < 0 {
		return fmt.Errorf("server write timeout cannot be negative")
	}

	// Validate logging configuration
	validLevels := []string{"debug", "info", "warn", "error"}
	isValidLevel := false
	for _, level := range validLevels {
		if cfg.Logging.Level == level {
			isValidLevel = true
			break
		}
	}
	if !isValidLevel {
		return fmt.Errorf("invalid logging level: %s", cfg.Logging.Level)
	}

	validFormats := []string{"json", "text"}
	isValidFormat := false
	for _, format := range validFormats {
		if cfg.Logging.Format == format {
			isValidFormat = true
			break
		}
	}
	if !isValidFormat {
		return fmt.Errorf("invalid logging format: %s", cfg.Logging.Format)
	}

	// Validate calculation configuration
	if cfg.Calculation.MaxWorkers <= 0 {
		return fmt.Errorf("max workers must be positive")
	}
	if cfg.Calculation.Timeout <= 0 {
		return fmt.Errorf("calculation timeout must be positive")
	}
	if cfg.Calculation.Precision < 1 || cfg.Calculation.Precision > 10 {
		return fmt.Errorf("precision must be between 1 and 10")
	}
	if cfg.Calculation.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	// Validate data configuration
	if cfg.Data.BackupPath == "" {
		return fmt.Errorf("backup path cannot be empty")
	}

	return nil
}

// GetServerAddress returns the server address string
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Logging.Level == "warn" || c.Logging.Level == "error"
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Logging.Level == "debug"
}

// GetDatabasePath returns the absolute database path
func (c *Config) GetDatabasePath() string {
	if c.Database.Path[0] == '/' {
		return c.Database.Path
	}

	// Handle relative path
	wd, err := os.Getwd()
	if err != nil {
		return c.Database.Path
	}

	return filepath.Join(wd, c.Database.Path)
}

// GetDataPath returns the absolute path for data files
func (c *Config) GetDataPath(relativePath string) string {
	if relativePath[0] == '/' {
		return relativePath
	}

	// Handle relative path
	wd, err := os.Getwd()
	if err != nil {
		return relativePath
	}

	return filepath.Join(wd, relativePath)
}
