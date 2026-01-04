package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/types"
)

// Group commands: follow, group, gtell, split, nofollow

func (d *CommandDispatcher) cmdFollow(ch *types.Character, args string) {
	if args == "" {
		if ch.Master != nil {
			d.send(ch, fmt.Sprintf("You are following %s.\r\n", ch.Master.Name))
		} else {
			d.send(ch, "You are not following anyone.\r\n")
		}
		return
	}

	// "follow self" to stop following
	if strings.EqualFold(args, "self") || strings.EqualFold(args, ch.Name) {
		if ch.Master == nil {
			d.send(ch, "You aren't following anyone.\r\n")
			return
		}
		d.stopFollowing(ch)
		return
	}

	// Find target to follow
	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		if ch.Master == nil {
			d.send(ch, "You already follow yourself.\r\n")
		} else {
			d.stopFollowing(ch)
		}
		return
	}

	// Can't follow someone who is following you (prevents loops)
	if victim.Master == ch {
		d.send(ch, "Following would cause a loop.\r\n")
		return
	}

	// Check if victim accepts followers
	if victim.PlayerAct.Has(types.PlrNoFollow) {
		d.send(ch, fmt.Sprintf("%s doesn't accept followers.\r\n", victim.Name))
		return
	}

	// Stop following current master if any
	if ch.Master != nil {
		d.stopFollowing(ch)
	}

	// Start following
	ch.Master = victim
	ch.Leader = nil

	d.send(ch, fmt.Sprintf("You now follow %s.\r\n", victim.Name))
	d.send(victim, fmt.Sprintf("%s now follows you.\r\n", ch.Name))
	ActToNotVict("$n starts following $N.", ch, victim, nil, d.Output)
}

func (d *CommandDispatcher) stopFollowing(ch *types.Character) {
	if ch.Master == nil {
		return
	}

	master := ch.Master
	ch.Master = nil
	ch.Leader = nil

	d.send(ch, fmt.Sprintf("You stop following %s.\r\n", master.Name))
	d.send(master, fmt.Sprintf("%s stops following you.\r\n", ch.Name))
}

func (d *CommandDispatcher) cmdGroup(ch *types.Character, args string) {
	if args == "" {
		// Show current group
		leader := d.getGroupLeader(ch)
		if leader == nil {
			d.send(ch, "You are not in a group.\r\n")
			return
		}

		d.send(ch, fmt.Sprintf("%s's group:\r\n", leader.Name))

		// Show all group members
		members := d.getGroupMembers(leader)
		for _, member := range members {
			hpPct := 100
			if member.MaxHit > 0 {
				hpPct = member.Hit * 100 / member.MaxHit
			}
			manaPct := 100
			if member.MaxMana > 0 {
				manaPct = member.Mana * 100 / member.MaxMana
			}
			movePct := 100
			if member.MaxMove > 0 {
				movePct = member.Move * 100 / member.MaxMove
			}

			leaderMark := ""
			if member == leader {
				leaderMark = " (Leader)"
			}

			d.send(ch, fmt.Sprintf("[%3d %s] %-15s %3d%% hp %3d%% mana %3d%% mv%s\r\n",
				member.Level,
				d.getClassName(member),
				member.Name,
				hpPct, manaPct, movePct,
				leaderMark))
		}
		return
	}

	// Group a specific person
	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		// Toggle grouping of self (become leader or leave group)
		if ch.Leader == ch {
			d.send(ch, "You disband the group.\r\n")
			d.disbandGroup(ch)
		} else if ch.Leader != nil {
			d.send(ch, "You leave the group.\r\n")
			ch.Leader = nil
		} else {
			ch.Leader = ch
			d.send(ch, "You form a new group.\r\n")
		}
		return
	}

	// Must be a leader to group others
	if ch.Leader != ch {
		if ch.Leader == nil {
			ch.Leader = ch // Become a leader
		} else {
			d.send(ch, "You are not the group leader.\r\n")
			return
		}
	}

	// Target must be following the leader
	if victim.Master != ch {
		d.send(ch, fmt.Sprintf("%s is not following you.\r\n", victim.Name))
		return
	}

	// Check level difference (within 8 levels typically)
	levelDiff := ch.Level - victim.Level
	if levelDiff < 0 {
		levelDiff = -levelDiff
	}
	if levelDiff > 8 {
		d.send(ch, "You can't group with someone more than 8 levels different.\r\n")
		return
	}

	// Toggle grouping
	if victim.Leader == ch {
		// Remove from group
		victim.Leader = nil
		d.send(ch, fmt.Sprintf("%s is no longer in your group.\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s removes you from the group.\r\n", ch.Name))
	} else {
		// Add to group
		victim.Leader = ch
		d.send(ch, fmt.Sprintf("%s joins your group.\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("You join %s's group.\r\n", ch.Name))
	}
}

func (d *CommandDispatcher) getGroupLeader(ch *types.Character) *types.Character {
	if ch.Leader != nil {
		return ch.Leader
	}
	return nil
}

func (d *CommandDispatcher) getGroupMembers(leader *types.Character) []*types.Character {
	members := []*types.Character{leader}

	if leader.InRoom == nil {
		return members
	}

	// Find all characters in the same room who are in this group
	// In a real implementation, we'd iterate all characters in the world
	for _, person := range leader.InRoom.People {
		if person != leader && person.Leader == leader {
			members = append(members, person)
		}
	}

	return members
}

func (d *CommandDispatcher) disbandGroup(leader *types.Character) {
	// Clear leader status for all group members
	if leader.InRoom != nil {
		for _, person := range leader.InRoom.People {
			if person.Leader == leader && person != leader {
				person.Leader = nil
				d.send(person, "The group has been disbanded.\r\n")
			}
		}
	}
	leader.Leader = nil
}

func (d *CommandDispatcher) getClassName(ch *types.Character) string {
	classes := []string{"Mag", "Cle", "Thi", "War"}
	if ch.Class >= 0 && ch.Class < len(classes) {
		return classes[ch.Class]
	}
	return "???"
}

func (d *CommandDispatcher) cmdGtell(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Tell your group what?\r\n")
		return
	}

	leader := d.getGroupLeader(ch)
	if leader == nil {
		d.send(ch, "You are not in a group.\r\n")
		return
	}

	// Send to all group members
	d.send(ch, fmt.Sprintf("You tell the group '%s'\r\n", args))

	if d.GameLoop != nil {
		for _, person := range d.GameLoop.Characters {
			if person != ch && person.Leader == leader {
				d.send(person, fmt.Sprintf("%s tells the group '%s'\r\n", ch.Name, args))
			}
		}
	}
}

