package magic

import (
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// ObjectFinderFunc searches the world for objects matching a name
type ObjectFinderFunc func(name string, level int) []*types.Object

// ObjectToRoomFunc places an object in a room
type ObjectToRoomFunc func(obj *types.Object, room *types.Room)

// OnSpellCastFunc is a callback when a spell is cast (for metrics)
type OnSpellCastFunc func()

// WeatherControlFunc changes the weather (positive = better, negative = worse)
type WeatherControlFunc func(change int)

// CharacterToRoomFunc moves a character to a room
type CharacterToRoomFunc func(ch *types.Character, room *types.Room)

// RoomFinderFunc finds a room by vnum
type RoomFinderFunc func(vnum int) *types.Room

// PortalCreatorFunc creates a portal object linking two rooms
type PortalCreatorFunc func(srcRoom, destRoom *types.Room, duration int, twoWay bool) *types.Object

// PetCreatorFunc creates a pet mobile and returns it
// petType: "zombie", "animate", or "demon"
// sourceObj: the object used to create the pet (corpse for zombie, body part for animate, nil for demon)
// Returns the created pet character or nil on failure
type PetCreatorFunc func(master *types.Character, petType string, sourceObj *types.Object) *types.Character

// CorpseFinderFunc finds a corpse in the room
type CorpseFinderFunc func(ch *types.Character) *types.Object

// ExtractObjectFunc removes an object from the game and handles its contents
type ExtractObjectFunc func(obj *types.Object)

// CheckImproveFunc is called to potentially improve a skill/spell proficiency
type CheckImproveFunc func(ch *types.Character, skillName string, success bool, multiplier int)

// InvObjectFinderFunc finds an object in a character's inventory by name
type InvObjectFinderFunc func(ch *types.Character, name string) *types.Object

// IdentifyOutputFunc displays identify information for an object
type IdentifyOutputFunc func(ch *types.Character, obj *types.Object)

// MagicSystem manages spell casting and effects
type MagicSystem struct {
	Registry        *SpellRegistry
	Output          func(ch *types.Character, msg string)
	ObjectFinder    ObjectFinderFunc    // For locate object spell
	ObjectToRoom    ObjectToRoomFunc    // For create food/spring spells
	OnSpellCast     OnSpellCastFunc     // Called when a spell is successfully cast (for metrics)
	WeatherControl  WeatherControlFunc  // For control weather spell
	CharacterToRoom CharacterToRoomFunc // For teleport/summon/gate spells
	RoomFinder      RoomFinderFunc      // For finding rooms by vnum
	PortalCreator   PortalCreatorFunc   // For portal/nexus spells
	PetCreator      PetCreatorFunc      // For animate/resurrect/conjure spells
	CorpseFinder    CorpseFinderFunc    // For finding corpses in room
	ExtractObject   ExtractObjectFunc   // For removing objects from the game
	CheckImprove    CheckImproveFunc    // For improving spell proficiency
	InvObjectFinder InvObjectFinderFunc // For finding objects in inventory (identify spell)
	IdentifyOutput  IdentifyOutputFunc  // For displaying identify spell output
}

// NewMagicSystem creates a new magic system with default spells
func NewMagicSystem() *MagicSystem {
	return &MagicSystem{
		Registry: DefaultSpells(),
	}
}

// Cast attempts to cast a spell
func (m *MagicSystem) Cast(caster *types.Character, spellName string, targetArg string, findTarget TargetFinder) bool {
	// Find the spell
	spell := m.Registry.FindByPrefix(strings.ToLower(spellName))
	if spell == nil {
		m.send(caster, "You don't know any spell by that name.\r\n")
		return false
	}

	// Check if caster can cast this spell
	if !spell.CanCast(caster) {
		if caster.Mana < spell.ManaCost {
			m.send(caster, "You don't have enough mana.\r\n")
		} else if caster.Position < spell.MinPosition {
			m.send(caster, "You can't concentrate enough in your current position.\r\n")
		} else {
			m.send(caster, "You haven't learned that spell yet.\r\n")
		}
		return false
	}

	// Resolve target
	target, err := m.resolveTarget(caster, spell, targetArg, findTarget)
	if err != "" {
		m.send(caster, err)
		return false
	}

	// Deduct mana (always costs mana even if spell fizzles)
	mana := spell.GetManaCost(caster, caster.Level)
	caster.Mana -= mana

	// Add lag
	caster.Wait += spell.Beats

	// Check spell proficiency - can fizzle based on skill level
	// Cap at 99% so even masters have a 1% chance to fail (like rolling 1 on d20)
	proficiency := 0
	if !caster.IsNPC() && caster.PCData != nil && caster.PCData.Learned != nil {
		proficiency = caster.PCData.Learned[spell.Name]
		if proficiency > 99 {
			proficiency = 99
		}
	} else {
		proficiency = 99 // NPCs almost always succeed (1% fail chance)
	}

	// Roll against proficiency - spell fizzles if roll > proficiency
	roll := combat.NumberPercent()
	if roll > proficiency {
		m.send(caster, "You lost your concentration.\r\n")
		// Chance to improve spell proficiency on failure
		if m.CheckImprove != nil {
			m.CheckImprove(caster, spell.Name, false, 1)
		}
		return false
	}

	// Cast the spell
	success := spell.Func(caster, caster.Level, target)

	// Chance to improve spell proficiency on success
	if success && m.CheckImprove != nil {
		m.CheckImprove(caster, spell.Name, true, 1)
	}

	// Record spell cast for metrics
	if success && m.OnSpellCast != nil {
		m.OnSpellCast()
	}

	// Handle special spells that need system access
	if success && spell.Slot == SlotLocateObject && targetArg != "" {
		m.handleLocateObject(caster, targetArg, caster.Level)
	}

	if success && spell.Slot == 12 { // Create food slot
		m.handleCreateFood(caster, caster.Level)
	}

	if success && spell.Slot == 80 { // Create spring slot
		m.handleCreateSpring(caster, caster.Level)
	}

	if spell.Slot == 413 { // Control weather slot
		m.handleControlWeather(caster, targetArg, caster.Level)
		// Skip success message - handleControlWeather sends its own
		return true
	}

	// Handle portal/nexus spells
	if success && spell.Slot == SlotPortal && target != nil {
		if victim, ok := target.(*types.Character); ok && victim.InRoom != nil {
			m.handlePortal(caster, victim.InRoom, caster.Level, false)
		}
	}

	if success && spell.Slot == SlotNexus && target != nil {
		if victim, ok := target.(*types.Character); ok && victim.InRoom != nil {
			m.handlePortal(caster, victim.InRoom, caster.Level, true)
		}
	}

	// Handle floating disc spell
	if success && spell.Slot == SlotFloatingDisc {
		m.handleFloatingDisc(caster, caster.Level)
	}

	// Handle create rose spell
	if success && spell.Slot == SlotCreateRose {
		m.handleCreateRose(caster)
	}

	// Handle identify spell
	if success && spell.Slot == SlotIdentify && target != nil {
		if obj, ok := target.(*types.Object); ok && m.IdentifyOutput != nil {
			m.IdentifyOutput(caster, obj)
		}
	}

	// Handle continual light spell creating light ball
	if success && spell.Slot == SlotContinualLight && target == nil {
		m.handleContinualLight(caster, caster.Level)
	}

	// Handle pet creation spells
	if spell.Slot == SlotAnimate && target != nil {
		if obj, ok := target.(*types.Object); ok {
			success = m.handleAnimate(caster, obj)
		}
	}

	if spell.Slot == SlotResurrect {
		success = m.handleResurrect(caster, caster.Level)
	}

	if spell.Slot == SlotConjure {
		success = m.handleConjure(caster, caster.Level)
	}

	if spell.Slot == SlotVoodoo && success {
		// Get the held body part (spellVoodoo already validated it)
		if heldItem := caster.GetEquipment(types.WearLocHold); heldItem != nil {
			success = m.handleVoodoo(caster, heldItem)
		}
	}

	// Generate messages
	m.spellMessage(caster, spell, target, success)

	// If offensive spell, start combat
	if spell.Target == TargetCharOffense && success {
		if victim, ok := target.(*types.Character); ok && victim != caster {
			if !victim.InCombat() && victim.InRoom == caster.InRoom {
				combat.SetFighting(victim, caster)
			}
			if !caster.InCombat() {
				combat.SetFighting(caster, victim)
			}
		}
	}

	return success
}

// TargetFinder is a function that finds a target by name
type TargetFinder func(ch *types.Character, name string, offensive bool) interface{}

// resolveTarget finds the appropriate target for a spell
func (m *MagicSystem) resolveTarget(caster *types.Character, spell *Spell, arg string, findTarget TargetFinder) (interface{}, string) {
	switch spell.Target {
	case TargetIgnore:
		return nil, ""

	case TargetCharSelf:
		return caster, ""

	case TargetCharDefense:
		if arg == "" {
			return caster, ""
		}
		target := findTarget(caster, arg, false)
		if target == nil {
			return nil, "They aren't here.\r\n"
		}
		return target, ""

	case TargetCharOffense:
		if arg == "" {
			// Default to current fighting target
			if caster.Fighting != nil {
				return caster.Fighting, ""
			}
			return nil, "Cast the spell on whom?\r\n"
		}
		target := findTarget(caster, arg, true)
		if target == nil {
			return nil, "They aren't here.\r\n"
		}
		// Check if we can attack them
		if victim, ok := target.(*types.Character); ok {
			if combat.IsSafe(caster, victim) {
				return nil, "You cannot attack them.\r\n"
			}
		}
		return target, ""

	case TargetObjInv:
		if arg == "" {
			return nil, "Cast the spell on what?\r\n"
		}
		if m.InvObjectFinder == nil {
			return nil, "You don't have that.\r\n"
		}
		obj := m.InvObjectFinder(caster, arg)
		if obj == nil {
			return nil, "You don't have that.\r\n"
		}
		return obj, ""

	default:
		return nil, "Something went wrong.\r\n"
	}
}

// spellMessage generates appropriate messages for spell casting
func (m *MagicSystem) spellMessage(caster *types.Character, spell *Spell, target interface{}, success bool) {
	if !success {
		// Check if spell failed because already affected
		if victim, ok := target.(*types.Character); ok {
			if IsAffectedBy(victim, spell.Name) {
				if victim == caster {
					m.send(caster, "You are already affected by "+spell.Name+".\r\n")
				} else {
					m.send(caster, victim.Name+" is already affected by "+spell.Name+".\r\n")
				}
				return
			}
		}
		m.send(caster, "The spell fizzles.\r\n")
		return
	}

	victim, isChar := target.(*types.Character)

	switch spell.Target {
	case TargetCharSelf:
		// Use custom message if available, otherwise generic
		if spell.MsgSelf != "" {
			m.send(caster, spell.MsgSelf+"\r\n")
		} else {
			m.send(caster, "Ok.\r\n")
		}

	case TargetCharDefense:
		if isChar && victim != caster {
			// Casting on someone else
			m.send(caster, "Ok.\r\n")
			if spell.MsgVictim != "" {
				m.send(victim, spell.MsgVictim+"\r\n")
			} else {
				m.send(victim, caster.Name+" casts "+spell.Name+" on you.\r\n")
			}
		} else {
			// Casting on self
			if spell.MsgSelf != "" {
				m.send(caster, spell.MsgSelf+"\r\n")
			} else {
				m.send(caster, "Ok.\r\n")
			}
		}

	case TargetCharOffense:
		if isChar {
			if spell.NounDamage != "" {
				m.send(caster, "Your "+spell.NounDamage+" hits "+victim.Name+"!\r\n")
				m.send(victim, caster.Name+"'s "+spell.NounDamage+" hits you!\r\n")
			} else {
				m.send(caster, "You cast "+spell.Name+" at "+victim.Name+"!\r\n")
				m.send(victim, caster.Name+" casts "+spell.Name+" at you!\r\n")
			}

			// Check for death
			if victim.Hit <= 0 {
				m.send(caster, victim.Name+" is DEAD!\r\n")
				m.send(victim, "You have been KILLED!\r\n")
			}
		}

	case TargetObjInv:
		// Object targeting spells like identify handle their own output
		// Don't send generic "Ok." message

	default:
		m.send(caster, "Ok.\r\n")
	}
}

// send outputs a message to a character
func (m *MagicSystem) send(ch *types.Character, msg string) {
	if m.Output != nil {
		m.Output(ch, msg)
	}
}

// handleLocateObject finds objects in the world matching the given name
func (m *MagicSystem) handleLocateObject(caster *types.Character, name string, level int) {
	if m.ObjectFinder == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return
	}

	objects := m.ObjectFinder(name, level)
	if len(objects) == 0 {
		m.send(caster, "Nothing like that in heaven or earth.\r\n")
		return
	}

	// Show found objects
	found := 0
	for _, obj := range objects {
		if found >= 20+level/2 {
			break // Level-based limit on results
		}

		// Determine location description
		var location string
		if obj.CarriedBy != nil {
			location = "carried by " + obj.CarriedBy.Name
		} else if obj.InRoom != nil {
			location = "in " + obj.InRoom.Name
		} else if obj.InObject != nil {
			location = "in " + obj.InObject.ShortDesc
		} else {
			continue // Skip objects with unknown location
		}

		m.send(caster, obj.ShortDesc+" is "+location+".\r\n")
		found++
	}

	if found == 0 {
		m.send(caster, "Nothing like that in heaven or earth.\r\n")
	}
}

