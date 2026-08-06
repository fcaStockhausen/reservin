# Diseño del RIS Simulado para Reservas de Rentas Vitalicias

**Propósito:** definir cómo declarar "familias tipo" (grupos familiares) en un RIS simulado, según la normativa chilena (D.L. 3.500, CMF), para estresar el motor de cálculo de reservas de rentas vitalicias del proyecto.

**Alcance:** investigación regulatoria resumida + modelo de declaración de familias + esquema de datos SQLite extendido + dimensiones de estrés.

---

## 1. Objetivo

El motor actual (`internal/calculator`, NCG 318) calcula `VPP_j = Σ FP_ji × (1+TC_j)^-i` por póliza, pero la tabla `poliza` solo modela un contratante y un beneficiario. Las reservas reales de rentas vitalicias dependen del **grupo familiar completo** del causante (cónyuge, conviviente civil, hijos, padres) y de las **cláusulas adicionales** contratadas (período garantizado, aumentos).

Este diseño define:

1. Las características de los tipos de pensión en Chile (vejez, invalidez, sobrevivencia) y del grupo familiar (D.L. 3.500).
2. Cómo se clasifican las reservas: **pensiones en curso de pago**, **probables** y **posibles**.
3. Cómo se declara una "familia tipo" y se materializa en registros RIS (Registro de Información de Seguros).
4. El esquema de datos necesario (tablas nuevas sobre las existentes).
5. El barrido de escenarios para estresar el modelo.

---

## 2. Tipos de pensión en Chile (D.L. 3.500)

### 2.1 Pensiones del sistema

| Tipo | Descripción | Edad/requisito |
|---|---|---|
| **Vejez** | Pensión por edad | Hombres ≥ 65 años; mujeres ≥ 60 años |
| **Invalidez** | Pensión por pérdida de capacidad laboral | ≥ 2/3 de capacidad de trabajo (comisión médica) |
| **Sobrevivencia** | Pensión para el grupo familiar al fallecer el causante | Requisitos del art. 5° al 10° D.L. 3.500 |

Modalidades de pensión vigentes (SP/CMF): retiro programado (AFP), renta vitalicia inmediata (CSV), renta temporal + renta vitalicia diferida, y renta vitalicia inmediata con retiro programado. El motor de reservas aplica a la **renta vitalicia** (inmediata y diferida).

### 2.2 Grupo familiar beneficiario (D.L. 3.500, art. 5° a 10°)

Beneficiarios de pensión de sobrevivencia y sus requisitos:

| Rol | Requisitos para tener derecho |
|---|---|
| **Cónyuge** | Casado ≥ 6 meses antes del fallecimiento si era activo; ≥ 3 años si el causante ya era pensionado de vejez/invalidez. No rige si hay hijos en común o embarazo. El cónyuge hombre solo tiene derecho desde el 1/10/2008. |
| **Conviviente civil (AUC)** | Acuerdo de Unión Civil vigente ≥ 1 año (activo) o ≥ 3 años (pensionado). No rige si hay hijos en común o embarazo. |
| **Hijos** | Solteros y: menores de 18 años; o mayores de 18 y menores de 24 si son estudiantes regulares (básica/media/técnica/superior, Chile o extranjero reconocido); o inválidos de cualquier edad (invalidez declarada antes de los 18/24 años). |
| **Madre/padre de hijos no matrimoniales** | Soltero/a o viudo/a, que vivía a expensas del causante. |
| **Padres del causante** | Solo si no existen los beneficiarios anteriores y eran cargas familiares (asignación familiar) del causante. |

**Nota del caso de ejemplo:** el "abuelo jubilado que se casa con una 20 años menor" cae en el requisito de **3 años de matrimonio previo** si fallece siendo pensionado; si no cumple, la cónyuge no tiene derecho (salvo hijos en común o embarazo). Esta es justamente una condición límite a estresar.

### 2.3 Porcentajes de pensión de sobrevivencia (D.L. 3.500, art. 5°)

Sobre la pensión de referencia del causante:

| Beneficiario | % | Condición |
|---|---|---|
| Cónyuge/conviviente | **60%** | Sin hijos con derecho a pensión |
| Cónyuge/conviviente | **50%** | Con hijos comunes con derecho; sube a 60% cuando dejan de tener derecho |
| Madre/padre de hijos no matrimoniales | **36%** | Sin hijos con derecho a pensión |
| Madre/padre de hijos no matrimoniales | **30%** | Con hijos con derecho a pensión (sube a 36%) |
| Padres del causante | **50%** | Causantes de asignación familiar |
| Cada hijo | **15%** | Con derecho a pensión |
| Hijo inválido parcial al cumplir 24 años | **11%** | Invalidez parcial |
| Conviviente civil | **15%** | Solo con hijos del causante con derecho (no comunes) |

