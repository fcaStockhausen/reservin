package loader

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"reservas/internal/models"
)

// MortalityLoader parses the CMF mortality tables Excel file into MortalityTable records.
type MortalityLoader struct {
	path string
}

// NewMortalityLoader creates a loader for the given Excel path.
func NewMortalityLoader(path string) *MortalityLoader {
	return &MortalityLoader{path: path}
}

// Load reads every table in every sheet and returns normalized records.
func (l *MortalityLoader) Load() ([]models.MortalityTable, error) {
	f, err := excelize.OpenFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("open mortality excel: %w", err)
	}
	defer f.Close()

	var records []models.MortalityTable
	// Dedupe by (standard name, edad): some tables (e.g. CB-2020-HOMBRES) are
	// published in both the Vejez and Sobrevivencia sheets with identical data.
	seen := make(map[string]bool)

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", sheet, err)
		}
		if len(rows) == 0 {
			continue
		}

		// Find every column whose header cell (within the first 5 rows) is "Edad".
		// Each such column marks the start of a mortality table group.
		for _, hdr := range findEdadHeaders(rows) {
			recs, err := l.parseTableGroup(rows, hdr, sheet)
			if err != nil {
				return nil, fmt.Errorf("sheet %s col %d: %w", sheet, hdr.col, err)
			}
			for _, rec := range recs {
				key := rec.NombreEstandar + "|" + strconv.Itoa(rec.Edad)
				if seen[key] {
					continue
				}
				seen[key] = true
				records = append(records, rec)
			}
		}
	}

	return records, nil
}

type tableHeader struct {
	row int
	col int
}

// findEdadHeaders scans the first 5 rows for cells equal to "Edad" (case-insensitive,
// trimmed). Each hit identifies the first column of a table group.
func findEdadHeaders(rows [][]string) []tableHeader {
	var headers []tableHeader
	maxRow := 5
	if len(rows) < maxRow {
		maxRow = len(rows)
	}
	for r := 0; r < maxRow; r++ {
		for c := 0; c < len(rows[r]); c++ {
			if strings.EqualFold(strings.TrimSpace(rows[r][c]), "Edad") {
				headers = append(headers, tableHeader{row: r, col: c})
			}
		}
	}
	return headers
}

// parseTableGroup reads the table whose "Edad" header is at (hdr.row, hdr.col).
// Column layout: edad=col, qx=col+1, factor_aax=col+2 (optional).
func (l *MortalityLoader) parseTableGroup(rows [][]string, hdr tableHeader, sheet string) ([]models.MortalityTable, error) {
	edadCol := hdr.col
	qxCol := hdr.col + 1
	aaxCol := hdr.col + 2

	// Determine the table name from row 0 at the header column.
	rawName := cellAt(rows, 0, edadCol)
	meta := parseTableName(rawName, sheet)

	var records []models.MortalityTable
	for r := hdr.row + 1; r < len(rows); r++ {
		edadStr := strings.TrimSpace(cellAt(rows, r, edadCol))
		if edadStr == "" {
			continue
		}
		edad, err := strconv.Atoi(edadStr)
		if err != nil {
			// Non-numeric edad means we left the data region.
			continue
		}
		qxStr := strings.TrimSpace(cellAt(rows, r, qxCol))
		if qxStr == "" {
			continue
		}
		qx, err := strconv.ParseFloat(qxStr, 64)
		if err != nil {
			continue
		}

		rec := models.MortalityTable{
			NombreEstandar: meta.standard,
			NombreOriginal: meta.original,
			Sexo:           meta.sex,
			TipoTabla:      meta.tipo,
			AñoTabla:       meta.year,
			Edad:           edad,
			ProbMuerte:     decimal.NewFromFloat(qx),
			VigenciaInicio: time.Date(meta.year, 1, 1, 0, 0, 0, 0, time.UTC),
		}

		// Factor Aax is optional and only present in pre-2020 tables.
		if aaxStr := strings.TrimSpace(cellAt(rows, r, aaxCol)); aaxStr != "" {
			if aax, err := strconv.ParseFloat(aaxStr, 64); err == nil {
				rec.FactorAax = decimal.NewFromFloat(aax)
			}
		}

		records = append(records, rec)
	}

	return records, nil
}

// cellAt safely retrieves a string cell from ragged rows.
func cellAt(rows [][]string, r, c int) string {
	if r < 0 || r >= len(rows) {
		return ""
	}
	if c < 0 || c >= len(rows[r]) {
		return ""
	}
	return rows[r][c]
}

var yearRe = regexp.MustCompile(`\d{4}`)

type tableMeta struct {
	standard string
	original string
	sex      string
	tipo     string
	year     int
}

// parseTableName normalizes the messy Excel table labels into the CMF standard
// naming convention "<PREFIX>-<SEX>-<YEAR>" (e.g. "CB-H-2020", "RV-M-2009").
func parseTableName(raw, sheet string) tableMeta {
	original := strings.TrimSpace(raw)
	original = strings.TrimPrefix(strings.ToUpper(original), "TABLA")
	original = strings.TrimSpace(original)

	// Derive sex from the label, falling back to the sheet name.
	sex := "H"
	upper := strings.ToUpper(original)
	switch {
	case strings.Contains(upper, "MUJ"):
		sex = "M"
	case strings.Contains(upper, "HOM"):
		sex = "H"
	default:
		if strings.HasSuffix(sheet, "Mujeres") {
			sex = "M"
		}
	}

	// Extract the table year.
	year := 0
	if m := yearRe.FindString(original); m != "" {
		year, _ = strconv.Atoi(m)
	}

	// Extract the prefix token (letters before the first digit/hyphen block).
	prefix := prefixFrom(original)

	tipo := tipoForPrefix(prefix)

	standard := fmt.Sprintf("%s-%s-%d", prefix, sex, year)
	if prefix == "" || year == 0 {
		// Fallback: keep the cleaned original so we never store an empty name.
		standard = original
	}

	return tableMeta{
		standard: standard,
		original: original,
		sex:      sex,
		tipo:     tipo,
		year:     year,
	}
}

// prefixFrom pulls the leading alphabetic code (RV, CB, B, MI) out of the label.
func prefixFrom(label string) string {
	label = strings.ToUpper(strings.TrimSpace(label))
	// Remove spaces around hyphens so "MI - 2020" splits cleanly.
	label = strings.ReplaceAll(label, " - ", "-")
	label = strings.ReplaceAll(label, " ", "")
	tokens := strings.Split(label, "-")
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		// A valid prefix is purely alphabetic and at least 1 char.
		if t != "" && isAlpha(t) {
			return strings.ToUpper(t)
		}
	}
	return ""
}

func isAlpha(s string) bool {
	for _, r := range s {
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')) {
			return false
		}
	}
	return len(s) > 0
}

// tipoForPrefix maps a table prefix to its regulatory category.
func tipoForPrefix(prefix string) string {
	switch strings.ToUpper(prefix) {
	case "MI":
		return string(models.TableTypeInvalidez)
	case "RV":
		return string(models.TableTypeVejez)
	case "B", "CB":
		return string(models.TableTypeSobrevivencia)
	default:
		return ""
	}
}
