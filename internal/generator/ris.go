// Package generator produces RIS (Registro de Informacion de Seguros) files
// in the fixed-width text format defined by the CMF Anexo Tecnico Circular 1194.
//
// Format rules (Anexo Tecnico seccion II.2):
//   - Fixed-width records, no delimiters
//   - Numeric fields: right-justified, zero-padded on the left, format 9(n)V9(m)
//     means n integer digits + m decimal digits (implied decimal, no point)
//   - Alphanumeric fields (X(n)): left-justified, space-padded on the right
//   - Dates: AAAAMMDD
//   - Amounts in UF with 2 decimals
//   - No tildes, no Ñ (use #), no special chars like º ª
//   - Each record ends with newline
package generator

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/models"
)

type RISRecord struct {
	ReportingPeriod time.Time
	RUTCompania     string
	VerRUTCompania  string
	NomCompania     string
	Policies        []RISPolicyRecord
}

type RISPolicyRecord struct {
	NumeroInterno        string
	NumeroPersonas       int
	RUTAfiliado          string
	VerRUTAfiliado       string
	TipoPension          string
	CompaniaObliga       string
	VigenciaPension      string
	CodigoAFP            string
	TipoAfiliado         string
	CuentaIndividual     decimal.Decimal
	IngresoBaseUF        decimal.Decimal
	PorcentajeCubierto   int
	FechaVigenciaInicial time.Time
	PrimaUnicaUF         decimal.Decimal
	RentaMensualUF       decimal.Decimal
	TipoRenta            string
	ModalidadRenta       string
	TipoOperacionRV      string
	PeriodoAumento       int
	PorcentajeAumento    decimal.Decimal
	TasaCtoEmision       decimal.Decimal
	TasaVenta            decimal.Decimal
	NumeroReaseguro      int
	Members              []RISMemberRecord
}

type RISMemberRecord struct {
	NumeroOrden             int
	RUT                     string
	VerRUT                  string
	PrimerApellido          string
	SegundoApellido         string
	Nombres                 string
	Genero                  string
	TipoBeneficiario        string
	SituacionInvalidez      string
	FechaNacimiento         time.Time
	FechaFallecimiento      *time.Time
	FechaInvalidez          *time.Time
	DerechoPension          string
	RequisitoPension        string
	RelacionHijoMadre       int
	FechaNacHijoMenor       *time.Time
	DerechoAcrecer          string
	PorcentajePension       decimal.Decimal
	PensionPersonaUF        decimal.Decimal
	RTBaseTotal             decimal.Decimal
	RTBaseTablaVigenteTotal decimal.Decimal
}

// Generate writes a complete RIS file to the given writer in fixed-width format.
func Generate(w io.Writer, rec *RISRecord) error {
	var b strings.Builder

	totalRegistros := 2 // reg1 + reg4
	for _, p := range rec.Policies {
		totalRegistros += 1 + len(p.Members) // reg2 + reg3s
	}

	b.WriteString(formatRegistro1(rec))

	var totalRTBase, totalRTBaseVigente decimal.Decimal
	for _, p := range rec.Policies {
		b.WriteString(formatRegistro2(&p))
		for _, m := range p.Members {
			b.WriteString(formatRegistro3(&m))
			totalRTBase = totalRTBase.Add(m.RTBaseTotal)
			totalRTBaseVigente = totalRTBaseVigente.Add(m.RTBaseTablaVigenteTotal)
		}
	}

	b.WriteString(formatRegistro4(len(rec.Policies), totalRegistros, totalRTBase, totalRTBaseVigente))

	_, err := io.WriteString(w, b.String())
	return err
}

// === Fixed-width field formatters ===

// num formats a decimal as 9(intDigits)V9(decDigits): zero-padded integer
// representation with implied decimal point.
func num(val decimal.Decimal, intDigits, decDigits int) string {
	multiplier := decimal.New(1, int32(decDigits))
	scaled := val.Mul(multiplier).Round(0)
	if scaled.IsNegative() {
		scaled = scaled.Neg()
	}
	totalDigits := intDigits + decDigits
	s := scaled.String()
	if len(s) < totalDigits {
		s = strings.Repeat("0", totalDigits-len(s)) + s
	}
	if len(s) > totalDigits {
		s = s[len(s)-totalDigits:]
	}
	return s
}

func numInt(val, width int) string {
	s := fmt.Sprintf("%d", val)
	if len(s) < width {
		s = strings.Repeat("0", width-len(s)) + s
	}
	if len(s) > width {
		s = s[len(s)-width:]
	}
	return s
}

