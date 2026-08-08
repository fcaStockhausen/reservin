package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fcaStockhausen/reservin/internal/loader"
	"github.com/fcaStockhausen/reservin/internal/models"
)

// risDump es el lector canónico del RIS (Circular 1194). Reemplaza los
// scripts ad-hoc de inspección: lee cualquier .vta/.zip con el mismo parser
// que usa la validación, imprime las pólizas con sus códigos decodificados
// por el diccionario (internal/models/ris_dict.go) y no requiere DB ni config.
//
// Modos:
//
//	-ris-dump <path>        lee y dumps pólizas (primeras 10 si no hay filtro)
//	  -svs <id>             solo esa póliza (recorre todo el archivo)
//	  -n <N>                máx. pólizas a dump (negativo = ilimitado)
//	  -filter dead|alive    solo pólizas con causante fallecido/vivo
//	  -json                 salida NDJSON (una póliza por línea, para jq)
//	  -legend               imprime layout + diccionario de códigos y sale
//	  -scan-codes           enumera los códigos observados en el archivo
func risDump(path, svs string, n int, filter string, asJSON, legend, scanCodes bool) {
	if legend {
		printRISLegend()
		if path == "" {
			return
		}
	}
	if scanCodes {
		scanRISCodes(path)
		return
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "ris-dump: falta el archivo .vta (uso: -ris-dump <path>)")
		os.Exit(1)
	}
	if svs == "" && n == 0 {
		n = 10
	}

	ld := loader.NewRIS1194Loader(path)
	if h, _ := ld.LoadHeader(); h != nil && !asJSON {
		// El Registro 1 real de la CMF solo trae FECHA-HASTA y el RUT de la
		// compañía (no conteos); LoadHeader no lee la razón social.
		fmt.Printf("Período: %s\n", h.FechaHasta.Format("2006-01-02"))
	}
	policies, errs := ld.Stream(-1)

	valuationDate := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
	scanned, dumped, skipped := 0, 0, 0

loop:
	for {
		select {
		case p, ok := <-policies:
			if !ok {
				policies = nil
				break // breaks the select, not the for; falls to nil check
			}
			scanned++
			dead := causanteMuerto(p)
			matchSVS := svs == "" || p.NumeroInternoSVS == svs
			matchFiltro := true
			switch filter {
			case "dead":
				matchFiltro = dead
			case "alive":
				matchFiltro = !dead
			}
			if !matchSVS || !matchFiltro {
				if svs != "" {
					// no encontrada aún; seguir recorriendo
					continue
				}
				skipped++
				continue
			}
			if asJSON {
				emitRISJSON(p)
			} else {
				printRISPolicy(p, valuationDate, dead)
			}
			dumped++
			// Terminación anticipada: al cortar policies=nil el goroutine del
			// stream queda bloqueado mandando (nunca cierra errs), así que hay
			// que salir del loop con break etiquetado (ver bug de deadlock en
			// AGENTS.md).
			if (svs == "" && n > 0 && dumped >= n) || (svs != "" && dumped >= 1) {
				break loop
			}
		case err, ok := <-errs:
			if ok && err != nil {
				fmt.Fprintln(os.Stderr, "ris-dump:", err)
			}
			errs = nil
		}
		if policies == nil && errs == nil {
			break loop
		}
	}

	if svs != "" && dumped == 0 {
		fmt.Fprintf(os.Stderr, "SVS %s no encontrada en %s\n", svs, path)
	}
	if !asJSON && !(svs != "" && dumped == 0) {
		fmt.Printf("\n--- escaneadas: %d | dumpadas: %d | omitidas por filtro: %d ---\n",
			scanned, dumped, skipped)
	}
}

func causanteMuerto(p models.RISPolicy) bool {
	for i := range p.Personas {
		per := &p.Personas[i]
		if per.TipoBeneficiario == models.C1194Afiliado && per.FechaFallecimiento != nil {
			return true
		}
	}
	return false
}

