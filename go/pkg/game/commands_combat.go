package game

import (
	"fmt"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Combat commands: kill, flee, and combat skill wrappers

func (d *CommandDispatcher) cmdKill(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Kill whom?\r\n")
		return
	}

	// Find the target
	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "Suicide is a mortal sin.\r\n")
		return
	}

	// Check if safe
	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	// Check if already fighting someone else
	if ch.Fighting != nil && ch.Fighting != victim {
		d.send(ch, "You're already fighting someone else!\r\n")
		return
	}

	// NPCs can't be killed by each other unless specified (prevent mob-on-mob chaos)
	if ch.IsNPC() && victim.IsNPC() {
		return
	}

	// Check level restrictions for PvP
	if !victim.IsNPC() && !ch.IsNPC() {
		// PvP level check - within 5 levels
		levelDiff := ch.Level - victim.Level
		if levelDiff < 0 {
			levelDiff = -levelDiff
		}
		if levelDiff > 5 && ch.Level < 100 {
			d.send(ch, "Pick on someone your own size.\r\n")
			return
		}
	}

	// Start combat
	d.send(ch, fmt.Sprintf("You attack %s!\r\n", victim.Name))
	ActToRoom("$n attacks $N!", ch, victim, nil, d.Output)

	// Set up fighting
	combat.SetFighting(ch, victim)

	// Perform first attack round if we have a combat system
	if d.Combat != nil {
		d.Combat.MultiHit(ch, victim)
	}
}

func (d *CommandDispatcher) cmdFlee(ch *types.Character, args string) {
	if ch.Fighting == nil {
		d.send(ch, "You aren't fighting anyone.\r\n")
		return
	}

	if ch.Position < types.PosFighting {
		d.send(ch, "You can't flee in your current position.\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You have nowhere to flee to!\r\n")
		return
	}

	// Check for available exits
	var validExits []types.Direction
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := ch.InRoom.GetExit(dir)
		if exit != nil && exit.ToRoom != nil && !exit.IsClosed() {
			validExits = append(validExits, dir)
		}
	}

	if len(validExits) == 0 {
		d.send(ch, "There's nowhere to run!\r\n")
		return
	}

	// Chance to flee based on dexterity
	chance := 75 + (ch.GetStat(types.StatDex)-15)*2

	// Reduced chance if stunned/dazed
	if ch.Daze > 0 {
		chance -= 20
	}

	if combat.NumberPercent() > chance {
		d.send(ch, "PANIC! You couldn't escape!\r\n")
		return
	}

	// Pick a random exit
	exitDir := validExits[combat.NumberRange(0, len(validExits)-1)]

	// Stop fighting
	combat.StopFighting(ch, false)

	// Move to the room
	exit := ch.InRoom.GetExit(exitDir)
	targetRoom := exit.ToRoom
	oldRoom := ch.InRoom

	// Leave message
	ActToRoom("$n flees in panic!", ch, nil, nil, d.Output)

	// Move character
	oldRoom.RemovePerson(ch)
	targetRoom.AddPerson(ch)
	ch.InRoom = targetRoom

	// Arrival message
	d.send(ch, fmt.Sprintf("You flee %s!\r\n", directionName(exitDir)))
	ActToRoom("$n arrives in a panic.", ch, nil, nil, d.Output)

	// Experience penalty for fleeing (small)
	if ch.Exp > 0 && ch.Level > 1 {
		xpLoss := ch.Level * 2
		if xpLoss > ch.Exp {
			xpLoss = ch.Exp
		}
		ch.Exp -= xpLoss
		d.send(ch, fmt.Sprintf("You lose %d experience points.\r\n", xpLoss))
	}

	// Show the new room
	d.cmdLook(ch, "")
}

// directionName returns the name of a direction
func directionName(dir types.Direction) string {
	names := []string{"north", "east", "south", "west", "up", "down"}
	if int(dir) < len(names) {
		return names[dir]
	}
	return "somewhere"
}

// Combat skill commands - these wrap the skills package

