# Technical Specifications - Reserves Calculator

## System Overview

**Purpose:** Parallelized reserves calculator for "Reservas de Rentas Vitalicias" according to Chilean CMF regulations
**Technology Stack:** Go + SQLite
**Target:** Regulatory compliant financial calculation system

## Core Mathematical Formulas

### 1. Present Value Calculation
```latex
VPP_j = \sum_{i=1}^{n} FP_{ji} \times (1 + TC_j)^{-i}
```

Where:
- $VPP_j$ = Present value of policy j at inception
- $FP_{ji}$ = Probabilistic flow of policy j in period i = Renta_j × {}_i p_x
- $TC_j$ = IRR/tasa costo equivalente (implicit in the policy)
- $n$ = Number of periods until policy termination

### 2. Discount Rate Determination
```latex
\text{Discount rate} = \min(TM_j, TC_j) \text{ per policy}
```

### 3. VTD Rate Vector (Vector Tasa de Descuento)
```latex
VTD = ET + AV
```
Where:
- $ET$ = Real risk-free term structure (120 years)
  - Liquid Segment (0-25 años): State/Central Bank market transactions
  - Extrapolated Segment (25-120 años): Smith-Wilson method, converges at 65 years to TILP
- $AV$ = 0.65 × (R_exceso - R_libre_riesgo) - 0.35 × SRC

### 4. Monthly Rate Conversion
```latex
i_{mensual}^{(j)} = (1 + i_{anual}^{(j)})^{\frac{1}{12}} - 1
```

### 5. Life Annuity Formulas
```latex
a_x = \sum_{t=1}^{\omega - x} v^t \times {}_t p_x
```
```latex
\ddot{a}_x = 1 + \sum_{t=1}^{\omega - x} v^t \times {}_t p_x
```

## Database Schema

### Core Tables

```sql
-- Mortality tables (primary data source)
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre_estandar VARCHAR(50), -- CMF standard: "CB-H-2020" (Causante/Beneficiario), "RV-M-2020", etc.
    nombre_original VARCHAR(50),   -- Excel name: "CB-2020-HOMBRES", etc.
    sexo CHAR(1),                  -- 'H', 'M'
    tipo_tabla VARCHAR(20),        -- 'VEJEZ', 'INVALIDEZ', 'SOBREVIVENCIA'
    año_tabla INTEGER,             -- 2004, 2006, 2009, 2014, 2020
    edad INTEGER,
    prob_muerte DECIMAL(10,8),     -- qx value
    factor_aax DECIMAL(10,8),      -- Factor Aax
    vigencia_inicio DATE,           -- Effectivity date
    vigencia_fin DATE,              -- End date if applicable
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(nombre_estandar, edad)
);

-- Policy data
CREATE TABLE poliza (
    id INTEGER PRIMARY KEY,
    numero_poliza VARCHAR(50) UNIQUE,
    tipo_renta VARCHAR(20),        -- 'VITALICIA', 'TEMPORARIA', 'DIFERIDA'
    fecha_inicio DATE,
    fecha_fin DATE,
    edad_contratante INTEGER,
    sexo_beneficiario CHAR(1),
    capital_asegurado DECIMAL(15,2),
    forma_pago VARCHAR(10),         -- 'MENSUAL', 'TRIMESTRAL', 'ANUAL'
    tasa_descuento DECIMAL(8,6),   -- "bautizo" rate (min TM, TC)
    tasa_tm DECIMAL(8,6),         -- Tasa venta
    tasa_tc DECIMAL(8,6),         -- Tasa costo
    estado VARCHAR(10) DEFAULT 'ACTIVA'
);

-- Reserve calculations with audit trail
CREATE TABLE reserva_calculada (
    id INTEGER PRIMARY KEY,
    poliza_id INTEGER,
    fecha_calculo DATE,
    valor_reserva DECIMAL(15,2),
    metodo_calculo VARCHAR(20),     -- 'VPPJ', 'TRADICIONAL'
    flujo_probabilistico DECIMAL(15,2),
    tasa_descuento_utilizada DECIMAL(8,6),
    tabla_mortalidad_utilizada VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (poliza_id) REFERENCES poliza(id)
);

-- VTD vector tasa de descuento (monthly CMF publications)
CREATE TABLE vtd_mensual (
    id INTEGER PRIMARY KEY,
    fecha_publicacion DATE,
    an INTEGER,                    -- Year 1 to 120
    tasa_liquida DECIMAL(8,6),      -- ET liquid segment
    tasa_extrapolada DECIMAL(8,6),  -- ET extrapolated segment
    ajuste_volatilidad DECIMAL(8,6), -- AV adjustment
    tasa_final DECIMAL(8,6),        -- Final VTD rate
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- TM (tasa venta) historical data
CREATE TABLE tm_historico (
    id INTEGER PRIMARY KEY,
    fecha DATE,
    tipo_poliza VARCHAR(20),        -- 'RV_VITALICIA', 'RV_TEMPORARIA', etc.
    tasa DECIMAL(8,6),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- CMF regulatory parameters
CREATE TABLE parametros_cmf (
    id INTEGER PRIMARY KEY,
    nombre_parametro VARCHAR(50),    -- 'TASA_MINIMA', 'CARGA_MAXIMA', 'TILP'
    valor DECIMAL(10,6),
    fecha_vigencia DATE,
    tipo VARCHAR(10),               -- 'TASA', 'PORCENTAJE', 'MONTO'
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Performance Requirements

### Processing Targets
- **Single Policy**: < 100ms calculation time
- **Batch Processing**: 10,000 policies < 5 minutes  
- **Memory Usage**: < 1GB for 100,000 policies
- **API Response**: < 200ms for single calculation

### Parallel Processing Architecture
```go
type ReserveCalculator struct {
    db        *sql.DB
    mortality map[string]*MortalityTable
    vtd       map[string]*VTDVector
    rates     map[string]map[string]decimal.Decimal
}