Reglas adicionales: el 60%/50% del cónyuge se incrementa al repartir el porcentaje de hijos que pierden el derecho (**derecho de acrecer**); si hay más de una madre, el porcentaje se divide en partes iguales; si no hay cónyuge, el porcentaje del cónyuge se reparte entre los hijos.

**Impacto en reservas:** el motor debe proyectar, para cada escenario de muerte del causante en cada periodo `i`, qué beneficiarios sobreviven y con qué porcentaje (según edades y vigencia de los requisitos: hijos que cumplen 18/24, cónyuge que cumplió los 3 años de matrimonio, etc.).

### 2.4 Cláusulas adicionales (SCOMP / NCG 218 / Circular 2062)

| Cláusula | Pensiones donde aplica | Efecto en flujos de reserva |
|---|---|---|
| **Período garantizado de pago** | Vejez, invalidez, sobrevivencia (RV inmediata, diferida, mixta) | Si el asegurado fallece antes de terminar el período, se paga el 100% de la renta a beneficiarios legales (o designados) por el tiempo restante. Típicamente 5/10/15/20 años. Reemplaza los porcentajes legales durante el período; al término, vuelven los % de sobrevivencia. |
| **Aumento de porcentaje de pensión de sobrevivencia** | Vejez, invalidez (requiere cónyuge) | Sube el % del cónyuge/beneficiarios sobre el legal (p. ej. cónyuge 100%). |
| **Aumento temporal / diferido vitalicio de pensión** | Vejez, invalidez (en conjunto solo con período garantizado) | Reduce la pensión temporalmente (p. ej. -33,3% o -66,7% durante 12/24/36/120 meses) y la aumenta vitaliciamente después. Afecta el flujo `FP_ji` en dos tramos. |

El clásico "con aumento / sin aumento" del usuario corresponde a estas cláusulas: **aumento de % de sobrevivencia** y **aumento temporal/diferido vitalicio**.

### 2.5 Hipótesis biométricas y financieras (proyecto)

- Tablas de mortalidad actuales (Circular 2332): **CB-H-2020** (causante y beneficiario hombre), **RV-M-2020** / **B-M-2020** (mujeres), **MI-H-2020** / **MI-M-2020** (invalidez). Edades 0–110.
- Tasa de descuento: `min(TM_j, TC_j)`; TM (tasa de venta) histórica disponible (última 2,41% sept 2025); TC implícita (tasa costo) por `VPP_j`.
- Vector de tasas: `VTD = ET + AV` (estructura temporal 120 años + ajuste por volatilidad). El proyecto aún usa fallback `[1,1,...]`; pendiente importar VTD real.
- Reajustabilidad: pensiones expresadas en UF (reajuste IPC). El flujo se proyecta en UF y se valora con tasa real.

### 2.6 Contexto regulatorio 2025–2026 (a vigilar)

- Ley 21.735 (Reforma Previsional): acceso a renta vitalicia desde pensión ≥ 2 UF (antes 3 UF); NCG 341 SP (sept 2025) elimina ofertas externas de RV.
- CMF abrió consulta (jun 2026) para modificar la **Circular 2062** (recálculo de pensiones por cambios en el grupo familiar) — afecta directamente la lógica de "recálculo por composición de grupo" que el RIS debe capturar.
- Nueva cláusula de **aumento diferido y vitalicio de pensión** (informe CMF, enero 2025) ya incorporada en SCOMP.

---

## 3. Clasificación de la reserva: en curso, probables y posibles

En la práctica actuarial/FECU de rentas vitalicias la reserva matemática se separa en:

| Categoría | Qué cubre | Cómo se valora |
|---|---|---|
| **Pensiones en curso de pago** | Rentas que ya se pagan (al causante y a beneficiarios que ya son rentistas de sobrevivencia). | Anualidad vitalicia de la renta en curso sobre la tabla correspondiente (RV-M-2020, CB-H-2020, MI-…) descontada a `min(TM,TC)`. |
| **Pensiones probables** | Pensiones de sobrevivencia que el **grupo familiar actual** devengará al fallecer el causante (contingencia solo por el transcurso del tiempo). | Flujos contingentes: por cada periodo `i`, la probabilidad de que el causante muera en `i` × (probabilidad de que cada beneficiario sobreviva y cumpla requisitos) × % de pensión × anualidad del beneficiario. |
| **Pensiones posibles** | Beneficiarios que **podrían** adquirir derecho bajo condiciones futuras (p. ej. futuro cónyuge, hijos por nacer, hijos que aún no alcanzan el % final, nuevos integrantes del grupo). | Se documenta para análisis de sensibilidad / TAP (NCG 209) y para recálculos por variación del grupo (Circular 2062). Suele modelarse como escenarios ponderados. |