func (d *CommandDispatcher) cmdBackstab(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Backstab whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "How can you sneak up on yourself?\r\n")
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	if ch.Fighting != nil {
		d.send(ch, "You're too busy fighting!\r\n")
		return
	}

	// Need a piercing weapon
	wield := ch.GetEquipment(types.WearLocWield)
	if wield == nil || wield.ItemType != types.ItemTypeWeapon {
		d.send(ch, "You need to wield a weapon to backstab.\r\n")
		return
	}

	// Check for piercing damage type
	if wield.DamageType() != types.DamPierce {
		d.send(ch, "You need a piercing weapon to backstab.\r\n")
		return
	}

	// Check if victim is fighting
	if victim.Fighting != nil {
		d.send(ch, "You can't backstab someone who is fighting.\r\n")
		return
	}

	// Perform backstab
	if d.Combat != nil {
		// Calculate backstab chance (simplified - based on level and dex)
		chance := 50 + ch.Level + (ch.GetStat(types.StatDex)-15)*3

		if combat.NumberPercent() < chance {
			// Hit! Calculate damage multiplier
			multiplier := 2 + ch.Level/10
			if multiplier > 5 {
				multiplier = 5
			}

			// Get base weapon damage
			baseDam := combat.Dice(wield.DiceNumber(), wield.DiceSize())
			dam := baseDam * multiplier

			// Add damroll
			dam += combat.GetDamroll(ch)

			d.send(ch, fmt.Sprintf("You backstab %s!\r\n", victim.Name))
			ActToRoom("$n backstabs $N!", ch, victim, nil, d.Output)

			// Apply damage
			d.Combat.Damage(ch, victim, dam, types.DamPierce, true)
		} else {
			// Miss
			d.send(ch, fmt.Sprintf("You fail to backstab %s.\r\n", victim.Name))
			ActToRoom("$n tries to backstab $N but misses.", ch, victim, nil, d.Output)

			// Start combat on miss
			combat.SetFighting(ch, victim)
			d.Combat.MultiHit(ch, victim)
		}
	}
}

func (d *CommandDispatcher) cmdAssassinate(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	if args == "" {
		d.send(ch, "Assassinate whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["assassinate"]
	}
	if skillLevel <= 0 {
		d.send(ch, "You don't know how to assassinate.\r\n")
		return
	}

	if d.Combat == nil {
		return
	}

	result := d.Combat.DoAssassinate(ch, victim)
	if result.Message != "" {
		d.send(ch, result.Message)
		return
	}

	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "assassinate", result.Success, 2)
	}
}

func (d *CommandDispatcher) cmdBash(ch *types.Character, args string) {
	var victim *types.Character

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "Bash whom?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if victim == ch {
		d.send(ch, "You can't bash yourself.\r\n")
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	// Need a shield to bash (optional in some versions)
	// For now, allow it without shield but with reduced effectiveness

	if ch.Daze > 0 {
		d.send(ch, "You're too dazed to bash.\r\n")
		return
	}

	// Calculate bash chance
	chance := 50 + (ch.Level-victim.Level)*2
	chance += (ch.GetStat(types.StatStr) - victim.GetStat(types.StatDex)) * 2

	// Shield bonus
	if ch.GetEquipment(types.WearLocShield) != nil {
		chance += 10
	}

	// Victim size bonus (larger = harder to bash)
	// Simplified - assume same size for now

	// Start combat if not fighting
	if ch.Fighting == nil {
		combat.SetFighting(ch, victim)
	}

	if combat.NumberPercent() < chance {
		// Success!
		d.send(ch, fmt.Sprintf("You bash %s!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s bashes you!\r\n", ch.Name))
		ActToNotVict("$n bashes $N!", ch, victim, nil, d.Output)

		// Deal damage
		dam := combat.NumberRange(1, ch.Level/2)
		dam += ch.GetStat(types.StatStr) - 15

		if d.Combat != nil {
			d.Combat.Damage(ch, victim, dam, types.DamBash, true)
		}

		// Stun the victim
		victim.Daze = 3
		victim.Position = types.PosSitting

		// Wait state for basher
		ch.Wait = 2
	} else {
		// Failure - basher falls down
		d.send(ch, "Your bash misses and you fall down!\r\n")
		ActToRoom("$n tries to bash $N but falls down.", ch, victim, nil, d.Output)

		ch.Position = types.PosSitting
		ch.Daze = 2
	}
}