func (d *CommandDispatcher) cmdSplit(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Split how much?\r\n")
		return
	}

	// Parse amount
	var amount int
	_, err := fmt.Sscanf(args, "%d", &amount)
	if err != nil || amount <= 0 {
		d.send(ch, "You must split a positive amount of gold.\r\n")
		return
	}

	if amount > ch.Gold {
		d.send(ch, "You don't have that much gold.\r\n")
		return
	}

	leader := d.getGroupLeader(ch)
	if leader == nil {
		d.send(ch, "You are not in a group.\r\n")
		return
	}

	// Count group members in the same room
	members := d.getGroupMembers(leader)
	memberCount := 0
	for _, m := range members {
		if m.InRoom == ch.InRoom {
			memberCount++
		}
	}

	if memberCount <= 1 {
		d.send(ch, "There's no one to split with.\r\n")
		return
	}

	// Calculate share
	share := amount / memberCount
	extra := amount - (share * memberCount)

	if share == 0 {
		d.send(ch, "That's not enough gold to split.\r\n")
		return
	}

	// Deduct from splitter
	ch.Gold -= amount

	// Give to all members in the room
	for _, member := range members {
		if member.InRoom != ch.InRoom {
			continue
		}
		if member == ch {
			// Splitter keeps their share plus any remainder
			member.Gold += share + extra
			d.send(member, fmt.Sprintf("You split %d gold. Your share is %d gold.\r\n", amount, share+extra))
		} else {
			member.Gold += share
			d.send(member, fmt.Sprintf("%s splits %d gold. Your share is %d gold.\r\n", ch.Name, amount, share))
		}
	}
}

func (d *CommandDispatcher) cmdNofollow(ch *types.Character, args string) {
	if ch.PCData == nil {
		d.send(ch, "Only players can use nofollow.\r\n")
		return
	}

	ch.PlayerAct.Toggle(types.PlrNoFollow)

	if ch.PlayerAct.Has(types.PlrNoFollow) {
		d.send(ch, "You no longer accept followers.\r\n")
		// Stop anyone following you
		for _, other := range d.GameLoop.GetCharacters() {
			if other.Master == ch {
				d.stopFollowing(other)
				d.send(other, fmt.Sprintf("%s no longer accepts followers.\r\n", ch.Name))
			}
		}
	} else {
		d.send(ch, "You now accept followers.\r\n")
	}
}

