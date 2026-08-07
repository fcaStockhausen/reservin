package main

import (
	"fmt"
	"log"
	"sort"

	"github.com/shopspring/decimal"

	"github.com/fcaStockhausen/reservin/internal/calculator"
	"github.com/fcaStockhausen/reservin/internal/database"
	"github.com/fcaStockhausen/reservin/internal/loader"
	"github.com/fcaStockhausen/reservin/internal/models"
)

// vtdSensResult holds the aggregate reserve under a single VTD curve.
type vtdSensResult struct {
	date  string
	total float64
	mean  float64
}

// vtdSensitivity measures the error introduced by using only the latest VTD
// curve instead of the historical curve of each valuation period. It streams a
// fixed sample of RIS policies once, then re-computes the aggregate reserve
// under every VTD vector available in the database, reporting the dispersion.
func vtdSensitivity(db *database.DB, path string, sample int) {
	mortRepo := database.NewMortalityRepository(db.DB)
	vtdRepo := database.NewVTDRepository(db.DB)

	dates, err := vtdRepo.GetAllVectorDates()
	if err != nil || len(dates) == 0 {
		log.Fatalf("no VTD vectors in database: %v", err)
	}
	fmt.Printf("VTD disponibles: %d vectores (%s a %s)\n", len(dates), dates[0], dates[len(dates)-1])

	// Stream a fixed sample of policies once and pre-compute their reserve
	// inputs (policy + group + renta anual) so re-calculating under each VTD
	// only touches the projection, not the RIS parsing.
	fmt.Printf("Streaming %d pólizas de %s...\n", sample, path)
	ld := loader.NewRIS1194Loader(path)
	policies, errs := ld.Stream(sample)

	type caseT struct {
		pol        models.Policy
		grupo      *models.GrupoFamiliar
		rentaAnual decimal.Decimal
	}
	var cases []caseT
	processed := 0
	for {
		select {
		case p, ok := <-policies:
			if !ok {
				policies = nil
				break // breaks the select, not the for; falls to nil check
			}
			rb := buildGrupoFromRIS(p, true) // TM-2020: aísla el efecto del VTD
			rentaMensual := rb.rentaMensual
			if rentaMensual.LessThanOrEqual(decimal.Zero) {
				rentaMensual = decimal.NewFromFloat(12).Mul(rb.pol.CapitalAsegurado).Div(decimal.NewFromFloat(120))
			}
			cases = append(cases, caseT{
				pol:        rb.pol,
				grupo:      rb.grupo,
				rentaAnual: rentaMensual.Mul(decimal.NewFromInt(12)),
			})
			processed++
			if processed >= sample {
				policies = nil
			}
		case err, ok := <-errs:
			if ok && err != nil {
				log.Printf("stream error: %v", err)
			}
			errs = nil
		}
		if policies == nil && errs == nil {
			break
		}
	}
	if len(cases) == 0 {
		log.Fatal("no policies streamed")
	}
	fmt.Printf("Pólizas en el set de sensibilidad: %d\n", len(cases))

	// Compute the aggregate reserve under each VTD vector.
	var results []vtdSensResult
	for _, dateStr := range dates {
		var year, month int
		fmt.Sscanf(dateStr, "%04d-%02d", &year, &month)

		calc := calculator.NewReserveCalculator(mortRepo, vtdRepo)
		if err := calc.LoadVTDFor(year, month); err != nil {
			log.Printf("skip %s: %v", dateStr, err)
			continue
		}

		var total decimal.Decimal
		count := 0
		for _, c := range cases {
			if c.grupo.Causante == nil || c.grupo.Causante.TablaAsignada == "" {
				continue
			}
			res, err := calc.Calculate(c.pol, c.grupo, c.rentaAnual)
			if err != nil {
				continue
			}
			total = total.Add(res.TotalReserve)
			count++
		}
		if count > 0 {
			results = append(results, vtdSensResult{
				date:  dateStr,
				total: total.InexactFloat64(),
				mean:  total.Div(decimal.NewFromInt(int64(count))).InexactFloat64(),
			})
		}
	}

	printVTDSensReport(results)
}

func printVTDSensReport(results []vtdSensResult) {
	if len(results) == 0 {
		fmt.Println("sin resultados")
		return
	}

	sort.Slice(results, func(i, j int) bool { return results[i].date < results[j].date })

	var totals []float64
	for _, r := range results {
		totals = append(totals, r.total)
	}
	minV, maxV := totals[0], totals[0]
	for _, v := range totals {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	last := results[len(results)-1].total

	fmt.Printf("\n=== ERROR POR VTD (sensibilidad a la curva de descuento) ===\n")
	fmt.Printf("Reserva total del set bajo cada VTD:\n")
	for _, r := range results {
		rel := (r.total/last - 1) * 100
		fmt.Printf("  VTD %s: %12.2f UF  (%+.2f%% vs último)\n", r.date, r.total, rel)
	}
	fmt.Printf("\nRango de la reserva total: %.2f UF a %.2f UF\n", minV, maxV)
	fmt.Printf("Spread (max-min):        %.2f UF\n", maxV-minV)
	fmt.Printf("Spread relativo (max-min)/último: %+.2f%%\n", (maxV-minV)/last*100)
	fmt.Printf("Último VTD (%s) es el usado hoy (GetLatestVector).\n", results[len(results)-1].date)
	if last > 0 {
		fmt.Printf("Desviación respecto al último: máx %+.2f%%, mín %+.2f%%\n",
			(maxV/last-1)*100, (minV/last-1)*100)
	}
	fmt.Printf("\nNota: este es el error de usar el VTD más reciente en vez del VTD\n")
	fmt.Printf("histórico de cada período de valuación. Para un período sin VTD\n")
	fmt.Printf("histórico, el error esperado está dentro de este rango.\n")
}