func (d *CommandDispatcher) cmdKick(ch *types.Character, args string) {
	var victim *types.Character

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "Kick whom?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if victim == ch {
		d.send(ch, "You kick yourself. Ow!\r\n")
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	if ch.Wait > 0 {
		d.send(ch, "You're not ready to kick again.\r\n")
		return
	}

	// Start combat if not fighting
	if ch.Fighting == nil {
		combat.SetFighting(ch, victim)
	}

	// Calculate kick chance
	chance := 50 + ch.Level + (ch.GetStat(types.StatDex)-15)*2

	if combat.NumberPercent() < chance {
		// Hit!
		dam := combat.NumberRange(1, ch.Level)
		dam += ch.GetStat(types.StatStr) - 15

		d.send(ch, fmt.Sprintf("Your kick hits %s!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s kicks you!\r\n", ch.Name))
		ActToNotVict("$n kicks $N!", ch, victim, nil, d.Output)

		if d.Combat != nil {
			d.Combat.Damage(ch, victim, dam, types.DamBash, true)
		}
	} else {
		d.send(ch, "Your kick misses.\r\n")
		ActToRoom("$n tries to kick $N but misses.", ch, victim, nil, d.Output)
	}

	// Wait state
	ch.Wait = 1
}

func (d *CommandDispatcher) cmdTrip(ch *types.Character, args string) {
	var victim *types.Character

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "Trip whom?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if victim == ch {
		d.send(ch, "You trip over your own feet.\r\n")
		ch.Position = types.PosSitting
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	if victim.Position < types.PosStanding {
		d.send(ch, "They're already down.\r\n")
		return
	}

	if ch.Daze > 0 {
		d.send(ch, "You're too dazed to trip anyone.\r\n")
		return
	}

	// Start combat if not fighting
	if ch.Fighting == nil {
		combat.SetFighting(ch, victim)
	}

	// Calculate trip chance
	chance := 50 + (ch.Level-victim.Level)*2
	chance += (ch.GetStat(types.StatDex) - victim.GetStat(types.StatDex)) * 3

	if combat.NumberPercent() < chance {
		// Success!
		d.send(ch, fmt.Sprintf("You trip %s and they go down!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s trips you!\r\n", ch.Name))
		ActToNotVict("$n trips $N, sending them to the ground!", ch, victim, nil, d.Output)

		// Deal minor damage
		dam := combat.NumberRange(1, ch.Level/4)
		if d.Combat != nil {
			d.Combat.Damage(ch, victim, dam, types.DamBash, true)
		}

		// Knock victim down
		victim.Position = types.PosSitting
		victim.Daze = 2

		// Wait state
		ch.Wait = 2
	} else {
		d.send(ch, "You fail to trip them.\r\n")
		ActToRoom("$n tries to trip $N but fails.", ch, victim, nil, d.Output)
		ch.Wait = 1
	}
}

