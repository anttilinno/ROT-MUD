package game

import (
	"fmt"
	"strconv"
	"strings"

	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// isSubIssueEquipment returns true if the object is sub issue (newbie) equipment
// These items disintegrate when dropped to prevent clutter
func isSubIssueEquipment(obj *types.Object) bool {
	// Sub issue equipment vnums are 3700-3722
	return obj.Vnum >= 3700 && obj.Vnum <= 3722
}

// parseQuantityArg parses a quantity+name argument like "5 coin" or just "coin"
// Returns the count (0 means no quantity specified, use single item), and the item name.
func parseQuantityArg(args string) (int, string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		return 0, args
	}

	// Try to parse the first word as a number
	count, err := strconv.Atoi(parts[0])
	if err != nil || count <= 0 {
		// Not a number, treat as single item with complex name
		return 0, args
	}

	return count, parts[1]
}

// applyObjectAffects applies all of an object's affects to a character (when equipping)
// Unlike spell affects, object affects just modify stats - they don't add to the Affected list
func applyObjectAffects(ch *types.Character, obj *types.Object) {
	for _, af := range obj.Affects.All() {
		magic.ApplyModifier(ch, af)
	}
}

// removeObjectAffects removes all of an object's affects from a character (when unequipping)
func removeObjectAffects(ch *types.Character, obj *types.Object) {
	for _, af := range obj.Affects.All() {
		magic.ReverseModifier(ch, af)
	}
}

// Object manipulation commands: get, drop, give, put, sacrifice

func (d *CommandDispatcher) cmdGet(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Get what?\r\n")
		return
	}

	// Parse "get <item>" or "get <item> <container>" or "get <count> <item>"
	parts := strings.SplitN(args, " ", 2)
	itemName := parts[0]

	// Check for "get all" or "get all.<item>" or "get all <container>"
	if itemName == "all" {
		// Check if there's a container specified: "get all corpse"
		if len(parts) > 1 {
			containerName := parts[1]
			d.getFromContainer(ch, "all", containerName)
			return
		}
		d.getAll(ch, "")
		return
	}

	if strings.HasPrefix(itemName, "all.") {
		// Check for "get all.item container" format
		if len(parts) > 1 {
			containerName := parts[1]
			d.getFromContainer(ch, itemName, containerName)
			return
		}
		d.getAll(ch, strings.TrimPrefix(itemName, "all."))
		return
	}

	// Check if first arg is a number (quantity syntax)
	count, err := strconv.Atoi(itemName)
	if err == nil && count > 0 && len(parts) > 1 {
		// "get 5 coin" or "get 5 coin bag" format
		remainingParts := strings.SplitN(parts[1], " ", 2)
		itemName = remainingParts[0]

		if len(remainingParts) > 1 {
			// "get 5 coin bag" - get from container
			d.getQuantityFromContainer(ch, itemName, remainingParts[1], count)
		} else {
			// "get 5 coin" - get from room
			d.getQuantity(ch, itemName, count)
		}
		return
	}

	// Get from container?
	if len(parts) > 1 {
		containerName := parts[1]
		d.getFromContainer(ch, itemName, containerName)
		return
	}

	// Get from room
	obj := FindObjInRoom(ch, itemName)
	if obj == nil {
		d.send(ch, "You don't see that here.\r\n")
		return
	}

	if !obj.CanTake() {
		d.send(ch, "You can't take that.\r\n")
		return
	}

	// Check weight/carry limits would go here
	// For now, just pick it up
	ObjFromRoom(obj)
	ObjToChar(obj, ch)

	d.send(ch, fmt.Sprintf("You get %s.\r\n", obj.ShortDesc))
	ActToRoom("$n gets $p.", ch, nil, obj, d.Output)

	// Check for quest progress
	d.checkQuestItemGet(ch, obj)
}

// getQuantity picks up a specific number of matching items from the room
func (d *CommandDispatcher) getQuantity(ch *types.Character, itemName string, count int) {
	objects := FindStackedObjectsInRoom(ch, itemName, count)

	if len(objects) == 0 {
		d.send(ch, "You don't see that here.\r\n")
		return
	}

	gotten := 0
	var lastObj *types.Object

	for _, obj := range objects {
		if !obj.CanTake() {
			continue
		}

		ObjFromRoom(obj)
		ObjToChar(obj, ch)
		d.checkQuestItemGet(ch, obj)
		gotten++
		lastObj = obj
	}

	if gotten == 0 {
		d.send(ch, "You can't take that.\r\n")
		return
	}

	if gotten == 1 {
		d.send(ch, fmt.Sprintf("You get %s.\r\n", lastObj.ShortDesc))
		ActToRoom("$n gets $p.", ch, nil, lastObj, d.Output)
	} else {
		d.send(ch, fmt.Sprintf("You get %d %s.\r\n", gotten, lastObj.ShortDesc))
		ActToRoom(fmt.Sprintf("$n gets %d $p.", gotten), ch, nil, lastObj, d.Output)
	}
}

