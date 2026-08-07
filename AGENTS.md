# AGENTS.md

Guía para cualquier agente (humano o IA) que trabaje en este repo. Léela antes
de tocar nada.

## Qué es este proyecto

Motor actuarial en Go que calcula reservas técnicas (VPPj) de rentas vitalicias
del sistema chileno y las valida contra el RIS (Circular 1194) que reportan las
compañías a la CMF. El objetivo actual es **cerrar el gap** entre la reserva
calculada y la reportada.

**Lee primero:** [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md)
— tiene el estado actual, los hallazgos clave, los bugs ya corregidos y los
pendientes. **No toques el motor sin haberlo leído.**

## Reglas del proyecto

- **No commitear** `.DS_Store`, `*.xlsx` (excepto `data/normativo/`, `data/vtd/`,
  `docs/normativo/`), `cpu.prof`, `prof`, `prof/`, `cmd/prof/`, `R*.txt`,
  `data/*.zip`, `data/*.vta`. Están en `.gitignore`.
- **No commitear la DB** `data/reservas.db` con datos personales o RIS crudo.
  Si la DB cambia por una migración, sí subirla (es pequeña, ~2.5MB).
- **Idioma**: los commits van en inglés (siguiendo el estilo del log). Código y
  comentarios en inglés o español según el archivo circundante — mira el archivo
  antes de añadir comentarios.
- **No añadir comentarios explicativos** salvo que el usuario lo pida. El código
  se explica solo.
- **No commitear nada sin que el usuario lo pida explícito.**

## Comandos esenciales

```bash
# Build
go build -o reservas ./cmd/calculator
go build ./...                          # verificar compila todo

# Tests (rápidos, <5s)
go test ./internal/calculator/ ./internal/models/ ./internal/portfolio/ ./internal/loader/ -timeout 60s

# Tests + build + vet (correr antes de commit)
go build ./... && go vet ./internal/... ./cmd/... && go test ./internal/...

# Formato
gofmt -w internal/ cmd/

# Validación RIS (10K pólizas, ~7s)
./reservas -validate-ris /tmp/ris_extract/ris20251231.vta -sample 10000

# Debug de una póliza específica
./reservas -validate-ris /tmp/ris_extract/ris20251231.vta -sample 20000 -debug-svs 099747

# Sensibilidad (sin mejoramiento AAx)
./reservas -validate-ris ... -no-mejoramiento
```

## Dónde están las cosas

```
internal/calculator/        # MOTOR
  reserve.go                #   ReserveCalculator.CalculateAt — entry point
  flow.go                   #   FlowProjector.Project — proyección de flujos
  mortality.go              #   MortalityEngine + mejoradaQx (Circular 2332)
internal/models/
  tabla.go                  #   Cuadro 4 (SelectBaseTable, cuadro4Table)
  ris.go                    #   RISPerson (campos C1194), ReserveForComparison
  policy.go                 #   Policy.GetEffectiveDiscountRate (min TM/TC)
  beneficiario.go           #   Roles, InvNo/InvTotal/InvParcial
internal/loader/
  ris1194.go                #   Parser streaming del RIS (fijo ancho)
  mortality.go              #   Parser xlsx (tablas + AAx)
  circular_491.go           #   Tablas 1985 (RV-85, B-85)
internal/database/
  migrations.go             #   Esquema versionado (v1..v4)
  factor_mejoramiento.go    #   Repo AAx (Circular 2332)
  vtd.go                    #   Repo VTD (vectores de descuento)
  mortality.go              #   Repo tablas
cmd/calculator/
  main.go                   #   CLI flags
  ris_validate.go           #   validateRIS, computeRISReserve, buildGrupoFromRIS
  vtd_sens.go               #   Sensibilidad al VTD
```

## Contexto normativo clave

Antes de tocar el cálculo de la tasa o las tablas, entender esto:

### Tasa de descuento por cohorte (NCG 318)

La "tasa de bautizo" **cambia de tipo** según la cohorte de emisión:

| Cohorte                  | Tasa                                   | Fuente del dato |
|--------------------------|----------------------------------------|-----------------|
| pre-2012                 | TM + calce (Circular 1512)             | RIS TasaCostoEmision (aprox) |
| 2012-may2015             | min(TM, TV)                            | RIS TasaCostoEmision/TasaVenta |
| **jun2015-nov2020**      | **TCj** (TIR con VTD del mes emisión)  | **VTD histórico** |
| **post-dic2020**         | min(TVj, TCj)                          | **VTD histórico** + RIS TasaVenta |

`TasaCostoEmision` del RIS es la **TM**, NO TCj. Para cohortes post-jun2015 se
necesita el VTD del mes de emisión. Solo tenemos VTD desde 2020-09 — el resto
cae a flat rate y el gap se mantiene.

### Reserva comparable del RIS (Circular 1194)

