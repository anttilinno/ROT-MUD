package game

import (
	"rotmud/pkg/types"
)

// Position commands: sit, stand, rest, sleep, wake

// countUsers returns how many people are on a piece of furniture (ROT count_users).
func countUsers(obj *types.Object) int {
	if obj.InRoom == nil {
		return 0
	}
	n := 0
	for _, p := range obj.InRoom.People {
		if p.On == obj {
			n++
		}
	}
	return n
}

// furnitureVerb picks the at/on/in wording for furniture flags v2, falling
// back to "in" like ROT's do_sit/do_rest/... else branches.
func furnitureVerb(obj *types.Object, at, on int) string {
	switch {
	case obj.Values[2]&at != 0:
		return "at"
	case obj.Values[2]&on != 0:
		return "on"
	}
	return "in"
}

// findFurniture resolves the furniture argument of sit/rest/sleep/stand:
// an object in the room, or the one ch is already on. It returns ok=false
// after telling ch why when the command must stop. Based on ROT act_move.c.
func (d *CommandDispatcher) findFurniture(ch *types.Character, args string, mask int, cantMsg, fullMsg string) (*types.Object, bool) {
	obj := ch.On
	if args != "" {
		if obj = FindObjInRoom(ch, args); obj == nil {
			d.send(ch, "You don't see that here.\r\n")
			return nil, false
		}
	}
	if obj == nil {
		return nil, true
	}
	if obj.ItemType != types.ItemTypeFurniture || obj.Values[2]&mask == 0 {
		d.send(ch, cantMsg)
		return nil, false
	}
	if ch.On != obj && countUsers(obj) >= obj.Values[0] {
		ActToChar(fullMsg, ch, nil, obj, d.Output)
		return nil, false
	}
	ch.On = obj
	return obj, true
}

