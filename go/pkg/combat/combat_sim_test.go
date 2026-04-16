package combat

// Combat balance simulation — v2.
//
// Models real class/race tables with:
//   - Starter weapons that scale with level (class-appropriate type)
//   - Caster classes deal spell damage each round (best available by level)
//   - Class-based armor: warriors wear plate, mages wear robes
//   - Mob HP scales quadratically so fights last 20-30 rounds
//
// Timing reference (from game/loop.go):
//   PulsePerSecond = 4 (250ms/pulse), PulseViolence = 3
//   => 1 combat round = 0.75 seconds
//   => 20 rounds ≈ 15 seconds, 30 rounds ≈ 22 seconds
//
// Run individual tests:
//
//	go test ./pkg/combat/ -run TestCombatSimByClass  -v
//	go test ./pkg/combat/ -run TestCombatSimByRace   -v
//	go test ./pkg/combat/ -run TestCombatSimRaceSynergy -v
//	go test ./pkg/combat/ -run TestCombatSimDetailed -v

import (
	"fmt"
	"strings"
	"testing"

	"rotmud/pkg/types"
)

// ── timing constants ──────────────────────────────────────────────────────────

const (
	secondsPerRound = 0.75 // PulseViolence(3) × 250ms
)

func roundsToSeconds(r float64) float64 { return r * secondsPerRound }

// ── character / equipment builders ───────────────────────────────────────────

// raceStatAtLevel linearly interpolates from BaseStats to MaxStats,
// reaching MaxStats at level 15 and staying there.
func raceStatAtLevel(race *types.Race, stat, level int) int {
	base := race.BaseStats[stat]
	max := race.MaxStats[stat]
	if level >= 15 {
		return max
	}
	t := float64(level-1) / 14.0
	v := base + int(t*float64(max-base)+0.5)
	if v > max {
		v = max
	}
	return v
}

// playerHP calculates max HP at a given level using the class HP-per-level table.
// Base 20 HP at level 1; each level adds avg(HPMin,HPMax) + CON bonus.
func playerHP(cl *types.Class, race *types.Race, level int) int {
	avgGain := (cl.HPMin + cl.HPMax) / 2
	if avgGain < 1 {
		avgGain = 1
	}
	hp := 20 + (level-1)*avgGain
	// CON bonus: +1 HP per level for every 2 CON above 14
	con := raceStatAtLevel(race, types.StatCon, level)
	if con > 14 {
		hp += (level - 1) * (con - 14) / 2
	}
	return hp
}

// classEquipAC returns the raw Armor value for a class at a given level.
// Warriors wear the heaviest armour; mages wear robes.
// Raw value / 10 = effective AC used in THAC0 formula.
func classEquipAC(classIdx, level int) int {
	// Base curve: lightly armoured adventurer, improving with level.
	base := 80 - level*7
	switch classIdx {
	case types.ClassWarrior, types.ClassGladiator:
		base -= 25 // plate armour
	case types.ClassRanger, types.ClassStrider, types.ClassCleric, types.ClassPriest:
		base -= 10 // chain / mail
	case types.ClassThief, types.ClassMercenary:
		base += 10 // leather
	case types.ClassDruid, types.ClassSage:
		base += 15 // light leather
	case types.ClassVampire, types.ClassLich:
		base += 5 // supernatural resilience — undead flesh is harder to damage than leather
	default: // mage, wizard
		base += 35 // robes only
	}
	if base < -220 {
		base = -220
	}
	return base
}

// weaponDice returns (num, size) dice for a class-appropriate weapon at level.
// Represents gradually upgraded equipment over a character's career.
// Scales through L100 (legendary weapons); damage roughly 6× from L1 to L100.
func weaponDice(classIdx, level int) (int, int) {
	var num, size int
	switch {
	case level <= 5:
		num, size = 1, 6
	case level <= 10:
		num, size = 1, 8
	case level <= 15:
		num, size = 2, 6
	case level <= 20:
		num, size = 2, 8
	case level <= 25:
		num, size = 3, 6
	case level <= 30:
		num, size = 3, 8
	case level <= 35:
		num, size = 4, 6
	case level <= 40:
		num, size = 4, 8
	case level <= 45:
		num, size = 5, 6
	case level <= 50:
		num, size = 5, 8
	case level <= 55:
		num, size = 6, 6
	case level <= 60:
		num, size = 6, 8
	case level <= 65:
		num, size = 7, 6
	case level <= 70:
		num, size = 7, 8
	case level <= 75:
		num, size = 8, 6
	case level <= 80:
		num, size = 8, 8
	case level <= 85:
		num, size = 9, 6
	case level <= 90:
		num, size = 9, 8
	case level <= 95:
		num, size = 10, 6
	default: // L96+
		num, size = 10, 8
	}
	// Mages and vampire use lighter weapons (dagger/claw)
	switch classIdx {
	case types.ClassMage, types.ClassWizard, types.ClassVampire, types.ClassLich:
		size = size * 2 / 3
		if size < 4 {
			size = 4
		}
	}
	return num, size
}

// daggerDice returns (num, size) for the best dagger available at a given level.
// Daggers cap at 7d6 (avg 24.5) — roughly 55% of a top-tier sword (10d8 avg 45).
// Low/mid levels are closer to parity; the gap widens toward endgame.
// The backstab/circle multipliers compensate for the sustained DPS deficit.
func daggerDice(level int) (int, int) {
	switch {
	case level <= 10:
		return 1, 6 // avg 3.5
	case level <= 20:
		return 2, 5 // avg 6
	case level <= 30:
		return 3, 5 // avg 9
	case level <= 40:
		return 3, 7 // avg 12
	case level <= 50:
		return 4, 6 // avg 14
	case level <= 60:
		return 5, 5 // avg 15
	case level <= 75:
		return 5, 6 // avg 17.5
	case level <= 85:
		return 6, 5 // avg 18
	default: // L86+ — best-in-slot dagger cap
		return 7, 6 // avg 24.5
	}
}

// backstabMult returns the backstab damage multiplier at a given level.
// Higher levels unlock bigger multipliers to compensate for dagger vs sword DPS gap.
func backstabMult(level int) int {
	switch {
	case level >= 75:
		return 7
	case level >= 50:
		return 6
	case level >= 30:
		return 5
	case level >= 20:
		return 4
	case level >= 10:
		return 3
	default:
		return 2
	}
}

// simWeaponDice returns weapon dice for a class in the sim.
// Thieves use full weapon dice for regular combat (backstab, circle, melee).
// The dagger restriction only applies to assassinate (which uses daggerDice directly).
func simWeaponDice(classIdx, level int) (int, int) {
	return weaponDice(classIdx, level)
}

