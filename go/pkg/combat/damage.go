package combat

import (
	"fmt"

	"rotmud/pkg/types"
)

// DamageResult contains the result of a damage operation
type DamageResult struct {
	Damage int  // Actual damage dealt
	Killed bool // True if victim was killed
	Immune bool // True if victim was immune
	Missed bool // True if attack missed
}

// Damage inflicts damage on a victim
// Returns true if the victim survives
func (c *CombatSystem) Damage(ch, victim *types.Character, dam int, damType types.DamageType, showMessage bool) DamageResult {
	result := DamageResult{}

	// Already dead
	if victim.Position == types.PosDead {
		return result
	}

	// Safety check
	if IsSafe(ch, victim) {
		return result
	}

	// Start combat if not fighting
	if victim != ch {
		if victim.Position > types.PosStunned {
			if victim.Fighting == nil {
				SetFighting(victim, ch)
			}
			if victim.Position != types.PosFighting {
				victim.Position = types.PosFighting
			}
		}

		if ch.Fighting == nil {
			SetFighting(ch, victim)
		}
	}

	// Check defensive skills (parry, dodge, shield block) for melee attacks
	if dam > 0 && ch != victim {
		defense := c.CheckDefenses(ch, victim)
		if defense != DefenseNone {
			// Attack was defended
			return result
		}
	}

	// Check immunity
	switch CheckImmune(victim, damType) {
	case ImmImmune:
		result.Immune = true
		dam = 0
	case ImmResistant:
		dam = dam * 2 / 3
	case ImmVulnerable:
		dam = dam * 3 / 2
	}

	// Sanctuary halves damage
	if dam > 1 && victim.IsAffected(types.AffSanctuary) {
		dam /= 2
	}

	// Drunk victims take 10% less damage (too numb to feel it)
	if dam > 1 && !victim.IsNPC() && victim.PCData != nil {
		if victim.PCData.Condition[types.CondDrunk] > 10 {
			dam = dam * 9 / 10
		}
	}

	// Damage reduction for high damage
	if dam > 35 {
		dam = (dam-35)/2 + 35
	}
	if dam > 80 {
		dam = (dam-80)/2 + 80
	}

	// Show damage message
	if showMessage {
		c.DamageMessage(ch, victim, dam, damType, result.Immune)
	}

	if dam == 0 {
		return result
	}

	result.Damage = dam

	// Record damage for metrics
	if c.OnDamage != nil {
		c.OnDamage(dam)
	}

	// Apply damage
	victim.Hit -= dam

	// Immortals don't die
	if victim.IsImmortal() && victim.Hit < 1 {
		victim.Hit = 1
	}

	// Update position based on HP
	UpdatePosition(victim)

	// Check for death
	if victim.Position == types.PosDead {
		result.Killed = true
		c.HandleDeath(ch, victim)
	} else {
		// Inform victim of condition
		c.sendConditionMessage(victim)
	}

	return result
}

