// Package portfolio generates synthetic insurance portfolios with realistic
// family compositions for stress-testing the reserve calculation engine.
package portfolio

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/models"
)

// FamilyArchetype defines a template for generating family groups.
// Each archetype has weights that control how frequently it appears.
type FamilyArchetype struct {
	Name        string
	Weight      int // relative probability
	Description string
	Generate    func(rnd *rand.Rand) (policyDef, []memberDef)
}

type policyDef struct {
	NumeroPoliza    string
	TipoRenta       string
	SexoCausante    string
	EdadCausante    int
	TipoPension     string
	ModalidadRenta  string
	VigenciaPension string
	CapitalUF       float64
	TasaTM          float64
	TasaTC          float64
	PeriodoAumento  int
	PctAumento      float64
	FechaInicio     time.Time
}

type memberDef struct {
	Rol             models.BeneficiarioRol
	Sexo            string
	Edad            int
	TipoC1194       string
	Condicion       string
	MatrimonioAnios int
	HijosComunes    int
	FinDerechoEdad  int
	PctRenta        float64
}

var archetypes = []FamilyArchetype{
	{"hombre_solo", 15, "Hombre solo pensionado, sin beneficiarios", genHombreSolo},
	{"mujer_sola", 12, "Mujer sola pensionada, sin beneficiarios", genMujerSola},
	{"pareja_simple", 25, "Pareja tradicional sin hijos con derecho", genParejaSimple},
	{"pareja_con_hijos", 20, "Pareja con 1-2 hijos menores/estudiantes", genParejaConHijos},
	{"viudo_sobrevivencia", 8, "Viudo/a recibiendo pension de sobrevivencia", genViudoSobrevivencia},
	{"viejo_caliente", 5, "Pensionado mayor con conyuge joven, alto riesgo", genViejoCaliente},
	{"invalido", 8, "Pensionado por invalidez con familia", genInvalido},
	{"familia_extensa", 4, "Familia con conyuge + 3+ hijos + madre no matrimonial", genFamiliaExtensa},
	{"conviviente_civil", 3, "Pareja con acuerdo de union civil", genConvivienteCivil},
	{"stock_pre2005", 5, "Poliza historica pre-2005 (estrato RV-85/B-85)", genStockPre2005},
}

var rnd = rand.New(rand.NewSource(time.Now().UnixNano()))

// PolicyResult holds a generated policy ready for calculation.
type PolicyResult struct {
	Policy  models.Policy
	Members []models.Beneficiario
	Grupo   *models.GrupoFamiliar
}

// Generate creates n synthetic policies distributed across archetypes.
func Generate(n int) []PolicyResult {
	results := make([]PolicyResult, 0, n)
	totalWeight := 0
	for _, a := range archetypes {
		totalWeight += a.Weight
	}

	for i := 0; i < n; i++ {
		pick := rnd.Intn(totalWeight)
		cumulative := 0
		var archetype *FamilyArchetype
		for j := range archetypes {
			cumulative += archetypes[j].Weight
			if pick < cumulative {
				archetype = &archetypes[j]
				break
			}
		}
		if archetype == nil {
			archetype = &archetypes[0]
		}

		pdef, mdefs := archetype.Generate(rnd)
		policy := buildPolicy(i, pdef)
		members := buildMembers(policy.ID, mdefs, policy.FechaInicio)
		grupo := buildGrupo(members)

		results = append(results, PolicyResult{
			Policy:  policy,
			Members: members,
			Grupo:   grupo,
		})
	}

	return results
}

func buildPolicy(idx int, pdef policyDef) models.Policy {
	tasaTM := decimal.NewFromFloat(pdef.TasaTM)
	tasaTC := decimal.NewFromFloat(pdef.TasaTC)

	return models.Policy{
		ID:                idx + 1,
		NumeroPoliza:      fmt.Sprintf("%s-%06d", pdef.NumeroPoliza, idx+1),
		TipoRenta:         pdef.TipoRenta,
		SexoBeneficiario:  pdef.SexoCausante,
		EdadContratante:   pdef.EdadCausante,
		FechaInicio:       pdef.FechaInicio,
		CapitalAsegurado:  decimal.NewFromFloat(pdef.CapitalUF),
		FormaPago:         "MENSUAL",
		TasaTM:            tasaTM,
		TasaTC:            tasaTC,
		Estado:            "ACTIVA",
		TipoPension:       pdef.TipoPension,
		ModalidadRenta:    pdef.ModalidadRenta,
		VigenciaPension:   pdef.VigenciaPension,
		PeriodoAumento:    pdef.PeriodoAumento,
		PorcentajeAumento: decimal.NewFromFloat(pdef.PctAumento),
	}
}