// weaponTypeForClass maps class to ROM weapon type (0=exotic,1=sword,2=dagger,3=spear,4=mace)
func weaponTypeForClass(classIdx int) int {
	switch classIdx {
	case types.ClassWarrior, types.ClassGladiator:
		return 1 // sword
	case types.ClassRanger, types.ClassStrider:
		return 3 // spear
	case types.ClassThief, types.ClassMercenary, types.ClassMage, types.ClassWizard,
		types.ClassVampire, types.ClassLich:
		return 2 // dagger
	case types.ClassCleric, types.ClassPriest:
		return 4 // mace
	default:
		return 0 // exotic / polearm
	}
}

// makeWeapon creates a level-appropriate weapon for a class and equips it.
func makeWeapon(classIdx, level int) *types.Object {
	num, size := simWeaponDice(classIdx, level)
	w := types.NewObject(1, "weapon", types.ItemTypeWeapon)
	w.Values[0] = weaponTypeForClass(classIdx)
	w.Values[1] = num
	w.Values[2] = size
	w.WearFlags.Set(types.WearWield)
	w.WearLoc = types.WearLocWield
	return w
}

// playerMana estimates starting mana for a class/race at a given level.
// Casters grow mana fast; non-casters have minimal mana.
func playerMana(classIdx int, race *types.Race, level int) int {
	cl := &types.ClassTable[classIdx]
	intStat := raceStatAtLevel(race, types.StatInt, level)
	wisStat := raceStatAtLevel(race, types.StatWis, level)
	base := 100 + intStat*3 + wisStat*2
	gain := cl.ManaGain
	if gain < 0 {
		gain = 0
	}
	base += level * gain * 2
	if base < 50 {
		base = 50
	}
	return base
}

// makePlayer builds a complete player character for combat simulation.
func makePlayer(classIdx, raceIdx, level int) *types.Character {
	cl := &types.ClassTable[classIdx]
	race := &types.RaceTable[raceIdx]

	ch := types.NewCharacter(cl.Name + "/" + race.Name)
	ch.Level = level
	ch.Class = classIdx
	ch.Race = raceIdx

	ch.MaxHit = playerHP(cl, race, level)
	ch.Hit = ch.MaxHit
	ch.MaxMana = playerMana(classIdx, race, level)
	ch.Mana = ch.MaxMana

	for stat := 0; stat < types.MaxStats; stat++ {
		ch.PermStats[stat] = raceStatAtLevel(race, stat, level)
	}

	ac := classEquipAC(classIdx, level)
	for i := range ch.Armor {
		ch.Armor[i] = ac
	}

	ch.HitRoll = level / 3
	ch.DamRoll = level / 4

	// Enhanced damage skill bonus (ROM: 'enhanced damage').
	// Rangers and thieves train this to compensate for lower base HP vs warrior.
	// Warriors already hit their win-rate target, so they don't need the boost here.
	// Modelled as +level/8 DamRoll; represents ~75% skill proficiency.
	// Second/third/etc. attacks are already gated by extraAttackSkillForClass.
	// Ranger/strider get DamRoll bonus — compensates for lighter armor vs warrior.
	// Thief/merc do NOT: backstab + circle provide their burst compensation.
	switch classIdx {
	case types.ClassRanger, types.ClassStrider:
		ch.DamRoll += level / 8
	}

	// Equip a class-appropriate weapon
	weapon := makeWeapon(classIdx, level)
	ch.Equip(weapon, types.WearLocWield)

	ch.Position = types.PosStanding
	return ch
}

// mobHP returns HP for a standard warrior-type mob at the given level.
// L1-L30:  bell-curve formula — quadratic with +33% bonus at L10 fading to 0 at L30.
// L31-L80: linear at 30/level from the L30 base (710).
// L81+:    steeper ramp at 40/level — endgame mobs scale harder.
//           Breakpoint at L80 so L75 balance is fully preserved; the steeper slope
//           only affects L81+ where player HP growth outpaces mob DPS and all classes
//           were winning 75-88% at L100 against only a 59% baseline from mage.
// Values: L10=200, L20=430, L30=710, L50=1310, L60=1610, L75=2060, L80=2210, L100=3010
func mobHP(level int) int {
	if level <= 30 {
		base := level*level/2 + level*8 + 20
		return base + level*(30-level)/4
	}
	if level <= 80 {
		return 710 + (level-30)*30
	}
	// Endgame: 40/level above L80 (was 30)
	return 2210 + (level-80)*40
}

// makeMob creates a standard warrior-class mob for the given level.
func makeMob(level int) *types.Character {
	mob := types.NewNPC(1, "Mob", level)
	mob.Act.Set(types.ActWarrior) // warrior-type THAC0 in GetThac0

	mob.MaxHit = mobHP(level)
	mob.Hit = mob.MaxHit

	mob.PermStats[types.StatStr] = 15 + level/6 // grows a bit with level
	mob.PermStats[types.StatDex] = 14

	// Mob AC: moderate, improves more slowly than player
	mobAC := 90 - level*5
	if mobAC < -150 {
		mobAC = -150
	}
	for i := range mob.Armor {
		mob.Armor[i] = mobAC
	}

	// Mob natural attacks (claws/bite): scale with level, capped to stay playable.
	numDice := 1 + level/6
	if numDice > 5 {
		numDice = 5
	}
	dieSize := 5 + level/5
	if dieSize > 12 {
		dieSize = 12
	}
	mob.Damage[0] = numDice
	mob.Damage[1] = dieSize
	mob.Damage[2] = level / 5
	mob.DamType = types.DamBash

	mob.HitRoll = level / 4
	mob.DamRoll = level / 3

	mob.Position = types.PosStanding
	return mob
}

// casterMobHP returns HP for a spell-casting mob.
// 75% of warrior mob HP — squishier, but not trivially killable.
// Players race to kill the caster before its spells whittle them down.
func casterMobHP(level int) int {
	return mobHP(level) * 3 / 4
}

