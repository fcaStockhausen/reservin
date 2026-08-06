package scenario

import (
	"fmt"
	"sort"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/calculator"
	"reservas/internal/database"
	"reservas/internal/models"
)

// Simulator evolves a scenario through time, applying events and recalculating
// the reserve at each time step. The output is a time series of reserve values
// and family group states.
type Simulator struct {
	mortRepo      *database.MortalityRepository
	contractDate  time.Time
	valuationDate time.Time
}

// NewSimulator creates a simulator wired to the database.
func NewSimulator(mortRepo *database.MortalityRepository) *Simulator {
	return &Simulator{mortRepo: mortRepo}
}

// StepResult captures the state at a single year of the simulation.
type StepResult struct {
	Year               int                        `json:"year"`
	Date               time.Time                  `json:"date"`
	ReserveValue       decimal.Decimal            `json:"reserve_value"`  // reserva total (base + descalce reconocido)
	ReservaBase        decimal.Decimal            `json:"reserva_base"`   // reserva con tabla de estrato anclada
	DescalceBruto      decimal.Decimal            `json:"descalce_bruto"` // reserva contemporánea - reserva base
	DescalceReconocido decimal.Decimal            `json:"descalce_reconocido"`
	CausanteAge        int                        `json:"causante_age"`
	Events             []string                   `json:"events"`
	MembersAlive       int                        `json:"members_alive"`
	Breakdown          map[string]decimal.Decimal `json:"breakdown"`
}

// SimulationResult holds the complete output of running a scenario.
type SimulationResult struct {
	ScenarioName string          `json:"scenario_name"`
	Steps        []StepResult    `json:"steps"`
	FinalReserve decimal.Decimal `json:"final_reserve"`
	MaxReserve   decimal.Decimal `json:"max_reserve"`
	MinReserve   decimal.Decimal `json:"min_reserve"`
	EventsTotal  int             `json:"events_total"`
	Tables       []MemberTable   `json:"tables"`
}

// MemberTable reports the mortality table assigned to a member at inception.
type MemberTable struct {
	Rol   string `json:"rol"`
	Sexo  string `json:"sexo"`
	Edad  int    `json:"edad"`
	Tabla string `json:"tabla"`
}

// Run executes the scenario and returns the evolution of the reserve over time.
func (sim *Simulator) Run(s *Scenario) (*SimulationResult, error) {
	startDate := policyStartDate()
	contractDate := contractDate(s.Policy.FechaContrato, startDate)
	valuationDate := startDate // descalce reference = tables in force at valuation
	sim.contractDate = contractDate
	sim.valuationDate = valuationDate
	methodology := models.MethodologyIFRS

	// Initialize family group from scenario definition.
	members := sim.buildMembers(s, startDate, contractDate, valuationDate)
	rentaAnual := decimal.NewFromFloat(s.Policy.CapitalUF / computeAnnuityFactor(sim.mortRepo, members, methodology, s))
	if rentaAnual.LessThanOrEqual(decimal.Zero) {
		rentaAnual = decimal.NewFromFloat(1000) // fallback
	}

	discountRate := decimal.NewFromFloat(s.Policy.TasaTC)
	if s.Policy.TasaTM > 0 && s.Policy.TasaTM < s.Policy.TasaTC {
		discountRate = decimal.NewFromFloat(s.Policy.TasaTM)
	}

	eventsByYear := groupEventsByYear(s.Events)

	var steps []StepResult
	maxReserve := decimal.Zero
	minReserve := decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(15)))

	for t := 0; t <= s.Horizon; t++ {
		currentDate := startDate.AddDate(t, 0, 0)
		var eventMsgs []string

		// Apply events at this year.
		for _, ev := range eventsByYear[t] {
			msg := sim.applyEvent(ev, &members, t)
			eventMsgs = append(eventMsgs, msg)
		}

		// Update ages and check for natural status changes (children aging out).
		sim.applyAging(members, t, &eventMsgs)

		// Build GrupoFamiliar for this time step.
		grupo := sim.buildGrupo(members)

		// Recalculate reserve with current family group (tabla de estrato).
		reservaBase, breakdown, err := sim.computeReserve(members, grupo, methodology, rentaAnual, discountRate, t)
		if err != nil {
			eventMsgs = append(eventMsgs, fmt.Sprintf("CALC_ERROR: %v", err))
			reservaBase = decimal.Zero
		}

		// Descalce: difference against the contemporaneous table, with gradual
		// recognition per the scenario curve.
		descalceBruto := sim.computeDescalce(members, grupo, methodology, rentaAnual, discountRate, t, reservaBase)
		descalceReconocido := descalceBruto.Mul(gradualRecognitionFactor(s.Policy, t))
		reserve := reservaBase.Add(descalceReconocido)

		// Count alive members.
		alive := 0
		for i := range members {
			if members[i].Beneficiario.Estado == "ACTIVO" {
				alive++
			}
		}

		causanteAge := 0
		for i := range members {
			if members[i].Beneficiario.Rol == models.RolCausante {
				causanteAge = members[i].Beneficiario.EdadContratacion + t
			}
		}

		steps = append(steps, StepResult{
			Year:               t,
			Date:               currentDate,
			ReserveValue:       reserve,
			ReservaBase:        reservaBase,
			DescalceBruto:      descalceBruto,
			DescalceReconocido: descalceReconocido,
			CausanteAge:        causanteAge,
			Events:             eventMsgs,
			MembersAlive:       alive,
			Breakdown:          breakdown,
		})

		if reserve.GreaterThan(maxReserve) {
			maxReserve = reserve
		}
		if reserve.LessThan(minReserve) && reserve.GreaterThan(decimal.Zero) {
			minReserve = reserve
		}

		// Stop if causante is dead and no beneficiaries remain.
		if alive == 0 {
			break
		}
	}

	result := &SimulationResult{
		ScenarioName: s.Name,
		Steps:        steps,
		MaxReserve:   maxReserve,
		MinReserve:   minReserve,
		Tables:       memberTables(members),
	}
	if len(steps) > 0 {
		result.FinalReserve = steps[len(steps)-1].ReserveValue
	}
	for _, evs := range eventsByYear {
		result.EventsTotal += len(evs)
	}

	return result, nil
}

