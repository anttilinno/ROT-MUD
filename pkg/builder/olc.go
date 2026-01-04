// Package builder implements the Online Creation (OLC) system
// for building and editing game content in-game
package builder

import (
	"strconv"
	"strings"

	"rotmud/pkg/types"
)

// EditorMode represents what type of entity is being edited
type EditorMode int

const (
	EditorNone EditorMode = iota
	EditorArea
	EditorRoom
	EditorMobile
	EditorObject
	EditorReset
	EditorHelp
	EditorSocial
)

// EditorState tracks the current editing state for a player
type EditorState struct {
	Mode     EditorMode  // What we're editing
	EditVnum int         // Vnum of what we're editing
	Modified bool        // Has changes
	Data     interface{} // The data being edited (Room, Mobile, Object, etc.)
}

// OLCSystem manages the online creation system
type OLCSystem struct {
	Output     func(ch *types.Character, msg string)
	SaveFunc   func(mode EditorMode, vnum int, data interface{}) error
	GetRoom    func(vnum int) *types.Room
	GetMobile  func(vnum int) *MobileTemplate
	GetObject  func(vnum int) *ObjectTemplate
	CreateRoom func(vnum int) *types.Room
}

// MobileTemplate represents a mob template for editing
type MobileTemplate struct {
	Vnum        int
	Keywords    []string
	ShortDesc   string
	LongDesc    string
	Description string
	Level       int
	Sex         string
	Race        string
	Alignment   int
	ActFlags    types.ActFlags
	AffectedBy  types.AffectFlags
	HitDice     [3]int // number, size, bonus
	ManaDice    [3]int
	DamageDice  [3]int
	DamageType  string
	AC          [4]int
	Hitroll     int
	Gold        int
	StartPos    string
	DefaultPos  string
	Special     string
}

// ObjectTemplate represents an object template for editing
type ObjectTemplate struct {
	Vnum       int
	Keywords   []string
	ShortDesc  string
	LongDesc   string
	ItemType   types.ItemType
	Level      int
	Weight     int
	Cost       int
	ExtraFlags types.ItemFlags
	WearFlags  types.WearFlags
	Values     [5]int
	Material   string
}

// NewOLCSystem creates a new OLC system
func NewOLCSystem() *OLCSystem {
	return &OLCSystem{}
}

// StartEditing begins an editing session
func (o *OLCSystem) StartEditing(ch *types.Character, mode EditorMode, vnum int) bool {
	if ch.PCData == nil {
		o.send(ch, "Only players can use the editor.\r\n")
		return false
	}

	// Check builder permissions
	if ch.PCData.Security < 1 {
		o.send(ch, "You don't have permission to build.\r\n")
		return false
	}

	// Store edit state in PCData
	state := &EditorState{
		Mode:     mode,
		EditVnum: vnum,
		Modified: false,
	}

	// Load the data being edited
	switch mode {
	case EditorRoom:
		room := o.GetRoom(vnum)
		if room == nil {
			o.send(ch, "That room doesn't exist. Use 'create' to make a new one.\r\n")
			return false
		}
		state.Data = room

	case EditorMobile:
		mob := o.GetMobile(vnum)
		if mob == nil {
			o.send(ch, "That mobile doesn't exist.\r\n")
			return false
		}
		state.Data = mob

	case EditorObject:
		obj := o.GetObject(vnum)
		if obj == nil {
			o.send(ch, "That object doesn't exist.\r\n")
			return false
		}
		state.Data = obj

	default:
		o.send(ch, "Invalid editor mode.\r\n")
		return false
	}

	// Store in descriptor or a custom field
	// For now, we'll use a simple approach
	o.send(ch, "Entering editor mode.\r\n")
	return true
}

