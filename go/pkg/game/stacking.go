package game

import (
	"rotmud/pkg/types"
)

// CanStackObjects returns true if two objects are identical and can be stacked together.
// Objects can stack if they have the same vnum, item type, no unique affects/enchantments,
// and are not containers.
func CanStackObjects(obj1, obj2 *types.Object) bool {
	if obj1 == nil || obj2 == nil {
		return false
	}

	// Must have the same vnum (template ID)
	if obj1.Vnum != obj2.Vnum {
		return false
	}

	// Must be the same item type
	if obj1.ItemType != obj2.ItemType {
		return false
	}

	// Containers cannot be stacked (bags, corpses, pits, etc.)
	switch obj1.ItemType {
	case types.ItemTypeContainer, types.ItemTypeCorpseNPC, types.ItemTypeCorpsePC, types.ItemTypePit:
		return false
	}

	// Enchanted objects cannot be stacked (they may have unique affects)
	if obj1.Enchanted || obj2.Enchanted {
		return false
	}

	// Objects with affects cannot be stacked (they might be unique)
	if len(obj1.Affects.All()) > 0 || len(obj2.Affects.All()) > 0 {
		return false
	}

	// Objects with different conditions shouldn't stack
	if obj1.Condition != obj2.Condition {
		return false
	}

	// Objects with timers shouldn't stack (they decay independently)
	if obj1.Timer != -1 || obj2.Timer != -1 {
		return false
	}

	// Objects with owner restrictions shouldn't stack
	if obj1.Owner != "" || obj2.Owner != "" {
		return false
	}

	// Quest items shouldn't stack
	if obj1.ExtraFlags.Has(types.ItemQuest) || obj2.ExtraFlags.Has(types.ItemQuest) {
		return false
	}

	// Extra flags must match exactly
	if obj1.ExtraFlags != obj2.ExtraFlags {
		return false
	}

	return true
}

// ObjectStack represents a group of identical stacked objects
type ObjectStack struct {
	Object *types.Object // The representative object
	Count  int           // Number of identical objects
}

// StackObjects groups a list of objects into stacks of identical items.
// Returns a slice of ObjectStacks where each stack contains identical objects.
func StackObjects(objects []*types.Object) []ObjectStack {
	if len(objects) == 0 {
		return nil
	}

	var stacks []ObjectStack

	for _, obj := range objects {
		// Try to find an existing stack this object can join
		found := false
		for i := range stacks {
			if CanStackObjects(stacks[i].Object, obj) {
				stacks[i].Count++
				found = true
				break
			}
		}

		// If no matching stack found, create a new one
		if !found {
			stacks = append(stacks, ObjectStack{
				Object: obj,
				Count:  1,
			})
		}
	}

	return stacks
}

// FindStackedObjectInInventory finds objects in inventory that match a name,
// returning all matching objects (for quantity operations).
// If count > 0, returns up to that many objects. If count <= 0, returns all matches.
func FindStackedObjectsInInventory(ch *types.Character, name string, count int) []*types.Object {
	if ch == nil {
		return nil
	}

	var matches []*types.Object

	for _, obj := range ch.Inventory {
		if nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name) {
			matches = append(matches, obj)
			if count > 0 && len(matches) >= count {
				break
			}
		}
	}

	return matches
}

// FindStackedObjectsInRoom finds objects in room that match a name,
// returning all matching objects (for quantity operations).
// If count > 0, returns up to that many objects. If count <= 0, returns all matches.
func FindStackedObjectsInRoom(ch *types.Character, name string, count int) []*types.Object {
	if ch == nil || ch.InRoom == nil {
		return nil
	}

	var matches []*types.Object

	for _, obj := range ch.InRoom.Objects {
		if nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name) {
			matches = append(matches, obj)
			if count > 0 && len(matches) >= count {
				break
			}
		}
	}

	return matches
}
