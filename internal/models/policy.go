package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Policy represents an insurance policy for rentas vitalicias
type Policy struct {
	ID               int             `json:"id" db:"id"`
	NumeroPoliza     string          `json:"numero_poliza" db:"numero_poliza"`
	TipoRenta        string          `json:"tipo_renta" db:"tipo_renta"`
	FechaInicio      time.Time       `json:"fecha_inicio" db:"fecha_inicio"`
	FechaFin         *time.Time      `json:"fecha_fin,omitempty" db:"fecha_fin"`
	EdadContratante  int             `json:"edad_contratante" db:"edad_contratante"`
	SexoBeneficiario string          `json:"sexo_beneficiario" db:"sexo_beneficiario"` // C1194: M, F
	CapitalAsegurado decimal.Decimal `json:"capital_asegurado" db:"capital_asegurado"`
	FormaPago        string          `json:"forma_pago" db:"forma_pago"`
	TasaDescuento    decimal.Decimal `json:"tasa_descuento" db:"tasa_descuento"`
	TasaTM           decimal.Decimal `json:"tasa_tm" db:"tasa_tm"`
	TasaTC           decimal.Decimal `json:"tasa_tc" db:"tasa_tc"`
	Estado           string          `json:"estado" db:"estado"`
	TipoTabla        string          `json:"tipo_tabla" db:"tipo_tabla"`

	// RIS C1194 fields (Registro 2)
	TipoPension       string          `json:"tipo_pension,omitempty" db:"tipo_pension"`             // 01-15 (C1194 campo 2.6)
	ModalidadRenta    string          `json:"modalidad_renta,omitempty" db:"modalidad_renta"`       // 1000/2xxx/3xxx/4xxx (C1194 campo 2.18)
	VigenciaPension   string          `json:"vigencia_pension,omitempty" db:"vigencia_pension"`     // 6/7/8/9 (C1194 campo 2.8)
	PeriodoAumento    int             `json:"periodo_aumento,omitempty" db:"periodo_aumento"`       // meses aumento temporal (C1194 campo 2.20)
	PorcentajeAumento decimal.Decimal `json:"porcentaje_aumento,omitempty" db:"porcentaje_aumento"` // % aumento (C1194 campo 2.21)

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PolicyMethodology represents the calculation methodology based on policy date
type PolicyMethodology string

const (
	MethodologyIFRS         PolicyMethodology = "IFRS"         // Post-January 1, 2012
	MethodologyTraditional  PolicyMethodology = "TRADICIONAL"  // Pre-January 1, 2012
	MethodologyTransitional PolicyMethodology = "TRANSITIONAL" // 2015-2020 period
)

// GetMethodology determines the appropriate calculation methodology based on policy date
func (p *Policy) GetMethodology() PolicyMethodology {
	// Check for transitional period first (NCG 374)
	if p.FechaInicio.After(time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC)) &&
		p.FechaInicio.Before(time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC)) {
		return MethodologyTransitional
	}

	// Post-IFRS policies
	if p.FechaInicio.After(time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC)) {
		return MethodologyIFRS
	}

	// Traditional pre-IFRS policies
	return MethodologyTraditional
}

// GetEffectiveDiscountRate returns the effective discount rate (min(TM, TC)).
// A zero TM/TC (e.g. pólizas que no reportan la tasa de costo de emisión) falls
// back to the other non-zero rate so the discount never collapses to 0.
func (p *Policy) GetEffectiveDiscountRate() decimal.Decimal {
	if p.TasaTM.IsZero() && !p.TasaTC.IsZero() {
		return p.TasaTC
	}
	if p.TasaTC.IsZero() && !p.TasaTM.IsZero() {
		return p.TasaTM
	}
	return decimal.Min(p.TasaTM, p.TasaTC)
}

// IsPost2012 returns true if the policy is subject to IFRS requirements
func (p *Policy) IsPost2012() bool {
	return p.FechaInicio.After(time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC))
}

// IsPost2020 returns true if the policy uses Annex rates with no calce
func (p *Policy) IsPost2020() bool {
	return p.FechaInicio.After(time.Date(2020, 12, 1, 0, 0, 0, 0, time.UTC))
}

// IsActive returns true if the policy is currently active
func (p *Policy) IsActive() bool {
	if p.Estado != "ACTIVA" {
		return false
	}
	if p.FechaFin != nil && time.Now().After(*p.FechaFin) {
		return false
	}
	return true
}

// PolicyType represents a type of insurance policy
type PolicyType string

const (
	PolicyTypeVitalicia PolicyType = "VITALICIA"
	PolicyTypeTemporal  PolicyType = "TEMPORARIA"
	PolicyTypeDiferida  PolicyType = "DIFERIDA"
)

// PaymentFrequency represents a payment frequency
type PaymentFrequency string

const (
	PaymentFrequencyMensual    PaymentFrequency = "MENSUAL"
	PaymentFrequencyTrimestral PaymentFrequency = "TRIMESTRAL"
	PaymentFrequencyAnual      PaymentFrequency = "ANUAL"
)

// PolicyStatus represents a status of a policy
type PolicyStatus string

const (
	PolicyStatusActiva    PolicyStatus = "ACTIVA"
	PolicyStatusVencida   PolicyStatus = "VENCIDA"
	PolicyStatusCancelada PolicyStatus = "CANCELADA"
)

// ValidatePolicy validates policy data before insertion/update
func ValidatePolicy(policy Policy) error {
	// Validate policy type
	validTypes := []string{"VITALICIA", "TEMPORARIA", "DIFERIDA"}
	isValidType := false
	for _, validType := range validTypes {
		if policy.TipoRenta == validType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return ErrInvalidPolicyType
	}

	// Validate sex (C1194 uses M=Masculino, F=Femenino)
	if policy.SexoBeneficiario != "M" && policy.SexoBeneficiario != "F" {
		return ErrInvalidSex
	}

	// Validate age
	if policy.EdadContratante < 18 || policy.EdadContratante > 120 {
		return ErrInvalidAge
	}

	// Validate capital
	if policy.CapitalAsegurado.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidCapital
	}

	// Validate rates
	if policy.TasaTM.LessThan(decimal.Zero) || policy.TasaTC.LessThan(decimal.Zero) {
		return ErrInvalidRate
	}

	// Validate dates
	if policy.FechaInicio.IsZero() || policy.FechaInicio.After(time.Now()) {
		return ErrInvalidDate
	}

	if policy.FechaFin != nil && policy.FechaFin.Before(policy.FechaInicio) {
		return ErrInvalidDate
	}

	return nil
}