// handleCreateFood creates a food item and places it in the room
func (m *MagicSystem) handleCreateFood(caster *types.Character, level int) {
	if caster.InRoom == nil || m.ObjectToRoom == nil {
		return
	}

	// Create a magic mushroom
	food := types.NewObject(21, "a magic mushroom", types.ItemTypeFood)
	food.Name = "mushroom magic"
	food.LongDesc = "A magic mushroom is here."
	food.Values[0] = level/2 + 5 // Food value (hours of fullness)
	food.Timer = level * 2       // Hours until decay

	m.ObjectToRoom(food, caster.InRoom)
	m.send(caster, "A magic mushroom suddenly appears.\r\n")
}

// handleCreateSpring creates a fountain in the room
func (m *MagicSystem) handleCreateSpring(caster *types.Character, level int) {
	if caster.InRoom == nil || m.ObjectToRoom == nil {
		return
	}

	// Create a spring/fountain
	spring := types.NewObject(22, "a magical spring", types.ItemTypeFountain)
	spring.Name = "spring water magical"
	spring.LongDesc = "A magical spring flows from the ground here."
	spring.Values[0] = 1000    // Capacity
	spring.Values[1] = 1000    // Current amount
	spring.Values[2] = 0       // Water type
	spring.Timer = level/2 + 5 // Hours until dry

	m.ObjectToRoom(spring, caster.InRoom)
	m.send(caster, "A magical spring flows from the ground.\r\n")
}

