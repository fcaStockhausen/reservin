package models

import (
	"database/sql"
	"time"

	"github.com/shopspring/decimal"
)

// VTDPoint represents a single point in the VTD vector
type VTDPoint struct {
	Year            int             `json:"year" db:"year"`
	Month           int             `json:"month" db:"month"`
	Period          int             `json:"period" db:"period"` // Year 1 to 120
	Rate            decimal.Decimal `json:"rate" db:"rate"`     // Discount rate
	PublicationDate time.Time       `json:"publication_date" db:"publication_date"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// VTDVector represents a complete VTD vector for a specific year/month
type VTDVector struct {
	Year            int        `json:"year"`
	Month           int        `json:"month"`
	PublicationDate time.Time  `json:"publication_date"`
	Rates           []VTDPoint `json:"rates"`
}

// VTDManager manages VTD data for discount rate calculations
type VTDManager struct {
	vectors map[int]map[int][]VTDPoint // year -> month -> rates
	db      *sql.DB
}

// NewVTDManager creates a new VTD manager
func NewVTDManager(db *sql.DB) *VTDManager {
	return &VTDManager{
		vectors: make(map[int]map[int][]VTDPoint),
		db:      db,
	}
}

// LoadVectors loads VTD vectors from database
func (vm *VTDManager) LoadVectors() error {
	// This would load from database in production
	// For now, initialize with empty structure
	vm.vectors = make(map[int]map[int][]VTDPoint)
	return nil
}

// GetRate retrieves VTD rate for specific year, month, and period
func (vm *VTDManager) GetRate(year, month, period int) (decimal.Decimal, error) {
	if monthly, ok := vm.vectors[year][month]; ok {
		for _, point := range monthly {
			if point.Period == period {
				return point.Rate, nil
			}
		}
	}
	return decimal.Zero, ErrVTDRateNotFound
}

// GetVector retrieves complete VTD vector for year and month
func (vm *VTDManager) GetVector(year, month int) (*VTDVector, error) {
	if monthly, ok := vm.vectors[year][month]; ok {
		return &VTDVector{
			Year:  year,
			Month: month,
			Rates: monthly,
		}, nil
	}
	return nil, ErrVTDRateNotFound
}

// GetVectorByDate retrieves VTD vector for specific date
func (vm *VTDManager) GetVectorByDate(date time.Time) (*VTDVector, error) {
	year := date.Year()
	month := int(date.Month())

	return vm.GetVector(year, month)
}

// AddVector adds a complete VTD vector
func (vm *VTDManager) AddVector(vector VTDVector) error {
	if vm.vectors[vector.Year] == nil {
		vm.vectors[vector.Year] = make(map[int][]VTDPoint)
	}

	vm.vectors[vector.Year][vector.Month] = vector.Rates
	return nil
}

// AddPoint adds a single VTD point
func (vm *VTDManager) AddPoint(point VTDPoint) error {
	if vm.vectors[point.Year] == nil {
		vm.vectors[point.Year] = make(map[int][]VTDPoint)
	}

	monthly := vm.vectors[point.Year][point.Month]
	for i, existing := range monthly {
		if existing.Period == point.Period {
			monthly[i] = point
			return nil
		}
	}

	monthly = append(monthly, point)
	vm.vectors[point.Year][point.Month] = monthly
	return nil
}

// GetAvailableDates returns a list of available VTD dates
func (vm *VTDManager) GetAvailableDates() []time.Time {
	var dates []time.Time

	for year, monthly := range vm.vectors {
		for month := range monthly {
			if len(monthly) > 0 {
				date := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				dates = append(dates, date)
			}
		}
	}

	return dates
}

// ValidateVector validates a complete VTD vector
func (vm *VTDManager) ValidateVector(vector VTDVector) error {
	// Check required periods (1-120 years)
	periods := make(map[int]bool)
	for _, point := range vector.Rates {
		if point.Period < 1 || point.Period > 120 {
			return ErrPeriodOutOfRange
		}
		periods[point.Period] = true
	}

	// Check monotonicity (optional validation)
	for i := 1; i <= 10; i++ { // Check first 10 years
		if !periods[i] {
			return ErrPeriodOutOfRange
		}
	}

	return nil
}

// GetLatestVector gets the most recent VTD vector available
func (vm *VTDManager) GetLatestVector() (*VTDVector, error) {
	var latestDate time.Time
	var latestVector *VTDVector

	for year, monthly := range vm.vectors {
		for month, rates := range monthly {
			if len(rates) > 0 {
				currentDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				if latestVector == nil || currentDate.After(latestDate) {
					latestDate = currentDate
					latestVector = &VTDVector{
						Year:  year,
						Month: month,
						Rates: rates,
					}
				}
			}
		}
	}

	if latestVector == nil {
		return nil, ErrVTDRateNotFound
	}

	return latestVector, nil
}

// CreateFallbackVector creates a fallback VTD vector with constant rates
func CreateFallbackVector(year, month int, rate decimal.Decimal, periods int) VTDVector {
	var rates []VTDPoint

	for period := 1; period <= periods; period++ {
		point := VTDPoint{
			Year:            year,
			Month:           month,
			Period:          period,
			Rate:            rate,
			PublicationDate: time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC),
			CreatedAt:       time.Now(),
		}
		rates = append(rates, point)
	}

	return VTDVector{
		Year:            year,
		Month:           month,
		PublicationDate: time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC),
		Rates:           rates,
	}
}

// ImportFromExcel creates VTD vectors from Excel import data
func (vm *VTDManager) ImportFromExcel(year int, monthlyData map[int]map[int]decimal.Decimal) error {
	for month, periods := range monthlyData {
		vector := VTDVector{
			Year:            year,
			Month:           month,
			PublicationDate: time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC),
			Rates:           make([]VTDPoint, 0),
		}

		for period, rate := range periods {
			point := VTDPoint{
				Year:            year,
				Month:           month,
				Period:          period,
				Rate:            rate,
				PublicationDate: vector.PublicationDate,
				CreatedAt:       time.Now(),
			}
			vector.Rates = append(vector.Rates, point)
		}

		// Validate vector before adding
		if err := vm.ValidateVector(vector); err != nil {
			return err
		}

		if err := vm.AddVector(vector); err != nil {
			return err
		}
	}

	return nil
}

// GetStatistics returns statistics about loaded VTD data
func (vm *VTDManager) GetStatistics() VTDStatistics {
	stats := VTDStatistics{
		TotalVectors: 0,
		TotalPoints:  0,
		Years:        make(map[int]bool),
		DateRange:    DateRange{},
	}

	for year, monthly := range vm.vectors {
		stats.Years[year] = true
		for month, rates := range monthly {
			if len(rates) > 0 {
				stats.TotalVectors++
				stats.TotalPoints += len(rates)

				date := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
				if stats.DateRange.Start.IsZero() || date.Before(stats.DateRange.Start) {
					stats.DateRange.Start = date
				}
				if stats.DateRange.End.IsZero() || date.After(stats.DateRange.End) {
					stats.DateRange.End = date
				}
			}
		}
	}

	return stats
}

// VTDStatistics provides statistics about VTD data
type VTDStatistics struct {
	TotalVectors int          `json:"total_vectors"`
	TotalPoints  int          `json:"total_points"`
	Years        map[int]bool `json:"years"`
	DateRange    DateRange    `json:"date_range"`
}

// DateRange represents a range of dates
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
