package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// ReserveResult represents the result of a reserve calculation
type ReserveResult struct {
	PolicyID        int               `json:"policy_id"`
	ReserveValue    decimal.Decimal   `json:"reserve_value"`
	Methodology     PolicyMethodology `json:"methodology"`
	DiscountRate    decimal.Decimal   `json:"discount_rate"`
	MortalityTable  string            `json:"mortality_table"`
	Flows           []CashFlow        `json:"flows"`
	DiscountFactors []decimal.Decimal `json:"discount_factors"`
	CalculationDate time.Time         `json:"calculation_date"`
	ProcessingTime  time.Duration     `json:"processing_time_ms"`
	AuditTrail      *AuditTrail       `json:"audit_trail"`
}

// CashFlow represents a single cash flow in reserve calculation
type CashFlow struct {
	Period         int             `json:"period"` // Year from policy start
	Date           time.Time       `json:"date"`
	Amount         decimal.Decimal `json:"amount"`
	SurvivalRate   decimal.Decimal `json:"survival_rate"`
	Probability    decimal.Decimal `json:"probability"`
	DiscountFactor decimal.Decimal `json:"discount_factor"`
	PresentValue   decimal.Decimal `json:"present_value"`
	Description    string          `json:"description"`
}

// AuditTrail represents the audit trail for CMF compliance
type AuditTrail struct {
	ID              int             `json:"id" db:"id"`
	PolicyID        int             `json:"policy_id" db:"policy_id"`
	CalculationDate time.Time       `json:"calculation_date" db:"calculation_date"`
	Methodology     string          `json:"methodology" db:"methodology"`
	Inputs          AuditInputs     `json:"inputs" db:"inputs"`
	Steps           []AuditStep     `json:"steps" db:"steps"`
	Outputs         AuditOutputs    `json:"outputs" db:"outputs"`
	Validation      AuditValidation `json:"validation" db:"validation"`
	Compliance      AuditCompliance `json:"compliance" db:"compliance"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// AuditInputs captures all inputs used in calculation
type AuditInputs struct {
	PolicyData            map[string]interface{} `json:"policy_data"`
	MortalityTable        map[string]interface{} `json:"mortality_table"`
	VTDData               map[string]interface{} `json:"vtd_data"`
	RateData              map[string]interface{} `json:"rate_data"`
	CalculationParameters map[string]interface{} `json:"calculation_parameters"`
}

// AuditStep represents each step in the calculation process
type AuditStep struct {
	StepNumber  int           `json:"step_number"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Inputs      interface{}   `json:"inputs"`
	Outputs     interface{}   `json:"outputs"`
	Formula     string        `json:"formula"`
	Timestamp   time.Time     `json:"timestamp"`
	Duration    time.Duration `json:"duration_ms"`
}

// AuditOutputs captures final calculation outputs
type AuditOutputs struct {
	ReserveValue       decimal.Decimal `json:"reserve_value"`
	TotalFlows         decimal.Decimal `json:"total_flows"`
	DiscountedFlows    decimal.Decimal `json:"discounted_flows"`
	NetPresentValue    decimal.Decimal `json:"net_present_value"`
	RoundingAdjustment decimal.Decimal `json:"rounding_adjustment"`
	FinalReserve       decimal.Decimal `json:"final_reserve"`
}

// AuditValidation represents validation checks performed
type AuditValidation struct {
	DataValidation        ValidationResults `json:"data_validation"`
	CalculationValidation ValidationResults `json:"calculation_validation"`
	ComplianceValidation  ValidationResults `json:"compliance_validation"`
}

// ValidationResults represents the results of validation checks
type ValidationResults struct {
	Checks   []ValidationCheck `json:"checks"`
	Passed   bool              `json:"passed"`
	Errors   []string          `json:"errors"`
	Warnings []string          `json:"warnings"`
}

// ValidationCheck represents a single validation check
type ValidationCheck struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Result      string    `json:"result"` // PASS, FAIL, WARNING
	Message     string    `json:"message"`
	CheckDate   time.Time `json:"check_date"`
}

// AuditCompliance represents compliance with CMF regulations
type AuditCompliance struct {
	CMFRules       []CMFComplianceRule `json:"cmf_rules"`
	OverallStatus  string              `json:"overall_status"` // COMPLIANT, NON_COMPLIANT, WARNING
	RegulatoryRef  string              `json:"regulatory_reference"`
	ReportRequired bool                `json:"report_required"`
}

// CMFComplianceRule represents a specific CMF compliance rule
type CMFComplianceRule struct {
	RuleID      string    `json:"rule_id"`
	Description string    `json:"description"`
	Requirement string    `json:"requirement"`
	Status      string    `json:"status"`    // COMPLIANT, NON_COMPLIANT, NOT_APPLICABLE
	Reference   string    `json:"reference"` // NCG 318, Circular 1512, etc.
	CheckDate   time.Time `json:"check_date"`
}

