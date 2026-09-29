package combatsim

// Combat balance simulation — v3.
//
// Fights run through the game's own code: MultiHit, the combat skills
// (backstab, assassinate, circle), MagicSystem.Cast for player spells and the
// AI specials (spec_cast_*) for mobs, with command lag honoured. The sim only
// models what a character has: class/race tables, level-scaled weapons and
// armour, and how well skills are learned (sim_class_test.go).
//
// Timing reference (from game/loop.go):
//   PulsePerSecond = 4 (250ms/pulse), PulseViolence = 3
//   => 1 combat round = 0.75 seconds
//   => 20 rounds ≈ 15 seconds, 30 rounds ≈ 22 seconds
//
// Run individual tests:
//
//	go test ./pkg/combatsim -run TestCombatSimByClass  -v
//	go test ./pkg/combatsim -run TestCombatSimByRace   -v
//	go test ./pkg/combatsim -run TestCombatSimRaceSynergy -v
//	go test ./pkg/combatsim -run TestCombatSimDetailed -v

import (
	"fmt"
	"rotmud/pkg/combat"
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
	case types.ClassWarrior:
		base -= 25 // plate armour
	case types.ClassRanger, types.ClassCleric:
		base -= 10 // chain / mail
	case types.ClassThief:
		base += 10 // leather
	case types.ClassDruid:
		base += 15 // light leather
	case types.ClassGhoul:
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
	case types.ClassMage, types.ClassGhoul:
		size = size * 2 / 3
		if size < 4 {
			size = 4
		}
	}
	return num, size
}

// weaponTypeForClass maps class to ROM weapon type (0=exotic,1=sword,2=dagger,3=spear,4=mace)
func weaponTypeForClass(classIdx int) int {
	switch classIdx {
	case types.ClassWarrior:
		return 1 // sword
	case types.ClassRanger:
		return 3 // spear
	case types.ClassThief, types.ClassMage,
		types.ClassGhoul:
		return 2 // dagger
	case types.ClassCleric:
		return 4 // mace
	default:
		return 0 // exotic / polearm
	}
}

// makeWeapon creates a level-appropriate weapon for a class and equips it.
func makeWeapon(classIdx, level int) *types.Object {
	num, size := weaponDice(classIdx, level)
	w := types.NewObject(1, "weapon", types.ItemTypeWeapon)
	w.Values[0] = weaponTypeForClass(classIdx)
	w.Values[1] = num
	w.Values[2] = size
	// Damage class matters now that mobs carry ROM imm/res/vuln flags.
	w.Values[3] = int(map[int]types.DamageType{1: types.DamSlash, 2: types.DamPierce, 3: types.DamPierce, 4: types.DamBash}[w.Values[0]])
	if w.Values[3] == 0 {
		w.Values[3] = int(types.DamBash)
	}
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

	// Equip a class-appropriate weapon, and a second one for dual wielders.
	ch.Equip(makeWeapon(classIdx, level), types.WearLocWield)
	learnSkills(ch)
	if classHasSkill(ch, "dual wield") {
		ch.Equip(makeWeapon(classIdx, level), types.WearLocSecondary)
	}

	ch.Position = types.PosStanding
	return ch
}

// mobHP returns HP for a standard warrior-type mob at the given level.
// L1-L30:  bell-curve formula — quadratic with +33% bonus at L10 fading to 0 at L30.
// L31-L80: linear at 30/level from the L30 base (710).
// L81+:    steeper ramp at 40/level — endgame mobs scale harder.
//
//	Breakpoint at L80 so L75 balance is fully preserved; the steeper slope
//	only affects L81+ where player HP growth outpaces mob DPS and all classes
//	were winning 75-88% at L100 against only a 59% baseline from mage.
//
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
	mob.Special = "spec_cast_mage"

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

