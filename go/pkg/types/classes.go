package types

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
}

// Class indices - Tier 1 (mortal)
const (
	ClassMage    = 0
	ClassCleric  = 1
	ClassThief   = 2
	ClassWarrior = 3
	ClassRanger  = 4
	ClassDruid   = 5
	ClassVampire = 6
	// Tier 2 (remort)
	ClassWizard    = 7
	ClassPriest    = 8
	ClassMercenary = 9
	ClassGladiator = 10
	ClassStrider   = 11
	ClassSage      = 12
	ClassLich      = 13
	MaxClass       = 14
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
		Name:         "vampire",
		ShortName:    "Vam",
		PrimeStat:    StatCon,
		StartWeapon:  3020, // OBJ_VNUM_SCHOOL_DAGGER
		Guilds:       [3]int{3375, 9758, 18113},
		Thac0_00:     20,
		Thac0_32:     -3,
		HPMin:        6,
		HPMax:        8,
		ManaGain:     -30,
		FreesMana:    true,
		BaseGroup:    "vampire basics",
		DefaultGroup: "vampire default",
	},
	// Tier 2 Classes (Remort) — better Thac0_32 than tier 1 equivalents
	{
		Name:         "wizard",
		ShortName:    "Wiz",
		PrimeStat:    StatInt,
		StartWeapon:  3020,
		Guilds:       [3]int{3018, 9618, 18113},
		Thac0_00:     20,
		Thac0_32:     4,
		HPMin:        6,
		HPMax:        18,
		ManaGain:     -4,
		FreesMana:    true,
		BaseGroup:    "wizard basics",
		DefaultGroup: "wizard default",
	},
	{
		Name:         "priest",
		ShortName:    "Prs",
		PrimeStat:    StatWis,
		StartWeapon:  3021,
		Guilds:       [3]int{3003, 9619, 5699},
		Thac0_00:     20,
		Thac0_32:     -1,
		HPMin:        -3,
		HPMax:        20,
		ManaGain:     2,
		FreesMana:    true,
		BaseGroup:    "priest basics",
		DefaultGroup: "priest default",
	},
	{
		Name:         "mercenary",
		ShortName:    "Mer",
		PrimeStat:    StatDex,
		StartWeapon:  3020,
		Guilds:       [3]int{3028, 9639, 5633},
		Thac0_00:     20,
		Thac0_32:     -6,
		HPMin:        8,
		HPMax:        23,
		ManaGain:     -14,
		FreesMana:    false,
		BaseGroup:    "mercenary basics",
		DefaultGroup: "mercenary default",
	},
	{
		Name:         "gladiator",
		ShortName:    "Gla",
		PrimeStat:    StatStr,
		StartWeapon:  3022,
		Guilds:       [3]int{3022, 9633, 5613},
		Thac0_00:     20,
		Thac0_32:     -14,
		HPMin:        14,
		HPMax:        25,
		ManaGain:     -20,
		FreesMana:    false,
		BaseGroup:    "gladiator basics",
		DefaultGroup: "gladiator default",
	},
	{
		Name:         "strider",
		ShortName:    "Str",
		PrimeStat:    StatInt,
		StartWeapon:  3020,
		Guilds:       [3]int{3372, 9752, 18111},
		Thac0_00:     20,
		Thac0_32:     -8,
		HPMin:        10,
		HPMax:        25,
		ManaGain:     -14,
		FreesMana:    true,
		BaseGroup:    "strider basics",
		DefaultGroup: "strider default",
	},
	{
		Name:         "sage",
		ShortName:    "Sag",
		PrimeStat:    StatWis,
		StartWeapon:  3024,
		Guilds:       [3]int{3369, 9755, 18111},
		Thac0_00:     20,
		Thac0_32:     -1,
		HPMin:        7,
		HPMax:        20,
		ManaGain:     -10,
		FreesMana:    true,
		BaseGroup:    "sage basics",
		DefaultGroup: "sage default",
	},
	{
		Name:         "lich",
		ShortName:    "Lic",
		PrimeStat:    StatCon,
		StartWeapon:  3020,
		Guilds:       [3]int{3375, 9758, 18113},
		Thac0_00:     20,
		Thac0_32:     -3,
		HPMin:        6,
		HPMax:        18,
		ManaGain:     -40,
		FreesMana:    true,
		BaseGroup:    "lich basics",
		DefaultGroup: "lich default",
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
	return classIndex >= ClassWizard && classIndex < MaxClass
}
