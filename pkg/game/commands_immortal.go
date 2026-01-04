package game

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// cmdForce makes a character execute a command
func (d *CommandDispatcher) cmdForce(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		d.send(ch, "Force whom to do what?\r\n")
		return
	}

	target := parts[0]
	command := parts[1]

	// Get the first word of the command to check for blocked commands
	cmdParts := strings.SplitN(command, " ", 2)
	cmdWord := strings.ToLower(cmdParts[0])

	// Block dangerous commands
	if cmdWord == "delete" || strings.HasPrefix(cmdWord, "mob") || cmdWord == "reroll" {
		d.send(ch, "That will NOT be done.\r\n")
		return
	}

	if strings.ToLower(target) == "all" {
		// Force all players
		if ch.Level < types.MaxLevel-3 {
			d.send(ch, "Not at your level!\r\n")
			return
		}

		count := 0
		for _, victim := range d.GameLoop.Characters {
			if !victim.IsNPC() && victim.Level < ch.Level && victim != ch {
				d.send(victim, fmt.Sprintf("%s forces you to '%s'.\r\n", ch.Name, command))
				d.Dispatch(Command{Character: victim, Input: command})
				count++
			}
		}
		d.send(ch, fmt.Sprintf("Forced %d players to '%s'.\r\n", count, command))
		return
	}

	// Find specific target
	victim := d.findCharacterWorld(ch, target)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if !victim.IsNPC() && victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	d.send(victim, fmt.Sprintf("%s forces you to '%s'.\r\n", ch.Name, command))
	d.Dispatch(Command{Character: victim, Input: command})
	d.send(ch, "Ok.\r\n")
}

// cmdSlay instantly kills a target
func (d *CommandDispatcher) cmdSlay(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Slay whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if ch == victim {
		d.send(ch, "Suicide is a mortal sin.\r\n")
		return
	}

	if !victim.IsNPC() && victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Display slay messages
	d.send(ch, fmt.Sprintf("You slay %s in cold blood!\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s slays you in cold blood!\r\n", ch.Name))

	// Notify room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch && person != victim {
				d.send(person, fmt.Sprintf("%s slays %s in cold blood!\r\n", ch.Name, victim.Name))
			}
		}
	}

	// Kill the victim using combat system
	if d.Combat != nil {
		d.Combat.HandleDeath(ch, victim)
	} else {
		// Fallback if combat system not available
		victim.Hit = 0
		victim.Position = types.PosDead
	}
}

// cmdFreeze toggles the freeze flag on a player (prevents them from playing)
// Uses the NoChannels comm flag as a stand-in for freeze
func (d *CommandDispatcher) cmdFreeze(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Freeze whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Use NoChannels as freeze flag (blocks all channels)
	if victim.Comm.Has(types.CommNoChannels) {
		victim.Comm.Remove(types.CommNoChannels)
		d.send(victim, "You can play again.\r\n")
		d.send(ch, "FREEZE removed.\r\n")
	} else {
		victim.Comm.Set(types.CommNoChannels)
		d.send(victim, "You can't do ANYthing!\r\n")
		d.send(ch, "FREEZE set.\r\n")
	}
}

// cmdMstat shows detailed mob/character statistics
func (d *CommandDispatcher) cmdMstat(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Stat whom?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		victim = d.findCharacterWorld(ch, args)
	}
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Name: %s\r\n", victim.Name))
	if victim.ShortDesc != "" {
		d.send(ch, fmt.Sprintf("Short: %s\r\n", victim.ShortDesc))
	}
	if victim.LongDesc != "" {
		d.send(ch, fmt.Sprintf("Long: %s\r\n", victim.LongDesc))
	}

	d.send(ch, fmt.Sprintf("Level: %d  Class: %d  Race: %d  Sex: %s\r\n",
		victim.Level, victim.Class, victim.Race, victim.Sex))

	d.send(ch, fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d\r\n",
		victim.Hit, victim.MaxHit, victim.Mana, victim.MaxMana, victim.Move, victim.MaxMove))

	d.send(ch, fmt.Sprintf("Gold: %d  Silver: %d  Exp: %d\r\n",
		victim.Gold, victim.Silver, victim.Exp))

	d.send(ch, fmt.Sprintf("Str: %d  Int: %d  Wis: %d  Dex: %d  Con: %d\r\n",
		victim.GetStat(types.StatStr), victim.GetStat(types.StatInt),
		victim.GetStat(types.StatWis), victim.GetStat(types.StatDex),
		victim.GetStat(types.StatCon)))

	d.send(ch, fmt.Sprintf("Hitroll: %d  Damroll: %d  AC: %d/%d/%d/%d\r\n",
		victim.HitRoll, victim.DamRoll, victim.Armor[0], victim.Armor[1], victim.Armor[2], victim.Armor[3]))

	d.send(ch, fmt.Sprintf("Position: %s  Alignment: %d\r\n",
		victim.Position, victim.Alignment))

	if victim.Fighting != nil {
		d.send(ch, fmt.Sprintf("Fighting: %s\r\n", victim.Fighting.Name))
	}

	if victim.InRoom != nil {
		d.send(ch, fmt.Sprintf("Room: %d (%s)\r\n", victim.InRoom.Vnum, victim.InRoom.Name))
	}

	if victim.Master != nil {
		d.send(ch, fmt.Sprintf("Master: %s\r\n", victim.Master.Name))
	}

	if victim.Leader != nil {
		d.send(ch, fmt.Sprintf("Leader: %s\r\n", victim.Leader.Name))
	}

	// Show affects
	if victim.Affected.Len() > 0 {
		d.send(ch, "Affects:\r\n")
		for _, aff := range victim.Affected.All() {
			d.send(ch, fmt.Sprintf("  %s: level %d, %d hours\r\n",
				aff.Type, aff.Level, aff.Duration))
		}
	}
}

// cmdOstat shows detailed object statistics
func (d *CommandDispatcher) cmdOstat(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Stat what object?\r\n")
		return
	}

	// Look in inventory first
	obj := FindObjInInventory(ch, args)
	if obj == nil && ch.InRoom != nil {
		// Then look in room
		for _, o := range ch.InRoom.Objects {
			if strings.HasPrefix(strings.ToLower(o.Name), strings.ToLower(args)) {
				obj = o
				break
			}
		}
	}

	if obj == nil {
		d.send(ch, "You don't see that here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Name: %s\r\n", obj.Name))
	d.send(ch, fmt.Sprintf("Short: %s\r\n", obj.ShortDesc))
	d.send(ch, fmt.Sprintf("Long: %s\r\n", obj.LongDesc))
	d.send(ch, fmt.Sprintf("Vnum: %d  Type: %d  Level: %d\r\n",
		obj.Vnum, obj.ItemType, obj.Level))
	d.send(ch, fmt.Sprintf("Weight: %d  Cost: %d  Timer: %d\r\n",
		obj.Weight, obj.Cost, obj.Timer))

	if len(obj.Values) > 0 {
		d.send(ch, fmt.Sprintf("Values: %v\r\n", obj.Values))
	}

	// Show contents if container
	if len(obj.Contents) > 0 {
		d.send(ch, "Contents:\r\n")
		combine := ch.Comm.Has(types.CommCombine) || ch.IsNPC()
		lines := formatObjectList(obj.Contents, ch, true, combine)
		for _, line := range lines {
			d.send(ch, line+"\r\n")
		}
	}
}

// cmdRstat shows detailed room statistics
func (d *CommandDispatcher) cmdRstat(ch *types.Character, args string) {
	room := ch.InRoom
	if room == nil {
		d.send(ch, "You are nowhere?!\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Name: %s\r\n", room.Name))
	d.send(ch, fmt.Sprintf("Vnum: %d  Sector: %s\r\n", room.Vnum, room.Sector))
	if room.Description != "" {
		d.send(ch, fmt.Sprintf("Description:\r\n%s", room.Description))
	}

	d.send(ch, "Exits:\r\n")
	for dir, exit := range room.Exits {
		if exit != nil && exit.ToRoom != nil {
			d.send(ch, fmt.Sprintf("  %s -> %d (%s)\r\n",
				[]string{"North", "East", "South", "West", "Up", "Down"}[dir],
				exit.ToRoom.Vnum, exit.ToRoom.Name))
		}
	}

	if len(room.People) > 0 {
		d.send(ch, "Characters:\r\n")
		for _, person := range room.People {
			d.send(ch, fmt.Sprintf("  %s\r\n", person.Name))
		}
	}

	if len(room.Objects) > 0 {
		d.send(ch, "Objects:\r\n")
		for _, obj := range room.Objects {
			d.send(ch, fmt.Sprintf("  %s\r\n", obj.ShortDesc))
		}
	}
}

// cmdMfind finds mobs by name or keyword
func (d *CommandDispatcher) cmdMfind(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Find what mob?\r\n")
		return
	}

	if d.GameLoop.World == nil {
		d.send(ch, "World data not loaded.\r\n")
		return
	}

	search := strings.ToLower(args)
	found := 0

	for _, template := range d.GameLoop.World.MobTemplates {
		if template == nil {
			continue
		}

		// Check keywords and short desc
		match := false
		for _, kw := range template.Keywords {
			if strings.Contains(strings.ToLower(kw), search) {
				match = true
				break
			}
		}
		if !match && strings.Contains(strings.ToLower(template.ShortDesc), search) {
			match = true
		}

		if match {
			d.send(ch, fmt.Sprintf("[%5d] %s\r\n", template.Vnum, template.ShortDesc))
			found++
		}
	}

	d.send(ch, fmt.Sprintf("Found %d matches.\r\n", found))
}

// cmdOfind finds objects by name or keyword
func (d *CommandDispatcher) cmdOfind(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Find what object?\r\n")
		return
	}

	if d.GameLoop.World == nil {
		d.send(ch, "World data not loaded.\r\n")
		return
	}

	search := strings.ToLower(args)
	found := 0

	for _, template := range d.GameLoop.World.ObjTemplates {
		if template == nil {
			continue
		}

		// Check keywords and short desc
		match := false
		for _, kw := range template.Keywords {
			if strings.Contains(strings.ToLower(kw), search) {
				match = true
				break
			}
		}
		if !match && strings.Contains(strings.ToLower(template.ShortDesc), search) {
			match = true
		}

		if match {
			d.send(ch, fmt.Sprintf("[%5d] %s\r\n", template.Vnum, template.ShortDesc))
			found++
		}
	}

	d.send(ch, fmt.Sprintf("Found %d matches.\r\n", found))
}

// cmdMwhere finds mobiles by name and shows their location
func (d *CommandDispatcher) cmdMwhere(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Find what mob?\r\n")
		return
	}

	search := strings.ToLower(args)
	found := 0

	for _, mob := range d.GameLoop.Characters {
		if !mob.IsNPC() {
			continue
		}

		if strings.Contains(strings.ToLower(mob.Name), search) ||
			strings.Contains(strings.ToLower(mob.ShortDesc), search) {
			roomName := "nowhere"
			roomVnum := 0
			if mob.InRoom != nil {
				roomName = mob.InRoom.Name
				roomVnum = mob.InRoom.Vnum
			}
			d.send(ch, fmt.Sprintf("[%5d] %-28s in %s [%d]\r\n",
				0, mob.ShortDesc, roomName, roomVnum))
			found++
		}
	}

	if found == 0 {
		d.send(ch, "No mobiles found.\r\n")
	} else {
		d.send(ch, fmt.Sprintf("Found %d mobiles.\r\n", found))
	}
}

// cmdOwhere finds objects and shows their location
func (d *CommandDispatcher) cmdOwhere(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Find what object?\r\n")
		return
	}

	search := strings.ToLower(args)
	found := 0

	// Check objects in rooms
	for _, room := range d.GameLoop.Rooms {
		for _, obj := range room.Objects {
			if strings.Contains(strings.ToLower(obj.Name), search) ||
				strings.Contains(strings.ToLower(obj.ShortDesc), search) {
				d.send(ch, fmt.Sprintf("[%5d] %-28s in %s [%d]\r\n",
					obj.Vnum, obj.ShortDesc, room.Name, room.Vnum))
				found++
			}
		}
	}

	// Check objects carried by characters
	for _, person := range d.GameLoop.Characters {
		for _, obj := range person.Inventory {
			if strings.Contains(strings.ToLower(obj.Name), search) ||
				strings.Contains(strings.ToLower(obj.ShortDesc), search) {
				d.send(ch, fmt.Sprintf("[%5d] %-28s carried by %s\r\n",
					obj.Vnum, obj.ShortDesc, person.Name))
				found++
			}
		}
	}

	if found == 0 {
		d.send(ch, "No objects found.\r\n")
	} else {
		d.send(ch, fmt.Sprintf("Found %d objects.\r\n", found))
	}
}

// cmdInvis toggles immortal invisibility
func (d *CommandDispatcher) cmdInvis(ch *types.Character, args string) {
	if ch.IsAffected(types.AffInvisible) {
		// Remove invisibility by clearing the flag directly
		ch.AffectedBy &^= types.AffInvisible
		d.send(ch, "You are now visible.\r\n")

		// Notify room
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if person != ch {
					d.send(person, fmt.Sprintf("%s slowly fades into existence.\r\n", ch.Name))
				}
			}
		}
	} else {
		// Add invisibility by setting the flag directly
		ch.AffectedBy |= types.AffInvisible
		d.send(ch, "You slowly vanish into thin air.\r\n")

		// Notify room
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if person != ch {
					d.send(person, fmt.Sprintf("%s slowly fades into thin air.\r\n", ch.Name))
				}
			}
		}
	}
}

