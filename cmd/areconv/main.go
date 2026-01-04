// areconv converts ROM .are area files to TOML format
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Area holds parsed area data
type Area struct {
	Name    string
	Credits string
	VnumMin int
	VnumMax int
	Rooms   []Room
	Mobiles []Mobile
	Objects []Object
	Resets  []Reset
}

type Room struct {
	Vnum        int
	Name        string
	Description string
	Sector      string
	Flags       []string
	Exits       []Exit
	ExtraDescs  []ExtraDesc
}

type Exit struct {
	Direction string
	ToVnum    int
	Key       int
	Flags     []string
	Keyword   string
}

type ExtraDesc struct {
	Keywords    string
	Description string
}

type Mobile struct {
	Vnum        int
	Keywords    []string
	ShortDesc   string
	LongDesc    string
	Description string
	Race        string
	Level       int
	Sex         string
	Alignment   int
	ActFlags    []string
	AffFlags    []string
	HitDice     Dice
	ManaDice    Dice
	DamageDice  Dice
	DamType     string
	AC          [4]int
	Gold        int
	OffFlags    []string
	ImmFlags    []string
	ResFlags    []string
	VulnFlags   []string
	StartPos    string
	DefaultPos  string
	Size        string
}

type Dice struct {
	Number int
	Size   int
	Bonus  int
}

type Object struct {
	Vnum       int
	Keywords   []string
	ShortDesc  string
	LongDesc   string
	ItemType   string
	ExtraFlags []string
	WearFlags  []string
	Level      int
	Weight     int
	Cost       int
	Condition  string
	Values     [5]int
	Affects    []ObjAffect
}

type ObjAffect struct {
	Location string
	Modifier int
}

type Reset struct {
	Command  string
	Arg1     int
	Arg2     int
	Arg3     int
	Arg4     int
	RoomVnum int // For M and O resets
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: areconv <input.are> <output_dir>")
		fmt.Println("       areconv --all <area_dir> <output_dir>")
		os.Exit(1)
	}

	if os.Args[1] == "--all" {
		if len(os.Args) < 4 {
			fmt.Println("Usage: areconv --all <area_dir> <output_dir>")
			os.Exit(1)
		}
		convertAll(os.Args[2], os.Args[3])
	} else {
		convertOne(os.Args[1], os.Args[2])
	}
}

func convertAll(areaDir, outputDir string) {
	// Read area.lst
	listPath := filepath.Join(areaDir, "area.lst")
	file, err := os.Open(listPath)
	if err != nil {
		fmt.Printf("Error opening area.lst: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var areaFiles []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "$" {
			continue
		}
		if strings.HasSuffix(line, ".are") {
			areaFiles = append(areaFiles, line)
		}
	}

	fmt.Printf("Found %d area files to convert\n", len(areaFiles))

	for _, areaFile := range areaFiles {
		inputPath := filepath.Join(areaDir, areaFile)
		// Create output directory named after the area file (without .are)
		areaName := strings.TrimSuffix(areaFile, ".are")
		areaOutputDir := filepath.Join(outputDir, areaName)

		fmt.Printf("Converting %s -> %s\n", areaFile, areaOutputDir)
		if err := convertAreaFile(inputPath, areaOutputDir); err != nil {
			fmt.Printf("  ERROR: %v\n", err)
		}
	}
}

func convertOne(inputPath, outputDir string) {
	if err := convertAreaFile(inputPath, outputDir); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func convertAreaFile(inputPath, outputDir string) error {
	data, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	area, err := parseArea(string(data))
	if err != nil {
		return fmt.Errorf("parse area: %w", err)
	}

	return writeAreaToml(area, outputDir)
}

func parseArea(data string) (*Area, error) {
	area := &Area{}
	lines := strings.Split(data, "\n")

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		switch {
		case line == "#AREADATA":
			i = parseAreaData(lines, i+1, area)
		case line == "#AREA":
			i = parseOldArea(lines, i+1, area)
		case line == "#ROOMS":
			i = parseRooms(lines, i+1, area)
		case line == "#MOBILES":
			i = parseMobiles(lines, i+1, area)
		case line == "#OBJECTS":
			i = parseObjects(lines, i+1, area)
		case line == "#RESETS":
			i = parseResets(lines, i+1, area)
		case line == "#SHOPS":
			i = skipSection(lines, i+1)
		case line == "#SPECIALS":
			i = skipSection(lines, i+1)
		case line == "#MOBPROGS":
			i = skipSection(lines, i+1)
		case line == "#$":
			return area, nil
		default:
			i++
		}
	}

	return area, nil
}