// liveMember is the internal mutable representation during simulation.
type liveMember struct {
	Beneficiario models.Beneficiario
	DeathYear    int // 0 = alive
	// tablaBase is the anchored (stratum) table this member was born under; it
	// is what TablaAsignada holds between steps. The contemporaneous table is
	// only swapped in transiently while measuring the descalce.
	tablaBase string
}

// memberTableBasis computes the two tables that define a member's reserva: the
// anchored stratum table (bautizo) and the contemporaneous one (descalce).
func (sim *Simulator) memberTableBasis(rol models.BeneficiarioRol, sexo, tipoPension, condicion string, contractDate, valuationDate time.Time) (base, cont string) {
	invalido := false
	switch invalidezFrom2(tipoPension, condicion) {
	case models.InvTotal, models.InvParcial:
		invalido = true
	}
	tipoTabla := string(models.TableTypeVejez)
	if invalido {
		tipoTabla = string(models.TableTypeInvalidez)
	}
	base = models.SelectBaseTable(rol, tipoTabla, sexo, contractDate)
	cont = models.SelectContemporaneaTable(rol, tipoTabla, sexo, valuationDate)
	return base, cont
}

// invalidezFrom2 merges the causante pension type and a member's CONDICION flag
// into a single SITUACION-INVALIDEZ code (used by memberTableBasis).
func invalidezFrom2(tipoPension, condicion string) string {
	switch tipoPension {
	case models.TipoPensionRVInvTotal:
		return models.InvTotal
	case models.TipoPensionRVInvParcial:
		return models.InvParcial
	}
	switch condicion {
	case "INVALIDO", "INVALIDO_TOTAL":
		return models.InvTotal
	case "INVALIDO_PARCIAL":
		return models.InvParcial
	}
	return models.InvNo
}

func (sim *Simulator) buildMembers(s *Scenario, startDate, contractDate, valuationDate time.Time) []*liveMember {
	year := startDate.Year()
	var members []*liveMember

	// Causante
	c := models.Beneficiario{
		Rol:                   models.RolCausante,
		Sexo:                  s.Causante.Sexo,
		EdadContratacion:      buildBirth(s.Causante, 0, year),
		FechaNacimiento:       parseBirthDate(s.Causante.FechaNacimiento),
		PorcentajeRenta:       decimal.NewFromFloat(1.0),
		Estado:                "ACTIVO",
		TipoBeneficiarioC1194: defaultIfEmpty(s.Causante.TipoC1194, models.C1194Afiliado),
		DerechoPension:        models.DerechoPensionSi,
		DerechoAcrecer:        "N",
		SituacionInvalidez:    invalidezFrom(s.Causante, s.Policy.TipoPension),
	}
	base, cont := sim.memberTableBasis(c.Rol, c.Sexo, s.Policy.TipoPension, s.Causante.Condicion, contractDate, valuationDate)
	c.TablaAsignada = base
	members = append(members, &liveMember{Beneficiario: c, tablaBase: base})
	_ = cont

	// Rest of family
	for _, mdef := range s.Grupo {
		b := sim.buildBeneficiario(mdef, members, 0, year, contractDate, valuationDate)
		members = append(members, b)
	}

	return members
}

