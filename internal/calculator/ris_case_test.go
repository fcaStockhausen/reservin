package calculator

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/database"
	"github.com/fcaStockhausen/reservin/internal/models"
)

// SVS=730135: causante F nac 1950-12-10, contract 2005-03-01, renta 9.43/mes
// RT-FINANCIERA-2020 reportado: 1444.57 UF (causante + hijo tb=35)
func TestRISCase730135(t *testing.T) {
	db, err := database.NewConnection(database.Config{Path: "/Users/fcaraneda/projects/utility_projects/reservas/data/reservas.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewMortalityRepository(db.DB)
	calc := NewReserveCalculator(repo)

	contrato := time.Date(2005, 3, 1, 0, 0, 0, 0, time.UTC)
	pol := models.Policy{
		TipoRenta:      string(models.PolicyTypeVitalicia),
		FechaInicio:    contrato,
		ModalidadRenta: "3180",
		TasaTM:         decimal.NewFromFloat(0.0298),
		TasaTC:         decimal.NewFromFloat(0.0396),
	}
	nac := time.Date(1950, 12, 10, 0, 0, 0, 0, time.UTC)
	edad := contrato.Year() - nac.Year()
	c := &models.Beneficiario{
		Rol: models.RolCausante, Sexo: "F", EdadContratacion: edad,
		TablaAsignada: "RV-M-2020", PorcentajeRenta: decimal.NewFromInt(1),
	}
	grupo := &models.GrupoFamiliar{Causante: c}
	rentaAnual := decimal.NewFromFloat(9.43).Mul(decimal.NewFromInt(12))
	currentYear := 2025 - contrato.Year()
	res, err := calc.CalculateAt(pol, grupo, rentaAnual, currentYear)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("causante solo: %.2f (reportado RTF20 causante: 1444.57)", res.TotalReserve.InexactFloat64())
	t.Logf("factor: %.2f", res.TotalReserve.InexactFloat64()/rentaAnual.InexactFloat64())
}
