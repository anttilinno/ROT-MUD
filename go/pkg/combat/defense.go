package combat

import (
	"rotmud/pkg/types"
)

// DefenseResult indicates what happened with a defensive check
type DefenseResult int

const (
	DefenseNone DefenseResult = iota
	DefenseDodged
	DefenseParried
	DefenseBlocked
)

// CheckDefenses checks if the victim successfully defends against an attack
// Returns the defense result and a message if defended
func (c *CombatSystem) CheckDefenses(ch, victim *types.Character) DefenseResult {
	// Can't defend if not in combat position
	if victim.Position < types.PosFighting {
		return DefenseNone
	}

	// Can't defend against yourself
	if ch == victim {
		return DefenseNone
	}

	// Check parry (requires weapon)
	if c.checkParry(ch, victim) {
		return DefenseParried
	}

	// Check dodge
	if c.checkDodge(ch, victim) {
		return DefenseDodged
	}

	// Check shield block (requires shield)
	if c.checkShieldBlock(ch, victim) {
		return DefenseBlocked
	}

	return DefenseNone
}

// checkParry checks if the victim parries the attack
func (c *CombatSystem) checkParry(ch, victim *types.Character) bool {
	// Unarmed: mobs parry at half chance, players can't (ROT).
	unarmed := victim.GetEquipment(types.WearLocWield) == nil
	if unarmed && !victim.IsNPC() {
		return false
	}

	// Get parry skill
	parrySkill := c.GetSkill(victim, "parry")
	if parrySkill <= 0 {
		return false
	}

	// ROT check_parry: skill/2, plus the level gap either way.
	chance := parrySkill / 2
	if unarmed {
		chance /= 2
	}
	chance += victim.Level - ch.Level

	if !CanSee(ch, victim) {
		chance /= 2
	}

	// Roll for parry
	if NumberPercent() >= chance {
		return false
	}

	// Parry successful
	if c.Output != nil {
		c.Output(victim, "You parry "+ch.Name+"'s attack.\r\n")
		c.Output(ch, victim.Name+" parries your attack.\r\n")

		// Notify others
		if victim.InRoom != nil {
			for _, person := range victim.InRoom.People {
				if person != ch && person != victim {
					c.Output(person, victim.Name+" parries "+ch.Name+"'s attack.\r\n")
				}
			}
		}
	}

	return true
}

// checkDodge checks if the victim dodges the attack
func (c *CombatSystem) checkDodge(ch, victim *types.Character) bool {
	// Get dodge skill
	dodgeSkill := c.GetSkill(victim, "dodge")
	if dodgeSkill <= 0 {
		return false
	}

	// ROT check_dodge: skill/2, plus the level gap either way.
	chance := dodgeSkill/2 + victim.Level - ch.Level

	if !CanSee(victim, ch) {
		chance /= 2
	}

	// Roll for dodge
	if NumberPercent() >= chance {
		return false
	}

	// Dodge successful
	if c.Output != nil {
		c.Output(victim, "You dodge "+ch.Name+"'s attack.\r\n")
		c.Output(ch, victim.Name+" dodges your attack.\r\n")

		// Notify others
		if victim.InRoom != nil {
			for _, person := range victim.InRoom.People {
				if person != ch && person != victim {
					c.Output(person, victim.Name+" dodges "+ch.Name+"'s attack.\r\n")
				}
			}
		}
	}

	return true
}

// checkShieldBlock checks if the victim blocks the attack with a shield
func (c *CombatSystem) checkShieldBlock(ch, victim *types.Character) bool {
	// Need a shield
	shield := victim.GetEquipment(types.WearLocShield)
	if shield == nil {
		return false
	}

	// Get shield block skill
	shieldSkill := c.GetSkill(victim, "shield block")
	if shieldSkill <= 0 {
		return false
	}

	// ROT check_shield_block: skill/5 + 3, plus the level gap either way.
	chance := shieldSkill/5 + 3 + victim.Level - ch.Level

	// Roll for block
	if NumberPercent() >= chance {
		return false
	}

	// Block successful
	if c.Output != nil {
		c.Output(victim, "You block "+ch.Name+"'s attack with your shield.\r\n")
		c.Output(ch, victim.Name+" blocks your attack with a shield.\r\n")

		// Notify others
		if victim.InRoom != nil {
			for _, person := range victim.InRoom.People {
				if person != ch && person != victim {
					c.Output(person, victim.Name+" blocks "+ch.Name+"'s attack with a shield.\r\n")
				}
			}
		}
	}

	return true
}
