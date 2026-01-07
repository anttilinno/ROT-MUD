package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Door commands: open, close, lock, unlock, pick

// findDoor finds a door by name or direction
func (d *CommandDispatcher) findDoor(ch *types.Character, arg string) (*types.Exit, types.Direction, string) {
	if ch.InRoom == nil {
		return nil, types.DirNorth, "You're not in a room.\r\n"
	}

	arg = strings.ToLower(arg)

	// Try to match as direction first
	directions := map[string]types.Direction{
		"north": types.DirNorth, "n": types.DirNorth,
		"east": types.DirEast, "e": types.DirEast,
		"south": types.DirSouth, "s": types.DirSouth,
		"west": types.DirWest, "w": types.DirWest,
		"up": types.DirUp, "u": types.DirUp,
		"down": types.DirDown, "d": types.DirDown,
	}

	if dir, ok := directions[arg]; ok {
		exit := ch.InRoom.GetExit(dir)
		if exit == nil {
			return nil, dir, "There's no exit in that direction.\r\n"
		}
		if !exit.IsDoor() {
			return nil, dir, "There's no door in that direction.\r\n"
		}
		return exit, dir, ""
	}

	// Try to match as a door keyword
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := ch.InRoom.GetExit(dir)
		if exit == nil || !exit.IsDoor() {
			continue
		}
		if exit.Keywords != "" && nameMatches(exit.Keywords, arg) {
			return exit, dir, ""
		}
	}

	return nil, types.DirNorth, "You don't see that here.\r\n"
}

// getReverseExit gets the exit in the target room pointing back to this room
func getReverseExit(fromRoom *types.Room, dir types.Direction) *types.Exit {
	exit := fromRoom.GetExit(dir)
	if exit == nil || exit.ToRoom == nil {
		return nil
	}

	reverseDir := dir.Reverse()
	reverseExit := exit.ToRoom.GetExit(reverseDir)

	// Verify it points back to our room
	if reverseExit != nil && reverseExit.ToRoom == fromRoom {
		return reverseExit
	}
	return nil
}

func (d *CommandDispatcher) cmdOpen(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Open what?\r\n")
		return
	}

	// Try to find a door first
	exit, dir, errMsg := d.findDoor(ch, args)
	if exit != nil {
		// Opening a door
		if !exit.IsClosed() {
			d.send(ch, "It's already open.\r\n")
			return
		}

		if exit.IsLocked() {
			d.send(ch, "It's locked.\r\n")
			return
		}

		// Check for NoClose flag (can't be opened either)
		if exit.Flags.Has(types.ExitNoClose) {
			d.send(ch, "You can't seem to open it.\r\n")
			return
		}

		// Open the door
		exit.Open()
		d.send(ch, "You open the door.\r\n")
		ActToRoom("$n opens the door.", ch, nil, nil, d.Output)

		// Open the reverse door too (if it exists)
		if reverseExit := getReverseExit(ch.InRoom, dir); reverseExit != nil && reverseExit.IsDoor() {
			reverseExit.Open()
			// Notify people in the other room
			for _, person := range exit.ToRoom.People {
				d.send(person, "The door opens.\r\n")
			}
		}
		return
	}

	// Try to find a container
	container := FindObjOnChar(ch, args)
	if container == nil {
		container = FindObjInRoom(ch, args)
	}

	if container == nil {
		d.send(ch, errMsg)
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if it's closeable
	if container.Values[1]&4 == 0 { // CONT_CLOSEABLE flag
		d.send(ch, "You can't open that.\r\n")
		return
	}

	// Check if already open
	if container.Values[1]&1 == 0 { // CONT_CLOSED flag not set
		d.send(ch, "It's already open.\r\n")
		return
	}

	// Check if locked
	if container.Values[1]&2 != 0 { // CONT_LOCKED flag
		d.send(ch, "It's locked.\r\n")
		return
	}

	// Open it
	container.Values[1] &^= 1 // Clear CONT_CLOSED
	d.send(ch, fmt.Sprintf("You open %s.\r\n", container.ShortDesc))
	ActToRoom("$n opens $p.", ch, nil, container, d.Output)
}

