package database

import (
	"database/sql"
	"fmt"
	"time"

	"reservas/internal/models"
)

// VTDRepository handles VTD vector database operations
type VTDRepository struct {
	db *sql.DB
}

// NewVTDRepository creates a new VTD repository
func NewVTDRepository(db *sql.DB) *VTDRepository {
	return &VTDRepository{db: db}
}

// CreateTable creates the VTD vector table
func (r *VTDRepository) CreateTable() error {
	sql := `
	CREATE TABLE IF NOT EXISTS vtd_vector (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		year INTEGER NOT NULL,
		month INTEGER NOT NULL,
		period INTEGER NOT NULL, -- Year 1 to 120
		rate DECIMAL(8,6) NOT NULL, -- Discount rate
		publication_date DATE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(year, month, period)
	);`

	_, err := r.db.Exec(sql)
	return err
}

// Insert inserts a single VTD point
func (r *VTDRepository) Insert(point models.VTDPoint) error {
	sql := `
	INSERT INTO vtd_vector (
		year, month, period, rate, publication_date
	) VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.Exec(sql,
		point.Year, point.Month, point.Period, point.Rate, point.PublicationDate,
	)
	return err
}

// BatchInsert inserts multiple VTD points
func (r *VTDRepository) BatchInsert(points []models.VTDPoint) error {
	if len(points) == 0 {
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
	INSERT INTO vtd_vector (
		year, month, period, rate, publication_date
	) VALUES (?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Insert all points
	for _, point := range points {
		_, err := stmt.Exec(
			point.Year, point.Month, point.Period, point.Rate, point.PublicationDate,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// InsertVector inserts a complete VTD vector
func (r *VTDRepository) InsertVector(vector models.VTDVector) error {
	points := make([]models.VTDPoint, len(vector.Rates))
	for i := range vector.Rates {
		points[i] = models.VTDPoint{
			Year:           vector.Year,
			Month:          vector.Month,
			Period:         vector.Rates[i].Period,
			Rate:           vector.Rates[i].Rate,
			PublicationDate: vector.PublicationDate,
		}
	}
	return r.BatchInsert(points)
}

// GetRate retrieves VTD rate for specific year, month, and period
func (r *VTDRepository) GetRate(year, month, period int) (*models.VTDPoint, error) {
	sql := `
	SELECT id, year, month, period, rate, publication_date, created_at
	FROM vtd_vector
	WHERE year = ? AND month = ? AND period = ?`

	point := &models.VTDPoint{}
	err := r.db.QueryRow(sql, year, month, period).Scan(
		&point.Year, &point.Month, &point.Period, &point.Rate,
		&point.PublicationDate, &point.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return point, nil
}

// GetVector retrieves complete VTD vector for year and month
func (r *VTDRepository) GetVector(year, month int) (*models.VTDVector, error) {
	sql := `
	SELECT year, month, period, rate, publication_date, created_at
	FROM vtd_vector
	WHERE year = ? AND month = ?
	ORDER BY period ASC`

	rows, err := r.db.Query(sql, year, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	vector := &models.VTDVector{
		Year:   year,
		Month:  month,
		Rates:  make([]models.VTDPoint, 0),
	}

	for rows.Next() {
		point := models.VTDPoint{}
		err := rows.Scan(
			&point.Year, &point.Month, &point.Period, &point.Rate,
			&point.PublicationDate, &point.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Set publication date once
		if vector.PublicationDate.IsZero() {
			vector.PublicationDate = point.PublicationDate
		}

		vector.Rates = append(vector.Rates, point)
	}

	return vector, nil
}

// GetVectorByDate retrieves VTD vector for specific date
func (r *VTDRepository) GetVectorByDate(dateStr string) (*models.VTDVector, error) {
	sql := `
	SELECT year, month, period, rate, publication_date, created_at
	FROM vtd_vector
	WHERE DATE(publication_date) = DATE(?)
	ORDER BY period ASC`

	rows, err := r.db.Query(sql, dateStr)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rates []models.VTDPoint
	var publicationDate interface{}

	for rows.Next() {
		point := models.VTDPoint{}
		err := rows.Scan(
			&point.Year, &point.Month, &point.Period, &point.Rate,
			&publicationDate, &point.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if pubDate, ok := publicationDate.(time.Time); ok {
			point.PublicationDate = pubDate
		}

		rates = append(rates, point)
	}

	if len(rates) == 0 {
		return nil, fmt.Errorf("no VTD data found for date: %s", dateStr)
	}

	return &models.VTDVector{
		Rates: rates,
	}, nil
}

// GetAvailableYears returns list of years with VTD data
func (r *VTDRepository) GetAvailableYears() ([]int, error) {
	sql := `
	SELECT DISTINCT year
	FROM vtd_vector
	ORDER BY year ASC`

	rows, err := r.db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var years []int
	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		years = append(years, year)
	}

	return years, nil
}

// GetAvailableMonths returns list of months for a given year
func (r *VTDRepository) GetAvailableMonths(year int) ([]int, error) {
	sql := `
	SELECT DISTINCT month
	FROM vtd_vector
	WHERE year = ?
	ORDER BY month ASC`

	rows, err := r.db.Query(sql, year)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var months []int
	for rows.Next() {
		var month int
		if err := rows.Scan(&month); err != nil {
			return nil, err
		}
		months = append(months, month)
	}

	return months, nil
}

// GetLatestVector retrieves the most recent VTD vector
func (r *VTDRepository) GetLatestVector() (*models.VTDVector, error) {
	sql := `
	SELECT year, month, period, rate, publication_date, created_at
	FROM vtd_vector
	ORDER BY year DESC, month DESC, period DESC
	LIMIT 1`

	point := &models.VTDPoint{}
	err := r.db.QueryRow(sql).Scan(
		&point.Year, &point.Month, &point.Period, &point.Rate,
		&point.PublicationDate, &point.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Now get complete vector for this year/month
	return r.GetVector(point.Year, point.Month)
}

// GetStatistics returns statistics about VTD data
func (r *VTDRepository) GetStatistics() (*models.VTDStatistics, error) {
	stats := &models.VTDStatistics{
		Years: make(map[int]bool),
	}

	// Total vectors (year/month combinations)
	err := r.db.QueryRow(`
		SELECT COUNT(DISTINCT year || '-' || month)
		FROM vtd_vector
	`).Scan(&stats.TotalVectors)
	if err != nil {
		return nil, err
	}

	// Total points
	err = r.db.QueryRow(`
		SELECT COUNT(*)
		FROM vtd_vector
	`).Scan(&stats.TotalPoints)
	if err != nil {
		return nil, err
	}

	// Get years with data
	rows, err := r.db.Query(`
		SELECT DISTINCT year
		FROM vtd_vector
		ORDER BY year ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var year int
		if err := rows.Scan(&year); err != nil {
			return nil, err
		}
		stats.Years[year] = true
	}

	// Get date range (stored as text in SQLite, parse after scanning)
	var minDate, maxDate string
	err = r.db.QueryRow(`
		SELECT MIN(publication_date), MAX(publication_date)
		FROM vtd_vector
	`).Scan(&minDate, &maxDate)
	if err != nil {
		return nil, err
	}
	stats.DateRange.Start, _ = time.Parse("2006-01-02 15:04:05Z07:00", minDate)
	stats.DateRange.End, _ = time.Parse("2006-01-02 15:04:05Z07:00", maxDate)
	if stats.DateRange.Start.IsZero() {
		stats.DateRange.Start, _ = time.Parse("2006-01-02", minDate)
	}
	if stats.DateRange.End.IsZero() {
		stats.DateRange.End, _ = time.Parse("2006-01-02", maxDate)
	}

	return stats, nil
}

