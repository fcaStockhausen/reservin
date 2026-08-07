package database

import (
	"database/sql"
	"fmt"

	"github.com/fcaStockhausen/reservin/internal/models"
)

type ClausulaRepository struct {
	db *sql.DB
}

func NewClausulaRepository(db *sql.DB) *ClausulaRepository {
	return &ClausulaRepository{db: db}
}

func (r *ClausulaRepository) Insert(c models.Clausula) (int, error) {
	query := `
	INSERT INTO clausula (poliza_id, tipo, parametros, modalidad_renta_c1194)
	VALUES (?, ?, ?, ?)`

	result, err := r.db.Exec(query, c.PolizaID, c.Tipo, c.Parametros, c.ModalidadRentaC1194)
	if err != nil {
		return 0, fmt.Errorf("insert clausula: %w", err)
	}

	id, err := result.LastInsertId()
	return int(id), err
}

func (r *ClausulaRepository) GetByPoliza(polizaID int) ([]models.Clausula, error) {
	query := `SELECT id, poliza_id, tipo, parametros, modalidad_renta_c1194, created_at
		FROM clausula WHERE poliza_id = ? ORDER BY id`

	rows, err := r.db.Query(query, polizaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clausulas []models.Clausula
	for rows.Next() {
		var c models.Clausula
		if err := rows.Scan(&c.ID, &c.PolizaID, &c.Tipo, &c.Parametros, &c.ModalidadRentaC1194, &c.CreatedAt); err != nil {
			return nil, err
		}
		clausulas = append(clausulas, c)
	}

	return clausulas, rows.Err()
}

func (r *ClausulaRepository) Delete(id int) error {
	_, err := r.db.Exec("DELETE FROM clausula WHERE id = ?", id)
	return err
}
