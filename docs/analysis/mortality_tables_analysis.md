# Mortality Tables Analysis & Database Migration Architecture

## Excel File Structure Analysis

### File Overview: `articles-20210_tablas_mort_hist.xlsx`

**Total Sheets: 6**
- **Vejez-Mujeres**: Life annuity tables for women
- **Invalidez-Mujeres**: Disability tables for women  
- **Sobrevivencia-Mujeres**: Survival tables for women
- **Vejez-Hombres**: Life annuity tables for men
- **Invalidez-Hombres**: Disability tables for men
- **Sobrevivencia-Hombres**: Survival tables for men

### Table Structure by Sheet

Each sheet contains **multiple tables in columns**:

#### Vejez-Mujeres (Life Annuity - Women)
- **RV-2004-MUJERES** (cols 0-3): Ages 20-110, qx 2004, Factor Aax
- **RV-2009-MUJERES** (cols 4-7): Ages 20-110, qx 2009, Factor Aax
- **RV-2014-MUJERES** (cols 8-11): Ages 20-110, qx 2014, Factor Aax
- **RV-2020-MUJERES** (cols 12-29): Ages 20-110, qx 2020, Factors de mejoramiento AAx,t

#### Invalidez-Mujeres (Disability - Women)
- **MI-2006-MUJERES** (cols 0-3): Ages 0-110, qx 2006, Factor Aax
- **MI-2014-MUJERES** (cols 4-7): Ages 0-110, qx 2014, Factor Aax
- **MI-2020-MUJERES** (cols 8-25): Ages 0-110, qx 2020, Factors de mejoramiento AAx,t

#### Sobrevivencia-Mujeres (Survival - Women)
- **B-2006-MUJERES** (cols 0-3): Ages 0-110, qx 2006, Factor Aax
- **B-2014-MUJERES** (cols 4-7): Ages 0-110, qx 2014, Factor Aax
- **B-2020-MUJERES** (cols 8-25): Ages 0-110, qx 2020, Factors de mejoramiento AAx,t

#### Vejez-Hombres (Life Annuity - Men)
- **RV-2004-HOMBRES** (cols 0-3): Ages 20-110, qx 2004, Factor Aax
- **RV-2009-HOMBRES** (cols 4-7): Ages 20-110, qx 2009, Factor Aax
- **CB-2014-HOMBRES** (cols 8-11): Ages 0-110, qx 2014, Factor Aax
- **CB-2020-HOMBRES** (cols 12-29): Ages 0-110, qx 2020, Factors de mejoramiento AAx,t

#### Invalidez-Hombres (Disability - Men)
- **MI-2006-HOMBRES** (cols 0-3): Ages 0-110, qx 2006, Factor Aax
- **MI-2014-HOMBRES** (cols 4-7): Ages 0-110, qx 2014, Factor Aax
- **MI-2020-HOMBRES** (cols 8-25): Ages 0-110, qx 2020, Factors de mejoramiento AAx,t

#### Sobrevivencia-Hombres (Survival - Men)
- **B-2006-HOMBRES** (cols 0-3): Ages 0-110, qx 2006, Factor Aax
- **CB-2014-HOMBRES** (cols 4-7): Ages 0-110, qx 2014, Factor Aax
- **CB-2020-HOMBRES** (cols 8-25): Ages 0-110, qx 2020, Factors de mejoramiento AAx,t

## Table Naming Mapping Issues

### Expected vs Actual Names

**Expected CMF Standard Names:**
- CB-H-2020 (Current Básica Chile Hombres)
- RV-M-2020 (Rentas Vitalicias Mujeres)  
- B-M-2020 (Básica Mujeres)
- MI-H-2020 (Mortalidad Invalidez Hombres)
- MI-M-2020 (Mortalidad Invalidez Mujeres)

**Actual Names Found:**
- CB-2020-HOMBRES → Should map to CB-H-2020
- RV-2020-MUJERES → Should map to RV-M-2020
- B-2020-MUJERES → Should map to B-M-2020
- MI-2020-HOMBRES → Should map to MI-H-2020
- MI-2020-MUJERES → Should map to MI-M-2020

**Legacy Tables Available:**
- B-2006-HOMBRES, B-2006-MUJERES ✓
- MI-2006-HOMBRES, MI-2006-MUJERES ✓

## Database Migration Architecture

### 1. Schema Design

