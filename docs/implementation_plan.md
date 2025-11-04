# Implementation Plan - Reservas Calculator

## Executive Summary

**Status:** Ready for Implementation
**Timeline:** 8 weeks (2 months)
**Approach:** Simple utility application with Go + SQLite
**Priority:** Regulatory compliance with CMF standards

## Project Readiness Assessment

### ✅ Complete Foundation
- **Requirements:** All technical and regulatory specifications documented
- **Architecture:** Simple utility approach validated (not over-engineered)
- **Data:** Mortality tables, VTD vectors, TM rates available
- **Compliance:** CMF regulatory framework fully analyzed

### ✅ Critical Dependencies Resolved
- **VTD Data:** Complete 2020-2025 discount rate vectors
- **Mortality Tables:** 18 tables with age ranges 0-110
- **Regulatory Documents:** All 6 PDF documents available
- **Database Schema:** Complete design with proper indexing

### 🔧 Technology Stack Ready
- **Go 1.21+:** Performance and parallel processing
- **SQLite 3.35+:** Embedded database with WAL optimization
- **Decimal Arithmetic:** shopspring/decimal for financial precision
- **Excel Processing:** tealeg/xlsx for mortality table parsing
- **Logging:** zerolog for CMF audit trails

## Implementation Phases

### Phase 1: Core Infrastructure (Week 1-2)

#### Week 1: Database Foundation
**Priority:** HIGH
**Deliverables:**
- Complete SQLite schema implementation
- Mortality table migration from Excel
- VTD data import from Excel
- Basic data validation

**Tasks:**
```bash
# 1.1 Database Schema
- Create all tables (mortality, policy, reserves, VTD, TM)
- Implement proper indexing for performance
- Add foreign key constraints and data validation
- Create migration system for schema updates

# 1.2 Mortality Table Import
- Parse articles-20210_tablas_mort_hist.xlsx
- Standardize table names (CB-H-2020, RV-M-2020, etc.)
- Validate age ranges (0-110 years)
- Load into tabla_mortalidad with proper naming

# 1.3 VTD Data Import
- Parse VTD_2025_.xlsx with monthly vectors
- Extract period-based rates (1-120 years)
- Load into vtd_vector table
- Validate rate ranges and completeness
```

**Acceptance Criteria:**
- [ ] All mortality tables loaded (18 total)
- [ ] VTD vectors imported (2020-2025 monthly)
- [ ] Database schema validated with constraints
- [ ] Basic data integrity checks pass

#### Week 2: Calculation Engine Foundation
**Priority:** HIGH
**Deliverables:**
- Basic VPPj formula implementation
- Decimal precision handling
- Unit testing framework
- Simple reserve calculation

**Tasks:**
```bash
# 2.1 Mathematical Engine
- Implement VPPj formula with decimal precision
- Add VTD rate lookup functions
- Create survival probability calculations
- Implement discount rate determination (min(TM,TC))

# 2.2 Basic Policy Processing
- Load policy data from structured input
- Select appropriate mortality tables
- Calculate basic reserve values
- Generate calculation audit trails

# 2.3 Testing Framework
- Unit tests for mathematical formulas
- Integration tests for policy calculations
- Performance benchmarks for single policies
- Accuracy validation against known examples
```

**Acceptance Criteria:**
- [ ] VPPj calculation accurate to 8 decimal places
- [ ] Single policy calculation < 100ms
- [ ] All unit tests pass
- [ ] Basic audit trail generation working

### Phase 2: Data Processing & Integration (Week 3-4)

#### Week 3: Advanced Data Processing
**Priority:** HIGH
**Deliverables:**
- Complete mortality table management
- VTD vector integration
- Policy date-based methodology selection

**Tasks:**
```bash
# 3.1 Mortality Table Management
- Implement table selection by policy date
- Add Causante/Beneficiario classification
- Handle legacy tables (B-2006, MI-2006) for pre-2012
- Validate qx ranges and monotonicity

# 3.2 VTD Vector Integration
- Implement period-based rate lookup (1-120 years)
- Add date-based vector selection
- Handle ET + AV component calculations
- Support monthly VTD publications

# 3.3 Policy Methodology Selection
- IFRS-aligned calculations (post-2012)
- Traditional methodology (pre-2012)
- Transitional rules (2015-2020)
- Calce measurement implementation
```

