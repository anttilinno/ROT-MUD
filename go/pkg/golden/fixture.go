package golden

import (
	"bytes"
	"fmt"
	"sort"

	"rotmud/pkg/ai"
	"rotmud/pkg/combat"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// runFixture is the entry point invoked by TestGolden. It emits every
// scenario section in a stable order into buf. All randomness is
// already seeded at the call site via combat.SetRand — this function
// never touches the global RNG directly.
func runFixture(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "# ROT-MUD entity golden snapshot")
	fmt.Fprintln(buf, "# Seed: 42 (do not change without regenerating)")
	fmt.Fprintln(buf, "# Coverage: 19 races x warrior Lv20 + 14 classes x human Lv20 + spells + skills + mob templates")
	fmt.Fprintln(buf)

	runRaceWarriorCombos(buf)
	fmt.Fprintln(buf)

	runClassHumanCombos(buf)
	fmt.Fprintln(buf)

	runSpellScenarios(buf)
	fmt.Fprintln(buf)

	runSkillScenarios(buf)
	fmt.Fprintln(buf)

	runMobTemplateSamples(buf)
}

// runRaceWarriorCombos: each of the 19 races paired with the warrior class at level 20.
// Captures race stat distribution, HP, immunity/vulnerability/resistance flags, and a
// deterministic 30-round combat log header vs a standard warrior mob.
func runRaceWarriorCombos(buf *bytes.Buffer) {
	const level = 20
	fmt.Fprintf(buf, "=== RACE x WARRIOR (Lv %d, seed=42) ===\n", level)
	for raceIdx := 0; raceIdx < len(types.RaceTable); raceIdx++ {
		emitRaceWarriorCombo(buf, raceIdx, level)
	}
}

func emitRaceWarriorCombo(buf *bytes.Buffer, raceIdx, level int) {
	ch := makePlayer(types.ClassWarrior, raceIdx, level)
	mob := makeMob(level)

	room := types.NewRoom(1, "Arena", "Arena.")
	ch.InRoom = room
	mob.InRoom = room
	room.AddPerson(ch)
	room.AddPerson(mob)

	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(_ *types.Character, _ string) int { return 75 }

	combat.SetFighting(ch, mob)
	combat.SetFighting(mob, ch)

	var dealt, hits, miss int
	for i := 0; i < 30; i++ {
		before := mob.Hit
		cs.OneHit(ch, mob, false)
		d := before - mob.Hit
		if d > 0 {
			dealt += d
			hits++
		} else {
			miss++
		}
		if mob.Position <= types.PosDead {
			break
		}
	}

	race := &types.RaceTable[raceIdx]
	fmt.Fprintf(buf, "Race=%-12s HP=%-4d Str=%-2d Dex=%-2d Con=%-2d Hit%%=%5.1f Dam=%4d Imm=%s Res=%s Vuln=%s\n",
		race.Name,
		ch.MaxHit,
		ch.PermStats[types.StatStr],
		ch.PermStats[types.StatDex],
		ch.PermStats[types.StatCon],
		safePct(hits, hits+miss),
		dealt,
		formatImmBits(ch.Imm),
		formatImmBits(ch.Res),
		formatImmBits(ch.Vuln),
	)
}

// runClassHumanCombos: each of the 14 classes paired with the human race at level 20.
func runClassHumanCombos(buf *bytes.Buffer) {
	const level = 20
	fmt.Fprintf(buf, "=== CLASS x HUMAN (Lv %d, seed=42) ===\n", level)
	for classIdx := 0; classIdx < len(types.ClassTable); classIdx++ {
		emitClassHumanCombo(buf, classIdx, level)
	}
}

func emitClassHumanCombo(buf *bytes.Buffer, classIdx, level int) {
	ch := makePlayer(classIdx, types.RaceHuman, level)
	mob := makeMob(level)

	room := types.NewRoom(2, "Arena", "Arena.")
	ch.InRoom = room
	mob.InRoom = room
	room.AddPerson(ch)
	room.AddPerson(mob)

	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(_ *types.Character, _ string) int { return 75 }

	combat.SetFighting(ch, mob)
	combat.SetFighting(mob, ch)

	var dealt, hits, miss int
	for i := 0; i < 30; i++ {
		before := mob.Hit
		cs.OneHit(ch, mob, false)
		d := before - mob.Hit
		if d > 0 {
			dealt += d
			hits++
		} else {
			miss++
		}
		if mob.Position <= types.PosDead {
			break
		}
	}

	cl := &types.ClassTable[classIdx]
	fmt.Fprintf(buf, "Class=%-12s THAC0_00=%-3d THAC0_32=%-3d HP=%-4d Mana=%-4d HitRoll=%-3d DamRoll=%-3d Hit%%=%5.1f Dam=%4d\n",
		cl.Name,
		cl.Thac0_00,
		cl.Thac0_32,
		ch.MaxHit,
		ch.MaxMana,
		ch.HitRoll,
		ch.DamRoll,
		safePct(hits, hits+miss),
		dealt,
	)
}

