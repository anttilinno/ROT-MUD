---
phase: 02-trait-type-system
fixed_at: 2026-06-01T00:00:00Z
review_path: .planning/phases/02-trait-type-system/02-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 02: Code Review Fix Report

**Fixed at:** 2026-06-01T00:00:00Z
**Source review:** .planning/phases/02-trait-type-system/02-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (1 critical + 5 warning; 3 info findings out of scope)
- Fixed: 6
- Skipped: 0

All fixes verified with `gofmt`, `go build ./...`, `go vet`, `go test ./...`, and
`go test -race ./pkg/traits/` (the CR-01 data-race fix) — all green.

## Fixed Issues

### CR-01: Capability registry mutated under no lock — data race on concurrent Resolve

**Files modified:** `go/pkg/traits/capability.go`
**Commit:** 1513ed9
**Applied fix:** Added a package-level `sync.RWMutex` (`capMu`) guarding
`capRegistry` and `capNextBit`. `internCapability` now uses a read-locked fast
path for already-registered keys and a write-locked slow path (with a re-check
under the write lock) for assigning new bits. `lookupCapability` reads under the
read lock. This removes the concurrent map read/write data race. Verified
zero-alloc requirement (SC#4) is preserved — uncontended `RWMutex` reads do not
allocate; `TestHasCapabilityZeroAlloc` and `TestCapabilityZeroAlloc` still pass,
and `go test -race ./pkg/traits/` is clean.

### WR-01: Query layer returns silent "no effect" defaults on an unresolved TraitSet

**Files modified:** `go/pkg/traits/query.go`, `go/pkg/traits/query_test.go`
**Commit:** 65b274c
**Applied fix:** `HasCapability`, `GetModifier`, and `ResolveImmunity` now check
`ts.resolved` and auto-resolve (`ts.Resolve()`) when it is false, so a
composed-but-unresolved set never silently reports "no effect" (a fire-immune
entity is no longer treated as taking normal damage). Chose auto-resolve over a
panic to honor CLAUDE.md's "no panic in game logic" convention. Added
`TestQueryAutoResolvesUnresolvedSet` pinning the documented behavior for all
three query methods. Zero-alloc path is unaffected: already-resolved sets skip
the auto-resolve branch.

### WR-02: Registry overflow makes resolved capability sets order-dependent and non-deterministic

**Files modified:** `go/pkg/traits/resolve.go`
**Commit:** f50c01e
**Applied fix:** `Resolve` now emits a `slog.Warn` (with the dropped key and the
256-bit ceiling) whenever `internCapability` reports overflow for a not-yet-
registered key, instead of silently discarding it. Surfacing the drop lets a
content author learn their capability was discarded. Did not change the
`Resolve()` signature (would break all callers and existing tests); the longer-
term ceiling/closed-vocabulary redesign noted in the review is left for a future
phase.

### WR-03: RIS sum has no overflow guard before clamping

**Files modified:** `go/pkg/traits/resolve.go`
**Commit:** 3833fc0
**Applied fix:** Replaced the accumulate-then-clamp pattern with incremental
clamping for both the per-axis RIS sums and the per-stat modifier sums. Each
running sum is `clamp`/`clampMod`-pinned after every addition, and each
individual magnitude is also clamped before being added, so the accumulator can
never grow past `[-CAP, +CAP]` / `[-ModCap, +ModCap]` and overflow `int`
regardless of data-file magnitudes. Tri-state results are unchanged for all
realistic inputs (existing tests pass); this only bounds pathological inputs.

### WR-04: `ResolveImmunity` does not bounds-check the axis but `GetModifier` does

**Files modified:** `go/pkg/traits/query.go`, `go/pkg/traits/query_test.go`
**Commit:** 65b274c (shared with WR-01 — both edits live in query.go)
**Applied fix:** Chose the "document the existing contract" policy from the
review. Added an explicit "Unknown-axis contract" doc block to `ResolveImmunity`
stating that an axis with no contributing RIS trait reads as 0 -> `ImmNormal`
(intentional, because `risSum` is a map, unlike `GetModifier`'s fixed array which
rejects out-of-range indices). Added `TestResolveImmunityUnknownAxisIsNormal`
pinning the contract. This makes the divergence from `GetModifier` deliberate and
reviewable rather than ad hoc.

### WR-05: `Has`/`Set` on `CapBits` have no bound check; out-of-range bit indexes the array out of bounds

**Files modified:** `go/pkg/traits/capability.go`
**Commit:** 5f25fc1
**Applied fix:** Added `bit < 0 || bit >= capBitsCeiling` guards to both
`CapBits.Has` (returns false out of range) and `CapBits.Set` (no-op out of
range), preventing an out-of-bounds array index panic from any future or
external caller passing an arbitrary bit. Existing `TestCapBits` (which exercises
bits 0..255) still passes.

## Skipped Issues

None — all in-scope findings were fixed.

## Out-of-Scope (Info findings, not addressed)

IN-01 (capBitsCeiling/CapBits coupling by comment), IN-02 (`HooksFor` per-call
allocation), and IN-03 (Immunity-kind sign-convention surprise) are Info-tier and
outside the `critical_warning` fix scope. They remain documented in 02-REVIEW.md
for a future pass.

---

_Fixed: 2026-06-01T00:00:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