// DamageMessage sends the damage message to participants
func (c *CombatSystem) DamageMessage(ch, victim *types.Character, dam int, damType types.DamageType, immune bool) {
	if c.Output == nil {
		return
	}

	var punct string
	var vp string

	if dam == 0 {
		vp = "miss"
		punct = "."
	} else if dam <= 4 {
		vp = "scratch"
		punct = "."
	} else if dam <= 8 {
		vp = "graze"
		punct = "."
	} else if dam <= 12 {
		vp = "hit"
		punct = "."
	} else if dam <= 16 {
		vp = "injure"
		punct = "."
	} else if dam <= 20 {
		vp = "wound"
		punct = "."
	} else if dam <= 24 {
		vp = "maul"
		punct = "."
	} else if dam <= 28 {
		vp = "decimate"
		punct = "!"
	} else if dam <= 32 {
		vp = "devastate"
		punct = "!"
	} else if dam <= 36 {
		vp = "maim"
		punct = "!"
	} else if dam <= 42 {
		vp = "MUTILATE"
		punct = "!"
	} else if dam <= 52 {
		vp = "DISEMBOWEL"
		punct = "!!"
	} else if dam <= 65 {
		vp = "DISMEMBER"
		punct = "!!"
	} else if dam <= 80 {
		vp = "MASSACRE"
		punct = "!!"
	} else if dam <= 100 {
		vp = "MANGLE"
		punct = "!!!"
	} else if dam <= 130 {
		vp = "*** DEMOLISH ***"
		punct = "!!!"
	} else if dam <= 175 {
		vp = "*** DEVASTATE ***"
		punct = "!!!"
	} else if dam <= 250 {
		vp = "=== OBLITERATE ==="
		punct = "!!!"
	} else {
		vp = ">>> ANNIHILATE <<<"
		punct = "!!!!"
	}

	// Attacker sees
	if immune {
		c.Output(ch, fmt.Sprintf("Your attack is ineffective against %s%s\r\n", victim.Name, punct))
	} else {
		c.Output(ch, fmt.Sprintf("Your attack %s %s%s\r\n", vp, victim.Name, punct))
	}

	// Victim sees
	if immune {
		c.Output(victim, fmt.Sprintf("%s's attack is ineffective against you%s\r\n", ch.Name, punct))
	} else {
		c.Output(victim, fmt.Sprintf("%s's attack %s you%s\r\n", ch.Name, vp, punct))
	}

	// Others in room see
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person == ch || person == victim {
				continue
			}
			if immune {
				c.Output(person, fmt.Sprintf("%s's attack is ineffective against %s%s\r\n", ch.Name, victim.Name, punct))
			} else {
				c.Output(person, fmt.Sprintf("%s's attack %s %s%s\r\n", ch.Name, vp, victim.Name, punct))
			}
		}
	}
}

// sendConditionMessage informs a character about their health status
func (c *CombatSystem) sendConditionMessage(victim *types.Character) {
	if c.Output == nil {
		return
	}

	switch victim.Position {
	case types.PosMortal:
		c.Output(victim, "You are mortally wounded, and will die soon if not aided.\r\n")
	case types.PosIncap:
		c.Output(victim, "You are incapacitated and will slowly die, if not aided.\r\n")
	case types.PosStunned:
		c.Output(victim, "You are stunned, but will probably recover.\r\n")
	}
}

// HandleDeath processes the death of a character
func (c *CombatSystem) HandleDeath(killer, victim *types.Character) {
	// Death cry to adjacent rooms
	c.deathCry(victim)

	// Stop fighting
	StopFighting(victim, true)

	// Death message
	if c.Output != nil {
		c.Output(victim, "You have been KILLED!\r\n")

		// Notify room
		if victim.InRoom != nil {
			for _, person := range victim.InRoom.People {
				if person != victim {
					c.Output(person, fmt.Sprintf("%s is DEAD!\r\n", victim.Name))
				}
			}
		}
	}

	// Award experience to killer (for NPC kills)
	if victim.IsNPC() && killer != nil && !killer.IsNPC() {
		c.GroupGain(killer, victim)
	}

	// Trigger quest callbacks
	if c.OnKill != nil && killer != nil {
		c.OnKill(killer, victim)
	}

	// Create corpse (for both NPCs and players)
	corpse := c.makeCorpse(victim)

	if victim.IsNPC() {
		// NPC death - remove from room
		if victim.InRoom != nil {
			victim.InRoom.RemovePerson(victim)
			victim.InRoom = nil
		}
	} else {
		// Player death
		c.handlePlayerDeath(killer, victim)
	}

	// Trigger autoloot/autosac callback
	if c.OnDeath != nil && corpse != nil {
		c.OnDeath(killer, victim, corpse)
	}
}