// getQuantityFromContainer picks up a specific number of items from a container
func (d *CommandDispatcher) getQuantityFromContainer(ch *types.Character, itemName, containerName string, count int) {
	// Find the container - check inventory first, then room
	container := FindObjOnChar(ch, containerName)
	if container == nil {
		container = FindObjInRoom(ch, containerName)
	}

	if container == nil {
		d.send(ch, fmt.Sprintf("You don't see any %s here.\r\n", containerName))
		return
	}

	if container.ItemType != types.ItemTypeContainer && container.ItemType != types.ItemTypeCorpseNPC && container.ItemType != types.ItemTypeCorpsePC {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if container is closed
	if container.Values[1]&1 != 0 { // CONT_CLOSED flag
		d.send(ch, "It is closed.\r\n")
		return
	}

	// Find matching objects in container
	var matches []*types.Object
	for _, obj := range container.Contents {
		if nameMatches(obj.Name, itemName) || keywordsMatch(obj.ShortDesc, itemName) {
			matches = append(matches, obj)
			if len(matches) >= count {
				break
			}
		}
	}

	if len(matches) == 0 {
		d.send(ch, fmt.Sprintf("You don't see that in %s.\r\n", container.ShortDesc))
		return
	}

	gotten := 0
	var lastObj *types.Object

	for _, obj := range matches {
		container.RemoveContent(obj)
		ObjToChar(obj, ch)
		d.checkQuestItemGet(ch, obj)
		gotten++
		lastObj = obj
	}

	if gotten == 1 {
		d.send(ch, fmt.Sprintf("You get %s from %s.\r\n", lastObj.ShortDesc, container.ShortDesc))
		ActToRoom("$n gets $p from $P.", ch, nil, lastObj, d.Output)
	} else {
		d.send(ch, fmt.Sprintf("You get %d %s from %s.\r\n", gotten, lastObj.ShortDesc, container.ShortDesc))
		ActToRoom(fmt.Sprintf("$n gets %d $p from $P.", gotten), ch, nil, lastObj, d.Output)
	}
}

func (d *CommandDispatcher) getAll(ch *types.Character, filter string) {
	if ch.InRoom == nil {
		return
	}

	found := false
	// Make a copy of the slice since we're modifying it
	objects := make([]*types.Object, len(ch.InRoom.Objects))
	copy(objects, ch.InRoom.Objects)

	for _, obj := range objects {
		if !obj.CanTake() {
			continue
		}
		if filter != "" && !nameMatches(obj.Name, filter) {
			continue
		}

		ObjFromRoom(obj)
		ObjToChar(obj, ch)
		d.send(ch, fmt.Sprintf("You get %s.\r\n", obj.ShortDesc))
		ActToRoom("$n gets $p.", ch, nil, obj, d.Output)
		d.checkQuestItemGet(ch, obj)
		found = true
	}

	if !found {
		if filter != "" {
			d.send(ch, fmt.Sprintf("You don't see any %s here.\r\n", filter))
		} else {
			d.send(ch, "You don't see anything here.\r\n")
		}
	}
}

func (d *CommandDispatcher) getFromContainer(ch *types.Character, itemName, containerName string) {
	// Find the container - check inventory first, then room
	container := FindObjOnChar(ch, containerName)
	if container == nil {
		container = FindObjInRoom(ch, containerName)
	}

	if container == nil {
		d.send(ch, fmt.Sprintf("You don't see any %s here.\r\n", containerName))
		return
	}

	if container.ItemType != types.ItemTypeContainer && container.ItemType != types.ItemTypeCorpseNPC && container.ItemType != types.ItemTypeCorpsePC {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if this is a player corpse with noloot flag
	if container.ItemType == types.ItemTypeCorpsePC {
		// Values[4] = 1 means noloot, Owner is the player name
		if container.Values[4] == 1 && container.Owner != ch.Name {
			d.send(ch, "That corpse is protected from looting.\r\n")
			return
		}
	}

	// Check if container is closed
	if container.Values[1]&1 != 0 { // CONT_CLOSED flag
		d.send(ch, "It is closed.\r\n")
		return
	}

	if itemName == "all" {
		d.getAllFromContainer(ch, container, "")
		return
	}

	if strings.HasPrefix(itemName, "all.") {
		d.getAllFromContainer(ch, container, strings.TrimPrefix(itemName, "all."))
		return
	}

	// Find specific item in container
	var obj *types.Object
	for _, item := range container.Contents {
		if nameMatches(item.Name, itemName) {
			obj = item
			break
		}
	}

	if obj == nil {
		d.send(ch, fmt.Sprintf("You don't see that in %s.\r\n", container.ShortDesc))
		return
	}

	container.RemoveContent(obj)

	// If looting money from own corpse, add directly to gold
	if obj.ItemType == types.ItemTypeMoney && container.ItemType == types.ItemTypeCorpsePC && container.Owner == ch.Name {
		ch.Gold += obj.Cost
		d.send(ch, fmt.Sprintf("You get %s from %s.\r\n", obj.ShortDesc, container.ShortDesc))
		ActToRoom("$n gets $p from $P.", ch, nil, obj, d.Output)
		return
	}

	ObjToChar(obj, ch)

	d.send(ch, fmt.Sprintf("You get %s from %s.\r\n", obj.ShortDesc, container.ShortDesc))
	ActToRoom("$n gets $p from $P.", ch, nil, obj, d.Output)
	d.checkQuestItemGet(ch, obj)
}

func (d *CommandDispatcher) getAllFromContainer(ch *types.Character, container *types.Object, filter string) {
	found := false
	contents := make([]*types.Object, len(container.Contents))
	copy(contents, container.Contents)

	// Check if this is the player's own corpse
	isOwnCorpse := container.ItemType == types.ItemTypeCorpsePC && container.Owner == ch.Name

	for _, obj := range contents {
		if filter != "" && !nameMatches(obj.Name, filter) {
			continue
		}

		container.RemoveContent(obj)

		// If looting money from own corpse, add directly to gold
		if obj.ItemType == types.ItemTypeMoney && isOwnCorpse {
			ch.Gold += obj.Cost
			d.send(ch, fmt.Sprintf("You get %s from %s.\r\n", obj.ShortDesc, container.ShortDesc))
			found = true
			continue
		}

		ObjToChar(obj, ch)
		d.send(ch, fmt.Sprintf("You get %s from %s.\r\n", obj.ShortDesc, container.ShortDesc))
		d.checkQuestItemGet(ch, obj)
		found = true
	}

	if !found {
		d.send(ch, fmt.Sprintf("You don't see anything in %s.\r\n", container.ShortDesc))
	}
}

func (d *CommandDispatcher) cmdDrop(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Drop what?\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You are nowhere!\r\n")
		return
	}

	// Check for "drop all" or "drop all.<item>"
	if args == "all" {
		d.dropAll(ch, "")
		return
	}

	if strings.HasPrefix(args, "all.") {
		d.dropAll(ch, strings.TrimPrefix(args, "all."))
		return
	}

	// Parse for quantity syntax: "drop 5 coin"
	count, itemName := parseQuantityArg(args)

	if count > 0 {
		// Drop multiple items
		d.dropQuantity(ch, itemName, count)
		return
	}

	// Find the object in inventory
	obj := FindObjInInventory(ch, itemName)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	// Check for no-drop flag
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		d.send(ch, "You can't let go of it.\r\n")
		return
	}

	// Sub issue equipment disintegrates when dropped
	if isSubIssueEquipment(obj) {
		ObjFromChar(obj)
		d.send(ch, fmt.Sprintf("You drop %s and it disintegrates.\r\n", obj.ShortDesc))
		ActToRoom("$n drops $p and it disintegrates.", ch, nil, obj, d.Output)
		return
	}

	ObjFromChar(obj)
	ObjToRoom(obj, ch.InRoom)

	d.send(ch, fmt.Sprintf("You drop %s.\r\n", obj.ShortDesc))
	ActToRoom("$n drops $p.", ch, nil, obj, d.Output)
}