func parseAreaData(lines []string, start int, area *Area) int {
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "End" || line == "" && i+1 < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i+1]), "#") {
			return i + 1
		}

		if strings.HasPrefix(line, "Name ") {
			area.Name = strings.TrimSuffix(strings.TrimPrefix(line, "Name "), "~")
		} else if strings.HasPrefix(line, "Credits ") {
			area.Credits = strings.TrimSuffix(strings.TrimPrefix(line, "Credits "), "~")
		} else if strings.HasPrefix(line, "VNUMs ") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				area.VnumMin, _ = strconv.Atoi(parts[1])
				area.VnumMax, _ = strconv.Atoi(parts[2])
			}
		}
		i++
	}
	return i
}

func parseOldArea(lines []string, start int, area *Area) int {
	// Old format: single line with credits
	if start < len(lines) {
		line := strings.TrimSuffix(strings.TrimSpace(lines[start]), "~")
		area.Credits = line
		// Try to extract name from credits
		if idx := strings.LastIndex(line, "}"); idx >= 0 {
			area.Name = strings.TrimSpace(line[idx+1:])
		} else {
			area.Name = line
		}
	}
	return start + 1
}

func parseRooms(lines []string, start int, area *Area) int {
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "#0" || line == "S" || strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#0") && len(line) > 1 && !isDigit(line[1]) {
			return i
		}

		if strings.HasPrefix(line, "#") && len(line) > 1 && isDigit(line[1]) {
			room := Room{}
			room.Vnum, _ = strconv.Atoi(line[1:])
			i++

			// Room name
			if i < len(lines) {
				room.Name = readTildeString(lines, &i)
			}

			// Room description
			if i < len(lines) {
				room.Description = readTildeString(lines, &i)
			}

			// Room flags line: area_number room_flags sector_type
			if i < len(lines) {
				flagLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(flagLine)
				if len(parts) >= 3 {
					room.Flags = parseRoomFlags(parts[1])
					room.Sector = parseSector(parts[2])
				}
				i++
			}

			// Parse exits and extra descriptions
			for i < len(lines) {
				line := strings.TrimSpace(lines[i])
				if line == "S" {
					i++
					break
				}

				if strings.HasPrefix(line, "D") {
					dir, _ := strconv.Atoi(line[1:])
					i++
					exit := Exit{Direction: directionName(dir)}

					// Exit description (skip it)
					readTildeString(lines, &i)

					// Exit keyword
					if i < len(lines) {
						exit.Keyword = readTildeString(lines, &i)
					}

					// Exit flags line: locks key to_room
					if i < len(lines) {
						exitLine := strings.TrimSpace(lines[i])
						parts := strings.Fields(exitLine)
						if len(parts) >= 3 {
							exit.Flags = parseExitFlags(parts[0])
							exit.Key, _ = strconv.Atoi(parts[1])
							exit.ToVnum, _ = strconv.Atoi(parts[2])
						}
						i++
					}

					room.Exits = append(room.Exits, exit)
				} else if line == "E" {
					i++
					ed := ExtraDesc{}
					if i < len(lines) {
						ed.Keywords = readTildeString(lines, &i)
					}
					if i < len(lines) {
						ed.Description = readTildeString(lines, &i)
					}
					room.ExtraDescs = append(room.ExtraDescs, ed)
				} else {
					i++
				}
			}

			area.Rooms = append(area.Rooms, room)
		} else {
			i++
		}
	}
	return i
}

