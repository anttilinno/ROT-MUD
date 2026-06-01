---
phase: 02-trait-type-system
plan: 02
subsystem: traits
tags: [traits, composition, resolve, query-api, ris, zero-alloc, parity]
requires: [rotmud/pkg/types, "go/pkg/traits (Plan 02-01)"]
provides:
  - "TraitSet composition value with six per-kind slices + deterministic Compose/Merge"
  - "Resolve() — clamped per-axis RIS sums + clamped per-stat modifier sums + OR'd capability bitset"
  - "Five-method query API: HasTrait, HasCapability, GetModifier, ResolveImmunity, HooksFor (TRAIT-03)"
  - "traits.ImmunityResult tri-state enum mirroring combat.ImmunityResult order without importing pkg/combat"
  - "Zero-allocation HasCapability path (SC#4, proven by AllocsPerRun == 0)"
affects:
  - "P3 TOML loaders will Compose race/class TraitSets and call Resolve()"
  - "P7 combat wiring translates traits.ImmunityResult -> combat.ImmunityResult (direct, same order)"
tech-stack:
  added: []
  patterns:
    - "Struct-of-homogeneous-slices composition via append-merge (D-04, no []Trait interface)"
    - "Cached-resolve: raw slices -> clamped sums + bitset, idempotent re-resolve by zeroing caches"
    - "Tri-state enum mirrored by order/value (not import) to avoid future import cycle (D-07)"
key-files:
  created:
    - go/pkg/traits/traitset.go
    - go/pkg/traits/resolve.go
    - go/pkg/traits/query.go
    - go/pkg/traits/traitset_test.go
    - go/pkg/traits/resolve_test.go
    - go/pkg/traits/query_test.go
  modified: []
decisions:
  - "RIS sign convention (D-01): Resistance/Immunity add POSITIVE magnitude, Vulnerability adds NEGATIVE; ResolveImmunity reads the same sign"
  - "ModCap = 25 per-stat modifier clamp (Claude's Discretion) — generous vs ~18-25 stat ceilings, prevents overflow downstream"
  - "ResolveImmunity thresholds (D-03): sum>=CAP Immune, +1..CAP-1 Resist, <0 Vuln, ==0 Normal; CAP boundary is inclusive-Immune"
  - "Resolve zeroes caches at entry for idempotent re-resolve; Merge/Compose mark the set unresolved"
metrics:
  duration: ~15m
  completed: 2026-06-01
  tasks: 3
  files: 6
---

# Phase 2 Plan 02: Trait Composition & Query Layer Summary

The composition + query layer of `go/pkg/traits/` on top of Plan 01's structs and `CapBits`: a `TraitSet` value with six deterministic per-kind slices, a `Resolve()` that collapses them into clamped per-axis RIS sums, clamped per-stat modifier sums, and an OR'd capability bitset, and the five-method read-only query API including the parity-critical numeric-to-tri-state `ResolveImmunity` bridge and the SC#4 zero-alloc `HasCapability`.

## What Was Built

