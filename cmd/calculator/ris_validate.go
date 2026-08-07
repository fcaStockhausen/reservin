package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/calculator"
	"reservas/internal/database"
	"reservas/internal/loader"
	"reservas/internal/models"
)

// diffRec records a per-policy comparison between reported and computed reserve.
type diffRec struct {
	svs         string
	reportado   float64
	calculado   float64
	diffPct     float64
	estrato     string
	renta       float64
	sexo        string
	edad        int
	modalidad   string
	tipoFamilia string // causante_solo / con_conyuge / con_hijos / mixto / sin_causante
	nPersonas   int
}

// risBuildResult holds the parsed group and context for a RIS policy.
type risBuildResult struct {
	pol          models.Policy
	grupo        *models.GrupoFamiliar
	contractDate time.Time
	causanteVivo bool
	rentaMensual decimal.Decimal
}

// buildGrupoFromRIS converts a RIS policy into the internal Policy + family
// group. When contemporanea is true the members are assigned the TM-2020 table
// in force today (the basis of RT-FINANCIERA-2020); otherwise the contract-date
// bautizo table is used (the basis of RT-BASE).
func buildGrupoFromRIS(ris models.RISPolicy, contemporanea bool) risBuildResult {
	// The reserve is valued with the contract date anchoring the table.
	contractDate := ris.FechaVigenciaInicial
	if contractDate.IsZero() {
		contractDate = time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	valuationDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)

	pol := models.Policy{
		NumeroPoliza:     ris.NumeroInternoSVS,
		TipoRenta:        string(models.PolicyTypeVitalicia),
		FechaInicio:      contractDate,
		CapitalAsegurado: ris.PrimaUnica,
		TasaTM:           ris.TasaCostoEmision.Div(decimal.NewFromInt(100)),
		TasaTC:           ris.TasaVenta.Div(decimal.NewFromInt(100)),
		Estado:           "ACTIVA",
		TipoPension:      ris.TipoPension,
		ModalidadRenta:   ris.ModalidadRenta,
	}

	grupo := &models.GrupoFamiliar{}
	causanteVivo := true

	for _, per := range ris.Personas {
		rol := risRolFor(per.TipoBeneficiario)
		b := models.Beneficiario{
			Rol:                   rol,
			Sexo:                  per.Genero,
			FechaNacimiento:       &per.FechaNacimiento,
			EdadContratacion:      ageAt(per.FechaNacimiento, contractDate),
			Estado:                "ACTIVO",
			TipoBeneficiarioC1194: per.TipoBeneficiario,
			DerechoPension:        per.DerechoPension,
			PorcentajeRenta:       per.PorcentajePension.Div(decimal.NewFromInt(100)),
		}
		switch per.SituacionInvalidez {
		case models.InvTotal, models.InvParcial:
			b.SituacionInvalidez = per.SituacionInvalidez
		default:
			b.SituacionInvalidez = models.InvNo
		}
		tipoTabla := ""
		if rol == models.RolCausante {
			tipoTabla = string(models.TableTypeVejez)
			if per.SituacionInvalidez != models.InvNo {
				tipoTabla = string(models.TableTypeInvalidez)
			}
		}
		if contemporanea {
			// TM-2020 in force at valuation: the basis of RT-FINANCIERA-2020.
			b.TablaAsignada = models.SelectContemporaneaTable(rol, tipoTabla, per.Genero, valuationDate)
		} else {
			b.TablaAsignada = models.SelectTableForBeneficiario(rol, per.Genero, per.TipoBeneficiario, contractDate, tipoTabla)
		}

		// Assign by role, not by position. The first causante becomes the
		// group's Causante; any further record is a beneficiary (covers the
		// full family: spouse, children, parents — regardless of DerechoPension).
		if rol == models.RolCausante && grupo.Causante == nil {
			if per.FechaFallecimiento != nil {
				causanteVivo = false
			}
			grupo.Causante = &b
		} else {
			grupo.Beneficiarios = append(grupo.Beneficiarios, &b)
		}
	}
	return risBuildResult{
		pol:          pol,
		grupo:        grupo,
		contractDate: contractDate,
		causanteVivo: causanteVivo,
		rentaMensual: ris.RentaMensual,
	}
}

