package calculator

import (
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

// ExportFlowsToExcel writes a FlowResult to an Excel file with one row per
// (member, period), fully disaggregated for actuarial analysis.
//
// Sheet 1 "Flujos": every flow with all its components.
// Sheet 2 "Resumen": summary by member and total reserve.
func ExportFlowsToExcel(result *FlowResult, outputPath string) error {
	f := excelize.NewFile()
	defer f.Close()

	// --- Sheet 1: Flujos ---
	sheet1 := "Flujos"
	f.SetSheetName(f.GetSheetName(0), sheet1)

	headers := []string{
		"Periodo", "Fecha", "Rol", "Sexo", "Tabla",
		"Edad", "Renta Base", "% Renta",
		"Prob Supervivencia", "Prob Causante Vivo", "Prob Flujo",
		"Monto Esperado", "Factor Descuento", "Valor Presente",
	}
	for c, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		f.SetCellValue(sheet1, cell, h)
	}

	// Style header row
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"#D9E1F2"}},
	})
	f.SetCellStyle(sheet1, "A1", "N1", style)

	zero := decimal.NewFromInt(0)

	for i, flow := range result.Flows {
		row := i + 2
		setDec := func(col int, val decimal.Decimal) {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			if val.Equal(zero) {
				f.SetCellValue(sheet1, cell, 0)
			} else {
				f.SetCellValue(sheet1, cell, val.InexactFloat64())
			}
		}

		f.SetCellValue(sheet1, cellAt(1, row), flow.Period)
		f.SetCellValue(sheet1, cellAt(2, row), flow.Date.Format("2006-01-02"))
		f.SetCellValue(sheet1, cellAt(3, row), flow.MemberRol)
		f.SetCellValue(sheet1, cellAt(4, row), flow.MemberSex)
		f.SetCellValue(sheet1, cellAt(5, row), flow.MemberTable)
		f.SetCellValue(sheet1, cellAt(6, row), flow.MemberAgeAtT)
		setDec(7, flow.RentaBase)
		setDec(8, flow.PctRenta)
		setDec(9, flow.SurvivalProb)
		setDec(10, flow.CausanteAlive)
		setDec(11, flow.FlowProb)
		setDec(12, flow.FlowAmount)
		setDec(13, flow.DiscountFactor)
		setDec(14, flow.PresentValue)
	}

	// Column widths
	widths := map[string]float64{"A": 8, "B": 12, "C": 12, "D": 6, "E": 14,
		"F": 6, "G": 16, "H": 8, "I": 18, "J": 18, "K": 12, "L": 16, "M": 16, "N": 16}
	for col, w := range widths {
		f.SetColWidth(sheet1, col, col, w)
	}

	// --- Sheet 2: Resumen ---
	sheet2 := "Resumen"
	f.NewSheet(sheet2)

	f.SetCellValue(sheet2, "A1", "Resumen Reserva")
	f.SetCellValue(sheet2, "A3", "Poliza")
	f.SetCellValue(sheet2, "B3", result.PolicyNumber)
	f.SetCellValue(sheet2, "A4", "Tasa Descuento")
	f.SetCellValue(sheet2, "B4", fmt.Sprintf("%.4f%%", result.DiscountRate.InexactFloat64()*100))
	f.SetCellValue(sheet2, "A5", "Periodos Proyectados")
	f.SetCellValue(sheet2, "B5", result.Periods)

	// Summary by member role
	f.SetCellValue(sheet2, "A7", "Rol")
	f.SetCellValue(sheet2, "B7", "Sexo")
	f.SetCellValue(sheet2, "C7", "Tabla")
	f.SetCellValue(sheet2, "D7", "Flujos")
	f.SetCellValue(sheet2, "E7", "VP Total")
	f.SetCellStyle(sheet2, "A7", "E7", style)

	type memberSummary struct {
		sex   string
		table string
		count int
		total decimal.Decimal
	}
	byMember := make(map[string]*memberSummary)
	for _, flow := range result.Flows {
		m, ok := byMember[flow.MemberRol]
		if !ok {
			m = &memberSummary{sex: flow.MemberSex, table: flow.MemberTable}
			byMember[flow.MemberRol] = m
		}
		m.count++
		m.total = m.total.Add(flow.PresentValue)
	}

	row := 8
	for rol, m := range byMember {
		f.SetCellValue(sheet2, cellAt(1, row), rol)
		f.SetCellValue(sheet2, cellAt(2, row), m.sex)
		f.SetCellValue(sheet2, cellAt(3, row), m.table)
		f.SetCellValue(sheet2, cellAt(4, row), m.count)
		f.SetCellValue(sheet2, cellAt(5, row), m.total.InexactFloat64())
		row++
	}

	// Total reserve
	f.SetCellValue(sheet2, cellAt(1, row+1), "RESERVA TOTAL")
	f.SetCellValue(sheet2, cellAt(5, row+1), result.TotalReserve.InexactFloat64())
	boldStyle, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	f.SetCellStyle(sheet2, cellAt(1, row+1), cellAt(5, row+1), boldStyle)

	f.SetColWidth(sheet2, "A", "A", 20)
	f.SetColWidth(sheet2, "B", "B", 10)
	f.SetColWidth(sheet2, "C", "C", 14)
	f.SetColWidth(sheet2, "D", "D", 10)
	f.SetColWidth(sheet2, "E", "E", 20)

	return f.SaveAs(outputPath)
}

func cellAt(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
