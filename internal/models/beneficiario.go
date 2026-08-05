package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// BeneficiarioRol represents the role of a family group member
type BeneficiarioRol string

const (
	RolCausante BeneficiarioRol = "CAUSANTE"
	RolConyuge  BeneficiarioRol = "CONYUGE"
	RolHijo     BeneficiarioRol = "HIJO"
	RolOtro     BeneficiarioRol = "OTRO"
)

// Beneficiario represents a member of a policy's family group.
// A policy has exactly one CAUSANTE (the primary insured) and zero or more
// additional members (CONYUGE, HIJO, OTRO) who may receive survivor benefits.
type Beneficiario struct {
	ID                int              `json:"id" db:"id"`
	PolizaID          int              `json:"poliza_id" db:"poliza_id"`
	Rol               BeneficiarioRol  `json:"rol" db:"rol"`
	Sexo              string           `json:"sexo" db:"sexo"` // H, M
	EdadContratacion  int              `json:"edad_contratacion" db:"edad_contratacion"`
	FechaNacimiento   *time.Time       `json:"fecha_nacimiento,omitempty" db:"fecha_nacimiento"`
	TablaAsignada     string           `json:"tabla_asignada" db:"tabla_asignada"`
	PorcentajeRenta   decimal.Decimal  `json:"porcentaje_renta" db:"porcentaje_renta"` // 0.0 - 1.0
	Estado            string           `json:"estado" db:"estado"` // ACTIVO, FALLECIDO, EXCLUIDO
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
}

// GrupoFamiliar represents the complete family group for a policy.
type GrupoFamiliar struct {
	PolizaID      int
	Causante      *Beneficiario
	Beneficiarios []*Beneficiario // all members except causante
}

// HasBeneficiarios returns true if the group has survivor beneficiaries.
func (gf *GrupoFamiliar) HasBeneficiarios() bool {
	return len(gf.Beneficiarios) > 0
}

// AllMembers returns causante first, then beneficiaries.
func (gf *GrupoFamiliar) AllMembers() []*Beneficiario {
	all := make([]*Beneficiario, 0, 1+len(gf.Beneficiarios))
	if gf.Causante != nil {
		all = append(all, gf.Causante)
	}
	all = append(all, gf.Beneficiarios...)
	return all
}

// SelectTableForBeneficiario determines the CMF mortality table for a family member
// based on their role, sex, and the policy's methodology period.
//
// Regulatory mapping (Circular N°2332, current tables):
//   - Causante hombre   → CB-H-2020
//   - Causante mujer RV → RV-M-2020 (Rentas Vitalicias)
//   - Beneficiario hombre (sobreviviente) → CB-H-2020
//   - Beneficiario mujer (sobreviviente)  → B-M-2020
//   - Invalidez hombre  → MI-H-2020
//   - Invalidez mujer   → MI-M-2020
//
// For pre-2012 (traditional) policies, legacy tables B-2006 / MI-2006 are used.
func SelectTableForBeneficiario(rol BeneficiarioRol, sexo string, methodology PolicyMethodology, tipoTabla string) string {
	// Legacy tables for pre-2012 policies
	if methodology == MethodologyTraditional {
		return selectLegacyTable(rol, sexo, tipoTabla)
	}

	// Current tables (IFRS + transitional)
	switch {
	case tipoTabla == string(TableTypeInvalidez):
		if sexo == "M" {
			return "MI-M-2020"
		}
		return "MI-H-2020"

	case rol == RolCausante && sexo == "M":
		return "RV-M-2020"

	case sexo == "M":
		return "B-M-2020"

	default:
		return "CB-H-2020"
	}
}

// selectLegacyTable returns the pre-2012 table names.
func selectLegacyTable(rol BeneficiarioRol, sexo string, tipoTabla string) string {
	switch {
	case tipoTabla == string(TableTypeInvalidez):
		if sexo == "M" {
			return "MI-M-2006"
		}
		return "MI-H-2006"

	case sexo == "M":
		return "B-M-2006"

	default:
		return "B-H-2006"
	}
}

// ValidateBeneficiario validates a beneficiario record before insertion.
func ValidateBeneficiario(b Beneficiario) error {
	validRoles := map[BeneficiarioRol]bool{
		RolCausante: true,
		RolConyuge:  true,
		RolHijo:     true,
		RolOtro:     true,
	}
	if !validRoles[b.Rol] {
		return fmt.Errorf("rol invalido: %s", b.Rol)
	}

	if b.Sexo != "H" && b.Sexo != "M" {
		return ErrInvalidSex
	}

	if b.EdadContratacion < 0 || b.EdadContratacion > 120 {
		return ErrInvalidAge
	}

	if b.PorcentajeRenta.LessThan(decimal.Zero) || b.PorcentajeRenta.GreaterThan(decimal.NewFromInt(1)) {
		return fmt.Errorf("porcentaje_renta fuera de rango [0,1]: %s", b.PorcentajeRenta.String())
	}

	if b.FechaNacimiento != nil && b.FechaNacimiento.After(time.Now()) {
		return ErrInvalidDate
	}

	return nil
}
