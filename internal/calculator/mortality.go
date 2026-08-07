package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/database"
)

// MortalityEngine provides survival probability lookups (tpx) from mortality
// tables loaded directly from the database. Each table is cached as a map of
// edad → qx, and survival probabilities are computed on demand.
type MortalityEngine struct {
	tables map[string]map[int]decimal.Decimal // tableName -> edad -> qx
	// mejoramiento: tableName -> edad -> año -> factor AAx (Circular 2332).
	// When loaded, Qx/Tpx apply the improvement formula of Nota Técnica N°9:
	//   qx,año = qx,2020 × Π_{t=2021}^{año} (1 - AAx,t)
	mejoramiento map[string]map[int]map[int]decimal.Decimal
	// añoCálculo is the valuation year used for improvement (default 2020 = no
	// improvement). Set via SetAñoCálculo.
	añoCálculo int
}

// NewMortalityEngine creates an empty engine.
func NewMortalityEngine() *MortalityEngine {
	return &MortalityEngine{
		tables:       make(map[string]map[int]decimal.Decimal),
		mejoramiento: make(map[string]map[int]map[int]decimal.Decimal),
		añoCálculo:   2020,
	}
}

// SetAñoCálculo sets the valuation year for mortality improvement.
func (me *MortalityEngine) SetAñoCálculo(año int) {
	me.añoCálculo = año
}

// LoadMejoramiento loads the annual improvement factors for a table.
func (me *MortalityEngine) LoadMejoramiento(repo *database.MortalityRepository, tableName string) error {
	mejRepo := database.NewFactorMejoramientoRepository(repo.DB())
	factors, err := mejRepo.GetFactors(tableName)
	if err != nil {
		return err
	}
	if len(factors) == 0 {
		return nil
	}
	byAge := make(map[int]map[int]decimal.Decimal)
	for _, f := range factors {
		if byAge[f.Edad] == nil {
			byAge[f.Edad] = make(map[int]decimal.Decimal)
		}
		byAge[f.Edad][f.Año] = f.FactorAA
	}
	me.mejoramiento[tableName] = byAge
	return nil
}

// EnsureMejoramiento loads improvement factors if not already cached.
func (me *MortalityEngine) EnsureMejoramiento(repo *database.MortalityRepository, tableName string) error {
	if _, ok := me.mejoramiento[tableName]; ok {
		return nil
	}
	return me.LoadMejoramiento(repo, tableName)
}

// mejoradaQx applies the mortality improvement factor for the valuation year.
func (me *MortalityEngine) mejoradaQx(tableName string, edad, añoBase int) (decimal.Decimal, error) {
	qx, err := me.Qx(tableName, edad)
	if err != nil {
		return decimal.Zero, err
	}
	if me.añoCálculo <= añoBase {
		return qx, nil
	}
	byAge, ok := me.mejoramiento[tableName]
	if !ok {
		return qx, nil // no improvement data -> base qx
	}
	añoFactors, ok := byAge[edad]
	if !ok {
		return qx, nil
	}
	// qx,año = qx,2020 × Π_{t=añoBase+1}^{añoCálculo} (1 - AAx,t)
	result := qx
	for t := añoBase + 1; t <= me.añoCálculo; t++ {
		aax, ok := añoFactors[t]
		if !ok {
			break // no factor for this year -> stop improving
		}
		result = result.Mul(decimal.NewFromInt(1).Sub(aax))
	}
	return result, nil
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
//
// A missing table is a hard error (caller must propagate). An age beyond the
// table's max is treated as certain death (qx=1), consistent with Tpx: this is
// the standard actuarial convention at the closing age of a life table.
func (me *MortalityEngine) Qx(tableName string, edad int) (decimal.Decimal, error) {
	t, ok := me.tables[tableName]
	if !ok {
		return decimal.Zero, fmt.Errorf("table %s not loaded", tableName)
	}
	qx, ok := t[edad]
	if !ok {
		return decimal.NewFromInt(1), nil // past closing age: certain death
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