// handlePlayerDeath handles player-specific death processing
func (c *CombatSystem) handlePlayerDeath(killer, victim *types.Character) {
	if c.Output != nil {
		c.Output(victim, "You feel yourself leaving your body.\r\n")
	}

	// Apply death penalty - lose XP
	xpLoss := c.calculateXPLoss(victim)
	if xpLoss > 0 {
		victim.Exp -= xpLoss
		if c.Output != nil {
			c.Output(victim, fmt.Sprintf("You lose %d experience points.\r\n", xpLoss))
		}
	}

	// Apply death penalty - lose 10% of gold on person (not in bank)
	goldLoss := victim.Gold / 10
	if goldLoss > 0 {
		victim.Gold -= goldLoss
		if c.Output != nil {
			c.Output(victim, fmt.Sprintf("You lose %d gold coins.\r\n", goldLoss))
		}
	}

	// Clear affects
	victim.AffectedBy = 0
	victim.Affected = types.AffectList{} // Reset to empty list

	// Reset armor
	for i := 0; i < 4; i++ {
		victim.Armor[i] = 100
	}

	// Find recall room and move player there
	recallRoom := c.findRecallRoom(victim)
	if recallRoom != nil && c.CharMover != nil {
		c.CharMover(victim, recallRoom)
	}

	// Restore player to alive state
	victim.Position = types.PosResting
	if victim.Hit < 1 {
		victim.Hit = 1
	}
	if victim.Mana < 1 {
		victim.Mana = 1
	}
	if victim.Move < 1 {
		victim.Move = 1
	}

	// Show the recall room
	if c.Output != nil && recallRoom != nil {
		c.Output(victim, "\r\nYou awaken in the temple.\r\n\r\n")
	}
}

// calculateXPLoss calculates XP penalty for death
// Lose 10% of experience accrued toward the next level
func (c *CombatSystem) calculateXPLoss(ch *types.Character) int {
	// Calculate XP needed for current level
	expForCurrentLevel := ExpToLevel(ch.Level)

	// Account for overspent creation points if applicable
	if ch.PCData != nil && ch.PCData.OverspentPoints > 0 {
		expForCurrentLevel = ExpToLevelWithPenalty(ch.Level, ch.PCData.OverspentPoints)
	}

	// Experience toward next level
	expTowardNextLevel := ch.Exp - expForCurrentLevel
	if expTowardNextLevel < 0 {
		expTowardNextLevel = 0
	}

	// Lose 10% of progress toward next level
	loss := expTowardNextLevel / 10

	// Don't lose more than current XP (can't go negative)
	if loss > ch.Exp {
		loss = ch.Exp
	}

	return loss
}

// findRecallRoom finds the appropriate recall room for the player
func (c *CombatSystem) findRecallRoom(ch *types.Character) *types.Room {
	if c.RoomFinder == nil {
		return nil
	}

	// Check player's personal recall point first
	if ch.PCData != nil && ch.PCData.Recall != 0 {
		room := c.RoomFinder(ch.PCData.Recall)
		if room != nil {
			return room
		}
	}

	// Try default temple
	room := c.RoomFinder(RoomVnumTemple)
	if room != nil {
		return room
	}

	// Try alternate temple
	room = c.RoomFinder(RoomVnumTempleB)
	if room != nil {
		return room
	}

	return nil
}

// deathCry sends death messages to adjacent rooms
func (c *CombatSystem) deathCry(ch *types.Character) {
	if c.Output == nil || ch.InRoom == nil {
		return
	}

	// Pick a death message
	messages := []string{
		"You hear $n's death cry.",
		"$n hits the ground ... DEAD.",
		"$n splatters blood on your armor.",
	}

	msg := messages[NumberRange(0, len(messages)-1)]

	// Send to adjacent rooms
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := ch.InRoom.GetExit(dir)
		if exit == nil || exit.ToRoom == nil || exit.ToRoom == ch.InRoom {
			continue
		}

		// Format message with character name
		formattedMsg := formatDeathCry(msg, ch)

		for _, person := range exit.ToRoom.People {
			c.Output(person, formattedMsg+"\r\n")
		}
	}
}

