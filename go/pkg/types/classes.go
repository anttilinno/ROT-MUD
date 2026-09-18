package types

import "slices"

// Class represents a character class
type Class struct {
	Name         string // Class name
	ShortName    string // Who list abbreviation
	PrimeStat    int    // Primary attribute (StatStr, StatInt, etc.)
	StartWeapon  int    // Starting weapon vnum
	Guilds       [3]int // Guild room vnums
	Thac0_00     int    // THAC0 at level 0
	Thac0_32     int    // THAC0 at level 32
	HPMin        int    // Minimum HP gain per level
	HPMax        int    // Maximum HP gain per level
	ManaGain     int    // Base mana gain per level
	FreesMana    bool   // Uses mana?
	BaseGroup    string // Base skill group
	DefaultGroup string // Default skill group
	Tier2        *Tier2 // nil for tier-1 classes
}

// Tier2 holds what only GodWars-style supernatural classes have
// (.planning/TIER2-GODWARS.md).
type Tier2 struct {
	Currency      string   // name of the power currency ("demonic power", "blood", ...)
	BarredRaces   []int    // races that may not take the rite
	OriginClasses []int    // if set, only these tier-1 classes may take the rite
	Vuln          ImmFlags // innate vulnerabilities
	Res           ImmFlags // innate resistances
	MasterVnum    int      // NPC who performs the rite (and, for demons, forges armour)
	Rite          Rite
}

// Rite is the deed a rerolled hero must complete before the master will
// ascend them: slay Count creatures matching the criteria.
type Rite struct {
	Desc     string   // shown by the master
	Count    int      // kills required
	MinLevel int      // victim level at least
	Align    int      // +1 good victims only, -1 evil only, 0 any
	Night    bool     // kills only count at night
	ActFlag  ActFlags // victim must have this act flag (0 = any)
}

// Class indices - Tier 1 (mortal)
const (
	ClassMage    = 0
	ClassCleric  = 1
	ClassThief   = 2
	ClassWarrior = 3
	ClassRanger  = 4
	ClassDruid   = 5
	ClassGhoul   = 6
	// Tier 2 (GodWars-style supernatural, reached by reroll + rite)
	ClassDemon      = 7
	ClassVampire    = 8
	ClassWerewolf   = 9
	ClassMagus      = 10
	ClassHighlander = 11
	ClassAngel      = 12
	MaxClass        = 13
)

