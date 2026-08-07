# Reservín

> *La clave está en lo simple.*

Motor actuarial en Go para calcular reservas técnicas (VPPj) de pólizas de renta
vitalicia del sistema chileno. **Fast, multinúcleo, performance-driven**: proyecta
**~1.350 pólizas/seg** en paralelo y valida el RIS completo (959K pólizas) en
**~10 minutos**.

Sigue la normativa de la CMF (NCG 318, Circular 2332, Circular 1512) y la Nota
Técnica N°9 de SPensiones. Se valida contra el archivo RIS (Circular 1194) que
reportan las compañías a la CMF.

[![Go Version](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Benchmark](https://img.shields.io/badge/performance-~1.35K%20p%C3%B3lizas%2Fs-blue)](#performance)

---

## Estado de la validación

Validación contra **RIS completo 2025-12-31** (794.187 pólizas procesadas de
959.664 sampleadas, 2,15M de beneficiarios):

| Tipo de familia            | Diferencia vs reportado |
|----------------------------|-------------------------|
| causante solo              | **+2.6%** ✓             |
| con cónyuge                | **+2.3%** ✓             |
| con hijos                  | +22.8%                  |
| cónyuge + hijos            | +11.8%                  |
| sin causante vivo          | +26.5%                  |
| **Global**                 | **+8.68%**              |

El motor está calibrado para pólizas modernas (post-sep-2020 con VTD del mes de
emisión: gap <3% por póliza). El gap residual se concentra en **stock pre-2020
sin VTD histórico disponible** — ver
[docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md).

## Quick start

```bash
# Clona y entra al proyecto
git clone https://github.com/fcaStockhausen/reservin.git
cd reservin

# La DB ya viene cargada en el repo (tablas CMF + 1 póliza demo).
# Para construir:
go build -o reservin ./cmd/calculator

# Probar que anda (calcula la póliza demo):
./reservin -calc 1

# Validar contra un archivo RIS real (descargar aparte, ver "Datos de origen"):
./reservin -validate-ris /path/to/ris20251231.vta -sample 10000
```

## Stack

- **Go 1.25** — `github.com/fcaStockhausen/reservin`, sin binarios externos
- **SQLite** (`data/reservas.db`) — tablas de mortalidad, VTD, factores de
  mejoramiento, resultados
- `github.com/shopspring/decimal` — aritmética exacta en UF
- `github.com/xuri/excelize/v2` — lectura de xlsx (tablas CMF)
- `github.com/mattn/go-sqlite3` — driver SQLite (requiere CGO)

## Comandos principales

Todos los comandos son flags del binario `reservin`:

```bash
# Inicializar / migrar DB (ya viene migrada al clonar, pero por si acaso)
./reservin -init
./reservin -migrate

# Importar datos normativos (xlsx) — solo si quieres recargar desde cero
./reservin -import data/normativo/articles-20210_tablas_mort_hist.xlsx
./reservin -import data/vtd/articles-51926_recurso_1.xlsx

# Ver estadísticas de la DB
./reservin -stats

# Calcular una póliza
./reservin -calc 1
./reservin -calc-export 1            # exporta flujos a Excel

# Generador de portafolio y stress test
./reservin -gen-ris 1
./reservin -stress 1000

# Escenarios YAML
./reservin -scenario configs/escenario.yaml
./reservin -scenario-all

# Validación contra RIS (lo principal del proyecto)
./reservin -validate-ris /path/to/ris20251231.vta -sample 10000
./reservin -validate-ris ... -retenida            # comparar contra reserva neta
./reservin -validate-ris ... -no-mejoramiento     # sin AAx (sensibilidad)
./reservin -validate-ris ... -debug-svs 099747    # detalle por persona

# Sensibilidad al VTD (todas las curvas cargadas)
./reservin -validate-ris ... -vtd-sens -sample 200
```

## Estructura del proyecto

```
cmd/
  calculator/         # binario principal (main.go + ris_validate.go + vtd_sens.go)
config/               # config.json
data/
  reservas.db         # SQLite (tablas, VTD, factores, resultados)
  normativo/          # xlsx originales (tablas CMF, factores AAx)
  vtd/                # xlsx con curvas VTD
  migrations/         # esquema SQL versionado
docs/
  analysis/           # observaciones_avance.md, validacion_ris_1194.md
  normativo/          # PDFs (NCG 318, Circular 2332/1512/491, NT9, CMF)
  ris_simulado_diseno.md
  normative_framework.md
  technical_specifications.md
  mortality_tables_guide.md
internal/
  calculator/         # motor: ReserveCalculator, FlowProjector, MortalityEngine
  database/           # repos SQLite (mortality, vtd, factor_mejoramiento, migrations)
  loader/             # parsers: RIS C1194, mortality xlsx, VTD, Circular 491
  models/             # Policy, GrupoFamiliar, Beneficiario, RISPerson, tablas (Cuadro 4)
  portfolio/          # generador de portafolios sintéticos
  scenario/           # simulador de escenarios
  config/             # lectura de config.json
scripts/              # utilidades varias
```

## Datos de origen

Al clonar el repo, la DB ya viene cargada con tablas normativas + 1 póliza demo.
Para recargar desde cero o conseguir datos adicionales:

1. **Tablas de mortalidad** (Circular 491 + históricas CMF):
   `data/normativo/articles-20210_tablas_mort_hist.xlsx`. Incluye:
   - B-85, RV-85 (Circular 491)
   - RV-2004, B-2006, MI-2006
   - RV-2009, B-2009
   - CB-2014, RV-2014, B-2014, MI-2014
   - CB-2020, RV-2020, B-2020, MI-2020
   - Factores de mejoramiento AAx (2021-2036) para tablas 2020

2. **Curvas VTD** (vectores de descuento, 120 períodos cada uno):
   `data/vtd/articles-51926_recurso_1.xlsx`. Cubre 2020-09 a 2026-07.

3. **Archivo RIS** (Circular 1194): `data/ris20251231.zip` (81MB, **no commiteado**,
   en `.gitignore`). Descargar aparte de la CMF, descomprimir y pasar la ruta a
   `-validate-ris`.

## Modelo actuarial

La reserva se calcula como valor presente de flujos probabilísticos:

```
causante:      R · tpx_causante(t)
sobreviviente: R · pct · tpx_ben(t) · [1 - tpx_causante(t)]
```

Componentes normativos implementados:

| Componente | Normativa | Estado |
|---|---|---|
| Cuadro 4 (tablas por rol/vigencia) | NT9 SPensiones | ✓ |
| Mejoramiento AAx (`qx,año = qx,2020 × Π(1-AAx,t)`) | Circular 2332 eq.2 | ✓ (TM-2020) |
| Ajuste -11/24 (mensualización) | NT9 eq.7 | ✓ |
| Período garantizado (modalidad 3xxx/4xxx) | contractual | ✓ (causante) |
| Tasa `min(TM, TV)` de emisión | NCG 318 §2.3a | ✓ (2012-may2015) |
| TCj con VTD del mes de emisión | NCG 318 §2.2 | ✓ (post-sep2020) |
| TCj para jun2015-nov2020 | NCG 318 §2.2 | ✗ (requiere VTD histórico) |
| PG a sobrevivientes tras muerte del causante | contractual | pendiente |

Para más detalle ver [AGENTS.md](AGENTS.md) (contexto normativo + decisiones de
diseño) y [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md).

## Performance

**~1.350 pólizas por segundo.** El RIS completo (959.664 pólizas, 794K procesadas)
se valida en **9 min 47 s** en un Apple Silicon; 50K muestra en 32 s.

Medición real sobre el archivo RIS 2025-12-31 (733 MB), con tablas, mejoramiento
AAx, VTD del mes de emisión y comparación contra reservas reportadas.

El motor es paralelo por diseño: cada póliza se calcula independiente, las
tablas y el VTD se cachean en memoria, y el RIS se streamea con canales para no
cargar 959K registros en RAM.

```bash
go build -o reservin ./cmd/calculator
time ./reservin -validate-ris /path/to/ris20251231.vta -sample 1000000
```

El gap global del archivo completo es **+8,68%** vs reportado (pre-2005 +29,6%,
post-2012 +2,9%) — ver [Estado de la validación](#estado-de-la-validación).

## Documentación

- [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md) — estado actual, evolución del gap, próximos pasos
- [docs/analysis/validacion_ris_1194.md](docs/analysis/validacion_ris_1194.md) — análisis de la validación
- [docs/ris_simulado_diseno.md](docs/ris_simulado_diseno.md) — diseño del parser RIS
- [docs/normative_framework.md](docs/normative_framework.md) — marco normativo
- [docs/mortality_tables_guide.md](docs/mortality_tables_guide.md) — guía de tablas
- [docs/normativo/](docs/normativo/) — PDFs originales (NCG, Circulares, NT9)
- [AGENTS.md](AGENTS.md) — guía para contribuir (convenciones, bugs corregidos, decisiones)

## Tests

```bash
go test ./internal/...
```

Cobertura: Cuadro 4 (`tabla_test.go`), PG (`pg_test.go`), casos RIS individuales
(`ris_case_test.go`), parser RIS (`ris1194_test.go`), generador
(`generator_test.go`).

## Contribuir

1. Lee [AGENTS.md](AGENTS.md) antes de tocar el motor — tiene las reglas, los
   bugs ya corregidos, y el contexto normativo.
2. Abre un issue primero para cambios grandes.
3. `gofmt -w` + `go build ./...` + `go test ./internal/...` antes de commit.
4. Commits en inglés, describiendo el cambio.

## Licencia

[MIT](LICENSE) — © Felipe Caraneda Stockhausen.

---

*Reservín calcula cuánta plata guardar para que alcance hasta el final. La
reserva es la partida que la compañía mantiene viva mientras la vida misma
sigue su curso.*
