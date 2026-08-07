# Calculadora de Reservas de Rentas Vitalicias

Motor actuarial en Go para calcular reservas técnicas (VPPj) de pólizas de renta
vitalicia del sistema chileno, siguiendo la normativa de la CMF (NCG 318,
Circular 2332, Circular 1512) y la Nota Técnica N°9 de SPensiones.

El motor se valida contra el archivo RIS (Circular 1194) reportado por las
compañías a la CMF, comparando la reserva calculada contra la reportada.

## Estado de la validación

Validación contra RIS 2025-12-31 (959.664 pólizas, 2,15M de beneficiarios):

| Tipo de familia            | Diferencia vs reportado |
|----------------------------|-------------------------|
| causante solo              | **+2.3%** ✓             |
| con cónyuge                | **+1.9%** ✓             |
| con hijos                  | +21.9%                  |
| cónyuge + hijos            | +11.1%                  |
| sin causante vivo (vivos)  | +28.1%                  |
| **Global (10K muestra)**   | **+8.5%**               |

El motor está calibrado para pólizas modernas (post-sep-2020, con VTD del mes de
emisión). El gap residual se concentra en **stock pre-2020 sin VTD histórico
disponible** — ver [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md)
para el detalle y los próximos pasos.

## Stack

- **Go 1.25** (paquete único `reservas`, sin binarios externos)
- **SQLite** (`data/reservas.db`) para tablas de mortalidad, VTD, factores de
  mejoramiento y resultados
- `github.com/shopspring/decimal` para aritmética exacta en UF
- `github.com/xuri/excelize/v2` para leer xlsx (tablas CMF)
- `github.com/mattn/go-sqlite3` driver

## Construcción

```bash
go build -o reservas ./cmd/calculator
```

Binario único `reservas` (Linux/macOS/Windows). Requiere CGO por sqlite3.

## Comandos principales

Todos los comandos son flags del binario `reservas`:

```bash
# Inicializar / migrar DB
./reservas -init
./reservas -migrate

# Importar datos normativos (xlsx)
./reservas -import data/normativo/articles-20210_tablas_mort_hist.xlsx
./reservas -import data/vtd/articles-51926_recurso_1.xlsx

# Ver estadísticas de la DB
./reservas -stats

# Calcular una póliza
./reservas -calc 1
./reservas -calc-export 1            # exporta flujos a Excel

# Generador de portafolio y stress test
./reservas -gen-ris 1
./reservas -stress 1000

# Escenarios YAML
./reservas -scenario configs/escenario.yaml
./reservas -scenario-all

# Validación contra RIS (lo principal del proyecto)
./reservas -validate-ris /path/to/ris20251231.vta -sample 10000
./reservas -validate-ris ... -retenida            # comparar contra reserva neta
./reservas -validate-ris ... -no-mejoramiento     # sin AAx (sensibilidad)
./reservas -validate-ris ... -debug-svs 099747    # detalle por persona

# Sensibilidad al VTD (todas las curvas cargadas)
./reservas -validate-ris ... -vtd-sens -sample 200
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

Para inicializar la DB desde cero:

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

3. **Archivo RIS** (Circular 1194): `data/ris20251231.zip` (81MB, no commiteado).
   Descomprimir y pasar la ruta a `-validate-ris`.

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
| Mejoramiento AAx (qx,año = qx,2020 × Π(1-AAx,t)) | Circular 2332 eq.2 | ✓ (TM-2020) |
| Ajuste -11/24 (mensualización) | NT9 eq.7 | ✓ |
| Período garantizado (modalidad 3xxx/4xxx) | contractual | ✓ (causante) |
| Tasa min(TM, TV) de emisión | NCG 318 §2.3a | ✓ (2012-may2015) |
| TCj con VTD del mes de emisión | NCG 318 §2.2 | ✓ (post-sep2020) |
| TCj para jun2015-nov2020 | NCG 318 §2.2 | ✗ (requiere VTD histórico) |
| PG a sobrevivientes tras muerte del causante | contractual | pendiente |

## Documentación

- [docs/analysis/observaciones_avance.md](docs/analysis/observaciones_avance.md) — estado actual, evolución del gap, próximos pasos
- [docs/analysis/validacion_ris_1194.md](docs/analysis/validacion_ris_1194.md) — análisis de la validación
- [docs/ris_simulado_diseno.md](docs/ris_simulado_diseno.md) — diseño del parser RIS
- [docs/normative_framework.md](docs/normative_framework.md) — marco normativo
- [docs/mortality_tables_guide.md](docs/mortality_tables_guide.md) — guía de tablas
- [docs/normativo/](docs/normativo/) — PDFs originales (NCG, Circulares, NT9)

## Tests

```bash
go test ./internal/...
```

Cobertura: Cuadro 4 (`tabla_test.go`), PG (`pg_test.go`), casos RIS individuales
(`ris_case_test.go`), parser RIS (`ris1194_test.go`), generador
(`generator_test.go`).
