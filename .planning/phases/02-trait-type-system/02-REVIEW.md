---
phase: 02-trait-type-system
reviewed: 2026-06-01T00:00:00Z
depth: standard
files_reviewed: 10
files_reviewed_list:
  - go/pkg/traits/doc.go
  - go/pkg/traits/traits.go
  - go/pkg/traits/capability.go
  - go/pkg/traits/capability_test.go
  - go/pkg/traits/traitset.go
  - go/pkg/traits/resolve.go
  - go/pkg/traits/query.go
  - go/pkg/traits/traitset_test.go
  - go/pkg/traits/resolve_test.go
  - go/pkg/traits/query_test.go
findings:
  critical: 1
  warning: 5
  info: 3
  total: 9
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-06-01T00:00:00Z
**Depth:** standard
**Files Reviewed:** 10
**Status:** issues_found

## Summary

The `traits` package is the data-driven trait type-system foundation: trait
kinds, a `TraitSet` with per-kind slices, additive `Resolve` with clamping, a
256-bit capability registry, and a query layer. The package builds, vets clean,
and all tests pass. Parity claims verified against the real codebase:
`combat.ImmunityResult` iota order (combat.go:309-314) matches `query.go:15-20`
exactly; `types.MaxStats == 5` and `types.Stat = int` (constants.go:345-354)
match the struct/loop assumptions; `types.DamageType` is a plain `int` enum.

The defects below are concentrated in two seams: (1) the package-global
capability registry mutated during `Resolve()` is a data race waiting for P3+
concurrent loading, and (2) the query layer silently returns "no effect"
defaults when `Resolve()` was never called, which under the migration's
behavioral-parity constraint means a fire-immune entity can be silently treated
as taking normal damage. There are no external callers yet (grep across
`pkg/`/`cmd/` finds none), so these are latent, but they are baked into the
contract this phase ships.

## Critical Issues

### CR-01: Capability registry mutated under no lock — data race on concurrent Resolve

**File:** `go/pkg/traits/capability.go:33-52`, `go/pkg/traits/resolve.go:85-89`
**Issue:** `capRegistry` (a `map[string]int`) and `capNextBit` are package-level
mutable globals with no synchronization. `Resolve()` calls `internCapability`,
which writes to the map on first sight of a key. Concurrently, `HasCapability`
→ `lookupCapability` (query.go:57) reads the same map. A concurrent map
write + read in Go is a hard data race that can panic the process
("concurrent map read and map write") or silently corrupt the bit assignment.

The phase doc and CLAUDE.md state the game loop is single-threaded, but two
realistic paths break that assumption:
- World data loading (the P3 consumer this registry is explicitly built for,
  per the comment at capability.go:30-32) commonly composes/resolves trait sets
  for many entities at startup, which is exactly where parallelization is
  tempting.
- Any future Resolve on a player's composed set during login (handled on the
  per-connection goroutine, not the game loop) races against in-game
  `HasCapability` reads.

Because the registry is a *global* keyed by string and assigned lazily at
`Resolve` time, the race is not confined to a single character's state and is
not covered by the "character state is single-threaded" guarantee.

**Fix:** Guard the registry with a mutex (intern under write lock, lookup under
read lock), or move interning out of the hot/Resolve path into an explicit,
documented single-threaded registration step performed before any concurrent
access. Minimal mutex version:
```go
var (
	capMu       sync.RWMutex
	capRegistry = map[string]int{}
	capNextBit  int
)

func internCapability(key string) (int, bool) {
	capMu.RLock()
	if bit, ok := capRegistry[key]; ok {
		capMu.RUnlock()
		return bit, true
	}
	capMu.RUnlock()
	capMu.Lock()
	defer capMu.Unlock()
	if bit, ok := capRegistry[key]; ok { // re-check under write lock
		return bit, true
	}
	if capNextBit >= capBitsCeiling {
		return 0, false
	}
	bit := capNextBit
	capRegistry[key] = bit
	capNextBit++
	return bit, true
}

func lookupCapability(key string) (int, bool) {
	capMu.RLock()
	defer capMu.RUnlock()
	bit, ok := capRegistry[key]
	return bit, ok
}
```
Note this changes `lookupCapability` from zero-alloc-no-sync to lock-protected;
confirm SC#4's zero-alloc requirement is still met (an uncontended `RWMutex`
read does not allocate, so `TestHasCapabilityZeroAlloc` should still pass).

