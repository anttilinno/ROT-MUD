package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Consumable commands: eat, drink, quaff, fill, pour
// These replace the stub implementations in commands.go

// Liquid types for drinks
type Liquid struct {
	Name   string
	Color  string
	Drunk  int // How drunk it makes you
	Full   int // How full it makes you
	Thirst int // How much it quenches thirst
	Hunger int // How much it satisfies hunger (usually 0)
	Ssize  int // Serving size
}

// Liquid table - matches ROM/ROT liquid types
var liquidTable = []Liquid{
	{"water", "clear", 0, 1, 10, 0, 16},
	{"beer", "amber", 3, 2, 5, 0, 12},
	{"red wine", "burgundy", 5, 2, 5, 0, 5},
	{"ale", "brown", 2, 2, 5, 0, 12},
	{"dark ale", "dark", 1, 2, 5, 0, 12},
	{"whisky", "golden", 6, 1, 4, 0, 2},
	{"lemonade", "pink", 0, 1, 8, 0, 12},
	{"firebreather", "boiling", 10, 0, 0, 0, 2},
	{"local specialty", "clear", 3, 1, 3, 0, 12},
	{"slime mold juice", "green", 0, 4, -8, 0, 2},
	{"milk", "white", 0, 3, 6, 0, 12},
	{"tea", "tan", 0, 1, 6, 0, 6},
	{"coffee", "black", 0, 1, 6, 0, 6},
	{"blood", "red", 0, 2, -1, 2, 6},
	{"salt water", "clear", 0, 1, -2, 0, 12},
	{"coke", "brown", 0, 2, 9, 0, 12},
	{"root beer", "brown", 0, 2, 9, 0, 12},
	{"elvish wine", "green", 4, 2, 8, 0, 5},
	{"white wine", "golden", 3, 2, 6, 0, 5},
	{"champagne", "golden", 4, 1, 6, 0, 5},
	{"mead", "honey-colored", 4, 2, 6, 0, 12},
	{"rose wine", "pink", 3, 2, 6, 0, 5},
	{"benedictine wine", "burgundy", 4, 2, 6, 0, 5},
	{"vodka", "clear", 7, 1, 4, 0, 2},
	{"cranberry juice", "red", 0, 1, 9, 2, 12},
	{"orange juice", "orange", 0, 2, 9, 3, 12},
	{"absinthe", "green", 9, 1, 0, 0, 2},
	{"brandy", "golden", 5, 1, 4, 0, 4},
	{"aquavit", "clear", 6, 1, 3, 0, 2},
	{"schnapps", "clear", 5, 1, 4, 0, 2},
	{"icewine", "white", 4, 2, 6, 0, 5},
	{"amontillado", "burgundy", 4, 2, 6, 0, 5},
	{"sherry", "gold", 3, 2, 5, 0, 5},
	{"framboise", "red", 4, 2, 6, 0, 5},
	{"rum", "amber", 6, 1, 3, 0, 2},
	{"cordial", "clear", 3, 1, 4, 0, 2},
}

