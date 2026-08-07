package models

import "time"

// mortalityCategory groups a family member into the mortality family that
// determines which table applies: rentista (RV), sobreviviente (B/CB) or
// invalidez (MI).
type mortalityCategory string

const (
	catRV        mortalityCategory = "RV"
	catSobreviv  mortalityCategory = "SOBREVIV"
	catInvalidez mortalityCategory = "INVALIDEZ"
)

func tableCategory(rol BeneficiarioRol, tipoTabla string) mortalityCategory {
	if tipoTabla == string(TableTypeInvalidez) {
		return catInvalidez
	}
	if rol == RolCausante {
		return catRV
	}
	return catSobreviv
}

// Cut-off dates from Cuadro 4 of Nota Técnica N°9 (SPensiones, Nov 2024).
// These are the official table-vigencia boundaries.
var (
	periodStart1 = time.Date(2005, 2, 1, 0, 0, 0, 0, time.UTC) // RV-2004 / B-85
	periodStart2 = time.Date(2008, 2, 1, 0, 0, 0, 0, time.UTC) // RV-2004 / B-2006
	periodStart3 = time.Date(2010, 7, 1, 0, 0, 0, 0, time.UTC) // RV-2009 / B-2006
	periodStart4 = time.Date(2016, 7, 1, 0, 0, 0, 0, time.UTC) // CB-2014 / B-2014
	periodStart5 = time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC) // CB-2020 / B-2020
)

// cuadro4Table returns the mortality table name for a given category, sex, and
// date, following the official Cuadro 4 of Nota Técnica N°9 (SPensiones).
// Each category (afiliado, beneficiario, inválido) has its OWN table vintage
// within the same period — they are NOT the same.
func cuadro4Table(cat mortalityCategory, mortSex string, d time.Time) string {
	isHombre := mortSex == "H"

	switch {
	case d.Before(periodStart1):
		// Pre 01-02-2005: RV-85 / B-85 / MI-85
		return cuadro4Prefix(cat, isHombre, "1985")

	case d.Before(periodStart2):
		// 01-02-2005 to 31-01-2008: RV-2004 / B-85 / MI-85
		if cat == catRV {
			return prefixSex("RV", isHombre, "2004")
		}
		return cuadro4Prefix(cat, isHombre, "1985")

	case d.Before(periodStart3):
		// 01-02-2008 to 30-06-2010: RV-2004 / B-2006 / MI-2006
		if cat == catRV {
			return prefixSex("RV", isHombre, "2004")
		}
		return cuadro4Prefix(cat, isHombre, "2006")

	case d.Before(periodStart4):
		// 01-07-2010 to 30-06-2016: RV-2009 / B-2006 / MI-2006
		if cat == catRV {
			return prefixSex("RV", isHombre, "2009")
		}
		return cuadro4Prefix(cat, isHombre, "2006")

	case d.Before(periodStart5):
		// 01-07-2016 to 30-06-2023: CB-2014/RV-2014 / CB-2014/B-2014 / MI-2014
		return cuadro4Period5(cat, isHombre)

	default:
		// 01-07-2023+: CB-2020/RV-2020 / CB-2020/B-2020 / MI-2020
		return cuadro4Period6(cat, isHombre)
	}
}

// cuadro4Prefix returns the table name for a given category, sex, and year.
// For B and MI categories it's straightforward. For RV (afiliado) the naming
// varies by era (men use CB in newer eras).
//
// Special case: MI-1985 (inválido pre-2005) does not exist — Circular 491
// only defined B-85 and RV-85. Pre-2005 invalidez is valued with B-85 tables
// (per Circular 491, no MI-85 was published).
func cuadro4Prefix(cat mortalityCategory, isHombre bool, year string) string {
	switch cat {
	case catRV:
		return rvTableFor(isHombre, year)
	case catInvalidez:
		if year == "1985" {
			return bTableFor(isHombre, "1985")
		}
		return prefixSex("MI", isHombre, year)
	default: // catSobreviv
		return bTableFor(isHombre, year)
	}
}

// rvTableFor resolves the afiliado (rentista) table name by sex and vintage.
func rvTableFor(isHombre bool, year string) string {
	switch year {
	case "1985":
		return prefixSex("RV", isHombre, "1985")
	case "2004":
		return prefixSex("RV", isHombre, "2004")
	case "2009":
		return prefixSex("RV", isHombre, "2009")
	case "2014":
		if isHombre {
			return "CB-H-2014"
		}
		return "RV-M-2014"
	case "2020":
		if isHombre {
			return "CB-H-2020"
		}
		return "RV-M-2020"
	default:
		return prefixSex("RV", isHombre, year)
	}
}

// bTableFor resolves the beneficiario (sobreviviente) table name by sex and vintage.
func bTableFor(isHombre bool, year string) string {
	switch year {
	case "1985":
		return prefixSex("B", isHombre, "1985")
	case "2006":
		if isHombre {
			return "B-H-2006"
		}
		return "B-M-2006"
	case "2014":
		if isHombre {
			return "CB-H-2014"
		}
		return "B-M-2014"
	case "2020":
		if isHombre {
			return "CB-H-2020"
		}
		return "B-M-2020"
	default:
		return prefixSex("B", isHombre, year)
	}
}

// cuadro4Period5 returns tables for 2016-07-01 to 2023-06-30.
func cuadro4Period5(cat mortalityCategory, isHombre bool) string {
	switch cat {
	case catRV:
		return rvTableFor(isHombre, "2014")
	case catInvalidez:
		return prefixSex("MI", isHombre, "2014")
	default:
		return bTableFor(isHombre, "2014")
	}
}

// cuadro4Period6 returns tables for 2023-07-01 onwards.
func cuadro4Period6(cat mortalityCategory, isHombre bool) string {
	switch cat {
	case catRV:
		return rvTableFor(isHombre, "2020")
	case catInvalidez:
		return prefixSex("MI", isHombre, "2020")
	default:
		return bTableFor(isHombre, "2020")
	}
}

func prefixSex(prefix string, isHombre bool, year string) string {
	if isHombre {
		return prefix + "-H-" + year
	}
	return prefix + "-M-" + year
}

// SelectBaseTable returns the mortality table anchored to a policy by its
// contract date, following Cuadro 4 of Nota Técnica N°9.
// Members of the same policy may get DIFFERENT table vintages because
// afiliado (RV), beneficiario (B), and inválido (MI) have independent vigencias.
func SelectBaseTable(rol BeneficiarioRol, tipoTabla, sexo string, contractDate time.Time) string {
	cat := tableCategory(rol, tipoTabla)
	return cuadro4Table(cat, MapSexoToMortality(sexo), contractDate)
}

// SelectContemporaneaTable returns the mortality table in force for new
// business on the given date (latest period in Cuadro 4).
func SelectContemporaneaTable(rol BeneficiarioRol, tipoTabla, sexo string, at time.Time) string {
	cat := tableCategory(rol, tipoTabla)
	return cuadro4Table(cat, MapSexoToMortality(sexo), at)
}
