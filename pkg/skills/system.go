package skills

import (
	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// SkillSystem manages skill lookup and improvement
type SkillSystem struct {
	Registry *SkillRegistry
	Output   func(ch *types.Character, msg string)
}

// NewSkillSystem creates a new skill system with default skills
func NewSkillSystem() *SkillSystem {
	return &SkillSystem{
		Registry: DefaultSkills(),
	}
}

// GetSkill returns the effective skill level for a character
// Returns 0-100 (percentage)
func (s *SkillSystem) GetSkill(ch *types.Character, skillName string) int {
	skill := s.Registry.FindByName(skillName)
	if skill == nil {
		return 0
	}

	return s.GetSkillByIndex(ch, s.Registry.GetIndex(skillName))
}

// GetSkillByIndex returns the effective skill level by skill index
func (s *SkillSystem) GetSkillByIndex(ch *types.Character, sn int) int {
	if sn < 0 {
		// -1 is shorthand for level-based skill
		return ch.Level * 5 / 2
	}

	skill := s.Registry.FindByIndex(sn)
	if skill == nil {
		return 0
	}

	// For players
	if !ch.IsNPC() {
		// Check if character has learned this skill
		if ch.PCData == nil || ch.PCData.Learned == nil {
			return 0
		}

		// Check level requirement
		reqLevel := skill.GetLevel(ch.Class)
		if reqLevel == 0 || ch.Level < reqLevel {
			return 0
		}

		// Return learned percentage
		if learned, ok := ch.PCData.Learned[skill.Name]; ok {
			// Drunk penalty: 10% skill reduction when drunk
			if ch.PCData.Condition[types.CondDrunk] > 10 {
				learned = learned * 9 / 10
			}
			return learned
		}
		return 0
	}

	// For NPCs - provide reasonable defaults based on type and level
	return s.getNPCSkill(ch, skill)
}

// getNPCSkill calculates skill level for NPCs
func (s *SkillSystem) getNPCSkill(ch *types.Character, skill *Skill) int {
	// Spells: NPCs get level-based spell ability
	if skill.Type == TypeSpell {
		return 40 + ch.Level
	}

	// Combat skills based on mob flags
	switch skill.Name {
	case "dodge":
		if ch.Act.Has(types.ActWarrior) || ch.Act.Has(types.ActThief) {
			return ch.Level
		}
	case "parry":
		if ch.Act.Has(types.ActWarrior) {
			return ch.Level
		}
	case "shield block":
		return 10 + ch.Level
	case "second attack":
		if ch.Act.Has(types.ActWarrior) || ch.Act.Has(types.ActThief) {
			return 10 + 3*(ch.Level/2)
		}
	case "third attack":
		if ch.Act.Has(types.ActWarrior) {
			return 2*ch.Level - 40
		}
	case "hand to hand":
		return 40 + ch.Level
	case "kick":
		return 10 + 3*(ch.Level/2)
	case "bash":
		if ch.Act.Has(types.ActWarrior) {
			return 10 + 3*(ch.Level/2)
		}
	case "backstab":
		if ch.Act.Has(types.ActThief) {
			return 20 + 3*(ch.Level/2)
		}
	case "sneak", "hide":
		if ch.Act.Has(types.ActThief) {
			return ch.Level + 20
		}
	}

	return 0
}

// CheckImprove checks if a skill improves from use
func (s *SkillSystem) CheckImprove(ch *types.Character, skillName string, success bool, multiplier int) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	skill := s.Registry.FindByName(skillName)
	if skill == nil {
		return
	}

	// Check if character knows this skill
	reqLevel := skill.GetLevel(ch.Class)
	rating := skill.GetRating(ch.Class)
	if reqLevel == 0 || rating == 0 || ch.Level < reqLevel {
		return
	}

	// Get current skill level
	learned := 0
	if ch.PCData.Learned != nil {
		learned = ch.PCData.Learned[skillName]
	}

	// Can't improve if not known or already maxed
	if learned == 0 || learned >= 100 {
		return
	}

	// Calculate chance to improve
	// Based on intelligence, skill rating, and multiplier
	intStat := ch.GetStat(types.StatInt)
	chance := 10 * intStat // Base chance from intelligence
	chance /= multiplier * rating * 4
	chance += ch.Level

	// Random check
	if combat.NumberRange(1, 1000) > chance {
		return
	}

	// Now check if they actually improve
	if success {
		// Success: easier to improve at lower skill levels
		improveChance := 100 - learned
		if improveChance < 5 {
			improveChance = 5
		}
		if improveChance > 95 {
			improveChance = 95
		}

		if combat.NumberPercent() < improveChance {
			if ch.PCData.Learned == nil {
				ch.PCData.Learned = make(map[string]int)
			}
			ch.PCData.Learned[skillName]++
			s.send(ch, "You have become better at "+skillName+"!\r\n")

			// Gain some XP
			xpGain := 2 * rating
			ch.Exp += xpGain
		}
	} else {
		// Failure: learn from mistakes
		improveChance := learned / 2
		if improveChance < 5 {
			improveChance = 5
		}
		if improveChance > 30 {
			improveChance = 30
		}

		if combat.NumberPercent() < improveChance {
			if ch.PCData.Learned == nil {
				ch.PCData.Learned = make(map[string]int)
			}
			gain := combat.NumberRange(1, 3)
			ch.PCData.Learned[skillName] += gain
			if ch.PCData.Learned[skillName] > 100 {
				ch.PCData.Learned[skillName] = 100
			}
			s.send(ch, "You learn from your mistakes, and your "+skillName+" skill improves.\r\n")

			// Gain some XP
			xpGain := 2 * rating
			ch.Exp += xpGain
		}
	}
}

// LearnSkill teaches a skill to a character (from trainer)
func (s *SkillSystem) LearnSkill(ch *types.Character, skillName string, amount int) bool {
	if ch.IsNPC() || ch.PCData == nil {
		return false
	}

	skill := s.Registry.FindByName(skillName)
	if skill == nil {
		return false
	}

	// Check if character can learn this skill
	reqLevel := skill.GetLevel(ch.Class)
	if reqLevel == 0 || ch.Level < reqLevel {
		return false
	}

	// Initialize learned map if needed
	if ch.PCData.Learned == nil {
		ch.PCData.Learned = make(map[string]int)
	}

	// Add to learned percentage
	current := ch.PCData.Learned[skillName]
	ch.PCData.Learned[skillName] = min(current+amount, 100)

	return true
}

// GetLearnedPercent returns how much of a skill a character has learned
func (s *SkillSystem) GetLearnedPercent(ch *types.Character, skillName string) int {
	if ch.IsNPC() || ch.PCData == nil || ch.PCData.Learned == nil {
		return 0
	}
	return ch.PCData.Learned[skillName]
}

// send outputs a message to a character
func (s *SkillSystem) send(ch *types.Character, msg string) {
	if s.Output != nil {
		s.Output(ch, msg)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
