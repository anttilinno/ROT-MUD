package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/types"
)

// cmdReroll marks a tier-1 hero as eligible for a tier-2 rite.
// Syntax: reroll confirm
func (d *CommandDispatcher) cmdReroll(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(args), "confirm") {
		d.send(ch, "Rerolling sets you on the path to a tier 2 class.\r\n")
		d.send(ch, "When you complete a rite you restart at level 1 in your new class;\r\n")
		d.send(ch, "your skills are kept. WARNING: this cannot be undone.\r\n")
		d.send(ch, "\r\nTo confirm, type: reroll confirm\r\n")
		return
	}
	if err := ch.Reroll(); err != nil {
		d.send(ch, capitalizeFirst(err.Error())+".\r\n")
		return
	}
	if d.OnSave != nil {
		d.OnSave(ch)
	}
	d.send(ch, "You turn your back on mortal paths. Seek out a rite to ascend.\r\n")
	d.WiznetBroadcast(fmt.Sprintf("%s has rerolled.", ch.Name), 100)
}

// cmdAscend completes a rerolled player's tier-2 rite by hand, until each
// class's in-game rite exists.
// Syntax: ascend <player> <class>
func (d *CommandDispatcher) cmdAscend(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}
	name, className, _ := strings.Cut(strings.TrimSpace(args), " ")
	if name == "" || className == "" {
		d.send(ch, "Syntax: ascend <player> <tier 2 class>\r\n")
		return
	}
	target := d.GameLoop.FindPlayer(name)
	if target == nil {
		d.send(ch, "Player not found.\r\n")
		return
	}
	class := types.ClassIndexByName(strings.ToLower(className))
	if class < 0 {
		d.send(ch, "No such class.\r\n")
		return
	}
	if err := target.Ascend(class); err != nil {
		d.send(ch, capitalizeFirst(err.Error())+".\r\n")
		return
	}
	if d.OnSave != nil {
		d.OnSave(target)
	}
	d.send(ch, fmt.Sprintf("%s ascends as a %s.\r\n", target.Name, types.ClassName(class)))
	d.send(target, fmt.Sprintf("You are reborn as a %s!\r\n", types.ClassName(class)))
}
