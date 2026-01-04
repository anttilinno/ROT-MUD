package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Thief commands: sneak, hide, steal, peek

func (d *CommandDispatcher) cmdSneak(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	// Check if already sneaking
	if ch.IsAffected(types.AffSneak) {
		d.send(ch, "You are already sneaking.\r\n")
		return
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["sneak"]
	}

	if skillLevel <= 0 {
		d.send(ch, "You don't know how to move silently.\r\n")
		return
	}

	// Check for sneak skill check
	if combat.NumberPercent() >= skillLevel {
		d.send(ch, "You fail to move silently.\r\n")
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "sneak", false, 3)
		}
		return
	}

	// Apply sneak affect
	aff := &types.Affect{
		Type:      "sneak",
		Level:     ch.Level,
		Duration:  ch.Level,
		Location:  types.ApplyNone,
		Modifier:  0,
		BitVector: types.AffSneak,
	}
	ch.AddAffect(aff)

	d.send(ch, "You begin to move silently.\r\n")

	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "sneak", true, 3)
	}
}

func (d *CommandDispatcher) cmdHide(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	// Can't hide while fighting
	if ch.InCombat() {
		d.send(ch, "You can't hide while fighting!\r\n")
		return
	}

	// Remove any existing hide affect first
	if ch.IsAffected(types.AffHide) {
		ch.AffectedBy.Remove(types.AffHide)
		// Remove the affect from the list too
		for _, aff := range ch.Affected.All() {
			if aff.Type == "hide" {
				ch.RemoveAffect(aff)
				break
			}
		}
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["hide"]
	}

	if skillLevel <= 0 {
		d.send(ch, "You don't know how to hide.\r\n")
		return
	}

	// Skill check
	if combat.NumberPercent() >= skillLevel {
		d.send(ch, "You attempt to hide but fail.\r\n")
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "hide", false, 3)
		}
		return
	}

	// Apply hide affect
	aff := &types.Affect{
		Type:      "hide",
		Level:     ch.Level,
		Duration:  ch.Level,
		Location:  types.ApplyNone,
		Modifier:  0,
		BitVector: types.AffHide,
	}
	ch.AddAffect(aff)

	d.send(ch, "You attempt to hide.\r\n")

	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "hide", true, 3)
	}
}

func (d *CommandDispatcher) cmdSteal(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Steal what from whom?\r\n")
		return
	}

	itemName := parts[0]
	victimName := parts[1]

	// Find the victim
	victim := FindCharInRoom(ch, victimName)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "That's pointless.\r\n")
		return
	}

	// Can't steal from players unless PKing is allowed
	if !victim.IsNPC() {
		canPK := false
		if d.Clans != nil {
			canPK = d.Clans.CanPK(ch, victim)
		}
		if !canPK {
			d.send(ch, "You can't steal from other players.\r\n")
			return
		}
	}

	// Check if in a safe room
	if ch.InRoom != nil && ch.InRoom.Flags.Has(types.RoomSafe) {
		d.send(ch, "This room is too safe for that kind of behavior.\r\n")
		return
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["steal"]
	}

	if skillLevel <= 0 {
		d.send(ch, "You don't know how to steal.\r\n")
		return
	}

	// Calculate chance of success
	// Base skill modified by victim's level and dexterity
	chance := skillLevel
	chance += (ch.GetStat(types.StatDex) - victim.GetStat(types.StatDex)) * 2
	chance += (ch.Level - victim.Level) * 3

	// Awake victims are harder to steal from
	if victim.Position >= types.PosStanding {
		chance -= 20
	}

	// Can't steal while victim is fighting
	if victim.InCombat() {
		d.send(ch, "They're moving around too much.\r\n")
		return
	}

	// Check for stealing gold
	if strings.ToLower(itemName) == "gold" || strings.ToLower(itemName) == "coins" {
		// Stealing gold
		if victim.Gold <= 0 {
			d.send(ch, "They don't have any gold.\r\n")
			return
		}

		if combat.NumberPercent() > chance {
			// Failed
			d.send(ch, "Oops!\r\n")
			d.send(victim, fmt.Sprintf("%s tried to steal from you!\r\n", ch.Name))

			// Victim attacks
			if victim.IsNPC() && !victim.InCombat() {
				d.startCombat(victim, ch)
			}

			if d.Skills != nil {
				d.Skills.CheckImprove(ch, "steal", false, 2)
			}
			return
		}

		// Success - steal 1/10 to 1/4 of their gold
		amount := combat.NumberRange(victim.Gold/10, victim.Gold/4)
		if amount <= 0 {
			amount = 1
		}

		ch.Gold += amount
		victim.Gold -= amount

		d.send(ch, fmt.Sprintf("You steal %d gold coins.\r\n", amount))

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "steal", true, 2)
		}
		return
	}

	// Stealing an item
	var obj *types.Object
	for _, item := range victim.Inventory {
		if strings.HasPrefix(strings.ToLower(item.Name), strings.ToLower(itemName)) {
			obj = item
			break
		}
	}

	if obj == nil {
		d.send(ch, "They don't have that item.\r\n")
		return
	}

	// Can't steal cursed items
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		d.send(ch, "You can't pry it away.\r\n")
		return
	}

	// Heavy items are harder to steal
	if obj.Weight > 50 {
		chance -= obj.Weight / 5
	}

	if combat.NumberPercent() > chance {
		// Failed
		d.send(ch, "Oops!\r\n")
		d.send(victim, fmt.Sprintf("%s tried to steal from you!\r\n", ch.Name))

		// Victim attacks
		if victim.IsNPC() && !victim.InCombat() {
			d.startCombat(victim, ch)
		}

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "steal", false, 2)
		}
		return
	}

	// Success - transfer the item
	victim.RemoveInventory(obj)
	ch.AddInventory(obj)

	d.send(ch, fmt.Sprintf("You steal %s.\r\n", obj.ShortDesc))

	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "steal", true, 2)
	}
}