func (sim *Simulator) buildBeneficiario(mdef MemberDef, existing []*liveMember, addYear, startYear int, contractDate, valuationDate time.Time) *liveMember {
	rol := models.BeneficiarioRol(mdef.Rol)

	b := models.Beneficiario{
		Rol:                   rol,
		Sexo:                  mdef.Sexo,
		EdadContratacion:      buildBirth(mdef, addYear, startYear),
		FechaNacimiento:       parseBirthDate(mdef.FechaNacimiento),
		Estado:                "ACTIVO",
		TipoBeneficiarioC1194: mdef.TipoC1194,
		DerechoPension:        models.DerechoPensionSi,
		DerechoAcrecer:        "N",
		SituacionInvalidez:    invalidezFrom(mdef, ""),
		Condicion:             mdef.Condicion,
		MatrimonioAnios:       mdef.MatrimonioAnios,
		HijosComunes:          mdef.HijosComunes,
	}
	if mdef.FinDerechoEdad > 0 {
		fin := mdef.FinDerechoEdad
		b.FinDerechoEdad = &fin
	}
	if mdef.PctRenta > 0 {
		b.PorcentajeRenta = decimal.NewFromFloat(mdef.PctRenta)
	} else {
		b.PorcentajeRenta = sim.legalPct(b.TipoBeneficiarioC1194, existing)
	}

	base, _ := sim.memberTableBasis(b.Rol, b.Sexo, "", mdef.Condicion, contractDate, valuationDate)
	b.TablaAsignada = base
	return &liveMember{Beneficiario: b, tablaBase: base}
}

// buildBirth returns the member's age at policy inception (t=0). For members
// that enter the family after inception (addYear>0, e.g. a child born during the
// simulation) the age is offset so that age(t) = edad + t is correct at t=addYear.
// When a birth date is provided it takes precedence; `edad` is else an age at
// the event (or at t=0 for base members).
func buildBirth(mdef MemberDef, addYear, startYear int) int {
	if bd := parseBirthDate(mdef.FechaNacimiento); bd != nil {
		return startYear - bd.Year()
	}
	return mdef.Edad - addYear
}

func (sim *Simulator) legalPct(tipoC1194 string, existing []*liveMember) decimal.Decimal {
	hasHijos := false
	for _, m := range existing {
		if m.Beneficiario.Rol == models.RolHijo && m.Beneficiario.Estado == "ACTIVO" {
			hasHijos = true
			break
		}
	}
	return models.CalcularPorcentajeSobrevivencia(tipoC1194, hasHijos)
}

func (sim *Simulator) buildGrupo(members []*liveMember) *models.GrupoFamiliar {
	gf := &models.GrupoFamiliar{}
	for i := range members {
		if members[i].Beneficiario.Estado != "ACTIVO" {
			continue
		}
		if members[i].Beneficiario.Rol == models.RolCausante {
			gf.Causante = &members[i].Beneficiario
		} else {
			gf.Beneficiarios = append(gf.Beneficiarios, &members[i].Beneficiario)
		}
	}
	return gf
}