```sql
CREATE TABLE tabla_mortalidad (
    id INTEGER PRIMARY KEY,
    nombre_estandar VARCHAR(50), -- CMF standard: "CB-H-2020", "RV-M-2020", etc.
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

-- Indexes for performance
CREATE INDEX idx_mortalidad_nombre_edad ON tabla_mortalidad(nombre_estandar, edad);
CREATE INDEX idx_mortalidad_sexo_tipo ON tabla_mortalidad(sexo, tipo_tabla);
CREATE INDEX idx_mortalidad_vigencia ON tabla_mortalidad(vigencia_inicio, vigencia_fin);

-- Table mapping for Excel names to CMF standards
CREATE TABLE nombre_tabla_mapeo (
    id INTEGER PRIMARY KEY,
    nombre_excel VARCHAR(50),        -- "CB-2020-HOMBRES"
    nombre_cmf VARCHAR(50),         -- "CB-H-2020"
    sexo CHAR(1),
    tipo VARCHAR(20),
    año INTEGER
);
```

### 2. Migration Process

#### Step 1: Name Mapping Setup
```sql
INSERT INTO nombre_tabla_mapeo VALUES 
('CB-2020-HOMBRES', 'CB-H-2020', 'H', 'SOBREVIVENCIA', 2020),
('RV-2020-MUJERES', 'RV-M-2020', 'M', 'VEJEZ', 2020),
('B-2020-MUJERES', 'B-M-2020', 'M', 'SOBREVIVENCIA', 2020),
('MI-2020-HOMBRES', 'MI-H-2020', 'H', 'INVALIDEZ', 2020),
('MI-2020-MUJERES', 'MI-M-2020', 'M', 'INVALIDEZ', 2020),
-- Legacy tables
('B-2006-HOMBRES', 'B-2006-H', 'H', 'SOBREVIVENCIA', 2006),
('B-2006-MUJERES', 'B-2006-M', 'M', 'SOBREVIVENCIA', 2006),
('MI-2006-HOMBRES', 'MI-2006-H', 'H', 'INVALIDEZ', 2006),
('MI-2006-MUJERES', 'MI-2006-M', 'M', 'INVALIDEZ', 2006);
```

#### Step 2: Data Extraction Algorithm

```python
def extract_mortality_table(excel_path, sheet_name, start_col, end_col, table_name):
    """
    Extract individual table from Excel sheet
    """
    df = pd.read_excel(excel_path, sheet_name=sheet_name, header=None)
    
    # Find data start row (row with 'Edad')
    data_start = None
    for row_idx in range(df.shape[0]):
        if df.iloc[row_idx, start_col] == 'Edad':
            data_start = row_idx
            break
    
    if data_start is None:
        raise ValueError(f"Could not find 'Edad' row in {sheet_name}")
    
    # Extract data
    table_data = []
    for row_idx in range(data_start + 1, df.shape[0]):
        edad = df.iloc[row_idx, start_col]
        if pd.notna(edad) and isinstance(edad, (int, float)):
            qx = df.iloc[row_idx, start_col + 1]
            factor_aax = df.iloc[row_idx, start_col + 2]
            
            if pd.notna(qx):
                table_data.append({
                    'edad': int(edad),
                    'prob_muerte': float(qx),
                    'factor_aax': float(factor_aax) if pd.notna(factor_aax) else None
                })
    
    return table_data

def migrate_all_tables(excel_path):
    """Migrate all tables from Excel to database"""
    # Define table locations in Excel
    tables_config = [
        ('Vejez-Mujeres', 0, 3, 'RV-2004-MUJERES'),
        ('Vejez-Mujeres', 4, 7, 'RV-2009-MUJERES'),
        ('Vejez-Mujeres', 8, 11, 'RV-2014-MUJERES'),
        ('Vejez-Mujeres', 12, 29, 'RV-2020-MUJERES'),
        # ... continue for all sheets and tables
    ]
    
    for sheet_name, start_col, end_col, table_name in tables_config:
        try:
            data = extract_mortality_table(excel_path, sheet_name, start_col, end_col, table_name)
            # Load into database with proper mapping
            load_to_database(data, table_name)
        except Exception as e:
            print(f"Error processing {table_name}: {e}")
```

### 3. Data Validation Rules