// computeRISReserve computes the reserve for a RIS policy. Two cases:
//
//   - Causante alive: full VPPj projection via FlowProjector (causante + survivor
//     flows, PG, VTD) valued at the current physical age.
//   - Causante deceased: each survivor is valued as a standalone life annuity
//     via FlowProjector (same engine, no manual loop).
//
// Returns the computed reserve and a non-nil error explaining why the policy
// could not be valued (so the caller can surface the cause, not swallow it).
//
// Tasa de descuento por cohorte (NCG 318):
//   - pre-2012 y 2012-may2015: min(TM, TV) de emisión (TasaCostoEmision/TasaVenta)
//   - jun2015-nov2020: TCj (TIR con VTD del mes de emisión)
//   - post-dic2020: min(TVj, TCj)
//
// Para cohortes con VTD histórico disponible (post-sep2020), instalamos la
// curva VTD del mes de emisión como descuento. Para el resto, cae al flat rate
// min(TM, TV) del RIS (que solo es correcto para 2012-may2015).
func computeRISReserve(calc *calculator.ReserveCalculator, rb risBuildResult, p models.RISPolicy) (float64, error) {
	rentaMensual := rb.rentaMensual
	// Fallback chain for the monthly pension: RentaMensual (campo 2.20) →
	// PensionPersona of the first persona with derecho 99/20 (campo 3.21,
	// pensión por persona en UF) → 12 × CapitalAsegurado / 120. Older stock
	// (modalidad "0000", pre-2005) only reports PensionPersona.
	if rentaMensual.LessThanOrEqual(decimal.Zero) {
		for _, per := range p.Personas {
			if per.PensionPersona.GreaterThan(decimal.Zero) {
				rentaMensual = per.PensionPersona
				break
			}
		}
	}
	if rentaMensual.LessThanOrEqual(decimal.Zero) {
		rentaMensual = decimal.NewFromFloat(12).Mul(rb.pol.CapitalAsegurado).Div(decimal.NewFromFloat(120))
	}
	rentaAnual := rentaMensual.Mul(decimal.NewFromInt(12))
	valuationDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	currentYear := valuationDate.Year() - rb.contractDate.Year()
	if currentYear < 0 {
		currentYear = 0
	}

	// Instalar VTD del mes de emisión si la cohorte lo requiere y está
	// disponible en la DB. Para cohortes pre-dic2020 sin VTD histórico, cae al
	// flat rate (con la limitación documentada).
	installIssuanceVTD(calc, rb.contractDate)

	if rb.causanteVivo && rb.grupo.Causante != nil && rb.grupo.Causante.TablaAsignada != "" {
		result, err := calc.CalculateAt(rb.pol, rb.grupo, rentaAnual, currentYear)
		if err != nil {
			return 0, fmt.Errorf("causante vivo: %w", err)
		}
		return result.TotalReserve.InexactFloat64(), nil
	}

	// Causante deceased: value each survivor as a standalone life annuity via
	// the same optimized FlowProjector (no manual decimal loop).
	// The guaranteed period of the original policy does NOT extend to the
	// survivor as a life-contingent annuity: once the causante is dead the
	// survivor simply receives the pension for the rest of their life, so we
	// zero out the modalidad to disable the PG branch in Project.
	total := decimal.Zero
	polNoPG := rb.pol
	polNoPG.ModalidadRenta = "1000"
	for _, b := range rb.grupo.Beneficiarios {
		if b.TablaAsignada == "" || b.PorcentajeRenta.IsZero() {
			continue
		}
		benRenta := rentaAnual.Mul(b.PorcentajeRenta)
		soloGrupo := &models.GrupoFamiliar{Causante: b}
		result, err := calc.CalculateAt(polNoPG, soloGrupo, benRenta, currentYear)
		if err != nil {
			return 0, fmt.Errorf("sobreviviente %s: %w", b.Rol, err)
		}
		total = total.Add(result.TotalReserve)
	}
	return total.InexactFloat64(), nil
}

// installIssuanceVTD loads the VTD curve for the policy's issuance month when
// the cohort requires TCj (jun2015+) and the historical VTD is available in
// the DB. When the VTD is unavailable (pre-sep2020 issuances), the projector
// falls back to the flat min(TM, TV) rate set in the policy.
//
// The cutoff 2020-09-01 reflects the earliest VTD vector currently loaded.
func installIssuanceVTD(calc *calculator.ReserveCalculator, contractDate time.Time) {
	vtdCutoff := time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC)
	tcjCutoff := time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC)
	if contractDate.Before(tcjCutoff) || contractDate.Before(vtdCutoff) {
		// Pre-TCj regime or VTD unavailable: use flat rate from policy.
		// Reset any previously installed curve so the next policy starts clean.
		calc.ClearVTD()
		return
	}
	if !calc.LoadVTDForCached(contractDate.Year(), int(contractDate.Month())) {
		calc.ClearVTD()
	}
}

