package loader

import "rotmud/pkg/types"

// Mob difficulty floor for levels 60-100. High-level world mobs were weaker
// than best-in-slot players could ever meet (combat_sim_world_test): the
// toughest reachable L100 mob had 4750 hp and 41 damage a hit. From level 60
// every mob is raised to at least this curve at spawn; mobs already above it
// keep their own values.
const mobFloorLevel = 60

type mobFloor struct{ hp, dam, hitroll, ac int }

// Anchors at L60 and L100; linear in between. The L100 anchor matches the
// toughest reachable L100 mob before the floor, so every L60-100 mob is at
// least that tough. Raising it further only punishes lightly geared players:
// best-in-slot gear still wins (see combat_sim_world_test).
var mobFloorLo = mobFloor{hp: 1000, dam: 25, hitroll: 12, ac: -110}
var mobFloorHi = mobFloor{hp: 4500, dam: 42, hitroll: 16, ac: -220}

func mobFloorAt(level int) mobFloor {
	level = min(level, 100)
	lerp := func(lo, hi int) int { return lo + (hi-lo)*(level-mobFloorLevel)/(100-mobFloorLevel) }
	return mobFloor{
		hp:      lerp(mobFloorLo.hp, mobFloorHi.hp),
		dam:     lerp(mobFloorLo.dam, mobFloorHi.dam),
		hitroll: lerp(mobFloorLo.hitroll, mobFloorHi.hitroll),
		ac:      lerp(mobFloorLo.ac, mobFloorHi.ac),
	}
}

// applyMobFloor raises a level 60+ mob to the difficulty floor.
func applyMobFloor(ch *types.Character) {
	if ch.Level < mobFloorLevel {
		return
	}
	f := mobFloorAt(ch.Level)
	ch.MaxHit = max(ch.MaxHit, f.hp)
	ch.Hit = ch.MaxHit
	if avg := ch.Damage[0]*(ch.Damage[1]+1)/2 + ch.Damage[2]; avg < f.dam {
		ch.Damage[2] += f.dam - avg // raise the bonus; keep the dice
	}
	ch.HitRoll = max(ch.HitRoll, f.hitroll)
	for i := range ch.Armor {
		ch.Armor[i] = min(ch.Armor[i], f.ac)
	}
}