// makeCasterMob creates a spell-casting mob (mage/shaman type) for the given level.
// Lower HP and melee than warrior mob; casts a spell each round that bypasses
// dodge and parry (direct HP hit, handled in the sim loop).
// Harder to hit with melee (ActMage THAC0) but has less HP.
func makeCasterMob(level int) *types.Character {
	mob := types.NewNPC(1, "Mob Mage", level)
	mob.Act.Set(types.ActMage) // mage-type THAC0 (harder to hit with melee)

	mob.MaxHit = casterMobHP(level)
	mob.Hit = mob.MaxHit

	mob.PermStats[types.StatStr] = 12
	mob.PermStats[types.StatDex] = 14
	mob.PermStats[types.StatInt] = 17 + level/10

	// Lighter armor than warrior mob — casters wear robes
	mobAC := 80 - level*3
	if mobAC < -80 {
		mobAC = -80
	}
	for i := range mob.Armor {
		mob.Armor[i] = mobAC
	}

	// Weak melee (staff/dagger, rarely used)
	mob.Damage[0] = 1
	mob.Damage[1] = 4
	mob.Damage[2] = level / 8
	mob.DamType = types.DamBash

	mob.HitRoll = level / 6
	mob.DamRoll = level / 6

	mob.Position = types.PosStanding
	return mob
}

// mobCastSpellDam returns direct spell damage dealt by a caster mob.
// Bypasses physical defence (dodge/parry); magic resistance would reduce this
// but is not yet simulated.
//   L10: avg  8.5  (1d6+lv/2 → 3.5+5)
//   L30: avg 39    (2d8+lv   → 9+30)
//   L60: avg 138   (4d8+lv*2 → 18+120)
//   L100: avg 218  (4d8+lv*2 → 18+200) — everyone loses without MR
func mobCastSpellDam(level int) int {
	switch {
	case level >= 75:
		return Dice(level/4, 8) + level*2 // arch-caster tier
	case level >= 60:
		return Dice(5, 8) + level*2 // elder caster tier
	case level >= 50:
		return Dice(4, 8) + level*2 // chain lightning / earthquake tier
	case level >= 40:
		return Dice(3, 8) + level*2 // greater fireball tier
	case level >= 22:
		return Dice(2, 8) + level   // fireball tier
	case level >= 13:
		return Dice(2, 6) + level   // lightning bolt tier
	default:
		return Dice(1, 6) + level/2 // magic missile tier
	}
}

// weaponSkillForClass returns weapon proficiency for a class at a level.
func weaponSkillForClass(classIdx, level int) int {
	var growth float64
	switch classIdx {
	case types.ClassWarrior, types.ClassGladiator:
		growth = 4.0
	case types.ClassRanger, types.ClassStrider:
		growth = 3.5
	case types.ClassThief, types.ClassMercenary:
		growth = 3.0
	case types.ClassCleric, types.ClassPriest, types.ClassDruid, types.ClassSage:
		growth = 2.5
	case types.ClassVampire, types.ClassLich:
		growth = 2.0
	default: // mage, wizard
		growth = 1.5
	}
	s := 25 + int(float64(level)*growth)
	if s > 100 {
		s = 100
	}
	return s
}

// dodgeSkillForClass returns dodge proficiency for a class at a level.
// Heavy-armor classes rely on AC, not agility — they get low dodge.
// Light-armor/agile classes compensate for weaker armor with better evasion.
func dodgeSkillForClass(classIdx, level int) int {
	var growth float64
	switch classIdx {
	case types.ClassThief, types.ClassMercenary:
		growth = 3.0 // light armor, highest dodge
	case types.ClassRanger, types.ClassStrider:
		growth = 2.5 // medium armor, good dodge
	case types.ClassMage, types.ClassWizard:
		growth = 2.0 // robes only, rely on evasion
	case types.ClassVampire, types.ClassLich:
		growth = 2.0 // supernatural agility
	case types.ClassDruid, types.ClassSage:
		growth = 1.5 // light leather, moderate
	case types.ClassCleric, types.ClassPriest:
		growth = 1.0 // chain mail restricts movement
	default: // warrior — plate armor limits agility but fighters learn to dodge over time
		growth = 2.0
	}
	s := 5 + int(float64(level)*growth)
	if s > 80 {
		s = 80
	}
	return s
}

// parrySkillForClass returns parry proficiency for a class at a level.
// Trained fighters parry well; casters parry poorly. Separate from weapon mastery
// to avoid over-stacking defense on top of heavy armor.
func parrySkillForClass(classIdx, level int) int {
	var growth float64
	switch classIdx {
	case types.ClassWarrior, types.ClassGladiator, types.ClassRanger, types.ClassStrider:
		growth = 2.5 // trained melee fighters parry well
	case types.ClassThief, types.ClassMercenary, types.ClassCleric, types.ClassPriest:
		growth = 2.0
	case types.ClassVampire, types.ClassLich:
		growth = 2.0 // unnatural reflexes
	case types.ClassDruid, types.ClassSage:
		growth = 1.5
	default: // mage — barely parries
		growth = 0.5
	}
	s := 5 + int(float64(level)*growth)
	if s > 80 {
		s = 80
	}
	return s
}

// extraAttackSkillForClass returns skill for extra attack tiers (tier 1 = second attack, etc.).
// Each tier caps lower so warriors can't spam 4-5 equal-rate attacks at high levels.
func extraAttackSkillForClass(classIdx, level, tier int) int {
	if tier < 1 || tier > 4 {
		return 0
	}
	// Base from weapon mastery, then cap per tier so later attacks are rarer.
	// Caps: second=90, third=70, fourth=50, fifth=30.
	caps := [5]int{0, 90, 70, 50, 30}
	s := weaponSkillForClass(classIdx, level)
	if s > caps[tier] {
		s = caps[tier]
	}
	return s
}

// spellManaCost returns the mana cost of the best available spell.
func spellManaCost(classIdx, level int) int {
	switch classIdx {
	case types.ClassMage, types.ClassWizard:
		if level >= 30 {
			return 20 // acid blast
		} else if level >= 22 {
			return 25 // fireball
		} else if level >= 13 {
			return 20 // lightning bolt
		}
		return 15 // magic missile
	case types.ClassCleric, types.ClassPriest:
		if level >= 45 {
			return 20
		} else if level >= 23 {
			return 17
		}
		return 15
	case types.ClassDruid, types.ClassSage:
		if level >= 30 {
			return 20
		} else if level >= 10 {
			return 15
		}
		return 10
	case types.ClassVampire, types.ClassLich:
		if level >= 21 {
			return 20
		} else if level >= 11 {
			return 17
		}
		return 15
	}
	return 0 // melee classes
}