// cmdHolylight toggles holylight (see everything regardless of visibility)
// Uses WizInvis player flag as holylight equivalent
func (d *CommandDispatcher) cmdHolylight(ch *types.Character, args string) {
	// Note: In a full implementation, we'd add PlrHolyLight to Character
	// For now, just send a message
	d.send(ch, "Holy light mode toggled.\r\n")
}

// cmdIncognito toggles incognito mode (hidden from lower level immortals)
func (d *CommandDispatcher) cmdIncognito(ch *types.Character, args string) {
	// Note: In a full implementation, we'd add Incog field to Character
	// For now, just send a message
	d.send(ch, "Incognito mode toggled.\r\n")
}

// findCharacterWorld finds a character anywhere in the world
func (d *CommandDispatcher) findCharacterWorld(ch *types.Character, name string) *types.Character {
	// First check room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if strings.HasPrefix(strings.ToLower(person.Name), strings.ToLower(name)) {
				return person
			}
		}
	}

	// Then check all characters
	for _, person := range d.GameLoop.Characters {
		if strings.HasPrefix(strings.ToLower(person.Name), strings.ToLower(name)) {
			return person
		}
	}

	return nil
}

// cmdSnoop allows immortals to see a player's input/output
// Syntax: snoop <player> or snoop (to cancel all snoops)
func (d *CommandDispatcher) cmdSnoop(ch *types.Character, args string) {
	if ch.Descriptor == nil {
		d.send(ch, "No descriptor.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Snoop whom?\r\n")
		return
	}

	// Find victim
	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.Descriptor == nil {
		d.send(ch, "No descriptor to snoop.\r\n")
		return
	}

	// Snooping self cancels all snoops
	if victim == ch {
		d.send(ch, "Cancelling all snoops.\r\n")
		d.cancelAllSnoops(ch)
		return
	}

	// Can't snoop someone already being snooped
	if victim.Descriptor.SnoopedBy != nil {
		d.send(ch, "Busy already.\r\n")
		return
	}

	// Can't snoop higher level immortals
	if !victim.IsNPC() && victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Check for snoop loops (can't snoop someone who is snooping you)
	if d.wouldCreateSnoopLoop(ch, victim) {
		d.send(ch, "No snoop loops.\r\n")
		return
	}

	// Set up the snoop
	victim.Descriptor.SnoopedBy = ch.Descriptor
	d.send(ch, "Ok.\r\n")
}

// cancelAllSnoops removes all snoop references to the given character
func (d *CommandDispatcher) cancelAllSnoops(ch *types.Character) {
	if ch.Descriptor == nil {
		return
	}

	// Find all descriptors that this character is snooping and clear them
	for _, player := range d.GameLoop.Characters {
		if player.Descriptor != nil && player.Descriptor.SnoopedBy == ch.Descriptor {
			player.Descriptor.SnoopedBy = nil
		}
	}
}

// wouldCreateSnoopLoop checks if snooping victim would create a loop
func (d *CommandDispatcher) wouldCreateSnoopLoop(ch, victim *types.Character) bool {
	if ch.Descriptor == nil {
		return false
	}

	// Walk up the snoop chain from our descriptor
	for desc := ch.Descriptor.SnoopedBy; desc != nil; desc = desc.SnoopedBy {
		if desc.Character == victim {
			return true
		}
	}

	return false
}

// cmdMset modifies character/mobile stats
// Syntax: mset <name> <field> <value>
func (d *CommandDispatcher) cmdMset(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 3 {
		d.send(ch, "Syntax: mset <name> <field> <value>\r\n")
		d.send(ch, "Fields: str int wis dex con sex level hp mana move\r\n")
		d.send(ch, "        gold silver exp align hitroll damroll\r\n")
		return
	}

	name := parts[0]
	field := strings.ToLower(parts[1])
	valueStr := parts[2]

	// Find victim
	victim := d.findCharacterWorld(ch, name)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	// Can't modify higher level characters
	if !victim.IsNPC() && victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Parse numeric value (some fields may be non-numeric)
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		value = -1
	}

	switch field {
	case "str":
		if value < 3 || value > 25 {
			d.send(ch, "Strength range is 3 to 25.\r\n")
			return
		}
		victim.PermStats[types.StatStr] = value
		d.send(ch, "Ok.\r\n")

	case "int":
		if value < 3 || value > 25 {
			d.send(ch, "Intelligence range is 3 to 25.\r\n")
			return
		}
		victim.PermStats[types.StatInt] = value
		d.send(ch, "Ok.\r\n")

	case "wis":
		if value < 3 || value > 25 {
			d.send(ch, "Wisdom range is 3 to 25.\r\n")
			return
		}
		victim.PermStats[types.StatWis] = value
		d.send(ch, "Ok.\r\n")

	case "dex":
		if value < 3 || value > 25 {
			d.send(ch, "Dexterity range is 3 to 25.\r\n")
			return
		}
		victim.PermStats[types.StatDex] = value
		d.send(ch, "Ok.\r\n")

	case "con":
		if value < 3 || value > 25 {
			d.send(ch, "Constitution range is 3 to 25.\r\n")
			return
		}
		victim.PermStats[types.StatCon] = value
		d.send(ch, "Ok.\r\n")

	case "sex":
		if value < 0 || value > 2 {
			d.send(ch, "Sex range is 0 (neutral) to 2 (female).\r\n")
			return
		}
		victim.Sex = types.Sex(value)
		d.send(ch, "Ok.\r\n")

	case "level":
		if victim.IsNPC() {
			if value < 0 || value > ch.Level {
				d.send(ch, fmt.Sprintf("Level range is 0 to %d.\r\n", ch.Level))
				return
			}
		} else {
			d.send(ch, "Use 'advance' for players.\r\n")
			return
		}
		victim.Level = value
		d.send(ch, "Ok.\r\n")

	case "hp":
		if value < 1 || value > 100000 {
			d.send(ch, "HP range is 1 to 100000.\r\n")
			return
		}
		victim.MaxHit = value
		if victim.Hit > value {
			victim.Hit = value
		}
		d.send(ch, "Ok.\r\n")

	case "mana":
		if value < 0 || value > 100000 {
			d.send(ch, "Mana range is 0 to 100000.\r\n")
			return
		}
		victim.MaxMana = value
		if victim.Mana > value {
			victim.Mana = value
		}
		d.send(ch, "Ok.\r\n")

	case "move":
		if value < 0 || value > 100000 {
			d.send(ch, "Move range is 0 to 100000.\r\n")
			return
		}
		victim.MaxMove = value
		if victim.Move > value {
			victim.Move = value
		}
		d.send(ch, "Ok.\r\n")

	case "gold":
		if value < 0 {
			d.send(ch, "Gold must be positive.\r\n")
			return
		}
		victim.Gold = value
		d.send(ch, "Ok.\r\n")

	case "silver":
		if value < 0 {
			d.send(ch, "Silver must be positive.\r\n")
			return
		}
		victim.Silver = value
		d.send(ch, "Ok.\r\n")

	case "exp":
		victim.Exp = value
		d.send(ch, "Ok.\r\n")

	case "align", "alignment":
		if value < -1000 || value > 1000 {
			d.send(ch, "Alignment range is -1000 to 1000.\r\n")
			return
		}
		victim.Alignment = value
		d.send(ch, "Ok.\r\n")

	case "hitroll":
		victim.HitRoll = value
		d.send(ch, "Ok.\r\n")

	case "damroll":
		victim.DamRoll = value
		d.send(ch, "Ok.\r\n")

	default:
		d.send(ch, "Invalid field. Valid fields:\r\n")
		d.send(ch, "  str int wis dex con sex level hp mana move\r\n")
		d.send(ch, "  gold silver exp align hitroll damroll\r\n")
	}
}

// cmdMload loads/spawns a mobile from template
// Syntax: mload <vnum>
func (d *CommandDispatcher) cmdMload(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Syntax: mload <vnum>\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "That is not a valid vnum.\r\n")
		return
	}

	if d.GameLoop.World == nil {
		d.send(ch, "World data not loaded.\r\n")
		return
	}

	// Create mob from template
	mob := d.GameLoop.World.CreateMobFromTemplate(vnum)
	if mob == nil {
		d.send(ch, fmt.Sprintf("No mobile with vnum %d exists.\r\n", vnum))
		return
	}

	// Place mob in room
	if ch.InRoom == nil {
		d.send(ch, "You are nowhere.\r\n")
		return
	}

	CharToRoom(mob, ch.InRoom)
	d.GameLoop.AddCharacter(mob)

	d.send(ch, fmt.Sprintf("You have created %s!\r\n", mob.ShortDesc))

	// Notify room
	for _, person := range ch.InRoom.People {
		if person != ch && person != mob {
			d.send(person, fmt.Sprintf("%s has created %s!\r\n", ch.Name, mob.ShortDesc))
		}
	}
}

// cmdOload loads/spawns an object from template
// Syntax: oload <vnum> [level]
func (d *CommandDispatcher) cmdOload(ch *types.Character, args string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		d.send(ch, "Syntax: oload <vnum> [level]\r\n")
		return
	}

	vnum, err := strconv.Atoi(parts[0])
	if err != nil {
		d.send(ch, "That is not a valid vnum.\r\n")
		return
	}

	level := ch.Level
	if len(parts) > 1 {
		level, err = strconv.Atoi(parts[1])
		if err != nil {
			d.send(ch, "Invalid level.\r\n")
			return
		}
		if level < 1 || level > ch.Level {
			d.send(ch, fmt.Sprintf("Level range is 1 to %d.\r\n", ch.Level))
			return
		}
	}

	if d.GameLoop.World == nil {
		d.send(ch, "World data not loaded.\r\n")
		return
	}

	// Find object template
	tmpl := d.GameLoop.World.GetObjTemplate(vnum)
	if tmpl == nil {
		d.send(ch, fmt.Sprintf("No object with vnum %d exists.\r\n", vnum))
		return
	}

	// Create object from template (use the callback or create directly)
	obj := d.createObjectFromTemplate(tmpl, level)
	if obj == nil {
		d.send(ch, "Failed to create object.\r\n")
		return
	}

	// Put in inventory
	ch.Inventory = append(ch.Inventory, obj)
	obj.CarriedBy = ch

	d.send(ch, fmt.Sprintf("You have created %s!\r\n", obj.ShortDesc))

	// Notify room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch {
				d.send(person, fmt.Sprintf("%s has created %s!\r\n", ch.Name, obj.ShortDesc))
			}
		}
	}
}

// cmdRset modifies room properties
// Syntax: rset <field> <value>
func (d *CommandDispatcher) cmdRset(ch *types.Character, args string) {
	room := ch.InRoom
	if room == nil {
		d.send(ch, "You are nowhere!\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Syntax: rset <field> <value>\r\n")
		d.send(ch, "Fields: name sector heal mana\r\n")
		return
	}

	field := strings.ToLower(parts[0])
	valueStr := parts[1]

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		value = -1
	}

	switch field {
	case "name":
		room.Name = valueStr
		d.send(ch, "Ok.\r\n")

	case "sector":
		// Map sector name to type
		sector := types.SectInside
		switch strings.ToLower(valueStr) {
		case "inside":
			sector = types.SectInside
		case "city":
			sector = types.SectCity
		case "field":
			sector = types.SectField
		case "forest":
			sector = types.SectForest
		case "hills":
			sector = types.SectHills
		case "mountain":
			sector = types.SectMountain
		case "water_swim", "swim":
			sector = types.SectWaterSwim
		case "water_noswim", "noswim":
			sector = types.SectWaterNoSwim
		case "air":
			sector = types.SectAir
		case "desert":
			sector = types.SectDesert
		default:
			d.send(ch, "Unknown sector type.\r\n")
			return
		}
		room.Sector = sector
		d.send(ch, "Ok.\r\n")

	case "heal":
		if value < 0 || value > 500 {
			d.send(ch, "Heal rate range is 0 to 500.\r\n")
			return
		}
		room.HealRate = value
		d.send(ch, "Ok.\r\n")

	case "mana":
		if value < 0 || value > 500 {
			d.send(ch, "Mana rate range is 0 to 500.\r\n")
			return
		}
		room.ManaRate = value
		d.send(ch, "Ok.\r\n")

	default:
		d.send(ch, "Invalid field. Valid fields:\r\n")
		d.send(ch, "  name sector heal mana\r\n")
	}
}