// risRolFor maps the C1194 TIPO-BENEFICIARIO code to our internal role.
func risRolFor(tipo string) models.BeneficiarioRol {
	switch tipo {
	case "99":
		return models.RolCausante
	case "10", "11":
		return models.RolConyuge
	case "20", "21":
		return models.RolMadrePadreNMat
	case "30", "35":
		return models.RolHijo
	case "41", "42":
		return models.RolPadres
	case "50", "51", "52":
		return models.RolConviviente
	default:
		return models.RolConyuge
	}
}

func ageAt(birth, at time.Time) int {
	if birth.IsZero() {
		return 65
	}
	age := at.Year() - birth.Year()
	if at.YearDay() < birth.YearDay() {
		age--
	}
	return age
}

// validateRIS runs the reserve calculator over a sample of the RIS file and
// compares computed reserves against the reported RT-FINANCIERA-2020 (total or
// retained, per the retenida flag). mejorar toggles mortality improvement.
func validateRIS(db *database.DB, path string, sample int, retenida bool, mejorar bool, debugSVS string) {
	mortRepo := database.NewMortalityRepository(db.DB)

	header, _ := loader.NewRIS1194Loader(path).LoadHeader()
	if header != nil {
		fmt.Printf("Período: %s | pólizas: %d | beneficiarios: %d\n",
			header.FechaHasta.Format("2006-01-02"), header.NumRegistros2, header.NumRegistros3)
	}

	if retenida {
		fmt.Printf("Validando muestra de %d pólizas contra RT-FINANCIERA-2020-RETENIDA (TM-2020, tasa emisión)...\n\n", sample)
	} else {
		fmt.Printf("Validando muestra de %d pólizas contra RT-FINANCIERA-2020 reportada (TM-2020, tasa emisión)...\n\n", sample)
	}
	ld := loader.NewRIS1194Loader(path)
	policies, errs := ld.Stream(sample)

	// La reserva técnica se descuenta a la tasa de la póliza (min TM, TV a la
	// fecha de emisión, NCG 318 N°2.3a), NO al VTD actual. El VTD solo aplica a
	// pólizas nuevas (post-2020). Por eso NO cargamos el VTD aquí.
	calc := calculator.NewReserveCalculator(mortRepo, database.NewVTDRepository(db.DB))
	calc.SetMejoramientoEnabled(mejorar)

	mejoraLabel := "ON"
	if !mejorar {
		mejoraLabel = "OFF"
	}
	fmt.Printf("Mejoramiento (Circular 2332): %s\n", mejoraLabel)

	var diffs []diffRec
	processed := 0
	skipped := 0
	skipCauses := map[string]int{} // cause -> count
	var sumReported, sumCalculated float64

	for {
		select {
		case p, ok := <-policies:
			if !ok {
				policies = nil
				break // breaks the select, not the for; falls to nil check
			}
			rb := buildGrupoFromRIS(p, true) // TM-2020 contemporánea: basis de RT-FINANCIERA-2020

			// Sum reported reserve across all persons in the policy (UF). The
			// comparison target depends on the contract date: pre-2012 reports
			// RT-FINANCIERA-2020 (revaluation onto TM-2020); post-2012 reports
			// RT-BASE-TABLA-VIGENTE (native table already TM/CB-2014/2020).
			reported := 0.0
			hasReported := false
			contractDate := rb.contractDate
			for _, per := range p.Personas {
				var r float64
				if retenida {
					r = per.ReserveForComparisonRetenida(contractDate).InexactFloat64()
				} else {
					r = per.ReserveForComparison(contractDate).InexactFloat64()
				}
				if r > 0 {
					reported += r
					hasReported = true
				}
			}
			if !hasReported {
				skipped++
				skipCauses["sin_reserve_reportada"]++
				continue
			}

			calculated, err := computeRISReserve(calc, rb, p)
			if err != nil {
				skipped++
				cause := categorizeErr(err)
				skipCauses[cause]++
				if debugSVS != "" && p.NumeroInternoSVS == debugSVS {
					fmt.Printf("\n=== DEBUG SVS=%s ===\n", p.NumeroInternoSVS)
					fmt.Printf("ERROR: %v\n", err)
				}
				continue
			}

			if debugSVS != "" && p.NumeroInternoSVS == debugSVS {
				printPolicyDebug(p, rb, calculated, retenida, contractDate)
			}

			diffPct := 0.0
			if reported > 0 {
				diffPct = (calculated - reported) / reported * 100
			}
			sumReported += reported
			sumCalculated += calculated

			sexo, edad, modalidad, tipoFamilia, nPersonas := policyProfile(p, rb)
			diffs = append(diffs, diffRec{
				svs:         p.NumeroInternoSVS,
				reportado:   reported,
				calculado:   calculated,
				diffPct:     diffPct,
				estrato:     estratoOf(rb.contractDate),
				renta:       rb.rentaMensual.Mul(decimal.NewFromInt(12)).InexactFloat64(),
				sexo:        sexo,
				edad:        edad,
				modalidad:   modalidad,
				tipoFamilia: tipoFamilia,
				nPersonas:   nPersonas,
			})
			processed++
			if processed%500 == 0 {
				fmt.Printf("  ...%d pólizas procesadas\n", processed)
			}
			if processed >= sample {
				policies = nil
			}

		case err, ok := <-errs:
			if ok && err != nil {
				log.Printf("stream error: %v", err)
			}
			errs = nil
		}
		if policies == nil && errs == nil {
			break
		}
	}

	printRISReport(diffs, sumReported, sumCalculated, processed, skipped, skipCauses)
}