| Cohorte  | Campo del RIS               | Notas |
|----------|-----------------------------|-------|
| pre-2012 | RT-FINANCIERA-2020 (3.31b)  | Stock revaluado a TM-2020 |
| post-2012| RT-BASE-TABLA-VIGENTE (3.26)| Tabla nativa ya CB-2014/2020 |

Implementado en `models/ris.go` `ReserveForComparison(contractDate)`.

### Tablas por rol (Cuadro 4, NT9)

- **Afiliado (RV/CB)**: hombres CB-H-XXXX, mujeres RV-M-XXXX.
- **Beneficiario (B/CB)**: hombres CB-H-XXXX, mujeres B-M-XXXX.
- **Inválido (MI)**: MI-H/M-XXXX. **MI-1985 no existe** (Circular 491 no la
  definió) — los inválidos pre-2005 caen a B-1985.

Códigos de invalidez del RIS: `N` (no), `T` (total), `P` (parcial). **NO "I"**.

## Bugs conocidos ya corregidos (no volver a meterlos)

1. **Exclusión de cónyuges** (commit 6a9a723): el filtro `DerechoPension == "99"
   || "20"` en `buildGrupoFromRIS` y en el loop de reserva reportada excluía a
   los cónyuges (DerechoPension="10"). Usar `rol` en vez de `DerechoPension`.
2. **PG en cónyuge fallecido**: en el branch de causante fallecido, resetear
   `polNoPG.ModalidadRenta = "1000"` para que `flow.go` no aplique el PG original
   al cónyuge tratado como causante.
3. **Deadlock del select**: en loops `for { select { case ...: continue } }`,
   usar `break` (no `continue`) tras cerrar un canal — `continue` salta el chequeo
   `if policies == nil && errs == nil { break }` y deadlockea.
4. **Errores swallowed en flow.go**: NO tragar errores de `qxMejorada` con
   `cSurv = zero` silencioso. Propagar (`return nil, err`). Edades fuera del
   rango de la tabla se tratan como qx=1 (muerte cierta), no error.
5. **MI-1985 inexistente**: caer a B-1985 en `cuadro4Table`.
6. **SituacionInvalidez**: comparar `"T"`/`"P"`, no `"I"`.

## Decisiones de diseño

- **Aritmética**: `decimal.Decimal` en todo cálculo monetario. NUNCA `float64`
  para reserva/renta/prima. Solo `float64` para reportes agregados (suma de
  reservas) y porcentajes.
- **Tablas cacheadas**: `MortalityEngine` carga cada tabla una vez en memoria
  (`map[tableName]map[edad]qx`). VTD se cachea por "YYYY-MM" en
  `ReserveCalculator.vtdCache` para no re-query la DB por póliza.
- **Stream del RIS**: el parser usa canales (`Stream(maxPolicies) (<-chan RISPolicy, <-chan error)`)
  para no cargar 959K pólizas en RAM.
- **Off-by-one edad**: `currentYear = valuationDate.Year() - contractDate.Year()`
  cuenta años calendarios, no años cumplidos. Hay ~1 año de error para personas
  que no habían cumplido años al contrato. Pendiente de revisar.

## Pendientes / próximos pasos

Ver [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md)
sección "Próximos pasos prioritarios". Resumen:

1. **Conseguir VTD histórico pre-2020-09** (publicación CMF) — bloqueante para
   cerrar el gap en stock 2015-2020.
2. Investigar `con_hijos` (+22%) — posible bug en porcentaje de hijos o tabla.
3. Implementar TCj iterativo (TIR con VTD) cuando el VTD no esté en DB.
4. PG a sobrevivientes tras muerte del causante (mejora, no bug).
5. Tests de mejoramiento (cobertura nula hoy).

## Datos de prueba

- **Sample**: `/tmp/ris_sample.vta` (500 pólizas) y `/tmp/ris_extract/ris20251231.vta`
  (archivo completo 768MB). Si no están, descomprimir `data/ris20251231.zip`.
- **DB**: `data/reservas.db` ya cargada con tablas + AAx + VTD 2020-09..2026-07.
- **Casos verificados**:
  - SVS=853852 (post-sep-2020): diff -2.46% ✓ (VTD cargado)
  - SVS=099747 (2016): diff +40% (sin VTD histórico, gap esperado)
  - SVS=730135 (causante solo): diff +8.9% (pre-cuadro-4)

## Cuando touches el motor

1. Lee `docs/analysis/observaciones_avance.md`.
2. Haz el cambio mínimo. No refactorices de paso.
3. `gofmt -w` los archivos tocados.
4. `go build ./... && go vet ./internal/... ./cmd/...`
5. `go test ./internal/...`
6. Re-correr `./reservas -validate-ris ... -sample 10000` y comparar el gap
   con el baseline (+8.5% global). Reportar si empeora.
7. Si el cambio toca normativa, actualiza `observaciones_avance.md`.