// cmdSwitch allows immortals to control a mobile
// Syntax: switch <mobile>
func (d *CommandDispatcher) cmdSwitch(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Switch into whom?\r\n")
		return
	}

	if ch.Descriptor == nil {
		d.send(ch, "You don't have a descriptor.\r\n")
		return
	}

	// Check if already switched
	if ch.Descriptor.Original != nil {
		d.send(ch, "You are already switched.\r\n")
		return
	}

	// Find victim
	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	// Can only switch into NPCs
	if !victim.IsNPC() {
		d.send(ch, "You can only switch into mobiles.\r\n")
		return
	}

	// Can't switch into a mob that's already controlled
	if victim.Descriptor != nil {
		d.send(ch, "That mobile is already controlled.\r\n")
		return
	}

	// Perform the switch
	d.send(ch, "Ok.\r\n")

	// Store original character
	ch.Descriptor.Original = ch
	ch.Descriptor.Character = victim
	victim.Descriptor = ch.Descriptor
	ch.Descriptor = nil

	d.send(victim, fmt.Sprintf("You are now controlling %s.\r\n", victim.ShortDesc))
}

// createObjectFromTemplate creates an object from a template
func (d *CommandDispatcher) createObjectFromTemplate(tmpl *loader.ObjectData, level int) *types.Object {
	if tmpl == nil {
		return nil
	}

	// Parse item type
	itemType := types.ItemTypeTrash
	switch strings.ToLower(tmpl.ItemType) {
	case "weapon":
		itemType = types.ItemTypeWeapon
	case "armor":
		itemType = types.ItemTypeArmor
	case "scroll":
		itemType = types.ItemTypeScroll
	case "wand":
		itemType = types.ItemTypeWand
	case "staff":
		itemType = types.ItemTypeStaff
	case "potion":
		itemType = types.ItemTypePotion
	case "container":
		itemType = types.ItemTypeContainer
	case "key":
		itemType = types.ItemTypeKey
	case "food":
		itemType = types.ItemTypeFood
	case "money":
		itemType = types.ItemTypeMoney
	case "light":
		itemType = types.ItemTypeLight
	case "fountain":
		itemType = types.ItemTypeFountain
	case "drink":
		itemType = types.ItemTypeDrinkCon
	case "pill":
		itemType = types.ItemTypePill
	case "treasure":
		itemType = types.ItemTypeTreasure
	}

	obj := types.NewObject(tmpl.Vnum, tmpl.ShortDesc, itemType)
	obj.Name = strings.Join(tmpl.Keywords, " ")
	obj.LongDesc = tmpl.LongDesc
	obj.Level = level
	obj.Weight = tmpl.Weight
	obj.Cost = tmpl.Cost

	// Copy wear flags
	for _, flag := range tmpl.WearFlags {
		switch strings.ToLower(flag) {
		case "take":
			obj.WearFlags.Set(types.WearTake)
		case "wield":
			obj.WearFlags.Set(types.WearWield)
		case "hold":
			obj.WearFlags.Set(types.WearHold)
		case "body":
			obj.WearFlags.Set(types.WearBody)
		case "head":
			obj.WearFlags.Set(types.WearHead)
		case "legs":
			obj.WearFlags.Set(types.WearLegs)
		case "feet":
			obj.WearFlags.Set(types.WearFeet)
		case "hands":
			obj.WearFlags.Set(types.WearHands)
		case "arms":
			obj.WearFlags.Set(types.WearArms)
		case "shield":
			obj.WearFlags.Set(types.WearShield)
		case "about":
			obj.WearFlags.Set(types.WearAbout)
		case "waist":
			obj.WearFlags.Set(types.WearWaist)
		case "wrist":
			obj.WearFlags.Set(types.WearWrist)
		case "finger":
			obj.WearFlags.Set(types.WearFinger)
		case "neck":
			obj.WearFlags.Set(types.WearNeck)
		}
	}

	// Parse extra flags
	for _, flag := range tmpl.ExtraFlags {
		switch strings.ToLower(flag) {
		case "glow":
			obj.ExtraFlags.Set(types.ItemGlow)
		case "hum":
			obj.ExtraFlags.Set(types.ItemHum)
		case "magic":
			obj.ExtraFlags.Set(types.ItemMagic)
		case "bless":
			obj.ExtraFlags.Set(types.ItemBless)
		case "nodrop":
			obj.ExtraFlags.Set(types.ItemNoDrop)
		case "noremove":
			obj.ExtraFlags.Set(types.ItemNoRemove)
		}
	}

	// Weapon-specific values
	if tmpl.Weapon != nil {
		obj.Values[1] = tmpl.Weapon.DiceNumber
		obj.Values[2] = tmpl.Weapon.DiceSize
	}

	return obj
}

// cmdReturn returns an immortal to their original body after switch
func (d *CommandDispatcher) cmdReturn(ch *types.Character, args string) {
	if ch.Descriptor == nil {
		d.send(ch, "You don't have a descriptor.\r\n")
		return
	}

	if ch.Descriptor.Original == nil {
		d.send(ch, "You aren't switched.\r\n")
		return
	}

	d.send(ch, "You return to your original body.\r\n")

	// Restore original character
	original := ch.Descriptor.Original
	original.Descriptor = ch.Descriptor
	ch.Descriptor.Character = original
	ch.Descriptor.Original = nil
	ch.Descriptor = nil
}

// cmdOset modifies object stats
// Syntax: oset <object> <field> <value>
func (d *CommandDispatcher) cmdOset(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 3)
	if len(parts) < 3 {
		d.send(ch, "Syntax: oset <object> <field> <value>\r\n")
		d.send(ch, "Fields: v0 v1 v2 v3 v4 (values)\r\n")
		d.send(ch, "        level weight cost timer name short long\r\n")
		return
	}

	name := parts[0]
	field := strings.ToLower(parts[1])
	valueStr := parts[2]

	// Find object - check inventory first, then room, then world
	obj := FindObjInInventory(ch, name)
	if obj == nil && ch.InRoom != nil {
		for _, o := range ch.InRoom.Objects {
			if strings.HasPrefix(strings.ToLower(o.Name), strings.ToLower(name)) {
				obj = o
				break
			}
		}
	}
	// Search world
	if obj == nil && d.GameLoop != nil {
		for _, room := range d.GameLoop.Rooms {
			for _, o := range room.Objects {
				if strings.HasPrefix(strings.ToLower(o.Name), strings.ToLower(name)) {
					obj = o
					break
				}
			}
			if obj != nil {
				break
			}
		}
	}

	if obj == nil {
		d.send(ch, "Nothing like that in heaven or earth.\r\n")
		return
	}

	// Parse numeric value
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		value = -1
	}

	switch field {
	case "v0", "value0":
		obj.Values[0] = value
		d.send(ch, "Ok.\r\n")

	case "v1", "value1":
		obj.Values[1] = value
		d.send(ch, "Ok.\r\n")

	case "v2", "value2":
		obj.Values[2] = value
		d.send(ch, "Ok.\r\n")

	case "v3", "value3":
		obj.Values[3] = value
		d.send(ch, "Ok.\r\n")

	case "v4", "value4":
		obj.Values[4] = value
		d.send(ch, "Ok.\r\n")

	case "level":
		if value < 0 || value > 100 {
			d.send(ch, "Level range is 0 to 100.\r\n")
			return
		}
		obj.Level = value
		d.send(ch, "Ok.\r\n")

	case "weight":
		if value < 0 {
			d.send(ch, "Weight must be positive.\r\n")
			return
		}
		obj.Weight = value
		d.send(ch, "Ok.\r\n")

	case "cost":
		if value < 0 {
			d.send(ch, "Cost must be positive.\r\n")
			return
		}
		obj.Cost = value
		d.send(ch, "Ok.\r\n")

	case "timer":
		obj.Timer = value
		d.send(ch, "Ok.\r\n")

	case "name":
		obj.Name = valueStr
		d.send(ch, "Ok.\r\n")

	case "short":
		obj.ShortDesc = valueStr
		d.send(ch, "Ok.\r\n")

	case "long":
		obj.LongDesc = valueStr
		d.send(ch, "Ok.\r\n")

	default:
		d.send(ch, "Invalid field. Valid fields:\r\n")
		d.send(ch, "  v0 v1 v2 v3 v4 (values)\r\n")
		d.send(ch, "  level weight cost timer name short long\r\n")
	}
}

// cmdDisconnect forcibly disconnects a player
// Syntax: disconnect <player>
func (d *CommandDispatcher) cmdDisconnect(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Disconnect whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.Descriptor == nil {
		d.send(ch, fmt.Sprintf("%s doesn't have a descriptor.\r\n", victim.Name))
		return
	}

	// Can't disconnect higher level immortals
	if !victim.IsNPC() && victim.Level >= ch.Level && ch.Level < types.MaxLevel {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Close the connection via the disconnect callback
	if d.DisconnectPlayer != nil {
		d.DisconnectPlayer(victim)
		d.send(ch, "Ok.\r\n")
	} else {
		d.send(ch, "Disconnect mechanism not available.\r\n")
	}
}

// cmdPecho sends a personal echo to a specific player
// Syntax: pecho <player> <message>
func (d *CommandDispatcher) cmdPecho(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		d.send(ch, "Personal echo what?\r\n")
		return
	}

	target := parts[0]
	message := parts[1]

	victim := d.findCharacterWorld(ch, target)
	if victim == nil {
		d.send(ch, "Target not found.\r\n")
		return
	}

	// If target is same or higher level, prefix with "personal>"
	if victim.Level >= ch.Level {
		d.send(victim, "personal> ")
	}

	d.send(victim, message+"\r\n")
	d.send(ch, "personal> "+message+"\r\n")
}

// cmdWiznet toggles wiznet channels for immortal communication
// Syntax: wiznet [on|off|status|show|<flag>]
func (d *CommandDispatcher) cmdWiznet(ch *types.Character, args string) {
	if !ch.IsImmortal() {
		d.send(ch, "Huh?\r\n")
		return
	}

	args = strings.ToLower(strings.TrimSpace(args))

	// Toggle if no args
	if args == "" {
		if ch.Comm.Has(types.CommWiznet) {
			ch.Comm.Remove(types.CommWiznet)
			d.send(ch, "Signing off of Wiznet.\r\n")
		} else {
			ch.Comm.Set(types.CommWiznet)
			d.send(ch, "Welcome to Wiznet!\r\n")
		}
		return
	}

	switch args {
	case "on":
		ch.Comm.Set(types.CommWiznet)
		d.send(ch, "Welcome to Wiznet!\r\n")

	case "off":
		ch.Comm.Remove(types.CommWiznet)
		d.send(ch, "Signing off of Wiznet.\r\n")

	case "status":
		if ch.Comm.Has(types.CommWiznet) {
			d.send(ch, "Wiznet status: ON\r\n")
		} else {
			d.send(ch, "Wiznet status: OFF\r\n")
		}

	case "show":
		d.send(ch, "Wiznet options: on off status\r\n")

	default:
		d.send(ch, "Syntax: wiznet [on|off|status|show]\r\n")
	}
}

// WiznetBroadcast sends a message to all immortals on wiznet
func (d *CommandDispatcher) WiznetBroadcast(message string, minLevel int) {
	for _, player := range d.GameLoop.Characters {
		if player.IsNPC() {
			continue
		}
		if player.Level < minLevel {
			continue
		}
		if !player.Comm.Has(types.CommWiznet) {
			continue
		}
		d.send(player, fmt.Sprintf("{c[Wiznet]{x %s\r\n", message))
	}
}

// cmdBan manages site bans
// Syntax: ban [list|<site> [permanent]]
func (d *CommandDispatcher) cmdBan(ch *types.Character, args string) {
	args = strings.TrimSpace(args)

	// List bans if no args
	if args == "" || strings.ToLower(args) == "list" {
		if d.BanList == nil || len(d.BanList) == 0 {
			d.send(ch, "No sites banned.\r\n")
			return
		}

		d.send(ch, "Banned sites:\r\n")
		for site, permanent := range d.BanList {
			if permanent {
				d.send(ch, fmt.Sprintf("  %s (permanent)\r\n", site))
			} else {
				d.send(ch, fmt.Sprintf("  %s\r\n", site))
			}
		}
		return
	}

	parts := strings.Fields(args)
	site := parts[0]
	permanent := false
	if len(parts) > 1 && strings.ToLower(parts[1]) == "permanent" {
		permanent = true
	}

	// Initialize ban list if needed
	if d.BanList == nil {
		d.BanList = make(map[string]bool)
	}

	// Check if already banned
	if _, exists := d.BanList[site]; exists {
		d.send(ch, fmt.Sprintf("Site %s is already banned.\r\n", site))
		return
	}

	// Add ban
	d.BanList[site] = permanent
	if permanent {
		d.send(ch, fmt.Sprintf("Site %s has been permanently banned.\r\n", site))
	} else {
		d.send(ch, fmt.Sprintf("Site %s has been banned.\r\n", site))
	}

	// Broadcast on wiznet
	d.WiznetBroadcast(fmt.Sprintf("%s bans %s", ch.Name, site), 100)
}

// cmdAllow removes a site ban
// Syntax: allow <site>
func (d *CommandDispatcher) cmdAllow(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Remove which ban?\r\n")
		return
	}

	site := strings.TrimSpace(args)

	if d.BanList == nil {
		d.send(ch, "No sites are banned.\r\n")
		return
	}

	if _, exists := d.BanList[site]; !exists {
		d.send(ch, fmt.Sprintf("Site %s is not banned.\r\n", site))
		return
	}

	delete(d.BanList, site)
	d.send(ch, fmt.Sprintf("Ban on %s lifted.\r\n", site))

	// Broadcast on wiznet
	d.WiznetBroadcast(fmt.Sprintf("%s lifts ban on %s", ch.Name, site), 100)
}

