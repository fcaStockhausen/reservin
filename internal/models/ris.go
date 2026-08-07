package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// RISPerson is a persona (afiliado o beneficiario) decoded from a Registro 3
// of the Circular 1194 RIS file. It mirrors the fixed-width layout of the
// regulatory filing.
type RISPerson struct {
	NumeroInternoSVS       string          // 2-7 (6)
	NumeroOrden            int             // 8-9 (2)
	Genero                 string          // 10 (1): M/F
	TipoBeneficiario       string          // 11-12 (2): 99/10/11/20/21/30/35/41/42/50/51/52/77
	SituacionInvalidez     string          // 13 (1): N (No) / T (Total) / P (Parcial)
	FechaNacimiento        time.Time       // 14-21 (8): YYYYMMDD
	FechaFallecimiento     *time.Time      // 22-29 (8)
	FechaInvalidez         *time.Time      // 30-37 (8)
	DerechoPension         string          // 38-39 (2): 99/10/20
	RequisitoPension       string          // 40 (1)
	RelacionHijoMadre      int             // 41-42 (2)
	FechaNacHijoMenor      *time.Time      // 43-50 (8)
	DerechoAcrecer         string          // 51 (1)
	PorcentajePension      decimal.Decimal // 52-56 (5) 9(03)V9(02)
	PensionPersona         decimal.Decimal // 57-61 (5) 9(03)V9(02)
	PctAnticipoRV          decimal.Decimal // 62-65 (4) 9(02)V9(02)
	PctPensionPostAnticipo decimal.Decimal // 66-69 (4)
	FechaAnticipoRV        *time.Time      // 70-77 (8)

	// Reservas tecnicas reportadas (Registro 3, campos 3.25-3.38).
	// Cada una es 9(05)V9(02) = 7 digitos con 2 decimales implicitos.
	RTBaseTotal         decimal.Decimal // 78-84  (3.25)
	RTBaseTablaVigTotal decimal.Decimal // 85-91  (3.26)
	RTFinanciera200485  decimal.Decimal // 92-98  (3.27)
	RTFinancieraStock85 decimal.Decimal // 99-105 (3.28)
	RTFinanciera200406  decimal.Decimal // 106-112 (3.29)
	RTFinanciera200906  decimal.Decimal // 113-119 (3.30)
	RTFinanciera2014    decimal.Decimal // 120-126 (3.31 a)
	RTFinanciera2020    decimal.Decimal // 127-133 (3.31 b)

	RTBaseRetenida         decimal.Decimal // 134-140 (3.32)
	RTBaseTablaVigRetenida decimal.Decimal // 141-147 (3.33)
	RTFin200485Retenida    decimal.Decimal // 148-154 (3.34)
	RTFinStock85Retenida   decimal.Decimal // 155-161 (3.35)
	RTFin200406Retenida    decimal.Decimal // 162-168 (3.36)
	RTFin200906Retenida    decimal.Decimal // 169-175 (3.37)
	RTFin2014Retenida      decimal.Decimal // 176-182 (3.38 a)
	RTFin2020Retenida      decimal.Decimal // 183-189 (3.38 b)

	// Beneficios estatales y bonos (3.39-3.41).
	MontoBeneficioEstatal1    decimal.Decimal // 190-197
	MontoBeneficioEstatal2    decimal.Decimal // 198-205
	MontoBeneficioEstatal3    decimal.Decimal // 206-213
	TipoPagoBeneficioEstatal1 string          // 214
	TipoPagoBeneficioEstatal2 string          // 215
	TipoPagoBeneficioEstatal3 string          // 216
	BonoPorHijo1              decimal.Decimal // 217-222
	BonoPorHijo2              decimal.Decimal // 223-228
	BonoPorHijo3              decimal.Decimal // 229-234
}

// RISPolicy is a póliza decoded from a Registro 2, together with the personas
// (Registro 3) that belong to it.
type RISPolicy struct {
	NumeroInternoSVS            string          // 2-7 (6)
	NumeroPersonas              int             // 8-9 (2)
	TipoPension                 string          // 10-11 (2)
	CompaniaObligada            string          // 12 (1)
	VigenciaPension             string          // 13 (1): 6/7/8/9
	CodigoAFP                   string          // 14-15 (2)
	TipoAfiliado                string          // 16 (1)
	CuentaIndividual            decimal.Decimal // 17-23 (7) 9(05)V9(02)
	IngresoBaseUF               decimal.Decimal // 24-28 (5) 9(03)V9(02)
	PorcentajeCubierto          decimal.Decimal // 29-31 (3)
	FechaVigenciaInicial        time.Time       // 32-39 (8)
	PrimaUnica                  decimal.Decimal // 40-46 (7) 9(05)V9(02)
	RentaMensual                decimal.Decimal // 47-51 (5) 9(03)V9(02)
	TipoRenta                   string          // 52-55 (4)
	ModalidadRenta              string          // 56-59 (4): 1000/2xxx/3xxx/4xxx
	TipoOperacionRV             string          // 60-61 (2)
	PeriodoAumento              int             // 62-64 (3)
	PorcentajeAumento           decimal.Decimal // 65-69 (5) 9(03)V(02)
	TasaCostoEmision            decimal.Decimal // 70-73 (4) 9(02)V9(02)
	TasaVenta                   decimal.Decimal // 74-77 (4)
	NumeroReaseguro             int             // 78 (1)
	PolizaConAnticipo           string          // 163 (1)
	FechaRecalculoActual        *time.Time      // 164-171 (8)
	FechaRecalculoAnterior      *time.Time      // 172-179 (8)
	RentaAnteriorRecalcActual   decimal.Decimal // 180-184 (5)
	RentaAnteriorRecalcAnterior decimal.Decimal // 185-189 (5)
	NumeroSVSRelacionado        string          // 190-195 (6)

	Personas []RISPerson
}

