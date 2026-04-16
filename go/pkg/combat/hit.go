package combat

import (
	"strings"

	"rotmud/pkg/types"
)

// OneHit performs one attack from ch against victim
func (c *CombatSystem) OneHit(ch, victim *types.Character, secondary bool) DamageResult {
	result := DamageResult{}

	// Safety checks
	if victim == ch || ch == nil || victim == nil {
		return result
	}

	// Can't hit a dead character
	if victim.Position == types.PosDead {
		return result
	}

	// Must be in the same room
	if ch.InRoom != victim.InRoom {
		return result
	}

	// Get weapon
	var wield *types.Object
	if secondary {
		wield = ch.GetEquipment(types.WearLocSecondary)
	} else {
		wield = ch.GetEquipment(types.WearLocWield)
	}

	// Determine damage type — silver weapons use DamSilver so vampire vulnerability applies
	damType := GetWeaponDamType(ch)
	if wield != nil && strings.ToLower(wield.Material) == "silver" {
		damType = types.DamSilver
	}

	// Get weapon skill based on weapon type
	skill := c.getWeaponSkill(ch, wield)

	// Calculate THAC0
	thac0 := GetThac0(ch)
	thac0 -= GetHitroll(ch) * skill / 100
	thac0 += 5 * (100 - skill) / 100

	// Get victim's AC
	victimAC := GetAC(victim, damType) / 10

	// Cap very low AC
	if victimAC < -15 {
		victimAC = (victimAC+15)/5 - 15
	}

	// Visibility modifiers
	if !CanSee(ch, victim) {
		victimAC -= 4
	}

	// Position modifiers
	if victim.Position < types.PosFighting {
		victimAC += 4
	}
	if victim.Position < types.PosResting {
		victimAC += 6
	}

	// Roll to hit (d20)
	diceroll := NumberBits(5) % 20

	// Check for hit
	// Roll of 0 always misses, roll of 19 always hits
	if diceroll == 0 || (diceroll != 19 && diceroll < thac0-victimAC) {
		// Miss
		result.Missed = true
		c.Damage(ch, victim, 0, damType, true)
		return result
	}

	// Hit! Calculate damage
	var dam int

	if ch.IsNPC() && wield == nil {
		// NPC without weapon uses damage dice
		if ch.Damage[0] > 0 && ch.Damage[1] > 0 {
			dam = Dice(ch.Damage[0], ch.Damage[1]) + ch.Damage[2]
		} else {
			// Fallback: level-based damage
			dam = NumberRange(ch.Level/2, ch.Level*3/2)
		}
	} else if wield != nil {
		// Weapon damage - base dice NOT reduced by skill (skill affects hit chance)
		// ROM formula: dam = dice(wield) * UMIN(skill, 100) / 100
		// But this is too harsh - instead use a gentler formula that gives
		// at least 50% damage even with low skill, scaling to 100% at skill 100
		dam = Dice(wield.DiceNumber(), wield.DiceSize())
		// Apply skill modifier: minimum 50% damage at 0 skill, 100% at 100 skill
		dam = dam * (50 + skill/2) / 100
		// No shield bonus (two-handed grip)
		if ch.GetEquipment(types.WearLocShield) == nil {
			dam = dam * 11 / 10
		}
	} else {
		// Unarmed damage (hand to hand) - skill matters more here
		// Minimum damage based on level, skill improves it
		baseDam := 1 + ch.Level/4
		skillBonus := ch.Level / 2 * skill / 100
		dam = baseDam + skillBonus
		if dam < 1 {
			dam = 1
		}
	}

	// Position bonus
	if !IsAwake(victim) {
		dam *= 2
	} else if victim.Position < types.PosFighting {
		dam = dam * 3 / 2
	}

	// Damroll bonus - not reduced by skill (it's raw bonus damage)
	dam += GetDamroll(ch)

	// Enhanced damage skill bonus
	enhancedDam := c.GetSkill(ch, "enhanced damage")
	if enhancedDam > 0 {
		dam += dam * enhancedDam / 200 // Up to 50% bonus at 100 skill
	}

	// Minimum damage of 1
	if dam < 1 {
		dam = 1
	}

	// Apply damage
	result = c.Damage(ch, victim, dam, damType, true)

	// Check for shield damage reflection
	// Shields damage the attacker when they successfully hit
	if result.Damage > 0 && ch.Fighting == victim {
		c.checkShieldReflection(ch, victim)
	}

	return result
}