func (d *CommandDispatcher) cmdDisarm(ch *types.Character, args string) {
	var victim *types.Character

	if ch.Fighting == nil {
		d.send(ch, "You aren't fighting anyone.\r\n")
		return
	}

	victim = ch.Fighting

	// Check if victim has a weapon
	victimWield := victim.GetEquipment(types.WearLocWield)
	if victimWield == nil {
		d.send(ch, "Your opponent is not wielding a weapon.\r\n")
		return
	}

	// Attacker needs a weapon too
	if ch.GetEquipment(types.WearLocWield) == nil {
		d.send(ch, "You need to wield a weapon to disarm.\r\n")
		return
	}

	if ch.Daze > 0 {
		d.send(ch, "You're too dazed to disarm.\r\n")
		return
	}

	// Calculate disarm chance
	chance := 50 + (ch.Level-victim.Level)*2
	chance += (ch.GetStat(types.StatDex) - victim.GetStat(types.StatStr)) * 2

	if combat.NumberPercent() < chance {
		// Success!
		d.send(ch, fmt.Sprintf("You disarm %s!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s disarms you!\r\n", ch.Name))
		ActToNotVict("$n disarms $N!", ch, victim, nil, d.Output)

		// Remove the weapon and drop it
		victim.Unequip(types.WearLocWield)
		if victim.InRoom != nil {
			victim.InRoom.AddObject(victimWield)
			victimWield.InRoom = victim.InRoom
			victimWield.CarriedBy = nil
		}
	} else {
		d.send(ch, "You fail to disarm your opponent.\r\n")
		ActToRoom("$n tries to disarm $N but fails.", ch, victim, nil, d.Output)
	}

	// Wait state
	ch.Wait = 2
}

func (d *CommandDispatcher) cmdRescue(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Rescue whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "What about rescuing yourself?\r\n")
		return
	}

	if victim.Fighting == nil {
		d.send(ch, "They aren't fighting anyone.\r\n")
		return
	}

	attacker := victim.Fighting

	// Can't rescue if the attacker is fighting you
	if attacker.Fighting == ch {
		d.send(ch, "They're already attacking you!\r\n")
		return
	}

	// Calculate rescue chance
	chance := 50 + ch.Level - victim.Level

	if combat.NumberPercent() < chance {
		d.send(ch, fmt.Sprintf("You rescue %s!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s rescues you!\r\n", ch.Name))
		ActToNotVict("$n rescues $N!", ch, victim, nil, d.Output)

		// Stop victim from fighting the attacker
		combat.StopFighting(victim, false)

		// Attacker now fights rescuer
		if attacker.Fighting == victim {
			attacker.Fighting = ch
		}

		// Rescuer fights attacker
		combat.SetFighting(ch, attacker)
	} else {
		d.send(ch, "You fail to rescue them.\r\n")
	}

	ch.Wait = 2
}

func (d *CommandDispatcher) cmdDirt(ch *types.Character, args string) {
	var victim *types.Character

	// Get skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["dirt kicking"]
	}
	if ch.IsNPC() {
		skillLevel = 75 // NPCs get reasonable skill
	}

	if skillLevel <= 0 {
		d.send(ch, "You get your feet dirty.\r\n")
		return
	}

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "But you aren't in combat!\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if victim == ch {
		d.send(ch, "Very funny.\r\n")
		return
	}

	if victim.IsAffected(types.AffBlind) {
		d.send(ch, "They're already been blinded.\r\n")
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	if ch.Daze > 0 {
		d.send(ch, "You're still a little woozy.\r\n")
		return
	}

	// Calculate chance based on skill, stats, and terrain
	chance := skillLevel
	chance += ch.GetStat(types.StatDex)
	chance -= 2 * victim.GetStat(types.StatDex)

	// Speed modifiers
	if ch.IsAffected(types.AffHaste) {
		chance += 10
	}
	if victim.IsAffected(types.AffHaste) {
		chance -= 25
	}

	// Level difference
	chance += (ch.Level - victim.Level) * 2

	// Terrain modifiers
	if ch.InRoom != nil {
		switch ch.InRoom.Sector {
		case types.SectInside:
			chance -= 20
		case types.SectCity:
			chance -= 10
		case types.SectField:
			chance += 5
		case types.SectDesert:
			chance += 10
		case types.SectWaterSwim, types.SectWaterNoSwim, types.SectAir:
			chance = 0
		}
	}

	if chance <= 0 {
		d.send(ch, "There isn't any dirt to kick.\r\n")
		return
	}

	// Start combat if not fighting
	if ch.Fighting == nil {
		combat.SetFighting(ch, victim)
	}

	if combat.NumberPercent() < chance {
		// Success - blind the victim
		d.send(ch, fmt.Sprintf("You kick dirt in %s's eyes!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s kicks dirt in your eyes!\r\n", ch.Name))
		ActToNotVict("$n kicks dirt in $N's eyes!", ch, victim, nil, d.Output)

		// Apply blind affect
		aff := &types.Affect{
			Type:      "dirt kicking",
			Level:     ch.Level,
			Duration:  0, // Very short
			Location:  types.ApplyHitroll,
			Modifier:  -4,
			BitVector: types.AffBlind,
		}
		victim.AddAffect(aff)

		d.send(victim, "You can't see a thing!\r\n")

		// Small damage
		dam := combat.NumberRange(2, 5)
		if d.Combat != nil {
			d.Combat.Damage(ch, victim, dam, types.DamNone, true)
		}

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "dirt kicking", true, 2)
		}
	} else {
		// Failure
		d.send(ch, "You kick up dirt but miss their eyes.\r\n")
		ActToRoom("$n kicks up dirt but misses $N's eyes.", ch, victim, nil, d.Output)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "dirt kicking", false, 2)
		}
	}

	ch.Wait = 2
}