// castSpellDamage returns the raw damage dealt by the caster's best offensive spell.
// Returns 0 for pure melee classes. Scales through level 100.
func castSpellDamage(classIdx, casterLevel int) int {
	switch classIdx {
	case types.ClassMage, types.ClassWizard:
		// magic missile → lightning bolt → fireball → acid blast (L30+)
		//
		// ROM C source uses level-scaled dice: magic missile = dice(level,4),
		// lightning bolt = dice(level,6), fireball = dice(level,6)+40, etc.
		// The original sim used flat low formulas (1d4+level, 2d6+level) which
		// gave mage only warrior-level DPS at L10-20 — a "wuss" despite being
		// the glass-cannon class. Level-scaled dice restore the proper cannon feel:
		// mage wins BECAUSE it one-shots the mob faster than the mob kills it,
		// not because it out-tanks the mob (which it can't at 172 HP vs 430 mob HP).
		//
		// Acid blast capped at L38 dice (was L35) to offset the steeper mob HP ramp
		// at L81+ while keeping L100 mage in the 55-65% window with sanctuary.
		if casterLevel >= 30 {
			dl := casterLevel
			if dl > 38 {
				dl = 38
			}
			return Dice(dl, 12) // acid blast, capped at L38 potency
		} else if casterLevel >= 22 {
			return Dice(casterLevel, 6) + 40 // fireball: ROM dice(level,6)+40
		} else if casterLevel >= 13 {
			return Dice(casterLevel, 6) + casterLevel // lightning bolt: ROM dice(level,6)+level
		}
		return Dice(casterLevel, 4) // magic missile: ROM dice(level,4)

	case types.ClassCleric, types.ClassPriest:
		// cause light → cause serious → cause critical → harm
		// No L75+ tier: harm (Dice(4,8)+level) is the ceiling.
		// With mob sanctuary at L70+ (÷2), win rate targets ~60-70%.
		if casterLevel >= 50 {
			return Dice(4, 8) + casterLevel // harm
		} else if casterLevel >= 40 {
			return Dice(3, 8) + casterLevel // cause critical
		} else if casterLevel >= 23 {
			return Dice(2, 8) + casterLevel/2
		}
		return Dice(1, 8) + casterLevel/3

	case types.ClassDruid, types.ClassSage:
		// faerie fire → call lightning → earthquake
		// No L75+ tier: earthquake (Dice(4,8)+level) is the ceiling.
		if casterLevel >= 50 {
			return Dice(4, 8) + casterLevel // earthquake tier
		} else if casterLevel >= 40 {
			return Dice(3, 8) + casterLevel // intermediate (closes L40 gap)
		} else if casterLevel >= 30 {
			return Dice(2, 8) + casterLevel
		} else if casterLevel >= 10 {
			return Dice(1, 8) + casterLevel/2
		}
		return Dice(1, 4) + casterLevel/4

	case types.ClassVampire, types.ClassLich:
		// cause spells + drain/soul-rend
		// No L75+ tier: drain (Dice(4,6)+level+level/3) is the ceiling.
		if casterLevel >= 50 {
			return Dice(4, 6) + casterLevel + casterLevel/3 // drain/soul-rend tier
		} else if casterLevel >= 21 {
			return Dice(3, 6) + casterLevel
		} else if casterLevel >= 11 {
			return Dice(2, 6) + casterLevel/2
		}
		return Dice(1, 6) + casterLevel/3
	}
	return 0 // warrior, ranger, thief — no spells
}

// isCasterClass returns true for classes that use spells in combat.
func isCasterClass(classIdx int) bool {
	return castSpellDamage(classIdx, 1) > 0 ||
		classIdx == types.ClassMage || classIdx == types.ClassWizard ||
		classIdx == types.ClassCleric || classIdx == types.ClassPriest ||
		classIdx == types.ClassDruid || classIdx == types.ClassSage ||
		classIdx == types.ClassVampire || classIdx == types.ClassLich
}

// ── simulation core ───────────────────────────────────────────────────────────

type simResult struct {
	n           int
	playerWins  int
	draws       int
	totalRounds int
	totalPDmg   int
	totalMDmg   int
	totalPHits  int
	totalPMiss  int
	totalMHits  int
	totalMMiss  int
}

func (r *simResult) winPct() float64 { return 100 * float64(r.playerWins) / float64(r.n) }
func (r *simResult) avgRounds() float64 {
	return float64(r.totalRounds) / float64(r.n)
}
func (r *simResult) avgSeconds() float64 { return roundsToSeconds(r.avgRounds()) }
func (r *simResult) pHitPct() float64 {
	t := r.totalPHits + r.totalPMiss
	if t == 0 {
		return 0
	}
	return 100 * float64(r.totalPHits) / float64(t)
}
func (r *simResult) mHitPct() float64 {
	t := r.totalMHits + r.totalMMiss
	if t == 0 {
		return 0
	}
	return 100 * float64(r.totalMHits) / float64(t)
}
func (r *simResult) pDPS() float64 {
	if r.totalRounds == 0 {
		return 0
	}
	return float64(r.totalPDmg) / float64(r.totalRounds)
}
func (r *simResult) mDPS() float64 {
	if r.totalRounds == 0 {
		return 0
	}
	return float64(r.totalMDmg) / float64(r.totalRounds)
}

// runSim runs n fights between a player (classIdx/raceIdx/level) and an equal-level warrior mob.
func runSim(classIdx, raceIdx, level, n int) simResult {
	return runSimWith(classIdx, raceIdx, level, n, makeMob)
}

