package models

import (
	"testing"
	"time"
)

var (
	d2009 = time.Date(2009, 6, 1, 0, 0, 0, 0, time.UTC) // contrato pre-2012
	d2015 = time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC) // contrato post-2012
	d2024 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
)

func TestSelectBaseTable(t *testing.T) {
	vejez := string(TableTypeVejez)
	inval := string(TableTypeInvalidez)

	cases := []struct {
		name      string
		rol       BeneficiarioRol
		tipoTabla string
		sexo      string
		d         time.Time
		want      string
	}{
		{"causante H 2024", RolCausante, vejez, "M", d2024, "CB-H-2020"},
		{"causante M 2024", RolCausante, vejez, "F", d2024, "RV-M-2020"},
		{"causante H 2009", RolCausante, vejez, "M", d2009, "RV-H-2009"},
		{"causante M 2009", RolCausante, vejez, "F", d2009, "RV-M-2009"},
		{"causante H pre2005", RolCausante, vejez, "M", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "RV-H-2004"},
		{"sobreviviente M post2012", RolConyuge, vejez, "F", d2015, "B-M-2020"},
		{"sobreviviente H post2012", RolHijo, vejez, "M", d2015, "CB-H-2020"},
		{"sobreviviente M 2009", RolConyuge, vejez, "F", d2009, "B-M-2006"},
		{"causante H invalido 2024", RolCausante, inval, "M", d2024, "MI-H-2020"},
		{"causante M invalido 2024", RolCausante, inval, "F", d2024, "MI-M-2020"},
		{"causante invalido 2009", RolCausante, inval, "H", d2009, "MI-H-2006"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := SelectBaseTable(c.rol, c.tipoTabla, c.sexo, c.d)
			if got != c.want {
				t.Errorf("SelectBaseTable(%s,%s,%s,%s) = %s, want %s",
					c.rol, c.tipoTabla, c.sexo, c.d.Format("2006"), got, c.want)
			}
		})
	}
}

func TestSelectContemporaneaTable(t *testing.T) {
	vejez := string(TableTypeVejez)
	if got := SelectContemporaneaTable(RolCausante, vejez, "F", d2024); got != "RV-M-2020" {
		t.Errorf("contemp 2024 = %s, want RV-M-2020", got)
	}
	if got := SelectContemporaneaTable(RolConyuge, vejez, "F", time.Date(2015, 3, 1, 0, 0, 0, 0, time.UTC)); got != "B-M-2014" {
		t.Errorf("contemp 2015 = %s, want B-M-2014", got)
	}
	if got := SelectContemporaneaTable(RolCausante, vejez, "F", d2009); got != "RV-M-2009" {
		t.Errorf("contemp 2009 = %s, want RV-M-2009", got)
	}
}