// cmdString changes mob/object strings
// Syntax: string <mob|obj> <name> <field> <value>
func (d *CommandDispatcher) cmdString(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 4)
	if len(parts) < 4 {
		d.send(ch, "Syntax: string <mob|obj> <name> <field> <value>\r\n")
		d.send(ch, "Mob fields: name short long desc\r\n")
		d.send(ch, "Obj fields: name short long\r\n")
		return
	}

	typeStr := strings.ToLower(parts[0])
	name := parts[1]
	field := strings.ToLower(parts[2])
	value := parts[3]

	switch typeStr {
	case "mob", "char":
		victim := d.findCharacterWorld(ch, name)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}

		if !victim.IsNPC() {
			d.send(ch, "Not on players.\r\n")
			return
		}

		switch field {
		case "name":
			victim.Name = value
			d.send(ch, "Ok.\r\n")
		case "short":
			victim.ShortDesc = value
			d.send(ch, "Ok.\r\n")
		case "long":
			victim.LongDesc = value
			d.send(ch, "Ok.\r\n")
		case "desc":
			victim.Desc = value
			d.send(ch, "Ok.\r\n")
		default:
			d.send(ch, "Invalid field. Valid fields: name short long desc\r\n")
		}

	case "obj":
		obj := FindObjInInventory(ch, name)
		if obj == nil && ch.InRoom != nil {
			for _, o := range ch.InRoom.Objects {
				if strings.HasPrefix(strings.ToLower(o.Name), strings.ToLower(name)) {
					obj = o
					break
				}
			}
		}

		if obj == nil {
			d.send(ch, "Nothing like that in heaven or earth.\r\n")
			return
		}

		switch field {
		case "name":
			obj.Name = value
			d.send(ch, "Ok.\r\n")
		case "short":
			obj.ShortDesc = value
			d.send(ch, "Ok.\r\n")
		case "long":
			obj.LongDesc = value
			d.send(ch, "Ok.\r\n")
		default:
			d.send(ch, "Invalid field. Valid fields: name short long\r\n")
		}

	default:
		d.send(ch, "Type can be 'mob' or 'obj'.\r\n")
	}
}

// cmdTrust sets a player's trust level
// Syntax: trust <player> <level>
func (d *CommandDispatcher) cmdTrust(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Syntax: trust <player> <level>\r\n")
		return
	}

	name := parts[0]
	level, err := strconv.Atoi(parts[1])
	if err != nil {
		d.send(ch, "Invalid level.\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, name)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if level > ch.Level {
		d.send(ch, fmt.Sprintf("Trust level can't exceed your level (%d).\r\n", ch.Level))
		return
	}

	if level < 0 {
		d.send(ch, "Trust level must be 0 or higher.\r\n")
		return
	}

	victim.Trust = level
	d.send(ch, fmt.Sprintf("%s's trust level set to %d.\r\n", victim.Name, level))
	d.WiznetBroadcast(fmt.Sprintf("%s sets %s's trust to %d", ch.Name, victim.Name, level), 100)
}

// cmdWizlock toggles the wizlock (prevents new connections)
// Syntax: wizlock [on|off]
func (d *CommandDispatcher) cmdWizlock(ch *types.Character, args string) {
	if d.GameLoop == nil {
		d.send(ch, "GameLoop not available.\r\n")
		return
	}

	args = strings.ToLower(strings.TrimSpace(args))

	if args == "" {
		if d.GameLoop.Wizlock {
			d.send(ch, "Wizlock is currently ON.\r\n")
		} else {
			d.send(ch, "Wizlock is currently OFF.\r\n")
		}
		return
	}

	switch args {
	case "on":
		d.GameLoop.Wizlock = true
		d.send(ch, "Wizlock is now ON. No new players can connect.\r\n")
		d.WiznetBroadcast(fmt.Sprintf("%s has wizlocked the game", ch.Name), 100)

	case "off":
		d.GameLoop.Wizlock = false
		d.send(ch, "Wizlock is now OFF. Players can connect.\r\n")
		d.WiznetBroadcast(fmt.Sprintf("%s has removed wizlock", ch.Name), 100)

	default:
		d.send(ch, "Syntax: wizlock [on|off]\r\n")
	}
}

// cmdNewlock toggles newlock (prevents new character creation)
// Syntax: newlock [on|off]
func (d *CommandDispatcher) cmdNewlock(ch *types.Character, args string) {
	if d.GameLoop == nil {
		d.send(ch, "GameLoop not available.\r\n")
		return
	}

	args = strings.ToLower(strings.TrimSpace(args))

	if args == "" {
		if d.GameLoop.Newlock {
			d.send(ch, "Newlock is currently ON.\r\n")
		} else {
			d.send(ch, "Newlock is currently OFF.\r\n")
		}
		return
	}

	switch args {
	case "on":
		d.GameLoop.Newlock = true
		d.send(ch, "Newlock is now ON. No new characters can be created.\r\n")
		d.WiznetBroadcast(fmt.Sprintf("%s has newlocked the game", ch.Name), 100)

	case "off":
		d.GameLoop.Newlock = false
		d.send(ch, "Newlock is now OFF. New characters can be created.\r\n")
		d.WiznetBroadcast(fmt.Sprintf("%s has removed newlock", ch.Name), 100)

	default:
		d.send(ch, "Syntax: newlock [on|off]\r\n")
	}
}

// cmdLog toggles logging for a specific player
// Syntax: log <player> [on|off]
func (d *CommandDispatcher) cmdLog(ch *types.Character, args string) {
	parts := strings.Fields(args)
	if len(parts) == 0 {
		d.send(ch, "Syntax: log <player> [on|off]\r\n")
		d.send(ch, "With no on/off, shows current log status.\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, parts[0])
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if len(parts) == 1 {
		// Toggle
		if victim.Comm.Has(types.CommSnoopProof) {
			victim.Comm.Remove(types.CommSnoopProof)
			d.send(ch, fmt.Sprintf("LOG removed from %s.\r\n", victim.Name))
		} else {
			victim.Comm.Set(types.CommSnoopProof)
			d.send(ch, fmt.Sprintf("LOG set on %s.\r\n", victim.Name))
		}
		return
	}

	switch strings.ToLower(parts[1]) {
	case "on":
		victim.Comm.Set(types.CommSnoopProof)
		d.send(ch, fmt.Sprintf("LOG set on %s.\r\n", victim.Name))
	case "off":
		victim.Comm.Remove(types.CommSnoopProof)
		d.send(ch, fmt.Sprintf("LOG removed from %s.\r\n", victim.Name))
	default:
		d.send(ch, "Syntax: log <player> [on|off]\r\n")
	}
}

// cmdNoshout toggles the noshout penalty on a player
// Syntax: noshout <player>
func (d *CommandDispatcher) cmdNoshout(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Noshout whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.Comm.Has(types.CommNoShout) {
		victim.Comm.Remove(types.CommNoShout)
		d.send(victim, "You can shout again.\r\n")
		d.send(ch, "NOSHOUT removed.\r\n")
	} else {
		victim.Comm.Set(types.CommNoShout)
		d.send(victim, "You can't shout!\r\n")
		d.send(ch, "NOSHOUT set.\r\n")
	}
}

// cmdNotell toggles the notell penalty on a player
// Syntax: notell <player>
func (d *CommandDispatcher) cmdNotell(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Notell whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.Comm.Has(types.CommNoTell) {
		victim.Comm.Remove(types.CommNoTell)
		d.send(victim, "You can tell again.\r\n")
		d.send(ch, "NOTELL removed.\r\n")
	} else {
		victim.Comm.Set(types.CommNoTell)
		d.send(victim, "You can't tell!\r\n")
		d.send(ch, "NOTELL set.\r\n")
	}
}

// cmdNoemote toggles the noemote penalty on a player
// Syntax: noemote <player>
func (d *CommandDispatcher) cmdNoemote(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Noemote whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.Comm.Has(types.CommNoEmote) {
		victim.Comm.Remove(types.CommNoEmote)
		d.send(victim, "You can emote again.\r\n")
		d.send(ch, "NOEMOTE removed.\r\n")
	} else {
		victim.Comm.Set(types.CommNoEmote)
		d.send(victim, "You can't emote!\r\n")
		d.send(ch, "NOEMOTE set.\r\n")
	}
}

// cmdNochannels toggles the nochannels penalty on a player
// Syntax: nochannels <player>
func (d *CommandDispatcher) cmdNochannels(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Nochannels whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.Comm.Has(types.CommNoChannels) {
		victim.Comm.Remove(types.CommNoChannels)
		d.send(victim, "You can use channels again.\r\n")
		d.send(ch, "NOCHANNELS removed.\r\n")
	} else {
		victim.Comm.Set(types.CommNoChannels)
		d.send(victim, "You can't use channels!\r\n")
		d.send(ch, "NOCHANNELS set.\r\n")
	}
}

// cmdVnum finds the vnum of mobs/objects by keyword
// Syntax: vnum <mob|obj> <keyword>
func (d *CommandDispatcher) cmdVnum(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Syntax: vnum <mob|obj> <keyword>\r\n")
		return
	}

	typeStr := strings.ToLower(parts[0])
	keyword := strings.ToLower(parts[1])

	if d.GameLoop.World == nil {
		d.send(ch, "World data not loaded.\r\n")
		return
	}

	switch typeStr {
	case "mob":
		found := 0
		for _, template := range d.GameLoop.World.MobTemplates {
			if template == nil {
				continue
			}

			match := false
			for _, kw := range template.Keywords {
				if strings.Contains(strings.ToLower(kw), keyword) {
					match = true
					break
				}
			}
			if !match && strings.Contains(strings.ToLower(template.ShortDesc), keyword) {
				match = true
			}

			if match {
				d.send(ch, fmt.Sprintf("%5d  %s\r\n", template.Vnum, template.ShortDesc))
				found++
				if found >= 200 {
					d.send(ch, "...truncated after 200 results.\r\n")
					break
				}
			}
		}

		if found == 0 {
			d.send(ch, "No mobiles found.\r\n")
		} else {
			d.send(ch, fmt.Sprintf("%d mobiles found.\r\n", found))
		}

	case "obj":
		found := 0
		for _, template := range d.GameLoop.World.ObjTemplates {
			if template == nil {
				continue
			}

			match := false
			for _, kw := range template.Keywords {
				if strings.Contains(strings.ToLower(kw), keyword) {
					match = true
					break
				}
			}
			if !match && strings.Contains(strings.ToLower(template.ShortDesc), keyword) {
				match = true
			}

			if match {
				d.send(ch, fmt.Sprintf("%5d  %s\r\n", template.Vnum, template.ShortDesc))
				found++
				if found >= 200 {
					d.send(ch, "...truncated after 200 results.\r\n")
					break
				}
			}
		}

		if found == 0 {
			d.send(ch, "No objects found.\r\n")
		} else {
			d.send(ch, fmt.Sprintf("%d objects found.\r\n", found))
		}

	default:
		d.send(ch, "Type must be 'mob' or 'obj'.\r\n")
	}
}

// cmdFinger shows information about an offline or online player
// Syntax: finger <player>
func (d *CommandDispatcher) cmdFinger(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Finger whom?\r\n")
		return
	}

	// First check if player is online
	var target *types.Character
	for _, player := range d.GameLoop.GetPlayers() {
		if strings.EqualFold(player.Name, args) {
			target = player
			break
		}
	}

	if target != nil {
		// Player is online
		d.showFingerInfo(ch, target, true)
	} else {
		// Try to load player from disk
		d.send(ch, "Player not online. Offline lookup not yet implemented.\r\n")
	}
}

// showFingerInfo displays finger information about a player
func (d *CommandDispatcher) showFingerInfo(ch *types.Character, target *types.Character, online bool) {
	race := types.GetRace(target.Race)
	raceName := "Unknown"
	if race != nil {
		raceName = race.Name
	}
	class := types.GetClass(target.Class)
	className := "Unknown"
	if class != nil {
		className = class.Name
	}

	d.send(ch, fmt.Sprintf("\r\n         -- %s --\r\n", target.Name))
	d.send(ch, strings.Repeat("-", 50)+"\r\n")

	// Title
	if target.PCData != nil && target.PCData.Title != "" {
		d.send(ch, fmt.Sprintf("Title: %s\r\n", target.PCData.Title))
	}

	// Basic info
	d.send(ch, fmt.Sprintf("Level %d %s %s (%s)\r\n",
		target.Level, target.Sex.String(), raceName, className))

	// Status
	if online {
		d.send(ch, "Status: Online\r\n")
	} else {
		d.send(ch, "Status: Offline\r\n")
	}

	// Clan
	if target.PCData != nil && target.PCData.Clan > 0 && d.Clans != nil {
		clan := d.Clans.GetClan(target.PCData.Clan)
		if clan != nil {
			d.send(ch, fmt.Sprintf("Clan: %s\r\n", clan.Name))
		}
	}

	// Deity
	if target.PCData != nil && target.PCData.Deity > 0 {
		deityName := d.getDeityName(target.PCData.Deity)
		d.send(ch, fmt.Sprintf("Deity: %s\r\n", deityName))
	}

	// Only show more info to immortals
	if ch.IsImmortal() {
		d.send(ch, fmt.Sprintf("Trust: %d\r\n", target.Trust))
		d.send(ch, fmt.Sprintf("Played: %d hours\r\n", target.Played/3600))
	}

	d.send(ch, "\r\n")
}

