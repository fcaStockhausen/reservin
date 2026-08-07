package database

import (
	"database/sql"
	"fmt"

	"github.com/fcaStockhausen/reservin/internal/models"
)

type BeneficiarioRepository struct {
	db *sql.DB
}

func NewBeneficiarioRepository(db *sql.DB) *BeneficiarioRepository {
	return &BeneficiarioRepository{db: db}
}

const beneficiarioColumns = `id, poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
	tabla_asignada, porcentaje_renta, estado,
	tipo_beneficiario_c1194, derecho_pension, requisito_pension,
	derecho_acrecer, situacion_invalidez, condicion,
	matrimonio_anios, hijos_comunes, fin_derecho_edad, created_at`

func scanBeneficiario(b *models.Beneficiario) []interface{} {
	return []interface{}{
		&b.ID, &b.PolizaID, &b.Rol, &b.Sexo, &b.EdadContratacion, &b.FechaNacimiento,
		&b.TablaAsignada, &b.PorcentajeRenta, &b.Estado,
		&b.TipoBeneficiarioC1194, &b.DerechoPension, &b.RequisitoPension,
		&b.DerechoAcrecer, &b.SituacionInvalidez, &b.Condicion,
		&b.MatrimonioAnios, &b.HijosComunes, &b.FinDerechoEdad, &b.CreatedAt,
	}
}

func (r *BeneficiarioRepository) Insert(b models.Beneficiario) (int, error) {
	query := `
	INSERT INTO beneficiario (
		poliza_id, rol, sexo, edad_contratacion, fecha_nacimiento,
		tabla_asignada, porcentaje_renta, estado,
		tipo_beneficiario_c1194, derecho_pension, requisito_pension,
		derecho_acrecer, situacion_invalidez, condicion,
		matrimonio_anios, hijos_comunes, fin_derecho_edad
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.Exec(query,
		b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion, b.FechaNacimiento,
		b.TablaAsignada, b.PorcentajeRenta, b.Estado,
		b.TipoBeneficiarioC1194, b.DerechoPension, b.RequisitoPension,
		b.DerechoAcrecer, b.SituacionInvalidez, b.Condicion,
		b.MatrimonioAnios, b.HijosComunes, b.FinDerechoEdad,
	)
	if err != nil {
		return 0, fmt.Errorf("insert beneficiario: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

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
		tabla_asignada, porcentaje_renta, estado,
		tipo_beneficiario_c1194, derecho_pension, requisito_pension,
		derecho_acrecer, situacion_invalidez, condicion,
		matrimonio_anios, hijos_comunes, fin_derecho_edad
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range members {
		_, err := stmt.Exec(
			b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion, b.FechaNacimiento,
			b.TablaAsignada, b.PorcentajeRenta, b.Estado,
			b.TipoBeneficiarioC1194, b.DerechoPension, b.RequisitoPension,
			b.DerechoAcrecer, b.SituacionInvalidez, b.Condicion,
			b.MatrimonioAnios, b.HijosComunes, b.FinDerechoEdad,
		)
		if err != nil {
			return fmt.Errorf("batch insert beneficiario: %w", err)
		}
	}

	return tx.Commit()
}

func (r *BeneficiarioRepository) GetByID(id int) (*models.Beneficiario, error) {
	query := `SELECT ` + beneficiarioColumns + ` FROM beneficiario WHERE id = ?`

	b := &models.Beneficiario{}
	err := r.db.QueryRow(query, id).Scan(scanBeneficiario(b)...)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *BeneficiarioRepository) GetByPoliza(polizaID int) ([]models.Beneficiario, error) {
	query := `SELECT ` + beneficiarioColumns + ` FROM beneficiario WHERE poliza_id = ?
	ORDER BY CASE rol
	    WHEN 'CAUSANTE' THEN 0
	    WHEN 'CONYUGE' THEN 1
	    WHEN 'CONVIVIENTE_CIVIL' THEN 2
	    WHEN 'HIJO' THEN 3
	    ELSE 4
	END, edad_contratacion DESC`

	rows, err := r.db.Query(query, polizaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []models.Beneficiario
	for rows.Next() {
		var b models.Beneficiario
		if err := rows.Scan(scanBeneficiario(&b)...); err != nil {
			return nil, err
		}
		members = append(members, b)
	}

	return members, rows.Err()
}

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

func (r *BeneficiarioRepository) Update(b models.Beneficiario) error {
	query := `
	UPDATE beneficiario SET
		poliza_id = ?, rol = ?, sexo = ?, edad_contratacion = ?,
		fecha_nacimiento = ?, tabla_asignada = ?, porcentaje_renta = ?, estado = ?,
		tipo_beneficiario_c1194 = ?, derecho_pension = ?, requisito_pension = ?,
		derecho_acrecer = ?, situacion_invalidez = ?, condicion = ?,
		matrimonio_anios = ?, hijos_comunes = ?, fin_derecho_edad = ?
	WHERE id = ?`

	_, err := r.db.Exec(query,
		b.PolizaID, b.Rol, b.Sexo, b.EdadContratacion,
		b.FechaNacimiento, b.TablaAsignada, b.PorcentajeRenta, b.Estado,
		b.TipoBeneficiarioC1194, b.DerechoPension, b.RequisitoPension,
		b.DerechoAcrecer, b.SituacionInvalidez, b.Condicion,
		b.MatrimonioAnios, b.HijosComunes, b.FinDerechoEdad,
		b.ID,
	)
	return err
}

func (r *BeneficiarioRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM beneficiario WHERE id = ?", id)
	return err
}

func (r *BeneficiarioRepository) DeleteByPoliza(polizaID int) error {
	_, err := r.db.Exec("DELETE FROM beneficiario WHERE poliza_id = ?", polizaID)
	return err
}

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
