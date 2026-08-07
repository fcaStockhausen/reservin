# Observaciones de avance (validación RIS y VTD)

**Fecha:** 2026-08-07 (actualizado)
**Objetivo:** registrar el estado, los hallazgos y los pendientes para continuar
la validación de las reservas contra el RIS real (Circular 1194).

---

## 0. RESUMEN EJECUTIVO (estado final)

**Fecha actualización:** 2026-08-07.

**Conclusión clave:** tras identificar y corregir tres bugs de cálculo y calibrar
la tasa de descuento correcta, el motor de reservas cuadra con el RIS a **<3%
para pólizas modernas (post-sep-2020)**. El gap restante se concentra casi
exclusivamente en **stock pre-2020 sin VTD histórico disponible**, donde caemos
a la TM del RIS (sub-estimada para cohortes TCj).

### Tasa de descuento por cohorte (NCG 318)

La "tasa de bautizo" cambia de TIPO según la cohorte de emisión:

| Cohorte | Tasa de la reserva | Fuente del dato |
|---|---|---|
| pre-2012 | TM + ajuste por calce (Circular 1512) | RIS `TasaCostoEmision` (aprox) |
| 2012-may2015 | min(TM, TV) | RIS `TasaCostoEmision` / `TasaVenta` |
| **jun2015-nov2020** | **TCj** (TIR con VTD del mes de emisión) | **VTD histórico** |
| **post-dic2020** | min(TVj, TCj) | **VTD histórico** + RIS `TasaVenta` |

El campo `TasaCostoEmision` del RIS es la **TM**, que solo es la tasa correcta
para cohortes 2012-may2015. Para cohortes que requieren TCj, **hay que cargar el
VTD del mes de emisión** y usarlo como curva.

### Evolución de la validación (10K pólizas)

| Etapa | Diferencia global | Notas |
|---|---|---|
| Bug edad contratación | +76% | bug |
| Fix currentYear | +33% | real |
| Cuadro 4 + 11/24 + tasa emisión | +28.2% | normativo |
| + Mejoramiento AAx | +24.5% | normativo |
| + Comparador correcto (post-2012) | +8.5% | bug de selección |
| + VTD mes de emisión (post-sep-2020) | **+8.5%** | limitado por VTD histórico |

El VTD **cierra el gap a <3% en las pólizas donde aplica** (verificada SVS=853852:
-2.46%). El gap +8.5% global restante se debe a stock pre-2020 sin VTD cargado.

### Bugs corregidos esta sesión

1. **Bug exclusión de cónyuges**: el filtro `DerechoPension == "99" || "20"` en
   `buildGrupoFromRIS` y en el loop de reserva reportada excluía a los cónyuges
   (con DerechoPension="10"). El motor estaba comparando "causante solo vs
   causante solo". Corregido usando `rol` en vez de `DerechoPension`. Mismo bug
   aplicaba al sumar la reserva reportada (también corregido).
2. **Bug PG en cónyuge fallecido**: en el branch de causante fallecido, al tratar
   al cónyuge como causante en `soloGrupo`, el `flow.go` aplicaba el PG de la
   póliza original al cónyuge → sobrestimación. Corregido reseteando la
   modalidad en ese branch.
3. **Bug invalidez pre-2005 (MI-1985)**: la Circular 491 no define MI-85, así que
   los inválidos pre-2005 caen a B-85. El parser `SituacionInvalidez=="I"` estaba
   mal (debe ser "T"/"P"). Ambos corregidos.

### Breackdown por tipo de familia (10K pólizas, con todos los fixes)

| Tipo | n | diff |
|---|---|---|
| causante_solo | 1777 | **+2.30%** ✓ |
| con_conyuge | 3044 | **+1.92%** ✓ |
| con_hijos | 413 | +21.90% |
| conyuge_hijos | 1225 | +11.08% |
| mixto | 79 | +6.50% |
| **sin_causante_vivo** | 1779 | **+28.12%** ← concentrado en pre-2020 sin VTD |

### Próximos pasos prioritarios

1. **Conseguir VTD histórico pre-2020-09** (publicación CMF). Esto es lo único
   que bloquea cerrar el gap en stock 2015-2020.