func alpha(val string, width int) string {
	clean := sanitize(val)
	if len(clean) >= width {
		return clean[:width]
	}
	return clean + strings.Repeat(" ", width-len(clean))
}

func dateFmt(t time.Time) string {
	if t.IsZero() {
		return "00000000"
	}
	return t.Format("20060102")
}

func sanitize(s string) string {
	s = strings.ToUpper(s)
	repl := strings.NewReplacer(
		"Á", "A", "É", "E", "Í", "I", "Ó", "O", "Ú", "U",
		"Ñ", "#", "º", "", "ª", "",
	)
	return repl.Replace(s)
}

// === Registro 1: Header ===
// Layout: tipo(1) + fecha(8) + rut(9) + ver(1) + nom(60) + filler(235) = 314
func formatRegistro1(rec *RISRecord) string {
	return "1" +
		dateFmt(rec.ReportingPeriod) +
		numInt(parseRUTBody(rec.RUTCompania), 9) +
		alpha(dflt(rec.VerRUTCompania, "0"), 1) +
		alpha(rec.NomCompania, 60) +
		strings.Repeat(" ", 235) + "\n"
}

// === Registro 2: Poliza ===
func formatRegistro2(p *RISPolicyRecord) string {
	var b strings.Builder
	b.WriteString("2")                                        // TIPO-REGISTRO
	b.WriteString(alpha(p.NumeroInterno, 10))                 // 2.2 NUMERO-INTERNO
	b.WriteString(numInt(p.NumeroPersonas, 2))                // 2.3 NUMERO-PERSONAS
	b.WriteString(numInt(parseRUTBody(p.RUTAfiliado), 9))     // 2.4 RUT-AFILIADO
	b.WriteString(alpha(dflt(p.VerRUTAfiliado, "0"), 1))      // 2.5 VER-RUT
	b.WriteString(numInt(parseIntSafe(p.TipoPension), 2))     // 2.6 TIPO-PENSION
	b.WriteString(alpha(dflt(p.CompaniaObliga, "N"), 1))      // 2.7 COMPAÑIA-OBLIGADA
	b.WriteString(numInt(parseIntSafe(p.VigenciaPension), 1)) // 2.8 VIGENCIA
	b.WriteString(numInt(parseIntSafe(p.CodigoAFP), 2))       // 2.9 CODIGO-AFP
	b.WriteString(alpha(dflt(p.TipoAfiliado, "R"), 1))        // 2.10 TIPO-AFILIADO
	b.WriteString(num(p.CuentaIndividual, 5, 2))              // 2.11 CUENTA-INDIVIDUAL
	b.WriteString(num(p.IngresoBaseUF, 3, 2))                 // 2.12 INGRESO-BASE
	b.WriteString(numInt(p.PorcentajeCubierto, 3))            // 2.13 PORCENTAJE-CUBIERTO
	b.WriteString(dateFmt(p.FechaVigenciaInicial))            // 2.14 FECHA-VIGENCIA
	b.WriteString(num(p.PrimaUnicaUF, 5, 2))                  // 2.15 PRIMA-UNICA
	b.WriteString(num(p.RentaMensualUF, 3, 2))                // 2.16 RENTA-MENSUAL
	b.WriteString(numInt(parseIntSafe(p.TipoRenta), 4))       // 2.17 TIPO-RENTA
	b.WriteString(numInt(parseIntSafe(p.ModalidadRenta), 4))  // 2.18 MODALIDAD-RENTA
	b.WriteString(alpha(dflt(p.TipoOperacionRV, "  "), 2))    // 2.19 TIPO-OPERACION
	b.WriteString(numInt(p.PeriodoAumento, 3))                // 2.20 PERIODO-AUMENTO
	b.WriteString(num(p.PorcentajeAumento, 3, 2))             // 2.21 PORCENTAJE-AUMENTO
	b.WriteString(num(p.TasaCtoEmision, 2, 2))                // 2.22 TASA-CTO-EMISION
	b.WriteString(num(p.TasaVenta, 2, 2))                     // 2.23 TASA-VENTA
	b.WriteString(numInt(p.NumeroReaseguro, 1))               // 2.24 NUMERO-REASEGURO

	// 3 reaseguro blocks (all zeros if no reaseguro)
	for i := 0; i < 3; i++ {
		b.WriteString("00")                    // COMPAÑIA-REASEGURO 9(02)
		b.WriteString(" ")                     // OPERACION-REASEGURO X(01)
		b.WriteString(" ")                     // MODO-REASEGURO X(01)
		b.WriteString(num(decimal.Zero, 3, 2)) // PORCENTAJE-RETENIDO
		b.WriteString("00000000")              // FECHA-INICIO
		b.WriteString("99991231")              // FECHA-TERMINO
		b.WriteString(num(decimal.Zero, 2, 2)) // TASA-CTO-EQUIV-RET
		b.WriteString("00000000")              // FECHA-SUSCRIPCION
		b.WriteString("00000000")              // FECHA-VIGENCIA-REASEGURO
	}

	b.WriteString("0")                     // 2.34 POLIZA-CON-ANTICIPO
	b.WriteString("00000000")              // 2.35 FECHA-RECALCULO-ACTUAL
	b.WriteString("00000000")              // 2.36 FECHA-RECALCULO-ANTERIOR
	b.WriteString(num(decimal.Zero, 3, 2)) // 2.37 RENTA-RECALCULO-ACTUAL
	b.WriteString(num(decimal.Zero, 3, 2)) // 2.38 RENTA-RECALCULO-ANTERIOR
	b.WriteString(strings.Repeat(" ", 60)) // FILLER
	b.WriteString("\n")
	return b.String()
}

