package database

import (
	"database/sql"
	"fmt"

	"reservas/internal/models"
)

// MortalityRepository handles mortality table database operations
type MortalityRepository struct {
	db *sql.DB
}

// NewMortalityRepository creates a new mortality repository
func NewMortalityRepository(db *sql.DB) *MortalityRepository {
	return &MortalityRepository{db: db}
}

// CreateTable creates the mortality table
func (r *MortalityRepository) CreateTable() error {
	sql := `
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
	);`

	_, err := r.db.Exec(sql)
	return err
}

// Insert inserts a mortality table record
func (r *MortalityRepository) Insert(table models.MortalityTable) error {
	sql := `
	INSERT INTO tabla_mortalidad (
		nombre_estandar, nombre_original, sexo, tipo_tabla, año_tabla,
		edad, prob_muerte, factor_aax, vigencia_inicio, vigencia_fin
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(sql,
		table.NombreEstandar, table.NombreOriginal, table.Sexo, table.TipoTabla, table.AñoTabla,
		table.Edad, table.ProbMuerte, table.FactorAax, table.VigenciaInicio, table.VigenciaFin,
	)
	return err
}

// BatchInsert inserts multiple mortality table records
func (r *MortalityRepository) BatchInsert(tables []models.MortalityTable) error {
	if len(tables) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Prepare insert statement
	sql := `
	INSERT INTO tabla_mortalidad (
		nombre_estandar, nombre_original, sexo, tipo_tabla, año_tabla,
		edad, prob_muerte, factor_aax, vigencia_inicio, vigencia_fin
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert all records
	for _, table := range tables {
		_, err := stmt.Exec(
			table.NombreEstandar, table.NombreOriginal, table.Sexo, table.TipoTabla, table.AñoTabla,
			table.Edad, table.ProbMuerte, table.FactorAax, table.VigenciaInicio, table.VigenciaFin,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetByNameAndAge retrieves mortality table by name and age
func (r *MortalityRepository) GetByNameAndAge(nombreEstandar string, edad int) (*models.MortalityTable, error) {
	sql := `
	SELECT id, nombre_estandar, nombre_original, sexo, tipo_tabla, año_tabla,
		   edad, prob_muerte, factor_aax, vigencia_inicio, vigencia_fin, created_at
	FROM tabla_mortalidad
	WHERE nombre_estandar = ? AND edad = ?`

	table := &models.MortalityTable{}
	err := r.db.QueryRow(sql, nombreEstandar, edad).Scan(
		&table.ID, &table.NombreEstandar, &table.NombreOriginal, &table.Sexo, &table.TipoTabla,
		&table.AñoTabla, &table.Edad, &table.ProbMuerte, &table.FactorAax,
		&table.VigenciaInicio, &table.VigenciaFin, &table.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return table, nil
}

// GetByTypeAndSex retrieves mortality tables by type and sex
func (r *MortalityRepository) GetByTypeAndSex(tipoTabla, sexo string) ([]models.MortalityTable, error) {
	sql := `
	SELECT id, nombre_estandar, nombre_original, sexo, tipo_tabla, año_tabla,
		   edad, prob_muerte, factor_aax, vigencia_inicio, vigencia_fin, created_at
	FROM tabla_mortalidad
	WHERE tipo_tabla = ? AND sexo = ?
	ORDER BY edad ASC`

	rows, err := r.db.Query(sql, tipoTabla, sexo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.MortalityTable
	for rows.Next() {
		table := models.MortalityTable{}
		err := rows.Scan(
			&table.ID, &table.NombreEstandar, &table.NombreOriginal, &table.Sexo, &table.TipoTabla,
			&table.AñoTabla, &table.Edad, &table.ProbMuerte, &table.FactorAax,
			&table.VigenciaInicio, &table.VigenciaFin, &table.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetByStandardName retrieves all records for a standard table name
func (r *MortalityRepository) GetByStandardName(nombreEstandar string) ([]models.MortalityTable, error) {
	sql := `
	SELECT id, nombre_estandar, nombre_original, sexo, tipo_tabla, año_tabla,
		   edad, prob_muerte, factor_aax, vigencia_inicio, vigencia_fin, created_at
	FROM tabla_mortalidad
	WHERE nombre_estandar = ?
	ORDER BY edad ASC`

	rows, err := r.db.Query(sql, nombreEstandar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []models.MortalityTable
	for rows.Next() {
		table := models.MortalityTable{}
		err := rows.Scan(
			&table.ID, &table.NombreEstandar, &table.NombreOriginal, &table.Sexo, &table.TipoTabla,
			&table.AñoTabla, &table.Edad, &table.ProbMuerte, &table.FactorAax,
			&table.VigenciaInicio, &table.VigenciaFin, &table.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetAllTables returns all distinct mortality tables
func (r *MortalityRepository) GetAllTables() ([]string, error) {
	sql := `
	SELECT DISTINCT nombre_estandar
	FROM tabla_mortalidad
	ORDER BY nombre_estandar ASC`

	rows, err := r.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetStatistics returns statistics about mortality tables
func (r *MortalityRepository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total tables
	var totalTables int
	err := r.db.QueryRow("SELECT COUNT(DISTINCT nombre_estandar) FROM tabla_mortalidad").Scan(&totalTables)
	if err != nil {
		return nil, err
	}
	stats["total_tables"] = totalTables

	// Total records
	var totalRecords int
	err = r.db.QueryRow("SELECT COUNT(*) FROM tabla_mortalidad").Scan(&totalRecords)
	if err != nil {
		return nil, err
	}
	stats["total_records"] = totalRecords

	// Age range
	var minAge, maxAge int
	err = r.db.QueryRow("SELECT MIN(edad) FROM tabla_mortalidad").Scan(&minAge)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow("SELECT MAX(edad) FROM tabla_mortalidad").Scan(&maxAge)
	if err != nil {
		return nil, err
	}
	stats["age_range"] = fmt.Sprintf("%d-%d", minAge, maxAge)

	// Tables by year
	rows, err := r.db.Query(`
		SELECT año_tabla, COUNT(DISTINCT nombre_estandar)
		FROM tabla_mortalidad
		WHERE año_tabla IS NOT NULL
		GROUP BY año_tabla
		ORDER BY año_tabla DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tablesByYear := make(map[string]int)
	for rows.Next() {
		var year int
		var count int
		if err := rows.Scan(&year, &count); err != nil {
			return nil, err
		}
		tablesByYear[fmt.Sprintf("%d", year)] = count
	}
	stats["tables_by_year"] = tablesByYear

	return stats, nil
}

// ValidateTable validates a mortality table for consistency
func (r *MortalityRepository) ValidateTable(nombreEstandar string) error {
	sql := `
	SELECT edad, prob_muerte
	FROM tabla_mortalidad
	WHERE nombre_estandar = ?
	ORDER BY edad ASC`

	rows, err := r.db.Query(sql, nombreEstandar)
	if err != nil {
		return err
	}
	defer rows.Close()

	var prevAge int
	var prevProbMuerte float64
	isFirst := true

	for rows.Next() {
		var age int
		var probMuerte float64
		if err := rows.Scan(&age, &probMuerte); err != nil {
			return err
		}

		// Check age sequence
		if !isFirst && age != prevAge+1 {
			return fmt.Errorf("age gap found: %d to %d", prevAge, age)
		}

		// Check qx range
		if probMuerte < 0 || probMuerte > 1 {
			return fmt.Errorf("invalid qx value at age %d: %f", age, probMuerte)
		}

		// Optional: Check monotonicity for certain age ranges
		if !isFirst && age >= 60 && probMuerte < prevProbMuerte {
			// Warning, not error: qx should generally increase with age
			fmt.Printf("Warning: qx decreased at age %d: %f to %f\n", prevAge, prevProbMuerte, probMuerte)
		}

		prevAge = age
		prevProbMuerte = probMuerte
		isFirst = false
	}

	return nil
}

// CreateIndexes creates performance indexes for mortality tables
func (r *MortalityRepository) CreateIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_nombre_edad ON tabla_mortalidad(nombre_estandar, edad);`,
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_vigencia ON tabla_mortalidad(vigencia_inicio, vigencia_fin);`,
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_sexo ON tabla_mortalidad(sexo);`,
		`CREATE INDEX IF NOT EXISTS idx_mortalidad_tipo ON tabla_mortalidad(tipo_tabla);`,
	}

	for _, index := range indexes {
		if _, err := r.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index '%s': %w", index, err)
		}
	}

	return nil
}

// Cleanup removes records with invalid data
func (r *MortalityRepository) Cleanup() error {
	sql := `
	DELETE FROM tabla_mortalidad
	WHERE prob_muerte IS NULL
	OR prob_muerte < 0
	OR prob_muerte > 1
	OR edad IS NULL
	OR edad < 0
	OR edad > 110`

	result, err := r.db.Exec(sql)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d invalid mortality table records\n", rowsAffected)
	}

	return nil
}