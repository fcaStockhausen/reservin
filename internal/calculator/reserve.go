package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/database"
	"github.com/fcaStockhausen/reservin/internal/models"
)

// ReserveCalculator is the top-level engine that computes VPPj reserves
// for a policy and its family group. It loads mortality tables on demand
// and delegates flow projection to FlowProjector.
type ReserveCalculator struct {
	mortality *MortalityEngine
	projector *FlowProjector
	mortRepo  *database.MortalityRepository
	vtdRepo   *database.VTDRepository
	// mejorar controls whether mortality improvement (Circular 2332) is
	// applied. Toggle for sensitivity analysis; default true.
	mejorar bool
	// vtdCache memoizes VTD vectors by "YYYY-MM" so the validator does not
	// re-query the DB for every policy in the same issuance month.
	vtdCache map[string][]decimal.Decimal
}

// NewReserveCalculator creates a calculator wired to the given DB repositories.
func NewReserveCalculator(mortRepo *database.MortalityRepository, vtdRepo ...*database.VTDRepository) *ReserveCalculator {
	me := NewMortalityEngine()
	rc := &ReserveCalculator{
		mortality: me,
		projector: NewFlowProjector(me),
		mortRepo:  mortRepo,
		mejorar:   true,
		vtdCache:  make(map[string][]decimal.Decimal),
	}
	if len(vtdRepo) > 0 {
		rc.vtdRepo = vtdRepo[0]
	}
	return rc
}

// SetMejoramientoEnabled toggles mortality improvement (Circular 2332).
// Disabling it leaves the engine using base qx (no AAx), useful for
// diagnosing the contribution of improvement to the reserve.
func (rc *ReserveCalculator) SetMejoramientoEnabled(enabled bool) {
	rc.mejorar = enabled
}

// LoadVTD installs the VTD discount curve into the projector. The vector's
// rates are indexed by period (1-120); Project consumes them by period k.
// If no VTD is available the projector falls back to flat-rate discounting.
func (rc *ReserveCalculator) LoadVTD() error {
	if rc.vtdRepo == nil {
		return nil
	}
	vector, err := rc.vtdRepo.GetLatestVector()
	if err != nil {
		return err
	}
	return rc.installVTD(vector)
}

// LoadVTDFor installs the VTD curve published for a specific (year, month).
// Returns an error if that vector is not in the database.
func (rc *ReserveCalculator) LoadVTDFor(year, month int) error {
	if rc.vtdRepo == nil {
		return nil
	}
	vector, err := rc.vtdRepo.GetVector(year, month)
	if err != nil {
		return err
	}
	return rc.installVTD(vector)
}

// ClearVTD removes any installed VTD curve, reverting the projector to flat
// discounting with the policy's effective rate.
func (rc *ReserveCalculator) ClearVTD() {
	rc.projector.SetDiscountRates(nil)
}

// SetTasaDescuento overrides the discount rate with a flat rate (e.g. the
// bautizo rate TCj, NCG 318). Takes precedence over VTD and the policy rate.
func (rc *ReserveCalculator) SetTasaDescuento(rate decimal.Decimal) {
	rc.projector.SetTasaDescuento(rate)
}

// ClearTasaDescuento removes the flat-rate override, reverting to VTD/policy
// rate behavior.
func (rc *ReserveCalculator) ClearTasaDescuento() {
	rc.projector.ClearTasaDescuento()
}