func (d *CommandDispatcher) cmdClose(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Close what?\r\n")
		return
	}

	// Try to find a door first
	exit, dir, errMsg := d.findDoor(ch, args)
	if exit != nil {
		// Closing a door
		if exit.IsClosed() {
			d.send(ch, "It's already closed.\r\n")
			return
		}

		// Check for NoClose flag
		if exit.Flags.Has(types.ExitNoClose) {
			d.send(ch, "You can't seem to close it.\r\n")
			return
		}

		// Close the door
		exit.Close()
		d.send(ch, "You close the door.\r\n")
		ActToRoom("$n closes the door.", ch, nil, nil, d.Output)

		// Close the reverse door too (if it exists)
		if reverseExit := getReverseExit(ch.InRoom, dir); reverseExit != nil && reverseExit.IsDoor() {
			reverseExit.Close()
			// Notify people in the other room
			for _, person := range exit.ToRoom.People {
				d.send(person, "The door closes.\r\n")
			}
		}
		return
	}

	// Try to find a container
	container := FindObjOnChar(ch, args)
	if container == nil {
		container = FindObjInRoom(ch, args)
	}

	if container == nil {
		d.send(ch, errMsg)
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if it's closeable
	if container.Values[1]&4 == 0 { // CONT_CLOSEABLE flag
		d.send(ch, "You can't close that.\r\n")
		return
	}

	// Check if already closed
	if container.Values[1]&1 != 0 { // CONT_CLOSED flag set
		d.send(ch, "It's already closed.\r\n")
		return
	}

	// Close it
	container.Values[1] |= 1 // Set CONT_CLOSED
	d.send(ch, fmt.Sprintf("You close %s.\r\n", container.ShortDesc))
	ActToRoom("$n closes $p.", ch, nil, container, d.Output)
}

func (d *CommandDispatcher) cmdLock(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Lock what?\r\n")
		return
	}

	// Try to find a door first
	exit, dir, errMsg := d.findDoor(ch, args)
	if exit != nil {
		// Locking a door
		if !exit.IsClosed() {
			d.send(ch, "It's not closed.\r\n")
			return
		}

		if exit.IsLocked() {
			d.send(ch, "It's already locked.\r\n")
			return
		}

		// Check for NoLock flag
		if exit.Flags.Has(types.ExitNoLock) {
			d.send(ch, "You can't lock that.\r\n")
			return
		}

		// Check for key
		if exit.Key > 0 {
			keyObj := d.findKey(ch, exit.Key)
			if keyObj == nil {
				d.send(ch, "You don't have the key.\r\n")
				return
			}
		}

		// Lock the door
		exit.Lock()
		d.send(ch, "You lock the door.\r\n")
		ActToRoom("$n locks the door.", ch, nil, nil, d.Output)

		// Lock the reverse door too (if it exists)
		if reverseExit := getReverseExit(ch.InRoom, dir); reverseExit != nil && reverseExit.IsDoor() {
			reverseExit.Lock()
		}
		return
	}

	// Try to find a container
	container := FindObjOnChar(ch, args)
	if container == nil {
		container = FindObjInRoom(ch, args)
	}

	if container == nil {
		d.send(ch, errMsg)
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if already closed
	if container.Values[1]&1 == 0 { // Not closed
		d.send(ch, "It's not closed.\r\n")
		return
	}

	// Check if already locked
	if container.Values[1]&2 != 0 { // Already locked
		d.send(ch, "It's already locked.\r\n")
		return
	}

	// Check for key
	keyVnum := container.Values[2]
	if keyVnum > 0 {
		keyObj := d.findKey(ch, keyVnum)
		if keyObj == nil {
			d.send(ch, "You don't have the key.\r\n")
			return
		}
	}

	// Lock it
	container.Values[1] |= 2 // Set CONT_LOCKED
	d.send(ch, fmt.Sprintf("You lock %s.\r\n", container.ShortDesc))
	ActToRoom("$n locks $p.", ch, nil, container, d.Output)
}