func (d *CommandDispatcher) doEat(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Eat what?\r\n")
		return
	}

	// Find the food in inventory
	obj := d.findObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	// Check if it's edible
	if obj.ItemType != types.ItemTypeFood && obj.ItemType != types.ItemTypePill {
		d.send(ch, "That's not edible.\r\n")
		return
	}

	// Check if player is already full (skip for immortals)
	if !ch.IsNPC() && ch.PCData != nil && !ch.IsImmortal() {
		if ch.PCData.Condition[types.CondFull] > 40 {
			d.send(ch, "You are too full to eat more.\r\n")
			return
		}
	}

	// Show eating messages
	ActToRoom("$n eats $p.", ch, nil, obj, d.Output)
	d.send(ch, fmt.Sprintf("You eat %s.\r\n", obj.ShortDesc))

	switch obj.ItemType {
	case types.ItemTypeFood:
		if !ch.IsNPC() && ch.PCData != nil {
			// Value[0] = hours of fullness
			// Value[1] = hours of hunger satisfaction
			// Value[3] = poisoned (non-zero = poisoned)
			oldHunger := ch.PCData.Condition[types.CondHunger]
			d.gainCondition(ch, types.CondFull, obj.Values[0])
			d.gainCondition(ch, types.CondHunger, obj.Values[1])

			if oldHunger == 0 && ch.PCData.Condition[types.CondHunger] > 0 {
				d.send(ch, "You are no longer hungry.\r\n")
			} else if ch.PCData.Condition[types.CondFull] > 40 {
				d.send(ch, "You are full.\r\n")
			}
		}

		// Check for poisoned food
		if obj.Values[3] != 0 {
			d.send(ch, "You feel sick.\r\n")
			// Apply poison affect
			aff := &types.Affect{
				Type:      "poison",
				Level:     obj.Values[0],
				Duration:  2 * obj.Values[0],
				Location:  types.ApplyStr,
				Modifier:  -2,
				BitVector: types.AffPoison,
			}
			ch.AddAffect(aff)
		}

	case types.ItemTypePill:
		// Pills cast spells when eaten
		// Value[0] = spell level
		// Value[1-4] = spell slot numbers
		d.castObjectSpell(ch, ch, nil, obj.Values[1], obj.Values[0])
		d.castObjectSpell(ch, ch, nil, obj.Values[2], obj.Values[0])
		d.castObjectSpell(ch, ch, nil, obj.Values[3], obj.Values[0])
		d.castObjectSpell(ch, ch, nil, obj.Values[4], obj.Values[0])
	}

	// Remove the food from inventory
	ch.RemoveInventory(obj)
}

func (d *CommandDispatcher) doDrink(ch *types.Character, args string) {
	var obj *types.Object

	if args == "" {
		// Look for a fountain in the room
		if ch.InRoom != nil {
			for _, roomObj := range ch.InRoom.Objects {
				if roomObj.ItemType == types.ItemTypeFountain {
					obj = roomObj
					break
				}
			}
		}
		if obj == nil {
			d.send(ch, "Drink what?\r\n")
			return
		}
	} else {
		// Find the drink in inventory or room
		obj = d.findObjInInventory(ch, args)
		if obj == nil && ch.InRoom != nil {
			obj = d.findObjInRoom(ch.InRoom, args)
		}
		if obj == nil {
			d.send(ch, "You can't find it.\r\n")
			return
		}
	}

	// Check if too drunk to drink
	if !ch.IsNPC() && ch.PCData != nil {
		if ch.PCData.Condition[types.CondDrunk] > 10 {
			d.send(ch, "You fail to reach your mouth. *Hic*\r\n")
			return
		}
	}

	var liquid int
	var amount int

	switch obj.ItemType {
	case types.ItemTypeFountain:
		// Value[2] = liquid type
		liquid = obj.Values[2]
		if liquid < 0 || liquid >= len(liquidTable) {
			liquid = 0
		}
		amount = liquidTable[liquid].Ssize * 3
	case types.ItemTypeDrinkCon:
		// Value[1] = current amount
		// Value[2] = liquid type
		if obj.Values[1] <= 0 {
			d.send(ch, "It is already empty.\r\n")
			return
		}
		liquid = obj.Values[2]
		if liquid < 0 || liquid >= len(liquidTable) {
			liquid = 0
		}
		amount = liquidTable[liquid].Ssize
		if amount > obj.Values[1] {
			amount = obj.Values[1]
		}
	default:
		d.send(ch, "You can't drink from that.\r\n")
		return
	}

	// Check if too full
	if !ch.IsNPC() && ch.PCData != nil && !ch.IsImmortal() {
		if ch.PCData.Condition[types.CondFull] > 45 {
			d.send(ch, "You're too full to drink more.\r\n")
			return
		}
	}

	liq := liquidTable[liquid]

	// Show drinking messages
	ActToRoom(fmt.Sprintf("$n drinks %s from $p.", liq.Name), ch, nil, obj, d.Output)
	d.send(ch, fmt.Sprintf("You drink %s from %s.\r\n", liq.Name, obj.ShortDesc))

	// Apply effects
	if !ch.IsNPC() && ch.PCData != nil {
		d.gainCondition(ch, types.CondDrunk, liq.Drunk*amount/10)
		d.gainCondition(ch, types.CondFull, liq.Full*amount/10)
		d.gainCondition(ch, types.CondThirst, liq.Thirst*amount/10)
		d.gainCondition(ch, types.CondHunger, liq.Hunger*amount/10)

		if ch.PCData.Condition[types.CondDrunk] > 10 {
			d.send(ch, "You feel drunk.\r\n")
		}

		if ch.PCData.Condition[types.CondFull] > 40 {
			d.send(ch, "You are full.\r\n")
		}

		if ch.PCData.Condition[types.CondThirst] > 40 {
			d.send(ch, "Your thirst is quenched.\r\n")
		}
	}

	// Check for poisoned drink
	if obj.ItemType == types.ItemTypeDrinkCon && obj.Values[3] != 0 {
		d.send(ch, "You feel sick.\r\n")
		aff := &types.Affect{
			Type:      "poison",
			Level:     obj.Values[3],
			Duration:  2 * obj.Values[3],
			Location:  types.ApplyStr,
			Modifier:  -2,
			BitVector: types.AffPoison,
		}
		ch.AddAffect(aff)
	}

	// Reduce drink container contents
	if obj.ItemType == types.ItemTypeDrinkCon {
		obj.Values[1] -= amount
		if obj.Values[1] <= 0 {
			obj.Values[1] = 0
			d.send(ch, "The container is now empty.\r\n")
		}
	}
}

