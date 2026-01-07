package game

import (
	"fmt"

	"rotmud/pkg/types"
)

// Position commands: sit, stand, rest, sleep, wake

func (d *CommandDispatcher) cmdSit(ch *types.Character, args string) {
	switch ch.Position {
	case types.PosSleeping:
		d.send(ch, "You wake and sit up.\r\n")
		ActToRoom("$n wakes and sits up.", ch, nil, nil, d.Output)
		ch.Position = types.PosSitting

	case types.PosResting:
		d.send(ch, "You stop resting and sit up.\r\n")
		ActToRoom("$n stops resting and sits up.", ch, nil, nil, d.Output)
		ch.Position = types.PosSitting

	case types.PosSitting:
		d.send(ch, "You are already sitting.\r\n")

	case types.PosFighting:
		d.send(ch, "Maybe you should finish this fight first?\r\n")

	case types.PosStanding:
		d.send(ch, "You sit down.\r\n")
		ActToRoom("$n sits down.", ch, nil, nil, d.Output)
		ch.Position = types.PosSitting

	default:
		d.send(ch, "You can't do that right now.\r\n")
	}
}

func (d *CommandDispatcher) cmdStand(ch *types.Character, args string) {
	switch ch.Position {
	case types.PosSleeping:
		d.send(ch, "You wake and stand up.\r\n")
		ActToRoom("$n wakes and stands up.", ch, nil, nil, d.Output)
		ch.Position = types.PosStanding

	case types.PosResting, types.PosSitting:
		d.send(ch, "You stand up.\r\n")
		ActToRoom("$n stands up.", ch, nil, nil, d.Output)
		ch.Position = types.PosStanding

	case types.PosFighting:
		d.send(ch, "You are already fighting!\r\n")

	case types.PosStanding:
		d.send(ch, "You are already standing.\r\n")

	default:
		d.send(ch, "You can't do that right now.\r\n")
	}
}

func (d *CommandDispatcher) cmdRest(ch *types.Character, args string) {
	switch ch.Position {
	case types.PosSleeping:
		d.send(ch, "You wake up and start resting.\r\n")
		ActToRoom("$n wakes up and starts resting.", ch, nil, nil, d.Output)
		ch.Position = types.PosResting

	case types.PosResting:
		d.send(ch, "You are already resting.\r\n")

	case types.PosSitting:
		d.send(ch, "You rest your tired bones.\r\n")
		ActToRoom("$n rests.", ch, nil, nil, d.Output)
		ch.Position = types.PosResting

	case types.PosFighting:
		d.send(ch, "You are already fighting!\r\n")

	case types.PosStanding:
		d.send(ch, "You sit down and rest your tired bones.\r\n")
		ActToRoom("$n sits down and rests.", ch, nil, nil, d.Output)
		ch.Position = types.PosResting

	default:
		d.send(ch, "You can't do that right now.\r\n")
	}
}

func (d *CommandDispatcher) cmdSleep(ch *types.Character, args string) {
	switch ch.Position {
	case types.PosSleeping:
		d.send(ch, "You are already sleeping.\r\n")

	case types.PosResting, types.PosSitting, types.PosStanding:
		d.send(ch, "You go to sleep.\r\n")
		ActToRoom("$n goes to sleep.", ch, nil, nil, d.Output)
		ch.Position = types.PosSleeping

	case types.PosFighting:
		d.send(ch, "You are already fighting!\r\n")

	default:
		d.send(ch, "You can't do that right now.\r\n")
	}
}

func (d *CommandDispatcher) cmdWake(ch *types.Character, args string) {
	if args != "" {
		// Wake someone else
		victim := FindCharInRoom(ch, args)
		if victim == nil {
			d.send(ch, "They aren't here.\r\n")
			return
		}

		if victim == ch {
			d.send(ch, "You wake yourself up.\r\n")
			ch.Position = types.PosStanding
			return
		}

		if victim.Position != types.PosSleeping {
			d.send(ch, "They aren't sleeping.\r\n")
			return
		}

		// Check if they're under magical sleep
		if victim.IsAffected(types.AffSleep) {
			d.send(ch, "You can't wake them.\r\n")
			return
		}

		victim.Position = types.PosSitting
		d.send(ch, fmt.Sprintf("You wake %s.\r\n", victim.Name))
		d.send(victim, fmt.Sprintf("%s wakes you.\r\n", ch.Name))
		return
	}

	// Wake self
	switch ch.Position {
	case types.PosSleeping:
		if ch.IsAffected(types.AffSleep) {
			d.send(ch, "You can't wake up!\r\n")
			return
		}
		d.send(ch, "You wake and sit up.\r\n")
		ActToRoom("$n wakes and sits up.", ch, nil, nil, d.Output)
		ch.Position = types.PosSitting

	case types.PosResting:
		d.send(ch, "You are already awake.\r\n")

	default:
		d.send(ch, "You are already awake.\r\n")
	}
}