func buildMembers(polizaID int, mdefs []memberDef, fechaContratacion time.Time) []models.Beneficiario {
	members := make([]models.Beneficiario, 0, len(mdefs))

	for _, md := range mdefs {
		b := models.Beneficiario{
			PolizaID:              polizaID,
			Rol:                   md.Rol,
			Sexo:                  md.Sexo,
			EdadContratacion:      md.Edad,
			Estado:                "ACTIVO",
			TipoBeneficiarioC1194: md.TipoC1194,
			DerechoPension:        models.DerechoPensionSi,
			DerechoAcrecer:        "N",
			SituacionInvalidez:    models.InvNo,
			Condicion:             md.Condicion,
			MatrimonioAnios:       md.MatrimonioAnios,
			HijosComunes:          md.HijosComunes,
		}
		if md.FinDerechoEdad > 0 {
			fin := md.FinDerechoEdad
			b.FinDerechoEdad = &fin
		}
		if md.PctRenta > 0 {
			b.PorcentajeRenta = decimal.NewFromFloat(md.PctRenta)
		} else {
			b.PorcentajeRenta = models.CalcularPorcentajeSobrevivencia(md.TipoC1194, false)
		}

		tipoTabla := ""
		if md.Rol == models.RolCausante {
			tipoTabla = string(models.TableTypeVejez)
		}
		b.TablaAsignada = models.SelectTableForBeneficiario(
			b.Rol, b.Sexo, b.TipoBeneficiarioC1194, fechaContratacion, tipoTabla,
		)

		members = append(members, b)
	}

	return members
}

func buildGrupo(members []models.Beneficiario) *models.GrupoFamiliar {
	gf := &models.GrupoFamiliar{}
	for i := range members {
		if members[i].Rol == models.RolCausante {
			gf.Causante = &members[i]
		} else {
			cp := members[i]
			gf.Beneficiarios = append(gf.Beneficiarios, &cp)
		}
	}
	return gf
}

// === Archetype generators ===

func genHombreSolo(rnd *rand.Rand) (policyDef, []memberDef) {
	edad := 60 + rnd.Intn(25) // 60-84
	pdef := policyDef{
		NumeroPoliza:    "RV-H",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edad,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       2000 + rnd.Float64()*8000,
		TasaTM:          0.035 + rnd.Float64()*0.010,
		TasaTC:          0.032 + rnd.Float64()*0.010,
		FechaInicio:     randomDate(rnd, 2015, 2025),
	}
	if rnd.Float64() < 0.3 {
		pdef.TipoPension = models.TipoPensionRVVejezAnticipada
		pdef.EdadCausante = 55 + rnd.Intn(10)
	}
	applyClauses(rnd, &pdef)
	return pdef, []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edad, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
	}
}

func genMujerSola(rnd *rand.Rand) (policyDef, []memberDef) {
	edad := 55 + rnd.Intn(25)
	pdef := policyDef{
		NumeroPoliza:    "RV-M",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoFemenino,
		EdadCausante:    edad,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       2000 + rnd.Float64()*8000,
		TasaTM:          0.035 + rnd.Float64()*0.010,
		TasaTC:          0.032 + rnd.Float64()*0.010,
		FechaInicio:     randomDate(rnd, 2015, 2025),
	}
	applyClauses(rnd, &pdef)
	return pdef, []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoFemenino, Edad: edad, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
	}
}