func (d *CommandDispatcher) doQuaff(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Quaff what?\r\n")
		return
	}

	obj := d.findObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that potion.\r\n")
		return
	}

	if obj.ItemType != types.ItemTypePotion {
		d.send(ch, "You can quaff only potions.\r\n")
		return
	}

	// Check level requirement
	if ch.Level < obj.Level {
		d.send(ch, "This liquid is too powerful for you to drink.\r\n")
		return
	}

	// Show quaffing messages
	ActToRoom("$n quaffs $p.", ch, nil, obj, d.Output)
	d.send(ch, fmt.Sprintf("You quaff %s.\r\n", obj.ShortDesc))

	// Cast spells from potion
	// Value[0] = spell level
	// Value[1-4] = spell slot numbers
	level := obj.Values[0]
	if level < 1 {
		level = ch.Level
	}
	d.castObjectSpell(ch, ch, nil, obj.Values[1], level)
	d.castObjectSpell(ch, ch, nil, obj.Values[2], level)
	d.castObjectSpell(ch, ch, nil, obj.Values[3], level)
	d.castObjectSpell(ch, ch, nil, obj.Values[4], level)

	// Remove the potion from inventory
	ch.RemoveInventory(obj)
}

func (d *CommandDispatcher) doFill(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Fill what?\r\n")
		return
	}

	// Find the container to fill
	obj := d.findObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	if obj.ItemType != types.ItemTypeDrinkCon {
		d.send(ch, "You can't fill that.\r\n")
		return
	}

	// Check if already full
	// Value[0] = max capacity
	// Value[1] = current amount
	if obj.Values[1] >= obj.Values[0] {
		d.send(ch, "Your container is already full.\r\n")
		return
	}

	// Find a fountain in the room
	var fountain *types.Object
	if ch.InRoom != nil {
		for _, roomObj := range ch.InRoom.Objects {
			if roomObj.ItemType == types.ItemTypeFountain {
				fountain = roomObj
				break
			}
		}
	}

	if fountain == nil {
		d.send(ch, "There is no fountain here.\r\n")
		return
	}

	// Fill the container
	// Match the liquid type of the fountain
	obj.Values[1] = obj.Values[0]
	obj.Values[2] = fountain.Values[2]
	obj.Values[3] = 0 // Not poisoned

	liquid := obj.Values[2]
	if liquid < 0 || liquid >= len(liquidTable) {
		liquid = 0
	}

	d.send(ch, fmt.Sprintf("You fill %s with %s from %s.\r\n",
		obj.ShortDesc, liquidTable[liquid].Name, fountain.ShortDesc))
	ActToRoom(fmt.Sprintf("$n fills %s from %s.", obj.ShortDesc, fountain.ShortDesc),
		ch, nil, obj, d.Output)
}