## Warnings

### WR-01: Query layer returns silent "no effect" defaults on an unresolved TraitSet

**File:** `go/pkg/traits/query.go:56-95`
**Issue:** `HasCapability`, `GetModifier`, and `ResolveImmunity` all read the
resolved caches (`ts.caps`, `ts.modSum`, `ts.risSum`) but never check the
`ts.resolved` guard that the struct deliberately carries (traitset.go:25). On a
freshly composed-but-unresolved set:
- `ResolveImmunity` reads `ts.risSum[axis]` on a nil map → returns 0 →
  `ImmNormal`. A fire-immune dragon is reported as taking *normal* fire damage.
- `GetModifier` returns 0; `HasCapability` returns false.

Under this project's hard behavioral-parity constraint ("migrated races/classes
must behave identically to current hardcoded definitions"), a forgotten
`Resolve()` does not crash — it silently produces wrong combat/magic outcomes,
which is far harder to catch than a panic. The struct already tracks
`resolved`, so the guard is free.
**Fix:** Either treat missing-Resolve as a programmer error caught loudly in
dev, or auto-resolve. Loud option:
```go
func (ts *TraitSet) ResolveImmunity(axis types.DamageType) ImmunityResult {
	if !ts.resolved {
		panic("traits: ResolveImmunity called before Resolve")
	}
	...
}
```
Per CLAUDE.md "no panic in game logic", prefer a debug-only assertion or having
callers route exclusively through a constructor that resolves. At minimum, add a
test asserting the documented behavior of querying an unresolved set so the
contract is pinned rather than accidental.

### WR-02: Registry overflow makes resolved capability sets order-dependent and non-deterministic across entities

**File:** `go/pkg/traits/resolve.go:85-89`, `go/pkg/traits/capability.go:41-52`
**Issue:** When the 256-bit ceiling is reached, `internCapability` returns
`ok=false` for any *not-yet-registered* key, and `Resolve` silently skips it
(resolve.go:86). Which capabilities "win" the 256 slots depends on global
registration order across the whole process, not on a single entity's input.
Two entities that both declare capability `"flight"` can end up with different
results purely based on which one resolved first and whether the registry was
already full — i.e., the same data file yields different in-game behavior
depending on load order. The doc claims determinism only "for the same input
order" (traitset.go:30, capability.go:26-28), which technically sidesteps this
but does not make the silent capability drop safe.
**Fix:** At minimum, surface overflow instead of silently dropping: have
`Resolve` collect dropped keys and return them / log via `slog` so a content
author learns their capability was discarded. Longer term, reconsider the fixed
256 ceiling versus the eventual capability vocabulary size, or reserve the
registry for a closed, statically known capability set registered once at init.

### WR-03: RIS sum has no overflow guard before clamping

**File:** `go/pkg/traits/resolve.go:59-69`
**Issue:** Per-axis magnitudes are summed as untyped `int` before `clamp` is
applied (lines 59-67), then clamped at line 69. Magnitudes come from data files
(P3) and are unvalidated `int`. A pathological data file with enough large
positive resistances and large negative vulnerabilities could in principle
overflow `int` during accumulation before the clamp ever runs, flipping the
sign and producing the opposite RIS result (e.g., an intended super-resistance
resolving to Vulnerable). Same applies to `modSum` accumulation (lines 73-78).
**Fix:** Clamp incrementally, or validate/clamp individual magnitudes at load
time so the running sum stays bounded:
```go
for _, r := range ts.Resistances {
	ts.risSum[r.DamageType] = clamp(ts.risSum[r.DamageType] + r.Magnitude)
}
```
Clamping each accumulation step keeps the running value in `[-CAP, CAP]` and
makes overflow impossible regardless of source magnitudes.

### WR-04: `ResolveImmunity` does not bounds-check the axis but `GetModifier` does

