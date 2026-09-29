package combatsim

// Combat sim regression assertions.
//
// These tests pin the current behaviour of the simulations in sim_test.go
// and fail on drift outside snapshot tolerances. They also add a new mob variant
// that starts with the sanctuary affect — modelling a buffed mob that casters
// can dispel and melee classes cannot, then asserting per-class outcomes.
//
// Tolerances (per-cell, snapshot-relative):
//   - Win%:  ±10 percentage points
//   - Rnds:  ±50% relative (floor 5 rounds)
//   - P/M DPS ratio: ±60% relative (skipped if either DPS < 1; absolute cap 20×)
//   - Cross-class Win% spread: ±15 percentage points
//
// Snapshots captured at N=1000 with default global rand source; the standard
// error at p=0.5 is ~1.5pp, so ±10pp easily absorbs run-to-run noise.
//
// Run:
//   go test ./pkg/combatsim -run TestCombatSimAssert -v
//   go test ./pkg/combatsim -run TestCombatSimVsSanctuaryMob -v

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// ── snapshot tables ──────────────────────────────────────────────────────────

// simSnapCell holds expected outcome metrics for one (class, level) cell.
type simSnapCell struct {
	win    float64 // expected Win%
	rounds float64 // expected avg rounds
	pDPS   float64 // expected player DPS
	mDPS   float64 // expected mob DPS
}

// Level columns; cell slices below are indexed in this order.
var simSnapLevels = []int{1, 10, 20, 30, 40, 50, 60, 75, 100}

// simSnapClasses is the row order for the snapshot tables.
var simSnapClasses = []int{
	types.ClassWarrior,
	types.ClassRanger,
	types.ClassThief,
	types.ClassCleric,
	types.ClassDruid,
	types.ClassGhoul,
	types.ClassMage,
}

// ── tolerances ───────────────────────────────────────────────────────────────

const (
	snapWinTolPP    = 10.0 // Win% ±10 percentage points
	snapRoundsRel   = 0.50 // rounds ±50% relative
	snapRoundsAbs   = 5.0  // floor: at least 5 rounds slack
	snapRatioRel    = 0.60 // P/M DPS ratio ±60% relative
	snapDPSNoise    = 1.0  // skip ratio check if either DPS below this
	snapDPSRatioCap = 20.0 // absolute cap: catches runaway (e.g. mage one-shotting)
	snapSpreadPP    = 15.0 // cross-class spread ±15pp
)

// ── helpers ──────────────────────────────────────────────────────────────────

func snapAbs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func snapAssertWin(t *testing.T, label string, got, want float64) {
	t.Helper()
	if snapAbs(got-want) > snapWinTolPP {
		t.Errorf("%s: Win%% = %.1f, want %.1f ±%.0fpp", label, got, want, snapWinTolPP)
	}
}

func snapAssertRounds(t *testing.T, label string, got, want float64) {
	t.Helper()
	tol := want * snapRoundsRel
	if tol < snapRoundsAbs {
		tol = snapRoundsAbs
	}
	if snapAbs(got-want) > tol {
		t.Errorf("%s: Rounds = %.1f, want %.1f ±%.1f", label, got, want, tol)
	}
}

func snapAssertRatio(t *testing.T, label string, gotP, gotM, wantP, wantM float64) {
	t.Helper()
	if gotM < snapDPSNoise || wantM < snapDPSNoise {
		return // noise floor — ratio meaningless
	}
	got := gotP / gotM
	if got > snapDPSRatioCap {
		t.Errorf("%s: P/M DPS ratio = %.2f exceeds absolute cap %.1f", label, got, snapDPSRatioCap)
	}
	if wantP < snapDPSNoise {
		return
	}
	want := wantP / wantM
	tol := want * snapRatioRel
	if snapAbs(got-want) > tol {
		t.Errorf("%s: P/M DPS ratio = %.2f, want %.2f ±%.2f", label, got, want, tol)
	}
}

// runForSnap runs N fights and returns the simResult for snapshot comparison.
type simRunner func(classIdx, raceIdx, level, n int) simResult

