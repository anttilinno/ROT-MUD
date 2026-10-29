package game

import (
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/loader"
	"rotmud/pkg/magic"
	"rotmud/pkg/types"
)

// ResetSystem handles area reset execution
type ResetSystem struct {
	World  *loader.World
	Output OutputFunc
	// OnSpawn registers a freshly reset mob with the game loop, so it fights,
	// wanders and ticks like the mobs spawned at boot.
	OnSpawn func(*types.Character)
}

// NewResetSystem creates a new reset system
func NewResetSystem(world *loader.World) *ResetSystem {
	return &ResetSystem{
		World: world,
	}
}

// ResetArea resets a single area, spawning mobs and objects according to reset data
func (r *ResetSystem) ResetArea(area *types.Area) {
	for _, room := range area.RoomsInResetOrder() {
		r.resetRoom(room)
	}
	area.Age = 0
}

// resetRoom executes all resets for a single room
func (r *ResetSystem) resetRoom(room *types.Room) {
	// Close/lock doors back to their reset state, both sides (ROT reset_room)
	for dir, exit := range room.Exits {
		if exit == nil {
			continue
		}
		exit.Flags = exit.ResetFlags
		if exit.ToRoom != nil {
			if back := exit.ToRoom.GetExit(types.Direction(dir).Reverse()); back != nil {
				back.Flags = back.ResetFlags
			}
		}
	}

	shuffleExits(room)

	// Execute mob resets
	for _, reset := range room.MobResets {
		r.executeMobReset(room, &reset)
	}

	// Execute object resets
	for _, reset := range room.ObjResets {
		r.executeObjReset(room, &reset)
	}
}

// shuffleExits swaps the room's first RandomExits exits around, for mazes
// (ROM reset_room 'R'). Only this side moves; the rooms behind stay as built.
func shuffleExits(room *types.Room) {
	n := min(room.RandomExits, len(room.Exits))
	for d0 := 0; d0 < n-1; d0++ {
		d1 := combat.NumberRange(d0, n-1)
		room.Exits[d0], room.Exits[d1] = room.Exits[d1], room.Exits[d0]
	}
	for d, exit := range room.Exits[:n] {
		if exit != nil {
			exit.Direction = types.Direction(d)
		}
	}
}

// executeMobReset spawns a mob according to reset data
func (r *ResetSystem) executeMobReset(room *types.Room, reset *types.MobReset) {
	if r.World == nil {
		return
	}

	// Check if we've reached the max for this mob
	if reset.Max > 0 {
		count := r.countMobInWorld(reset.Vnum)
		if count >= reset.Max {
			return
		}
	}

	// Check if mob already exists in this room
	for _, ch := range room.People {
		if ch.IsNPC() && ch.MobVnum == reset.Vnum {
			return // Already have one here
		}
	}

	// Spawn the mob
	for i := 0; i < reset.Count; i++ {
		mob := r.World.CreateMobFromTemplate(reset.Vnum)
		if mob == nil {
			continue
		}

		// Place in room
		mob.InRoom = room
		room.People = append(room.People, mob)
		if r.OnSpawn != nil {
			r.OnSpawn(mob)
		}

		// Equip the mob with items
		for _, equipReset := range reset.Equips {
			r.executeEquipReset(mob, &equipReset)
		}
	}
}

// executeEquipReset gives an object to a mob
func (r *ResetSystem) executeEquipReset(mob *types.Character, reset *types.EquipReset) {
	if r.World == nil {
		return
	}

	// Check if we've reached the limit for this object
	if reset.Limit > 0 {
		count := r.countObjInWorld(reset.Vnum)
		if count >= reset.Limit {
			return
		}
	}

	// Create the object
	obj := r.CreateObjFromTemplate(reset.Vnum)
	if obj == nil {
		return
	}

	r.fillContainer(obj, reset.Contents)

	// Either put in inventory or equip
	if reset.InvOnly {
		obj.CarriedBy = mob
		mob.Inventory = append(mob.Inventory, obj)
	} else {
		// Equip at the specified wear location
		if reset.WearLoc >= 0 && int(reset.WearLoc) < len(mob.Equipment) {
			mob.Equipment[reset.WearLoc] = obj
			obj.WearLoc = reset.WearLoc
			obj.CarriedBy = mob
		} else {
			// Invalid wear location, put in inventory
			obj.CarriedBy = mob
			mob.Inventory = append(mob.Inventory, obj)
		}
	}
}

