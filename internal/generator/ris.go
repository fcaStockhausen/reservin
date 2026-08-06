// Package generator produces RIS (Registro de Informacion de Seguros) files
// in the fixed-width text format defined by the CMF Anexo Tecnico Circular 1194.
//
// The RIS file contains 4 record types:
//
//	Registro 1: Header del archivo (1 por archivo)
//	Registro 2: Poliza / siniestro (1 por causante)
//	Registro 3: Asegurados y beneficiarios (N por cada registro 2)
//	Registro 4: Totales del archivo (1 por archivo)
//
// File naming: Raaaamm.txt where aaaa=year, mm=month of reporting period.
package generator

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/models"
)

// RISRecord holds the data needed to generate a complete RIS file.
type RISRecord struct {
	ReportingPeriod time.Time // last day of the reporting trimester
	RUTCompania     string    // RUT of the insurance company

	Policies []RISPolicyRecord
}

// RISPolicyRecord holds data for one policy and its family group.
type RISPolicyRecord struct {
	// Registro 2 fields
	NumeroInterno   string // unique policy ID within the file
	RUTAfiliado     string
	VerRUTAfiliado  string
	TipoPension     string // 01-15 (C1194 campo 2.6)
	CompaniaObliga  string // O / N (campo 2.7)
	VigenciaPension string // 6/7/8/9 (campo 2.8)
	CodigoAFP       string // AFP code
	TipoAfiliado    string // D / I / R (campo 2.10)

	FechaVigenciaInicial time.Time // campo 2.14
	PrimaUnicaUF         decimal.Decimal
	RentaMensualUF       decimal.Decimal // campo 2.16
	TipoRenta            string          // 1000/2xxx/3000 (campo 2.17)
	ModalidadRenta       string          // 1000/2xxx/3xxx/4xxx (campo 2.18)
	TipoOperacionRV      string          // SM / CM (campo 2.19)
	PeriodoAumento       int             // campo 2.20
	PorcentajeAumento    decimal.Decimal // campo 2.21
	TasaCtoEmision       decimal.Decimal // campo 2.22 (TCj)
	TasaVenta            decimal.Decimal // campo 2.23 (TVj)
	NumeroReaseguro      int             // campo 2.24

	// Members for Registro 3
	Members []RISMemberRecord
}

// RISMemberRecord holds data for one person in Registro 3.
type RISMemberRecord struct {
	NumeroOrden        int // position within the policy
	RUT                string
	VerRUT             string
	PrimerApellido     string
	SegundoApellido    string
	Nombres            string
	Genero             string // M / F (campo 3.9)
	TipoBeneficiario   string // 99/10/11/20/21/30/35/41/42/50/51/52/77 (campo 3.10)
	SituacionInvalidez string // N / T / P (campo 3.11)
	FechaNacimiento    time.Time
	FechaFallecimiento *time.Time
	FechaInvalidez     *time.Time
	DerechoPension     string          // 99 / 10 / 20 (campo 3.15)
	RequisitoPension   string          // 1-9 (campo 3.16)
	DerechoAcrecer     string          // S / N (campo 3.19)
	PorcentajePension  decimal.Decimal // campo 3.20
	PensionPersonaUF   decimal.Decimal // campo 3.21

	// Reserves (campos 3.25-3.26)
	RTBaseTotal             decimal.Decimal
	RTBaseTablaVigenteTotal decimal.Decimal
}

// Generate writes a complete RIS file to the given writer.
func Generate(w io.Writer, rec *RISRecord) error {
	var b strings.Builder

	// Registro 1: Header
	b.WriteString(formatRegistro1(rec))

	// Registro 2 + Registro 3 (per policy)
	totalPersonas := 0
	for _, p := range rec.Policies {
		totalPersonas += len(p.Members)
		b.WriteString(formatRegistro2(&p, rec))
		for _, m := range p.Members {
			b.WriteString(formatRegistro3(&m, &p, rec))
		}
	}

	// Registro 4: Totales
	b.WriteString(formatRegistro4(rec, totalPersonas))

	_, err := io.WriteString(w, b.String())
	return err
}

