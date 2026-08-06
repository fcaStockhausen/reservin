package models

import (
	"fmt"
	"strconv"
	"time"
)

// TipoClausula identifies the type of additional clause on a policy.
type TipoClausula string

const (
	ClausulaSinAdicionales       TipoClausula = "SIN_ADICIONALES"
	ClausulaAumentoTemporal      TipoClausula = "AUMENTO_TEMPORAL"
	ClausulaPeriodoGarantizado   TipoClausula = "PERIODO_GARANTIZADO"
	ClausulaAumentoPctSobrev     TipoClausula = "AUMENTO_PCT_SOBREVIVENCIA"
	ClausulaAumentoDiferido      TipoClausula = "AUMENTO_DIFERIDO_VITALICIO"
)

// Clausula represents an additional clause on a policy (CAD polizas).
type Clausula struct {
	ID                int       `json:"id" db:"id"`
	PolizaID          int       `json:"poliza_id" db:"poliza_id"`
	Tipo              TipoClausula `json:"tipo" db:"tipo"`
	Parametros        string    `json:"parametros,omitempty" db:"parametros"` // JSON
	ModalidadRentaC1194 string  `json:"modalidad_renta_c1194" db:"modalidad_renta_c1194"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
}

// ModalidadRenta encodes the C1194 campo 2.18 MODALIDAD-RENTA value.
// This single field captures all clause combinations:
//
//	1000  = sin adicionales
//	2xxx  = aumento temporal (xxx = meses de aumento temporal)
//	3xxx  = solo periodo garantizado (xxx = meses garantizados, xxx >= 001)
//	4xxx  = aumento de % sobrevivencia (xxx = meses garantizados si tambien tiene PG)
//
// When combining aumento temporal + periodo garantizado, the code is 2xxx
// where xxx = meses garantizados (0 if no PG).
func ModalidadRenta(tipo TipoClausula, mesesGarantizados int, mesesAumentoTemporal int) string {
	switch tipo {
	case ClausulaSinAdicionales:
		return "1000"

	case ClausulaAumentoTemporal:
		// 2xxx: xxx = meses garantizados (0 if no PG)
		return fmt.Sprintf("2%03d", mesesGarantizados)

	case ClausulaPeriodoGarantizado:
		// 3xxx: xxx >= 001
		if mesesGarantizados < 1 {
			mesesGarantizados = 1
		}
		return fmt.Sprintf("3%03d", mesesGarantizados)

	case ClausulaAumentoPctSobrev, ClausulaAumentoDiferido:
		// 4xxx: xxx = meses garantizados (0 if no PG)
		return fmt.Sprintf("4%03d", mesesGarantizados)

	default:
		return "1000"
	}
}

// ParseModalidadRenta decodes a C1194 MODALIDAD-RENTA code into its components.
func ParseModalidadRenta(codigo string) (tipo TipoClausula, mesesGarantizados int, err error) {
	if len(codigo) != 4 {
		return "", 0, fmt.Errorf("modalidad_renta invalida: %s", codigo)
	}

	prefix := codigo[0]
	suffix := codigo[1:]
	meses, _ := strconv.Atoi(suffix)

	switch prefix {
	case '1':
		return ClausulaSinAdicionales, 0, nil
	case '2':
		return ClausulaAumentoTemporal, meses, nil
	case '3':
		return ClausulaPeriodoGarantizado, meses, nil
	case '4':
		return ClausulaAumentoPctSobrev, meses, nil
	default:
		return "", 0, fmt.Errorf("modalidad_renta prefix invalido: %c", prefix)
	}
}

// TipoPensionC1194 codes from Anexo Tecnico campo 2.6.
const (
	TipoPensionSobrev528              = "01"
	TipoPensionInv528                 = "02"
	TipoPensionSobrevInv528           = "03"
	TipoPensionRVVejezJubilacion      = "04"
	TipoPensionRVVejezAnticipada      = "05"
	TipoPensionRVInvTotal             = "06"
	TipoPensionRVInvParcial           = "07"
	TipoPensionRVSobrevivencia        = "08"
	TipoPensionSobrevRVVejezJubilac   = "09"
	TipoPensionSobrevRVVejezAnticip   = "10"
	TipoPensionSobrevRVInvTotal       = "11"
	TipoPensionSobrevRVInvParcial     = "12"
	TipoPensionSobrevTraspasoCartera  = "13"
	TipoPensionInvTraspasoCartera     = "14"
	TipoPensionSobrevInvTraspaso      = "15"
)

// VigenciaPensionC1194 codes from Anexo Tecnico campo 2.8.
const (
	VigenciaEnPago           = "6"
	VigenciaGarantizadaDesig = "7" // Pagando a designados
	VigenciaDiferida         = "8"
	VigenciaExtinguida       = "9"
)