func (d *CommandDispatcher) cmdGouge(ch *types.Character, args string) {
	var victim *types.Character

	// Get skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["gouge"]
	}
	if ch.IsNPC() {
		skillLevel = 75
	}

	if skillLevel <= 0 {
		d.send(ch, "Gouge? What's that?\r\n")
		return
	}

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "But you aren't in combat!\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if victim == ch {
		d.send(ch, "Very funny.\r\n")
		return
	}

	if victim.IsAffected(types.AffBlind) {
		d.send(ch, "They've already been blinded.\r\n")
		return
	}

	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	if ch.Daze > 0 {
		d.send(ch, "You're still a little woozy.\r\n")
		return
	}

	// Calculate chance
	chance := skillLevel
	chance += ch.GetStat(types.StatDex)
	chance -= 2 * victim.GetStat(types.StatDex)

	// Speed modifiers
	if ch.IsAffected(types.AffHaste) {
		chance += 10
	}
	if victim.IsAffected(types.AffHaste) {
		chance -= 25
	}

	// Level difference
	chance += (ch.Level - victim.Level) * 2

	// Start combat if not fighting
	if ch.Fighting == nil {
		combat.SetFighting(ch, victim)
	}

	if combat.NumberPercent() < chance {
		// Success - blind the victim
		d.send(ch, fmt.Sprintf("You gouge at %s's eyes!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s gouges at your eyes!\r\n", ch.Name))
		ActToNotVict("$n gouges at $N's eyes!", ch, victim, nil, d.Output)

		// Apply blind affect
		aff := &types.Affect{
			Type:      "gouge",
			Level:     ch.Level,
			Duration:  0,
			Location:  types.ApplyHitroll,
			Modifier:  -4,
			BitVector: types.AffBlind,
		}
		victim.AddAffect(aff)

		d.send(victim, "You see nothing but stars!\r\n")

		// Small damage
		dam := combat.NumberRange(2, 5)
		if d.Combat != nil {
			d.Combat.Damage(ch, victim, dam, types.DamNone, true)
		}

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "gouge", true, 2)
		}
	} else {
		// Failure
		d.send(ch, "You try to gouge their eyes but miss.\r\n")
		ActToRoom("$n tries to gouge $N's eyes but misses.", ch, victim, nil, d.Output)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "gouge", false, 2)
		}
	}

	ch.Wait = 2
}

