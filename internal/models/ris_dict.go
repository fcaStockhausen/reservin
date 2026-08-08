package models

import "fmt"

// Diccionario canónico de códigos del Anexo Técnico C1194 (RIS).
//
// Fuentes:
//   - Circular 1194 de 20.01.1995 (texto refundido, sección "Contenido de los
//     Campos") y sus modificaciones: cir_1772_2005, cir_2184_2015,
//     cir_2208_2016, cir_2308_2022 (todas en docs/normativo/).
//   - Anexo Técnico C1194 actualizado (Anexo_Tecnico_Actualizado_11194.pdf,
//     docs/normativo/) — versión vigente del layout y los códigos.
//   - Codificación CMF, Circular N° 1194 (módulo SEIL,
//     certificacion_cir1194.php; respaldo en
//     docs/normativo/seil_codigos_cir1194.md) — tabla oficial de CODIGO-AFP,
//     COMPAÑIA-REASEGURO y códigos de excepción.
//
// Cada entrada tiene: Code = valor tal como va en el archivo, Label =
// significado, Note = fuente o advertencia.

// RISCode es una entrada código→significado.
type RISCode struct {
	Code  string
	Label string
	Note  string
}

// RISFieldDef describe un campo de códigos del Anexo Técnico C1194.
type RISFieldDef struct {
	Name  string // nombre canónico (ej. "TIPO-PENSION")
	Campo string // número de campo (ej. "2.6")
	Pos   string // posiciones 1-indexadas inclusivas (ej. "10-11")
	Desc  string // qué representa el campo
	Codes []RISCode
}

func c(code, label string) RISCode { return RISCode{Code: code, Label: label} }
func n(code, label, note string) RISCode {
	return RISCode{Code: code, Label: label, Note: note}
}