// executeObjReset spawns an object according to reset data
func (r *ResetSystem) executeObjReset(room *types.Room, reset *types.ObjReset) {
	if r.World == nil {
		return
	}

	// Check if object already exists in this room; a container still gets
	// its contents topped up, but not under players' noses (ROT 'P')
	for _, obj := range room.Objects {
		if obj.Vnum == reset.Vnum {
			if room.Area == nil || room.Area.NumPlayers == 0 {
				r.fillContainer(obj, reset.Contents)
			}
			return // Already have one here
		}
	}

	// Check if we've reached the max for this object
	if reset.Max > 0 && r.countObjInWorld(reset.Vnum) >= reset.Max {
		return
	}

	// Spawn the object
	for i := 0; i < reset.Count; i++ {
		obj := r.CreateObjFromTemplate(reset.Vnum)
		if obj == nil {
			continue
		}

		// Place in room
		obj.InRoom = room
		room.Objects = append(room.Objects, obj)
		r.fillContainer(obj, reset.Contents)
	}
}

// fillContainer tops a container up to its reset contents and closes and
// locks it again as built (ROM reset_room 'P').
func (r *ResetSystem) fillContainer(container *types.Object, contents []types.ObjReset) {
	if len(contents) == 0 {
		return
	}
	for _, reset := range contents {
		have := 0
		for _, obj := range container.Contents {
			if obj.Vnum == reset.Vnum {
				have++
			}
		}
		for ; have < reset.Count; have++ {
			if reset.Max > 0 && r.countObjInWorld(reset.Vnum) >= reset.Max {
				break
			}
			obj := r.CreateObjFromTemplate(reset.Vnum)
			if obj == nil {
				break
			}
			container.AddContent(obj)
			r.fillContainer(obj, reset.Contents)
		}
	}
	if tmpl := r.World.GetObjTemplate(container.Vnum); tmpl != nil && tmpl.Container != nil {
		container.Values[1] = containerFlagBits(tmpl.Container.Flags)
	}
}

// countMobInWorld counts how many instances of a mob vnum exist
func (r *ResetSystem) countMobInWorld(vnum int) int {
	count := 0
	for _, room := range r.World.Rooms {
		for _, ch := range room.People {
			if ch.IsNPC() && ch.MobVnum == vnum {
				count++
			}
		}
	}
	return count
}

// countObjInWorld counts how many instances of an object vnum exist in rooms,
// on characters, and inside their containers
func (r *ResetSystem) countObjInWorld(vnum int) int {
	count := 0
	for _, room := range r.World.Rooms {
		count += countObjs(room.Objects, vnum)
		for _, ch := range room.People {
			count += countObjs(ch.Inventory, vnum)
			count += countObjs(ch.Equipment[:], vnum)
		}
	}
	return count
}

func countObjs(objs []*types.Object, vnum int) int {
	count := 0
	for _, obj := range objs {
		if obj == nil {
			continue
		}
		if obj.Vnum == vnum {
			count++
		}
		count += countObjs(obj.Contents, vnum)
	}
	return count
}

