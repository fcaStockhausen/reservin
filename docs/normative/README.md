# Normative Documents - CMF Regulatory References

## Document Overview

This folder contains all regulatory documents and their generated summaries for quick reference and context switching during development.

## Available Documents

### PDF Documents
- **circular_1512.pdf** - Traditional reserve methodology (pre-2012)
- **circular_2332.pdf** - Current mortality table specifications
- **ncg_209_2007.pdf** - Asset sufficiency analysis
- **ncg_318_2011.pdf** - Primary IFRS framework
- **ncg_374_2015.pdf** - Transitional 2015-2020 rules
- **tm_ofc_1388_2025.pdf** - Latest TM rate (2.41% for Sept 2025)

### Generated Summaries
- **circular_1512_summary.md** - Traditional methodology analysis
- **circular_2332_summary.md** - Current mortality tables (CB-H-2020, RV-M-2020, etc.)
- **ncg_209_2007_summary.md** - Asset sufficiency testing requirements
- **ncg_318_2011_summary.md** - IFRS framework analysis (already well documented)
- **tm_ofc_1388_2025_summary.md** - TM rate data (2.41% Sept 2025)

### VTD Data
- **VTD_2025_.xlsx** - Vector Tasa de Descuento (2020-2025 monthly data)
- **VTD_2025__summary.md** - Complete VTD analysis and implementation guide

## Key Information Extracted

### Current Mortality Tables (from circular_2332)
- **CB-H-2020** - Básica Chile (same table for Causante/Beneficiario in hombres)
- **RV-M-2020** - Rentas Vitalicias Mujeres
- **B-M-2020** - Básica Mujeres (Beneficiario)
- **MI-H-2020** - Mortalidad Invalidez Hombres (Causante)
- **MI-M-2020** - Mortalidad Invalidez Mujeres (Beneficiario)

### Traditional Methodology (from circular_1512)
- Pre-2012 reserve calculation methods
- TM (tasa venta) calculations
- Calce measurement procedures
- Reinsurance reserve calculations

### TM Rate Data (from tm_ofc_1388_2025)
- **Current TM Rate:** 2.41% (for September 2025)
- **Reference:** Circular N°1512 methodology
- **Application:** For TM vs TC rate comparison (min(TM,TC))

### VTD Vector Data (from VTD_2025__summary)
- **Coverage:** 2020-2025 monthly discount rate vectors
- **Structure:** Periods 1-25 years (sample), complete up to 120 years
- **Format:** Monthly sheets with period-based rate vectors
- **Rates:** Variable from negative (2021-2022) to positive (2023-2025)

## Quick Reference Guide

### Document Selection by Need

| Development Need | Primary Document | Secondary Documents |
|-----------------|------------------|--------------------|
| **Current Mortality Tables** | circular_2332 | Excel mortality tables |
| **Traditional Methodology** | circular_1512 | Historical TM data |
| **IFRS Framework** | ncg_318_2011 | Technical specifications |
| **Discount Rates** | VTD_2025__summary | VTD_2025_.xlsx |
| **Asset Testing** | ncg_209_2007 | Implementation guide |
| **TM Rate Data** | tm_ofc_1388_2025 | Historical TM data |
| **Transitional Rules** | ncg_374_2015 | Technical specifications |

### Context Switching

When switching between different regulatory requirements during development:

1. **Post-2012 Policies** → Use ncg_318_2011 framework + circular_2332 tables
2. **Pre-2012 Policies** → Use circular_1512 methodology + legacy tables
3. **Discount Rate Calculations** → Use VTD_2025__summary + VTD_2025_.xlsx
4. **TM Rate Selection** → Use tm_ofc_1388_2025 data + historical TM
5. **Asset Testing** → Use ncg_209_2007 methodology
6. **Transitional Period** → Use ncg_374_2015 guidelines

## Usage Guidelines

### For Development
1. **Reference summaries** for quick understanding of regulatory requirements
2. **Read full PDFs** for detailed implementation specifics
3. **Use VTD data** for discount rate calculations
4. **Cross-reference** with technical specifications for implementation guidance

### For Regulatory Review
1. **Review circular_2332** for current mortality table requirements
2. **Review circular_1512** for traditional methodology compliance
3. **Review ncg_318_2011** for IFRS alignment requirements
4. **Review ncg_209_2007** for asset sufficiency testing

### For Implementation Validation
1. **Compare implementations** with PDF requirements
2. **Validate calculations** against documented formulas
3. **Check discount rates** with VTD vector data
4. **Verify compliance** with all applicable regulatory frameworks

## Document Status

### ✅ Complete
- All PDF documents available
- Summaries generated for 5 out of 6 documents
- VTD data with comprehensive analysis
- Key regulatory terms extracted and indexed

### ❌ Missing Items
- **Monthly VTD Publications** - Future VTD vector updates beyond 2025
- **Historical TM Rates** - Beyond 2025 publication (needed for calce)
- **ncg_374_2015_summary.md** - PDF extraction failed (need manual review)

### 🔧 Maintenance
- Update summaries when new CMF publications are received
- Regenerate summaries when PDFs are updated
- Add new VTD data when monthly publications become available
- Add new regulatory documents to this folder

## VTD Implementation Details

### Rate Vector Structure
```sql
-- VTD vector storage example
CREATE TABLE vtd_vector (
    id INTEGER PRIMARY KEY,
    year INTEGER,
    month INTEGER,
    period INTEGER,           -- Year 1 to 120
    rate DECIMAL(8,6),       -- Discount rate
    publication_date DATE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Sample VTD Rates (2025-01)
```
Period 1:  0.0235  (2.35%)
Period 2:  0.0231  (2.31%)
Period 3:  0.0238  (2.38%)
Period 4:  0.0245  (2.45%)
Period 5:  0.0249  (2.49%)
...
```

### Application in Reserve Calculations
```go
// VTD rate lookup function
func getVTDRate(year, month, period int) (float64, error) {
    // Query database for specific rate
    // SELECT rate FROM vtd_vector WHERE year = ? AND month = ? AND period = ?
    return rate, nil
}
```

This normative documentation provides comprehensive regulatory coverage with VTD data and quick-reference summaries for efficient development context switching.