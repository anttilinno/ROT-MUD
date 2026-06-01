package traits

import (
	"testing"

	"rotmud/pkg/types"
)

func TestResolve(t *testing.T) {
	t.Run("two +60 Fire vulnerabilities clamp at the negative cap (no stacking blowup)", func(t *testing.T) {
		// Vulnerabilities contribute negative magnitude into the per-axis sum.
		// 60 + 60 = 120 -> clamp to -CAP since the sign is negative.
		ts := TraitSet{
			Vulnerabilities: []Vulnerability{
				{DamageType: types.DamFire, Magnitude: 60},
				{DamageType: types.DamFire, Magnitude: 60},
			},
		}
		ts.Resolve()
		if got := ts.risSum[types.DamFire]; got != -CAP {
			t.Errorf("risSum[DamFire] = %d, expected -CAP (%d) after clamp", got, -CAP)
		}
	})

	t.Run("two +60 Fire resistances clamp at the positive cap", func(t *testing.T) {
		ts := TraitSet{
			Resistances: []Resistance{
				{DamageType: types.DamFire, Magnitude: 60},
				{DamageType: types.DamFire, Magnitude: 60},
			},
		}
		ts.Resolve()
		if got := ts.risSum[types.DamFire]; got != CAP {
			t.Errorf("risSum[DamFire] = %d, expected +CAP (%d) after clamp", got, CAP)
		}
	})

	t.Run("Resistance(+40) + Vulnerability(30) on same axis net to +10", func(t *testing.T) {
		ts := TraitSet{
			Resistances:     []Resistance{{DamageType: types.DamCold, Magnitude: 40}},
			Vulnerabilities: []Vulnerability{{DamageType: types.DamCold, Magnitude: 30}},
		}
		ts.Resolve()
		if got := ts.risSum[types.DamCold]; got != 10 {
			t.Errorf("risSum[DamCold] = %d, expected +10 (40 - 30)", got)
		}
	})

	t.Run("Immunity adds positive contribution toward Immune", func(t *testing.T) {
		ts := TraitSet{
			Immunities: []Immunity{{DamageType: types.DamPoison, Magnitude: 100}},
		}
		ts.Resolve()
		if got := ts.risSum[types.DamPoison]; got != CAP {
			t.Errorf("risSum[DamPoison] = %d, expected +CAP (%d)", got, CAP)
		}
	})

	t.Run("two StatModifiers on StatStr (+3, +2) sum to +5", func(t *testing.T) {
		ts := TraitSet{
			Modifiers: []StatModifier{
				{Stat: types.StatStr, Delta: 3},
				{Stat: types.StatStr, Delta: 2},
			},
		}
		ts.Resolve()
		if got := ts.modSum[types.StatStr]; got != 5 {
			t.Errorf("modSum[StatStr] = %d, expected +5", got)
		}
	})

	t.Run("stat modifier sum clamps to per-stat cap", func(t *testing.T) {
		ts := TraitSet{
			Modifiers: []StatModifier{
				{Stat: types.StatStr, Delta: ModCap},
				{Stat: types.StatStr, Delta: ModCap},
			},
		}
		ts.Resolve()
		if got := ts.modSum[types.StatStr]; got != ModCap {
			t.Errorf("modSum[StatStr] = %d, expected +ModCap (%d) after clamp", got, ModCap)
		}
		ts2 := TraitSet{
			Modifiers: []StatModifier{
				{Stat: types.StatStr, Delta: -ModCap},
				{Stat: types.StatStr, Delta: -ModCap},
			},
		}
		ts2.Resolve()
		if got := ts2.modSum[types.StatStr]; got != -ModCap {
			t.Errorf("modSum[StatStr] = %d, expected -ModCap (%d) after clamp", got, -ModCap)
		}
	})

	t.Run("interns each Capability.Key and ORs its bit into caps", func(t *testing.T) {
		ts := TraitSet{
			Capabilities: []Capability{{Key: "resolve-test-cap-a"}, {Key: "resolve-test-cap-b"}},
		}
		ts.Resolve()
		bitA, okA := lookupCapability("resolve-test-cap-a")
		bitB, okB := lookupCapability("resolve-test-cap-b")
		if !okA || !okB {
			t.Fatalf("expected both capabilities interned, got okA=%v okB=%v", okA, okB)
		}
		if !ts.caps.Has(bitA) || !ts.caps.Has(bitB) {
			t.Errorf("expected both capability bits OR'd into caps")
		}
	})

	t.Run("resolving twice is idempotent", func(t *testing.T) {
		ts := TraitSet{
			Vulnerabilities: []Vulnerability{{DamageType: types.DamFire, Magnitude: 20}},
			Resistances:     []Resistance{{DamageType: types.DamFire, Magnitude: 50}},
			Modifiers:       []StatModifier{{Stat: types.StatStr, Delta: 4}},
			Capabilities:    []Capability{{Key: "resolve-idempotent-cap"}},
		}
		ts.Resolve()
		firstRis := ts.risSum[types.DamFire]
		firstMod := ts.modSum[types.StatStr]
		firstCaps := ts.caps
		ts.Resolve()
		if ts.risSum[types.DamFire] != firstRis {
			t.Errorf("risSum changed on re-resolve: %d -> %d", firstRis, ts.risSum[types.DamFire])
		}
		if ts.modSum[types.StatStr] != firstMod {
			t.Errorf("modSum changed on re-resolve: %d -> %d", firstMod, ts.modSum[types.StatStr])
		}
		if ts.caps != firstCaps {
			t.Errorf("caps changed on re-resolve")
		}
		if !ts.resolved {
			t.Error("resolved flag should be true after Resolve")
		}
	})

	t.Run("capability overflow during Resolve is skipped without panic", func(t *testing.T) {
		// Fill the registry to its ceiling with fresh keys, then resolve a set
		// referencing an over-the-ceiling key. Must not panic.
		caps := make([]Capability, 0, capBitsCeiling+1)
		for i := 0; i < capBitsCeiling+1; i++ {
			caps = append(caps, Capability{Key: "overflow-fill-" + string(rune('A'+i%26)) + itoa(i)})
		}
		ts := TraitSet{Capabilities: caps}
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("Resolve panicked on capability overflow: %v", r)
			}
		}()
		ts.Resolve()
	})
}