func (d *CommandDispatcher) doPour(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		d.send(ch, "Pour what into what?\r\n")
		return
	}

	// Find the source container
	obj := d.findObjInInventory(ch, parts[0])
	if obj == nil {
		d.send(ch, "You don't have that item.\r\n")
		return
	}

	if obj.ItemType != types.ItemTypeDrinkCon {
		d.send(ch, "You can't pour from that.\r\n")
		return
	}

	if obj.Values[1] <= 0 {
		d.send(ch, "It's already empty.\r\n")
		return
	}

	// Pour out on ground
	if len(parts) < 2 || strings.ToLower(parts[1]) == "out" {
		liquid := obj.Values[2]
		if liquid < 0 || liquid >= len(liquidTable) {
			liquid = 0
		}

		d.send(ch, fmt.Sprintf("You pour %s out of %s.\r\n",
			liquidTable[liquid].Name, obj.ShortDesc))
		ActToRoom(fmt.Sprintf("$n pours %s out of %s.", liquidTable[liquid].Name, obj.ShortDesc),
			ch, nil, obj, d.Output)

		obj.Values[1] = 0
		return
	}

	// Pour into another container
	into := d.findObjInInventory(ch, parts[1])
	if into == nil {
		d.send(ch, "You don't have that item.\r\n")
		return
	}

	if into == obj {
		d.send(ch, "You can't pour something into itself.\r\n")
		return
	}

	if into.ItemType != types.ItemTypeDrinkCon {
		d.send(ch, "You can't pour into that.\r\n")
		return
	}

	if into.Values[1] >= into.Values[0] {
		d.send(ch, "That container is already full.\r\n")
		return
	}

	// Check if liquids match (can't mix)
	if into.Values[1] > 0 && into.Values[2] != obj.Values[2] {
		d.send(ch, "You can't mix different liquids.\r\n")
		return
	}

	// Pour
	amount := into.Values[0] - into.Values[1]
	if amount > obj.Values[1] {
		amount = obj.Values[1]
	}

	into.Values[1] += amount
	into.Values[2] = obj.Values[2]
	obj.Values[1] -= amount

	liquid := obj.Values[2]
	if liquid < 0 || liquid >= len(liquidTable) {
		liquid = 0
	}

	d.send(ch, fmt.Sprintf("You pour %s from %s into %s.\r\n",
		liquidTable[liquid].Name, obj.ShortDesc, into.ShortDesc))
}

// Helper: apply condition change
func (d *CommandDispatcher) gainCondition(ch *types.Character, cond int, value int) {
	if ch.IsNPC() || ch.PCData == nil || ch.IsImmortal() {
		return
	}

	condition := ch.PCData.Condition[cond]
	ch.PCData.Condition[cond] = boundValue(-48, condition+value, 48)

	// Check for starvation/dehydration effects
	if ch.PCData.Condition[cond] <= 0 {
		switch cond {
		case types.CondHunger:
			d.send(ch, "You are hungry.\r\n")
		case types.CondThirst:
			d.send(ch, "You are thirsty.\r\n")
		}
	}
}