// printRISPolicy imprime una póliza con los códigos decodificados por el
// diccionario canónico (models/ris_dict.go).
func printRISPolicy(p models.RISPolicy, valuationDate time.Time, dead bool) {
	muerto := "vivo"
	if dead {
		muerto = "FALLECIDO"
	}
	fmt.Printf("\n=== PÓLIZA SVS=%s === (%s)\n", p.NumeroInternoSVS, muerto)
	fmt.Printf("  tipo_pension=%s (%s)\n", p.TipoPension,
		models.LookupRISCode("TIPO-PENSION", p.TipoPension))
	fmt.Printf("  modalidad_renta=%s (%s)\n", p.ModalidadRenta,
		models.DescribeModalidadRenta(p.ModalidadRenta, p.PeriodoAumento))
	fmt.Printf("  tipo_renta=%s (%s)\n", p.TipoRenta, modelosTipoRenta(p.TipoRenta))
	fmt.Printf("  vigencia=%s (%s) | compañía_obligada=%s (%s)\n",
		p.VigenciaPension, models.LookupRISCode("VIGENCIA-PENSION", p.VigenciaPension),
		p.CompaniaObligada, models.LookupRISCode("COMPANIA-OBLIGADA", p.CompaniaObligada))
	fmt.Printf("  afiliado=%s (%s) | AFP=%s | operación_RV=%s | reaseguro=%d\n",
		p.TipoAfiliado, models.LookupRISCode("TIPO-AFILIADO", p.TipoAfiliado),
		p.CodigoAFP, p.TipoOperacionRV, p.NumeroReaseguro)
	fmt.Printf("  fecha_vigencia=%s | prima=%.2f UF | renta_mensual=%.2f UF | TCj=%s%% | TVj=%s%%\n",
		p.FechaVigenciaInicial.Format("2006-01-02"), p.PrimaUnica.InexactFloat64(),
		p.RentaMensual.InexactFloat64(), p.TasaCostoEmision.String(), p.TasaVenta.String())

	contractDate := p.FechaVigenciaInicial
	if contractDate.IsZero() {
		contractDate = time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	fmt.Printf("  personas (%d):\n", len(p.Personas))
	fmt.Printf("    %-3s %-24s %-4s %-4s %-4s %-6s %-6s %-10s %-10s %-6s %-8s %-6s %-12s %-12s\n",
		"#", "rol", "C1194", "der", "sexo", "edad_c", "edad_v", "nacimiento", "fallecimiento", "inv", "pensión", "pct%", "RT-BASE-VIG", "RT-FIN-2020")
	for _, per := range p.Personas {
		rol := risRolFor(per.TipoBeneficiario)
		fallec := "-"
		if per.FechaFallecimiento != nil {
			fallec = per.FechaFallecimiento.Format("2006-01-02")
		}
		edadC := ageAt(per.FechaNacimiento, contractDate)
		edadV := ageAt(per.FechaNacimiento, valuationDate)
		fmt.Printf("    %-3d %-24s %-4s %-4s %-4s %-6d %-6d %-10s %-10s %-6s %-8s %-6s %-12.2f %-12.2f\n",
			per.NumeroOrden, rol, per.TipoBeneficiario, per.DerechoPension, per.Genero,
			edadC, edadV, per.FechaNacimiento.Format("2006-01-02"), fallec,
			per.SituacionInvalidez, per.PensionPersona.StringFixed(2),
			per.PorcentajePension.StringFixed(0), per.RTBaseTablaVigTotal.InexactFloat64(),
			per.RTFinanciera2020.InexactFloat64())
	}
}

func modelosTipoRenta(code string) string {
	switch {
	case code == "0000":
		return "póliza pre-2005 (campo no informado)"
	case code == "1000":
		return "inmediata"
	case code == "3000":
		return "inmediata con retiro programado"
	case strings.HasPrefix(code, "2"):
		return fmt.Sprintf("diferida %s meses", strings.TrimLeft(code[1:], "0"))
	default:
		return "?"
	}
}

// emitRISJSON emite una póliza como JSON en una línea (NDJSON).
type risPolicyJSON struct {
	SVS            string          `json:"svs"`
	NumeroPersonas int             `json:"numero_personas"`
	TipoPension    string          `json:"tipo_pension"`
	TipoPensionTxt string          `json:"tipo_pension_txt"`
	Vigencia       string          `json:"vigencia_pension"`
	VigenciaTxt    string          `json:"vigencia_pension_txt"`
	CompaniaOblig  string          `json:"compania_obligada"`
	AFP            string          `json:"codigo_afp"`
	TipoAfiliado   string          `json:"tipo_afiliado"`
	FechaVigencia  string          `json:"fecha_vigencia_inicial"`
	PrimaUnica     float64         `json:"prima_unica_uf"`
	RentaMensual   float64         `json:"renta_mensual_uf"`
	TipoRenta      string          `json:"tipo_renta"`
	Modalidad      string          `json:"modalidad_renta"`
	ModalidadTxt   string          `json:"modalidad_renta_txt"`
	PeriodoAumento int             `json:"periodo_aumento_meses"`
	TasaCtoEmision float64         `json:"tasa_costo_emision_pct"`
	TasaVenta      float64         `json:"tasa_venta_pct"`
	Personas       []risPersonJSON `json:"personas"`
}

type risPersonJSON struct {
	Orden              int     `json:"numero_orden"`
	Rol                string  `json:"rol"`
	TipoBeneficiario   string  `json:"tipo_beneficiario"`
	DerechoPension     string  `json:"derecho_pension"`
	Genero             string  `json:"genero"`
	SituacionInvalidez string  `json:"situacion_invalidez"`
	FechaNacimiento    string  `json:"fecha_nacimiento"`
	FechaFallecimiento string  `json:"fecha_fallecimiento,omitempty"`
	PorcentajePension  float64 `json:"porcentaje_pension_pct"`
	PensionPersona     float64 `json:"pension_persona_uf"`
	RTBaseTablaVigente float64 `json:"rt_base_tabla_vigente_uf"`
	RTFinanciera2020   float64 `json:"rt_financiera_2020_uf"`
	RTBaseRetenida     float64 `json:"rt_base_retenida_uf"`
}

func emitRISJSON(p models.RISPolicy) {
	out := risPolicyJSON{
		SVS:            p.NumeroInternoSVS,
		NumeroPersonas: p.NumeroPersonas,
		TipoPension:    p.TipoPension,
		TipoPensionTxt: models.LookupRISCode("TIPO-PENSION", p.TipoPension),
		Vigencia:       p.VigenciaPension,
		VigenciaTxt:    models.LookupRISCode("VIGENCIA-PENSION", p.VigenciaPension),
		CompaniaOblig:  p.CompaniaObligada,
		AFP:            p.CodigoAFP,
		TipoAfiliado:   p.TipoAfiliado,
		FechaVigencia:  p.FechaVigenciaInicial.Format("2006-01-02"),
		PrimaUnica:     p.PrimaUnica.InexactFloat64(),
		RentaMensual:   p.RentaMensual.InexactFloat64(),
		TipoRenta:      p.TipoRenta,
		Modalidad:      p.ModalidadRenta,
		ModalidadTxt:   models.DescribeModalidadRenta(p.ModalidadRenta, p.PeriodoAumento),
		PeriodoAumento: p.PeriodoAumento,
		TasaCtoEmision: p.TasaCostoEmision.InexactFloat64(),
		TasaVenta:      p.TasaVenta.InexactFloat64(),
		Personas:       make([]risPersonJSON, 0, len(p.Personas)),
	}
	for _, per := range p.Personas {
		ff := ""
		if per.FechaFallecimiento != nil {
			ff = per.FechaFallecimiento.Format("2006-01-02")
		}
		out.Personas = append(out.Personas, risPersonJSON{
			Orden:              per.NumeroOrden,
			Rol:                string(risRolFor(per.TipoBeneficiario)),
			TipoBeneficiario:   per.TipoBeneficiario,
			DerechoPension:     per.DerechoPension,
			Genero:             per.Genero,
			SituacionInvalidez: per.SituacionInvalidez,
			FechaNacimiento:    per.FechaNacimiento.Format("2006-01-02"),
			FechaFallecimiento: ff,
			PorcentajePension:  per.PorcentajePension.InexactFloat64(),
			PensionPersona:     per.PensionPersona.InexactFloat64(),
			RTBaseTablaVigente: per.RTBaseTablaVigTotal.InexactFloat64(),
			RTFinanciera2020:   per.RTFinanciera2020.InexactFloat64(),
			RTBaseRetenida:     per.RTBaseRetenida.InexactFloat64(),
		})
	}
	b, _ := json.Marshal(out)
	fmt.Println(string(b))
}

// printRISLegend imprime el diccionario canónico de códigos y el layout del
// archivo. Es la referencia para interpretar cualquier póliza sin depender de
// conocimiento de sesión.
func printRISLegend() {
	fmt.Println("===================================================================")
	fmt.Println("DICCIONARIO DE CÓDIGOS — Anexo Técnico Circular 1194 (RIS)")
	fmt.Println("===================================================================")
	fmt.Println("Fuentes: C1194 (20.01.1995) y cir_1772/2184/2208/2308 en")
	fmt.Println("docs/normativo/; códigos modernos publicados en el módulo SEIL")
	fmt.Println("(Anexo Técnico 'Seguros previsionales e índices de cobertura').")
	fmt.Println("Los marcados SEIL se enumeraron del archivo RIS real y faltan")
	fmt.Println("por confirmar en la tabla oficial del módulo SEIL.")
	fmt.Println("Registros: 1=identificación compañía, 2=póliza, 3=persona, 4=totales.")
	fmt.Println("Formato: texto fijo, campos numéricos 9(p)V(q) con decimales implícitos.")
	fmt.Println()
	for _, f := range models.RISDictionary() {
		fmt.Printf("  %s (%s, pos %s)\n", f.Name, f.Campo, f.Pos)
		fmt.Printf("      %s\n", f.Desc)
		for _, rc := range f.Codes {
			if rc.Note != "" {
				fmt.Printf("      %-12s %s  [%s]\n", rc.Code, rc.Label, rc.Note)
			} else {
				fmt.Printf("      %-12s %s\n", rc.Code, rc.Label)
			}
		}
	}
	fmt.Println()
}

// scanRISCodes recorre el archivo y enumera los valores observados de cada
// campo de códigos. Útil para descubrir códigos no documentados (ver -legend).
func scanRISCodes(path string) {
	ld := loader.NewRIS1194Loader(path)
	policies, errs := ld.Stream(-1)
	counters := map[string]map[string]int{}
	for _, f := range models.RISDictionary() {
		counters[f.Name] = map[string]int{}
	}
	// Campos R2 adicionales no incluidos en el diccionario.
	extraR2 := map[string]map[string]int{
		"TIPO-OPERACION-RV": {},
	}
	for name, m := range extraR2 {
		counters[name] = m
	}
	n := 0
	for {
		select {
		case p, ok := <-policies:
			if !ok {
				policies = nil
				break
			}
			n++
			counters["TIPO-PENSION"][orVacio(p.TipoPension)]++
			counters["COMPANIA-OBLIGADA"][orVacio(p.CompaniaObligada)]++
			counters["VIGENCIA-PENSION"][orVacio(p.VigenciaPension)]++
			counters["CODIGO-AFP"][orVacio(p.CodigoAFP)]++
			counters["TIPO-AFILIADO"][orVacio(p.TipoAfiliado)]++
			counters["TIPO-RENTA"][orVacio(p.TipoRenta)]++
			counters["MODALIDAD-RENTA"][orVacio(p.ModalidadRenta)]++
			counters["TIPO-OPERACION-RV"][orVacio(p.TipoOperacionRV)]++
			for _, per := range p.Personas {
				counters["GENERO"][orVacio(per.Genero)]++
				counters["TIPO-BENEFICIARIO"][orVacio(per.TipoBeneficiario)]++
				counters["SITUACION-INVALIDEZ"][orVacio(per.SituacionInvalidez)]++
				counters["DERECHO-PENSION"][orVacio(per.DerechoPension)]++
				counters["REQUISITO-PENSION"][orVacio(per.RequisitoPension)]++
				counters["DERECHO-ACRECER"][orVacio(per.DerechoAcrecer)]++
				counters["TIPO-PAGO-BENEFICIO-ESTATAL"][orVacio(per.TipoPagoBeneficioEstatal1)]++
				counters["TIPO-PAGO-BENEFICIO-ESTATAL"][orVacio(per.TipoPagoBeneficioEstatal2)]++
				counters["TIPO-PAGO-BENEFICIO-ESTATAL"][orVacio(per.TipoPagoBeneficioEstatal3)]++
			}
		case err, ok := <-errs:
			if ok && err != nil {
				fmt.Fprintln(os.Stderr, "scan-codes:", err)
			}
			errs = nil
		}
		if policies == nil && errs == nil {
			break
		}
	}
	fmt.Printf("pólizas escaneadas: %d\n\n", n)
	names := make([]string, 0, len(counters))
	for name := range counters {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		fmt.Printf("%s:\n", name)
		codes := make([]string, 0, len(counters[name]))
		for code := range counters[name] {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		for _, code := range codes {
			fmt.Printf("  %-12q %d\n", code, counters[name][code])
		}
	}
}

func orVacio(s string) string {
	if s == "" || strings.TrimSpace(s) == "" {
		return "(vacío)"
	}
	return s
}
