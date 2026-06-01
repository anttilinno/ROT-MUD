# Phase 2: Trait Type System - Pattern Map

**Mapped:** 2026-06-01
**Files analyzed:** 7 (all new — new standalone package `go/pkg/traits/`, zero edits to existing files per D-07)
**Analogs found:** 7 / 7 (all analogs in-repo; the capability string→bit registry is the only partial — no exact precedent exists)

This phase creates a **new standalone package** `go/pkg/traits/`. No existing files are modified. Every analog below is for pattern-copying only — the executor mirrors the *style* (enum + iota, flag receiver methods, `var` tables, table-driven tests) into new files. The numeric→tri-state RIS bridge must map back to `combat.ImmunityResult` exactly (D-01), so `ResolveImmunity` is written to *target* that enum's vocabulary even though `pkg/combat` is not imported (importing combat would create a cycle later; mirror the enum's semantics, do not depend on it).

## File Classification

Suggested file layout (planner may split/merge; roles + analogs are what matter):

| New File | Role | Data Flow | Closest Analog | Match Quality |
|----------|------|-----------|----------------|---------------|
| `go/pkg/traits/doc.go` | package-doc | n/a | `go/pkg/combat/doc.go:1-12` | exact |
| `go/pkg/traits/traits.go` (trait structs + `TraitKind`/event enums) | model / value-types | transform (data-shaped, no I/O) | `go/pkg/types/constants.go:302-327` (`DamageType` enum), `go/pkg/types/races.go:4-13` (struct field doc style) | exact |
| `go/pkg/traits/capability.go` (string→bit registry + `[4]uint64` bitset + `Has`) | registry + bitset | transform / lookup | `go/pkg/types/flags.go:357-397` (`ImmFlags` bitmask + receiver methods) for the bitset; **no exact analog** for string-interning registry | role-match (bitset exact; registry partial) |
| `go/pkg/traits/traitset.go` (`TraitSet` struct, per-kind slices) | model / composition value | event-driven aggregation (append-merge) | `go/pkg/types/races.go:4-13` + `go/pkg/combat/combat.go:33-48` (struct-of-fields composition) | role-match |
| `go/pkg/traits/resolve.go` (`Resolve()` → bitset + summed caches) | service / pure transform | batch (iterate slices → clamp → bitset) | `go/pkg/types/constants.go:289-300` (table→value transform style) | role-match |
| `go/pkg/traits/query.go` (`HasTrait`, `HasCapability`, `GetModifier`, `ResolveImmunity`, `HooksFor`) | query API | request-response (read-only lookups) | `go/pkg/combat/combat.go:308-373` (`CheckImmune`/`ImmunityResult` tri-state contract) | exact (semantic target) |
| `go/pkg/traits/traits_test.go` (+ optional `resolve_test.go`, `query_test.go`) | test | request-response (subtest assertions) | `go/pkg/types/flags_test.go:1-53`, `go/pkg/combat/combat_test.go:1-27` | exact |

## Pattern Assignments

### `go/pkg/traits/doc.go` (package-doc)

**Analog:** `go/pkg/combat/doc.go` (lines 1-12)

**Doc-file structure** (copy verbatim shape):
```go
// Package traits implements the data-driven trait type system for the ROT MUD.
//
// Traits are typed, parameterized annotations (vulnerabilities, resistances,
// immunities, stat modifiers, capability flags, behavior hooks) that compose
// additively into a resolved [TraitSet] queried by combat, magic, and skills.
//
// # Trait Kinds
//
//   - ...
package traits
```
**What to copy:** package comment block immediately above `package`; `# Section` headings rendered by `go doc`; `[TypeName]` cross-reference links.

---

### `go/pkg/traits/traits.go` (model — trait structs + closed enums)

