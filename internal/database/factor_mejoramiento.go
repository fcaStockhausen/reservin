package database

import (
	"database/sql"

	"github.com/shopspring/decimal"

	"reservas/internal/models"
)

// FactorMejoramientoRepository handles the mortality improvement factors table.
type FactorMejoramientoRepository struct {
	db *sql.DB
}

// NewFactorMejoramientoRepository creates the repository.
func NewFactorMejoramientoRepository(db *sql.DB) *FactorMejoramientoRepository {
	return &FactorMejoramientoRepository{db: db}
}

// CreateTable creates the factor_mejoramiento table.
func (r *FactorMejoramientoRepository) CreateTable() error {
	_, err := r.db.Exec(`
	CREATE TABLE IF NOT EXISTS factor_mejoramiento (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		nombre_estandar VARCHAR(50) NOT NULL,
		edad INTEGER NOT NULL,
		año INTEGER NOT NULL,
		factor_aa DECIMAL(10,8) NOT NULL,
		UNIQUE(nombre_estandar, edad, año)
	);`)
	return err
}

// BatchInsert inserts improvement factors (idempotent).
func (r *FactorMejoramientoRepository) BatchInsert(factors []models.FactorMejoramiento) error {
	if len(factors) == 0 {
		return nil
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT OR REPLACE INTO factor_mejoramiento (nombre_estandar, edad, año, factor_aa)
	VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, f := range factors {
		if _, err := stmt.Exec(f.NombreEstandar, f.Edad, f.Año, f.FactorAA); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// GetFactors returns all improvement factors for a table.
func (r *FactorMejoramientoRepository) GetFactors(nombreEstandar string) ([]models.FactorMejoramiento, error) {
	rows, err := r.db.Query(`
	SELECT id, nombre_estandar, edad, año, factor_aa
	FROM factor_mejoramiento
	WHERE nombre_estandar = ?
	ORDER BY edad, año`, nombreEstandar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var factors []models.FactorMejoramiento
	for rows.Next() {
		var f models.FactorMejoramiento
		var raw string
		if err := rows.Scan(&f.ID, &f.NombreEstandar, &f.Edad, &f.Año, &raw); err != nil {
			return nil, err
		}
		d, err := decimal.NewFromString(raw)
		if err != nil {
			return nil, err
		}
		f.FactorAA = d
		factors = append(factors, f)
	}
	return factors, nil
}

// Count returns the number of improvement factors in the table.
func (r *FactorMejoramientoRepository) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM factor_mejoramiento`).Scan(&n)
	return n, err
}
