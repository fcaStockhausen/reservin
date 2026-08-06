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

// Cut-off dates from the CMF Circular N°2332 / NCG N°318 regime.
var (
	cutoff2012 = time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff2008 = time.Date(2008, 2, 1, 0, 0, 0, 0, time.UTC)
	cutoff2005 = time.Date(2005, 3, 9, 0, 0, 0, 0, time.UTC)
	cutoff2014 = time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)
	cutoff2023 = time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)
)

// baseTableYear returns the vintage year of the mortality table anchored to the
// policy contract date, following the Circular N°2332 stratification.
func baseTableYear(cat mortalityCategory, d time.Time) int {
	if !d.Before(cutoff2012) {
		return 2020
	}
	switch cat {
	case catInvalidez:
		return 2006 // MI-2006 (MI-85 not loaded in repo data)
	case catSobreviv:
		return 2006 // B-2006 (B-85 not loaded in repo data)
	default: // catRV
		if !d.Before(cutoff2008) {
			return 2009
		}
		if !d.Before(cutoff2005) {
			return 2009
		}
		return 2004 // RV-85 fallback -> oldest RV table loaded
	}
}

// contemporaneaYear returns the vintage of the mortality table in force for new
// business on date d.
func contemporaneaYear(cat mortalityCategory, d time.Time) int {
	if !d.Before(cutoff2023) {
		return 2020
	}
	if !d.Before(cutoff2014) {
		return 2014
	}
	if cat == catRV {
		return 2009
	}
	return 2006
}

// tableName resolves the standard table name for a category/sex/vintage combo,
// falling back to the closest available table when an exact vintage is absent.
func tableName(cat mortalityCategory, mortSex string, year int) string {
	switch cat {
	case catInvalidez:
		return "MI-" + mortSex + "-" + itoa(clampYear(year, 2006, 2020))

	case catRV:
		// Male RV tables exist only for 2004/2009; newer eras use CB-H.
		if mortSex == "H" {
			if year <= 2009 {
				return "RV-H-" + itoa(year)
			}
			return "CB-H-" + itoa(clampYear(year, 2014, 2020))
		}
		return "RV-M-" + itoa(clampYear(year, 2004, 2020))

	default: // catSobreviv
		if mortSex == "H" {
			if year == 2006 {
				return "B-H-2006"
			}
			return "CB-H-" + itoa(clampYear(year, 2014, 2020))
		}
		return "B-M-" + itoa(clampYear(year, 2006, 2020))
	}
}

func clampYear(year, lo, hi int) int {
	if year < lo {
		return lo
	}
	if year > hi {
		return hi
	}
	return year
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// SelectBaseTable returns the mortality table anchored to a policy by its
// contract date (the "tabla de bautizo" for the reserva base of the stale
// regime). Members of the same policy share the stratum table; only sexo,
// rol and invalidez status differ.
func SelectBaseTable(rol BeneficiarioRol, tipoTabla, sexo string, contractDate time.Time) string {
	cat := tableCategory(rol, tipoTabla)
	return tableName(cat, MapSexoToMortality(sexo), baseTableYear(cat, contractDate))
}

// SelectContemporaneaTable returns the mortality table in force for new
// business on the given date. It is the reference table for the descalce.
func SelectContemporaneaTable(rol BeneficiarioRol, tipoTabla, sexo string, at time.Time) string {
	cat := tableCategory(rol, tipoTabla)
	return tableName(cat, MapSexoToMortality(sexo), contemporaneaYear(cat, at))
}