// === Registro 3: Afiliado/Beneficiario ===
func formatRegistro3(m *RISMemberRecord) string {
	var b strings.Builder
	b.WriteString("3")                                         // TIPO-REGISTRO
	b.WriteString(alpha("", 10))                               // 3.2 NUMERO-INTERNO
	b.WriteString(numInt(m.NumeroOrden, 2))                    // 3.3 NUMERO-DE-ORDEN
	b.WriteString(numInt(parseRUTBody(m.RUT), 9))              // 3.4 RUT
	b.WriteString(alpha(dflt(m.VerRUT, "0"), 1))               // 3.5 VER-RUT
	b.WriteString(alpha(m.PrimerApellido, 25))                 // 3.6 PRIMER-APELLIDO
	b.WriteString(alpha(m.SegundoApellido, 25))                // 3.7 SEGUNDO-APELLIDO
	b.WriteString(alpha(m.Nombres, 30))                        // 3.8 NOMBRES
	b.WriteString(alpha(m.Genero, 1))                          // 3.9 GENERO
	b.WriteString(numInt(parseIntSafe(m.TipoBeneficiario), 2)) // 3.10 TIPO-BENEFICIARIO
	b.WriteString(alpha(dflt(m.SituacionInvalidez, "N"), 1))   // 3.11 SITUACION-INVALIDEZ
	b.WriteString(dateFmt(m.FechaNacimiento))                  // 3.12 FECHA-NACIMIENTO

	if m.FechaFallecimiento != nil {
		b.WriteString(dateFmt(*m.FechaFallecimiento)) // 3.13
	} else {
		b.WriteString("00000000")
	}
	if m.FechaInvalidez != nil {
		b.WriteString(dateFmt(*m.FechaInvalidez)) // 3.14
	} else {
		b.WriteString("00000000")
	}

	b.WriteString(numInt(parseIntSafe(m.DerechoPension), 2))   // 3.15 DERECHO-PENSION
	b.WriteString(numInt(parseIntSafe(m.RequisitoPension), 1)) // 3.16 REQUISITO-PENSION
	b.WriteString(numInt(m.RelacionHijoMadre, 2))              // 3.17 RELACION-HIJO-MADRE

	if m.FechaNacHijoMenor != nil { // 3.18 FECHA-NAC-HIJO-MENOR
		b.WriteString(dateFmt(*m.FechaNacHijoMenor))
	} else {
		b.WriteString("00000000")
	}

	b.WriteString(alpha(dflt(m.DerechoAcrecer, "N"), 1)) // 3.19 DERECHO-ACRECER
	b.WriteString(num(m.PorcentajePension, 3, 2))        // 3.20 PORCENTAJE-PENSION
	b.WriteString(num(m.PensionPersonaUF, 3, 2))         // 3.21 PENSION-PERSONA
	b.WriteString(num(decimal.Zero, 2, 2))               // 3.22 PORCENTAJE-ANTICIPO
	b.WriteString(num(decimal.Zero, 2, 2))               // 3.23 PORCENTAJE-PENSION-POST-ANTICIPO
	b.WriteString("00000000")                            // 3.24 FECHA-ANTICIPO
	b.WriteString(num(m.RTBaseTotal, 5, 2))              // 3.25 RT-BASE-TOTAL
	b.WriteString(num(m.RTBaseTablaVigenteTotal, 5, 2))  // 3.26 RT-BASE-TABLA-VIGENTE

	// Campos 3.27-3.38: 12 RT financieras (total + retenida), all 9(05)V9(02)
	for i := 0; i < 12; i++ {
		b.WriteString(num(decimal.Zero, 5, 2))
	}

	// Campos 3.39-3.41: beneficios estatales + bono por hijo
	for i := 0; i < 3; i++ {
		b.WriteString(num(decimal.Zero, 2, 6)) // MONTO-PAGO-BENEFICIO-ESTATAL
	}
	for i := 0; i < 3; i++ {
		b.WriteString("0") // TIPO-PAGO-BENEFICIO-ESTATAL
	}
	for i := 0; i < 3; i++ {
		b.WriteString(num(decimal.Zero, 2, 4)) // BONO-POR-HIJO
	}

	b.WriteString("\n")
	return b.String()
}