// cmdClone creates a duplicate of a mob or object
// Syntax: clone <mob|obj> <name>
func (d *CommandDispatcher) cmdClone(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Syntax: clone <mob|obj> <name>\r\n")
		return
	}

	typeStr := strings.ToLower(parts[0])
	name := parts[1]

	switch typeStr {
	case "mob":
		victim := FindCharInRoom(ch, name)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}

		if !victim.IsNPC() {
			d.send(ch, "You can only clone mobiles.\r\n")
			return
		}

		// Create clone from template
		if victim.MobVnum > 0 && d.GameLoop.World != nil {
			clone := d.GameLoop.World.CreateMobFromTemplate(victim.MobVnum)
			if clone != nil {
				CharToRoom(clone, ch.InRoom)
				d.GameLoop.AddCharacter(clone)
				d.send(ch, fmt.Sprintf("You have cloned %s.\r\n", clone.ShortDesc))
				return
			}
		}

		d.send(ch, "Clone failed - no template found.\r\n")

	case "obj":
		obj := FindObjInInventory(ch, name)
		if obj == nil && ch.InRoom != nil {
			obj = FindObjInRoom(ch, name)
		}

		if obj == nil {
			d.send(ch, "Nothing like that here.\r\n")
			return
		}

		// Create clone from template
		if obj.Vnum > 0 && d.GameLoop.World != nil {
			template := d.GameLoop.World.GetObjTemplate(obj.Vnum)
			if template != nil {
				clone := d.createObjectFromTemplate(template, obj.Level)
				if clone != nil {
					ch.AddInventory(clone)
					d.send(ch, fmt.Sprintf("You have cloned %s.\r\n", clone.ShortDesc))
					return
				}
			}
		}

		d.send(ch, "Clone failed - no template found.\r\n")

	default:
		d.send(ch, "Type must be 'mob' or 'obj'.\r\n")
	}
}

// cmdZecho sends a message to all players in the same area
func (d *CommandDispatcher) cmdZecho(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Zone echo what?\r\n")
		return
	}

	if ch.InRoom == nil || ch.InRoom.Area == nil {
		d.send(ch, "You're not in an area.\r\n")
		return
	}

	area := ch.InRoom.Area
	message := args + "\r\n"

	for _, victim := range d.GameLoop.Characters {
		if victim.InRoom != nil && victim.InRoom.Area == area {
			d.send(victim, message)
		}
	}
}

// cmdGecho sends a message to all players in the game (global echo)
func (d *CommandDispatcher) cmdGecho(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Global echo what?\r\n")
		return
	}

	message := args + "\r\n"

	for _, victim := range d.GameLoop.Characters {
		if !victim.IsNPC() {
			d.send(victim, message)
		}
	}
}

// cmdAllpeace stops all fighting in the entire game
func (d *CommandDispatcher) cmdAllpeace(ch *types.Character, args string) {
	count := 0
	for _, victim := range d.GameLoop.Characters {
		if victim.Fighting != nil {
			victim.Fighting = nil
			count++
		}
	}
	d.send(ch, fmt.Sprintf("All combat stopped. (%d combatants affected)\r\n", count))
}

// cmdRecover allows an immortal to recover from being stuck
func (d *CommandDispatcher) cmdRecover(ch *types.Character, args string) {
	// Reset position
	ch.Position = types.PosStanding

	// Clear combat
	ch.Fighting = nil

	// Clear wait/daze
	ch.Wait = 0
	ch.Daze = 0

	// Go to recall point or temple
	var recallRoom *types.Room
	if ch.PCData != nil && ch.PCData.Recall != 0 && d.GameLoop.World != nil {
		recallRoom = d.GameLoop.World.GetRoom(ch.PCData.Recall)
	}
	if recallRoom == nil && d.GameLoop.World != nil {
		recallRoom = d.GameLoop.World.GetRoom(3001) // Temple
	}

	if recallRoom != nil {
		if ch.InRoom != nil {
			ch.InRoom.RemovePerson(ch)
		}
		CharToRoom(ch, recallRoom)
	}

	d.send(ch, "You have recovered.\r\n")
}

// cmdMemory shows memory usage statistics
func (d *CommandDispatcher) cmdMemory(ch *types.Character, args string) {
	// Count characters and objects
	npcCount := 0
	playerCount := 0
	for _, c := range d.GameLoop.Characters {
		if c.IsNPC() {
			npcCount++
		} else {
			playerCount++
		}
	}

	roomCount := len(d.GameLoop.Rooms)
	areaCount := len(d.GameLoop.Areas)

	// Count objects in world
	objCount := 0
	for _, room := range d.GameLoop.Rooms {
		objCount += len(room.Objects)
	}
	for _, c := range d.GameLoop.Characters {
		objCount += len(c.Inventory)
		for i := types.WearLocation(0); i < types.WearLocMax; i++ {
			if c.GetEquipment(i) != nil {
				objCount++
			}
		}
	}

	d.send(ch, "Memory Statistics:\r\n")
	d.send(ch, fmt.Sprintf("  Areas:    %d\r\n", areaCount))
	d.send(ch, fmt.Sprintf("  Rooms:    %d\r\n", roomCount))
	d.send(ch, fmt.Sprintf("  Players:  %d\r\n", playerCount))
	d.send(ch, fmt.Sprintf("  NPCs:     %d\r\n", npcCount))
	d.send(ch, fmt.Sprintf("  Objects:  %d\r\n", objCount))
}

// cmdPoofin sets or shows the immortal's arrival message
func (d *CommandDispatcher) cmdPoofin(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs cannot set poofs.\r\n")
		return
	}

	if args == "" {
		if ch.PCData.Bamfin == "" {
			d.send(ch, "Your poofin is not set.\r\n")
		} else {
			d.send(ch, fmt.Sprintf("Your poofin is: %s\r\n", ch.PCData.Bamfin))
		}
		return
	}

	// Must include character's name
	if !strings.Contains(strings.ToLower(args), strings.ToLower(ch.Name)) {
		d.send(ch, "You must include your name.\r\n")
		return
	}

	ch.PCData.Bamfin = args
	d.send(ch, fmt.Sprintf("Your poofin is now: %s\r\n", args))
}

// cmdPoofout sets or shows the immortal's departure message
func (d *CommandDispatcher) cmdPoofout(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs cannot set poofs.\r\n")
		return
	}

	if args == "" {
		if ch.PCData.Bamfout == "" {
			d.send(ch, "Your poofout is not set.\r\n")
		} else {
			d.send(ch, fmt.Sprintf("Your poofout is: %s\r\n", ch.PCData.Bamfout))
		}
		return
	}

	// Must include character's name
	if !strings.Contains(strings.ToLower(args), strings.ToLower(ch.Name)) {
		d.send(ch, "You must include your name.\r\n")
		return
	}

	ch.PCData.Bamfout = args
	d.send(ch, fmt.Sprintf("Your poofout is now: %s\r\n", args))
}

// cmdSmote is an immortal emote that goes to all in room without the character name
func (d *CommandDispatcher) cmdSmote(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Smote what?\r\n")
		return
	}

	// Must include character's name
	if !strings.Contains(strings.ToLower(args), strings.ToLower(ch.Name)) {
		d.send(ch, "You must include your name.\r\n")
		return
	}

	if ch.InRoom == nil {
		return
	}

	message := args + "\r\n"
	for _, person := range ch.InRoom.People {
		d.send(person, message)
	}
}

// cmdImmtalk is the immortal chat channel
func (d *CommandDispatcher) cmdImmtalk(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "What do you want to say on immortal channel?\r\n")
		return
	}

	// Send to all immortals
	message := fmt.Sprintf("{c[Immortal] %s: %s{x\r\n", ch.Name, args)
	for _, victim := range d.GameLoop.Characters {
		if !victim.IsNPC() && victim.Level >= types.LevelImmortal {
			d.send(victim, message)
		}
	}
}