// runSimWith runs n fights using a custom mob factory, enabling caster-mob tests.
func runSimWith(classIdx, raceIdx, level, n int, mobFn func(int) *types.Character) simResult {
	cs := NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(ch *types.Character, skillName string) int {
		if ch.IsNPC() {
			// Mobs have moderate combat skill — capped lower than players to
			// prevent excessive dodge rate and extra attacks at high levels.
			s := 20 + ch.Level*2
			if s > 80 {
				s = 80
			}
			return s
		}
		// Route skill-specific lookups to class-appropriate functions.
		switch skillName {
		case "dodge":
			return dodgeSkillForClass(ch.Class, ch.Level)
		case "parry":
			return parrySkillForClass(ch.Class, ch.Level)
		case "second attack":
			return extraAttackSkillForClass(ch.Class, ch.Level, 1)
		case "third attack":
			return extraAttackSkillForClass(ch.Class, ch.Level, 2)
		case "fourth attack":
			return extraAttackSkillForClass(ch.Class, ch.Level, 3)
		case "fifth attack":
			return extraAttackSkillForClass(ch.Class, ch.Level, 4)
		}
		return weaponSkillForClass(ch.Class, ch.Level)
	}

	var res simResult
	res.n = n

	for i := 0; i < n; i++ {
		p := makePlayer(classIdx, raceIdx, level)
		m := mobFn(level)
		isMobCaster := m.Act.Has(types.ActMage) || m.Act.Has(types.ActCleric)

		room := types.NewRoom(1, "Arena", "Arena.")
		p.InRoom = room
		m.InRoom = room
		room.AddPerson(p)
		room.AddPerson(m)

		SetFighting(p, m)
		SetFighting(m, p)

		const maxRounds = 200
		rounds := 0
		pMana := p.MaxMana
		// L70+: guarantee casters enough mana for a full fight.
		// Druid (ManaGain=0) and vampire (ManaGain=-30→0) only accumulate ~190 mana
		// from the base formula, which runs out after ~9 casts. At high levels, real
		// characters compensate with gear, mana regen, and items.
		if isCasterClass(classIdx) && level >= 70 {
			minMana := spellManaCost(classIdx, level) * 35
			if pMana < minMana {
				pMana = minMana
			}
		}
		// Caster mobs get enough mana to cast every round for a full fight.
		// Without this, mana depletion causes erratic results (mob runs out at
		// low levels where spell cost is high relative to the small mana pool).
		// The smooth mobCastSpellDam formula already limits damage — mana is not
		// the design constraint here.
		mobMana := 9999

		for rounds < maxRounds {
			if p.Position <= types.PosDead || m.Position <= types.PosDead {
				break
			}
			rounds++

			// ── Player attacks ────────────────────────────────────────────

			// Thief opener: assassinate at L75+ (dagger, 75% mob max HP),
			// regular backstab below L75 (any weapon, level-scaled multiplier).
			if rounds == 1 && (classIdx == types.ClassThief || classIdx == types.ClassMercenary) &&
				p.Fighting == m && IsAwake(p) {
				var burstDam int
				if level >= 75 {
					// Assassinate: fixed 75% of mob max HP (dagger + poison coat)
					burstDam = m.MaxHit * 3 / 4
				} else {
					// Regular backstab: weapon dice × level-scaled multiplier
					wNum, wSize := simWeaponDice(classIdx, level)
					burstDam = Dice(wNum, wSize)*backstabMult(level) + p.DamRoll*2
				}
				if burstDam > 0 {
					m.Hit -= burstDam
					res.totalPDmg += burstDam
					UpdatePosition(m)
				}
			}

			// Thief circle: every 4th round (ROM skill — stab a distracted target).
			// 2× weapon damage; available from round 4 to avoid stacking with backstab.
			if rounds > 1 && rounds%4 == 0 &&
				(classIdx == types.ClassThief || classIdx == types.ClassMercenary) &&
				m.Position > types.PosDead && p.Fighting == m && IsAwake(p) {
				wNum, wSize := simWeaponDice(classIdx, level)
				circleDam := Dice(wNum, wSize)*2 + p.DamRoll
				if circleDam > 0 {
					m.Hit -= circleDam
					res.totalPDmg += circleDam
					UpdatePosition(m)
				}
			}

			// Ranger dual-wield: one off-hand weapon strike per round.
			// Off-hand accuracy = weapon skill / 2 (less accurate than main hand).
			// Compensates for the lack of dual-wield and enhanced-dodge mechanics
			// in this sim; rangers would normally win via twin weapons + better evasion.
			if (classIdx == types.ClassRanger || classIdx == types.ClassStrider) &&
				m.Position > types.PosDead && p.Fighting == m && IsAwake(p) {
				offSkill := weaponSkillForClass(classIdx, level) / 3
				if Dice(1, 100) <= offSkill {
					wNum, wSize := weaponDice(classIdx, level)
					offDam := Dice(wNum, wSize) + p.DamRoll/2
					if offDam > 0 {
						m.Hit -= offDam
						res.totalPDmg += offDam
						UpdatePosition(m)
					}
				}
			}

			// Caster: cast best spell if fighting and conscious, then also melee.
			// Guard prevents spell-casting after HandleDeath clears Fighting refs
			// (otherwise a "dead" mage keeps blasting while the mob can't hit back).
			if isCasterClass(classIdx) && p.Fighting == m && IsAwake(p) {
				cost := spellManaCost(classIdx, level)
				if pMana >= cost {
					pMana -= cost
					spellDam := castSpellDamage(classIdx, level)
					// High-level mobs (L70+) have sanctuary: halves all incoming spell damage.
					// Models endgame mobs that buff before combat. Dispel magic removes it,
					// but that's not simulated here — this is the worst-case / no-dispel scenario.
					if m.Level >= 70 {
						spellDam = (spellDam + 1) / 2
					}
					if spellDam > 0 {
						m.Hit -= spellDam
						res.totalPDmg += spellDam
						UpdatePosition(m)

						// Vampire feed: drain/life-tap heals a fraction of spell damage.
						// ROM "feed" / vampiric touch — 1/12 of damage returned as HP.
						// Low ratio keeps vampire mortal; pure healing would break balance.
						if (classIdx == types.ClassVampire || classIdx == types.ClassLich) &&
							p.Hit < p.MaxHit {
							p.Hit += spellDam / 12
							if p.Hit > p.MaxHit {
								p.Hit = p.MaxHit
							}
						}
					}
				}
			}

			// All classes make melee attacks (MultiHit — skill-gated).
			// Diff mob HP before/after to track melee DPS for all classes.
			if m.Position > types.PosDead && p.Fighting == m && IsAwake(p) {
				mHPBefore := m.Hit
				cs.MultiHit(p, m)
				meleeDam := mHPBefore - m.Hit
				if meleeDam > 0 {
					res.totalPDmg += meleeDam
				}
			}

			if m.Position <= types.PosDead || m.Hit <= 0 {
				break // mob killed
			}

			// ── Mob attacks ───────────────────────────────────────────────

			// Caster mobs cast a spell before melee — bypasses dodge/parry.
			if isMobCaster && m.Fighting == p && IsAwake(m) {
				cost := spellManaCost(types.ClassMage, m.Level)
				if mobMana >= cost {
					mobMana -= cost
					spellDam := mobCastSpellDam(m.Level)
					if spellDam > 0 {
						p.Hit -= spellDam
						res.totalMDmg += spellDam
						UpdatePosition(p)
					}
				}
			}

			// Check if player was killed by mob spell (before mob melee).
			if p.Position <= types.PosDead {
				break
			}

			if m.Fighting == p && IsAwake(m) {
				pBefore := p.Hit
				cs.MultiHit(m, p)

				// Detect player death: HandleDeath revives the player (Hit=1,
				// PosResting) and clears both Fighting refs. If the mob is still
				// alive but lost its Fighting target, the player was just killed.
				if m.Hit > 0 && m.Fighting != p {
					// Track real damage: player went from pBefore to at most -11.
					dmgTaken := pBefore + 11 // conservative minimum
					if dmgTaken > 0 {
						res.totalMDmg += dmgTaken
					}
					break // mob wins — don't count as draw
				}

				mDmg := pBefore - p.Hit
				if mDmg < 0 {
					mDmg = 0
				}
				res.totalMDmg += mDmg
			}
		}

		res.totalRounds += rounds
		switch {
		case m.Hit <= 0:
			res.playerWins++
		case rounds >= maxRounds:
			res.draws++
		// else: mob won (player was killed) — implicit, not incremented
		}

		p.Fighting = nil
		m.Fighting = nil
		room.RemovePerson(p)
		room.RemovePerson(m)
	}

	return res
}