// CreateObjFromTemplate creates an Object instance from an object template
func (r *ResetSystem) CreateObjFromTemplate(vnum int) *types.Object {
	tmpl := r.World.GetObjTemplate(vnum)
	if tmpl == nil {
		return nil
	}

	obj := types.NewObject(vnum, tmpl.ShortDesc, parseItemType(tmpl.ItemType))
	obj.Name = stringJoin(tmpl.Keywords)
	obj.LongDesc = tmpl.LongDesc
	obj.Level = tmpl.Level
	obj.Weight = tmpl.Weight
	obj.Cost = tmpl.Cost
	obj.Condition = tmpl.Condition
	if obj.Condition == 0 {
		obj.Condition = 100
	}
	obj.Material = tmpl.Material

	// Parse wear flags
	for _, flag := range tmpl.WearFlags {
		switch flag {
		case "take":
			obj.WearFlags.Set(types.WearTake)
		case "finger":
			obj.WearFlags.Set(types.WearFinger)
		case "neck":
			obj.WearFlags.Set(types.WearNeck)
		case "body":
			obj.WearFlags.Set(types.WearBody)
		case "head":
			obj.WearFlags.Set(types.WearHead)
		case "legs":
			obj.WearFlags.Set(types.WearLegs)
		case "feet":
			obj.WearFlags.Set(types.WearFeet)
		case "hands":
			obj.WearFlags.Set(types.WearHands)
		case "arms":
			obj.WearFlags.Set(types.WearArms)
		case "shield":
			obj.WearFlags.Set(types.WearShield)
		case "about":
			obj.WearFlags.Set(types.WearAbout)
		case "waist":
			obj.WearFlags.Set(types.WearWaist)
		case "wrist":
			obj.WearFlags.Set(types.WearWrist)
		case "wield":
			obj.WearFlags.Set(types.WearWield)
		case "hold":
			obj.WearFlags.Set(types.WearHold)
		case "float":
			obj.WearFlags.Set(types.WearFloat)
		}
	}

	// Parse extra flags
	for _, flag := range tmpl.ExtraFlags {
		switch flag {
		case "glow":
			obj.ExtraFlags.Set(types.ItemGlow)
		case "hum":
			obj.ExtraFlags.Set(types.ItemHum)
		case "dark":
			obj.ExtraFlags.Set(types.ItemDark)
		case "lock":
			obj.ExtraFlags.Set(types.ItemLock)
		case "evil":
			obj.ExtraFlags.Set(types.ItemEvil)
		case "invis":
			obj.ExtraFlags.Set(types.ItemInvis)
		case "magic":
			obj.ExtraFlags.Set(types.ItemMagic)
		case "nodrop":
			obj.ExtraFlags.Set(types.ItemNoDrop)
		case "bless":
			obj.ExtraFlags.Set(types.ItemBless)
		case "anti_good":
			obj.ExtraFlags.Set(types.ItemAntiGood)
		case "anti_evil":
			obj.ExtraFlags.Set(types.ItemAntiEvil)
		case "anti_neutral":
			obj.ExtraFlags.Set(types.ItemAntiNeutral)
		case "noremove":
			obj.ExtraFlags.Set(types.ItemNoRemove)
		case "nopurge":
			obj.ExtraFlags.Set(types.ItemNoPurge)
		}
	}

	// Set type-specific values
	if tmpl.Weapon != nil {
		obj.Values[0] = parseWeaponType(tmpl.Weapon.WeaponType)
		obj.Values[1] = tmpl.Weapon.DiceNumber
		obj.Values[2] = tmpl.Weapon.DiceSize
		obj.Values[3] = parseDamageType(tmpl.Weapon.DamageType)
	}

	if tmpl.Armor != nil {
		obj.Values[0] = tmpl.Armor.ACPierce
		obj.Values[1] = tmpl.Armor.ACBash
		obj.Values[2] = tmpl.Armor.ACSlash
		obj.Values[3] = tmpl.Armor.ACExotic
	}

	if tmpl.Container != nil {
		// ROM layout: v0 capacity, v1 flags, v2 key, v3 max item weight
		obj.Values[0] = tmpl.Container.Capacity
		obj.Values[1] = containerFlagBits(tmpl.Container.Flags)
		obj.Values[2] = tmpl.Container.Key
		obj.Values[3] = tmpl.Container.MaxWeight
	}
	if m := tmpl.Magic; m != nil {
		obj.Values[0] = m.Level
		switch obj.ItemType {
		case types.ItemTypeWand, types.ItemTypeStaff:
			obj.Values[1], obj.Values[2] = m.MaxCharges, m.Charges
			if len(m.Spells) > 0 {
				obj.Values[3] = spellSlot(m.Spells[0])
			}
		default: // potion, scroll, pill: v1..v4 are spells
			for i, name := range m.Spells {
				if i < 4 {
					obj.Values[1+i] = spellSlot(name)
				}
			}
		}
	}
	if dr := tmpl.Drink; dr != nil {
		obj.Values[0], obj.Values[1], obj.Values[2] = dr.Capacity, dr.Current, liquidIndex(dr.Liquid)
		if dr.Poisoned {
			obj.Values[3] = 1
		}
	}
	if fd := tmpl.Food; fd != nil {
		obj.Values[0], obj.Values[1] = fd.Full, fd.Hunger
		if fd.Poisoned {
			obj.Values[3] = 1
		}
	}
	if tmpl.Money != nil {
		obj.Cost = tmpl.Money.Amount // coin piles carry their copper in Cost
	}

	if tmpl.Light != nil {
		obj.Values[2] = tmpl.Light.Duration
	}

	if tmpl.Portal != nil {
		obj.Values[0] = tmpl.Portal.Charges  // uses remaining (-1 = infinite)
		obj.Values[3] = tmpl.Portal.DestVnum // destination room vnum
	}

	for _, ed := range tmpl.ExtraDescs {
		obj.ExtraDescriptions = append(obj.ExtraDescriptions, &types.ExtraDescription{
			Keywords: ed.Keywords, Description: ed.Description,
		})
	}

	// Parse affects (e.g., hitroll, damroll bonuses)
	for _, affData := range tmpl.Affects {
		aff := &types.Affect{
			Type:     "equipment",
			Level:    obj.Level,
			Duration: -1, // Permanent while equipped
			Location: parseApplyType(affData.Location),
			Modifier: affData.Modifier,
		}
		obj.Affects.Add(aff)
	}

	return obj
}

