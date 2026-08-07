package calculator

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/database"
	"github.com/fcaStockhausen/reservin/internal/models"
)

func TestPeriodoGarantizado(t *testing.T) {
	db, err := database.NewConnection(database.Config{Path: "/Users/fcaraneda/projects/utility_projects/reservas/data/reservas.db"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo := database.NewMortalityRepository(db.DB)
	me := NewMortalityEngine()
	if err := me.EnsureLoaded(repo, "CB-H-2020"); err != nil {
		t.Fatal(err)
	}
	fp := NewFlowProjector(me)

	contrato := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	c := &models.Beneficiario{
		Rol: models.RolCausante, Sexo: "M",
		EdadContratacion: 60, TablaAsignada: "CB-H-2020",
		PorcentajeRenta: decimal.NewFromInt(1),
	}
	grupo := &models.GrupoFamiliar{Causante: c}

	pol := models.Policy{
		TipoRenta:   string(models.PolicyTypeVitalicia),
		FechaInicio: contrato,
		TasaTM:      decimal.NewFromFloat(0.04), TasaTC: decimal.NewFromFloat(0.04),
	}

	// Without PG
	pol.ModalidadRenta = "1000"
	noPG, err := fp.Project(pol, grupo, decimal.NewFromInt(100), decimal.NewFromFloat(0.04), 0)
	if err != nil {
		t.Fatal(err)
	}

	// With PG 240 months (20 years)
	pol.ModalidadRenta = "3240"
	withPG, err := fp.Project(pol, grupo, decimal.NewFromInt(100), decimal.NewFromFloat(0.04), 0)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("sin PG: %s, con PG 240m: %s", noPG.TotalReserve.StringFixed(2), withPG.TotalReserve.StringFixed(2))
	if !withPG.TotalReserve.GreaterThan(noPG.TotalReserve) {
		t.Fatalf("esperaba reserva mayor con PG, got sin=%s con=%s", noPG.TotalReserve.String(), withPG.TotalReserve.String())
	}

	// PG of 0 months (modalidad 1000) should equal no PG
	if got := models.GarantizedMonths("1000"); got != 0 {
		t.Fatalf("GarantizedMonths(1000) = %d, want 0", got)
	}
	if got := models.GarantizedMonths("3180"); got != 180 {
		t.Fatalf("GarantizedMonths(3180) = %d, want 180", got)
	}
}
