package traits

import "rotmud/pkg/types"

// TraitSet holds composed traits as homogeneous per-kind slices (D-04).
//
// Composition (Compose/Merge) only concatenates the per-kind slices in source
// order — summing, clamping, and capability interning happen later in
// [TraitSet.Resolve], which populates the unexported cache fields. The query
// API ([TraitSet.HasCapability], [TraitSet.GetModifier],
// [TraitSet.ResolveImmunity]) reads those caches and assumes Resolve has run.
type TraitSet struct {
	Vulnerabilities []Vulnerability
	Resistances     []Resistance
	Immunities      []Immunity
	Modifiers       []StatModifier
	Capabilities    []Capability
	Hooks           []BehaviorHook

	// Resolved caches, populated by Resolve(). A fresh Compose/Merge result is
	// unresolved (resolved == false) and these caches are zero/empty until then.
	caps     CapBits                  // OR of interned Capability bits (D-06)
	risSum   map[types.DamageType]int // clamped summed magnitude per RIS axis
	modSum   [types.MaxStats]int      // clamped summed stat deltas
	resolved bool                     // guard: true after Resolve() has run
}

// Compose concatenates the per-kind slices of each set in argument order
// (race -> class -> skill -> ...), preserving left-to-right source order in
// every kind slice so Resolve and HooksFor output is reproducible. The result
// is unresolved; call Resolve() before querying it.
func Compose(sets ...TraitSet) TraitSet {
	var out TraitSet
	for _, s := range sets {
		out.Merge(s)
	}
	return out
}

// Merge appends other's per-kind slices onto ts in place, preserving source
// order. It does not touch the resolved caches: a merged set must be
// re-resolved before its query results are valid.
func (ts *TraitSet) Merge(other TraitSet) {
	ts.Vulnerabilities = append(ts.Vulnerabilities, other.Vulnerabilities...)
	ts.Resistances = append(ts.Resistances, other.Resistances...)
	ts.Immunities = append(ts.Immunities, other.Immunities...)
	ts.Modifiers = append(ts.Modifiers, other.Modifiers...)
	ts.Capabilities = append(ts.Capabilities, other.Capabilities...)
	ts.Hooks = append(ts.Hooks, other.Hooks...)
	ts.resolved = false
}
