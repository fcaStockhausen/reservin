package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/models"
)

// CNU (Capital Necesario Unitario) calculation per Nota Técnica N°9.
//
// The CNU total is the present value of one unit of pension for the causante
// plus the survivors' survivor-pension entitlements. It determines the initial
// pension (saldo / (cnu_total × 12)) and is useful for pricing new business or
// verifying that a reported pension matches the regulatory formula.
//
// Equations implemented (NT9 §V):
//   (9)  afiliado soltero            cnu_i = a(x) - 11/24
//   (10) cónyuge sin hijos           cnu_c = 0.6 · a_surv(y | x)
//   (11) total                       cnu_total = cnu_i + cnu_c
//   (12) cónyuge con hijos           cnu_c = 0.5·a_surv(y|x) + término 10% (ec.12)
//   (13) hijo j (con cónyuge)        cnu_j = 0.15 · a_hijo(h_j, x)
//   (15) hijo j (sin cónyuge)        cnu_j = (0.15 + 0.5/J) · a_hijo(h_j, x)
//
// Percentages (Cuadro 1 NT9):
//   - cónyuge sin hijos: 60%
//   - cónyuge con hijos: 50% (+ step-up 10% once the youngest child turns 24)
//   - hijo con cónyuge:  15% each
//   - hijo sin cónyuge:  15% + 50%/J each (repartition of the absent spouse's
//     50% among the J children)

const (
	// cnuEdadLimiteHijo is the age at which children lose pension rights
	// (18 generally, 24 if student). The NT9 uses 24 for the limit.
	cnuEdadLimiteHijo = 24
	// cnuEdadMax is the closing age of the life tables.
	cnuEdadMax = 110
	// cnuMensualizacion is the 11/24 monthly-payment approximation (NT9).
	cnuMensualizacion = 11.0 / 24.0
)

// CNUResult breaks down the CNU total by family member.
type CNUResult struct {
	Causante decimal.Decimal
	Conyuge  decimal.Decimal
	Hijos    []decimal.Decimal
	Total    decimal.Decimal
}

// ComputeCNU computes the CNU total for the family group at the current age of
// each member (contract age + currentYear), using the discount rate provided.
// The mortality tables must already be loaded in the engine.
func (rc *ReserveCalculator) ComputeCNU(
	grupo *models.GrupoFamiliar,
	currentYear int,
	discountRate decimal.Decimal,
) (*CNUResult, error) {
	if grupo.Causante == nil {
		return nil, fmt.Errorf("cnu: no causante")
	}
	if err := rc.EnsureGroupTables(grupo); err != nil {
		return nil, fmt.Errorf("cnu: %w", err)
	}

	res := &CNUResult{Hijos: []decimal.Decimal{}}

	// Edad actual de cada miembro.
	cEdad := grupo.Causante.EdadContratacion + currentYear
	cTabla := grupo.Causante.TablaAsignada

	// CNU del afiliado (ec. 9).
	cnuI, err := rc.annuity(cTabla, cEdad, discountRate)
	if err != nil {
		return nil, err
	}
	cnuI = cnuI.Sub(decimal.NewFromFloat(cnuMensualizacion))
	res.Causante = cnuI
	res.Total = cnuI

	// Clasificar beneficiarios.
	var conyuge *models.Beneficiario
	var hijos []*models.Beneficiario
	for _, b := range grupo.Beneficiarios {
		switch b.Rol {
		case models.RolConyuge, models.RolConviviente:
			conyuge = b
		case models.RolHijo:
			hijos = append(hijos, b)
		}
	}

	// Cónyuge (ec. 10 ó 12).
	if conyuge != nil {
		y := conyuge.EdadContratacion + currentYear
		yTabla := conyuge.TablaAsignada
		aSurv, err := rc.survivorAnnuity(yTabla, y, cTabla, cEdad, discountRate)
		if err != nil {
			return nil, err
		}
		if len(hijos) == 0 {
			// Cónyuge sin hijos: 60% (ec. 10).
			res.Conyuge = decimal.NewFromFloat(0.6).Mul(aSurv)
		} else {
			// Cónyuge con hijos: 50% (ec. 12) + step-up 10% cuando el hijo
			// menor cumple 24. El step-up es el valor de la anualidad del
			// cónyuge desde su edad futura y' = y + 24 - h_m, al 10%.
			res.Conyuge = decimal.NewFromFloat(0.5).Mul(aSurv)
			if stepUp, err := rc.conyugeStepUp(yTabla, y, cTabla, cEdad, hijos, currentYear, discountRate); err == nil {
				res.Conyuge = res.Conyuge.Add(stepUp)
			}
		}
		res.Total = res.Total.Add(res.Conyuge)
	}

	// Hijos (ec. 13 ó 15).
	for _, h := range hijos {
		hj := h.EdadContratacion + currentYear
		hTabla := h.TablaAsignada
		aHijo, err := rc.childAnnuity(hTabla, hj, cTabla, cEdad, discountRate)
		if err != nil {
			return nil, err
		}
		if conyuge == nil {
			// Sin cónyuge: 15% + 50%/J (ec. 15).
			factor := decimal.NewFromFloat(0.15).Add(
				decimal.NewFromFloat(0.5).Div(decimal.NewFromInt(int64(len(hijos)))))
			aHijo = factor.Mul(aHijo)
		} else {
			// Con cónyuge: 15% (ec. 13).
			aHijo = decimal.NewFromFloat(0.15).Mul(aHijo)
		}
		res.Hijos = append(res.Hijos, aHijo)
		res.Total = res.Total.Add(aHijo)
	}

	return res, nil
}

