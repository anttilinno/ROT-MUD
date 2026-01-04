package game

import (
	"fmt"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Utility commands: recall, scan, visible, where (player version)

// Default recall room vnum
const RecallVnum = 3001

func (d *CommandDispatcher) cmdRecall(ch *types.Character, args string) {
	// If fighting, need to skill check first
	if ch.Fighting != nil {
		// Get recall skill level
		skill := 50 // Default skill
		if ch.PCData != nil && ch.PCData.Learned != nil {
			if s, ok := ch.PCData.Learned["recall"]; ok && s > skill {
				skill = s
			}
		}

		// 80% * skill/100 chance to fail when fighting
		if combat.NumberPercent() < 80*skill/100 {
			d.send(ch, "You failed!\r\n")
			ch.Wait = 4
			if d.Skills != nil {
				d.Skills.CheckImprove(ch, "recall", false, 6)
			}
			return
		}

		// Success - stop fighting but lose experience
		expLoss := 25
		if ch.Exp > expLoss {
			ch.Exp -= expLoss
			d.send(ch, fmt.Sprintf("You recall from combat! You lose %d experience points.\r\n", expLoss))
		} else {
			d.send(ch, "You recall from combat!\r\n")
		}
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "recall", true, 4)
		}
		combat.StopFighting(ch, true)
	}

	// Find recall room
	var recallVnum int
	if ch.PCData != nil && ch.PCData.Recall != 0 {
		recallVnum = ch.PCData.Recall
	} else {
		recallVnum = RecallVnum
	}

	var recallRoom *types.Room
	if d.GameLoop != nil && d.GameLoop.Rooms != nil {
		recallRoom = d.GameLoop.Rooms[recallVnum]
	}

	if recallRoom == nil {
		d.send(ch, "You can't seem to recall right now.\r\n")
		return
	}

	if ch.InRoom == recallRoom {
		d.send(ch, "You are already there!\r\n")
		return
	}

	// Cost: some movement points
	moveCost := ch.Level / 2
	if moveCost < 10 {
		moveCost = 10
	}

	if ch.Move < moveCost {
		d.send(ch, "You are too exhausted to recall.\r\n")
		return
	}

	ch.Move -= moveCost

	// Leave message
	ActToRoom("$n disappears in a flash of light.", ch, nil, nil, d.Output)

	// Move character
	oldRoom := ch.InRoom
	if oldRoom != nil {
		oldRoom.RemovePerson(ch)
	}
	recallRoom.AddPerson(ch)
	ch.InRoom = recallRoom

	d.send(ch, "You recall to safety.\r\n")
	ActToRoom("$n appears in a flash of light.", ch, nil, nil, d.Output)

	// Show the new room
	d.cmdLook(ch, "")
}

func (d *CommandDispatcher) cmdScan(ch *types.Character, args string) {
	if ch.InRoom == nil {
		d.send(ch, "You can't scan here.\r\n")
		return
	}

	// Check for blind
	if ch.IsAffected(types.AffBlind) {
		d.send(ch, "You can't see anything!\r\n")
		return
	}

	d.send(ch, "Looking around you see:\r\n")

	// Scan current room
	d.scanRoom(ch, ch.InRoom, "here")

	// Scan adjacent rooms
	dirNames := []string{"north", "east", "south", "west", "up", "down"}
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := ch.InRoom.GetExit(dir)
		if exit == nil || exit.ToRoom == nil {
			continue
		}

		// Can't see through closed doors
		if exit.IsClosed() {
			continue
		}

		d.scanRoom(ch, exit.ToRoom, dirNames[dir])

		// Scan two rooms away
		exit2 := exit.ToRoom.GetExit(dir)
		if exit2 != nil && exit2.ToRoom != nil && !exit2.IsClosed() {
			d.scanRoom(ch, exit2.ToRoom, fmt.Sprintf("far %s", dirNames[dir]))
		}
	}
}

func (d *CommandDispatcher) scanRoom(ch *types.Character, room *types.Room, distance string) {
	if room == nil || len(room.People) == 0 {
		return
	}

	for _, person := range room.People {
		if person == ch {
			continue
		}

		// Check visibility
		if !combat.CanSee(ch, person) {
			continue
		}

		name := person.Name
		if person.IsNPC() && person.ShortDesc != "" {
			name = person.ShortDesc
		}

		d.send(ch, fmt.Sprintf("  %-12s - %s\r\n", distance, name))
	}
}

func (d *CommandDispatcher) cmdVisible(ch *types.Character, args string) {
	removed := false

	// Remove invisibility and hidden
	if ch.IsAffected(types.AffInvisible) {
		ch.AffectedBy.Remove(types.AffInvisible)
		d.send(ch, "You fade into existence.\r\n")
		ActToRoom("$n fades into existence.", ch, nil, nil, d.Output)
		removed = true
	}

	if ch.IsAffected(types.AffHide) {
		ch.AffectedBy.Remove(types.AffHide)
		d.send(ch, "You step out of the shadows.\r\n")
		ActToRoom("$n steps out of the shadows.", ch, nil, nil, d.Output)
		removed = true
	}

	if !removed {
		d.send(ch, "You are already visible.\r\n")
	}
}