func parseMobiles(lines []string, start int, area *Area) int {
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "#0" || (strings.HasPrefix(line, "#") && !isDigit(line[1])) {
			return i
		}

		if strings.HasPrefix(line, "#") && len(line) > 1 && isDigit(line[1]) {
			mob := Mobile{}
			mob.Vnum, _ = strconv.Atoi(line[1:])
			i++

			// Keywords
			if i < len(lines) {
				kw := readTildeString(lines, &i)
				mob.Keywords = strings.Fields(kw)
			}

			// Short desc
			if i < len(lines) {
				mob.ShortDesc = readTildeString(lines, &i)
			}

			// Long desc
			if i < len(lines) {
				mob.LongDesc = readTildeString(lines, &i)
			}

			// Description
			if i < len(lines) {
				mob.Description = readTildeString(lines, &i)
			}

			// Race
			if i < len(lines) {
				mob.Race = readTildeString(lines, &i)
			}

			// Act/aff/align/group line
			if i < len(lines) {
				flagLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(flagLine)
				if len(parts) >= 4 {
					mob.ActFlags = parseActFlags(parts[0])
					mob.AffFlags = parseAffFlags(parts[1])
					mob.Alignment, _ = strconv.Atoi(parts[2])
					// group is parts[3], skip
				}
				i++
			}

			// Level/hitroll/hit_dice/mana_dice/dam_dice/dam_type
			if i < len(lines) {
				statLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(statLine)
				if len(parts) >= 6 {
					mob.Level, _ = strconv.Atoi(parts[0])
					// hitroll is parts[1]
					mob.HitDice = parseDice(parts[2])
					mob.ManaDice = parseDice(parts[3])
					mob.DamageDice = parseDice(parts[4])
					mob.DamType = parts[5]
				}
				i++
			}

			// AC values
			if i < len(lines) {
				acLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(acLine)
				for j := 0; j < 4 && j < len(parts); j++ {
					mob.AC[j], _ = strconv.Atoi(parts[j])
				}
				i++
			}

			// Off/imm/res/vuln flags
			if i < len(lines) {
				flagLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(flagLine)
				if len(parts) >= 4 {
					mob.OffFlags = parseOffFlags(parts[0])
					mob.ImmFlags = parseImmFlags(parts[1])
					mob.ResFlags = parseResFlags(parts[2])
					mob.VulnFlags = parseVulnFlags(parts[3])
				}
				i++
			}

			// Start_pos default_pos sex gold
			if i < len(lines) {
				posLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(posLine)
				if len(parts) >= 4 {
					mob.StartPos = parts[0]
					mob.DefaultPos = parts[1]
					mob.Sex = parts[2]
					mob.Gold, _ = strconv.Atoi(parts[3])
				}
				i++
			}

			// Form/parts/size/material (or just form line in newer format)
			if i < len(lines) {
				formLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(formLine)
				// Look for size keyword
				for _, p := range parts {
					if p == "tiny" || p == "small" || p == "medium" || p == "large" || p == "huge" || p == "giant" {
						mob.Size = p
						break
					}
				}
				if mob.Size == "" {
					mob.Size = "medium"
				}
				i++
			}

			area.Mobiles = append(area.Mobiles, mob)
		} else {
			i++
		}
	}
	return i
}

