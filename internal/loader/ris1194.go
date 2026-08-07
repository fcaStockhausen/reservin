package loader

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/models"
)

// RIS1194Loader streams the Circular 1194 RIS file (.vta), a fixed-width text
// file of 246-byte records, and decodes policies (Registro 2) and persons
// (Registro 3). It never loads the whole file in memory: records are grouped
// per policy and yielded as a stream.
type RIS1194Loader struct {
	path string
}

// NewRIS1194Loader creates a loader for the given .vta (or .zip) path.
func NewRIS1194Loader(path string) *RIS1194Loader {
	return &RIS1194Loader{path: path}
}

// RISHeader holds the Registro 1 control data.
type RISHeader struct {
	FechaHasta    time.Time
	NumRegistros2 int
	NumRegistros3 int
}

// decodeDecimal9 parses a COBOL 9(p)V(q) fixed-width number into a decimal.
// The raw value has (p+q) digits with q implied decimals.
func decodeDecimal9(s string, q int) decimal.Decimal {
	s = strings.TrimSpace(s)
	if s == "" {
		return decimal.Zero
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	if len(s) <= q {
		s = strings.Repeat("0", q+1-len(s)) + s
	}
	intPart := s[:len(s)-q]
	decPart := s[len(s)-q:]
	d, err := decimal.NewFromString(intPart + "." + decPart)
	if err != nil {
		return decimal.Zero
	}
	if neg {
		return d.Neg()
	}
	return d
}

// parseRISDate parses a YYYYMMDD fixed field; returns nil when all zeros.
func parseRISDate(s string) *time.Time {
	s = strings.TrimSpace(s)
	if len(s) != 8 || s == "00000000" || strings.Trim(s, "0") == "" {
		return nil
	}
	t, err := time.Parse("20060102", s)
	if err != nil {
		return nil
	}
	return &t
}

// Stream decodes the RIS file and yields each policy through the channel.
// If maxPolicies > 0, it stops after that many policies. A policy is emitted
// once its Registro 2 is seen and all subsequent Registro 3 with the same
// NUMERO-INTERNO-SVS are attached.
func (l *RIS1194Loader) Stream(maxPolicies int) (<-chan models.RISPolicy, <-chan error) {
	policies := make(chan models.RISPolicy)
	errs := make(chan error, 1)

	go func() {
		defer close(policies)
		defer close(errs)

		f, err := os.Open(l.path)
		if err != nil {
			errs <- fmt.Errorf("open ris: %w", err)
			return
		}
		defer f.Close()

		var reader io.Reader = f
		if strings.HasSuffix(strings.ToLower(l.path), ".gz") {
			gz, err := gzip.NewReader(f)
			if err != nil {
				errs <- fmt.Errorf("gzip: %w", err)
				return
			}
			defer gz.Close()
			reader = gz
		}

		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

		var current *models.RISPolicy
		emitted := 0

		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) < 1 {
				continue
			}
			switch line[0] {
			case '1':
				// header; ignore (we use it only for control totals)
			case '2':
				if current != nil {
					policies <- *current
					emitted++
					if maxPolicies > 0 && emitted >= maxPolicies {
						return
					}
				}
				p := parseRISReg2(line)
				current = &p
			case '3':
				if current == nil {
					continue
				}
				person := parseRISReg3(line)
				current.Personas = append(current.Personas, person)
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- fmt.Errorf("scan ris: %w", err)
			return
		}
		if current != nil {
			policies <- *current
			emitted++
		}
	}()

	return policies, errs
}

// LoadHeaders reads only the Registro 1 control totals.
func (l *RIS1194Loader) LoadHeader() (*RISHeader, error) {
	f, err := os.Open(l.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) < 16 {
			continue
		}
		if line[0] != '1' {
			continue
		}
		h := &RISHeader{}
		if t := parseRISDate(string(line[1:9])); t != nil {
			h.FechaHasta = *t
		}
		h.NumRegistros2 = atoiSafe(string(line[9:18]))
		h.NumRegistros3 = atoiSafe(string(line[18:27]))
		return h, nil
	}
	return nil, fmt.Errorf("registro 1 not found")
}

func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

func parseRISReg2(line []byte) models.RISPolicy {
	str := string(line)
	pol := models.RISPolicy{
		NumeroInternoSVS:            strings.TrimSpace(sub(str, 2, 7)),
		NumeroPersonas:              atoiSafe(sub(str, 8, 9)),
		TipoPension:                 sub(str, 10, 11),
		CompaniaObligada:            sub(str, 12, 12),
		VigenciaPension:             sub(str, 13, 13),
		CodigoAFP:                   sub(str, 14, 15),
		TipoAfiliado:                sub(str, 16, 16),
		CuentaIndividual:            decodeDecimal9(sub(str, 17, 23), 2),
		IngresoBaseUF:               decodeDecimal9(sub(str, 24, 28), 2),
		PorcentajeCubierto:          decodeDecimal9(sub(str, 29, 31), 0),
		PrimaUnica:                  decodeDecimal9(sub(str, 40, 46), 2),
		RentaMensual:                decodeDecimal9(sub(str, 47, 51), 2),
		TipoRenta:                   sub(str, 52, 55),
		ModalidadRenta:              sub(str, 56, 59),
		TipoOperacionRV:             sub(str, 60, 61),
		PeriodoAumento:              atoiSafe(sub(str, 62, 64)),
		PorcentajeAumento:           decodeDecimal9(sub(str, 65, 69), 2),
		TasaCostoEmision:            decodeDecimal9(sub(str, 70, 73), 2),
		TasaVenta:                   decodeDecimal9(sub(str, 74, 77), 2),
		NumeroReaseguro:             atoiSafe(sub(str, 78, 78)),
		PolizaConAnticipo:           sub(str, 163, 163),
		FechaRecalculoActual:        parseRISDate(sub(str, 164, 171)),
		FechaRecalculoAnterior:      parseRISDate(sub(str, 172, 179)),
		RentaAnteriorRecalcActual:   decodeDecimal9(sub(str, 180, 184), 2),
		RentaAnteriorRecalcAnterior: decodeDecimal9(sub(str, 185, 189), 2),
		NumeroSVSRelacionado:        strings.TrimSpace(sub(str, 190, 195)),
	}
	if t := parseRISDate(sub(str, 32, 39)); t != nil {
		pol.FechaVigenciaInicial = *t
	}
	return pol
}