// parseApplyType converts a location string to an ApplyType constant
func parseApplyType(s string) types.ApplyType {
	switch strings.ToLower(s) {
	case "str", "strength":
		return types.ApplyStr
	case "dex", "dexterity":
		return types.ApplyDex
	case "int", "intelligence":
		return types.ApplyInt
	case "wis", "wisdom":
		return types.ApplyWis
	case "con", "constitution":
		return types.ApplyCon
	case "mana":
		return types.ApplyMana
	case "hp", "hit", "hitpoints":
		return types.ApplyHit
	case "move", "movement":
		return types.ApplyMove
	case "ac", "armor":
		return types.ApplyAC
	case "hitroll":
		return types.ApplyHitroll
	case "damroll":
		return types.ApplyDamroll
	case "saves", "saving_throw":
		return types.ApplySaves
	}
	return types.ApplyNone
}

// Helper: join strings with space
func stringJoin(strs []string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += " "
		}
		result += s
	}
	return result
}

// Helper: parse item type string
func parseItemType(s string) types.ItemType {
	switch s {
	case "light":
		return types.ItemTypeLight
	case "scroll":
		return types.ItemTypeScroll
	case "wand":
		return types.ItemTypeWand
	case "staff":
		return types.ItemTypeStaff
	case "weapon":
		return types.ItemTypeWeapon
	case "treasure":
		return types.ItemTypeTreasure
	case "armor":
		return types.ItemTypeArmor
	case "potion":
		return types.ItemTypePotion
	case "clothing":
		return types.ItemTypeClothing
	case "furniture":
		return types.ItemTypeFurniture
	case "trash":
		return types.ItemTypeTrash
	case "container":
		return types.ItemTypeContainer
	case "drink":
		return types.ItemTypeDrinkCon
	case "key":
		return types.ItemTypeKey
	case "food":
		return types.ItemTypeFood
	case "money":
		return types.ItemTypeMoney
	case "boat":
		return types.ItemTypeBoat
	case "fountain":
		return types.ItemTypeFountain
	case "pill":
		return types.ItemTypePill
	case "npc_corpse":
		return types.ItemTypeCorpseNPC
	case "pc_corpse":
		return types.ItemTypeCorpsePC
	case "protect":
		return types.ItemTypeProtect
	case "map":
		return types.ItemTypeMap
	case "portal":
		return types.ItemTypePortal
	case "warpstone":
		return types.ItemTypeWarpStone
	case "room_key":
		return types.ItemTypeRoomKey
	case "gem":
		return types.ItemTypeGem
	case "jewelry":
		return types.ItemTypeJewelry
	case "jukebox":
		return types.ItemTypeJukebox
	case "demon_stone":
		return types.ItemTypeDemonStone
	case "pit":
		return types.ItemTypePit
	default:
		return types.ItemTypeTrash
	}
}

