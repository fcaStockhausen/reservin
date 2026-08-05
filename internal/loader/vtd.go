package loader

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"

	"reservas/internal/models"
)

// VTDLoader parses the CMF VTD (Vector Tasa de Descuento) Excel file into VTDPoint records.
type VTDLoader struct {
	path string
}

// NewVTDLoader creates a loader for the given Excel path.
func NewVTDLoader(path string) *VTDLoader {
	return &VTDLoader{path: path}
}

// Load reads every yearly sheet and returns one VTDPoint per (year, month, period).
func (l *VTDLoader) Load() ([]models.VTDPoint, error) {
	f, err := excelize.OpenFile(l.path)
	if err != nil {
		return nil, fmt.Errorf("open vtd excel: %w", err)
	}
	defer f.Close()

	var points []models.VTDPoint

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil {
			return nil, fmt.Errorf("read sheet %s: %w", sheet, err)
		}
		if len(rows) < 4 {
			continue
		}

		// Layout (0-based):
		//   row 1: "Mes" at col 1, then publication dates across columns.
		//   row 2: "Año" at col 1, then the series name ("Spot Rate AC").
		//   row 3+: col 1 = period (1..120), col 2+ = rates per publication date.
		dateRow := rows[1]
		seriesRow := rows[2]

		for c := 2; c < len(dateRow); c++ {
			pubDate, err := parseExcelDate(dateRow[c])
			if err != nil {
				continue
			}
			// Only ingest "Spot Rate AC" series; skip "Curva Cero" duplicates.
			if c < len(seriesRow) && !strings.EqualFold(strings.TrimSpace(seriesRow[c]), "Spot Rate AC") {
				continue
			}

			year := pubDate.Year()
			month := int(pubDate.Month())

			for r := 3; r < len(rows); r++ {
				periodStr := strings.TrimSpace(cellAt(rows, r, 1))
				if periodStr == "" {
					continue
				}
				period, err := strconv.Atoi(periodStr)
				if err != nil || period < 1 || period > 120 {
					continue
				}
				rateStr := strings.TrimSpace(cellAt(rows, r, c))
				if rateStr == "" {
					continue
				}
				rate, err := parsePercent(rateStr)
				if err != nil {
					continue
				}

				points = append(points, models.VTDPoint{
					Year:            year,
					Month:           month,
					Period:          period,
					Rate:            decimal.NewFromFloat(rate),
					PublicationDate: pubDate,
				})
			}
		}
	}

	return points, nil
}

// parseExcelDate accepts ISO-style date strings, Excel serial numbers,
// and short month-year labels like "Jan-25".
func parseExcelDate(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	// Try numeric Excel serial first.
	if f, err := strconv.ParseFloat(v, 64); err == nil {
		t, err := excelize.ExcelDateToTime(f, false)
		if err == nil {
			return t, nil
		}
	}
	// Try common formats (including short month-year labels from the VTD file).
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
		"Jan-06",
		"02-01-2006",
		"02/01/2006",
	}
	for _, layout := range formats {
		if t, err := time.Parse(layout, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized date %q", v)
}

// parsePercent parses rate strings like "3.08%" or "0.0308" into a float64
// representing the decimal rate (0.0308).
func parsePercent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	multiplier := 1.0
	if strings.HasSuffix(s, "%") {
		s = strings.TrimSuffix(s, "%")
		multiplier = 0.01
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return f * multiplier, nil
}