// formatRegistro1 produces the header record.
// Format: TIPO|PERIODO|RUT_COMPAÑIA
func formatRegistro1(rec *RISRecord) string {
	return fmt.Sprintf("1|%s|%s\n",
		rec.ReportingPeriod.Format("20060102"),
		padRUT(rec.RUTCompania),
	)
}

// formatRegistro2 produces a policy record.
func formatRegistro2(p *RISPolicyRecord, rec *RISRecord) string {
	numPersonas := len(p.Members)

	return fmt.Sprintf("2|%s|%d|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%d|%s|%d|%s|%s|%s|%d\n",
		p.NumeroInterno,                 // 2.2 NUMERO-INTERNO
		numPersonas,                     // 2.3 NUMERO-PERSONAS
		p.RUTAfiliado,                   // 2.4 RUT-AFILIADO
		p.VerRUTAfiliado,                // 2.5 VER-RUT-AFILIADO
		p.TipoPension,                   // 2.6 TIPO-PENSION
		p.CompaniaObliga,                // 2.7 COMPAÑIA-OBLIGADA
		p.VigenciaPension,               // 2.8 VIGENCIA-PENSION
		defaultStr(p.CodigoAFP, "00"),   // 2.9 CODIGO-AFP
		defaultStr(p.TipoAfiliado, "R"), // 2.10 TIPO-AFILIADO
		zeroIfEmpty(""),                 // 2.11 CUENTA-INDIVIDUAL (RV only)
		zeroIfEmpty(""),                 // 2.12 INGRESO-BASE-EN-UF
		zeroIfEmpty(""),                 // 2.13 PORCENTAJE-CUBIERTO
		p.FechaVigenciaInicial.Format("20060102"), // 2.14
		p.PrimaUnicaUF.StringFixed(2),             // 2.15
		p.RentaMensualUF.StringFixed(2),           // 2.16
		numPersonas,                               // 2.3 (repeated for RV section)
		p.TipoRenta,                               // 2.17 TIPO-RENTA
		p.PeriodoAumento,                          // 2.20 PERIODO-AUMENTO
		p.PorcentajeAumento.StringFixed(2),        // 2.21
		p.TasaCtoEmision.StringFixed(2),           // 2.22
		p.TasaVenta.StringFixed(2),                // 2.23
		p.NumeroReaseguro,                         // 2.24
	)
}

// formatRegistro3 produces a person/beneficiary record.
func formatRegistro3(m *RISMemberRecord, p *RISPolicyRecord, rec *RISRecord) string {
	fechaNac := m.FechaNacimiento.Format("20060102")
	fechaFall := "0"
	if m.FechaFallecimiento != nil {
		fechaFall = m.FechaFallecimiento.Format("20060102")
	}
	fechaInv := "0"
	if m.FechaInvalidez != nil {
		fechaInv = m.FechaInvalidez.Format("20060102")
	}

	return fmt.Sprintf("3|%d|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\n",
		m.NumeroOrden,                            // 3.3 NUMERO-DE-ORDEN
		m.RUT,                                    // 3.4 RUT
		m.VerRUT,                                 // 3.5 VER-RUT
		padRight(m.PrimerApellido, 30),           // 3.6
		padRight(m.SegundoApellido, 30),          // 3.7
		padRight(m.Nombres, 30),                  // 3.8
		m.Genero,                                 // 3.9 GENERO
		m.TipoBeneficiario,                       // 3.10 TIPO-BENEFICIARIO
		m.SituacionInvalidez,                     // 3.11 SITUACION-INVALIDEZ
		fechaNac,                                 // 3.12 FECHA-NACIMIENTO
		fechaFall,                                // 3.13 FECHA-FALLECIMIENTO
		fechaInv,                                 // 3.14 FECHA-INVALIDEZ
		m.DerechoPension,                         // 3.15 DERECHO-PENSION
		m.RequisitoPension,                       // 3.16 REQUISITO-PENSION
		"0",                                      // 3.17 RELACION-HIJO-MADRE
		"0",                                      // 3.18 FECHA-NAC-HIJO-MENOR
		m.DerechoAcrecer,                         // 3.19 DERECHO-ACRECER
		m.PorcentajePension.StringFixed(2),       // 3.20
		m.PensionPersonaUF.StringFixed(2),        // 3.21
		m.RTBaseTotal.StringFixed(2),             // 3.25 RT-BASE-TOTAL
		m.RTBaseTablaVigenteTotal.StringFixed(2), // 3.26
	)
}