// RISDictionary devuelve el diccionario completo de campos de códigos.
func RISDictionary() []RISFieldDef {
	return []RISFieldDef{
		{
			Name: "TIPO-PENSION", Campo: "2.6", Pos: "10-11",
			Desc: "Tipo de pensión que es o ha sido responsabilidad de la compañía.",
			Codes: []RISCode{
				c("01", "Sobrevivencia (Circular 528)"),
				c("02", "Invalidez (Circular 528)"),
				c("03", "Sobrevivencia de la Invalidez (Circular 528)"),
				c("04", "Renta vitalicia de Vejez a edad de jubilación"),
				c("05", "Renta vitalicia de Vejez a edad anticipada"),
				c("06", "Renta vitalicia de Invalidez Total"),
				c("07", "Renta vitalicia de Invalidez Parcial"),
				c("08", "Renta vitalicia de Sobrevivencia"),
				c("09", "Sobrevivencia de RV de Vejez a edad de jubilación (de 04)"),
				c("10", "Sobrevivencia de RV de Vejez a edad anticipada (de 05)"),
				c("11", "Sobrevivencia de RV de Invalidez Total (de 06)"),
				c("12", "Sobrevivencia de RV de Invalidez Parcial (de 07)"),
				c("13", "Sobrevivencia por traspaso o compra de cartera"),
				c("14", "Invalidez por traspaso o compra de cartera"),
				c("15", "Sobrevivencia de la invalidez por traspaso o compra de cartera (de 14)"),
			},
		},
		{
			Name: "COMPANIA-OBLIGADA", Campo: "2.7", Pos: "12",
			Desc: "Compañía obligada al pago del Aporte Adicional (Art. 62 inciso final, D.L. 3.500).",
			Codes: []RISCode{
				c("O", "Obligada al pago del Aporte Adicional"),
				c("N", "No obligada al pago del Aporte Adicional"),
			},
		},
		{
			Name: "VIGENCIA-PENSION", Campo: "2.8", Pos: "13",
			Desc: "Estado de vigencia de la pensión.",
			Codes: []RISCode{
				c("6", "Pensión en pago"),
				c("7", "Pensión garantizada (pagando a designados)"),
				c("8", "Pensión diferida"),
				c("9", "Pensión extinguida"),
			},
		},
		{
			Name: "CODIGO-AFP", Campo: "2.9", Pos: "14-15",
			Desc: "Administradora de Fondos de Pensiones en que estaba afiliado el causante.",
			Codes: []RISCode{
				c("01", "Alameda"), c("02", "Concordia"), c("03", "Cuprum"),
				c("04", "El Libertador"), c("05", "Habitat"), c("06", "Invierta (hasta 30.11.1993)"),
				c("07", "Planvital (hasta 30.11.1993)"), c("08", "Provida (fusión Unión 01.06.1998)"),
				c("09", "San Cristóbal"), c("10", "Santa María"), c("11", "Summa"),
				c("12", "Magister (fusión Qualitas 01.09.1998)"), c("13", "Unión (fusión Provida 01.06.1998)"),
				c("14", "Protección"), c("15", "Futuro"), c("16", "Bannuestra"),
				c("17", "Banguardia"), c("18", "Qualitas (fusión Magister 01.09.1998)"),
				c("19", "Bansander"), c("20", "Laboral"), c("21", "Previpan"),
				c("22", "Fomenta"), c("23", "Genera"), c("24", "Valora"),
				c("25", "Aporta"), c("26", "Planvital (desde 01.12.1993)"), c("27", "Armoniza"),
				c("28", "Summa Bansander"), c("29", "Aporta + Fomenta"),
				c("30", "Provida (fusión con Protección 31.12.1998)"),
				c("31", "Capital"), c("32", "Modelo"), c("33", "Uno"),
			},
		},
		{
			Name: "TIPO-AFILIADO", Campo: "2.10", Pos: "16",
			Desc: "Calidad del afiliado que da origen al pago de pensión.",
			Codes: []RISCode{
				c("D", "Dependiente"),
				c("I", "Independiente"),
				c("R", "Rentista vitalicio"),
			},
		},
		{
			Name: "TIPO-RENTA", Campo: "2.17", Pos: "52-55",
			Desc: "Renta vitalicia inmediata o diferida.",
			Codes: []RISCode{
				c("1000", "Inmediata"),
				n("2xxx", "Diferida, xxx = meses que se difiere la renta", "ej. 2012 = diferida 12 meses"),
				n("3000", "Inmediata con retiro programado", "agregado por cir_1772_2005"),
				n("0000", "Pólizas pre-2005 que no informan el campo", "observado en el RIS real"),
			},
		},
		{
			Name: "MODALIDAD-RENTA", Campo: "2.18", Pos: "56-59",
			Desc: "Modalidad de pago de la renta vitalicia (cláusulas adicionales).",
			Codes: []RISCode{
				c("1000", "Renta vitalicia simple, sin adicionales"),
				n("2xxx", "Con cláusula de aumento temporal; xxx = meses del período garantizado (0 si no tiene PG). La duración del aumento temporal va en el campo PERIODO-AUMENTO (2.20, pos 62-64)", "semántica confirmada contra el RIS real: 2120/2180/2240 = PG 120/180/240 meses"),
				n("3xxx", "Con período garantizado de pago; xxx = meses garantizados", "ej. 3120 = PG 120 meses"),
				n("4xxx", "Con cláusula de aumento de % de pensión de sobrevivencia (Art. 58 D.L. 3.500); xxx = meses de PG si además tiene PG", "redefinida por cir_1772_2005"),
				n("0000", "Pólizas pre-2005 que no informan el campo", "observado en el RIS real"),
			},
		},
		{
			Name: "TIPO-OPERACION-RV", Campo: "2.19", Pos: "60-61",
			Desc: "Si la renta vitalicia se origina en una selección de modalidad de pensión o en un cambio de modalidad de pensión.",
			Codes: []RISCode{
				n("SM", "Selección de modalidad de pensión", "Anexo Técnico C1194 actualizado"),
				n("CM", "Cambio de modalidad de pensión", "Anexo Técnico C1194 actualizado"),
				n("(vacío)", "No informado (pólizas antiguas/pre-2005)", "observado en el RIS real"),
			},
		},
		{
			Name: "TASA-CTO-EMISION", Campo: "2.22", Pos: "70-73",
			Desc: "Tasa de costo de emisión equivalente (TCj): iguala los flujos actuariales de la póliza con su reserva técnica base a la FECHA-VIGENCIA-INICIAL. No confundir con la TVj (campo 2.23). Desde cir_2208_2016, para pólizas post-jun2015 es la TCj calculada con el VTD (NCG 318).",
			Codes: []RISCode{
				n("TCj", "Tasa de costo de emisión equivalente", "C1194 2.22 + cir_2208_2016; NO es la TM"),
			},
		},
		{
			Name: "TASA-VENTA", Campo: "2.23", Pos: "74-77",
			Desc: "Tasa de venta (TVj): tasa que iguala los flujos actuariales de la póliza con el valor de la prima única.",
			Codes: []RISCode{
				n("TVj", "Tasa de venta", "C1194 2.23"),
			},
		},
		{
			Name: "NUMERO-REASEGURO", Campo: "2.24", Pos: "78",
			Desc: "Número de operaciones de reaseguro efectuadas para la póliza (0 a 3).",
		},
		{
			Name: "COMPANIA-REASEGURO", Campo: "2.25", Pos: "79-80",
			Desc: "Código de la compañía cedente o aceptante de la operación de reaseguro (hasta 3 operaciones: (1), (2), (3)).",
			Codes: []RISCode{
				c("00", "Ex Centenario"), c("01", "Aetna"), c("02", "Caja Reaseguradora"),
				c("03", "Zurich Chile (ex Chilena Consolidada)"), c("04", "Cigna"),
				c("05", "Consorcio Nacional"), c("06", "Construcción"), c("07", "Huelén"),
				c("08", "Ise Las Américas"), c("09", "Interamericana"), c("10", "Previsión"),
				c("11", "ABN"), c("12", "Renta Nacional"), c("13", "Roble"),
				c("14", "Euroamérica"), c("15", "Santander"), c("16", "Raulí"),
				c("17", "ING"), c("18", "Banrenta"), c("19", "Axa (ex UAP)"),
				c("20", "Vida Corp (ex Compensa)"), c("21", "CNA (ex Convida)"),
				c("22", "Soince Re"), c("23", "Cruz del Sur"), c("24", "Le Mans"),
				c("25", "BICE"), c("26", "Interrentas"), c("27", "Ohio"),
				c("28", "Mass"), c("29", "RGA"), c("30", "AGF Allianz"),
				c("31", "BCI"), c("32", "Cardif"), c("33", "Altavida Santander"),
				c("34", "Vitalis"), c("35", "Principal"), c("36", "Mapfre"),
				c("37", "Bbva"), c("38", "Penta"), c("39", "Banchile"),
				c("40", "ACE"), c("41", "ING Seguros de Rentas Vitalicias"),
				c("60", "Augustar"),
			},
		},
		{
			Name: "GENERO", Campo: "3.5", Pos: "10",
			Desc: "Sexo de la persona.",
			Codes: []RISCode{
				c("M", "Masculino"), c("F", "Femenino"),
			},
		},
		{
			Name: "TIPO-BENEFICIARIO", Campo: "3.6", Pos: "11-12",
			Desc: "Tipo de persona que se informa.",
			Codes: []RISCode{
				c("99", "Afiliado (causante)"),
				c("10", "Cónyuge sin hijos con derecho a pensión"),
				c("11", "Cónyuge con hijos con derecho a pensión"),
				c("20", "Madre de hijo de filiación no matrimonial, sin hijos con derecho a pensión"),
				c("21", "Madre de hijo de filiación no matrimonial, con hijos con derecho a pensión"),
				c("30", "Hijo sin derecho a incremento"),
				c("35", "Hijo con derecho a incremento"),
				c("41", "Padre"),
				c("42", "Madre"),
				c("50", "Conviviente civil sin hijos comunes con derecho a pensión ni hijos del causante con derecho a pensión"),
				c("51", "Conviviente civil con hijos comunes con derecho a pensión"),
				c("52", "Conviviente civil cuando existen hijos del causante con derecho a pensión y no existen hijos comunes o no tienen derecho a pensión"),
				c("77", "Beneficiario designado"),
			},
		},
		{
			Name: "SITUACION-INVALIDEZ", Campo: "3.7", Pos: "13",
			Desc: "Situación de invalidez de la persona.",
			Codes: []RISCode{
				c("N", "No inválido"), c("T", "Inválido total"), c("P", "Inválido parcial"),
			},
		},
		{
			Name: "DERECHO-PENSION", Campo: "3.11", Pos: "38-39",
			Desc: "Si la persona tiene o no derecho a pensión (independiente de si recibe renta).",
			Codes: []RISCode{
				c("99", "Tiene derecho a pensión"),
				c("10", "No tiene derecho a pensión"),
				c("20", "Derecho a pensión no acreditado"),
			},
		},
		{
			Name: "REQUISITO-PENSION", Campo: "3.12", Pos: "40",
			Desc: "Situación de excepción que motiva la pérdida de requisitos para tener derecho a pensión de sobrevivencia.",
			Codes: []RISCode{
				c("1", "No constituye excepción"),
				c("2", "Ex-cónyuge"),
				c("3", "Ex-MHN (madre/padre de hijo de filiación no matrimonial)"),
				c("4", "Hijo sin derecho"),
				c("5", "Cónyuge post-póliza"),
				c("6", "Ex-beneficiario designado"),
				c("7", "Hijo post-póliza"),
				c("8", "Conviviente civil post-póliza"),
				c("9", "Ex-conviviente civil"),
			},
		},
		{
			Name: "DERECHO-ACRECER", Campo: "3.15", Pos: "51",
			Desc: "Si la persona tiene o no derecho a acrecer.",
			Codes: []RISCode{
				c("S", "Sí (cónyuge o MHN con derecho a acrecer)"),
				c("N", "No (cónyuge/MHN sin derecho a acrecer; afiliado y demás beneficiarios siempre N)"),
			},
		},
		{
			Name: "TIPO-PAGO-BENEFICIO-ESTATAL", Campo: "3.40", Pos: "214-216",
			Desc: "Tipo de beneficio estatal que se pagó a la persona (por mes del trimestre: ocurrencia 1/2/3).",
			Codes: []RISCode{
				c("0", "No corresponde informar"),
				c("1", "Garantía Estatal por pensión mínima"),
				c("2", "Aporte Previsional Solidario (APS)"),
				c("3", "No se efectuó pago"),
			},
		},
	}
}

