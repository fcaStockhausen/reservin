package models

import "github.com/shopspring/decimal"

// FactorMejoramiento is one mortality improvement factor AAx for a table, age,
// and improvement year (Circular 2332 / Nota Técnica N°9, ecuación 2).
type FactorMejoramiento struct {
	ID             int
	NombreEstandar string
	Edad           int
	Año            int
	FactorAA       decimal.Decimal
}
