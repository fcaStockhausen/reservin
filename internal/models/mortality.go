package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// MortalityTable represents a mortality table (tabla de mortalidad)
type MortalityTable struct {
	ID             int             `json:"id" db:"id"`
	NombreEstandar string          `json:"nombre_estandar" db:"nombre_estandar"` // CMF standard: "CB-H-2020", "RV-M-2020", etc.
	NombreOriginal string          `json:"nombre_original" db:"nombre_original"` // Excel name: "CB-2020-HOMBRES", etc.
	Sexo           string          `json:"sexo" db:"sexo"`                       // 'H', 'M', 'A'
	TipoTabla      string          `json:"tipo_tabla" db:"tipo_tabla"`           // 'VEJEZ', 'INVALIDEZ', 'SOBREVIVENCIA'
	AñoTabla       int             `json:"año_tabla" db:"año_tabla"`             // 2004, 2006, 2009, 2014, 2020
	Edad           int             `json:"edad" db:"edad"`
	ProbMuerte     decimal.Decimal `json:"prob_muerte" db:"prob_muerte"` // qx value
	FactorAax      decimal.Decimal `json:"factor_aax" db:"factor_aax"`   // Factor Aax
	VigenciaInicio time.Time       `json:"vigencia_inicio" db:"vigencia_inicio"`
	VigenciaFin    *time.Time      `json:"vigencia_fin,omitempty" db:"vigencia_fin"`
	CreatedAt      time.Time       `json:"created_at" db:"created_at"`
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
	TableTypeVejez         TableType = "VEJEZ"
	TableTypeInvalidez     TableType = "INVALIDEZ"
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
	// The reserve table is anchored to the contract date (Circular N°2332
	// stratification), regardless of methodology label.
	tableName := SelectBaseTable(beneficiaryRoleForPolicy(policy), policy.TipoTabla, policy.SexoBeneficiario, policy.FechaInicio)
	return mts.GetTableForName(tableName)
}

// GetTableForName returns the first record for a given standard table name.
func (mts *MortalityTableSelector) GetTableForName(tableName string) (*MortalityTable, error) {
	for _, t := range mts.Tables {
		if t.NombreEstandar == tableName {
			return t, nil
		}
	}
	return nil, ErrMortalityTableNotFound
}

// beneficiaryRoleForPolicy maps a policy to the beneficiary role used for
// mortality table selection: a renta vitalicia causante is a rentista (RV);
// otherwise the family member is a survivor (B/CB) or inválido (MI).
func beneficiaryRoleForPolicy(policy Policy) BeneficiarioRol {
	if policy.TipoRenta == string(PolicyTypeVitalicia) {
		return RolCausante
	}
	if policy.TipoTabla == string(TableTypeInvalidez) {
		return RolCausante // inválido: category decides via tipoTabla
	}
	return RolConyuge // survivor branch for non-rentista policies
}

// selectCurrentTable selects tables for post-2012 policies (IFRS)
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