func (rc *ReserveCalculator) CalculateBatch(policies []Policy) []ReserveResult {
    results := make([]ReserveResult, len(policies))
    var wg sync.WaitGroup
    
    // Worker pool for parallel calculation
    workers := runtime.NumCPU()
    jobs := make(chan PolicyJob, len(policies))
    results := make(chan ReserveResult, len(policies))
    
    for i := 0; i < workers; i++ {
        wg.Add(1)
        go rc.worker(jobs, results, &wg)
    }
    
    // Distribute policies to workers
    for i, policy := range policies {
        jobs <- PolicyJob{Index: i, Policy: policy}
    }
    close(jobs)
    
    wg.Wait()
    close(results)
    
    // Collect results
    for result := range results {
        results[result.Index] = result
    }
    
    return results
}
```

## Implementation Architecture

### Project Structure
```
reservas/
├── cmd/calculator/
│   └── main.go                 # Application entry point
├── internal/
│   ├── calculator/
│   │   ├── reserve.go          # VPPj calculation engine
│   │   ├── vtd.go             # VTD rate processing
│   │   ├── mortality.go       # Mortality table management
│   │   ├── tap.go             # TAP testing
│   │   └── parallel.go        # Batch processing
│   ├── models/
│   │   ├── policy.go          # Policy data structures
│   │   ├── mortality.go       # Mortality table models
│   │   ├── reserve.go         # Reserve calculation results
│   │   └── vtd.go            # VTD vector models
│   ├── database/
│   │   ├── migrations/        # Schema migrations
│   │   ├── connection.go      # SQLite management
│   │   └── queries.go         # Database operations
│   ├── api/
│   │   ├── handlers.go        # HTTP endpoints
│   │   ├── middleware.go      # Request processing
│   │   └── routes.go          # API routing
│   └── config/
│       └── config.go          # Configuration management
├── data/
│   └── normativo/
│       └── articles-20210_tablas_mort_hist.xlsx
├── config/
│   └── config.json
├── docs/
└── go.mod
```

### Key Go Dependencies
```go
require (
    github.com/shopspring/decimal v1.3.1      // Financial decimal arithmetic
    github.com/mattn/go-sqlite3 v1.14.18      // Database operations
    github.com/gin-gonic/gin v1.9.1           // HTTP API
    github.com/tealeg/xlsx v1.0.5            // Excel parsing
    github.com/rs/zerolog v1.30.0             // Structured logging
)
```

## Quality Assurance

### Validation Rules
- **qx Values**: Must be between 0 and 1
- **Age Ranges**: Complete coverage (typically 0-110+)
- **Monotonicity**: Death rates generally increase with age (after infancy)
- **Survival Consistency**: ${}_t p_x = \frac{l_{x+t}}{l_x}$

### Audit Trail Requirements
```go
// Structured logging for regulatory compliance
logger.Info().
    Str("policy_id", policy.ID).
    Str("calculation_type", "VPPj").
    Str("mortality_table", "CB-H-2020 (Causante)").
    Float64("reserve_amount", reserveValue).
    Float64("discount_rate_tm", tmRate).
    Float64("discount_rate_tc", tcRate).
    Float64("final_discount_rate", min(tmRate, tcRate)).
    Dur("calculation_time", duration).
    Str("user", "system").
    Str("version", "1.0.0").
    Msg("Reserve calculation completed")
