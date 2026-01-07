package game

import (
	"fmt"
	"sort"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/skills"
	"rotmud/pkg/types"
)

// Skill commands: train, practice, skills

// Training costs
const (
	TrainCostStat = 1 // Trains to gain a stat point
	TrainCostHP   = 1 // Trains to gain HP
	TrainCostMana = 1 // Trains to gain Mana
	TrainCostMove = 1 // Trains to gain Move
)

func (d *CommandDispatcher) cmdTrain(ch *types.Character, args string) {
	if ch.IsNPC() {
		d.send(ch, "NPCs can't train.\r\n")
		return
	}

	// Find a trainer in the room
	trainer := d.findTrainer(ch)
	if trainer == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	if args == "" {
		// Show what can be trained
		d.send(ch, "You can train:\r\n")
		d.send(ch, fmt.Sprintf("  str  - Strength     (cost: %d trains, current: %d)\r\n", TrainCostStat, ch.GetStat(types.StatStr)))
		d.send(ch, fmt.Sprintf("  int  - Intelligence (cost: %d trains, current: %d)\r\n", TrainCostStat, ch.GetStat(types.StatInt)))
		d.send(ch, fmt.Sprintf("  wis  - Wisdom       (cost: %d trains, current: %d)\r\n", TrainCostStat, ch.GetStat(types.StatWis)))
		d.send(ch, fmt.Sprintf("  dex  - Dexterity    (cost: %d trains, current: %d)\r\n", TrainCostStat, ch.GetStat(types.StatDex)))
		d.send(ch, fmt.Sprintf("  con  - Constitution (cost: %d trains, current: %d)\r\n", TrainCostStat, ch.GetStat(types.StatCon)))
		d.send(ch, fmt.Sprintf("  hp   - Hit Points   (cost: %d train, gain 10 max hp)\r\n", TrainCostHP))
		d.send(ch, fmt.Sprintf("  mana - Mana         (cost: %d train, gain 10 max mana)\r\n", TrainCostMana))
		d.send(ch, fmt.Sprintf("  move - Movement     (cost: %d train, gain 10 max move)\r\n", TrainCostMove))
		d.send(ch, fmt.Sprintf("\r\nYou have %d training sessions.\r\n", ch.Train))
		return
	}

	stat := strings.ToLower(args)

	// Check for stat training
	statMap := map[string]types.Stat{
		"str":          types.StatStr,
		"strength":     types.StatStr,
		"int":          types.StatInt,
		"intelligence": types.StatInt,
		"wis":          types.StatWis,
		"wisdom":       types.StatWis,
		"dex":          types.StatDex,
		"dexterity":    types.StatDex,
		"con":          types.StatCon,
		"constitution": types.StatCon,
	}

	if statType, ok := statMap[stat]; ok {
		// Train a stat
		if ch.Train < TrainCostStat {
			d.send(ch, fmt.Sprintf("You need %d training sessions for that.\r\n", TrainCostStat))
			return
		}

		// Check max stat (typically 18 + racial bonus)
		maxStat := 18
		if ch.PermStats[statType] >= maxStat {
			d.send(ch, "That stat is already at maximum.\r\n")
			return
		}

		ch.Train -= TrainCostStat
		ch.PermStats[statType]++
		d.send(ch, fmt.Sprintf("Your %s increases!\r\n", types.StatName(statType)))
		return
	}

	// Check for hp/mana/move training
	switch stat {
	case "hp", "hitpoints", "hits":
		if ch.Train < TrainCostHP {
			d.send(ch, "You don't have enough training sessions.\r\n")
			return
		}
		ch.Train -= TrainCostHP
		ch.MaxHit += 10
		if ch.PCData != nil {
			ch.PCData.PermHit += 10
		}
		d.send(ch, "Your maximum hit points increase by 10!\r\n")

	case "mana":
		if ch.Train < TrainCostMana {
			d.send(ch, "You don't have enough training sessions.\r\n")
			return
		}
		ch.Train -= TrainCostMana
		ch.MaxMana += 10
		if ch.PCData != nil {
			ch.PCData.PermMana += 10
		}
		d.send(ch, "Your maximum mana increases by 10!\r\n")

	case "move", "moves", "mv":
		if ch.Train < TrainCostMove {
			d.send(ch, "You don't have enough training sessions.\r\n")
			return
		}
		ch.Train -= TrainCostMove
		ch.MaxMove += 10
		if ch.PCData != nil {
			ch.PCData.PermMove += 10
		}
		d.send(ch, "Your maximum movement increases by 10!\r\n")

	default:
		d.send(ch, "You can't train that.\r\n")
	}
}