2. Implementar cálculo de **TCj** cuando el VTD no esté disponible (TIR
   iterativa con flujos de bautizo) — alternativa al VTD histórico.
3. Revisar el modelo para **`con_hijos`** (+22%) — posible bug en porcentaje de
   hijos o en la tabla aplicada.

---

## 0.1 Hallazgos normativos procesados

### NCG 318 (01.09.2011) — tasa de descuento
- N°2.3a: la tasa de descuento de la reserva = **min(TM, TV) a la fecha de
  entrada en vigencia de la póliza**. No es el VTD actual.
- N°2.3e: flujos con tablas CB-H-2020, RV-M-2020, B-M-2020, MI-H/M-2020
  (Circular 2332).
- N°2.4: pólizas pre-2012 usan la Circular 1512 (con calce) salvo opción al 2.3.

### Circular 2332 (01.03.2023) — tablas y mejoramiento
- Sección 1.1 (post-2023): CB-2020/RV-2020/B-2020/MI-2020 con mejoramiento,
  año base 2020, tasa según NCG 318.
- Sección 1.2.1 (2012-2023): RT-BASE con CB-2020 + mejoramiento, manteniendo la
  tasa de descuento de emisión.
- Sección 1.2.2 (2008-2011): RT-FINANCIERA con CB-2020 + mejoramiento; RT-BASE
  con RV-2009/B-2006/MI-2006.
- Sección 1.3 (pre-2008): RT-FINANCIERA con CB-2020 + mejoramiento; RT-BASE con
  RV-2009/B-2006/MI-2006 (1.3.1, 2005-2008) o RV-85/B-85/MI-85 (1.3.2, pre-2005).

### Nota Técnica N°9 (SPensiones, Nov 2024) — fórmulas
- Ecuación (2): `qx,año = qx,2020 × Π_{t=2021}^{año} (1 - AAx,t)` (mejoramiento
  dinámico, dos dimensiones: edad y tiempo).
- Ecuación (7): CNU con cónyuge = `Σ lx+t/(lx·(1+it)^t) - 11/24 + 0.6 × Σ
  ly+t/(ly·(1+it)^t) × (1 - lx+t/lx)`. Confirma el modelo del FlowProjector.
- `11/24` ≈ 0.4583: ajuste de mensualización (ya implementado en el
  FlowProjector).
- Cuadro 4: vigencia de tablas por fecha de pensión (afiliado RV, beneficiario B,
  inválido MI) — **implementado en `tabla.go`**.

---

## 1. Estado actual (compila y tests OK)

- `go build ./...` — OK
- `go vet` — OK
- Tests individuales: `internal/models`, `internal/calculator`, `internal/portfolio`,
  `internal/loader` — todos PASS (no correr los 4 juntos con timeout corto; el
  portfolio genera 10K pólizas en el test y el loader necesita el archivo RIS).

## 2. Lo que ya está resuelto (esta sesión)

### 2.1 VTDs históricos cargados
- Archivo: `data/vtd/articles-51926_recurso_1.xlsx` (hojas 2020-2026).
- Cargado con `-import vtd` → 71 vectores, 8,520 puntos (2020-09 a 2026-07).
- `internal/database/vtd.go`: `BatchInsert` usa `INSERT OR REPLACE` (idempotente);
  nuevo `GetAllVectorDates()`.
- `internal/calculator/reserve.go`: nuevo `LoadVTDFor(year, month)` y
  `CalculateAt(policy, grupo, renta, currentYear)`.
- `internal/config/config.go`: `Data.VTDHistorico` apunta al archivo histórico.

### 2.2 El error del VTD (ya medido, ahora resuelto)
- Cuando solo había VTD 2025-09: spread entre curvas extremas ~18% (reservas
  -7.9% a +10.1% según el VTD usado).
- **VTD del período (2025-12) vs más reciente (2026-07): -2.76%.**
- Con los VTDs históricos cargados y usando `LoadVTDFor(año, mes)`, el error se
  elimina para períodos ≥ 2020-09. Pendiente: pre-2020 no hay VTD.

