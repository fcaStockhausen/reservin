package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// MortalityTable represents a mortality table (tabla de mortalidad)
type MortalityTable struct {
	ID             int              `json:"id" db:"id"`
	NombreEstandar  string           `json:"nombre_estandar" db:"nombre_estandar"` // CMF standard: "CB-H-2020", "RV-M-2020", etc.
	NombreOriginal  string           `json:"nombre_original" db:"nombre_original"` // Excel name: "CB-2020-HOMBRES", etc.
	Sexo           string           `json:"sexo" db:"sexo"`                     // 'H', 'M', 'A'
	TipoTabla      string           `json:"tipo_tabla" db:"tipo_tabla"`            // 'VEJEZ', 'INVALIDEZ', 'SOBREVIVENCIA'
	AñoTabla       int              `json:"año_tabla" db:"año_tabla"`              // 2004, 2006, 2009, 2014, 2020
	Edad           int              `json:"edad" db:"edad"`
	ProbMuerte     decimal.Decimal   `json:"prob_muerte" db:"prob_muerte"`           // qx value
	FactorAax      decimal.Decimal   `json:"factor_aax" db:"factor_aax"`            // Factor Aax
	VigenciaInicio time.Time        `json:"vigencia_inicio" db:"vigencia_inicio"`
	VigenciaFin    *time.Time       `json:"vigencia_fin,omitempty" db:"vigencia_fin"`
	CreatedAt      time.Time        `json:"created_at" db:"created_at"`
}

// GetSurvivalProbability calculates survival probability to age x
func (mt *MortalityTable) GetSurvivalProbability(targetAge int) decimal.Decimal {
	if targetAge < mt.Edad {
		return decimal.NewFromInt(1)
	}

	// For now, return 1 - qx (simplified survival probability)
	return decimal.NewFromInt(1).Sub(mt.ProbMuerte)
}

// TableType represents the type of mortality table
type TableType string

const (
	TableTypeVejez      TableType = "VEJEZ"
	TableTypeInvalidez   TableType = "INVALIDEZ"
	TableTypeSobrevivencia TableType = "SOBREVIVENCIA"
)

// MortalityTableSelector determines which table to use based on policy characteristics
type MortalityTableSelector struct {
	Tables map[string]*MortalityTable
}

// NewMortalityTableSelector creates a new selector with loaded tables
func NewMortalityTableSelector() *MortalityTableSelector {
	return &MortalityTableSelector{
		Tables: make(map[string]*MortalityTable),
	}
}

// SelectTable chooses appropriate mortality table based on policy
func (mts *MortalityTableSelector) SelectTable(policy Policy) (*MortalityTable, error) {
	methodology := policy.GetMethodology()
	
	switch methodology {
	case MethodologyIFRS:
		return mts.selectCurrentTable(policy)
	case MethodologyTransitional:
		return mts.selectTransitionalTable(policy)
	case MethodologyTraditional:
		return mts.selectLegacyTable(policy)
	default:
		return mts.selectCurrentTable(policy) // Default to current
	}
}

// selectCurrentTable selects tables for post-2012 policies (IFRS)
func (mts *MortalityTableSelector) selectCurrentTable(policy Policy) (*MortalityTable, error) {
	switch {
	// Rentas Vitalicias - Mujeres
	case policy.TipoRenta == string(PolicyTypeVitalicia) && policy.SexoBeneficiario == "M":
		return mts.Tables["RV-M-2020"], nil
	
	// Básica - Sobrevivencia (Beneficiario Mujeres)
	case policy.TipoRenta == string(PolicyTypeTemporal) && policy.SexoBeneficiario == "M":
		return mts.Tables["B-M-2020"], nil
	
	// Básica Chile - Hombres (Causante/Beneficiario same table)
	case policy.TipoRenta == string(PolicyTypeTemporal) && policy.SexoBeneficiario == "H":
		return mts.Tables["CB-H-2020"], nil
	
	// Invalidez - Hombres (Causante)
	case policy.TipoRenta == string(PolicyTypeTemporal) && policy.TipoTabla == string(TableTypeInvalidez) && policy.SexoBeneficiario == "H":
		return mts.Tables["MI-H-2020"], nil
	
	// Invalidez - Mujeres (Beneficiario)
	case policy.TipoRenta == string(PolicyTypeTemporal) && policy.TipoTabla == string(TableTypeInvalidez) && policy.SexoBeneficiario == "M":
		return mts.Tables["MI-M-2020"], nil
	
	// Default fallback
	default:
		if policy.SexoBeneficiario == "M" {
			return mts.Tables["RV-M-2020"], nil
		}
		return mts.Tables["CB-H-2020"], nil
	}
}

// selectTransitionalTable selects tables for 2015-2020 transitional period
func (mts *MortalityTableSelector) selectTransitionalTable(policy Policy) (*MortalityTable, error) {
	// Transitional period uses current table selection methodology
	// but with specific transition rules from NCG 374
	return mts.selectCurrentTable(policy)
}

// selectLegacyTable selects tables for pre-2012 traditional policies
func (mts *MortalityTableSelector) selectLegacyTable(policy Policy) (*MortalityTable, error) {
	// Pre-2012 policies may use legacy tables with gradual application
	switch {
	case policy.SexoBeneficiario == "M":
		// Prefer B-2006 for mujeres if available
		if table, ok := mts.Tables["B-2006"]; ok {
			return table, nil
		}
		// Fallback to current table
		return mts.Tables["B-M-2020"], nil
	
	case policy.TipoTabla == string(TableTypeInvalidez):
		// Prefer MI-2006 for disability if available
		if table, ok := mts.Tables["MI-2006"]; ok {
			return table, nil
		}
		// Fallback to current table
		if policy.SexoBeneficiario == "H" {
			return mts.Tables["MI-H-2020"], nil
		}
		return mts.Tables["MI-M-2020"], nil
	
	default:
		// Default to current basic table
		return mts.Tables["CB-H-2020"], nil
	}
}

// GetTableForAge returns the mortality table record for a specific age
func (mts *MortalityTableSelector) GetTableForAge(tableName string, age int) (*MortalityTable, error) {
	key := mts.getAgeKey(tableName, age)
	if table, ok := mts.Tables[key]; ok {
		return table, nil
	}
	return nil, ErrMortalityTableNotFound
}

// getAgeKey creates a lookup key for a specific table and age
func (mts *MortalityTableSelector) getAgeKey(tableName string, age int) string {
	return tableName + "_age_" + string(rune(age))
}

// LoadTables loads all mortality tables from database
func (mts *MortalityTableSelector) LoadTables(tables []MortalityTable) error {
	mts.Tables = make(map[string]*MortalityTable)
	
	for i := range tables {
		table := &tables[i]
		key := table.NombreEstandar + "_age_" + string(rune(table.Edad))
		mts.Tables[key] = table
	}
	
	return nil
}