// categorizeErr reduces an error to a coarse cause bucket for skip accounting.
func categorizeErr(err error) string {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "not loaded") || strings.Contains(msg, "no records"):
		return "tabla_mortalidad_faltante"
	case strings.Contains(msg, "table") && strings.Contains(msg, "loading"):
		return "tabla_mortalidad_carga"
	case strings.Contains(msg, "improvement"):
		return "factor_mejoramiento"
	case strings.Contains(msg, "sobreviviente"):
		return "sobreviviente_error"
	default:
		return "otro"
	}
}

// policyProfile extracts the causante sex, current age, modalidad, and family
// composition for stratified reporting.
func policyProfile(p models.RISPolicy, rb risBuildResult) (sexo string, edad int, modalidad string, tipoFamilia string, nPersonas int) {
	modalidad = p.ModalidadRenta
	nPersonas = len(p.Personas)
	if rb.grupo != nil && rb.grupo.Causante != nil {
		sexo = rb.grupo.Causante.Sexo
		valuationDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		if rb.grupo.Causante.FechaNacimiento != nil {
			edad = ageAt(*rb.grupo.Causante.FechaNacimiento, valuationDate)
		}
	}
	// Classify family composition by role of the beneficiaries.
	hasConyuge, hasHijos, hasOtro := false, false, false
	for _, b := range rb.grupo.Beneficiarios {
		switch b.Rol {
		case models.RolConyuge, models.RolConviviente:
			hasConyuge = true
		case models.RolHijo:
			hasHijos = true
		default:
			hasOtro = true
		}
	}
	switch {
	case !rb.causanteVivo:
		tipoFamilia = "sin_causante_vivo"
	case len(rb.grupo.Beneficiarios) == 0:
		tipoFamilia = "causante_solo"
	case hasConyuge && !hasHijos && !hasOtro:
		tipoFamilia = "con_conyuge"
	case hasHijos && !hasConyuge && !hasOtro:
		tipoFamilia = "con_hijos"
	case hasConyuge && hasHijos:
		tipoFamilia = "conyuge_hijos"
	default:
		tipoFamilia = "mixto"
	}
	return
}

// estratoOf classifies a policy by its contract-date stratum.
func estratoOf(d time.Time) string {
	switch {
	case d.Before(time.Date(2005, 3, 9, 0, 0, 0, 0, time.UTC)):
		return "pre-2005 (RV-85)"
	case d.Before(time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)):
		return "2005-2011 (RV-2009)"
	default:
		return "post-2012 (TM-2020)"
	}
}