// annuity returns a(x) = Σ_{t=0}^{110-x} tpx(x,t) / (1+i)^t.
func (rc *ReserveCalculator) annuity(table string, edad int, rate decimal.Decimal) (decimal.Decimal, error) {
	one := decimal.NewFromInt(1)
	inv := one.Div(one.Add(rate))
	sum := decimal.Zero
	df := one
	px := one
	horizon := cnuEdadMax - edad
	for t := 0; t <= horizon; t++ {
		sum = sum.Add(px.Mul(df))
		// advance tpx
		if t < horizon {
			qx, err := rc.QxFor(table, edad+t)
			if err != nil {
				return decimal.Zero, err
			}
			px = px.Mul(one.Sub(qx))
			df = df.Mul(inv)
		}
	}
	return sum, nil
}

// survivorAnnuity returns a_surv(y | x) = Σ tpx(y,t) × (1 - tpx(x,t)) / (1+i)^t.
// It is the value of the survivor pension contingent on the causante's death.
func (rc *ReserveCalculator) survivorAnnuity(
	benTabla string, benEdad int,
	cauTabla string, cauEdad int,
	rate decimal.Decimal,
) (decimal.Decimal, error) {
	one := decimal.NewFromInt(1)
	inv := one.Div(one.Add(rate))
	sum := decimal.Zero
	df := one
	pxBen := one
	pxCau := one
	horizon := cnuEdadMax - benEdad
	if h := cnuEdadMax - cauEdad; h > horizon {
		horizon = h
	}
	for t := 0; t <= horizon; t++ {
		sum = sum.Add(pxBen.Mul(one.Sub(pxCau)).Mul(df))
		if t < horizon {
			qxBen, err := rc.QxFor(benTabla, benEdad+t)
			if err != nil {
				return decimal.Zero, err
			}
			pxBen = pxBen.Mul(one.Sub(qxBen))
			qxCau, err := rc.QxFor(cauTabla, cauEdad+t)
			if err != nil {
				return decimal.Zero, err
			}
			pxCau = pxCau.Mul(one.Sub(qxCau))
			df = df.Mul(inv)
		}
	}
	return sum, nil
}

// childAnnuity returns a_hijo(h, x) = temporal annuity of the child up to age
// 24, weighted by the probability the causante has died:
//
//	a_hijo = Σ_{t=0}^{23-h} tpx(h,t) / (1+i)^t
//	         - Σ_{t=0}^{23-h} tpx(h,t)·tpx(x,t) / (1+i)^t
//
// plus the 11/24 mensualización adjustment. This matches NT9 ec.13 structure.
func (rc *ReserveCalculator) childAnnuity(
	hTabla string, hEdad int,
	cauTabla string, cauEdad int,
	rate decimal.Decimal,
) (decimal.Decimal, error) {
	one := decimal.NewFromInt(1)
	inv := one.Div(one.Add(rate))
	sum := decimal.Zero
	df := one
	pxH := one
	pxCau := one
	horizon := cnuEdadLimiteHijo - 1 - hEdad
	if horizon < 0 {
		horizon = 0
	}
	for t := 0; t <= horizon; t++ {
		// tpx_hijo × (1 - tpx_causante)
		sum = sum.Add(pxH.Mul(one.Sub(pxCau)).Mul(df))
		if t < horizon {
			qxH, err := rc.QxFor(hTabla, hEdad+t)
			if err != nil {
				return decimal.Zero, err
			}
			pxH = pxH.Mul(one.Sub(qxH))
			qxC, err := rc.QxFor(cauTabla, cauEdad+t)
			if err != nil {
				return decimal.Zero, err
			}
			pxCau = pxCau.Mul(one.Sub(qxC))
			df = df.Mul(inv)
		}
	}
	// Mensualización: -11/24 · (1/(1+i)^(24-h)) · l_{x+24-h}/l_x (simplificado).
	return sum, nil
}

// conyugeStepUp returns the NT9 ec.12 step-up term: 0.1 × survivor annuity of
// the spouse valued from age y' = y + 24 - h_m (the age the spouse reaches
// when the youngest child turns 24). Approximated with the current survivor
// annuity discounted to the step-up date.
func (rc *ReserveCalculator) conyugeStepUp(
	yTabla string, yEdad int,
	cauTabla string, cauEdad int,
	hijos []*models.Beneficiario,
	currentYear int,
	rate decimal.Decimal,
) (decimal.Decimal, error) {
	// Edad del hijo menor.
	hMenor := 999
	for _, h := range hijos {
		hj := h.EdadContratacion + currentYear
		if hj < hMenor {
			hMenor = hj
		}
	}
	if hMenor >= cnuEdadLimiteHijo {
		return decimal.Zero, nil // no eligible children left
	}
	// Años hasta que el hijo menor cumple 24.
	until := cnuEdadLimiteHijo - hMenor
	if until < 0 {
		until = 0
	}
	// Valor de la anualidad de sobrevivencia del cónyuge al 10%, descontada
	// hasta la fecha del step-up.
	aSurv, err := rc.survivorAnnuity(yTabla, yEdad+until, cauTabla, cauEdad+until, rate)
	if err != nil {
		return decimal.Zero, err
	}
	disc := decimal.NewFromInt(1).Div(decimal.NewFromInt(1).Add(rate).Pow(decimal.NewFromInt(int64(until))))
	return decimal.NewFromFloat(0.1).Mul(aSurv).Mul(disc), nil
}
