package game

import (
	"rotmud/pkg/types"
)

// Shop commands: buy, sell, list, value

func (d *CommandDispatcher) cmdBuy(ch *types.Character, args string) {
	if d.Shops == nil {
		d.send(ch, "The shop system is not available.\r\n")
		return
	}
	d.Shops.DoBuy(ch, args)
}

func (d *CommandDispatcher) cmdSell(ch *types.Character, args string) {
	if d.Shops == nil {
		d.send(ch, "The shop system is not available.\r\n")
		return
	}
	d.Shops.DoSell(ch, args)
}

func (d *CommandDispatcher) cmdList(ch *types.Character, args string) {
	if d.Shops == nil {
		d.send(ch, "The shop system is not available.\r\n")
		return
	}
	d.Shops.DoList(ch, args)
}

func (d *CommandDispatcher) cmdValue(ch *types.Character, args string) {
	if d.Shops == nil {
		d.send(ch, "The shop system is not available.\r\n")
		return
	}
	d.Shops.DoValue(ch, args)
}
