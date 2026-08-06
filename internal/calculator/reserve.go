package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"reservas/internal/database"
	"reservas/internal/models"
)

// ReserveCalculator is the top-level engine that computes VPPj reserves
// for a policy and its family group. It loads mortality tables on demand
// and delegates flow projection to FlowProjector.
type ReserveCalculator struct {
	mortality *MortalityEngine
	projector *FlowProjector
	mortRepo  *database.MortalityRepository
}

// NewReserveCalculator creates a calculator wired to the given DB repositories.
func NewReserveCalculator(mortRepo *database.MortalityRepository) *ReserveCalculator {
	me := NewMortalityEngine()
	return &ReserveCalculator{
		mortality: me,
		projector: NewFlowProjector(me),
		mortRepo:  mortRepo,
	}
}

// Calculate computes the full reserve for a policy with its family group.
// The rentaAnual is the base annual pension; the discount rate is taken from
// the policy's effective rate (min TM, TC) unless overridden.
func (rc *ReserveCalculator) Calculate(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
) (*FlowResult, error) {

	// Ensure all needed mortality tables are loaded.
	tablesNeeded := rc.collectTables(grupo)
	for _, tableName := range tablesNeeded {
		if err := rc.mortality.EnsureLoaded(rc.mortRepo, tableName); err != nil {
			return nil, fmt.Errorf("loading mortality tables: %w", err)
		}
	}

	discountRate := policy.GetEffectiveDiscountRate()

	result, err := rc.projector.Project(policy, grupo, rentaAnual, discountRate, 0)
	if err != nil {
		return nil, fmt.Errorf("project flows: %w", err)
	}

	return result, nil
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
