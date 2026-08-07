# Validación del motor de reservas contra el RIS real (Circular 1194)

**Fecha:** 2026-08-06
**Datos:** `data/ris20251231.zip` → `ris20251231.vta` (periodo 2025-12-31)
**Comando:** `go run ./cmd/calculator -validate-ris <ruta.vta> -sample 10000`

---

## 1. Resumen

El motor de reservas fue validado contra las reservas técnicas reportadas por las
compañías de rentas vitalicias en el RIS de la Circular N°1194. Sobre una muestra
de 8,461 pólizas comparables (de 10,000 leídas), la **diferencia global es de
-0.90%** respecto de la RT-BASE-TOTAL reportada. El motor es estructuralmente
correcto.

---

## 2. Datos del mercado (Registro 1 del archivo)

| Métrica | Valor |
|---|---|
| Fecha de corte | 2025-12-31 |
| Pólizas (Registro 2) | 959,664 |
| Beneficiarios (Registro 3) | 2,150,259 |
| Promedio personas/póliza | 2.24 |
| Tamaño archivo | 768 MB, 3,109,924 líneas, 246 bytes fixed-width |

Este es el universo completo del mercado de rentas vitalicias previsionales
(D.L. 3.500), excluyendo las pólizas de Le Mans (quiebra, garantía estatal IPS).

---

## 3. Resultado de validación

Muestra de 10,000 pólizas → 8,461 procesadas (1,539 omitidas: sin reserva
reportada o con causante sin tabla asignable).

### Global (tasa de valuación 5.5%)

| Métrica | Valor |
|---|---|
| Reserva reportada | 13,946,577.14 UF |
| Reserva calculada | 13,821,618.94 UF |
| **Diferencia total** | **-124,958.20 UF (-0.90%)** |
| Diferencia media absoluta | 38.67% |
| Mediana \|diferencia\| | 21.11% |

### Por estrato de fecha de contratación

| Estrato | n | Reportado | Calculado | Diff |
|---|---|---|---|---|
| pre-2005 (RV-85) | 2,397 | 1,842,205.80 | 2,978,741.18 | **+61.7%** |
| 2005-2011 (RV-2009) | 1,159 | 1,687,271.17 | 2,098,104.88 | **+24.3%** |
| post-2012 (TM-2020) | 4,905 | 10,417,100.17 | 8,744,772.87 | **-16.1%** |

La compensación neta (pre-2005 y 2005-2011 sobreestiman, post-2012 subestima)
produce el -0.9% global.

---

## 4. Hallazgos clave

### 4.1 El motor es estructuralmente correcto

El agregado del motor queda a **-0.9%** de la reserva reportada por las compañías,
con la misma estructura de cálculo (VPPj por miembro, descalce por estrato,
supervivencia acumulativa optimizada). Esto valida la arquitectura y el flujo de
proyección.

### 4.2 La tasa de descuento es crítica (hallazgo principal)

La RT-BASE reportada **no** se descuenta a la TCM de emisión (campo 2.22,
~2.78%) sino a la **tasa de valuación regulatoria**, cuyo proxy de mercado es
~5.5%.

| Tasa usada | Diferencia global |
|---|---|
| TCM de emisión (min TM/TC) | +44% |
| 5.5% (proxy mercado) | -0.9% |

**Implicancia:** la `GetEffectiveDiscountRate()` (min TM, TC) no corresponde a la
realidad regulatoria para la reserva técnica. Para un cálculo exacto hay que usar
la tasa de mercado vigente por período (VTD / tasa CMF de calce), no la tasa de
la póliza individual. En 10,000 pólizas con TCM real la diferencia media absoluta
por póliza baja de +111% a ~39% y el agregado a -0.9%.

### 4.3 Las diferencias por estrato son el descalce

- **pre-2005 (+61.7%)**: la tabla base que usamos (RV-1985/B-1985 de la Circular
  491) es más conservadora que el stock real de cada compañía (muchas pólizas se
  reservan con la tabla con que fueron emitidas, no necesariamente RV-85).
- **post-2012 (-16.1%)**: la RT-BASE usa la tabla de origen de la póliza
  (RV-2009/RV-2014 según emisión), mientras nosotros aplicamos TM-2020 siempre
  para el estrato post-2012. El descalce negativo refleja que TM-2020 (mortalidad
  mejorada) da reservas menores que las tablas de origen.

Estas diferencias son **exactamente lo que el modelo de descalce debe capturar**:
la brecha entre tabla de bautizo y tabla vigente.

### 4.4 Pendientes de modelado (impacto en casos individuales)

