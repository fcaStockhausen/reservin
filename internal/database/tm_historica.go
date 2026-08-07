package database

import (
	"database/sql"
	"fmt"
)

// TMRecord is one month of the historical market rate (TM) published by the
// CMF via Oficio Circular. It is the reserve discount rate input for cohorts
// where the rate is min(TM, TV) or TM (Circular 1512 / NCG 318 §2.3a).
type TMRecord struct {
	Year  int
	Month int
	// Tasa is the TM as a fraction (0.0558 = 5.58%).
	Tasa float64
}

// TMRepository provides access to the tm_historica table.
type TMRepository struct {
	db *sql.DB
}

// NewTMRepository creates a TM repository on the given database.
func NewTMRepository(db *sql.DB) *TMRepository {
	return &TMRepository{db: db}
}

// BatchInsert inserts TM records, replacing existing rows for the same month.
func (r *TMRepository) BatchInsert(records []TMRecord) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO tm_historica (year, month, tasa)
		VALUES (?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, rec := range records {
		if _, err := stmt.Exec(rec.Year, rec.Month, rec.Tasa); err != nil {
			return fmt.Errorf("insert TM %04d-%02d: %w", rec.Year, rec.Month, err)
		}
	}
	return tx.Commit()
}

// GetByYearMonth returns the TM for a specific month, or 0 if not present.
func (r *TMRepository) GetByYearMonth(year, month int) (float64, error) {
	var tasa float64
	err := r.db.QueryRow(
		`SELECT tasa FROM tm_historica WHERE year = ? AND month = ?`,
		year, month).Scan(&tasa)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return tasa, nil
}

// Count returns the number of loaded TM months.
func (r *TMRepository) Count() (int, error) {
	var n int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM tm_historica`).Scan(&n)
	return n, err
}
