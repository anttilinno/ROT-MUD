package game

import (
	"fmt"
	"sort"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// Tier-2 GodWars-style classes: powers, active power commands, rites, demon
// armour forging, and the kill faucet (.planning/TIER2-GODWARS.md T2–T4).

// Active power tuning.
const (
	travelMoveCost = 50
	blastManaCost  = 40
	formMoveCost   = 50
)

// blastPowers maps area-damage power commands to their damage type and messages.
var blastPowers = map[string]struct {
	damType        types.DamageType
	toChar, toRoom string
}{
	"inferno":   {types.DamFire, "You call forth the fires of hell!", "$n calls forth the fires of hell!"},
	"forcebolt": {types.DamEnergy, "Raw force erupts from your hands!", "Raw force erupts from $n's hands!"},
	"smite":     {types.DamHoly, "You call down holy fire!", "$n calls down holy fire!"},
}

// formPowers maps battle-form commands to their transformation messages.
var formPowers = map[string]struct{ toChar, toRoom, wearOff string }{
	"demonform":  {"Your body grows and distorts into a great demonic beast.", "$n's body grows and distorts into a great demonic beast.", "Your body shrinks back to its normal form."},
	"bloodrage":  {"The Beast rises and your eyes burn red.", "$n's eyes burn red as the Beast rises.", "The Beast sinks back into its slumber."},
	"crinos":     {"You swell into the towering war form.", "$n swells into a towering wolf-man.", "You shrink back into your breed form."},
	"quickening": {"Lightning crackles along your blade.", "Lightning crackles along $n's blade.", "The quickening fades."},
	"angelform":  {"Your true glory blazes forth.", "$n's true glory blazes forth, blinding to behold.", "Your glory dims to a mortal glow."},
}

func init() {
	for name, form := range formPowers {
		magic.WearOffMessages[name] = form.wearOff
	}
}

func (d *CommandDispatcher) registerPowerCommands() {
	d.Registry.Register("powers", d.cmdPowers, types.PosResting, 0)
	d.Registry.Register("travel", d.cmdTravel, types.PosStanding, 0)
	d.Registry.Register("rite", d.cmdRite, types.PosStanding, 0)
	d.Registry.Register("demonarmour", d.cmdDemonArmour, types.PosStanding, 0)
	for name := range blastPowers {
		d.Registry.Register(name, d.powerBlast(name), types.PosFighting, 0)
	}
	for name := range formPowers {
		d.Registry.Register(name, d.powerForm(name), types.PosFighting, 0)
	}
}

// cmdPowers lists the character's class powers, or buys one.
// Syntax: powers [power]
func (d *CommandDispatcher) cmdPowers(ch *types.Character, args string) {
	class := types.GetClass(ch.Class)
	if ch.IsNPC() || class == nil || class.Tier2 == nil {
		d.send(ch, "You have no supernatural powers.\r\n")
		return
	}
	if key := strings.ToLower(strings.TrimSpace(args)); key != "" {
		p, err := ch.BuyPower(key)
		if err != nil {
			d.send(ch, capitalizeFirst(err.Error())+".\r\n")
			return
		}
		d.send(ch, fmt.Sprintf("You gain the power of %s. %s\r\n", p.Key, p.Desc))
		if d.OnSave != nil {
			d.OnSave(ch)
		}
		return
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "You have %d %s (%d earned in total).\r\n\r\n", ch.PCData.Power, class.Tier2.Currency, ch.PCData.PowerTotal)
	for _, p := range types.PowerTable {
		if p.Class != ch.Class {
			continue
		}
		mark := " "
		if ch.HasPower(p.Key) {
			mark = "*"
		}
		req := ""
		if p.Requires != "" {
			req = " (needs " + p.Requires + ")"
		}
		fmt.Fprintf(&sb, " %s %-15s %6d  %s%s\r\n", mark, p.Key, p.Cost, p.Desc, req)
	}
	sb.WriteString("\r\n* = owned.  Type 'powers <power>' to gain one.\r\n")
	d.send(ch, sb.String())
}

// cmdTravel moves the character to another player's room.
// Syntax: travel <player>
func (d *CommandDispatcher) cmdTravel(ch *types.Character, args string) {
	if ch.CommandPower("travel") == nil {
		d.send(ch, "Huh?\r\n")
		return
	}
	if ch.Fighting != nil {
		d.send(ch, "Not while fighting!\r\n")
		return
	}
	target := d.GameLoop.FindPlayer(strings.TrimSpace(args))
	switch {
	case args == "":
		d.send(ch, "Travel to whom?\r\n")
		return
	case target == nil || target == ch || target.InRoom == nil:
		d.send(ch, "You cannot sense them.\r\n")
		return
	case ch.InRoom != nil && ch.InRoom.Flags.Has(types.RoomNoRecall),
		target.InRoom.Flags.Has(types.RoomPrivate),
		target.InRoom.Flags.Has(types.RoomSolitary),
		target.InRoom.Flags.Has(types.RoomGodsOnly),
		target.InRoom.Flags.Has(types.RoomImpOnly):
		d.send(ch, "Something blocks your passage.\r\n")
		return
	case ch.Move < travelMoveCost:
		d.send(ch, "You are too exhausted.\r\n")
		return
	}
	ch.Move -= travelMoveCost
	ActToRoom("$n vanishes.", ch, nil, nil, d.Output)
	CharFromRoom(ch)
	CharToRoom(ch, target.InRoom)
	ActToRoom("$n appears from nowhere.", ch, nil, nil, d.Output)
	d.doLook(ch, "")
}

// powerBlast returns the handler for an area-damage power (inferno, forcebolt, smite).
func (d *CommandDispatcher) powerBlast(name string) CommandHandler {
	blast := blastPowers[name]
	return func(ch *types.Character, _ string) {
		if ch.CommandPower(name) == nil {
			d.send(ch, "Huh?\r\n")
			return
		}
		if ch.InRoom == nil || ch.InRoom.Flags.Has(types.RoomSafe) {
			d.send(ch, "Not here.\r\n")
			return
		}
		if ch.Mana < blastManaCost {
			d.send(ch, "You don't have enough mana.\r\n")
			return
		}
		ch.Mana -= blastManaCost
		combat.WaitState(ch, 1)
		d.send(ch, blast.toChar+"\r\n")
		ActToRoom(blast.toRoom, ch, nil, nil, d.Output)
		// Copy: Damage can kill and remove people from the room.
		for _, victim := range append([]*types.Character(nil), ch.InRoom.People...) {
			if victim == ch || !victim.IsNPC() || victim.Master == ch || victim.InRoom != ch.InRoom {
				continue
			}
			d.Combat.Damage(ch, victim, combat.Dice(ch.Level/2+5, 8), blast.damType, true)
			if victim.InRoom == ch.InRoom && victim.Fighting == nil {
				combat.SetFighting(victim, ch)
			}
			if ch.Fighting == nil && victim.InRoom == ch.InRoom {
				combat.SetFighting(ch, victim)
			}
		}
	}
}

// powerForm returns the handler for a timed battle form (demonform, crinos, ...).
// The affect is only a timer; the bonus is derived in types.Character.PowerBonus.
func (d *CommandDispatcher) powerForm(name string) CommandHandler {
	form := formPowers[name]
	return func(ch *types.Character, _ string) {
		if ch.CommandPower(name) == nil {
			d.send(ch, "Huh?\r\n")
			return
		}
		if ch.Affected.HasType(name) {
			d.send(ch, "You are already transformed.\r\n")
			return
		}
		if ch.Move < formMoveCost {
			d.send(ch, "You are too exhausted.\r\n")
			return
		}
		ch.Move -= formMoveCost
		ch.AddAffect(types.NewAffect(name, ch.Level, types.FormDuration, types.ApplyNone, 0, 0))
		d.send(ch, form.toChar+"\r\n")
		ActToRoom(form.toRoom, ch, nil, nil, d.Output)
	}
}

// riteMaster returns the tier-2 class whose rite master is in the room, or -1.
func riteMaster(ch *types.Character) (int, *types.Character) {
	if ch.InRoom == nil {
		return -1, nil
	}
	for _, mob := range ch.InRoom.People {
		if !mob.IsNPC() {
			continue
		}
		for i := types.ClassDemon; i < types.MaxClass; i++ {
			if types.ClassTable[i].Tier2.MasterVnum == mob.MobVnum {
				return i, mob
			}
		}
	}
	return -1, nil
}

// cmdRite starts or completes a tier-2 rite at that class's master.
// Syntax: rite [begin|complete]
func (d *CommandDispatcher) cmdRite(ch *types.Character, args string) {
	class, master := riteMaster(ch)
	if ch.IsNPC() || ch.PCData == nil || master == nil {
		d.send(ch, "There is no one here to perform a rite.\r\n")
		return
	}
	c := &types.ClassTable[class]
	say := func(msg string) { d.send(ch, fmt.Sprintf("%s says, '%s'\r\n", capitalizeFirst(master.ShortDesc), msg)) }
	pc := ch.PCData

	switch {
	case types.IsTier2Class(ch.Class):
		say("You have already left mortality behind.")
		return
	case pc.Tier != types.TierRerolled:
		say("Only a hero who has turned from mortal paths may take my rite. (See 'help reroll'.)")
		return
	case !c.RaceAllowed(ch.Race):
		say(fmt.Sprintf("No %s may take this rite.", types.RaceName(ch.Race)))
		return
	case !c.OriginAllowed(ch.Class):
		say(fmt.Sprintf("You are no %s. This rite is not for you.", types.ClassName(c.Tier2.OriginClasses[0])))
		return
	}

	arg := strings.ToLower(strings.TrimSpace(args))
	switch {
	case pc.Rite != class && arg == "begin":
		pc.Rite, pc.RiteKills = class, 0
		say(c.Tier2.Rite.Desc)
		d.send(ch, fmt.Sprintf("You begin the rite of the %s.\r\n", c.Name))
	case pc.Rite != class:
		say(c.Tier2.Rite.Desc)
		if pc.Rite != 0 {
			d.send(ch, fmt.Sprintf("Beginning this rite abandons your rite of the %s.\r\n", types.ClassName(pc.Rite)))
		}
		d.send(ch, "Type 'rite begin' to take up the rite.\r\n")
		return
	case arg == "complete" && pc.RiteKills >= c.Tier2.Rite.Count:
		if err := ch.Ascend(class); err != nil {
			say(capitalizeFirst(err.Error()) + ".")
			return
		}
		say("It is done.")
		d.send(ch, fmt.Sprintf("You are reborn as a %s! Type 'powers' to see what you may become.\r\n", c.Name))
		ActToRoom(fmt.Sprintf("$n is reborn as a %s!", c.Name), ch, nil, nil, d.Output)
		d.WiznetBroadcast(fmt.Sprintf("%s has ascended as a %s.", ch.Name, c.Name), 100)
	default:
		say(fmt.Sprintf("%d of %d. %s", pc.RiteKills, c.Tier2.Rite.Count, c.Tier2.Rite.Desc))
		if pc.RiteKills >= c.Tier2.Rite.Count {
			d.send(ch, "Type 'rite complete' to ascend.\r\n")
		}
		return
	}
	if d.OnSave != nil {
		d.OnSave(ch)
	}
}

// cmdDemonArmour forges a piece of demonic armour at the demon lord.
// Syntax: demonarmour <tier> <slot>
func (d *CommandDispatcher) cmdDemonArmour(ch *types.Character, args string) {
	if ch.IsNPC() || ch.Class != types.ClassDemon {
		d.send(ch, "Huh?\r\n")
		return
	}
	if class, _ := riteMaster(ch); class != types.ClassDemon {
		d.send(ch, "Only the demon lord can forge demonic armour.\r\n")
		return
	}
	tierName, slot, _ := strings.Cut(strings.ToLower(strings.TrimSpace(args)), " ")
	tier := -1
	for i, n := range types.DemonTierNames {
		if n == tierName {
			tier = i
		}
	}
	vnum, ok := types.DemonicSlots[strings.TrimSpace(slot)]
	if tier < 0 || !ok {
		slots := make([]string, 0, len(types.DemonicSlots))
		for s := range types.DemonicSlots {
			slots = append(slots, s)
		}
		sort.Strings(slots)
		var sb strings.Builder
		sb.WriteString("Syntax: demonarmour <tier> <slot>\r\n")
		fmt.Fprintf(&sb, "Every piece costs %d demonic power, plus:\r\n", types.DemonArmourPower)
		for i, n := range types.DemonTierNames {
			fmt.Fprintf(&sb, "  %-7s %s\r\n", n, types.FormatCoin(types.DemonArmourCoin[i]))
		}
		fmt.Fprintf(&sb, "Slots: %s\r\n", strings.Join(slots, " "))
		d.send(ch, sb.String())
		return
	}
	coin := types.DemonArmourCoin[tier]
	if ch.PCData.Power < types.DemonArmourPower || ch.Coin < coin {
		d.send(ch, fmt.Sprintf("A %s piece costs %d demonic power and %s.\r\n", tierName, types.DemonArmourPower, types.FormatCoin(coin)))
		return
	}
	var obj *types.Object
	if d.GameLoop != nil && d.GameLoop.World != nil {
		obj = d.createObjectFromTemplate(d.GameLoop.World.GetObjTemplate(vnum), 1)
	}
	if obj == nil {
		d.send(ch, "The forge is cold. (Missing object, tell an immortal.)\r\n")
		return
	}
	ch.PCData.Power -= types.DemonArmourPower
	ch.Coin -= coin
	types.ForgeDemonic(obj, tier, ch.Name)
	ch.AddInventory(obj)
	ActToChar("$p appears in your hands in a blast of flames.", ch, nil, obj, d.Output)
	ActToRoom("$p appears in $n's hands in a blast of flames.", ch, nil, obj, d.Output)
	if d.OnSave != nil {
		d.OnSave(ch)
	}
}

// sacrificeDemonic drains a demonic item's power back into a demon (GodWars'
// recycle loop). Returns false when the normal sacrifice should happen.
func (d *CommandDispatcher) sacrificeDemonic(ch *types.Character, obj *types.Object) bool {
	if !types.IsDemonic(obj) || ch.Class != types.ClassDemon || !ch.GainPower(types.DemonArmourPower) {
		return false
	}
	ObjFromRoom(obj)
	d.send(ch, fmt.Sprintf("You drain %d points of demonic power from %s.\r\n", types.DemonArmourPower, obj.ShortDesc))
	ActToRoom("$p vanishes in a blast of flames.", ch, nil, obj, d.Output)
	return true
}

// onKill is the combat kill hook: tier-2 players earn power (the victim's
// level, as GodWars demons did) and rerolled heroes advance their rite.
func (d *CommandDispatcher) onKill(killer, victim *types.Character) {
	if killer.IsNPC() && killer.Master != nil && !killer.Master.IsNPC() {
		killer = killer.Master // pets earn for their owner
	}
	if killer.IsNPC() || killer.PCData == nil || !victim.IsNPC() {
		return
	}
	pc := killer.PCData
	if killer.GainPower(victim.Level) {
		class := types.GetClass(killer.Class)
		d.send(killer, fmt.Sprintf("You gain %d %s.\r\n", victim.Level, class.Tier2.Currency))
		return
	}
	if pc.Tier != types.TierRerolled || pc.Rite == 0 {
		return
	}
	rite := types.ClassTable[pc.Rite].Tier2.Rite
	night := d.GameLoop != nil && d.GameLoop.WorldTime != nil && d.GameLoop.WorldTime.IsDark()
	if pc.RiteKills >= rite.Count || !rite.Matches(victim, night) {
		return
	}
	pc.RiteKills++
	if pc.RiteKills == rite.Count {
		d.send(killer, fmt.Sprintf("Your rite of the %s is fulfilled. Return to your master.\r\n", types.ClassName(pc.Rite)))
	} else {
		d.send(killer, fmt.Sprintf("Your rite of the %s: %d of %d.\r\n", types.ClassName(pc.Rite), pc.RiteKills, rite.Count))
	}
}