**Acceptance Criteria:**
- [ ] All mortality tables correctly selectable
- [ ] VTD rates accurate from imported data
- [ ] Policy methodology selection working
- [ ] Both IFRS and traditional calculations supported

#### Week 4: Policy Processing Engine
**Priority:** HIGH
**Deliverables:**
- Complete policy data processing
- Batch calculation capabilities
- Reinsurance handling
- TM rate integration

**Tasks:**
```bash
# 4.1 Complete Policy Engine
- Handle all policy types (vitalicia, temporal, diferida)
- Implement proper discount rate selection
- Add reinsurance ceded/retained calculations
- Support multiple payment frequencies

# 4.2 TM Rate Integration
- Import TM historical rates when available
- Implement min(TM,TC) rate selection
- Handle calce measurements for traditional policies
- Generate rate comparison reports

# 4.3 Batch Processing
- Implement parallel policy calculation
- Add progress tracking and error handling
- Optimize for 10,000+ policy batches
- Generate batch calculation reports
```

**Acceptance Criteria:**
- [ ] All policy types supported
- [ ] Batch processing 10K policies < 5 min
- [ ] TM rate comparison working
- [ ] Reinsurance calculations accurate

### Phase 3: Application Development (Week 5-6)

#### Week 5: Command-Line Utility
**Priority:** MEDIUM
**Deliverables:**
- Complete CLI application
- Configuration management
- Input/output handling
- Error handling and validation

**Tasks:**
```bash
# 5.1 CLI Implementation
- Command-line argument parsing
- Policy input from CSV/JSON files
- Output formatting (CSV, JSON, reports)
- Configuration file support

# 5.2 Input/Output Processing
- Validate input data formats
- Generate calculation reports
- Export results in multiple formats
- Handle large input files efficiently

# 5.3 Configuration Management
- Environment variable support
- Configuration file parsing
- Database connection management
- Logging configuration
```

**Acceptance Criteria:**
- [ ] CLI processes policy files correctly
- [ ] Output formats meet regulatory requirements
- [ ] Configuration system flexible
- [ ] Error handling comprehensive

#### Week 6: User Interface (Optional)
**Priority:** LOW
**Deliverables:**
- Terminal User Interface (TUI)
- Interactive policy entry
- Real-time calculation feedback
- Data exploration tools

**Tasks:**
```bash
# 6.1 TUI Development
- Terminal-based user interface
- Interactive policy data entry
- Real-time calculation display
- Data validation and error display

# 6.2 Data Exploration Tools
- Browse mortality tables
- View VTD rate vectors
- Analyze calculation results
- Export data for external analysis
```

**Acceptance Criteria:**
- [ ] TUI responsive and user-friendly
- [ ] Interactive calculations accurate
- [ ] Data exploration tools functional
- [ ] Export capabilities working

### Phase 4: Compliance & Testing (Week 7-8)

#### Week 7: Regulatory Compliance
**Priority:** HIGH
**Deliverables:**
- Complete CMF compliance implementation
- TAP testing functionality
- Asset sufficiency analysis
- Audit trail generation

**Tasks:**
```bash
# 7.1 CMF Compliance Implementation
- IFRS-aligned reserve calculations
- Traditional methodology support
- Proper mortality table application
- Correct reinsurance treatment

# 7.2 TAP Testing Implementation
- Test de Adecuación de Pasivos methodology
- Company estimates vs calculated reserves
- Additional reserve determination
- Quarterly compliance testing support

# 7.3 Audit Trail Generation
- Complete calculation logging
- Regulatory-compliant audit format
- Data source tracking and validation
- CMF reporting format generation
```

**Acceptance Criteria:**
- [ ] All calculations CMF compliant
- [ ] TAP testing implemented correctly
- [ ] Audit trails complete and accurate
- [ ] Regulatory reports generated properly

#### Week 8: Integration & Performance Testing
**Priority:** HIGH
**Deliverables:**
- End-to-end system testing
- Performance optimization
- Production deployment ready
- Documentation completion

**Tasks:**
```bash
# 8.1 Integration Testing
- End-to-end reserve calculations
- Multi-policy batch processing
- Data accuracy validation
- Regulatory compliance verification

# 8.2 Performance Optimization
- Database query optimization
- Parallel processing tuning
- Memory usage optimization
- Calculation speed improvements

# 8.3 Production Readiness
- Production build testing
- Installation script validation
- Performance benchmarking
- Documentation finalization
```