// handleControlWeather changes the weather based on the argument
func (m *MagicSystem) handleControlWeather(caster *types.Character, arg string, level int) {
	arg = strings.ToLower(strings.TrimSpace(arg))

	if arg == "better" {
		if m.WeatherControl != nil {
			// Positive change makes weather better (increase pressure)
			change := dice(level/3, 4)
			m.WeatherControl(change)
		}
		m.send(caster, "Ok.\r\n")
	} else if arg == "worse" {
		if m.WeatherControl != nil {
			// Negative change makes weather worse (decrease pressure)
			change := -dice(level/3, 4)
			m.WeatherControl(change)
		}
		m.send(caster, "Ok.\r\n")
	} else {
		m.send(caster, "Do you want it to get better or worse?\r\n")
	}
}

// dice rolls number d size dice
func dice(number, size int) int {
	if size <= 0 || number <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < number; i++ {
		total += combat.Dice(1, size)
	}
	return total
}

// ProcessAffectTick handles affect decay for all characters
func (m *MagicSystem) ProcessAffectTick(characters []*types.Character) {
	for _, ch := range characters {
		messages := AffectTick(ch, m.Registry)
		for _, msg := range messages {
			m.send(ch, msg+"\r\n")
		}
	}
}

// handlePortal creates a portal to the destination room
func (m *MagicSystem) handlePortal(caster *types.Character, destRoom *types.Room, level int, twoWay bool) {
	if caster.InRoom == nil || destRoom == nil || m.PortalCreator == nil {
		return
	}

	// Don't create portal to same room
	if caster.InRoom == destRoom {
		m.send(caster, "You can't create a portal to here.\r\n")
		return
	}

	// Calculate duration based on level
	duration := level / 5

	// Create the portal
	portal := m.PortalCreator(caster.InRoom, destRoom, duration, twoWay)
	if portal == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return
	}

	if twoWay {
		m.send(caster, "A two-way nexus appears before you.\r\n")
	} else {
		m.send(caster, "A shimmering portal appears before you.\r\n")
	}
}

