# Phase 1: Golden-Master Safety Net - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-17
**Phase:** 01-golden-master-safety-net
**Areas discussed:** Snapshot format, RNG seeding, Coverage scope, Fixture vs sim

---

## Snapshot Format

| Option | Description | Selected |
|--------|-------------|----------|
| Checked-in `.golden` text file | Output compared against committed testdata file; `go test -update` regenerates; clean git diffs | ✓ |
| Inline expected values in Go test code | Hardcoded as Go structs/maps; regression shows as test failure with before/after | |
| Hash/checksum | SHA256 of output; compact but diff is useless ("hash changed") | |

**User's choice:** A — Checked-in `.golden` text file
**Notes:** None

---

## RNG Seeding

| Option | Description | Selected |
|--------|-------------|----------|
| Inject `rand.Source` into `CombatSystem` | Add `Rand *rand.Rand` field; fixture injects fixed seed; parallel-safe; small API change | ✓ |
| Seed global rand at test startup | `rand.Seed(42)` at top of golden test; no API change; fragile with parallelism | |
| Statistical fixture with large N | No seeding; snapshot win% to 1 decimal; weak parity signal | |

**User's choice:** 1 — Inject `rand.Source` into `CombatSystem`
**Notes:** None

---

## Coverage Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Representative samples (33 combos) | All 19 races × warrior + all 14 classes × human; fast CI | ✓ |
| Full 19×14 matrix (266 combos) | Exhaustive; large .golden file; slow | |
| Combat API only | Approximated spell/skill effects as in combat_sim_test.go | |
| Real `pkg/magic` + `pkg/skills` calls | Actual CastSpell() and CheckSkill() paths; requires pkg/golden/ to avoid cycles | ✓ |

**User's choice:** A + D — Representative samples AND real magic/skills calls
**Notes:** None

---

## Fixture vs Sim

| Option | Description | Selected |
|--------|-------------|----------|
| Dedicated `pkg/golden/` package | Separate package; avoids import cycles (magic imports combat); cleanest boundary | ✓ |
| Separate `golden_test.go` in `pkg/combat/` | New file, no import cycle solution; wouldn't work with real magic/skills calls | |
| Extend `combat_sim_test.go` | Fewer files; mixes concerns; import cycle problem with real magic/skills | |

**User's choice:** C — Dedicated `pkg/golden/` package
**Notes:** Chosen partly because real magic/skills calls (D-04) require it to avoid the combat←magic import cycle

---

## Claude's Discretion

- Internal structure of the `.golden` file (table vs. per-combo blocks)
- Whether to include mob AI behavior coverage or defer
- Exact fixture runner architecture

## Deferred Ideas

- Full 19×14 matrix — deferred to Phase 8 when all combos are actively migrated
- Mob AI coverage (`pkg/ai/`) — complex wiring, defer unless it fits cleanly