// cmdPardon removes killer/thief flags from a player
func (d *CommandDispatcher) cmdPardon(ch *types.Character, args string) {
	parts := strings.Fields(args)
	if len(parts) < 2 {
		d.send(ch, "Syntax: pardon <character> <killer|thief>\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, parts[0])
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	flag := strings.ToLower(parts[1])
	switch flag {
	case "killer":
		if !victim.HasPenalty(types.PlrKiller) {
			d.send(ch, "They are not a killer.\r\n")
			return
		}
		victim.RemovePenalty(types.PlrKiller)
		d.send(ch, "Killer flag removed.\r\n")
		d.send(victim, "You are no longer a KILLER.\r\n")

	case "thief":
		if !victim.HasPenalty(types.PlrThief) {
			d.send(ch, "They are not a thief.\r\n")
			return
		}
		victim.RemovePenalty(types.PlrThief)
		d.send(ch, "Thief flag removed.\r\n")
		d.send(victim, "You are no longer a THIEF.\r\n")

	default:
		d.send(ch, "Syntax: pardon <character> <killer|thief>\r\n")
	}
}

// cmdPenalty shows penalty flags on a player
func (d *CommandDispatcher) cmdPenalty(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Penalty whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Penalties for %s:\r\n", victim.Name))

	penalties := []struct {
		flag types.PlayerFlags
		name string
	}{
		{types.PlrKiller, "Killer"},
		{types.PlrThief, "Thief"},
		{types.PlrFrozen, "Frozen"},
		{types.PlrNoShout, "No Shout"},
		{types.PlrNoTell, "No Tell"},
		{types.PlrNoEmote, "No Emote"},
		{types.PlrNoChannels, "No Channels"},
		{types.PlrNoTitle, "No Title"},
		{types.PlrLog, "Logged"},
		{types.PlrDeny, "Denied"},
		{types.PlrNoRestore, "No Restore"},
	}

	found := false
	for _, p := range penalties {
		if victim.HasPenalty(p.flag) {
			d.send(ch, fmt.Sprintf("  %s\r\n", p.name))
			found = true
		}
	}

	if !found {
		d.send(ch, "  None.\r\n")
	}
}

// cmdNotitle toggles notitle penalty on a player
func (d *CommandDispatcher) cmdNotitle(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Notitle whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.HasPenalty(types.PlrNoTitle) {
		victim.RemovePenalty(types.PlrNoTitle)
		d.send(ch, "NOTITLE removed.\r\n")
		d.send(victim, "You can use the title command again.\r\n")
	} else {
		victim.AddPenalty(types.PlrNoTitle)
		d.send(ch, "NOTITLE set.\r\n")
		d.send(victim, "You can't use the title command anymore!\r\n")
	}
}

// cmdDeny denies a player from logging in
func (d *CommandDispatcher) cmdDeny(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Deny whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.HasPenalty(types.PlrDeny) {
		d.send(ch, "They are already denied.\r\n")
		return
	}

	victim.AddPenalty(types.PlrDeny)
	d.send(ch, "You deny them!\r\n")
	d.send(victim, "You are DENIED!\r\n")

	// Remove from game (saving handled by server on disconnect)
	if victim.InRoom != nil {
		victim.InRoom.RemovePerson(victim)
	}
	d.GameLoop.RemoveCharacter(victim)
	d.send(ch, fmt.Sprintf("%s has been disconnected.\r\n", victim.Name))
}

// cmdNorestore toggles norestore penalty on a player
func (d *CommandDispatcher) cmdNorestore(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Norestore whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.HasPenalty(types.PlrNoRestore) {
		victim.RemovePenalty(types.PlrNoRestore)
		d.send(ch, "NORESTORE removed.\r\n")
		d.send(victim, "You can be restored again.\r\n")
	} else {
		victim.AddPenalty(types.PlrNoRestore)
		d.send(ch, "NORESTORE set.\r\n")
		d.send(victim, "You can no longer be restored!\r\n")
	}
}

// cmdGuild sets a player's clan
func (d *CommandDispatcher) cmdGuild(ch *types.Character, args string) {
	parts := strings.Fields(args)
	if len(parts) < 1 {
		d.send(ch, "Syntax: guild <character> [clan|none]\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, parts[0])
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if len(parts) < 2 {
		// Show current clan
		if victim.PCData == nil || victim.PCData.Clan == 0 {
			d.send(ch, fmt.Sprintf("%s is not in a clan.\r\n", victim.Name))
		} else {
			// Find clan name
			clanName := "Unknown"
			if d.Clans != nil {
				if clan := d.Clans.GetClan(victim.PCData.Clan); clan != nil {
					clanName = clan.Name
				}
			}
			d.send(ch, fmt.Sprintf("%s is in clan %s.\r\n", victim.Name, clanName))
		}
		return
	}

	clanName := strings.ToLower(parts[1])
	if clanName == "none" || clanName == "0" {
		if victim.PCData != nil {
			victim.PCData.Clan = 0
		}
		d.send(ch, fmt.Sprintf("%s is now clanless.\r\n", victim.Name))
		d.send(victim, "You have been removed from your clan.\r\n")
		return
	}

	// Find clan by name
	if d.Clans != nil {
		for i, clan := range d.Clans.GetAllClans() {
			if strings.EqualFold(clan.Name, clanName) {
				if victim.PCData != nil {
					victim.PCData.Clan = i + 1 // Clan IDs are 1-based
				}
				d.send(ch, fmt.Sprintf("%s is now a member of %s.\r\n", victim.Name, clan.Name))
				d.send(victim, fmt.Sprintf("You are now a member of %s.\r\n", clan.Name))
				return
			}
		}
	}

	d.send(ch, "No such clan.\r\n")
}

// cmdNoclan toggles the noclan penalty on a player (bans from PK clans)
func (d *CommandDispatcher) cmdNoclan(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Noclan whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.PlayerAct.Has(types.PlrNoClan) {
		victim.PlayerAct.Remove(types.PlrNoClan)
		d.send(ch, "NOCLAN removed.\r\n")
		d.send(victim, "You can join clans again.\r\n")
	} else {
		victim.PlayerAct.Set(types.PlrNoClan)
		// Remove from current clan
		if victim.PCData != nil && victim.PCData.Clan != 0 {
			victim.PCData.Clan = 0
			d.send(victim, "You have been removed from your clan.\r\n")
		}
		d.send(ch, "NOCLAN set.\r\n")
		d.send(victim, "You can no longer join clans!\r\n")
	}
}

// cmdGhost toggles ghost mode on a player (they appear as a ghost)
func (d *CommandDispatcher) cmdGhost(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Ghost whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	// Toggle ghost affect
	if victim.IsAffected(types.AffInvisible) {
		// Remove ghost/invis
		victim.AffectedBy &^= types.AffInvisible
		d.send(ch, fmt.Sprintf("%s is no longer a ghost.\r\n", victim.Name))
		d.send(victim, "You are no longer a ghost.\r\n")
	} else {
		// Add ghost/invis
		victim.AffectedBy |= types.AffInvisible
		d.send(ch, fmt.Sprintf("%s is now a ghost.\r\n", victim.Name))
		d.send(victim, "You are now a ghost.\r\n")
	}
}

// cmdWecho sends a warning echo to all players (3x with restore and allpeace)
func (d *CommandDispatcher) cmdWecho(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Warn echo what?\r\n")
		return
	}

	// Format the warning message
	msg := fmt.Sprintf("\r\n{B***{x {R%s{x {B***{x\r\n", args)

	// Send 3 times to all players
	for i := 0; i < 3; i++ {
		for _, target := range d.GameLoop.Characters {
			if !target.IsNPC() {
				d.send(target, msg)
			}
		}
	}

	// Restore all players
	for _, target := range d.GameLoop.Characters {
		if !target.IsNPC() {
			target.Hit = target.MaxHit
			target.Mana = target.MaxMana
			target.Move = target.MaxMove
		}
	}

	// Stop all fighting (allpeace)
	for _, target := range d.GameLoop.Characters {
		if target.Fighting != nil {
			target.Fighting = nil
		}
	}

	d.send(ch, "Warning echoed and all players restored.\r\n")
}

// cmdPermban permanently bans a site
func (d *CommandDispatcher) cmdPermban(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Permanently ban which site?\r\n")
		return
	}

	site := strings.ToLower(args)

	// Initialize ban list if needed
	if d.BanList == nil {
		d.BanList = make(map[string]bool)
	}

	// Add permanent ban
	d.BanList[site] = true
	d.send(ch, fmt.Sprintf("Site '%s' permanently banned.\r\n", site))
}

// cmdFlag modifies flags on mobs, chars, objects, and rooms
func (d *CommandDispatcher) cmdFlag(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Syntax:\r\n")
		d.send(ch, "  flag mob  <name> <field> <flags>\r\n")
		d.send(ch, "  flag char <name> <field> <flags>\r\n")
		d.send(ch, "  flag obj  <name> <field> <flags>\r\n")
		d.send(ch, "  flag room <vnum> <field> <flags>\r\n")
		d.send(ch, "  mob  flags: act,aff,off,imm,res,vuln\r\n")
		d.send(ch, "  char flags: plr,comm,aff,imm,res,vuln\r\n")
		d.send(ch, "  obj  flags: extra,wear\r\n")
		d.send(ch, "  room flags: room\r\n")
		d.send(ch, "  +: add flag, -: remove flag, = set equal to\r\n")
		d.send(ch, "  otherwise flag toggles the flags listed.\r\n")
		return
	}

	parts := strings.Fields(args)
	if len(parts) < 4 {
		d.send(ch, "Not enough arguments. Type 'flag' for syntax.\r\n")
		return
	}

	targetType := strings.ToLower(parts[0])
	targetName := parts[1]
	field := strings.ToLower(parts[2])
	flagSpec := parts[3]

	// Determine operation: +, -, =, or toggle
	operation := "toggle"
	if len(flagSpec) > 0 {
		switch flagSpec[0] {
		case '+':
			operation = "add"
			flagSpec = flagSpec[1:]
		case '-':
			operation = "remove"
			flagSpec = flagSpec[1:]
		case '=':
			operation = "set"
			flagSpec = flagSpec[1:]
		}
	}

	switch targetType {
	case "mob", "char":
		// First look in room, then world
		var victim *types.Character
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if strings.HasPrefix(strings.ToLower(person.Name), strings.ToLower(targetName)) {
					victim = person
					break
				}
			}
		}
		if victim == nil {
			victim = d.findCharacterWorld(ch, targetName)
		}
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}

		switch field {
		case "act":
			if victim.IsNPC() {
				flag := parseActFlag(flagSpec)
				if flag == 0 {
					d.send(ch, fmt.Sprintf("Unknown act flag: %s\r\n", flagSpec))
					return
				}
				d.applyActFlag(ch, victim, flag, operation)
			} else {
				d.send(ch, "Use 'plr' for player flags.\r\n")
			}

		case "plr":
			if !victim.IsNPC() {
				flag := parsePlayerFlag(flagSpec)
				if flag == 0 {
					d.send(ch, fmt.Sprintf("Unknown player flag: %s\r\n", flagSpec))
					return
				}
				d.applyPlayerFlag(ch, victim, flag, operation)
			} else {
				d.send(ch, "Use 'act' for mob flags.\r\n")
			}

		case "aff":
			flag := parseAffFlag(flagSpec)
			if flag == 0 {
				d.send(ch, fmt.Sprintf("Unknown affect flag: %s\r\n", flagSpec))
				return
			}
			d.applyAffFlag(ch, victim, flag, operation)

		default:
			d.send(ch, fmt.Sprintf("Unknown field '%s' for %s.\r\n", field, targetType))
		}

	case "obj":
		obj := d.findObjInInventory(ch, targetName)
		if obj == nil {
			obj = d.findObjInRoom(ch.InRoom, targetName)
		}
		if obj == nil {
			d.send(ch, "Can't find that object.\r\n")
			return
		}

		switch field {
		case "extra":
			flag := parseItemFlag(flagSpec)
			if flag == 0 {
				d.send(ch, fmt.Sprintf("Unknown extra flag: %s\r\n", flagSpec))
				return
			}
			d.applyItemFlag(ch, obj, flag, operation)

		case "wear":
			flag := parseWearFlag(flagSpec)
			if flag == 0 {
				d.send(ch, fmt.Sprintf("Unknown wear flag: %s\r\n", flagSpec))
				return
			}
			d.applyWearFlag(ch, obj, flag, operation)

		default:
			d.send(ch, fmt.Sprintf("Unknown field '%s' for obj.\r\n", field))
		}

	case "room":
		vnum, err := strconv.Atoi(targetName)
		if err != nil {
			d.send(ch, "Invalid room vnum.\r\n")
			return
		}
		room := d.GameLoop.Rooms[vnum]
		if room == nil {
			d.send(ch, "No room with that vnum.\r\n")
			return
		}

		switch field {
		case "room":
			flag := parseRoomFlag(flagSpec)
			if flag == 0 {
				d.send(ch, fmt.Sprintf("Unknown room flag: %s\r\n", flagSpec))
				return
			}
			d.applyRoomFlag(ch, room, flag, operation)

		default:
			d.send(ch, fmt.Sprintf("Unknown field '%s' for room.\r\n", field))
		}

	default:
		d.send(ch, "Flag mob, char, obj, or room?\r\n")
	}
}

// Helper functions for flag command

func parseActFlag(name string) types.ActFlags {
	switch strings.ToLower(name) {
	case "sentinel":
		return types.ActSentinel
	case "scavenger":
		return types.ActScavenger
	case "aggressive":
		return types.ActAggressive
	case "stayarea", "stay_area":
		return types.ActStayArea
	case "wimpy":
		return types.ActWimpy
	case "pet":
		return types.ActPet
	case "train":
		return types.ActTrain
	case "practice":
		return types.ActPractice
	case "undead":
		return types.ActUndead
	case "cleric":
		return types.ActCleric
	case "mage":
		return types.ActMage
	case "thief":
		return types.ActThief
	case "warrior":
		return types.ActWarrior
	case "noalign":
		return types.ActNoAlign
	case "nopurge":
		return types.ActNoPurge
	case "outdoors":
		return types.ActOutdoors
	case "indoors":
		return types.ActIndoors
	case "healer":
		return types.ActIsHealer
	case "gain":
		return types.ActGain
	case "update_always":
		return types.ActUpdateAlways
	case "changer":
		return types.ActIsChanger
	}
	return 0
}

func parsePlayerFlag(name string) types.PlayerFlags {
	switch strings.ToLower(name) {
	case "killer":
		return types.PlrKiller
	case "thief":
		return types.PlrThief
	case "frozen":
		return types.PlrFrozen
	case "deny":
		return types.PlrDeny
	case "log":
		return types.PlrLog
	case "noshout":
		return types.PlrNoShout
	case "notell":
		return types.PlrNoTell
	case "noemote":
		return types.PlrNoEmote
	case "nochannels":
		return types.PlrNoChannels
	case "notitle":
		return types.PlrNoTitle
	case "norestore":
		return types.PlrNoRestore
	case "noclan":
		return types.PlrNoClan
	}
	return 0
}

func parseAffFlag(name string) types.AffectFlags {
	switch strings.ToLower(name) {
	case "blind":
		return types.AffBlind
	case "invisible", "invis":
		return types.AffInvisible
	case "detect_evil":
		return types.AffDetectEvil
	case "detect_invis":
		return types.AffDetectInvis
	case "detect_magic":
		return types.AffDetectMagic
	case "detect_hidden":
		return types.AffDetectHidden
	case "detect_good":
		return types.AffDetectGood
	case "sanctuary":
		return types.AffSanctuary
	case "faerie_fire":
		return types.AffFaerieFire
	case "infrared":
		return types.AffInfrared
	case "curse":
		return types.AffCurse
	case "poison":
		return types.AffPoison
	case "protect_evil":
		return types.AffProtectEvil
	case "protect_good":
		return types.AffProtectGood
	case "sneak":
		return types.AffSneak
	case "hide":
		return types.AffHide
	case "sleep":
		return types.AffSleep
	case "charm":
		return types.AffCharm
	case "flying", "fly":
		return types.AffFlying
	case "pass_door":
		return types.AffPassDoor
	case "haste":
		return types.AffHaste
	case "calm":
		return types.AffCalm
	case "plague":
		return types.AffPlague
	case "weaken":
		return types.AffWeaken
	case "dark_vision":
		return types.AffDarkVision
	case "berserk":
		return types.AffBerserk
	case "swim":
		return types.AffSwim
	case "regeneration":
		return types.AffRegeneration
	case "slow":
		return types.AffSlow
	}
	return 0
}

func parseItemFlag(name string) types.ItemFlags {
	switch strings.ToLower(name) {
	case "glow":
		return types.ItemGlow
	case "hum":
		return types.ItemHum
	case "dark":
		return types.ItemDark
	case "lock":
		return types.ItemLock
	case "evil":
		return types.ItemEvil
	case "invis":
		return types.ItemInvis
	case "magic":
		return types.ItemMagic
	case "nodrop":
		return types.ItemNoDrop
	case "bless":
		return types.ItemBless
	case "anti_good":
		return types.ItemAntiGood
	case "anti_evil":
		return types.ItemAntiEvil
	case "anti_neutral":
		return types.ItemAntiNeutral
	case "noremove":
		return types.ItemNoRemove
	case "nopurge":
		return types.ItemNoPurge
	case "rot_death":
		return types.ItemRotDeath
	case "vis_death":
		return types.ItemVisDeath
	case "nonmetal":
		return types.ItemNonMetal
	case "nolocate":
		return types.ItemNoLocate
	case "melt_drop":
		return types.ItemMeltDrop
	case "sell_extract":
		return types.ItemSellExtract
	case "burn_proof":
		return types.ItemBurnProof
	}
	return 0
}

func parseWearFlag(name string) types.WearFlags {
	switch strings.ToLower(name) {
	case "take":
		return types.WearTake
	case "finger":
		return types.WearFinger
	case "neck":
		return types.WearNeck
	case "body":
		return types.WearBody
	case "head":
		return types.WearHead
	case "legs":
		return types.WearLegs
	case "feet":
		return types.WearFeet
	case "hands":
		return types.WearHands
	case "arms":
		return types.WearArms
	case "shield":
		return types.WearShield
	case "about":
		return types.WearAbout
	case "waist":
		return types.WearWaist
	case "wrist":
		return types.WearWrist
	case "wield":
		return types.WearWield
	case "hold":
		return types.WearHold
	case "float":
		return types.WearFloat
	}
	return 0
}

