package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"reservas/internal/database"
)

// MortalityEngine provides survival probability lookups (tpx) from mortality
// tables loaded directly from the database. Each table is cached as a map of
// edad → qx, and survival probabilities are computed on demand.
type MortalityEngine struct {
	tables map[string]map[int]decimal.Decimal // tableName -> edad -> qx
}

// NewMortalityEngine creates an empty engine.
func NewMortalityEngine() *MortalityEngine {
	return &MortalityEngine{
		tables: make(map[string]map[int]decimal.Decimal),
	}
}

// LoadTable fetches a mortality table from the database and caches it.
func (me *MortalityEngine) LoadTable(repo *database.MortalityRepository, tableName string) error {
	records, err := repo.GetByStandardName(tableName)
	if err != nil {
		return fmt.Errorf("load table %s: %w", tableName, err)
	}
	if len(records) == 0 {
		return fmt.Errorf("table %s has no records", tableName)
	}

	qxMap := make(map[int]decimal.Decimal, len(records))
	for _, r := range records {
		qxMap[r.Edad] = r.ProbMuerte
	}
	me.tables[tableName] = qxMap
	return nil
}

// EnsureLoaded loads the table if not already cached.
func (me *MortalityEngine) EnsureLoaded(repo *database.MortalityRepository, tableName string) error {
	if _, ok := me.tables[tableName]; ok {
		return nil
	}
	return me.LoadTable(repo, tableName)
}

// Qx returns the probability of death at a given age for a table.
func (me *MortalityEngine) Qx(tableName string, edad int) (decimal.Decimal, error) {
	t, ok := me.tables[tableName]
	if !ok {
		return decimal.Zero, fmt.Errorf("table %s not loaded", tableName)
	}
	qx, ok := t[edad]
	if !ok {
		return decimal.Zero, fmt.Errorf("age %d not in table %s", edad, tableName)
	}
	return qx, nil
}

// Tpx computes the probability of surviving t years from a given age.
//
//	tpx(x, t) = product_{k=0}^{t-1} [1 - qx(x+k)]
//
// At t=0, tpx = 1.0 (certainly alive at the starting point).
// If we run past the table's max age, qx is treated as 1.0 (certain death).
func (me *MortalityEngine) Tpx(tableName string, edad, t int) (decimal.Decimal, error) {
	if t <= 0 {
		return decimal.Decimal{}, nil // caller should treat 0-year as 1.0
	}

	one := decimal.NewFromInt(1)
	result := one

	table, ok := me.tables[tableName]
	if !ok {
		return decimal.Zero, fmt.Errorf("table %s not loaded", tableName)
	}

	for k := 0; k < t; k++ {
		qx, exists := table[edad+k]
		if !exists {
			// Past max table age: death is certain.
			return decimal.Zero, nil
		}
		px := one.Sub(qx)
		result = result.Mul(px)
		// Once survival hits zero we can stop.
		if result.IsZero() {
			break
		}
	}

	return result, nil
}
