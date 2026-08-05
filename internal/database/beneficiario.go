package database

import (
	"database/sql"
	"fmt"

	"reservas/internal/models"
)

// BeneficiarioRepository handles beneficiario database operations.
type BeneficiarioRepository struct {
	db *sql.DB
}

// NewBeneficiarioRepository creates a new beneficiario repository.
func NewBeneficiarioRepository(db *sql.DB) *BeneficiarioRepository {
	return &BeneficiarioRepository{db: db}
}

// Insert inserts a single beneficiario and returns the new ID.
func (r *BeneficiarioRepository) Insert(b models.Beneficiario) (int, error) {
	query := `
	INSERT INTO beneficiario (
		poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
		tabla_asignada, porcentaje_renta, estado
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion, b.FechaNacimiento,
		b.TablaAsignada, b.PorcentajeRenta, b.Estado,
	)
	if err != nil {
		return 0, fmt.Errorf("insert beneficiario: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

// BatchInsert inserts multiple beneficiarios in a single transaction.
func (r *BeneficiarioRepository) BatchInsert(members []models.Beneficiario) error {
	if len(members) == 0 {
		return nil
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
	INSERT INTO beneficiario (
		poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
		tabla_asignada, porcentaje_renta, estado
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range members {
		_, err := stmt.Exec(
			b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion, b.FechaNacimiento,
			b.TablaAsignada, b.PorcentajeRenta, b.Estado,
		)
		if err != nil {
			return fmt.Errorf("batch insert beneficiario: %w", err)
		}
	}

	return tx.Commit()
}

// GetByID retrieves a beneficiario by ID.
func (r *BeneficiarioRepository) GetByID(id int) (*models.Beneficiario, error) {
	query := `
	SELECT id, poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
	       tabla_asignada, porcentaje_renta, estado, created_at
	FROM beneficiario
	WHERE id = ?`

	b := &models.Beneficiario{}
	err := r.db.QueryRow(query, id).Scan(
		&b.ID, &b.PolizaID, &b.Rol, &b.Sexo, &b.EdadContratacion, &b.FechaNacimiento,
		&b.TablaAsignada, &b.PorcentajeRenta, &b.Estado, &b.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// GetByPoliza retrieves all family group members for a policy.
func (r *BeneficiarioRepository) GetByPoliza(polizaID int) ([]models.Beneficiario, error) {
	query := `
	SELECT id, poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
	       tabla_asignada, porcentaje_renta, estado, created_at
	FROM beneficiario
	WHERE poliza_id = ?
	ORDER BY CASE rol
	    WHEN 'CAUSANTE' THEN 0
	    WHEN 'CONYUGE' THEN 1
	    WHEN 'HIJO' THEN 2
	    ELSE 3
	END, edad_contratacion DESC`

	rows, err := r.db.Query(query, polizaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Beneficiario
	for rows.Next() {
		var b models.Beneficiario
		err := rows.Scan(
			&b.ID, &b.PolizaID, &b.Rol, &b.Sexo, &b.EdadContratacion, &b.FechaNacimiento,
			&b.TablaAsignada, &b.PorcentajeRenta, &b.Estado, &b.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		members = append(members, b)
	}

	return members, rows.Err()
}

// GetGrupoFamiliar assembles the family group structure for a policy.
func (r *BeneficiarioRepository) GetGrupoFamiliar(polizaID int) (*models.GrupoFamiliar, error) {
	members, err := r.GetByPoliza(polizaID)
	if err != nil {
		return nil, err
	}

	gf := &models.GrupoFamiliar{PolizaID: polizaID}

	for i := range members {
		if members[i].Rol == models.RolCausante {
			gf.Causante = &members[i]
		} else {
			gf.Beneficiarios = append(gf.Beneficiarios, &members[i])
		}
	}

	return gf, nil
}

// Update updates an existing beneficiario.
func (r *BeneficiarioRepository) Update(b models.Beneficiario) error {
	query := `
	UPDATE beneficiario SET
		poliza_id = ?, rol = ?, sexo = ?, edad_contratacion = ?,
		fecha_nacimiento = ?, tabla_asignada = ?, porcentaje_renta = ?, estado = ?
	WHERE id = ?`

	_, err := r.db.Exec(query,
		b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion,
		b.FechaNacimiento, b.TablaAsignada, b.PorcentajeRenta, b.Estado,
		b.ID,
	)
	return err
}

// Delete removes a beneficiario permanently.
func (r *BeneficiarioRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM beneficiario WHERE id = ?", id)
	return err
}

// DeleteByPoliza removes all family group members for a policy.
func (r *BeneficiarioRepository) DeleteByPoliza(polizaID int) error {
	_, err := r.db.Exec("DELETE FROM beneficiario WHERE poliza_id = ?", polizaID)
	return err
}

// GetStatistics returns counts about beneficiarios.
func (r *BeneficiarioRepository) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var total int
	err := r.db.QueryRow("SELECT COUNT(*) FROM beneficiario").Scan(&total)
	if err != nil {
		return nil, err
	}
	stats["total"] = total

	var polizasConGrupo int
	err = r.db.QueryRow("SELECT COUNT(DISTINCT poliza_id) FROM beneficiario").Scan(&polizasConGrupo)
	if err != nil {
		return nil, err
	}
	stats["polizas_con_grupo"] = polizasConGrupo

	rows, err := r.db.Query(`
		SELECT rol, COUNT(*) FROM beneficiario GROUP BY rol ORDER BY rol`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byRol := make(map[string]int)
	for rows.Next() {
		var rol string
		var count int
		if err := rows.Scan(&rol, &count); err != nil {
			return nil, err
		}
		byRol[rol] = count
	}
	stats["by_rol"] = byRol

	return stats, rows.Err()
}
