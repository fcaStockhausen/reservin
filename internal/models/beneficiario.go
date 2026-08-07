package models

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// BeneficiarioRol represents the role of a family group member.
// Internal representation uses descriptive names; RIS C1194 uses numeric
// codes stored in TipoBeneficiarioC1194.
type BeneficiarioRol string

const (
	RolCausante       BeneficiarioRol = "CAUSANTE"
	RolConyuge        BeneficiarioRol = "CONYUGE"
	RolConviviente    BeneficiarioRol = "CONVIVIENTE_CIVIL"
	RolHijo           BeneficiarioRol = "HIJO"
	RolMadrePadreNMat BeneficiarioRol = "MADRE_PADRE_NMAT"
	RolPadres         BeneficiarioRol = "PADRES"
	RolDesignado      BeneficiarioRol = "DESIGNADO"
)

// TipoBeneficiarioC1194 codes from Anexo Tecnico Circular 1194 campo 3.10.
const (
	C1194Afiliado           = "99" // Causante
	C1194ConyugeSinHijos    = "10" // Conyuge sin hijos con derecho
	C1194ConyugeConHijos    = "11" // Conyuge con hijos con derecho
	C1194MHNsinHijos        = "20" // Madre/padre no matrimonial sin hijos c/derecho
	C1194MHNconHijos        = "21" // Madre/padre no matrimonial con hijos c/derecho
	C1194HijoSinIncremento  = "30" // Hijo sin derecho a incremento
	C1194HijoConIncremento  = "35" // Hijo con derecho a incremento
	C1194Padre              = "41"
	C1194Madre              = "42"
	C1194CCsinHijos         = "50" // Conviviente civil sin hijos comunes ni del causante
	C1194CCconHijosComunes  = "51" // Conviviente civil con hijos comunes c/derecho
	C1194CCconHijosCausante = "52" // Conviviente civil con hijos del causante, no comunes
	C1194Designado          = "77"
)

// DerechoPension codes from C1194 campo 3.15.
const (
	DerechoPensionSi      = "99" // Tiene derecho a pension
	DerechoPensionNo      = "10" // No tiene derecho a pension
	DerechoPensionNoAcred = "20" // Derecho a pension no acreditado
)

// SituacionInvalidez codes from C1194 campo 3.11.
const (
	InvNo      = "N" // No invalido
	InvTotal   = "T" // Invalido total
	InvParcial = "P" // Invalido parcial
)

// SexoC1194 codes from C1194 campo 3.9.
const (
	SexoMasculino = "M"
	SexoFemenino  = "F"
)

// MapSexoToMortality maps C1194 sex codes (M/F) to mortality table sex codes (H/M).
// Mortality tables loaded from Excel use H=Hombres, M=Mujeres.
func MapSexoToMortality(sexoC1194 string) string {
	if sexoC1194 == SexoFemenino {
		return "M" // Mujeres in mortality tables
	}
	return "H" // Hombres
}

