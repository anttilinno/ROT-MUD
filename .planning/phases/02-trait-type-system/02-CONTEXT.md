# Phase 2: Trait Type System - Context

**Gathered:** 2026-06-01
**Status:** Ready for planning

<domain>
## Phase Boundary

Deliver a **pure-Go trait type system** as a new standalone package (`pkg/traits`): typed
trait structs, additive composition with per-axis caps, and a query API
(`HasTrait`, `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`) covered by unit
tests. Satisfies TRAIT-01, TRAIT-02, TRAIT-03.

**In scope:** trait structs, a `TraitSet` composition value, `Resolve()`, the query API,
a capability string→bit registry, and unit tests.

**Out of scope (later phases):** TOML loaders (P3), Lua hook execution (P4), wiring traits
into `Character`/combat/magic, and replacing existing identity checks (P7/P8). No edits to
`Character`, `pkg/combat`, or `pkg/magic` in this phase.
</domain>

<decisions>
## Implementation Decisions

### RIS model (Resist / Immune / Vuln)
- **D-01: Numeric internally, tri-state output.** Each `Resistance`/`Vulnerability`/`Immunity`
  trait carries a numeric magnitude. Sources sum (race + class + ...) per axis, clamp to the
  cap, then `ResolveImmunity` maps the summed value back to the existing tri-state
  `combat.ImmunityResult` (Immune / Resist / Vuln / Normal) that combat already consumes.
  Goal: data stacks additively, but combat's damage math is unchanged → Phase-1 golden parity.
- **D-02: Axis = `types.DamageType`.** RIS axis keyed by the existing `types.DamageType` enum
  (fire, cold, silver, etc.) — reuse the vocabulary combat already speaks. No new string axis.
- **D-03: Configurable cap.** Summed magnitude clamped to `[-CAP, +CAP]` where `CAP` is a
  package constant (default `100`). Mapping: `>= +CAP` → Immune, `+1..<CAP` → Resist,
  `< 0` → Vuln, `0` → Normal. (Planner: confirm exact thresholds reproduce current
  halve/double/zero behavior for parity.)

### Trait storage shape
- **D-04: Per-kind typed slices.** A `TraitSet` struct holds homogeneous slices:
  `Vulnerabilities`, `Resistances`, `Immunities`, `Modifiers` (StatModifier),
  `Capabilities`, `Hooks` (BehaviorHook). Merge/compose = append per slice — no `[]Trait`
  interface, no type assertions in the query path. Most Go-idiomatic for a closed trait-kind set.

### Capability flags
- **D-05: Interned string keys.** Capabilities are strings (forward-compatible with TOML in P3).
  A package-level registry interns each known capability string to a stable bit at registration/load.
- **D-06: Fixed `[4]uint64` bitset.** `Resolve()` ORs the `Capabilities` slice into a 256-bit
  fixed-array bitset stored on the resolved `TraitSet`. `HasCapability` = bit test → O(1),
  zero-alloc (value-type array, not a heap slice). 256 bits gives headroom for
  skills/spells/mobs/items capabilities without a registry rewrite.

### Wiring scope
- **D-07: Standalone package only.** `pkg/traits` with types + composition + query API + unit
  tests. Zero edits to `Character`, `pkg/combat`, `pkg/magic`. Golden-master suite untouched
  by construction. Hook points into the live game come in P3/P7/P8.

### Claude's Discretion
- **StatModifier shape & stacking** — stat index + signed delta; summed across sources via
  `GetModifier`, with a sane per-stat cap. Exact cap value at planner discretion (caps exist
  per TRAIT-02; no current code constrains the number).
- **BehaviorHook representation** — event enum for the five named events (OnBeforeDamage,
  OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp) + a script-reference string (Lua host is P4;
  Phase 2 only stores the reference, does not execute). `HooksFor(event)` returns hooks in a
  deterministic source order.
- **Merge determinism** — composition order must be deterministic so `Resolve()` output is
  reproducible (matters for golden-master once wired later).
- **Closed `TraitKind` enum** — per TRAIT-01, a closed enum tagging the six kinds.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Requirements
- `.planning/REQUIREMENTS.md` — TRAIT-01, TRAIT-02, TRAIT-03 (exact acceptance wording, e.g.
  `Vulnerability{DamageType: Silver}` not `VulnerableToSilver`)
- `.planning/ROADMAP.md` §"Phase 2: Trait Type System" — goal + 4 success criteria (incl. SC#4
  O(1) zero-alloc HasCapability)

### Existing code to mirror / stay compatible with
- `go/pkg/types/constants.go:302` — `DamageType` enum (RIS axis vocabulary)
- `go/pkg/types/flags.go:357` — current `ImmFlags` bitmask + `Has/Set/Remove` (the tri-state
  model the numeric output must map back to)
- `go/pkg/combat/combat.go:308` — `CheckImmune` / `ImmunityResult` (the tri-state contract
  `ResolveImmunity` targets for parity)
- `go/pkg/types/races.go`, `go/pkg/types/classes.go` — current hardcoded tables traits will
  eventually replace (read for the axes/flags that must be expressible)

### Phase 1 (style + parity gate)
- `.planning/phases/01-golden-master-safety-net/01-CONTEXT.md` — golden-master conventions
- `go/pkg/golden/` — parity suite the trait system must not break (it won't this phase, by D-07)
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `types.DamageType` (constants.go:302): the RIS axis enum — reuse directly (D-02).
- `combat.ImmunityResult` + `CheckImmune` (combat.go:308–360): the tri-state output contract
  `ResolveImmunity` maps to; do not modify, just match its semantics.
- `ImmFlags` bitmask pattern (flags.go:357): precedent for the capability bitset approach.

### Established Patterns
- Closed `Prefix`-enum + `iota` constants throughout `pkg/types` — mirror for `TraitKind` and
  the BehaviorHook event enum.
- `Has`/`Set` receiver methods on flag types — mirror for the capability bitset API.
- Go `var` table definitions (races.go/classes.go) — the trait set is the future replacement;
  no need to touch them now.

### Integration Points
- None this phase (D-07). Future: composing a `TraitSet` per Character (P8) and querying it in
  combat/magic (P7). Keep the API shaped so those call sites are clean.
</code_context>

<specifics>
## Specific Ideas

- TRAIT-01 explicitly wants **parameterized** trait fields (`Vulnerability{DamageType: Silver}`),
  not boolean-named constants — keep traits data-shaped.
- The numeric→tri-state bridge (D-01) is the single most parity-sensitive piece; planner should
  call out reproducing current resist=halve / vuln=double / immune=zero behavior exactly when
  the system is eventually wired (verification happens P7/P8, not here).
</specifics>

<deferred>
## Deferred Ideas

- **Numeric RIS surfacing in combat** (replacing tri-state halve/double with graded percentages)
  — explicitly rejected for now; output stays tri-state for parity. Revisit only as a future
  balance milestone.
- **Open/unbounded capability count** (>256) — deferred; fixed `[4]uint64` is the ceiling. Grow
  to a sized bitset only if it ever overflows.
- **String/non-damage RIS axes** (e.g. saves) — deferred; `DamageType` axis only for now.
</deferred>

---

*Phase: 2-trait-type-system*
*Context gathered: 2026-06-01*
