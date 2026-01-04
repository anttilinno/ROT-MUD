package combat

import (
	"fmt"

	"rotmud/pkg/types"
)

// Combat skill execution functions.
// These are called when a player uses a combat skill like backstab, kick, etc.

// SkillResult contains the result of executing a combat skill
type SkillResult struct {
	Success bool   // Did the skill work?
	Damage  int    // Damage dealt (if any)
	Message string // Error or info message
}

// DoBackstab executes a backstab attack
func (c *CombatSystem) DoBackstab(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Must have a weapon
	weapon := ch.GetEquipment(types.WearLocWield)
	if weapon == nil {
		result.Message = "You need to wield a weapon to backstab.\r\n"
		return result
	}

	// Can't backstab if already fighting
	if ch.Fighting != nil {
		result.Message = "You're facing the wrong end.\r\n"
		return result
	}

	// Can't backstab yourself
	if victim == ch {
		result.Message = "How can you sneak up on yourself?\r\n"
		return result
	}

	// Check safety
	if IsSafe(ch, victim) {
		result.Message = "You cannot attack them.\r\n"
		return result
	}

	// Can't backstab hurt victims - they're suspicious
	if victim.Hit < victim.MaxHit/3 {
		result.Message = fmt.Sprintf("%s is hurt and suspicious... you can't sneak up.\r\n", victim.Name)
		return result
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["backstab"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2 // NPCs get level-based skill
	}

	// Add command lag (24 = 3 seconds roughly)
	ch.Wait = 24

	// Check for success
	if NumberPercent() < skillLevel || !IsAwake(victim) {
		// Success! Deal backstab damage
		// Backstab does multiplied damage based on level
		multiplier := 2
		if ch.Level >= 10 {
			multiplier = 3
		}
		if ch.Level >= 20 {
			multiplier = 4
		}
		if ch.Level >= 30 {
			multiplier = 5
		}

		// Calculate damage
		dam := c.calculateWeaponDamage(ch, weapon) * multiplier

		// Start combat
		SetFighting(ch, victim)

		// Deal damage
		c.Damage(ch, victim, dam, types.DamPierce, true)

		result.Success = true
		result.Damage = dam
	} else {
		// Failure - still start combat
		SetFighting(ch, victim)
		c.Damage(ch, victim, 0, types.DamPierce, true)
		result.Success = false
	}

	return result
}

// DoBash executes a bash attack
func (c *CombatSystem) DoBash(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["bash"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "Bashing? What's that?\r\n"
		return result
	}

	// Must be fighting or specify target
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "But you aren't fighting anyone!\r\n"
		return result
	}

	// Can't bash downed opponents
	if victim.Position < types.PosFighting {
		result.Message = "You'll have to let them get back up first.\r\n"
		return result
	}

	if victim == ch {
		result.Message = "You try to bash your brains out, but fail.\r\n"
		return result
	}

	if IsSafe(ch, victim) {
		result.Message = "You cannot attack them.\r\n"
		return result
	}

	// Can't see the target
	if !CanSee(ch, victim) {
		result.Message = "You get a running start, and slam right into a wall.\r\n"
		return result
	}

	// Calculate chance
	chance := skillLevel

	// Size modifier
	if int(ch.Size) < int(victim.Size) {
		chance += (int(ch.Size) - int(victim.Size)) * 15
	} else {
		chance += (int(ch.Size) - int(victim.Size)) * 10
	}

	// Stats
	chance += ch.GetStat(types.StatStr)
	chance -= victim.GetStat(types.StatDex) * 4 / 3

	// Level difference
	chance += (ch.Level - victim.Level)

	// Add lag
	ch.Wait = 24

	// Start combat if not already fighting
	if ch.Fighting == nil {
		SetFighting(ch, victim)
	}

	// Roll for success
	if NumberPercent() < chance {
		// Success!
		victim.Position = types.PosResting
		victim.Daze = 3 // Daze for 3 violence ticks

		// Calculate damage
		dam := NumberRange(2, 2+2*int(ch.Size)+chance/20)

		c.Damage(ch, victim, dam, types.DamBash, false)

		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You slam into %s, and send them flying!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("%s sends you sprawling with a powerful bash!\r\n", ch.Name))
		}

		result.Success = true
		result.Damage = dam
	} else {
		// Failure - fall down
		c.Damage(ch, victim, 0, types.DamBash, false)
		ch.Position = types.PosResting

		if c.Output != nil {
			c.Output(ch, "You fall flat on your face!\r\n")
			c.Output(victim, fmt.Sprintf("%s falls flat on their face.\r\n", ch.Name))
		}

		ch.Wait = 36 // Extra lag on failure
		result.Success = false
	}

	return result
}

