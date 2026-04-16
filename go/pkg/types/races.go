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

// RaceTable contains all race definitions.
// Stats order: [Str, Int, Wis, Dex, Con]
// Design principle: every race has clear bonuses AND penalties.
// No race should excel at everything.  Fighter races get high Str/Con
// with low Int/Wis; caster races get high Int/Wis with low Str/Con.
// Dex is the swing stat: each point above 15 gives −2 effective AC
// (harder for mobs to hit), so fighter races have Dex 14-17 rather
// than the old extremes of 13 (penalty) or 20 (free defense bonus).
var RaceTable = []Race{
	{
		// Human: perfectly balanced — no penalty, no great bonus.
		// Best XP rates for advanced classes; the "safe" choice.
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
		// Elf: agile and intelligent, but physically fragile.
		// Bonus: Int/Dex (mage/thief).  Penalty: Str/Con (dies easily).
		Name:            "elf",
		ShortName:       "Elf",
		Points:          5,
		ClassMultiplier: [MaxClass]int{100, 125, 100, 120, 120, 105, 115, 90, 113, 90, 108, 95, 95, 104},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{11, 15, 13, 16, 10},
		MaxStats:        [MaxStats]int{16, 20, 18, 21, 15},
		Size:            SizeSmall,
	},
	{
		// Dwarf: tough and spiritually strong, slow and not clever.
		// Bonus: Str/Con/Wis (warrior/cleric).  Penalty: Int/Dex (no dodge, no magic).
		Name:            "dwarf",
		ShortName:       "Dwarf",
		Points:          8,
		ClassMultiplier: [MaxClass]int{150, 100, 125, 100, 110, 110, 110, 135, 90, 113, 90, 113, 99, 99},
		BonusSkills:     []string{"berserk"},
		BaseStats:       [MaxStats]int{15, 9, 14, 10, 17},
		MaxStats:        [MaxStats]int{20, 14, 19, 15, 22},
		Size:            SizeMedium,
	},
	{
		// Giant: pure brute force — massive Str/Con, dumb and slow.
		// Bonus: Str/Con.  Penalty: Int/Wis/Dex (clumsy, ignorant).
		// Dex=14 gives a small −2 AC penalty, offset by huge Con HP pool.
		Name:            "giant",
		ShortName:       "Giant",
		Points:          6,
		ClassMultiplier: [MaxClass]int{200, 125, 150, 105, 125, 150, 120, 180, 113, 135, 95, 135, 135, 108},
		BonusSkills:     []string{"bash", "fast healing"},
		BaseStats:       [MaxStats]int{17, 8, 8, 9, 16},
		MaxStats:        [MaxStats]int{22, 12, 13, 14, 21},
		Size:            SizeLarge,
	},
	{
		// Pixie: tiny, magical, extremely fragile.
		// Bonus: Int/Wis/Dex.  Penalty: Str/Con (dies in two hits).
		Name:            "pixie",
		ShortName:       "Pixie",
		Points:          6,
		ClassMultiplier: [MaxClass]int{100, 100, 120, 200, 150, 100, 150, 90, 90, 108, 180, 95, 90, 135},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{8, 17, 16, 16, 8},
		MaxStats:        [MaxStats]int{11, 22, 21, 21, 13},
		Size:            SizeTiny,
	},
	{
		// Halfling: master thief — best Dex in the game, very weak.
		// Bonus: Dex/Int.  Penalty: Str (barely hurts).
		// Dex max reduced 22→20: Dex=22 gave warrior win% equal to fighter races,
		// which is wrong for a thief race. 20 still makes halfling the Dex king for thieves.
		Name:            "halfling",
		ShortName:       "Hfling",
		Points:          5,
		ClassMultiplier: [MaxClass]int{105, 120, 100, 150, 150, 120, 120, 95, 108, 90, 135, 108, 108, 108},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{8, 13, 10, 17, 13},
		MaxStats:        [MaxStats]int{13, 18, 15, 20, 18},
		Size:            SizeSmall,
	},
	{
		// Half-orc: fighter-thief hybrid — strong and tough but brutish.
		// Bonus: Str/Con.  Penalty: Int/Wis (dumb and faithless).
		// Dex max reduced 18→17: Dex=18 combined with Str/Con gave warrior win% of 70%,
		// well above the 60-65% target. Dex=17 keeps a modest AC bonus without dominating.
		Name:            "halforc",
		ShortName:       "Hf-Orc",
		Points:          6,
		ClassMultiplier: [MaxClass]int{200, 200, 120, 100, 125, 150, 105, 180, 180, 108, 90, 135, 135, 95},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{15, 8, 8, 13, 16},
		MaxStats:        [MaxStats]int{20, 12, 13, 17, 21},
		Size:            SizeMedium,
	},
	{
		// Goblin: sneaky and quick, but weak and faithless.
		// Bonus: Dex/Con.  Penalty: Str/Wis.
		Name:            "goblin",
		ShortName:       "Goblin",
		Points:          5,
		ClassMultiplier: [MaxClass]int{105, 125, 110, 125, 120, 120, 110, 95, 113, 99, 113, 99, 108, 99},
		BonusSkills:     []string{"sneak", "hide"},
		BaseStats:       [MaxStats]int{9, 11, 8, 15, 13},
		MaxStats:        [MaxStats]int{14, 16, 12, 20, 18},
		Size:            SizeSmall,
	},
	{
		// Half-elf: versatile but unremarkable — slight penalties vs full-blooded races.
		// Bonus: none stand-out.  Penalty: minor Str/Con vs human.
		Name:            "halfelf",
		ShortName:       "Hf-Elf",
		Points:          2,
		ClassMultiplier: [MaxClass]int{105, 105, 105, 105, 105, 105, 105, 95, 95, 95, 95, 95, 95, 95},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{12, 13, 13, 13, 12},
		MaxStats:        [MaxStats]int{17, 18, 18, 18, 17},
		Size:            SizeMedium,
	},
	{
		// Avian: graceful and spiritually aware, but physically weak.
		// Bonus: Wis/Dex (druid/cleric/ranger).  Penalty: Str/Int.
		Name:            "avian",
		ShortName:       "Avian",
		Points:          5,
		ClassMultiplier: [MaxClass]int{110, 105, 150, 125, 120, 100, 120, 99, 95, 135, 113, 108, 90, 108},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{9, 10, 15, 13, 12},
		MaxStats:        [MaxStats]int{14, 15, 20, 18, 17},
		Size:            SizeLarge,
	},
	{
		// Gnome: excellent mage/cleric, terrible fighter.
		// Bonus: Int/Wis.  Penalty: Str/Con/Dex (fragile and uncoordinated).
		Name:            "gnome",
		ShortName:       "Gnome",
		Points:          4,
		ClassMultiplier: [MaxClass]int{100, 110, 150, 150, 125, 105, 150, 90, 99, 135, 135, 99, 95, 135},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{8, 16, 15, 10, 9},
		MaxStats:        [MaxStats]int{13, 21, 20, 15, 14},
		Size:            SizeSmall,
	},
	{
		// Draconian: dragon-blooded fighter-mage — strong and clever, impious and slow.
		// Bonus: Str/Con/Int.  Penalty: Wis/Dex (arrogant, not agile).
		Name:            "draconian",
		ShortName:       "Dracon",
		Points:          11,
		ClassMultiplier: [MaxClass]int{125, 150, 200, 100, 110, 125, 150, 113, 135, 180, 90, 108, 113, 135},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{17, 13, 8, 9, 16},
		MaxStats:        [MaxStats]int{22, 18, 13, 14, 21},
		Size:            SizeHuge,
	},
	{
		// Centaur: powerful mobile fighter — Str/Con fighter with Dex=17 for mobility.
		// Old Dex=13 was a crippling penalty; centaurs are fast, not slow.
		// Bonus: Str/Con/Dex.  Penalty: Int/Wis (beast-minded).
		Name:            "centaur",
		ShortName:       "Centr",
		Points:          9,
		ClassMultiplier: [MaxClass]int{100, 110, 100, 175, 110, 110, 95, 90, 100, 90, 165, 100, 100, 85},
		BonusSkills:     []string{"enhanced damage"},
		BaseStats:       [MaxStats]int{16, 9, 9, 12, 16},
		MaxStats:        [MaxStats]int{21, 14, 14, 17, 21},
		Size:            SizeLarge,
	},
	{
		// Gnoll: feral pack fighter — strong and tough but dull-witted.
		// Bonus: Str/Con.  Penalty: Int/Wis.
		// Dex=17 (was 20) removes the free AC advantage while keeping mobility.
		Name:            "gnoll",
		ShortName:       "Gnoll",
		Points:          7,
		ClassMultiplier: [MaxClass]int{110, 110, 125, 110, 175, 110, 110, 100, 100, 115, 100, 165, 100, 100},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{15, 8, 8, 12, 15},
		MaxStats:        [MaxStats]int{20, 12, 12, 17, 20},
		Size:            SizeLarge,
	},
	{
		// Heucuva: undead warrior — supernaturally strong, mindless.
		// Bonus: Str/Dex (undead speed).  Penalty: Int/Wis (mindless undead).
		// Old triple-25 was completely broken; now a viable but niche fighter.
		Name:            "heucuva",
		ShortName:       "Heucuv",
		Points:          10,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 100, 110, 110, 100, 100, 100, 100, 90, 100, 100, 90},
		BonusSkills:     []string{"second attack"},
		BaseStats:       [MaxStats]int{16, 8, 8, 11, 14},
		MaxStats:        [MaxStats]int{21, 10, 10, 16, 19},
		Size:            SizeMedium,
	},
	{
		// Kenku: crow-like predator — fast and spiritually aware, but physically frail.
		// Bonus: Dex/Wis.  Penalty: Str/Int/Con (old 19-21 everywhere had no penalties).
		Name:            "kenku",
		ShortName:       "Kenku",
		Points:          5,
		ClassMultiplier: [MaxClass]int{125, 110, 150, 150, 110, 125, 180, 115, 100, 140, 140, 100, 115, 170},
		BonusSkills:     []string{"meditation"},
		BaseStats:       [MaxStats]int{11, 9, 15, 16, 10},
		MaxStats:        [MaxStats]int{16, 14, 20, 21, 15},
		Size:            SizeMedium,
	},
	{
		// Minotaur: massive berserker — top Str/Con, very dumb and slow.
		// Bonus: Str/Con.  Penalty: Int/Wis/Dex (hulking brute).
		// Dex=14 gives −2 AC penalty; offset by Con=22 HP pool.
		Name:            "minotaur",
		ShortName:       "Minotr",
		Points:          7,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 95, 110, 110, 110, 100, 100, 100, 85, 100, 100, 100},
		BonusSkills:     []string{"enhanced damage"},
		BaseStats:       [MaxStats]int{18, 8, 8, 9, 17},
		MaxStats:        [MaxStats]int{23, 11, 11, 14, 22},
		Size:            SizeLarge,
	},
	{
		// Satyr: chaotic nature warrior — strong and resilient, godless and reckless.
		// Bonus: Str/Con/Dex.  Penalty: Wis (completely faithless), Int.
		// Old Int=19 made no thematic sense; Dex raised from 14 to give mobility.
		Name:            "satyr",
		ShortName:       "Satyr",
		Points:          6,
		ClassMultiplier: [MaxClass]int{110, 110, 110, 175, 110, 110, 150, 100, 100, 100, 165, 100, 100, 140},
		BonusSkills:     []string{},
		BaseStats:       [MaxStats]int{16, 8, 8, 12, 15},
		MaxStats:        [MaxStats]int{21, 13, 10, 17, 20},
		Size:            SizeLarge,
	},
	{
		// Titan: god-race — supreme Str/Con, but enormous and clumsy.
		// Bonus: Str/Con (unmatched).  Penalty: Dex (−6 AC, mob hits more often).
		// The Dex=12 penalty and Con=24 HP buffer make for exciting high-variance fights.
		Name:            "titan",
		ShortName:       "Titan",
		Points:          11,
		ClassMultiplier: [MaxClass]int{180, 105, 130, 105, 105, 130, 100, 170, 93, 115, 95, 95, 115, 98},
		BonusSkills:     []string{"fast healing"},
		BaseStats:       [MaxStats]int{20, 11, 10, 8, 19},
		MaxStats:        [MaxStats]int{25, 16, 15, 12, 24},
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
