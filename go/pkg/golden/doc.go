// Package golden contains the deterministic entity-behavior snapshot
// fixture (the "golden master") used as the ROT-MUD trait-migration
// parity gate.
//
// # What This Package Is
//
// A test-only package. It exercises real [combat.CombatSystem],
// [magic.MagicSystem], and [skills.SkillSystem] APIs against a
// representative sample of the 19 races x 14 classes entity matrix plus
// representative spells and skills, captures the output of every call
// into a bytes.Buffer, and either writes that buffer to
// testdata/entities.golden (when run with -update) or diffs against the
// committed copy. Any behavioral drift — in races, classes, spells,
// skills, or their interactions — shows up as a readable `git diff`
// against testdata/entities.golden.
//
// Location rationale (D-05): this package lives at pkg/golden/ rather
// than inside pkg/combat/ because pkg/magic imports pkg/combat, so a
// fixture in pkg/combat that imports pkg/magic would create an import
// cycle. The standalone pkg/golden/ sits above all three and imports
// them cleanly.
//
// # Seed
//
// The fixture pins math/rand.Rand to rand.NewSource(42) via
// [combat.SetRand]. Seed 42 is chosen by fiat; do not change it
// lightly — every committed line in testdata/entities.golden was
// produced under this seed. A seed change invalidates the entire
// snapshot in one step and provides no benefit.
//
// # Usage
//
//	# Run the parity check (default, for CI):
//	go test ./pkg/golden/ -run TestGolden
//
//	# Regenerate the snapshot after an intentional behavior change:
//	go test ./pkg/golden/ -run TestGolden -update
//
//	# Determinism check (fixture must produce identical bytes twice):
//	go test ./pkg/golden/ -run TestGolden -count=2
//
// # Coverage Scope (D-03)
//
//   - All 19 races x warrior class at level 20.
//   - All 14 classes x human race at level 20.
//   - Representative damage / affect / heal spells through Spell.Func.
//   - Representative skills: backstab, kick, dodge, parry.
//
// Mob aggro / assist / AI behavior is explicitly deferred to a later
// phase (Research Open Question #2) — only mob stat/immunity parity is
// exercised here via makeMob.
//
// # Do Not
//
//   - Reuse a *types.Character across scenarios (ROM death handling
//     strips equipment and mutates armor; fresh builders per
//     iteration).
//   - Emit map-ranged data without sorting keys (Go map iteration is
//     randomized).
//   - Include timestamps, wall-clock durations, or pointer addresses
//     in the output.
package golden