**Acceptance Criteria:**
- [ ] End-to-end calculations accurate
- [ ] Performance targets met
- [ ] Production deployment ready
- [ ] Documentation complete

## Technical Implementation Details

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
│   │   ├── policy.go          # Policy processing
│   │   └── compliance.go      # CMF compliance
│   ├── models/
│   │   ├── policy.go          # Policy data structures
│   │   ├── mortality.go       # Mortality table models
│   │   ├── reserve.go         # Reserve calculation results
│   │   └── vtd.go            # VTD vector models
│   ├── database/
│   │   ├── migrations/        # Schema migrations
│   │   ├── connection.go      # SQLite management
│   │   ├── mortality.go       # Mortality table queries
│   │   ├── vtd.go             # VTD vector queries
│   │   └── policies.go        # Policy data queries
│   ├── cli/
│   │   ├── commands.go        # CLI command definitions
│   │   ├── input.go          # Input processing
│   │   └── output.go         # Output formatting
│   └── config/
│       └── config.go          # Configuration management
├── data/
│   ├── normativo/
│   │   ├── articles-20210_tablas_mort_hist.xlsx
│   │   └── VTD_2025_.xlsx
│   └── migrations/
├── config/
│   └── config.json
├── test/
│   ├── integration/
│   ├── unit/
│   └── performance/
├── docs/
└── scripts/
    ├── build.sh
    ├── test.sh
    └── deploy.sh
```

### Database Schema Implementation
```sql
-- Core tables with proper indexing
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre_estandar VARCHAR(50), -- CMF standard: "CB-H-2020", "RV-M-2020", etc.
    nombre_original VARCHAR(50),   -- Excel name: "CB-2020-HOMBRES", etc.
    sexo CHAR(1),                  -- 'H', 'M', 'A'
    tipo_tabla VARCHAR(20),        -- 'VEJEZ', 'INVALIDEZ', 'SOBREVIVENCIA'
    año_tabla INTEGER,             -- 2004, 2006, 2009, 2014, 2020
    edad INTEGER,
    prob_muerte DECIMAL(10,8),     -- qx value
    factor_aax DECIMAL(10,8),      -- Factor Aax
    vigencia_inicio DATE,           -- Effectivity date
    vigencia_fin DATE,              -- End date if applicable
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(nombre_estandar, edad),
    INDEX idx_nombre_edad (nombre_estandar, edad),
    INDEX idx_vigencia (vigencia_inicio, vigencia_fin)
);

CREATE TABLE vtd_vector (
    id INTEGER PRIMARY KEY,
    year INTEGER,
    month INTEGER,
    period INTEGER,           -- Year 1 to 120
    rate DECIMAL(8,6),       -- Discount rate
    publication_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(year, month, period),
    INDEX idx_year_month (year, month),
    INDEX idx_publication (publication_date)
);

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
    estado VARCHAR(10) DEFAULT 'ACTIVA',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_estado (estado),
    INDEX idx_fecha_inicio (fecha_inicio)
);

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
    FOREIGN KEY (poliza_id) REFERENCES poliza(id),
    INDEX idx_poliza_fecha (poliza_id, fecha_calculo),
    INDEX idx_fecha_calculo (fecha_calculo)
);
```

### Key Implementation Components

#### 1. VPPj Calculation Engine
```go
type VPPJCalculator struct {
    db *sql.DB
    mortality map[string]*MortalityTable
    vtd       map[int]map[int][]VTDPoint // year -> month -> rates
}

func (vc *VPPJCalculator) CalculateReserve(policy Policy) (*ReserveResult, error) {
    // 1. Select appropriate mortality table
    table := vc.selectMortalityTable(policy)
    
    // 2. Get VTD rates for policy date
    rates := vc.getVTDRates(policy.FechaInicio)
    
    // 3. Calculate probabilistic flows
    flows := vc.calculateFlows(policy, table)
    
    // 4. Apply discount rates
    presentValue := vc.applyDiscount(flows, rates, policy.TasaTC)
    
    // 5. Generate audit trail
    audit := vc.generateAudit(policy, table, rates, presentValue)
    
    return &ReserveResult{
        Value: presentValue,
        Audit: audit,
    }, nil
}
```

#### 2. Mortality Table Management
```go
type MortalityManager struct {
    tables map[string]*MortalityTable
    db     *sql.DB
}

