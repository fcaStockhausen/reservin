package portfolio

import (
	"strings"
	"testing"
	"time"
)

// TestGenAssignsTablesByContractDate verifies that synthetic policies get the
// mortality table matching their contract-date stratum (Cuadro 4 Nota Técnica
// N°9), including the legacy pre-2005 era (RV-85/B-85).
func TestGenAssignsTablesByContractDate(t *testing.T) {
	policies := Generate(10000)

	counts := map[string]int{}
	pre2005 := 0
	for _, p := range policies {
		d := p.Policy.FechaInicio
		for _, m := range p.Members {
			tbl := m.TablaAsignada
			counts[tbl]++
			switch {
			case m.Rol == "CAUSANTE" && m.Sexo == "M":
				want := "RV-H-1985"
				switch {
				case d.Before(time.Date(2005, 2, 1, 0, 0, 0, 0, time.UTC)):
					want = "RV-H-1985"
				case d.Before(time.Date(2010, 7, 1, 0, 0, 0, 0, time.UTC)):
					want = "RV-H-2004"
				case d.Before(time.Date(2016, 7, 1, 0, 0, 0, 0, time.UTC)):
					want = "RV-H-2009"
				case d.Before(time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)):
					want = "CB-H-2014"
				default:
					want = "CB-H-2020"
				}
				if tbl != want {
					t.Fatalf("fecha %s causante H -> %s, want %s", d.Format("2006-01-02"), tbl, want)
				}
			case m.Rol == "CONYUGE" && m.Sexo == "F":
				want := "B-M-1985"
				switch {
				case d.Before(time.Date(2005, 2, 1, 0, 0, 0, 0, time.UTC)):
					want = "B-M-1985"
				case d.Before(time.Date(2008, 2, 1, 0, 0, 0, 0, time.UTC)):
					want = "B-M-1985"
				case d.Before(time.Date(2016, 7, 1, 0, 0, 0, 0, time.UTC)):
					want = "B-M-2006"
				case d.Before(time.Date(2023, 7, 1, 0, 0, 0, 0, time.UTC)):
					want = "B-M-2014"
				default:
					want = "B-M-2020"
				}
				if tbl != want {
					t.Fatalf("fecha %s conyuge F -> %s, want %s", d.Format("2006-01-02"), tbl, want)
				}
			}
			if d.Year() < 2005 {
				pre2005++
			}
		}
	}
	if pre2005 == 0 {
		t.Fatal("no pre-2005 policies generated; legacy stratum untested")
	}
	for tbl := range counts {
		if strings.HasSuffix(tbl, "1985") == false {
			continue
		}
		t.Logf("legacy table %s present (%d)", tbl, counts[tbl])
	}
}