func (d *CommandDispatcher) findTrainer(ch *types.Character) *types.Character {
	if ch.InRoom == nil {
		return nil
	}

	for _, mob := range ch.InRoom.People {
		// Both trainers and guildmasters (practicers) can train stats
		if mob.IsNPC() && (mob.Act.Has(types.ActTrain) || mob.Act.Has(types.ActPractice)) {
			return mob
		}
	}
	return nil
}

func (d *CommandDispatcher) cmdPractice(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs can't practice.\r\n")
		return
	}

	if args == "" {
		// List all skills and spells
		d.listPracticableSkillsAndSpells(ch)
		return
	}

	// Find a practicer in the room
	practicer := d.findPracticer(ch)
	if practicer == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	if ch.Practice <= 0 {
		d.send(ch, "You have no practice sessions left.\r\n")
		return
	}

	// Find the skill or spell
	skillName := strings.ToLower(args)

	// Check if skill exists in learned map
	if ch.PCData.Learned == nil {
		ch.PCData.Learned = make(map[string]int)
	}

	// Find matching skill or spell that the player can use at their level
	var foundSkill string
	var requiredLevel int
	var isSpell bool

	// Max level a player can reach - spells with higher level requirements are not learnable
	const maxPlayerLevel = 51

	// First check for exact match in already learned skills/spells
	if ch.PCData.Learned[skillName] > 0 {
		// Exact match in learned - check if it's a skill or spell
		if d.Skills != nil {
			if skillDef := d.Skills.Registry.FindByName(skillName); skillDef != nil {
				reqLvl := skillDef.GetLevel(ch.Class)
				if reqLvl > 0 && ch.Level >= reqLvl {
					foundSkill = skillName
					requiredLevel = reqLvl
					isSpell = false
				}
			}
		}
		if foundSkill == "" && d.Magic != nil && d.Magic.Registry != nil {
			if spellDef := d.Magic.Registry.FindByName(skillName); spellDef != nil {
				reqLvl := spellDef.GetClassLevel(ch.Class)
				if reqLvl > 0 && reqLvl <= maxPlayerLevel && ch.Level >= reqLvl {
					foundSkill = skillName
					requiredLevel = reqLvl
					isSpell = true
				}
			}
		}
	}

	// Check for exact match in registries (not yet learned)
	if foundSkill == "" && d.Skills != nil {
		if skillDef := d.Skills.Registry.FindByName(skillName); skillDef != nil {
			classLevel := skillDef.GetLevel(ch.Class)
			if classLevel > 0 && ch.Level >= classLevel {
				foundSkill = skillDef.Name
				requiredLevel = classLevel
				isSpell = false
				if ch.PCData.Learned[foundSkill] == 0 {
					ch.PCData.Learned[foundSkill] = 1 // Start at 1%
				}
			} else if classLevel > 0 && ch.Level < classLevel {
				d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", classLevel, skillDef.Name))
				return
			}
		}
	}
	if foundSkill == "" && d.Magic != nil && d.Magic.Registry != nil {
		if spellDef := d.Magic.Registry.FindByName(skillName); spellDef != nil {
			classLevel := spellDef.GetClassLevel(ch.Class)
			if classLevel > 0 && classLevel <= maxPlayerLevel {
				if ch.Level >= classLevel {
					foundSkill = spellDef.Name
					requiredLevel = classLevel
					isSpell = true
					if ch.PCData.Learned[foundSkill] == 0 {
						ch.PCData.Learned[foundSkill] = 1 // Start at 1%
					}
				} else {
					d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", classLevel, spellDef.Name))
					return
				}
			}
		}
	}

	// Now check prefix matches in already learned skills/spells
	if foundSkill == "" {
		for skill, learned := range ch.PCData.Learned {
			if learned <= 0 {
				continue // Not learned yet
			}
			if strings.HasPrefix(strings.ToLower(skill), skillName) {
				// Check skill registry
				if d.Skills != nil {
					if skillDef := d.Skills.Registry.FindByName(skill); skillDef != nil {
						reqLvl := skillDef.GetLevel(ch.Class)
						if reqLvl > 0 && ch.Level >= reqLvl {
							foundSkill = skill
							requiredLevel = reqLvl
							isSpell = false
							break
						}
					}
				}
				// Check magic registry
				if d.Magic != nil && d.Magic.Registry != nil {
					if spellDef := d.Magic.Registry.FindByName(skill); spellDef != nil {
						reqLvl := spellDef.GetClassLevel(ch.Class)
						if reqLvl > 0 && reqLvl <= maxPlayerLevel && ch.Level >= reqLvl {
							foundSkill = skill
							requiredLevel = reqLvl
							isSpell = true
							break
						}
					}
				}
			}
		}
	}

	// Finally check prefix matches in registries (not yet learned)
	if foundSkill == "" && d.Skills != nil {
		if skillDef := d.Skills.Registry.FindByPrefix(skillName); skillDef != nil {
			classLevel := skillDef.GetLevel(ch.Class)
			if classLevel > 0 && ch.Level >= classLevel {
				foundSkill = skillDef.Name
				requiredLevel = classLevel
				isSpell = false
				ch.PCData.Learned[foundSkill] = 1 // Start at 1%
			} else if classLevel > 0 && ch.Level < classLevel {
				d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", classLevel, skillDef.Name))
				return
			}
		}
	}
	if foundSkill == "" && d.Magic != nil && d.Magic.Registry != nil {
		if spellDef := d.Magic.Registry.FindByPrefix(skillName); spellDef != nil {
			classLevel := spellDef.GetClassLevel(ch.Class)
			if classLevel > 0 && classLevel <= maxPlayerLevel {
				if ch.Level >= classLevel {
					foundSkill = spellDef.Name
					requiredLevel = classLevel
					isSpell = true
					ch.PCData.Learned[foundSkill] = 1 // Start at 1%
				} else {
					d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", classLevel, spellDef.Name))
					return
				}
			}
		}
	}

	if foundSkill == "" {
		d.send(ch, "You can't practice that.\r\n")
		return
	}

	// Double-check level requirement
	if requiredLevel > 0 && ch.Level < requiredLevel {
		d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", requiredLevel, foundSkill))
		return
	}

	current := ch.PCData.Learned[foundSkill]

	// Check max practice level (usually 75% from practice, 100% from use)
	maxPractice := 75
	if current >= maxPractice {
		if isSpell {
			d.send(ch, "You are already learned in that spell.\r\n")
		} else {
			d.send(ch, "You are already proficient in that skill.\r\n")
		}
		return
	}

	// Gain skill - based on wisdom and intelligence
	// Wisdom affects how much you gain per practice
	// Intelligence affects how quickly you learn
	wisBonus := (ch.GetStat(types.StatWis) - 10) / 2
	intBonus := (ch.GetStat(types.StatInt) - 10) / 2
	gain := 5 + wisBonus + intBonus + combat.NumberRange(1, 5)
	if gain < 1 {
		gain = 1
	}

	ch.Practice--
	ch.PCData.Learned[foundSkill] = min(current+gain, maxPractice)

	if isSpell {
		d.send(ch, fmt.Sprintf("You practice %s. Your proficiency improves to %d%%.\r\n",
			foundSkill, ch.PCData.Learned[foundSkill]))
	} else {
		d.send(ch, fmt.Sprintf("You practice %s. Your skill improves to %d%%.\r\n",
			foundSkill, ch.PCData.Learned[foundSkill]))
	}
}

