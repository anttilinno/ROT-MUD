package types

// Direction represents a compass direction
type Direction int

const (
	DirNorth Direction = iota
	DirEast
	DirSouth
	DirWest
	DirUp
	DirDown
	DirMax
)

// String returns the direction name
func (d Direction) String() string {
	names := []string{"north", "east", "south", "west", "up", "down"}
	if d >= 0 && int(d) < len(names) {
		return names[d]
	}
	return "unknown"
}

// Reverse returns the opposite direction
func (d Direction) Reverse() Direction {
	switch d {
	case DirNorth:
		return DirSouth
	case DirSouth:
		return DirNorth
	case DirEast:
		return DirWest
	case DirWest:
		return DirEast
	case DirUp:
		return DirDown
	case DirDown:
		return DirUp
	}
	return d
}

// Position represents character position
type Position int

const (
	PosDead Position = iota
	PosMortal
	PosIncap
	PosStunned
	PosSleeping
	PosResting
	PosSitting
	PosFighting
	PosStanding
)

// String returns the position name
func (p Position) String() string {
	names := []string{
		"dead", "mortally wounded", "incapacitated", "stunned",
		"sleeping", "resting", "sitting", "fighting", "standing",
	}
	if p >= 0 && int(p) < len(names) {
		return names[p]
	}
	return "unknown"
}

// Sex represents character sex
type Sex int

const (
	SexNeutral Sex = iota
	SexMale
	SexFemale
)

