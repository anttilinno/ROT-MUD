package game

import (
	"fmt"
	"strconv"
	"strings"

	"rotmud/pkg/types"
)

// Bank commands: deposit, withdraw, balance.
//
// Amount parsing accepts:
//   - "all"                       — entire balance
//   - "<N>"                       — N gold pieces (backcompat default)
//   - "<N> <copper|silver|gold|platinum>"  — explicit denomination
//   - "<N> <c|s|g|p>"             — short denomination
//
// Both player Coin and PCData.BankCoin are stored in copper.

// parseCoinArg parses a bank/deposit argument into a copper amount.
// Returns (copper, allRequested, ok). When allRequested is true the caller
// should substitute the relevant balance for the amount.
func parseCoinArg(arg string) (int64, bool, bool) {
	arg = strings.TrimSpace(strings.ToLower(arg))
	if arg == "" {
		return 0, false, false
	}
	if arg == "all" {
		return 0, true, true
	}

	parts := strings.Fields(arg)
	n, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || n < 0 {
		return 0, false, false
	}

	// No denomination → legacy default of gold.
	if len(parts) == 1 {
		return n * types.CopperPerGold, false, true
	}

	mul := types.ParseCoinDenom(parts[1])
	if mul == 0 {
		return 0, false, false
	}
	return n * mul, false, true
}

// cmdDeposit deposits coin into the bank.
func (d *CommandDispatcher) cmdDeposit(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}
	if d.findBanker(ch) == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	amount, all, ok := parseCoinArg(args)
	if !ok {
		d.send(ch, "Deposit how much? (e.g. 'deposit 5 gold', 'deposit 100 copper', 'deposit all')\r\n")
		return
	}
	if all {
		amount = ch.Coin
	}
	if amount <= 0 {
		d.send(ch, "You have nothing to deposit.\r\n")
		return
	}
	if amount > ch.Coin {
		d.send(ch, fmt.Sprintf("You only have %s.\r\n", types.FormatCoin(ch.Coin)))
		return
	}

	ch.Coin -= amount
	ch.PCData.BankCoin += amount

	d.send(ch, fmt.Sprintf("You deposit %s. Your bank balance is now %s.\r\n",
		types.FormatCoin(amount), types.FormatCoin(ch.PCData.BankCoin)))
	ActToRoom("$n makes a bank deposit.", ch, nil, nil, d.Output)
}

// cmdWithdraw withdraws coin from the bank.
func (d *CommandDispatcher) cmdWithdraw(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}
	if d.findBanker(ch) == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	amount, all, ok := parseCoinArg(args)
	if !ok {
		d.send(ch, "Withdraw how much? (e.g. 'withdraw 5 gold', 'withdraw all')\r\n")
		return
	}
	if all {
		amount = ch.PCData.BankCoin
	}
	if amount <= 0 {
		d.send(ch, "You have nothing to withdraw.\r\n")
		return
	}
	if amount > ch.PCData.BankCoin {
		d.send(ch, fmt.Sprintf("You only have %s in the bank.\r\n",
			types.FormatCoin(ch.PCData.BankCoin)))
		return
	}

	ch.PCData.BankCoin -= amount
	ch.Coin += amount

	d.send(ch, fmt.Sprintf("You withdraw %s. Your bank balance is now %s.\r\n",
		types.FormatCoin(amount), types.FormatCoin(ch.PCData.BankCoin)))
	ActToRoom("$n makes a bank withdrawal.", ch, nil, nil, d.Output)
}

// cmdBalance shows the player's bank balance.
func (d *CommandDispatcher) cmdBalance(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "Only players can use the bank.\r\n")
		return
	}
	if d.findBanker(ch) == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Your current bank balance is %s.\r\n",
		types.FormatCoin(ch.PCData.BankCoin)))
}

// findBanker looks for a banker NPC in the room.
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
