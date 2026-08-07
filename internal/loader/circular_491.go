package loader

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"github.com/fcaStockhausen/reservin/internal/models"
)

// Circular491Loader parses the Circular N°491 (29.03.1985) mortality tables
// Excel export into MortalityTable records. The file is a manual transcription
// of the scanned SBIF circular and follows its own layout, distinct from the
// modern CMF xlsx:
//
//	Sheet layout (per table sheet):
//	  row 0:  "Tabla de Mortalidad <NAME>"
//	  row 4:  EDAD | q (por mil) | l(x) | e(x) | D(x) | N(x)
//	  rows 5+: age, qx-in-per-mille, ...
//	  trailing rows: transcription notes (free text)
//
// The q(x) column is expressed in per mille and must be divided by 1000 to
// obtain the death probability used across the repo. The Circular defines 4
// tables: B-85-H, B-85-M (SOBREVIVENCIA) and RV-85-H, RV-85-M (VEJEZ). There
// is no MI-85; pre-2005 invalidez is valued with the B-85 tables.
type Circular491Loader struct {
	path string
}

// NewCircular491Loader creates a loader for the given Excel path.
func NewCircular491Loader(path string) *Circular491Loader {
	return &Circular491Loader{path: path}
}

// vigencia491 is the Circular's publication date.
var vigencia491 = time.Date(1985, 3, 29, 0, 0, 0, 0, time.UTC)

// tableSpec maps a sheet name to its standard table metadata.
var circular491Sheets = map[string]struct {
	standard string
	sex      string
	tipo     string
}{
	"B-85-H":  {"B-H-1985", "H", string(models.TableTypeSobrevivencia)},
	"B-85-M":  {"B-M-1985", "M", string(models.TableTypeSobrevivencia)},
	"RV-85-H": {"RV-H-1985", "H", string(models.TableTypeVejez)},
	"RV-85-M": {"RV-M-1985", "M", string(models.TableTypeVejez)},
}

// Load reads the four mortality tables and returns normalized records.
func (l *Circular491Loader) Load() ([]models.MortalityTable, error) {
	f, err := excelize.OpenFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("open circular 491 excel: %w", err)
	}
	defer f.Close()

	sheetSet := map[string]bool{}
	for name := range circular491Sheets {
		sheetSet[name] = true
	}

	var records []models.MortalityTable
	for _, sheet := range f.GetSheetList() {
		spec, ok := circular491Sheets[sheet]
		if !ok {
			continue // skip "Indice" and any other sheet
		}
		recs, err := parse491Sheet(f, sheet, spec.standard, spec.sex, spec.tipo)
		if err != nil {
			return nil, fmt.Errorf("sheet %s: %w", sheet, err)
		}
		records = append(records, recs...)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no mortality records found in %s", l.path)
	}
	return records, nil
}

// parse491Sheet reads one table sheet, locating the header row dynamically.
func parse491Sheet(f *excelize.File, sheet, standard, sex, tipo string) ([]models.MortalityTable, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("empty sheet")
	}

	// Find the header row whose first cell is "EDAD".
	hdr := -1
	for i, r := range rows {
		if len(r) > 0 && strings.EqualFold(strings.TrimSpace(r[0]), "EDAD") {
			hdr = i
			break
		}
	}
	if hdr < 0 {
		return nil, fmt.Errorf("EDAD header not found")
	}

	var records []models.MortalityTable
	for i := hdr + 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) == 0 {
			continue
		}
		edadStr := strings.TrimSpace(row[0])
		if edadStr == "" {
			continue
		}
		edad, err := strconv.Atoi(edadStr)
		if err != nil {
			// Left the numeric region (trailing notes).
			continue
		}

		qxPerMille := ""
		if len(row) > 1 {
			qxPerMille = strings.TrimSpace(row[1])
		}
		if qxPerMille == "" {
			// A numeric edad with missing qx is a data gap; skip it.
			continue
		}
		qxPerMille = strings.ReplaceAll(qxPerMille, ",", ".")
		qxPerMille, err = normalize491Number(qxPerMille)
		if err != nil {
			return nil, fmt.Errorf("age %d: qx parse: %v", edad, err)
		}
		qxVal, err := strconv.ParseFloat(qxPerMille, 64)
		if err != nil {
			return nil, fmt.Errorf("age %d: qx value: %v", edad, err)
		}

		// q(x) is expressed in per mille -> divide by 1000.
		qx := decimal.NewFromFloat(qxVal / 1000.0)

		records = append(records, models.MortalityTable{
			NombreEstandar: standard,
			NombreOriginal: sheet,
			Sexo:           sex,
			TipoTabla:      tipo,
			AñoTabla:       1985,
			Edad:           edad,
			ProbMuerte:     qx,
			VigenciaInicio: vigencia491,
		})
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("no numeric rows found")
	}
	return records, nil
}

// normalize491Number cleans OCR transcription artifacts in numeric cells
// (e.g. trailing punctuation, underscores used as decimal separators).
func normalize491Number(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("empty number")
	}
	// Some transcription cells may use an underscore for the decimal point
	// or contain stray characters; keep only digits and one dot.
	var b strings.Builder
	seenDot := false
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.':
			if !seenDot {
				b.WriteRune(r)
				seenDot = true
			}
		default:
			// Ignore any other character (commas, spaces, stray text).
		}
	}
	out := b.String()
	if out == "" || out == "." {
		return "", fmt.Errorf("invalid number %q", s)
	}
	return out, nil
}