// handleFloatingDisc creates a floating disc container for the caster
func (m *MagicSystem) handleFloatingDisc(caster *types.Character, level int) {
	if caster.InRoom == nil || m.ObjectToRoom == nil {
		return
	}

	// Create a floating disc container
	disc := types.NewObject(23, "a floating disc", types.ItemTypeContainer)
	disc.Name = "disc floating"
	disc.LongDesc = "A floating disc hovers here."
	disc.Values[0] = level * 10 // Weight capacity
	disc.Values[1] = 0          // Container flags
	disc.Timer = level * 2      // Duration
	disc.Level = caster.Level
	// Don't set WearTake flag - disc cannot be picked up
	disc.WearFlags = 0

	// The disc follows the caster (set owner)
	disc.Owner = caster.Name

	m.ObjectToRoom(disc, caster.InRoom)
	m.send(caster, "A floating disc appears at your feet.\r\n")
}

// handleCreateRose creates a red rose
func (m *MagicSystem) handleCreateRose(caster *types.Character) {
	if caster.InRoom == nil || m.ObjectToRoom == nil {
		return
	}

	// Create a red rose
	rose := types.NewObject(24, "a red rose", types.ItemTypeTrash)
	rose.Name = "rose red"
	rose.LongDesc = "A beautiful red rose lies here."
	rose.Timer = 24 // Lasts one day

	m.ObjectToRoom(rose, caster.InRoom)
	m.send(caster, "A beautiful red rose appears.\r\n")
}