// runSpellScenarios: representative damage, affect, and healing spells.
// Uses Spell.Func directly (bypassing MagicSystem.Cast) to isolate spell
// damage parity from the defense pipeline (Research Pitfall #4).
func runSpellScenarios(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "=== SPELL EXECUTIONS (seed=42) ===")

	// Stable ordering: alphabetical by spell name.
	cases := []struct {
		spellName   string
		casterLevel int
		spellClass  int // caster class
	}{
		{"acid blast", 22, types.ClassMage},
		{"bless", 10, types.ClassCleric},
		{"cure light", 10, types.ClassCleric},
		{"fireball", 22, types.ClassMage},
		{"heal", 20, types.ClassCleric},
		{"magic missile", 10, types.ClassMage},
		{"sanctuary", 20, types.ClassCleric},
	}

	ms := magic.NewMagicSystem()
	ms.Output = func(_ *types.Character, _ string) {}

	for _, c := range cases {
		emitSpellScenario(buf, ms, c.spellName, c.casterLevel, c.spellClass)
	}
}

func emitSpellScenario(buf *bytes.Buffer, ms *magic.MagicSystem, spellName string, casterLevel, casterClass int) {
	spell := ms.Registry.FindByName(spellName)
	if spell == nil {
		fmt.Fprintf(buf, "Spell=%-20s NOT_FOUND\n", spellName)
		return
	}

	caster := makePlayer(casterClass, types.RaceHuman, casterLevel)
	victim := makePlayer(types.ClassWarrior, types.RaceHuman, casterLevel)

	// Ensure both participants start in a neutral, testable state.
	caster.Mana = 1000
	caster.MaxMana = 1000
	victim.Hit = 1000
	victim.MaxHit = 1000

	beforeHit := victim.Hit
	beforeMana := caster.Mana

	// Note: spell.Func signature is (caster *Character, level int, target interface{}) bool.
	// The victim is passed as interface{}.
	success := spell.Func(caster, casterLevel, victim)

	fmt.Fprintf(buf, "Spell=%-18s Lv=%-3d Class=%-10s success=%-5v dHP=%-5d dMana=%-4d victimHp=%d->%d\n",
		spellName,
		casterLevel,
		types.ClassTable[casterClass].Name,
		success,
		beforeHit-victim.Hit,
		beforeMana-caster.Mana,
		beforeHit,
		victim.Hit,
	)
}

// runSkillScenarios: representative offensive and defensive skills.
func runSkillScenarios(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "=== SKILL EXECUTIONS (seed=42) ===")

	emitBackstabScenario(buf, 20)
	emitKickScenario(buf, 20)
	emitDefenseTrial(buf, types.ClassWarrior, types.ClassThief, 20)
	emitDefenseTrial(buf, types.ClassThief, types.ClassWarrior, 20)
}

func emitBackstabScenario(buf *bytes.Buffer, level int) {
	ch := makePlayer(types.ClassThief, types.RaceHuman, level)
	victim := makeMob(level)

	room := types.NewRoom(3, "Arena", "Arena.")
	ch.InRoom = room
	victim.InRoom = room
	room.AddPerson(ch)
	room.AddPerson(victim)

	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(_ *types.Character, name string) int {
		if name == "backstab" {
			return 80
		}
		return 75
	}

	beforeHp := victim.Hit
	// Signature deviation (Rule 3): DoBackstab takes (ch, victim *types.Character),
	// not (ch *types.Character, victimName string) as the plan's <interfaces> block
	// claimed. Pass the victim pointer directly.
	cs.DoBackstab(ch, victim)
	fmt.Fprintf(buf, "Backstab Lv=%-3d thief vs mob: victimHp=%d->%d damage=%d fighting=%v\n",
		level, beforeHp, victim.Hit, beforeHp-victim.Hit, ch.Fighting != nil)
}