func parseObjects(lines []string, start int, area *Area) int {
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "#0" || (strings.HasPrefix(line, "#") && len(line) > 1 && !isDigit(line[1])) {
			return i
		}

		if strings.HasPrefix(line, "#") && len(line) > 1 && isDigit(line[1]) {
			obj := Object{}
			obj.Vnum, _ = strconv.Atoi(line[1:])
			i++

			// Keywords
			if i < len(lines) {
				kw := readTildeString(lines, &i)
				obj.Keywords = strings.Fields(kw)
			}

			// Short desc
			if i < len(lines) {
				obj.ShortDesc = readTildeString(lines, &i)
			}

			// Long desc
			if i < len(lines) {
				obj.LongDesc = readTildeString(lines, &i)
			}

			// Material (skip)
			if i < len(lines) {
				readTildeString(lines, &i)
			}

			// Type/extra/wear
			if i < len(lines) {
				typeLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(typeLine)
				if len(parts) >= 3 {
					obj.ItemType = parts[0]
					obj.ExtraFlags = parseExtraFlags(parts[1])
					obj.WearFlags = parseWearFlags(parts[2])
				}
				i++
			}

			// Values
			if i < len(lines) {
				valLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(valLine)
				for j := 0; j < 5 && j < len(parts); j++ {
					// Handle tilde-terminated values
					p := strings.TrimSuffix(parts[j], "~")
					obj.Values[j], _ = strconv.Atoi(p)
				}
				i++
			}

			// Level/weight/cost/condition
			if i < len(lines) {
				miscLine := strings.TrimSpace(lines[i])
				parts := strings.Fields(miscLine)
				if len(parts) >= 4 {
					obj.Level, _ = strconv.Atoi(parts[0])
					obj.Weight, _ = strconv.Atoi(parts[1])
					obj.Cost, _ = strconv.Atoi(parts[2])
					obj.Condition = parts[3]
				}
				i++
			}

			// Parse affects and extra descriptions
			for i < len(lines) {
				line := strings.TrimSpace(lines[i])
				if line == "" || strings.HasPrefix(line, "#") {
					break
				}
				if line == "A" {
					i++
					if i < len(lines) {
						affLine := strings.TrimSpace(lines[i])
						parts := strings.Fields(affLine)
						if len(parts) >= 2 {
							aff := ObjAffect{}
							loc, _ := strconv.Atoi(parts[0])
							aff.Location = applyTypeName(loc)
							aff.Modifier, _ = strconv.Atoi(parts[1])
							obj.Affects = append(obj.Affects, aff)
						}
						i++
					}
				} else if line == "F" {
					// Flag affect line, skip
					i++
					if i < len(lines) {
						i++
					}
				} else if line == "E" {
					// Extra description, skip
					i++
					readTildeString(lines, &i)
					readTildeString(lines, &i)
				} else {
					break
				}
			}

			area.Objects = append(area.Objects, obj)
		} else {
			i++
		}
	}
	return i
}

func parseResets(lines []string, start int, area *Area) int {
	i := start
	lastRoom := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "S" || (strings.HasPrefix(line, "#") && line != "#RESETS") {
			return i + 1
		}

		if len(line) == 0 || line[0] == '*' {
			i++
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 1 {
			i++
			continue
		}

		cmd := parts[0]
		reset := Reset{Command: cmd}

		switch cmd {
		case "M": // Mobile reset
			if len(parts) >= 5 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // mob vnum
				reset.Arg2, _ = strconv.Atoi(parts[3]) // max count
				reset.Arg3, _ = strconv.Atoi(parts[4]) // room vnum
				reset.RoomVnum = reset.Arg3
				lastRoom = reset.Arg3
			}
		case "O": // Object reset
			if len(parts) >= 5 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // obj vnum
				reset.Arg3, _ = strconv.Atoi(parts[4]) // room vnum
				reset.RoomVnum = reset.Arg3
				lastRoom = reset.Arg3
			}
		case "P": // Put in container
			if len(parts) >= 5 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // obj vnum
				reset.Arg3, _ = strconv.Atoi(parts[4]) // container vnum
			}
		case "G": // Give to mobile
			if len(parts) >= 3 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // obj vnum
			}
			reset.RoomVnum = lastRoom
		case "E": // Equip mobile
			if len(parts) >= 4 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // obj vnum
				reset.Arg3, _ = strconv.Atoi(parts[4]) // wear location
			}
			reset.RoomVnum = lastRoom
		case "D": // Door reset
			if len(parts) >= 5 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // room vnum
				reset.Arg2, _ = strconv.Atoi(parts[3]) // direction
				reset.Arg3, _ = strconv.Atoi(parts[4]) // state
			}
		case "R": // Randomize exits
			if len(parts) >= 4 {
				reset.Arg1, _ = strconv.Atoi(parts[2]) // room vnum
				reset.Arg2, _ = strconv.Atoi(parts[3]) // num exits
			}
		}

		area.Resets = append(area.Resets, reset)
		i++
	}
	return i
}

