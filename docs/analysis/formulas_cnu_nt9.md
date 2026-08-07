# Fórmulas del CNU por tipo de familia (Nota Técnica N°9, SPensiones)

Fuente: `docs/normativo/nota_tecnica_9_CNU_spensiones.pdf` (Nov 2024).
Texto extraído en `/tmp/nt9/nt9.txt`.

Este documento es la **referencia normativa** para las fórmulas actuariales del
Capital Neto Unitario (CNU) total, que determina tanto la pensión inicial como
la estructura de la reserva. El motor debe replicar estas ecuaciones.

## Notación

- `x` = edad del afiliado/causante
- `y` = edad del cónyuge
- `h_j` = edad del hijo j
- `h_m` = edad del hijo menor
- `l_x` = número de sobrevivientes a edad x (tabla de mortalidad)
- `i_t` = tasa de interés en t (VTD o flat según cohorte, NCG 318)
- `w` = 110 - y (fin de tabla del cónyuge)
- `J` = número de hijos con derecho a pensión
- `z = 24` (edad límite de derecho de hijos)

## Pensión de vejez o invalidez (causante vivo)

### Afiliado soltero (ec. 9)

```
cnu_total = [ Σ_{t=0}^{110-x} l_{x+t} / (l_x · (1+i_t)^t) ] - 11/24
```

### Afiliado con cónyuge SIN hijos (ec. 10 + 11)

```
cnu_total = cnu_i + cnu_c
cnu_c     = 0.6 · [ Σ_{t=0}^{w} l_{y+t}/(l_y·(1+i_t)^t) · (1 - l_{x+t}/l_x) ]
```

### Afiliado con cónyuge CON hijos (ec. 12 + 13 + 14)

```
cnu_total = cnu_i + cnu_c(con hijos) + Σ_j cnu_hijo_j

cnu_c(con hijos) = 0.5 · [ Σ l_{y+t}/(l_y·(1+i_t)^t) · (1 - l_{x+t}/l_x) ]
                 + 0.1 · [ Σ l_{y'+t}/(l_{y'}·(1+i_{y'-y+t})^t) · (1 - l_{x'+t}/l_x) ]
                 - 0.1 · [ (11/24) · l_{y'}/(l_{y'}·(1+i_{y'-y})^{y'-y}) · (1 - l_{x'}/l_x) ]

donde y' = y + z - h_m, x' = x + z - h_m  (z=24, h_m = edad del hijo menor)

cnu_hijo_j = 0.15 · [ Σ_{t=0}^{23-h_j} l_{h_j+t}/(l_{h_j}·(1+i_t)^t)
                   - Σ_{t=0}^{23-h_j} l_{h_j+t}·l_{x+t}/(l_{h_j}·l_x·(1+i_t)^t) ]
            - 0.15 · [ (11/24) · (1/(1+i_{24-h_j})^{24-h_j}) · (l_{x+24-h_j}/l_x - l_24/l_{h_j} - l_24/l_{h_j}) ]
```

**Límite de hijos**: la suma va hasta `t = 23 - h_j`, i.e. el hijo recibe pensión
solo hasta los **24 años**.

### Afiliado SIN cónyuge CON hijos (ec. 15 + 16)

```
cnu_total = cnu_i + Σ_j cnu_hijo_j(sin cónyuge)

cnu_hijo_j(sin cónyuge) = (0.15 + 0.5/J) · [ Σ - Σ ]  (mismos límites hasta 24-h_j)
                        - (0.15 + 0.5/J) · [ ajuste 11/24 ]
```

Cuando no hay cónyuge, cada hijo recibe `15% + 50%/J` (reparten el 50% del
cónyuge ausente).

## Pensión de sobrevivencia (causante fallecido)

### Cónyuge sin hijos (ec. 17)

```
cnu_conyuge = 0.6 · [ Σ_{t=0}^{w} l_{y+t}/(l_y·(1+i_t)^t) - 11/24 ]
```

Anualidad vitalicia simple del cónyuge, al 60%.

### Cónyuge con hijos (ec. 18)

```
cnu_conyuge = 0.5 · [ Σ l_{y+t}/(l_y·(1+i_t)^t) - 11/24 ]
            + 0.1 · [ Σ l_{y'+t}/(l_{y'}·(1+i_{y'-y+t})^t) ]
            - 0.1 · [ (11/24) · l_{y'}/(l_{y}·(1+i_{y'-y})^{y'-y}) ]
```

### Hijos con cónyuge (ec. 19)

```
cnu_hijo_j = 0.15 · [ Σ_{t=0}^{23-h_j} l_{h_j+t}/(l_{h_j}·(1+i_t)^t) - 11/24 ]
            - 0.15 · [ (11/24) · (1 - l_24/(l_{h_j}·(1+i_{24-h_j})^{24-h_j})) ]
```

### Hijos sin cónyuge (ec. 20)

```
cnu_hijo_j = (0.15 + 0.5/J) · [ Σ_{t=0}^{23-h_j} ... - ajuste ]
```

## Implicaciones para el motor

1. **Horizonte de hijos**: limitado a `24 - edad_actual` años. El motor actual
   proyecta hijos hasta edad 110 → sobrestima reservas de `con_hijos`.
2. **Porcentajes del CNU** (al calcular pensión inicial):
   - Cónyuge sin hijos: 60%
   - Cónyuge con hijos: 50% + término 10% (modela reajuste al cumplir 24 el hijo menor)
   - Hijos: 15% c/u (con cónyuge), 15% + 50%/J c/u (sin cónyuge)
3. **Para reserva de pólizas emitidas**: los porcentajes del contrato
   (reportados en `PorcentajePension` del RIS) son los que aplican al VPP, no
   los del CNU. Pero la **estructura temporal** (hijos hasta 24) sí aplica
   siempre.

## Reglas de derecho de pensión (NT9 nota 9)

Los hijos tienen derecho a pensión de sobrevivencia si son:
- **Menores de 18 años**, o
- **Estudiantes menores de 24 años**.

Cuando el RIS reporta un hijo, asume que tiene derecho (menor de 24). El motor
debe verificar la edad actual y truncar la proyección a los 24.