// Beneficiario represents a member of a policy's family group.
type Beneficiario struct {
	ID               int             `json:"id" db:"id"`
	PolizaID         int             `json:"poliza_id" db:"poliza_id"`
	Rol              BeneficiarioRol `json:"rol" db:"rol"`
	Sexo             string          `json:"sexo" db:"sexo"` // C1194: M (masculino), F (femenino)
	EdadContratacion int             `json:"edad_contratacion" db:"edad_contratacion"`
	FechaNacimiento  *time.Time      `json:"fecha_nacimiento,omitempty" db:"fecha_nacimiento"`
	TablaAsignada    string          `json:"tabla_asignada" db:"tabla_asignada"`
	PorcentajeRenta  decimal.Decimal `json:"porcentaje_renta" db:"porcentaje_renta"`
	Estado           string          `json:"estado" db:"estado"` // ACTIVO, FALLECIDO, EXCLUIDO

	// RIS C1194 fields
	TipoBeneficiarioC1194 string `json:"tipo_beneficiario_c1194" db:"tipo_beneficiario_c1194"`
	DerechoPension        string `json:"derecho_pension" db:"derecho_pension"`
	RequisitoPension      string `json:"requisito_pension" db:"requisito_pension"`
	DerechoAcrecer        string `json:"derecho_acrecer" db:"derecho_acrecer"`
	SituacionInvalidez    string `json:"situacion_invalidez" db:"situacion_invalidez"`
	Condicion             string `json:"condicion,omitempty" db:"condicion"` // MENOR, ESTUDIANTE, INVALIDO
	MatrimonioAnios       int    `json:"matrimonio_anios" db:"matrimonio_anios"`
	HijosComunes          int    `json:"hijos_comunes" db:"hijos_comunes"`
	FinDerechoEdad        *int   `json:"fin_derecho_edad,omitempty" db:"fin_derecho_edad"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// GrupoFamiliar represents the complete family group for a policy.
type GrupoFamiliar struct {
	PolizaID      int
	Causante      *Beneficiario
	Beneficiarios []*Beneficiario
}

func (gf *GrupoFamiliar) HasBeneficiarios() bool {
	return len(gf.Beneficiarios) > 0
}

func (gf *GrupoFamiliar) AllMembers() []*Beneficiario {
	all := make([]*Beneficiario, 0, 1+len(gf.Beneficiarios))
	if gf.Causante != nil {
		all = append(all, gf.Causante)
	}
	all = append(all, gf.Beneficiarios...)
	return all
}

// HasHijosConDerecho returns true if there are active hijos with pension rights.
// This determines whether conyuge gets 50% (with hijos) or 60% (without).
func (gf *GrupoFamiliar) HasHijosConDerecho() bool {
	for _, b := range gf.Beneficiarios {
		if b.Rol == RolHijo && b.DerechoPension == DerechoPensionSi {
			return true
		}
	}
	return false
}

// SelectTableForBeneficiario determines the CMF mortality table for a family member
// anchored to the policy's contract date (the "tabla de bautizo"). It delegates
// to SelectBaseTable, which applies the Circular N°2332 stratification (pre-2005
// → RV-85/B-85, 2005-2011 → RV-2009/B-2006, post-2012 → TM-2020).
// sexo is in C1194 format: M (masculino) or F (femenino).
// tipoC1194 is the C1194 beneficiary type code (10/11/20/21/30/35/41/42/50/51/52/77/99).
func SelectTableForBeneficiario(
	rol BeneficiarioRol,
	sexo string,
	tipoC1194 string,
	fechaContratacion time.Time,
	tipoTabla string,
) string {
	return SelectBaseTable(rol, tipoTabla, sexo, fechaContratacion)
}

// CalcularPorcentajeSobrevivencia computes the legal survivor pension percentage
// based on C1194 beneficiary type and family group composition.
func CalcularPorcentajeSobrevivencia(tipoC1194 string, hasHijosConDerecho bool) decimal.Decimal {
	switch tipoC1194 {
	case C1194ConyugeSinHijos, C1194CCsinHijos, C1194CCconHijosCausante:
		// Conyuge/CC sin hijos con derecho: 60%
		if !hasHijosConDerecho {
			return decimal.NewFromFloat(0.60)
		}
		return decimal.NewFromFloat(0.50)

	case C1194ConyugeConHijos, C1194CCconHijosComunes:
		return decimal.NewFromFloat(0.50)

	case C1194MHNsinHijos:
		if !hasHijosConDerecho {
			return decimal.NewFromFloat(0.36)
		}
		return decimal.NewFromFloat(0.30)

	case C1194MHNconHijos:
		return decimal.NewFromFloat(0.30)

	case C1194HijoSinIncremento, C1194HijoConIncremento:
		return decimal.NewFromFloat(0.15)

	case C1194Padre, C1194Madre:
		return decimal.NewFromFloat(0.50)

	case C1194Designado:
		return decimal.Zero // Designados solo reciben dentro de periodo garantizado

	default:
		return decimal.Zero
	}
}

func ValidateBeneficiario(b Beneficiario) error {
	validRoles := map[BeneficiarioRol]bool{
		RolCausante: true, RolConyuge: true, RolConviviente: true,
		RolHijo: true, RolMadrePadreNMat: true, RolPadres: true, RolDesignado: true,
	}
	if !validRoles[b.Rol] {
		return fmt.Errorf("rol invalido: %s", b.Rol)
	}

	if b.Sexo != SexoMasculino && b.Sexo != SexoFemenino {
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
