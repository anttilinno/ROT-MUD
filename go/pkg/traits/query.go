package traits

import "rotmud/pkg/types"

// ImmunityResult is the tri-state outcome of a resolved RIS query. Its
// constant order mirrors combat.ImmunityResult (combat.go:308-316) EXACTLY so
// the P7 combat wiring is a direct translation — but pkg/combat is NOT imported
// here (importing it would create a future cycle, D-07).
//
// Precedence: Immune > Resist > Vuln > Normal. The eventual combat damage
// semantics are resist = 2/3 damage, vuln = 3/2 damage, immune = 0 damage;
// verifying that parity is deferred to P7/P8, not this phase.
type ImmunityResult int

const (
	ImmNormal     ImmunityResult = iota // No RIS effect on this axis
	ImmImmune                           // Nullifies damage on this axis
	ImmResistant                        // Reduces damage on this axis
	ImmVulnerable                       // Increases damage on this axis
)

// String returns the immunity-result name, or "unknown" if out of range.
func (r ImmunityResult) String() string {
	names := []string{"normal", "immune", "resistant", "vulnerable"}
	if r >= 0 && int(r) < len(names) {
		return names[r]
	}
	return "unknown"
}

// HasTrait reports whether the per-kind slice for kind is non-empty. It reads
// the raw slices and does not require Resolve to have run.
func (ts *TraitSet) HasTrait(kind TraitKind) bool {
	switch kind {
	case KindVulnerability:
		return len(ts.Vulnerabilities) > 0
	case KindResistance:
		return len(ts.Resistances) > 0
	case KindImmunity:
		return len(ts.Immunities) > 0
	case KindModifier:
		return len(ts.Modifiers) > 0
	case KindCapability:
		return len(ts.Capabilities) > 0
	case KindHook:
		return len(ts.Hooks) > 0
	default:
		return false
	}
}

// HasCapability reports whether the resolved capability bitset contains key.
// It auto-resolves if Resolve has not yet run, so a freshly composed set never
// silently reports a present capability as absent — a forgotten Resolve would
// otherwise produce wrong-but-silent results under the behavioral-parity
// constraint. On an already-resolved set this path is O(1) and zero-allocation
// (SC#4): lookupCapability is a non-allocating map read and CapBits.Has is a
// value receiver — no strings are built, no slices ranged, no maps allocated.
func (ts *TraitSet) HasCapability(key string) bool {
	if !ts.resolved {
		ts.Resolve()
	}
	bit, ok := lookupCapability(key)
	if !ok {
		return false
	}
	return ts.caps.Has(bit)
}

// GetModifier returns the resolved, clamped summed delta for stat. It
// auto-resolves if Resolve has not yet run, and bounds-guards the index
// (out-of-range returns 0).
func (ts *TraitSet) GetModifier(stat types.Stat) int {
	if stat < 0 || stat >= types.MaxStats {
		return 0
	}
	if !ts.resolved {
		ts.Resolve()
	}
	return ts.modSum[stat]
}

// ResolveImmunity maps the resolved, clamped per-axis RIS sum to the tri-state
// vocabulary (D-03 thresholds). It auto-resolves if Resolve has not yet run, so
// a fire-immune entity is never silently treated as taking normal damage:
//
//	sum >= +CAP    -> Immune
//	+1 .. +CAP-1   -> Resist
//	sum  < 0       -> Vuln
//	sum == 0       -> Normal
//
// Unknown-axis contract: an axis with no contributing RIS trait has no entry in
// the risSum map and reads as 0 -> ImmNormal. This is intentional and matches
// the absence of any RIS effect; a garbage/out-of-enum DamageType is therefore
// treated as Normal rather than flagged. (GetModifier instead rejects
// out-of-range stat indices because modSum is a fixed array, not a map.)
//
// This mirrors combat.ImmunityResult's precedence (Immune > Resist > Vuln >
// Normal) without importing pkg/combat.
func (ts *TraitSet) ResolveImmunity(axis types.DamageType) ImmunityResult {
	if !ts.resolved {
		ts.Resolve()
	}
	sum := ts.risSum[axis]
	switch {
	case sum >= CAP:
		return ImmImmune
	case sum > 0:
		return ImmResistant
	case sum < 0:
		return ImmVulnerable
	default:
		return ImmNormal
	}
}

// HooksFor returns the behavior hooks bound to event, in stored (source) order.
// It reads the raw Hooks slice and does not require Resolve to have run.
func (ts *TraitSet) HooksFor(event HookEvent) []BehaviorHook {
	var out []BehaviorHook
	for _, h := range ts.Hooks {
		if h.Event == event {
			out = append(out, h)
		}
	}
	return out
}
