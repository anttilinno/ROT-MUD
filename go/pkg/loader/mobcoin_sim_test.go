package loader

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// TestMobCoinDropDistribution loads every area in ../../data/areas, then for
// each mob template draws N samples from mobCoinDrop and reports the
// per-level-bucket distribution (count, median, p10, p90, total in copper).
//
// Run:  go test ./pkg/loader/ -run TestMobCoinDropDistribution -v
func TestMobCoinDropDistribution(t *testing.T) {
	const samplesPerMob = 50

	loader := NewAreaLoader("../../data/areas")
	world, err := loader.LoadAll()
	if err != nil {
		t.Fatalf("load areas: %v", err)
	}

	type bucket struct {
		mobCount int
		samples  []int64
	}
	buckets := map[string]*bucket{}
	bucketOrder := []string{
		"L1-5", "L6-10", "L11-20", "L21-30", "L31-50", "L51-75", "L76-100", "L101+",
	}
	bucketFor := func(lv int) string {
		switch {
		case lv <= 5:
			return "L1-5"
		case lv <= 10:
			return "L6-10"
		case lv <= 20:
			return "L11-20"
		case lv <= 30:
			return "L21-30"
		case lv <= 50:
			return "L31-50"
		case lv <= 75:
			return "L51-75"
		case lv <= 100:
			return "L76-100"
		default:
			return "L101+"
		}
	}
	for _, name := range bucketOrder {
		buckets[name] = &bucket{}
	}

	mobsWithGoldField := 0
	totalMobs := 0
	for _, tmpl := range world.MobTemplates {
		if tmpl.Level <= 0 {
			continue
		}
		totalMobs++
		if tmpl.Gold > 0 {
			mobsWithGoldField++
		}
		b := buckets[bucketFor(tmpl.Level)]
		b.mobCount++
		for i := 0; i < samplesPerMob; i++ {
			b.samples = append(b.samples, mobCoinDrop(tmpl))
		}
	}

	t.Logf("")
	t.Logf("=== MOB COIN DROP DISTRIBUTION (samples per mob = %d) ===", samplesPerMob)
	t.Logf("Total mob templates: %d   With explicit gold field: %d (%.1f%%)",
		totalMobs, mobsWithGoldField,
		100*float64(mobsWithGoldField)/float64(totalMobs))
	t.Logf("")

	hdr := fmt.Sprintf("%-10s  %6s  %8s  %12s  %12s  %12s",
		"Bucket", "Mobs", "Samples", "P10", "Median", "P90")
	t.Log(hdr)
	t.Log(strings.Repeat("-", len(hdr)))

	for _, name := range bucketOrder {
		b := buckets[name]
		if b.mobCount == 0 {
			continue
		}
		sort.Slice(b.samples, func(i, j int) bool { return b.samples[i] < b.samples[j] })
		n := len(b.samples)
		p10 := b.samples[n/10]
		p50 := b.samples[n/2]
		p90 := b.samples[n*9/10]
		t.Logf("%-10s  %6d  %8d  %12s  %12s  %12s",
			name, b.mobCount, n,
			types.FormatCoin(p10),
			types.FormatCoin(p50),
			types.FormatCoin(p90))
	}

	// Total economy: if every mob killed once, how much copper enters circulation?
	var totalDrop int64
	for _, b := range buckets {
		for _, v := range b.samples[:b.mobCount] { // first sample per mob = one kill
			totalDrop += v
		}
	}
	// Re-iterate cleanly (the slice above does not align with mob order; redo).
	totalDrop = 0
	for _, tmpl := range world.MobTemplates {
		if tmpl.Level <= 0 {
			continue
		}
		totalDrop += mobCoinDrop(tmpl)
	}
	t.Logf("")
	t.Logf("If every mob in world killed exactly once: %s enters economy.",
		types.FormatCoin(totalDrop))
}