**Consecuencia para el RIS simulado:** el registro debe distinguir el **rol actual** de cada miembro del grupo (¿es ya rentista? ¿es beneficiario probable? ¿es contingente/posible?), porque eso define si su flujo entra en "en curso", "probables" o "posibles".

> Nota: validar la denominación exacta de estas categorías contra las fichas técnicas SEIL/RIS y los estados financieros (FECU) que emita tu compañía, ya que la nomenclatura puede variar entre reportes.

---

## 4. El RIS (Registro de Información de Seguros)

El RIS es el conjunto de archivos planos/estructurados (.txt/.csv delimitados) que las aseguradoras envían a la CMF por SEIL. Es una base de microdatos a nivel contrato/póliza. Para rentas vitalicias interesan principalmente:

1. **Cartera de pólizas** — identificación, fechas, modalidad, estado, primas y montos en UF.
2. **Asegurados y beneficiarios** — fecha de nacimiento, sexo, parentesco en rentas vitalicias, estado de invalidez.
3. **Rentas vitalicias** — detalle de pensiones y estructura de beneficiarios supervivientes para proyectar flujos bajo las tablas vigentes.

El **RIS simulado** genera estos registros de forma sintética: cada "familia tipo" instancia una póliza (o variantes de ella) con su grupo de asegurados/beneficiarios, y el motor de reservas consume esos registros. Así se puede "escribir el RIS" para cualquier familia y estresar la reserva.

---

## 5. Declaración de una "familia tipo"

### 5.1 Modelo de entrada (canónico)

Se propone declarar cada familia tipo como un **escenario YAML/JSON** (config), que es lo que escribirás tú. El generador lo convierte en registros RIS y luego en pólizas + grupo familiar en SQLite.

```yaml
familia: "abuelo_renovado"          # nombre corto del escenario
causante:
  rol: VEJEZ                        # VEJEZ | INVALIDEZ | SOBREVIVENCIA
  sexo: H
  edad: 78
  renta_mensual_uf: 45.00           # renta en curso del causante
  pensionado: true                  # true => regla cónyuge ≥3 años
grupo_familiar:
  - rol: CONYUGE                    # roles: CONYUGE, CONVIVIENTE_CIVIL, HIJO,
    sexo: M                         #       MADRE_PADRE_NMAT, PADRES, DESIGNADO
    edad: 58                        # (20 años menor, se casó siendo pensionado)
    matrimonio_anios: 2             # 2 < 3 => cónyuge SIN derecho (estrés)
    hijos_comunes: 1                # si >0 relaja el requisito de años
  - rol: HIJO
    sexo: M
    edad: 10
    condicion: MENOR                # MENOR | ESTUDIANTE | INVALIDO
clausulas:
  periodo_garantizado_meses: 120    # 0 = sin período garantizado
  aumento_pct_sobrevivencia: 100    # % del cónyuge si aplica (0 = legal)
  aumento_temporal:
    pct_reduccion: -33.3            # -33.3 | -66.7 | 0 (sin aumento)
    meses: 24
    pct_aumento_vitalicio: 15.0
modo_calculo:
  fecha_calculo: "2026-01-01"
  tasa_descuento: 0.0241            # min(TM,TC); 0 => usar regla NCG 318
  tabla_causante: "CB-H-2020"
  tabla_beneficiario: "RV-M-2020"
```

### 5.2 Catálogo de roles del grupo familiar

| Código rol | Rol | Clasificación reserva por defecto |
|---|---|---|
| `CAUSANTE` | Titular de la renta (vejez/invalidez) o causante fallecido (sobrevivencia) | En curso de pago (si rentista) |
| `CONYUGE` | Cónyuge (o ex cónyuge con derecho) | Probable → en curso al fallecer el causante |
| `CONVIVIENTE_CIVIL` | Conviviente civil (AUC) | Probable → en curso |
| `HIJO` | Hijo con derecho (menor/estudiante/inválido) | Probable → en curso; **termina** a los 18/24 o por invalidez 11% |
| `MADRE_PADRE_NMAT` | Madre/padre de hijos no matrimoniales | Probable → en curso |
| `PADRES` | Padres del causante (solo si no hay otros) | Probable → en curso |
| `DESIGNADO` | Beneficiario designado (solo opera dentro del período garantizado o sin beneficiarios legales) | Posible |