// ── test: class comparison ────────────────────────────────────────────────────

// TestCombatSimByClass shows all tier-1 classes (human race) vs equal-level mob,
// at levels 1, 5, 10, 15, 20, 25, 30.
func TestCombatSimByClass(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	levels := []int{1, 10, 20, 30, 40, 50, 60, 75, 100}

	tier1 := []int{
		types.ClassWarrior,
		types.ClassRanger,
		types.ClassThief,
		types.ClassCleric,
		types.ClassDruid,
		types.ClassVampire,
		types.ClassMage,
	}

	t.Log("")
	t.Log("=== TIER-1 CLASSES vs equal-level warrior mob  (human race, N=1000) ===")
	t.Logf("Round = %.2fs  |  Target: 20-30 rounds (15-22s) for a normal mob", secondsPerRound)
	t.Log("Casters use best available spell each round + melee")
	t.Log("Warriors wear plate; mages wear robes (different AC)")
	t.Log("")

	printTable := func(title string, cellFn func(r simResult) string) {
		hdr := fmt.Sprintf("%-10s", title)
		for _, lv := range levels {
			hdr += fmt.Sprintf("  Lv%-2d ", lv)
		}
		t.Log(hdr)
		t.Log(strings.Repeat("-", len(hdr)))
		for _, ci := range tier1 {
			row := fmt.Sprintf("%-10s", types.ClassTable[ci].Name)
			for _, lv := range levels {
				r := runSim(ci, raceIdx, lv, n)
				row += fmt.Sprintf("  %-5s", cellFn(r))
			}
			t.Log(row)
		}
		t.Log("")
	}

	printTable("Win%", func(r simResult) string {
		return fmt.Sprintf("%4.0f%%", r.winPct())
	})

	printTable("Rounds", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.avgRounds())
	})

	printTable("Secs", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.avgSeconds())
	})

	printTable("P-DPS", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.pDPS())
	})

	printTable("M-DPS", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.mDPS())
	})

	// HP table (deterministic)
	hdr := fmt.Sprintf("%-10s", "HP@level")
	for _, lv := range levels {
		hdr += fmt.Sprintf("  Lv%-2d ", lv)
	}
	t.Log(hdr)
	t.Log(strings.Repeat("-", len(hdr)))
	race := &types.RaceTable[raceIdx]
	for _, ci := range tier1 {
		cl := &types.ClassTable[ci]
		row := fmt.Sprintf("%-10s", cl.Name)
		for _, lv := range levels {
			row += fmt.Sprintf("  %-5d", playerHP(cl, race, lv))
		}
		t.Log(row)
	}
	// Mob HP for reference
	mobRow := fmt.Sprintf("%-10s", "mob HP")
	for _, lv := range levels {
		mobRow += fmt.Sprintf("  %-5d", mobHP(lv))
	}
	t.Log(mobRow)
	t.Log("")

	t.Log("Tune targets:")
	t.Log("  Win%:   55-65% player edge on even fights")
	t.Log("  Rounds: 20-30  (~15-22 seconds at 0.75s/round)")
	t.Log("  P-DPS / M-DPS should not differ by more than 3× for any class")
}

// ── test: race comparison ─────────────────────────────────────────────────────

// TestCombatSimByRace tests warrior class across all races at levels 10, 20, 30, 50.
func TestCombatSimByRace(t *testing.T) {
	const classIdx = types.ClassWarrior
	const n = 1000
	levels := []int{10, 20, 30, 50}

	t.Log("")
	t.Log("=== RACE COMPARISON  warrior class vs equal-level mob (N=1000) ===")
	t.Log("Str/Con drive HP and hit/dam; Dex drives dodge and AC bonus")
	t.Log("")

	hdr := fmt.Sprintf("%-12s  %3s %3s %3s %4s", "Race", "Str", "Dex", "Con", "HP15")
	for _, lv := range levels {
		hdr += fmt.Sprintf("  W%%Lv%-2d", lv)
	}
	hdr += fmt.Sprintf("  %7s", "Rds@10")
	t.Log(hdr)
	t.Log(strings.Repeat("-", len(hdr)))

	race15 := &types.RaceTable[0] // placeholder
	_ = race15
	for ri := 0; ri < types.MaxRace; ri++ {
		race := &types.RaceTable[ri]
		str15 := raceStatAtLevel(race, types.StatStr, 15)
		dex15 := raceStatAtLevel(race, types.StatDex, 15)
		con15 := raceStatAtLevel(race, types.StatCon, 15)
		hp15 := playerHP(&types.ClassTable[classIdx], race, 15)
		row := fmt.Sprintf("%-12s  %3d %3d %3d %4d", race.Name, str15, dex15, con15, hp15)
		for _, lv := range levels {
			r := runSim(classIdx, ri, lv, n)
			row += fmt.Sprintf("  %5.1f%%", r.winPct())
		}
		r10 := runSim(classIdx, ri, 10, n)
		row += fmt.Sprintf("  %7.1f", r10.avgRounds())
		t.Log(row)
	}
	t.Log("")
}

// ── test: race × class synergy ────────────────────────────────────────────────

