package types

// Race represents a playable character race
type Race struct {
	Name            string        // Race name
	ShortName       string        // Who list abbreviation
	Points          int           // Creation point cost
	ClassMultiplier [MaxClass]int // XP multiplier per class (100 = normal)
	BonusSkills     []string      // Skills gained at creation
	BaseStats       [MaxStats]int // Starting stats
	MaxStats        [MaxStats]int // Maximum stats
	Size            Size          // Race size
}

// Race indices
const (
	RaceHuman     = 0
	RaceElf       = 1
	RaceDwarf     = 2
	RaceGiant     = 3
	RacePixie     = 4
	RaceHalfling  = 5
	RaceHalfOrc   = 6
	RaceGoblin    = 7
	RaceHalfElf   = 8
	RaceAvian     = 9
	RaceGnome     = 10
	RaceDraconian = 11
	RaceCentaur   = 12
	RaceGnoll     = 13
	RaceHeucuva   = 14
	RaceKenku     = 15
	RaceMinotaur  = 16
	RaceSatyr     = 17
	RaceTitan     = 18
	MaxRace       = 19
)

// RaceTable contains all race definitions
var RaceTable = []Race{
	{
		Name:            "human",
		ShortName:       "Human",
		Points:          0,
		ClassMultiplier: [MaxClass]int{100, 100, 100, 100, 100, 100, 100, 90, 90, 90, 90, 90, 90, 90},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{13, 13, 13, 13, 13},
		MaxStats:        [MaxStats]int{18, 18, 18, 18, 18},
		Size:            SizeMedium,
	},
	{
		Name:            "elf",
		ShortName:       "Elf",
		Points:          5,
		ClassMultiplier: [MaxClass]int{100, 125, 100, 120, 120, 105, 115, 90, 113, 90, 108, 95, 95, 104},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{12, 14, 13, 15, 11},
		MaxStats:        [MaxStats]int{16, 20, 18, 21, 15},
		Size:            SizeSmall,
	},
	{
		Name:            "dwarf",
		ShortName:       "Dwarf",
		Points:          8,
		ClassMultiplier: [MaxClass]int{150, 100, 125, 100, 110, 110, 110, 135, 90, 113, 90, 113, 99, 99},
		BonusSkills:     []string{"berserk"},
		BaseStats:       [MaxStats]int{14, 12, 13, 11, 15},
		MaxStats:        [MaxStats]int{20, 16, 19, 15, 21},
		Size:            SizeMedium,
	},
	{
		Name:            "giant",
		ShortName:       "Giant",
		Points:          6,
		ClassMultiplier: [MaxClass]int{200, 125, 150, 105, 125, 150, 120, 180, 113, 135, 95, 135, 135, 108},
		BonusSkills:     []string{"bash", "fast healing"},
		BaseStats:       [MaxStats]int{16, 11, 13, 11, 14},
		MaxStats:        [MaxStats]int{22, 15, 18, 15, 20},
		Size:            SizeLarge,
	},
	{
		Name:            "pixie",
		ShortName:       "Pixie",
		Points:          6,
		ClassMultiplier: [MaxClass]int{100, 100, 120, 200, 150, 100, 150, 90, 90, 108, 180, 95, 90, 135},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{10, 15, 15, 15, 10},
		MaxStats:        [MaxStats]int{14, 21, 21, 20, 14},
		Size:            SizeTiny,
	},
	{
		Name:            "halfling",
		ShortName:       "Hfling",
		Points:          5,
		ClassMultiplier: [MaxClass]int{105, 120, 100, 150, 150, 120, 120, 95, 108, 90, 135, 108, 108, 108},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{11, 14, 12, 15, 13},
		MaxStats:        [MaxStats]int{15, 20, 16, 21, 18},
		Size:            SizeSmall,
	},
	{
		Name:            "halforc",
		ShortName:       "Hf-Orc",
		Points:          6,
		ClassMultiplier: [MaxClass]int{200, 200, 120, 100, 125, 150, 105, 180, 180, 108, 90, 135, 135, 95},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{14, 11, 11, 14, 15},
		MaxStats:        [MaxStats]int{19, 15, 15, 20, 21},
		Size:            SizeMedium,
	},
	{
		Name:            "goblin",
		ShortName:       "Goblin",
		Points:          5,
		ClassMultiplier: [MaxClass]int{105, 125, 110, 125, 120, 120, 110, 95, 113, 99, 113, 99, 108, 99},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{11, 14, 12, 15, 14},
		MaxStats:        [MaxStats]int{16, 20, 16, 19, 20},
		Size:            SizeSmall,
	},
	{
		Name:            "halfelf",
		ShortName:       "Hf-Elf",
		Points:          2,
		ClassMultiplier: [MaxClass]int{105, 105, 105, 105, 105, 105, 105, 95, 95, 95, 95, 95, 95, 95},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{12, 13, 14, 13, 13},
		MaxStats:        [MaxStats]int{17, 18, 19, 18, 18},
		Size:            SizeMedium,
	},
	{
		Name:            "avian",
		ShortName:       "Avian",
		Points:          5,
		ClassMultiplier: [MaxClass]int{110, 105, 150, 125, 120, 100, 120, 99, 95, 135, 113, 108, 90, 108},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{12, 14, 15, 11, 12},
		MaxStats:        [MaxStats]int{17, 19, 20, 16, 17},
		Size:            SizeLarge,
	},
	{
		Name:            "gnome",
		ShortName:       "Gnome",
		Points:          4,
		ClassMultiplier: [MaxClass]int{100, 110, 150, 150, 125, 105, 150, 90, 99, 135, 135, 99, 95, 135},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{11, 15, 14, 12, 12},
		MaxStats:        [MaxStats]int{16, 20, 19, 15, 15},
		Size:            SizeSmall,
	},
	{
		Name:            "draconian",
		ShortName:       "Dracon",
		Points:          11,
		ClassMultiplier: [MaxClass]int{125, 150, 200, 100, 110, 125, 150, 113, 135, 180, 90, 108, 113, 135},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{16, 13, 12, 11, 15},
		MaxStats:        [MaxStats]int{22, 18, 16, 15, 21},
		Size:            SizeHuge,
	},
	{
		Name:            "centaur",
		ShortName:       "Centr",
		Points:          9,
		ClassMultiplier: [MaxClass]int{100, 110, 100, 175, 110, 110, 95, 90, 100, 90, 165, 100, 100, 85},
		BonusSkills:     []string{"enhanced damage"},
		BaseStats:       [MaxStats]int{15, 12, 10, 8, 16},
		MaxStats:        [MaxStats]int{20, 17, 15, 13, 21},
		Size:            SizeLarge,
	},
	{
		Name:            "gnoll",
		ShortName:       "Gnoll",
		Points:          7,
		ClassMultiplier: [MaxClass]int{110, 110, 125, 110, 175, 110, 110, 100, 100, 115, 100, 165, 100, 100},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{15, 11, 10, 16, 15},
		MaxStats:        [MaxStats]int{20, 16, 15, 20, 19},
		Size:            SizeLarge,
	},
	{
		Name:            "heucuva",
		ShortName:       "Heucuv",
		Points:          10,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 100, 110, 110, 100, 100, 100, 100, 90, 100, 100, 90},
		BonusSkills:     []string{"second attack"},
		BaseStats:       [MaxStats]int{20, 5, 5, 20, 20},
		MaxStats:        [MaxStats]int{25, 10, 10, 25, 25},
		Size:            SizeMedium,
	},
	{
		Name:            "kenku",
		ShortName:       "Kenku",
		Points:          5,
		ClassMultiplier: [MaxClass]int{125, 110, 150, 150, 110, 125, 180, 115, 100, 140, 140, 100, 115, 170},
		BonusSkills:     []string{"meditation"},
		BaseStats:       [MaxStats]int{14, 14, 16, 15, 14},
		MaxStats:        [MaxStats]int{19, 19, 21, 20, 19},
		Size:            SizeMedium,
	},
	{
		Name:            "minotaur",
		ShortName:       "Minotr",
		Points:          7,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 95, 110, 110, 110, 100, 100, 100, 85, 100, 100, 100},
		BonusSkills:     []string{"enhanced damage"},
		BaseStats:       [MaxStats]int{18, 11, 10, 11, 17},
		MaxStats:        [MaxStats]int{23, 16, 15, 16, 22},
		Size:            SizeLarge,
	},
	{
		Name:            "satyr",
		ShortName:       "Satyr",
		Points:          6,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 175, 110, 110, 150, 100, 100, 100, 165, 100, 100, 140},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{18, 14, 5, 9, 16},
		MaxStats:        [MaxStats]int{23, 19, 10, 14, 21},
		Size:            SizeLarge,
	},
	{
		Name:            "titan",
		ShortName:       "Titan",
		Points:          11,
		ClassMultiplier: [MaxClass]int{180, 105, 130, 105, 105, 130, 100, 170, 93, 115, 95, 95, 115, 98},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{20, 13, 13, 10, 20},
		MaxStats:        [MaxStats]int{25, 18, 18, 15, 25},
		Size:            SizeGiant,
	},
}

// GetRace returns the race at the given index
func GetRace(index int) *Race {
	if index >= 0 && index < len(RaceTable) {
		return &RaceTable[index]
	}
	return nil
}

// RaceByName returns the race with the given name
func RaceByName(name string) *Race {
	for i := range RaceTable {
		if RaceTable[i].Name == name {
			return &RaceTable[i]
		}
	}
	return nil
}

// RaceIndexByName returns the race index for a name
func RaceIndexByName(name string) int {
	for i := range RaceTable {
		if RaceTable[i].Name == name {
			return i
		}
	}
	return -1
}

// RaceName returns the name of a race by index
func RaceName(raceIndex int) string {
	if r := GetRace(raceIndex); r != nil {
		return r.Name
	}
	return "unknown"
}