// listPracticableSkillsAndSpells shows all skills and spells the character has learned (> 0%)
func (d *CommandDispatcher) listPracticableSkillsAndSpells(ch *types.Character) {
	if ch.PCData.Learned == nil {
		ch.PCData.Learned = make(map[string]int)
	}

	// Collect learned skills and spells
	type practiceEntry struct {
		name    string
		level   int
		rating  string
		isSpell bool
	}
	var entries []practiceEntry

	// Check each learned skill/spell
	for name, learned := range ch.PCData.Learned {
		if learned <= 0 {
			continue // Not learned yet - hide from list
		}

		// Check if it's a skill
		if d.Skills != nil {
			if skillDef := d.Skills.Registry.FindByName(name); skillDef != nil {
				reqLevel := skillDef.GetLevel(ch.Class)
				if reqLevel > 0 && ch.Level >= reqLevel {
					entries = append(entries, practiceEntry{
						name:    name,
						level:   learned,
						rating:  skillRating(learned),
						isSpell: false,
					})
					continue
				}
			}
		}

		// Check if it's a spell
		// Max level a player can reach - spells with higher level requirements are not learnable
		const maxPlayerLevel = 51
		if d.Magic != nil && d.Magic.Registry != nil {
			if spellDef := d.Magic.Registry.FindByName(name); spellDef != nil {
				reqLevel := spellDef.GetClassLevel(ch.Class)
				// Only show spells the class can actually learn (reqLevel <= maxPlayerLevel)
				if reqLevel > 0 && reqLevel <= maxPlayerLevel && ch.Level >= reqLevel {
					entries = append(entries, practiceEntry{
						name:    name,
						level:   learned,
						rating:  skillRating(learned),
						isSpell: true,
					})
				}
			}
		}
	}

	// Sort alphabetically
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].name < entries[j].name
	})

	// Display skills
	d.send(ch, "Your skills:\r\n")
	col := 0
	hasSkills := false
	for _, entry := range entries {
		if entry.isSpell {
			continue
		}
		hasSkills = true
		d.send(ch, fmt.Sprintf("%-18s %3d%% %-10s  ", entry.name, entry.level, entry.rating))
		col++
		if col >= 2 {
			d.send(ch, "\r\n")
			col = 0
		}
	}
	if col > 0 {
		d.send(ch, "\r\n")
	}
	if !hasSkills {
		d.send(ch, "  (none)\r\n")
	}

	// Display spells
	d.send(ch, "\r\nYour spells:\r\n")
	col = 0
	hasSpells := false
	for _, entry := range entries {
		if !entry.isSpell {
			continue
		}
		hasSpells = true
		d.send(ch, fmt.Sprintf("%-18s %3d%% %-10s  ", entry.name, entry.level, entry.rating))
		col++
		if col >= 2 {
			d.send(ch, "\r\n")
			col = 0
		}
	}
	if col > 0 {
		d.send(ch, "\r\n")
	}
	if !hasSpells {
		d.send(ch, "  (none)\r\n")
	}

	d.send(ch, fmt.Sprintf("\r\nYou have %d practice sessions left.\r\n", ch.Practice))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (d *CommandDispatcher) findPracticer(ch *types.Character) *types.Character {
	if ch.InRoom == nil {
		return nil
	}

	for _, mob := range ch.InRoom.People {
		if mob.IsNPC() && mob.Act.Has(types.ActPractice) {
			return mob
		}
	}
	return nil
}