func emitKickScenario(buf *bytes.Buffer, level int) {
	ch := makePlayer(types.ClassWarrior, types.RaceHuman, level)
	victim := makeMob(level)

	room := types.NewRoom(4, "Arena", "Arena.")
	ch.InRoom = room
	victim.InRoom = room
	room.AddPerson(ch)
	room.AddPerson(victim)

	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(_ *types.Character, name string) int {
		if name == "kick" {
			return 80
		}
		return 75
	}

	combat.SetFighting(ch, victim)
	combat.SetFighting(victim, ch)

	beforeHp := victim.Hit
	// Signature deviation (Rule 3): DoKick takes (ch, victim *types.Character).
	// Pass the current opponent (victim) explicitly; nil would fall back to
	// ch.Fighting internally, but the plan intends an explicit attack here.
	cs.DoKick(ch, victim)
	fmt.Fprintf(buf, "Kick Lv=%-3d warrior vs mob: victimHp=%d->%d damage=%d\n",
		level, beforeHp, victim.Hit, beforeHp-victim.Hit)
}

func emitDefenseTrial(buf *bytes.Buffer, defenderClass, attackerClass, level int) {
	def := makePlayer(defenderClass, types.RaceHuman, level)
	atk := makePlayer(attackerClass, types.RaceHuman, level)

	room := types.NewRoom(5, "Arena", "Arena.")
	def.InRoom = room
	atk.InRoom = room
	room.AddPerson(def)
	room.AddPerson(atk)

	cs := combat.NewCombatSystem()
	cs.Output = func(_ *types.Character, _ string) {}
	cs.SkillGetter = func(_ *types.Character, name string) int {
		switch name {
		case "dodge", "parry", "shield block":
			return 80
		}
		return 75
	}

	combat.SetFighting(atk, def)
	combat.SetFighting(def, atk)

	// DefenseResult constants in pkg/combat are: DefenseNone, DefenseDodged,
	// DefenseParried, DefenseBlocked. The plan's <interfaces> block referenced
	// DefenseHit / DefenseMissed which do not exist — treat DefenseNone as
	// "no defensive reaction fired" (i.e. the attack would hit or miss on the
	// normal THAC0 path). Count it as "hit" for reporting purposes.
	var dodged, parried, blocked, hit int
	for i := 0; i < 100; i++ {
		switch cs.CheckDefenses(atk, def) {
		case combat.DefenseDodged:
			dodged++
		case combat.DefenseParried:
			parried++
		case combat.DefenseBlocked:
			blocked++
		default:
			hit++
		}
	}

	fmt.Fprintf(buf, "Defense Lv=%-3d %-10s vs %-10s: dodged=%-3d parried=%-3d blocked=%-3d hit=%-3d\n",
		level,
		types.ClassTable[defenderClass].Name,
		types.ClassTable[attackerClass].Name,
		dodged, parried, blocked, hit)
}

// ===== helpers =====

func safePct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100 * float64(n) / float64(d)
}

// immFlagNames is a local, deterministic mapping from immunity-flag bit to
// a stable human-readable name. The types package does not export such a
// map, so the golden fixture maintains its own. Keep this list in sync
// with types/flags.go const block starting at ImmSummon.
var immFlagNames = []struct {
	bit  types.ImmFlags
	name string
}{
	{types.ImmSummon, "summon"},
	{types.ImmCharm, "charm"},
	{types.ImmMagic, "magic"},
	{types.ImmWeapon, "weapon"},
	{types.ImmBash, "bash"},
	{types.ImmPierce, "pierce"},
	{types.ImmSlash, "slash"},
	{types.ImmFire, "fire"},
	{types.ImmCold, "cold"},
	{types.ImmLightning, "lightning"},
	{types.ImmAcid, "acid"},
	{types.ImmPoison, "poison"},
	{types.ImmNegative, "negative"},
	{types.ImmHoly, "holy"},
	{types.ImmEnergy, "energy"},
	{types.ImmMental, "mental"},
	{types.ImmDisease, "disease"},
	{types.ImmDrowning, "drowning"},
	{types.ImmLight, "light"},
	{types.ImmSound, "sound"},
	{types.ImmSilver, "silver"},
}