// Helper: bound a value between min and max
func boundValue(min, value, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Helper: cast a spell from an object (potion/pill/scroll/wand/staff)
func (d *CommandDispatcher) castObjectSpell(ch, victim *types.Character, obj *types.Object, spellSlot, level int) {
	if d.Magic == nil || spellSlot <= 0 {
		return
	}

	// Look up spell by slot number
	spell := d.Magic.Registry.FindBySlot(spellSlot)
	if spell == nil {
		return
	}

	// Cast directly without mana cost
	target := interface{}(victim)
	if victim == nil {
		target = ch
	}

	spell.Func(ch, level, target)
}

// doRecite implements the recite command for scrolls
func (d *CommandDispatcher) doRecite(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		d.send(ch, "Recite what?\r\n")
		return
	}

	// Find the scroll in inventory
	scroll := d.findObjInInventory(ch, parts[0])
	if scroll == nil {
		d.send(ch, "You do not have that scroll.\r\n")
		return
	}

	if scroll.ItemType != types.ItemTypeScroll {
		d.send(ch, "You can recite only scrolls.\r\n")
		return
	}

	// Check level requirement
	if ch.Level < scroll.Level {
		d.send(ch, "This scroll is too complex for you to comprehend.\r\n")
		return
	}

	// Find target if specified
	var victim *types.Character
	var targetObj *types.Object
	if len(parts) > 1 {
		victim = FindCharInRoom(ch, parts[1])
		if victim == nil {
			targetObj = d.findObjInInventory(ch, parts[1])
			if targetObj == nil && ch.InRoom != nil {
				targetObj = d.findObjInRoom(ch.InRoom, parts[1])
			}
			if victim == nil && targetObj == nil {
				d.send(ch, "You can't find it.\r\n")
				return
			}
		}
	} else {
		victim = ch
	}

	// Show recite messages
	ActToRoom("$n recites $p.", ch, nil, scroll, d.Output)
	d.send(ch, fmt.Sprintf("You recite %s.\r\n", scroll.ShortDesc))

	// Add lag
	ch.Wait += 6 // 2 * PULSE_VIOLENCE equivalent

	// Check skill - get scrolls skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["scrolls"]
	}
	if ch.IsNPC() {
		skillLevel = 100 // NPCs always succeed
	}

	// Skill check - 20 + skill*4/5 chance to succeed
	if combat.NumberPercent() >= 20+skillLevel*4/5 {
		d.send(ch, "You mispronounce a syllable.\r\n")
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "scrolls", false, 2)
		}
	} else {
		// Cast all spells from scroll
		// Value[0] = spell level
		// Value[1-4] = spell slot numbers
		level := scroll.Values[0]
		if level < 1 {
			level = ch.Level
		}
		d.castObjectSpell(ch, victim, targetObj, scroll.Values[1], level)
		d.castObjectSpell(ch, victim, targetObj, scroll.Values[2], level)
		d.castObjectSpell(ch, victim, targetObj, scroll.Values[3], level)
		d.castObjectSpell(ch, victim, targetObj, scroll.Values[4], level)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "scrolls", true, 2)
		}
	}

	// Remove the scroll
	ch.RemoveInventory(scroll)
}

// doZap implements the zap command for wands
func (d *CommandDispatcher) doZap(ch *types.Character, args string) {
	// Find what to zap
	var victim *types.Character
	var targetObj *types.Object

	if args == "" {
		if ch.Fighting != nil {
			victim = ch.Fighting
		} else {
			d.send(ch, "Zap whom or what?\r\n")
			return
		}
	} else {
		victim = FindCharInRoom(ch, args)
		if victim == nil {
			targetObj = d.findObjInInventory(ch, args)
			if targetObj == nil && ch.InRoom != nil {
				targetObj = d.findObjInRoom(ch.InRoom, args)
			}
			if victim == nil && targetObj == nil {
				d.send(ch, "You can't find it.\r\n")
				return
			}
		}
	}

	// Find the wand in the hold slot
	wand := ch.GetEquipment(types.WearLocHold)
	if wand == nil {
		d.send(ch, "You hold nothing in your hand.\r\n")
		return
	}

	if wand.ItemType != types.ItemTypeWand {
		d.send(ch, "You can zap only with a wand.\r\n")
		return
	}

	// Check charges
	// Value[1] = current charges
	// Value[2] = max charges
	if wand.Values[1] <= 0 {
		d.send(ch, "The wand has no more charges.\r\n")
		return
	}

	// Show zap messages
	if victim != nil {
		ActToRoom(fmt.Sprintf("$n zaps %s with $p.", victim.Name), ch, nil, wand, d.Output)
		d.send(ch, fmt.Sprintf("You zap %s with %s.\r\n", victim.Name, wand.ShortDesc))
	} else if targetObj != nil {
		ActToRoom(fmt.Sprintf("$n zaps %s with $p.", targetObj.ShortDesc), ch, nil, wand, d.Output)
		d.send(ch, fmt.Sprintf("You zap %s with %s.\r\n", targetObj.ShortDesc, wand.ShortDesc))
	}

	// Add lag
	ch.Wait += 6

	// Check skill
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["wands"]
	}
	if ch.IsNPC() {
		skillLevel = 100
	}

	// Check level and skill - level check + 20 + skill*4/5 chance to succeed
	if ch.Level < wand.Level || combat.NumberPercent() >= 20+skillLevel*4/5 {
		d.send(ch, "Your efforts with the wand produce only smoke and sparks.\r\n")
		ActToRoom("$n's efforts with $p produce only smoke and sparks.", ch, nil, wand, d.Output)
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "wands", false, 2)
		}
	} else {
		// Cast spell from wand
		// Value[0] = spell level
		// Value[3] = spell slot number
		level := wand.Values[0]
		if level < 1 {
			level = ch.Level
		}
		d.castObjectSpell(ch, victim, targetObj, wand.Values[3], level)

		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "wands", true, 2)
		}
	}

	// Reduce charges
	wand.Values[1]--

	// Check if wand is depleted
	if wand.Values[1] <= 0 {
		d.send(ch, fmt.Sprintf("Your %s explodes into fragments.\r\n", wand.ShortDesc))
		ActToRoom("$n's $p explodes into fragments.", ch, nil, wand, d.Output)
		// Remove from equipment
		ch.Equipment[types.WearLocHold] = nil
	}
}