func genParejaSimple(rnd *rand.Rand) (policyDef, []memberDef) {
	esHombre := rnd.Float64() < 0.55
	edadCausante := 60 + rnd.Intn(20)
	edadConyuge := edadCausante - 5 + rnd.Intn(10)
	if edadConyuge < 40 {
		edadConyuge = 40
	}

	sexoC := models.SexoMasculino
	sexoB := models.SexoFemenino
	if !esHombre {
		sexoC = models.SexoFemenino
		sexoB = models.SexoMasculino
		edadCausante = 55 + rnd.Intn(20)
		edadConyuge = edadCausante - 5 + rnd.Intn(10)
	}

	pdef := policyDef{
		NumeroPoliza:    "RV-PC",
		TipoRenta:       "VITALICIA",
		SexoCausante:    sexoC,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       3000 + rnd.Float64()*10000,
		TasaTM:          0.035 + rnd.Float64()*0.008,
		TasaTC:          0.032 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2015, 2025),
	}
	applyClauses(rnd, &pdef)

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: sexoC, Edad: edadCausante, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
		{Rol: models.RolConyuge, Sexo: sexoB, Edad: edadConyuge, TipoC1194: models.C1194ConyugeSinHijos,
			MatrimonioAnios: 15 + rnd.Intn(30), HijosComunes: 0, PctRenta: 0.60},
	}
	return pdef, members
}

func genParejaConHijos(rnd *rand.Rand) (policyDef, []memberDef) {
	esHombre := rnd.Float64() < 0.55
	edadCausante := 55 + rnd.Intn(15)
	edadConyuge := edadCausante - 5 + rnd.Intn(8)
	nHijos := 1 + rnd.Intn(2)

	sexoC := models.SexoMasculino
	sexoB := models.SexoFemenino
	if !esHombre {
		sexoC = models.SexoFemenino
		sexoB = models.SexoMasculino
	}

	pdef := policyDef{
		NumeroPoliza:    "RV-PH",
		TipoRenta:       "VITALICIA",
		SexoCausante:    sexoC,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       3000 + rnd.Float64()*12000,
		TasaTM:          0.035 + rnd.Float64()*0.008,
		TasaTC:          0.032 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2015, 2025),
	}
	applyClauses(rnd, &pdef)

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: sexoC, Edad: edadCausante, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
		{Rol: models.RolConyuge, Sexo: sexoB, Edad: edadConyuge, TipoC1194: models.C1194ConyugeConHijos,
			MatrimonioAnios: 15 + rnd.Intn(20), HijosComunes: nHijos, PctRenta: 0.50},
	}

	for h := 0; h < nHijos; h++ {
		edadHijo := 5 + rnd.Intn(18)
		sexoHijo := models.SexoMasculino
		if rnd.Float64() < 0.5 {
			sexoHijo = models.SexoFemenino
		}
		cond := "MENOR"
		tipo := models.C1194HijoSinIncremento
		if edadHijo >= 18 {
			cond = "ESTUDIANTE"
		}
		members = append(members, memberDef{
			Rol: models.RolHijo, Sexo: sexoHijo, Edad: edadHijo,
			TipoC1194: tipo, Condicion: cond, FinDerechoEdad: 24, PctRenta: 0.15,
		})
	}

	return pdef, members
}

func genViudoSobrevivencia(rnd *rand.Rand) (policyDef, []memberDef) {
	edad := 55 + rnd.Intn(25)
	esHombre := rnd.Float64() < 0.3

	sexo := models.SexoFemenino
	if esHombre {
		sexo = models.SexoMasculino
	}

	pdef := policyDef{
		NumeroPoliza:    "RV-SV",
		TipoRenta:       "VITALICIA",
		SexoCausante:    sexo,
		EdadCausante:    edad,
		TipoPension:     models.TipoPensionRVSobrevivencia,
		ModalidadRenta:  "1000",
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       1500 + rnd.Float64()*5000,
		TasaTM:          0.036 + rnd.Float64()*0.008,
		TasaTC:          0.033 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2012, 2024),
	}

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: sexo, Edad: edad, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
	}
	return pdef, members
}