// ProcessCommand handles an OLC command
func (o *OLCSystem) ProcessCommand(ch *types.Character, state *EditorState, input string) bool {
	if state == nil || state.Mode == EditorNone {
		return false
	}

	parts := strings.SplitN(strings.TrimSpace(input), " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch state.Mode {
	case EditorRoom:
		return o.processRoomCommand(ch, state, cmd, args)
	case EditorMobile:
		return o.processMobileCommand(ch, state, cmd, args)
	case EditorObject:
		return o.processObjectCommand(ch, state, cmd, args)
	}

	return false
}

// === Room Editor Commands ===

func (o *OLCSystem) processRoomCommand(ch *types.Character, state *EditorState, cmd, args string) bool {
	room, ok := state.Data.(*types.Room)
	if !ok {
		return false
	}

	switch cmd {
	case "show":
		o.showRoom(ch, room)
		return true

	case "name":
		if args == "" {
			o.send(ch, "Syntax: name <room name>\r\n")
			return true
		}
		room.Name = args
		state.Modified = true
		o.send(ch, "Room name set.\r\n")
		return true

	case "desc", "description":
		if args == "" {
			o.send(ch, "Syntax: desc <description>\r\n")
			return true
		}
		room.Description = args
		state.Modified = true
		o.send(ch, "Room description set.\r\n")
		return true

	case "sector":
		if args == "" {
			o.send(ch, "Syntax: sector <type>\r\n")
			o.send(ch, "Types: inside city field forest hills mountain water_swim water_noswim air desert\r\n")
			return true
		}
		// Set sector type
		sector := parseSector(args)
		if sector < 0 {
			o.send(ch, "Invalid sector type.\r\n")
			return true
		}
		room.Sector = types.Sector(sector)
		state.Modified = true
		o.send(ch, "Sector type set.\r\n")
		return true

	case "north", "south", "east", "west", "up", "down":
		return o.setRoomExit(ch, state, room, cmd, args)

	case "flags":
		if args == "" {
			o.showRoomFlags(ch, room)
			return true
		}
		o.toggleRoomFlag(ch, state, room, args)
		return true

	case "done":
		if state.Modified {
			if o.SaveFunc != nil {
				if err := o.SaveFunc(EditorRoom, room.Vnum, room); err != nil {
					o.send(ch, "Error saving: "+err.Error()+"\r\n")
				} else {
					o.send(ch, "Room saved.\r\n")
				}
			}
		}
		state.Mode = EditorNone
		o.send(ch, "Exiting room editor.\r\n")
		return true

	case "?", "help", "commands":
		o.send(ch, "Room editor commands:\r\n")
		o.send(ch, "  show              - Display room info\r\n")
		o.send(ch, "  name <name>       - Set room name\r\n")
		o.send(ch, "  desc <desc>       - Set description\r\n")
		o.send(ch, "  sector <type>     - Set sector type\r\n")
		o.send(ch, "  north/south/etc   - Set exits\r\n")
		o.send(ch, "  flags [flag]      - Show/toggle flags\r\n")
		o.send(ch, "  done              - Save and exit\r\n")
		return true
	}

	return false
}

func (o *OLCSystem) showRoom(ch *types.Character, room *types.Room) {
	o.send(ch, "=== Room Editor ===\r\n")
	o.send(ch, "Vnum:        "+itoa(room.Vnum)+"\r\n")
	o.send(ch, "Name:        "+room.Name+"\r\n")
	o.send(ch, "Sector:      "+room.Sector.String()+"\r\n")
	o.send(ch, "Description:\r\n"+room.Description+"\r\n")

	// Show exits
	o.send(ch, "Exits:\r\n")
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := room.GetExit(dir)
		if exit != nil {
			o.send(ch, "  "+dir.String()+": vnum "+itoa(exit.ToVnum)+"\r\n")
		}
	}
}

