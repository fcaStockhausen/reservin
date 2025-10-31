# Missing Regulatory References Analysis

## Summary of Available vs Missing Documentation

### ✅ Currently Available

| Document | Status | Coverage | Priority |
|----------|--------|----------|----------|
| NCG 318 (2011) | ✅ COMPLETE | Reserve calculation formulas, VTD methodology | HIGH |
| Mortality Tables (Excel) | ✅ COMPLETE | All qx values for 2004-2020 tables | HIGH |
| Implementation Plan | ✅ COMPLETE | Technical architecture and development roadmap | HIGH |
| Mathematical Formulas | ✅ COMPLETE | LaTeX formulas for all calculations | HIGH |

### ❌ Critical Missing References

| Document | Required For | Impact if Missing | Priority |
|----------|--------------|------------------|----------|
| Circular N°1512 (2001) | Pre-2012 policy calculations, TM rates, calce methodology | Cannot calculate 80%+ of existing policies | CRITICAL |
| Circular N°2332 | Current mortality table specifications | Risk of non-compliance with latest tables | HIGH |
| Monthly VTD Rate Data | Discount rate calculations for all reserves | No discount rates possible | CRITICAL |
| TM Historical Rates | Calce calculations and rate comparisons | Cannot determine minimum(TM,TC) | HIGH |
| NCG N°374 (2015) | 2015-2020 policy discount rates | Policy calculation gaps | MEDIUM |
| NCG N°209 (2007) | Asset sufficiency analysis | Cannot perform required testing | MEDIUM |

## Detailed Impact Analysis

### 1. Circular N°1512 (2001) - CRITICAL

**Why Critical:**
- Defines traditional reserve calculation methodology for pre-2012 policies
- Contains TM (tasa venta) calculation formulas
- Specifies calce measurement methodology
- Required for financial reserve calculations

**Impact if Missing:**
- **Policy Coverage Gap**: Cannot calculate reserves for policies before 2012
- **Regulatory Non-Compliance**: Failure to meet CMF requirements for legacy policies
- **Financial Risk**: Incorrect reserve calculations for majority of existing portfolio
- **Audit Issues**: No methodology for regulatory reviews

**Specific Content Needed:**
```
- Título III: Cálculo de TM (tasa de venta)
- Título IV: Medición de calce
- Título V: Reservas con reaseguro
- Formulas for financial reserves
- Calce adjustment methodologies
```

### 2. Monthly VTD Rate Data - CRITICAL

**Why Critical:**
- VTD (Vector de Tasas de Descuento) is fundamental for ALL reserve calculations
- Contains ET (estructura temporal) + AV (ajuste por volatilidad)
- Updated monthly by CMF via Oficio Ordinario
- Required for discount rate determination: min(TM, TC)

**Impact if Missing:**
- **No Discount Rates**: Cannot calculate any present values
- **Calculation Block**: All reserve calculations impossible
- **Regulatory Failure**: Cannot meet CMF reserve requirements

**Required Data Structure:**
```
- Monthly VTD publications (120-year vectors)
- ET components (liquid + extrapolated segments)
- AV adjustments by period
- TILP (tasa de interés de largo plazo) historical values
- SRC (spread por riesgo de crédito) data
```

### 3. TM Historical Rates - HIGH

**Why Important:**
- Required for min(TM, TC) discount rate comparison
- Essential for calce calculations
- Needed for rate selection methodology
- Historical data needed for existing policy valuations

**Impact if Missing:**
- **Incomplete Rate Selection**: Cannot determine optimal discount rate
- **Calce Calculation Issues**: Cannot perform proper calce measurements
- **Policy Valuation Errors**: Incorrect discount rates for existing policies

## Secondary Missing References

### 4. Circular N°2332 - HIGH

**Required For:**
- Official current mortality table specifications
- Application rules for CB-H-2020, RV-M-2020, etc.
- Validation of Excel table implementations
- Effectivity dates and transition rules

### 5. NCG N°374 (2015) - MEDIUM

**Required For:**
- Discount rate methodology for policies 2015-2020
- Annex with transitional calculation rules
- Bridge between old and new methodologies

### 6. NCG N°209 (2007) - MEDIUM

