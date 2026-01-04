package game

import (
	"fmt"
	"sort"
	"strings"

	"rotmud/pkg/combat"
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
		if mob.IsNPC() && mob.Act.Has(types.ActTrain) {
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
		// List all skills and their levels
		d.send(ch, "Your skills:\r\n")

		if ch.PCData.Learned == nil || len(ch.PCData.Learned) == 0 {
			d.send(ch, "You have no skills.\r\n")
		} else {
			// Sort skills alphabetically
			skills := make([]string, 0, len(ch.PCData.Learned))
			for skill := range ch.PCData.Learned {
				skills = append(skills, skill)
			}
			sort.Strings(skills)

			col := 0
			for _, skill := range skills {
				// Check if player meets level requirement for this skill
				requiredLevel := 0
				if d.Skills != nil {
					if skillDef := d.Skills.Registry.FindByName(skill); skillDef != nil {
						requiredLevel = skillDef.GetLevel(ch.Class)
					}
				}

				// Skip skills the player can't use yet (level 0 means can't learn, or level too low)
				if requiredLevel == 0 || ch.Level < requiredLevel {
					continue
				}

				level := ch.PCData.Learned[skill]
				rating := skillRating(level)
				d.send(ch, fmt.Sprintf("%-18s %3d%% %-10s  ", skill, level, rating))
				col++
				if col >= 2 {
					d.send(ch, "\r\n")
					col = 0
				}
			}
			if col > 0 {
				d.send(ch, "\r\n")
			}
		}

		d.send(ch, fmt.Sprintf("\r\nYou have %d practice sessions left.\r\n", ch.Practice))
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

	// Find the skill
	skillName := strings.ToLower(args)

	// Check if skill exists in learned map
	if ch.PCData.Learned == nil {
		ch.PCData.Learned = make(map[string]int)
	}

	// Find matching skill that the player can use at their level
	var foundSkill string
	var requiredLevel int
	for skill := range ch.PCData.Learned {
		if strings.HasPrefix(strings.ToLower(skill), skillName) {
			// Check level requirement
			if d.Skills != nil {
				if skillDef := d.Skills.Registry.FindByName(skill); skillDef != nil {
					reqLvl := skillDef.GetLevel(ch.Class)
					if reqLvl > 0 && ch.Level >= reqLvl {
						foundSkill = skill
						requiredLevel = reqLvl
						break
					}
				}
			}
		}
	}

	// Also check if it's a skill the character could learn based on class
	if foundSkill == "" && d.Skills != nil {
		// Check skill registry for learnable skills
		if skillDef := d.Skills.Registry.FindByName(skillName); skillDef != nil {
			classLevel := skillDef.GetLevel(ch.Class)
			if classLevel > 0 && ch.Level >= classLevel {
				foundSkill = skillDef.Name
				requiredLevel = classLevel
				ch.PCData.Learned[foundSkill] = 1 // Start at 1%
			} else if classLevel > 0 && ch.Level < classLevel {
				d.send(ch, fmt.Sprintf("You must be level %d to practice %s.\r\n", classLevel, skillDef.Name))
				return
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
		d.send(ch, "You are already proficient in that skill.\r\n")
		return
	}

	// Gain skill - based on wisdom
	wisBonus := (ch.GetStat(types.StatWis) - 10) / 2
	gain := 5 + wisBonus + combat.NumberRange(1, 5)
	if gain < 1 {
		gain = 1
	}

	ch.Practice--
	ch.PCData.Learned[foundSkill] = min(current+gain, maxPractice)

	d.send(ch, fmt.Sprintf("You practice %s. Your skill improves to %d%%.\r\n",
		foundSkill, ch.PCData.Learned[foundSkill]))
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

	d.cmdPractice(ch, "") // Same as practice with no args
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
		d.send(ch, "Gain what?\r\n")
		d.send(ch, "You can gain: convert, revert\r\n")
		d.send(ch, "  convert - Convert 6 practices into 1 train\r\n")
		d.send(ch, "  study   - Convert 1 train into 6 practices\r\n")
		return
	}

	switch strings.ToLower(args) {
	case "convert":
		if ch.Practice < 6 {
			d.send(ch, "You are not yet ready.\r\n")
			return
		}
		ch.Practice -= 6
		ch.Train++
		d.send(ch, fmt.Sprintf("%s helps you apply your practice to training.\r\n", guildmaster.ShortDesc))

	case "study", "revert":
		if ch.Train < 1 {
			d.send(ch, "You are not yet ready.\r\n")
			return
		}
		ch.Train--
		ch.Practice += 6
		d.send(ch, fmt.Sprintf("%s helps you apply your training to practice.\r\n", guildmaster.ShortDesc))

	default:
		d.send(ch, "You can gain: convert, revert\r\n")
	}
}