// handleContinualLight creates a ball of light
func (m *MagicSystem) handleContinualLight(caster *types.Character, level int) {
	if caster.InRoom == nil || m.ObjectToRoom == nil {
		return
	}

	// Create a ball of light
	light := types.NewObject(25, "a ball of light", types.ItemTypeLight)
	light.Name = "ball light continual"
	light.LongDesc = "A ball of glowing light hovers here."
	light.Values[2] = -1 // Infinite duration light
	light.Level = 1

	m.ObjectToRoom(light, caster.InRoom)
	m.send(caster, "A ball of light appears.\r\n")
}

// handleAnimate handles the animate spell - creates a follower from a body part
// NOTE: Unlike resurrect/conjure, animate does NOT create a pet - only a follower.
// The C code only calls add_follower() without setting ch->pet.
// This means a player can have multiple animated followers plus one pet.
func (m *MagicSystem) handleAnimate(caster *types.Character, bodyPart *types.Object) bool {
	if m.PetCreator == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// NOTE: No check for existing pet - animate creates followers, not pets
	// A player can have multiple animated body parts following them

	// Create the follower via callback
	follower := m.PetCreator(caster, "animate", bodyPart)
	if follower == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Remove the body part object
	if m.ExtractObject != nil {
		m.ExtractObject(bodyPart)
	}

	m.send(caster, follower.ShortDesc+" floats up and starts following you.\r\n")
	return true
}

// handleResurrect handles the resurrect spell - creates a zombie from a corpse
func (m *MagicSystem) handleResurrect(caster *types.Character, level int) bool {
	if m.PetCreator == nil || m.CorpseFinder == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Check if caster already has a pet
	if caster.Pet != nil {
		m.send(caster, "You already have a follower.\r\n")
		return false
	}

	// Find a corpse in the room
	corpse := m.CorpseFinder(caster)
	if corpse == nil {
		m.send(caster, "There's no corpse here.\r\n")
		return false
	}

	// Create the zombie pet via callback
	pet := m.PetCreator(caster, "zombie", corpse)
	if pet == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Extract the corpse (dropping its contents to the room first is handled by callback)
	if m.ExtractObject != nil {
		m.ExtractObject(corpse)
	}

	m.send(caster, pet.ShortDesc+" stands up and starts following you.\r\n")
	return true
}

