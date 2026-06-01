# Phase 2: Trait Type System - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-01
**Phase:** 2-trait-type-system
**Areas discussed:** RIS model, Trait storage shape, Capability flags, Phase 2 wiring scope

---

## RIS model

| Option | Description | Selected |
|--------|-------------|----------|
| Numeric pct, capped | Per-source percents sum and clamp; immunity = floor | |
| Tri-state + priority | Keep categorical; resolve by priority, no stacking | |
| Numeric internally, tri-state output | Stack numeric, map summed value back to ImmunityResult | ✓ |

**User's choice:** Numeric internally, tri-state output
**Notes:** Data stacks additively; combat's damage math stays unchanged → preserves Phase-1 golden parity.

### RIS axis (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| DamageType, ±100 cap | DamageType axis, hardcoded ±100 clamp | |
| DamageType, configurable cap | DamageType axis, cap is a package constant (default ±100) | ✓ |
| String axis key | Free-string axis for non-damage axes later | |

**User's choice:** DamageType, configurable cap
**Notes:** Axis reuses existing `types.DamageType`; cap value lives in one package constant for future tuning.

---

## Trait storage shape

| Option | Description | Selected |
|--------|-------------|----------|
| Per-kind typed slices | TraitSet of homogeneous slices; merge = append per slice | ✓ |
| []Trait interface slice | One slice + Kind() interface; query via type switch | |
| Raw + resolved split | Raw slices plus a built resolved cache | |

**User's choice:** Per-kind typed slices
**Notes:** Type-safe, no assertions in query path. Resolved capability cache still built (see Capability flags).

---

## Capability flags

| Option | Description | Selected |
|--------|-------------|----------|
| Closed enum + cache | Go const enum ORed into a uint64 mask | |
| Interned string keys | Registry maps capability strings → bits at load | ✓ |
| Defer bitmask to resolve step | Linear scan now, bitmask later | |

**User's choice:** Interned string keys
**Notes:** Strings forward-compatible with TOML (P3); registry interns to stable bits.

### Capability bit width (follow-up)

| Option | Description | Selected |
|--------|-------------|----------|
| uint64, hard cap 64 | Single mask, error at >64 capabilities | |
| Fixed [4]uint64 bitset | 256-bit value-type array, zero-alloc, headroom | ✓ |
| Grow as needed | []uint64 sized at load, unlimited but heap alloc | |

**User's choice:** Fixed [4]uint64 bitset
**Notes:** Keeps O(1) zero-alloc HasCapability (SC#4) with room for skills/spells/mobs/items.

---

## Phase 2 wiring scope

| Option | Description | Selected |
|--------|-------------|----------|
| Standalone pkg only | pkg/traits types + API + tests; no edits elsewhere | ✓ |
| Standalone + Character field | Also add unused ResolvedTraits to Character | |
| Standalone + RIS bridge | Also adapter mapping TraitSet RIS → ImmFlags/CheckImmune | |

**User's choice:** Standalone pkg only
**Notes:** Cleanest boundary; golden-master untouched by construction. Wiring deferred to P3/P7/P8.

---

## Claude's Discretion

- StatModifier shape (stat index + signed delta) and per-stat cap value.
- BehaviorHook representation: 5-event enum + script-reference string (no execution this phase); HooksFor ordering deterministic.
- Composition/merge determinism for reproducible Resolve() output.
- Closed TraitKind enum tagging the six kinds.

## Deferred Ideas

- Surfacing numeric/graded RIS in combat (replacing tri-state) — rejected for parity; future balance milestone.
- Capability count beyond 256 — fixed [4]uint64 ceiling; grow only on overflow.
- String/non-damage RIS axes (saves, etc.) — DamageType axis only for now.
