package scenario

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// LoadBuiltin loads a predefined scenario by name from the catalog.
func LoadBuiltin(name string) (*Scenario, error) {
	yamlStr, exists := BuiltinScenarios[name]
	if !exists {
		return nil, fmt.Errorf("unknown builtin scenario: %s (available: %s)",
			name, strings.Join(builtinNames(), ", "))
	}
	return parseYAML(yamlStr)
}

func parseYAML(yamlStr string) (*Scenario, error) {
	var s Scenario
	if err := yamlUnmarshal([]byte(yamlStr), &s); err != nil {
		return nil, err
	}
	if s.Horizon == 0 {
		s.Horizon = 50
	}
	if s.Policy.ModalidadRenta == "" {
		s.Policy.ModalidadRenta = "1000"
	}
	if s.Policy.TipoPension == "" {
		s.Policy.TipoPension = "04"
	}
	return &s, nil
}

func builtinNames() []string {
	var names []string
	for k := range BuiltinScenarios {
		names = append(names, k)
	}
	return names
}

// ExportSimulation writes a single simulation result to Excel.
func ExportSimulation(result *SimulationResult, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Evolucion"
	f.SetSheetName(f.GetSheetName(0), sheet)

	headers := []string{"Ano", "Edad Causante", "Reserva", "Reserva Base", "Descalce Bruto", "Descalce Reconocido", "Miembros Vivos", "Eventos"}
	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
	})
	f.SetCellStyle(sheet, "A1", cellName(len(headers), 1), style)

	for i, step := range result.Steps {
		row := i + 2
		f.SetCellValue(sheet, cellName(1, row), step.Year)
		f.SetCellValue(sheet, cellName(2, row), step.CausanteAge)
		f.SetCellValue(sheet, cellName(3, row), step.ReserveValue.InexactFloat64())
		f.SetCellValue(sheet, cellName(4, row), step.ReservaBase.InexactFloat64())
		f.SetCellValue(sheet, cellName(5, row), step.DescalceBruto.InexactFloat64())
		f.SetCellValue(sheet, cellName(6, row), step.DescalceReconocido.InexactFloat64())
		f.SetCellValue(sheet, cellName(7, row), step.MembersAlive)
		events := ""
		if len(step.Events) > 0 {
			events = strings.Join(step.Events, "; ")
		}
		f.SetCellValue(sheet, cellName(8, row), events)
	}

	f.SetColWidth(sheet, "A", "A", 8)
	f.SetColWidth(sheet, "B", "B", 14)
	f.SetColWidth(sheet, "C", "C", 18)
	f.SetColWidth(sheet, "D", "D", 18)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 18)
	f.SetColWidth(sheet, "G", "G", 14)
	f.SetColWidth(sheet, "H", "H", 50)

	// Summary sheet
	f.NewSheet("Resumen")
	f.SetCellValue("Resumen", "A1", "Simulacion: "+result.ScenarioName)
	f.SetCellValue("Resumen", "A3", "Reserva Maxima")
	f.SetCellValue("Resumen", "B3", result.MaxReserve.InexactFloat64())
	f.SetCellValue("Resumen", "A4", "Reserva Minima")
	f.SetCellValue("Resumen", "B4", result.MinReserve.InexactFloat64())
	f.SetCellValue("Resumen", "A5", "Reserva Final")
	f.SetCellValue("Resumen", "B5", result.FinalReserve.InexactFloat64())
	f.SetCellValue("Resumen", "A6", "Total Eventos")
	f.SetCellValue("Resumen", "B6", result.EventsTotal)

	return f.SaveAs(outputPath)
}

// ExportComparative writes multiple simulation results side by side for comparison.
func ExportComparative(results []*SimulationResult, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Comparativo"
	f.SetSheetName(f.GetSheetName(0), sheet)

	// Header row: Year | scenario1 | scenario2 | ...
	f.SetCellValue(sheet, "A1", "Ano")
	for i, r := range results {
		cell, _ := excelize.CoordinatesToCellName(i+2, 1)
		f.SetCellValue(sheet, cell, r.ScenarioName)
	}
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
	})
	lastCol, _ := excelize.CoordinatesToCellName(len(results)+1, 1)
	f.SetCellStyle(sheet, "A1", lastCol, style)

	// Find max horizon
	maxYear := 0
	for _, r := range results {
		if len(r.Steps) > 0 {
			last := r.Steps[len(r.Steps)-1].Year
			if last > maxYear {
				maxYear = last
			}
		}
	}

	for y := 0; y <= maxYear; y++ {
		row := y + 2
		f.SetCellValue(sheet, cellName(1, row), y)
		for i, r := range results {
			cell, _ := excelize.CoordinatesToCellName(i+2, row)
			val := 0.0
			for _, step := range r.Steps {
				if step.Year == y {
					val = step.ReserveValue.InexactFloat64()
					break
				}
			}
			f.SetCellValue(sheet, cell, val)
		}
	}

	f.SetColWidth(sheet, "A", "A", 8)
	for i := range results {
		col, _ := excelize.CoordinatesToCellName(i+2, 1)
		f.SetColWidth(sheet, col, col, 22)
	}

	// Summary sheet
	f.NewSheet("Resumen")
	f.SetCellValue("Resumen", "A1", "Escenario")
	f.SetCellValue("Resumen", "B1", "Reserva Max")
	f.SetCellValue("Resumen", "C1", "Reserva Min")
	f.SetCellValue("Resumen", "D1", "Reserva Final")
	f.SetCellValue("Resumen", "E1", "Eventos")
	f.SetCellStyle("Resumen", "A1", "E1", style)

	for i, r := range results {
		row := i + 2
		f.SetCellValue("Resumen", cellName(1, row), r.ScenarioName)
		f.SetCellValue("Resumen", cellName(2, row), r.MaxReserve.InexactFloat64())
		f.SetCellValue("Resumen", cellName(3, row), r.MinReserve.InexactFloat64())
		f.SetCellValue("Resumen", cellName(4, row), r.FinalReserve.InexactFloat64())
		f.SetCellValue("Resumen", cellName(5, row), r.EventsTotal)
	}

	return f.SaveAs(outputPath)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
