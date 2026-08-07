package calculator

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/database"
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
	RentaBase      decimal.Decimal `json:"renta_base"`     // annual pension entitled this period
	PctRenta       decimal.Decimal `json:"pct_renta"`      // share 0..1
	SurvivalProb   decimal.Decimal `json:"survival_prob"`  // tpx for this member
	CausanteAlive  decimal.Decimal `json:"causante_alive"` // tpx causante same period
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
	// discountRates, when set, is the VTD curve used per projection period
	// (period k discounted at discountRates[k]). When empty, a flat rate is
	// used (the discountRate passed to Project).
	discountRates []decimal.Decimal
}

// NewFlowProjector creates a projector with the given mortality engine.
func NewFlowProjector(me *MortalityEngine) *FlowProjector {
	return &FlowProjector{mortality: me}
}

// SetDiscountRates installs a per-period discount curve (VTD). Period k is
// discounted with discountRates[k]; periods beyond the slice fall back to the
// last rate. Calling with an empty slice reverts to flat-rate discounting.
func (fp *FlowProjector) SetDiscountRates(rates []decimal.Decimal) {
	fp.discountRates = rates
}

// rateFor returns the discount rate applied to period k: the VTD curve when
// available, otherwise the flat discountRate.
func (fp *FlowProjector) rateFor(k int, discountRate decimal.Decimal) decimal.Decimal {
	if len(fp.discountRates) == 0 {
		return discountRate
	}
	if k < len(fp.discountRates) {
		return fp.discountRates[k]
	}
	return fp.discountRates[len(fp.discountRates)-1]
}

// SetAñoCálculo sets the valuation year for mortality improvement on the engine.
func (fp *FlowProjector) SetAñoCálculo(año int) {
	fp.mortality.SetAñoCálculo(año)
}

// EnsureMejoramiento loads the improvement factors for a table into the engine.
func (fp *FlowProjector) EnsureMejoramiento(repo *database.MortalityRepository, tableName string) error {
	return fp.mortality.EnsureMejoramiento(repo, tableName)
}

// qxMejorada returns the death probability for an age, applying the mortality
// improvement factor (Circular 2332) when the table carries one and the engine
// has a valuation year > the table base year (2020).
func (fp *FlowProjector) qxMejorada(tableName string, edad int) (decimal.Decimal, error) {
	return fp.mortality.mejoradaQx(tableName, edad, 2020)
}

