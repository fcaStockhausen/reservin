# Documentación

El proyecto tiene una sola fuente de verdad viva y un set cerrado de documentos
de referencia. Cualquier análisis temporal o WIP vive en `analysis/` (raíz,
**gitignoreado**) y no se commitea.

## Fuente de verdad viva

- [`analysis/observaciones_avance.md`](analysis/observaciones_avance.md) en el
  repo raíz — estado actual, hallazgos, bugs corregidos, próximos pasos. Es lo
  primero que debe leer quien toca el motor.

  > Nota: la versión tracked está en `docs/analysis/observaciones_avance.md`.
  > `analysis/` (raíz) es local-only.

## Documentos commiteados

### Diseño y normativa

- [`docs/ris_simulado_diseno.md`](ris_simulado_diseno.md) — diseño del parser RIS (Circular 1194)
- [`docs/normative_framework.md`](normative_framework.md) — marco normativo CMF
- [`docs/technical_specifications.md`](technical_specifications.md) — especificaciones técnicas
- [`docs/mortality_tables_guide.md`](mortality_tables_guide.md) — guía de tablas de mortalidad

### Análisis (curado, aporta al desarrollo)

- [`docs/analysis/observaciones_avance.md`](analysis/observaciones_avance.md) — estado, evolución del gap, próximos pasos
- [`docs/analysis/validacion_ris_1194.md`](analysis/validacion_ris_1194.md) — análisis de la validación contra RIS
- [`docs/analysis/mathematical_formulas.md`](analysis/mathematical_formulas.md) — fórmulas actuariales
- [`docs/analysis/reserve_calculation_guide.md`](analysis/reserve_calculation_guide.md) — guía de cálculo (NCG 318)

### Normativa primaria (PDFs y summaries)

- [`docs/normativo/`](normativo/) — PDFs originales CMF/SPensiones + summaries generados

  - NCG 318, NCG 209, NCG 374 — cálculo de reservas
  - Circular 2332, Circular 1512, Circular 491 — tablas y metodología
  - Nota Técnica N°9 (SPensiones) — fórmulas del CNU y Cuadro 4
  - Manual Usuario C1194, Oficio TM 1388/2025, VTD 2025

## Quick reference

| Necesito... | Ver |
|---|---|
| Estado actual del proyecto | `docs/analysis/observaciones_avance.md` |
| Cómo se calcula la reserva | `docs/analysis/mathematical_formulas.md` + `docs/analysis/reserve_calculation_guide.md` |
| Qué normativa aplica | `docs/normative_framework.md` + `docs/normativo/` |
| Cómo se parsea el RIS | `docs/ris_simulado_diseno.md` |
| Reglas para contribuir | [`AGENTS.md`](../AGENTS.md) en el raíz |