// dropQuantity drops a specific number of matching items
func (d *CommandDispatcher) dropQuantity(ch *types.Character, itemName string, count int) {
	objects := FindStackedObjectsInInventory(ch, itemName, count)

	if len(objects) == 0 {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	dropped := 0
	var lastObj *types.Object

	disintegrated := 0
	for _, obj := range objects {
		if obj.ExtraFlags.Has(types.ItemNoDrop) {
			continue
		}

		ObjFromChar(obj)
		if isSubIssueEquipment(obj) {
			disintegrated++
		} else {
			ObjToRoom(obj, ch.InRoom)
		}
		dropped++
		lastObj = obj
	}

	if dropped == 0 {
		d.send(ch, "You can't let go of it.\r\n")
		return
	}

	if dropped == 1 {
		if disintegrated > 0 {
			d.send(ch, fmt.Sprintf("You drop %s and it disintegrates.\r\n", lastObj.ShortDesc))
			ActToRoom("$n drops $p and it disintegrates.", ch, nil, lastObj, d.Output)
		} else {
			d.send(ch, fmt.Sprintf("You drop %s.\r\n", lastObj.ShortDesc))
			ActToRoom("$n drops $p.", ch, nil, lastObj, d.Output)
		}
	} else {
		if disintegrated > 0 {
			d.send(ch, fmt.Sprintf("You drop %d %s and they disintegrate.\r\n", dropped, lastObj.ShortDesc))
			ActToRoom(fmt.Sprintf("$n drops %d $p and they disintegrate.", dropped), ch, nil, lastObj, d.Output)
		} else {
			d.send(ch, fmt.Sprintf("You drop %d %s.\r\n", dropped, lastObj.ShortDesc))
			ActToRoom(fmt.Sprintf("$n drops %d $p.", dropped), ch, nil, lastObj, d.Output)
		}
	}
}

func (d *CommandDispatcher) dropAll(ch *types.Character, filter string) {
	found := false
	inventory := make([]*types.Object, len(ch.Inventory))
	copy(inventory, ch.Inventory)

	for _, obj := range inventory {
		if filter != "" && !nameMatches(obj.Name, filter) {
			continue
		}
		if obj.ExtraFlags.Has(types.ItemNoDrop) {
			continue
		}

		ObjFromChar(obj)
		if isSubIssueEquipment(obj) {
			d.send(ch, fmt.Sprintf("You drop %s and it disintegrates.\r\n", obj.ShortDesc))
			ActToRoom("$n drops $p and it disintegrates.", ch, nil, obj, d.Output)
		} else {
			ObjToRoom(obj, ch.InRoom)
			d.send(ch, fmt.Sprintf("You drop %s.\r\n", obj.ShortDesc))
			ActToRoom("$n drops $p.", ch, nil, obj, d.Output)
		}
		found = true
	}

	if !found {
		if filter != "" {
			d.send(ch, fmt.Sprintf("You don't have any %s.\r\n", filter))
		} else {
			d.send(ch, "You aren't carrying anything.\r\n")
		}
	}
}

func (d *CommandDispatcher) cmdGive(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Give what to whom?\r\n")
		return
	}

	// Parse "give <item> <target>" or "give <count> <item> <target>"
	parts := strings.Fields(args)
	if len(parts) < 2 {
		d.send(ch, "Give it to whom?\r\n")
		return
	}

	// Try to parse first word as a count
	count, err := strconv.Atoi(parts[0])
	var itemName, targetName string

	if err == nil && count > 0 && len(parts) >= 3 {
		// "give 5 coin player" format
		itemName = parts[1]
		targetName = parts[2]
		// If there are more words, append them to target name
		if len(parts) > 3 {
			targetName = strings.Join(parts[2:], " ")
		}
	} else {
		// "give item player" format - standard two-arg case
		count = 0
		itemName = parts[0]
		targetName = strings.Join(parts[1:], " ")
	}

	// Find the target first (it's in the room)
	victim := FindCharInRoom(ch, targetName)
	if victim == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "Give it to yourself? How generous.\r\n")
		return
	}

	if count > 0 {
		// Give multiple items
		d.giveQuantity(ch, victim, itemName, count)
		return
	}

	// Find the item
	obj := FindObjInInventory(ch, itemName)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	// Check for no-drop flag
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		d.send(ch, "You can't let go of it.\r\n")
		return
	}

	// Transfer the object
	ObjFromChar(obj)
	ObjToChar(obj, victim)

	d.send(ch, fmt.Sprintf("You give %s to %s.\r\n", obj.ShortDesc, victim.Name))
	d.send(victim, fmt.Sprintf("%s gives you %s.\r\n", ch.Name, obj.ShortDesc))
	ActToNotVict("$n gives $p to $N.", ch, victim, obj, d.Output)
}