**Required For:**
- Asset sufficiency analysis methodology
- Additional reserve calculation rules
- Regulatory compliance testing

## Acquisition Strategy

### Primary Sources

#### 1. CMF Official Website (www.cmf.cl)
**Sections to Check:**
- **Normativa > Normas de Carácter General**: Download NCGs and Circulares
- **Normativa > Oficios Ordinarios**: Monthly VTD publications
- **Seguros > Información Relevante**: Industry communications
- **Mercado de Capitales > Tasas de Interés**: Rate data

#### 2. CMF Contact Methods
**Direct Requests:**
- **Email**: normativa@cmf.cl for regulatory documents
- **Phone**: +56 2 2690 4000 for document requests
- **Physical Visit**: CMF offices at Teatinos 280, Santiago

#### 3. Insurance Industry Sources
**Professional Networks:**
- **Chilean Insurance Association (AChI)**: achi@achi.cl
- **Actuarial Association of Chile**: Professional actuarial contacts
- **Insurance Company Actuarial Departments**: Internal references
- **Actuarial Consulting Firms**: Specialized regulatory compliance

### Secondary Sources

#### 4. Academic & Research Institutions
**Chilean Universities:**
- **Pontificia Universidad Católica**: Actuarial Science program
- **Universidad de Chile**: Business School actuarial research
- **Universidad Diego Portales**: Risk management programs

#### 5. Legal & Compliance Databases
**Professional Services:**
- **Legal databases**: ICR, Thomson Reuters for Chilean regulations
- **Compliance services**: Regulatory document subscriptions
- **Law firms**: Insurance regulation specialists

### Immediate Action Plan

#### Week 1 - Critical Documents
1. **Contact CMF directly** for Circular N°1512 and N°2332
2. **Request recent VTD publications** (last 12 months)
3. **Search AChI website** for available references
4. **Contact insurance industry colleagues** for document sharing

#### Week 2 - Rate Data
1. **Obtain TM historical rates** from insurance industry sources
2. **Get VTD historical data** for the last 5 years
3. **Download available circulars** from CMF website
4. **Request NCG N°374 and N°209** for secondary requirements

#### Week 3 - Validation
1. **Cross-reference obtained documents** with NCG 318 requirements
2. **Validate table specifications** with Circular N°2332
3. **Test rate calculation methodologies** with obtained formulas
4. **Document any gaps** for further research

### Document Request Templates

#### Email to CMF
```
Subject: Solicitud de documentos normativos - Sistema de cálculo de reservas

Estimados Sres. de la CMF,

Soy responsable del desarrollo de un sistema de cálculo de reservas técnicas de rentas vitalicias conforme a la normativa vigente. Para asegurar el cumplimiento regulatorio, solicito los siguientes documentos:

1. Circular N°1512 de 2001 - Instrucciones sobre constitución de reservas técnicas
2. Circular N°2332 - Especificaciones de tablas de mortalidad vigentes
3. Oficios Ordinarios con VTD mensual de los últimos 12 meses
4. NCG N°374 de 2015 - Tasa de descuento para pólizas 2015-2020
5. NCG N°209 de 2007 - Análisis de suficiencia de activos

Agradezco su colaboración en facilitar estos documentos para asegurar el cumplimiento de la normativa CMF.

Atentamente,
[Nombre]
[Cargo]
[Empresa]
[Contacto]
```

#### Industry Network Request
```
Subject: Referencia normativa - Reservas rentas vitalicias

Estimado(a) colega,

Estoy desarrollando un sistema de cálculo de reservas y necesito referencias específicas de la normativa CMF. ¿Alguien tendría acceso a:

- Circular N°1512 (método tradicional pre-2012)
- Datos históricos de VTD mensual
- Tasas TM históricas por tipo de póliza
- Circular N°2332 (tablas 2020)

Puedo compartir el análisis completo de NCG 318 y la arquitectura técnica a cambio. Muchas gracias por su ayuda.

Saludos,
[Nombre y contacto]
```

This analysis identifies exactly what's missing, why it's critical, and provides a concrete action plan for obtaining the necessary regulatory references to complete the reserves calculator implementation.