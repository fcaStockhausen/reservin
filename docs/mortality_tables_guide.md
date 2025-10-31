# Mortality Tables Guide

## Overview

Chilean CMF provides specific mortality tables for calculating life annuity reserves. These tables are fundamental to accurate reserve calculations and must be properly loaded and applied according to policy characteristics and effective dates.

## Available Mortality Tables

### Current Tables (Circular N°2332)

| Table Code | Description | Gender | Applicable For | Year |
|------------|-------------|--------|----------------|------|
| **CB-H-2020** | Current Básica Chile Hombres | Male | General population | 2020 |
| **RV-M-2020** | Rentas Vitalicias Mujeres | Female | Life annuities | 2020 |
| **B-M-2020** | Básica Mujeres | Female | General population | 2020 |
| **MI-H-2020** | Mortalidad Invalidez Hombres | Male | Disability/survival | 2020 |
| **MI-M-2020** | Mortalidad Invalidez Mujeres | Female | Disability/survival | 2020 |

### Legacy Tables (Pre-2012 Policies)

| Table Code | Description | Gender | Application |
|------------|-------------|--------|-------------|
| **B-2006** | Básica 2006 | Both | Gradual application for pre-2012 policies |
| **MI-2006** | Mortalidad Invalidez 2006 | Both | Gradual application for pre-2012 policies |

## Data Source Location

```
data/normativo/articles-20210_tablas_mort_hist.xlsx
```

This file contains historical mortality data that needs to be processed and loaded into the database structure.

## Table Application Rules

### Policy Effective Date Rules

1. **Post-2012 Policies (Jan 1, 2012 +)**: Use current tables (CB-H-2020, RV-M-2020 series)
2. **Pre-2012 Policies**: Use legacy tables with gradual transition
3. **Gender Selection**: 
   - Male beneficiaries: CB-H-2020, MI-H-2020
   - Female beneficiaries: RV-M-2020, B-M-2020, MI-M-2020
   - Gender-neutral: Use combined tables where applicable

### Insurance Type Mapping

| Insurance Type | Recommended Table(s) | Notes |
|----------------|---------------------|-------|
| **Rentas Vitalicias** | RV-M-2020 (female), CB-H-2020 (male) | Primary for life annuities |
| **Invalidez y Sobrevivencia** | MI-H-2020, MI-M-2020 | Disability/survival policies |
| **General Population** | CB-H-2020, B-M-2020 | Base mortality for calculations |

## Database Schema

```sql
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre VARCHAR(50), -- "CB-H-2020", "RV-M-2020", etc.
    sexo CHAR(1), -- 'H', 'M', 'A' (ambos)
    edad INTEGER, -- Age from 0 to 120+
    prob_muerte DECIMAL(10,8), -- Death probability at age x
    prob_supervivencia DECIMAL(10,8), -- Survival probability from birth
    prob_supervivencia_anual DECIMAL(10,8), -- Annual survival probability
    anos_calculo INTEGER, -- Calendar year of calculation
    fuente VARCHAR(20) DEFAULT 'CMF',
    vigencia_inicio DATE, -- Effectivity start date
    vigencia_fin DATE, -- Effectivity end date
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for performance
CREATE INDEX idx_mortalidad_nombre_edad ON tabla_mortalidad(nombre, edad);
CREATE INDEX idx_mortalidad_sexo_edad ON tabla_mortalidad(sexo, edad);
```

## Data Loading Process

### Step 1: Extract from Excel
```python
import pandas as pd

def load_mortality_table(file_path, sheet_name, table_name, gender):
    df = pd.read_excel(file_path, sheet_name=sheet_name)
    # Process age columns and mortality rates
    for age in range(0, 121):
        prob_death = df.loc[df['edad'] == age, 'prob_muerte'].iloc[0]
        prob_survival = df.loc[df['edad'] == age, 'prob_supervivencia'].iloc[0]
        # Insert into database
```

### Step 2: Validate Data Quality
- **Age Coverage**: Ensure 0-120+ years covered
- **Probability Range**: Death + Survival = 1.0 for each age
- **Monotonicity**: Death probabilities generally increase with age
- **Missing Values**: Handle gaps in data appropriately

### Step 3: Apply Business Rules
```sql
-- Example: Select appropriate table based on policy
SELECT tm.*, p.fecha_inicio, p.sexo_beneficiario
FROM tabla_mortalidad tm
JOIN poliza p ON (
    (tm.sexo = p.sexo_beneficiario OR tm.sexo = 'A')
    AND p.fecha_inicio >= tm.vigencia_inicio 
    AND p.fecha_inicio <= COALESCE(tm.vigencia_fin, '9999-12-31')
)
WHERE tm.nombre = CASE 
    WHEN p.fecha_inicio >= '2012-01-01' THEN
        CASE p.sexo_beneficiario
            WHEN 'F' THEN 'RV-M-2020'
            WHEN 'M' THEN 'CB-H-2020'
        END
    ELSE
        CASE p.sexo_beneficiario
            WHEN 'F' THEN 'B-2006'
            WHEN 'M' THEN 'B-2006'
        END
END
AND tm.edad = p.edad_contratante;
```

## Calculation Usage Examples

### Life Annuity Present Value
```go
func (r *ReserveCalculator) CalculateAnnuityPV(policy Policy) float64 {
    var pv float64
    table := r.selectMortalityTable(policy)
    
    for age := policy.EdadContratante; age <= 120; age++ {
        survivalProb := table.getSurvivalProbability(age)
        discountFactor := math.Pow(1+r.DiscountRate, float64(age-policy.EdadContratante))
        pv += policy.MontoRenta * survivalProb / discountFactor
    }
    
    return pv
}
```

### Annual Survival Probability
```latex
p_x = \frac{l_{x+1}}{l_x} = 1 - q_x
```

### Cumulative Survival Probability
```latex
{}_t p_x = \frac{l_{x+t}}{l_x} = p_x \times p_{x+1} \times \cdots \times p_{x+t-1}
```

## Quality Control

### Validation Checks
1. **Completeness**: All ages 0-120 present
2. **Consistency**: Probabilities sum to 1.0
3. **Monotonicity**: Death rates increase with age (after infancy)
4. **Regulatory Compliance**: Match CMF published tables exactly

### Data Integrity Monitoring
```sql
-- Check for data anomalies
SELECT 
    nombre,
    COUNT(*) AS total_registros,
    MIN(edad) AS edad_minima,
    MAX(edad) AS edad_maxima,
    SUM(prob_muerte) AS suma_muerte,
    SUM(prob_supervivencia) AS suma_supervivencia
FROM tabla_mortalidad
GROUP BY nombre
HAVING COUNT(*) != 121 OR MIN(edad) != 0 OR MAX(edad) != 120;
```

## Maintenance Process

### Annual Updates
1. **CMF Publications**: Monitor for new mortality tables
2. **Backward Compatibility**: Maintain legacy tables for existing policies
3. **Testing**: Validate against CMF reference calculations
4. **Documentation**: Update application rules for new tables

### Version Control
- **Table Versions**: Track effective dates for each table version
- **Policy Mapping**: Maintain mapping from policy dates to table versions
- **Audit Trail**: Log table changes and their impact on reserves

## Integration with Reserve Calculation

The mortality tables feed directly into:

1. **Policy Flow Calculations**: Survival probabilities determine payment streams
2. **Present Value Calculations**: Age-specific rates drive discounting
3. **TAP Testing**: Company-specific mortality experience vs. CMF tables
4. **Scenario Analysis**: Sensitivity testing using alternative mortality assumptions

Proper implementation of these tables is essential for regulatory compliance and accurate reserve calculations.