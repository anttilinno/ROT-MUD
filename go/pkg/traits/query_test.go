package traits

import (
	"testing"

	"rotmud/pkg/types"
)

func TestResolveImmunity(t *testing.T) {
	tests := []struct {
		name string
		ts   TraitSet
		axis types.DamageType
		want ImmunityResult
	}{
		{
			name: "sum >= +CAP -> Immune",
			ts:   TraitSet{Immunities: []Immunity{{DamageType: types.DamFire, Magnitude: 100}}},
			axis: types.DamFire,
			want: ImmImmune,
		},
		{
			name: "sum == +CAP exactly -> Immune (boundary)",
			ts:   TraitSet{Resistances: []Resistance{{DamageType: types.DamCold, Magnitude: 100}}},
			axis: types.DamCold,
			want: ImmImmune,
		},
		{
			name: "sum == +CAP-1 -> Resist (boundary just below immune)",
			ts:   TraitSet{Resistances: []Resistance{{DamageType: types.DamAcid, Magnitude: CAP - 1}}},
			axis: types.DamAcid,
			want: ImmResistant,
		},
		{
			name: "small positive sum -> Resist",
			ts:   TraitSet{Resistances: []Resistance{{DamageType: types.DamPoison, Magnitude: 1}}},
			axis: types.DamPoison,
			want: ImmResistant,
		},
		{
			name: "sum == -1 -> Vuln (boundary just below normal)",
			ts:   TraitSet{Vulnerabilities: []Vulnerability{{DamageType: types.DamLightning, Magnitude: 1}}},
			axis: types.DamLightning,
			want: ImmVulnerable,
		},
		{
			name: "large negative sum -> Vuln",
			ts:   TraitSet{Vulnerabilities: []Vulnerability{{DamageType: types.DamBash, Magnitude: 80}}},
			axis: types.DamBash,
			want: ImmVulnerable,
		},
		{
			name: "sum == 0 -> Normal",
			ts:   TraitSet{},
			axis: types.DamSlash,
			want: ImmNormal,
		},
		{
			name: "net-zero (resist + equal vuln) -> Normal",
			ts: TraitSet{
				Resistances:     []Resistance{{DamageType: types.DamSilver, Magnitude: 30}},
				Vulnerabilities: []Vulnerability{{DamageType: types.DamSilver, Magnitude: 30}},
			},
			axis: types.DamSilver,
			want: ImmNormal,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.ts.Resolve()
			if got := tc.ts.ResolveImmunity(tc.axis); got != tc.want {
				t.Errorf("ResolveImmunity(%v) = %v, expected %v", tc.axis, got, tc.want)
			}
		})
	}
}

func TestImmunityResultOrderMirrorsCombat(t *testing.T) {
	// Must match combat.ImmunityResult iota order exactly (D-07 parity).
	if ImmNormal != 0 || ImmImmune != 1 || ImmResistant != 2 || ImmVulnerable != 3 {
		t.Errorf("ImmunityResult order = (%d,%d,%d,%d), expected (0,1,2,3) to mirror combat",
			ImmNormal, ImmImmune, ImmResistant, ImmVulnerable)
	}
}

func TestHasCapability(t *testing.T) {
	t.Run("present after Resolve", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "query-flight"}}}
		ts.Resolve()
		if !ts.HasCapability("query-flight") {
			t.Error("expected query-flight capability present after Resolve")
		}
	})
	t.Run("absent capability returns false", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "query-flight"}}}
		ts.Resolve()
		if ts.HasCapability("query-never-registered-xyz") {
			t.Error("expected unregistered capability to be absent")
		}
	})
}

func TestQueryAutoResolvesUnresolvedSet(t *testing.T) {
	// WR-01: querying a composed-but-unresolved set must not silently report
	// "no effect". The query methods auto-resolve so a forgotten Resolve()
	// cannot produce wrong-but-silent combat/magic outcomes.
	t.Run("ResolveImmunity auto-resolves (fire-immune entity is not Normal)", func(t *testing.T) {
		ts := TraitSet{Immunities: []Immunity{{DamageType: types.DamFire, Magnitude: 100}}}
		// Deliberately NOT calling ts.Resolve() here.
		if got := ts.ResolveImmunity(types.DamFire); got != ImmImmune {
			t.Errorf("ResolveImmunity on unresolved set = %v, expected ImmImmune (auto-resolve)", got)
		}
	})
	t.Run("HasCapability auto-resolves", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "autoresolve-flight"}}}
		if !ts.HasCapability("autoresolve-flight") {
			t.Error("HasCapability on unresolved set = false, expected true (auto-resolve)")
		}
	})
	t.Run("GetModifier auto-resolves", func(t *testing.T) {
		ts := TraitSet{Modifiers: []StatModifier{{Stat: types.StatStr, Delta: 4}}}
		if got := ts.GetModifier(types.StatStr); got != 4 {
			t.Errorf("GetModifier on unresolved set = %d, expected 4 (auto-resolve)", got)
		}
	})
}