// HasReserve reports whether the person reports a non-zero RT-BASE-TOTAL.
func (p *RISPerson) HasReserve() bool {
	return p.RTBaseTotal.GreaterThan(decimal.Zero)
}

// ReserveBase returns the reported base technical reserve in UF.
// The raw field is 9(05)V9(02): decodeDecimal9 already placed the implied
// decimals, so this is the value as-is.
func (p *RISPerson) ReserveBase() decimal.Decimal {
	return p.RTBaseTotal
}

// ReserveTablaVigente returns the reported base reserve with current tables in UF.
func (p *RISPerson) ReserveTablaVigente() decimal.Decimal {
	return p.RTBaseTablaVigTotal
}

// ReserveForComparison returns the reported reserve that is the natural target
// for validating the engine's calculation, given the policy's contract date.
//
// Per Circular 2332 the RT-FINANCIERA-2020 field is only reported for stock
// pre-2012 policies re-valued onto TM-2020 tables. Post-2012 policies already
// carry CB-2014/CB-2020 as the bautizo table, so their RT-BASE-TABLA-VIGENTE
// (= reserve with current tables) is the equivalent comparison target.
func (p *RISPerson) ReserveForComparison(contractDate time.Time) decimal.Decimal {
	cutoff := time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)
	if contractDate.Before(cutoff) {
		return p.RTFinanciera2020
	}
	return p.RTBaseTablaVigTotal
}

// ReserveForComparisonRetenida is the retained (post-reinsurance) version of
// ReserveForComparison.
func (p *RISPerson) ReserveForComparisonRetenida(contractDate time.Time) decimal.Decimal {
	cutoff := time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)
	if contractDate.Before(cutoff) {
		return p.RTFin2020Retenida
	}
	return p.RTBaseTablaVigRetenida
}

// ReserveFinanciera2020 returns the reported financial reserve with the TM-2020
// tables in UF. It is the natural target for calculations that discount with
// the VTD curve (RT-FINANCIERA-2020-TOTAL, campo 3.31 b).
func (p *RISPerson) ReserveFinanciera2020() decimal.Decimal {
	return p.RTFinanciera2020
}

// ReserveFinanciera2009 returns the reported financial reserve with the
// RV-2009/BMI-2006 tables in UF (RT-FINANCIERA-2009-2006-TOTAL, campo 3.30).
func (p *RISPerson) ReserveFinanciera2009() decimal.Decimal {
	return p.RTFinanciera200906
}

// ReserveFinanciera1985 returns the reported financial reserve with the
// RV-85/B-85 tables in UF (RT-FINANCIERA-2004-85-TOTAL, campo 3.27).
func (p *RISPerson) ReserveFinanciera1985() decimal.Decimal {
	return p.RTFinanciera200485
}

// ReserveBaseRetenida returns the reported base technical reserve after
// reinsurance cessions in UF (RT-BASE-RETENIDA, campo 3.32). This is the
// company's net exposure; the difference vs ReserveBase() is the ceded share.
func (p *RISPerson) ReserveBaseRetenida() decimal.Decimal {
	return p.RTBaseRetenida
}

// CededShare returns the fraction of the base reserve ceded to reinsurers
// (0..1), or 0 when there is no base reserve.
func (p *RISPerson) CededShare() decimal.Decimal {
	if p.RTBaseTotal.IsZero() {
		return decimal.Zero
	}
	return decimal.NewFromInt(1).Sub(p.RTBaseRetenida.Div(p.RTBaseTotal))
}

// ReserveFinanciera2020Retenida returns the reported financial reserve with the
// TM-2020 tables after reinsurance cessions in UF (RT-FINANCIERA-2020-RETENIDA).
func (p *RISPerson) ReserveFinanciera2020Retenida() decimal.Decimal {
	return p.RTFin2020Retenida
}
