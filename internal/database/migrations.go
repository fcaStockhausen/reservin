package database

import (
	"database/sql"
	"fmt"
	"log"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         []string
}

// Migrator handles database migrations
type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

// NewMigrator creates a new migrator
func NewMigrator(db *sql.DB) *Migrator {
	return &Migrator{
		db: db,
		migrations: []Migration{
			{
				Version:     1,
				Description: "Create initial schema",
				SQL: []string{
					`-- Mortality tables (from Excel data/normativo/articles-20210_tablas_mort_hist.xlsx)
					CREATE TABLE IF NOT EXISTS tabla_mortalidad (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						nombre_estandar VARCHAR(50) NOT NULL, -- CMF standard: "CB-H-2020", "RV-M-2020", etc.
						nombre_original VARCHAR(50), -- Excel name: "CB-2020-HOMBRES", etc.
						sexo CHAR(1), -- 'H', 'M', 'A'
						tipo_tabla VARCHAR(20), -- 'VEJEZ', 'INVALIDEZ', 'SOBREVIVENCIA'
						año_tabla INTEGER, -- 2004, 2006, 2009, 2014, 2020
						edad INTEGER NOT NULL,
						prob_muerte DECIMAL(10,8), -- qx value
						factor_aax DECIMAL(10,8), -- Factor Aax
						vigencia_inicio DATE,
						vigencia_fin DATE,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						UNIQUE(nombre_estandar, edad)
					);`,
					`-- VTD vector tasa de descuento (monthly CMF publications)
					CREATE TABLE IF NOT EXISTS vtd_vector (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						year INTEGER NOT NULL,
						month INTEGER NOT NULL,
						period INTEGER NOT NULL, -- Year 1 to 120
						rate DECIMAL(8,6) NOT NULL, -- Discount rate
						publication_date DATE NOT NULL,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						UNIQUE(year, month, period)
					);`,
					`-- Policy data
					CREATE TABLE IF NOT EXISTS poliza (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						numero_poliza VARCHAR(50) UNIQUE NOT NULL,
						tipo_renta VARCHAR(20) NOT NULL, -- 'VITALICIA', 'TEMPORARIA', 'DIFERIDA'
						fecha_inicio DATE NOT NULL,
						fecha_fin DATE,
						edad_contratante INTEGER NOT NULL,
						sexo_beneficiario CHAR(1) NOT NULL, -- 'H', 'M'
						capital_asegurado DECIMAL(15,2) NOT NULL,
						forma_pago VARCHAR(10), -- 'MENSUAL', 'TRIMESTRAL', 'ANUAL'
						tasa_descuento DECIMAL(8,6) NOT NULL, -- "bautizo" rate (min TM, TC)
						tasa_tm DECIMAL(8,6) NOT NULL, -- Tasa venta
						tasa_tc DECIMAL(8,6) NOT NULL, -- Tasa costo
						estado VARCHAR(10) DEFAULT 'ACTIVA', -- 'ACTIVA', 'VENCIDA', 'CANCELADA'
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);`,
					`-- Calculated reserves
					CREATE TABLE IF NOT EXISTS reserva_calculada (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						poliza_id INTEGER NOT NULL,
						fecha_calculo DATE NOT NULL,
						valor_reserva DECIMAL(15,2) NOT NULL,
						metodo_calculo VARCHAR(20) NOT NULL, -- 'VPPJ', 'TRADICIONAL'
						flujo_probabilistico DECIMAL(15,2) NOT NULL,
						tasa_descuento_utilizada DECIMAL(8,6) NOT NULL,
						tabla_mortalidad_utilizada VARCHAR(50) NOT NULL,
						audit_trail_id INTEGER,
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						FOREIGN KEY (poliza_id) REFERENCES poliza(id) ON DELETE CASCADE
					);`,
					`-- Audit trail for CMF compliance
					CREATE TABLE IF NOT EXISTS audit_trail (
						id INTEGER PRIMARY KEY AUTOINCREMENT,
						poliza_id INTEGER NOT NULL,
						calculation_date TIMESTAMP NOT NULL,
						methodology VARCHAR(20) NOT NULL, -- 'IFRS', 'TRADICIONAL', 'TRANSITIONAL'
						inputs TEXT, -- JSON input data
						steps TEXT, -- JSON calculation steps
						outputs TEXT, -- JSON calculation outputs
						validation TEXT, -- JSON validation results
						compliance TEXT, -- JSON compliance results
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						FOREIGN KEY (poliza_id) REFERENCES poliza(id) ON DELETE CASCADE
					);`,
					`-- Migration tracking
					CREATE TABLE IF NOT EXISTS migration_history (
						version INTEGER PRIMARY KEY,
						description TEXT NOT NULL,
						applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					);`,
				},
			},
		},
	}
}

