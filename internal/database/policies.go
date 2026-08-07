package database

import (
	"database/sql"
	"time"

	"github.com/fcaStockhausen/reservin/internal/models"
	"github.com/shopspring/decimal"
)

type PolicyRepository struct {
	db *sql.DB
}

func NewPolicyRepository(db *sql.DB) *PolicyRepository {
	return &PolicyRepository{db: db}
}

const policyColumns = `id, numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
	edad_contratante, sexo_beneficiario, capital_asegurado,
	forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado,
	tipo_pension, modalidad_renta, vigencia_pension,
	periodo_aumento, porcentaje_aumento, created_at`

func scanPolicy(p *models.Policy) []interface{} {
	return []interface{}{
		&p.ID, &p.NumeroPoliza, &p.TipoRenta, &p.FechaInicio, &p.FechaFin,
		&p.EdadContratante, &p.SexoBeneficiario, &p.CapitalAsegurado,
		&p.FormaPago, &p.TasaDescuento, &p.TasaTM, &p.TasaTC, &p.Estado,
		&p.TipoPension, &p.ModalidadRenta, &p.VigenciaPension,
		&p.PeriodoAumento, &p.PorcentajeAumento, &p.CreatedAt,
	}
}

func (r *PolicyRepository) Insert(policy models.Policy) (int, error) {
	query := `
	INSERT INTO poliza (
		numero_poliza, tipo_renta, fecha_inicio, fecha_fin,
		edad_contratante, sexo_beneficiario, capital_asegurado,
		forma_pago, tasa_descuento, tasa_tm, tasa_tc, estado,
		tipo_pension, modalidad_renta, vigencia_pension,
		periodo_aumento, porcentaje_aumento
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		policy.NumeroPoliza, policy.TipoRenta, policy.FechaInicio, policy.FechaFin,
		policy.EdadContratante, policy.SexoBeneficiario, policy.CapitalAsegurado,
		policy.FormaPago, policy.TasaDescuento, policy.TasaTM, policy.TasaTC, policy.Estado,
		policy.TipoPension, policy.ModalidadRenta, policy.VigenciaPension,
		policy.PeriodoAumento, policy.PorcentajeAumento,
	)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (r *PolicyRepository) GetByID(id int) (*models.Policy, error) {
	query := `SELECT ` + policyColumns + ` FROM poliza WHERE id = ?`

	policy := &models.Policy{}
	err := r.db.QueryRow(query, id).Scan(scanPolicy(policy)...)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (r *PolicyRepository) GetByNumeroPoliza(numeroPoliza string) (*models.Policy, error) {
	query := `SELECT ` + policyColumns + ` FROM poliza WHERE numero_poliza = ?`

	policy := &models.Policy{}
	err := r.db.QueryRow(query, numeroPoliza).Scan(scanPolicy(policy)...)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (r *PolicyRepository) GetAll(limit, offset int, estado string) ([]models.Policy, error) {
	query := `SELECT ` + policyColumns + ` FROM poliza`

	var args []interface{}
	if estado != "" {
		query += " WHERE estado = ?"
		args = append(args, estado)
	}

	query += " ORDER BY fecha_inicio DESC"
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		query += " OFFSET ?"
		args = append(args, offset)
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		if err := rows.Scan(scanPolicy(&p)...); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}

	return policies, nil
}

func (r *PolicyRepository) GetActivePolicies() ([]models.Policy, error) {
	return r.GetAll(0, 0, "ACTIVA")
}

func (r *PolicyRepository) Update(policy models.Policy) error {
	query := `
	UPDATE poliza SET
		numero_poliza = ?, tipo_renta = ?, fecha_inicio = ?, fecha_fin = ?,
		edad_contratante = ?, sexo_beneficiario = ?, capital_asegurado = ?,
		forma_pago = ?, tasa_descuento = ?, tasa_tm = ?, tasa_tc = ?, estado = ?,
		tipo_pension = ?, modalidad_renta = ?, vigencia_pension = ?,
		periodo_aumento = ?, porcentaje_aumento = ?
	WHERE id = ?`

	_, err := r.db.Exec(query,
		policy.NumeroPoliza, policy.TipoRenta, policy.FechaInicio, policy.FechaFin,
		policy.EdadContratante, policy.SexoBeneficiario, policy.CapitalAsegurado,
		policy.FormaPago, policy.TasaDescuento, policy.TasaTM, policy.TasaTC, policy.Estado,
		policy.TipoPension, policy.ModalidadRenta, policy.VigenciaPension,
		policy.PeriodoAumento, policy.PorcentajeAumento,
		policy.ID,
	)
	return err
}

func (r *PolicyRepository) Delete(id int) error {
	_, err := r.db.Exec("UPDATE poliza SET estado = 'CANCELADA' WHERE id = ?", id)
	return err
}

func (r *PolicyRepository) GetPoliciesByDateRange(startDate, endDate time.Time) ([]models.Policy, error) {
	query := `SELECT ` + policyColumns + ` FROM poliza WHERE fecha_inicio BETWEEN ? AND ? ORDER BY fecha_inicio ASC`

	rows, err := r.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		if err := rows.Scan(scanPolicy(&p)...); err != nil {
			return nil, err
		}
		policies = append(policies, p)
	}

	return policies, nil
}

func (r *PolicyRepository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalPolicies int
	err := r.db.QueryRow("SELECT COUNT(*) FROM poliza").Scan(&totalPolicies)
	if err != nil {
		return nil, err
	}
	stats["total_policies"] = totalPolicies

	rows, err := r.db.Query(`SELECT estado, COUNT(*) FROM poliza GROUP BY estado ORDER BY estado`)
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

	rows2, err := r.db.Query(`SELECT tipo_renta, COUNT(*) FROM poliza GROUP BY tipo_renta ORDER BY tipo_renta`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	policiesByType := make(map[string]int)
	for rows2.Next() {
		var tipor string
		var count int
		if err := rows2.Scan(&tipor, &count); err != nil {
			return nil, err
		}
		policiesByType[tipor] = count
	}
	stats["policies_by_type"] = policiesByType

	var totalCapital decimal.NullDecimal
	err = r.db.QueryRow("SELECT SUM(capital_asegurado) FROM poliza WHERE estado = 'ACTIVA'").Scan(&totalCapital)
	if err == nil && totalCapital.Valid {
		stats["total_capital_asegurado"] = totalCapital.Decimal
	}

	return stats, nil
}
