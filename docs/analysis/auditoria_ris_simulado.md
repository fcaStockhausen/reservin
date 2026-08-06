# Auditoria del diseno RIS Simulado (`ris_simulado_diseno.md`)

**Fecha:** 2026-08-05
**Fuentes de validacion:** Anexo Tecnico Circular 1194 (55 paginas, CMF), codigo implementado en repo, normativa D.L. 3500 / NCG 318.

---

## 1. Verdict general

El documento es **solido como investigacion regulatoria y propuesta de arquitectura**, pero tiene tres problemas serios:

1. **Esta desactualizado respecto al codigo**: describe cosas como pendientes que ya estan implementadas (grupo familiar, VTD real, motor de flujos).
2. **El modelo de datos propuesto (`beneficiario`)** colisiona con la tabla que ya existe en produccion con otra estructura.
3. **La nomenclatura "en curso/probables/posibles" no existe en el RIS**: es una clasificacion actuarial/FECU interna, no un campo del reporte regulatorio. Hay que alinear con los campos reales del C1194.

---

## 2. Validacion contra el Anexo Tecnico C1194 (RIS real)

### 2.1 Estructura del RIS (confirmada)

| Registro | Contenido | Campos |
|---|---|---|
| Tipo 1 | Header / control del archivo | Periodo, compania |
| Tipo 2 | P / siniestro (1 por causante) | 2.1 a 2.38: tipo pension, vigencia, modalidad, tasas, reaseguro |
| Tipo 3 | Afiliado + beneficiarios (N por registro 2) | 3.1 a 3.40+: identificacion, tipo beneficiario, % pension, reservas |
| Tipo 4 | Totales del archivo | Sumatorias de control |

**Diseno dice:** "Fila poliza / Fila asegurado-beneficiario / Fila renta vitalicia".  
**Realidad:** Son registros tipo 2 (poliza) y tipo 3 (personas), no tres tablas separadas. Las "rentas vitalicias" son campos dentro de los registros 2 y 3, no un archivo aparte.

### 2.2 Codigos de TIPO-BENEFICIARIO (Registro 3, campo 3.10)

El diseno inventa codos (`CONYUGE`, `HIJO`, etc.). Los codigos reales del C1194 son **numericos**:

| Codigo C1194 | Significado | Diseno propuesto |
|---|---|---|
| 99 | Afiliado (causante) | `CAUSANTE` |
| 10 | Conyuge sin hijos con derecho | `CONYUGE` |
| 11 | Conyuge con hijos con derecho | (no distingue) |
| 20 | Madre/padre no matrimonial sin hijos con derecho | `MADRE_PADRE_NMAT` |
| 21 | Madre/padre no matrimonial con hijos con derecho | (no distingue) |
| 30 | Hijo sin derecho a incremento | `HIJO` |
| 35 | Hijo con derecho a incremento | (no distingue) |
| 41 | Padre del causante | `PADRES` |
| 42 | Madre del causante | `PADRES` |
| 50 | Conviviente civil sin hijos comunes ni del causante | `CONVIVIENTE_CIVIL` |
| 51 | Conviviente civil con hijos comunes con derecho | (no distingue) |
| 52 | Conviviente civil con hijos del causante, no comunes | (no distingue) |
| 77 | Beneficiario designado | `DESIGNADO` |

**Problema:** El diseno agrupa codigos que el RIS reporta separadamente (10 vs 11, 30 vs 35, etc.). La distincion sin/con hijos determina el % de sobrevivencia y debe modelarse.

### 2.3 Campo DERECHO-PENSION (Registro 3, campo 3.15)

**Este es el campo que el diseno llama "en curso/probables/posibles" pero con otra denominacion.** Los codigos reales son:

| Codigo | Significado | Equivalencia actuarial |
|---|---|---|
| 99 | Tiene derecho a pension | "En curso" (si recibiendo) o "Probable" (si causante vivo) |
| 10 | No tiene derecho a pension | Excluido / "posible" en el sentido de que existe pero no genera reserva |
| 20 | Derecho a pensin no acreditado | **Constituye reserva** (hijo >18 sin certificado de estudios) |

**Adicionalmente**, el campo 3.16 REQUISITO-PENSION captura excepciones: ex-conyuge (2), hijo sin derecho (4), conyuge post-poliza (5), etc.

**El diseno no menciona este campo.** Es critico: es como el RIS distingue beneficiarios activos de contingentes.

### 2.4 Campo VIGENCIA-PENSION (Registro 2, campo 2.8)

Controla el estado de pago de la poliza:

| Codigo | Significado |
|---|---|
| 6 | En pago (activa, con o sin periodo garantizado) |
| 7 | Pagando renta garantizada a designados (sin beneficiarios legales) |
| 8 | Diferida (no se esta pagando aun) |
| 9 | Extinguida (no se paga ni pagara) |