// Migrate runs all pending migrations
func (m *Migrator) Migrate() error {
	// Create migration table if not exists
	if err := m.createMigrationTable(); err != nil {
		return fmt.Errorf("failed to create migration table: %w", err)
	}

	// Get current migration version
	currentVersion, err := m.getCurrentVersion()
	if err != nil {
		return fmt.Errorf("failed to get current migration version: %w", err)
	}

	// Run pending migrations
	for _, migration := range m.migrations {
		if migration.Version <= currentVersion {
			continue
		}

		log.Printf("Running migration %d: %s", migration.Version, migration.Description)

		if err := m.runMigration(migration); err != nil {
			return fmt.Errorf("failed to run migration %d: %w", migration.Version, err)
		}

		if err := m.recordMigration(migration); err != nil {
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		log.Printf("Migration %d completed successfully", migration.Version)
	}

	log.Printf("Database migration completed. Current version: %d", m.migrations[len(m.migrations)-1].Version)
	return nil
}

// createMigrationTable creates the migration tracking table
func (m *Migrator) createMigrationTable() error {
	sql := `
	CREATE TABLE IF NOT EXISTS migration_history (
		version INTEGER PRIMARY KEY,
		description TEXT NOT NULL,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := m.db.Exec(sql)
	return err
}

// getCurrentVersion returns the current migration version
func (m *Migrator) getCurrentVersion() (int, error) {
	var version int
	err := m.db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migration_history").Scan(&version)
	return version, err
}

// runMigration executes a single migration
func (m *Migrator) runMigration(migration Migration) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, sql := range migration.SQL {
		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("failed to execute SQL '%s': %w", sql, err)
		}
	}

	// Create indexes after migration
	if err := m.createIndexes(tx); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return tx.Commit()
}

// createIndexes creates performance indexes
func (m *Migrator) createIndexes(tx *sql.Tx) error {
	indexes := []string{
		// Mortality table indexes
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_nombre_edad ON tabla_mortalidad(nombre_estandar, edad);`,
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_vigencia ON tabla_mortalidad(vigencia_inicio, vigencia_fin);`,
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_sexo ON tabla_mortalidad(sexo);`,
		
		// VTD vector indexes
		`CREATE INDEX IF NOT EXISTS idx_vtd_year_month ON vtd_vector(year, month);`,
		`CREATE INDEX IF NOT EXISTS idx_vtd_publication ON vtd_vector(publication_date);`,
		`CREATE INDEX IF NOT EXISTS idx_vtd_period ON vtd_vector(period);`,
		
		// Policy indexes
		`CREATE INDEX IF NOT EXISTS idx_poliza_estado ON poliza(estado);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_fecha_inicio ON poliza(fecha_inicio);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_numero ON poliza(numero_poliza);`,
		
		// Reserve calculation indexes
		`CREATE INDEX IF NOT EXISTS idx_reserva_poliza_fecha ON reserva_calculada(poliza_id, fecha_calculo);`,
		`CREATE INDEX IF NOT EXISTS idx_reserva_fecha_calculo ON reserva_calculada(fecha_calculo);`,
		
		// Audit trail indexes
		`CREATE INDEX IF NOT EXISTS idx_audit_poliza_fecha ON audit_trail(poliza_id, calculation_date);`,
		`CREATE INDEX IF NOT EXISTS idx_audit_metodologia ON audit_trail(methodology);`,
	}

	for _, indexSQL := range indexes {
		if _, err := tx.Exec(indexSQL); err != nil {
			return fmt.Errorf("failed to create index '%s': %w", indexSQL, err)
		}
	}

	return nil
}

// recordMigration records a migration as applied
func (m *Migrator) recordMigration(migration Migration) error {
	sql := `
	INSERT INTO migration_history (version, description) 
	VALUES (?, ?)`

	_, err := m.db.Exec(sql, migration.Version, migration.Description)
	return err
}

// GetMigrationHistory returns the migration history
func (m *Migrator) GetMigrationHistory() ([]MigrationRecord, error) {
	sql := `
	SELECT version, description, applied_at 
	FROM migration_history 
	ORDER BY version ASC`

	rows, err := m.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []MigrationRecord
	for rows.Next() {
		var record MigrationRecord
		err := rows.Scan(&record.Version, &record.Description, &record.AppliedAt)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, rows.Err()
}

// MigrationRecord represents a migration record
type MigrationRecord struct {
	Version     int       `json:"version"`
	Description string    `json:"description"`
	AppliedAt   string    `json:"applied_at"`
}