// doBrandish implements the brandish command for staves
func (d *CommandDispatcher) doBrandish(ch *types.Character, args string) {
	// Find the staff in the hold slot
	staff := ch.GetEquipment(types.WearLocHold)
	if staff == nil {
		d.send(ch, "You hold nothing in your hand.\r\n")
		return
	}

	if staff.ItemType != types.ItemTypeStaff {
		d.send(ch, "You can brandish only with a staff.\r\n")
		return
	}

	// Check charges
	// Value[1] = current charges
	// Value[2] = max charges
	if staff.Values[1] <= 0 {
		d.send(ch, "The staff has no more charges.\r\n")
		return
	}

	// Show brandish messages
	ActToRoom("$n brandishes $p.", ch, nil, staff, d.Output)
	d.send(ch, fmt.Sprintf("You brandish %s.\r\n", staff.ShortDesc))

	// Add lag
	ch.Wait += 6

	// Check skill
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["staves"]
	}
	if ch.IsNPC() {
		skillLevel = 100
	}

	// Check level and skill
	if ch.Level < staff.Level || combat.NumberPercent() >= 20+skillLevel*4/5 {
		d.send(ch, "You fail to invoke the staff.\r\n")
		ActToRoom("...and nothing happens.", ch, nil, nil, d.Output)
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "staves", false, 2)
		}
	} else {
		// Get spell info
		// Value[0] = spell level
		// Value[3] = spell slot number
		level := staff.Values[0]
		if level < 1 {
			level = ch.Level
		}

		spell := d.Magic.Registry.FindBySlot(staff.Values[3])
		if spell != nil {
			// Cast on appropriate targets based on spell type
			if ch.InRoom != nil {
				for _, vch := range ch.InRoom.People {
					switch spell.Target {
					case 0: // TargetIgnore - only affects caster
						if vch != ch {
							continue
						}
					case 1: // TargetCharOffense - affects enemies
						if ch.IsNPC() == vch.IsNPC() {
							continue
						}
					case 2: // TargetCharDefense - affects allies
						if ch.IsNPC() != vch.IsNPC() {
							continue
						}
					case 3: // TargetCharSelf - only affects caster
						if vch != ch {
							continue
						}
					}
					d.castObjectSpell(ch, vch, nil, staff.Values[3], level)
				}
			}

			if d.Skills != nil {
				d.Skills.CheckImprove(ch, "staves", true, 2)
			}
		}
	}

	// Reduce charges
	staff.Values[1]--

	// Check if staff is depleted
	if staff.Values[1] <= 0 {
		d.send(ch, fmt.Sprintf("Your %s blazes bright and is gone.\r\n", staff.ShortDesc))
		ActToRoom("$n's $p blazes bright and is gone.", ch, nil, staff, d.Output)
		// Remove from equipment
		ch.Equipment[types.WearLocHold] = nil
	}
}