// formatImmBits renders an ImmFlags bitmask deterministically. Emits a
// sorted, comma-joined list of set flag names, or "-" if no flags set.
func formatImmBits(f types.ImmFlags) string {
	var names []string
	for _, entry := range immFlagNames {
		if f.Has(entry.bit) {
			names = append(names, entry.name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "-"
	}
	return "[" + joinStrings(names, ",") + "]"
}

func joinStrings(xs []string, sep string) string {
	// Using sort + joinStrings keeps the fixture free of any hidden map order.
	out := ""
	for i, s := range xs {
		if i > 0 {
			out += sep
		}
		out += s
	}
	return out
}

// ===== duplicated helpers from pkg/combat/combat_sim_test.go =====
// Per Research Open Question #1: duplicate rather than extract to testutil.
// Copied verbatim from combat_sim_test.go lines 43-362 (at commit 6e0810e).
// These helpers must remain byte-for-byte stable regardless of sim tuning
// — the golden snapshot is a function of their output under seed 42.
//
// Helpers included:
//   - raceStatAtLevel(race *types.Race, stat, level int) int
//   - playerHP(cl *types.Class, race *types.Race, level int) int
//   - classEquipAC(classIdx, level int) int
//   - weaponDice(classIdx, level int) (int, int)
//   - weaponTypeForClass(classIdx int) int
//   - makeWeapon(classIdx, level int) *types.Object
//   - playerMana(classIdx int, race *types.Race, level int) int
//   - makePlayer(classIdx, raceIdx, level int) *types.Character
//   - mobHP(level int) int
//   - makeMob(level int) *types.Character

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
// Scales through L100 (legendary weapons); damage roughly 6x from L1 to L100.
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
	num, size := weaponDice(classIdx, level)
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

	// Enhanced damage skill bonus: ranger/strider get DamRoll bonus
	// (mirrors combat_sim_test.go logic at the snapshot commit).
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
func mobHP(level int) int {
	if level <= 30 {
		base := level*level/2 + level*8 + 20
		return base + level*(30-level)/4
	}
	if level <= 80 {
		return 710 + (level-30)*30
	}
	// Endgame: 40/level above L80
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

// runMobTemplateSamples exercises mob-template behavior (immunities, aggro,
// caster special). Covers ROADMAP Phase 1 success criterion #3. Each
// sub-scenario builds fresh characters (Pitfall #3) and a fresh AISystem
// — no state leaks between scenarios.
//
// The three sub-scenarios and what they pin into the snapshot:
//
//	MobImm   — that mob.Imm/Res/Vuln flag bits render through formatImmBits
//	           (watches the immunity data surface).
//	MobAggro — that ai.AISystem.ProcessMobile + types.ActAggressive
//	           triggers StartCombat against a player in the same room
//	           (watches the aggro branch of defaultBehavior).
//	MobCast  — that the spec_cast_mage SpecialFunc registered by
//	           ai.NewSpecialRegistry is invoked when mob.Special is set,
//	           and that it routes through ctx.CastSpell
//	           (watches the mob-special dispatch path).
func runMobTemplateSamples(buf *bytes.Buffer) {
	fmt.Fprintln(buf, "=== MOB TEMPLATES (seed=42) ===")
	emitMobImmunityScenario(buf, 20)
	emitMobAggroScenario(buf, 20)
	emitMobCasterScenario(buf, 22)
}

// emitMobImmunityScenario builds a warrior-class mob at the given level,
// sets a known cocktail of immunity / resistance / vulnerability flags,
// and emits one line rendering them through formatImmBits. No AI or
// combat is run — this scenario only proves the data surface is visible
// in the snapshot.
func emitMobImmunityScenario(buf *bytes.Buffer, level int) {
	mob := makeMob(level)
	// Known flag cocktail — if the trait-migration phases change how
	// these are stored (e.g. move from bitfield to trait struct), the
	// snapshot MUST show the same flag names rendered here.
	mob.Imm.Set(types.ImmFire)
	mob.Imm.Set(types.ImmSilver)
	mob.Res.Set(types.ImmCharm)
	mob.Vuln.Set(types.ImmCold)

	fmt.Fprintf(buf, "MobImm   Lv=%-3d name=%-10s Imm=%s Res=%s Vuln=%s\n",
		level,
		mob.Name,
		formatImmBits(mob.Imm),
		formatImmBits(mob.Res),
		formatImmBits(mob.Vuln),
	)
}

// emitMobAggroScenario builds a level-N aggressive warrior mob and a
// level-N human warrior player in the same room (neither fighting),
// wires a minimal ai.AISystem with a recording StartCombat callback,
// and invokes ProcessMobile on the mob. It emits whether aggro fired
// and against whom.
//
// This exercises the ActAggressive branch in ai.(*AISystem).defaultBehavior.
func emitMobAggroScenario(buf *bytes.Buffer, level int) {
	mob := makeMob(level)
	mob.Name = "AggroMob"
	mob.Act.Set(types.ActAggressive)

	player := makePlayer(types.ClassWarrior, types.RaceHuman, level)
	player.Name = "AggroTarget"

	room := types.NewRoom(101, "AggroArena", "AggroArena.")
	mob.InRoom = room
	player.InRoom = room
	room.AddPerson(mob)
	room.AddPerson(player)

	var aggroFiredAgainst string

	aiSys := ai.NewAISystem()
	aiSys.Output = func(_ *types.Character, _ string) {}
	aiSys.ActToRoom = func(_ string, _, _ *types.Character, _ func(ch *types.Character, msg string)) {}
	aiSys.StartCombat = func(attacker, victim *types.Character) {
		// Record the aggro target; do NOT actually start combat (we want
		// the rest of the fixture to remain deterministic regardless of
		// post-aggro combat dice).
		if attacker == mob && victim != nil {
			aggroFiredAgainst = victim.Name
		}
	}
	aiSys.MoveChar = func(_ *types.Character, _ types.Direction) {}

	aiSys.ProcessMobile(mob)

	victimName := aggroFiredAgainst
	if victimName == "" {
		victimName = "-"
	}
	fmt.Fprintf(buf, "MobAggro Lv=%-3d name=%-10s Act=aggressive aggroFired=%-5v victim=%s\n",
		level,
		mob.Name,
		aggroFiredAgainst != "",
		victimName,
	)
}

// emitMobCasterScenario builds a level-N caster mob with Special =
// "spec_cast_mage" and a same-level player victim, puts them in combat
// (both Fighting set and PosFighting), wires a minimal ai.AISystem with
// a recording CastSpell callback, and invokes the special function
// directly via the Registry. It emits whether the special fired and
// which spell it attempted.
//
// This exercises the ch.Special -> SpecialRegistry.Find ->
// SpecialFunc(specCastMage) path in ai.(*AISystem).ProcessMobile.
func emitMobCasterScenario(buf *bytes.Buffer, level int) {
	mob := makeMob(level)
	mob.Name = "CasterMob"
	mob.Level = level
	mob.Mana = 1000
	mob.MaxMana = 1000
	mob.Special = "spec_cast_mage"

	victim := makePlayer(types.ClassWarrior, types.RaceHuman, level)
	victim.Name = "CasterTarget"
	victim.Hit = 1000
	victim.MaxHit = 1000

	room := types.NewRoom(102, "CasterArena", "CasterArena.")
	mob.InRoom = room
	victim.InRoom = room
	room.AddPerson(mob)
	room.AddPerson(victim)

	// spec_cast_mage requires Position == PosFighting and a victim whose
	// Fighting field points at the caster. Set both sides without
	// actually running combat.SetFighting (which may have other side
	// effects we want to keep out of this scenario).
	mob.Position = types.PosFighting
	mob.Fighting = victim
	victim.Position = types.PosFighting
	victim.Fighting = mob

	var attempted string
	castFired := false

	// Use the Registry directly so we can supply a custom SpecialContext
	// with a recording CastSpell. This is the explicitly authorised path
	// per VERIFICATION gap SC #3.
	aiSys := ai.NewAISystem()
	specFn := aiSys.Registry.Find("spec_cast_mage")
	if specFn == nil {
		fmt.Fprintf(buf, "MobCast  Lv=%-3d name=%-10s Special=spec_cast_mage REGISTRY_MISSING\n", level, mob.Name)
		return
	}

	beforeHp := victim.Hit
	ctx := &ai.SpecialContext{
		Magic:       nil,
		Output:      func(_ *types.Character, _ string) {},
		ActToRoom:   func(_ string, _, _ *types.Character, _ func(ch *types.Character, msg string)) {},
		StartCombat: func(_, _ *types.Character) {},
		CastSpell: func(_ *types.Character, spellName string, v *types.Character) bool {
			if attempted == "" {
				attempted = spellName
			}
			castFired = true
			// Pretend the spell landed for 1 HP so the snapshot shows a
			// non-zero delta whenever cast fired.
			if v != nil {
				v.Hit -= 1
			}
			return true
		},
		MoveChar: func(_ *types.Character, _ types.Direction) {},
	}

	specFn(mob, ctx)

	spellLabel := attempted
	if spellLabel == "" {
		spellLabel = "none"
	}
	fmt.Fprintf(buf, "MobCast  Lv=%-3d name=%-10s Special=spec_cast_mage fighting=%-5v spellAttempted=%-15s castFired=%-5v victimHp=%d->%d\n",
		level,
		mob.Name,
		mob.Fighting != nil,
		spellLabel,
		castFired,
		beforeHp,
		victim.Hit,
	)
}
