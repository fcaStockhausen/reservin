package portfolio

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"reservas/internal/calculator"
	"reservas/internal/database"
)

// BatchResult holds the result of calculating one policy in the batch.
type BatchResult struct {
	PolicyID     int
	NumeroPoliza string
	ReserveValue decimal.Decimal
	Error        error
	Duration     time.Duration
	Archetype    string
}

// BatchReport summarizes the batch run.
type BatchReport struct {
	TotalPolicies    int
	Successful       int
	Failed           int
	TotalReserve     decimal.Decimal
	MaxReserve       decimal.Decimal
	MinReserve       decimal.Decimal
	AvgReserve       decimal.Decimal
	TotalDuration    time.Duration
	AvgDuration      time.Duration
	ThroughputPerSec float64
	PeakMemoryMB     float64
	Results          []BatchResult
}

// CalculateBatch computes reserves for all policies in parallel using goroutines.
// It pre-loads all mortality tables once and shares the engine across workers.
func CalculateBatch(
	policies []PolicyResult,
	mortRepo *database.MortalityRepository,
	workers int,
) *BatchReport {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Pre-load all mortality tables once (shared, read-only after load).
	me := calculator.NewMortalityEngine()
	tableNames := []string{
		"CB-H-2020", "RV-M-2020", "B-M-2020",
		"MI-H-2020", "MI-M-2020",
		"CB-H-2014", "RV-M-2014", "B-M-2014",
		"MI-H-2014", "MI-M-2014",
		"B-H-2006", "B-M-2006", "MI-H-2006", "MI-M-2006",
		"RV-H-2004", "RV-M-2004", "RV-H-2009", "RV-M-2009",
	}
	for _, t := range tableNames {
		_ = me.EnsureLoaded(mortRepo, t)
	}

	n := len(policies)
	results := make([]BatchResult, n)

	jobs := make(chan int, n)
	var wg sync.WaitGroup

	startTime := time.Now()

	// Launch workers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each worker gets its own projector (lightweight, shares the cached engine).
			projector := calculator.NewFlowProjector(me)

			for idx := range jobs {
				pr := &policies[idx]
				polStart := time.Now()

				result := BatchResult{
					PolicyID:     pr.Policy.ID,
					NumeroPoliza: pr.Policy.NumeroPoliza,
				}

				if pr.Grupo.Causante == nil {
					result.Error = fmt.Errorf("no causante")
					results[idx] = result
					continue
				}

				// Step 1: unitary annuity factor
				policy := pr.Policy
				discountRate := policy.GetEffectiveDiscountRate()

				unitResult, err := projector.Project(policy, pr.Grupo, decimal.NewFromInt(1), discountRate, 0)
				if err != nil {
					result.Error = fmt.Errorf("annuity: %w", err)
					result.Duration = time.Since(polStart)
					results[idx] = result
					continue
				}

				annuityFactor := unitResult.TotalReserve
				if annuityFactor.LessThanOrEqual(decimal.Zero) {
					result.Error = fmt.Errorf("annuity factor <= 0")
					result.Duration = time.Since(polStart)
					results[idx] = result
					continue
				}

				// Step 2: derive pension and compute real reserve
				rentaAnual := policy.CapitalAsegurado.Div(annuityFactor)
				realResult, err := projector.Project(policy, pr.Grupo, rentaAnual, discountRate, 0)
				if err != nil {
					result.Error = fmt.Errorf("reserve: %w", err)
					result.Duration = time.Since(polStart)
					results[idx] = result
					continue
				}

				result.ReserveValue = realResult.TotalReserve
				result.Duration = time.Since(polStart)
				results[idx] = result
			}
		}()
	}

	// Distribute jobs
	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Build report
	report := &BatchReport{
		TotalPolicies: n,
		Results:       results,
		TotalDuration: totalDuration,
	}

	totalReserve := decimal.Zero
	maxReserve := decimal.Zero
	minReserve := decimal.NewFromInt(1).Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt(15)))
	var totalCalcTime time.Duration

	for _, r := range results {
		if r.Error != nil {
			report.Failed++
			continue
		}
		report.Successful++
		totalReserve = totalReserve.Add(r.ReserveValue)
		if r.ReserveValue.GreaterThan(maxReserve) {
			maxReserve = r.ReserveValue
		}
		if r.ReserveValue.LessThan(minReserve) {
			minReserve = r.ReserveValue
		}
		totalCalcTime += r.Duration
	}

	report.TotalReserve = totalReserve
	report.MaxReserve = maxReserve
	if report.Successful > 0 {
		report.MinReserve = minReserve
		report.AvgReserve = totalReserve.Div(decimal.NewFromInt(int64(report.Successful)))
		report.AvgDuration = totalCalcTime / time.Duration(report.Successful)
	}

	if totalDuration.Seconds() > 0 {
		report.ThroughputPerSec = float64(report.Successful) / totalDuration.Seconds()
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	report.PeakMemoryMB = float64(m.Sys) / (1024 * 1024)

	return report
}
