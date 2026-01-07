package game

import (
	"strings"

	"rotmud/pkg/loader"
	"rotmud/pkg/types"
)

// ResetSystem handles area reset execution
type ResetSystem struct {
	World  *loader.World
	Output OutputFunc
}

// NewResetSystem creates a new reset system
func NewResetSystem(world *loader.World) *ResetSystem {
	return &ResetSystem{
		World: world,
	}
}

// ResetArea resets a single area, spawning mobs and objects according to reset data
func (r *ResetSystem) ResetArea(area *types.Area) {
	for _, room := range area.Rooms {
		r.resetRoom(room)
	}
	area.Age = 0
}

// resetRoom executes all resets for a single room
func (r *ResetSystem) resetRoom(room *types.Room) {
	// Execute mob resets
	for _, reset := range room.MobResets {
		r.executeMobReset(room, &reset)
	}

	// Execute object resets
	for _, reset := range room.ObjResets {
		r.executeObjReset(room, &reset)
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

	// Check if we've reached the max for this object
	if reset.Max > 0 {
		count := r.countObjInWorld(reset.Vnum)
		if count >= reset.Max {
			return
		}
	}

	// Check if object already exists in this room
	for _, obj := range room.Objects {
		if obj.Vnum == reset.Vnum {
			return // Already have one here
		}
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

// countObjInWorld counts how many instances of an object vnum exist
func (r *ResetSystem) countObjInWorld(vnum int) int {
	count := 0
	for _, room := range r.World.Rooms {
		for _, obj := range room.Objects {
			if obj.Vnum == vnum {
				count++
			}
		}
		// Also count in mob inventory
		for _, ch := range room.People {
			for _, obj := range ch.Inventory {
				if obj.Vnum == vnum {
					count++
				}
			}
			for _, obj := range ch.Equipment {
				if obj != nil && obj.Vnum == vnum {
					count++
				}
			}
		}
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
		obj.Values[0] = tmpl.Container.Capacity
		obj.Values[3] = tmpl.Container.Key
		obj.Values[4] = tmpl.Container.MaxWeight
	}

	if tmpl.Light != nil {
		obj.Values[2] = tmpl.Light.Duration
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