func skipSection(lines []string, start int) int {
	i := start
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "S" || line == "#0" || (strings.HasPrefix(line, "#") && len(line) > 1 && !isDigit(line[1])) {
			return i + 1
		}
		i++
	}
	return i
}

func readTildeString(lines []string, i *int) string {
	var sb strings.Builder
	for *i < len(lines) {
		line := lines[*i]
		if idx := strings.Index(line, "~"); idx >= 0 {
			sb.WriteString(line[:idx])
			(*i)++
			break
		}
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(line)
		(*i)++
	}
	return strings.TrimSpace(sb.String())
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func directionName(d int) string {
	dirs := []string{"north", "east", "south", "west", "up", "down"}
	if d >= 0 && d < len(dirs) {
		return dirs[d]
	}
	return "north"
}

func parseDice(s string) Dice {
	// Format: NdS+B
	d := Dice{}
	re := regexp.MustCompile(`(\d+)d(\d+)\+?(-?\d+)?`)
	m := re.FindStringSubmatch(s)
	if len(m) >= 3 {
		d.Number, _ = strconv.Atoi(m[1])
		d.Size, _ = strconv.Atoi(m[2])
		if len(m) >= 4 && m[3] != "" {
			d.Bonus, _ = strconv.Atoi(m[3])
		}
	}
	return d
}

// Flag parsing functions - simplified versions
func parseRoomFlags(s string) []string {
	flags := []string{}
	flagMap := map[byte]string{
		'A': "dark", 'B': "no_mob", 'C': "indoors", 'D': "private",
		'E': "safe", 'F': "solitary", 'G': "pet_shop", 'H': "no_recall",
		'I': "imp_only", 'J': "gods_only", 'K': "heroes_only", 'L': "newbies_only",
		'M': "law", 'N': "nowhere",
	}
	for i := 0; i < len(s); i++ {
		if name, ok := flagMap[s[i]]; ok {
			flags = append(flags, name)
		}
	}
	return flags
}

func parseSector(s string) string {
	num, _ := strconv.Atoi(s)
	sectors := []string{"inside", "city", "field", "forest", "hills", "mountain", "water_swim", "water_noswim", "unused", "air", "desert"}
	if num >= 0 && num < len(sectors) {
		return sectors[num]
	}
	return "city"
}

func parseExitFlags(s string) []string {
	flags := []string{}
	num, _ := strconv.Atoi(s)
	if num&1 != 0 {
		flags = append(flags, "door")
	}
	if num&2 != 0 {
		flags = append(flags, "closed")
	}
	if num&4 != 0 {
		flags = append(flags, "locked")
	}
	if num&8 != 0 {
		flags = append(flags, "pickproof")
	}
	return flags
}

func parseActFlags(s string) []string {
	flags := []string{"npc"}
	flagMap := map[byte]string{
		'B': "sentinel", 'C': "scavenger", 'D': "key", 'F': "aggressive",
		'G': "stay_area", 'H': "wimpy", 'I': "pet", 'J': "train",
		'K': "practice", 'L': "undead", 'Q': "cleric", 'R': "mage",
		'S': "thief", 'T': "warrior", 'U': "no_align", 'V': "no_purge",
		'W': "outdoors", 'Y': "indoors", 'Z': "healer",
	}
	for i := 0; i < len(s); i++ {
		if name, ok := flagMap[s[i]]; ok {
			flags = append(flags, name)
		}
	}
	return flags
}

func parseAffFlags(s string) []string {
	flags := []string{}
	flagMap := map[byte]string{
		'A': "blind", 'B': "invisible", 'C': "detect_evil", 'D': "detect_invis",
		'E': "detect_magic", 'F': "detect_hidden", 'G': "detect_good", 'H': "sanctuary",
		'I': "faerie_fire", 'J': "infrared", 'K': "curse", 'L': "flaming",
		'M': "poison", 'N': "protect_evil", 'O': "protect_good", 'P': "sneak",
		'Q': "hide", 'R': "sleep", 'S': "charm", 'T': "flying",
		'U': "pass_door", 'V': "haste", 'W': "calm", 'X': "plague",
		'Y': "weaken", 'Z': "dark_vision",
	}
	for i := 0; i < len(s); i++ {
		if name, ok := flagMap[s[i]]; ok {
			flags = append(flags, name)
		}
	}
	return flags
}

func parseOffFlags(s string) []string   { return []string{} }
func parseImmFlags(s string) []string   { return []string{} }
func parseResFlags(s string) []string   { return []string{} }
func parseVulnFlags(s string) []string  { return []string{} }
func parseExtraFlags(s string) []string { return parseFlagLetters(s, extraFlagMap) }
func parseWearFlags(s string) []string  { return parseFlagLetters(s, wearFlagMap) }

// wearFlagMap maps ROM letter codes to wear flag names
var wearFlagMap = map[rune]string{
	'A': "take",
	'B': "finger",
	'C': "neck",
	'D': "body",
	'E': "head",
	'F': "legs",
	'G': "feet",
	'H': "hands",
	'I': "arms",
	'J': "shield",
	'K': "about",
	'L': "waist",
	'M': "wrist",
	'N': "wield",
	'O': "hold",
	'P': "no_sac",
	'Q': "float",
	'R': "face",
}

// extraFlagMap maps ROM letter codes to extra flag names
var extraFlagMap = map[rune]string{
	'A': "glow",
	'B': "hum",
	'C': "dark",
	'D': "lock",
	'E': "evil",
	'F': "invis",
	'G': "magic",
	'H': "nodrop",
	'I': "bless",
	'J': "anti_good",
	'K': "anti_evil",
	'L': "anti_neutral",
	'M': "noremove",
	'N': "inventory",
	'O': "nopurge",
	'P': "rot_death",
	'Q': "vis_death",
	'R': "nonmetal",
	'S': "nolocate",
	'T': "melt_drop",
	'U': "had_timer",
	'V': "sell_extract",
	'W': "burn_proof",
	'X': "nouncurse",
}

// parseFlagLetters converts ROM letter-coded flags to string names
func parseFlagLetters(s string, flagMap map[rune]string) []string {
	var flags []string
	for _, c := range s {
		if name, ok := flagMap[c]; ok {
			flags = append(flags, name)
		}
	}
	return flags
}

func applyTypeName(n int) string {
	names := []string{
		"none", "str", "dex", "int", "wis", "con",
		"sex", "class", "level", "age", "height", "weight",
		"mana", "hit", "move", "gold", "exp", "ac",
		"hitroll", "damroll", "saves", "save_para", "save_rod",
		"save_petri", "save_breath", "save_spell",
	}
	if n >= 0 && n < len(names) {
		return names[n]
	}
	return "none"
}

func writeAreaToml(area *Area, outputDir string) error {
	// Create directory structure
	if err := os.MkdirAll(filepath.Join(outputDir, "rooms"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "mobs"), 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "objects"), 0755); err != nil {
		return err
	}

	// Write area.toml
	areaFile, err := os.Create(filepath.Join(outputDir, "area.toml"))
	if err != nil {
		return err
	}
	defer areaFile.Close()

	// Clean name for ID
	id := strings.ToLower(strings.ReplaceAll(area.Name, " ", "_"))
	id = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(id, "")
	if id == "" {
		id = "area"
	}

	fmt.Fprintf(areaFile, "id = %q\n", id)
	fmt.Fprintf(areaFile, "name = %q\n", area.Name)
	fmt.Fprintf(areaFile, "credits = %q\n", area.Credits)
	fmt.Fprintf(areaFile, "reset_interval = 120\n\n")
	fmt.Fprintf(areaFile, "[vnum_range]\n")
	fmt.Fprintf(areaFile, "min = %d\n", area.VnumMin)
	fmt.Fprintf(areaFile, "max = %d\n", area.VnumMax)

	// Build room reset map from resets
	roomResets := make(map[int][]Reset)
	for _, r := range area.Resets {
		if r.Command == "M" || r.Command == "O" {
			roomResets[r.RoomVnum] = append(roomResets[r.RoomVnum], r)
		}
	}

	// Write rooms
	if len(area.Rooms) > 0 {
		roomsFile, err := os.Create(filepath.Join(outputDir, "rooms", "rooms.toml"))
		if err != nil {
			return err
		}
		defer roomsFile.Close()

		for _, room := range area.Rooms {
			fmt.Fprintf(roomsFile, "[[rooms]]\n")
			fmt.Fprintf(roomsFile, "vnum = %d\n", room.Vnum)
			fmt.Fprintf(roomsFile, "name = %q\n", room.Name)
			fmt.Fprintf(roomsFile, "sector = %q\n", room.Sector)
			if len(room.Flags) > 0 {
				fmt.Fprintf(roomsFile, "room_flags = %s\n", toTomlArray(room.Flags))
			}
			fmt.Fprintf(roomsFile, "description = %s\n", toTomlMultiline(room.Description))

			for _, exit := range room.Exits {
				fmt.Fprintf(roomsFile, "\n  [[rooms.exits]]\n")
				fmt.Fprintf(roomsFile, "  direction = %q\n", exit.Direction)
				fmt.Fprintf(roomsFile, "  to_vnum = %d\n", exit.ToVnum)
				if exit.Key > 0 {
					fmt.Fprintf(roomsFile, "  key = %d\n", exit.Key)
				}
				if len(exit.Flags) > 0 {
					fmt.Fprintf(roomsFile, "  flags = %s\n", toTomlArray(exit.Flags))
				}
			}

			// Add mob/obj resets
			if resets, ok := roomResets[room.Vnum]; ok {
				for _, r := range resets {
					if r.Command == "M" {
						fmt.Fprintf(roomsFile, "\n  [[rooms.mob_resets]]\n")
						fmt.Fprintf(roomsFile, "  vnum = %d\n", r.Arg1)
						fmt.Fprintf(roomsFile, "  max = %d\n", r.Arg2)
					} else if r.Command == "O" {
						fmt.Fprintf(roomsFile, "\n  [[rooms.obj_resets]]\n")
						fmt.Fprintf(roomsFile, "  vnum = %d\n", r.Arg1)
						fmt.Fprintf(roomsFile, "  max = 1\n")
					}
				}
			}

			fmt.Fprintf(roomsFile, "\n")
		}
	}

	// Write mobiles
	if len(area.Mobiles) > 0 {
		mobsFile, err := os.Create(filepath.Join(outputDir, "mobs", "mobs.toml"))
		if err != nil {
			return err
		}
		defer mobsFile.Close()

		for _, mob := range area.Mobiles {
			fmt.Fprintf(mobsFile, "[[mobiles]]\n")
			fmt.Fprintf(mobsFile, "vnum = %d\n", mob.Vnum)
			fmt.Fprintf(mobsFile, "keywords = %s\n", toTomlArray(mob.Keywords))
			fmt.Fprintf(mobsFile, "short_desc = %q\n", mob.ShortDesc)
			fmt.Fprintf(mobsFile, "long_desc = %q\n", mob.LongDesc)
			if mob.Description != "" {
				fmt.Fprintf(mobsFile, "description = %s\n", toTomlMultiline(mob.Description))
			}
			fmt.Fprintf(mobsFile, "level = %d\n", mob.Level)
			fmt.Fprintf(mobsFile, "sex = %q\n", mob.Sex)
			if len(mob.ActFlags) > 0 {
				fmt.Fprintf(mobsFile, "act_flags = %s\n", toTomlArray(mob.ActFlags))
			}
			if len(mob.AffFlags) > 0 {
				fmt.Fprintf(mobsFile, "affected_by = %s\n", toTomlArray(mob.AffFlags))
			}
			fmt.Fprintf(mobsFile, "alignment = %d\n", mob.Alignment)

			fmt.Fprintf(mobsFile, "\n  [mobiles.hit_dice]\n")
			fmt.Fprintf(mobsFile, "  number = %d\n", mob.HitDice.Number)
			fmt.Fprintf(mobsFile, "  size = %d\n", mob.HitDice.Size)
			fmt.Fprintf(mobsFile, "  bonus = %d\n", mob.HitDice.Bonus)

			if mob.DamageDice.Number > 0 {
				fmt.Fprintf(mobsFile, "\n  [mobiles.damage_dice]\n")
				fmt.Fprintf(mobsFile, "  number = %d\n", mob.DamageDice.Number)
				fmt.Fprintf(mobsFile, "  size = %d\n", mob.DamageDice.Size)
				fmt.Fprintf(mobsFile, "  bonus = %d\n", mob.DamageDice.Bonus)
			}

			fmt.Fprintf(mobsFile, "\n")
		}
	}

	// Write objects
	if len(area.Objects) > 0 {
		objsFile, err := os.Create(filepath.Join(outputDir, "objects", "objects.toml"))
		if err != nil {
			return err
		}
		defer objsFile.Close()

		for _, obj := range area.Objects {
			fmt.Fprintf(objsFile, "[[objects]]\n")
			fmt.Fprintf(objsFile, "vnum = %d\n", obj.Vnum)
			fmt.Fprintf(objsFile, "keywords = %s\n", toTomlArray(obj.Keywords))
			fmt.Fprintf(objsFile, "short_desc = %q\n", obj.ShortDesc)
			fmt.Fprintf(objsFile, "long_desc = %q\n", obj.LongDesc)
			fmt.Fprintf(objsFile, "item_type = %q\n", obj.ItemType)
			fmt.Fprintf(objsFile, "level = %d\n", obj.Level)
			fmt.Fprintf(objsFile, "weight = %d\n", obj.Weight)
			fmt.Fprintf(objsFile, "cost = %d\n", obj.Cost)
			if len(obj.WearFlags) > 0 {
				fmt.Fprintf(objsFile, "wear_flags = %s\n", toTomlArray(obj.WearFlags))
			}
			if len(obj.ExtraFlags) > 0 {
				fmt.Fprintf(objsFile, "extra_flags = %s\n", toTomlArray(obj.ExtraFlags))
			}

			// Write values for weapons
			if obj.ItemType == "weapon" {
				fmt.Fprintf(objsFile, "\n  [objects.weapon]\n")
				fmt.Fprintf(objsFile, "  dice_number = %d\n", obj.Values[1])
				fmt.Fprintf(objsFile, "  dice_size = %d\n", obj.Values[2])
			}

			// Write affects
			for _, aff := range obj.Affects {
				fmt.Fprintf(objsFile, "\n  [[objects.affects]]\n")
				fmt.Fprintf(objsFile, "  location = %q\n", aff.Location)
				fmt.Fprintf(objsFile, "  modifier = %d\n", aff.Modifier)
			}

			fmt.Fprintf(objsFile, "\n")
		}
	}

	return nil
}

func toTomlArray(strs []string) string {
	if len(strs) == 0 {
		return "[]"
	}
	quoted := make([]string, len(strs))
	for i, s := range strs {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func toTomlMultiline(s string) string {
	if strings.Contains(s, "\n") || len(s) > 60 {
		return `"""` + "\n" + s + `"""`
	}
	return fmt.Sprintf("%q", s)
}
