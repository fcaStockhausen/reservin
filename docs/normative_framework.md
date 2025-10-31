# Normative Framework - CMF Regulatory Requirements

## Regulatory Foundation

**Primary Authority:** Comisión para el Mercado Financiero (CMF) - Chile
**Scope:** Reservas de Rentas Vitalicias calculation according to Chilean insurance law
**Key Documents:** NCG 318 (2011) with subsequent modifications

## Core Regulatory Requirements

### 1. NCG 318 (2011) - Primary Framework

**Status:** ✅ Complete analysis available
**Scope:** Instructions for IFRS application to technical reserves
**Effective:** January 1, 2012

#### Key Requirements:

**Reserve Calculation Methods by Policy Date:**
- **December 1, 2020+**: Use Annex rates, no calce measurement
- **June 1, 2015+**: Use NCG N°374 Annex rates  
- **January 1, 2012+**: IFRS-aligned method (most important)
- **Pre-January 1, 2012**: Traditional method (Circular N°1512)

**Post-2012 Methodology Changes:**
- No calce measurement in discount rate calculation
- No reserve adjustment by calce - only base technical reserve
- Discount rate = minimum between TM (tasa venta) and TC (tasa costo)
- Reinsurance: Calculate 100% commitments, ceded flows as assets
- Mortality tables: Apply CB-H-2020, RV-M-2020, B-M-2020, MI-H-2020, MI-M-2020

#### Mathematical Framework:
```latex
VPP_j = \sum_{i=1}^{n} FP_{ji} \times (1 + TC_j)^{-i}
```
```latex
VTD = ET + AV
```

### 2. Critical Missing Documents

#### Circular N°1512 (2001) - MISSING
**Impact:** CRITICAL - Cannot calculate pre-2012 policies
**Required For:**
- Traditional reserve calculation methodology
- TM (tasa venta) calculation formulas
- Calce measurement methodology
- Financial reserve calculations
- Reinsurance reserve calculation (retained vs ceded)

**Key Sections Needed:**
- Título III: Cálculo de TM (tasa de venta)
- Título IV: Medición de calce
- Título V: Reservas con reaseguro

#### Monthly VTD Rate Data - MISSING
**Impact:** CRITICAL - Cannot perform any discount rate calculations
**Required For:**
- All reserve calculations (VTD = ET + AV)
- Discount rate determination: min(TM, TC)
- Monthly CMF publications via Oficio Ordinario
- ET (estructura temporal) + AV (ajuste por volatilidad) components

#### TM Historical Rates - MISSING
**Impact:** HIGH - Cannot determine optimal discount rates
**Required For:**
- Rate comparison calculations (min(TM, TC))
- Calce measurement methodology
- Historical policy valuations

### 3. Secondary Regulatory Documents

#### Circular N°2332 - MISSING
**Impact:** HIGH - Risk of non-compliance with current tables
**Required For:**
- Official current mortality table specifications
- CB-H-2020, RV-M-2020, B-M-2020, MI-H-2020, MI-M-2020 validation
- Application rules and effectivity dates

#### NCG N°374 (2015) - MISSING
**Impact:** MEDIUM - Policy calculation gaps 2015-2020
**Required For:**
- Discount rate methodology for policies 2015-2020
- Annex with transitional calculation rules
- Bridge between old and new methodologies

#### NCG N°209 (2007) - MISSING
**Impact:** MEDIUM - Cannot perform required testing
**Required For:**
- Asset sufficiency analysis methodology
- Additional reserve calculation rules
- Regulatory compliance testing requirements

## Mortality Table Requirements

### Current Tables (Circular N°2332)
**Required Implementation:**
- **CB-H-2020** - Current Básica Chile (Causante/Beneficiario same table for hombres)
- **RV-M-2020** - Rentas Vitalicias Mujeres
- **B-M-2020** - Básica Mujeres (Beneficiario)
- **MI-H-2020** - Mortalidad Invalidez Hombres (Causante)
- **MI-M-2020** - Mortalidad Invalidez Mujeres (Beneficiario)

### Legacy Tables (Pre-2012)
**Required Implementation:**
- **B-2006** - Gradual application for existing policies (Beneficiario)
- **MI-2006** - Gradual application for disability policies

### Data Availability
**Status:** ✅ Complete Excel file available
**Source:** `data/normativo/articles-20210_tablas_mort_hist.xlsx`
**Coverage:** 18 tables across 6 sheets, ages 0-110, qx values and factors

## Policy Application Rules

### Mortality Table Selection

**Post-January 1, 2012 Policies:**
```
Female beneficiaries → RV-M-2020 (life annuity) or B-M-2020 (survival - Beneficiario)
Male beneficiaries → CB-H-2020 (basic/survival - Causante)
Disability policies → MI-H-2020 / MI-M-2020 (by gender)
```