Cada miembro lleva además: sexo, fecha de nacimiento, condición (MENOR/ESTUDIANTE/INVALIDO), años de matrimonio/AUC, hijos comunes, y si es **actualmente rentista** (en curso) o **contingente** (probable).

### 5.3 Materialización en registros RIS

El generador traduce el escenario a tres tipos de filas:

- **Fila póliza**: `NUM_POLIZA`, `TIPO_REPORTE`, `MODALIDAD` (RVI/RVD/RVM), `FEC_INICIO`, `TIPO_RENTA`, `RENTA_UF`, `TASA_BAUTIZO`, `ESTADO`.
- **Fila asegurado/beneficiario**: `NUM_POLIZA`, `ROL_PARENTESCO`, `FEC_NACIMIENTO`, `SEXO`, `ESTADO_INVALIDEZ`, `CONDICION_EDUC`.
- **Fila renta vitalicia**: pensiones en curso + estructura de supervivientes, con flag `EN_CURSO/PROBABLE/POSIBLE`, `PCT_SOBREVIVENCIA` y vigencias (edad fin de derecho).

Con esas filas, el motor de reservas (módulo `internal/calculator`) calcula:

- Reserva **en curso**: anualidad del causante + de beneficiarios ya rentistas.
- Reserva **probable**: para cada `i` hasta `ω`, probabilidad de muerte del causante en `i`, supervivencia y elegibilidad de cada beneficiario, con su % (incluyendo acrecimiento y cambios de %), descontado a la VTD.
- Componentes de **posible**: escenarios de grupo (matrimonio futuro, hijos por nacer) como variantes ponderadas (para TAP/sensibilidad).

---

## 6. Esquema de datos (SQLite) — extensión propuesta

Las tablas existentes se mantienen. Se agregan cuatro tablas para capturar grupo familiar y cláusulas (estilo de las ya definidas en `technical_specifications.md`):

```sql
-- Miembros del grupo familiar de una póliza (one-to-many)
CREATE TABLE beneficiario (
    id INTEGER PRIMARY KEY,
    poliza_id INTEGER NOT NULL,
    rol VARCHAR(20),              -- CONYUGE, CONVIVIENTE_CIVIL, HIJO, PADRES, DESIGNADO...
    sexo CHAR(1),
    fecha_nacimiento DATE,
    condicion VARCHAR(20),        -- MENOR, ESTUDIANTE, INVALIDO, NINGUNA
    invalidez_parcial BOOLEAN DEFAULT 0,
    matrimonio_anios INTEGER,     -- años de matrimonio/AUC al fallecimiento
    hijos_comunes INTEGER DEFAULT 0,
    es_rentista BOOLEAN DEFAULT 0, -- true => reserva "en curso"; false => "probable"
    pct_sobrevivencia DECIMAL(5,2), -- % legal o contractual (si aplica)
    fin_derecho_edad INTEGER,     -- 18, 24 (estudiante), NULL (vitalicia)
    FOREIGN KEY (poliza_id) REFERENCES poliza(id)
);

-- Cláusulas adicionales de la póliza
CREATE TABLE clausula (
    id INTEGER PRIMARY KEY,
    poliza_id INTEGER NOT NULL,
    tipo VARCHAR(30),             -- PERIODO_GARANTIZADO, AUMENTO_PCT_SOBREVIVENCIA,
                                  -- AUMENTO_TEMPORAL_DIFERIDO
    parametros TEXT,              -- JSON: {"meses":120}, {"pct":100}, {"pct_red":-33.3,"meses":24,...}
    FOREIGN KEY (poliza_id) REFERENCES poliza(id)
);

-- Desglose de reserva por categoría (curso/probable/posible)
ALTER TABLE reserva_calculada ADD COLUMN categoria VARCHAR(20); -- EN_CURSO, PROBABLE, POSIBLE
ALTER TABLE reserva_calculada ADD COLUMN beneficiario_id INTEGER;
ALTER TABLE reserva_calculada ADD COLUMN escenario VARCHAR(50); -- nombre familia tipo

-- Bibliotecas de familias tipo y escenarios generados
CREATE TABLE familia_tipo (
    id INTEGER PRIMARY KEY,
    nombre VARCHAR(50) UNIQUE,
    definicion TEXT,              -- YAML/JSON del escenario (sección 5.1)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE escenario_generado (
    id INTEGER PRIMARY KEY,
    familia_tipo_id INTEGER,
    variante VARCHAR(30),         -- base, +garantia120, +aumento33, sin_conyuge...
    poliza_id INTEGER,            -- póliza materializada en RIS simulado
    FOREIGN KEY (familia_tipo_id) REFERENCES familia_tipo(id)
);
```