// applyEvent mutates the member list according to the event type.
func (sim *Simulator) applyEvent(ev EventDef, members *[]*liveMember, currentYear int) string {
	switch ev.Type {
	case EventAddMember:
		newB := sim.buildBeneficiario(ev.Member, *members, currentYear, policyStartDate().Year(), sim.contractDate, sim.valuationDate)
		*members = append(*members, newB)
		return fmt.Sprintf("t%d: +ADD %s (%s, edad %d)", currentYear, ev.Member.Rol, ev.Member.Sexo, ev.Member.Edad)

	case EventKillMember:
		for i := range *members {
			if (*members)[i].Beneficiario.Estado == "ACTIVO" &&
				(string((*members)[i].Beneficiario.Rol) == ev.TargetRol || ev.TargetRol == "") &&
				((*members)[i].Beneficiario.Sexo == ev.TargetSexo || ev.TargetSexo == "") {
				(*members)[i].Beneficiario.Estado = "FALLECIDO"
				return fmt.Sprintf("t%d: DEATH %s (%s)", currentYear, (*members)[i].Beneficiario.Rol, (*members)[i].Beneficiario.Sexo)
			}
		}
		return fmt.Sprintf("t%d: KILL no target found", currentYear)

	case EventRemoveMember:
		for i := range *members {
			if (*members)[i].Beneficiario.Estado == "ACTIVO" &&
				(string((*members)[i].Beneficiario.Rol) == ev.TargetRol || ev.TargetRol == "") &&
				((*members)[i].Beneficiario.Sexo == ev.TargetSexo || ev.TargetSexo == "") {
				(*members)[i].Beneficiario.Estado = "EXCLUIDO"
				(*members)[i].Beneficiario.DerechoPension = models.DerechoPensionNo
				return fmt.Sprintf("t%d: REMOVE %s (%s) - pierde derecho", currentYear, (*members)[i].Beneficiario.Rol, (*members)[i].Beneficiario.Sexo)
			}
		}
		return fmt.Sprintf("t%d: REMOVE no target found", currentYear)

	default:
		return fmt.Sprintf("t%d: %s", currentYear, ev.Type)
	}
}

// applyAging handles natural lifecycle changes: children reaching fin_derecho_edad.
func (sim *Simulator) applyAging(members []*liveMember, currentYear int, msgs *[]string) {
	for i := range members {
		m := &members[i].Beneficiario
		if members[i].Beneficiario.Estado != "ACTIVO" {
			continue
		}
		// Check fin_derecho_edad for hijos
		if m.Rol == models.RolHijo && m.FinDerechoEdad != nil {
			currentAge := m.EdadContratacion + currentYear
			if currentAge >= *m.FinDerechoEdad {
				members[i].Beneficiario.Estado = "EXCLUIDO"
				members[i].Beneficiario.DerechoPension = models.DerechoPensionNo
				*msgs = append(*msgs, fmt.Sprintf("t%d: HIJO cumple %d, pierde derecho", currentYear, *m.FinDerechoEdad))
			}
		}
	}

	// Recalculate conyuge % if hijo status changed (acrecimiento).
	hasHijos := false
	for i := range members {
		if members[i].Beneficiario.Rol == models.RolHijo && members[i].Beneficiario.Estado == "ACTIVO" {
			hasHijos = true
			break
		}
	}
	for i := range members {
		m := &members[i].Beneficiario
		if m.Estado == "ACTIVO" &&
			(m.Rol == models.RolConyuge || m.Rol == models.RolConviviente) {
			newPct := models.CalcularPorcentajeSobrevivencia(m.TipoBeneficiarioC1194, hasHijos)
			if !newPct.Equal(m.PorcentajeRenta) && !newPct.IsZero() {
				old := m.PorcentajeRenta
				m.PorcentajeRenta = newPct
				if currentYear > 0 {
					*msgs = append(*msgs, fmt.Sprintf("t%d: CONYUGE %s -> %s (acrecimiento)", currentYear,
						old.Mul(decimal.NewFromInt(100)).StringFixed(0)+"%",
						newPct.Mul(decimal.NewFromInt(100)).StringFixed(0)+"%"))
				}
			}
		}
	}
}

