package loader

import "rotmud/pkg/types"

// Mob difficulty floor for levels 60-100. High-level world mobs were weaker
// than best-in-slot players could ever meet (combat_sim_world_test): the
// toughest reachable L100 mob had 4750 hp and 41 damage a hit. From level 60
// every mob is raised to at least this curve at spawn; mobs already above it
// keep their own values.
const mobFloorLevel = 60

type mobFloor struct{ dam, hitroll, ac int }

// Anchors at L60 and L100; linear in between. The L100 anchor matches the
// toughest reachable L100 mob before the floor, so every L60-100 mob is at
// least that tough. Raising it further only punishes lightly geared players:
// best-in-slot gear still wins (see combat_sim_world_test).
var mobFloorLo = mobFloor{dam: 25, hitroll: 12, ac: -110}
var mobFloorHi = mobFloor{dam: 42, hitroll: 16, ac: -220}

func mobFloorAt(level int) mobFloor {
	level = min(level, 100)
	lerp := func(lo, hi int) int { return lo + (hi-lo)*(level-mobFloorLevel)/(100-mobFloorLevel) }
	return mobFloor{
		dam:     lerp(mobFloorLo.dam, mobFloorHi.dam),
		hitroll: lerp(mobFloorLo.hitroll, mobFloorHi.hitroll),
		ac:      lerp(mobFloorLo.ac, mobFloorHi.ac),
	}
}

// applyMobFloor raises a mob to the HP floor for its level and, from level
// 60, to the damage/hitroll/AC floor.
func applyMobFloor(ch *types.Character) {
	ch.MaxHit = max(ch.MaxHit, mobHPFloorAt(ch.Level))
	ch.Hit = ch.MaxHit
	if ch.Level < mobFloorLevel {
		return
	}
	f := mobFloorAt(ch.Level)
	if avg := ch.Damage[0]*(ch.Damage[1]+1)/2 + ch.Damage[2]; avg < f.dam {
		ch.Damage[2] += f.dam - avg // raise the bonus; keep the dice
	}
	ch.HitRoll = max(ch.HitRoll, f.hitroll)
	for i := range ch.Armor {
		ch.Armor[i] = min(ch.Armor[i], f.ac)
	}
}

// mobHPFloor is the minimum mob HP at anchor levels, linear in between.
// World mobs died in 1-13 s against geared players at L10-L60, far under the
// 15-22 s fight target. The anchors are the median reachable mob's HP times a
// multiplier fitted with the world combat sim so best-in-slot fights against
// a median mob last about 15 s; from L60 they join the difficulty floor.
// It is a floor, not a multiplier, so mobs that were already tough keep their
// HP instead of compounding.
var mobHPFloor = [][2]int{{1, 16}, {5, 40}, {10, 264}, {20, 762}, {30, 1100}, {40, 1380}, {50, 2012}, {60, 2222}, {100, 4500}}

func mobHPFloorAt(level int) int {
	if level <= mobHPFloor[0][0] {
		return mobHPFloor[0][1]
	}
	for i := 1; i < len(mobHPFloor); i++ {
		lo, hi := mobHPFloor[i-1], mobHPFloor[i]
		if level <= hi[0] {
			return lo[1] + (hi[1]-lo[1])*(level-lo[0])/(hi[0]-lo[0])
		}
	}
	return mobHPFloor[len(mobHPFloor)-1][1]
}