// === Registro 4: Totales ===
func formatRegistro4(numPolizas, numRegistros int, totalRTBase, totalRTBaseVigente decimal.Decimal) string {
	var b strings.Builder
	b.WriteString("4")                            // TIPO-REGISTRO
	b.WriteString(numInt(numPolizas, 6))          // 4.2 NUMERO-POLIZAS
	b.WriteString(numInt(numRegistros, 6))        // 4.3 NUMERO-REGISTROS
	b.WriteString(num(totalRTBase, 15, 2))        // 4.4 TOTAL-RT-BASE-TOTAL
	b.WriteString(num(totalRTBaseVigente, 15, 2)) // 4.5 TOTAL-RT-BASE-TABLA-VIGENTE

	// Campos 4.6-4.17: 11 RT financieras/retenidas, all 9(15)V9(02)
	for i := 0; i < 11; i++ {
		b.WriteString(num(decimal.Zero, 15, 2))
	}

	b.WriteString(strings.Repeat(" ", 63)) // FILLER
	b.WriteString("\n")
	return b.String()
}

// === Conversion from models ===

func FromSimulation(
	policy *models.Policy,
	members []models.Beneficiario,
	rentaMensualUF decimal.Decimal,
	reserves map[string]decimal.Decimal,
) *RISPolicyRecord {
	rec := &RISPolicyRecord{
		NumeroInterno:        policy.NumeroPoliza,
		NumeroPersonas:       len(members),
		TipoPension:          dflt(policy.TipoPension, models.TipoPensionRVVejezJubilacion),
		CompaniaObliga:       "N",
		VigenciaPension:      dflt(policy.VigenciaPension, models.VigenciaEnPago),
		CodigoAFP:            "00",
		TipoAfiliado:         "R",
		FechaVigenciaInicial: policy.FechaInicio,
		PrimaUnicaUF:         policy.CapitalAsegurado,
		RentaMensualUF:       rentaMensualUF.Div(decimal.NewFromInt(12)),
		TipoRenta:            policy.TipoRenta,
		ModalidadRenta:       dflt(policy.ModalidadRenta, "1000"),
		TipoOperacionRV:      "SM",
		PeriodoAumento:       policy.PeriodoAumento,
		PorcentajeAumento:    policy.PorcentajeAumento,
		TasaCtoEmision:       policy.TasaTC,
		TasaVenta:            policy.TasaTM,
	}

	for i := range members {
		m := &members[i]
		rtBase := reserves[string(m.Rol)]

		member := RISMemberRecord{
			NumeroOrden:        i + 1,
			Genero:             m.Sexo,
			TipoBeneficiario:   dflt(m.TipoBeneficiarioC1194, "99"),
			SituacionInvalidez: dflt(m.SituacionInvalidez, "N"),
			DerechoPension:     dflt(m.DerechoPension, "99"),
			RequisitoPension:   dflt(m.RequisitoPension, "1"),
			DerechoAcrecer:     dflt(m.DerechoAcrecer, "N"),
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
		}

		rec.Members = append(rec.Members, member)
	}

	return rec
}

func FileName(period time.Time) string {
	return fmt.Sprintf("R%s.txt", period.Format("200601"))
}

// === helpers ===

func dflt(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

func parseIntSafe(s string) int {
	if s == "" {
		return 0
	}
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func parseRUTBody(rut string) int {
	var n int
	fmt.Sscanf(rut, "%d", &n)
	return n
}