// String returns the sex name
func (s Sex) String() string {
	names := []string{"neutral", "male", "female"}
	if s >= 0 && int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// Size represents character/object size
type Size int

const (
	SizeTiny Size = iota
	SizeSmall
	SizeMedium
	SizeLarge
	SizeHuge
	SizeGiant
)

// String returns the size name
func (sz Size) String() string {
	names := []string{"tiny", "small", "medium", "large", "huge", "giant"}
	if sz >= 0 && int(sz) < len(names) {
		return names[sz]
	}
	return "unknown"
}

// Sector represents room terrain type
type Sector int

const (
	SectInside Sector = iota
	SectCity
	SectField
	SectForest
	SectHills
	SectMountain
	SectWaterSwim
	SectWaterNoSwim
	SectUnused
	SectAir
	SectDesert
)

// String returns the sector name
func (s Sector) String() string {
	names := []string{
		"inside", "city", "field", "forest", "hills", "mountain",
		"swim", "noswim", "unused", "air", "desert",
	}
	if s >= 0 && int(s) < len(names) {
		return names[s]
	}
	return "unknown"
}

// MoveCost returns movement point cost for this sector
func (s Sector) MoveCost() int {
	costs := []int{1, 1, 2, 3, 4, 6, 4, 1, 6, 10, 6}
	if s >= 0 && int(s) < len(costs) {
		return costs[s]
	}
	return 1
}

// ItemType represents object types
type ItemType int

const (
	ItemTypeLight ItemType = iota + 1
	ItemTypeScroll
	ItemTypeWand
	ItemTypeStaff
	ItemTypeWeapon
	_
	_
	ItemTypeTreasure
	ItemTypeArmor
	ItemTypePotion
	ItemTypeClothing
	ItemTypeFurniture
	ItemTypeTrash
	_
	ItemTypeContainer
	_
	ItemTypeDrinkCon
	ItemTypeKey
	ItemTypeFood
	ItemTypeMoney
	_
	ItemTypeBoat
	ItemTypeCorpseNPC
	ItemTypeCorpsePC
	ItemTypeFountain
	ItemTypePill
	ItemTypeProtect
	ItemTypeMap
	ItemTypePortal
	ItemTypeWarpStone
	ItemTypeRoomKey
	ItemTypeGem
	ItemTypeJewelry
	ItemTypeJukebox
	ItemTypeDemonStone // 35 - demon stone for conjure spell
	_                  // 36 - unused
	ItemTypePit        // 37 - donation pit
)

// String returns the item type name
func (t ItemType) String() string {
	names := map[ItemType]string{
		ItemTypeLight:      "light",
		ItemTypeScroll:     "scroll",
		ItemTypeWand:       "wand",
		ItemTypeStaff:      "staff",
		ItemTypeWeapon:     "weapon",
		ItemTypeTreasure:   "treasure",
		ItemTypeArmor:      "armor",
		ItemTypePotion:     "potion",
		ItemTypeClothing:   "clothing",
		ItemTypeFurniture:  "furniture",
		ItemTypeTrash:      "trash",
		ItemTypeContainer:  "container",
		ItemTypeDrinkCon:   "drink",
		ItemTypeKey:        "key",
		ItemTypeFood:       "food",
		ItemTypeMoney:      "money",
		ItemTypeBoat:       "boat",
		ItemTypeCorpseNPC:  "npc_corpse",
		ItemTypeCorpsePC:   "pc_corpse",
		ItemTypeFountain:   "fountain",
		ItemTypePill:       "pill",
		ItemTypeProtect:    "protect",
		ItemTypeMap:        "map",
		ItemTypePortal:     "portal",
		ItemTypeWarpStone:  "warpstone",
		ItemTypeRoomKey:    "room_key",
		ItemTypeGem:        "gem",
		ItemTypeJewelry:    "jewelry",
		ItemTypeJukebox:    "jukebox",
		ItemTypeDemonStone: "demon_stone",
		ItemTypePit:        "pit",
	}
	if name, ok := names[t]; ok {
		return name
	}
	return "unknown"
}

// WearLocation represents where equipment is worn
type WearLocation int

const (
	WearLocNone      WearLocation = -1
	WearLocLight     WearLocation = 0
	WearLocFingerL   WearLocation = 1
	WearLocFingerR   WearLocation = 2
	WearLocNeck1     WearLocation = 3
	WearLocNeck2     WearLocation = 4
	WearLocBody      WearLocation = 5
	WearLocHead      WearLocation = 6
	WearLocLegs      WearLocation = 7
	WearLocFeet      WearLocation = 8
	WearLocHands     WearLocation = 9
	WearLocArms      WearLocation = 10
	WearLocShield    WearLocation = 11
	WearLocAbout     WearLocation = 12
	WearLocWaist     WearLocation = 13
	WearLocWristL    WearLocation = 14
	WearLocWristR    WearLocation = 15
	WearLocWield     WearLocation = 16
	WearLocHold      WearLocation = 17
	WearLocFloat     WearLocation = 18
	WearLocSecondary WearLocation = 19
	WearLocFace      WearLocation = 20
	WearLocMax       WearLocation = 21
)

// ApplyType represents what stat an affect modifies
type ApplyType int

const (
	ApplyNone ApplyType = iota
	ApplyStr
	ApplyDex
	ApplyInt
	ApplyWis
	ApplyCon
	ApplySex
	ApplyClass
	ApplyLevel
	ApplyAge
	ApplyHeight
	ApplyWeight
	ApplyMana
	ApplyHit
	ApplyMove
	ApplyGold
	ApplyExp
	ApplyAC
	ApplyHitroll
	ApplyDamroll
	ApplySaves
)

// String returns the apply type name
func (a ApplyType) String() string {
	names := []string{
		"none", "strength", "dexterity", "intelligence", "wisdom",
		"constitution", "sex", "class", "level", "age", "height",
		"weight", "mana", "hp", "move", "gold", "exp", "ac",
		"hitroll", "damroll", "saves",
	}
	if a >= 0 && int(a) < len(names) {
		return names[a]
	}
	return "unknown"
}

// DamageType represents damage types
type DamageType int

const (
	DamNone DamageType = iota
	DamBash
	DamPierce
	DamSlash
	DamFire
	DamCold
	DamLightning
	DamAcid
	DamPoison
	DamNegative
	DamHoly
	DamEnergy
	DamMental
	DamDisease
	DamDrowning
	DamLight
	DamOther
	DamHarm
	DamCharm
	DamSound
)

// WeaponClass represents weapon types
type WeaponClass int

const (
	WeaponExotic WeaponClass = iota
	WeaponSword
	WeaponDagger
	WeaponSpear
	WeaponMace
	WeaponAxe
	WeaponFlail
	WeaponWhip
	WeaponPolearm
)

// Stat represents a character stat index
type Stat = int

// Stat indices
const (
	StatStr  Stat = 0
	StatInt  Stat = 1
	StatWis  Stat = 2
	StatDex  Stat = 3
	StatCon  Stat = 4
	MaxStats      = 5
)

// StatName returns the name of a stat
func StatName(stat Stat) string {
	names := []string{"strength", "intelligence", "wisdom", "dexterity", "constitution"}
	if stat >= 0 && stat < len(names) {
		return names[stat]
	}
	return "unknown"
}

// Class indices are defined in classes.go

// AC indices
const (
	ACPierce = 0
	ACBash   = 1
	ACSlash  = 2
	ACExotic = 3
)

// Game limits
const (
	MaxLevel      = 110
	LevelHero     = MaxLevel - 9 // 101
	LevelImmortal = MaxLevel - 8 // 102
	MaxTrack      = 12           // Track history size for tracking skill
)

// Pulse timing (4 pulses per second = 250ms per pulse)
const (
	PulsePerSecond = 4
	PulseViolence  = 3 * PulsePerSecond   // 750ms
	PulseMobile    = 4 * PulsePerSecond   // 1s
	PulseMusic     = 6 * PulsePerSecond   // 1.5s
	PulseTick      = 60 * PulsePerSecond  // 15s
	PulseArea      = 120 * PulsePerSecond // 30s
)

// Connection states
type ConnState int

const (
	ConPlaying ConnState = iota
	ConGetName
	ConGetOldPassword
	ConConfirmNewName
	ConGetNewPassword
	ConConfirmNewPassword
	ConGetNewRace
	ConGetNewSex
	ConGetNewClass
	ConGetAlignment
	ConDefaultChoice
	ConGenGroups
	ConPickWeapon
	ConReadIMOTD
	ConReadMOTD
	ConBreakConnect
)

// Condition indices for PCData.Condition array
const (
	CondDrunk  = 0 // Intoxication level
	CondFull   = 1 // Fullness from eating
	CondThirst = 2 // Thirst level
	CondHunger = 3 // Hunger level
)