**El diseno no lo menciona.** Determina si una poliza genera reserva "en curso" o no.

### 2.5 Clausulas: campo MODALIDAD-RENTA (Registro 2, campo 2.18)

El RIS codifica clausulas como un solo campo numerico:

| Codigo | Significado |
|---|---|
| 1000 | Sin adicionales |
| 2xxx | Aumento temporal de pension (xxx = meses de aumento temporal) |
| 3xxx | Solo periodo garantizado (xxx = meses garantizados, xxx >= 001) |
| 4xxx | Aumento de % de sobrevivencia (xxx = meses garantizados si tambien tiene PG) |

Mas los campos:
- **2.20 PERIODO-AUMENTO**: meses del aumento temporal
- **2.21 PORCENTAJE-AUMENTO**: % de aumento temporal

**El diseno propone tabla `clausula` separada con JSON.** Eso es valido para el modelo interno, pero para generar RIS real hay que mapear a un solo campo `MODALIDAD-RENTA`.

### 2.6 Campos de reserva (Registro 3, campos 3.25-3.31)

El RIS reporta **multiple versiones de reserva por persona**:

| Campo | Que es |
|---|---|
| 3.25 RT-BASE-TOTAL | VPPj con tablas de origen |
| 3.26 RT-BASE-TABLA-VIGENTE-TOTAL | VPPj con tablas vigentes actuales |
| 3.27 RT-FINANCIERA-2004-85-TOTAL | RT financiera (RV-85/B-85/MI-85) |
| 3.28 RT-FINANCIERA-STOCK-RV-85-TOTAL | Stock heredado RV-85 |
| 3.29-3.31 | Otras RT financieras por periodo de tablas |

**El diseno solo contempla `reserva_calculada` con un valor.** Para RIS real hay que reportar minimo 2 versiones por persona (base + base tabla vigente).

### 2.7 Sexo

**Diseno usa:** `H`, `M`  
**RIS C1194 usa:** `M` (masculino), `F` (femenino)

**Inconsistencia a corregir.**

---

## 3. Validacion contra codigo implementado

### 3.1 Cosas que el diseno dice como pendientes pero YA ESTAN LISTAS

| Diseno dice | Estado real | Donde |
|---|---|---|
| "la tabla `poliza` solo modela un contratante y un beneficiario" | **FALSO**: tabla `beneficiario` existe desde migration v2 | `internal/database/migrations.go:112` |
| "El proyecto aun usa fallback [1,1,...]; pendiente importar VTD real" | **FALSO**: VTD real cargado, 7320 puntos, 2020-2025 | `vtd_vector` table, 61 vectores |
| "Motor de grupo familiar" como roadmap paso 4 | **IMPLEMENTADO**: `FlowProjector` proyecta flujos por miembro con logica de sobrevivencia | `internal/calculator/flow.go` |
| "Desglose en curso/probable/posible" como roadmap paso 5 | **PARCIAL**: motor ya desglosa VP por rol (causante/conyuge/hijo) | `internal/calculator/reserve.go` |

### 3.2 Tabla `beneficiario` propuesta vs existente

El diseno propone una tabla nueva con campos distintos a la que ya existe:

| Campo | Implementado (v2) | Diseno propuesto |
|---|---|---|
| `rol` | VARCHAR(20): CAUSANTE, CONYUGE, HIJO, OTRO | Idem + CONVIVIENTE_CIVIL, MADRE_PADRE_NMAT, PADRES, DESIGNADO |
| `sexo` | H, M | H, M (deberia ser M, F segun C1194) |
| `edad_contratacion` | INTEGER | No tiene |
| `fecha_nacimiento` | DATE | DATE |
| `tabla_asignada` | VARCHAR(50) | No tiene (la asigna el motor) |
| `porcentaje_renta` | DECIMAL(5,4) | `pct_sobrevivencia` DECIMAL(5,2) |
| `estado` | ACTIVO/FALLECIDO/EXCLUIDO | `es_rentista` BOOLEAN |
| | | `condicion` (MENOR/ESTUDIANTE/INVALIDO) |
| | | `matrimonio_anios` INTEGER |
| | | `hijos_comunes` INTEGER |
| | | `fin_derecho_edad` INTEGER |
| | | `invalidez_parcial` BOOLEAN |

**Problema:** Hay que hacer migration v3 que `ALTER TABLE` para agregar los campos nuevos, no crear la tabla de nuevo.

### 3.3 Tabla `poliza` faltante de campos RIS

El diseno pide agregar a `poliza`: `tipo_pension`, `modalidad`, `periodo_garantizado`, `vigencia_pension`. Estos son necesarios para RIS y no existen hoy.

---

## 4. Validacion regulatoria (D.L. 3500)

### 4.1 Porcentajes de sobrevivencia (seccion 2.3 del diseno)