func (d *CommandDispatcher) cmdSit(ch *types.Character, args string) {
	if ch.Position == types.PosFighting {
		d.send(ch, "Maybe you should finish this fight first?\r\n")
		return
	}
	obj, ok := d.findFurniture(ch, args, types.SitAt|types.SitOn|types.SitIn,
		"You can't sit on that.\r\n", "There's no more room on $p.")
	if !ok {
		return
	}
	verb := ""
	if obj != nil {
		verb = furnitureVerb(obj, types.SitAt, types.SitOn)
	}

	switch ch.Position {
	case types.PosSleeping:
		if obj == nil {
			d.send(ch, "You wake and sit up.\r\n")
			ActToRoom("$n wakes and sits up.", ch, nil, nil, d.Output)
		} else {
			ActToChar("You wake and sit "+verb+" $p.", ch, nil, obj, d.Output)
			roomVerb := verb
			if verb == "on" {
				roomVerb = "at" // sic, ROT do_sit
			}
			ActToRoom("$n wakes and sits "+roomVerb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosSitting

	case types.PosResting:
		if obj == nil {
			d.send(ch, "You stop resting.\r\n")
		} else if verb != "in" { // ROT has no message for sitting in
			ActToChar("You sit "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n sits "+verb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosSitting

	case types.PosSitting:
		d.send(ch, "You are already sitting down.\r\n")

	case types.PosStanding:
		switch verb {
		case "":
			d.send(ch, "You sit down.\r\n")
			ActToRoom("$n sits down on the ground.", ch, nil, nil, d.Output)
		case "on":
			ActToChar("You sit on $p.", ch, nil, obj, d.Output)
			ActToRoom("$n sits on $p.", ch, nil, obj, d.Output)
		default:
			ActToChar("You sit down "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n sits down "+verb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosSitting
	}
}

func (d *CommandDispatcher) cmdStand(ch *types.Character, args string) {
	var obj *types.Object
	if args != "" {
		if ch.Position == types.PosFighting {
			d.send(ch, "Maybe you should finish fighting first?\r\n")
			return
		}
		var ok bool
		if obj, ok = d.findFurniture(ch, args, types.StandAt|types.StandOn|types.StandIn,
			"You can't seem to find a place to stand.\r\n", "There's no room to stand on $p."); !ok {
			return
		}
	}
	verb := ""
	if obj != nil {
		verb = furnitureVerb(obj, types.StandAt, types.StandOn)
	}

	switch ch.Position {
	case types.PosSleeping:
		if ch.IsAffected(types.AffSleep) {
			d.send(ch, "You can't wake up!\r\n")
			return
		}
		if obj == nil {
			d.send(ch, "You wake and stand up.\r\n")
			ActToRoom("$n wakes and stands up.", ch, nil, nil, d.Output)
			ch.On = nil
		} else {
			ActToChar("You wake and stand "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n wakes and stands "+verb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosStanding

	case types.PosResting, types.PosSitting:
		if obj == nil {
			d.send(ch, "You stand up.\r\n")
			ActToRoom("$n stands up.", ch, nil, nil, d.Output)
			ch.On = nil
		} else {
			roomVerb := verb
			if verb == "in" {
				roomVerb = "on" // sic, ROT do_stand
			}
			ActToChar("You stand "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n stands "+roomVerb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosStanding

	case types.PosStanding:
		d.send(ch, "You are already standing.\r\n")

	case types.PosFighting:
		d.send(ch, "You are already fighting!\r\n")
	}
}

func (d *CommandDispatcher) cmdRest(ch *types.Character, args string) {
	if ch.Position == types.PosFighting {
		d.send(ch, "You are already fighting!\r\n")
		return
	}
	obj, ok := d.findFurniture(ch, args, types.RestAt|types.RestOn|types.RestIn,
		"You can't rest on that.\r\n", "There's no more room on $p.")
	if !ok {
		return
	}
	verb := ""
	if obj != nil {
		verb = furnitureVerb(obj, types.RestAt, types.RestOn)
	}

	switch ch.Position {
	case types.PosSleeping:
		if obj == nil {
			d.send(ch, "You wake up and start resting.\r\n")
			ActToRoom("$n wakes up and starts resting.", ch, nil, nil, d.Output)
		} else {
			ActToChar("You wake up and rest "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n wakes up and rests "+verb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosResting

	case types.PosResting:
		d.send(ch, "You are already resting.\r\n")

	case types.PosStanding:
		switch verb {
		case "":
			d.send(ch, "You rest.\r\n")
			ActToRoom("$n sits down and rests.", ch, nil, nil, d.Output)
		case "at":
			ActToChar("You sit down at $p and rest.", ch, nil, obj, d.Output)
			ActToRoom("$n sits down at $p and rests.", ch, nil, obj, d.Output)
		case "on":
			ActToChar("You sit on $p and rest.", ch, nil, obj, d.Output)
			ActToRoom("$n sits on $p and rests.", ch, nil, obj, d.Output)
		default:
			ActToChar("You rest in $p.", ch, nil, obj, d.Output)
			ActToRoom("$n rests in $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosResting

	case types.PosSitting:
		if obj == nil {
			d.send(ch, "You rest.\r\n")
			ActToRoom("$n rests.", ch, nil, nil, d.Output)
		} else {
			ActToChar("You rest "+verb+" $p.", ch, nil, obj, d.Output)
			ActToRoom("$n rests "+verb+" $p.", ch, nil, obj, d.Output)
		}
		ch.Position = types.PosResting
	}
}

func (d *CommandDispatcher) cmdSleep(ch *types.Character, args string) {
	switch ch.Position {
	case types.PosSleeping:
		d.send(ch, "You are already sleeping.\r\n")

	case types.PosResting, types.PosSitting, types.PosStanding:
		if args == "" && ch.On == nil {
			d.send(ch, "You go to sleep.\r\n")
			ActToRoom("$n goes to sleep.", ch, nil, nil, d.Output)
			ch.Position = types.PosSleeping
			return
		}
		obj, ok := d.findFurniture(ch, args, types.SleepAt|types.SleepOn|types.SleepIn,
			"You can't sleep on that!\r\n", "There is no room on $p for you.")
		if !ok {
			return
		}
		verb := furnitureVerb(obj, types.SleepAt, types.SleepOn)
		ActToChar("You go to sleep "+verb+" $p.", ch, nil, obj, d.Output)
		ActToRoom("$n goes to sleep "+verb+" $p.", ch, nil, obj, d.Output)
		ch.Position = types.PosSleeping

	case types.PosFighting:
		d.send(ch, "You are already fighting!\r\n")
	}
}

// cmdWake wakes someone else, or stands ch up (ROT do_wake).
func (d *CommandDispatcher) cmdWake(ch *types.Character, args string) {
	if args == "" {
		d.cmdStand(ch, "")
		return
	}
	if ch.Position <= types.PosSleeping {
		d.send(ch, "You are asleep yourself!\r\n")
		return
	}
	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}
	if victim.Position > types.PosSleeping {
		ActToChar("$N is already awake.", ch, victim, nil, d.Output)
		return
	}
	if victim.IsAffected(types.AffSleep) {
		ActToChar("You can't wake $M!", ch, victim, nil, d.Output)
		return
	}
	ActToVict("$n wakes you.", ch, victim, nil, d.Output)
	d.cmdStand(victim, "")
}