// giveQuantity gives a specific number of matching items to a target
func (d *CommandDispatcher) giveQuantity(ch *types.Character, victim *types.Character, itemName string, count int) {
	objects := FindStackedObjectsInInventory(ch, itemName, count)

	if len(objects) == 0 {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	given := 0
	var lastObj *types.Object

	for _, obj := range objects {
		if obj.ExtraFlags.Has(types.ItemNoDrop) {
			continue
		}

		ObjFromChar(obj)
		ObjToChar(obj, victim)
		given++
		lastObj = obj
	}

	if given == 0 {
		d.send(ch, "You can't let go of it.\r\n")
		return
	}

	if given == 1 {
		d.send(ch, fmt.Sprintf("You give %s to %s.\r\n", lastObj.ShortDesc, victim.Name))
		d.send(victim, fmt.Sprintf("%s gives you %s.\r\n", ch.Name, lastObj.ShortDesc))
		ActToNotVict("$n gives $p to $N.", ch, victim, lastObj, d.Output)
	} else {
		d.send(ch, fmt.Sprintf("You give %d %s to %s.\r\n", given, lastObj.ShortDesc, victim.Name))
		d.send(victim, fmt.Sprintf("%s gives you %d %s.\r\n", ch.Name, given, lastObj.ShortDesc))
		ActToNotVict(fmt.Sprintf("$n gives %d $p to $N.", given), ch, victim, lastObj, d.Output)
	}
}

func (d *CommandDispatcher) cmdPut(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Put what in what?\r\n")
		return
	}

	// Parse "put <item> <container>"
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Put it in what?\r\n")
		return
	}

	itemName := parts[0]
	containerName := parts[1]

	// Find the container
	container := FindObjOnChar(ch, containerName)
	if container == nil {
		container = FindObjInRoom(ch, containerName)
	}

	if container == nil {
		d.send(ch, fmt.Sprintf("You don't see any %s here.\r\n", containerName))
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if container is closed
	if container.Values[1]&1 != 0 { // CONT_CLOSED flag
		d.send(ch, "It is closed.\r\n")
		return
	}

	if itemName == "all" {
		d.putAll(ch, container, "")
		return
	}

	if strings.HasPrefix(itemName, "all.") {
		d.putAll(ch, container, strings.TrimPrefix(itemName, "all."))
		return
	}

	// Find the item
	obj := FindObjInInventory(ch, itemName)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	if obj == container {
		d.send(ch, "You can't fold it into itself.\r\n")
		return
	}

	// Check weight capacity
	if obj.Weight+container.ContentsWeight() > container.Capacity()*10 {
		d.send(ch, "It won't fit.\r\n")
		return
	}

	ObjFromChar(obj)
	container.AddContent(obj)

	d.send(ch, fmt.Sprintf("You put %s in %s.\r\n", obj.ShortDesc, container.ShortDesc))
	ActToRoom("$n puts $p in $P.", ch, nil, obj, d.Output)
}