func (o *OLCSystem) setRoomExit(ch *types.Character, state *EditorState, room *types.Room, dir, args string) bool {
	direction := parseDirection(dir)
	if direction < 0 {
		return false
	}

	if args == "" {
		o.send(ch, "Syntax: "+dir+" <vnum> | delete\r\n")
		return true
	}

	if strings.ToLower(args) == "delete" {
		room.SetExit(types.Direction(direction), nil)
		state.Modified = true
		o.send(ch, "Exit deleted.\r\n")
		return true
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		o.send(ch, "Invalid vnum.\r\n")
		return true
	}

	exit := &types.Exit{
		ToVnum: vnum,
		ToRoom: o.GetRoom(vnum),
	}
	room.SetExit(types.Direction(direction), exit)
	state.Modified = true
	o.send(ch, "Exit set.\r\n")
	return true
}

func (o *OLCSystem) showRoomFlags(ch *types.Character, room *types.Room) {
	o.send(ch, "Room flags: "+formatRoomFlags(room.Flags)+"\r\n")
	o.send(ch, "Available: dark nomob indoors private safe solitary norecall law\r\n")
}

func (o *OLCSystem) toggleRoomFlag(ch *types.Character, state *EditorState, room *types.Room, flag string) {
	f := parseRoomFlag(flag)
	if f == 0 {
		o.send(ch, "Unknown flag: "+flag+"\r\n")
		return
	}
	room.Flags.Toggle(f)
	state.Modified = true
	if room.Flags.Has(f) {
		o.send(ch, "Flag "+flag+" enabled.\r\n")
	} else {
		o.send(ch, "Flag "+flag+" disabled.\r\n")
	}
}

// === Mobile Editor Commands ===

func (o *OLCSystem) processMobileCommand(ch *types.Character, state *EditorState, cmd, args string) bool {
	mob, ok := state.Data.(*MobileTemplate)
	if !ok {
		return false
	}

	switch cmd {
	case "show":
		o.showMobile(ch, mob)
		return true

	case "name":
		if args == "" {
			o.send(ch, "Syntax: name <keywords>\r\n")
			return true
		}
		mob.Keywords = strings.Fields(args)
		state.Modified = true
		o.send(ch, "Keywords set.\r\n")
		return true

	case "short":
		if args == "" {
			o.send(ch, "Syntax: short <description>\r\n")
			return true
		}
		mob.ShortDesc = args
		state.Modified = true
		o.send(ch, "Short description set.\r\n")
		return true

	case "long":
		if args == "" {
			o.send(ch, "Syntax: long <description>\r\n")
			return true
		}
		mob.LongDesc = args
		state.Modified = true
		o.send(ch, "Long description set.\r\n")
		return true

	case "level":
		level, err := strconv.Atoi(args)
		if err != nil || level < 1 || level > 200 {
			o.send(ch, "Syntax: level <1-200>\r\n")
			return true
		}
		mob.Level = level
		state.Modified = true
		o.send(ch, "Level set to "+itoa(level)+".\r\n")
		return true

	case "align", "alignment":
		align, err := strconv.Atoi(args)
		if err != nil || align < -1000 || align > 1000 {
			o.send(ch, "Syntax: align <-1000 to 1000>\r\n")
			return true
		}
		mob.Alignment = align
		state.Modified = true
		o.send(ch, "Alignment set to "+itoa(align)+".\r\n")
		return true

	case "special", "spec":
		if args == "" {
			o.send(ch, "Current special: "+mob.Special+"\r\n")
			o.send(ch, "Syntax: special <name> | none\r\n")
			return true
		}
		if strings.ToLower(args) == "none" {
			mob.Special = ""
		} else {
			mob.Special = args
		}
		state.Modified = true
		o.send(ch, "Special set.\r\n")
		return true

	case "done":
		if state.Modified {
			if o.SaveFunc != nil {
				if err := o.SaveFunc(EditorMobile, mob.Vnum, mob); err != nil {
					o.send(ch, "Error saving: "+err.Error()+"\r\n")
				} else {
					o.send(ch, "Mobile saved.\r\n")
				}
			}
		}
		state.Mode = EditorNone
		o.send(ch, "Exiting mobile editor.\r\n")
		return true

	case "?", "help", "commands":
		o.send(ch, "Mobile editor commands:\r\n")
		o.send(ch, "  show              - Display mobile info\r\n")
		o.send(ch, "  name <keywords>   - Set keywords\r\n")
		o.send(ch, "  short <desc>      - Set short description\r\n")
		o.send(ch, "  long <desc>       - Set long description\r\n")
		o.send(ch, "  level <1-200>     - Set level\r\n")
		o.send(ch, "  align <-1000 to 1000> - Set alignment\r\n")
		o.send(ch, "  special <name>    - Set special function\r\n")
		o.send(ch, "  done              - Save and exit\r\n")
		return true
	}

	return false
}