// computeReserve runs the VPPj calculation for the current group state.
func (sim *Simulator) computeReserve(
	members []*liveMember,
	grupo *models.GrupoFamiliar,
	methodology models.PolicyMethodology,
	rentaAnual decimal.Decimal,
	discountRate decimal.Decimal,
	currentYear int,
) (decimal.Decimal, map[string]decimal.Decimal, error) {
	me := calculator.NewMortalityEngine()
	tablesNeeded := uniqueTables(members)
	for _, t := range tablesNeeded {
		if err := me.EnsureLoaded(sim.mortRepo, t); err != nil {
			return decimal.Zero, nil, fmt.Errorf("load table %s: %w", t, err)
		}
	}

	projector := calculator.NewFlowProjector(me)

	policy := models.Policy{
		TasaTM: discountRate,
		TasaTC: discountRate,
	}
	policy.TasaDescuento = discountRate

	// If causante is alive, project from inception (VPPj).
	// If causante is dead, project survivor pensions directly.
	if grupo.Causante != nil {
		result, err := projector.Project(policy, grupo, rentaAnual, discountRate, currentYear)
		if err != nil {
			return decimal.Zero, nil, err
		}

		breakdown := make(map[string]decimal.Decimal)
		for _, f := range result.Flows {
			breakdown[f.MemberRol] = breakdown[f.MemberRol].Add(f.PresentValue)
		}
		return result.TotalReserve, breakdown, nil
	}

	// Causante dead: calculate survivor reserves for remaining beneficiaries.
	total := decimal.Zero
	breakdown := make(map[string]decimal.Decimal)

	// Survivors inherit the reserve from the current point in time: they have
	// already aged `currentYear` years since policy inception, so the annuity
	// must start from their actual physical age, not the contractual age.
	one := decimal.NewFromInt(1)
	for _, b := range grupo.Beneficiarios {
		benRenta := rentaAnual.Mul(b.PorcentajeRenta)
		startAge := b.EdadContratacion + currentYear

		// Simple annuity: sum of tpx * discount_factor for each future year.
		benTotal := decimal.Zero
		for t := 0; t <= 110-startAge; t++ {
			survProb, err := me.Tpx(b.TablaAsignada, startAge, t)
			if err != nil {
				break
			}
			if t == 0 {
				survProb = one
			} else if survProb.IsZero() {
				break
			}
			df := discountFactorAt(discountRate, t)
			pv := benRenta.Mul(survProb).Mul(df)

			// Check fin_derecho_edad against the member's actual age.
			if b.FinDerechoEdad != nil {
				actualAge := startAge + t
				if actualAge >= *b.FinDerechoEdad {
					break
				}
			}

			benTotal = benTotal.Add(pv)
		}
		total = total.Add(benTotal)
		breakdown[string(b.Rol)] = breakdown[string(b.Rol)].Add(benTotal)
	}

	return total, breakdown, nil
}

// computeDescalce measures the difference between the reserve valued with the
// contemporaneous table and the reserve valued with the anchored (bautizo)
// table. Each member's table is transiently swapped, the contemporaneous
// reserve is measured, and the base tables are restored.
func (sim *Simulator) computeDescalce(members []*liveMember, grupo *models.GrupoFamiliar, methodology models.PolicyMethodology, rentaAnual, discountRate decimal.Decimal, t int, base decimal.Decimal) decimal.Decimal {
	type swap struct {
		i    int
		name string
	}
	var swaps []swap
	for i := range members {
		cur := &members[i].Beneficiario
		if cur.TablaAsignada == "" || cur.Estado != "ACTIVO" {
			continue
		}
		cont := models.SelectContemporaneaTable(cur.Rol, tableTypeOf(*cur), cur.Sexo, sim.valuationDate)
		if cont != cur.TablaAsignada {
			swaps = append(swaps, swap{i, cur.TablaAsignada})
			cur.TablaAsignada = cont
		}
	}
	if len(swaps) == 0 {
		return decimal.Zero
	}
	contReserve, _, err := sim.computeReserve(members, grupo, methodology, rentaAnual, discountRate, t)
	if err != nil {
		contReserve = decimal.Zero
	}
	for _, w := range swaps {
		members[w.i].Beneficiario.TablaAsignada = w.name
	}
	return contReserve.Sub(base)
}

// tableTypeOf derives the TableType for a beneficiary for table selection.
func tableTypeOf(b models.Beneficiario) string {
	if b.SituacionInvalidez == models.InvTotal || b.SituacionInvalidez == models.InvParcial {
		return string(models.TableTypeInvalidez)
	}
	return string(models.TableTypeVejez)
}

// gradualRecognitionFactor returns the fraction of the descalce recognized in
// year t, driven by the scenario's curva_descalce or gradualidad_anios (linear
// default). Immediate recognition corresponds to a full factor of 1.
func gradualRecognitionFactor(p PolicyDef, t int) decimal.Decimal {
	var curva []float64
	switch {
	case len(p.CurvaDescalce) > 0:
		curva = p.CurvaDescalce
	case p.GradualidadAnios > 0:
		n := p.GradualidadAnios
		for i := 0; i < n; i++ {
			curva = append(curva, 1/float64(n))
		}
	default:
		return decimal.NewFromInt(1)
	}
	total := 0.0
	for _, c := range curva {
		total += c
	}
	if total <= 0 {
		return decimal.NewFromInt(1)
	}
	acc := 0.0
	for i := 0; i < t && i < len(curva); i++ {
		acc += curva[i]
	}
	factor := acc / total
	if factor > 1 {
		factor = 1
	}
	if factor < 0 {
		factor = 0
	}
	return decimal.NewFromFloat(factor)
}

