package scenario

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Scenario defines a complete simulation case: a policy, an initial family
// group, and a timeline of events that mutate the group over time.
type Scenario struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Horizon     int    `yaml:"horizon"` // years to simulate

	Policy PolicyDef `yaml:"policy"`

	Causante MemberDef   `yaml:"causante"`
	Grupo    []MemberDef `yaml:"grupo_familiar"`

	Events []EventDef `yaml:"events"`
}

type PolicyDef struct {
	CapitalUF          float64   `yaml:"capital_uf"`
	FechaContrato      string    `yaml:"fecha_contratacion"` // YYYY-MM-DD, ancla tabla de estrato y tasa de bautizo
	TasaTM             float64   `yaml:"tasa_tm"`
	TasaTC             float64   `yaml:"tasa_tc"`         // tasa de bautizo (fija toda la vida)
	TipoPension        string    `yaml:"tipo_pension"`    // C1194 code
	ModalidadRenta     string    `yaml:"modalidad_renta"` // 1000/2xxx/3xxx/4xxx
	PeriodoGarantizado int       `yaml:"periodo_garantizado_meses"`
	PeriodoAumento     int       `yaml:"periodo_aumento_meses"`
	PorcentajeAumento  float64   `yaml:"porcentaje_aumento"`
	GradualidadAnios   int       `yaml:"gradualidad_anios"` // 0 = reconocimiento inmediato (default)
	CurvaDescalce      []float64 `yaml:"curva_descalce"`    // fracción del descalce reconocida por año post-transición
}

type MemberDef struct {
	Rol             string  `yaml:"rol"`              // CAUSANTE, CONYUGE, HIJO, etc.
	Sexo            string  `yaml:"sexo"`             // M, F (C1194)
	Edad            int     `yaml:"edad"`             // age at t=0 (0 = derive from fecha_nacimiento)
	FechaNacimiento string  `yaml:"fecha_nacimiento"` // YYYY-MM-DD, drives cohort mortality table
	TipoC1194       string  `yaml:"tipo_c1194"`       // C1194 codigo
	Condicion       string  `yaml:"condicion"`        // MENOR, ESTUDIANTE, INVALIDO
	MatrimonioAnios int     `yaml:"matrimonio_anios"`
	HijosComunes    int     `yaml:"hijos_comunes"`
	FinDerechoEdad  int     `yaml:"fin_derecho_edad"` // 0 = nil
	PctRenta        float64 `yaml:"pct_renta"`        // override (0 = legal)
}

// EventDef represents a mutation to the family group at a specific year.
type EventDef struct {
	Year int    `yaml:"year"` // t=0 is policy inception
	Type string `yaml:"type"` // see Event* constants

	// For ADD_MEMBER / REMOVE_MEMBER
	Member MemberDef `yaml:"member,omitempty"`

	// For KILL_MEMBER (by rol + edad at event time)
	TargetRol  string `yaml:"target_rol,omitempty"`
	TargetSexo string `yaml:"target_sexo,omitempty"`
}

const (
	EventAddMember     = "ADD_MEMBER"    // birth, new marriage
	EventRemoveMember  = "REMOVE_MEMBER" // divorce, child loses rights
	EventKillMember    = "KILL_MEMBER"   // death of a family member
	EventChangePct     = "CHANGE_PCT"    // contractual % change
	EventPeriodoGarEnd = "PERIODO_GARANTIZADO_END"
	EventAumentoStart  = "AUMENTO_TEMPORAL_START"
	EventAumentoEnd    = "AUMENTO_TEMPORAL_END"
)

// Load reads a scenario from a YAML file.
func Load(path string) (*Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseYAML(string(data))
}

func yamlUnmarshal(data []byte, s *Scenario) error {
	return yaml.Unmarshal(data, s)
}

// policyStartDate returns a fixed inception date for simulation.
func policyStartDate() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}