func skillRating(level int) string {
	switch {
	case level <= 0:
		return "(not learned)"
	case level < 15:
		return "(awful)"
	case level < 30:
		return "(bad)"
	case level < 45:
		return "(poor)"
	case level < 60:
		return "(average)"
	case level < 75:
		return "(fair)"
	case level < 90:
		return "(good)"
	case level < 100:
		return "(very good)"
	default:
		return "(superb)"
	}
}

func (d *CommandDispatcher) cmdSkills(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs don't have skills.\r\n")
		return
	}

	if d.Skills == nil || d.Skills.Registry == nil {
		d.send(ch, "No skills available.\r\n")
		return
	}

	// Collect all skills for this class (excluding spells)
	type skillInfo struct {
		name    string
		level   int
		learned int
		canUse  bool
	}

	var classSkills []skillInfo
	for _, skill := range d.Skills.Registry.All() {
		if skill.Type != skills.TypeSkill {
			continue // Skip spells - those go in cmdSpells
		}

		reqLevel := skill.GetLevel(ch.Class)
		if reqLevel == 0 {
			continue // Class can't learn this skill
		}

		learned := 0
		if ch.PCData.Learned != nil {
			learned = ch.PCData.Learned[skill.Name]
		}

		classSkills = append(classSkills, skillInfo{
			name:    skill.Name,
			level:   reqLevel,
			learned: learned,
			canUse:  ch.Level >= reqLevel,
		})
	}

	// Sort by level requirement
	sort.Slice(classSkills, func(i, j int) bool {
		if classSkills[i].level != classSkills[j].level {
			return classSkills[i].level < classSkills[j].level
		}
		return classSkills[i].name < classSkills[j].name
	})

	if len(classSkills) == 0 {
		d.send(ch, "You have no skills available.\r\n")
		return
	}

	d.send(ch, "Skills available to your class:\r\n\r\n")
	d.send(ch, "Skill               Lv  Proficiency\r\n")
	d.send(ch, "-----------------------------------\r\n")

	for _, s := range classSkills {
		var profStr string
		if s.canUse {
			rating := skillRating(s.learned)
			profStr = fmt.Sprintf("%3d%% %s", s.learned, rating)
		} else {
			profStr = "n/a"
		}
		d.send(ch, fmt.Sprintf("%-18s %3d  %s\r\n", s.name, s.level, profStr))
	}

	d.send(ch, fmt.Sprintf("\r\nYou have %d practice sessions left.\r\n", ch.Practice))
}