**Analog A (closed enum, the `TraitKind` + BehaviorHook event enum):** `go/pkg/types/constants.go:302-327`
```go
// DamageType represents damage types
type DamageType int

const (
	DamNone DamageType = iota
	DamBash
	DamPierce
	...
	DamSilver // Silver weapons — vampires are vulnerable
)
```
**What to copy:** named int type + `const (... = iota)` block. Mirror for:
- `TraitKind` (closed, six kinds per D-04/TRAIT-01): `KindVulnerability, KindResistance, KindImmunity, KindModifier, KindCapability, KindHook`.
- `HookEvent` enum for the five named events (Claude's Discretion): `OnBeforeDamage, OnAfterDamage, OnDeath, OnSpellCast, OnLevelUp`.
- Add a `String()` method on each enum following the `names []string` slice-index pattern at `constants.go:289-300` and `357-364`.

**Analog B (parameterized, data-shaped struct fields — NOT boolean-named constants, per TRAIT-01 / specifics):** `go/pkg/types/races.go:4-13`
```go
// Race represents a playable character race
type Race struct {
	Name            string        // Race name
	...
	Size            Size          // Race size
}
```
**What to copy:** one struct per trait kind with trailing inline `//` doc comments. Per the requirement wording (`Vulnerability{DamageType: Silver}` not `VulnerableToSilver`):
```go
// Vulnerability increases damage taken on a damage axis.
type Vulnerability struct {
	DamageType types.DamageType // RIS axis (D-02: reuse existing enum)
	Magnitude  int              // numeric, summed across sources (D-01)
}
// Resistance, Immunity mirror this shape.
// StatModifier: { Stat types.Stat; Delta int }
// Capability:   { Key string }            // interned at Resolve (D-05)
// BehaviorHook: { Event HookEvent; Script string }  // Script = ref only; Lua exec is P4
```
**Axis vocabulary (D-02):** import `rotmud/pkg/types` and key RIS on `types.DamageType`; stat index on `types.Stat`. Do **not** invent a new string axis.

---

### `go/pkg/traits/capability.go` (registry + fixed bitset)

**Analog (bitset receiver-method pattern):** `go/pkg/types/flags.go:357-397`
```go
type ImmFlags uint32

const (
	ImmSummon ImmFlags = 1 << iota // Immune to summon
	...
)

// Has returns true if the flag is set
func (f ImmFlags) Has(flag ImmFlags) bool { return f&flag != 0 }
// Set adds a flag
func (f *ImmFlags) Set(flag ImmFlags) { *f |= flag }
// Remove clears a flag
func (f *ImmFlags) Remove(flag ImmFlags) { *f &^= flag }
```
**What to copy:** the `Has`/`Set` receiver-method convention (value receiver for `Has`, pointer receiver for mutators). **Adapt** from a single `uintNN` to a fixed `[4]uint64` (256 bits, D-06) so `HasCapability` is O(1) and zero-alloc (value-type array, not heap slice):
```go
type CapBits [4]uint64

func (b CapBits) Has(bit int) bool { return b[bit>>6]&(1<<(uint(bit)&63)) != 0 }
func (b *CapBits) Set(bit int)     { b[bit>>6] |= 1 << (uint(bit) & 63) }
```

**Registry (NO exact analog — see "No Analog Found"):** package-level `map[string]int` interning each known capability string to a stable bit at registration/load (D-05). Closest in-repo precedent for a package-level mutable lookup is `map[string]int` usage at `go/pkg/skills/system.go` (`ch.PCData.Learned = make(map[string]int)`), but that is per-character state, not a global intern table — treat as a hint only. Keep registration deterministic (assign bits in registration order) so `Resolve()` output is reproducible (Claude's Discretion: merge determinism). Guard against >256 overflow (deferred ceiling per Deferred Ideas).

---

### `go/pkg/traits/traitset.go` (`TraitSet` composition value — D-04 per-kind typed slices)

**Analog (struct-of-homogeneous-fields):** `go/pkg/types/races.go:4-13` for the struct+slice field shape; `go/pkg/combat/combat.go:33-48` for the "struct holds the moving parts, methods operate on it" convention.
```go
// TraitSet holds composed traits as homogeneous per-kind slices (D-04).
type TraitSet struct {
	Vulnerabilities []Vulnerability
	Resistances     []Resistance
	Immunities      []Immunity
	Modifiers       []StatModifier
	Capabilities    []Capability
	Hooks           []BehaviorHook
	// resolved caches (populated by Resolve()):
	caps    CapBits          // OR of Capabilities (D-06)
	risSum  map[types.DamageType]int // summed magnitude per axis
	modSum  [types.MaxStats]int      // summed stat deltas
}
```
**What to copy:** merge = append per slice (no `[]Trait` interface, no type assertions — D-04). A `Merge(other TraitSet)` / `Compose(sets ...TraitSet)` appends in deterministic source order (race → class → ...) so output is reproducible.

---

### `go/pkg/traits/resolve.go` (`Resolve()` — slices → bitset + summed caches)

**Analog (table → derived-value transform):** `go/pkg/types/constants.go:289-300` (the `ApplyType.String()` table-walk) and the RIS clamp semantics from `go/pkg/combat/combat.go:308-316`.

**What to copy / build:**
- Iterate `Capabilities`, intern each `Key` via the registry, OR its bit into `CapBits` (D-06).
- Sum `Vulnerabilities`/`Resistances`/`Immunities` magnitude per `types.DamageType`, then clamp to `[-CAP, +CAP]` where `CAP` is a package const (default `100`, D-03):
```go
const CAP = 100 // D-03: per-axis magnitude clamp

func clamp(v int) int {
	if v > CAP { return CAP }
	if v < -CAP { return -CAP }
	return v
}
```
- Sum `Modifiers` per stat index into `[types.MaxStats]int` (planner sets a sane per-stat cap — Claude's Discretion).
- Deterministic iteration order (Claude's Discretion: merge determinism) — iterate slices in stored order; if iterating a map for output, sort keys.

---

### `go/pkg/traits/query.go` (query API — the parity-critical bridge)

**Analog (the tri-state contract `ResolveImmunity` must reproduce — D-01):** `go/pkg/combat/combat.go:308-373`
```go
// CheckImmune returns the immunity status for a damage type
type ImmunityResult int

const (
	ImmNormal ImmunityResult = iota
	ImmImmune
	ImmResistant
	ImmVulnerable
)

func CheckImmune(victim *types.Character, damType types.DamageType) ImmunityResult {
	...
	if victim.Imm.Has(immFlag)  { return ImmImmune }
	if victim.Res.Has(resFlag)  { return ImmResistant }
	if victim.Vuln.Has(vulnFlag){ return ImmVulnerable }
	return ImmNormal
}
```
**What to copy:** the **four-value tri-state output and its precedence** (Immune > Resist > Vuln > Normal). `ResolveImmunity(axis)` reads the summed/clamped magnitude from `Resolve()` and maps it back to this vocabulary per D-03 thresholds:
```
sum >= +CAP        -> Immune
+1 .. +CAP-1       -> Resist
sum  < 0           -> Vuln
sum == 0           -> Normal
```
**Parity note (most parity-sensitive piece, per specifics):** do **not** import `pkg/combat` (avoid a future cycle). Either (a) define `traits.ImmunityResult` mirroring the combat enum's order/values exactly, or (b) return an int the combat wiring (P7) translates. The planner must call out reproducing current resist=halve / vuln=double / immune=zero behavior — but **verification is P7/P8, not this phase** (D-07).

**Other query methods (all read-only, mirror `Has` value-receiver style from flags.go):**
- `HasTrait(kind TraitKind) bool` — scans the relevant per-kind slice non-empty.
- `HasCapability(key string) bool` — intern `key` → `CapBits.Has(bit)`, O(1) zero-alloc (SC#4).
- `GetModifier(stat types.Stat) int` — read `modSum[stat]`.
- `HooksFor(event HookEvent) []BehaviorHook` — filter `Hooks` by event in deterministic source order.

---

### `go/pkg/traits/traits_test.go` (table-driven unit tests)

**Analog:** `go/pkg/types/flags_test.go:1-80` (subtest + `Has`/`Set`/`Remove` assertions) and `go/pkg/combat/combat_test.go:1-27` (range-loop range checks).

**What to copy:**
- `package traits` (in-package white-box test — matches `package types` in flags_test.go), `import "testing"`.
- `func TestXxx(t *testing.T)` with `t.Run("description", func(t *testing.T){ ... })` subtests.
- Assertion style: `if got != want { t.Errorf("Foo() = %d, expected %d", got, want) }` — no testify needed (none of the existing type/combat unit tests use it).
```go
func TestHasCapability(t *testing.T) {
	t.Run("set capability is present", func(t *testing.T) {
		ts := TraitSet{Capabilities: []Capability{{Key: "flight"}}}
		ts.Resolve()
		if !ts.HasCapability("flight") {
			t.Error("expected flight capability present after Resolve")
		}
	})
}
```
**Required coverage (per phase boundary / SC):** additive RIS stacking + cap clamp, tri-state mapping at each threshold (`>=CAP`, mid, `<0`, `0`), capability O(1) presence/absence, `GetModifier` summing, `HooksFor` ordering, merge determinism.

## Shared Patterns

### Closed enum + `iota` (apply to: `TraitKind`, `HookEvent`)
**Source:** `go/pkg/types/constants.go:302-327`, `go/pkg/types/flags.go:360-382`
Named `int` type, `const (... Type = iota)` block, one inline `//` comment per value, plus a `String()` method using a `names []string` slice indexed by the enum value (`constants.go:289-300`).

### `Has`/`Set` receiver methods (apply to: `CapBits`)
**Source:** `go/pkg/types/flags.go:384-397`
Value receiver for read (`Has`), pointer receiver for mutation (`Set`/`Remove`). One-line doc comment per method.

### Tri-state RIS output (apply to: `ResolveImmunity`)
**Source:** `go/pkg/combat/combat.go:308-316`
Four-value enum `Normal/Immune/Resistant/Vulnerable` with precedence Immune > Resist > Vuln > Normal. The traits package mirrors the *vocabulary and order* without importing `pkg/combat`.

### Table-driven subtests (apply to: all `_test.go`)
**Source:** `go/pkg/types/flags_test.go:1-53`, `go/pkg/combat/combat_test.go:9-27`
`package traits` white-box, stdlib `testing` only, `t.Run` subtests, `t.Errorf` with format strings. No testify in unit tests.

### Go conventions (apply to: all files)
**Source:** `./CLAUDE.md` (Conventions)
Tabs; opening brace same line; full import paths (`"rotmud/pkg/types"`); PascalCase exported / camelCase private; constructor `NewXxx`; getter `GetXxx`; boolean `HasXxx`/`IsXxx`; `(value, error)` only for fallible ops (most trait queries are pure → single return). Optional `// Based on ...` references not applicable (no C source for this new system).

## No Analog Found

| File / Component | Role | Data Flow | Reason |
|------------------|------|-----------|--------|
| `go/pkg/traits/capability.go` — string→bit **registry** (intern table) | registry | transform | No global string-interning registry exists in the codebase. Closest hint is `map[string]int` at `go/pkg/skills/system.go` (per-character `Learned`), which is state, not an intern table. Planner should design from D-05/D-06 directly: package-level `map[string]int` + deterministic bit assignment + 256-bit ceiling guard. The **bitset half** (`CapBits`) has an exact analog in `ImmFlags` (flags.go:357). |

## Metadata

**Analog search scope:** `go/pkg/types/` (constants.go, flags.go, races.go, classes.go, *_test.go), `go/pkg/combat/` (combat.go, doc.go, combat_test.go), `go/pkg/skills/system.go`
**Files scanned:** ~10
**Pattern extraction date:** 2026-06-01
**Target package (new, does not yet exist):** `go/pkg/traits/`
