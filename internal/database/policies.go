package database

import (
	"database/sql"
	"time"

	"reservas/internal/models"
	"github.com/shopspring/decimal"
)

// PolicyRepository handles policy database operations
type PolicyRepository struct {
	db *sql.DB
}

// NewPolicyRepository creates a new policy repository
func NewPolicyRepository(db *sql.DB) *PolicyRepository {
	return &PolicyRepository{db: db}
}

// CreateTable creates the policies table
func (r *PolicyRepository) CreateTable() error {
	sql := `
	CREATE TABLE IF NOT EXISTS poliza (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		numero_poliza VARCHAR(50) UNIQUE NOT NULL,
		tipo_renta VARCHAR(20) NOT NULL, -- 'VITALICIA', 'TEMPORARIA', 'DIFERIDA'
		fecha_inicio DATE NOT NULL,
		fecha_fin DATE,
		edad_contratante INTEGER NOT NULL,
		sexo_beneficiario CHAR(1) NOT NULL, -- 'H', 'M'
		capital_asegurado DECIMAL(15,2) NOT NULL,
		forma_pago VARCHAR(10), -- 'MENSUAL', 'TRIMESTRAL', 'ANUAL'
		tasa_descuento DECIMAL(8,6) NOT NULL, -- "bautizo" rate (min TM, TC)
		tasa_tm DECIMAL(8,6) NOT NULL, -- Tasa venta
		tasa_tc DECIMAL(8,6) NOT NULL, -- Tasa costo
		estado VARCHAR(10) DEFAULT 'ACTIVA', -- 'ACTIVA', 'VENCIDA', 'CANCELADA'
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err := r.db.Exec(sql)
	return err
}

// Insert inserts a new policy
func (r *PolicyRepository) Insert(policy models.Policy) (int, error) {
	sql := `
	INSERT INTO poliza (
		numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		edad_contratante, sexo_beneficiario, capital_asegurado,
		forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(sql,
		policy.NumeroPoliza, policy.TipoRenta, policy.FechaInicio, policy.FechaFin,
		policy.EdadContratante, policy.SexoBeneficiario, policy.CapitalAsegurado,
		policy.FormaPago, policy.TasaDescuento, policy.TasaTM, policy.TasaTC, policy.Estado,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// GetByID retrieves a policy by ID
func (r *PolicyRepository) GetByID(id int) (*models.Policy, error) {
	sql := `
	SELECT id, numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		   edad_contratante, sexo_beneficiario, capital_asegurado,
		   forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado, created_at
	FROM poliza
	WHERE id = ?`

	policy := &models.Policy{}
	err := r.db.QueryRow(sql, id).Scan(
		&policy.ID, &policy.NumeroPoliza, &policy.TipoRenta, &policy.FechaInicio, &policy.FechaFin,
		&policy.EdadContratante, &policy.SexoBeneficiario, &policy.CapitalAsegurado,
		&policy.FormaPago, &policy.TasaDescuento, &policy.TasaTM, &policy.TasaTC, &policy.Estado, &policy.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return policy, nil
}

// GetByNumeroPoliza retrieves a policy by policy number
func (r *PolicyRepository) GetByNumeroPoliza(numeroPoliza string) (*models.Policy, error) {
	sql := `
	SELECT id, numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		   edad_contratante, sexo_beneficiario, capital_asegurado,
		   forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado, created_at
	FROM poliza
	WHERE numero_poliza = ?`

	policy := &models.Policy{}
	err := r.db.QueryRow(sql, numeroPoliza).Scan(
		&policy.ID, &policy.NumeroPoliza, &policy.TipoRenta, &policy.FechaInicio, &policy.FechaFin,
		&policy.EdadContratante, &policy.SexoBeneficiario, &policy.CapitalAsegurado,
		&policy.FormaPago, &policy.TasaDescuento, &policy.TasaTM, &policy.TasaTC, &policy.Estado, &policy.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return policy, nil
}

// GetAll retrieves all policies with optional filtering
func (r *PolicyRepository) GetAll(limit, offset int, estado string) ([]models.Policy, error) {
	sql := `
	SELECT id, numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		   edad_contratante, sexo_beneficiario, capital_asegurado,
		   forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado, created_at
	FROM poliza`

	var args []interface{}
	
	// Add WHERE clause if status specified
	if estado != "" {
		sql += " WHERE estado = ?"
		args = append(args, estado)
	}

	// Add ORDER BY and LIMIT
	sql += " ORDER BY fecha_inicio DESC"
	if limit > 0 {
		sql += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		sql += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := r.db.Query(sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		policy := models.Policy{}
		err := rows.Scan(
			&policy.ID, &policy.NumeroPoliza, &policy.TipoRenta, &policy.FechaInicio, &policy.FechaFin,
			&policy.EdadContratante, &policy.SexoBeneficiario, &policy.CapitalAsegurado,
			&policy.FormaPago, &policy.TasaDescuento, &policy.TasaTM, &policy.TasaTC, &policy.Estado, &policy.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

// GetActivePolicies retrieves all active policies
func (r *PolicyRepository) GetActivePolicies() ([]models.Policy, error) {
	return r.GetAll(0, 0, "ACTIVA")
}

// Update updates an existing policy
func (r *PolicyRepository) Update(policy models.Policy) error {
	sql := `
	UPDATE poliza
	SET numero_poliza = ?, tipo_renta = ?, fecha_inicio = ?, fecha_fin = ?,
		edad_contratante = ?, sexo_beneficiario = ?, capital_asegurado = ?,
		forma_pago = ?, tasa_descuento = ?, tasa_tm = ?, tasa_tc = ?, estado = ?
	WHERE id = ?`

	_, err := r.db.Exec(sql,
		policy.NumeroPoliza, policy.TipoRenta, policy.FechaInicio, policy.FechaFin,
		policy.EdadContratante, policy.SexoBeneficiario, policy.CapitalAsegurado,
		policy.FormaPago, policy.TasaDescuento, policy.TasaTM, policy.TasaTC, policy.Estado,
		policy.ID,
	)
	return err
}

// Delete soft deletes a policy by changing status
func (r *PolicyRepository) Delete(id int) error {
	sql := `UPDATE poliza SET estado = 'CANCELADA' WHERE id = ?`
	_, err := r.db.Exec(sql, id)
	return err
}

// GetPoliciesByDateRange retrieves policies within a date range
func (r *PolicyRepository) GetPoliciesByDateRange(startDate, endDate time.Time) ([]models.Policy, error) {
	sql := `
	SELECT id, numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		   edad_contratante, sexo_beneficiario, capital_asegurado,
		   forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado, created_at
	FROM poliza
	WHERE fecha_inicio BETWEEN ? AND ?
	ORDER BY fecha_inicio ASC`

	rows, err := r.db.Query(sql, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		policy := models.Policy{}
		err := rows.Scan(
			&policy.ID, &policy.NumeroPoliza, &policy.TipoRenta, &policy.FechaInicio, &policy.FechaFin,
			&policy.EdadContratante, &policy.SexoBeneficiario, &policy.CapitalAsegurado,
			&policy.FormaPago, &policy.TasaDescuento, &policy.TasaTM, &policy.TasaTC, &policy.Estado, &policy.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}

	return policies, nil
}

// GetStatistics returns policy statistics
func (r *PolicyRepository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total policies
	var totalPolicies int
	err := r.db.QueryRow("SELECT COUNT(*) FROM poliza").Scan(&totalPolicies)
	if err != nil {
		return nil, err
	}
	stats["total_policies"] = totalPolicies

	// Policies by status
	rows, err := r.db.Query(`
		SELECT estado, COUNT(*) 
		FROM poliza 
		GROUP BY estado
		ORDER BY estado ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policiesByStatus := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		policiesByStatus[status] = count
	}
	stats["policies_by_status"] = policiesByStatus

	// Policies by type
	rows, err = r.db.Query(`
		SELECT tipo_renta, COUNT(*) 
		FROM poliza 
		GROUP BY tipo_renta
		ORDER BY tipo_renta ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policiesByType := make(map[string]int)
	for rows.Next() {
		var tipo string
		var count int
		if err := rows.Scan(&tipo, &count); err != nil {
			return nil, err
		}
		policiesByType[tipo] = count
	}
	stats["policies_by_type"] = policiesByType

	// Total insured capital
	var totalCapital float64
	err = r.db.QueryRow("SELECT SUM(capital_asegurado) FROM poliza WHERE estado = 'ACTIVA'").Scan(&totalCapital)
	if err != nil {
		return nil, err
	}
	stats["total_capital_asegurado"] = totalCapital

	return stats, nil
}

// CreateIndexes creates performance indexes for policies table
func (r *PolicyRepository) CreateIndexes() error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_poliza_estado ON poliza(estado);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_fecha_inicio ON poliza(fecha_inicio);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_numero ON poliza(numero_poliza);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_tipo_renta ON poliza(tipo_renta);`,
		`CREATE INDEX IF NOT EXISTS idx_poliza_sexo_beneficiario ON poliza(sexo_beneficiario);`,
	}

	for _, index := range indexes {
		if _, err := r.db.Exec(index); err != nil {
			return err
		}
	}

	return nil
}

// ValidatePolicy validates policy data before insertion/update
func (r *PolicyRepository) ValidatePolicy(policy models.Policy) error {
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
		return models.ErrInvalidPolicyType
	}

	// Validate sex
	if policy.SexoBeneficiario != "H" && policy.SexoBeneficiario != "M" {
		return models.ErrInvalidSex
	}

	// Validate age
	if policy.EdadContratante < 18 || policy.EdadContratante > 120 {
		return models.ErrInvalidAge
	}

	// Validate capital
	if policy.CapitalAsegurado.LessThanOrEqual(decimal.Zero) {
		return models.ErrInvalidCapital
	}

	// Validate rates
	if policy.TasaTM.LessThan(decimal.Zero) || policy.TasaTC.LessThan(decimal.Zero) {
		return models.ErrInvalidRate
	}

	// Validate dates
	if policy.FechaInicio.IsZero() || policy.FechaInicio.After(time.Now()) {
		return models.ErrInvalidDate
	}

	if policy.FechaFin != nil && policy.FechaFin.Before(policy.FechaInicio) {
		return models.ErrInvalidDate
	}

	return nil
}