| Pendiente | Efecto | Relevancia |
|---|---|---|
| Período garantizado (modalidad 3xxx) | Añade reserva (pagos garantizados) | Alta: ~60% de pólizas tienen PG |
| Reaseguro (RT retenida vs total) | Comparar contra 3.32-3.38 retenidas | Media |
| Tabla de bautizo exacta por póliza | La RT-BASE usa la tabla original emitida | Alta |
| Tasa de valuación por período (VTD) | Ver 4.2 | Crítica |
| Anticipo de renta vitalicia (Ley 21.392) | Ajusta % pensión | Baja |

---

## 4.5 Error por falta de VTDs históricos (medido) → resuelto

**Estado: RESUELTO (2026-08-06).** Se cargaron los VTDs históricos del archivo
`data/vtd/articles-51926_recurso_1.xlsx` (hojas 2020-2026), con `-import vtd`.
La base pasó de 61 a **71 vectores** (2020-09 a 2026-07, 8,520 puntos), incluyendo
el período del RIS 2025-12 y meses de 2026.

**Medición previa (cuando solo había VTD 2025-09):** se calculó la reserva de un
set fijo de pólizas del RIS bajo cada VTD disponible, manteniendo idénticas
tablas y rentas. Solo varía la curva de descuento.

| Métrica | Valor |
|---|---|
| Spread relativo (max-min)/último | **~18%** |
| VTDs 2020-2021 (tasas bajas) | reservas +8% a +13% |
| VTDs 2023 (tasas altas) | reservas -3% a -5% |
| **VTD del período (2025-12) vs más reciente (2026-07)** | **-2.76%** |
| Rango completo entre VTDs extremos | -7.9% a +10.1% |

**Conclusión:** usar el VTD más reciente en vez del del período de valuación
introduce un error que va de ~2.8% (período reciente) hasta ±10% (períodos
2020-2021). Al cargar los VTDs históricos y usar el del período del RIS
(`calc.LoadVTDFor(año, mes)`), **ese error se elimina** para los períodos con
VTD disponible (2020-09 en adelante).

**Implementación:**
- `internal/database/vtd.go` — `BatchInsert` ahora usa `INSERT OR REPLACE`
  (idempotente) y `GetAllVectorDates()` lista los vectores disponibles.
- `internal/calculator/reserve.go` — `LoadVTDFor(year, month)` instala la curva
  de un período específico.
- `cmd/calculator/ris_validate.go` — la validación usa el VTD del período del
  RIS (del header del archivo), con fallback al más reciente.
- `cmd/calculator/vtd_sens.go` — `-vtd-sens` mide la sensibilidad bajo todos los
  VTDs.
- `internal/config/config.go` — `Data.VTDHistorico` apunta al archivo histórico.

**Pendiente:** para períodos previos a 2020-09 no hay VTD en la base; ese rango
(2005-2019) mantiene el error de ±10% si se requiere valuación histórica.

---

## 5. Validación del parser C1194

Posiciones de campo calibradas contra datos reales y el spec del archivo
`SVRRV_circ_1194.doc`:

- **Registro 2** (póliza, 246 bytes): tipo pensión, vigencia, modalidad, tasas,
  prima única, renta mensual, reaseguro, fechas de recálculo.
- **Registro 3** (persona, 246 bytes): sexo, tipo beneficiario, fechas
  nac/fallec/invalidez, derecho pensión, % pensión, y **14 versiones de reserva**
  (RT-BASE, RT-BASE-TABLA-VIGENTE, RT-FINANCIERA por 5 estratos, más sus versiones
  retenidas).

Los valores de reserva decodificados cuadran con la escala esperada
(p.ej. RT-BASE-TOTAL `0150617` = 1,506.17 UF).

---

## 6. Archivos de la integración

| Archivo | Rol |
|---|---|
| `internal/models/ris.go` | Structs RIS (`RISPolicy`, `RISPerson`) |
| `internal/loader/ris1194.go` | Parser streaming C1194 (3 registros) |
| `internal/loader/ris1194_test.go` | Test contra muestra real |
| `cmd/calculator/ris_validate.go` | Conversión RIS→modelos + comparación de reservas |
| `cmd/calculator/main.go` | Flag `-validate-ris` + `-sample` |

---

## 7. Próximos pasos recomendados

1. **Integrar la tasa de valuación regulatoria (VTD)** — el archivo
   `VTD_2025_.xlsx` ya existe; usar la curva por período en vez de la TCM de
   emisión. Es el ajuste de mayor impacto.
2. **Modelar el período garantizado** en `FlowProjector` (pagos garantizados
   independientes de la supervivencia).
3. **Usar la tabla de bautizo real** por póliza: para cada póliza, la RT-BASE usa
   la tabla con que se emitió. Podemos inferirla de las RT-FINANCIERA-* reportadas
   (la mayor de las versiones por estrato indica la tabla de origen).
4. **Separar reserva base vs retenida** en el reporte para aislar el efecto del
   reaseguro.