func snapCheckTable(t *testing.T, raceIdx, n int, snap map[int][]simSnapCell, run simRunner) {
	t.Helper()
	for _, ci := range simSnapClasses {
		cells, ok := snap[ci]
		if !ok {
			t.Fatalf("missing snapshot for class index %d", ci)
		}
		for i, lv := range simSnapLevels {
			cell := cells[i]
			name := fmt.Sprintf("%s/L%d", types.ClassTable[ci].Name, lv)
			t.Run(name, func(t *testing.T) {
				r := run(ci, raceIdx, lv, n)
				snapAssertWin(t, name, r.winPct(), cell.win)
				snapAssertRounds(t, name, r.avgRounds(), cell.rounds)
				snapAssertRatio(t, name, r.pDPS(), r.mDPS(), cell.pDPS, cell.mDPS)
			})
		}
	}
}

// ── tests: regression-gate against the existing sim outcomes ─────────────────

// TestCombatSimAssertVsWarriorMob locks the per-class/level outcomes of
// TestCombatSimByClass. A failure here means a balance change moved a cell
// outside its snapshot band; update the snapshot if the change was intentional.
func TestCombatSimAssertVsWarriorMob(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	snapCheckTable(t, raceIdx, n, simSnapWarriorMob, runSim)
}

// TestCombatSimAssertVsCasterMob locks the per-class/level outcomes of
// TestCombatSimVsCasterMob.
func TestCombatSimAssertVsCasterMob(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	run := func(ci, ri, lv, n int) simResult {
		return runSimWith(ci, ri, lv, n, makeCasterMob)
	}
	snapCheckTable(t, raceIdx, n, simSnapCasterMob, run)
}

// TestCombatSimAssertSpread checks the max-min Win% spread across tier-1
// classes at each level matches the snapshot within ±15pp. Catches changes
// that uniformly shift one class far above or below the rest.
func TestCombatSimAssertSpread(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	for _, lv := range simSnapLevels {
		want := simSnapWarriorMobSpread[lv]
		t.Run(fmt.Sprintf("L%d", lv), func(t *testing.T) {
			lo, hi := 100.0, 0.0
			for _, ci := range simSnapClasses {
				r := runSim(ci, raceIdx, lv, n)
				w := r.winPct()
				if w < lo {
					lo = w
				}
				if w > hi {
					hi = w
				}
			}
			got := hi - lo
			if snapAbs(got-want) > snapSpreadPP {
				t.Errorf("L%d cross-class spread = %.0fpp, want %.0fpp ±%.0fpp",
					lv, got, want, snapSpreadPP)
			}
		})
	}
}

// ── snapshot regeneration ────────────────────────────────────────────────────

var updateSnap = flag.Bool("update", false, "rewrite sim_snapshot_test.go from the current sim")

