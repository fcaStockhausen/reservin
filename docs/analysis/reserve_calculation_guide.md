# Reserve Calculation Guide - NCG 318

## Regulatory Framework Overview

This document summarizes the key requirements for calculating **Reservas de Rentas Vitalicias** according to Chile's CMF NCG 318 (2011) and subsequent modifications.

## Reserve Calculation Methods by Policy Date

### 1. Policies from December 1, 2020
- Use discount rate from Annex to this norm
- No calce measurement in discount rate calculation
- Only base technical reserve (no financial reserve)

### 2. Policies from June 1, 2015  
- Use discount rate from NCG N°374 Annex
- Same methodology as post-2020 policies

### 3. Policies from January 1, 2012 (IFRS-aligned)
**Key Changes:**
- No calce measurement in discount rate determination
- No reserve adjustment by calce - only base technical reserve
- **Discount Rate**: Minimum between TM (tasa venta) and TC (tasa costo)
- **Reinsurance**: Calculate 100% of commitments, ceded flows as assets
- **Mortality Tables**: Apply CB-H-2020, RV-M-2020, B-M-2020, MI-H-2020, MI-M-2020

### 4. Policies before January 1, 2012
- Traditional method using Circular N°1512
- Gradual application of B-2006 and MI-2006 tables
- Optional voluntary adoption of post-2012 methodology

## Core Mathematical Formulas

### Present Value Calculation
```latex
VPP_j = \sum_{i=1}^{n} FP_{ji} \times (1 + TC_j)^{-i}
```

Where:
- $VPP_j$ = Present value of policy j at inception
- $FP_{ji}$ = Probabilistic flow of policy j in period i (no reinsurance deductions)
- $TC_j$ = IRR/tasa costo equivalente (implicit in the policy)
- $n$ = Number of periods until policy termination

### Discount Rate Vector (VTD)
```latex
VTD = ET + AV
```

Where:
$ET$ = Real risk-free term structure (120 years)
- **Segmento Líquido (0-25 años)**: State/Central Bank market transactions
- **Segmento Extrapolado (25-120 años)**: Smith-Wilson method, converges at 65 years to TILP

$AV$ = Volatility Adjustment 
```latex
AV = 0.65 \times (R_{exceso} - R_{libre\_riesgo}) - 0.35 \times SRC
```

## Required Mortality Tables

### Current Tables (Circular N°2332)
- **CB-H-2020** - Current Básica Chile Hombres
- **RV-M-2020** - Rentas Vitalicias Mujeres  
- **B-M-2020** - Básica Mujeres
- **MI-H-2020** - Mortalidad Invalidez Hombres
- **MI-M-2020** - Mortalidad Invalidez Mujeres

### Legacy Tables (for pre-2012 policies)
- **B-2006** - Gradual application
- **MI-2006** - Gradual application

## Key Regulatory Requirements

### 1. Test de Adecuación de Pasivos (TAP)
- **Frequency**: Quarterly financial statement closure
- **Methodology**: Use company's own mortality and interest rate estimates
- **Purpose**: Evaluate technical reserve sufficiency
- **Requirements**: 
  - Consider policyholder options and guarantees
  - Recognize reinsurance as assets when additional reserves needed
  - Technical validation by external auditors

### 2. Análisis de Suficiencia de Activos
- Continue using NCG N°209 methodology
- Constituir technical reserve additional when required
- Separate from TAP but complementary

### 3. Reinsurance Treatment
- **Post-2012 policies**: Calculate 100% of commitments, ceded flows as assets
- **Pre-2012 policies**: Traditional retained/ceded reserve calculation
- **Asset Recognition**: Subject to IFRS impairment testing
- **Prima Differences**: Recognize immediately in results if differences exist

## Implementation Requirements

### Data Structure
```sql
-- Mortality Tables (from data/normativo/)
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre VARCHAR(50), -- "CB-H-2020", "RV-M-2020", etc.
    sexo CHAR(1), -- 'H', 'M', 'A' (ambos)
    edad INTEGER,
    prob_muerte DECIMAL(10,8),
    prob_supervivencia DECIMAL(10,8),
    anos_calculo INTEGER,
    fuente VARCHAR(20) DEFAULT 'CMF'
);

-- Discount Rate Vectors (monthly CMF publication)
CREATE TABLE vtd_mensual (
    id INTEGER PRIMARY KEY,
    fecha_publicacion DATE,
    anio INTEGER,
    tasa_liquida DECIMAL(8,6),
    tasa_extrapolada DECIMAL(8,6),
    ajuste_volatilidad DECIMAL(8,6),
    tasa_final DECIMAL(8,6)
);
```

### Calculation Workflow
1. **Load mortality tables** from `data/normativo/articles-20210_tablas_mort_hist.xlsx`
2. **Import monthly VTD** from CMF publications
3. **Calculate policy flows** using appropriate mortality tables
4. **Determine discount rate** (minimum TM vs TC)
5. **Apply VPPj formula** for present value calculation
6. **Perform TAP testing** quarterly for adequacy validation

## Technology Implementation Notes

### Go Components Required
```go
// Core calculation structures
type MortalityTable struct {
    Name    string
    Sex     string
    Rates   map[int]float64  // age -> survival probability
}

type VTDVector struct {
    LiquidRates     []float64  // 0-25 years
    Extrapolated    []float64  // 26-120 years  
    VolatilityAdj   []float64  // AV adjustment
}

type ReserveCalculator struct {
    MortalityTables map[string]MortalityTable
    DiscountVectors map[string]VTDVector
    Rates          map[string]map[string]float64
}

func (r *ReserveCalculator) CalculateVPPj(policy Policy) float64 {
    // Implement: Σ FPji × (1 + TCj)^-i
}
```

## Key Documentation References

- **NCG 318 (2011)**: Base normative for reserve calculations
- **Circular N°1512 (2001)**: Traditional calculation methodology
- **Circular N°2332**: Current mortality table specifications  
- **NCG N°374 (2015)**: Discount rate modifications
- **NCG N°209 (2007)**: Asset sufficiency analysis

## Monthly Process Requirements

1. **Import VTD Rates**: Monthly CMF publication via Ordinary Office
2. **Update Reference Portfolio**: CMF updates for AV and SRC calculations
3. **Apply Monthly Rates**: Convert annual to monthly using formula:
   ```
   i_mensual = (1 + i_anual)^(1/12) - 1
   ```
4. **TAP Testing**: Quarterly using company's own estimates
5. **Asset Sufficiency**: Ongoing NCG N°209 compliance

This framework provides the complete regulatory foundation for implementing the reserve calculation system according to current CMF requirements.