// Project generates per-member, per-period cash flows for the family group.
//
// Model:
//
//	Causante:   flow(t) = R × tpx_causante(t)
//	Beneficiary: flow(t) = R × pct × tpx_ben(t) × [1 - tpx_causante(t)]
//
// The beneficiary has a survivor pension only after the causante has died.
// currentYear is the valuation offset in years: the annuity is re-valued from
// the physical age reached after that many years, so the reserve is dynamic
// (it decreases as payments lapse) rather than re-anchored at inception.
//
// Performance: survival probabilities are accumulated in a single forward pass
// (O(horizon) per member) instead of recomputing each period's tpx from scratch
// (O(horizon²)). The discount factor is also precomputed as a recurrence.
func (fp *FlowProjector) Project(
	policy models.Policy,
	grupo *models.GrupoFamiliar,
	rentaAnual decimal.Decimal,
	discountRate decimal.Decimal,
	currentYear int,
) (*FlowResult, error) {
	if grupo.Causante == nil {
		return nil, fmt.Errorf("policy %d has no causante", policy.ID)
	}

	causante := grupo.Causante
	maxAge := 110

	causanteStartAge := causante.EdadContratacion + currentYear
	causanteTable := causante.TablaAsignada

	// Período garantizado (modalidad 3xxx/4xxx): months of guaranteed payments.
	pgYears := 0
	if pgMonths := models.GarantizedMonths(policy.ModalidadRenta); pgMonths > 0 {
		pgYears = (pgMonths + 11) / 12 // ceil to whole years
	}

	horizon := maxAge - causanteStartAge
	if horizon <= 0 {
		horizon = 1
	}

	// Gather active beneficiaries.
	type benRec struct {
		b        *models.Beneficiario
		startAge int
		table    string
		pct      decimal.Decimal
	}
	var beneficiaries []benRec
	for _, b := range grupo.Beneficiarios {
		if b.Estado != "ACTIVO" {
			continue
		}
		sa := b.EdadContratacion + currentYear
		bh := maxAge - sa
		if bh > horizon {
			horizon = bh
		}
		beneficiaries = append(beneficiaries, benRec{
			b:        b,
			startAge: sa,
			table:    b.TablaAsignada,
			pct:      b.PorcentajeRenta,
		})
	}

	one := decimal.NewFromInt(1)
	zero := decimal.Zero
	var flows []MemberFlow
	totalReserve := decimal.Zero
	startDate := policy.FechaInicio

	// Discount recurrence: df[k] = df[k-1] × 1/(1+rate[k]), df[0]=1.
	// With the VTD curve installed the rate varies by period; otherwise it is
	// the flat discountRate. The 1/(1+r) values are precomputed once so the
	// loop only does multiplications.
	discountInv := make([]decimal.Decimal, horizon+1)
	for k := 1; k <= horizon; k++ {
		discountInv[k] = one.Div(one.Add(fp.rateFor(k, discountRate)))
	}

	// Cumulative survival from valuation age.
	cSurv := one
	benSurvs := make([]decimal.Decimal, len(beneficiaries))
	for i := range beneficiaries {
		benSurvs[i] = one
	}

	df := one

	for k := 0; k <= horizon; k++ {
		date := startDate.AddDate(currentYear+k, 0, 0)

		// --- Evolve survivals and discount for k > 0 ---
		if k > 0 {
			// Causante: multiply by px = 1 - qx(x + k - 1).
			// qxMejorada only errors on a missing table (a hard data error);
			// ages past the closing age return qx=1 (certain death) and so
			// drive cSurv to zero, which we propagate as zero.
			if cSurv.GreaterThan(zero) {
				qxC, err := fp.qxMejorada(causanteTable, causanteStartAge+k-1)
				if err != nil {
					return nil, fmt.Errorf("causante table %s: %w", causanteTable, err)
				}
				cSurv = cSurv.Mul(one.Sub(qxC))
			}

			// Beneficiaries.
			for i, rec := range beneficiaries {
				if benSurvs[i].IsZero() {
					continue
				}
				qxB, err := fp.qxMejorada(rec.table, rec.startAge+k-1)
				if err != nil {
					return nil, fmt.Errorf("beneficiary table %s: %w", rec.table, err)
				}
				benSurvs[i] = benSurvs[i].Mul(one.Sub(qxB))
			}

			df = df.Mul(discountInv[k])
		}

		// --- Causante flow (rent vitalicio) ---
		// During the guaranteed period the payment is certain (prob=1) even if
		// the causante dies; after it the flow is contingent on survival.
		causanteProb := cSurv
		if k < pgYears {
			causanteProb = one
		}
		if causanteProb.GreaterThan(zero) {
			cf := MemberFlow{
				Period:         k,
				Date:           date,
				MemberRol:      string(causante.Rol),
				MemberSex:      causante.Sexo,
				MemberTable:    causanteTable,
				MemberAgeAtT:   causanteStartAge + k,
				RentaBase:      rentaAnual,
				PctRenta:       one,
				SurvivalProb:   cSurv,
				CausanteAlive:  cSurv,
				FlowProb:       causanteProb,
				FlowAmount:     rentaAnual.Mul(causanteProb),
				DiscountFactor: df,
			}
			cf.PresentValue = cf.FlowAmount.Mul(df)
			flows = append(flows, cf)
			totalReserve = totalReserve.Add(cf.PresentValue)
		}

		// --- Beneficiary flows (survivor pension) ---
		// Survivors receive only after the guaranteed period expires; during PG
		// the guaranteed payments go to the estate/designated beneficiaries.
		if k >= pgYears {
			probCausanteDead := one.Sub(cSurv)
			if probCausanteDead.GreaterThan(zero) {
				for i, rec := range beneficiaries {
					if benSurvs[i].IsZero() {
						continue
					}
					flowProb := benSurvs[i].Mul(probCausanteDead)
					if flowProb.IsZero() {
						continue
					}
					benRenta := rentaAnual.Mul(rec.pct)
					bf := MemberFlow{
						Period:         k,
						Date:           date,
						MemberRol:      string(rec.b.Rol),
						MemberSex:      rec.b.Sexo,
						MemberTable:    rec.table,
						MemberAgeAtT:   rec.startAge + k,
						RentaBase:      benRenta,
						PctRenta:       rec.pct,
						SurvivalProb:   benSurvs[i],
						CausanteAlive:  cSurv,
						FlowProb:       flowProb,
						FlowAmount:     benRenta.Mul(flowProb),
						DiscountFactor: df,
					}
					bf.PresentValue = bf.FlowAmount.Mul(df)
					flows = append(flows, bf)
					totalReserve = totalReserve.Add(bf.PresentValue)
				}
			}
		}
	}

	// Mensualización adjustment (Nota Técnica N°9, equation 7): the annual
	// annuity factor overcounts because monthly payments occur mid-period. The
	// SPensiones CNU formula subtracts 11/24 of one pension from the causante's
	// own annuity value. Applied only to the causante term.
	adjustment := rentaAnual.Mul(decimal.NewFromFloat(11.0 / 24.0))
	totalReserve = totalReserve.Sub(adjustment)

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

// computeDiscountFactor returns (1 + rate)^(-k). At k=0 the factor is 1.0.
func computeDiscountFactor(rate decimal.Decimal, k int) decimal.Decimal {
	if k == 0 {
		return decimal.NewFromInt(1)
	}
	base := decimal.NewFromInt(1).Add(rate)
	pow := decimal.NewFromInt(1)
	for i := 0; i < k; i++ {
		pow = pow.Mul(base)
	}
	return decimal.NewFromInt(1).Div(pow)
}
