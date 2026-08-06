package scenario

// This file contains predefined family type scenarios that can be written
// to disk as YAML and then run through the simulator.

// BuiltinScenarios returns predefined family type YAML strings.
var BuiltinScenarios = map[string]string{
	"familia_tradicional": familiaTradicional,
	"familia_vieja":       familiaVieja,
	"viejo_caliente":      viejoCaliente,
	"viejo_caliente_ext":  viejoCalienteExtendido,
	"invalido_joven":      invalidoJoven,
	"descalce_2009":       descalce2009,
	"descalce_tm2020":     descalceTM2020,
}

const familiaTradicional = `
name: familia_tradicional
description: Familia nuclear estandar - hombre 65, mujer 60, hijo 15
horizon: 50

policy:
  capital_uf: 5000.0
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "04"
  modalidad_renta: "1000"

causante:
  rol: CAUSANTE
  sexo: M
  edad: 65
  fecha_nacimiento: "1959-01-15"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 60
    fecha_nacimiento: "1964-03-10"
    tipo_c1194: "10"
    matrimonio_anios: 35
    hijos_comunes: 1
  - rol: HIJO
    sexo: F
    edad: 15
    fecha_nacimiento: "2009-07-21"
    tipo_c1194: "30"
    condicion: MENOR
    fin_derecho_edad: 24

events:
  - year: 9
    type: HIJO_CUMPLE_24
    target_rol: HIJO
`

const familiaVieja = `
name: familia_vieja
description: Pareja mayor sin hijos con derecho - ambos 70+, sin sobrevivencia prolongada
horizon: 45

policy:
  capital_uf: 8000.0
  tasa_tm: 0.040
  tasa_tc: 0.038
  tipo_pension: "04"
  modalidad_renta: "3120"
  periodo_garantizado_meses: 120

causante:
  rol: CAUSANTE
  sexo: M
  edad: 72
  fecha_nacimiento: "1952-05-20"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 70
    fecha_nacimiento: "1954-11-02"
    tipo_c1194: "10"
    matrimonio_anios: 45
    hijos_comunes: 0

events:
  - year: 8
    type: KILL_MEMBER
    target_rol: CONYUGE
    target_sexo: F
`

const viejoCaliente = `
name: viejo_caliente
description: Viejo jubilado se casa con jovencita, tiene hijo, alta complejidad actuarial
horizon: 50

policy:
  capital_uf: 6000.0
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "04"
  modalidad_renta: "1000"

causante:
  rol: CAUSANTE
  sexo: M
  edad: 78
  fecha_nacimiento: "1946-01-01"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 45
    fecha_nacimiento: "1979-06-15"
    tipo_c1194: "10"
    matrimonio_anios: 2
    hijos_comunes: 0

events:
  # Nace hijo en comun al ano 2 - activa derecho de conyuge
  - year: 2
    type: ADD_MEMBER
    member:
      rol: HIJO
      sexo: M
      edad: 0
      tipo_c1194: "30"
      condicion: MENOR
      fin_derecho_edad: 24
  # Conyuge cambia a tipo 11 (con hijos) en ano 3
  # El hijo hace que conyuge tenga derecho aun sin 3 anos de matrimonio
  # Causante muere al ano 5
  - year: 5
    type: KILL_MEMBER
    target_rol: CAUSANTE
    target_sexo: M
  # Hijo cumple 24 al ano 26 (nacio t=2, cumple 24 en t=26)
  # Esto se maneja automaticamente por fin_derecho_edad
`

const viejoCalienteExtendido = `
name: viejo_caliente_ext
description: Viejo divorciado, se casa con jovencita, ex no tiene derecho, hijo en comun, periodo garantizado
horizon: 50

policy:
  capital_uf: 6000.0
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "04"
  modalidad_renta: "4120"
  periodo_garantizado_meses: 120

causante:
  rol: CAUSANTE
  sexo: M
  edad: 75
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 38
    tipo_c1194: "10"
    matrimonio_anios: 1
    hijos_comunes: 0
  - rol: CONYUGE
    sexo: F
    edad: 60
    tipo_c1194: "10"
    hijos_comunes: 0

events:
  # Ex pierde derecho inmediatamente (divorciada)
  - year: 0
    type: REMOVE_MEMBER
    target_rol: CONYUGE
    target_sexo: F
  # Nace hijo en comun al ano 2
  - year: 2
    type: ADD_MEMBER
    member:
      rol: HIJO
      sexo: F
      edad: 0
      tipo_c1194: "30"
      condicion: MENOR
      fin_derecho_edad: 24
  # Causante fallece al ano 6
  - year: 6
    type: KILL_MEMBER
    target_rol: CAUSANTE
    target_sexo: M
`

const invalidoJoven = `
name: invalido_joven
description: Causante joven con Invalidez Total (tipo_pension 06 - RV invalidez), conyuge e hijo
horizon: 60

policy:
  capital_uf: 5000.0
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "06"   # RV-Invalidez Total
  modalidad_renta: "1000"

causante:
  rol: CAUSANTE
  sexo: M
  edad: 45
  fecha_nacimiento: "1979-04-12"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 40
    fecha_nacimiento: "1984-02-28"
    tipo_c1194: "10"
    matrimonio_anios: 12
    hijos_comunes: 1
  - rol: HIJO
    sexo: F
    edad: 9
    fecha_nacimiento: "2015-11-03"
    tipo_c1194: "30"
    condicion: MENOR
    fin_derecho_edad: 24

events:
  - year: 30
    type: KILL_MEMBER
    target_rol: CAUSANTE
    target_sexo: M
`

const descalce2009 = `
name: descalce_2009
description: Poliza 2009 (estrato pre-2012, base RV-2009/B-2006) con descalce gradual vs TM-2020
horizon: 40

policy:
  capital_uf: 5000.0
  fecha_contratacion: "2009-06-01"
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "04"
  modalidad_renta: "1000"
  gradualidad_anios: 10

causante:
  rol: CAUSANTE
  sexo: M
  edad: 70
  fecha_nacimiento: "1939-01-15"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 65
    fecha_nacimiento: "1944-03-10"
    tipo_c1194: "10"
    matrimonio_anios: 40
    hijos_comunes: 0

events:
  - year: 25
    type: KILL_MEMBER
    target_rol: CAUSANTE
    target_sexo: M
`

const descalceTM2020 = `
name: descalce_tm2020
description: Poliza 2024 (estrato post-2012, base TM-2020) - sin descalce, control
horizon: 40

policy:
  capital_uf: 5000.0
  fecha_contratacion: "2024-01-15"
  tasa_tm: 0.038
  tasa_tc: 0.035
  tipo_pension: "04"
  modalidad_renta: "1000"

causante:
  rol: CAUSANTE
  sexo: M
  edad: 65
  fecha_nacimiento: "1959-01-15"
  tipo_c1194: "99"

grupo_familiar:
  - rol: CONYUGE
    sexo: F
    edad: 60
    fecha_nacimiento: "1964-03-10"
    tipo_c1194: "10"
    matrimonio_anios: 35
    hijos_comunes: 0

events: []
`
