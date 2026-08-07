package loader

import (
	"testing"
)

func TestRIS1194StreamSample(t *testing.T) {
	ld := NewRIS1194Loader("/tmp/ris_sample.vta")
	policies, errs := ld.Stream(100)

	got := 0
	var totalReserve float64
	for {
		select {
		case p, ok := <-policies:
			if !ok {
				policies = nil
				break // breaks the select, not the for; falls to nil check
			}
			got++
			if len(p.Personas) == 0 {
				t.Fatalf("policy %s has no personas", p.NumeroInternoSVS)
			}
			for _, per := range p.Personas {
				totalReserve += per.ReserveBase().InexactFloat64()
			}
		case err, ok := <-errs:
			if ok && err != nil {
				t.Fatalf("stream error: %v", err)
			}
			errs = nil
		}
		if policies == nil && errs == nil {
			break
		}
	}
	if got == 0 {
		t.Fatal("no policies streamed")
	}
	t.Logf("streamed %d policies, total reported RT-BASE: %.2f UF", got, totalReserve)
}
