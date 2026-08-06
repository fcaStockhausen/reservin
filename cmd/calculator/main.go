package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/calculator"
	"reservas/internal/config"
	"reservas/internal/database"
	"reservas/internal/loader"
	"reservas/internal/models"
	"reservas/internal/scenario"
)

const (
	version = "0.1.0"
	appName = "reservas-calculator"
)

func main() {
	// Define command-line flags
	versionFlag := flag.Bool("version", false, "Show version information")
	configFlag := flag.String("config", "./config/config.json", "Configuration file path")
	initFlag := flag.Bool("init", false, "Initialize database with schema")
	migrateFlag := flag.Bool("migrate", false, "Run database migrations")
	importFlag := flag.String("import", "", "Import data from file (mortality|vtd)")
	statsFlag := flag.Bool("stats", false, "Show database statistics")
	seedFlag := flag.Bool("seed-demo", false, "Create a demo policy with family group")
	familiaFlag := flag.String("familia", "", "Show family group for policy ID")
	calcFlag := flag.String("calc", "", "Calculate reserve for policy ID")
	exportFlag := flag.String("calc-export", "", "Calculate and export flows to Excel (policy ID)")
	scenarioFlag := flag.String("scenario", "", "Run simulation from YAML file or builtin name")
	scenarioAllFlag := flag.Bool("scenario-all", false, "Run all builtin scenarios and compare")

	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("%s version %s\n", appName, version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configFlag)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database connection
	db, err := database.NewConnection(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Perform health check
	if err := db.HealthCheck(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}

	fmt.Printf("Reservas Calculator v%s initialized\n", version)

	// Handle different command modes
	switch {
	case *initFlag:
		handleInitialize(db)
	case *migrateFlag:
		handleMigrate(db)
	case *importFlag != "":
		handleImport(db, cfg, *importFlag)
	case *statsFlag:
		handleStats(db)
	case *familiaFlag != "":
		handleFamilia(db, *familiaFlag)
	case *seedFlag:
		handleSeedDemo(db)
	case *calcFlag != "":
		handleCalc(db, *calcFlag, "")
	case *exportFlag != "":
		handleCalc(db, *exportFlag, "export")
	case *scenarioFlag != "":
		handleScenario(db, *scenarioFlag)
	case *scenarioAllFlag:
		handleScenarioAll(db)
	default:
		fmt.Println("Use -help for available commands")
		fmt.Println("Available options:")
		fmt.Println("  -init              Initialize database")
		fmt.Println("  -migrate           Run migrations")
		fmt.Println("  -import <type>     Import data (mortality|vtd)")
		fmt.Println("  -stats             Show database statistics")
		fmt.Println("  -seed-demo         Create demo policy with family group")
		fmt.Println("  -familia <id>      Show family group for policy")
		fmt.Println("  -calc <id>         Calculate reserve for policy")
		fmt.Println("  -calc-export <id>  Calculate and export flows to Excel")
		fmt.Println("  -scenario <name>    Run simulation (YAML file or builtin name)")
		fmt.Println("  -scenario-all       Run all builtin scenarios and compare")
		fmt.Println("  -version           Show version")
		fmt.Println("  -config <path>     Configuration file path")
	}
}

func handleInitialize(db *database.DB) {
	fmt.Println("Initializing database...")

	// Create migrator
	migrator := database.NewMigrator(db.DB)

	// Run migrations
	if err := migrator.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	fmt.Println("Database initialization completed successfully")
}

func handleMigrate(db *database.DB) {
	fmt.Println("Running database migrations...")

	// Create migrator
	migrator := database.NewMigrator(db.DB)

	// Run migrations
	if err := migrator.Migrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Show migration history
	history, err := migrator.GetMigrationHistory()
	if err != nil {
		log.Printf("Warning: Failed to get migration history: %v", err)
	} else {
		fmt.Println("Migration history:")
		for _, record := range history {
			fmt.Printf("  Version %d: %s (applied: %s)\n",
				record.Version, record.Description, record.AppliedAt)
		}
	}
}

func handleImport(db *database.DB, cfg config.Config, importType string) {
	fmt.Printf("Importing %s data...\n", importType)

	switch importType {
	case "mortality":
		handleImportMortality(db, cfg)
	case "vtd":
		handleImportVTD(db, cfg)
	default:
		log.Fatalf("Unknown import type: %s. Use 'mortality' or 'vtd'", importType)
	}
}

func handleImportMortality(db *database.DB, cfg config.Config) {
	path := cfg.Data.MortalityTables.Path
	fmt.Printf("Importing mortality tables from %s...\n", path)

	ld := loader.NewMortalityLoader(path)
	records, err := ld.Load()
	if err != nil {
		log.Fatalf("Failed to parse mortality Excel: %v", err)
	}
	fmt.Printf("Parsed %d mortality records from Excel\n", len(records))

	repo := database.NewMortalityRepository(db.DB)
	if err := repo.BatchInsert(records); err != nil {
		log.Fatalf("Failed to insert mortality records: %v", err)
	}

	fmt.Printf("Successfully imported %d mortality records\n", len(records))

	// Show statistics after import
	stats, err := repo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get statistics: %v", err)
	} else {
		fmt.Println("Mortality table statistics:")
		for key, value := range stats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func handleImportVTD(db *database.DB, cfg config.Config) {
	path := cfg.Data.VTDData.Path
	fmt.Printf("Importing VTD vectors from %s...\n", path)

	ld := loader.NewVTDLoader(path)
	points, err := ld.Load()
	if err != nil {
		log.Fatalf("Failed to parse VTD Excel: %v", err)
	}
	fmt.Printf("Parsed %d VTD points from Excel\n", len(points))

	repo := database.NewVTDRepository(db.DB)
	if err := repo.BatchInsert(points); err != nil {
		log.Fatalf("Failed to insert VTD points: %v", err)
	}

	fmt.Printf("Successfully imported %d VTD points\n", len(points))

	// Show statistics after import
	stats, err := repo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get statistics: %v", err)
	} else {
		fmt.Println("VTD vector statistics:")
		fmt.Printf("  Total vectors: %d\n", stats.TotalVectors)
		fmt.Printf("  Total points: %d\n", stats.TotalPoints)
		if !stats.DateRange.Start.IsZero() {
			fmt.Printf("  Date range: %s to %s\n",
				stats.DateRange.Start.Format("2006-01-02"),
				stats.DateRange.End.Format("2006-01-02"))
		}
	}
}

func handleStats(db *database.DB) {
	fmt.Println("Database statistics:")

	// Get database stats
	dbStats, err := db.GetStats()
	if err != nil {
		log.Printf("Warning: Failed to get database statistics: %v", err)
	} else {
		fmt.Println("Database:")
		for key, value := range dbStats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Get mortality table stats
	mortalityRepo := database.NewMortalityRepository(db.DB)
	mortalityStats, err := mortalityRepo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get mortality table statistics: %v", err)
	} else {
		fmt.Println("\nMortality tables:")
		for key, value := range mortalityStats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Get VTD vector stats
	vtdRepo := database.NewVTDRepository(db.DB)
	vtdStats, err := vtdRepo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get VTD statistics: %v", err)
	} else {
		fmt.Println("\nVTD vectors:")
		fmt.Printf("  Total vectors: %d\n", vtdStats.TotalVectors)
		fmt.Printf("  Total points: %d\n", vtdStats.TotalPoints)
		fmt.Printf("  Date range: %s to %s\n",
			vtdStats.DateRange.Start.Format("2006-01-02"),
			vtdStats.DateRange.End.Format("2006-01-02"))
	}

	// Get policy stats
	policyRepo := database.NewPolicyRepository(db.DB)
	policyStats, err := policyRepo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get policy statistics: %v", err)
	} else {
		fmt.Println("\nPolicy statistics:")
		for key, value := range policyStats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	// Get beneficiario stats
	benRepo := database.NewBeneficiarioRepository(db.DB)
	benStats, err := benRepo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get beneficiario statistics: %v", err)
	} else {
		fmt.Println("\nBeneficiarios:")
		for key, value := range benStats {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}
}

func handleSeedDemo(db *database.DB) {
	fmt.Println("Creating demo policy with family group...")

	policyRepo := database.NewPolicyRepository(db.DB)

	if existing, _ := policyRepo.GetByNumeroPoliza("DEMO-001"); existing != nil {
		fmt.Printf("Demo policy already exists (ID: %d). Use -familia %d to view.\n", existing.ID, existing.ID)
		return
	}

	policy := models.Policy{
		NumeroPoliza:     "DEMO-001",
		TipoRenta:        "VITALICIA",
		FechaInicio:      time.Date(2023, 3, 15, 0, 0, 0, 0, time.UTC),
		EdadContratante:  65,
		SexoBeneficiario: models.SexoFemenino,
		CapitalAsegurado: decimal.NewFromFloat(50000000),
		FormaPago:        "MENSUAL",
		TasaTM:           decimal.NewFromFloat(0.038),
		TasaTC:           decimal.NewFromFloat(0.035),
		Estado:           "ACTIVA",
		TipoPension:      models.TipoPensionRVVejezJubilacion,
		ModalidadRenta:   "1000",
		VigenciaPension:  models.VigenciaEnPago,
	}
	policy.TasaDescuento = decimal.Min(policy.TasaTM, policy.TasaTC)

	polizaID, err := policyRepo.Insert(policy)
	if err != nil {
		log.Fatalf("Failed to create demo policy: %v", err)
	}
	fmt.Printf("Created policy DEMO-001 (ID: %d)\n", polizaID)

	methodology := policy.GetMethodology()
	members := []models.Beneficiario{
		{
			PolizaID:              polizaID,
			Rol:                   models.RolCausante,
			Sexo:                  models.SexoFemenino,
			EdadContratacion:      65,
			PorcentajeRenta:       decimal.NewFromFloat(1.0),
			Estado:                "ACTIVO",
			TipoBeneficiarioC1194: models.C1194Afiliado,
			DerechoPension:        models.DerechoPensionSi,
			DerechoAcrecer:        "N",
			SituacionInvalidez:    models.InvNo,
		},
		{
			PolizaID:              polizaID,
			Rol:                   models.RolConyuge,
			Sexo:                  models.SexoMasculino,
			EdadContratacion:      68,
			PorcentajeRenta:       decimal.NewFromFloat(0.50),
			Estado:                "ACTIVO",
			TipoBeneficiarioC1194: models.C1194ConyugeConHijos,
			DerechoPension:        models.DerechoPensionSi,
			DerechoAcrecer:        "S",
			SituacionInvalidez:    models.InvNo,
			MatrimonioAnios:       10,
		},
		{
			PolizaID:              polizaID,
			Rol:                   models.RolHijo,
			Sexo:                  models.SexoMasculino,
			EdadContratacion:      20,
			PorcentajeRenta:       decimal.NewFromFloat(0.15),
			Estado:                "ACTIVO",
			TipoBeneficiarioC1194: models.C1194HijoSinIncremento,
			DerechoPension:        models.DerechoPensionSi,
			SituacionInvalidez:    models.InvNo,
			Condicion:             "ESTUDIANTE",
			FinDerechoEdad:        intPtr(24),
		},
	}

	for i := range members {
		tipoTabla := ""
		if members[i].Rol == models.RolCausante {
			tipoTabla = string(models.TableTypeVejez)
		}
		members[i].TablaAsignada = models.SelectTableForBeneficiario(
			members[i].Rol, members[i].Sexo, members[i].TipoBeneficiarioC1194, methodology, tipoTabla,
		)
	}

	benRepo := database.NewBeneficiarioRepository(db.DB)
	if err := benRepo.BatchInsert(members); err != nil {
		log.Fatalf("Failed to insert family group: %v", err)
	}

	fmt.Printf("Created family group with %d members\n", len(members))
	fmt.Println("\nFamily group details:")
	for _, m := range members {
		fmt.Printf("  %s | %s | edad %d | tabla %s | %.0f%% renta\n",
			m.Rol, m.Sexo, m.EdadContratacion, m.TablaAsignada,
			m.PorcentajeRenta.Mul(decimal.NewFromInt(100)).InexactFloat64())
	}
	fmt.Printf("\nUse -familia %d to view the group anytime.\n", polizaID)
}

func handleFamilia(db *database.DB, polizaIDStr string) {
	var polizaID int
	if _, err := fmt.Sscanf(polizaIDStr, "%d", &polizaID); err != nil {
		log.Fatalf("Invalid policy ID: %s", polizaIDStr)
	}

	policyRepo := database.NewPolicyRepository(db.DB)
	policy, err := policyRepo.GetByID(polizaID)
	if err != nil {
		log.Fatalf("Policy not found: %v", err)
	}

	fmt.Printf("Policy: %s (ID: %d)\n", policy.NumeroPoliza, policy.ID)
	fmt.Printf("  Tipo: %s | Inicio: %s | Sexo: %s | Edad: %d\n",
		policy.TipoRenta, policy.FechaInicio.Format("2006-01-02"),
		policy.SexoBeneficiario, policy.EdadContratante)
	fmt.Printf("  Capital: %s | TM: %s%% | TC: %s%%\n",
		policy.CapitalAsegurado.String(),
		policy.TasaTM.Mul(decimal.NewFromInt(100)).StringFixed(2),
		policy.TasaTC.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  Methodology: %s\n", policy.GetMethodology())

	benRepo := database.NewBeneficiarioRepository(db.DB)
	gf, err := benRepo.GetGrupoFamiliar(polizaID)
	if err != nil {
		log.Fatalf("Failed to get family group: %v", err)
	}

	if gf.Causante == nil && !gf.HasBeneficiarios() {
		fmt.Println("\nNo family group members found for this policy.")
		return
	}

	fmt.Println("\nFamily group:")
	if gf.Causante != nil {
		printMember(gf.Causante, true)
	}
	for _, b := range gf.Beneficiarios {
		printMember(b, false)
	}

	mortRepo := database.NewMortalityRepository(db.DB)
	for _, m := range gf.AllMembers() {
		tables, err := mortRepo.GetByStandardName(m.TablaAsignada)
		if err != nil || len(tables) == 0 {
			fmt.Printf("\nWARNING: Table %s not found in database for %s!",
				m.TablaAsignada, m.Rol)
		}
	}
}

func printMember(b *models.Beneficiario, isCausante bool) {
	pct := b.PorcentajeRenta.Mul(decimal.NewFromInt(100)).StringFixed(1)
	birthDate := "N/A"
	if b.FechaNacimiento != nil {
		birthDate = b.FechaNacimiento.Format("2006-01-02")
	}
	label := ""
	if isCausante {
		label = " <-- CAUSANTE"
	}
	finDer := ""
	if b.FinDerechoEdad != nil {
		finDer = fmt.Sprintf(" | fin_der: %d", *b.FinDerechoEdad)
	}
	fmt.Printf("  [%s] C1194:%s | sexo: %s | edad: %d | nac: %s | tabla: %s | renta: %s%% | der: %s | inv: %s%s%s\n",
		b.Rol, b.TipoBeneficiarioC1194, b.Sexo, b.EdadContratacion,
		birthDate, b.TablaAsignada, pct, b.DerechoPension, b.SituacionInvalidez, finDer, label)
}

func intPtr(v int) *int {
	return &v
}

func handleCalc(db *database.DB, polizaIDStr string, mode string) {
	var polizaID int
	if _, err := fmt.Sscanf(polizaIDStr, "%d", &polizaID); err != nil {
		log.Fatalf("Invalid policy ID: %s", polizaIDStr)
	}

	policyRepo := database.NewPolicyRepository(db.DB)
	policy, err := policyRepo.GetByID(polizaID)
	if err != nil {
		log.Fatalf("Policy not found: %v", err)
	}

	benRepo := database.NewBeneficiarioRepository(db.DB)
	grupo, err := benRepo.GetGrupoFamiliar(polizaID)
	if err != nil {
		log.Fatalf("Failed to get family group: %v", err)
	}
	if grupo.Causante == nil {
		log.Fatalf("Policy has no causante in family group")
	}

	mortRepo := database.NewMortalityRepository(db.DB)
	calc := calculator.NewReserveCalculator(mortRepo)

	// Step 1: Project with unitary rent (R=1) to compute the annuity factor.
	// This gives us the factor by which we divide the capital to get the annual pension.
	unitResult, err := calc.Calculate(*policy, grupo, decimal.NewFromInt(1))
	if err != nil {
		log.Fatalf("Annuity factor calculation failed: %v", err)
	}
	annuityFactor := unitResult.TotalReserve

	if annuityFactor.LessThanOrEqual(decimal.Zero) {
		log.Fatalf("Annuity factor is zero or negative — check mortality tables")
	}

	// Step 2: Derive the annual pension from the capital and annuity factor.
	rentaAnual := policy.CapitalAsegurado.Div(annuityFactor)

	// Step 3: Project real flows with the derived pension.
	result, err := calc.Calculate(*policy, grupo, rentaAnual)
	if err != nil {
		log.Fatalf("Reserve calculation failed: %v", err)
	}

	discountRate := policy.GetEffectiveDiscountRate()

	fmt.Printf("Policy: %s (ID: %d)\n", policy.NumeroPoliza, policy.ID)
	fmt.Printf("  Capital:        $%s\n", policy.CapitalAsegurado.StringFixed(0))
	fmt.Printf("  Renta anual:    $%s\n", rentaAnual.StringFixed(0))
	fmt.Printf("  Renta mensual:  $%s\n", rentaAnual.Div(decimal.NewFromInt(12)).StringFixed(0))
	fmt.Printf("  Tasa descuento: %s%% (min TM %s%% / TC %s%%)\n",
		discountRate.Mul(decimal.NewFromInt(100)).StringFixed(4),
		policy.TasaTM.Mul(decimal.NewFromInt(100)).StringFixed(2),
		policy.TasaTC.Mul(decimal.NewFromInt(100)).StringFixed(2))
	fmt.Printf("  Metodologia:    %s\n", policy.GetMethodology())
	fmt.Printf("  Periodos:       %d\n", result.Periods)
	fmt.Printf("  Flujos totales: %d\n", len(result.Flows))
	fmt.Printf("  RESERVA VPP:    $%s\n", result.TotalReserve.StringFixed(2))

	// Breakdown by role
	byRole := make(map[string]decimal.Decimal)
	for _, f := range result.Flows {
		byRole[f.MemberRol] = byRole[f.MemberRol].Add(f.PresentValue)
	}
	fmt.Println("\n  Desglose por rol:")
	for _, m := range grupo.AllMembers() {
		vp := byRole[string(m.Rol)]
		fmt.Printf("    %s (%s, tabla %s): $%s\n",
			m.Rol, m.Sexo, m.TablaAsignada, vp.StringFixed(2))
	}

	if mode == "export" {
		outputPath := fmt.Sprintf("flujos_poliza_%d.xlsx", polizaID)
		if err := calculator.ExportFlowsToExcel(result, outputPath); err != nil {
			log.Fatalf("Excel export failed: %v", err)
		}
		fmt.Printf("\nFlujos exportados a: %s\n", outputPath)
		fmt.Println("  Sheet 'Flujos': flujo a flujo desagregado por miembro")
		fmt.Println("  Sheet 'Resumen': VP total por rol y reserva total")
	}
}

func handleScenario(db *database.DB, name string) {
	mortRepo := database.NewMortalityRepository(db.DB)

	var s *scenario.Scenario
	var err error

	// Check if it's a YAML file or a builtin name
	if _, exists := scenario.BuiltinScenarios[name]; exists {
		s, err = scenario.LoadBuiltin(name)
	} else {
		s, err = scenario.Load(name)
	}
	if err != nil {
		log.Fatalf("Failed to load scenario: %v", err)
	}

	runAndPrintScenario(mortRepo, s)
}

func handleScenarioAll(db *database.DB) {
	mortRepo := database.NewMortalityRepository(db.DB)

	var allResults []*scenario.SimulationResult

	for name := range scenario.BuiltinScenarios {
		s, err := scenario.LoadBuiltin(name)
		if err != nil {
			log.Printf("Failed to load scenario %s: %v", name, err)
			continue
		}
		fmt.Printf("\n%s\n%s\n", s.Name, s.Description)
		result, err := runScenario(mortRepo, s)
		if err != nil {
			log.Printf("Failed to run %s: %v", name, err)
			continue
		}
		allResults = append(allResults, result)
	}

	if len(allResults) == 0 {
		log.Fatalf("No scenarios ran successfully")
	}

	// Comparative summary
	fmt.Println("\n" + repeat("=", 80))
	fmt.Println("COMPARATIVO DE ESCENARIOS")
	fmt.Println(repeat("=", 80))
	fmt.Printf("%-25s %15s %15s %15s %8s\n", "Escenario", "Reserva Max", "Reserva Min", "Reserva Final", "Eventos")
	fmt.Println(repeat("-", 80))
	for _, r := range allResults {
		fmt.Printf("%-25s %15s %15s %15s %8d\n",
			r.ScenarioName,
			r.MaxReserve.StringFixed(2),
			r.MinReserve.StringFixed(2),
			r.FinalReserve.StringFixed(2),
			r.EventsTotal,
		)
	}

	// Export comparative Excel
	outputPath := "comparativo_escenarios.xlsx"
	if err := scenario.ExportComparative(allResults, outputPath); err != nil {
		log.Printf("Warning: Excel export failed: %v", err)
	} else {
		fmt.Printf("\nComparativo exportado a: %s\n", outputPath)
	}
}

func runAndPrintScenario(mortRepo *database.MortalityRepository, s *scenario.Scenario) {
	result, err := runScenario(mortRepo, s)
	if err != nil {
		log.Fatalf("Simulation failed: %v", err)
	}

	fmt.Printf("\n%s\n%s\n", s.Name, s.Description)
	fmt.Printf("Horizonte: %d años | Eventos: %d\n\n", s.Horizon, result.EventsTotal)

	fmt.Println("Tablas de mortalidad asignadas (por estrato de fecha de contratación):")
	for _, mt := range result.Tables {
		fmt.Printf("  %-14s %s  edad %3d  ->  %s\n", mt.Rol, mt.Sexo, mt.Edad, mt.Tabla)
	}
	fmt.Println()

	fmt.Printf("%-6s %-6s %15s %15s %15s %15s %8s %s\n", "Año", "Edad", "Reserva", "Reserva Base", "Descalce Bruto", "Descalce Recon.", "Vivos", "Eventos")
	fmt.Println(repeat("-", 105))

	for _, step := range result.Steps {
		events := ""
		if len(step.Events) > 0 {
			events = joinStrings(step.Events, "; ")
		}
		marker := ""
		if len(step.Events) > 0 {
			marker = " <<<"
		}
		fmt.Printf("%-6d %-6d %15s %15s %15s %15s %8d %s%s\n",
			step.Year, step.CausanteAge,
			step.ReserveValue.StringFixed(2),
			step.ReservaBase.StringFixed(2),
			step.DescalceBruto.StringFixed(2),
			step.DescalceReconocido.StringFixed(2),
			step.MembersAlive,
			events, marker,
		)
	}

	fmt.Println(repeat("-", 70))
	fmt.Printf("Max: %s | Min: %s | Final: %s\n",
		result.MaxReserve.StringFixed(2),
		result.MinReserve.StringFixed(2),
		result.FinalReserve.StringFixed(2))

	// Export
	outputPath := fmt.Sprintf("simulacion_%s.xlsx", s.Name)
	if err := scenario.ExportSimulation(result, outputPath); err != nil {
		log.Printf("Warning: Excel export failed: %v", err)
	} else {
		fmt.Printf("Exportado a: %s\n", outputPath)
	}
}

func runScenario(mortRepo *database.MortalityRepository, s *scenario.Scenario) (*scenario.SimulationResult, error) {
	sim := scenario.NewSimulator(mortRepo)
	return sim.Run(s)
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func joinStrings(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += sep + ss[i]
	}
	return result
}