```

## Compliance Features

### TAP (Test de Adecuación de Pasivos)
```go
type TAPTester struct {
    calculator *ReserveCalculator
    audit      AuditLogger
}

func (tt *TAPTester) PerformQuarterlyTest(date time.Time) error {
    // Load all active policies
    // Calculate reserves using company estimates
    // Compare with existing reserves  
    // Identify additional reserve requirements
    // Generate audit trail for CMF reporting
}
```

### Asset Sufficiency Analysis
- NCG N°209 methodology implementation
- Additional reserve calculation when required
- Comprehensive reporting for regulatory compliance

## Data Migration Process

### Mortality Table Loading
1. **Parse Excel** - Extract individual tables from multi-column sheets
2. **Standardize Names** - Map Excel names to CMF standards
3. **Validate Data** - Check ranges, monotonicity, completeness
4. **Load Database** - Insert with proper effectivity dates

### Table Name Mapping
```
Excel Name → CMF Standard
CB-2020-HOMBRES → CB-H-2020 (Causante)
RV-2020-MUJERES → RV-M-2020 (Beneficiario)  
B-2020-MUJERES → B-M-2020 (Beneficiario)
MI-2020-HOMBRES → MI-H-2020 (Causante)
MI-2020-MUJERES → MI-M-2020 (Beneficiario)
```

## Regulatory Calculations by Policy Date

### Post-December 1, 2020
- Use rate from NCG 318 Annex
- No calce measurement
- Only base technical reserve

### Post-January 1, 2012 (IFRS-aligned)
- Discount rate = min(TM, TC)
- No reserve adjustment by calce
- 100% commitments, ceded flows as assets
- Current mortality tables (CB-H-2020 series - Causante, RV-M-2020/B-M-2020 - Beneficiario)

### Pre-January 1, 2012
- Traditional Circular N°1512 methodology
- Gradual B-2006/MI-2006 table application
- Optional voluntary adoption of post-2012 rules

## Missing Technical Dependencies

### Critical External Data
1. **Monthly VTD Publications** - CMF discount rate vectors
2. **TM Historical Rates** - Tasa venta data for rate comparison
3. **Circular N°1512** - Pre-2012 calculation methodology
4. **Circular N°2332** - Current mortality table specifications

### Implementation Priority
1. **Database Schema** - Core tables and indexes
2. **Mortality Loader** - Excel parsing and validation
3. **Calculation Engine** - VPPj with decimal precision
4. **Parallel Processing** - Worker pool for batch calculations
5. **API Layer** - HTTP endpoints for system integration
6. **Compliance Modules** - TAP testing and audit trails

This technical specification provides complete implementation requirements for a production-ready reserves calculator meeting all CMF regulatory requirements.