package combat

import (
	"fmt"

	"rotmud/pkg/types"
)

// BaseExpPerLevel is the base experience needed per level
const BaseExpPerLevel = 1000

// ExpPerOverspentPoint is the additional exp per level for each overspent creation point
const ExpPerOverspentPoint = 100

// ExpToLevel returns the experience required to reach a given level
// This uses a constant XP per level (no exponential scaling)
func ExpToLevel(level int) int {
	if level <= 1 {
		return 0
	}
	// Constant 1000 XP per level: level 2 = 1000, level 3 = 2000, etc.
	return BaseExpPerLevel * (level - 1)
}

// ExpToLevelWithPenalty returns the experience required for a character
// taking into account any creation point overspending penalty
func ExpToLevelWithPenalty(level int, overspentPoints int) int {
	if level <= 1 {
		return 0
	}
	expPerLevel := BaseExpPerLevel
	if overspentPoints > 0 {
		expPerLevel += overspentPoints * ExpPerOverspentPoint
	}
	return expPerLevel * (level - 1)
}

// ExpForKill calculates the experience gained for killing a victim
func ExpForKill(killer, victim *types.Character) int {
	if victim == nil {
		return 0
	}

	// Base exp is 100 * victim level
	baseExp := 100 * victim.Level

	// Level difference modifier
	levelDiff := victim.Level - killer.Level

	// Bonus for fighting higher level opponents
	if levelDiff > 0 {
		baseExp += levelDiff * 50
	} else if levelDiff < -5 {
		// Penalty for fighting much lower level opponents
		baseExp = baseExp * (10 + levelDiff) / 10
		if baseExp < 1 {
			baseExp = 1
		}
	}

	// Random variance (+/- 20%)
	variance := NumberRange(-20, 20)
	baseExp = baseExp * (100 + variance) / 100

	if baseExp < 1 {
		baseExp = 1
	}

	return baseExp
}

// GainExp adds experience to a character and handles level ups
func (c *CombatSystem) GainExp(ch *types.Character, exp int) {
	if ch.IsNPC() {
		return // NPCs don't gain exp
	}

	if exp <= 0 {
		return
	}

	oldLevel := ch.Level
	ch.Exp += exp

	if c.Output != nil {
		c.Output(ch, fmt.Sprintf("You receive %d experience points.\r\n", exp))
	}

	// Check for level up - use overspent points if available
	overspent := 0
	if ch.PCData != nil {
		overspent = ch.PCData.OverspentPoints
	}
	for ch.Level < types.MaxLevel && ch.Exp >= ExpToLevelWithPenalty(ch.Level+1, overspent) {
		ch.Level++
		c.levelUp(ch)
	}

	// Announce level up and trigger callback
	if ch.Level > oldLevel {
		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You have advanced to level %d!\r\n", ch.Level))

			// Notify room
			if ch.InRoom != nil {
				for _, person := range ch.InRoom.People {
					if person != ch {
						c.Output(person, fmt.Sprintf("%s has advanced to level %d!\r\n", ch.Name, ch.Level))
					}
				}
			}
		}

		// Call level up callback (used for auto-saving when reaching level 2)
		if c.OnLevelUp != nil {
			c.OnLevelUp(ch, oldLevel, ch.Level)
		}
	}
}

// levelUp applies level-up bonuses
func (c *CombatSystem) levelUp(ch *types.Character) {
	// Increase max HP
	hpGain := 10 + ch.GetStat(types.StatCon)
	if hpGain < 5 {
		hpGain = 5
	}
	ch.MaxHit += hpGain
	ch.Hit = ch.MaxHit // Full heal on level

	// Increase max mana
	manaGain := 5 + ch.GetStat(types.StatInt)
	if manaGain < 3 {
		manaGain = 3
	}
	ch.MaxMana += manaGain
	ch.Mana = ch.MaxMana

	// Increase max move
	moveGain := 5 + ch.GetStat(types.StatDex)
	if moveGain < 3 {
		moveGain = 3
	}
	ch.MaxMove += moveGain
	ch.Move = ch.MaxMove

	// Add training and practice sessions
	// Practice gains are based on wisdom: base 2 + (wis - 10) / 2
	ch.Train += 1
	practiceGain := 2 + (ch.GetStat(types.StatWis)-10)/2
	if practiceGain < 2 {
		practiceGain = 2 // Minimum 2 practices per level
	}
	ch.Practice += practiceGain
}

// GroupGain distributes experience to all group members in the room
func (c *CombatSystem) GroupGain(killer, victim *types.Character) {
	if killer.InRoom == nil {
		return
	}

	// Calculate base experience
	baseExp := ExpForKill(killer, victim)

	// Find the group leader
	leader := killer
	if killer.Leader != nil {
		leader = killer.Leader
	}

	// Count group members in the same room
	members := make([]*types.Character, 0)
	totalLevels := 0

	for _, ch := range killer.InRoom.People {
		// Check if same group (shares leader)
		chLeader := ch
		if ch.Leader != nil {
			chLeader = ch.Leader
		}

		if chLeader == leader && !ch.IsNPC() && ch.Position > types.PosSleeping {
			members = append(members, ch)
			totalLevels += ch.Level
		}
	}

	// If no group or solo, give full exp to killer
	if len(members) <= 1 {
		c.GainExp(killer, baseExp)
		return
	}

	// Group bonus: 10% per additional member (up to 50%)
	groupBonus := (len(members) - 1) * 10
	if groupBonus > 50 {
		groupBonus = 50
	}
	totalExp := baseExp * (100 + groupBonus) / 100

	// Distribute exp proportionally by level
	for _, ch := range members {
		share := (totalExp * ch.Level) / totalLevels

		// Minimum 1 exp per share
		if share < 1 {
			share = 1
		}

		c.GainExp(ch, share)
	}
}
