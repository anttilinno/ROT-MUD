package game

import (
	"strings"

	"rotmud/pkg/types"
)

// CharToRoom moves a character to a room safely
// This handles removing them from their old room and adding to the new one
func CharToRoom(ch *types.Character, room *types.Room) {
	if ch == nil || room == nil {
		return
	}

	// Remove from old room if present
	if ch.InRoom != nil {
		ch.InRoom.RemovePerson(ch)
	}

	// Add to new room
	ch.InRoom = room
	room.AddPerson(ch)

	// Update light level if carrying a light
	for _, obj := range ch.Inventory {
		if obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
			room.Light++
		}
	}
	for i := 0; i < int(types.WearLocMax); i++ {
		obj := ch.Equipment[i]
		if obj != nil && obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
			room.Light++
		}
	}
}

// CharFromRoom removes a character from their current room
func CharFromRoom(ch *types.Character) {
	if ch == nil || ch.InRoom == nil {
		return
	}

	room := ch.InRoom

	// Update light level if carrying a light
	for _, obj := range ch.Inventory {
		if obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
			room.Light--
		}
	}
	for i := 0; i < int(types.WearLocMax); i++ {
		obj := ch.Equipment[i]
		if obj != nil && obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
			room.Light--
		}
	}

	room.RemovePerson(ch)
	ch.InRoom = nil
}

// ObjToRoom places an object in a room
func ObjToRoom(obj *types.Object, room *types.Room) {
	if obj == nil || room == nil {
		return
	}

	obj.InRoom = room
	obj.CarriedBy = nil
	obj.InObject = nil
	room.AddObject(obj)

	// Update light if it's a light source
	if obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
		room.Light++
	}
}

// ObjFromRoom removes an object from a room
func ObjFromRoom(obj *types.Object) {
	if obj == nil || obj.InRoom == nil {
		return
	}

	room := obj.InRoom

	// Update light if it's a light source
	if obj.ItemType == types.ItemTypeLight && obj.Values[2] > 0 {
		room.Light--
	}

	room.RemoveObject(obj)
	obj.InRoom = nil
}

// ObjToChar gives an object to a character (inventory)
func ObjToChar(obj *types.Object, ch *types.Character) {
	if obj == nil || ch == nil {
		return
	}

	obj.CarriedBy = ch
	obj.InRoom = nil
	obj.InObject = nil
	ch.AddInventory(obj)
}

// ObjFromChar removes an object from a character's inventory
func ObjFromChar(obj *types.Object) {
	if obj == nil || obj.CarriedBy == nil {
		return
	}

	ch := obj.CarriedBy
	ch.RemoveInventory(obj)
	obj.CarriedBy = nil
}

// FindCharInRoom finds a character in the room by name/keyword
func FindCharInRoom(ch *types.Character, name string) *types.Character {
	if ch == nil || ch.InRoom == nil {
		return nil
	}

	name = strings.ToLower(name)

	// Check "self" or "me"
	if name == "self" || name == "me" {
		return ch
	}

	for _, person := range ch.InRoom.People {
		if person == ch {
			continue
		}
		if nameMatches(person.Name, name) || keywordsMatch(person.ShortDesc, name) {
			return person
		}
	}

	return nil
}

// FindObjInInventory finds an object in a character's inventory by name/keyword
func FindObjInInventory(ch *types.Character, name string) *types.Object {
	if ch == nil {
		return nil
	}

	name = strings.ToLower(name)

	for _, obj := range ch.Inventory {
		if nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name) {
			return obj
		}
	}

	return nil
}

// FindObjInRoom finds an object in the room by name/keyword
func FindObjInRoom(ch *types.Character, name string) *types.Object {
	if ch == nil || ch.InRoom == nil {
		return nil
	}

	name = strings.ToLower(name)

	for _, obj := range ch.InRoom.Objects {
		if nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name) {
			return obj
		}
	}

	return nil
}

// FindObjOnChar finds an object in inventory or equipment
func FindObjOnChar(ch *types.Character, name string) *types.Object {
	if ch == nil {
		return nil
	}

	name = strings.ToLower(name)

	// Check inventory first
	obj := FindObjInInventory(ch, name)
	if obj != nil {
		return obj
	}

	// Check equipment
	for i := 0; i < int(types.WearLocMax); i++ {
		obj := ch.Equipment[i]
		if obj != nil && (nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name)) {
			return obj
		}
	}

	return nil
}

// FindEquipped finds an equipped object by name/keyword
func FindEquipped(ch *types.Character, name string) *types.Object {
	if ch == nil {
		return nil
	}

	name = strings.ToLower(name)

	for i := 0; i < int(types.WearLocMax); i++ {
		obj := ch.Equipment[i]
		if obj != nil && (nameMatches(obj.Name, name) || keywordsMatch(obj.ShortDesc, name)) {
			return obj
		}
	}

	return nil
}

