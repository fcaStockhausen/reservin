package calculator

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/models"
)

// MemberFlow represents a single projected cash flow for one family member
// in one period. This is the atomic unit for Excel export.
type MemberFlow struct {
	Period         int             `json:"period"`
	Date           time.Time       `json:"date"`
	MemberRol      string          `json:"member_rol"`
	MemberSex      string          `json:"member_sex"`
	MemberTable    string          `json:"member_table"`
	MemberAgeAtT   int             `json:"member_age_at_t"`
	RentaBase      decimal.Decimal `json:"renta_base"`     // annual pension this member is entitled to
	PctRenta       decimal.Decimal `json:"pct_renta"`      // share 0..1
	SurvivalProb   decimal.Decimal `json:"survival_prob"`  // tpx for this member
	CausanteAlive  decimal.Decimal `json:"causante_alive"` // tpx for causante at same period
	FlowProb       decimal.Decimal `json:"flow_prob"`      // combined probability of payment
	FlowAmount     decimal.Decimal `json:"flow_amount"`    // expected amount = renta × pct × flow_prob
	DiscountFactor decimal.Decimal `json:"discount_factor"`
	PresentValue   decimal.Decimal `json:"present_value"`
}

// FlowResult holds the complete flow projection for a policy.
type FlowResult struct {
	PolicyID        int
	PolicyNumber    string
	Flows           []MemberFlow
	TotalReserve    decimal.Decimal
	DiscountRate    decimal.Decimal
	Periods         int
	CalculationDate time.Time
}

// FlowProjector projects probabilistic cash flows for a policy's family group.
type FlowProjector struct {
	mortality *MortalityEngine
}

// NewFlowProjector creates a projector with the given mortality engine.
func NewFlowProjector(me *MortalityEngine) *FlowProjector {
	return &FlowProjector{mortality: me}
}

// Project generates per-member, per-period cash flows for the family group.
//
// Model:
//
//	Causante:   flow(t) = R × tpx_causante(t)
//	Beneficiary: flow(t) = R × pct × tpx_ben(t) × [1 - tpx_causante(t)]
//
// The beneficiary receives their share only after the causante has died.
// Projection runs until all members are certainly dead (max table age reached).
func (fp *FlowProjector) Project(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
	discountRate decimal.Decimal,
) (*FlowResult, error) {
	if grupo.Causante == nil {
		return nil, fmt.Errorf("policy %d has no causante", policy.ID)
	}

	causante := grupo.Causante
	maxAge := 110 // CMF tables go to age 110

	// Determine projection horizon: from causante age to max table age.
	horizon := maxAge - causante.EdadContratacion
	if horizon <= 0 {
		horizon = 1
	}

	// For each beneficiary, also consider their horizon.
	for _, b := range grupo.Beneficiarios {
		benHorizon := maxAge - b.EdadContratacion
		if benHorizon > horizon {
			horizon = benHorizon
		}
	}

	one := decimal.NewFromInt(1)
	var flows []MemberFlow
	totalReserve := decimal.Zero

	startDate := policy.FechaInicio

	for t := 0; t <= horizon; t++ {
		date := startDate.AddDate(t, 0, 0)
		discountFactor := computeDiscountFactor(discountRate, t)

		// --- Causante flow ---
		tpxCausante, err := fp.mortality.Tpx(causante.TablaAsignada, causante.EdadContratacion, t)
		if err != nil {
			return nil, fmt.Errorf("causante tpx at t=%d: %w", t, err)
		}
		if t == 0 {
			tpxCausante = one
		}

		if tpxCausante.GreaterThan(decimal.Zero) {
			cf := MemberFlow{
				Period:         t,
				Date:           date,
				MemberRol:      string(causante.Rol),
				MemberSex:      causante.Sexo,
				MemberTable:    causante.TablaAsignada,
				MemberAgeAtT:   causante.EdadContratacion + t,
				RentaBase:      rentaAnual,
				PctRenta:       one,
				SurvivalProb:   tpxCausante,
				CausanteAlive:  tpxCausante,
				FlowProb:       tpxCausante,
				FlowAmount:     rentaAnual.Mul(tpxCausante),
				DiscountFactor: discountFactor,
			}
			cf.PresentValue = cf.FlowAmount.Mul(discountFactor)
			flows = append(flows, cf)
			totalReserve = totalReserve.Add(cf.PresentValue)
		}

		// --- Beneficiary flows (survivor pension) ---
		for _, b := range grupo.Beneficiarios {
			if b.Estado != "ACTIVO" {
				continue
			}

			tpxBen, err := fp.mortality.Tpx(b.TablaAsignada, b.EdadContratacion, t)
			if err != nil {
				return nil, fmt.Errorf("beneficiary %s tpx at t=%d: %w", b.Rol, t, err)
			}
			if t == 0 {
				tpxBen = one
			}
			if tpxBen.IsZero() {
				continue
			}

			// Beneficiary receives pension only if causante is dead by period t.
			probCausanteDead := one.Sub(tpxCausante)
			if probCausanteDead.LessThanOrEqual(decimal.Zero) {
				continue
			}

			flowProb := tpxBen.Mul(probCausanteDead)
			if flowProb.IsZero() {
				continue
			}

			benRenta := rentaAnual.Mul(b.PorcentajeRenta)

			bf := MemberFlow{
				Period:         t,
				Date:           date,
				MemberRol:      string(b.Rol),
				MemberSex:      b.Sexo,
				MemberTable:    b.TablaAsignada,
				MemberAgeAtT:   b.EdadContratacion + t,
				RentaBase:      benRenta,
				PctRenta:       b.PorcentajeRenta,
				SurvivalProb:   tpxBen,
				CausanteAlive:  tpxCausante,
				FlowProb:       flowProb,
				FlowAmount:     benRenta.Mul(flowProb),
				DiscountFactor: discountFactor,
			}
			bf.PresentValue = bf.FlowAmount.Mul(discountFactor)
			flows = append(flows, bf)
			totalReserve = totalReserve.Add(bf.PresentValue)
		}
	}

	return &FlowResult{
		PolicyID:        policy.ID,
		PolicyNumber:    policy.NumeroPoliza,
		Flows:           flows,
		TotalReserve:    totalReserve,
		DiscountRate:    discountRate,
		Periods:         horizon,
		CalculationDate: time.Now(),
	}, nil
}

// computeDiscountFactor returns (1 + rate)^(-t).
// At t=0 the factor is 1.0.
func computeDiscountFactor(rate decimal.Decimal, t int) decimal.Decimal {
	if t == 0 {
		return decimal.NewFromInt(1)
	}
	base := decimal.NewFromInt(1).Add(rate)
	// base^(-t) = 1 / base^t
	pow := decimal.NewFromInt(1)
	for i := 0; i < t; i++ {
		pow = pow.Mul(base)
	}
	return decimal.NewFromInt(1).Div(pow)
}