func printRISReport(diffs []diffRec, sumReported, sumCalculated float64, processed, skipped int, skipCauses map[string]int) {
	fmt.Printf("=== VALIDACIÓN RIS ===\n")
	fmt.Printf("Pólizas procesadas: %d | omitidas: %d\n", processed, skipped)
	fmt.Printf("Reserva reportada:  %12.2f UF\n", sumReported)
	fmt.Printf("Reserva calculada:  %12.2f UF\n", sumCalculated)
	if sumReported > 0 {
		fmt.Printf("Diferencia total:   %12.2f UF (%+.2f%%)\n", sumCalculated-sumReported, (sumCalculated-sumReported)/sumReported*100)
	}

	if len(skipCauses) > 0 {
		fmt.Printf("\nCausas de omisión:\n")
		keys := make([]string, 0, len(skipCauses))
		for k := range skipCauses {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Printf("  %-30s %d\n", k, skipCauses[k])
		}
	}

	if len(diffs) == 0 {
		fmt.Println("Sin pólizas comparables.")
		return
	}

	// Aggregate by estrato.
	type agg struct {
		sumR, sumC float64
		n          int
	}
	byEstrato := map[string]*agg{}
	bySexo := map[string]*agg{}
	byModal := map[string]*agg{}
	byEdad := map[string]*agg{}
	byFamilia := map[string]*agg{}
	var medAbs []float64
	var outliers []diffRec
	for _, d := range diffs {
		add := func(m map[string]*agg, key string, d diffRec) {
			a := m[key]
			if a == nil {
				a = &agg{}
				m[key] = a
			}
			a.sumR += d.reportado
			a.sumC += d.calculado
			a.n++
		}
		add(byEstrato, d.estrato, d)
		add(bySexo, d.sexo, d)
		add(byModal, d.modalidad, d)
		add(byEdad, edadBucket(d.edad), d)
		add(byFamilia, d.tipoFamilia, d)
		medAbs = append(medAbs, math.Abs(d.diffPct))
		if math.Abs(d.diffPct) > 20 {
			outliers = append(outliers, d)
		}
	}

	printAgg := func(label, header string, m map[string]*agg) {
		fmt.Printf("\n%s:\n", header)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			a := m[k]
			diff := 0.0
			if a.sumR > 0 {
				diff = (a.sumC - a.sumR) / a.sumR * 100
			}
			fmt.Printf("  %-22s n=%-6d reportado=%12.2f calculado=%12.2f diff=%+.2f%%\n",
				k, a.n, a.sumR, a.sumC, diff)
		}
		_ = label
	}

	printAgg("estrato", "Por estrato", byEstrato)
	printAgg("familia", "Por tipo de familia", byFamilia)
	printAgg("sexo", "Por sexo del causante", bySexo)
	printAgg("modal", "Por modalidad de renta", byModal)
	printAgg("edad", "Por edad del causante (buckets 10 años)", byEdad)

	sort.Float64s(medAbs)
	med := 0.0
	if len(medAbs) > 0 {
		if len(medAbs)%2 == 0 {
			med = (medAbs[len(medAbs)/2-1] + medAbs[len(medAbs)/2]) / 2
		} else {
			med = medAbs[len(medAbs)/2]
		}
	}
	fmt.Printf("\nDiferencia media absoluta: %.2f%% | mediana |diff|: %.2f%%\n",
		meanAbs(diffs), med)
	fmt.Printf("Outliers (>20%% diff): %d\n", len(outliers))
	if len(outliers) > 0 {
		fmt.Println("Primeros 5 outliers:")
		n := len(outliers)
		if n > 5 {
			n = 5
		}
		for _, o := range outliers[:n] {
			fmt.Printf("  SVS=%s reportado=%.2f calculado=%.2f diff=%+.1f%% estrato=%s sexo=%s edad=%d modal=%s renta=%.2f\n",
				o.svs, o.reportado, o.calculado, o.diffPct, o.estrato, o.sexo, o.edad, o.modalidad, o.renta)
		}
	}
}

// edadBucket groups ages into 10-year buckets for stratified reporting.
func edadBucket(edad int) string {
	if edad <= 0 {
		return "desconocida"
	}
	lo := (edad / 10) * 10
	return fmt.Sprintf("%d-%d", lo, lo+9)
}