// LookupRISCode busca el label de un código dentro de un campo del diccionario.
// Devuelve "" si el campo o el código no existen.
func LookupRISCode(fieldName, code string) string {
	for _, f := range RISDictionary() {
		if f.Name != fieldName {
			continue
		}
		for _, rc := range f.Codes {
			if rc.Code == code {
				return rc.Label
			}
		}
	}
	return ""
}

// DescribeModalidadRenta interpreta un código MODALIDAD-RENTA (pos 56-59),
// incluidas las familias 2xxx/3xxx/4xxx, y devuelve una descripción completa.
func DescribeModalidadRenta(code string, periodoAumento int) string {
	if len(code) != 4 {
		return code
	}
	meses := code[1:]
	switch code[0] {
	case '1':
		return "RV simple, sin adicionales"
	case '2':
		s := "RV con aumento temporal"
		if periodoAumento > 0 {
			s += fmtAumento(periodoAumento)
		}
		if meses != "000" {
			s += "; período garantizado " + meses + " meses"
		}
		return s
	case '3':
		return "RV con período garantizado de pago, " + meses + " meses"
	case '4':
		s := "RV con cláusula de aumento de % de pensión de sobrevivencia"
		if meses != "000" {
			s += "; período garantizado " + meses + " meses"
		}
		return s
	case '0':
		return "Póliza pre-2005 que no informa modalidad"
	default:
		return code
	}
}

func fmtAumento(meses int) string {
	return fmt.Sprintf(" (aumento temporal %d meses)", meses)
}