func (d *CommandDispatcher) putAll(ch *types.Character, container *types.Object, filter string) {
	found := false
	inventory := make([]*types.Object, len(ch.Inventory))
	copy(inventory, ch.Inventory)

	for _, obj := range inventory {
		if obj == container {
			continue
		}
		if filter != "" && !nameMatches(obj.Name, filter) {
			continue
		}

		// Check weight capacity
		if obj.Weight+container.ContentsWeight() > container.Capacity()*10 {
			continue
		}

		ObjFromChar(obj)
		container.AddContent(obj)
		d.send(ch, fmt.Sprintf("You put %s in %s.\r\n", obj.ShortDesc, container.ShortDesc))
		found = true
	}

	if !found {
		d.send(ch, "You can't put anything in there.\r\n")
	}
}

func (d *CommandDispatcher) cmdSacrifice(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Sacrifice what?\r\n")
		return
	}

	obj := FindObjInRoom(ch, args)
	if obj == nil {
		d.send(ch, "You don't see that here.\r\n")
		return
	}

	// Corpses can always be sacrificed, other items must be takeable
	isCorpse := obj.ItemType == types.ItemTypeCorpseNPC || obj.ItemType == types.ItemTypeCorpsePC
	if !isCorpse && !obj.CanTake() {
		d.send(ch, "You can't sacrifice that.\r\n")
		return
	}

	// Calculate gold reward (1 gold per 10 levels of item, minimum 1)
	gold := obj.Level / 10
	if gold < 1 {
		gold = 1
	}

	// Sacrifice the object
	ObjFromRoom(obj)
	ch.Gold += gold

	if gold == 1 {
		d.send(ch, fmt.Sprintf("The gods give you one gold coin for your sacrifice of %s.\r\n", obj.ShortDesc))
	} else {
		d.send(ch, fmt.Sprintf("The gods give you %d gold coins for your sacrifice of %s.\r\n", gold, obj.ShortDesc))
	}
	ActToRoom("$n sacrifices $p to the gods.", ch, nil, obj, d.Output)
}

// DonationPitVnum is the vnum of the donation pit (from merc.h: OBJ_VNUM_PIT = 3010)
const DonationPitVnum = 3010