// ComputeTCj computes the Tasa de Costo Equivalente (TCj) for the policy per
// NCG 318 Anexo §1: the flat rate (TIR) that reproduces the present value of
// the probabilistic flows when discounted with the VTD curve of the issuance
// month.
//
//	VPPj = Σ_i FP_i × D_vtd(i) = Σ_i FP_i / (1 + TCj)^i
//
// The reserve is then discounted at TCj (flat) for the rest of the policy's
// life, per the "bautizo" principle.
//
// Precondition: the issuance-month VTD curve must be installed (LoadVTDFor /
// LoadVTDForCached) before calling this.
func (rc *ReserveCalculator) ComputeTCj(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
) (decimal.Decimal, error) {
	// Ensure tables + mejoramiento, same as CalculateAt.
	tablesNeeded := rc.collectTables(grupo)
	for _, tableName := range tablesNeeded {
		if err := rc.mortality.EnsureLoaded(rc.mortRepo, tableName); err != nil {
			return decimal.Zero, fmt.Errorf("loading mortality tables: %w", err)
		}
		if rc.mejorar {
			if err := rc.mortality.EnsureMejoramiento(rc.mortRepo, tableName); err != nil {
				return decimal.Zero, fmt.Errorf("loading improvement factors for %s: %w", tableName, err)
			}
		}
	}
	if rc.mejorar && !policy.FechaInicio.IsZero() {
		rc.mortality.SetAñoCálculo(policy.FechaInicio.Year())
	} else if !rc.mejorar {
		rc.mortality.SetAñoCálculo(2020)
	}

	// Proyectar a la emisión (currentYear=0) con el VTD instalado. Como la
	// tasa flat override no está seteada y el VTD está instalado, Project
	// descuenta cada flujo con su tasa VTD, así que TotalReserve + adjustment
	// es el VPPj objetivo (la mensualización 11/24 se resta una vez y no
	// depende de la tasa, se re-agrega para el TIR).
	res, err := rc.projector.Project(policy, grupo, rentaAnual, decimal.Zero, 0)
	if err != nil {
		return decimal.Zero, fmt.Errorf("compute tcj: %w", err)
	}
	adjustment := rentaAnual.Mul(decimal.NewFromFloat(11.0 / 24.0))
	target := res.TotalReserve.Add(adjustment)
	if target.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, fmt.Errorf("compute tcj: zero present value")
	}

	// Bisección sobre TCj: Σ FP_i / (1+r)^i = target. PV decrece con r, así que
	// si pv(mid) > target la raíz está más arriba. 40 iteraciones dan ~1e-12.
	lo := decimal.NewFromFloat(0.0001)
	hi := decimal.NewFromFloat(0.25)
	for i := 0; i < 40; i++ {
		mid := lo.Add(hi).Div(decimal.NewFromInt(2))
		pv := pvFlows(res.Flows, mid)
		if pv.GreaterThan(target) {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo.Add(hi).Div(decimal.NewFromInt(2)), nil
}

// pvFlows returns Σ FP_i / (1+r)^i for a set of flows. The discount factor is
// accumulated with multiplications (flows are in ascending period order).
func pvFlows(flows []MemberFlow, rate decimal.Decimal) decimal.Decimal {
	one := decimal.NewFromInt(1)
	inv := one.Div(one.Add(rate))
	pv := decimal.Zero
	df := one
	last := 0
	for _, f := range flows {
		for j := last; j < f.Period; j++ {
			df = df.Mul(inv)
		}
		last = f.Period
		pv = pv.Add(f.FlowAmount.Mul(df))
	}
	return pv
}

// LoadVTDForCached installs the VTD curve for a given issuance month using an
// in-memory cache keyed by "YYYY-MM". When the same month recurs across many
// policies (typical in the validator), only the first call hits the DB.
// Returns true when a curve was installed, false when the vector is missing
// (caller falls back to flat rate).
func (rc *ReserveCalculator) LoadVTDForCached(year, month int) bool {
	if rc.vtdRepo == nil {
		return false
	}
	key := fmt.Sprintf("%04d-%02d", year, month)
	if rates, ok := rc.vtdCache[key]; ok {
		if rates == nil {
			return false // cached negative lookup
		}
		rc.projector.SetDiscountRates(rates)
		return true
	}
	vector, err := rc.vtdRepo.GetVector(year, month)
	if err != nil || vector == nil || len(vector.Rates) == 0 {
		rc.vtdCache[key] = nil // mark missing
		return false
	}
	rates := rc.convertVTD(vector)
	rc.vtdCache[key] = rates
	rc.projector.SetDiscountRates(rates)
	return true
}

// convertVTD converts a VTD vector (periods 1..120) into a 0-indexed slice.
func (rc *ReserveCalculator) convertVTD(vector *models.VTDVector) []decimal.Decimal {
	maxPeriod := 0
	for _, p := range vector.Rates {
		if p.Period > maxPeriod {
			maxPeriod = p.Period
		}
	}
	rates := make([]decimal.Decimal, maxPeriod)
	for _, p := range vector.Rates {
		if p.Period >= 1 && p.Period <= maxPeriod {
			rates[p.Period-1] = p.Rate
		}
	}
	return rates
}

// installVTD converts a VTD vector (periods 1..120) into a 0-indexed slice for
// the projector and installs it.
func (rc *ReserveCalculator) installVTD(vector *models.VTDVector) error {
	if vector == nil || len(vector.Rates) == 0 {
		return nil
	}
	maxPeriod := 0
	for _, p := range vector.Rates {
		if p.Period > maxPeriod {
			maxPeriod = p.Period
		}
	}
	rates := make([]decimal.Decimal, maxPeriod)
	for _, p := range vector.Rates {
		if p.Period >= 1 && p.Period <= maxPeriod {
			rates[p.Period-1] = p.Rate
		}
	}
	rc.projector.SetDiscountRates(rates)
	return nil
}

// Calculate computes the full reserve for a policy with its family group.
// The rentaAnual is the base annual pension; the discount rate is taken from
// the policy's effective rate (min TM, TC) unless overridden.
func (rc *ReserveCalculator) Calculate(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
) (*FlowResult, error) {
	return rc.CalculateAt(policy, grupo, rentaAnual, 0)
}

// CalculateAt computes the full reserve for a policy valued at a given
// projection offset. currentYear is the number of years already elapsed since
// inception: the annuity is re-valued from the physical age reached then
// (EdadContratacion + currentYear), so a policy valued "today" uses the current
// age, not the age at issue.
func (rc *ReserveCalculator) CalculateAt(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
	currentYear int,
) (*FlowResult, error) {

	// Ensure all needed mortality tables are loaded.
	tablesNeeded := rc.collectTables(grupo)
	for _, tableName := range tablesNeeded {
		if err := rc.mortality.EnsureLoaded(rc.mortRepo, tableName); err != nil {
			return nil, fmt.Errorf("loading mortality tables: %w", err)
		}
		// Load improvement factors (Circular 2332) when enabled and present.
		if rc.mejorar {
			if err := rc.mortality.EnsureMejoramiento(rc.mortRepo, tableName); err != nil {
				return nil, fmt.Errorf("loading improvement factors for %s: %w", tableName, err)
			}
		}
	}

	// Set the valuation year for mortality improvement: policy start + elapsed
	// years. E.g. contract 2001 valued in 2025 -> añoCálculo = 2025.
	if rc.mejorar && !policy.FechaInicio.IsZero() {
		rc.mortality.SetAñoCálculo(policy.FechaInicio.Year() + currentYear)
	} else if !rc.mejorar {
		rc.mortality.SetAñoCálculo(2020) // base year: improvement is a no-op
	}

	discountRate := policy.GetEffectiveDiscountRate()

	result, err := rc.projector.Project(policy, grupo, rentaAnual, discountRate, currentYear)
	if err != nil {
		return nil, fmt.Errorf("project flows: %w", err)
	}

	return result, nil
}

// TpxFor exposes the mortality engine's survival probability to external
// callers (e.g. survivor-only reserve valuation in the RIS validator).
func (rc *ReserveCalculator) TpxFor(tableName string, edad, t int) (decimal.Decimal, error) {
	return rc.mortality.Tpx(tableName, edad, t)
}

// QxFor exposes the mortality engine's one-year death probability.
func (rc *ReserveCalculator) QxFor(tableName string, edad int) (decimal.Decimal, error) {
	return rc.mortality.Qx(tableName, edad)
}

// RateAt returns the discount rate for period k, honoring the installed VTD
// curve (via the projector) and falling back to the flat effective rate.
func (rc *ReserveCalculator) RateAt(policy models.Policy, k int) decimal.Decimal {
	return rc.projector.rateFor(k, policy.GetEffectiveDiscountRate())
}

// EnsureGroupTables loads every mortality table referenced by the family group.
func (rc *ReserveCalculator) EnsureGroupTables(grupo *models.GrupoFamiliar) error {
	for _, name := range rc.collectTables(grupo) {
		if err := rc.mortality.EnsureLoaded(rc.mortRepo, name); err != nil {
			return fmt.Errorf("loading table %s: %w", name, err)
		}
	}
	return nil
}

// collectTables returns the set of mortality table names used by the group.
func (rc *ReserveCalculator) collectTables(grupo *models.GrupoFamiliar) []string {
	seen := make(map[string]bool)
	var tables []string

	if grupo.Causante != nil && grupo.Causante.TablaAsignada != "" {
		t := grupo.Causante.TablaAsignada
		if !seen[t] {
			seen[t] = true
			tables = append(tables, t)
		}
	}

	for _, b := range grupo.Beneficiarios {
		if b.TablaAsignada != "" && !seen[b.TablaAsignada] {
			seen[b.TablaAsignada] = true
			tables = append(tables, b.TablaAsignada)
		}
	}

	return tables
}