// ReserveCalculated represents a calculated reserve stored in database
type ReserveCalculated struct {
	ID                       int             `json:"id" db:"id"`
	PolicyID                 int             `json:"policy_id" db:"policy_id"`
	FechaCalculo             time.Time       `json:"fecha_calculo" db:"fecha_calculo"`
	ValorReserva             decimal.Decimal `json:"valor_reserva" db:"valor_reserva"`
	MetodoCalculo            string          `json:"metodo_calculo" db:"metodo_calculo"` // VPPJ, TRADICIONAL
	FlujoProbabilistico      decimal.Decimal `json:"flujo_probabilistico" db:"flujo_probabilistico"`
	TasaDescuentoUtilizada   decimal.Decimal `json:"tasa_descuento_utilizada" db:"tasa_descuento_utilizada"`
	TablaMortalidadUtilizada string          `json:"tabla_mortalidad_utilizada" db:"tabla_mortalidad_utilizada"`
	AuditTrailID             *int            `json:"audit_trail_id,omitempty" db:"audit_trail_id"`
	CreatedAt                time.Time       `json:"created_at" db:"created_at"`
}

// CalculationMethod represents the method used for reserve calculation
type CalculationMethod string

const (
	CalculationMethodVPPJ         CalculationMethod = "VPPJ"         // IFRS method
	CalculationMethodTraditional  CalculationMethod = "TRADICIONAL"  // Pre-IFRS method
	CalculationMethodTransitional CalculationMethod = "TRANSITIONAL" // Transitional period
)

// NewReserveResult creates a new reserve result
func NewReserveResult(policyID int) *ReserveResult {
	return &ReserveResult{
		PolicyID:        policyID,
		Flows:           make([]CashFlow, 0),
		DiscountFactors: make([]decimal.Decimal, 0),
		CalculationDate: time.Now(),
		AuditTrail: &AuditTrail{
			Steps: make([]AuditStep, 0),
		},
	}
}

// AddCashFlow adds a cash flow to the reserve calculation
func (rr *ReserveResult) AddCashFlow(flow CashFlow) {
	rr.Flows = append(rr.Flows, flow)
}

// SetDiscountRate sets the discount rate used in calculation
func (rr *ReserveResult) SetDiscountRate(rate decimal.Decimal) {
	rr.DiscountRate = rate
}

// SetMethodology sets the calculation methodology
func (rr *ReserveResult) SetMethodology(methodology PolicyMethodology) {
	rr.Methodology = methodology
}

// SetMortalityTable sets the mortality table used
func (rr *ReserveResult) SetMortalityTable(table string) {
	rr.MortalityTable = table
}

// CalculatePresentValues calculates present values for all cash flows
func (rr *ReserveResult) CalculatePresentValues() error {
	for i := range rr.Flows {
		flow := &rr.Flows[i]

		// Calculate discount factor: (1 + rate)^-period
		discountFactor := decimal.NewFromInt(1).Add(rr.DiscountRate).Pow(decimal.NewFromInt(-int64(flow.Period)))

		// Calculate present value
		presentValue := flow.Amount.Mul(flow.Probability).Mul(discountFactor)

		flow.DiscountFactor = discountFactor
		flow.PresentValue = presentValue

		rr.DiscountFactors = append(rr.DiscountFactors, discountFactor)
	}

	return nil
}

// CalculateTotalPresentValue calculates the total present value of all cash flows
func (rr *ReserveResult) CalculateTotalPresentValue() decimal.Decimal {
	total := decimal.Zero
	for _, flow := range rr.Flows {
		total = total.Add(flow.PresentValue)
	}
	return total
}

// SetProcessingTime sets the processing time of the calculation
func (rr *ReserveResult) SetProcessingTime(duration time.Duration) {
	rr.ProcessingTime = duration
}

// GetProcessingTimeMs returns processing time in milliseconds
func (rr *ReserveResult) GetProcessingTimeMs() int64 {
	return rr.ProcessingTime.Nanoseconds() / 1e6
}

// ToReserveCalculated converts to database storage format
func (rr *ReserveResult) ToReserveCalculated() *ReserveCalculated {
	return &ReserveCalculated{
		PolicyID:                 rr.PolicyID,
		FechaCalculo:             rr.CalculationDate,
		ValorReserva:             rr.CalculateTotalPresentValue(),
		MetodoCalculo:            string(CalculationMethodVPPJ), // Default for now
		FlujoProbabilistico:      rr.CalculateTotalCashFlow(),
		TasaDescuentoUtilizada:   rr.DiscountRate,
		TablaMortalidadUtilizada: rr.MortalityTable,
		CreatedAt:                time.Now(),
	}
}

// CalculateTotalCashFlow calculates the total nominal cash flow amount
func (rr *ReserveResult) CalculateTotalCashFlow() decimal.Decimal {
	total := decimal.Zero
	for _, flow := range rr.Flows {
		total = total.Add(flow.Amount.Mul(flow.Probability))
	}
	return total
}

// AddAuditStep adds a step to the audit trail
func (rr *ReserveResult) AddAuditStep(step AuditStep) {
	if rr.AuditTrail != nil {
		rr.AuditTrail.Steps = append(rr.AuditTrail.Steps, step)
	}
}

// InitializeAuditTrail initializes the audit trail with basic information
func (rr *ReserveResult) InitializeAuditTrail(policy Policy, inputs AuditInputs) {
	if rr.AuditTrail == nil {
		rr.AuditTrail = &AuditTrail{
			Steps: make([]AuditStep, 0),
		}
	}

	rr.AuditTrail.PolicyID = policy.ID
	rr.AuditTrail.CalculationDate = rr.CalculationDate
	rr.AuditTrail.Methodology = string(rr.Methodology)
	rr.AuditTrail.Inputs = inputs
	rr.AuditTrail.Outputs = AuditOutputs{
		ReserveValue: rr.CalculateTotalPresentValue(),
		TotalFlows:   rr.CalculateTotalCashFlow(),
	}
}