func (mm *MortalityManager) SelectTable(policy Policy) (*MortalityTable, error) {
    switch {
    case policy.FechaInicio.After(time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)):
        // Post-2012: Current tables
        return mm.selectCurrentTable(policy)
    default:
        // Pre-2012: Legacy tables with gradual application
        return mm.selectLegacyTable(policy)
    }
}

func (mm *MortalityManager) selectCurrentTable(policy Policy) (*MortalityTable, error) {
    switch {
    case policy.TipoRenta == "VITALICIA" && policy.SexoBeneficiario == "M":
        return mm.tables["RV-M-2020"], nil
    case policy.TipoTabla == "SOBREVIVENCIA":
        if policy.SexoBeneficiario == "M" {
            return mm.tables["B-M-2020"], nil
        }
        return mm.tables["CB-H-2020"], nil
    case policy.TipoRenta == "INVALIDEZ":
        return mm.selectDisabilityTable(policy)
    default:
        return mm.tables["CB-H-2020"], nil
    }
}
```

#### 3. VTD Rate Processing
```go
type VTDManager struct {
    vectors map[int]map[int][]VTDPoint // year -> month -> rates
    db      *sql.DB
}

func (vm *VTDManager) GetRate(year, month, period int) (float64, error) {
    if monthly, ok := vm.vectors[year][month]; ok {
        for _, point := range monthly {
            if point.Period == period {
                return point.Rate, nil
            }
        }
    }
    return 0, errors.New("VTD rate not found")
}

func (vm *VTDManager) GetVector(year, month int) ([]VTDPoint, error) {
    if monthly, ok := vm.vectors[year][month]; ok {
        return monthly, nil
    }
    return nil, errors.New("VTD vector not found")
}
```

## Risk Management

### Technical Risks
1. **Data Accuracy:** Mortality table and VTD data validation
2. **Performance:** Batch processing of 10,000+ policies
3. **Precision:** Decimal arithmetic for financial calculations
4. **Compliance:** CMF regulatory requirement implementation

### Mitigation Strategies
1. **Comprehensive Testing:** Unit, integration, and performance tests
2. **Data Validation:** Multiple validation layers for imported data
3. **Audit Trails:** Complete logging for regulatory compliance
4. **Performance Benchmarking:** Regular performance testing

### Regulatory Risks
1. **Interpretation Errors:** Misunderstanding CMF requirements
2. **Calculation Errors:** Incorrect reserve calculations
3. **Documentation Gaps:** Missing regulatory compliance evidence

### Compliance Assurance
1. **Document Analysis:** Continuous review of CMF documents
2. **Expert Review:** Actuarial validation of calculations
3. **Audit Trails:** Complete audit documentation
4. **Regulatory Testing:** End-to-end compliance verification

## Success Criteria

### Functional Requirements
- [ ] All policy types calculated accurately
- [ ] Both IFRS and traditional methodologies supported
- [ ] Batch processing meets performance targets
- [ ] Regulatory compliance verified

### Performance Requirements
- [ ] Single policy calculation < 100ms
- [ ] 10,000 policy batch < 5 minutes
- [ ] Memory usage < 1GB for 100,000 policies
- [ ] Database queries optimized with proper indexing

### Compliance Requirements
- [ ] CMF regulatory requirements fully implemented
- [ ] Audit trails complete and accurate
- [ ] TAP testing functionality operational
- [ ] Asset sufficiency analysis capabilities

## Next Steps

### Immediate (This Week)
1. **Environment Setup:** Install Go, SQLite, development tools
2. **Database Schema:** Implement complete SQLite schema
3. **Mortality Import:** Parse and load Excel mortality tables
4. **Basic Engine:** Implement VPPj formula with decimal precision

### Week 1-2 Deliverables
1. **Database Foundation:** Complete schema with data imports
2. **Calculation Engine:** Basic VPPj calculations working
3. **Testing Framework:** Unit tests for core functionality
4. **Performance Baseline:** Single policy calculation benchmarks

### Long-term Success
1. **Production Deployment:** CLI application ready for use
2. **Regulatory Approval:** CMF compliance verification complete
3. **User Training:** Documentation and training materials
4. **Maintenance Plan:** Ongoing support and update procedures

This implementation plan provides a comprehensive roadmap for building a CMF-compliant reserves calculator with clear milestones, technical details, and risk management strategies.