func (o *OLCSystem) showMobile(ch *types.Character, mob *MobileTemplate) {
	o.send(ch, "=== Mobile Editor ===\r\n")
	o.send(ch, "Vnum:        "+itoa(mob.Vnum)+"\r\n")
	o.send(ch, "Keywords:    "+strings.Join(mob.Keywords, " ")+"\r\n")
	o.send(ch, "Short:       "+mob.ShortDesc+"\r\n")
	o.send(ch, "Long:        "+mob.LongDesc+"\r\n")
	o.send(ch, "Level:       "+itoa(mob.Level)+"\r\n")
	o.send(ch, "Alignment:   "+itoa(mob.Alignment)+"\r\n")
	o.send(ch, "Special:     "+mob.Special+"\r\n")
}

// === Object Editor Commands ===

func (o *OLCSystem) processObjectCommand(ch *types.Character, state *EditorState, cmd, args string) bool {
	obj, ok := state.Data.(*ObjectTemplate)
	if !ok {
		return false
	}

	switch cmd {
	case "show":
		o.showObject(ch, obj)
		return true

	case "name":
		if args == "" {
			o.send(ch, "Syntax: name <keywords>\r\n")
			return true
		}
		obj.Keywords = strings.Fields(args)
		state.Modified = true
		o.send(ch, "Keywords set.\r\n")
		return true

	case "short":
		if args == "" {
			o.send(ch, "Syntax: short <description>\r\n")
			return true
		}
		obj.ShortDesc = args
		state.Modified = true
		o.send(ch, "Short description set.\r\n")
		return true

	case "long":
		if args == "" {
			o.send(ch, "Syntax: long <description>\r\n")
			return true
		}
		obj.LongDesc = args
		state.Modified = true
		o.send(ch, "Long description set.\r\n")
		return true

	case "level":
		level, err := strconv.Atoi(args)
		if err != nil || level < 0 || level > 200 {
			o.send(ch, "Syntax: level <0-200>\r\n")
			return true
		}
		obj.Level = level
		state.Modified = true
		o.send(ch, "Level set to "+itoa(level)+".\r\n")
		return true

	case "cost":
		cost, err := strconv.Atoi(args)
		if err != nil || cost < 0 {
			o.send(ch, "Syntax: cost <amount>\r\n")
			return true
		}
		obj.Cost = cost
		state.Modified = true
		o.send(ch, "Cost set to "+itoa(cost)+".\r\n")
		return true

	case "weight":
		weight, err := strconv.Atoi(args)
		if err != nil || weight < 0 {
			o.send(ch, "Syntax: weight <amount>\r\n")
			return true
		}
		obj.Weight = weight
		state.Modified = true
		o.send(ch, "Weight set to "+itoa(weight)+".\r\n")
		return true

	case "done":
		if state.Modified {
			if o.SaveFunc != nil {
				if err := o.SaveFunc(EditorObject, obj.Vnum, obj); err != nil {
					o.send(ch, "Error saving: "+err.Error()+"\r\n")
				} else {
					o.send(ch, "Object saved.\r\n")
				}
			}
		}
		state.Mode = EditorNone
		o.send(ch, "Exiting object editor.\r\n")
		return true

	case "?", "help", "commands":
		o.send(ch, "Object editor commands:\r\n")
		o.send(ch, "  show              - Display object info\r\n")
		o.send(ch, "  name <keywords>   - Set keywords\r\n")
		o.send(ch, "  short <desc>      - Set short description\r\n")
		o.send(ch, "  long <desc>       - Set long description\r\n")
		o.send(ch, "  level <0-200>     - Set level\r\n")
		o.send(ch, "  cost <amount>     - Set cost\r\n")
		o.send(ch, "  weight <amount>   - Set weight\r\n")
		o.send(ch, "  done              - Save and exit\r\n")
		return true
	}

	return false
}