// DoKick executes a kick attack
func (c *CombatSystem) DoKick(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["kick"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "You better leave the martial arts to fighters.\r\n"
		return result
	}

	// Must be fighting
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "You aren't fighting anyone.\r\n"
		return result
	}

	// Add lag
	ch.Wait = 12

	// Roll for success
	if NumberPercent() < skillLevel {
		// Success!
		dam := NumberRange(1, ch.Level) + NumberRange(0, ch.Level/2)
		c.Damage(ch, victim, dam, types.DamBash, true)

		result.Success = true
		result.Damage = dam
	} else {
		// Failure
		c.Damage(ch, victim, 0, types.DamBash, true)
		result.Success = false
	}

	return result
}

// DoTrip executes a trip attack
func (c *CombatSystem) DoTrip(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["trip"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "Tripping? What's that?\r\n"
		return result
	}

	// Must be fighting or specify target
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "But you aren't fighting anyone!\r\n"
		return result
	}

	if IsSafe(ch, victim) {
		result.Message = "You cannot attack them.\r\n"
		return result
	}

	// Can't trip flying targets
	if victim.IsAffected(types.AffFlying) {
		result.Message = "Their feet aren't on the ground.\r\n"
		return result
	}

	// Can't trip downed targets
	if victim.Position < types.PosFighting {
		result.Message = "They are already down.\r\n"
		return result
	}

	if victim == ch {
		result.Message = "You fall flat on your face!\r\n"
		ch.Position = types.PosResting
		ch.Wait = 24
		return result
	}

	// Calculate chance
	chance := skillLevel

	// Size modifier
	if int(ch.Size) < int(victim.Size) {
		chance += (int(ch.Size) - int(victim.Size)) * 10
	}

	// Dex modifier
	chance += ch.GetStat(types.StatDex)
	chance -= victim.GetStat(types.StatDex) * 3 / 2

	// Level modifier
	chance += (ch.Level - victim.Level) * 2

	// Add lag
	ch.Wait = 16

	// Start combat if not already fighting
	if ch.Fighting == nil {
		SetFighting(ch, victim)
	}

	// Roll for success
	if NumberPercent() < chance {
		// Success!
		victim.Position = types.PosResting
		victim.Daze = 2

		dam := NumberRange(2, 2+2*int(victim.Size))
		c.Damage(ch, victim, dam, types.DamBash, true)

		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You trip %s and they go down!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("%s trips you and you go down!\r\n", ch.Name))
		}

		result.Success = true
		result.Damage = dam
	} else {
		// Failure
		c.Damage(ch, victim, 0, types.DamBash, true)
		ch.Wait = 24 // Extra lag
		result.Success = false
	}

	return result
}

// DoDisarm executes a disarm attempt
func (c *CombatSystem) DoDisarm(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["disarm"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "You don't know how to disarm opponents.\r\n"
		return result
	}

	// Must have a weapon or hand to hand skill
	chWeapon := ch.GetEquipment(types.WearLocWield)
	handToHand := 0
	if ch.PCData != nil {
		handToHand = ch.PCData.Learned["hand to hand"]
	}

	if chWeapon == nil && handToHand == 0 {
		result.Message = "You must wield a weapon to disarm.\r\n"
		return result
	}

	// Must be fighting
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "You aren't fighting anyone.\r\n"
		return result
	}

	// Target must have a weapon
	victimWeapon := victim.GetEquipment(types.WearLocWield)
	if victimWeapon == nil {
		result.Message = "Your opponent is not wielding a weapon.\r\n"
		return result
	}

	// Calculate chance
	chance := skillLevel

	// Stats
	chance += ch.GetStat(types.StatDex)
	chance -= 2 * victim.GetStat(types.StatStr)

	// Level
	chance += (ch.Level - victim.Level) * 2

	chance /= 2

	// Add lag
	ch.Wait = 16

	// Roll for success
	if NumberPercent() < chance {
		// Check for grip skill
		gripSkill := 0
		if victim.PCData != nil {
			gripSkill = victim.PCData.Learned["grip"]
		}

		if gripSkill > 0 && NumberPercent() < (gripSkill*4)/5 {
			// Grip prevented disarm
			if c.Output != nil {
				c.Output(ch, fmt.Sprintf("%s grips their weapon tightly and you fail to disarm them.\r\n", victim.Name))
				c.Output(victim, fmt.Sprintf("You grip your weapon tightly and %s fails to disarm you.\r\n", ch.Name))
			}
			result.Success = false
			return result
		}

		// Success! Disarm them
		victim.Unequip(types.WearLocWield)
		victim.AddInventory(victimWeapon)

		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You disarm %s!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("%s disarms you!\r\n", ch.Name))
		}

		result.Success = true
	} else {
		// Failure
		if c.Output != nil {
			c.Output(ch, "You failed to disarm your opponent.\r\n")
		}
		result.Success = false
	}

	return result
}