// ClassTable contains all class definitions
var ClassTable = []Class{
	// Tier 1 Classes
	// Thac0_00/Thac0_32: ROM 2.4 values — level-0 and level-32 THAC0.
	// Lower Thac0_32 = better fighter (warrior hits most, mage hits least).
	{
		Name:         "mage",
		ShortName:    "Mag",
		PrimeStat:    StatInt,
		StartWeapon:  3020, // OBJ_VNUM_SCHOOL_DAGGER
		Guilds:       [3]int{3018, 9618, 18113},
		Thac0_00:     20,
		Thac0_32:     6,
		HPMin:        6,
		HPMax:        6,
		ManaGain:     8,
		FreesMana:    true,
		BaseGroup:    "mage basics",
		DefaultGroup: "mage default",
	},
	{
		Name:         "cleric",
		ShortName:    "Cle",
		PrimeStat:    StatWis,
		StartWeapon:  3021, // OBJ_VNUM_SCHOOL_MACE
		Guilds:       [3]int{3003, 9619, 5699},
		Thac0_00:     20,
		Thac0_32:     2,
		HPMin:        7,
		HPMax:        10,
		ManaGain:     2,
		FreesMana:    true,
		BaseGroup:    "cleric basics",
		DefaultGroup: "cleric default",
	},
	{
		Name:         "thief",
		ShortName:    "Thi",
		PrimeStat:    StatDex,
		StartWeapon:  3020, // OBJ_VNUM_SCHOOL_DAGGER
		Guilds:       [3]int{3028, 9639, 5633},
		Thac0_00:     20,
		Thac0_32:     -4,
		HPMin:        8,
		HPMax:        13,
		ManaGain:     -4,
		FreesMana:    false,
		BaseGroup:    "thief basics",
		DefaultGroup: "thief default",
	},
	{
		Name:         "warrior",
		ShortName:    "War",
		PrimeStat:    StatStr,
		StartWeapon:  3022, // OBJ_VNUM_SCHOOL_SWORD
		Guilds:       [3]int{3022, 9633, 5613},
		Thac0_00:     20,
		Thac0_32:     -10,
		HPMin:        13,
		HPMax:        18,
		ManaGain:     -10,
		FreesMana:    false,
		BaseGroup:    "warrior basics",
		DefaultGroup: "warrior default",
	},
	{
		Name:         "ranger",
		ShortName:    "Ran",
		PrimeStat:    StatStr,
		StartWeapon:  3023, // OBJ_VNUM_SCHOOL_SPEAR
		Guilds:       [3]int{3372, 9752, 18111},
		Thac0_00:     20,
		Thac0_32:     -6,
		HPMin:        9,
		HPMax:        13,
		ManaGain:     -4,
		FreesMana:    true,
		BaseGroup:    "ranger basics",
		DefaultGroup: "ranger default",
	},
	{
		Name:         "druid",
		ShortName:    "Dru",
		PrimeStat:    StatWis,
		StartWeapon:  3024, // OBJ_VNUM_SCHOOL_POLEARM
		Guilds:       [3]int{3369, 9755, 18111},
		Thac0_00:     20,
		Thac0_32:     2,
		HPMin:        7,
		HPMax:        10,
		ManaGain:     0,
		FreesMana:    true,
		BaseGroup:    "druid basics",
		DefaultGroup: "druid default",
	},
	{
		Name:         "ghoul",
		ShortName:    "Gho",
		PrimeStat:    StatCon,
		StartWeapon:  3020, // OBJ_VNUM_SCHOOL_DAGGER
		Guilds:       [3]int{3375, 9758, 18113},
		Thac0_00:     20,
		Thac0_32:     -3,
		HPMin:        6,
		HPMax:        8,
		ManaGain:     -30,
		FreesMana:    true,
		BaseGroup:    "ghoul basics",
		DefaultGroup: "ghoul default",
	},
	// Tier 2 Classes — see .planning/TIER2-GODWARS.md. Skills and spells come
	// from the origin tier-1 class (Character.SkillClass); powers from PowerTable.
	{
		Name: "demon", ShortName: "Dem", PrimeStat: StatStr, StartWeapon: 3022,
		Guilds: [3]int{3022, 9633, 5613}, Thac0_00: 20, Thac0_32: -10,
		HPMin: 13, HPMax: 20, ManaGain: -10, FreesMana: false,
		BaseGroup: "demon basics", DefaultGroup: "demon default",
		Tier2: &Tier2{
			Currency: "demonic power",
			// GodWars: "You cannot make a pact with the undead"; sky and fey races as celestial counterweights.
			BarredRaces: []int{RaceHeucuva, RaceAvian, RacePixie},
			Vuln:        ImmHoly,
			Res:         ImmFire,
			MasterVnum:  29601,
			Rite: Rite{Desc: "Slay ten good-hearted creatures of at least level 50 and I will take your soul in pact.",
				Count: 10, MinLevel: 50, Align: 1},
		},
	},
	{
		Name: "vampire", ShortName: "Vam", PrimeStat: StatCon, StartWeapon: 3020,
		Guilds: [3]int{3375, 9758, 18113}, Thac0_00: 20, Thac0_32: -6,
		HPMin: 8, HPMax: 16, ManaGain: -20, FreesMana: true,
		BaseGroup: "vampire basics", DefaultGroup: "vampire default",
		Tier2: &Tier2{
			Currency:      "blood",
			BarredRaces:   []int{RaceHeucuva, RaceTitan, RaceGiant},
			OriginClasses: []int{ClassGhoul},
			Vuln:          ImmFire | ImmSilver,
			Res:           ImmNegative | ImmPoison,
			MasterVnum:    29602,
			Rite: Rite{Desc: "Drain ten victims of at least level 50 under the night sky, ghoul, and I will give you the Embrace.",
				Count: 10, MinLevel: 50, Night: true},
		},
	},
	{
		Name: "werewolf", ShortName: "Wlf", PrimeStat: StatStr, StartWeapon: 3022,
		Guilds: [3]int{3022, 9633, 5613}, Thac0_00: 20, Thac0_32: -9,
		HPMin: 12, HPMax: 20, ManaGain: -15, FreesMana: false,
		BaseGroup: "werewolf basics", DefaultGroup: "werewolf default",
		Tier2: &Tier2{
			Currency:    "rage",
			BarredRaces: []int{RaceHeucuva, RacePixie, RaceAvian, RaceKenku},
			Vuln:        ImmSilver,
			MasterVnum:  29603,
			Rite: Rite{Desc: "Hunt ten creatures of at least level 50 by moonlight and survive the bite.",
				Count: 10, MinLevel: 50, Night: true},
		},
	},
	{
		Name: "magus", ShortName: "Mgs", PrimeStat: StatInt, StartWeapon: 3020,
		Guilds: [3]int{3018, 9618, 18113}, Thac0_00: 20, Thac0_32: 2,
		HPMin: 6, HPMax: 12, ManaGain: 10, FreesMana: true,
		BaseGroup: "magus basics", DefaultGroup: "magus default",
		Tier2: &Tier2{
			Currency:    "quintessence",
			BarredRaces: []int{RaceMinotaur, RaceGnoll, RaceGiant},
			MasterVnum:  29604,
			Rite: Rite{Desc: "Defeat ten spellcasters of at least level 50 and claim their quintessence to awaken.",
				Count: 10, MinLevel: 50, ActFlag: ActMage},
		},
	},
	{
		Name: "highlander", ShortName: "Hgh", PrimeStat: StatStr, StartWeapon: 3022,
		Guilds: [3]int{3022, 9633, 5613}, Thac0_00: 20, Thac0_32: -12,
		HPMin: 12, HPMax: 18, ManaGain: -10, FreesMana: false,
		BaseGroup: "highlander basics", DefaultGroup: "highlander default",
		Tier2: &Tier2{
			Currency:    "quickening",
			BarredRaces: []int{RaceHeucuva, RacePixie},
			MasterVnum:  29605,
			Rite: Rite{Desc: "Best ten warriors of at least level 50 in single combat and take their quickening.",
				Count: 10, MinLevel: 50, ActFlag: ActWarrior},
		},
	},
	{
		Name: "angel", ShortName: "Ang", PrimeStat: StatWis, StartWeapon: 3022,
		Guilds: [3]int{3003, 9619, 5699}, Thac0_00: 20, Thac0_32: -8,
		HPMin: 10, HPMax: 18, ManaGain: 5, FreesMana: true,
		BaseGroup: "angel basics", DefaultGroup: "angel default",
		Tier2: &Tier2{
			Currency:    "grace",
			BarredRaces: []int{RaceHeucuva, RaceGoblin, RaceGnoll, RaceHalfOrc},
			Vuln:        ImmNegative,
			Res:         ImmHoly,
			MasterVnum:  29606,
			Rite: Rite{Desc: "Cast down ten evil creatures of at least level 50 and you shall be raised among the host.",
				Count: 10, MinLevel: 50, Align: -1},
		},
	},
}

