package models

import "errors"

var (
	// Common errors
	ErrPolicyNotFound          = errors.New("policy not found")
	ErrMortalityTableNotFound  = errors.New("mortality table not found")
	ErrVTDRateNotFound        = errors.New("VTD rate not found")
	ErrInvalidPolicyData       = errors.New("invalid policy data")
	ErrCalculationError        = errors.New("calculation error")
	ErrDatabaseError          = errors.New("database error")
	ErrFileNotFound           = errors.New("file not found")
	ErrInvalidInput           = errors.New("invalid input")
	ErrPermissionDenied       = errors.New("permission denied")
	
	// Policy-specific errors
	ErrInvalidPolicyType      = errors.New("invalid policy type")
	ErrInvalidPaymentFrequency = errors.New("invalid payment frequency")
	ErrInvalidSex             = errors.New("invalid sex specified")
	ErrInvalidDate            = errors.New("invalid date specified")
	ErrInvalidAge             = errors.New("invalid age specified")
	ErrInvalidCapital         = errors.New("invalid capital amount")
	ErrInvalidRate            = errors.New("invalid rate specified")
	
	// Mortality table errors
	ErrTableNotEffective       = errors.New("mortality table not effective for policy date")
	ErrAgeOutOfRange          = errors.New("age out of range for mortality table")
	ErrInvalidTableType       = errors.New("invalid table type")
	ErrTableIncompatible      = errors.New("mortality table incompatible with policy")
	
	// VTD errors
	ErrVTDDataNotAvailable    = errors.New("VTD data not available for requested date")
	ErrPeriodOutOfRange       = errors.New("period out of range for VTD vector")
	ErrInvalidYear            = errors.New("invalid year for VTD data")
	ErrInvalidMonth           = errors.New("invalid month for VTD data")
	
	// Reserve calculation errors
	ErrDiscountRateNotFound    = errors.New("discount rate not found")
	ErrInsufficientFlows      = errors.New("insufficient cash flows for calculation")
	ErrInvalidDiscountMethod   = errors.New("invalid discount method specified")
	ErrCalculationOverflow    = errors.New("calculation overflow")
	ErrNegativeReserve        = errors.New("negative reserve calculated")
	
	// Configuration errors
	ErrConfigNotFound         = errors.New("configuration file not found")
	ErrInvalidConfig          = errors.New("invalid configuration")
	ErrDatabaseConnection     = errors.New("database connection failed")
	ErrMigrationFailed       = errors.New("database migration failed")
	
	// Compliance errors
	ErrComplianceViolation   = errors.New("compliance violation detected")
	ErrAuditTrailMissing     = errors.New("audit trail missing")
	ErrRegulatoryRequirement  = errors.New("regulatory requirement not met")
)