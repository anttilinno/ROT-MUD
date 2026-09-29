package combatsim

import (
	"rotmud/pkg/ai"
	"rotmud/pkg/combat"
	"rotmud/pkg/magic"
	"rotmud/pkg/skills"
	"rotmud/pkg/types"
)

// The sim's players act through the game's own code: thieves open with
// DoBackstab/DoAssassinate and circle with DoCircle, casters cast their
// best damage spell through MagicSystem.Cast, and skill availability comes
// from the skill registry. Only proficiency (how well a skill is learned at
// a level) is modelled here.

var (
	simSkills   = skills.DefaultSkills()
	simSkillSys = &skills.SkillSystem{Registry: simSkills}
	simMagic    = magic.NewMagicSystem()
	simAI       = &ai.AISystem{Registry: ai.NewSpecialRegistry(), Magic: simMagic}
	// measureMagic casts bestSpell's test spells, apart from the fight in progress.
	measureMagic = magic.NewMagicSystem()
)

// classHasSkill reports whether ch's class has learned skill name by now.
// Names the registry doesn't know are treated as available.
func classHasSkill(ch *types.Character, name string) bool {
	sk := simSkills.FindByName(name)
	if sk == nil {
		return true
	}
	lv := sk.GetLevel(ch.SkillClass())
	return lv > 0 && ch.Level >= lv
}

// learnSkills fills PCData.Learned the way a practised character's would be.
func learnSkills(ch *types.Character) {
	if ch.PCData == nil {
		ch.PCData = &types.PCData{Learned: map[string]int{}}
	}
	prof := weaponSkillForClass(ch.Class, ch.Level)
	for _, sk := range simSkills.All() {
		if classHasSkill(ch, sk.Name) {
			ch.PCData.Learned[sk.Name] = prof
		}
	}
	for _, sp := range simMagic.Registry.All() {
		if lv := sp.GetClassLevel(ch.SkillClass()); lv > 0 && lv <= ch.Level {
			ch.PCData.Learned[sp.Name] = 90
		}
	}
}

// selfBuffs are the defensive spells a player casts before a fight.
var selfBuffs = []string{"armor", "bless", "shield", "stone skin", "sanctuary"}

// preBuff casts each self-buff p's class knows, retrying a fizzle twice, as a
// player would before engaging, then rests.
func preBuff(p *types.Character) {
	for _, name := range selfBuffs {
		for try := 0; try < 3 && classCanCast(p, name) && !p.Affected.HasType(name); try++ {
			simMagic.Cast(p, name, "", nil)
		}
	}
	p.Mana = p.MaxMana // buffs outlast a rest back to full mana
}

// opener is the player's first move before the fight starts.
func opener(cs *combat.CombatSystem, p, m *types.Character) {
	if p.Class != types.ClassThief {
		return
	}
	if classHasSkill(p, "assassinate") {
		p.AffectedBy.Set(types.AffVenomReady) // coated with venom beforehand
		cs.DoAssassinate(p, m)
	} else if classHasSkill(p, "backstab") {
		cs.DoBackstab(p, m)
	}
}

// classAction is the player's command for a round in which they are not lagged.
func classAction(cs *combat.CombatSystem, p, m *types.Character) {
	switch {
	case p.Class == types.ClassThief && classHasSkill(p, "circle"):
		cs.DoCircle(p)
	case isCasterClass(p.Class):
		if m.IsAffected(types.AffSanctuary) && classCanCast(p, "dispel magic") {
			simMagic.Cast(p, "dispel magic", "", nil)
		} else if sp := bestSpell(p); sp != nil {
			simMagic.Cast(p, sp.Name, "", nil)
		}
	}
}

func classCanCast(p *types.Character, name string) bool {
	sp := simMagic.Registry.FindByName(name)
	return sp != nil && sp.CanCast(p)
}

type spellKey struct{ class, level int }

var bestSpellCache = map[spellKey]*magic.Spell{}

// bestSpell is the damage spell with the highest average damage that p's
// class can cast at p's level, measured by casting each candidate.
func bestSpell(p *types.Character) *magic.Spell {
	if !isCasterClass(p.Class) {
		return nil
	}
	key := spellKey{p.Class, p.Level}
	if sp, ok := bestSpellCache[key]; ok {
		return sp
	}
	var best *magic.Spell
	bestDam := 0
	for _, sp := range simMagic.Registry.All() {
		if sp.Target != magic.TargetCharOffense && !roomDamageSpells[sp.Name] {
			continue
		}
		if lv := sp.GetClassLevel(p.SkillClass()); lv == 0 || lv > p.Level {
			continue
		}
		if d := spellAvgDamage(p.Class, p.Level, sp); d > bestDam {
			best, bestDam = sp, d
		}
	}
	bestSpellCache[key] = best
	return best
}

var roomDamageSpells = map[string]bool{"earthquake": true, "call lightning": true, "chain lightning": true}

func spellAvgDamage(class, level int, sp *magic.Spell) int {
	const n = 50
	total := 0
	for i := 0; i < n; i++ {
		c := makePlayer(class, types.RaceHuman, level)
		c.Mana = 1 << 20
		c.PCData.Learned[sp.Name] = 100
		dummy := types.NewNPC(2, "dummy", level)
		dummy.MaxHit, dummy.Hit = 1<<20, 1<<20
		room := types.NewRoom(2, "Range", "Range.")
		room.AddPerson(c)
		room.AddPerson(dummy)
		c.InRoom, dummy.InRoom = room, room
		combat.SetFighting(c, dummy)
		measureMagic.Cast(c, sp.Name, "", nil)
		total += dummy.MaxHit - dummy.Hit
	}
	return total / n
}