func (o *OLCSystem) showObject(ch *types.Character, obj *ObjectTemplate) {
	o.send(ch, "=== Object Editor ===\r\n")
	o.send(ch, "Vnum:        "+itoa(obj.Vnum)+"\r\n")
	o.send(ch, "Keywords:    "+strings.Join(obj.Keywords, " ")+"\r\n")
	o.send(ch, "Short:       "+obj.ShortDesc+"\r\n")
	o.send(ch, "Long:        "+obj.LongDesc+"\r\n")
	o.send(ch, "Type:        "+obj.ItemType.String()+"\r\n")
	o.send(ch, "Level:       "+itoa(obj.Level)+"\r\n")
	o.send(ch, "Cost:        "+itoa(obj.Cost)+"\r\n")
	o.send(ch, "Weight:      "+itoa(obj.Weight)+"\r\n")
}

// send outputs a message to a character
func (o *OLCSystem) send(ch *types.Character, msg string) {
	if o.Output != nil {
		o.Output(ch, msg)
	}
}

// === Helper functions ===

func itoa(n int) string {
	return strconv.Itoa(n)
}

func parseDirection(dir string) int {
	switch strings.ToLower(dir) {
	case "north", "n":
		return int(types.DirNorth)
	case "south", "s":
		return int(types.DirSouth)
	case "east", "e":
		return int(types.DirEast)
	case "west", "w":
		return int(types.DirWest)
	case "up", "u":
		return int(types.DirUp)
	case "down", "d":
		return int(types.DirDown)
	}
	return -1
}

func parseSector(s string) int {
	switch strings.ToLower(s) {
	case "inside":
		return int(types.SectInside)
	case "city":
		return int(types.SectCity)
	case "field":
		return int(types.SectField)
	case "forest":
		return int(types.SectForest)
	case "hills":
		return int(types.SectHills)
	case "mountain":
		return int(types.SectMountain)
	case "water_swim", "swim":
		return int(types.SectWaterSwim)
	case "water_noswim", "noswim":
		return int(types.SectWaterNoSwim)
	case "air":
		return int(types.SectAir)
	case "desert":
		return int(types.SectDesert)
	}
	return -1
}

func parseRoomFlag(flag string) types.RoomFlags {
	switch strings.ToLower(flag) {
	case "dark":
		return types.RoomDark
	case "nomob":
		return types.RoomNoMob
	case "indoors":
		return types.RoomIndoors
	case "private":
		return types.RoomPrivate
	case "safe":
		return types.RoomSafe
	case "solitary":
		return types.RoomSolitary
	case "norecall":
		return types.RoomNoRecall
	case "law":
		return types.RoomLaw
	}
	return 0
}

func formatRoomFlags(flags types.RoomFlags) string {
	var parts []string
	if flags.Has(types.RoomDark) {
		parts = append(parts, "dark")
	}
	if flags.Has(types.RoomNoMob) {
		parts = append(parts, "nomob")
	}
	if flags.Has(types.RoomIndoors) {
		parts = append(parts, "indoors")
	}
	if flags.Has(types.RoomPrivate) {
		parts = append(parts, "private")
	}
	if flags.Has(types.RoomSafe) {
		parts = append(parts, "safe")
	}
	if flags.Has(types.RoomSolitary) {
		parts = append(parts, "solitary")
	}
	if flags.Has(types.RoomNoRecall) {
		parts = append(parts, "norecall")
	}
	if flags.Has(types.RoomLaw) {
		parts = append(parts, "law")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}