// TestCombatSimSnapshotUpdate regenerates the snapshot tables after an
// intentional balance change:
//
//	go test ./pkg/combatsim -run TestCombatSimSnapshotUpdate -update
func TestCombatSimSnapshotUpdate(t *testing.T) {
	if !*updateSnap {
		t.Skip("run with -update to regenerate the sim snapshots")
	}
	const n = 1000
	var b strings.Builder
	b.WriteString("// Code generated by TestCombatSimSnapshotUpdate -update. DO NOT EDIT.\n\npackage combatsim\n")
	table := func(name, doc string, mobFn func(int) *types.Character) {
		fmt.Fprintf(&b, "\n// %s\nvar %s = map[int][]simSnapCell{\n", doc, name)
		for _, ci := range simSnapClasses {
			fmt.Fprintf(&b, "\t%d: { // %s\n", ci, types.ClassTable[ci].Name)
			for _, lv := range simSnapLevels {
				r := runSimWith(ci, types.RaceHuman, lv, n, mobFn)
				fmt.Fprintf(&b, "\t\t{%.0f, %.1f, %.1f, %.1f},\n", r.winPct(), r.avgRounds(), r.pDPS(), r.mDPS())
			}
			b.WriteString("\t},\n")
		}
		b.WriteString("}\n")
	}
	table("simSnapWarriorMob", "Warrior mob (TestCombatSimByClass), human race, N=1000.", makeMob)
	table("simSnapCasterMob", "Caster mob (TestCombatSimVsCasterMob), human race, N=1000.", makeCasterMob)
	table("simSnapSanctuaryMob", "Sanctuary mob (TestCombatSimVsSanctuaryMob), human race, N=1000.", makeSanctuaryMob)
	b.WriteString("\n// Cross-class Win% spread (max - min) vs the warrior mob.\nvar simSnapWarriorMobSpread = map[int]float64{\n")
	for _, lv := range simSnapLevels {
		lo, hi := 100.0, 0.0
		for _, ci := range simSnapClasses {
			r := runSim(ci, types.RaceHuman, lv, n)
			w := r.winPct()
			lo, hi = min(lo, w), max(hi, w)
		}
		fmt.Fprintf(&b, "\t%d: %.0f,\n", lv, hi-lo)
	}
	b.WriteString("}\n")
	src, err := format.Source([]byte(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("sim_snapshot_test.go", src, 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── sanctuary mob variant ────────────────────────────────────────────────────

// makeSanctuaryMob returns a standard warrior mob that starts the fight with
// the sanctuary affect active (halves all incoming damage). Stays buffed until
// a player dispels it.
func makeSanctuaryMob(level int) *types.Character {
	m := makeMob(level)
	m.AffectedBy.Set(types.AffSanctuary)
	return m
}

// TestCombatSimVsSanctuaryMob runs the sanctuary-mob variant for all tier-1
// classes at the standard level grid. It logs the results table and, if a
// snapshot exists for a cell, asserts against it. The very first run should be
// used to populate simSnapSanctuaryMob; subsequent runs gate regressions.
//
// Melee classes eat halved damage all fight; casters that know dispel magic
// cast it until the sanctuary is gone (25% per try at equal level).
//
// Run:  go test ./pkg/combatsim -run TestCombatSimVsSanctuaryMob -v
func TestCombatSimVsSanctuaryMob(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000

	t.Log("")
	t.Log("=== TIER-1 CLASSES vs equal-level mob with SANCTUARY  (human race, N=1000) ===")
	t.Log("Mob enters fight with sanctuary (halves all incoming damage).")
	t.Log("Caster classes attempt dispel each round (15 mana, 50% at equal level).")
	t.Log("Melee classes have no dispel — sanctuary persists the whole fight.")
	t.Log("")

	printTable := func(title string, cellFn func(r simResult) string) {
		hdr := fmt.Sprintf("%-10s", title)
		for _, lv := range simSnapLevels {
			hdr += fmt.Sprintf("  Lv%-3d ", lv)
		}
		t.Log(hdr)
		t.Log(strings.Repeat("-", len(hdr)))
		for _, ci := range simSnapClasses {
			row := fmt.Sprintf("%-10s", types.ClassTable[ci].Name)
			for _, lv := range simSnapLevels {
				r := runSimWith(ci, raceIdx, lv, n, makeSanctuaryMob)
				row += fmt.Sprintf("  %-6s", cellFn(r))
			}
			t.Log(row)
		}
		t.Log("")
	}

	printTable("Win%", func(r simResult) string { return fmt.Sprintf("%4.0f%%", r.winPct()) })
	printTable("Rounds", func(r simResult) string { return fmt.Sprintf("%5.1f", r.avgRounds()) })
	printTable("P-DPS", func(r simResult) string { return fmt.Sprintf("%5.1f", r.pDPS()) })
	printTable("M-DPS", func(r simResult) string { return fmt.Sprintf("%5.1f", r.mDPS()) })

	// Assertion phase — runs only for cells with snapshot entries.
	if len(simSnapSanctuaryMob) == 0 {
		t.Log("simSnapSanctuaryMob is empty — populate it from the tables above to gate regressions.")
		return
	}
	for _, ci := range simSnapClasses {
		cells, ok := simSnapSanctuaryMob[ci]
		if !ok {
			continue
		}
		for i, lv := range simSnapLevels {
			cell := cells[i]
			name := fmt.Sprintf("%s/L%d", types.ClassTable[ci].Name, lv)
			t.Run(name, func(t *testing.T) {
				r := runSimWith(ci, raceIdx, lv, n, makeSanctuaryMob)
				snapAssertWin(t, name, r.winPct(), cell.win)
				snapAssertRounds(t, name, r.avgRounds(), cell.rounds)
				snapAssertRatio(t, name, r.pDPS(), r.mDPS(), cell.pDPS, cell.mDPS)
			})
		}
	}
}