#### Range Validation
```sql
-- Validate qx values (death probabilities)
SELECT nombre_estandar, COUNT(*) as invalid_qx
FROM tabla_mortalidad 
WHERE prob_muerte < 0 OR prob_muerte > 1
GROUP BY nombre_estandar;

-- Validate age ranges
SELECT nombre_estandar, MIN(edad) as min_age, MAX(edad) as max_age, COUNT(*) as count
FROM tabla_mortalidad 
GROUP BY nombre_estandar
HAVING MAX(edad) - MIN(edad) + 1 != COUNT(*);
```

#### Monotonicity Check
```sql
-- Check if death rates generally increase with age (after infancy)
WITH consecutive_ages AS (
    SELECT 
        nombre_estandar,
        edad,
        prob_muerte,
        LEAD(prob_muerte) OVER (PARTITION BY nombre_estandar ORDER BY edad) as next_qx
    FROM tabla_mortalidad
    WHERE edad >= 5  -- Skip infant mortality where patterns differ
)
SELECT 
    nombre_estandar,
    COUNT(*) as decreasing_count
FROM consecutive_ages 
WHERE prob_muerte > next_qx  -- Current death rate > next death rate
GROUP BY nombre_estandar
ORDER BY decreasing_count DESC;
```

#### Survival Probability Consistency
```sql
-- Verify survival probabilities make sense
WITH survival_check AS (
    SELECT 
        nombre_estandar,
        edad,
        prob_muerte,
        LEAD(prob_muerte) OVER (PARTITION BY nombre_estandar ORDER BY edad) as next_qx
    FROM tabla_mortalidad
)
SELECT 
    nombre_estandar,
    AVG(prob_muerte) as avg_qx,
    MAX(prob_muerte) as max_qx,
    MIN(prob_muerte) as min_qx,
    COUNT(*) as total_ages
FROM survival_check
WHERE edad >= 0 AND edad <= 110
GROUP BY nombre_estandar;
```

## Missing Documentation & References

### Current References Available
✅ **NCG 318 (2011)** - Complete analysis with mathematical formulas
✅ **Mortality Tables** - Excel file with all qx values and factors
✅ **Implementation Architecture** - Complete technical design

### Missing Critical References

#### 1. Circular N°1512 (2001)
**Status**: ❌ **MISSING** - Critical for pre-2012 policy calculations
**Content Needed**:
- Traditional reserve calculation methodology
- TM (tasa venta) calculation formulas
- Calce measurement methodology
- Financial reserve calculation rules

#### 2. Circular N°2332
**Status**: ❌ **MISSING** - Essential for current mortality table specifications
**Content Needed**:
- Official mortality table specifications (CB-H-2020, RV-M-2020, etc.)
- Application rules and effectivity dates
- Validation requirements

#### 3. NCG N°374 (2015)
**Status**: ❌ **MISSING** - Discount rate methodology for 2015-2020 policies
**Content Needed**:
- Annex with discount rate calculation rules
- Transitional rules between methodologies

#### 4. NCG N°209 (2007)
**Status**: ❌ **MISSING** - Asset sufficiency analysis requirements
**Content Needed**:
- Sufficiency testing methodology
- Additional reserve calculation rules

#### 5. VTD Rate Data
**Status**: ❌ **MISSING** - Monthly CMF discount rate publications
**Content Needed**:
- Historical VTD vectors (ET + AV)
- Monthly rate publications from CMF
- TILP (long-term interest rate) historical values

#### 6. TM Rate Data
**Status**: ❌ **MISSING** - Historical tasa venta data
**Content Needed**:
- Monthly TM rates by policy type
- Historical data for calce calculations

### Immediate Action Items

#### High Priority (Required for Development)
1. **Obtain Circular N°1512** - Without this, cannot implement pre-2012 calculations
2. **Obtain Circular N°2332** - Needed to validate current table implementations
3. **Get sample VTD data** - Essential for discount rate calculations
4. **Find TM historical rates** - Required for calce calculations

#### Medium Priority
1. **NCG N°374** - For transitional policy calculations
2. **NCG N°209** - For asset sufficiency analysis
3. **Policy data examples** - For testing and validation

#### Where to Find Missing References
- **CMF Website**: www.cmf.cl - Official publications section
- **Insurance Company Actuarial Departments**: May have internal copies
- **Regulatory Compliance Teams**: Should maintain reference library
- **Chilean Insurance Association**: AChI - may provide guidance
- **Actuarial Consulting Firms**: Specialized in Chilean regulations

This analysis provides the complete foundation for migrating mortality tables to the database while identifying the critical missing regulatory documents needed for full system implementation.