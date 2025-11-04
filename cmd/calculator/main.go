package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"reservas/internal/config"
	"reservas/internal/database"
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
		handleImport(db, *importFlag)
	case *statsFlag:
		handleStats(db)
	default:
		fmt.Println("Use -help for available commands")
		fmt.Println("Available options:")
		fmt.Println("  -init              Initialize database")
		fmt.Println("  -migrate           Run migrations")
		fmt.Println("  -import <type>     Import data (mortality|vtd)")
		fmt.Println("  -stats             Show database statistics")
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

func handleImport(db *database.DB, importType string) {
	fmt.Printf("Importing %s data...\n", importType)

	switch importType {
	case "mortality":
		handleImportMortality(db)
	case "vtd":
		handleImportVTD(db)
	default:
		log.Fatalf("Unknown import type: %s. Use 'mortality' or 'vtd'", importType)
	}
}

func handleImportMortality(db *database.DB) {
	fmt.Println("Importing mortality tables from Excel...")
	
	// TODO: Implement Excel import for mortality tables
	fmt.Println("Mortality table import not yet implemented")
	
	// Create mortality repository
	repo := database.NewMortalityRepository(db.DB)
	
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

func handleImportVTD(db *database.DB) {
	fmt.Println("Importing VTD vectors from Excel...")
	
	// TODO: Implement Excel import for VTD data
	fmt.Println("VTD vector import not yet implemented")
	
	// Create VTD repository
	repo := database.NewVTDRepository(db.DB)
	
	// Show statistics after import
	stats, err := repo.GetStatistics()
	if err != nil {
		log.Printf("Warning: Failed to get statistics: %v", err)
	} else {
		fmt.Println("VTD vector statistics:")
		fmt.Printf("  Total vectors: %d\n", stats.TotalVectors)
		fmt.Printf("  Total points: %d\n", stats.TotalPoints)
		fmt.Printf("  Date range: %s to %s\n", 
			stats.DateRange.Start.Format("2006-01-02"), 
			stats.DateRange.End.Format("2006-01-02"))
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
}