// handleConjure handles the conjure spell - creates a demon follower
// Requires a demon stone to be held (unless immortal). The stone has charges
// in value[0] that are consumed. When charges run out, stone explodes.
func (m *MagicSystem) handleConjure(caster *types.Character, level int) bool {
	if m.PetCreator == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Check if caster already has a pet
	if caster.Pet != nil {
		m.send(caster, "You already have a follower.\r\n")
		return false
	}

	// Check for demon stone in held position (unless immortal)
	var demonStone *types.Object
	if !caster.IsImmortal() {
		heldItem := caster.GetEquipment(types.WearLocHold)
		if heldItem == nil || heldItem.ItemType != types.ItemTypeDemonStone {
			m.send(caster, "You lack the proper component for this spell.\r\n")
			return false
		}
		demonStone = heldItem

		// Check if stone has charges remaining
		if demonStone.Values[0] < 1 {
			// Stone explodes - no demon is summoned
			m.send(caster, "You draw upon the power of "+demonStone.ShortDesc+".\r\n")
			m.send(caster, "It flares brightly and explodes into dust.\r\n")
			// Extract the stone
			if m.ExtractObject != nil {
				m.ExtractObject(demonStone)
			}
			return false
		}
	}

	// Create the demon pet via callback
	pet := m.PetCreator(caster, "demon", demonStone)
	if pet == nil {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Decrement stone charges on successful summon
	if demonStone != nil {
		demonStone.Values[0]--
		if demonStone.Values[0] < 0 {
			demonStone.Values[0] = 0
		}
		m.send(caster, "You draw upon the power of "+demonStone.ShortDesc+".\r\n")
		m.send(caster, "It flares brightly and "+pet.ShortDesc+" appears.\r\n")
	} else {
		m.send(caster, pet.ShortDesc+" suddenly appears.\r\n")
	}

	return true
}

// handleVoodoo handles the voodoo spell - creates a voodoo doll from a body part
// The body part must be held and must be from a player (not NPC)
func (m *MagicSystem) handleVoodoo(caster *types.Character, bodyPart *types.Object) bool {
	// Validate body part is from a player (value[4] = 1 for player parts)
	if bodyPart.Values[4] == 0 {
		m.send(caster, "This body part is from a mobile.\r\n")
		return false
	}

	// Extract victim's name from the body part's name field
	// In C code: one_argument(bpart->name, name) gets first word
	victimName := ""
	if parts := strings.Fields(bodyPart.Name); len(parts) > 0 {
		victimName = parts[0]
	}
	if victimName == "" {
		m.send(caster, "The spell fizzles.\r\n")
		return false
	}

	// Create the voodoo doll
	doll := types.NewObject(VnumVoodooDoll, "a voodoo doll of "+victimName, types.ItemTypeTrash)
	doll.Name = victimName + " doll voodoo"
	doll.LongDesc = "A voodoo doll of " + victimName + " lies here."
	doll.Level = bodyPart.Level
	doll.WearFlags = types.WearTake | types.WearHold
	doll.ExtraFlags.Set(types.ItemMagic)

	// Unequip the body part first (it was in HOLD slot)
	caster.Unequip(types.WearLocHold)
	// Remove the body part from inventory
	caster.RemoveInventory(bodyPart)
	if m.ExtractObject != nil {
		m.ExtractObject(bodyPart)
	}

	// Give doll to caster and equip it
	caster.AddInventory(doll)
	doll.CarriedBy = caster
	caster.Equip(doll, types.WearLocHold)

	m.send(caster, bodyPart.ShortDesc+" morphs into a voodoo doll.\r\n")
	return true
}

// ObjectCast casts a spell from an object (scroll/wand/staff) at the given level
// Returns true if the spell was cast successfully
func (m *MagicSystem) ObjectCast(spellSlot int, level int, caster *types.Character, victim *types.Character, obj *types.Object) bool {
	if spellSlot <= 0 {
		return true // No spell to cast
	}

	spell := m.Registry.FindBySlot(spellSlot)
	if spell == nil {
		return false
	}

	// Determine target based on spell type
	var target interface{}
	switch spell.Target {
	case TargetIgnore:
		target = nil
	case TargetCharSelf:
		target = caster
	case TargetCharDefense:
		if victim != nil {
			target = victim
		} else {
			target = caster
		}
	case TargetCharOffense:
		if victim != nil {
			target = victim
		} else if caster.Fighting != nil {
			target = caster.Fighting
		} else {
			// No valid target for offensive spell
			return false
		}
	case TargetObjInv:
		target = obj
	default:
		target = nil
	}

	// Cast the spell
	success := spell.Func(caster, level, target)

	// Generate messages for successful casts
	if success && victim != nil && victim != caster {
		if spell.Target == TargetCharOffense {
			if spell.NounDamage != "" {
				m.send(victim, caster.Name+"'s "+spell.NounDamage+" hits you!\r\n")
			}
			// Start combat if not already fighting
			if !victim.InCombat() && victim.InRoom == caster.InRoom {
				combat.SetFighting(victim, caster)
			}
			if !caster.InCombat() {
				combat.SetFighting(caster, victim)
			}
		} else if spell.Target == TargetCharDefense {
			m.send(victim, "You feel a magical effect from "+caster.Name+".\r\n")
		}
	}

	return success
}
