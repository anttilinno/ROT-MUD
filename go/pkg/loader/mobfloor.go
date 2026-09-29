package loader

import (
	"rotmud/pkg/skills"
	"rotmud/pkg/types"
)

// Mob floors and ceilings, applied at spawn. Mobs keep their own values when
// they are inside the band. Every table is {level, value} anchors, linear in
// between, fitted with the world combat sim (pkg/combatsim, real game code,
// best-in-slot players who buff before a fight). The goal is a reaction
// window: damage arrives steadily enough that a player sees trouble coming
// and has time to flee or heal, rather than rare spikes.

// mobHitrollFloor: world mobs landed only 6-20% of their swings on geared
// players, so their damage came as rare spikes. From L30 it is fitted so
// about half of a mid-level mob's swings land on a pre-buffed mage; L10/L20
// are lower because thieves, ghouls and mages there fight slowly and lost
// most fights at that accuracy.
var mobHitrollFloor = [][2]int{{1, 0}, {10, 4}, {20, 8}, {30, 11}, {40, 11}, {50, 13}, {60, 21}, {75, 21}, {100, 23}}

// mobDamFloor and mobDamCeiling bound a mob's damage a round (average hit
// times expected swings, before hit chance). The floor is fitted so a
// mid-level mob would need about 30 s to kill a fragile class (mage, ghoul)
// from full, twice their fight length, so they finish around half HP; the
// ceiling so the toughest mob of a level needs at least 12 s to kill any
// class from full. Tanks (warriors, sanctuary casters) finish higher; that
// is the chosen trade-off. Every table rises with level.
var mobDamFloor = [][2]int{{1, 2}, {10, 2}, {20, 3}, {30, 4}, {40, 17}, {50, 34}, {60, 51}, {75, 51}, {100, 104}}
var mobDamCeiling = [][2]int{{1, 5}, {10, 6}, {20, 8}, {30, 12}, {40, 51}, {50, 66}, {60, 70}, {75, 153}, {100, 312}}

// Mob AC floor from level 60: high-level world mobs had weaker armour than
// best-in-slot players could ever meet.
const mobACFloorLevel = 60

var mobACFloor = [][2]int{{60, -110}, {100, -220}}

// applyMobFloor brings a freshly created mob inside the floors and ceilings.
func applyMobFloor(ch *types.Character) {
	ch.MaxHit = max(ch.MaxHit, anchorAt(mobHPFloor, ch.Level))
	ch.Hit = ch.MaxHit
	ch.HitRoll = max(ch.HitRoll, anchorAt(mobHitrollFloor, ch.Level))

	// Damage a round between the floor and the ceiling; the dice stay where
	// they can, the bonus moves.
	swings := mobSwings(ch)
	floor := float64(anchorAt(mobDamFloor, ch.Level))
	ceil := max(floor, float64(anchorAt(mobDamCeiling, ch.Level)))
	if perRound := float64(mobAvgDamage(ch)) * swings; perRound < floor {
		ch.Damage[2] += int(floor/swings+0.5) - mobAvgDamage(ch)
	} else if perRound > ceil {
		want := max(1, int(ceil/swings))
		for ch.Damage[0] > 1 && mobAvgDamage(ch)-ch.Damage[2] > want {
			ch.Damage[0]--
		}
		ch.Damage[2] += want - mobAvgDamage(ch)
	}

	if ch.Level >= mobACFloorLevel {
		for i := range ch.Armor {
			ch.Armor[i] = min(ch.Armor[i], anchorAt(mobACFloor, ch.Level))
		}
	}
}

var mobSkills = skills.NewSkillSystem()

// mobSwings is a mob's expected swings a round (ROT mob_hit): one, one more
// when fast or hasted, then second and third attack chained at skill/2 %.
func mobSwings(ch *types.Character) float64 {
	n := 1.0
	if ch.Off.Has(types.OffFast) || ch.IsAffected(types.AffHaste) {
		n++
	}
	p2 := float64(mobSkills.GetSkill(ch, "second attack")) / 200
	p3 := float64(mobSkills.GetSkill(ch, "third attack")) / 200
	return n + p2 + p2*p3
}

// mobAvgDamage is a mob's average damage a hit from its damage dice.
func mobAvgDamage(ch *types.Character) int {
	return ch.Damage[0]*(ch.Damage[1]+1)/2 + ch.Damage[2]
}

// mobHPFloor is the minimum mob HP at anchor levels, linear in between.
// L10 is lowered from the fit so slow low-level classes finish sooner.
// From L30 the anchors are fitted with the world combat sim (real game code,
// mob off flags) so a best-in-slot warrior needs about 20 s against the
// middle third of reachable mobs at that level; casters take about 11-15 s.
// L10 and L20 stay lower: fitting them to 20 s made low-level mages, thieves
// and ghouls lose to median mobs (they run out of mana or HP first). L100
// mobs are above the floor already. It is a floor, not a multiplier, so mobs
// that were already tough keep their HP instead of compounding.
var mobHPFloor = [][2]int{{1, 16}, {5, 40}, {10, 200}, {20, 762}, {30, 1680}, {40, 1680}, {50, 2436}, {60, 2580}, {75, 3291}, {100, 4500}}

// anchorAt interpolates a {level, value} anchor table linearly.
func anchorAt(anchors [][2]int, level int) int {
	if level <= anchors[0][0] {
		return anchors[0][1]
	}
	for i := 1; i < len(anchors); i++ {
		lo, hi := anchors[i-1], anchors[i]
		if level <= hi[0] {
			return lo[1] + (hi[1]-lo[1])*(level-lo[0])/(hi[0]-lo[0])
		}
	}
	return anchors[len(anchors)-1][1]
}