// Gain command - for learning skill groups (usually at creation)
func (d *CommandDispatcher) cmdGain(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs can't gain.\r\n")
		return
	}

	// Find a guildmaster
	guildmaster := d.findPracticer(ch) // Guildmasters also practice
	if guildmaster == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "You can gain:\r\n")
		d.send(ch, "  convert      - Convert 6 practices into 1 train\r\n")
		d.send(ch, "  study        - Convert 1 train into 6 practices\r\n")
		d.send(ch, "  list         - List available skills and spell groups\r\n")
		d.send(ch, "  <skill/group> - Purchase a skill or spell group\r\n")
		d.send(ch, fmt.Sprintf("\r\nYou have %d trains and %d practices.\r\n", ch.Train, ch.Practice))
		return
	}

	argLower := strings.ToLower(args)

	switch argLower {
	case "convert":
		if ch.Practice < 6 {
			d.send(ch, "You need at least 6 practices to convert.\r\n")
			return
		}
		ch.Practice -= 6
		ch.Train++
		d.send(ch, fmt.Sprintf("%s helps you apply your practice to training.\r\n", guildmaster.ShortDesc))

	case "study", "revert":
		if ch.Train < 1 {
			d.send(ch, "You need at least 1 train to study.\r\n")
			return
		}
		ch.Train--
		ch.Practice += 6
		d.send(ch, fmt.Sprintf("%s helps you apply your training to practice.\r\n", guildmaster.ShortDesc))

	case "list":
		d.listGainableSkills(ch)

	default:
		// Try to purchase a skill or group
		d.tryGainSkillOrGroup(ch, guildmaster, argLower)
	}
}

// listGainableSkills shows available skills and spell groups for purchase
func (d *CommandDispatcher) listGainableSkills(ch *types.Character) {
	groups := d.getAvailableGroups(ch)
	skills := d.getAvailableSkills(ch)

	// Determine if groups are spell groups or skill groups based on class
	groupType := "Spell"
	if ch.Class == types.ClassWarrior || ch.Class == types.ClassThief {
		groupType = "Skill"
	}

	d.send(ch, fmt.Sprintf("%s groups available:\r\n", groupType))
	hasGroups := false
	for _, g := range groups {
		// Skip if already known
		if d.hasAllGroupSkills(ch, g.Skills) {
			continue
		}
		hasGroups = true
		d.send(ch, fmt.Sprintf("  %-20s %2d trains\r\n", g.Name, g.Cost))
	}
	if !hasGroups {
		d.send(ch, "  (none)\r\n")
	}

	d.send(ch, "\r\nIndividual skills available:\r\n")
	col := 0
	hasSkills := false
	for _, s := range skills {
		// Skip if already known
		if ch.PCData.Learned != nil && ch.PCData.Learned[s.Name] > 0 {
			continue
		}
		hasSkills = true
		d.send(ch, fmt.Sprintf("  %-18s %2d", s.Name, s.Cost))
		col++
		if col >= 2 {
			d.send(ch, "\r\n")
			col = 0
		}
	}
	if col > 0 {
		d.send(ch, "\r\n")
	}
	if !hasSkills {
		d.send(ch, "  (none)\r\n")
	}

	d.send(ch, fmt.Sprintf("\r\nYou have %d trains and %d practices.\r\n", ch.Train, ch.Practice))
}