func (d *CommandDispatcher) cmdDonate(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Donate what?\r\n")
		return
	}

	// Don't allow "all" or "all.item"
	if args == "all" || strings.HasPrefix(args, "all.") {
		d.send(ch, "One item at a time please.\r\n")
		return
	}

	// Find the donation pit in the game world
	var pit *types.Object
	if d.GameLoop != nil {
		for _, room := range d.GameLoop.GetAllRooms() {
			for _, obj := range room.Objects {
				if obj.ItemType == types.ItemTypePit && obj.Vnum == DonationPitVnum {
					pit = obj
					break
				}
			}
			if pit != nil {
				break
			}
		}
	}

	if pit == nil {
		d.send(ch, "I can't seem to find the donation pit!\r\n")
		return
	}

	// Find the item in inventory
	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	// Can't drop cursed items
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		d.send(ch, "You can't let go of it.\r\n")
		return
	}

	// Can't donate quest items
	if obj.ExtraFlags.Has(types.ItemQuest) {
		d.send(ch, "You can't donate a quest item.\r\n")
		return
	}

	// Can't donate melt-drop items
	if obj.ExtraFlags.Has(types.ItemMeltDrop) {
		d.send(ch, "You have a feeling no one's going to want that.\r\n")
		return
	}

	// Can't donate trash
	if obj.ItemType == types.ItemTypeTrash {
		d.send(ch, "The donation pit is not a trash can.\r\n")
		return
	}

	// Move object from inventory to pit
	ch.RemoveInventory(obj)
	pit.AddContent(obj)

	d.send(ch, fmt.Sprintf("You donate %s.\r\n", obj.ShortDesc))
	ActToRoom("$n donates $p.", ch, nil, obj, d.Output)
}

// wearLocationForItem determines which wear location to use for an item
func wearLocationForItem(obj *types.Object) types.WearLocation {
	// Light is always held in light slot
	if obj.ItemType == types.ItemTypeLight {
		return types.WearLocLight
	}

	// Check wear flags and return appropriate location
	if obj.WearFlags.Has(types.WearFinger) {
		return types.WearLocFingerL
	}
	if obj.WearFlags.Has(types.WearNeck) {
		return types.WearLocNeck1
	}
	if obj.WearFlags.Has(types.WearBody) {
		return types.WearLocBody
	}
	if obj.WearFlags.Has(types.WearHead) {
		return types.WearLocHead
	}
	if obj.WearFlags.Has(types.WearLegs) {
		return types.WearLocLegs
	}
	if obj.WearFlags.Has(types.WearFeet) {
		return types.WearLocFeet
	}
	if obj.WearFlags.Has(types.WearHands) {
		return types.WearLocHands
	}
	if obj.WearFlags.Has(types.WearArms) {
		return types.WearLocArms
	}
	if obj.WearFlags.Has(types.WearShield) {
		return types.WearLocShield
	}
	if obj.WearFlags.Has(types.WearAbout) {
		return types.WearLocAbout
	}
	if obj.WearFlags.Has(types.WearWaist) {
		return types.WearLocWaist
	}
	if obj.WearFlags.Has(types.WearWrist) {
		return types.WearLocWristL
	}
	if obj.WearFlags.Has(types.WearWield) {
		return types.WearLocWield
	}
	if obj.WearFlags.Has(types.WearHold) {
		return types.WearLocHold
	}
	if obj.WearFlags.Has(types.WearFloat) {
		return types.WearLocFloat
	}
	if obj.WearFlags.Has(types.WearFace) {
		return types.WearLocFace
	}

	return types.WearLocNone
}

// wearLocationName returns the display name for a wear location
func wearLocationName(loc types.WearLocation) string {
	names := map[types.WearLocation]string{
		types.WearLocLight:     "as a light",
		types.WearLocFingerL:   "on your left finger",
		types.WearLocFingerR:   "on your right finger",
		types.WearLocNeck1:     "around your neck",
		types.WearLocNeck2:     "around your neck",
		types.WearLocBody:      "on your torso",
		types.WearLocHead:      "on your head",
		types.WearLocLegs:      "on your legs",
		types.WearLocFeet:      "on your feet",
		types.WearLocHands:     "on your hands",
		types.WearLocArms:      "on your arms",
		types.WearLocShield:    "as a shield",
		types.WearLocAbout:     "about your body",
		types.WearLocWaist:     "around your waist",
		types.WearLocWristL:    "on your left wrist",
		types.WearLocWristR:    "on your right wrist",
		types.WearLocWield:     "wielded",
		types.WearLocHold:      "held in your hands",
		types.WearLocFloat:     "floating nearby",
		types.WearLocSecondary: "as a secondary weapon",
		types.WearLocFace:      "on your face",
	}
	if name, ok := names[loc]; ok {
		return name
	}
	return "somewhere"
}

func (d *CommandDispatcher) cmdWear(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Wear what?\r\n")
		return
	}

	// Check for "wear all"
	if args == "all" {
		d.wearAll(ch)
		return
	}

	// Find the item in inventory
	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	d.wearObj(ch, obj)
}