// ValidateVector validates a complete VTD vector
func (r *VTDRepository) ValidateVector(vector models.VTDVector) error {
	// Check required periods (1-120 years)
	periods := make(map[int]bool)
	for _, point := range vector.Rates {
		if point.Period < 1 || point.Period > 120 {
			return models.ErrPeriodOutOfRange
		}
		periods[point.Period] = true
	}

	// Check for first 10 years (critical periods)
	for period := 1; period <= 10; period++ {
		if !periods[period] {
			return fmt.Errorf("missing critical period: %d", period)
		}
	}

	return nil
}

// CreateIndexes creates performance indexes for VTD table
func (r *VTDRepository) CreateIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_vtd_year_month ON vtd_vector(year, month);`,
		`CREATE INDEX IF NOT EXISTS idx_vtd_publication ON vtd_vector(publication_date);`,
		`CREATE INDEX IF NOT EXISTS idx_vtd_period ON vtd_vector(period);`,
		`CREATE INDEX IF NOT EXISTS idx_vtd_year_month_period ON vtd_vector(year, month, period);`,
	}

	for _, index := range indexes {
		if _, err := r.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index '%s': %w", index, err)
		}
	}

	return nil
}

// Cleanup removes invalid VTD data
func (r *VTDRepository) Cleanup() error {
	sql := `
	DELETE FROM vtd_vector
	WHERE year IS NULL OR month IS NULL OR period IS NULL
	OR year < 2000 OR year > 2100
	OR month < 1 OR month > 12
	OR period < 1 OR period > 120
	OR rate IS NULL
	OR rate < -1 OR rate > 1`

	result, err := r.db.Exec(sql)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		fmt.Printf("Cleaned up %d invalid VTD records\n", rowsAffected)
	}

	return nil
}