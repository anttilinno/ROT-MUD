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
	BarredRaces  []int  // Tier 2: races that may not take this class's rite
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
	ClassDemon = 7
	MaxClass   = 8
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
	// Tier 2 Classes — see .planning/TIER2-GODWARS.md
	{
		Name:         "demon",
		ShortName:    "Dem",
		PrimeStat:    StatStr,
		StartWeapon:  3022,
		Guilds:       [3]int{3022, 9633, 5613},
		Thac0_00:     20,
		Thac0_32:     -10,
		HPMin:        13,
		HPMax:        20,
		ManaGain:     -10,
		FreesMana:    false,
		BaseGroup:    "demon basics",
		DefaultGroup: "demon default",
		// GodWars: "You cannot make a pact with the undead"; sky/fey races as celestial counterweights.
		BarredRaces: []int{RaceHeucuva, RaceAvian, RacePixie},
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

// RaceAllowed reports whether a race may take this class.
func (c *Class) RaceAllowed(race int) bool {
	return !slices.Contains(c.BarredRaces, race)
}