func genViejoCaliente(rnd *rand.Rand) (policyDef, []memberDef) {
	edadCausante := 70 + rnd.Intn(15)
	edadConyuge := 35 + rnd.Intn(15)
	nHijos := rnd.Intn(2)

	pdef := policyDef{
		NumeroPoliza:    "RV-VC",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       4000 + rnd.Float64()*10000,
		TasaTM:          0.038 + rnd.Float64()*0.008,
		TasaTC:          0.035 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2018, 2025),
	}
	if rnd.Float64() < 0.4 {
		pdef.ModalidadRenta = "3120" // con periodo garantizado 120 meses
	}
	members := []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edadCausante, TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
		{Rol: models.RolConyuge, Sexo: models.SexoFemenino, Edad: edadConyuge,
			TipoC1194: models.C1194ConyugeSinHijos, MatrimonioAnios: 1 + rnd.Intn(3),
			HijosComunes: nHijos, PctRenta: 0.60},
	}
	if nHijos > 0 {
		members = append(members, memberDef{
			Rol: models.RolHijo, Sexo: models.SexoFemenino, Edad: rnd.Intn(5),
			TipoC1194: models.C1194HijoSinIncremento, Condicion: "MENOR",
			FinDerechoEdad: 24, PctRenta: 0.15,
		})
	}
	return pdef, members
}

func genInvalido(rnd *rand.Rand) (policyDef, []memberDef) {
	edadCausante := 35 + rnd.Intn(25)
	esTotal := rnd.Float64() < 0.6

	tipoPension := models.TipoPensionRVInvParcial
	edadConyuge := edadCausante - 3 + rnd.Intn(6)

	if esTotal {
		tipoPension = models.TipoPensionRVInvTotal
	}

	pdef := policyDef{
		NumeroPoliza:    "RV-INV",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edadCausante,
		TipoPension:     tipoPension,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       2000 + rnd.Float64()*6000,
		TasaTM:          0.038 + rnd.Float64()*0.010,
		TasaTC:          0.035 + rnd.Float64()*0.010,
		FechaInicio:     randomDate(rnd, 2013, 2025),
	}
	applyClauses(rnd, &pdef)

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edadCausante,
			TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
	}
	if rnd.Float64() < 0.7 {
		members = append(members, memberDef{
			Rol: models.RolConyuge, Sexo: models.SexoFemenino, Edad: edadConyuge,
			TipoC1194: models.C1194ConyugeSinHijos, MatrimonioAnios: 5 + rnd.Intn(15),
			PctRenta: 0.60,
		})
	}
	if rnd.Float64() < 0.5 {
		edadHijo := 5 + rnd.Intn(15)
		members = append(members, memberDef{
			Rol: models.RolHijo, Sexo: models.SexoMasculino, Edad: edadHijo,
			TipoC1194: models.C1194HijoSinIncremento, Condicion: "MENOR",
			FinDerechoEdad: 24, PctRenta: 0.15,
		})
	}
	return pdef, members
}

func genFamiliaExtensa(rnd *rand.Rand) (policyDef, []memberDef) {
	edadCausante := 50 + rnd.Intn(15)
	edadConyuge := edadCausante - 5 + rnd.Intn(5)
	nHijos := 2 + rnd.Intn(2)

	pdef := policyDef{
		NumeroPoliza:    "RV-FE",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       5000 + rnd.Float64()*15000,
		TasaTM:          0.036 + rnd.Float64()*0.008,
		TasaTC:          0.033 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2015, 2025),
	}
	applyClauses(rnd, &pdef)

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edadCausante,
			TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
		{Rol: models.RolConyuge, Sexo: models.SexoFemenino, Edad: edadConyuge,
			TipoC1194: models.C1194ConyugeConHijos, MatrimonioAnios: 20 + rnd.Intn(15),
			HijosComunes: nHijos, PctRenta: 0.50},
	}
	for h := 0; h < nHijos; h++ {
		edadH := 5 + rnd.Intn(20)
		s := models.SexoMasculino
		if rnd.Float64() < 0.5 {
			s = models.SexoFemenino
		}
		cond := "MENOR"
		if edadH >= 18 {
			cond = "ESTUDIANTE"
		}
		members = append(members, memberDef{
			Rol: models.RolHijo, Sexo: s, Edad: edadH,
			TipoC1194: models.C1194HijoSinIncremento, Condicion: cond,
			FinDerechoEdad: 24, PctRenta: 0.15,
		})
	}
	return pdef, members
}