// cmdOrder orders charmed followers to perform commands
// Syntax: order <target|all> <command>
func (d *CommandDispatcher) cmdOrder(ch *types.Character, args string) {
	// Can't order if you're charmed
	if ch.IsAffected(types.AffCharm) {
		d.send(ch, "You feel like taking, not giving, orders.\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		d.send(ch, "Order whom to do what?\r\n")
		return
	}

	targetArg := strings.ToLower(parts[0])
	command := parts[1]

	// Block dangerous commands
	cmdWord := strings.ToLower(strings.Fields(command)[0])
	dangerousCmds := []string{"delete", "mob", "reroll", "quit", "password", "order"}
	for _, dangerous := range dangerousCmds {
		if cmdWord == dangerous {
			d.send(ch, "That will NOT be done.\r\n")
			return
		}
	}

	if ch.InRoom == nil {
		d.send(ch, "You have no followers here.\r\n")
		return
	}

	var victim *types.Character
	orderAll := false

	if targetArg == "all" {
		orderAll = true
	} else {
		// Find specific target
		victim = FindCharInRoom(ch, targetArg)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}

		if victim == ch {
			d.send(ch, "Aye aye, right away!\r\n")
			return
		}

		// Check if victim is charmed by this character
		if !victim.IsAffected(types.AffCharm) || victim.Master != ch {
			d.send(ch, "Do it yourself!\r\n")
			return
		}

		// Can't order immortals
		if !victim.IsNPC() && victim.Level >= 100 {
			d.send(ch, "Do it yourself!\r\n")
			return
		}
	}

	// Execute orders on followers
	found := false
	for _, follower := range ch.InRoom.People {
		// Check if this person should be ordered
		if !follower.IsAffected(types.AffCharm) {
			continue
		}
		if follower.Master != ch {
			continue
		}
		if !orderAll && follower != victim {
			continue
		}

		found = true

		// Notify the follower
		name := ch.Name
		if ch.IsNPC() && ch.ShortDesc != "" {
			name = ch.ShortDesc
		}
		d.send(follower, fmt.Sprintf("%s orders you to '%s'.\r\n", name, command))

		// Execute the command as the follower
		d.Dispatch(Command{
			Character: follower,
			Input:     command,
		})
	}

	if found {
		// Add lag to the orderer
		ch.Wait += 12 // PULSE_VIOLENCE equivalent
		d.send(ch, "Ok.\r\n")
	} else {
		d.send(ch, "You have no followers here.\r\n")
	}
}

// cmdDismiss dismisses the character's pet or a charmed follower
// This removes the pet from the game entirely
// Syntax: dismiss [target]
func (d *CommandDispatcher) cmdDismiss(ch *types.Character, args string) {
	if args == "" {
		// Dismiss the player's primary pet
		if ch.Pet == nil {
			d.send(ch, "You have no pet to dismiss.\r\n")
			return
		}

		pet := ch.Pet
		name := pet.ShortDesc
		if name == "" {
			name = pet.Name
		}

		d.send(ch, fmt.Sprintf("You dismiss %s.\r\n", name))

		// Notify room
		if ch.InRoom != nil {
			for _, other := range ch.InRoom.People {
				if other != ch && other != pet {
					d.send(other, fmt.Sprintf("%s dismisses %s.\r\n", ch.Name, name))
				}
			}
		}

		// Remove pet from game
		DismissPet(ch)

		// Remove from game loop if OnRemoveCharacter is set
		if d.OnRemoveCharacter != nil {
			d.OnRemoveCharacter(pet)
		}
		return
	}

	// Dismiss a specific charmed follower
	if ch.InRoom == nil {
		d.send(ch, "You have no followers here.\r\n")
		return
	}

	target := FindCharInRoom(ch, args)
	if target == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if target == ch {
		d.send(ch, "You can't dismiss yourself.\r\n")
		return
	}

	// Check if target is charmed by this character
	if !target.IsAffected(types.AffCharm) || target.Master != ch {
		d.send(ch, "You don't control them.\r\n")
		return
	}

	name := target.ShortDesc
	if name == "" {
		name = target.Name
	}

	d.send(ch, fmt.Sprintf("You dismiss %s.\r\n", name))

	// Notify room
	for _, other := range ch.InRoom.People {
		if other != ch && other != target {
			d.send(other, fmt.Sprintf("%s dismisses %s.\r\n", ch.Name, name))
		}
	}

	// If this is the player's pet, use DismissPet
	if ch.Pet == target {
		DismissPet(ch)
	} else {
		// Just a charmed follower, remove from game
		StopFollower(target)
		if target.InRoom != nil {
			target.InRoom.RemovePerson(target)
			target.InRoom = nil
		}
	}

	// Remove from game loop
	if d.OnRemoveCharacter != nil {
		d.OnRemoveCharacter(target)
	}
}