// DoStun executes a stun attack
func (c *CombatSystem) DoStun(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["stun"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "You don't know how to stun opponents.\r\n"
		return result
	}

	// Must be fighting
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "You aren't fighting anyone.\r\n"
		return result
	}

	if victim == ch {
		result.Message = "You can't stun yourself.\r\n"
		return result
	}

	if IsSafe(ch, victim) {
		result.Message = "You cannot attack them.\r\n"
		return result
	}

	// Can't stun already dazed targets
	if victim.Daze > 0 {
		result.Message = "They are already stunned.\r\n"
		return result
	}

	// Calculate chance
	chance := skillLevel

	// Stats modifiers
	chance += ch.GetStat(types.StatStr)
	chance -= victim.GetStat(types.StatCon)

	// Level difference
	chance += (ch.Level - victim.Level) * 2

	// Size modifier - harder to stun larger targets
	if int(victim.Size) > int(ch.Size) {
		chance -= (int(victim.Size) - int(ch.Size)) * 10
	}

	// Add lag
	ch.Wait = 24

	// Start combat if not already fighting
	if ch.Fighting == nil {
		SetFighting(ch, victim)
	}

	// Roll for success
	if NumberPercent() < chance {
		// Success! Stun the victim
		victim.Daze = 3 + ch.Level/10 // Daze for 3-8 violence ticks based on level

		// Small damage from the blow
		dam := NumberRange(1, ch.Level/2)
		c.Damage(ch, victim, dam, types.DamBash, false)

		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You strike %s with a stunning blow!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("%s strikes you with a stunning blow! You see stars!\r\n", ch.Name))
		}

		result.Success = true
		result.Damage = dam
	} else {
		// Failure
		c.Damage(ch, victim, 0, types.DamBash, false)

		if c.Output != nil {
			c.Output(ch, "Your stunning blow misses its mark.\r\n")
		}

		ch.Wait = 32 // Extra lag on failure
		result.Success = false
	}

	return result
}

// DoFeed executes a vampire feed/bite attack
func (c *CombatSystem) DoFeed(ch, victim *types.Character) SkillResult {
	result := SkillResult{}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["feed"]
	}
	if ch.IsNPC() {
		skillLevel = ch.Level * 2
	}

	if skillLevel == 0 {
		result.Message = "Feed? What's that?\r\n"
		return result
	}

	// Must be fighting
	if victim == nil {
		victim = ch.Fighting
	}

	if victim == nil {
		result.Message = "You aren't fighting anyone.\r\n"
		return result
	}

	// Can't feed on hurt victims - they're suspicious
	if victim.Hit < victim.MaxHit/6 {
		result.Message = fmt.Sprintf("%s is hurt and suspicious... you can't get close enough.\r\n", victim.Name)
		return result
	}

	// Check if stunned
	if ch.Daze > 0 {
		result.Message = "You're still a little woozy.\r\n"
		return result
	}

	// Add lag
	ch.Wait = 16

	// Roll for success (skill/3 is harder than most attacks)
	if NumberPercent() < skillLevel/3 || (skillLevel >= 2 && !IsAwake(victim)) {
		// Success! Bite damage with negative energy
		// Damage based on average of both levels
		avgLevel := (ch.Level + victim.Level) / 2
		dam := NumberRange(avgLevel/3, (avgLevel/3)*2)

		c.Damage(ch, victim, dam, types.DamNegative, true)

		if c.Output != nil {
			c.Output(ch, fmt.Sprintf("You bite %s!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("%s bites you!\r\n", ch.Name))
		}

		result.Success = true
		result.Damage = dam
	} else {
		// Failure
		c.Damage(ch, victim, 0, types.DamNegative, true)

		if c.Output != nil {
			c.Output(ch, "You chomp a mouthful of air.\r\n")
			c.Output(victim, fmt.Sprintf("%s tries to bite you, but hits only air.\r\n", ch.Name))
		}

		result.Success = false
	}

	return result
}

// calculateWeaponDamage calculates damage for a weapon
func (c *CombatSystem) calculateWeaponDamage(ch *types.Character, weapon *types.Object) int {
	if weapon == nil {
		return 0
	}

	// Use weapon damage dice (stored in Values)
	diceNum := weapon.Values[1]
	diceSize := weapon.Values[2]

	if diceNum <= 0 {
		diceNum = 1
	}
	if diceSize <= 0 {
		diceSize = 4
	}

	dam := Dice(diceNum, diceSize)

	// Add damroll bonus
	dam += GetDamroll(ch)

	return dam
}