// tryGainSkillOrGroup attempts to purchase a skill or group
func (d *CommandDispatcher) tryGainSkillOrGroup(ch *types.Character, guildmaster *types.Character, name string) {
	// Determine group type based on class
	groupType := "spells"
	if ch.Class == types.ClassWarrior || ch.Class == types.ClassThief {
		groupType = "skills"
	}

	// First check groups
	groups := d.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), name) {
			// Found a matching group
			if d.hasAllGroupSkills(ch, g.Skills) {
				d.send(ch, "You already know that group.\r\n")
				return
			}
			if ch.Train < g.Cost {
				d.send(ch, fmt.Sprintf("You need %d trains to learn %s.\r\n", g.Cost, g.Name))
				return
			}
			// Purchase the group
			ch.Train -= g.Cost
			if ch.PCData.Learned == nil {
				ch.PCData.Learned = make(map[string]int)
			}
			for _, skill := range g.Skills {
				if ch.PCData.Learned[skill] == 0 {
					ch.PCData.Learned[skill] = 1
				}
			}
			d.send(ch, fmt.Sprintf("%s trains you in the %s of %s.\r\n", guildmaster.ShortDesc, groupType, g.Name))
			return
		}
	}

	// Then check individual skills
	skills := d.getAvailableSkills(ch)
	for _, s := range skills {
		if strings.HasPrefix(strings.ToLower(s.Name), name) {
			// Found a matching skill
			if ch.PCData.Learned != nil && ch.PCData.Learned[s.Name] > 0 {
				d.send(ch, "You already know that skill.\r\n")
				return
			}
			if ch.Train < s.Cost {
				d.send(ch, fmt.Sprintf("You need %d trains to learn %s.\r\n", s.Cost, s.Name))
				return
			}
			// Purchase the skill
			ch.Train -= s.Cost
			if ch.PCData.Learned == nil {
				ch.PCData.Learned = make(map[string]int)
			}
			ch.PCData.Learned[s.Name] = 1
			d.send(ch, fmt.Sprintf("%s trains you in the skill of %s.\r\n", guildmaster.ShortDesc, s.Name))
			return
		}
	}

	d.send(ch, "You can't learn that.\r\n")
}

// hasAllGroupSkills checks if character has all skills from a group
func (d *CommandDispatcher) hasAllGroupSkills(ch *types.Character, skills []string) bool {
	if ch.PCData == nil || ch.PCData.Learned == nil {
		return false
	}
	for _, skill := range skills {
		if ch.PCData.Learned[skill] == 0 {
			return false
		}
	}
	return true
}

// SkillGroup represents a bundle of skills that can be learned together
type SkillGroup struct {
	Name   string
	Cost   int
	Skills []string
}

// IndividualSkill represents a single skill that can be purchased
type IndividualSkill struct {
	Name string
	Cost int
}

