# Lector canónico del RIS (Circular 1194)

`./reservas -ris-dump <archivo.vta>` es la forma canónica de leer un archivo
RIS. Reemplaza los scripts ad-hoc de inspección: usa el **mismo parser** que la
validación (`internal/loader/ris1194.go`), decodifica los códigos con el
**diccionario** (`internal/models/ris_dict.go`) y **no requiere base de datos ni
config**. Todo lo que necesitas para interpretar una póliza está en el repo, no
en una sesión.

## Uso

```bash
# Dump de las primeras 10 pólizas
./reservas -ris-dump /tmp/ris_extract/ris20251231.vta

# Una póliza específica (recorre todo el archivo, ~10 min en el RIS completo)
./reservas -ris-dump /tmp/ris_extract/ris20251231.vta -svs 099747

# Con filtros de causante (muy útil para analizar el gap sin_causante_vivo)
./reservas -ris-dump <archivo> -filter dead -n 5
./reservas -ris-dump <archivo> -filter alive -n 5

# NDJSON para piping a jq/python (máquina-legible y estable)
./reservas -ris-dump <archivo> -filter dead -json | jq '.svs, .modalidad_renta_txt'
./reservas -ris-dump <archivo> -json | python3 -c "import sys,json;[print(json.loads(l)['svs']) for l in sys.stdin]"

# Diccionario de códigos completo (no requiere archivo)
./reservas -legend

# Enumerar los códigos observados en un archivo (descubrir códigos no documentados)
./reservas -ris-dump <archivo> -scan-codes
```

Flags:

| Flag | Descripción |
|------|-------------|
| `-ris-dump <path>` | Activa el lector. Sin `-svs` ni `-filter`, dumps 10 pólizas. |
| `-svs <id>` | Solo esa póliza (SVS sin ceros a la izquierda). |
| `-n <N>` | Máx. pólizas a imprimir. `-n -1` = ilimitado (cuidado con 959K). |
| `-filter dead\|alive` | Solo pólizas con causante fallecido / vivo. |
| `-json` | Salida NDJSON (una póliza por línea). |
| `-legend` | Imprime layout + diccionario de códigos y termina. |
| `-scan-codes` | Enumera los valores observados de cada campo de códigos. |

## Salida (modo texto)

```
=== PÓLIZA SVS=099747 === (FALLECIDO)
  tipo_pension=10 (Sobrevivencia de RV de Vejez a edad anticipada (de 05))
  modalidad_renta=3180 (RV con período garantizado de pago, 180 meses)
  tipo_renta=1000 (inmediata)
  ...
  personas (2):
    #   rol                      C1194 der  sexo edad_c edad_v nacimiento fallecimiento inv    pensión  pct%   RT-BASE-VIG  RT-FIN-2020
    1   CAUSANTE                 99   10   M    57     67     1958-05-14 2024-02-23 N      0.00     100    0.00         0.00
    2   CONYUGE                  10   99   F    56     66     1959-03-30 -          N      1.07     100    1526.01      0.00
```

Cada código se imprime con su significado del diccionario entre paréntesis.
`edad_c` = edad a la emisión, `edad_v` = edad a la valuación (2025-12-31).
`RT-BASE-VIG` y `RT-FIN-2020` son los campos 3.26 y 3.31b del RIS.

## El diccionario (`internal/models/ris_dict.go`)

Mapa canónico código→significado de los campos del Anexo Técnico C1194.
`RISDictionary()` devuelve todos los campos; `LookupRISCode(field, code)` y
`DescribeModalidadRenta(code, periodoAumento)` para consultar en código.

**Fuentes** (todas en el repo, `docs/normativo/`):
- Circular 1194 de 20.01.1995 (`cir_1194_1995.pdf`) — base.
- Modificaciones: `cir_1772_2005.pdf` (TIPO-RENTA 3000, redefine 4xxx),
  `cir_2184_2015.pdf` (conviviente civil 50/51/52, REQUISITO 7/8/9),
  `cir_2208_2016.pdf` (TCj post-jun2015 = tasa calculada con VTD según NCG 318),
  `cir_2308_2022.pdf`.
- **Anexo Técnico C1194 actualizado** (`Anexo_Tecnico_Actualizado_11194.pdf`) —
  versión vigente del layout y los códigos (TIPO-OPERACION-RV, REQUISITO 5/6,
  TIPO-PAGO-BENEFICIO-ESTATAL, PERIODO-AUMENTO).
- **Codificación CMF C1194 (módulo SEIL)** (`seil_codigos_cir1194.md` —
  respaldo de certificacion_cir1194.php) — tabla oficial de `CODIGO-AFP`
  (incluye 28-33) y `COMPAÑIA-REASEGURO`.

El diccionario está **100% confirmado por fuentes autoritativas**: ya no quedan
códigos pendientes de verificación.

## Notas de semántica confirmadas contra los datos

- **MODALIDAD-RENTA 2xxx** = cláusula de **aumento temporal**; `xxx` = meses
  del **período garantizado** (0 si no lo tiene). La duración del aumento
  temporal va en `PERIODO-AUMENTO` (pos 62-64). Verificado: 2120/2180/2240 =
  PG de 120/180/240 meses con aumento temporal de 12-60 meses.
- **TASA-CTO-EMISION** (2.22) = **TCj** (tasa de costo de emisión equivalente,
  iguala flujos con la reserva base a la emisión). **TASA-VENTA** (2.23) =
  **TVj** (iguala flujos con la prima única). **Ninguno es la TM**. Para pólizas
  2012-may2015, TCj viene en cero (cir_2208); para pre-2012 solo si la compañía
  adoptó NCG 318. Ver `docs/analysis/observaciones_avance.md` §2.12-2.14.
- **PENSION-PERSONA** (3.21): solo se informa para quien está recibiendo la
  renta. Si el causante está vivo, los beneficiarios llevan cero; si está
  fallecido, el causante lleva cero y los sobrevivientes su pensión efectiva.
  Es la pista clave del grupo `sin_causante_vivo`.
- El header (Registro 1) de la CMF no trae conteos de pólizas/beneficiarios
  (posiciones 10-18 son el RUT de la compañía); `loader.LoadHeader` solo
  expone `FechaHasta`.