// GetClass returns the class at the given index
func GetClass(index int) *Class {
	if index >= 0 && index < len(ClassTable) {
		return &ClassTable[index]
	}
	return nil
}

// ClassByName returns the class with the given name
func ClassByName(name string) *Class {
	for i := range ClassTable {
		if ClassTable[i].Name == name {
			return &ClassTable[i]
		}
	}
	return nil
}

// ClassIndexByName returns the class index for a name
func ClassIndexByName(name string) int {
	for i := range ClassTable {
		if ClassTable[i].Name == name {
			return i
		}
	}
	return -1
}

// ClassName returns the name of a class by index
func ClassName(classIndex int) string {
	if c := GetClass(classIndex); c != nil {
		return c.Name
	}
	return "unknown"
}

// IsTier2Class returns true if the class is a remort/tier 2 class
func IsTier2Class(classIndex int) bool {
	return classIndex >= ClassDemon && classIndex < MaxClass
}

// RaceAllowed reports whether a race may take this class's rite.
func (c *Class) RaceAllowed(race int) bool {
	return c.Tier2 == nil || !slices.Contains(c.Tier2.BarredRaces, race)
}

// OriginAllowed reports whether a tier-1 class may take this class's rite.
func (c *Class) OriginAllowed(origin int) bool {
	return c.Tier2 == nil || len(c.Tier2.OriginClasses) == 0 || slices.Contains(c.Tier2.OriginClasses, origin)
}