// TestCombatSimRaceSynergy answers: "Does picking a race that matches your class matter?"
// Shows caster-friendly races vs fighter-friendly races for both mage and warrior.
func TestCombatSimRaceSynergy(t *testing.T) {
	const n = 1000

	// Hand-picked archetypes: pure fighter races, pure caster races, balanced
	fightRaces := []int{types.RaceGiant, types.RaceDwarf, types.RaceHalfOrc, types.RaceMinotaur, types.RaceTitan}
	castRaces := []int{types.RaceElf, types.RacePixie, types.RaceGnome, types.RaceHalfling}
	midRaces := []int{types.RaceHuman, types.RaceHalfElf, types.RaceKenku}

	levels := []int{10, 20, 30, 50}

	printSynergy := func(title string, classIdx int, raceLists [][]int, raceLabels []string) {
		t.Logf("--- %s (class: %s) ---", title, types.ClassTable[classIdx].Name)
		hdr := fmt.Sprintf("%-12s  %3s %3s %3s", "Race", "Str", "Int", "Con")
		for _, lv := range levels {
			hdr += fmt.Sprintf("  W%%@%-2d", lv)
		}
		t.Log(hdr)
		t.Log(strings.Repeat("-", len(hdr)))

		for li, raceList := range raceLists {
			if li > 0 {
				t.Log(fmt.Sprintf("  [%s]", raceLabels[li]))
			} else {
				t.Log(fmt.Sprintf("  [%s]", raceLabels[0]))
			}
			for _, ri := range raceList {
				race := &types.RaceTable[ri]
				str := raceStatAtLevel(race, types.StatStr, 15)
				intS := raceStatAtLevel(race, types.StatInt, 15)
				con := raceStatAtLevel(race, types.StatCon, 15)
				row := fmt.Sprintf("  %-10s  %3d %3d %3d", race.Name, str, intS, con)
				for _, lv := range levels {
					r := runSim(classIdx, ri, lv, n)
					row += fmt.Sprintf("  %4.0f%%", r.winPct())
				}
				t.Log(row)
			}
		}
		t.Log("")
	}

	t.Log("")
	t.Log("=== RACE × CLASS SYNERGY (N=1000) ===")
	t.Log("Fighter races: high Str/Con  |  Caster races: high Int/Wis  |  Str→hit/dam, Con→HP")
	t.Log("")

	// Warrior: fighter races should outperform caster races
	printSynergy("WARRIOR — fighter races vs caster races", types.ClassWarrior,
		[][]int{fightRaces, castRaces, midRaces},
		[]string{"fighter races (high Str/Con)", "caster races (high Int/Wis)", "balanced races"})

	// Mage: caster races get more mana, but spell damage is level-based not Int-based.
	// HP difference between races IS meaningful — low-Con mages die faster.
	printSynergy("MAGE — caster races vs fighter races", types.ClassMage,
		[][]int{castRaces, fightRaces, midRaces},
		[]string{"caster races (high Int/Wis/Dex)", "fighter races (high Str/Con)", "balanced races"})

	// Cleric: balanced — needs Wis for mana, Con for HP survival
	printSynergy("CLERIC — race impact on hybrid caster", types.ClassCleric,
		[][]int{castRaces, fightRaces, midRaces},
		[]string{"caster races", "fighter races", "balanced races"})

	t.Log("Notes:")
	t.Log("  - Fighter races win more as warrior because Str→damroll/hitroll via strTable,")
	t.Log("    and Con→HP per level (implemented in playerHP).")
	t.Log("  - Caster races get more MANA (Int/Wis) which allows more spells before OOM.")
	t.Log("  - Spell DAMAGE is level-based (1d4+level for magic missile), so race Int/Wis")
	t.Log("    does NOT boost spell damage in the current codebase — only mana pool.")
	t.Log("  - To give casters a bigger racial advantage, add an Int/Wis modifier to spell")
	t.Log("    damage: e.g.  dam += (caster.Int - 15) * level / 10")
}

// ── test: detailed single combo ───────────────────────────────────────────────

// TestCombatSimDetailed shows full per-level detail for one class+race combo.
// Edit classIdx and raceIdx to investigate a specific combination.
func TestCombatSimDetailed(t *testing.T) {
	classIdx := types.ClassWarrior
	raceIdx := types.RaceHuman
	const n = 2000
	levels := []int{1, 10, 20, 30, 40, 50, 60, 75, 100}

	cl := &types.ClassTable[classIdx]
	race := &types.RaceTable[raceIdx]

	t.Logf("=== DETAILED: %s / %s vs equal-level mob (N=%d) ===", cl.Name, race.Name, n)
	t.Logf("Round = %.2fs | Target: 20-30 rounds (15-22s)", secondsPerRound)
	t.Log("")

	hdr := fmt.Sprintf("%-5s  %-4s  %-6s  %-5s  %-6s  %-5s  %-5s  %-6s  %-4s",
		"Lv", "P-HP", "Win%", "Rnds", "Secs", "P-DPS", "M-DPS", "MobHP", "AC")
	t.Log(hdr)
	t.Log(strings.Repeat("-", len(hdr)))

	for _, lv := range levels {
		r := runSim(classIdx, raceIdx, lv, n)
		p := makePlayer(classIdx, raceIdx, lv)
		m := makeMob(lv)
		t.Logf("%-5d  %-4d  %5.1f%%  %5.1f  %5.1fs  %5.1f  %5.1f  %-6d  %-4d",
			lv, p.MaxHit,
			r.winPct(), r.avgRounds(), r.avgSeconds(),
			r.pDPS(), r.mDPS(),
			m.MaxHit, p.Armor[0])
	}

	t.Log("")
	t.Log("Character stats at each level:")
	hdr2 := fmt.Sprintf("%-5s  %-3s  %-3s  %-3s  %-3s  %-3s  %-4s  %-5s  %-5s  %-5s",
		"Lv", "Str", "Int", "Wis", "Dex", "Con", "Hit+", "Dam+", "Mana", "AC")
	t.Log(hdr2)
	for _, lv := range levels {
		p := makePlayer(classIdx, raceIdx, lv)
		t.Logf("%-5d  %-3d  %-3d  %-3d  %-3d  %-3d  %-4d  %-5d  %-5d  %-5d",
			lv,
			p.PermStats[types.StatStr], p.PermStats[types.StatInt],
			p.PermStats[types.StatWis], p.PermStats[types.StatDex],
			p.PermStats[types.StatCon],
			p.HitRoll, p.DamRoll, p.MaxMana, p.Armor[0])
	}

	if isCasterClass(classIdx) {
		t.Log("")
		t.Log("Spell damage (best available, no resistance):")
		hdr3 := fmt.Sprintf("%-5s  %-20s  %-6s", "Lv", "Spell", "AvgDam")
		t.Log(hdr3)
		for _, lv := range levels {
			var spellName string
			switch classIdx {
			case types.ClassMage, types.ClassWizard:
				if lv >= 22 {
					spellName = "fireball (3d6+lv*2)"
				} else if lv >= 13 {
					spellName = "lightning bolt (2d6+lv)"
				} else {
					spellName = "magic missile (1d4+lv)"
				}
			case types.ClassCleric, types.ClassPriest:
				if lv >= 45 {
					spellName = "cause critical (3d8+lv)"
				} else if lv >= 23 {
					spellName = "cause serious (2d8+lv/2)"
				} else {
					spellName = "cause light (1d8+lv/3)"
				}
			}
			// Estimate avg damage
			var avgDam float64
			switch classIdx {
			case types.ClassMage, types.ClassWizard:
				if lv >= 22 {
					avgDam = 10.5 + float64(lv)*2
				} else if lv >= 13 {
					avgDam = 7.0 + float64(lv)
				} else {
					avgDam = 2.5 + float64(lv)
				}
			case types.ClassCleric, types.ClassPriest:
				if lv >= 45 {
					avgDam = 13.5 + float64(lv)
				} else if lv >= 23 {
					avgDam = 9.0 + float64(lv)/2
				} else {
					avgDam = 4.5 + float64(lv)/3
				}
			}
			if spellName != "" {
				t.Logf("%-5d  %-20s  %6.1f", lv, spellName, avgDam)
			}
		}
	}
}

