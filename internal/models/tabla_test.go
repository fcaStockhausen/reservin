package models

import (
	"testing"
	"time"
)

var (
	d2000 = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC) // pre-2005: RV-85/B-85/MI-85
	d2006 = time.Date(2006, 6, 1, 0, 0, 0, 0, time.UTC) // 2005-2008: RV-2004/B-85/MI-85
	d2009 = time.Date(2009, 6, 1, 0, 0, 0, 0, time.UTC) // 2008-2010: RV-2004/B-2006/MI-2006
	d2013 = time.Date(2013, 6, 1, 0, 0, 0, 0, time.UTC) // 2010-2016: RV-2009/B-2006/MI-2006
	d2015 = time.Date(2015, 6, 1, 0, 0, 0, 0, time.UTC) // 2010-2016: RV-2009/B-2006/MI-2006
	d2021 = time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC) // 2016-2023: CB-2014/RV-2014/MI-2014
	d2024 = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC) // 2023+: CB-2020/RV-2020/MI-2020
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
		// Período 2023+ (periodStart5)
		{"causante H 2024", RolCausante, vejez, "M", d2024, "CB-H-2020"},
		{"causante M 2024", RolCausante, vejez, "F", d2024, "RV-M-2020"},
		{"sobreviviente H 2024", RolHijo, vejez, "M", d2024, "CB-H-2020"},
		{"sobreviviente M 2024", RolConyuge, vejez, "F", d2024, "B-M-2020"},
		{"invalido H 2024", RolCausante, inval, "M", d2024, "MI-H-2020"},
		{"invalido M 2024", RolCausante, inval, "F", d2024, "MI-M-2020"},

		// Período 2016-2023 (periodStart4): CB-2014/RV-2014, B-2014, MI-2014
		{"causante H 2021", RolCausante, vejez, "M", d2021, "CB-H-2014"},
		{"causante M 2021", RolCausante, vejez, "F", d2021, "RV-M-2014"},
		{"sobreviviente H 2021", RolHijo, vejez, "M", d2021, "CB-H-2014"},
		{"sobreviviente M 2021", RolConyuge, vejez, "F", d2021, "B-M-2014"},
		{"invalido H 2021", RolCausante, inval, "M", d2021, "MI-H-2014"},

		// Período 2010-2016 (periodStart3): RV-2009/B-2006, B-2006, MI-2006
		{"causante H 2015", RolCausante, vejez, "M", d2015, "RV-H-2009"},
		{"causante M 2015", RolCausante, vejez, "F", d2015, "RV-M-2009"},
		{"sobreviviente H 2015", RolHijo, vejez, "M", d2015, "B-H-2006"},
		{"sobreviviente M 2015", RolConyuge, vejez, "F", d2015, "B-M-2006"},
		{"sobreviviente H 2013", RolHijo, vejez, "M", d2013, "B-H-2006"},
		{"sobreviviente M 2013", RolConyuge, vejez, "F", d2013, "B-M-2006"},
		{"invalido H 2015", RolCausante, inval, "M", d2015, "MI-H-2006"},

		// Período 2008-2010 (periodStart2): RV-2004/B-2006, B-2006, MI-2006
		{"causante H 2009", RolCausante, vejez, "M", d2009, "RV-H-2004"},
		{"causante M 2009", RolCausante, vejez, "F", d2009, "RV-M-2004"},
		{"sobreviviente M 2009", RolConyuge, vejez, "F", d2009, "B-M-2006"},
		{"invalido H 2009", RolCausante, inval, "M", d2009, "MI-H-2006"},

		// Período 2005-2008 (periodStart1): RV-2004/B-85, B-85, MI-85
		{"causante H 2006", RolCausante, vejez, "M", d2006, "RV-H-2004"},
		{"causante M 2006", RolCausante, vejez, "F", d2006, "RV-M-2004"},
		{"sobreviviente M 2006", RolConyuge, vejez, "F", d2006, "B-M-1985"},
		{"sobreviviente H 2006", RolHijo, vejez, "M", d2006, "B-H-1985"},

		// Período pre-2005: RV-85/B-85/MI-85
		{"causante H pre2005", RolCausante, vejez, "M", d2000, "RV-H-1985"},
		{"causante M pre2005", RolCausante, vejez, "F", d2000, "RV-M-1985"},
		{"sobreviviente M pre2005", RolConyuge, vejez, "F", d2000, "B-M-1985"},
		{"sobreviviente H pre2005", RolHijo, vejez, "M", d2000, "B-H-1985"},
		{"invalido H pre2005", RolCausante, inval, "M", d2000, "B-H-1985"},
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
	if got := SelectContemporaneaTable(RolConyuge, vejez, "F", d2024); got != "B-M-2020" {
		t.Errorf("contemp 2024 conyuge = %s, want B-M-2020", got)
	}
	if got := SelectContemporaneaTable(RolCausante, vejez, "F", d2021); got != "RV-M-2014" {
		t.Errorf("contemp 2021 = %s, want RV-M-2014", got)
	}
	if got := SelectContemporaneaTable(RolConyuge, vejez, "F", d2015); got != "B-M-2006" {
		t.Errorf("contemp 2015 conyuge = %s, want B-M-2006", got)
	}
	if got := SelectContemporaneaTable(RolCausante, vejez, "F", d2009); got != "RV-M-2004" {
		t.Errorf("contemp 2009 = %s, want RV-M-2004", got)
	}
}
