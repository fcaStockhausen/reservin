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

## Key Information Extracted

### Current Mortality Tables (from circular_2332)
- **CB-H-2020** - Básica Chile (same table for Causante/Beneficiario hombres)
- **RV-M-2020** - Rentas Vitalicias Mujeres
- **B-M-2020** - Básica Mujeres (Beneficiario)
- **MI-H-2020** - Mortalidad Invalidez Hombres
- **MI-M-2020** - Mortalidad Invalidez Mujeres

### Traditional Methodology (from circular_1512)
- Pre-2012 reserve calculation methods
- TM (tasa venta) calculations
- Calce measurement procedures
- Reinsurance reserve calculations

### TM Rate Data (from tm_ofc_1388_2025)
- **Current TM Rate:** 2.41% (for September 2025)
- **Reference:** Circular N°1512 methodology
- **Application:** For TM vs TC rate comparison (min(TM,TC))

### Missing Critical Data
- **Monthly VTD Publications** - Indicatriz vector for discount rates
- **Historical TM Rates** - For calce calculations and comparisons
- **VTD Indicatriz Vector** - Currently using [1,1,1,...] fallback

## Quick Reference Guide

### Document Selection by Need

| Development Need | Primary Document | Secondary Documents |
|-----------------|------------------|--------------------|
| **Current Mortality Tables** | circular_2332 | Excel mortality tables |
| **Pre-2012 Calculations** | circular_1512 | Historical TM data |
| **IFRS Framework** | ncg_318_2011 | Technical specifications |
| **Asset Testing** | ncg_209_2007 | Implementation guide |
| **Transitional Rules** | ncg_374_2015 | Technical specifications |
| **TM Rate Data** | tm_ofc_1388_2025 | Historical TM data |
| **Discount Rates** | VTD publications (missing) | Technical specifications |

### Context Switching

When switching between different regulatory requirements during development:

1. **Post-2012 Policies** → Use ncg_318_2011 framework + circular_2332 tables
2. **Pre-2012 Policies** → Use circular_1512 methodology + legacy tables
3. **2015-2020 Policies** → Use ncg_374_2015 transitional rules
4. **Asset Testing** → Use ncg_209_2007 methodology
5. **TM Rate Selection** → Use tm_ofc_1388_2025 data + historical TM

## Usage Guidelines

### For Development
1. **Reference summaries** for quick understanding of regulatory requirements
2. **Read full PDFs** for detailed implementation specifics
3. **Cross-reference** with technical specifications for implementation guidance

### For Regulatory Compliance
1. **Review circular_2332** for current mortality table requirements
2. **Review circular_1512** for traditional methodology compliance
3. **Review ncg_318_2011** for IFRS alignment requirements
4. **Review ncg_209_2007** for asset sufficiency testing

### For Implementation Validation
1. **Compare implementations** with PDF requirements
2. **Validate calculations** against documented formulas
3. **Check compliance** with all applicable regulatory frameworks

## Document Status

### ✅ Complete
- All PDF documents available
- Summaries generated for 5 out of 6 documents
- Key regulatory terms extracted and indexed

### ❌ Missing Items
- ncg_374_2015_summary.md (PDF extraction failed - need manual review)
- Monthly VTD publications (vector tasa de descuento)
- Historical TM rate data beyond Sept 2025

### 🔧 Maintenance
- Update summaries when new CMF publications are received
- Regenerate summaries when PDFs are updated
- Add new regulatory documents to this folder

This normative documentation provides comprehensive regulatory coverage with quick-reference summaries for efficient development context switching.