func parseRISReg3(line []byte) models.RISPerson {
	str := string(line)
	p := models.RISPerson{
		NumeroInternoSVS:          strings.TrimSpace(sub(str, 2, 7)),
		NumeroOrden:               atoiSafe(sub(str, 8, 9)),
		Genero:                    sub(str, 10, 10),
		TipoBeneficiario:          sub(str, 11, 12),
		SituacionInvalidez:        sub(str, 13, 13),
		DerechoPension:            sub(str, 38, 39),
		RequisitoPension:          sub(str, 40, 40),
		RelacionHijoMadre:         atoiSafe(sub(str, 41, 42)),
		DerechoAcrecer:            sub(str, 51, 51),
		PorcentajePension:         decodeDecimal9(sub(str, 52, 56), 2),
		PensionPersona:            decodeDecimal9(sub(str, 57, 61), 2),
		PctAnticipoRV:             decodeDecimal9(sub(str, 62, 65), 2),
		PctPensionPostAnticipo:    decodeDecimal9(sub(str, 66, 69), 2),
		RTBaseTotal:               decodeDecimal9(sub(str, 78, 84), 2),
		RTBaseTablaVigTotal:       decodeDecimal9(sub(str, 85, 91), 2),
		RTFinanciera200485:        decodeDecimal9(sub(str, 92, 98), 2),
		RTFinancieraStock85:       decodeDecimal9(sub(str, 99, 105), 2),
		RTFinanciera200406:        decodeDecimal9(sub(str, 106, 112), 2),
		RTFinanciera200906:        decodeDecimal9(sub(str, 113, 119), 2),
		RTFinanciera2014:          decodeDecimal9(sub(str, 120, 126), 2),
		RTFinanciera2020:          decodeDecimal9(sub(str, 127, 133), 2),
		RTBaseRetenida:            decodeDecimal9(sub(str, 134, 140), 2),
		RTBaseTablaVigRetenida:    decodeDecimal9(sub(str, 141, 147), 2),
		RTFin200485Retenida:       decodeDecimal9(sub(str, 148, 154), 2),
		RTFinStock85Retenida:      decodeDecimal9(sub(str, 155, 161), 2),
		RTFin200406Retenida:       decodeDecimal9(sub(str, 162, 168), 2),
		RTFin200906Retenida:       decodeDecimal9(sub(str, 169, 175), 2),
		RTFin2014Retenida:         decodeDecimal9(sub(str, 176, 182), 2),
		RTFin2020Retenida:         decodeDecimal9(sub(str, 183, 189), 2),
		MontoBeneficioEstatal1:    decodeDecimal9(sub(str, 190, 197), 6),
		MontoBeneficioEstatal2:    decodeDecimal9(sub(str, 198, 205), 6),
		MontoBeneficioEstatal3:    decodeDecimal9(sub(str, 206, 213), 6),
		TipoPagoBeneficioEstatal1: sub(str, 214, 214),
		TipoPagoBeneficioEstatal2: sub(str, 215, 215),
		TipoPagoBeneficioEstatal3: sub(str, 216, 216),
		BonoPorHijo1:              decodeDecimal9(sub(str, 217, 222), 4),
		BonoPorHijo2:              decodeDecimal9(sub(str, 223, 228), 4),
		BonoPorHijo3:              decodeDecimal9(sub(str, 229, 234), 4),
	}
	if t := parseRISDate(sub(str, 14, 21)); t != nil {
		p.FechaNacimiento = *t
	}
	if t := parseRISDate(sub(str, 22, 29)); t != nil {
		p.FechaFallecimiento = t
	}
	if t := parseRISDate(sub(str, 30, 37)); t != nil {
		p.FechaInvalidez = t
	}
	if t := parseRISDate(sub(str, 43, 50)); t != nil {
		p.FechaNacHijoMenor = t
	}
	if t := parseRISDate(sub(str, 70, 77)); t != nil {
		p.FechaAnticipoRV = t
	}
	return p
}

// sub extracts a fixed-width slice [start,end] inclusive, 1-indexed.
func sub(s string, start, end int) string {
	if start < 1 {
		start = 1
	}
	if end > len(s) {
		end = len(s)
	}
	if start > len(s) || start > end {
		return ""
	}
	return s[start-1 : end]
}