// doEnvenom implements the envenom command for poisoning food/drink and weapons
func (d *CommandDispatcher) doEnvenom(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Envenom what item?\r\n")
		return
	}

	// Find the object in inventory
	obj := d.findObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You don't have that item.\r\n")
		return
	}

	// Get skill level
	skillLevel := 0
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Learned != nil {
		skillLevel = ch.PCData.Learned["envenom"]
	}
	if ch.IsNPC() {
		skillLevel = 75
	}

	if skillLevel <= 0 {
		d.send(ch, "Are you crazy? You'd poison yourself!\r\n")
		return
	}

	// Handle food and drink containers
	if obj.ItemType == types.ItemTypeFood || obj.ItemType == types.ItemTypeDrinkCon {
		// Can't poison blessed or burn-proof items
		if obj.ExtraFlags.Has(types.ItemBless) || obj.ExtraFlags.Has(types.ItemBurnProof) {
			ActObj("You fail to poison $p.", ch, obj, d.Output)
			return
		}

		if combat.NumberPercent() < skillLevel {
			// Success - poison the food/drink
			ActToRoom("$n treats $p with deadly poison.", ch, nil, obj, d.Output)
			ActObj("You treat $p with deadly poison.", ch, obj, d.Output)

			if obj.Values[3] == 0 {
				obj.Values[3] = 1
				if d.Skills != nil {
					d.Skills.CheckImprove(ch, "envenom", true, 4)
				}
			}
			ch.Wait = 3
			return
		}

		// Failure
		ActObj("You fail to poison $p.", ch, obj, d.Output)
		if obj.Values[3] == 0 {
			if d.Skills != nil {
				d.Skills.CheckImprove(ch, "envenom", false, 4)
			}
		}
		ch.Wait = 3
		return
	}

	// Handle weapons
	if obj.ItemType == types.ItemTypeWeapon {
		// Check for special weapon properties that prevent envenoming
		// Check object flags
		if obj.ExtraFlags.Has(types.ItemBless) || obj.ExtraFlags.Has(types.ItemBurnProof) {
			ActObj("You can't seem to envenom $p.", ch, obj, d.Output)
			return
		}

		// Check for existing weapon affects that block envenoming
		// (flaming, frost, vampiric, shocking, vorpal, sharp)
		blockingAffects := []string{"flaming", "frost", "vampiric", "shocking", "vorpal", "sharp"}
		for _, affType := range blockingAffects {
			if obj.Affects.HasType(affType) {
				ActObj("You can't seem to envenom $p.", ch, obj, d.Output)
				return
			}
		}

		// Check damage type - can only envenom edged weapons (not bash)
		damType := obj.DamageType()
		if damType == types.DamBash {
			d.send(ch, "You can only envenom edged weapons.\r\n")
			return
		}

		// Check if already poisoned
		if obj.Affects.HasType("poison") || obj.Affects.HasType("envenom") {
			ActObj("$p is already envenomed.", ch, obj, d.Output)
			return
		}

		percent := combat.NumberPercent()
		if percent < skillLevel {
			// Success - apply poison affect to weapon
			affLevel := ch.Level * percent / 100
			affDuration := ch.Level / 2 * percent / 100
			if affLevel < 1 {
				affLevel = 1
			}
			if affDuration < 1 {
				affDuration = 1
			}

			aff := &types.Affect{
				Type:      "envenom",
				Level:     affLevel,
				Duration:  affDuration,
				Location:  0, // No stat modification
				Modifier:  0,
				BitVector: 0,
			}
			obj.Affects.Add(aff)

			ActToRoom("$n coats $p with deadly venom.", ch, nil, obj, d.Output)
			ActObj("You coat $p with venom.", ch, obj, d.Output)

			if d.Skills != nil {
				d.Skills.CheckImprove(ch, "envenom", true, 3)
			}
			ch.Wait = 3
			return
		}

		// Failure
		ActObj("You fail to envenom $p.", ch, obj, d.Output)
		if d.Skills != nil {
			d.Skills.CheckImprove(ch, "envenom", false, 3)
		}
		ch.Wait = 3
		return
	}

	ActObj("You can't poison $p.", ch, obj, d.Output)
}

// ActObj sends an object-targeted message to the character
func ActObj(format string, ch *types.Character, obj *types.Object, output func(*types.Character, string)) {
	if output == nil || ch == nil {
		return
	}

	msg := strings.ReplaceAll(format, "$p", obj.ShortDesc)
	output(ch, msg+"\r\n")
}