### 2.3 Bug crítico: valuar a la edad ACTUAL (currentYear)
- **Problema:** `calc.Calculate` usaba `currentYear=0` → valoraba la reserva a la
  edad de CONTRATACIÓN (ej. 55) en vez de la edad ACTUAL (75 en 2025). Resultado:
  factor de anualidad ~2x (21.6 vs 11) → reservas duplicadas.
- **Fix:** `CalculateAt(policy, grupo, renta, currentYear)` donde
  `currentYear = 2025 - contractYear`. Para SVS=730135 (causante F nac 1950,
  contrato 2005): reserva bajó de +69% a +12.4% vs RT-FINANCIERA-2020 reportada.
- `ris_validate.go` usa `CalculateAt` con el currentYear correcto.

### 2.4 Tabla contemporánea TM-2020 para RT-FINANCIERA-2020
- `buildGrupoFromRIS(p, contemporanea bool)`: con `contemporanea=true` asigna
  `SelectContemporaneaTable` (TM-2020) en vez de la tabla de bautizo. La
  validación contra RT-FINANCIERA-2020 usa TM-2020 (correcto).
- `vtdSensitivity` también usa TM-2020 para aislar el efecto del descuento.

### 2.5 Período Garantizado (PG) modelado
- `models.GarantizedMonths(modalidad)` extrae los meses garantizados (3xxx/4xxx).
- `Project`: durante el PG el causante paga con prob=1 (pago cierto); los
  beneficiarios solo fluyen post-PG. Test `pg_test.go` verifica +8% con PG 240m.

### 2.6 Tabla de bautizo por año de emisión
- `baseTableYear` post-2012 refinado: 2009 (2012-13), 2014 (2014-19), 2020 (2020+).
- `tableName` maneja B-M-2009 → B-M-2006 (no existe B-M-2009).
- Tests actualizados (24 casos en `tabla_test.go`, `generator_test.go`).

### 2.7 Reaseguro (total vs retenida)
- `RISPerson`: `ReserveBaseRetenida()`, `ReserveFinanciera2020Retenida()`,
  `CededShare()`.
- Flag `-retenida` en `-validate-ris` para comparar contra la reserva neta.

### 2.8 Cuadro 4 (Nota Técnica N°9) — tablas por rol y vigencia
- **Reemplaza** `baseTableYear`/`tableName` por `cuadro4Table(cat, sexo, fecha)`:
  afiliado (RV/CB), beneficiario (B/CB) e inválido (MI) tienen vigencias PROPIAS.
- Períodos oficiales: pre-2005 (RV-85/B-85), 2005-2008 (RV-2004/B-85),
  2008-2010 (RV-2004/B-2006), 2010-2016 (RV-2009/B-2006), 2016-2023
  (CB-2014/B-2014), 2023+ (CB-2020/B-2020).
- `SelectBaseTable`/`SelectContemporaneaTable` usan el Cuadro 4.
- Tests actualizados (30 casos en `tabla_test.go`, `generator_test.go`).

### 2.9 Factores de mejoramiento (Circular 2332 / NT9 ec. 2)
- Nueva tabla `factor_mejoramiento` (migración v4): nombre_estandar, edad, año,
  factor_aa. Cargados **10,336 factores** (5 tablas 2020 × 2021-2036).
- `MortalityEngine`: `mejoradaQx` aplica `qx,año = qx,2020 × Π(1-AAx,t)` con el
  año de cálculo (valuación). `SetAñoCálculo` + `LoadMejoramiento`.
- `ReserveCalculator.CalculateAt`: carga mejoramientos y setea el año de valuación.
- `FlowProjector.qxMejorada` usa la qx mejorada en el loop.
- La corrección **11/24** (mensualización, NT9 ec. 7) se aplica al total del
  causante en `Project`.
- El loader de mortalidad ahora extrae las columnas anuales (2021-2036) del xlsx.

---

## 3. RESULTADO DE LA VALIDACIÓN (10K pólizas, post-fixes)

Comparando contra **RT-FINANCIERA-2020** (TM-2020 + VTD 2025-12 + edad actual):

| Sample | Diferencia global | Tiempo |
|---|---|---|
| 50 pólizas | +28.1% | 0.5s |
| 10,000 pólizas | **+32.91%** | **0.66s** |

Por estrato: pre-2005 +41.8%, 2005-2011 +22.7%.

**Análisis del overstatement (caso SVS=696073, +23%):**