// wearObjWithReplace attempts to wear an object, optionally replacing existing equipment
// If allowReplace is true, existing equipment will be removed and put in inventory
// If allowReplace is false, the wear will fail if the slot is occupied
func (d *CommandDispatcher) wearObjWithReplace(ch *types.Character, obj *types.Object, allowReplace bool) bool {
	// Check if already worn
	if obj.WearLoc != types.WearLocNone {
		d.send(ch, "You are already wearing that.\r\n")
		return false
	}

	// Check level
	if obj.Level > ch.Level {
		d.send(ch, fmt.Sprintf("You must be level %d to use this object.\r\n", obj.Level))
		return false
	}

	// Determine wear location
	loc := wearLocationForItem(obj)
	if loc == types.WearLocNone {
		d.send(ch, "You can't wear that.\r\n")
		return false
	}

	// Handle dual slots (fingers, wrists, neck)
	switch loc {
	case types.WearLocFingerL:
		if ch.GetEquipment(types.WearLocFingerL) != nil {
			if ch.GetEquipment(types.WearLocFingerR) != nil {
				if allowReplace {
					// Replace left finger
					loc = types.WearLocFingerL
				} else {
					return false // Skip silently for wear all
				}
			} else {
				loc = types.WearLocFingerR
			}
		}
	case types.WearLocNeck1:
		if ch.GetEquipment(types.WearLocNeck1) != nil {
			if ch.GetEquipment(types.WearLocNeck2) != nil {
				if allowReplace {
					// Replace neck1
					loc = types.WearLocNeck1
				} else {
					return false // Skip silently for wear all
				}
			} else {
				loc = types.WearLocNeck2
			}
		}
	case types.WearLocWristL:
		if ch.GetEquipment(types.WearLocWristL) != nil {
			if ch.GetEquipment(types.WearLocWristR) != nil {
				if allowReplace {
					// Replace left wrist
					loc = types.WearLocWristL
				} else {
					return false // Skip silently for wear all
				}
			} else {
				loc = types.WearLocWristR
			}
		}
	default:
		// Check if slot is already occupied
		if ch.GetEquipment(loc) != nil && !allowReplace {
			return false // Skip silently for wear all
		}
	}

	// Remove existing equipment if present
	if existing := ch.GetEquipment(loc); existing != nil {
		// Remove affects from the old item
		removeObjectAffects(ch, existing)
		// Unequip and put back in inventory
		ch.Unequip(loc)
		ch.AddInventory(existing)
		d.send(ch, fmt.Sprintf("You remove %s.\r\n", existing.ShortDesc))
		ActToRoom("$n stops using $p.", ch, nil, existing, d.Output)
	}

	// Equip the item
	ch.RemoveInventory(obj)
	ch.Equip(obj, loc)

	// Apply object affects to character
	applyObjectAffects(ch, obj)

	d.send(ch, fmt.Sprintf("You wear %s %s.\r\n", obj.ShortDesc, wearLocationName(loc)))
	ActToRoom("$n wears $p.", ch, nil, obj, d.Output)
	return true
}

func (d *CommandDispatcher) wearObj(ch *types.Character, obj *types.Object) {
	d.wearObjWithReplace(ch, obj, true) // Direct wear allows replacement
}

func (d *CommandDispatcher) wearAll(ch *types.Character) {
	found := false
	inventory := make([]*types.Object, len(ch.Inventory))
	copy(inventory, ch.Inventory)

	for _, obj := range inventory {
		loc := wearLocationForItem(obj)
		if loc == types.WearLocNone {
			continue
		}
		if obj.Level > ch.Level {
			continue
		}

		// Try to wear it without replacing existing equipment
		if d.wearObjWithReplace(ch, obj, false) {
			found = true
		}
	}

	if !found {
		d.send(ch, "You have nothing else you can wear.\r\n")
	}
}

func (d *CommandDispatcher) cmdWield(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Wield what?\r\n")
		return
	}

	// Find the weapon in inventory
	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	// Check if it's a weapon
	if obj.ItemType != types.ItemTypeWeapon {
		d.send(ch, "That's not a weapon.\r\n")
		return
	}

	// Check if already worn
	if obj.WearLoc != types.WearLocNone {
		d.send(ch, "You are already using that.\r\n")
		return
	}

	// Check level
	if obj.Level > ch.Level {
		d.send(ch, fmt.Sprintf("You must be level %d to use this weapon.\r\n", obj.Level))
		return
	}

	// Check if primary slot is occupied
	if existing := ch.GetEquipment(types.WearLocWield); existing != nil {
		d.send(ch, fmt.Sprintf("You're already wielding %s.\r\n", existing.ShortDesc))
		return
	}

	// Equip the weapon
	ch.RemoveInventory(obj)
	ch.Equip(obj, types.WearLocWield)

	// Apply object affects to character
	applyObjectAffects(ch, obj)

	d.send(ch, fmt.Sprintf("You wield %s.\r\n", obj.ShortDesc))
	ActToRoom("$n wields $p.", ch, nil, obj, d.Output)
}