// formatRegistro4 produces the totals record.
func formatRegistro4(rec *RISRecord, totalPersonas int) string {
	totalPolizas := len(rec.Policies)
	return fmt.Sprintf("4|%d|%d\n", totalPolizas, totalPersonas)
}

// FromSimulation converts a policy + beneficiario slice into RIS records.
func FromSimulation(
	policy *models.Policy,
	members []models.Beneficiario,
	rentaMensualUF decimal.Decimal,
	reserves map[string]decimal.Decimal,
) *RISPolicyRecord {
	rec := &RISPolicyRecord{
		NumeroInterno:        policy.NumeroPoliza,
		TipoPension:          defaultStr(policy.TipoPension, models.TipoPensionRVVejezJubilacion),
		CompaniaObliga:       "N",
		VigenciaPension:      defaultStr(policy.VigenciaPension, models.VigenciaEnPago),
		TipoAfiliado:         "R",
		FechaVigenciaInicial: policy.FechaInicio,
		PrimaUnicaUF:         policy.CapitalAsegurado,
		RentaMensualUF:       rentaMensualUF.Div(decimal.NewFromInt(12)),
		TipoRenta:            defaultStr(policy.TipoRenta, "1000"),
		ModalidadRenta:       defaultStr(policy.ModalidadRenta, "1000"),
		TipoOperacionRV:      "SM",
		PeriodoAumento:       policy.PeriodoAumento,
		PorcentajeAumento:    policy.PorcentajeAumento,
		TasaCtoEmision:       policy.TasaTC,
		TasaVenta:            policy.TasaTM,
	}

	for i := range members {
		m := &members[i]
		rtBase := reserves[string(m.Rol)]
		if rtBase.IsZero() {
			rtBase = reserves[string(m.TipoBeneficiarioC1194)]
		}

		member := RISMemberRecord{
			NumeroOrden:        i + 1,
			Genero:             m.Sexo,
			TipoBeneficiario:   defaultStr(m.TipoBeneficiarioC1194, "99"),
			SituacionInvalidez: defaultStr(m.SituacionInvalidez, "N"),
			DerechoPension:     defaultStr(m.DerechoPension, "99"),
			RequisitoPension:   defaultStr(m.RequisitoPension, "1"),
			DerechoAcrecer:     defaultStr(m.DerechoAcrecer, "N"),
			PorcentajePension:  m.PorcentajeRenta.Mul(decimal.NewFromInt(100)),
			RTBaseTotal:        rtBase,
		}

		if m.FechaNacimiento != nil {
			member.FechaNacimiento = *m.FechaNacimiento
		} else {
			member.FechaNacimiento = policy.FechaInicio.AddDate(-m.EdadContratacion, 0, 0)
		}

		if m.Estado == "FALLECIDO" {
			member.FechaFallecimiento = &policy.FechaInicio
			member.DerechoPension = models.DerechoPensionNo
		}

		rec.Members = append(rec.Members, member)
	}

	return rec
}

// Helper functions

func padRUT(rut string) string {
	return padRight(rut, 12)
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func defaultStr(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func zeroIfEmpty(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// FileName generates the RIS file name: Raaaamm.txt
func FileName(period time.Time) string {
	return fmt.Sprintf("R%s.txt", period.Format("200601"))
}