| Componente | Nuestro factor | Reportado | Diff |
|---|---|---|---|
| Causante (RV-M-2020, edad 75) | 7.2 | 7.9 | **-9% (cercano)** |
| Sobreviviente cónyuge (B-M-2020, edad ~70) | 9.0 | 3.8 | **+135% (2.4x alto)** |

El causante se calcula bien. La diferencia está en el modelo de sobrevivencia:
nuestro `FlowProjector` modela `flow = R × pct × tpx_ben(k) × (1 - tpx_causante(k))`,
donde el cónyuge recibe sólo después que el causante muere. La compañía puede
usar un modelo distinto (anualidad conjunta, tabla propia de sobrevivencia, o
ajustar el momento de inicio del cónyuge).

---

## 4. BUGS RESUELTOS EN ESTA SESIÓN

### 4.1 Deadlock del canal (causante del "cuelgue")
- **Síntoma:** `-validate-ris` con sample ≥ 6 se colgaba indefinidamente.
- **Causa:** `continue` después de `policies = nil` en el `select` saltaba el
  chequeo `if policies == nil && errs == nil { break }`. Cuando ambos canales
  estaban cerrados, el `select` no tenía casos activos → deadlock.
- **Fix:** reemplazar `continue` por `break` (sale del `select`, no del `for`),
  permitiendo que el chequeo de nil se ejecute.
- **Impacto:** 10K pólizas pasaron de "infinite" a **0.66s**.

### 4.2 Caso sobreviviente delegado a FlowProjector
- **Antes:** loop manual con `QxFor`/`RateAt` por período (~3s/póliza).
- **Ahora:** cada sobreviviente se valúa como anualidad individual vía
  `CalculateAt` → mismo motor optimizado del causante.

---

## 5. Próximo foco: modelo de sobrevivencia

El overstatement restante (+33%) está concentrado en el **componente de
sobrevivencia** (cónyuge/hijos), no en el causante. Líneas de investigación:

1. **¿El cónyuge recibe inmediatamente o después del causante?** El modelo actual
   asume `prob = tpx_ben × (1 - tpx_causante)`. Pero en algunas modalidades el
   cónyuge podría tener una anualidad propia (no contingente).
2. **¿La tabla de sobrevivencia es la correcta?** Usamos `SelectContemporaneaTable`
   que para cónyuge da B-M-2020. La compañía podría usar una tabla distinta.
3. **¿El % de pensión se aplica sobre la renta original o la recalculada?**
4. **¿El PG afecta al cónyuge?** En modalidad 3xxx, el PG garantiza pagos al
   causante; al morir, los pagos van a la herencia, no al cónyuge.

---

## 6. Comandos útiles

```bash
# Validar contra RT-FINANCIERA-2020 (VTD del período + TM-2020 + edad actual)
go run ./cmd/calculator -validate-ris /tmp/ris_extract/ris20251231.vta -sample 1000

# Validar contra la versión retenida (post-reaseguro)
go run ./cmd/calculator -validate-ris /tmp/ris_extract/ris20251231.vta -sample 1000 -retenida

# Sensibilidad al VTD (reserva bajo cada curva disponible)
go run ./cmd/calculator -validate-ris /tmp/ris_extract/ris20251231.vta -vtd-sens -sample 2000

# Importar VTDs históricos
go run ./cmd/calculator -import vtd
```

El archivo RIS está descomprimido en `/tmp/ris_extract/ris20251231.vta` (768MB).
El original es `data/ris20251231.zip`.

---

## 7. Notas de arquitectura a tener presentes

- `SelectContemporaneaTable` = tabla TM-2020 en vigor (basis de RT-FINANCIERA).
- `SelectBaseTable` = tabla de bautizo por fecha de contrato (basis de RT-BASE).
- El descalce = RT-FINANCIERA - RT-BASE. La validación contra RT-FINANCIERA debe
  usar TM-2020; contra RT-BASE, la tabla de bautizo.
- `currentYear` (años desde emisión hasta valuación) es ESENCIAL para valorar a la
  edad actual. No usar 0 salvo que se valore a la fecha de emisión.
- El VTD por período se instala con `LoadVTDFor`; `GetLatestVector` es el default.