**Verificado contra D.L. 3500 art. 5**: los porcentajes son correctos:
- Conyuge 60% sin hijos / 50% con hijos: correcto
- Hijo 15%: correcto
- Hijo invalido parcial 11% al cumplir 24: correcto
- Madre/padre no matrimonial 36%/30%: correcto
- Padres del causante 50%: correcto
- Derecho de acrecer: correcto

### 4.2 Requisitos del conyuge (seccion 2.2)

**Verificado**: 6 meses (activo) / 3 anos (pensionado), no aplica con hijos en comun o embarazo. Correcto.

**Observacion:** El diseno menciona "conyuge hombre solo tiene derecho desde 1/10/2008". Esto es correcto (Ley 20.255), pero **no se modela** en el diseno ni en el codigo.

### 4.3 Caso "abuelo renovado" (seccion 7)

**Verificado como caso de prueba valido**: conyuge casada hace <3 anos con causante pensionado, sin hijos en comun = sin derecho. Nacimiento de hijo en comun activa el derecho. Es un escenario de estres correcto y bien construido.

---

## 5. Hallazgos de la bibliografia de Christian

Christian menciono:
> "validar denominacion exacta de las categorias contra las fichas tecnicas SEIL/RIS"

**Resultado:** Confirmado que **"en curso/probables/posibles" NO es terminologia RIS**. El RIS usa:
- `VIGENCIA-PENSION` (campo 2.8): 6/7/8/9
- `DERECHO-PENSION` (campo 3.15): 99/10/20
- `PENSION-PERSONA` (campo 3.21): monto que efectivamente recibe

La clasificacion "curso/probable/posible" es una **construccion actuarial interna (FECU/NCLE)** para presentacion de estados financieros, no un campo de reporte.

**Recomendacion:** mantener la clasificacion para analisis interno pero mapear correctamente a los campos RIS reales al generar el archivo.

---

## 6. Acciones requeridas

### Prioridad alta (bloquean RIS real)

1. **Corregir codigos de sexo a M/F** (no H/M) en todo el modelo para alinear con C1194
2. **Agregar campos RIS faltantes a `poliza`**: `tipo_pension` (01-15), `modalidad_renta` (1000/2xxx/3xxx/4xxx), `vigencia_pension` (6/7/8/9), `periodo_aumento`, `porcentaje_aumento`
3. **Extender `beneficiario` con campos RIS**: `tipo_beneficiario_c1194` (99/10/11/20/21/30/35/41/42/50/51/52/77), `derecho_pension` (99/10/20), `requisito_pension` (1-9), `derecho_acrecer` (S/N), `situacion_invalidez` (N/T/P)
4. **Distinguir tipo 10 vs 11, 30 vs 35, 50 vs 51 vs 52** en el modelo (hoy agrupados)
5. **Agregar campos de reserva multiple** a `reserva_calculada`: `rt_base`, `rt_base_tabla_vigente`, `rt_financiera`

### Prioridad media

6. **Crear tabla `clausula`** pero con mapeo a `MODALIDAD-RENTA` C1194
7. **Crear generador RIS** que produzca registros tipo 1/2/3/4 en formato texto plano (longitud fija, encoding ASCII sin tildes)
8. **Modelar "derecho a pension no acreditado" (codigo 20)**: hijo >18 sin certificado de estudios que constituye reserva

### Prioridad baja

9. **Modelar conviviente civil (AUC)** completamente (codigos 50/51/52)
10. **Modelar reaseguro** (campos 2.24-2.33: compania, modo, fechas, % retenido)
11. **Modelar anticipo de RV** (campos 3.22-3.24: Ley 21.392)
12. **Validar tablas RV-85/B-85/MI-85** para polizas pre-2005 (no tenemos esos datos)

---

## 7. Resumen para Christian

> El diseno esta bien investigado regulatoriamente (D.L. 3500 correcto) pero tiene dos problemas: (a) describe como pendientes cosas que ya implementamos (grupo familiar, VTD, motor de flujos), y (b) los codigos de beneficiario que inventa no coinciden con los del RIS real (C1194 usa codigos numericos 10/11/20/21/30/35/etc., no strings como "CONYUGE").
>
> Baje y lei el Anexo Tecnico C1194 completo (55 paginas). La clasificacion "curso/probable/posible" no existe como campo en el RIS: el RIS reporta `DERECHO-PENSION` (99=tiene derecho / 10=no tiene / 20=no acreditado) y `VIGENCIA-PENSION` (6=activa / 7=designados / 8=diferida / 9=extinguida). Hay que mapear nuestra clasificacion a esos codigos.
>
> Las clausulas ("con/sin aumento") se codifican en un solo campo `MODALIDAD-RENTA` (1000/2xxx/3xxx/4xxx), no en tabla separada.
>
> Faltan 5 campos en `poliza` y ~6 campos en `beneficiario` para poder generar un RIS real. Es migration v3 + ajuste de codigos.
