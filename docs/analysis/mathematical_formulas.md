# Mathematical Formulas - Reservas de Rentas Vitalicias

## Core Reserve Calculation Formulas

### 1. Present Value of Policy (VPPj)

The fundamental formula for calculating the present value of a policy j:

```latex
VPP_j = \sum_{i=1}^{n} FP_{ji} \times (1 + TC_j)^{-i}
```

Where:
- $VPP_j$ = Present value of policy j at inception
- $FP_{ji}$ = Probabilistic flow of policy j in period i (no reinsurance deductions)
- $TC_j$ = Tasa costo equivalente (IRR implicita en la póliza)
- $n$ = Number of periods until policy termination

### 2. Discount Rate Determination

For policies from January 1, 2012 onwards:

```latex
\text{Discount Rate}_j = \min(TM_j, TC_j)
```

Where:
- $TM_j$ = Tasa de venta (definida en Título III Circular N°1512)
- $TC_j$ = Tasa costo equivalente (calculada mediante VPPj)

### 3. Discount Rate Vector (VTD)

The discount rate vector used for VPPj calculation:

```latex
VTD = ET + AV
```

#### 3.1 Structure Components

**Real Risk-Free Term Structure (ET):**

```latex
ET = \begin{bmatrix}
t_1 \\
t_2 \\
\vdots \\
t_{25} \\
t_{26} \\
\vdots \\
t_{120}
\end{bmatrix} = \begin{bmatrix}
\text{Líquido (0-25 años)} \\
\text{Extrapolado (26-120 años)}
\end{bmatrix}
```

Where:
- **Segmento Líquido (0-25 años)**: Basado en transacciones observadas del Estado y Banco Central
- **Segmento Extrapolado (26-120 años)**: Metodología Smith-Wilson, convergencia a TILP a 65 años

**Volatility Adjustment (AV):**

```latex
AV = 0.65 \times (R_{exceso} - R_{libre\_riesgo}) - 0.35 \times SRC
```

Where:
- $R_{exceso}$ = Exceso de retorno sobre retorno libre de riesgo
- $R_{libre\_riesgo}$ = Retorno libre de riesgo
- $SRC$ = Spread por riesgo de crédito

### 4. Monthly Rate Conversion

Annual to monthly discount rate conversion:

```latex
i_{mensual}^{(j)} = (1 + i_{anual}^{(j)})^{\frac{1}{12}} - 1
```

## Life Annuity Formulas

### 1. Immediate Life Annuity

```latex
a_x = \sum_{t=1}^{\omega - x} v^t \times {}_t p_x
```

### 2. Life Annuity Due

```latex
\ddot{a}_x = \sum_{t=0}^{\omega - x} v^t \times {}_t p_x = 1 + \sum_{t=1}^{\omega - x} v^t \times {}_t p_x
```

### 3. Temporary Life Annuity (n years)

```latex
a_{x:\overline{n}|} = \sum_{t=1}^{n} v^t \times {}_t p_x
```

### 4. Deferred Life Annuity (m years deferral, n years payment)

```latex
{}_{m|}a_{x:\overline{n}|} = \sum_{t=m+1}^{m+n} v^t \times {}_t p_x
```

## Commutation Functions

### 1. Basic Commutation Functions

```latex
D_x = v^x \times l_x
```

```latex
N_x = \sum_{t=x}^{\omega} D_t = D_x + D_{x+1} + \cdots + D_{\omega}
```

```latex
C_x = v^{x+1} \times d_x = v^{x+1} \times l_x \times q_x
```

```latex
M_x = \sum_{t=x}^{\omega} C_t
```

### 2. Annuity Using Commutation Functions

**Deferred Annuity Example (10 years deferred, age 75):**

```latex
{}_{10|}\ddot{a}_{75} = \frac{N_{85}}{D_{75}}
```

## Reserve Calculation with Premiums

### 1. Benefit Reserve at Time t

```latex
{}_tV = \text{Beneficio}_{x+t} - \text{Prima}_{x+t}
```

```latex
{}_tV = \text{Pago} \times a_{x+t} - \text{Prima} \times a_{x+t:\overline{n-t}|}
```

### 2. Reserve After Payment Period

For t ≥ payment period:

```latex
{}_tV = \text{Pago} \times a_{x+t}
```

## Survival and Death Probabilities

### 1. Annual Survival Probability

```latex
p_x = \frac{l_{x+1}}{l_x} = 1 - q_x
```

### 2. Cumulative Survival Probability

```latex
{}_t p_x = \frac{l_{x+t}}{l_x} = p_x \times p_{x+1} \times \cdots \times p_{x+t-1}
```

### 3. Death Probability Between x and x+t

```latex
{}_t q_x = 1 - {}_t p_x
```

## TAP (Test de Adecuación de Pasivos) Formulas

### 1. Best Estimate Liabilities (BEL)

```latex
BEL = \sum_{i=1}^{n} CF_i \times e^{-r_i \times t_i} \times S_i
```

Where:
- $CF_i$ = Cash flow in period i
- $r_i$ = Risk-free rate for period i
- $t_i$ = Time to period i
- $S_i$ = Survival probability to period i

### 2. Risk Margin

```latex
\text{Risk Margin} = \text{CoC} \times \sum_{t=1}^{T} \text{BEL}_t \times v^t \times {}_t p_x
```

Where:
- $\text{CoC}$ = Cost of capital (típicamente 6%)
- $T$ = Tiempo de runoff de las obligaciones

### 3. Technical Reserve Requirements

```latex
\text{Reserva Técnica} = \max(\text{Reserva Contable}, BEL + \text{Risk Margin})
```

## Reinsurance Formulas

### 1. Gross vs. Net Reserve

```latex
\text{Reserva Bruta} = \text{Reserva Neta} + \text{Activo por Reaseguro}
```

### 2. Reinsurance Asset

```latex
\text{Activo Reaseguro} = \text{Proporción Cesada} \times \text{Reserva Bruta}
```

**Deterioro Testing:**

```latex
\text{Deterioro} = \max(0, \text{Valor Contable} - \text{Valor Recuperable})
```

## Integration Examples

### 1. Complete Policy Calculation

```latex
VPP_j = \sum_{i=1}^{\omega-x_j} \text{Renta}_j \times {}_i p_{x_j} \times (1 + \min(TM_j, TC_j))^{-i}
```

### 2. with Different Mortality Tables

For female beneficiaries (post-2012):

```latex
VPP_j = \sum_{i=1}^{\omega-x_j} \text{Renta}_j \times {}_i p^{RV-M-2020}_{x_j} \times (1 + \min(TM_j, TC_j))^{-i}
```

For male beneficiaries (post-2012):

```latex
VPP_j = \sum_{i=1}^{\omega-x_j} \text{Renta}_j \times {}_i p^{CB-H-2020}_{x_j} \times (1 + \min(TM_j, TC_j))^{-i}
```

These formulas provide the complete mathematical foundation for implementing the reserves calculator according to CMF NCG 318 requirements.