// printPolicyDebug dumps the full per-person breakdown of a policy for manual
// comparison against the RIS reported reserves.
func printPolicyDebug(p models.RISPolicy, rb risBuildResult, calculated float64, retenida bool, contractDate time.Time) {
	fmt.Printf("\n=== DEBUG PÓLIZA SVS=%s ===\n", p.NumeroInternoSVS)
	fmt.Printf("FechaVigenciaInicial: %s | contractDate: %s\n", p.FechaVigenciaInicial.Format("2006-01-02"), contractDate.Format("2006-01-02"))
	fmt.Printf("TipoPension: %s | ModalidadRenta: %s | TipoRenta: %s\n", p.TipoPension, p.ModalidadRenta, p.TipoRenta)
	fmt.Printf("PrimaUnica: %s | RentaMensual: %s\n", p.PrimaUnica.String(), p.RentaMensual.String())
	fmt.Printf("TasaCostoEmision: %s%% | TasaVenta: %s%%\n", p.TasaCostoEmision.String(), p.TasaVenta.String())
	valuationDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	currentYear := valuationDate.Year() - contractDate.Year()
	fmt.Printf("currentYear (años desde emisión): %d | causanteVivo: %v\n", currentYear, rb.causanteVivo)
	pgMonths := models.GarantizedMonths(p.ModalidadRenta)
	fmt.Printf("PG (meses): %d | PG (años ceil): %d\n", pgMonths, (pgMonths+11)/12)

	fmt.Printf("\nPersonas en el RIS (%d):\n", len(p.Personas))
	fmt.Printf("%-3s %-4s %-6s %-8s %-12s %-6s %-6s %-10s %-12s %-12s %-12s %-12s %-12s\n",
		"#", "Rol", "TipoB", "Derecho", "Sexo", "EdadC", "EdadV", "Nacimiento", "Tabla", "PensionPers", "PctRenta", "RT-FIN2020", "RT-BASE-VIG")
	for i, per := range p.Personas {
		rol := risRolFor(per.TipoBeneficiario)
		edadC := ageAt(per.FechaNacimiento, contractDate)
		edadV := ageAt(per.FechaNacimiento, valuationDate)
		rtFin := per.ReserveForComparison(contractDate).InexactFloat64()
		fall := ""
		if per.FechaFallecimiento != nil {
			fall = " †" + per.FechaFallecimiento.Format("2006-01-02")
		}
		// Find the table assigned by buildGrupoFromRIS for this person.
		tabla := "-"
		for _, b := range rb.grupo.Beneficiarios {
			if b.TipoBeneficiarioC1194 == per.TipoBeneficiario && b.Sexo == per.Genero {
				tabla = b.TablaAsignada
				break
			}
		}
		if rb.grupo.Causante != nil && rb.grupo.Causante.TipoBeneficiarioC1194 == per.TipoBeneficiario {
			tabla = rb.grupo.Causante.TablaAsignada
		}
		fmt.Printf("%-3d %-4s %-6s %-8s %-12s %-6d %-6d %-10s %-12s %-12s %-12s %-12.2f %-12s%s\n",
			i, rol, per.TipoBeneficiario, per.DerechoPension, per.Genero,
			edadC, edadV, per.FechaNacimiento.Format("2006-01-02"),
			tabla,
			per.PensionPersona.StringFixed(2),
			per.PorcentajePension.StringFixed(2),
			rtFin,
			per.RTBaseTablaVigTotal.StringFixed(2),
			fall,
		)
	}

	reportedTotal := 0.0
	for _, per := range p.Personas {
		var r float64
		if retenida {
			r = per.ReserveForComparisonRetenida(contractDate).InexactFloat64()
		} else {
			r = per.ReserveForComparison(contractDate).InexactFloat64()
		}
		reportedTotal += r
	}
	diff := 0.0
	if reportedTotal > 0 {
		diff = (calculated - reportedTotal) / reportedTotal * 100
	}
	fmt.Printf("\nReserva reportada total: %.2f UF\n", reportedTotal)
	fmt.Printf("Reserva calculada:       %.2f UF\n", calculated)
	fmt.Printf("Diferencia:              %+.2f%%\n", diff)
	fmt.Printf("===========================================\n\n")
}

func meanAbs(diffs []diffRec) float64 {
	if len(diffs) == 0 {
		return 0
	}
	sum := 0.0
	for _, d := range diffs {
		sum += math.Abs(d.diffPct)
	}
	return sum / float64(len(diffs))
}