// nameMatches checks if a name matches (case-insensitive, prefix matching)
func nameMatches(fullName, search string) bool {
	fullName = strings.ToLower(fullName)
	search = strings.ToLower(search)

	// Check exact match
	if fullName == search {
		return true
	}

	// Check prefix match
	if strings.HasPrefix(fullName, search) {
		return true
	}

	// Check individual keywords in the name
	for _, word := range strings.Fields(fullName) {
		if strings.HasPrefix(word, search) {
			return true
		}
	}

	return false
}

// keywordsMatch checks if any keyword in the description matches
func keywordsMatch(desc, search string) bool {
	desc = strings.ToLower(desc)
	search = strings.ToLower(search)

	for _, word := range strings.Fields(desc) {
		// Remove punctuation
		word = strings.Trim(word, ".,!?;:'\"")
		if strings.HasPrefix(word, search) {
			return true
		}
	}

	return false
}

// WearLocationName returns a human-readable name for an equipment slot
func WearLocationName(loc types.WearLocation) string {
	names := map[types.WearLocation]string{
		types.WearLocLight:     "<used as light>",
		types.WearLocFingerL:   "<worn on finger>",
		types.WearLocFingerR:   "<worn on finger>",
		types.WearLocNeck1:     "<worn around neck>",
		types.WearLocNeck2:     "<worn around neck>",
		types.WearLocBody:      "<worn on torso>",
		types.WearLocHead:      "<worn on head>",
		types.WearLocLegs:      "<worn on legs>",
		types.WearLocFeet:      "<worn on feet>",
		types.WearLocHands:     "<worn on hands>",
		types.WearLocArms:      "<worn on arms>",
		types.WearLocShield:    "<worn as shield>",
		types.WearLocAbout:     "<worn about body>",
		types.WearLocWaist:     "<worn about waist>",
		types.WearLocWristL:    "<worn on wrist>",
		types.WearLocWristR:    "<worn on wrist>",
		types.WearLocWield:     "<wielded>",
		types.WearLocHold:      "<held>",
		types.WearLocFloat:     "<floating nearby>",
		types.WearLocSecondary: "<secondary weapon>",
		types.WearLocFace:      "<worn on face>",
	}

	if name, ok := names[loc]; ok {
		return name
	}
	return "<unknown>"
}

// CanWearAt returns the wear location for an object, or WearLocNone if it can't be worn
func CanWearAt(obj *types.Object, ch *types.Character) types.WearLocation {
	if obj == nil {
		return types.WearLocNone
	}

	// Check each possible wear location
	if obj.WearFlags.Has(types.WearFinger) {
		if ch.Equipment[types.WearLocFingerL] == nil {
			return types.WearLocFingerL
		}
		if ch.Equipment[types.WearLocFingerR] == nil {
			return types.WearLocFingerR
		}
	}

	if obj.WearFlags.Has(types.WearNeck) {
		if ch.Equipment[types.WearLocNeck1] == nil {
			return types.WearLocNeck1
		}
		if ch.Equipment[types.WearLocNeck2] == nil {
			return types.WearLocNeck2
		}
	}

	if obj.WearFlags.Has(types.WearBody) && ch.Equipment[types.WearLocBody] == nil {
		return types.WearLocBody
	}

	if obj.WearFlags.Has(types.WearHead) && ch.Equipment[types.WearLocHead] == nil {
		return types.WearLocHead
	}

	if obj.WearFlags.Has(types.WearLegs) && ch.Equipment[types.WearLocLegs] == nil {
		return types.WearLocLegs
	}

	if obj.WearFlags.Has(types.WearFeet) && ch.Equipment[types.WearLocFeet] == nil {
		return types.WearLocFeet
	}

	if obj.WearFlags.Has(types.WearHands) && ch.Equipment[types.WearLocHands] == nil {
		return types.WearLocHands
	}

	if obj.WearFlags.Has(types.WearArms) && ch.Equipment[types.WearLocArms] == nil {
		return types.WearLocArms
	}

	if obj.WearFlags.Has(types.WearAbout) && ch.Equipment[types.WearLocAbout] == nil {
		return types.WearLocAbout
	}

	if obj.WearFlags.Has(types.WearWaist) && ch.Equipment[types.WearLocWaist] == nil {
		return types.WearLocWaist
	}

	if obj.WearFlags.Has(types.WearWrist) {
		if ch.Equipment[types.WearLocWristL] == nil {
			return types.WearLocWristL
		}
		if ch.Equipment[types.WearLocWristR] == nil {
			return types.WearLocWristR
		}
	}

	if obj.WearFlags.Has(types.WearShield) && ch.Equipment[types.WearLocShield] == nil {
		return types.WearLocShield
	}

	if obj.WearFlags.Has(types.WearHold) && ch.Equipment[types.WearLocHold] == nil {
		return types.WearLocHold
	}

	if obj.WearFlags.Has(types.WearFloat) && ch.Equipment[types.WearLocFloat] == nil {
		return types.WearLocFloat
	}

	if obj.WearFlags.Has(types.WearFace) && ch.Equipment[types.WearLocFace] == nil {
		return types.WearLocFace
	}

	return types.WearLocNone
}