Además, se recomienda marcar en `poliza` el `tipo_renta` con valores `VITALICIA_INMEDIATA`, `VITALICIA_DIFERIDA`, `VITALICIA_MIXTA` y agregar `edad_causante`, `sexo_causante`, `tipo_pension` (VEJEZ/INVALIDEZ/SOBREVIVENCIA).

---

## 7. Dimensiones de estrés (barrido de escenarios)

El generador debe poder combinar:

1. **Composición del grupo**: hombre/mujer solo; cónyuge; cónyuge + 1…N hijos; conviviente civil; hijos no matrimoniales; padres; sin beneficiarios legales (→ designados, solo período garantizado).
2. **Edades límite**: cónyuge que cumple los 3 años de matrimonio durante el período; hijos que cumplen 18 y 24 años; cónyuge < 45 años (pensiones temporales en regímenes antiguos); cónyuge embarazada.
3. **Causante**: vejez vs. invalidez (tablas MI-…), hombres vs. mujeres (CB-H-2020 vs. RV-M-2020), renta UF baja/alta.
4. **Cláusulas**: con/sin período garantizado (5/10/15/20 años); con/sin aumento de % de sobrevivencia; con/sin aumento temporal/diferido vitalicio (-33,3% / -66,7%, 12/24/36/120 meses).
5. **Tasa**: min(TM,TC) vs. tasa fija de estrés (sensibilidad ±100 pb); VTD real vs. fallback.
6. **Fecha**: pólizas pre-2012 (Circular 1512), 2012–2015 (NCG 318), 2015–2020 (NCG 374), post-2020 (anexo NCG 318) — cambia metodología y tablas.

**Ejemplo (caso del usuario "abuelo renovado"):**
- Base: abuelo 78 años, vejez, casado hace 2 años con mujer de 58 (→ **cónyuge sin derecho** por no cumplir 3 años ni tener hijos comunes). Grupo: solo causante → reserva baja (sin sobrevivencia probable). Estrés válido.
- Variante "hijo en común": nace hijo → se relaja el requisito de matrimonio, la cónyuge pasa a "probable" con 50% (y 60% cuando el hijo deja de tener derecho) → la reserva salta. Este es el caso de estrés por **composición de grupo familiar**.
- Variante "+período garantizado 120": aunque la cónyuge no tenga derecho legal, dentro de los 120 meses los flujos van a beneficiarios designados → cambia la reserva "posible".
- Variante "+aumento temporal": renta reducida 24 meses y luego mayor → cambia `FP_ji` en dos tramos.

---

## 8. Roadmap de implementación

1. **Modelo de escenarios** — parser YAML de familias tipo + catálogo de roles (sección 5).
2. **Generador RIS** — materializar escenario en filas póliza/asegurado/renta (sección 5.3).
3. **Esquema SQLite** — migraciones con tablas `beneficiario`, `clausula`, `familia_tipo`, `escenario_generado` + columnas nuevas (sección 6).
4. **Motor de grupo familiar** — en `internal/calculator`, proyección por beneficiario (elegibilidad, %, acrecimiento, fin de derecho) sobre las tablas.
5. **Desglose en curso/probable/posible** — agregar categorías a `reserva_calculada`.
6. **Cláusulas** — período garantizado y aumentos sobre el flujo `FP_ji`.
7. **Barrido/estres** — script de variantes (sección 7) + reporte comparativo de reservas.

## 9. Preguntas abiertas

- Validar denominación exacta y detalle de campos del RIS/SEIL de rentas vitalicias (ficha Circular 1194: `Manual_Usuario_C1194.pdf` ya está en `docs/normativo/`) — confirmar los códigos de parentesco y flags de "en curso/probable/posible" reales.
- Confirmar si el recálculo por variación del grupo (Circular 2062, en consulta 2026) debe modelarse como un disparador de nuevo cálculo o como un módulo aparte.
- Definir el tratamiento de "pensiones posibles" en el balance: ¿se valoran dentro de la reserva técnica o solo en TAP/sensibilidad?
