# VTD_2025_.xlsx Summary

## Document Overview

**File Type:** VTD Vector Tasa de Descuento
**Content:** Monthly discount rate vectors for multiple years (2020-2025)
**Structure:** Monthly sheets with period-based rate vectors (1-120 years)

## Data Analysis

### 2025

- **Months Available:** 9
- **Period Range:** 1 - 25 years
- **Rate Range:** 0.0231 - 0.0254
- **Average Rate:** 0.0247
- **Sample Data (2025-01):**
  - Year 1: 0.0235
  - Year 2: 0.0231
  - Year 3: 0.0238
  - Year 4: 0.0245
  - Year 5: 0.0249
  - Year 6: 0.0252
  - Year 7: 0.0253
  - Year 8: 0.0254
  - Year 9: 0.0254
  - Year 10: 0.0254
  - ... (15 more)
- **All Months:** 2025-01, 2025-02, 2025-03, 2025-04, 2025-05, 2025-06, 2025-07, 2025-08, 2025-09

### 2024

- **Months Available:** 12
- **Period Range:** 1 - 25 years
- **Rate Range:** 0.0213 - 0.0445
- **Average Rate:** 0.0241
- **Sample Data (2024-01):**
  - Year 1: 0.0445
  - Year 2: 0.0339
  - Year 3: 0.0289
  - Year 4: 0.0264
  - Year 5: 0.0250
  - Year 6: 0.0241
  - Year 7: 0.0235
  - Year 8: 0.0231
  - Year 9: 0.0228
  - Year 10: 0.0226
  - ... (15 more)
- **All Months:** 2024-01, 2024-02, 2024-03, 2024-04, 2024-05, 2024-06, 2024-07, 2024-08, 2024-09, 2024-10, 2024-11, 2024-12

### 2023

- **Months Available:** 12
- **Period Range:** 1 - 25 years
- **Rate Range:** 0.0181 - 0.0223
- **Average Rate:** 0.0214
- **Sample Data (2023-01):**
  - Year 1: 0.0181
  - Year 2: 0.0207
  - Year 3: 0.0206
  - Year 4: 0.0205
  - Year 5: 0.0206
  - Year 6: 0.0207
  - Year 7: 0.0208
  - Year 8: 0.0210
  - Year 9: 0.0212
  - Year 10: 0.0213
  - ... (15 more)
- **All Months:** 2023-01, 2023-02, 2023-03, 2023-04, 2023-05, 2023-06, 2023-07, 2023-08, 2023-09, 2023-10, 2023-11, 2023-12

### 2022

- **Months Available:** 12
- **Period Range:** 1 - 25 years
- **Rate Range:** -0.0172 - 0.0208
- **Average Rate:** 0.0125
- **Sample Data (2022-01):**
  - Year 1: -0.0172
  - Year 2: -0.0088
  - Year 3: -0.0027
  - Year 4: 0.0017
  - Year 5: 0.0050
  - Year 6: 0.0077
  - Year 7: 0.0097
  - Year 8: 0.0114
  - Year 9: 0.0128
  - Year 10: 0.0140
  - ... (15 more)
- **All Months:** 2022-01, 2022-02, 2022-03, 2022-04, 2022-05, 2022-06, 2022-07, 2022-08, 2022-09, 2022-10, 2022-11, 2022-12

### 2021

- **Months Available:** 12
- **Period Range:** 1 - 25 years
- **Rate Range:** -0.0163 - 0.0073
- **Average Rate:** 0.0005
- **Sample Data (2021-01):**
  - Year 1: -0.0163
  - Year 2: -0.0149
  - Year 3: -0.0117
  - Year 4: -0.0086
  - Year 5: -0.0061
  - Year 6: -0.0041
  - Year 7: -0.0025
  - Year 8: -0.0011
  - Year 9: 0.0000
  - Year 10: 0.0010
  - ... (15 more)
- **All Months:** 2021-01, 2021-02, 2021-03, 2021-04, 2021-05, 2021-06, 2021-07, 2021-08, 2021-09, 2021-10, 2021-11, 2021-12

### 2020

- **Months Available:** 4
- **Period Range:** 1 - 25 years
- **Rate Range:** -0.0107 - 0.0075
- **Average Rate:** 0.0021
- **Sample Data (2020-09):**
  - Year 1: -0.0107
  - Year 2: -0.0096
  - Year 3: -0.0073
  - Year 4: -0.0051
  - Year 5: -0.0032
  - Year 6: -0.0017
  - Year 7: -0.0005
  - Year 8: 0.0006
  - Year 9: 0.0015
  - Year 10: 0.0023
  - ... (15 more)
- **All Months:** 2020-09, 2020-10, 2020-11, 2020-12


## Implementation Notes

### Rate Vector Structure
- **Periods 1-120:** Complete term structure for discount rate calculations
- **Monthly Updates:** Each sheet represents a year with monthly rate vectors
- **Date-based Selection:** Use publication date to select appropriate VTD vector

### Database Integration
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

### Application in Reserve Calculations
```go
// VTD rate lookup function
func getVTDRate(year, month, period int) (float64, error) {
    // Query database for specific rate
    // SELECT rate FROM vtd_vector WHERE year = ? AND month = ? AND period = ?
    return rate, nil
}
```

---

*This analysis shows the VTD file contains complete discount rate vectors for reserve calculations.*

