package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Policy represents an insurance policy for rentas vitalicias
type Policy struct {
	ID               int              `json:"id" db:"id"`
	NumeroPoliza     string           `json:"numero_poliza" db:"numero_poliza"`
	TipoRenta        string           `json:"tipo_renta" db:"tipo_renta"` // VITALICIA, TEMPORARIA, DIFERIDA
	FechaInicio      time.Time        `json:"fecha_inicio" db:"fecha_inicio"`
	FechaFin         *time.Time       `json:"fecha_fin,omitempty" db:"fecha_fin"`
	EdadContratante  int              `json:"edad_contratante" db:"edad_contratante"`
	SexoBeneficiario string           `json:"sexo_beneficiario" db:"sexo_beneficiario"` // H, M
	CapitalAsegurado decimal.Decimal   `json:"capital_asegurado" db:"capital_asegurado"`
	FormaPago       string           `json:"forma_pago" db:"forma_pago"` // MENSUAL, TRIMESTRAL, ANUAL
	TasaDescuento    decimal.Decimal   `json:"tasa_descuento" db:"tasa_descuento"` // "bautizo" rate
	TasaTM           decimal.Decimal   `json:"tasa_tm" db:"tasa_tm"`          // Tasa venta
	TasaTC           decimal.Decimal   `json:"tasa_tc" db:"tasa_tc"`          // Tasa costo
	Estado           string           `json:"estado" db:"estado"`            // ACTIVA, VENCIDA, CANCELADA
	TipoTabla        string           `json:"tipo_tabla" db:"tipo_tabla"`    // VEJEZ, INVALIDEZ, SOBREVIVENCIA
	CreatedAt        time.Time        `json:"created_at" db:"created_at"`
}

// PolicyMethodology represents the calculation methodology based on policy date
type PolicyMethodology string

const (
	MethodologyIFRS        PolicyMethodology = "IFRS"         // Post-January 1, 2012
	MethodologyTraditional PolicyMethodology = "TRADICIONAL"   // Pre-January 1, 2012
	MethodologyTransitional PolicyMethodology = "TRANSITIONAL"  // 2015-2020 period
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

// GetEffectiveDiscountRate returns the effective discount rate (min(TM, TC))
func (p *Policy) GetEffectiveDiscountRate() decimal.Decimal {
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
	PolicyTypeTemporal   PolicyType = "TEMPORARIA"
	PolicyTypeDiferida   PolicyType = "DIFERIDA"
)

// PaymentFrequency represents a payment frequency
type PaymentFrequency string

const (
	PaymentFrequencyMensual   PaymentFrequency = "MENSUAL"
	PaymentFrequencyTrimestral PaymentFrequency = "TRIMESTRAL"
	PaymentFrequencyAnual     PaymentFrequency = "ANUAL"
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

	// Validate sex
	if policy.SexoBeneficiario != "H" && policy.SexoBeneficiario != "M" {
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