// ── test: vs spell-casting mob ────────────────────────────────────────────────

// TestCombatSimVsCasterMob compares all tier-1 classes against a spell-casting mob
// instead of the standard warrior mob. Key differences:
//   - Mob casts a spell each round (bypasses dodge/parry, direct HP hit)
//   - Mob melee is weak (staff/dagger level attacks)
//   - Physical-defense classes (warrior/ranger parry/dodge) lose their advantage
//   - Caster classes race to kill before the mob's spells whittle them down
//
// Interpretation guide:
//   - Results are highly sensitive to level-based spell tier transitions (L13, L22).
//     Large swings between adjacent levels are expected — the sim has no smoothing.
//   - Low levels (L1-L8): almost everyone loses. Caster mobs are genuinely dangerous
//     and low-level characters simply lack the HP/DPS to trade effectively.
//   - Mid levels (L10-L20): warriors and high-DPS casters can win; low-HP classes struggle.
//   - High levels (L25-L30): varies. Fireball-tier spells are devastating without MR.
//   - Without magic resistance (MR), ALL classes take full spell damage. MR mechanics
//     would boost physical classes relative to caster classes vs this mob type.
//
// Run:  go test ./pkg/combat/ -run TestCombatSimVsCasterMob -v
func TestCombatSimVsCasterMob(t *testing.T) {
	const raceIdx = types.RaceHuman
	const n = 1000
	levels := []int{1, 10, 20, 30, 40, 50, 60, 75, 100}

	tier1 := []int{
		types.ClassWarrior,
		types.ClassRanger,
		types.ClassThief,
		types.ClassCleric,
		types.ClassDruid,
		types.ClassVampire,
		types.ClassMage,
	}

	t.Log("")
	t.Log("=== TIER-1 CLASSES vs equal-level SPELL-CASTING mob  (human race, N=1000) ===")
	t.Log("Mob casts a spell EACH ROUND (bypasses dodge/parry) + weak melee")
	t.Log("Mob spell: magic missile(L<13) → lightning bolt(L13-21) → fireball(L22+)")
	t.Log("Same mob HP formula as warrior mob. Light armor (robes).")
	t.Log("")

	printTable := func(title string, cellFn func(r simResult) string) {
		hdr := fmt.Sprintf("%-10s", title)
		for _, lv := range levels {
			hdr += fmt.Sprintf("  Lv%-2d ", lv)
		}
		t.Log(hdr)
		t.Log(strings.Repeat("-", len(hdr)))
		for _, ci := range tier1 {
			row := fmt.Sprintf("%-10s", types.ClassTable[ci].Name)
			for _, lv := range levels {
				r := runSimWith(ci, raceIdx, lv, n, makeCasterMob)
				row += fmt.Sprintf("  %-5s", cellFn(r))
			}
			t.Log(row)
		}
		t.Log("")
	}

	printTable("Win%", func(r simResult) string {
		return fmt.Sprintf("%4.0f%%", r.winPct())
	})

	printTable("Rounds", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.avgRounds())
	})

	printTable("P-DPS", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.pDPS())
	})

	printTable("M-DPS", func(r simResult) string {
		return fmt.Sprintf("%4.1f", r.mDPS())
	})

	// Reference HP table
	t.Log("HP reference:")
	hdr := fmt.Sprintf("%-10s", "HP@level")
	for _, lv := range levels {
		hdr += fmt.Sprintf("  Lv%-2d ", lv)
	}
	t.Log(hdr)
	t.Log(strings.Repeat("-", len(hdr)))
	race := &types.RaceTable[raceIdx]
	for _, ci := range tier1 {
		cl := &types.ClassTable[ci]
		row := fmt.Sprintf("%-10s", cl.Name)
		for _, lv := range levels {
			row += fmt.Sprintf("  %-5d", playerHP(cl, race, lv))
		}
		t.Log(row)
	}
	casterMobRow := fmt.Sprintf("%-10s", "casterHP")
	for _, lv := range levels {
		casterMobRow += fmt.Sprintf("  %-5d", casterMobHP(lv))
	}
	t.Log(casterMobRow)
	t.Log("")
	t.Log("Spell DPS from caster mob (avg/round, bypasses dodge/parry):")
	hdr2 := fmt.Sprintf("%-10s", "MobSpell")
	for _, lv := range levels {
		hdr2 += fmt.Sprintf("  Lv%-2d ", lv)
	}
	t.Log(hdr2)
	t.Log(strings.Repeat("-", len(hdr2)))
	spellRow := fmt.Sprintf("%-10s", "avg/rnd")
	for _, lv := range levels {
		// Match mobCastSpellDam tiers
		var avg float64
		switch {
		case lv >= 22:
			avg = 9.0 + float64(lv) // 2d8+lv
		case lv >= 13:
			avg = 7.0 + float64(lv) // 2d6+lv
		default:
			avg = 3.5 + float64(lv)/2 // 1d6+lv/2
		}
		spellRow += fmt.Sprintf("  %-5.1f", avg)
	}
	t.Log(spellRow)
	t.Log("")
	t.Log("Key insight: warrior/ranger parry+dodge doesn't help vs spell damage.")
	t.Log("  Physical defence classes rely on HP to tank; casters race to kill the mob first.")
	t.Log("  Resistance/MR mechanics (not yet simulated) would help physical classes here.")
}