func genConvivienteCivil(rnd *rand.Rand) (policyDef, []memberDef) {
	edadCausante := 55 + rnd.Intn(15)
	edadCC := edadCausante - 3 + rnd.Intn(8)

	pdef := policyDef{
		NumeroPoliza:    "RV-CC",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  "1000",
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       3000 + rnd.Float64()*7000,
		TasaTM:          0.037 + rnd.Float64()*0.008,
		TasaTC:          0.034 + rnd.Float64()*0.008,
		FechaInicio:     randomDate(rnd, 2018, 2025),
	}

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edadCausante,
			TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
		{Rol: models.RolConviviente, Sexo: models.SexoFemenino, Edad: edadCC,
			TipoC1194: models.C1194CCsinHijos, MatrimonioAnios: 3 + rnd.Intn(5),
			PctRenta: 0.60},
	}
	return pdef, members
}

// genStockPre2005 generates a historical policy contracted before the 2005
// table change (RV-85/B-85 stratum). This exercises the legacy stratification
// in the batch engine, which otherwise only produces 2012+ synthetic policies.
func genStockPre2005(rnd *rand.Rand) (policyDef, []memberDef) {
	edadCausante := 60 + rnd.Intn(10) // 60-69 at contract
	edadConyuge := edadCausante - 5 + rnd.Intn(8)
	if edadConyuge < 40 {
		edadConyuge = 40
	}

	// Contract between 1995 and 2004 so the policy falls in the pre-2005
	// stratum (base RV-1985 / B-1985).
	fechaInicio := randomDate(rnd, 1995, 2004)

	pdef := policyDef{
		NumeroPoliza:    "RV-85",
		TipoRenta:       "VITALICIA",
		SexoCausante:    models.SexoMasculino,
		EdadCausante:    edadCausante,
		TipoPension:     models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:  pickModalidad(rnd),
		VigenciaPension: models.VigenciaEnPago,
		CapitalUF:       2000 + rnd.Float64()*6000,
		TasaTM:          0.036 + rnd.Float64()*0.008,
		TasaTC:          0.033 + rnd.Float64()*0.008,
		FechaInicio:     fechaInicio,
	}

	members := []memberDef{
		{Rol: models.RolCausante, Sexo: models.SexoMasculino, Edad: edadCausante,
			TipoC1194: models.C1194Afiliado, PctRenta: 1.0},
	}
	if rnd.Float64() < 0.7 {
		members = append(members, memberDef{
			Rol: models.RolConyuge, Sexo: models.SexoFemenino, Edad: edadConyuge,
			TipoC1194: models.C1194ConyugeSinHijos, MatrimonioAnios: 15 + rnd.Intn(25),
			HijosComunes: 0, PctRenta: 0.60,
		})
	}
	return pdef, members
}

// === Helpers ===

func pickModalidad(rnd *rand.Rand) string {
	r := rnd.Float64()
	switch {
	case r < 0.50:
		return "1000"
	case r < 0.70:
		return "3120"
	case r < 0.85:
		return "2000"
	default:
		return "4120"
	}
}

func applyClauses(rnd *rand.Rand, pdef *policyDef) {
	if pdef.ModalidadRenta == "2000" {
		pdef.PeriodoAumento = 12 + rnd.Intn(4)*12
		pdef.PctAumento = 10 + rnd.Float64()*15
	}
}

func randomDate(rnd *rand.Rand, yearStart, yearEnd int) time.Time {
	year := yearStart + rnd.Intn(yearEnd-yearStart+1)
	month := 1 + rnd.Intn(12)
	day := 1 + rnd.Intn(28)
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// ArchetypeSummary returns names and weights for reporting.
func ArchetypeSummary() []string {
	var result []string
	totalWeight := 0
	for _, a := range archetypes {
		totalWeight += a.Weight
	}
	for _, a := range archetypes {
		pct := float64(a.Weight) * 100 / float64(totalWeight)
		result = append(result, fmt.Sprintf("  %s (%.0f%%): %s", a.Name, pct, a.Description))
	}
	return result
}