func (d *CommandDispatcher) cmdCircle(ch *types.Character, args string) {
	// Get skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["circle"]
	}
	if ch.IsNPC() {
		skillLevel = 75
	}

	if skillLevel <= 0 {
		d.send(ch, "Circle? What's that?\r\n")
		return
	}

	victim := ch.Fighting
	if victim == nil {
		d.send(ch, "You aren't fighting anyone.\r\n")
		return
	}

	// Need a weapon
	wield := ch.GetEquipment(types.WearLocWield)
	if wield == nil {
		d.send(ch, "You need to wield a primary weapon to circle.\r\n")
		return
	}

	// Victim can't be too hurt (they become too suspicious)
	if victim.Hit < victim.MaxHit/6 {
		d.send(ch, fmt.Sprintf("%s is hurt and suspicious... you can't sneak around.\r\n", victim.Name))
		return
	}

	if ch.Daze > 0 {
		d.send(ch, "You're still a little woozy.\r\n")
		return
	}

	// Calculate chance
	chance := skillLevel
	if !combat.IsAwake(victim) {
		chance = 100 // Automatic success vs sleeping target
	}

	if combat.NumberPercent() < chance {
		// Success!
		d.send(ch, fmt.Sprintf("You circle around %s!\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s circles around behind you!\r\n", ch.Name))
		ActToNotVict("$n circles around behind $N!", ch, victim, nil, d.Output)

		// Perform a backstab-like attack
		if d.Combat != nil {
			// Calculate damage multiplier (less than backstab)
			multiplier := 1 + ch.Level/15
			if multiplier > 3 {
				multiplier = 3
			}

			// Get base weapon damage
			baseDam := combat.Dice(wield.DiceNumber(), wield.DiceSize())
			dam := baseDam * multiplier
			dam += combat.GetDamroll(ch)

			d.Combat.Damage(ch, victim, dam, wield.DamageType(), true)
		}

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "circle", true, 1)
		}
	} else {
		// Failure
		d.send(ch, fmt.Sprintf("%s circles with you, blocking your attempt.\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s tries to circle around you.\r\n", ch.Name))
		ActToNotVict("$n tries to circle around $N.", ch, victim, nil, d.Output)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "circle", false, 1)
		}
	}

	ch.Wait = 3
}

func (d *CommandDispatcher) cmdBerserk(ch *types.Character, args string) {
	// Get skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["berserk"]
	}
	if ch.IsNPC() {
		skillLevel = 75
	}

	if skillLevel <= 0 {
		d.send(ch, "You turn red in the face, but nothing happens.\r\n")
		return
	}

	// Check if already berserking
	if ch.IsAffected(types.AffBerserk) {
		d.send(ch, "You get a little madder.\r\n")
		return
	}

	// Check for calm spell
	if ch.IsAffected(types.AffCalm) {
		d.send(ch, "You're feeling too mellow to berserk.\r\n")
		return
	}

	// Check mana cost
	if ch.Mana < 50 {
		d.send(ch, "You can't get up enough energy.\r\n")
		return
	}

	// Calculate chance
	chance := skillLevel

	// Fighting bonus
	if ch.Position == types.PosFighting {
		chance += 10
	}

	// HP percentage - below 50% helps, above hurts
	hpPercent := 100 * ch.Hit / ch.MaxHit
	chance += 25 - hpPercent/2

	if combat.NumberPercent() < chance {
		// Success!
		ch.Mana -= 50
		ch.Move /= 2

		// Heal a little
		ch.Hit += ch.Level * 2
		if ch.Hit > ch.MaxHit {
			ch.Hit = ch.MaxHit
		}

		d.send(ch, "Your pulse races as you are consumed by rage!\r\n")
		ActToRoom("$n gets a wild look in their eyes.", ch, nil, nil, d.Output)

		// Apply berserk affect with bonuses
		bonusValue := ch.Level / 5
		if bonusValue < 1 {
			bonusValue = 1
		}
		duration := ch.Level / 8
		if duration < 1 {
			duration = 1
		}

		// Hitroll bonus
		affHit := &types.Affect{
			Type:      "berserk",
			Level:     ch.Level,
			Duration:  duration,
			Location:  types.ApplyHitroll,
			Modifier:  bonusValue,
			BitVector: types.AffBerserk,
		}
		ch.AddAffect(affHit)

		// Damroll bonus
		affDam := &types.Affect{
			Type:      "berserk",
			Level:     ch.Level,
			Duration:  duration,
			Location:  types.ApplyDamroll,
			Modifier:  bonusValue,
			BitVector: 0,
		}
		ch.AddAffect(affDam)

		// AC penalty
		acPenalty := 10 * (ch.Level / 5)
		if acPenalty < 10 {
			acPenalty = 10
		}
		affAC := &types.Affect{
			Type:      "berserk",
			Level:     ch.Level,
			Duration:  duration,
			Location:  types.ApplyAC,
			Modifier:  acPenalty,
			BitVector: 0,
		}
		ch.AddAffect(affAC)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "berserk", true, 2)
		}

		ch.Wait = 1
	} else {
		// Failure
		ch.Mana -= 25
		ch.Move /= 2

		d.send(ch, "Your pulse speeds up, but nothing happens.\r\n")

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "berserk", false, 2)
		}

		ch.Wait = 3
	}
}