// formatDeathCry replaces $n with character name
func formatDeathCry(msg string, ch *types.Character) string {
	name := ch.Name
	if ch.IsNPC() && ch.ShortDesc != "" {
		name = ch.ShortDesc
	}

	result := ""
	for i := 0; i < len(msg); i++ {
		if i < len(msg)-1 && msg[i] == '$' && msg[i+1] == 'n' {
			result += name
			i++ // Skip 'n'
		} else {
			result += string(msg[i])
		}
	}
	return result
}

// makeCorpse creates a corpse object from a dead character
func (c *CombatSystem) makeCorpse(ch *types.Character) *types.Object {
	// Create a corpse object
	var corpse *types.Object
	if ch.IsNPC() {
		corpse = types.NewObject(0, "the corpse of "+ch.ShortDesc, types.ItemTypeCorpseNPC)
	} else {
		corpse = types.NewObject(0, "the corpse of "+ch.Name, types.ItemTypeCorpsePC)
	}

	corpse.Name = "corpse"
	corpse.LongDesc = "The corpse of " + ch.Name + " is lying here."
	corpse.Timer = 10 // Corpse decays after 10 ticks

	// For player corpses, track owner and noloot status
	if !ch.IsNPC() {
		corpse.Owner = ch.Name
		// Store noloot flag in Values[4] (1 = noloot)
		if ch.PlayerAct.Has(types.PlrNoLoot) {
			corpse.Values[4] = 1
		}
	}

	// Transfer inventory to corpse (for NPCs) or leave on corpse (for players)
	for len(ch.Inventory) > 0 {
		obj := ch.Inventory[0]
		ch.RemoveInventory(obj)
		corpse.AddContent(obj)
	}

	// Drop equipment into corpse
	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		obj := ch.Unequip(i)
		if obj != nil {
			corpse.AddContent(obj)
		}
	}

	// Add gold to corpse
	if ch.Gold > 0 {
		goldObj := types.NewObject(0, fmt.Sprintf("%d gold coins", ch.Gold), types.ItemTypeMoney)
		goldObj.Name = "gold coins"
		goldObj.Cost = ch.Gold
		corpse.AddContent(goldObj)
		ch.Gold = 0
	}

	// Place corpse in room
	if ch.InRoom != nil {
		corpse.InRoom = ch.InRoom
		ch.InRoom.AddObject(corpse)
	}

	return corpse
}

// checkShieldReflection applies elemental shield damage to attackers
// When someone attacks a shielded target, they take damage back
func (c *CombatSystem) checkShieldReflection(attacker, victim *types.Character) {
	// Ice shield - cold damage to attacker (unless they also have ice shield)
	if victim.IsShielded(types.ShdIce) && !attacker.IsShielded(types.ShdIce) {
		dam := NumberRange(5, 15)
		if c.Output != nil {
			c.Output(attacker, fmt.Sprintf("%s's icy shield freezes you!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("Your icy shield freezes %s!\r\n", attacker.Name))
		}
		c.Damage(victim, attacker, dam, types.DamCold, false)
	}

	// Fire shield - fire damage to attacker (unless they also have fire shield)
	if victim.IsShielded(types.ShdFire) && !attacker.IsShielded(types.ShdFire) {
		dam := NumberRange(10, 20)
		if c.Output != nil {
			c.Output(attacker, fmt.Sprintf("%s's fiery shield burns you!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("Your fiery shield burns %s!\r\n", attacker.Name))
		}
		c.Damage(victim, attacker, dam, types.DamFire, false)
	}

	// Shock shield - lightning damage to attacker (unless they also have shock shield)
	if victim.IsShielded(types.ShdShock) && !attacker.IsShielded(types.ShdShock) {
		dam := NumberRange(15, 25)
		if c.Output != nil {
			c.Output(attacker, fmt.Sprintf("%s's crackling shield shocks you!\r\n", victim.Name))
			c.Output(victim, fmt.Sprintf("Your crackling shield shocks %s!\r\n", attacker.Name))
		}
		c.Damage(victim, attacker, dam, types.DamLightning, false)
	}
}