func (d *CommandDispatcher) cmdPeek(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	if args == "" {
		d.send(ch, "Peek at whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.cmdInventory(ch, "")
		return
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["peek"]
	}

	if skillLevel <= 0 {
		d.send(ch, "You don't know how to peek.\r\n")
		return
	}

	// Skill check
	chance := skillLevel
	chance += (ch.GetStat(types.StatDex) - victim.GetStat(types.StatDex)) * 2

	if combat.NumberPercent() > chance {
		d.send(ch, "You fail to see their inventory.\r\n")
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "peek", false, 4)
		}
		return
	}

	// Show victim's inventory
	victimName := victim.Name
	if victim.IsNPC() && victim.ShortDesc != "" {
		victimName = victim.ShortDesc
	}

	d.send(ch, fmt.Sprintf("%s is carrying:\r\n", victimName))
	if len(victim.Inventory) == 0 {
		d.send(ch, "  Nothing.\r\n")
	} else {
		for _, obj := range victim.Inventory {
			d.send(ch, fmt.Sprintf("  %s\r\n", obj.ShortDesc))
		}
	}

	// Also show their gold if peeking at NPC
	if victim.IsNPC() && victim.Gold > 0 {
		d.send(ch, fmt.Sprintf("  %d gold coins\r\n", victim.Gold))
	}

	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "peek", true, 4)
	}
}

// Helper to start combat
func (d *CommandDispatcher) startCombat(attacker, victim *types.Character) {
	if attacker.Fighting == nil {
		attacker.Fighting = victim
		attacker.Position = types.PosFighting
	}
	if victim.Fighting == nil {
		victim.Fighting = attacker
		victim.Position = types.PosFighting
	}
}