func (d *CommandDispatcher) cmdMurder(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Murder whom?\r\n")
		return
	}

	// NPCs don't murder - they use kill
	if ch.IsNPC() {
		return
	}

	// Charmed characters can't initiate murder
	if ch.IsAffected(types.AffCharm) {
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "Suicide is a mortal sin.\r\n")
		return
	}

	// Check if safe
	if combat.IsSafe(ch, victim) {
		d.send(ch, "You can't attack them here.\r\n")
		return
	}

	// Can't kill-steal from other players
	if victim.IsNPC() && victim.Fighting != nil && victim.Fighting != ch {
		// Check if the person fighting victim is in our group
		if victim.Fighting.Leader != ch && ch.Leader != victim.Fighting {
			d.send(ch, "Kill stealing is not permitted.\r\n")
			return
		}
	}

	// Can't attack your own master
	if ch.IsAffected(types.AffCharm) && ch.Master == victim {
		d.send(ch, fmt.Sprintf("%s is your beloved master.\r\n", victim.Name))
		return
	}

	// Already fighting
	if ch.Position == types.PosFighting {
		d.send(ch, "You do the best you can!\r\n")
		return
	}

	// Wait state
	ch.Wait = 1

	// Victim yells for help
	var helpMsg string
	if ch.IsNPC() {
		helpMsg = fmt.Sprintf("Help! I am being attacked by %s!", ch.ShortDesc)
	} else {
		helpMsg = fmt.Sprintf("Help! I am being attacked by %s!", ch.Name)
	}
	d.cmdYell(victim, helpMsg)

	// Start combat
	combat.SetFighting(ch, victim)
	if d.Combat != nil {
		d.Combat.MultiHit(ch, victim)
	}
}

func (d *CommandDispatcher) cmdSurrender(ch *types.Character, args string) {
	if ch.Fighting == nil {
		d.send(ch, "But you're not fighting!\r\n")
		return
	}

	mob := ch.Fighting

	d.send(ch, fmt.Sprintf("You surrender to %s!\r\n", mob.Name))
	d.send(mob, fmt.Sprintf("%s surrenders to you!\r\n", ch.Name))
	ActToNotVict("$n tries to surrender to $N!", ch, mob, nil, d.Output)

	// Stop fighting
	combat.StopFighting(ch, true)

	// If surrendering to an NPC, check for surrender trigger
	// If no trigger or trigger doesn't accept, mob continues attacking
	if !ch.IsNPC() && mob.IsNPC() {
		// TODO: Check for MOBprog TRIG_SURR trigger
		// For now, just have mob ignore surrender and attack
		d.send(ch, fmt.Sprintf("%s seems to ignore your cowardly act!\r\n", mob.Name))
		if d.Combat != nil {
			d.Combat.MultiHit(mob, ch)
		}
	}
}

func (d *CommandDispatcher) cmdStun(ch *types.Character, args string) {
	var victim *types.Character

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "Stun whom?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if d.Combat != nil {
		result := d.Combat.DoStun(ch, victim)
		if result.Message != "" {
			d.send(ch, result.Message)
		}
	}
}

func (d *CommandDispatcher) cmdFeed(ch *types.Character, args string) {
	var victim *types.Character

	if args == "" {
		victim = ch.Fighting
		if victim == nil {
			d.send(ch, "Feed on whom?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}
	}

	if d.Combat != nil {
		result := d.Combat.DoFeed(ch, victim)
		if result.Message != "" {
			d.send(ch, result.Message)
		}
	}
}