// cmdSecond equips a secondary weapon for dual wielding
func (d *CommandDispatcher) cmdSecond(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Wield what in your off-hand?\r\n")
		return
	}

	// Check for dual wield skill
	dualWieldSkill := 0
	if ch.PCData != nil {
		dualWieldSkill = ch.PCData.Learned["dual wield"]
	}
	if ch.IsNPC() {
		dualWieldSkill = 50 + ch.Level // NPCs get reasonable dual wield
	}

	if dualWieldSkill <= 0 {
		d.send(ch, "You don't know how to dual wield.\r\n")
		return
	}

	// Must have a primary weapon first
	primary := ch.GetEquipment(types.WearLocWield)
	if primary == nil {
		d.send(ch, "You need to wield a primary weapon first.\r\n")
		return
	}

	// Find the weapon in inventory
	obj := FindObjInInventory(ch, args)
	if obj == nil {
		d.send(ch, "You don't have that.\r\n")
		return
	}

	// Check if it's a weapon
	if obj.ItemType != types.ItemTypeWeapon {
		d.send(ch, "That's not a weapon.\r\n")
		return
	}

	// Check if already worn
	if obj.WearLoc != types.WearLocNone {
		d.send(ch, "You are already using that.\r\n")
		return
	}

	// Check level
	if obj.Level > ch.Level {
		d.send(ch, fmt.Sprintf("You must be level %d to use this weapon.\r\n", obj.Level))
		return
	}

	// Check if secondary slot is occupied
	if existing := ch.GetEquipment(types.WearLocSecondary); existing != nil {
		d.send(ch, fmt.Sprintf("You're already wielding %s in your off-hand.\r\n", existing.ShortDesc))
		return
	}

	// Cannot hold item and dual wield
	if ch.GetEquipment(types.WearLocHold) != nil {
		d.send(ch, "You cannot dual wield while holding something.\r\n")
		return
	}

	// Cannot use shield and dual wield
	if ch.GetEquipment(types.WearLocShield) != nil {
		d.send(ch, "You cannot dual wield while using a shield.\r\n")
		return
	}

	// Equip the weapon
	ch.RemoveInventory(obj)
	ch.Equip(obj, types.WearLocSecondary)

	// Apply object affects to character
	applyObjectAffects(ch, obj)

	d.send(ch, fmt.Sprintf("You wield %s in your off-hand.\r\n", obj.ShortDesc))
	ActToRoom("$n wields $p in $s off-hand.", ch, nil, obj, d.Output)
}

func (d *CommandDispatcher) cmdRemove(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Remove what?\r\n")
		return
	}

	// Check for "remove all"
	if args == "all" {
		d.removeAll(ch)
		return
	}

	// Find the item - check equipment first
	var obj *types.Object
	var loc types.WearLocation = types.WearLocNone

	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		eq := ch.GetEquipment(i)
		if eq != nil && nameMatches(eq.Name, args) {
			obj = eq
			loc = i
			break
		}
	}

	if obj == nil {
		d.send(ch, "You're not wearing that.\r\n")
		return
	}

	// Check for cursed items
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		d.send(ch, "You can't remove it.\r\n")
		return
	}

	// Remove object affects from character
	removeObjectAffects(ch, obj)

	// Unequip the item
	ch.Unequip(loc)
	ch.AddInventory(obj)

	d.send(ch, fmt.Sprintf("You stop using %s.\r\n", obj.ShortDesc))
	ActToRoom("$n stops using $p.", ch, nil, obj, d.Output)
}

func (d *CommandDispatcher) removeAll(ch *types.Character) {
	found := false

	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		obj := ch.GetEquipment(i)
		if obj == nil {
			continue
		}

		// Skip cursed items
		if obj.ExtraFlags.Has(types.ItemNoDrop) {
			continue
		}

		// Remove object affects from character
		removeObjectAffects(ch, obj)

		// Unequip the item
		ch.Unequip(i)
		ch.AddInventory(obj)

		d.send(ch, fmt.Sprintf("You stop using %s.\r\n", obj.ShortDesc))
		ActToRoom("$n stops using $p.", ch, nil, obj, d.Output)
		found = true
	}

	if !found {
		d.send(ch, "You aren't using anything.\r\n")
	}
}

// checkQuestItemGet checks if picking up an item updates quest progress
func (d *CommandDispatcher) checkQuestItemGet(ch *types.Character, obj *types.Object) {
	if d.Quests == nil || ch.PCData == nil || obj == nil {
		return
	}

	if d.Quests.OnItemGet(ch, obj) {
		d.send(ch, "{YQuest progress updated!{x\r\n")
	}
}

// checkQuestRoomEnter checks if entering a room updates quest progress
func (d *CommandDispatcher) checkQuestRoomEnter(ch *types.Character, room *types.Room) {
	if d.Quests == nil || ch.PCData == nil || room == nil {
		return
	}

	if d.Quests.OnRoomEnter(ch, room) {
		d.send(ch, "{YQuest completed! You have explored the area.{x\r\n")
	}
}