**Pre-January 1, 2012 Policies:**
```
Gradual application of B-2006 / MI-2006 tables
Voluntary adoption of current methodology permitted
Traditional Circular N°1512 calculations required
```

### Discount Rate Application

**IFRS-aligned (Post-2012):**
```
Discount rate = min(TM_j, TC_j)
TM_j = Tasa venta (Circular N°1512 Título III)
TC_j = Tasa costo (calculated via VPPj formula)
```

**Traditional (Pre-2012):**
```
Includes calce measurement
Financial reserve calculations
Traditional rate selection methodology
```

## Compliance Testing Requirements

### 1. Test de Adecuación de Pasivos (TAP)

**Frequency:** Quarterly financial statement closure
**Methodology:** Use company's own mortality and interest rate estimates
**Purpose:** Evaluate technical reserve sufficiency
**Requirements:**
- Consider policyholder options and guarantees
- Recognize reinsurance as assets when additional reserves needed
- Technical validation by external auditors

**Formula:**
```latex
BEL = \sum_{i=1}^{n} CF_i \times e^{-r_i \times t_i} \times S_i
```

### 2. Análisis de Suficiencia de Activos

**Framework:** NCG N°209 methodology
**Purpose:** Asset sufficiency analysis
**Requirements:**
- Additional reserve constitution when required
- Separate from TAP but complementary
- Regulatory compliance reporting

### 3. Reinsurance Treatment

**Post-2012 Policies:**
```
Calculate 100% of commitments
Ceded flows recognized as assets
Subject to IFRS impairment testing
Prima differences recognized immediately in results
```

**Pre-2012 Policies:**
```
Traditional retained/ceded reserve calculation
Bruto presentation in financial statements
Ceded portion as reinsurance asset
```

## Documentation Mapping

### Source of Truth References

| Category | Document | Status | Criticality |
|----------|----------|---------|------------|
| **Primary Framework** | NCG 318 (2011) | ✅ Complete | CRITICAL |
| **Current Mortality** | Articles Excel file | ✅ Complete | HIGH |
| **Legacy Methodology** | Circular N°1512 (2001) | ❌ Missing | CRITICAL |
| **Current Table Specs** | Circular N°2332 | ❌ Missing | HIGH |
| **Discount Data** | Monthly VTD publications | ❌ Missing | CRITICAL |
| **TM Rate Data** | TM historical rates | ❌ Missing | HIGH |
| **Transitional Rules** | NCG N°374 (2015) | ❌ Missing | MEDIUM |
| **Asset Testing** | NCG N°209 (2007) | ❌ Missing | MEDIUM |

## Implementation Compliance Strategy

### Phase 1: Critical Compliance
1. **Obtain VTD Data** - Required for all discount rate calculations
2. **Get TM Historical Rates** - Needed for rate comparisons
3. **Process Available Documents** - Extract rate data from normative folder
4. **Extract TM Data** - From TM Oficio 1388/2025 publication

### Phase 2: Current Standards
1. **Review Circular N°2332** - Validate current table implementations
2. **Implement Current Mortality Tables** - From Excel data
3. **Build IFRS-aligned Calculations** - Post-2012 methodology

### Phase 3: Complete Compliance
1. **Review NCG N°374** - Transitional period calculations
2. **Implement NCG N°209** - Asset sufficiency testing
3. **Complete TAP Implementation** - Quarterly compliance testing

## Regulatory Contact Strategy

### Primary Sources
- **CMF Website:** www.cmf.cl (Normativa section)
- **Direct Contact:** normativa@cmf.cl
- **Physical Office:** Teatinos 280, Santiago

### Industry Sources
- **AChI (Chilean Insurance Association):** Industry guidance
- **Actuarial Professional Networks:** Document sharing
- **Insurance Companies:** Internal regulatory compliance teams

### Acquisition Priority
1. **Week 1:** Circular N°1512 and VTD data (critical blockers)
2. **Week 2:** Circular N°2332 and TM rates (current implementation)
3. **Week 3:** Secondary documents (complete compliance)

## Documentation Status Update

### ✅ Documents Available in normative/ folder
- **circular_1512.pdf** - Traditional pre-2012 methodology
- **circular_2332.pdf** - Current mortality table specifications  
- **ncg_209_2007.pdf** - Asset sufficiency analysis
- **ncg_318_2011.pdf** - Primary framework (already analyzed)
- **ncg_374_2015.pdf** - Transitional 2015-2020 rules
- **tm_ofc_1388_2025.pdf** - TM rate data (latest)

### ❌ Still Missing
- **Monthly VTD Publications** - Vector tasa de descuento (critical for discount rates)
- **Historical TM Rates** - Beyond 2025 publication (needed for calce)

This normative framework provides complete regulatory analysis with clear documentation status and implementation strategy for full CMF compliance.