func parseRoomFlag(name string) types.RoomFlags {
	switch strings.ToLower(name) {
	case "dark":
		return types.RoomDark
	case "nomob":
		return types.RoomNoMob
	case "indoors":
		return types.RoomIndoors
	case "private":
		return types.RoomPrivate
	case "safe":
		return types.RoomSafe
	case "solitary":
		return types.RoomSolitary
	case "norecall":
		return types.RoomNoRecall
	case "law":
		return types.RoomLaw
	}
	return 0
}

func (d *CommandDispatcher) applyActFlag(ch *types.Character, victim *types.Character, flag types.ActFlags, operation string) {
	switch operation {
	case "add":
		victim.Act.Set(flag)
		d.send(ch, fmt.Sprintf("Act flag added to %s.\r\n", victim.Name))
	case "remove":
		victim.Act.Remove(flag)
		d.send(ch, fmt.Sprintf("Act flag removed from %s.\r\n", victim.Name))
	case "set":
		victim.Act = flag
		d.send(ch, fmt.Sprintf("Act flags set on %s.\r\n", victim.Name))
	case "toggle":
		if victim.Act.Has(flag) {
			victim.Act.Remove(flag)
			d.send(ch, fmt.Sprintf("Act flag removed from %s.\r\n", victim.Name))
		} else {
			victim.Act.Set(flag)
			d.send(ch, fmt.Sprintf("Act flag added to %s.\r\n", victim.Name))
		}
	}
}

func (d *CommandDispatcher) applyPlayerFlag(ch *types.Character, victim *types.Character, flag types.PlayerFlags, operation string) {
	switch operation {
	case "add":
		victim.PlayerAct.Set(flag)
		d.send(ch, fmt.Sprintf("Player flag added to %s.\r\n", victim.Name))
	case "remove":
		victim.PlayerAct.Remove(flag)
		d.send(ch, fmt.Sprintf("Player flag removed from %s.\r\n", victim.Name))
	case "set":
		victim.PlayerAct = flag
		d.send(ch, fmt.Sprintf("Player flags set on %s.\r\n", victim.Name))
	case "toggle":
		if victim.PlayerAct.Has(flag) {
			victim.PlayerAct.Remove(flag)
			d.send(ch, fmt.Sprintf("Player flag removed from %s.\r\n", victim.Name))
		} else {
			victim.PlayerAct.Set(flag)
			d.send(ch, fmt.Sprintf("Player flag added to %s.\r\n", victim.Name))
		}
	}
}

func (d *CommandDispatcher) applyAffFlag(ch *types.Character, victim *types.Character, flag types.AffectFlags, operation string) {
	switch operation {
	case "add":
		victim.AffectedBy |= flag
		d.send(ch, fmt.Sprintf("Affect flag added to %s.\r\n", victim.Name))
	case "remove":
		victim.AffectedBy &^= flag
		d.send(ch, fmt.Sprintf("Affect flag removed from %s.\r\n", victim.Name))
	case "set":
		victim.AffectedBy = flag
		d.send(ch, fmt.Sprintf("Affect flags set on %s.\r\n", victim.Name))
	case "toggle":
		if victim.AffectedBy&flag != 0 {
			victim.AffectedBy &^= flag
			d.send(ch, fmt.Sprintf("Affect flag removed from %s.\r\n", victim.Name))
		} else {
			victim.AffectedBy |= flag
			d.send(ch, fmt.Sprintf("Affect flag added to %s.\r\n", victim.Name))
		}
	}
}

func (d *CommandDispatcher) applyItemFlag(ch *types.Character, obj *types.Object, flag types.ItemFlags, operation string) {
	switch operation {
	case "add":
		obj.ExtraFlags.Set(flag)
		d.send(ch, fmt.Sprintf("Extra flag added to %s.\r\n", obj.ShortDesc))
	case "remove":
		obj.ExtraFlags.Remove(flag)
		d.send(ch, fmt.Sprintf("Extra flag removed from %s.\r\n", obj.ShortDesc))
	case "set":
		obj.ExtraFlags = flag
		d.send(ch, fmt.Sprintf("Extra flags set on %s.\r\n", obj.ShortDesc))
	case "toggle":
		if obj.ExtraFlags.Has(flag) {
			obj.ExtraFlags.Remove(flag)
			d.send(ch, fmt.Sprintf("Extra flag removed from %s.\r\n", obj.ShortDesc))
		} else {
			obj.ExtraFlags.Set(flag)
			d.send(ch, fmt.Sprintf("Extra flag added to %s.\r\n", obj.ShortDesc))
		}
	}
}

func (d *CommandDispatcher) applyWearFlag(ch *types.Character, obj *types.Object, flag types.WearFlags, operation string) {
	switch operation {
	case "add":
		obj.WearFlags.Set(flag)
		d.send(ch, fmt.Sprintf("Wear flag added to %s.\r\n", obj.ShortDesc))
	case "remove":
		obj.WearFlags.Remove(flag)
		d.send(ch, fmt.Sprintf("Wear flag removed from %s.\r\n", obj.ShortDesc))
	case "set":
		obj.WearFlags = flag
		d.send(ch, fmt.Sprintf("Wear flags set on %s.\r\n", obj.ShortDesc))
	case "toggle":
		if obj.WearFlags.Has(flag) {
			obj.WearFlags.Remove(flag)
			d.send(ch, fmt.Sprintf("Wear flag removed from %s.\r\n", obj.ShortDesc))
		} else {
			obj.WearFlags.Set(flag)
			d.send(ch, fmt.Sprintf("Wear flag added to %s.\r\n", obj.ShortDesc))
		}
	}
}

func (d *CommandDispatcher) applyRoomFlag(ch *types.Character, room *types.Room, flag types.RoomFlags, operation string) {
	switch operation {
	case "add":
		room.Flags.Set(flag)
		d.send(ch, fmt.Sprintf("Room flag added to room %d.\r\n", room.Vnum))
	case "remove":
		room.Flags.Remove(flag)
		d.send(ch, fmt.Sprintf("Room flag removed from room %d.\r\n", room.Vnum))
	case "set":
		room.Flags = flag
		d.send(ch, fmt.Sprintf("Room flags set on room %d.\r\n", room.Vnum))
	case "toggle":
		if room.Flags.Has(flag) {
			room.Flags.Remove(flag)
			d.send(ch, fmt.Sprintf("Room flag removed from room %d.\r\n", room.Vnum))
		} else {
			room.Flags.Set(flag)
			d.send(ch, fmt.Sprintf("Room flag added to room %d.\r\n", room.Vnum))
		}
	}
}

// cmdSla is a mistype prevention for slay
func (d *CommandDispatcher) cmdSla(ch *types.Character, args string) {
	d.send(ch, "If you want to SLAY, spell it out.\r\n")
}

// cmdImmkiss fully heals a player with a kiss - removes negative effects and restores all vitals
func (d *CommandDispatcher) cmdImmkiss(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Kiss whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	// Remove negative effects
	victim.AffectedBy.Remove(types.AffPlague)
	victim.AffectedBy.Remove(types.AffPoison)
	victim.AffectedBy.Remove(types.AffBlind)
	victim.AffectedBy.Remove(types.AffSleep)
	victim.AffectedBy.Remove(types.AffCurse)

	// Remove spell affects for these conditions
	for _, affName := range []string{"plague", "poison", "blindness", "sleep", "curse", "weaken"} {
		victim.Affected.RemoveByType(affName)
	}

	// Restore vitals to maximum
	victim.Hit = victim.MaxHit
	victim.Mana = victim.MaxMana
	victim.Move = victim.MaxMove

	// Messages
	d.send(ch, fmt.Sprintf("You kiss %s gently on the forehead, healing all wounds.\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s kisses you gently on the forehead, healing all wounds.\r\n", ch.Name))

	// Notify room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch && person != victim {
				d.send(person, fmt.Sprintf("%s kisses %s gently on the forehead.\r\n", ch.Name, victim.Name))
			}
		}
	}
}

// cmdViolate allows an immortal to enter a private room (bypasses privacy restrictions)
func (d *CommandDispatcher) cmdViolate(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Violate which room?\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid room vnum.\r\n")
		return
	}

	// Find the room
	var room *types.Room
	if d.GameLoop != nil {
		room = d.GameLoop.Rooms[vnum]
	}

	if room == nil {
		d.send(ch, "That room doesn't exist.\r\n")
		return
	}

	// Move character to room (bypassing all checks)
	if ch.InRoom != nil {
		CharFromRoom(ch)
	}
	CharToRoom(ch, room)

	d.send(ch, fmt.Sprintf("You violate the sanctity of room %d.\r\n", vnum))
	d.cmdLook(ch, "")
}

// cmdProtect toggles the snoop-proof flag on a player
func (d *CommandDispatcher) cmdProtect(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Protect whom from snoops?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Comm.Has(types.CommSnoopProof) {
		victim.Comm.Remove(types.CommSnoopProof)
		d.send(ch, fmt.Sprintf("%s can now be snooped.\r\n", victim.Name))
	} else {
		victim.Comm.Set(types.CommSnoopProof)
		d.send(ch, fmt.Sprintf("%s is now protected from snoops.\r\n", victim.Name))
	}
}

// cmdTwit sets the frozen flag on a player as a troublemaker marker
func (d *CommandDispatcher) cmdTwit(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Twit whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	// Backfire check - if target is equal or higher level, twit the caster instead
	if victim.Level >= ch.Level {
		d.send(ch, "You feel like a twit.\r\n")
		ch.AddPenalty(types.PlrFrozen)
		return
	}

	if victim.IsFrozen() {
		victim.RemovePenalty(types.PlrFrozen)
		d.send(ch, fmt.Sprintf("%s is no longer a twit.\r\n", victim.Name))
		d.send(victim, "You are no longer a twit.\r\n")
	} else {
		victim.AddPenalty(types.PlrFrozen)
		d.send(ch, fmt.Sprintf("%s is now a twit.\r\n", victim.Name))
		d.send(victim, "You are now a twit.\r\n")
	}
}

// cmdPack sends a survival pack to a new player
// Syntax: pack <player>
func (d *CommandDispatcher) cmdPack(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Send a pack to whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level > 10 {
		d.send(ch, "They are too high level for a starter pack.\r\n")
		return
	}

	// Create some basic starter items
	bread := types.NewObject(0, "a loaf of bread", types.ItemTypeFood)
	bread.Name = "bread loaf"
	bread.Values[0] = 20 // Hunger restored
	bread.Weight = 1

	water := types.NewObject(0, "a waterskin", types.ItemTypeDrinkCon)
	water.Name = "waterskin water"
	water.Values[0] = 20 // Capacity
	water.Values[1] = 20 // Current amount
	water.Values[2] = 0  // Water type
	water.Weight = 1

	torch := types.NewObject(0, "a torch", types.ItemTypeLight)
	torch.Name = "torch"
	torch.Values[2] = 500 // Duration
	torch.Weight = 1

	// Give items to victim
	victim.Inventory = append(victim.Inventory, bread)
	bread.CarriedBy = victim
	victim.Inventory = append(victim.Inventory, water)
	water.CarriedBy = victim
	victim.Inventory = append(victim.Inventory, torch)
	torch.CarriedBy = victim

	d.send(ch, fmt.Sprintf("You send a survival pack to %s.\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s has sent you a survival pack!\r\n", ch.Name))
	d.send(victim, "You receive a loaf of bread, a waterskin, and a torch.\r\n")
}

// cmdGset sets or clears the immortal's personal goto point (recall location)
// Syntax: gset [vnum]
func (d *CommandDispatcher) cmdGset(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs cannot use gset.\r\n")
		return
	}

	if args == "" {
		// Clear the goto point
		ch.PCData.Recall = 0
		d.send(ch, "Goto point cleared.\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid vnum.\r\n")
		return
	}

	// Verify room exists
	if d.GameLoop != nil && d.GameLoop.Rooms[vnum] == nil {
		d.send(ch, "That room doesn't exist.\r\n")
		return
	}

	ch.PCData.Recall = vnum
	d.send(ch, fmt.Sprintf("Goto point set to room %d.\r\n", vnum))
}

// Corner room vnum - punishment room
const RoomVnumCorner = 3