// contractDate resolves the policy's contract date, falling back to the
// simulation start when absent.
func contractDate(s string, fallback time.Time) time.Time {
	if t := parseFechaContrato(s); t != nil {
		return *t
	}
	return fallback
}

// parseFechaContrato converts a YYYY-MM-DD contract date, or nil.
func parseFechaContrato(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func discountFactorAt(rate decimal.Decimal, t int) decimal.Decimal {
	if t == 0 {
		return decimal.NewFromInt(1)
	}
	base := decimal.NewFromInt(1).Add(rate)
	pow := decimal.NewFromInt(1)
	for i := 0; i < t; i++ {
		pow = pow.Mul(base)
	}
	return decimal.NewFromInt(1).Div(pow)
}

func groupEventsByYear(events []EventDef) map[int][]EventDef {
	m := make(map[int][]EventDef)
	for _, ev := range events {
		m[ev.Year] = append(m[ev.Year], ev)
	}
	return m
}

func uniqueTables(members []*liveMember) []string {
	seen := make(map[string]bool)
	var tables []string
	for _, m := range members {
		if m.Beneficiario.Estado == "ACTIVO" && m.Beneficiario.TablaAsignada != "" && !seen[m.Beneficiario.TablaAsignada] {
			seen[m.Beneficiario.TablaAsignada] = true
			tables = append(tables, m.Beneficiario.TablaAsignada)
		}
	}
	return tables
}

func defaultIfEmpty(val, def string) string {
	if val == "" {
		return def
	}
	return val
}

// memberTables snapshots the contract-time mortality tables for the report.
func memberTables(members []*liveMember) []MemberTable {
	var out []MemberTable
	for _, m := range members {
		out = append(out, MemberTable{
			Rol:   string(m.Beneficiario.Rol),
			Sexo:  m.Beneficiario.Sexo,
			Edad:  m.Beneficiario.EdadContratacion,
			Tabla: m.Beneficiario.TablaAsignada,
		})
	}
	return out
}

// parseBirthDate converts a YYYY-MM-DD birth date into a time, or nil when absent.
func parseBirthDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// invalidezFrom maps the scenario definition onto the C1194 SITUACION-INVALIDEZ
// field. The pension type takes precedence for the causante (RV codes 06/07 =
// invalidez total/parcial); otherwise a member's CONDICION flag decides.
func invalidezFrom(mdef MemberDef, tipoPension string) string {
	switch tipoPension {
	case models.TipoPensionRVInvTotal:
		return models.InvTotal
	case models.TipoPensionRVInvParcial:
		return models.InvParcial
	}
	switch mdef.Condicion {
	case "INVALIDO", "INVALIDO_TOTAL":
		return models.InvTotal
	case "INVALIDO_PARCIAL":
		return models.InvParcial
	}
	return models.InvNo
}

// computeAnnuityFactor pre-computes the annuity factor for renta derivation.
// This is a simplified version; the real one runs through FlowProjector.
func computeAnnuityFactor(mortRepo *database.MortalityRepository, members []*liveMember, methodology models.PolicyMethodology, s *Scenario) float64 {
	me := calculator.NewMortalityEngine()
	for _, t := range uniqueTables(members) {
		_ = me.EnsureLoaded(mortRepo, t)
	}

	grupo := &models.GrupoFamiliar{}
	for i := range members {
		if members[i].Beneficiario.Estado != "ACTIVO" {
			continue
		}
		m := members[i].Beneficiario
		if m.Rol == models.RolCausante {
			gf := grupo
			gf.Causante = &m
		} else {
			cp := members[i].Beneficiario
			grupo.Beneficiarios = append(grupo.Beneficiarios, &cp)
		}
	}

	if grupo.Causante == nil {
		return 1.0
	}

	projector := calculator.NewFlowProjector(me)
	policy := models.Policy{}
	result, err := projector.Project(policy, grupo, decimal.NewFromInt(1), decimal.NewFromFloat(s.Policy.TasaTC), 0)
	if err != nil || result.TotalReserve.LessThanOrEqual(decimal.Zero) {
		return 1.0
	}
	f, _ := result.TotalReserve.Float64()
	return f
}

// SortSteps ensures chronological order.
func SortSteps(steps []StepResult) {
	sort.Slice(steps, func(i, j int) bool {
		return steps[i].Year < steps[j].Year
	})
}