// weaponSkillForClass returns weapon proficiency for a class at a level.
func weaponSkillForClass(classIdx, level int) int {
	var growth float64
	switch classIdx {
	case types.ClassWarrior:
		growth = 4.0
	case types.ClassRanger:
		growth = 3.5
	case types.ClassThief:
		growth = 3.0
	case types.ClassCleric, types.ClassDruid:
		growth = 2.5
	case types.ClassGhoul:
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
	case types.ClassThief:
		growth = 3.0 // light armor, highest dodge
	case types.ClassRanger:
		growth = 2.5 // medium armor, good dodge
	case types.ClassMage:
		growth = 2.0 // robes only, rely on evasion
	case types.ClassGhoul:
		growth = 2.0 // supernatural agility
	case types.ClassDruid:
		growth = 1.5 // light leather, moderate
	case types.ClassCleric:
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
	case types.ClassWarrior, types.ClassRanger:
		growth = 2.5 // trained melee fighters parry well
	case types.ClassThief, types.ClassCleric:
		growth = 2.0
	case types.ClassGhoul:
		growth = 2.0 // unnatural reflexes
	case types.ClassDruid:
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

// isCasterClass returns true for classes that use spells in combat.
func isCasterClass(classIdx int) bool {
	switch classIdx {
	case types.ClassMage, types.ClassCleric, types.ClassDruid, types.ClassGhoul:
		return true
	}
	return false
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
	return runSimFull(classIdx, raceIdx, level, n, makePlayer, mobFn)
}

// runSimFull runs n fights with custom player and mob factories.
func runSimFull(classIdx, raceIdx, level, n int,
	playerFn func(classIdx, raceIdx, level int) *types.Character,
	mobFn func(int) *types.Character) simResult {
	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(ch *types.Character, skillName string) int {
		if !ch.IsNPC() && !classHasSkill(ch, skillName) {
			return 0
		}
		if ch.IsNPC() {
			return simSkillSys.GetSkill(ch, skillName) // the game's mob skills
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

	simMagic.Combat = cs

	var res simResult
	res.n = n

	for i := 0; i < n; i++ {
		p := playerFn(classIdx, raceIdx, level)
		m := mobFn(level)

		room := types.NewRoom(1, "Arena", "Arena.")
		p.InRoom = room
		m.InRoom = room
		room.AddPerson(p)
		room.AddPerson(m)

		combat.SetFighting(p, m)
		combat.SetFighting(m, p)

		const maxRounds = 200
		rounds := 0
		// L70+: guarantee casters enough mana for a full fight. Druid
		// (ManaGain=0) and ghoul only accumulate ~190 mana from the base
		// formula; real characters compensate with gear, regen and items.
		if sp := bestSpell(p); sp != nil && level >= 70 {
			p.Mana = max(p.Mana, sp.ManaCost*35)
		}

		for rounds < maxRounds {
			if p.Position <= types.PosDead || m.Position <= types.PosDead {
				break
			}
			rounds++

			// ── Player attacks ────────────────────────────────────────────

			// Lag wears off one round at a time, as in the game loop.
			p.Wait = max(0, p.Wait-1)
			if p.Wait == 0 && p.Fighting == m && combat.IsAwake(p) && m.Position > types.PosDead {
				mBefore := m.Hit
				classAction(cs, p, m)
				if d := mBefore - m.Hit; d > 0 {
					res.totalPDmg += d
				}
			}

			// All classes make melee attacks (MultiHit — skill-gated).
			// Diff mob HP before/after to track melee DPS for all classes.
			if m.Position > types.PosDead && p.Fighting == m && combat.IsAwake(p) {
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

			// Mob specials (spec_cast_* and the rest) run on the game's mobile
			// pulse: 4 pulses against 3 per round, so 3 rounds in 4.
			if rounds%4 != 0 && m.Fighting == p {
				pBefore := p.Hit
				simAI.ProcessMobile(m)
				if m.Fighting != p { // player killed and revived by HandleDeath
					res.totalMDmg += pBefore + 11
					break
				}
				if d := pBefore - p.Hit; d > 0 {
					res.totalMDmg += d
				}
			}

			// Check if player was killed by mob spell (before mob melee).
			if p.Position <= types.PosDead {
				break
			}

			if m.Fighting == p && combat.IsAwake(m) {
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
		types.ClassGhoul,
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
		t.Log("Spell damage (best available, measured on an equal-level mob):")
		t.Log(fmt.Sprintf("%-5s  %-20s  %-6s", "Lv", "Spell", "AvgDam"))
		for _, lv := range levels {
			if sp := bestSpell(makePlayer(classIdx, types.RaceHuman, lv)); sp != nil {
				t.Logf("%-5d  %-20s  %6d", lv, sp.Name, spellAvgDamage(classIdx, lv, sp))
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
// Run:  go test ./pkg/combatsim -run TestCombatSimVsCasterMob -v
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
		types.ClassGhoul,
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