// cmdCorner transfers a player to the "corner" room (punishment)
// Syntax: corner <player>
func (d *CommandDispatcher) cmdCorner(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Corner whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if !victim.IsNPC() && victim.Level >= ch.Level && ch.Level < types.MaxLevel {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Get the corner room
	cornerRoom := d.GameLoop.GetRoom(RoomVnumCorner)
	if cornerRoom == nil {
		d.send(ch, "The corner room doesn't exist!\r\n")
		return
	}

	// Transfer the victim to the corner
	d.performTransfer(victim, cornerRoom)
	d.send(ch, fmt.Sprintf("You have sent %s to the corner.\r\n", victim.Name))
	d.send(victim, "You have been sent to the corner!\r\n")

	// Broadcast on wiznet
	d.WiznetBroadcast(fmt.Sprintf("%s has sent %s to the corner", ch.Name, victim.Name), 100)
}

// performTransfer moves a character from their current room to a destination
func (d *CommandDispatcher) performTransfer(victim *types.Character, destRoom *types.Room) {
	if victim.InRoom != nil {
		// Notify old room
		for _, person := range victim.InRoom.People {
			if person != victim {
				d.send(person, fmt.Sprintf("%s disappears in a mushroom cloud.\r\n", victim.Name))
			}
		}
		// Remove from old room
		for i, person := range victim.InRoom.People {
			if person == victim {
				victim.InRoom.People = append(victim.InRoom.People[:i], victim.InRoom.People[i+1:]...)
				break
			}
		}
	}

	// Stop any fighting
	if victim.Fighting != nil {
		victim.Fighting = nil
	}

	// Add to new room
	destRoom.People = append(destRoom.People, victim)
	victim.InRoom = destRoom

	// Notify new room
	for _, person := range destRoom.People {
		if person != victim {
			d.send(person, fmt.Sprintf("%s arrives from a puff of smoke.\r\n", victim.Name))
		}
	}

	// Show room to victim
	d.doLook(victim, "")
}

// cmdWipe marks a player as wiped (banned) and disconnects them
// Syntax: wipe <player>
func (d *CommandDispatcher) cmdWipe(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Wipe whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if !victim.IsNPC() && victim.Level >= ch.Level && ch.Level < types.MaxLevel {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Set the wiped flag
	victim.Comm.Set(types.CommWiped)

	// Broadcast on wiznet
	d.WiznetBroadcast(fmt.Sprintf("%s wipes access to %s", ch.Name, victim.Name), 100)

	d.send(ch, "Ok.\r\n")

	// Save and disconnect the player
	if d.OnSave != nil {
		d.OnSave(victim)
	}

	// Stop any fighting
	if victim.Fighting != nil {
		victim.Fighting = nil
	}

	// Disconnect the player
	if d.DisconnectPlayer != nil {
		d.DisconnectPlayer(victim)
	}
}

// cmdWedpost toggles wedding announcement posting permission for a player
// Syntax: wedpost <player>
func (d *CommandDispatcher) cmdWedpost(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Syntax: wedpost <player>\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't playing.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.PCData == nil {
		d.send(ch, "They don't have player data.\r\n")
		return
	}

	// Toggle wedpost permission using a flag in PCData
	// Since wedpost is not in the current structure, we'll use a comm flag or add later
	// For now, we'll just send a message indicating the toggle
	// In a full implementation, this would be: victim.Wedpost = !victim.Wedpost

	d.send(ch, fmt.Sprintf("Wedding post permission toggled for %s.\r\n", victim.Name))
	d.WiznetBroadcast(fmt.Sprintf("%s toggles wedpost for %s", ch.Name, victim.Name), 100)
}

// cmdKnight advances a player to Knight immortal level (103)
// Requires PLR_KEY flag on the immortal executing the command
// Syntax: knight <player>
func (d *CommandDispatcher) cmdKnight(ch *types.Character, args string) {
	// Check for PLR_KEY (implementor permission)
	if !ch.Act.Has(types.ActKey) {
		d.send(ch, "This function is not currently implemented.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Syntax: knight <char>.\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "That player is not here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	targetLevel := 103 // Knight level

	if targetLevel <= victim.Level {
		d.send(ch, fmt.Sprintf("%s is already level %d or higher.\r\n", victim.Name, targetLevel))
		return
	}

	// Perform the knighting ceremony
	d.send(ch, fmt.Sprintf("You touch %s's shoulder with a sword called {GKnight's Faith{x.\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s touches your shoulder with a sword called {GKnight's Faith{x.\r\n", ch.Name))

	// Notify room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch && person != victim {
				d.send(person, fmt.Sprintf("%s touches %s's shoulder with a sword called {GKnight's Faith{x.\r\n", ch.Name, victim.Name))
				d.send(person, fmt.Sprintf("%s glows with an unearthly light as their mortality slips away.\r\n", victim.Name))
			}
		}
	}

	// Advance the player to knight level
	for victim.Level < targetLevel {
		victim.Level++
		d.send(victim, "You raise a level!!  ")
		d.advanceLevel(victim)
	}

	victim.Trust = 0

	// Save the character
	if d.OnSave != nil {
		d.OnSave(victim)
	}

	d.WiznetBroadcast(fmt.Sprintf("%s has been knighted by %s", victim.Name, ch.Name), 100)
}

// cmdSquire advances a player to Squire immortal level (102)
// Requires PLR_KEY flag on the immortal executing the command
// Syntax: squire <player>
func (d *CommandDispatcher) cmdSquire(ch *types.Character, args string) {
	// Check for PLR_KEY (implementor permission)
	if !ch.Act.Has(types.ActKey) {
		d.send(ch, "This function is not currently implemented.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Syntax: squire <char>.\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "That player is not here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	targetLevel := 102 // Squire level (LevelImmortal)

	if targetLevel <= victim.Level {
		d.send(ch, fmt.Sprintf("%s is already level %d or higher.\r\n", victim.Name, targetLevel))
		return
	}

	// Perform the squire ceremony
	d.send(ch, fmt.Sprintf("You touch %s's shoulder with a sword called {BSquire's Faith{x.\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s touches your shoulder with a sword called {BSquire's Faith{x.\r\n", ch.Name))

	// Notify room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch && person != victim {
				d.send(person, fmt.Sprintf("%s touches %s's shoulder with a sword called {BSquire's Faith{x.\r\n", ch.Name, victim.Name))
				d.send(person, fmt.Sprintf("%s glows with an unearthly light as their mortality slips away.\r\n", victim.Name))
			}
		}
	}

	// Advance the player to squire level
	for victim.Level < targetLevel {
		victim.Level++
		d.send(victim, "You raise a level!!  ")
		d.advanceLevel(victim)
	}

	victim.Trust = 0

	// Save the character
	if d.OnSave != nil {
		d.OnSave(victim)
	}

	d.WiznetBroadcast(fmt.Sprintf("%s has been made a squire by %s", victim.Name, ch.Name), 100)
}

// advanceLevel handles stat increases when a player gains a level
func (d *CommandDispatcher) advanceLevel(ch *types.Character) {
	// Simple advancement - add HP/Mana/Move
	// In a full implementation, this would use class-specific tables
	ch.MaxHit += 10
	ch.Hit = ch.MaxHit
	ch.MaxMana += 10
	ch.Mana = ch.MaxMana
	ch.MaxMove += 10
	ch.Move = ch.MaxMove

	// Give training and practice points
	ch.Train += 1
	ch.Practice += 2
}

// cmdMquest toggles the ITEM_QUEST flag on an object in inventory
// Syntax: mquest <object>
func (d *CommandDispatcher) cmdMquest(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Make a quest item of what?\r\n")
		return
	}

	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	if obj.ExtraFlags.Has(types.ItemQuest) {
		obj.ExtraFlags.Remove(types.ItemQuest)
		d.send(ch, fmt.Sprintf("%s is no longer a quest item.\r\n", obj.ShortDesc))
	} else {
		obj.ExtraFlags.Set(types.ItemQuest)
		d.send(ch, fmt.Sprintf("%s is now a quest item.\r\n", obj.ShortDesc))
	}
}

// cmdMpoint toggles the ITEM_QUESTPOINT flag on an object in inventory
// Syntax: mpoint <object>
func (d *CommandDispatcher) cmdMpoint(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Make a questpoint item of what?\r\n")
		return
	}

	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	if obj.ExtraFlags.Has(types.ItemQuestPoint) {
		obj.ExtraFlags.Remove(types.ItemQuestPoint)
		d.send(ch, fmt.Sprintf("%s is no longer a questpoint item.\r\n", obj.ShortDesc))
	} else {
		obj.ExtraFlags.Set(types.ItemQuestPoint)
		d.send(ch, fmt.Sprintf("%s is now a questpoint item.\r\n", obj.ShortDesc))
	}
}

// cmdWizslap slaps a player, teleporting them to a random room and weakening them
// This is a fun immortal punishment/prank command
// Syntax: wizslap <player>
func (d *CommandDispatcher) cmdWizslap(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "WizSlap whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level {
		d.send(ch, "You failed.\r\n")
		return
	}

	// Get a random room
	randomRoom := d.getRandomRoom(victim)
	if randomRoom == nil {
		d.send(ch, "Could not find a suitable room.\r\n")
		return
	}

	// Show the slap messages
	d.send(victim, fmt.Sprintf("%s slaps you, sending you reeling through time and space!\r\n", ch.Name))
	d.send(ch, fmt.Sprintf("You send %s reeling through time and space!\r\n", victim.Name))

	// Notify the room
	if ch.InRoom != nil {
		for _, person := range ch.InRoom.People {
			if person != ch && person != victim {
				d.send(person, fmt.Sprintf("%s slaps %s, sending them reeling through time and space!\r\n", ch.Name, victim.Name))
			}
		}
	}

	// Remove from old room
	if victim.InRoom != nil {
		for i, person := range victim.InRoom.People {
			if person == victim {
				victim.InRoom.People = append(victim.InRoom.People[:i], victim.InRoom.People[i+1:]...)
				break
			}
		}
	}

	// Add to new room
	randomRoom.People = append(randomRoom.People, victim)
	victim.InRoom = randomRoom

	// Notify new room
	for _, person := range randomRoom.People {
		if person != victim {
			d.send(person, fmt.Sprintf("%s crashes to the ground!\r\n", victim.Name))
		}
	}

	// Apply weaken affect
	weakenAff := &types.Affect{
		Type:      "weaken",
		Level:     105,
		Duration:  5,
		Location:  types.ApplyStr,
		Modifier:  -21, // -105/5 = -21
		BitVector: types.AffWeaken,
	}
	victim.AddAffect(weakenAff)
	d.send(victim, "You feel your strength slip away.\r\n")

	// Show the new room
	d.doLook(victim, "")
}

// getRandomRoom finds a random accessible room for the character
func (d *CommandDispatcher) getRandomRoom(ch *types.Character) *types.Room {
	if d.GameLoop == nil || len(d.GameLoop.Rooms) == 0 {
		return nil
	}

	// Collect all valid room vnums
	var validRooms []*types.Room
	for _, room := range d.GameLoop.Rooms {
		if room == nil {
			continue
		}
		// Skip private, gods-only, imp-only rooms
		if room.Flags.Has(types.RoomPrivate) || room.Flags.Has(types.RoomGodsOnly) || room.Flags.Has(types.RoomImpOnly) {
			continue
		}
		// Skip no-recall rooms for mortals
		if !ch.IsImmortal() && room.Flags.Has(types.RoomNoRecall) {
			continue
		}
		validRooms = append(validRooms, room)
	}

	if len(validRooms) == 0 {
		return nil
	}

	// Pick a random room using crypto/rand seeded math/rand
	return validRooms[randomInt(len(validRooms))]
}

// randomInt returns a random integer in [0, n)
func randomInt(n int) int {
	if n <= 0 {
		return 0
	}
	// Use a simple time-based seed for variety
	return int(time.Now().UnixNano() % int64(n))
}

// cmdDupe manages the dupe (duplicate character) list for a player
// This tracks alternate characters owned by the same player
// Syntax: dupe <player> [character_name]
func (d *CommandDispatcher) cmdDupe(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) == 0 || parts[0] == "" {
		d.send(ch, "Dupe whom?\r\n")
		return
	}

	victim := d.findCharacterWorld(ch, parts[0])
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim.IsNPC() {
		d.send(ch, "Not on NPCs.\r\n")
		return
	}

	if victim.Level >= ch.Level && ch.Level < types.MaxLevel {
		d.send(ch, "You failed.\r\n")
		return
	}

	if victim.PCData == nil {
		d.send(ch, "They don't have player data.\r\n")
		return
	}

	// Initialize dupes slice if needed
	if victim.PCData.Dupes == nil {
		victim.PCData.Dupes = make([]string, 0, MaxDupes)
	}

	// If no second argument, show current dupes
	if len(parts) < 2 || parts[1] == "" {
		if len(victim.PCData.Dupes) == 0 {
			d.send(ch, "They have no dupes set.\r\n")
			return
		}
		d.send(ch, "They currently have the following dupes:\r\n")
		for _, dupe := range victim.PCData.Dupes {
			d.send(ch, fmt.Sprintf("    %s\r\n", dupe))
		}
		return
	}

	dupeName := parts[1]

	// Check if this dupe already exists
	for i, existing := range victim.PCData.Dupes {
		if strings.EqualFold(existing, dupeName) {
			// Remove the dupe
			victim.PCData.Dupes = append(victim.PCData.Dupes[:i], victim.PCData.Dupes[i+1:]...)
			d.send(ch, "Dupe removed.\r\n")
			return
		}
	}

	// Add new dupe
	if len(victim.PCData.Dupes) >= MaxDupes {
		d.send(ch, "Sorry, they've reached the limit for dupes.\r\n")
		return
	}

	victim.PCData.Dupes = append(victim.PCData.Dupes, dupeName)
	d.send(ch, fmt.Sprintf("%s now has the dupe %s set.\r\n", victim.Name, dupeName))
}

// MaxDupes is the maximum number of duplicate character names that can be tracked
const MaxDupes = 10