// getAvailableGroups returns skill groups available to a character's class
func (d *CommandDispatcher) getAvailableGroups(ch *types.Character) []SkillGroup {
	classGroups := map[int][]SkillGroup{
		types.ClassWarrior: {
			{"weaponsmaster", 10, []string{"axe", "dagger", "flail", "mace", "polearm", "spear", "sword", "whip"}},
		},
		types.ClassThief: {
			{"stealth", 6, []string{"sneak", "hide", "backstab", "circle"}},
		},
		types.ClassMage: {
			{"attack", 3, []string{"magic missile", "burning hands", "chill touch", "colour spray"}},
			{"beguiling", 4, []string{"charm person", "sleep"}},
			{"combat", 7, []string{"acid blast", "fireball", "lightning bolt"}},
			{"detection", 5, []string{"detect evil", "detect good", "detect hidden", "detect magic", "detect invis", "identify"}},
			{"enhancement", 6, []string{"giant strength", "haste", "infravision"}},
			{"illusion", 4, []string{"invisibility", "mass invis", "ventriloquate"}},
			{"maladictions", 5, []string{"blindness", "curse", "poison", "plague", "weaken"}},
			{"protective", 6, []string{"armor", "shield", "stone skin"}},
			{"transportation", 5, []string{"fly", "pass door", "teleport", "gate"}},
		},
		types.ClassCleric: {
			{"attack", 3, []string{"cause light", "cause serious", "cause critical", "flamestrike"}},
			{"benedictions", 5, []string{"bless", "calm", "holy word", "remove curse"}},
			{"creation", 2, []string{"create food", "create water", "create spring"}},
			{"curative", 4, []string{"cure blindness", "cure disease", "cure poison"}},
			{"detection", 5, []string{"detect evil", "detect good", "detect hidden", "detect magic", "detect invis", "identify"}},
			{"healing", 7, []string{"cure light", "cure serious", "cure critical", "heal", "mass healing"}},
			{"protective", 6, []string{"armor", "sanctuary", "shield"}},
			{"transportation", 4, []string{"fly", "word of recall", "summon"}},
			{"weather", 3, []string{"call lightning", "control weather"}},
		},
	}

	if cg, ok := classGroups[ch.Class]; ok {
		return cg
	}
	return nil
}

// getAvailableSkills returns individual skills available for purchase
func (d *CommandDispatcher) getAvailableSkills(ch *types.Character) []IndividualSkill {
	skillCosts := map[string][]int{
		// Combat skills - [mage, cleric, thief, warrior]
		"second attack":   {6, 5, 4, 2},
		"third attack":    {0, 0, 0, 4},
		"fourth attack":   {0, 0, 0, 6},
		"dual wield":      {0, 0, 5, 5},
		"dodge":           {7, 6, 2, 5},
		"parry":           {7, 5, 5, 2},
		"shield block":    {0, 4, 0, 2},
		"enhanced damage": {0, 7, 6, 3},
		"grip":            {0, 0, 0, 3},
		"kick":            {0, 3, 5, 2},
		"bash":            {0, 0, 0, 2},
		"trip":            {0, 0, 3, 5},
		"dirt kicking":    {0, 0, 2, 3},
		"disarm":          {0, 0, 4, 3},
		"gouge":           {0, 0, 3, 0},
		"stun":            {0, 0, 0, 5},
		"backstab":        {0, 0, 6, 0},
		"circle":          {0, 0, 5, 0},
		"berserk":         {0, 0, 0, 6},
		"rescue":          {0, 4, 0, 3},
		"hand to hand":    {6, 4, 5, 3},
		// Thief skills
		"sneak":     {0, 0, 3, 0},
		"hide":      {0, 0, 2, 0},
		"steal":     {0, 0, 4, 0},
		"pick lock": {0, 0, 3, 0},
		"peek":      {0, 0, 1, 0},
		"envenom":   {0, 0, 5, 0},
		"track":     {0, 0, 4, 0},
		// Weapon skills
		"sword":   {5, 4, 3, 2},
		"dagger":  {2, 4, 2, 3},
		"spear":   {0, 4, 0, 3},
		"mace":    {0, 2, 5, 3},
		"axe":     {0, 0, 0, 2},
		"flail":   {0, 2, 0, 3},
		"whip":    {0, 0, 4, 0},
		"polearm": {0, 0, 0, 3},
		// Utility skills
		"meditation":   {3, 3, 0, 0},
		"fast healing": {6, 4, 5, 3},
		"haggle":       {4, 4, 2, 6},
		"lore":         {0, 0, 4, 5},
		"scrolls":      {1, 1, 4, 6},
		"staves":       {1, 1, 5, 7},
		"wands":        {1, 1, 4, 6},
	}

	result := make([]IndividualSkill, 0)
	for name, costs := range skillCosts {
		cost := 0
		switch ch.Class {
		case types.ClassMage:
			cost = costs[0]
		case types.ClassCleric:
			cost = costs[1]
		case types.ClassThief:
			cost = costs[2]
		case types.ClassWarrior:
			cost = costs[3]
		}
		if cost > 0 {
			result = append(result, IndividualSkill{Name: name, Cost: cost})
		}
	}

	// Sort by name
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}