func (d *CommandDispatcher) cmdTrack(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	// Get skill level
	skillLevel := 0
	if ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["track"]
	}

	if skillLevel <= 0 {
		d.send(ch, "You don't know how to track.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Track whom?\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You can't track here.\r\n")
		return
	}

	// Calculate movement cost based on terrain
	moveCost := ch.InRoom.Sector.MoveCost()
	if ch.Move < moveCost {
		d.send(ch, "You are too exhausted.\r\n")
		return
	}

	// Skill check - failure
	if combat.NumberPercent() > skillLevel {
		d.send(ch, fmt.Sprintf("You can find no recent tracks for %s.\r\n", args))
		ch.Move -= moveCost / 2
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "track", false, 1)
		}
		return
	}

	// Try to find a player first
	trackVnum := ch.InRoom.Vnum
	for _, player := range d.GameLoop.GetPlayers() {
		if player.InRoom == nil || !d.canSee(ch, player) {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(player.Name), strings.ToLower(args)) {
			continue
		}

		// Found the player - check if they're in the same room
		if player.InRoom.Vnum == trackVnum {
			d.send(ch, fmt.Sprintf("The tracks end right under %s's feet.\r\n", player.Name))
			return
		}

		// Search their track history for our current room
		for i := 0; i < types.MaxTrack; i++ {
			if player.TrackFrom[i] == trackVnum {
				// Found! Now find which exit leads to where they went
				for dir := types.Direction(0); dir < types.DirMax; dir++ {
					exit := ch.InRoom.GetExit(dir)
					if exit != nil && exit.ToRoom != nil && exit.ToRoom.Vnum == player.TrackTo[i] {
						d.send(ch, fmt.Sprintf("Some tracks lead off to the %s.\r\n", dir.String()))
						ch.Move -= moveCost
						if d.Skills != nil {
							d.Skills.CheckImprove(ch, "track", true, 1)
						}
						return
					}
				}
				// They were here but must have teleported or something
				d.send(ch, fmt.Sprintf("%s seems to have vanished here.\r\n", player.Name))
				ch.Move -= moveCost
				if d.Skills != nil {
					d.Skills.CheckImprove(ch, "track", true, 1)
				}
				return
			}
		}

		// Player exists but no tracks here
		d.send(ch, fmt.Sprintf("You can find no recent tracks for %s.\r\n", player.Name))
		ch.Move -= moveCost / 2
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "track", false, 1)
		}
		return
	}

	// Try to find an NPC in the same area
	for _, npc := range d.GameLoop.GetCharacters() {
		if !npc.IsNPC() || npc.InRoom == nil {
			continue
		}
		// Must be in same area
		if npc.InRoom.Area != ch.InRoom.Area {
			continue
		}
		if !d.canSee(ch, npc) {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(npc.Name), strings.ToLower(args)) {
			continue
		}

		// Found NPC - check if in same room
		if npc.InRoom.Vnum == trackVnum {
			d.send(ch, fmt.Sprintf("The tracks end right under %s.\r\n", npc.ShortDesc))
			return
		}

		// Search their track history
		for i := 0; i < types.MaxTrack; i++ {
			if npc.TrackFrom[i] == trackVnum {
				// Found! Now find which exit leads to where they went
				for dir := types.Direction(0); dir < types.DirMax; dir++ {
					exit := ch.InRoom.GetExit(dir)
					if exit != nil && exit.ToRoom != nil && exit.ToRoom.Vnum == npc.TrackTo[i] {
						d.send(ch, fmt.Sprintf("Some tracks lead off to the %s.\r\n", dir.String()))
						ch.Move -= moveCost
						if d.Skills != nil {
							d.Skills.CheckImprove(ch, "track", true, 1)
						}
						return
					}
				}
				// They were here but must have teleported
				npcName := npc.Name
				if npc.ShortDesc != "" {
					npcName = npc.ShortDesc
				}
				d.send(ch, fmt.Sprintf("%s seems to have vanished here.\r\n", npcName))
				ch.Move -= moveCost
				if d.Skills != nil {
					d.Skills.CheckImprove(ch, "track", true, 1)
				}
				return
			}
		}
	}

	// No target found
	d.send(ch, fmt.Sprintf("You can find no recent tracks for %s.\r\n", args))
	ch.Move -= moveCost / 2
	if d.Skills != nil {
		d.Skills.CheckImprove(ch, "track", false, 1)
	}
}

// canSee checks if ch can see victim (basic implementation)
func (d *CommandDispatcher) canSee(ch, victim *types.Character) bool {
	// Can always see yourself
	if ch == victim {
		return true
	}

	// Blind characters can't see
	if ch.IsAffected(types.AffBlind) {
		return false
	}

	// Hidden/invisible checks
	if victim.IsAffected(types.AffHide) || victim.IsAffected(types.AffInvisible) {
		if !ch.IsAffected(types.AffDetectHidden) && !ch.IsAffected(types.AffDetectInvis) {
			// Immortals can always see
			if ch.Level < 100 {
				return false
			}
		}
	}

	return true
}
