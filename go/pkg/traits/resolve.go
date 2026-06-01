package traits

import "rotmud/pkg/types"

// CAP is the per-axis RIS magnitude clamp (D-03). Each damage axis's summed
// magnitude is pinned to [-CAP, +CAP] so additive stacking across an arbitrary
// number of sources cannot blow up (TRAIT-02). Positive trends toward
// Resist/Immune; negative toward Vulnerable.
const CAP = 100

// ModCap is the per-stat summed-modifier clamp (Claude's Discretion). A single
// stat's combined delta from all sources is pinned to [-ModCap, +ModCap]. The
// bound is generous relative to the game's ~18-25 stat ceilings but prevents an
// unbounded pile of modifiers from overflowing the stat math downstream (P7).
const ModCap = 25

// clamp pins v to the per-axis RIS range [-CAP, +CAP].
func clamp(v int) int {
	if v > CAP {
		return CAP
	}
	if v < -CAP {
		return -CAP
	}
	return v
}

// clampMod pins v to the per-stat modifier range [-ModCap, +ModCap].
func clampMod(v int) int {
	if v > ModCap {
		return ModCap
	}
	if v < -ModCap {
		return -ModCap
	}
	return v
}

// Resolve collapses the composed per-kind slices into the cached query state:
// per-axis clamped RIS sums, per-stat clamped modifier sums, and the OR of
// interned capability bits. It is idempotent — re-resolving the same TraitSet
// yields identical caches because the caches are zeroed at the start.
//
// Sign convention (D-01): Resistance and Immunity magnitudes contribute
// POSITIVE to the per-axis sum (trending toward Resist/Immune); Vulnerability
// magnitudes contribute NEGATIVE (trending toward Vulnerable). The query layer
// ([TraitSet.ResolveImmunity]) maps the clamped sum back to the tri-state
// vocabulary using these same signs (Task 3 / D-03 thresholds).
//
// Iteration walks the slices in stored (source) order, so output is
// deterministic for a given composition.
func (ts *TraitSet) Resolve() {
	// Reset caches so re-resolving is idempotent.
	ts.caps = CapBits{}
	ts.risSum = make(map[types.DamageType]int, len(ts.Resistances)+len(ts.Immunities)+len(ts.Vulnerabilities))
	ts.modSum = [types.MaxStats]int{}

	// Per-axis RIS sum: Resistance/Immunity add positive, Vulnerability negative.
	// Clamp incrementally so the running sum stays in [-CAP, +CAP] after every
	// step. Each individual magnitude is also clamped before being added. This
	// makes int overflow impossible regardless of how large the unvalidated,
	// data-sourced magnitudes (P3 TOML) are: a pathological pile of huge
	// magnitudes can never accumulate past the bound and flip sign.
	for _, r := range ts.Resistances {
		ts.risSum[r.DamageType] = clamp(ts.risSum[r.DamageType] + clamp(r.Magnitude))
	}
	for _, im := range ts.Immunities {
		ts.risSum[im.DamageType] = clamp(ts.risSum[im.DamageType] + clamp(im.Magnitude))
	}
	for _, v := range ts.Vulnerabilities {
		ts.risSum[v.DamageType] = clamp(ts.risSum[v.DamageType] - clamp(v.Magnitude))
	}

	// Per-stat modifier sum, clamped incrementally for the same overflow-safety
	// reason as the RIS sums above.
	for _, m := range ts.Modifiers {
		if m.Stat < 0 || m.Stat >= types.MaxStats {
			continue // bounds-guard: ignore out-of-range stat indices
		}
		ts.modSum[m.Stat] = clampMod(ts.modSum[m.Stat] + clampMod(m.Delta))
	}

	// Intern each capability and OR its bit in. Overflow (>256 distinct) is
	// skipped without panic (bounded-growth defense carried forward to P3).
	for _, c := range ts.Capabilities {
		if bit, ok := internCapability(c.Key); ok {
			ts.caps.Set(bit)
		}
	}

	ts.resolved = true
}
