# CRUSH.md - Agent Guide for Reservas Calculator

## Project Overview

This is a **reserves calculator** for actuarial sciences, specifically for calculating "Reservas de Rentas Vitalicias" according to CMF (Chile's financial regulator) instructions. The project is currently in early planning stage with documentation only.

## Current State

- **Stage**: Planning/Architecture design complete
- **Language**: Go selected for performance and regulatory compliance
- **Database**: SQLite with optimized schema for mortality tables and policy data
- **Key Documentation**: Complete analysis of NCG 318, mortality tables, and implementation plan
- **Data Source**: Mortality tables available in `data/normativo/articles-20210_tablas_mort_hist.xlsx`

## Technology Stack

**Go with SQLite** (final decision):

**Key Go Packages:**
- `github.com/shopspring/decimal` - Financial decimal handling
- `github.com/mattn/go-sqlite3` - Database operations
- `github.com/gin-gonic/gin` - HTTP API layer
- `github.com/tealeg/xlsx` - Excel mortality table parsing
- `github.com/rs/zerolog` - Structured logging for audit trails

## Database Schema

**Core Tables:**
```sql
-- Mortality tables (from Excel data/normativo/articles-20210_tablas_mort_hist.xlsx)
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre VARCHAR(50), -- "CB-H-2020", "RV-M-2020", etc.
    sexo CHAR(1), -- 'H', 'M', 'A'
    edad INTEGER,
    prob_muerte DECIMAL(10,8),
    prob_supervivencia DECIMAL(10,8),
    vigencia_inicio DATE
);

-- Policy data
CREATE TABLE poliza (
    id INTEGER PRIMARY KEY,
    numero_poliza VARCHAR(50) UNIQUE,
    tipo_renta VARCHAR(20),
    fecha_inicio DATE,
    edad_contratante INTEGER,
    sexo_beneficiario CHAR(1),
    capital_asegurado DECIMAL(15,2),
    tasa_descuento DECIMAL(8,6) -- "bautizo" rate (min TM, TC)
);

-- Reserve calculations with audit trail
CREATE TABLE reserva_calculada (
    id INTEGER PRIMARY KEY,
    poliza_id INTEGER,
    fecha_calculo DATE,
    valor_reserva DECIMAL(15,2),
    metodo_calculo VARCHAR(20),
    flujo_probabilistico DECIMAL(15,2),
    tasa_descuento_utilizada DECIMAL(8,6)
);
```

## Key Implementation Requirements

**Critical Formulas:**
```latex
VPP_j = \sum_{i=1}^{n} FP_{ji} \times (1 + TC_j)^{-i}
```
```latex
VTD = ET + AV
```
```latex
\text{Discount rate} = \min(TM_j, TC_j) \text{ per policy}
```
```latex
FP_{ji} = \text{Renta}_j \times {}_i p_x
```

**Regulatory Tables Required:**
- CB-H-2020, RV-M-2020, B-M-2020 (current from Circular N°2332)
- MI-H-2020, MI-M-2020 (disability tables)
- B-2006, MI-2006 (legacy for pre-2012 policies)

**Parallel Processing:**
- Each policy calculated independently using goroutines
- Worker pool pattern for batch processing
- Thread-safe decimal arithmetic for financial precision

**Compliance Features:**
- TAP (Test de Adecuación de Pasivos) quarterly testing
- NCG N°209 asset sufficiency analysis
- Complete audit trail for CMF regulatory review
- Automatic impairment testing for reinsurance assets

## Git Workflow & Development Process

**Branching Strategy:**
- `main` - Production-ready code, only after testing completion
- `feature/` - All new features in separate feature branches
- `bugfix/` - Bug fixes in dedicated branches
- `docs/` - Documentation updates in separate branches

**Git Commands:**
```bash
# Create feature branch
git checkout -b feature/mortality-table-loader

# Stage changes with clear messages
git add src/loader/mortality.go
git commit -m "Implement mortality table loader from Excel files

- Parse articles-20210_tablas_mort_hist.xlsx for CB-H-2020, RV-M-2020 tables
- Validate data integrity and age range coverage
- Load into tabla_mortalidad with proper indexing

💘 Generated with Crush

Co-Authored-By: Crush <crush@charm.land>"

# Merge to main after testing
git checkout main
git merge feature/mortality-table-loader
git branch -d feature/mortality-table-loader

# Never push - only user pushes to remote
```

**Commit Message Format:**
```
Brief description of change

Detailed explanation of what was changed and why
Technical details or regulatory references

💘 Generated with Crush

Co-Authored-By: Crush <crush@charm.land>
```

**Rules:**
- NO emojis in code, comments, or documentation
- Professional scientific/regulatory tone
- Each logical change in separate commit
- Test before merging to main
- Never `git push` - only authorized user pushes

## Development Environment

**Prerequisites:**
- Go 1.21+ for parallel calculation engine
- SQLite3 for local development database
- Conda available (`/Users/fcaraneda/anaconda3/bin/conda`) for potential Python utilities
- Excel reader for mortality table import from `data/normativo/articles-20210_tablas_mort_hist.xlsx`

**Build Commands:**
```bash
# Build the calculator binary
go build -o bin/calculator cmd/calculator/main.go

# Run with configuration
./bin/calculator --config config/config.json

# Run tests
go test ./...

# Lint and format
go fmt ./...
go vet ./...
```

## Documentation Architecture

## Documentation Architecture

**Source of Truth Files (4 total):**
- **`CRUSH.md`** - This file, complete agent guide
- **`docs/technical_specifications.md`** - All technical requirements, formulas, and implementation details
- **`docs/normative_framework.md`** - Complete regulatory analysis and reference mapping  
- **`docs/deployment_guide.md`** - Build, deployment, and production operations

**Supporting Analysis:**
- **`docs/analysis/`** - Temporary analysis documents to be consolidated into source of truth
- Remove fragmented documentation files after consolidation

**Documentation Philosophy:**
- Single source of truth for each domain
- Consolidate related information into 4 core documents
- Avoid documentation fragmentation
- Maintain audit trail in analysis folder

## Implementation Status

**Documentation Complete:**
- `docs/reserve_calculation_guide.md` - NCG 318 analysis and formulas
- `docs/mortality_tables_guide.md` - Mortality table loading and usage
- `docs/implementation_plan.md` - 12-week development roadmap

**Ready for Development:**
- Complete regulatory analysis from NCG 318
- Mortality table schema design
- Calculation formulas (VPPj = Σ FPji × (1 + TCj)^-i)
- VTD discount rate methodology
- Technology stack finalized (Go + SQLite)

**Next Steps:**
1. Initialize Go project structure
2. Implement mortality table loader from Excel
3. Build VPPj calculation engine with decimal precision
4. Add parallel processing for policy batch calculations
5. Implement TAP testing for quarterly compliance

## Key Implementation Files

**Core Calculation Engine:**
- `internal/calculator/reserve.go` - VPPj formula implementation
- `internal/calculator/vtd.go` - VTD rate vector processing
- `internal/calculator/mortality.go` - Mortality table management
- `internal/calculator/tap.go` - Quarterly TAP testing

**Data Models:**
- `internal/models/policy.go` - Policy data structure
- `internal/models/mortality.go` - Mortality table models
- `internal/models/reserve.go` - Reserve calculation results

**Database:**
- `internal/database/migrations/` - Schema migrations
- `internal/database/connection.go` - SQLite connection management

**Configuration:**
- `internal/config/config.go` - CMF rates, calculation parameters
- `cmd/calculator/main.go` - Application entry point