**File:** `go/pkg/traits/query.go:66-95`
**Issue:** Inconsistent defensive posture between two sibling query methods.
`GetModifier` guards `stat < 0 || stat >= types.MaxStats` (lines 67-69), but
`ResolveImmunity` indexes `ts.risSum[axis]` with no validation. Because
`risSum` is a map this won't panic, but an invalid/garbage `DamageType` (e.g.,
a value outside the defined enum range loaded from a malformed data file) is
silently treated as `ImmNormal` rather than flagged. The asymmetry also signals
the bounds discipline is ad hoc rather than a deliberate contract.
**Fix:** Decide one policy. If invalid axes should be rejected, validate against
the known `DamageType` range and log; if "unknown axis == Normal" is the
intended contract, document it explicitly on `ResolveImmunity` so the
divergence from `GetModifier` is intentional and reviewable.

### WR-05: `Has`/`Set` on `CapBits` have no bound check; out-of-range bit indexes the array out of bounds

**File:** `go/pkg/traits/capability.go:15-22`
**Issue:** `Has(bit)` computes `b[bit>>6]`; for `bit >= 256` (or negative) this
indexes outside the `[4]uint64` array and panics, and for negatives the
shift/mask math is undefined-ish. Internally callers only pass interned bits in
`[0,256)`, so it is safe *today*, but these are exported methods on an exported
type (`CapBits`), so any future or external caller passing an arbitrary bit
crashes the server. Given CLAUDE.md's "no panic in game logic" and "nil/bounds
checks throughout to prevent panics" conventions, an exported bit-set without
bounds guards is below the bar set elsewhere in the codebase.
**Fix:** Guard both methods:
```go
func (b CapBits) Has(bit int) bool {
	if bit < 0 || bit >= capBitsCeiling {
		return false
	}
	return b[bit>>6]&(1<<(uint(bit)&63)) != 0
}
func (b *CapBits) Set(bit int) {
	if bit < 0 || bit >= capBitsCeiling {
		return
	}
	b[bit>>6] |= 1 << (uint(bit) & 63)
}
```

## Info

### IN-01: `capBitsCeiling` and `CapBits` array width are coupled by a comment, not by code

**File:** `go/pkg/traits/capability.go:3-12`
**Issue:** `capBitsCeiling = 256` and `CapBits [4]uint64` must stay in lockstep
(4*64 = 256). Today only a comment enforces this. If someone widens `CapBits`
to `[8]uint64` and forgets the const (or vice versa), `Set`/`Has` silently
under/over-index relative to the ceiling.
**Fix:** Derive one from the other, e.g. `const capBitsWords = 4`, then
`type CapBits [capBitsWords]uint64` and `const capBitsCeiling = capBitsWords * 64`.

### IN-02: `HooksFor` allocates a fresh slice per call; returns shared underlying `BehaviorHook` values

**File:** `go/pkg/traits/query.go:99-107`
**Issue:** `HooksFor` is documented as a read path but builds a new slice each
call. Minor, and out of scope for v1 performance, but worth noting for the P7
hook-dispatch hot path: if hooks are queried per-damage-event this allocates on
every hit. Not a correctness issue. (Flagged as Info only.)
**Fix:** Consider a callback-style `RangeHooks(event, func(BehaviorHook))` for
the hot path later, or document that callers should resolve hooks once.

### IN-03: Sign-convention asymmetry — Immunity stored with a magnitude that must be >= CAP to actually immunize

**File:** `go/pkg/traits/traits.go:65-69`, `go/pkg/traits/resolve.go:62-64`, `go/pkg/traits/query.go:83-94`
**Issue:** An `Immunity` does not guarantee immunity: it contributes its
`Magnitude` positively to the same axis sum as `Resistance`, so an
`Immunity{Magnitude: 50}` only yields `ImmResistant`, while a
`Resistance{Magnitude: 100}` yields `ImmImmune` (verified by
`resolve_test.go:23-27` / `query_test.go:22-27`). The kind name implies a
guarantee the math does not provide. This is by design per D-01/D-03, but it is
a surprising contract for content authors and a likely source of
behavioral-parity bugs during migration (an author marks a creature "immune"
and it still takes damage).
**Fix:** Document prominently on the `Immunity` type that immunity is a function
of summed magnitude crossing CAP, not the kind alone — or treat `Immunity` as a
hard override that forces the axis to `ImmImmune` regardless of sum, which
matches author intent more closely. Confirm against the original hardcoded
RIS semantics before P7 wiring.

---

_Reviewed: 2026-06-01T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