func (d *CommandDispatcher) cmdUnlock(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Unlock what?\r\n")
		return
	}

	// Try to find a door first
	exit, dir, errMsg := d.findDoor(ch, args)
	if exit != nil {
		// Unlocking a door
		if !exit.IsLocked() {
			d.send(ch, "It's not locked.\r\n")
			return
		}

		// Check for key
		if exit.Key > 0 {
			keyObj := d.findKey(ch, exit.Key)
			if keyObj == nil {
				d.send(ch, "You don't have the key.\r\n")
				return
			}
		}

		// Unlock the door
		exit.Unlock()
		d.send(ch, "You unlock the door.\r\n")
		ActToRoom("$n unlocks the door.", ch, nil, nil, d.Output)

		// Unlock the reverse door too (if it exists)
		if reverseExit := getReverseExit(ch.InRoom, dir); reverseExit != nil && reverseExit.IsDoor() {
			reverseExit.Unlock()
		}
		return
	}

	// Try to find a container
	container := FindObjOnChar(ch, args)
	if container == nil {
		container = FindObjInRoom(ch, args)
	}

	if container == nil {
		d.send(ch, errMsg)
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not a container.\r\n")
		return
	}

	// Check if locked
	if container.Values[1]&2 == 0 { // Not locked
		d.send(ch, "It's not locked.\r\n")
		return
	}

	// Check for key
	keyVnum := container.Values[2]
	if keyVnum > 0 {
		keyObj := d.findKey(ch, keyVnum)
		if keyObj == nil {
			d.send(ch, "You don't have the key.\r\n")
			return
		}
	}

	// Unlock it
	container.Values[1] &^= 2 // Clear CONT_LOCKED
	d.send(ch, fmt.Sprintf("You unlock %s.\r\n", container.ShortDesc))
	ActToRoom("$n unlocks $p.", ch, nil, container, d.Output)
}

// findKey finds a key by vnum in character's inventory/equipment
func (d *CommandDispatcher) findKey(ch *types.Character, keyVnum int) *types.Object {
	// Check inventory
	for _, obj := range ch.Inventory {
		if obj.Vnum == keyVnum && obj.ItemType == types.ItemTypeKey {
			return obj
		}
	}

	// Check equipment
	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		obj := ch.GetEquipment(i)
		if obj != nil && obj.Vnum == keyVnum && obj.ItemType == types.ItemTypeKey {
			return obj
		}
	}

	return nil
}

func (d *CommandDispatcher) cmdPick(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Pick what?\r\n")
		return
	}

	// Try to find a door first
	exit, dir, errMsg := d.findDoor(ch, args)
	if exit != nil {
		// Picking a lock on a door
		if !exit.IsLocked() {
			d.send(ch, "It's not locked.\r\n")
			return
		}

		// Check for pickproof
		if exit.Flags.Has(types.ExitPickproof) {
			d.send(ch, "You can't seem to pick this lock.\r\n")
			return
		}

		// Calculate pick chance based on dex and lock difficulty
		chance := 50 + (ch.GetStat(types.StatDex)-15)*3

		// Adjust for lock difficulty
		if exit.Flags.Has(types.ExitEasy) {
			chance += 20
		} else if exit.Flags.Has(types.ExitHard) {
			chance -= 20
		} else if exit.Flags.Has(types.ExitInfuriating) {
			chance -= 40
		}

		// Roll
		if numberPercent() > chance {
			d.send(ch, "You failed to pick the lock.\r\n")
			return
		}

		// Success!
		exit.Unlock()
		d.send(ch, "You pick the lock!\r\n")
		ActToRoom("$n picks the lock.", ch, nil, nil, d.Output)

		// Unlock the reverse door too
		if reverseExit := getReverseExit(ch.InRoom, dir); reverseExit != nil && reverseExit.IsDoor() {
			reverseExit.Unlock()
		}
		return
	}

	// Try to find a container
	container := FindObjOnChar(ch, args)
	if container == nil {
		container = FindObjInRoom(ch, args)
	}

	if container == nil {
		d.send(ch, errMsg)
		return
	}

	if container.ItemType != types.ItemTypeContainer {
		d.send(ch, "That's not something you can pick.\r\n")
		return
	}

	// Check if locked
	if container.Values[1]&2 == 0 { // Not locked
		d.send(ch, "It's not locked.\r\n")
		return
	}

	// Calculate pick chance
	chance := 50 + (ch.GetStat(types.StatDex)-15)*3

	if numberPercent() > chance {
		d.send(ch, "You failed to pick the lock.\r\n")
		return
	}

	// Success!
	container.Values[1] &^= 2 // Clear CONT_LOCKED
	d.send(ch, fmt.Sprintf("You pick the lock on %s!\r\n", container.ShortDesc))
	ActToRoom("$n picks the lock on $p.", ch, nil, container, d.Output)
}

// numberPercent returns a random number from 1 to 100
func numberPercent() int {
	return combat.NumberPercent()
}
