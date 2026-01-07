package game

import (
	"fmt"
	"strconv"
	"strings"

	"rotmud/pkg/types"
)

// Bank commands: deposit, withdraw, balance

// cmdDeposit deposits gold into the bank
func (d *CommandDispatcher) cmdDeposit(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}

	// Check if there's a banker in the room
	banker := d.findBanker(ch)
	if banker == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Deposit how much gold?\r\n")
		return
	}

	var amount int
	if strings.ToLower(args) == "all" {
		amount = ch.Gold
	} else {
		var err error
		amount, err = strconv.Atoi(args)
		if err != nil || amount <= 0 {
			d.send(ch, "That's not a valid amount.\r\n")
			return
		}
	}

	if amount > ch.Gold {
		d.send(ch, fmt.Sprintf("You only have %d gold coins.\r\n", ch.Gold))
		return
	}

	ch.Gold -= amount
	ch.PCData.BankGold += amount

	d.send(ch, fmt.Sprintf("You deposit %d gold coins. Your bank balance is now %d gold.\r\n",
		amount, ch.PCData.BankGold))
	ActToRoom("$n makes a bank deposit.", ch, nil, nil, d.Output)
}

// cmdWithdraw withdraws gold from the bank
func (d *CommandDispatcher) cmdWithdraw(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}

	// Check if there's a banker in the room
	banker := d.findBanker(ch)
	if banker == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Withdraw how much gold?\r\n")
		return
	}

	var amount int
	if strings.ToLower(args) == "all" {
		amount = ch.PCData.BankGold
	} else {
		var err error
		amount, err = strconv.Atoi(args)
		if err != nil || amount <= 0 {
			d.send(ch, "That's not a valid amount.\r\n")
			return
		}
	}

	if amount > ch.PCData.BankGold {
		d.send(ch, fmt.Sprintf("You only have %d gold in the bank.\r\n", ch.PCData.BankGold))
		return
	}

	ch.PCData.BankGold -= amount
	ch.Gold += amount

	d.send(ch, fmt.Sprintf("You withdraw %d gold coins. Your bank balance is now %d gold.\r\n",
		amount, ch.PCData.BankGold))
	ActToRoom("$n makes a bank withdrawal.", ch, nil, nil, d.Output)
}

// cmdBalance shows the player's bank balance
func (d *CommandDispatcher) cmdBalance(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}

	// Check if there's a banker in the room
	banker := d.findBanker(ch)
	if banker == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Your current bank balance is %d gold coins.\r\n", ch.PCData.BankGold))
}

// findBanker looks for a banker NPC in the room
func (d *CommandDispatcher) findBanker(ch *types.Character) *types.Character {
	if ch.InRoom == nil {
		return nil
	}

	for _, person := range ch.InRoom.People {
		if person.IsNPC() && person.Act.Has(types.ActBanker) {
			return person
		}
	}

	return nil
}