// MultiHit performs a round of attacks from ch against victim
func (c *CombatSystem) MultiHit(ch, victim *types.Character) {
	// Decrement wait/daze timers for NPCs
	if ch.Descriptor == nil {
		ch.Wait = Max(0, ch.Wait-3)
		ch.Daze = Max(0, ch.Daze-3)
	}

	// Can't attack while stunned
	if ch.Position < types.PosResting {
		return
	}

	// Primary attack
	c.OneHit(ch, victim, false)
	if ch.Fighting != victim {
		return
	}

	// Haste gives extra attack
	if ch.IsAffected(types.AffHaste) {
		c.OneHit(ch, victim, false)
		if ch.Fighting != victim {
			return
		}
	}

	// Secondary weapon attack (dual wield skill)
	if ch.GetEquipment(types.WearLocSecondary) != nil {
		dualWieldSkill := c.GetSkill(ch, "dual wield")
		chance := dualWieldSkill / 2 // 0-50% chance based on skill
		if ch.IsAffected(types.AffSlow) {
			chance /= 2
		}
		if NumberPercent() < chance {
			c.OneHit(ch, victim, true)
		}
		if ch.Fighting != victim {
			return
		}
	}

	// Second attack skill
	secondAttack := c.GetSkill(ch, "second attack")
	if secondAttack > 0 {
		chance := secondAttack / 2 // Skill gives up to 50% chance
		if ch.IsAffected(types.AffSlow) {
			chance /= 2
		}
		if NumberPercent() < chance {
			c.OneHit(ch, victim, false)
			if ch.Fighting != victim {
				return
			}
		}
	}

	// Third attack skill
	thirdAttack := c.GetSkill(ch, "third attack")
	if thirdAttack > 0 {
		chance := thirdAttack / 3 // Skill gives up to ~33% chance
		if ch.IsAffected(types.AffSlow) {
			chance /= 2
		}
		if NumberPercent() < chance {
			c.OneHit(ch, victim, false)
			if ch.Fighting != victim {
				return
			}
		}
	}

	// Fourth attack skill
	fourthAttack := c.GetSkill(ch, "fourth attack")
	if fourthAttack > 0 {
		chance := fourthAttack / 4 // Skill gives up to 25% chance
		if ch.IsAffected(types.AffSlow) {
			chance /= 2
		}
		if NumberPercent() < chance {
			c.OneHit(ch, victim, false)
			if ch.Fighting != victim {
				return
			}
		}
	}

	// Fifth attack skill (rare)
	fifthAttack := c.GetSkill(ch, "fifth attack")
	if fifthAttack > 0 {
		chance := fifthAttack / 5 // Skill gives up to 20% chance
		if ch.IsAffected(types.AffSlow) {
			chance /= 2
		}
		if NumberPercent() < chance {
			c.OneHit(ch, victim, false)
		}
	}
}

// ViolenceUpdate processes combat for all fighting characters
func (c *CombatSystem) ViolenceUpdate(characters []*types.Character) {
	for _, ch := range characters {
		victim := ch.Fighting
		if victim == nil || ch.InRoom == nil {
			continue
		}

		if IsAwake(ch) && ch.InRoom == victim.InRoom {
			c.MultiHit(ch, victim)
		} else {
			StopFighting(ch, false)
		}
	}
}

// getWeaponSkill returns the character's proficiency with their current weapon
func (c *CombatSystem) getWeaponSkill(ch *types.Character, wield *types.Object) int {
	var skillName string

	if wield == nil {
		// Unarmed combat - use hand to hand skill
		skillName = "hand to hand"
	} else {
		// Get weapon type from weapon flags
		skillName = c.getWeaponTypeName(wield)
	}

	// Get skill level
	skill := c.GetSkill(ch, skillName)

	// Minimum skill of 40 for NPCs
	if ch.IsNPC() && skill < 40 {
		skill = 40 + ch.Level/2
	}

	// Cap at 100
	if skill > 100 {
		skill = 100
	}

	return skill
}

// getWeaponTypeName returns the skill name for a weapon type
func (c *CombatSystem) getWeaponTypeName(wield *types.Object) string {
	if wield == nil || wield.ItemType != types.ItemTypeWeapon {
		return "hand to hand"
	}

	// Weapon type is stored in Values[0]
	// Based on C source weapon_class (const.c)
	weaponType := 0
	if len(wield.Values) > 0 {
		weaponType = wield.Values[0]
	}

	switch weaponType {
	case 0: // WEAPON_EXOTIC
		return "sword" // Default fallback
	case 1: // WEAPON_SWORD
		return "sword"
	case 2: // WEAPON_DAGGER
		return "dagger"
	case 3: // WEAPON_SPEAR
		return "spear"
	case 4: // WEAPON_MACE
		return "mace"
	case 5: // WEAPON_AXE
		return "axe"
	case 6: // WEAPON_FLAIL
		return "flail"
	case 7: // WEAPON_WHIP
		return "whip"
	case 8: // WEAPON_POLEARM
		return "polearm"
	default:
		return "sword"
	}
}