func TestResolveImmunityUnknownAxisIsNormal(t *testing.T) {
	// WR-04: an axis with no contributing RIS trait reads as ImmNormal. This is
	// the documented contract for ResolveImmunity (unknown axis == Normal).
	ts := TraitSet{Resistances: []Resistance{{DamageType: types.DamFire, Magnitude: 50}}}
	ts.Resolve()
	if got := ts.ResolveImmunity(types.DamCold); got != ImmNormal {
		t.Errorf("ResolveImmunity(unknown axis) = %v, expected ImmNormal", got)
	}
}

func TestHasCapabilityZeroAlloc(t *testing.T) {
	ts := TraitSet{Capabilities: []Capability{{Key: "zeroalloc-flight"}}}
	ts.Resolve()
	// SC#4: HasCapability must be O(1) and zero-allocation per call.
	allocs := testing.AllocsPerRun(1000, func() {
		_ = ts.HasCapability("zeroalloc-flight")
	})
	if allocs != 0 {
		t.Errorf("HasCapability allocs/op = %v, expected 0 (SC#4 zero-alloc)", allocs)
	}
}

func TestGetModifier(t *testing.T) {
	t.Run("returns clamped summed delta", func(t *testing.T) {
		ts := TraitSet{Modifiers: []StatModifier{
			{Stat: types.StatStr, Delta: 3},
			{Stat: types.StatStr, Delta: 2},
		}}
		ts.Resolve()
		if got := ts.GetModifier(types.StatStr); got != 5 {
			t.Errorf("GetModifier(StatStr) = %d, expected 5", got)
		}
	})
	t.Run("out-of-range stat returns 0 without panic", func(t *testing.T) {
		ts := TraitSet{}
		ts.Resolve()
		if got := ts.GetModifier(-1); got != 0 {
			t.Errorf("GetModifier(-1) = %d, expected 0", got)
		}
		if got := ts.GetModifier(types.MaxStats + 5); got != 0 {
			t.Errorf("GetModifier(out-of-range) = %d, expected 0", got)
		}
	})
}

func TestHasTrait(t *testing.T) {
	ts := TraitSet{
		Vulnerabilities: []Vulnerability{{DamageType: types.DamFire, Magnitude: 1}},
		Hooks:           []BehaviorHook{{Event: OnDeath, Script: "x.lua"}},
	}
	t.Run("present kind", func(t *testing.T) {
		if !ts.HasTrait(KindHook) {
			t.Error("expected HasTrait(KindHook) true when Hooks non-empty")
		}
		if !ts.HasTrait(KindVulnerability) {
			t.Error("expected HasTrait(KindVulnerability) true")
		}
	})
	t.Run("absent kind", func(t *testing.T) {
		if ts.HasTrait(KindCapability) {
			t.Error("expected HasTrait(KindCapability) false when Capabilities empty")
		}
		if ts.HasTrait(KindModifier) {
			t.Error("expected HasTrait(KindModifier) false when Modifiers empty")
		}
	})
}

func TestHooksFor(t *testing.T) {
	t.Run("returns only matching events in source order", func(t *testing.T) {
		ts := TraitSet{Hooks: []BehaviorHook{
			{Event: OnDeath, Script: "death-1.lua"},
			{Event: OnLevelUp, Script: "level.lua"},
			{Event: OnDeath, Script: "death-2.lua"},
		}}
		got := ts.HooksFor(OnDeath)
		if len(got) != 2 {
			t.Fatalf("HooksFor(OnDeath) len = %d, expected 2", len(got))
		}
		if got[0].Script != "death-1.lua" || got[1].Script != "death-2.lua" {
			t.Errorf("HooksFor(OnDeath) = %+v, expected [death-1, death-2] in source order", got)
		}
	})
	t.Run("no matching events returns empty", func(t *testing.T) {
		ts := TraitSet{Hooks: []BehaviorHook{{Event: OnDeath, Script: "x.lua"}}}
		if got := ts.HooksFor(OnSpellCast); len(got) != 0 {
			t.Errorf("HooksFor(OnSpellCast) = %+v, expected empty", got)
		}
	})
}

func TestImmunityResultString(t *testing.T) {
	cases := map[ImmunityResult]string{
		ImmNormal:     "normal",
		ImmImmune:     "immune",
		ImmResistant:  "resistant",
		ImmVulnerable: "vulnerable",
	}
	for r, want := range cases {
		if got := r.String(); got != want {
			t.Errorf("ImmunityResult(%d).String() = %q, expected %q", int(r), got, want)
		}
	}
}
