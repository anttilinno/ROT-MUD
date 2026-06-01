---
phase: 02-trait-type-system
plan: 01
subsystem: traits
tags: [traits, capability-registry, bitset, enums, foundation]
requires: [rotmud/pkg/types]
provides:
  - "go/pkg/traits package (standalone, compiles)"
  - "Six parameterized trait structs (Vulnerability, Resistance, Immunity, StatModifier, Capability, BehaviorHook)"
  - "Closed TraitKind enum (six kinds) and HookEvent enum (five events), each with String()"
  - "CapBits [4]uint64 256-bit primitive with O(1) zero-alloc Has and pointer-receiver Set"
  - "Deterministic string->bit capability intern registry with 256-bit ceiling guard"
  - "Non-allocating lookupCapability read path for the Plan 02 query layer"
affects:
  - "Plan 02-02 (composition/Resolve/query) builds TraitSet against these contracts"
tech-stack:
  added: []
  patterns:
    - "Closed enum + iota with names []string slice-index String() (mirrors types.DamageType)"
    - "Has value-receiver / Set pointer-receiver convention (mirrors types.ImmFlags)"
    - "Package-level string-interning registry with deterministic ascending bit assignment"
key-files:
  created:
    - go/pkg/traits/doc.go
    - go/pkg/traits/traits.go
    - go/pkg/traits/capability.go
    - go/pkg/traits/capability_test.go
  modified: []
decisions:
  - "Reused types.DamageType for the RIS axis and types.Stat for stat index (D-02) — no new string axis invented"
  - "Capability bits assigned in registration order for reproducible Resolve output (merge determinism)"
  - "256-bit ceiling enforced by non-panicking (0, false) overflow signal (D-06 / threat T-02-01)"
  - "HookEvent chosen as the five named events OnBeforeDamage/OnAfterDamage/OnDeath/OnSpellCast/OnLevelUp (LUA-02 alignment, Claude's Discretion)"
metrics:
  duration: ~20m
  completed: 2026-06-01
  tasks: 3
  files: 4
---

# Phase 2 Plan 01: Trait Type System Foundation Summary

Standalone `go/pkg/traits/` foundation package: six parameterized trait structs, closed `TraitKind`/`HookEvent` enums, and a deterministic capability string-interning registry built on a fixed 256-bit zero-alloc `CapBits` primitive — the typed contracts Plan 02's composition/query layer builds against.

## What Was Built

- **`doc.go`** — Package documentation block mirroring `combat/doc.go` shape, with `# Trait Kinds`, `# Behavior Hooks`, `# Capabilities` sections and cross-reference links.
- **`traits.go`** — `TraitKind` (six kinds: Vulnerability/Resistance/Immunity/Modifier/Capability/Hook) and `HookEvent` (five events) closed enums, each with a bounds-guarded `String()` using the `names []string` slice-index idiom. Six parameterized data-shaped structs reusing `types.DamageType` (RIS axis) and `types.Stat` (stat index). `BehaviorHook.Script` is a reference only — no Lua execution this phase.
- **`capability.go`** — `CapBits [4]uint64` value-type bitset with O(1) zero-alloc `Has` (value receiver) and `Set` (pointer receiver); a package-level `map[string]int` intern registry assigning bits in registration order with a 256-bit ceiling guard that returns `(0, false)` on overflow (never panics); and a non-allocating `lookupCapability` read path.
- **`capability_test.go`** — White-box `package traits` stdlib-only tests covering CapBits Has/Set round-trip across word boundaries (bits 0/63/64/255), registry determinism, non-inserting lookup, 257th-key overflow without panic, and `AllocsPerRun == 0` for both `CapBits.Has` and `lookupCapability`.

## Verification Results

- `cd go && go build ./pkg/traits/` — exit 0
- `cd go && go vet ./pkg/traits/` — exit 0
- `cd go && go test ./pkg/traits/` — all subtests pass (CapBits, Registry, Overflow, ZeroAlloc)
- `cd go && go test ./...` — full suite passes; no existing test broken
- Scope fence (D-07): `git diff --name-only` vs base shows only `go/pkg/traits/{doc,traits,capability,capability_test}.go` — zero edits to `Character`, `pkg/combat`, `pkg/magic`, `races.go`, `classes.go`, or `pkg/golden/`

## Deviations from Plan

None - plan executed exactly as written.

## Threat Model Notes

- **T-02-01 (DoS, capability registry):** Mitigated as planned. `internCapability` enforces the 256-bit ceiling with a non-panicking `(0, false)` overflow signal; covered by the 257th-key overflow test. No new untrusted-input boundary introduced this phase (interning is fed only by trusted in-process callers).
- **T-02-SC (supply chain):** No package-manager installs; pure stdlib + existing `rotmud/pkg/types`. No new Go modules added.

## Commits

- `96bcc17` feat(02-01): add traits package doc, trait structs, and closed enums
- `d40cc9b` feat(02-01): add capability registry and 256-bit CapBits primitive
- `8433483` test(02-01): cover CapBits zero-alloc, registry determinism, 256 ceiling

## Self-Check: PASSED

- FOUND: go/pkg/traits/doc.go
- FOUND: go/pkg/traits/traits.go
- FOUND: go/pkg/traits/capability.go
- FOUND: go/pkg/traits/capability_test.go
- FOUND commit: 96bcc17
- FOUND commit: d40cc9b
- FOUND commit: 8433483