// Helper: parse weapon type string
func parseWeaponType(s string) int {
	switch s {
	case "sword":
		return int(types.WeaponSword)
	case "dagger":
		return int(types.WeaponDagger)
	case "spear":
		return int(types.WeaponSpear)
	case "mace":
		return int(types.WeaponMace)
	case "axe":
		return int(types.WeaponAxe)
	case "flail":
		return int(types.WeaponFlail)
	case "whip":
		return int(types.WeaponWhip)
	case "polearm":
		return int(types.WeaponPolearm)
	default:
		return int(types.WeaponExotic)
	}
}

// Helper: parse damage type string
func parseDamageType(s string) int {
	switch s {
	case "bash":
		return int(types.DamBash)
	case "pierce":
		return int(types.DamPierce)
	case "slash":
		return int(types.DamSlash)
	case "fire":
		return int(types.DamFire)
	case "cold":
		return int(types.DamCold)
	case "lightning":
		return int(types.DamLightning)
	case "acid":
		return int(types.DamAcid)
	case "poison":
		return int(types.DamPoison)
	case "negative":
		return int(types.DamNegative)
	case "holy":
		return int(types.DamHoly)
	case "energy":
		return int(types.DamEnergy)
	default:
		return int(types.DamNone)
	}
}

// AreaResetAge is the number of ticks between resets for an area with players
const AreaResetAge = 3

// AreaResetAgeEmpty is the number of ticks between resets for an empty area
const AreaResetAgeEmpty = 1

// ProcessAreaResets checks all areas and resets those that need it
func (r *ResetSystem) ProcessAreaResets(areas []*types.Area) {
	for _, area := range areas {
		// Increment age
		area.Age++

		// Count players in area
		area.NumPlayers = 0
		for _, room := range area.Rooms {
			for _, ch := range room.People {
				if !ch.IsNPC() {
					area.NumPlayers++
				}
			}
		}
		area.Empty = (area.NumPlayers == 0)

		// Check if area needs reset
		resetAge := AreaResetAge
		if area.Empty {
			resetAge = AreaResetAgeEmpty
		}

		if area.Age >= resetAge {
			r.ResetArea(area)
		}
	}
}

// objSpells resolves spell names on magic items to registry slots.
var objSpells = magic.DefaultSpells()

// spellSlot returns the slot of the named spell, or 0 (no spell).
func spellSlot(name string) int {
	if sp := objSpells.FindByName(name); sp != nil {
		return sp.Slot
	}
	return 0
}

// liquidIndex returns the liquid table index for a liquid name (0, water,
// if unknown).
func liquidIndex(name string) int {
	for i, l := range liquidTable {
		if strings.EqualFold(l.Name, name) {
			return i
		}
	}
	return 0
}

// containerFlagBits converts container flag names to the bits the game
// checks (CONT_CLOSED 1, CONT_LOCKED 2, CONT_CLOSEABLE 4, CONT_PICKPROOF 8).
func containerFlagBits(flags []string) int {
	bits := 0
	for _, f := range flags {
		switch strings.ToLower(f) {
		case "closed":
			bits |= 1
		case "locked":
			bits |= 2
		case "closeable":
			bits |= 4
		case "pickproof":
			bits |= 8
		}
	}
	return bits
}