- **`traitset.go`** — `TraitSet` struct with the six exported homogeneous per-kind slices (`Vulnerabilities, Resistances, Immunities, Modifiers, Capabilities, Hooks`, D-04 — no `[]Trait` interface, no type assertions) plus unexported resolved caches (`caps CapBits`, `risSum map[types.DamageType]int`, `modSum [types.MaxStats]int`, `resolved bool`). `Compose(sets ...TraitSet)` and in-place `Merge(other)` append per-kind slices in left-to-right source order (race -> class -> ...); merging marks the set unresolved so a fresh `Compose` result must be `Resolve()`d before querying.
- **`resolve.go`** — `const CAP = 100` (D-03 per-axis clamp) and `const ModCap = 25` (per-stat clamp) with `clamp`/`clampMod` helpers. `Resolve()` zeroes its caches (idempotent), sums RIS magnitude per `types.DamageType` (Resistance/Immunity positive, Vulnerability negative — D-01), clamps each axis to `[-CAP, +CAP]`, sums stat modifiers per index (bounds-guarded) clamped to `[-ModCap, +ModCap]`, interns each `Capability.Key` and ORs its bit into `caps` (overflow >256 skipped without panic), and sets `resolved = true`.
- **`query.go`** — `type ImmunityResult int` with `ImmNormal/ImmImmune/ImmResistant/ImmVulnerable` in the SAME iota order as `combat.ImmunityResult` (mirrored, NOT imported — D-07) plus a `String()`. Five read-only methods on `*TraitSet`: `HasTrait(TraitKind)`, `HasCapability(string)` (O(1) zero-alloc via non-allocating `lookupCapability` + value-receiver `CapBits.Has`), `GetModifier(types.Stat)` (bounds-guarded), `ResolveImmunity(types.DamageType)` (maps clamped `risSum` per D-03 thresholds), and `HooksFor(HookEvent)` (filters in source order).
- **Tests** — `traitset_test.go` (Compose/Merge order preservation, empty/zero-arg, unresolved-after-Compose), `resolve_test.go` (+60+60 Fire cap-boundary clamp proving TRAIT-02, RIS net, Immunity contribution, stat sum + ModCap clamp, idempotent re-resolve, capability overflow without panic), `query_test.go` (ResolveImmunity at every threshold boundary, enum-order-mirrors-combat, HasCapability present/absent, `AllocsPerRun(1000, ...) == 0` for SC#4, GetModifier clamp + bounds guard, HasTrait present/absent, HooksFor ordering, String()).

## Verification Results

- `cd go && go test ./pkg/traits/` — all 57 subtests pass (incl. zero-alloc `HasCapability`, +60+60 cap clamp, idempotent Resolve, overflow-no-panic)
- `cd go && go vet ./pkg/traits/` — exit 0
- `cd go && ! grep -rq '"rotmud/pkg/combat"' pkg/traits/` — confirmed: `pkg/combat` is NOT imported (no future cycle, D-07)
- `cd go && go test ./...` — full suite passes; no existing test broken (golden-master untouched)
- Scope fence (D-07): `git diff --name-only` vs base shows only the six new `go/pkg/traits/{traitset,resolve,query}{,_test}.go` files — zero edits to `Character`, `pkg/combat`, `pkg/magic`, `races.go`, `classes.go`, or `pkg/golden/`

## Deviations from Plan

None - plan executed exactly as written.

(One trivial collision handled while writing tests: the overflow test initially declared a local `itoa` helper that already existed in `capability_test.go` from Plan 01; the duplicate was removed and the existing helper reused. Not a behavior change — noted only for completeness.)

## Threat Model Notes

- **T-02-02 (DoS, per-axis RIS sum):** Mitigated as planned. The `[-CAP, +CAP]` clamp in `Resolve()` bounds each axis's magnitude regardless of source count; covered by the +60+60 Fire cap-boundary test (TRAIT-02). Stat sums are likewise bounded by `ModCap`.
- **T-02-03 (DoS, capability interning):** Accepted as planned. The 256-bit ceiling from Plan 01's registry bounds growth; `Resolve()` skips overflow via the `(_, false)` signal without panic, proven by the overflow-fill test.
- **T-02-SC (supply chain):** No package-manager installs; pure stdlib + existing `rotmud/pkg/types`. No new Go modules added.
- No new trust boundary introduced: pure in-memory composition/query, no network I/O, no untrusted input this phase.

## Commits

- `3dc5ffe` test(02-02): add failing tests for TraitSet Compose/Merge order preservation
- `60339a8` feat(02-02): add TraitSet composition value with deterministic Merge/Compose
- `db1c311` test(02-02): add failing tests for Resolve clamped RIS/stat sums + capability bits
- `071bd6d` feat(02-02): add Resolve producing clamped RIS/stat sums + capability bitset
- `e1f13c0` test(02-02): add failing tests for query API + tri-state ResolveImmunity + zero-alloc
- `ff04706` feat(02-02): add query API + tri-state ResolveImmunity + zero-alloc HasCapability

## TDD Gate Compliance

All three tasks followed RED -> GREEN: each `test(...)` commit precedes its paired `feat(...)` commit and was verified failing (compile-fail RED) before implementation. No REFACTOR commits were needed.

## Self-Check: PASSED

- FOUND: go/pkg/traits/traitset.go
- FOUND: go/pkg/traits/resolve.go
- FOUND: go/pkg/traits/query.go
- FOUND: go/pkg/traits/traitset_test.go
- FOUND: go/pkg/traits/resolve_test.go
- FOUND: go/pkg/traits/query_test.go
- FOUND commit: 3dc5ffe
- FOUND commit: 60339a8
- FOUND commit: db1c311
- FOUND commit: 071bd6d
- FOUND commit: e1f13c0
- FOUND commit: ff04706
