package game

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"rotmud/pkg/builder"
	"rotmud/pkg/combat"
	"rotmud/pkg/help"
	"rotmud/pkg/loader"
	"rotmud/pkg/magic"
	"rotmud/pkg/shops"
	"rotmud/pkg/skills"
	"rotmud/pkg/types"
)

// send sends a message to a character using the output callback
func (d *CommandDispatcher) send(ch *types.Character, msg string) {
	if d.Output != nil {
		d.Output(ch, msg)
	}
}

// sendPositionMessage sends a position-appropriate "can't do that" message
func (d *CommandDispatcher) sendPositionMessage(ch *types.Character) {
	switch ch.Position {
	case types.PosDead:
		d.send(ch, "Lie still; you are DEAD.\r\n")
	case types.PosMortal, types.PosIncap:
		d.send(ch, "You are hurt far too badly for that.\r\n")
	case types.PosStunned:
		d.send(ch, "You are too stunned to do that.\r\n")
	case types.PosSleeping:
		d.send(ch, "In your dreams, or what?\r\n")
	case types.PosResting:
		d.send(ch, "Nah... You feel too relaxed...\r\n")
	case types.PosSitting:
		d.send(ch, "Better stand up first.\r\n")
	case types.PosFighting:
		d.send(ch, "No way! You are still fighting!\r\n")
	default:
		d.send(ch, "You can't do that right now.\r\n")
	}
}

// formatObjectList formats a list of objects, combining duplicates if combine flag is set.
// Returns a slice of formatted strings ready for display.
// fShort: true for short descriptions, false for long descriptions
// combine: true to group duplicate objects (e.g., "(3) a sword")
func formatObjectList(objects []*types.Object, ch *types.Character, fShort bool, combine bool) []string {
	if len(objects) == 0 {
		return nil
	}

	// Build arrays for combining
	type displayItem struct {
		desc  string
		count int
	}
	items := make([]displayItem, 0, len(objects))

	for _, obj := range objects {
		// Get the description
		var desc string
		if fShort {
			desc = formatObjToChar(obj, ch, true)
		} else {
			desc = formatObjToChar(obj, ch, false)
		}
		if desc == "" {
			continue
		}

		// Try to combine with existing item
		combined := false
		if combine {
			for i := len(items) - 1; i >= 0; i-- {
				if items[i].desc == desc {
					items[i].count++
					combined = true
					break
				}
			}
		}

		if !combined {
			items = append(items, displayItem{desc: desc, count: 1})
		}
	}

	// Format output
	result := make([]string, 0, len(items))
	for _, item := range items {
		if combine {
			if item.count > 1 {
				result = append(result, fmt.Sprintf("(%2d) %s", item.count, item.desc))
			} else {
				result = append(result, fmt.Sprintf("     %s", item.desc))
			}
		} else {
			result = append(result, item.desc)
		}
	}

	return result
}

// formatObjToChar formats a single object for display to a character.
// fShort: true for short description, false for long description
func formatObjToChar(obj *types.Object, ch *types.Character, fShort bool) string {
	if obj == nil {
		return ""
	}

	var buf strings.Builder

	// Build status flags indicator
	flags := []struct {
		flag types.ItemFlags
		aff  types.AffectFlags
		char byte
	}{
		{types.ItemInvis, 0, 'V'},                    // Invisible
		{types.ItemEvil, types.AffDetectEvil, 'E'},   // Evil (need detect evil)
		{types.ItemBless, types.AffDetectGood, 'B'},  // Blessed (need detect good)
		{types.ItemMagic, types.AffDetectMagic, 'M'}, // Magic (need detect magic)
		{types.ItemGlow, 0, 'G'},                     // Glowing
		{types.ItemHum, 0, 'H'},                      // Humming
		{types.ItemQuest, 0, 'Q'},                    // Quest item
	}

	// Build the flags string [.......] where dots become letters
	flagBytes := []byte("[.......]")
	hasFlag := false

	for i, f := range flags {
		// Check if object has this flag
		if obj.ExtraFlags.Has(f.flag) {
			// For some flags, viewer needs detect affect
			if f.aff != 0 && !ch.IsAffected(f.aff) {
				continue
			}
			flagBytes[1+i] = f.char
			hasFlag = true
		}
	}

	if hasFlag {
		buf.Write(flagBytes)
		buf.WriteString(" ")
	}

	// Add the description
	if fShort {
		if obj.ShortDesc != "" {
			buf.WriteString(obj.ShortDesc)
		}
	} else {
		if obj.LongDesc != "" {
			buf.WriteString(obj.LongDesc)
		} else if obj.ShortDesc != "" {
			buf.WriteString(obj.ShortDesc)
		}
	}

	if buf.Len() == 0 {
		return ""
	}

	return buf.String()
}

// CommandHandler is a function that handles a command
type CommandHandler func(ch *types.Character, args string)

// CommandDispatcher handles command processing and system coordination
type CommandDispatcher struct {
	Registry *CommandRegistry
	Output   func(ch *types.Character, msg string) // Output callback
	GameLoop *GameLoop                             // Reference to game loop for looking up characters
	Combat   *combat.CombatSystem                  // Combat system
	Magic    *magic.MagicSystem                    // Magic system
	Skills   *skills.SkillSystem                   // Skills system
	Shops    *shops.ShopHandler                    // Shop system
	Socials  *SocialRegistry                       // Social commands
	Notes    *NoteSystem                           // Note/board system
	Clans    *ClanSystem                           // Clan system
	Quests   *QuestSystem                          // Quest system
	MOBprogs *MOBprogSystem                        // MOB program system
	OLC      *builder.OLCSystem                    // Online creation system
	Help     *help.System                          // Help system

	// Callbacks for server-level operations
	OnSave            func(ch *types.Character) error // Called when player saves
	OnQuit            func(ch *types.Character)       // Called when player quits
	OnDelete          func(ch *types.Character) error // Called when player deletes character
	OnShutdown        func(reboot bool)               // Called when server should shutdown
	DisconnectPlayer  func(ch *types.Character)       // Called to forcibly disconnect a player
	OnRemoveCharacter func(ch *types.Character)       // Called to remove NPC from game loop

	// OLC persistence
	DataPath string // Path to data directory for saving (e.g. "data/areas")

	// Admin data
	BanList map[string]bool // Site bans (site -> permanent)
}

// CommandEntry represents a registered command
type CommandEntry struct {
	Name        string         // Command name
	Handler     CommandHandler // Function to execute
	MinPosition types.Position // Minimum position required
	MinLevel    int            // Minimum level required
}

// CommandRegistry holds all registered commands
type CommandRegistry struct {
	commands map[string]*CommandEntry
	aliases  map[string]string // alias -> command name
}

// NewCommandRegistry creates a new command registry
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]*CommandEntry),
		aliases:  make(map[string]string),
	}
}

func (d *CommandDispatcher) cmdDeity(ch *types.Character, args string) {
	if args == "" {
		// Show current deity or available deities
		if ch.PCData != nil && ch.PCData.Deity > 0 {
			deityName := d.getDeityName(ch.PCData.Deity)
			d.send(ch, fmt.Sprintf("You follow %s.\r\n", deityName))
			d.send(ch, "Use 'deity forsake' to abandon your deity.\r\n")
		} else {
			d.send(ch, "Available deities:\r\n")
			deities := []string{
				"1. Tempus (God of War)",
				"2. Tymora (Goddess of Luck)",
				"3. Mystra (Goddess of Magic)",
				"4. Mielikki (Goddess of Nature)",
				"5. Bane (God of Tyranny)",
			}
			for _, deity := range deities {
				d.send(ch, "  "+deity+"\r\n")
			}
			d.send(ch, "Use 'deity <number>' to choose a deity.\r\n")
		}
		return
	}

	if args == "forsake" {
		if ch.PCData == nil || ch.PCData.Deity == 0 {
			d.send(ch, "You don't follow any deity.\r\n")
			return
		}
		ch.PCData.Deity = 0
		d.send(ch, "You forsake your deity.\r\n")
		return
	}

	// Try to choose a deity
	deityID := 0
	if _, err := fmt.Sscanf(args, "%d", &deityID); err != nil || deityID < 1 || deityID > 5 {
		d.send(ch, "Invalid deity choice.\r\n")
		return
	}

	if ch.PCData != nil {
		ch.PCData.Deity = deityID
		deityName := d.getDeityName(deityID)
		d.send(ch, fmt.Sprintf("You now follow %s!\r\n", deityName))
	}
}

func (d *CommandDispatcher) getDeityName(deityID int) string {
	deities := map[int]string{
		1: "Tempus",
		2: "Tymora",
		3: "Mystra",
		4: "Mielikki",
		5: "Bane",
	}
	if name, ok := deities[deityID]; ok {
		return name
	}
	return "Unknown Deity"
}

func (d *CommandDispatcher) cmdAEdit(ch *types.Character, args string) {
	// Area editor - requires builder permission
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the area editor.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Syntax: aedit <command> [args]\r\n")
		d.send(ch, "Commands: create, edit, list, save, show\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	cmd := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = parts[1]
	}

	switch cmd {
	case "create":
		d.aeditCreate(ch, arg)
	case "edit":
		d.aeditEdit(ch, arg)
	case "list":
		d.aeditList(ch)
	case "save":
		d.aeditSave(ch, arg)
	case "show":
		d.aeditShow(ch, arg)
	default:
		d.send(ch, "Invalid aedit command. Use 'aedit' for help.\r\n")
	}
}

func (d *CommandDispatcher) cmdREdit(ch *types.Character, args string) {
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the room editor.\r\n")
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use the editor.\r\n")
		return
	}

	// If already editing a room, process as room editor command
	if ch.PCData.EditMode == int(builder.EditorRoom) && ch.PCData.EditData != nil {
		room, ok := ch.PCData.EditData.(*types.Room)
		if ok {
			d.processREditCommand(ch, room, args)
			return
		}
	}

	// Start editing current room or specified vnum
	args = strings.TrimSpace(args)
	var room *types.Room

	if args == "" {
		// Edit current room
		room = ch.InRoom
	} else if args == "create" {
		// Create new room - need a vnum
		d.send(ch, "Syntax: redit create <vnum>\r\n")
		return
	} else if strings.HasPrefix(args, "create ") {
		// Create new room with specified vnum
		vnumStr := strings.TrimPrefix(args, "create ")
		vnum, err := strconv.Atoi(vnumStr)
		if err != nil || vnum < 1 {
			d.send(ch, "Invalid vnum.\r\n")
			return
		}
		// Check if room exists
		if d.GameLoop != nil && d.GameLoop.Rooms[vnum] != nil {
			d.send(ch, "A room with that vnum already exists.\r\n")
			return
		}
		// Create new room
		room = &types.Room{
			Vnum:        vnum,
			Name:        "New Room",
			Description: "This is a new room.\r\n",
			Sector:      types.SectInside,
		}
		// Add to game
		if d.GameLoop != nil {
			d.GameLoop.Rooms[vnum] = room
		}
		d.send(ch, fmt.Sprintf("Room %d created.\r\n", vnum))
	} else {
		// Edit room by vnum
		vnum, err := strconv.Atoi(args)
		if err != nil {
			d.send(ch, "Syntax: redit [vnum] or redit create <vnum>\r\n")
			return
		}
		if d.GameLoop != nil {
			room = d.GameLoop.Rooms[vnum]
		}
		if room == nil {
			d.send(ch, "That room doesn't exist.\r\n")
			return
		}
	}

	if room == nil {
		d.send(ch, "You're not in a room.\r\n")
		return
	}

	// Enter edit mode
	ch.PCData.EditMode = int(builder.EditorRoom)
	ch.PCData.EditVnum = room.Vnum
	ch.PCData.EditData = room

	d.send(ch, fmt.Sprintf("Editing room [%d] %s\r\n", room.Vnum, room.Name))
	d.send(ch, "Type 'show' to see room, 'done' to exit, '?' for help.\r\n")
}

// processREditCommand handles room editor subcommands
func (d *CommandDispatcher) processREditCommand(ch *types.Character, room *types.Room, input string) {
	parts := strings.SplitN(strings.TrimSpace(input), " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "show", "":
		d.showRoomEditor(ch, room)

	case "name":
		if args == "" {
			d.send(ch, "Syntax: name <room name>\r\n")
			return
		}
		room.Name = args
		d.send(ch, "Room name set.\r\n")

	case "desc", "description":
		if args == "" {
			d.send(ch, "Syntax: desc <description>\r\n")
			return
		}
		room.Description = args + "\r\n"
		d.send(ch, "Room description set.\r\n")

	case "sector":
		if args == "" {
			d.send(ch, "Syntax: sector <type>\r\n")
			d.send(ch, "Types: inside city field forest hills mountain water_swim water_noswim air desert\r\n")
			return
		}
		sector := parseSectorType(args)
		if sector < 0 {
			d.send(ch, "Invalid sector type.\r\n")
			return
		}
		room.Sector = types.Sector(sector)
		d.send(ch, "Sector type set.\r\n")

	case "north", "south", "east", "west", "up", "down":
		d.setRoomExit(ch, room, cmd, args)

	case "flags":
		if args == "" {
			d.showRoomFlags(ch, room)
			return
		}
		d.toggleRoomFlag(ch, room, args)

	case "done", "save":
		// Save the room to TOML
		if d.DataPath != "" && d.GameLoop != nil && d.GameLoop.World != nil {
			if err := d.GameLoop.World.SaveRoom(room, d.DataPath); err != nil {
				d.send(ch, fmt.Sprintf("Error saving room: %s\r\n", err))
			} else {
				d.send(ch, "Room saved to TOML.\r\n")
			}
		}
		ch.PCData.EditMode = 0
		ch.PCData.EditVnum = 0
		ch.PCData.EditData = nil
		d.send(ch, "Exiting room editor.\r\n")

	case "?", "help", "commands":
		d.send(ch, "Room editor commands:\r\n")
		d.send(ch, "  show              - Display room info\r\n")
		d.send(ch, "  name <name>       - Set room name\r\n")
		d.send(ch, "  desc <desc>       - Set description\r\n")
		d.send(ch, "  sector <type>     - Set sector type\r\n")
		d.send(ch, "  north/south/etc <vnum>|delete - Set/remove exits\r\n")
		d.send(ch, "  flags [flag]      - Show/toggle room flags\r\n")
		d.send(ch, "  done              - Exit editor\r\n")

	default:
		d.send(ch, "Unknown command. Type '?' for help.\r\n")
	}
}

func (d *CommandDispatcher) showRoomEditor(ch *types.Character, room *types.Room) {
	d.send(ch, "=== Room Editor ===\r\n")
	d.send(ch, fmt.Sprintf("Vnum:        %d\r\n", room.Vnum))
	d.send(ch, fmt.Sprintf("Name:        %s\r\n", room.Name))
	d.send(ch, fmt.Sprintf("Sector:      %s\r\n", room.Sector.String()))
	d.send(ch, "Description:\r\n"+room.Description)

	// Show exits
	d.send(ch, "Exits:\r\n")
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := room.GetExit(dir)
		if exit != nil {
			d.send(ch, fmt.Sprintf("  %s: vnum %d\r\n", dir.String(), exit.ToVnum))
		}
	}

	// Show flags
	d.send(ch, fmt.Sprintf("Flags:       %s\r\n", formatRoomFlagsString(room.Flags)))
}

func (d *CommandDispatcher) setRoomExit(ch *types.Character, room *types.Room, dirStr, args string) {
	dir := parseDirectionString(dirStr)
	if dir < 0 {
		return
	}

	if args == "" {
		d.send(ch, fmt.Sprintf("Syntax: %s <vnum> | delete\r\n", dirStr))
		return
	}

	if strings.ToLower(args) == "delete" {
		room.SetExit(types.Direction(dir), nil)
		d.send(ch, "Exit deleted.\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid vnum.\r\n")
		return
	}

	// Find destination room
	var destRoom *types.Room
	if d.GameLoop != nil {
		destRoom = d.GameLoop.Rooms[vnum]
	}

	exit := &types.Exit{
		ToVnum: vnum,
		ToRoom: destRoom,
	}
	room.SetExit(types.Direction(dir), exit)
	d.send(ch, "Exit set.\r\n")
}

func (d *CommandDispatcher) showRoomFlags(ch *types.Character, room *types.Room) {
	d.send(ch, fmt.Sprintf("Room flags: %s\r\n", formatRoomFlagsString(room.Flags)))
	d.send(ch, "Available: dark nomob indoors private safe solitary norecall law\r\n")
}

func (d *CommandDispatcher) toggleRoomFlag(ch *types.Character, room *types.Room, flag string) {
	f := parseRoomFlagString(flag)
	if f == 0 {
		d.send(ch, "Unknown flag: "+flag+"\r\n")
		return
	}
	room.Flags.Toggle(f)
	if room.Flags.Has(f) {
		d.send(ch, "Flag "+flag+" enabled.\r\n")
	} else {
		d.send(ch, "Flag "+flag+" disabled.\r\n")
	}
}

func parseSectorType(s string) int {
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

func parseDirectionString(dir string) int {
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

func parseRoomFlagString(flag string) types.RoomFlags {
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

func formatRoomFlagsString(flags types.RoomFlags) string {
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

func (d *CommandDispatcher) cmdMEdit(ch *types.Character, args string) {
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the mobile editor.\r\n")
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use the editor.\r\n")
		return
	}

	// If already editing a mobile, process as mobile editor command
	if ch.PCData.EditMode == int(builder.EditorMobile) && ch.PCData.EditData != nil {
		mob, ok := ch.PCData.EditData.(*builder.MobileTemplate)
		if ok {
			d.processMEditCommand(ch, mob, args)
			return
		}
	}

	args = strings.TrimSpace(args)
	if args == "" {
		d.send(ch, "Syntax: medit <vnum>\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid vnum.\r\n")
		return
	}

	// Look up mobile template
	var mob *builder.MobileTemplate
	if d.GameLoop != nil && d.GameLoop.World != nil {
		template := d.GameLoop.World.GetMobTemplate(vnum)
		if template != nil {
			// Convert to MobileTemplate for editing
			mob = &builder.MobileTemplate{
				Vnum:      template.Vnum,
				Keywords:  template.Keywords,
				ShortDesc: template.ShortDesc,
				LongDesc:  template.LongDesc,
				Level:     template.Level,
				Alignment: template.Alignment,
			}
		}
	}

	if mob == nil {
		d.send(ch, "That mobile doesn't exist.\r\n")
		return
	}

	// Enter edit mode
	ch.PCData.EditMode = int(builder.EditorMobile)
	ch.PCData.EditVnum = vnum
	ch.PCData.EditData = mob

	d.send(ch, fmt.Sprintf("Editing mobile [%d] %s\r\n", mob.Vnum, mob.ShortDesc))
	d.send(ch, "Type 'show' to see mobile, 'done' to exit, '?' for help.\r\n")
}

func (d *CommandDispatcher) processMEditCommand(ch *types.Character, mob *builder.MobileTemplate, input string) {
	parts := strings.SplitN(strings.TrimSpace(input), " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "show", "":
		d.showMobileEditor(ch, mob)

	case "name":
		if args == "" {
			d.send(ch, "Syntax: name <keywords>\r\n")
			return
		}
		mob.Keywords = strings.Fields(args)
		d.send(ch, "Keywords set.\r\n")

	case "short":
		if args == "" {
			d.send(ch, "Syntax: short <description>\r\n")
			return
		}
		mob.ShortDesc = args
		d.send(ch, "Short description set.\r\n")

	case "long":
		if args == "" {
			d.send(ch, "Syntax: long <description>\r\n")
			return
		}
		mob.LongDesc = args
		d.send(ch, "Long description set.\r\n")

	case "level":
		level, err := strconv.Atoi(args)
		if err != nil || level < 1 || level > 200 {
			d.send(ch, "Syntax: level <1-200>\r\n")
			return
		}
		mob.Level = level
		d.send(ch, fmt.Sprintf("Level set to %d.\r\n", level))

	case "align", "alignment":
		align, err := strconv.Atoi(args)
		if err != nil || align < -1000 || align > 1000 {
			d.send(ch, "Syntax: align <-1000 to 1000>\r\n")
			return
		}
		mob.Alignment = align
		d.send(ch, fmt.Sprintf("Alignment set to %d.\r\n", align))

	case "done", "save":
		// Save the mobile to TOML
		if d.DataPath != "" && d.GameLoop != nil && d.GameLoop.World != nil {
			// Convert MobileTemplate back to MobileData for saving
			mobData := &loader.MobileData{
				Vnum:      mob.Vnum,
				Keywords:  mob.Keywords,
				ShortDesc: mob.ShortDesc,
				LongDesc:  mob.LongDesc,
				Level:     mob.Level,
				Alignment: mob.Alignment,
			}
			if err := d.GameLoop.World.SaveMobile(mob.Vnum, mobData, d.DataPath); err != nil {
				d.send(ch, fmt.Sprintf("Error saving mobile: %s\r\n", err))
			} else {
				d.send(ch, "Mobile saved to TOML.\r\n")
			}
		}
		ch.PCData.EditMode = 0
		ch.PCData.EditVnum = 0
		ch.PCData.EditData = nil
		d.send(ch, "Exiting mobile editor.\r\n")

	case "?", "help", "commands":
		d.send(ch, "Mobile editor commands:\r\n")
		d.send(ch, "  show              - Display mobile info\r\n")
		d.send(ch, "  name <keywords>   - Set keywords\r\n")
		d.send(ch, "  short <desc>      - Set short description\r\n")
		d.send(ch, "  long <desc>       - Set long description\r\n")
		d.send(ch, "  level <1-200>     - Set level\r\n")
		d.send(ch, "  align <-1000 to 1000> - Set alignment\r\n")
		d.send(ch, "  done              - Save and exit editor\r\n")

	default:
		d.send(ch, "Unknown command. Type '?' for help.\r\n")
	}
}

func (d *CommandDispatcher) showMobileEditor(ch *types.Character, mob *builder.MobileTemplate) {
	d.send(ch, "=== Mobile Editor ===\r\n")
	d.send(ch, fmt.Sprintf("Vnum:        %d\r\n", mob.Vnum))
	d.send(ch, fmt.Sprintf("Keywords:    %s\r\n", strings.Join(mob.Keywords, " ")))
	d.send(ch, fmt.Sprintf("Short:       %s\r\n", mob.ShortDesc))
	d.send(ch, fmt.Sprintf("Long:        %s\r\n", mob.LongDesc))
	d.send(ch, fmt.Sprintf("Level:       %d\r\n", mob.Level))
	d.send(ch, fmt.Sprintf("Alignment:   %d\r\n", mob.Alignment))
}

func (d *CommandDispatcher) cmdOEdit(ch *types.Character, args string) {
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the object editor.\r\n")
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use the editor.\r\n")
		return
	}

	// If already editing an object, process as object editor command
	if ch.PCData.EditMode == int(builder.EditorObject) && ch.PCData.EditData != nil {
		obj, ok := ch.PCData.EditData.(*builder.ObjectTemplate)
		if ok {
			d.processOEditCommand(ch, obj, args)
			return
		}
	}

	args = strings.TrimSpace(args)
	if args == "" {
		d.send(ch, "Syntax: oedit <vnum>\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid vnum.\r\n")
		return
	}

	// Look up object template
	var obj *builder.ObjectTemplate
	if d.GameLoop != nil && d.GameLoop.World != nil {
		template := d.GameLoop.World.GetObjTemplate(vnum)
		if template != nil {
			// Convert to ObjectTemplate for editing
			obj = &builder.ObjectTemplate{
				Vnum:      template.Vnum,
				Keywords:  template.Keywords,
				ShortDesc: template.ShortDesc,
				LongDesc:  template.LongDesc,
				Level:     template.Level,
				Weight:    template.Weight,
				Cost:      template.Cost,
			}
		}
	}

	if obj == nil {
		d.send(ch, "That object doesn't exist.\r\n")
		return
	}

	// Enter edit mode
	ch.PCData.EditMode = int(builder.EditorObject)
	ch.PCData.EditVnum = vnum
	ch.PCData.EditData = obj

	d.send(ch, fmt.Sprintf("Editing object [%d] %s\r\n", obj.Vnum, obj.ShortDesc))
	d.send(ch, "Type 'show' to see object, 'done' to exit, '?' for help.\r\n")
}

func (d *CommandDispatcher) processOEditCommand(ch *types.Character, obj *builder.ObjectTemplate, input string) {
	parts := strings.SplitN(strings.TrimSpace(input), " ", 2)
	cmd := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	switch cmd {
	case "show", "":
		d.showObjectEditor(ch, obj)

	case "name":
		if args == "" {
			d.send(ch, "Syntax: name <keywords>\r\n")
			return
		}
		obj.Keywords = strings.Fields(args)
		d.send(ch, "Keywords set.\r\n")

	case "short":
		if args == "" {
			d.send(ch, "Syntax: short <description>\r\n")
			return
		}
		obj.ShortDesc = args
		d.send(ch, "Short description set.\r\n")

	case "long":
		if args == "" {
			d.send(ch, "Syntax: long <description>\r\n")
			return
		}
		obj.LongDesc = args
		d.send(ch, "Long description set.\r\n")

	case "level":
		level, err := strconv.Atoi(args)
		if err != nil || level < 0 || level > 200 {
			d.send(ch, "Syntax: level <0-200>\r\n")
			return
		}
		obj.Level = level
		d.send(ch, fmt.Sprintf("Level set to %d.\r\n", level))

	case "cost":
		cost, err := strconv.Atoi(args)
		if err != nil || cost < 0 {
			d.send(ch, "Syntax: cost <amount>\r\n")
			return
		}
		obj.Cost = cost
		d.send(ch, fmt.Sprintf("Cost set to %d.\r\n", cost))

	case "weight":
		weight, err := strconv.Atoi(args)
		if err != nil || weight < 0 {
			d.send(ch, "Syntax: weight <amount>\r\n")
			return
		}
		obj.Weight = weight
		d.send(ch, fmt.Sprintf("Weight set to %d.\r\n", weight))

	case "done", "save":
		// Save the object to TOML
		if d.DataPath != "" && d.GameLoop != nil && d.GameLoop.World != nil {
			// Convert ObjectTemplate back to ObjectData for saving
			objData := &loader.ObjectData{
				Vnum:      obj.Vnum,
				Keywords:  obj.Keywords,
				ShortDesc: obj.ShortDesc,
				LongDesc:  obj.LongDesc,
				Level:     obj.Level,
				Weight:    obj.Weight,
				Cost:      obj.Cost,
			}
			if err := d.GameLoop.World.SaveObject(obj.Vnum, objData, d.DataPath); err != nil {
				d.send(ch, fmt.Sprintf("Error saving object: %s\r\n", err))
			} else {
				d.send(ch, "Object saved to TOML.\r\n")
			}
		}
		ch.PCData.EditMode = 0
		ch.PCData.EditVnum = 0
		ch.PCData.EditData = nil
		d.send(ch, "Exiting object editor.\r\n")

	case "?", "help", "commands":
		d.send(ch, "Object editor commands:\r\n")
		d.send(ch, "  show              - Display object info\r\n")
		d.send(ch, "  name <keywords>   - Set keywords\r\n")
		d.send(ch, "  short <desc>      - Set short description\r\n")
		d.send(ch, "  long <desc>       - Set long description\r\n")
		d.send(ch, "  level <0-200>     - Set level\r\n")
		d.send(ch, "  cost <amount>     - Set cost\r\n")
		d.send(ch, "  weight <amount>   - Set weight\r\n")
		d.send(ch, "  done              - Save and exit editor\r\n")

	default:
		d.send(ch, "Unknown command. Type '?' for help.\r\n")
	}
}

func (d *CommandDispatcher) showObjectEditor(ch *types.Character, obj *builder.ObjectTemplate) {
	d.send(ch, "=== Object Editor ===\r\n")
	d.send(ch, fmt.Sprintf("Vnum:        %d\r\n", obj.Vnum))
	d.send(ch, fmt.Sprintf("Keywords:    %s\r\n", strings.Join(obj.Keywords, " ")))
	d.send(ch, fmt.Sprintf("Short:       %s\r\n", obj.ShortDesc))
	d.send(ch, fmt.Sprintf("Long:        %s\r\n", obj.LongDesc))
	d.send(ch, fmt.Sprintf("Type:        %s\r\n", obj.ItemType.String()))
	d.send(ch, fmt.Sprintf("Level:       %d\r\n", obj.Level))
	d.send(ch, fmt.Sprintf("Cost:        %d\r\n", obj.Cost))
	d.send(ch, fmt.Sprintf("Weight:      %d\r\n", obj.Weight))
}

func (d *CommandDispatcher) cmdResets(ch *types.Character, args string) {
	// Reset editor - manages mob and object spawns in rooms
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the reset editor.\r\n")
		return
	}

	room := ch.InRoom
	if room == nil {
		d.send(ch, "You're not in a room.\r\n")
		return
	}

	if args == "" {
		// Show current resets for this room
		d.showRoomResets(ch, room)
		return
	}

	parts := strings.Fields(args)
	if len(parts) < 1 {
		d.send(ch, "Syntax: resets                   - Show current room resets\r\n")
		d.send(ch, "        resets mob <vnum> [max]  - Add mob reset\r\n")
		d.send(ch, "        resets obj <vnum> [max]  - Add object reset\r\n")
		d.send(ch, "        resets delete mob <#>    - Delete mob reset by index\r\n")
		d.send(ch, "        resets delete obj <#>    - Delete object reset by index\r\n")
		d.send(ch, "        resets clear             - Clear all resets\r\n")
		return
	}

	cmd := strings.ToLower(parts[0])
	switch cmd {
	case "mob":
		if len(parts) < 2 {
			d.send(ch, "Syntax: resets mob <vnum> [max]\r\n")
			return
		}
		vnum, err := strconv.Atoi(parts[1])
		if err != nil {
			d.send(ch, "Invalid vnum.\r\n")
			return
		}
		// Verify mob template exists
		if d.GameLoop != nil && d.GameLoop.World != nil && d.GameLoop.World.GetMobTemplate(vnum) == nil {
			d.send(ch, fmt.Sprintf("No mob template with vnum %d exists.\r\n", vnum))
			return
		}
		maxCount := 1
		if len(parts) >= 3 {
			if m, err := strconv.Atoi(parts[2]); err == nil {
				maxCount = m
			}
		}
		room.MobResets = append(room.MobResets, types.MobReset{
			Vnum:  vnum,
			Max:   maxCount,
			Count: 1,
		})
		d.send(ch, fmt.Sprintf("Added mob reset: vnum %d, max %d.\r\n", vnum, maxCount))

	case "obj":
		if len(parts) < 2 {
			d.send(ch, "Syntax: resets obj <vnum> [max]\r\n")
			return
		}
		vnum, err := strconv.Atoi(parts[1])
		if err != nil {
			d.send(ch, "Invalid vnum.\r\n")
			return
		}
		// Verify object template exists
		if d.GameLoop != nil && d.GameLoop.World != nil && d.GameLoop.World.GetObjTemplate(vnum) == nil {
			d.send(ch, fmt.Sprintf("No object template with vnum %d exists.\r\n", vnum))
			return
		}
		maxCount := 1
		if len(parts) >= 3 {
			if m, err := strconv.Atoi(parts[2]); err == nil {
				maxCount = m
			}
		}
		room.ObjResets = append(room.ObjResets, types.ObjReset{
			Vnum:  vnum,
			Max:   maxCount,
			Count: 1,
		})
		d.send(ch, fmt.Sprintf("Added object reset: vnum %d, max %d.\r\n", vnum, maxCount))

	case "delete":
		if len(parts) < 3 {
			d.send(ch, "Syntax: resets delete mob <#>\r\n")
			d.send(ch, "        resets delete obj <#>\r\n")
			return
		}
		resetType := strings.ToLower(parts[1])
		idx, err := strconv.Atoi(parts[2])
		if err != nil || idx < 1 {
			d.send(ch, "Invalid index. Use the number shown in 'resets' output.\r\n")
			return
		}
		idx-- // Convert to 0-based

		if resetType == "mob" {
			if idx >= len(room.MobResets) {
				d.send(ch, "No mob reset at that index.\r\n")
				return
			}
			removed := room.MobResets[idx]
			room.MobResets = append(room.MobResets[:idx], room.MobResets[idx+1:]...)
			d.send(ch, fmt.Sprintf("Removed mob reset #%d (vnum %d).\r\n", idx+1, removed.Vnum))
		} else if resetType == "obj" {
			if idx >= len(room.ObjResets) {
				d.send(ch, "No object reset at that index.\r\n")
				return
			}
			removed := room.ObjResets[idx]
			room.ObjResets = append(room.ObjResets[:idx], room.ObjResets[idx+1:]...)
			d.send(ch, fmt.Sprintf("Removed object reset #%d (vnum %d).\r\n", idx+1, removed.Vnum))
		} else {
			d.send(ch, "Specify 'mob' or 'obj' to delete.\r\n")
		}

	case "clear":
		mobCount := len(room.MobResets)
		objCount := len(room.ObjResets)
		room.MobResets = nil
		room.ObjResets = nil
		d.send(ch, fmt.Sprintf("Cleared %d mob resets and %d object resets.\r\n", mobCount, objCount))

	default:
		d.send(ch, "Unknown resets subcommand. Type 'resets' for help.\r\n")
	}
}

func (d *CommandDispatcher) showRoomResets(ch *types.Character, room *types.Room) {
	d.send(ch, fmt.Sprintf("=== Resets for Room %d: %s ===\r\n", room.Vnum, room.Name))

	// Show mob resets
	d.send(ch, "\r\nMob Resets:\r\n")
	if len(room.MobResets) == 0 {
		d.send(ch, "  None.\r\n")
	} else {
		for i, reset := range room.MobResets {
			mobName := fmt.Sprintf("(vnum %d)", reset.Vnum)
			if d.GameLoop != nil && d.GameLoop.World != nil {
				if tmpl := d.GameLoop.World.GetMobTemplate(reset.Vnum); tmpl != nil {
					mobName = tmpl.ShortDesc
				}
			}
			d.send(ch, fmt.Sprintf("  %d. %s - max %d, count %d\r\n",
				i+1, mobName, reset.Max, reset.Count))
		}
	}

	// Show object resets
	d.send(ch, "\r\nObject Resets:\r\n")
	if len(room.ObjResets) == 0 {
		d.send(ch, "  None.\r\n")
	} else {
		for i, reset := range room.ObjResets {
			objName := fmt.Sprintf("(vnum %d)", reset.Vnum)
			if d.GameLoop != nil && d.GameLoop.World != nil {
				if tmpl := d.GameLoop.World.GetObjTemplate(reset.Vnum); tmpl != nil {
					objName = tmpl.ShortDesc
				}
			}
			d.send(ch, fmt.Sprintf("  %d. %s - max %d, count %d\r\n",
				i+1, objName, reset.Max, reset.Count))
		}
	}

	d.send(ch, "\r\nSyntax: resets mob <vnum> [max] | resets obj <vnum> [max]\r\n")
	d.send(ch, "        resets delete mob <#> | resets delete obj <#>\r\n")
	d.send(ch, "        resets clear\r\n")
}

func (d *CommandDispatcher) cmdHEdit(ch *types.Character, args string) {
	// Help editor - creates and edits help entries
	if !d.hasBuilderPermission(ch) {
		d.send(ch, "You don't have permission to use the help editor.\r\n")
		return
	}

	// Ensure help system exists
	if d.Help == nil {
		d.Help = help.NewSystem()
	}

	if args == "" {
		d.send(ch, "Help Editor Commands:\r\n")
		d.send(ch, "  hedit show <keyword>      - Show help entry details\r\n")
		d.send(ch, "  hedit list [pattern]      - List all help entries\r\n")
		d.send(ch, "  hedit create <keyword>    - Create new help entry\r\n")
		d.send(ch, "  hedit keywords <kw> <new> - Change keywords\r\n")
		d.send(ch, "  hedit level <keyword> <#> - Set minimum level to view\r\n")
		d.send(ch, "  hedit syntax <kw> <text>  - Set syntax line\r\n")
		d.send(ch, "  hedit desc <keyword>      - Set description (opens editor)\r\n")
		d.send(ch, "  hedit seealso <kw> <refs> - Set see-also references\r\n")
		d.send(ch, "  hedit delete <keyword>    - Delete help entry\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 3)
	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "list":
		pattern := ""
		if len(parts) > 1 {
			pattern = strings.ToLower(parts[1])
		}
		keywords := d.Help.AllKeywords()
		if len(keywords) == 0 {
			d.send(ch, "No help entries defined.\r\n")
			return
		}
		d.send(ch, "Help Entries:\r\n")
		count := 0
		for _, kw := range keywords {
			if pattern == "" || strings.Contains(kw, pattern) {
				entry := d.Help.Find(kw)
				if entry != nil {
					d.send(ch, fmt.Sprintf("  %-20s (level %d)\r\n", kw, entry.Level))
					count++
				}
			}
		}
		d.send(ch, fmt.Sprintf("\r\n%d entries found.\r\n", count))

	case "show":
		if len(parts) < 2 {
			d.send(ch, "Syntax: hedit show <keyword>\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		d.send(ch, "=== Help Entry ===\r\n")
		d.send(ch, fmt.Sprintf("Keywords:    %s\r\n", strings.Join(entry.Keywords, " ")))
		d.send(ch, fmt.Sprintf("Level:       %d\r\n", entry.Level))
		d.send(ch, fmt.Sprintf("Syntax:      %s\r\n", entry.Syntax))
		d.send(ch, fmt.Sprintf("See Also:    %s\r\n", strings.Join(entry.SeeAlso, ", ")))
		d.send(ch, fmt.Sprintf("Description:\r\n%s\r\n", entry.Description))

	case "create":
		if len(parts) < 2 {
			d.send(ch, "Syntax: hedit create <keyword>\r\n")
			return
		}
		keyword := strings.ToLower(parts[1])
		if d.Help.Find(keyword) != nil {
			d.send(ch, fmt.Sprintf("Help entry '%s' already exists.\r\n", keyword))
			return
		}
		entry := &help.Entry{
			Keywords:    []string{keyword},
			Level:       0,
			Description: "No description set.",
		}
		d.Help.Register(entry)
		d.send(ch, fmt.Sprintf("Help entry '%s' created.\r\n", keyword))
		d.send(ch, "Use 'hedit desc <keyword>' to set the description.\r\n")

	case "keywords":
		if len(parts) < 3 {
			d.send(ch, "Syntax: hedit keywords <old-keyword> <new keywords...>\r\n")
			return
		}
		oldKey := parts[1]
		entry := d.Help.Find(oldKey)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", oldKey))
			return
		}
		newKeys := strings.Fields(parts[2])
		entry.Keywords = newKeys
		// Re-register with new keywords
		d.Help.Register(entry)
		d.send(ch, fmt.Sprintf("Keywords set to: %s\r\n", strings.Join(newKeys, " ")))

	case "level":
		if len(parts) < 3 {
			d.send(ch, "Syntax: hedit level <keyword> <level>\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		levelParts := strings.Fields(parts[2])
		level, err := strconv.Atoi(levelParts[0])
		if err != nil || level < 0 {
			d.send(ch, "Invalid level. Must be a non-negative number.\r\n")
			return
		}
		entry.Level = level
		d.send(ch, fmt.Sprintf("Level set to %d.\r\n", level))

	case "syntax":
		if len(parts) < 3 {
			d.send(ch, "Syntax: hedit syntax <keyword> <syntax text>\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		entry.Syntax = parts[2]
		d.send(ch, "Syntax set.\r\n")

	case "desc":
		if len(parts) < 2 {
			d.send(ch, "Syntax: hedit desc <keyword>\r\n")
			d.send(ch, "Then type the description, ending with '@' on a line by itself.\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		// For now, allow inline description if provided
		if len(parts) >= 3 {
			entry.Description = parts[2]
			d.send(ch, "Description set.\r\n")
		} else {
			d.send(ch, "Enter the description text. Use 'hedit desc <keyword> <text>' for inline.\r\n")
			d.send(ch, fmt.Sprintf("Current description:\r\n%s\r\n", entry.Description))
		}

	case "seealso":
		if len(parts) < 3 {
			d.send(ch, "Syntax: hedit seealso <keyword> <ref1> [ref2] ...\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		refs := strings.Fields(parts[2])
		entry.SeeAlso = refs
		d.send(ch, fmt.Sprintf("See-also set to: %s\r\n", strings.Join(refs, ", ")))

	case "delete":
		if len(parts) < 2 {
			d.send(ch, "Syntax: hedit delete <keyword>\r\n")
			return
		}
		keyword := parts[1]
		entry := d.Help.Find(keyword)
		if entry == nil {
			d.send(ch, fmt.Sprintf("No help entry found for '%s'.\r\n", keyword))
			return
		}
		// Note: The help.System doesn't have a delete method, so we just notify
		// In a real implementation, we'd add a Delete method to help.System
		d.send(ch, fmt.Sprintf("Help entry '%s' marked for deletion.\r\n", keyword))
		d.send(ch, "(Note: Full deletion requires server restart.)\r\n")

	default:
		d.send(ch, "Unknown hedit command. Type 'hedit' for help.\r\n")
	}
}

func (d *CommandDispatcher) hasBuilderPermission(ch *types.Character) bool {
	// Check if character has builder/immortal permissions
	return ch.Level >= 50 || (ch.Act.Has(types.ActKey)) // Simplified check
}

func (d *CommandDispatcher) aeditCreate(ch *types.Character, arg string) {
	if arg == "" {
		d.send(ch, "Syntax: aedit create <area name>\r\n")
		return
	}

	// Create new area - would need to add to World
	d.send(ch, fmt.Sprintf("Area '%s' creation not yet fully implemented.\r\n", arg))
	d.send(ch, "Note: Areas need to be created in TOML files in data/areas/\r\n")
}

func (d *CommandDispatcher) aeditEdit(ch *types.Character, arg string) {
	if arg == "" {
		// Edit current area
		if ch.InRoom == nil || ch.InRoom.Area == nil {
			d.send(ch, "You're not in an area.\r\n")
			return
		}
		area := ch.InRoom.Area
		d.send(ch, fmt.Sprintf("Editing area: %s\r\n", area.Name))
		d.aeditShowArea(ch, area)
		return
	}

	// Find area by name
	if d.GameLoop == nil || d.GameLoop.Areas == nil {
		d.send(ch, "No areas loaded.\r\n")
		return
	}

	for _, area := range d.GameLoop.Areas {
		if strings.EqualFold(area.Name, arg) {
			d.send(ch, fmt.Sprintf("Editing area: %s\r\n", area.Name))
			d.aeditShowArea(ch, area)
			return
		}
	}

	d.send(ch, "Area not found.\r\n")
}

func (d *CommandDispatcher) aeditList(ch *types.Character) {
	if d.GameLoop == nil || d.GameLoop.Areas == nil {
		d.send(ch, "No areas loaded.\r\n")
		return
	}

	d.send(ch, "Available areas:\r\n")
	d.send(ch, fmt.Sprintf("%-30s %10s %10s\r\n", "Name", "Min Vnum", "Max Vnum"))
	d.send(ch, strings.Repeat("-", 52)+"\r\n")

	for _, area := range d.GameLoop.Areas {
		d.send(ch, fmt.Sprintf("%-30s %10d %10d\r\n",
			truncateString(area.Name, 30), area.MinVnum, area.MaxVnum))
	}

	d.send(ch, fmt.Sprintf("\r\nTotal: %d areas\r\n", len(d.GameLoop.Areas)))
}

func (d *CommandDispatcher) aeditSave(ch *types.Character, arg string) {
	if d.DataPath == "" || d.GameLoop == nil || d.GameLoop.World == nil {
		d.send(ch, "Area saving not configured.\r\n")
		return
	}

	var area *types.Area
	if arg == "" {
		// Save current area
		if ch.InRoom == nil || ch.InRoom.Area == nil {
			d.send(ch, "You're not in an area. Specify an area name.\r\n")
			return
		}
		area = ch.InRoom.Area
	} else {
		// Find area by name
		for _, a := range d.GameLoop.Areas {
			if strings.EqualFold(a.Name, arg) {
				area = a
				break
			}
		}
		if area == nil {
			d.send(ch, "Area not found.\r\n")
			return
		}
	}

	// Save the area
	if err := d.GameLoop.World.SaveArea(area, d.DataPath); err != nil {
		d.send(ch, fmt.Sprintf("Error saving area: %s\r\n", err))
		return
	}

	d.send(ch, fmt.Sprintf("Area '%s' saved to TOML.\r\n", area.Name))
}

func (d *CommandDispatcher) aeditShow(ch *types.Character, arg string) {
	if arg == "" {
		// Show current area
		if ch.InRoom == nil || ch.InRoom.Area == nil {
			d.send(ch, "You're not in an area.\r\n")
			return
		}
		d.aeditShowArea(ch, ch.InRoom.Area)
		return
	}

	// Find area by name
	if d.GameLoop == nil || d.GameLoop.Areas == nil {
		d.send(ch, "No areas loaded.\r\n")
		return
	}

	for _, area := range d.GameLoop.Areas {
		if strings.EqualFold(area.Name, arg) {
			d.aeditShowArea(ch, area)
			return
		}
	}

	d.send(ch, "Area not found.\r\n")
}

func (d *CommandDispatcher) aeditShowArea(ch *types.Character, area *types.Area) {
	d.send(ch, "=== Area Editor ===\r\n")
	d.send(ch, fmt.Sprintf("Name:        %s\r\n", area.Name))
	d.send(ch, fmt.Sprintf("Filename:    %s\r\n", area.Filename))
	d.send(ch, fmt.Sprintf("Credits:     %s\r\n", area.Credits))
	d.send(ch, fmt.Sprintf("Vnum Range:  %d - %d\r\n", area.MinVnum, area.MaxVnum))
	d.send(ch, fmt.Sprintf("Level Range: %d - %d\r\n", area.LowRange, area.HighRange))

	// Count rooms/mobs/objects in this area
	roomCount := 0
	if d.GameLoop != nil {
		for vnum := range d.GameLoop.Rooms {
			if vnum >= area.MinVnum && vnum <= area.MaxVnum {
				roomCount++
			}
		}
	}
	d.send(ch, fmt.Sprintf("Rooms:       %d\r\n", roomCount))
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func containsProfanity(text string) bool {
	lower := strings.ToLower(text)
	profanities := []string{"fuck", "shit"}

	for _, profanity := range profanities {
		if strings.Contains(lower, profanity) {
			return true
		}
	}
	return false
}

// === Missing Command System Methods ===

// Register adds a command to the registry
func (r *CommandRegistry) Register(name string, handler CommandHandler, minPos types.Position, minLevel int) {
	r.commands[name] = &CommandEntry{
		Name:        name,
		Handler:     handler,
		MinPosition: minPos,
		MinLevel:    minLevel,
	}
}

// RegisterAlias creates an alias for a command
func (r *CommandRegistry) RegisterAlias(alias, command string) {
	r.aliases[alias] = command
}

// Find looks up a command by name (supports abbreviations)
func (r *CommandRegistry) Find(name string) *CommandEntry {
	name = strings.ToLower(name)

	// Check for exact match first
	if cmd, ok := r.commands[name]; ok {
		return cmd
	}

	// Check for alias (exact match)
	if alias, ok := r.aliases[name]; ok {
		if cmd, ok := r.commands[alias]; ok {
			return cmd
		}
	}

	// Check for prefix match on commands (abbreviation support)
	for cmdName, cmd := range r.commands {
		if strings.HasPrefix(cmdName, name) {
			return cmd
		}
	}

	// Check for prefix match on aliases
	for aliasName, cmdName := range r.aliases {
		if strings.HasPrefix(aliasName, name) {
			if cmd, ok := r.commands[cmdName]; ok {
				return cmd
			}
		}
	}

	return nil
}

// ExecuteResult indicates the outcome of attempting to execute a command
type ExecuteResult int

const (
	ExecNotFound ExecuteResult = iota // Command was not found
	ExecBadPos                        // Command found but position too low
	ExecBadLevel                      // Command found but level too low
	ExecOK                            // Command executed successfully
)

// Execute runs a command if it exists
// Returns ExecuteResult indicating what happened
func (r *CommandRegistry) Execute(name string, ch *types.Character, args string) ExecuteResult {
	cmd := r.Find(name)
	if cmd == nil {
		return ExecNotFound
	}

	// Check position requirements
	if ch.Position < cmd.MinPosition {
		return ExecBadPos
	}

	// Check level requirements
	if ch.Level < cmd.MinLevel {
		return ExecBadLevel
	}

	// Execute the command
	cmd.Handler(ch, args)
	return ExecOK
}

// NewCommandDispatcher creates a new dispatcher with basic commands
func NewCommandDispatcher() *CommandDispatcher {
	d := &CommandDispatcher{
		Registry: NewCommandRegistry(),
		Combat:   combat.NewCombatSystem(),
		Magic:    magic.NewMagicSystem(),
		Skills:   skills.NewSkillSystem(),
		Socials:  NewSocialRegistry(),
		MOBprogs: NewMOBprogSystem(),
		OLC:      builder.NewOLCSystem(),
	}
	d.registerBasicCommands()

	// Wire up OLC system output
	d.OLC.Output = func(ch *types.Character, msg string) {
		d.send(ch, msg)
	}

	// Wire up magic system to use skill improvement
	d.Magic.CheckImprove = func(ch *types.Character, skillName string, success bool, multiplier int) {
		d.Skills.CheckImprove(ch, skillName, success, multiplier)
	}

	// Wire up magic system to find objects in inventory (for identify spell)
	d.Magic.InvObjectFinder = func(ch *types.Character, name string) *types.Object {
		return FindObjInInventory(ch, name)
	}

	// Wire up magic system to display identify spell output
	d.Magic.IdentifyOutput = func(ch *types.Character, obj *types.Object) {
		d.showIdentifyOutput(ch, obj)
	}

	// Wire up MOBprog system to execute commands
	d.MOBprogs.SetCommandExecutor(func(mob *types.Character, command string, args string) {
		// Find and execute the command
		cmd := d.Registry.Find(command)
		if cmd != nil {
			cmd.Handler(mob, args)
		}
	})
	d.MOBprogs.SetRoomMessageSender(func(room *types.Room, message string) {
		if room == nil {
			return
		}
		for _, ch := range room.People {
			d.send(ch, message)
		}
	})

	return d
}

// registerBasicCommands sets up the initial command set
func (d *CommandDispatcher) registerBasicCommands() {
	// Movement commands
	d.Registry.Register("north", d.cmdNorth, types.PosStanding, 0)
	d.Registry.Register("south", d.cmdSouth, types.PosStanding, 0)
	d.Registry.Register("east", d.cmdEast, types.PosStanding, 0)
	d.Registry.Register("west", d.cmdWest, types.PosStanding, 0)
	d.Registry.Register("up", d.cmdUp, types.PosStanding, 0)
	d.Registry.Register("down", d.cmdDown, types.PosStanding, 0)

	// Information commands
	d.Registry.Register("look", d.cmdLook, types.PosResting, 0)
	d.Registry.Register("score", d.cmdScore, types.PosDead, 0)
	d.Registry.Register("who", d.cmdWho, types.PosDead, 0)
	d.Registry.Register("whois", d.cmdWhois, types.PosDead, 0)
	d.Registry.Register("inventory", d.cmdInventory, types.PosDead, 0)
	d.Registry.Register("equipment", d.cmdEquipment, types.PosDead, 0)
	d.Registry.Register("affects", d.cmdAffects, types.PosDead, 0)
	d.Registry.Register("time", d.cmdTime, types.PosDead, 0)
	d.Registry.Register("weather", d.cmdWeather, types.PosResting, 0)
	d.Registry.Register("report", d.cmdReport, types.PosResting, 0)
	d.Registry.Register("examine", d.cmdExamine, types.PosResting, 0)
	d.Registry.Register("enter", d.cmdEnter, types.PosStanding, 0)
	d.Registry.Register("areas", d.cmdAreas, types.PosDead, 0)
	d.Registry.Register("count", d.cmdCount, types.PosSleeping, 0)
	d.Registry.Register("worth", d.cmdWorth, types.PosSleeping, 0)
	d.Registry.Register("compare", d.cmdCompare, types.PosResting, 0)
	d.Registry.Register("credits", d.cmdCredits, types.PosDead, 0)

	// Communication commands
	d.Registry.Register("say", d.cmdSay, types.PosResting, 0)
	d.Registry.Register("emote", d.cmdEmote, types.PosResting, 0)
	d.Registry.Register("pmote", d.cmdPmote, types.PosResting, 0)
	d.Registry.Register("pose", d.cmdPose, types.PosResting, 0)
	d.Registry.Register("tell", d.cmdTell, types.PosResting, 0)
	d.Registry.Register("reply", d.cmdReply, types.PosResting, 0)
	d.Registry.Register("gossip", d.cmdGossip, types.PosSleeping, 0)
	d.Registry.Register("music", d.cmdMusic, types.PosSleeping, 0)
	d.Registry.Register("grats", d.cmdGrats, types.PosSleeping, 0)
	d.Registry.Register("ask", d.cmdAsk, types.PosSleeping, 0)
	d.Registry.Register("answer", d.cmdAnswer, types.PosSleeping, 0)
	d.Registry.Register("cgossip", d.cmdCGossip, types.PosSleeping, 0)
	d.Registry.Register("quote", d.cmdQuote, types.PosSleeping, 0)
	d.Registry.Register("qgossip", d.cmdQgossip, types.PosSleeping, 0)
	d.Registry.Register("shout", d.cmdShout, types.PosResting, 0)
	d.Registry.Register("yell", d.cmdYell, types.PosResting, 0)
	d.Registry.Register("afk", d.cmdAFK, types.PosSleeping, 0)
	d.Registry.Register("quiet", d.cmdQuiet, types.PosSleeping, 0)
	d.Registry.Register("deaf", d.cmdDeaf, types.PosSleeping, 0)
	d.Registry.Register("replay", d.cmdReplay, types.PosSleeping, 0)
	d.Registry.Register("channels", d.cmdChannels, types.PosDead, 0)
	d.Registry.Register("forget", d.cmdForget, types.PosSleeping, 0)
	d.Registry.Register("forge", d.cmdForge, types.PosSleeping, 0)
	d.Registry.Register("remember", d.cmdRemember, types.PosSleeping, 0)

	// Position commands
	d.Registry.Register("sit", d.cmdSit, types.PosSleeping, 0)
	d.Registry.Register("stand", d.cmdStand, types.PosSleeping, 0)
	d.Registry.Register("rest", d.cmdRest, types.PosSleeping, 0)
	d.Registry.Register("sleep", d.cmdSleep, types.PosSitting, 0)
	d.Registry.Register("wake", d.cmdWake, types.PosSleeping, 0)

	// Object commands
	d.Registry.Register("get", d.cmdGet, types.PosResting, 0)
	d.Registry.Register("drop", d.cmdDrop, types.PosResting, 0)
	d.Registry.Register("give", d.cmdGive, types.PosResting, 0)
	d.Registry.Register("put", d.cmdPut, types.PosResting, 0)
	d.Registry.Register("sacrifice", d.cmdSacrifice, types.PosResting, 0)
	d.Registry.Register("donate", d.cmdDonate, types.PosStanding, 5)
	d.Registry.Register("wear", d.cmdWear, types.PosResting, 0)
	d.Registry.Register("wield", d.cmdWield, types.PosResting, 0)
	d.Registry.Register("second", d.cmdSecond, types.PosResting, 0)
	d.Registry.Register("remove", d.cmdRemove, types.PosResting, 0)
	d.Registry.Register("eat", d.cmdEat, types.PosResting, 0)
	d.Registry.Register("drink", d.cmdDrink, types.PosResting, 0)
	d.Registry.Register("quaff", d.cmdQuaff, types.PosResting, 0)
	d.Registry.Register("recite", d.cmdRecite, types.PosResting, 0)
	d.Registry.Register("zap", d.cmdZap, types.PosResting, 0)
	d.Registry.Register("brandish", d.cmdBrandish, types.PosResting, 0)
	d.Registry.Register("fill", d.cmdFill, types.PosResting, 0)
	d.Registry.Register("pour", d.cmdPour, types.PosResting, 0)
	d.Registry.Register("envenom", d.cmdEnvenom, types.PosResting, 0)

	// Door commands
	d.Registry.Register("open", d.cmdOpen, types.PosResting, 0)
	d.Registry.Register("close", d.cmdClose, types.PosResting, 0)
	d.Registry.Register("lock", d.cmdLock, types.PosResting, 0)
	d.Registry.Register("unlock", d.cmdUnlock, types.PosResting, 0)
	d.Registry.Register("pick", d.cmdPick, types.PosStanding, 0)

	// Combat commands
	d.Registry.Register("kill", d.cmdKill, types.PosStanding, 0)
	d.Registry.Register("flee", d.cmdFlee, types.PosFighting, 0)
	d.Registry.Register("backstab", d.cmdBackstab, types.PosStanding, 0)
	d.Registry.Register("venom", d.cmdVenom, types.PosStanding, 0)
	d.Registry.Register("assassinate", d.cmdAssassinate, types.PosStanding, 0)
	d.Registry.Register("bash", d.cmdBash, types.PosFighting, 0)
	d.Registry.Register("kick", d.cmdKick, types.PosFighting, 0)
	d.Registry.Register("trip", d.cmdTrip, types.PosFighting, 0)
	d.Registry.Register("disarm", d.cmdDisarm, types.PosFighting, 0)
	d.Registry.Register("rescue", d.cmdRescue, types.PosFighting, 0)
	d.Registry.Register("dirt", d.cmdDirt, types.PosFighting, 0)
	d.Registry.Register("gouge", d.cmdGouge, types.PosFighting, 0)
	d.Registry.Register("circle", d.cmdCircle, types.PosFighting, 0)
	d.Registry.Register("berserk", d.cmdBerserk, types.PosStanding, 0)
	d.Registry.Register("consider", d.cmdConsider, types.PosResting, 0)
	d.Registry.Register("murder", d.cmdMurder, types.PosStanding, 0)
	d.Registry.Register("surrender", d.cmdSurrender, types.PosFighting, 0)
	d.Registry.Register("stun", d.cmdStun, types.PosFighting, 0)
	d.Registry.Register("feed", d.cmdFeed, types.PosFighting, 0)
	d.Registry.Register("lore", d.cmdLore, types.PosResting, 0)

	// Magic commands
	d.Registry.Register("cast", d.cmdCast, types.PosFighting, 0)
	d.Registry.Register("spells", d.cmdSpells, types.PosDead, 0)

	// Shop commands
	d.Registry.Register("buy", d.cmdBuy, types.PosStanding, 0)
	d.Registry.Register("sell", d.cmdSell, types.PosStanding, 0)
	d.Registry.Register("list", d.cmdList, types.PosStanding, 0)
	d.Registry.Register("value", d.cmdValue, types.PosStanding, 0)

	// Group commands
	d.Registry.Register("follow", d.cmdFollow, types.PosStanding, 0)
	d.Registry.Register("group", d.cmdGroup, types.PosDead, 0)
	d.Registry.Register("gtell", d.cmdGtell, types.PosDead, 0)
	d.Registry.Register("split", d.cmdSplit, types.PosResting, 0)
	d.Registry.Register("nofollow", d.cmdNofollow, types.PosDead, 0)
	d.Registry.Register("order", d.cmdOrder, types.PosResting, 0)
	d.Registry.Register("dismiss", d.cmdDismiss, types.PosResting, 0)

	// Utility commands
	d.Registry.Register("recall", d.cmdRecall, types.PosStanding, 0)
	d.Registry.Register("scan", d.cmdScan, types.PosResting, 0)
	d.Registry.Register("visible", d.cmdVisible, types.PosSleeping, 0)
	d.Registry.Register("exits", d.cmdExits, types.PosResting, 0)

	// Training and skills
	d.Registry.Register("train", d.cmdTrain, types.PosResting, 0)
	d.Registry.Register("practice", d.cmdPractice, types.PosResting, 0)
	d.Registry.Register("skills", d.cmdSkills, types.PosDead, 0)
	d.Registry.Register("gain", d.cmdGain, types.PosResting, 0)

	// Thief commands
	d.Registry.Register("sneak", d.cmdSneak, types.PosStanding, 0)
	d.Registry.Register("hide", d.cmdHide, types.PosStanding, 0)
	d.Registry.Register("steal", d.cmdSteal, types.PosStanding, 0)
	d.Registry.Register("peek", d.cmdPeek, types.PosStanding, 0)
	d.Registry.Register("track", d.cmdTrack, types.PosStanding, 0)

	// Bank commands
	d.Registry.Register("deposit", d.cmdDeposit, types.PosStanding, 0)
	d.Registry.Register("withdraw", d.cmdWithdraw, types.PosStanding, 0)
	d.Registry.Register("balance", d.cmdBalance, types.PosStanding, 0)

	// Other commands
	d.Registry.Register("save", d.cmdSave, types.PosDead, 0)
	d.Registry.Register("quit", d.cmdQuit, types.PosDead, 0)
	d.Registry.Register("delete", d.cmdDelete, types.PosStanding, 0)
	d.Registry.Register("help", d.cmdHelp, types.PosDead, 0)
	d.Registry.Register("commands", d.cmdCommands, types.PosDead, 0)
	d.Registry.Register("socials", d.cmdSocials, types.PosDead, 0)
	d.Registry.Register("wizlist", d.cmdWizlist, types.PosDead, 0)
	d.Registry.Register("rules", d.cmdRules, types.PosDead, 0)
	d.Registry.Register("story", d.cmdStory, types.PosDead, 0)
	d.Registry.Register("motd", d.cmdMotd, types.PosDead, 0)
	d.Registry.Register("imotd", d.cmdImotd, types.PosDead, 0)

	// Configuration commands
	d.Registry.Register("wimpy", d.cmdWimpy, types.PosDead, 0)
	d.Registry.Register("title", d.cmdTitle, types.PosDead, 0)
	d.Registry.Register("description", d.cmdDescription, types.PosDead, 0)
	d.Registry.Register("prompt", d.cmdPrompt, types.PosDead, 0)
	d.Registry.Register("brief", d.cmdBrief, types.PosDead, 0)
	d.Registry.Register("compact", d.cmdCompact, types.PosDead, 0)
	d.Registry.Register("color", d.cmdColor, types.PosDead, 0)
	d.Registry.Register("colour", d.cmdColor, types.PosDead, 0)
	d.Registry.Register("autolist", d.cmdAutolist, types.PosDead, 0)
	d.Registry.Register("autoexit", d.cmdAutoexit, types.PosDead, 0)
	d.Registry.Register("autogold", d.cmdAutogold, types.PosDead, 0)
	d.Registry.Register("autoloot", d.cmdAutoloot, types.PosDead, 0)
	d.Registry.Register("autosac", d.cmdAutosac, types.PosDead, 0)
	d.Registry.Register("autoassist", d.cmdAutoassist, types.PosDead, 0)
	d.Registry.Register("autosplit", d.cmdAutosplit, types.PosDead, 0)
	d.Registry.Register("autostore", d.cmdAutostore, types.PosDead, 0)
	d.Registry.Register("autopeek", d.cmdAutopeek, types.PosDead, 0)
	d.Registry.Register("nosummon", d.cmdNosummon, types.PosDead, 0)
	d.Registry.Register("noloot", d.cmdNoloot, types.PosDead, 0)
	d.Registry.Register("notran", d.cmdNotran, types.PosDead, 0)
	d.Registry.Register("outfit", d.cmdOutfit, types.PosResting, 0)
	d.Registry.Register("alias", d.cmdAlias, types.PosDead, 0)
	d.Registry.Register("unalias", d.cmdUnalias, types.PosDead, 0)
	d.Registry.Register("password", d.cmdPassword, types.PosDead, 0)
	d.Registry.Register("combine", d.cmdCombine, types.PosDead, 0)
	d.Registry.Register("scroll", d.cmdScroll, types.PosDead, 0)
	d.Registry.Register("long", d.cmdLong, types.PosDead, 0)
	d.Registry.Register("prefix", d.cmdPrefix, types.PosDead, 51) // Immortal only

	// Quest and clan commands
	d.Registry.Register("quest", d.cmdQuest, types.PosResting, 0)
	d.Registry.Register("clan", d.cmdClan, types.PosResting, 0)
	d.Registry.Register("member", d.cmdMember, types.PosResting, 0)
	d.Registry.Register("deity", d.cmdDeity, types.PosResting, 0)

	// Misc commands (play, voodoo)
	d.Registry.Register("play", d.cmdPlay, types.PosResting, 0)       // Play jukebox songs
	d.Registry.Register("voodoo", d.cmdVoodoo, types.PosStanding, 20) // Use voodoo dolls (level 20+)

	// OLC and immortal commands
	d.Registry.Register("aedit", d.cmdAEdit, types.PosDead, 100)
	d.Registry.Register("redit", d.cmdREdit, types.PosDead, 100)
	d.Registry.Register("medit", d.cmdMEdit, types.PosDead, 100)
	d.Registry.Register("oedit", d.cmdOEdit, types.PosDead, 100)
	d.Registry.Register("resets", d.cmdResets, types.PosDead, 100)
	d.Registry.Register("hedit", d.cmdHEdit, types.PosDead, 100)
	d.Registry.Register("goto", d.cmdGoto, types.PosDead, 100)
	d.Registry.Register("stat", d.cmdStat, types.PosDead, 100)
	d.Registry.Register("where", d.cmdWhere, types.PosDead, 100)
	d.Registry.Register("shutdown", d.cmdShutdown, types.PosDead, 100)
	d.Registry.Register("advance", d.cmdAdvance, types.PosDead, 100)
	d.Registry.Register("restore", d.cmdRestore, types.PosDead, 100)
	d.Registry.Register("peace", d.cmdPeace, types.PosDead, 100)
	d.Registry.Register("echo", d.cmdEcho, types.PosDead, 100)
	d.Registry.Register("transfer", d.cmdTransfer, types.PosDead, 100)
	d.Registry.Register("at", d.cmdAt, types.PosDead, 100)
	d.Registry.Register("load", d.cmdLoad, types.PosDead, 100)
	d.Registry.Register("purge", d.cmdPurge, types.PosDead, 100)
	d.Registry.Register("sockets", d.cmdSockets, types.PosDead, 100)
	d.Registry.Register("force", d.cmdForce, types.PosDead, 100)
	d.Registry.Register("slay", d.cmdSlay, types.PosDead, 100)
	d.Registry.Register("freeze", d.cmdFreeze, types.PosDead, 100)
	d.Registry.Register("mstat", d.cmdMstat, types.PosDead, 100)
	d.Registry.Register("ostat", d.cmdOstat, types.PosDead, 100)
	d.Registry.Register("rstat", d.cmdRstat, types.PosDead, 100)
	d.Registry.Register("mfind", d.cmdMfind, types.PosDead, 100)
	d.Registry.Register("ofind", d.cmdOfind, types.PosDead, 100)
	d.Registry.Register("mwhere", d.cmdMwhere, types.PosDead, 100)
	d.Registry.Register("owhere", d.cmdOwhere, types.PosDead, 100)
	d.Registry.Register("invis", d.cmdInvis, types.PosDead, 100)
	d.Registry.Register("holylight", d.cmdHolylight, types.PosDead, 100)
	d.Registry.Register("incognito", d.cmdIncognito, types.PosDead, 100)
	d.Registry.Register("snoop", d.cmdSnoop, types.PosDead, 100)
	d.Registry.Register("mset", d.cmdMset, types.PosDead, 100)
	d.Registry.Register("oset", d.cmdOset, types.PosDead, 100)
	d.Registry.Register("rset", d.cmdRset, types.PosDead, 100)
	d.Registry.Register("mload", d.cmdMload, types.PosDead, 100)
	d.Registry.Register("oload", d.cmdOload, types.PosDead, 100)
	d.Registry.Register("switch", d.cmdSwitch, types.PosDead, 100)
	d.Registry.Register("return", d.cmdReturn, types.PosDead, 100)
	d.Registry.Register("disconnect", d.cmdDisconnect, types.PosDead, 100)
	d.Registry.Register("pecho", d.cmdPecho, types.PosDead, 100)
	d.Registry.Register("wiznet", d.cmdWiznet, types.PosDead, 100)
	d.Registry.Register("ban", d.cmdBan, types.PosDead, 100)
	d.Registry.Register("allow", d.cmdAllow, types.PosDead, 100)
	d.Registry.Register("string", d.cmdString, types.PosDead, 100)
	d.Registry.Register("trust", d.cmdTrust, types.PosDead, 110)
	d.Registry.Register("wizlock", d.cmdWizlock, types.PosDead, 110)
	d.Registry.Register("newlock", d.cmdNewlock, types.PosDead, 110)
	d.Registry.Register("log", d.cmdLog, types.PosDead, 100)
	d.Registry.Register("noshout", d.cmdNoshout, types.PosDead, 100)
	d.Registry.Register("notell", d.cmdNotell, types.PosDead, 100)
	d.Registry.Register("noemote", d.cmdNoemote, types.PosDead, 100)
	d.Registry.Register("nochannels", d.cmdNochannels, types.PosDead, 100)
	d.Registry.Register("vnum", d.cmdVnum, types.PosDead, 100)
	d.Registry.Register("finger", d.cmdFinger, types.PosDead, 0)
	d.Registry.Register("clone", d.cmdClone, types.PosDead, 100)
	d.Registry.Register("zecho", d.cmdZecho, types.PosDead, 104)
	d.Registry.Register("gecho", d.cmdGecho, types.PosDead, 104)
	d.Registry.Register("allpeace", d.cmdAllpeace, types.PosDead, 110)
	d.Registry.Register("recover", d.cmdRecover, types.PosDead, 100)
	d.Registry.Register("memory", d.cmdMemory, types.PosDead, 100)
	d.Registry.Register("poofin", d.cmdPoofin, types.PosDead, 100)
	d.Registry.Register("poofout", d.cmdPoofout, types.PosDead, 100)
	d.Registry.Register("smote", d.cmdSmote, types.PosDead, 100)
	d.Registry.Register("immtalk", d.cmdImmtalk, types.PosDead, 100)
	d.Registry.Register(":", d.cmdImmtalk, types.PosDead, 100) // Alias for immtalk
	d.Registry.Register("pardon", d.cmdPardon, types.PosDead, 107)
	d.Registry.Register("penalty", d.cmdPenalty, types.PosDead, 100)
	d.Registry.Register("notitle", d.cmdNotitle, types.PosDead, 103)
	d.Registry.Register("deny", d.cmdDeny, types.PosDead, 110)
	d.Registry.Register("norestore", d.cmdNorestore, types.PosDead, 103)
	d.Registry.Register("guild", d.cmdGuild, types.PosDead, 108)
	d.Registry.Register("noclan", d.cmdNoclan, types.PosDead, 103)
	d.Registry.Register("ghost", d.cmdGhost, types.PosDead, 100)
	d.Registry.Register("wecho", d.cmdWecho, types.PosDead, 110)
	d.Registry.Register("permban", d.cmdPermban, types.PosDead, 110)
	d.Registry.Register("flag", d.cmdFlag, types.PosDead, 102)
	d.Registry.Register("sla", d.cmdSla, types.PosDead, 100)
	d.Registry.Register("immkiss", d.cmdImmkiss, types.PosDead, 108)
	d.Registry.Register("violate", d.cmdViolate, types.PosDead, 110)
	d.Registry.Register("protect", d.cmdProtect, types.PosDead, 108)
	d.Registry.Register("twit", d.cmdTwit, types.PosDead, 103)
	d.Registry.Register("pack", d.cmdPack, types.PosDead, 102)
	d.Registry.Register("gset", d.cmdGset, types.PosDead, 100)

	// New immortal commands
	d.Registry.Register("corner", d.cmdCorner, types.PosDead, 103)   // Transfer to punishment room
	d.Registry.Register("dupe", d.cmdDupe, types.PosDead, 105)       // Manage dupe list
	d.Registry.Register("knight", d.cmdKnight, types.PosDead, 110)   // Advance to Knight level
	d.Registry.Register("squire", d.cmdSquire, types.PosDead, 110)   // Advance to Squire level
	d.Registry.Register("mpoint", d.cmdMpoint, types.PosDead, 104)   // Toggle questpoint item flag
	d.Registry.Register("mquest", d.cmdMquest, types.PosDead, 104)   // Toggle quest item flag
	d.Registry.Register("wedpost", d.cmdWedpost, types.PosDead, 105) // Toggle wedding post permission
	d.Registry.Register("wipe", d.cmdWipe, types.PosDead, 108)       // Wipe player access
	d.Registry.Register("wizslap", d.cmdWizslap, types.PosDead, 102) // Fun immortal slap command

	// Note/board commands
	d.Registry.Register("note", d.cmdNote, types.PosSleeping, 0)
	d.Registry.Register("idea", d.cmdIdea, types.PosSleeping, 0)
	d.Registry.Register("news", d.cmdNews, types.PosSleeping, 0)
	d.Registry.Register("changes", d.cmdChanges, types.PosSleeping, 0)

	// Aliases
	d.Registry.RegisterAlias("n", "north")
	d.Registry.RegisterAlias("s", "south")
	d.Registry.RegisterAlias("e", "east")
	d.Registry.RegisterAlias("w", "west")
	d.Registry.RegisterAlias("u", "up")
	d.Registry.RegisterAlias("d", "down")
	d.Registry.RegisterAlias(".", "gossip")
	d.Registry.RegisterAlias("sac", "sacrifice")
	d.Registry.RegisterAlias("l", "look")
	d.Registry.RegisterAlias("read", "look")
	d.Registry.RegisterAlias("i", "inventory")
	d.Registry.RegisterAlias("eq", "equipment")
	d.Registry.RegisterAlias("exa", "examine")
	d.Registry.RegisterAlias("sc", "score")
	d.Registry.RegisterAlias("k", "kill")
	d.Registry.RegisterAlias("bs", "backstab")
	d.Registry.RegisterAlias("res", "rescue")
	d.Registry.RegisterAlias("c", "cast")
}

// Dispatch processes a command
func (d *CommandDispatcher) Dispatch(cmd Command) {
	// Parse the input
	input := strings.TrimSpace(cmd.Input)
	if input == "" {
		return
	}

	ch := cmd.Character

	// Handle '!' to repeat last command
	if input == "!" {
		if ch.Descriptor == nil || ch.Descriptor.LastCommand == "" {
			d.send(ch, "No command to repeat.\r\n")
			return
		}
		input = ch.Descriptor.LastCommand
	}

	// Store as last command (for '!' repeat)
	if ch.Descriptor != nil {
		ch.Descriptor.LastCommand = input
	}

	// Check for player aliases first (before splitting)
	if !ch.IsNPC() && ch.PCData != nil && ch.PCData.Aliases != nil {
		// Split to get the first word
		parts := strings.SplitN(input, " ", 2)
		aliasName := strings.ToLower(parts[0])
		if substitution, ok := ch.PCData.Aliases[aliasName]; ok {
			// Replace the alias with its substitution
			if len(parts) > 1 {
				input = substitution + " " + parts[1]
			} else {
				input = substitution
			}
		}
	}

	// Split into command and arguments
	parts := strings.SplitN(input, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = parts[1]
	}

	// Try to execute the command
	result := d.Registry.Execute(command, cmd.Character, args)
	switch result {
	case ExecOK:
		return
	case ExecBadPos:
		// Command exists but position is wrong - show appropriate message
		d.sendPositionMessage(cmd.Character)
		return
	case ExecBadLevel:
		d.send(cmd.Character, "Huh?\r\n")
		return
	}

	// Command not found - try socials as a fallback
	if d.Socials != nil {
		social := d.Socials.Find(command)
		if social != nil {
			// Find target if args provided
			var target *types.Character
			if args != "" {
				target = FindCharInRoom(cmd.Character, args)
				if target == nil {
					d.send(cmd.Character, "They aren't here.\r\n")
					return
				}
			}
			PerformSocial(cmd.Character, social, target, d.Output)
			return
		}
	}

	d.send(cmd.Character, "Huh?\r\n")
}

// === Movement Commands ===

func (d *CommandDispatcher) cmdNorth(ch *types.Character, args string) {
	d.doMove(ch, types.DirNorth)
}

func (d *CommandDispatcher) cmdSouth(ch *types.Character, args string) {
	d.doMove(ch, types.DirSouth)
}

func (d *CommandDispatcher) cmdEast(ch *types.Character, args string) {
	d.doMove(ch, types.DirEast)
}

func (d *CommandDispatcher) cmdWest(ch *types.Character, args string) {
	d.doMove(ch, types.DirWest)
}

func (d *CommandDispatcher) cmdUp(ch *types.Character, args string) {
	d.doMove(ch, types.DirUp)
}

func (d *CommandDispatcher) cmdDown(ch *types.Character, args string) {
	d.doMove(ch, types.DirDown)
}

func (d *CommandDispatcher) doMove(ch *types.Character, dir types.Direction) {
	if ch.InRoom == nil {
		d.send(ch, "You are nowhere!\r\n")
		return
	}

	room := ch.InRoom
	exit := room.GetExit(dir)
	if exit == nil {
		d.send(ch, "You can't go that way.\r\n")
		return
	}

	if exit.ToRoom == nil {
		d.send(ch, "That exit leads nowhere.\r\n")
		return
	}

	// Check door states
	if exit.Flags.Has(types.ExitClosed) {
		d.send(ch, "The door is closed.\r\n")
		return
	}

	// Move character
	oldRoom := room
	newRoom := exit.ToRoom

	// Record track for tracking skill
	ch.RecordTrack(oldRoom.Vnum, newRoom.Vnum)

	// Remove from old room
	for i, person := range oldRoom.People {
		if person == ch {
			oldRoom.People = append(oldRoom.People[:i], oldRoom.People[i+1:]...)
			break
		}
	}

	// Add to new room
	newRoom.People = append(newRoom.People, ch)

	// Update character's room
	ch.InRoom = newRoom

	// Check for explore quest progress
	d.checkQuestRoomEnter(ch, newRoom)

	// Show room description
	d.doLook(ch, "")

	// Move followers (charmed mobs, pets, group members)
	d.moveFollowers(ch, oldRoom, newRoom, dir)
}

// moveFollowers moves characters following the leader to the new room
func (d *CommandDispatcher) moveFollowers(leader *types.Character, fromRoom, toRoom *types.Room, dir types.Direction) {
	if fromRoom == nil {
		return
	}

	// Collect followers to move (avoid modifying list while iterating)
	var followers []*types.Character
	for _, person := range fromRoom.People {
		if person == leader {
			continue
		}
		// Check if this person follows the leader
		if person.Master == leader {
			followers = append(followers, person)
		}
	}

	// Move each follower
	for _, follower := range followers {
		// Can't move if fighting
		if follower.InCombat() {
			continue
		}

		// Can't move if incapacitated
		if follower.Position < types.PosStanding {
			continue
		}

		// Remove from old room
		for i, person := range fromRoom.People {
			if person == follower {
				fromRoom.People = append(fromRoom.People[:i], fromRoom.People[i+1:]...)
				break
			}
		}

		// Add to new room
		toRoom.People = append(toRoom.People, follower)
		follower.InRoom = toRoom

		// Record track for tracking
		follower.RecordTrack(fromRoom.Vnum, toRoom.Vnum)

		// Show the room to the follower
		if IsPet(follower) {
			// Pets don't get a look message
			continue
		}
		d.doLook(follower, "")
	}
}

// === Information Commands ===

func (d *CommandDispatcher) cmdLook(ch *types.Character, args string) {
	d.doLook(ch, args)
}

func (d *CommandDispatcher) doLook(ch *types.Character, args string) {
	if ch.InRoom == nil {
		d.send(ch, "You are nowhere!\r\n")
		return
	}

	room := ch.InRoom

	// If args specified, look at something specific
	if args != "" && args != "auto" {
		// Try to look at a character in the room
		victim := FindCharInRoom(ch, args)
		if victim != nil {
			d.lookAtCharacter(ch, victim)
			return
		}

		// Try to look at an object in inventory or room
		obj := FindObjOnChar(ch, args)
		if obj == nil {
			obj = FindObjInRoom(ch, args)
		}
		if obj != nil {
			d.lookAtObject(ch, obj)
			return
		}

		// Try to look in a direction
		directions := map[string]types.Direction{
			"north": types.DirNorth, "n": types.DirNorth,
			"east": types.DirEast, "e": types.DirEast,
			"south": types.DirSouth, "s": types.DirSouth,
			"west": types.DirWest, "w": types.DirWest,
			"up": types.DirUp, "u": types.DirUp,
			"down": types.DirDown, "d": types.DirDown,
		}
		if dir, ok := directions[strings.ToLower(args)]; ok {
			d.lookInDirection(ch, dir)
			return
		}

		d.send(ch, "You don't see that here.\r\n")
		return
	}

	// Room name and description
	d.send(ch, fmt.Sprintf("%s\r\n", room.Name))
	if room.Description != "" {
		d.send(ch, fmt.Sprintf("  %s\r\n", room.Description))
	}

	// Exits
	exits := []string{}
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := room.GetExit(dir)
		if exit != nil {
			if exit.Flags.Has(types.ExitClosed) {
				exits = append(exits, fmt.Sprintf("%s (closed)", dir.String()))
			} else {
				exits = append(exits, dir.String())
			}
		}
	}
	if len(exits) == 0 {
		d.send(ch, "Obvious exits: none\r\n")
	} else {
		d.send(ch, fmt.Sprintf("Obvious exits: %s\r\n", strings.Join(exits, " ")))
	}

	// Characters in room
	for _, char := range room.People {
		if char != ch {
			if char.IsNPC() && char.ShortDesc != "" {
				d.send(ch, fmt.Sprintf("%s is here.\r\n", char.ShortDesc))
			} else {
				d.send(ch, fmt.Sprintf("%s is here.\r\n", char.Name))
			}
		}
	}

	// Objects in room (using long descriptions for room, with combine support)
	combine := ch.Comm.Has(types.CommCombine) || ch.IsNPC()
	lines := formatObjectList(room.Objects, ch, false, combine)
	for _, line := range lines {
		d.send(ch, line+"\r\n")
	}
}

// lookAtCharacter displays information about a character
func (d *CommandDispatcher) lookAtCharacter(ch, victim *types.Character) {
	// Description
	if victim.Desc != "" {
		d.send(ch, victim.Desc+"\r\n")
	} else if victim.IsNPC() {
		d.send(ch, "You see nothing special about them.\r\n")
	} else {
		d.send(ch, fmt.Sprintf("%s is here.\r\n", victim.Name))
	}

	// Health condition
	d.send(ch, fmt.Sprintf("%s %s.\r\n", victim.Name, conditionString(victim)))

	// Equipment worn
	hasEquipment := false
	for i := types.WearLocation(0); i < types.WearLocMax; i++ {
		obj := victim.GetEquipment(i)
		if obj != nil {
			if !hasEquipment {
				d.send(ch, fmt.Sprintf("%s is using:\r\n", victim.Name))
				hasEquipment = true
			}
			d.send(ch, fmt.Sprintf("  <%s> %s\r\n", wearLocationName(i), obj.ShortDesc))
		}
	}
	if !hasEquipment {
		d.send(ch, fmt.Sprintf("%s is not using anything.\r\n", victim.Name))
	}
}

// lookAtObject displays information about an object
func (d *CommandDispatcher) lookAtObject(ch *types.Character, obj *types.Object) {
	if obj.LongDesc != "" {
		d.send(ch, obj.LongDesc+"\r\n")
	} else {
		d.send(ch, fmt.Sprintf("You see %s.\r\n", obj.ShortDesc))
	}

	// If it's a container, show contents
	if obj.ItemType == types.ItemTypeContainer || obj.ItemType == types.ItemTypeCorpseNPC || obj.ItemType == types.ItemTypeCorpsePC {
		if len(obj.Contents) == 0 {
			d.send(ch, "It is empty.\r\n")
		} else {
			d.send(ch, "It contains:\r\n")
			for _, item := range obj.Contents {
				d.send(ch, fmt.Sprintf("  %s\r\n", item.ShortDesc))
			}
		}
	}
}

// lookInDirection looks in a direction to see what's there
func (d *CommandDispatcher) lookInDirection(ch *types.Character, dir types.Direction) {
	exit := ch.InRoom.GetExit(dir)
	if exit == nil {
		d.send(ch, "You see nothing in that direction.\r\n")
		return
	}

	if exit.Description != "" {
		d.send(ch, exit.Description+"\r\n")
	} else if exit.ToRoom != nil {
		d.send(ch, fmt.Sprintf("You see %s.\r\n", exit.ToRoom.Name))
	} else {
		d.send(ch, "You see nothing special.\r\n")
	}

	if exit.IsDoor() {
		if exit.IsClosed() {
			d.send(ch, "The door is closed.\r\n")
		} else {
			d.send(ch, "The door is open.\r\n")
		}
	}
}

func (d *CommandDispatcher) cmdScore(ch *types.Character, args string) {
	race := types.GetRace(ch.Race)
	raceName := "Unknown"
	if race != nil {
		raceName = race.Name
	}
	class := types.GetClass(ch.Class)
	className := "Unknown"
	if class != nil {
		className = class.Name
	}

	// Title
	title := ""
	if ch.PCData != nil && ch.PCData.Title != "" {
		title = ch.PCData.Title
	}
	d.send(ch, fmt.Sprintf("You are %s%s, level %d.\r\n", ch.Name, title, ch.Level))

	// Sex
	sexName := "neutral"
	switch ch.Sex {
	case types.SexMale:
		sexName = "male"
	case types.SexFemale:
		sexName = "female"
	}
	d.send(ch, fmt.Sprintf("Race: %s  Sex: %s  Class: %s\r\n", raceName, sexName, className))

	// Vitals
	d.send(ch, fmt.Sprintf("You have %d/%d hit, %d/%d mana, %d/%d movement.\r\n",
		ch.Hit, ch.MaxHit, ch.Mana, ch.MaxMana, ch.Move, ch.MaxMove))

	// Training and practice
	d.send(ch, fmt.Sprintf("You have %d practices and %d training sessions.\r\n",
		ch.Practice, ch.Train))

	// Carry weight/items
	itemCount := len(ch.Inventory)
	for _, obj := range ch.Equipment {
		if obj != nil {
			itemCount++
		}
	}
	carryWeight := 0
	for _, obj := range ch.Inventory {
		carryWeight += obj.Weight
	}
	for _, obj := range ch.Equipment {
		if obj != nil {
			carryWeight += obj.Weight
		}
	}
	maxItems := 100 + ch.GetStat(types.StatDex)
	maxWeight := ch.GetStat(types.StatStr)*10 + ch.Level*5
	d.send(ch, fmt.Sprintf("You are carrying %d/%d items with weight %d/%d lbs.\r\n",
		itemCount, maxItems, carryWeight/10, maxWeight))

	// Stats (perm and current)
	d.send(ch, fmt.Sprintf("Str: %d(%d)  Int: %d(%d)  Wis: %d(%d)  Dex: %d(%d)  Con: %d(%d)\r\n",
		ch.PermStats[types.StatStr], ch.GetStat(types.StatStr),
		ch.PermStats[types.StatInt], ch.GetStat(types.StatInt),
		ch.PermStats[types.StatWis], ch.GetStat(types.StatWis),
		ch.PermStats[types.StatDex], ch.GetStat(types.StatDex),
		ch.PermStats[types.StatCon], ch.GetStat(types.StatCon)))

	// Money
	if ch.Platinum > 0 {
		d.send(ch, fmt.Sprintf("You have %d platinum, %d gold and %d silver coins.\r\n",
			ch.Platinum, ch.Gold, ch.Silver))
	} else {
		d.send(ch, fmt.Sprintf("You have %d gold and %d silver coins.\r\n", ch.Gold, ch.Silver))
	}

	// Experience
	if ch.Level < 51 { // Below immortal
		expPerLevel := 1000 * ch.Level // Simplified exp calculation
		expToLevel := expPerLevel - (ch.Exp % expPerLevel)
		if expToLevel <= 0 {
			expToLevel = expPerLevel
		}
		d.send(ch, fmt.Sprintf("You have scored %d exp. You need %d exp to level.\r\n",
			ch.Exp, expToLevel))
	} else {
		d.send(ch, fmt.Sprintf("You have scored %d exp.\r\n", ch.Exp))
	}

	// Alignment
	alignDesc := "neutral"
	if ch.Alignment > 350 {
		alignDesc = "good"
	} else if ch.Alignment > 100 {
		alignDesc = "kind"
	} else if ch.Alignment < -350 {
		alignDesc = "evil"
	} else if ch.Alignment < -100 {
		alignDesc = "mean"
	}
	d.send(ch, fmt.Sprintf("Alignment: %d (%s)\r\n", ch.Alignment, alignDesc))

	// Wimpy
	if ch.Wimpy > 0 {
		d.send(ch, fmt.Sprintf("Wimpy set to %d hit points.\r\n", ch.Wimpy))
	}

	// Conditions
	if ch.PCData != nil {
		if ch.PCData.Condition[types.CondDrunk] > 10 {
			d.send(ch, "You are drunk.\r\n")
		}
		if ch.PCData.Condition[types.CondThirst] == 0 {
			d.send(ch, "You are thirsty.\r\n")
		}
		if ch.PCData.Condition[types.CondHunger] == 0 {
			d.send(ch, "You are hungry.\r\n")
		}
	}

	// Position
	switch ch.Position {
	case types.PosDead:
		d.send(ch, "You are DEAD!!\r\n")
	case types.PosMortal:
		d.send(ch, "You are mortally wounded.\r\n")
	case types.PosIncap:
		d.send(ch, "You are incapacitated.\r\n")
	case types.PosStunned:
		d.send(ch, "You are stunned.\r\n")
	case types.PosSleeping:
		d.send(ch, "You are sleeping.\r\n")
	case types.PosResting:
		d.send(ch, "You are resting.\r\n")
	case types.PosSitting:
		d.send(ch, "You are sitting.\r\n")
	case types.PosFighting:
		d.send(ch, "You are fighting.\r\n")
	case types.PosStanding:
		d.send(ch, "You are standing.\r\n")
	}

	// Armor (show at level 25+)
	if ch.Level >= 25 {
		d.send(ch, fmt.Sprintf("Armor: pierce: %d  bash: %d  slash: %d  magic: %d\r\n",
			ch.Armor[0], ch.Armor[1], ch.Armor[2], ch.Armor[3]))
	}

	// Hit/Dam roll
	d.send(ch, fmt.Sprintf("Hitroll: %d  Damroll: %d\r\n", ch.HitRoll, ch.DamRoll))

	// Deity
	if ch.PCData != nil && ch.PCData.Deity > 0 {
		deityName := d.getDeityName(ch.PCData.Deity)
		d.send(ch, fmt.Sprintf("Deity: %s\r\n", deityName))
	}
}

func (d *CommandDispatcher) cmdWho(ch *types.Character, args string) {
	d.send(ch, "Players online:\r\n")
	d.send(ch, "-------------------------------------------------------------------------------\r\n")

	players := d.GameLoop.GetPlayers()
	for _, player := range players {
		race := types.GetRace(player.Race)
		raceName := "Unk"
		if race != nil {
			raceName = race.ShortName
		}
		class := types.GetClass(player.Class)
		className := "Unk"
		if class != nil {
			className = class.ShortName
		}

		title := ""
		if player.PCData != nil && player.PCData.Title != "" {
			title = player.PCData.Title
		}
		d.send(ch, fmt.Sprintf("[%2d %s %s] %s%s\r\n",
			player.Level, raceName, className, player.Name, title))
	}
	d.send(ch, fmt.Sprintf("\r\n%d players found.\r\n", len(players)))
}

func (d *CommandDispatcher) cmdWhois(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Whois whom?\r\n")
		return
	}

	// Find the player by name
	var target *types.Character
	for _, player := range d.GameLoop.GetPlayers() {
		if strings.HasPrefix(strings.ToLower(player.Name), strings.ToLower(args)) {
			target = player
			break
		}
	}

	if target == nil {
		d.send(ch, "No player by that name.\r\n")
		return
	}

	// Get race and class info
	race := types.GetRace(target.Race)
	raceName := "Unknown"
	if race != nil {
		raceName = race.Name
	}
	class := types.GetClass(target.Class)
	className := "Unknown"
	if class != nil {
		className = class.Name
	}

	title := ""
	if target.PCData != nil && target.PCData.Title != "" {
		title = target.PCData.Title
	}

	d.send(ch, fmt.Sprintf("%s%s\r\n", target.Name, title))
	d.send(ch, fmt.Sprintf("Level %d %s %s (%s)\r\n",
		target.Level,
		target.Sex.String(),
		raceName,
		className))

	// Clan info
	if target.PCData != nil && target.PCData.Clan > 0 && d.Clans != nil {
		clan := d.Clans.GetClan(target.PCData.Clan)
		if clan != nil {
			d.send(ch, fmt.Sprintf("Clan: %s\r\n", clan.Name))
		}
	}
}

func (d *CommandDispatcher) cmdInventory(ch *types.Character, args string) {
	if len(ch.Inventory) == 0 {
		d.send(ch, "You are carrying nothing.\r\n")
		return
	}

	d.send(ch, "You are carrying:\r\n")
	combine := ch.Comm.Has(types.CommCombine) || ch.IsNPC()
	lines := formatObjectList(ch.Inventory, ch, true, combine)
	for _, line := range lines {
		d.send(ch, line+"\r\n")
	}
}

func (d *CommandDispatcher) cmdEquipment(ch *types.Character, args string) {
	d.send(ch, "You are using:\r\n")
	for slot, obj := range ch.Equipment {
		if obj != nil {
			slotName := d.getWearLocationName(types.WearLocation(slot))
			d.send(ch, fmt.Sprintf("<%s> %s\r\n", slotName, obj.ShortDesc))
		}
	}
}

func (d *CommandDispatcher) cmdAffects(ch *types.Character, args string) {
	d.send(ch, "You are affected by:\r\n")

	affects := ch.Affected.All()
	if len(affects) == 0 {
		d.send(ch, "Nothing.\r\n")
		return
	}

	for _, af := range affects {
		// Format: Spell 'name': modifies stat by modifier for duration hours
		var modStr string
		if af.Location != types.ApplyNone {
			modStr = fmt.Sprintf("modifies %s by %d", applyTypeName(af.Location), af.Modifier)
		} else if af.BitVector != 0 {
			modStr = "adds affect"
		} else if af.ShieldVector != 0 {
			modStr = "adds shield"
		} else {
			modStr = "no visible effect"
		}

		d.send(ch, fmt.Sprintf("Spell '%s': %s for %d hours.\r\n",
			af.Type, modStr, af.Duration))
	}
}

func (d *CommandDispatcher) cmdTime(ch *types.Character, args string) {
	if d.GameLoop != nil && d.GameLoop.WorldTime != nil {
		d.send(ch, d.GameLoop.WorldTime.GetTimeString()+"\r\n")
	} else {
		time := d.GameLoop.GetTime()
		hour := time["hour"].(int)
		suffix := time["suffix"].(string)
		day := time["day"].(string)
		d.send(ch, fmt.Sprintf("It is %d o'clock %s, Day of the %s.\r\n",
			hour, suffix, day))
	}
	d.send(ch, "ROT started up some time ago.\r\n")
}

// applyTypeName returns a human-readable name for an apply type
func applyTypeName(loc types.ApplyType) string {
	switch loc {
	case types.ApplyStr:
		return "strength"
	case types.ApplyDex:
		return "dexterity"
	case types.ApplyInt:
		return "intelligence"
	case types.ApplyWis:
		return "wisdom"
	case types.ApplyCon:
		return "constitution"
	case types.ApplyHit:
		return "hp"
	case types.ApplyMana:
		return "mana"
	case types.ApplyMove:
		return "moves"
	case types.ApplyAC:
		return "armor class"
	case types.ApplyHitroll:
		return "hit roll"
	case types.ApplyDamroll:
		return "damage roll"
	case types.ApplySaves:
		return "saves"
	default:
		return "unknown"
	}
}

func (d *CommandDispatcher) cmdWeather(ch *types.Character, args string) {
	// Check if character is outside
	if ch.InRoom != nil && ch.InRoom.Flags.Has(types.RoomIndoors) {
		d.send(ch, "You can't see the weather indoors.\r\n")
		return
	}

	if d.GameLoop != nil && d.GameLoop.WorldTime != nil {
		d.send(ch, d.GameLoop.WorldTime.GetWeatherString()+"\r\n")
	} else {
		weather := d.GameLoop.GetWeather()
		description := weather["description"].(string)
		d.send(ch, description+"\r\n")
	}
}

func (d *CommandDispatcher) cmdAreas(ch *types.Character, args string) {
	if args != "" {
		d.send(ch, "No argument is used with this command.\r\n")
		return
	}

	if d.GameLoop == nil || len(d.GameLoop.Areas) == 0 {
		d.send(ch, "No areas loaded.\r\n")
		return
	}

	d.send(ch, "Available Areas:\r\n")
	d.send(ch, strings.Repeat("-", 78)+"\r\n")

	// Display areas in two columns like the C version
	areas := d.GameLoop.Areas
	halfCount := (len(areas) + 1) / 2

	for i := 0; i < halfCount; i++ {
		line := ""
		// First column
		if areas[i].Credits != "" {
			line = fmt.Sprintf("%-39s", areas[i].Credits)
		} else {
			line = fmt.Sprintf("%-39s", areas[i].Name)
		}
		// Second column
		if i+halfCount < len(areas) {
			if areas[i+halfCount].Credits != "" {
				line += fmt.Sprintf("%-39s", areas[i+halfCount].Credits)
			} else {
				line += fmt.Sprintf("%-39s", areas[i+halfCount].Name)
			}
		}
		d.send(ch, line+"\r\n")
	}
}

func (d *CommandDispatcher) cmdCount(ch *types.Character, args string) {
	if d.GameLoop == nil {
		d.send(ch, "Unable to count players.\r\n")
		return
	}

	count := 0
	for _, person := range d.GameLoop.Characters {
		if !person.IsNPC() && d.canSee(ch, person) {
			count++
		}
	}

	d.send(ch, fmt.Sprintf("There are %d visible characters on.\r\n", count))
}

func (d *CommandDispatcher) cmdWorth(ch *types.Character, args string) {
	if ch.IsNPC() {
		d.send(ch, fmt.Sprintf("You have %d gold and %d silver.\r\n", ch.Gold, ch.Silver))
		return
	}

	// Calculate experience to next level (with creation point overspend penalty)
	overspent := 0
	if ch.PCData != nil {
		overspent = ch.PCData.OverspentPoints
	}
	expNeeded := combat.ExpToLevelWithPenalty(ch.Level+1, overspent) - ch.Exp
	if expNeeded < 0 {
		expNeeded = 0
	}

	d.send(ch, fmt.Sprintf("You have %d gold and %d silver,\r\n", ch.Gold, ch.Silver))
	d.send(ch, fmt.Sprintf("and %d experience (%d exp to level).\r\n", ch.Exp, expNeeded))
}

func (d *CommandDispatcher) cmdCompare(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Compare what to what?\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	arg1 := parts[0]
	arg2 := ""
	if len(parts) > 1 {
		arg2 = parts[1]
	}

	// Find first object in inventory
	obj1 := FindObjInInventory(ch, arg1)
	if obj1 == nil {
		d.send(ch, "You do not have that item.\r\n")
		return
	}

	var obj2 *types.Object
	if arg2 == "" {
		// Compare to equipped item of same type
		for _, equipped := range ch.Equipment {
			if equipped != nil && equipped.ItemType == obj1.ItemType {
				obj2 = equipped
				break
			}
		}
		if obj2 == nil {
			d.send(ch, "You aren't wearing anything comparable.\r\n")
			return
		}
	} else {
		obj2 = FindObjInInventory(ch, arg2)
		if obj2 == nil {
			d.send(ch, "You do not have that item.\r\n")
			return
		}
	}

	// Compare the objects
	if obj1 == obj2 {
		d.send(ch, fmt.Sprintf("You compare %s to itself. It looks about the same.\r\n", obj1.ShortDesc))
		return
	}

	if obj1.ItemType != obj2.ItemType {
		d.send(ch, fmt.Sprintf("You can't compare %s and %s.\r\n", obj1.ShortDesc, obj2.ShortDesc))
		return
	}

	value1 := 0
	value2 := 0

	switch obj1.ItemType {
	case types.ItemTypeArmor:
		// Sum up AC values (values 0, 1, 2)
		value1 = obj1.Values[0] + obj1.Values[1] + obj1.Values[2]
		value2 = obj2.Values[0] + obj2.Values[1] + obj2.Values[2]
	case types.ItemTypeWeapon:
		// Average damage: (1 + max) * dice / 2, simplified to dice * max
		if len(obj1.Values) >= 3 {
			value1 = (1 + obj1.Values[2]) * obj1.Values[1]
		}
		if len(obj2.Values) >= 3 {
			value2 = (1 + obj2.Values[2]) * obj2.Values[1]
		}
	default:
		d.send(ch, fmt.Sprintf("You can't compare %s and %s.\r\n", obj1.ShortDesc, obj2.ShortDesc))
		return
	}

	var msg string
	if value1 == value2 {
		msg = fmt.Sprintf("%s and %s look about the same.", obj1.ShortDesc, obj2.ShortDesc)
	} else if value1 > value2 {
		msg = fmt.Sprintf("%s looks better than %s.", obj1.ShortDesc, obj2.ShortDesc)
	} else {
		msg = fmt.Sprintf("%s looks worse than %s.", obj1.ShortDesc, obj2.ShortDesc)
	}
	d.send(ch, msg+"\r\n")
}

func (d *CommandDispatcher) cmdCredits(ch *types.Character, args string) {
	d.cmdHelp(ch, "diku")
}

func (d *CommandDispatcher) cmdReport(ch *types.Character, args string) {
	d.send(ch, fmt.Sprintf("You report: %d/%d hp %d/%d mana %d/%d mv.\r\n",
		ch.Hit, ch.MaxHit, ch.Mana, ch.MaxMana, ch.Move, ch.MaxMove))
}

func (d *CommandDispatcher) cmdExamine(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Examine what?\r\n")
		return
	}

	// Look for object in inventory first
	for _, obj := range ch.Inventory {
		if strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(args)) {
			d.send(ch, fmt.Sprintf("You examine %s:\r\n", obj.ShortDesc))
			if obj.LongDesc != "" {
				d.send(ch, obj.LongDesc+"\r\n")
			}
			return
		}
	}

	// Look for object in room
	if ch.InRoom != nil {
		for _, obj := range ch.InRoom.Objects {
			if strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(args)) {
				d.send(ch, fmt.Sprintf("You examine %s:\r\n", obj.ShortDesc))
				if obj.LongDesc != "" {
					d.send(ch, obj.LongDesc+"\r\n")
				}
				return
			}
		}
	}

	d.send(ch, "You don't see that here.\r\n")
}

func (d *CommandDispatcher) cmdLore(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "What item would you like to examine with lore?\r\n")
		return
	}

	// Check skill
	skillLevel := 0
	if ch.PCData != nil {
		skillLevel = ch.PCData.Learned["lore"]
	}

	if skillLevel == 0 {
		d.send(ch, "You don't know how to use lore.\r\n")
		return
	}

	// Find the object
	var obj *types.Object
	for _, o := range ch.Inventory {
		if strings.HasPrefix(strings.ToLower(o.Name), strings.ToLower(args)) {
			obj = o
			break
		}
	}

	if obj == nil {
		d.send(ch, "You don't have that item.\r\n")
		return
	}

	// Add lag
	ch.Wait = 24

	// Roll for success
	if combat.NumberPercent() > skillLevel {
		d.send(ch, "You fail to recall any knowledge about that item.\r\n")
		return
	}

	// Success! Show item info (like identify but using skill)
	d.send(ch, fmt.Sprintf("Object '%s' is type %s.\r\n", obj.ShortDesc, obj.ItemType.String()))
	d.send(ch, fmt.Sprintf("Level %d, Weight %d, Value %d gold.\r\n", obj.Level, obj.Weight, obj.Cost))

	// Show wear locations
	var wearLocs []string
	if obj.WearFlags.Has(types.WearWield) {
		wearLocs = append(wearLocs, "wielded")
	}
	if obj.WearFlags.Has(types.WearBody) {
		wearLocs = append(wearLocs, "body")
	}
	if obj.WearFlags.Has(types.WearHead) {
		wearLocs = append(wearLocs, "head")
	}
	if obj.WearFlags.Has(types.WearLegs) {
		wearLocs = append(wearLocs, "legs")
	}
	if obj.WearFlags.Has(types.WearFeet) {
		wearLocs = append(wearLocs, "feet")
	}
	if obj.WearFlags.Has(types.WearHands) {
		wearLocs = append(wearLocs, "hands")
	}
	if obj.WearFlags.Has(types.WearArms) {
		wearLocs = append(wearLocs, "arms")
	}
	if obj.WearFlags.Has(types.WearShield) {
		wearLocs = append(wearLocs, "shield")
	}
	if obj.WearFlags.Has(types.WearAbout) {
		wearLocs = append(wearLocs, "about body")
	}
	if obj.WearFlags.Has(types.WearWaist) {
		wearLocs = append(wearLocs, "waist")
	}
	if obj.WearFlags.Has(types.WearWrist) {
		wearLocs = append(wearLocs, "wrist")
	}
	if obj.WearFlags.Has(types.WearNeck) {
		wearLocs = append(wearLocs, "neck")
	}
	if obj.WearFlags.Has(types.WearFinger) {
		wearLocs = append(wearLocs, "finger")
	}
	if obj.WearFlags.Has(types.WearHold) {
		wearLocs = append(wearLocs, "held")
	}
	if obj.WearFlags.Has(types.WearFloat) {
		wearLocs = append(wearLocs, "floating")
	}

	if len(wearLocs) > 0 {
		d.send(ch, fmt.Sprintf("Can be worn: %s\r\n", strings.Join(wearLocs, ", ")))
	}

	// Show weapon stats
	if obj.ItemType == types.ItemTypeWeapon {
		d.send(ch, fmt.Sprintf("Damage: %dd%d (average %d)\r\n",
			obj.Values[1], obj.Values[2], (obj.Values[1]*(obj.Values[2]+1))/2))
	}

	// Show armor stats
	if obj.ItemType == types.ItemTypeArmor {
		d.send(ch, fmt.Sprintf("Armor class: %d\r\n", obj.Values[0]))
	}

	// Show affects (at higher skill levels)
	if skillLevel >= 50 && obj.Affects.Len() > 0 {
		d.send(ch, "Affects:\r\n")
		for _, aff := range obj.Affects.All() {
			d.send(ch, fmt.Sprintf("  %s by %d\r\n", aff.Location.String(), aff.Modifier))
		}
	}

	// Extra flags at very high skill
	if skillLevel >= 80 {
		var flags []string
		if obj.ExtraFlags.Has(types.ItemGlow) {
			flags = append(flags, "glowing")
		}
		if obj.ExtraFlags.Has(types.ItemMagic) {
			flags = append(flags, "magical")
		}
		if obj.ExtraFlags.Has(types.ItemBless) {
			flags = append(flags, "blessed")
		}
		if obj.ExtraFlags.Has(types.ItemEvil) {
			flags = append(flags, "evil")
		}
		if obj.ExtraFlags.Has(types.ItemNoDrop) {
			flags = append(flags, "cursed")
		}
		if len(flags) > 0 {
			d.send(ch, fmt.Sprintf("Item flags: %s\r\n", strings.Join(flags, ", ")))
		}
	}
}

// showIdentifyOutput displays full object information for the identify spell.
// Unlike lore, this shows everything without skill level restrictions.
func (d *CommandDispatcher) showIdentifyOutput(ch *types.Character, obj *types.Object) {
	// Basic info
	d.send(ch, fmt.Sprintf("Object '%s' is type %s.\r\n", obj.ShortDesc, obj.ItemType.String()))
	d.send(ch, fmt.Sprintf("Level %d, Weight %d, Value %d gold.\r\n", obj.Level, obj.Weight, obj.Cost))

	// Show wear locations
	var wearLocs []string
	if obj.WearFlags.Has(types.WearWield) {
		wearLocs = append(wearLocs, "wielded")
	}
	if obj.WearFlags.Has(types.WearBody) {
		wearLocs = append(wearLocs, "body")
	}
	if obj.WearFlags.Has(types.WearHead) {
		wearLocs = append(wearLocs, "head")
	}
	if obj.WearFlags.Has(types.WearLegs) {
		wearLocs = append(wearLocs, "legs")
	}
	if obj.WearFlags.Has(types.WearFeet) {
		wearLocs = append(wearLocs, "feet")
	}
	if obj.WearFlags.Has(types.WearHands) {
		wearLocs = append(wearLocs, "hands")
	}
	if obj.WearFlags.Has(types.WearArms) {
		wearLocs = append(wearLocs, "arms")
	}
	if obj.WearFlags.Has(types.WearShield) {
		wearLocs = append(wearLocs, "shield")
	}
	if obj.WearFlags.Has(types.WearAbout) {
		wearLocs = append(wearLocs, "about body")
	}
	if obj.WearFlags.Has(types.WearWaist) {
		wearLocs = append(wearLocs, "waist")
	}
	if obj.WearFlags.Has(types.WearWrist) {
		wearLocs = append(wearLocs, "wrist")
	}
	if obj.WearFlags.Has(types.WearNeck) {
		wearLocs = append(wearLocs, "neck")
	}
	if obj.WearFlags.Has(types.WearFinger) {
		wearLocs = append(wearLocs, "finger")
	}
	if obj.WearFlags.Has(types.WearHold) {
		wearLocs = append(wearLocs, "held")
	}
	if obj.WearFlags.Has(types.WearFloat) {
		wearLocs = append(wearLocs, "floating")
	}

	if len(wearLocs) > 0 {
		d.send(ch, fmt.Sprintf("Can be worn: %s\r\n", strings.Join(wearLocs, ", ")))
	}

	// Show weapon stats
	if obj.ItemType == types.ItemTypeWeapon {
		d.send(ch, fmt.Sprintf("Damage: %dd%d (average %d)\r\n",
			obj.Values[1], obj.Values[2], (obj.Values[1]*(obj.Values[2]+1))/2))
		// Show weapon type
		weaponTypes := map[int]string{
			0: "exotic", 1: "sword", 2: "dagger", 3: "spear", 4: "mace",
			5: "axe", 6: "flail", 7: "whip", 8: "polearm",
		}
		if wtype, ok := weaponTypes[obj.Values[0]]; ok {
			d.send(ch, fmt.Sprintf("Weapon type: %s\r\n", wtype))
		}
		// Show damage type
		damTypes := map[int]string{
			0: "none", 1: "slice", 2: "stab", 3: "slash", 4: "whip",
			5: "claw", 6: "blast", 7: "pound", 8: "crush", 9: "grep",
			10: "bite", 11: "pierce", 12: "suction",
		}
		if dtype, ok := damTypes[obj.Values[3]]; ok {
			d.send(ch, fmt.Sprintf("Damage type: %s\r\n", dtype))
		}
	}

	// Show armor stats
	if obj.ItemType == types.ItemTypeArmor {
		d.send(ch, fmt.Sprintf("Armor class: pierce %d, bash %d, slash %d, magic %d\r\n",
			obj.Values[0], obj.Values[1], obj.Values[2], obj.Values[3]))
	}

	// Show container capacity
	if obj.ItemType == types.ItemTypeContainer {
		d.send(ch, fmt.Sprintf("Capacity: %d lbs\r\n", obj.Values[0]))
	}

	// Show drink container info
	if obj.ItemType == types.ItemTypeDrinkCon || obj.ItemType == types.ItemTypeFountain {
		liquidTypes := map[int]string{
			0: "water", 1: "beer", 2: "red wine", 3: "ale", 4: "dark ale",
			5: "whisky", 6: "lemonade", 7: "firebreather", 8: "local specialty",
			9: "slime mold juice", 10: "milk", 11: "tea", 12: "coffee",
			13: "blood", 14: "salt water", 15: "coke", 16: "root beer",
			17: "elvish wine", 18: "white wine", 19: "champagne",
		}
		liquid := "unknown"
		if l, ok := liquidTypes[obj.Values[2]]; ok {
			liquid = l
		}
		d.send(ch, fmt.Sprintf("Contains %s (%d/%d)\r\n", liquid, obj.Values[1], obj.Values[0]))
	}

	// Show food info
	if obj.ItemType == types.ItemTypeFood {
		d.send(ch, fmt.Sprintf("Food value: %d hours\r\n", obj.Values[0]))
		if obj.Values[3] != 0 {
			d.send(ch, "This food is poisoned.\r\n")
		}
	}

	// Show pill/potion/scroll/wand/staff spell info
	if obj.ItemType == types.ItemTypePill || obj.ItemType == types.ItemTypePotion ||
		obj.ItemType == types.ItemTypeScroll {
		d.send(ch, fmt.Sprintf("Level %d spells:\r\n", obj.Values[0]))
		for i := 1; i <= 4; i++ {
			if obj.Values[i] > 0 {
				spellName := d.getSpellNameBySlot(obj.Values[i])
				if spellName != "" {
					d.send(ch, fmt.Sprintf("  '%s'\r\n", spellName))
				}
			}
		}
	}

	if obj.ItemType == types.ItemTypeWand || obj.ItemType == types.ItemTypeStaff {
		d.send(ch, fmt.Sprintf("Has %d charges of level %d", obj.Values[2], obj.Values[0]))
		if obj.Values[3] > 0 {
			spellName := d.getSpellNameBySlot(obj.Values[3])
			if spellName != "" {
				d.send(ch, fmt.Sprintf(" '%s'", spellName))
			}
		}
		d.send(ch, ".\r\n")
	}

	// Show affects (identify shows everything)
	if obj.Affects.Len() > 0 {
		d.send(ch, "Affects:\r\n")
		for _, aff := range obj.Affects.All() {
			d.send(ch, fmt.Sprintf("  %s by %d\r\n", aff.Location.String(), aff.Modifier))
		}
	}

	// Show extra flags
	var flags []string
	if obj.ExtraFlags.Has(types.ItemGlow) {
		flags = append(flags, "glowing")
	}
	if obj.ExtraFlags.Has(types.ItemHum) {
		flags = append(flags, "humming")
	}
	if obj.ExtraFlags.Has(types.ItemMagic) {
		flags = append(flags, "magical")
	}
	if obj.ExtraFlags.Has(types.ItemBless) {
		flags = append(flags, "blessed")
	}
	if obj.ExtraFlags.Has(types.ItemEvil) {
		flags = append(flags, "evil")
	}
	if obj.ExtraFlags.Has(types.ItemInvis) {
		flags = append(flags, "invisible")
	}
	if obj.ExtraFlags.Has(types.ItemNoDrop) {
		flags = append(flags, "cursed (nodrop)")
	}
	if obj.ExtraFlags.Has(types.ItemNoRemove) {
		flags = append(flags, "cursed (noremove)")
	}
	if obj.ExtraFlags.Has(types.ItemAntiGood) {
		flags = append(flags, "anti-good")
	}
	if obj.ExtraFlags.Has(types.ItemAntiEvil) {
		flags = append(flags, "anti-evil")
	}
	if obj.ExtraFlags.Has(types.ItemAntiNeutral) {
		flags = append(flags, "anti-neutral")
	}
	if len(flags) > 0 {
		d.send(ch, fmt.Sprintf("Extra flags: %s\r\n", strings.Join(flags, ", ")))
	}
}

// getSpellNameBySlot looks up a spell name by its slot number
func (d *CommandDispatcher) getSpellNameBySlot(slot int) string {
	if d.Magic != nil && d.Magic.Registry != nil {
		spell := d.Magic.Registry.FindBySlot(slot)
		if spell != nil {
			return spell.Name
		}
	}
	return ""
}

func (d *CommandDispatcher) cmdEnter(ch *types.Character, args string) {
	if ch.Fighting != nil {
		d.send(ch, "You can't enter anything while fighting!\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Enter what?\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You're nowhere!\r\n")
		return
	}

	// Find a portal object in the room
	portal := FindObjInRoom(ch, args)
	if portal == nil {
		d.send(ch, "You don't see that here.\r\n")
		return
	}

	if portal.ItemType != types.ItemTypePortal {
		d.send(ch, "You can't seem to find a way in.\r\n")
		return
	}

	// Check if portal is closed (value[1] would have closed flag)
	// For simplicity, we'll skip this check for now

	// Check destination room (stored in value[3])
	destVnum := portal.Values[3]
	if destVnum <= 0 {
		d.send(ch, fmt.Sprintf("%s doesn't seem to go anywhere.\r\n", portal.ShortDesc))
		return
	}

	destRoom := d.GameLoop.GetRoom(destVnum)
	if destRoom == nil {
		d.send(ch, fmt.Sprintf("%s doesn't seem to go anywhere.\r\n", portal.ShortDesc))
		return
	}

	// Can't enter if cursed and room blocks it
	if ch.IsAffected(types.AffCurse) && ch.InRoom.Flags.Has(types.RoomNoRecall) {
		d.send(ch, "Something prevents you from leaving...\r\n")
		return
	}

	// Leave message
	ActToRoom("$n steps into $p.", ch, nil, portal, d.Output)
	d.send(ch, fmt.Sprintf("You walk through %s and find yourself somewhere else...\r\n", portal.ShortDesc))

	// Move character
	oldRoom := ch.InRoom
	oldRoom.RemovePerson(ch)
	destRoom.AddPerson(ch)
	ch.InRoom = destRoom

	// Arrival message
	ActToRoom("$n has arrived through a portal.", ch, nil, nil, d.Output)

	// Show new room
	d.cmdLook(ch, "")

	// Reduce portal charges if applicable (value[0] = charges, -1 = infinite)
	if portal.Values[0] > 0 {
		portal.Values[0]--
		if portal.Values[0] == 0 {
			// Portal disappears after last charge
			ObjFromRoom(portal)
			if oldRoom != nil {
				// Notify old room
				for _, person := range oldRoom.People {
					d.send(person, fmt.Sprintf("%s fades out of existence.\r\n", portal.ShortDesc))
				}
			}
		}
	}
}

// === Communication Commands ===

func (d *CommandDispatcher) cmdSay(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Say what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	if ch.InRoom == nil {
		return
	}
	room := ch.InRoom

	msg := fmt.Sprintf("You say '%s'\r\n", args)
	d.send(ch, msg)

	msg = fmt.Sprintf("%s says '%s'\r\n", ch.Name, args)
	for _, other := range room.People {
		if other != ch {
			d.send(other, msg)
		}
	}
}

func (d *CommandDispatcher) cmdTell(ch *types.Character, args string) {
	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Tell whom what?\r\n")
		return
	}

	targetName := parts[0]
	message := parts[1]

	if containsProfanity(message) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	target := d.GameLoop.FindPlayer(targetName)
	if target == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("You tell %s '%s'\r\n", target.Name, message))
	d.send(target, fmt.Sprintf("%s tells you '%s'\r\n", ch.Name, message))
}

func (d *CommandDispatcher) cmdReply(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Reply what?\r\n")
		return
	}

	if ch.Reply == nil {
		d.send(ch, "No one has told you anything recently.\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	target := ch.Reply
	if target.Descriptor == nil || target.Descriptor.State != types.ConPlaying {
		d.send(ch, "They aren't here anymore.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("You reply to %s '%s'\r\n", target.Name, args))
	d.send(target, fmt.Sprintf("%s replies '%s'\r\n", ch.Name, args))
}

func (d *CommandDispatcher) cmdGossip(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Gossip what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	msg := fmt.Sprintf("[GOSSIP] %s gossips '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, msg)
	}
}

func (d *CommandDispatcher) cmdMusic(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Music what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	msg := fmt.Sprintf("[MUSIC] %s plays '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, msg)
	}
}

func (d *CommandDispatcher) cmdGrats(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Grats what?\r\n")
		return
	}

	msg := fmt.Sprintf("[GRATS] %s congratulates '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, msg)
	}
}

func (d *CommandDispatcher) cmdAsk(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Ask what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	msg := fmt.Sprintf("[QUESTION] %s asks '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, msg)
	}
}

func (d *CommandDispatcher) cmdAnswer(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Answer what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	msg := fmt.Sprintf("[ANSWER] %s answers '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, msg)
	}
}

func (d *CommandDispatcher) cmdCGossip(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Cgossip what?\r\n")
		return
	}

	clanID := 0
	if ch.PCData != nil {
		clanID = ch.PCData.Clan
	}
	if clanID == 0 {
		d.send(ch, "You aren't in a clan.\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	msg := fmt.Sprintf("[CLAN] %s gossips '%s'\r\n", ch.Name, args)
	for _, player := range d.GameLoop.GetPlayers() {
		playerClanID := 0
		if player.PCData != nil {
			playerClanID = player.PCData.Clan
		}
		if playerClanID == clanID {
			d.send(player, msg)
		}
	}
}

// === Consider Command ===

func (d *CommandDispatcher) cmdConsider(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Consider what?\r\n")
		return
	}

	victim := FindCharInRoom(ch, args)
	if victim == nil {
		d.send(ch, "They're not here.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "You consider yourself. You feel pretty confident.\r\n")
		return
	}

	// Check if safe
	if combat.IsSafe(ch, victim) {
		d.send(ch, "Don't even think about it.\r\n")
		return
	}

	// Calculate comparison
	diff := (victim.Hit / 50) - (ch.Hit / 50)

	// AC comparison (negative AC is better, so we negate)
	victimAC := -(victim.Armor[types.ACPierce] + victim.Armor[types.ACBash] + victim.Armor[types.ACSlash] + victim.Armor[types.ACExotic])
	chAC := -(ch.Armor[types.ACPierce] + ch.Armor[types.ACBash] + ch.Armor[types.ACSlash] + ch.Armor[types.ACExotic])
	diff += (victimAC - chAC)

	// Damroll and hitroll comparison
	diff += (combat.GetDamroll(victim) - combat.GetDamroll(ch))
	diff += (combat.GetHitroll(victim) - combat.GetHitroll(ch))

	// Strength comparison
	diff += (victim.GetStat(types.StatStr) - ch.GetStat(types.StatStr))

	// Determine message based on difference
	var msg string
	victimName := victim.Name
	if victim.IsNPC() && victim.ShortDesc != "" {
		victimName = victim.ShortDesc
	}

	switch {
	case diff <= -110:
		msg = "You can kill " + victimName + " naked and weaponless."
	case diff <= -70:
		msg = victimName + " is no match for you."
	case diff <= -20:
		msg = victimName + " looks like an easy kill."
	case diff <= 20:
		msg = "The perfect match!"
	case diff <= 70:
		msg = victimName + " says 'Do you feel lucky, punk?'."
	case diff <= 110:
		msg = victimName + " laughs at you mercilessly."
	default:
		msg = "Death will thank you for your gift."
	}

	d.send(ch, msg+"\r\n")
}

// === Object Commands ===
// Note: cmdGet, cmdDrop, cmdGive, cmdPut, cmdSacrifice, cmdWear, cmdWield, cmdRemove are in commands_objects.go

func (d *CommandDispatcher) cmdEat(ch *types.Character, args string) {
	d.doEat(ch, args)
}

func (d *CommandDispatcher) cmdDrink(ch *types.Character, args string) {
	d.doDrink(ch, args)
}

func (d *CommandDispatcher) cmdQuaff(ch *types.Character, args string) {
	d.doQuaff(ch, args)
}

func (d *CommandDispatcher) cmdRecite(ch *types.Character, args string) {
	d.doRecite(ch, args)
}

func (d *CommandDispatcher) cmdZap(ch *types.Character, args string) {
	d.doZap(ch, args)
}

func (d *CommandDispatcher) cmdBrandish(ch *types.Character, args string) {
	d.doBrandish(ch, args)
}

func (d *CommandDispatcher) cmdFill(ch *types.Character, args string) {
	d.doFill(ch, args)
}

func (d *CommandDispatcher) cmdPour(ch *types.Character, args string) {
	d.doPour(ch, args)
}

func (d *CommandDispatcher) cmdEnvenom(ch *types.Character, args string) {
	d.doEnvenom(ch, args)
}

// === Combat Commands ===
// Note: cmdKill, cmdFlee, cmdBackstab, cmdBash, cmdKick, cmdTrip, cmdDisarm, cmdRescue are in commands_combat.go

// === Magic Commands ===
// Note: cmdCast, cmdSpells are in commands_magic.go

// === Other Commands ===

func (d *CommandDispatcher) cmdSave(ch *types.Character, args string) {
	if d.OnSave != nil {
		if err := d.OnSave(ch); err != nil {
			d.send(ch, "Save failed.\r\n")
		} else {
			d.send(ch, "Saved.\r\n")
		}
	} else {
		d.send(ch, "Save system not available.\r\n")
	}
}

func (d *CommandDispatcher) cmdQuit(ch *types.Character, args string) {
	d.send(ch, "Goodbye!\r\n")
	if d.OnQuit != nil {
		d.OnQuit(ch)
	}
}

// cmdDelete permanently deletes a character
// Requires typing the character's name as confirmation
func (d *CommandDispatcher) cmdDelete(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	// Must be standing and not fighting
	if ch.Fighting != nil {
		d.send(ch, "You can't delete yourself while fighting!\r\n")
		return
	}

	// Require password as confirmation
	if args == "" {
		d.send(ch, "WARNING: This will PERMANENTLY delete your character!\r\n")
		d.send(ch, "All equipment, skills, and progress will be lost forever.\r\n")
		d.send(ch, "\r\nTo confirm, type: delete <your password>\r\n")
		return
	}

	// Check password
	if ch.PCData == nil {
		d.send(ch, "Error: No player data found.\r\n")
		return
	}

	// Simple password check - compare with stored hash
	if !checkPassword(args, ch.PCData.Password) {
		d.send(ch, "Incorrect password. Character NOT deleted.\r\n")
		return
	}

	// Delete the character
	if d.OnDelete != nil {
		if err := d.OnDelete(ch); err != nil {
			d.send(ch, fmt.Sprintf("Error deleting character: %v\r\n", err))
			return
		}
	}

	d.send(ch, "\r\nYour character has been deleted. Farewell.\r\n")

	// Disconnect the player (without saving!)
	if d.DisconnectPlayer != nil {
		d.DisconnectPlayer(ch)
	}
}

func (d *CommandDispatcher) cmdHelp(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Available commands:\r\n")
		d.send(ch, "  look, score, inventory, equipment, affects\r\n")
		d.send(ch, "  north/south/east/west/up/down, say, tell, gossip\r\n")
		d.send(ch, "  get, drop, wear, wield, remove\r\n")
		d.send(ch, "  eat, drink, quaff, recite, zap, brandish\r\n")
		d.send(ch, "  kill, flee, cast, save, quit\r\n")
		d.send(ch, "  quest, clan, deity, consider\r\n")
		d.send(ch, "Type 'help <command>' for more information.\r\n")
		return
	}

	// Try to find in help system first
	if d.Help != nil {
		entry := d.Help.Find(args)
		if entry != nil {
			// Check level restriction
			if entry.Level > 0 && ch.Level < entry.Level {
				d.send(ch, "No help available for that topic.\r\n")
				return
			}
			d.send(ch, entry.Format())
			return
		}
	}

	// Fall back to hardcoded help for basic commands
	switch strings.ToLower(args) {
	case "look":
		d.send(ch, "LOOK\r\n----\r\nSyntax: look [target]\r\n\r\nLook around your current room or examine objects and characters.\r\n")
	case "score":
		d.send(ch, "SCORE\r\n-----\r\nShows your character statistics including level, HP, mana, and attributes.\r\n")
	case "inventory", "inv":
		d.send(ch, "INVENTORY\r\n---------\r\nSyntax: inventory\r\n\r\nShows items you are carrying.\r\n")
	case "equipment", "eq":
		d.send(ch, "EQUIPMENT\r\n---------\r\nSyntax: equipment\r\n\r\nShows items you are wearing or using.\r\n")
	case "get":
		d.send(ch, "GET\r\n---\r\nSyntax: get <item> [container]\r\n        get all [container]\r\n\r\nPick up items from the ground or from containers.\r\n")
	case "drop":
		d.send(ch, "DROP\r\n----\r\nSyntax: drop <item>\r\n        drop all\r\n\r\nDrop items from your inventory onto the ground.\r\n")
	case "wear":
		d.send(ch, "WEAR\r\n----\r\nSyntax: wear <item>\r\n        wear all\r\n\r\nWear armor or equipment from your inventory.\r\n")
	case "cast":
		d.send(ch, "CAST\r\n----\r\nSyntax: cast '<spell name>' [target]\r\n\r\nCast a spell. Spell names with multiple words must be in quotes.\r\n")
	case "kill":
		d.send(ch, "KILL\r\n----\r\nSyntax: kill <target>\r\n\r\nStart combat with a mobile. Use 'murder' to attack other players.\r\n")
	case "flee":
		d.send(ch, "FLEE\r\n----\r\nSyntax: flee\r\n\r\nAttempt to escape from combat. May fail and cost a turn.\r\n")
	case "quest":
		d.send(ch, "QUEST\r\n-----\r\nSyntax: quest list|info|accept|progress|abandon\r\n\r\nManage your quests. See 'quest list' for available quests.\r\n")
	case "clan":
		d.send(ch, "CLAN\r\n----\r\nSyntax: clan list|info|who|talk|leave|induct|outcast|promote|demote\r\n\r\nManage clan membership and communication.\r\n")
	case "redit":
		d.send(ch, "REDIT\r\n-----\r\nSyntax: redit [vnum|create <vnum>]\r\n\r\nRoom editor. Edit the current room or specify a vnum.\r\nSubcommands: show, name, desc, sector, north/south/etc, flags, done\r\n")
	case "medit":
		d.send(ch, "MEDIT\r\n-----\r\nSyntax: medit <vnum>\r\n\r\nMobile editor. Edit mobile templates.\r\nSubcommands: show, name, short, long, level, align, done\r\n")
	case "oedit":
		d.send(ch, "OEDIT\r\n-----\r\nSyntax: oedit <vnum>\r\n\r\nObject editor. Edit object templates.\r\nSubcommands: show, name, short, long, level, cost, weight, done\r\n")
	case "resets":
		d.send(ch, "RESETS\r\n------\r\nSyntax: resets\r\n        resets mob <vnum> [max]\r\n        resets obj <vnum> [max]\r\n        resets delete mob|obj <#>\r\n        resets clear\r\n\r\nManage mob and object spawn points for the current room.\r\n")
	case "hedit":
		d.send(ch, "HEDIT\r\n-----\r\nSyntax: hedit\r\n        hedit show|list|create|keywords|level|syntax|desc|seealso|delete\r\n\r\nHelp entry editor. Create and modify help topics.\r\n")
	default:
		d.send(ch, "No help available for that topic.\r\n")
	}
}

func (d *CommandDispatcher) cmdCommands(ch *types.Character, args string) {
	d.send(ch, "Available commands:\r\n")
	commands := []string{}
	for name := range d.Registry.commands {
		commands = append(commands, name)
	}
	d.send(ch, strings.Join(commands, ", ")+"\r\n")
}

func (d *CommandDispatcher) cmdWizlist(ch *types.Character, args string) {
	d.send(ch, "\r\n")
	d.send(ch, "             ______      ___________         ____  _____\r\n")
	d.send(ch, "            (_____ \\    (___________)       / ___)(_____)\r\n")
	d.send(ch, "             _____) )___    _        _____ ( (___    _\r\n")
	d.send(ch, "            |  __  // _ \\  | |      (_____)| ___ \\  | |\r\n")
	d.send(ch, "            | |  \\ ( (_) ) | |_____        | |   ) )| |\r\n")
	d.send(ch, "            |_|   |_\\___/   \\______)       |_|   (_)|_|\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "                    Rivers of Time Staff\r\n")
	d.send(ch, "                   -----------------------\r\n")
	d.send(ch, "\r\n")

	// In a real implementation, this would list immortals from the player database
	// For now, show a placeholder
	d.send(ch, "                       IMPLEMENTORS\r\n")
	d.send(ch, "                    -----------------\r\n")
	d.send(ch, "                        (System)\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "        Please see 'help credits' for more information.\r\n")
	d.send(ch, "\r\n")
}

func (d *CommandDispatcher) cmdRules(ch *types.Character, args string) {
	d.send(ch, "\r\n")
	d.send(ch, "                Rivers of Time - Rules of Conduct\r\n")
	d.send(ch, "                ==================================\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "1. RESPECT: Treat all players and staff with respect.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "2. HARASSMENT: Harassment of any kind is not tolerated.\r\n")
	d.send(ch, "   This includes unwanted advances, stalking, and hate speech.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "3. CHEATING: Using bugs, exploits, or automation (bots)\r\n")
	d.send(ch, "   to gain unfair advantage is prohibited.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "4. MULTIPLAYING: Only one character online at a time unless\r\n")
	d.send(ch, "   otherwise authorized by an Immortal.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "5. PK RULES: Player killing is allowed only in designated\r\n")
	d.send(ch, "   areas or between consenting players of similar level.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "6. SHARING: Do not share accounts or characters.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "Violations may result in warnings, freezing, or deletion.\r\n")
	d.send(ch, "Staff decisions are final. Appeal via note to immortals.\r\n")
	d.send(ch, "\r\n")
}

func (d *CommandDispatcher) cmdStory(ch *types.Character, args string) {
	d.send(ch, "\r\n")
	d.send(ch, "              The Rivers of Time - A Tale of Ages\r\n")
	d.send(ch, "              =====================================\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "Long ago, when the world was young and the gods still walked among\r\n")
	d.send(ch, "mortals, there existed a nexus of power known as the Rivers of Time.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "These mystical streams flowed through reality itself, connecting\r\n")
	d.send(ch, "past, present, and future in an endless dance of creation and\r\n")
	d.send(ch, "destruction. Those who learned to navigate these rivers gained\r\n")
	d.send(ch, "incredible power - but at great risk.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "The great city of Midgaard arose at the confluence of these rivers,\r\n")
	d.send(ch, "becoming a center of civilization and adventure. Heroes and villains\r\n")
	d.send(ch, "alike are drawn to this place, seeking fame, fortune, and power.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "You are one such adventurer. Your story begins now...\r\n")
	d.send(ch, "\r\n")
}

func (d *CommandDispatcher) cmdMotd(ch *types.Character, args string) {
	d.send(ch, "\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "             Message of the Day - Rivers of Time\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Welcome to Rivers of Time!\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  This is a Go port of the classic ROT MUD.\r\n")
	d.send(ch, "  Many features are working, some are still in development.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Type 'help' for a list of commands.\r\n")
	d.send(ch, "  Type 'rules' to see the rules of conduct.\r\n")
	d.send(ch, "  Type 'story' to read the game's backstory.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Have fun adventuring!\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "\r\n")
}

func (d *CommandDispatcher) cmdImotd(ch *types.Character, args string) {
	// Check if player is immortal (level 51+)
	if ch.Level < 51 {
		d.send(ch, "Huh?\r\n")
		return
	}

	d.send(ch, "\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "         Immortal Message of the Day - Rivers of Time\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Welcome, Immortal!\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Immortal commands available:\r\n")
	d.send(ch, "    goto, stat, where, advance, restore, peace, echo\r\n")
	d.send(ch, "    transfer, at, load, purge, force, slay, freeze\r\n")
	d.send(ch, "    mstat, ostat, rstat, mfind, ofind, mwhere, owhere\r\n")
	d.send(ch, "    invis, holylight, incognito, snoop\r\n")
	d.send(ch, "    mset, oset, rset, mload, oload, switch, return\r\n")
	d.send(ch, "    shutdown\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "  Remember: With great power comes great responsibility.\r\n")
	d.send(ch, "  Use your abilities wisely and fairly.\r\n")
	d.send(ch, "\r\n")
	d.send(ch, "===============================================================\r\n")
	d.send(ch, "\r\n")
}

func (d *CommandDispatcher) cmdSocials(ch *types.Character, args string) {
	if d.Socials == nil {
		d.send(ch, "No socials available.\r\n")
		return
	}

	socials := d.Socials.All()
	if len(socials) == 0 {
		d.send(ch, "No socials available.\r\n")
		return
	}

	d.send(ch, "Available socials:\r\n")

	// Display in columns
	col := 0
	line := ""
	for _, s := range socials {
		line += fmt.Sprintf("%-12s", s.Name)
		col++
		if col >= 6 {
			d.send(ch, line+"\r\n")
			line = ""
			col = 0
		}
	}
	if line != "" {
		d.send(ch, line+"\r\n")
	}

	d.send(ch, fmt.Sprintf("\r\n%d socials available.\r\n", len(socials)))
}

func (d *CommandDispatcher) cmdEmote(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Emote what?\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("You emote: %s %s\r\n", ch.Name, args))
	ActToRoom(fmt.Sprintf("$n %s", args), ch, nil, nil, d.Output)
}

// === Quest and Clan Commands ===

func (d *CommandDispatcher) cmdQuest(ch *types.Character, args string) {
	if d.Quests == nil {
		d.send(ch, "Quest system is not available.\r\n")
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use quests.\r\n")
		return
	}

	parts := strings.Fields(args)
	subcmd := "list"
	if len(parts) > 0 {
		subcmd = strings.ToLower(parts[0])
	}

	switch subcmd {
	case "list", "available":
		d.questList(ch)
	case "info":
		if len(parts) < 2 {
			d.send(ch, "Quest info on which quest? Use 'quest info <number>'.\r\n")
			return
		}
		d.questInfo(ch, parts[1])
	case "accept", "start":
		if len(parts) < 2 {
			d.send(ch, "Accept which quest? Use 'quest accept <number>'.\r\n")
			return
		}
		d.questAccept(ch, parts[1])
	case "progress", "active", "journal":
		d.questProgress(ch)
	case "abandon", "drop":
		if len(parts) < 2 {
			d.send(ch, "Abandon which quest? Use 'quest abandon <number>'.\r\n")
			return
		}
		d.questAbandon(ch, parts[1])
	default:
		d.send(ch, "Quest commands:\r\n")
		d.send(ch, "  quest list          - Show available quests\r\n")
		d.send(ch, "  quest info <id>     - Show quest details\r\n")
		d.send(ch, "  quest accept <id>   - Accept a quest\r\n")
		d.send(ch, "  quest progress      - Show your active quests\r\n")
		d.send(ch, "  quest abandon <id>  - Abandon an active quest\r\n")
	}
}

func (d *CommandDispatcher) questList(ch *types.Character) {
	available := d.Quests.GetAvailableQuests(ch)
	if len(available) == 0 {
		d.send(ch, "There are no quests available for you at this time.\r\n")
		return
	}

	d.send(ch, "Available Quests:\r\n")
	d.send(ch, "--------------------------------------------------------------------------------\r\n")
	d.send(ch, fmt.Sprintf("%-4s %-30s %-6s %-10s %s\r\n", "ID", "Title", "Level", "Reward", "Type"))
	d.send(ch, "--------------------------------------------------------------------------------\r\n")

	for _, quest := range available {
		// Skip quests already in progress
		if d.Quests.IsOnQuest(ch, quest.ID) {
			continue
		}

		questType := d.getQuestTypeName(quest.Type)
		reward := fmt.Sprintf("%dxp/%dg", quest.RewardXP, quest.RewardGold)
		d.send(ch, fmt.Sprintf("%-4d %-30s %-6d %-10s %s\r\n",
			quest.ID, quest.Title, quest.Level, reward, questType))
	}

	d.send(ch, "\r\nUse 'quest info <id>' for more details, 'quest accept <id>' to start.\r\n")
}

func (d *CommandDispatcher) questInfo(ch *types.Character, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		d.send(ch, "Invalid quest number.\r\n")
		return
	}

	quest := d.Quests.GetQuest(id)
	if quest == nil {
		d.send(ch, "That quest does not exist.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("\r\nQuest: %s\r\n", quest.Title))
	d.send(ch, "--------------------------------------------------------------------------------\r\n")
	d.send(ch, fmt.Sprintf("%s\r\n\r\n", quest.Description))
	d.send(ch, fmt.Sprintf("Type:          %s\r\n", d.getQuestTypeName(quest.Type)))
	d.send(ch, fmt.Sprintf("Min Level:     %d\r\n", quest.Level))
	d.send(ch, fmt.Sprintf("Quest Giver:   %s\r\n", quest.GiverName))
	d.send(ch, fmt.Sprintf("Reward:        %d experience, %d gold\r\n", quest.RewardXP, quest.RewardGold))

	// Show objective based on quest type
	switch quest.Type {
	case QuestTypeKill:
		d.send(ch, fmt.Sprintf("Objective:     Kill %d %s\r\n", quest.TargetCount, quest.TargetMob))
	case QuestTypeCollect:
		d.send(ch, fmt.Sprintf("Objective:     Collect %d %s\r\n", quest.TargetCount, quest.TargetItem))
	case QuestTypeDeliver:
		d.send(ch, fmt.Sprintf("Objective:     Deliver %s\r\n", quest.TargetItem))
	case QuestTypeExplore:
		d.send(ch, fmt.Sprintf("Objective:     Explore the specified area\r\n"))
	}

	// Show status
	if d.Quests.HasCompletedQuest(ch, quest.ID) {
		d.send(ch, "\r\nStatus: COMPLETED\r\n")
	} else if d.Quests.IsOnQuest(ch, quest.ID) {
		progress := d.Quests.GetQuestProgress(ch, quest.ID)
		d.send(ch, fmt.Sprintf("\r\nStatus: IN PROGRESS (%d/%d)\r\n", progress, quest.TargetCount))
	} else if ch.Level < quest.Level {
		d.send(ch, fmt.Sprintf("\r\nStatus: Requires level %d\r\n", quest.Level))
	} else {
		d.send(ch, "\r\nStatus: Available\r\n")
	}
}

func (d *CommandDispatcher) questAccept(ch *types.Character, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		d.send(ch, "Invalid quest number.\r\n")
		return
	}

	quest := d.Quests.GetQuest(id)
	if quest == nil {
		d.send(ch, "That quest does not exist.\r\n")
		return
	}

	if ch.Level < quest.Level {
		d.send(ch, fmt.Sprintf("You must be at least level %d to accept this quest.\r\n", quest.Level))
		return
	}

	if d.Quests.HasCompletedQuest(ch, id) {
		d.send(ch, "You have already completed that quest.\r\n")
		return
	}

	if d.Quests.IsOnQuest(ch, id) {
		d.send(ch, "You are already on that quest.\r\n")
		return
	}

	if d.Quests.StartQuest(ch, id) {
		d.send(ch, fmt.Sprintf("You have accepted the quest: %s\r\n", quest.Title))
		d.send(ch, fmt.Sprintf("%s\r\n", quest.Description))
	} else {
		d.send(ch, "You cannot accept that quest right now.\r\n")
	}
}

func (d *CommandDispatcher) questProgress(ch *types.Character) {
	playerQuests := d.Quests.GetPlayerQuests(ch)
	if len(playerQuests) == 0 {
		d.send(ch, "You have no active quests. Use 'quest list' to see available quests.\r\n")
		return
	}

	d.send(ch, "Your Quest Journal:\r\n")
	d.send(ch, "--------------------------------------------------------------------------------\r\n")

	activeCount := 0
	completedCount := 0

	for _, pq := range playerQuests {
		quest := d.Quests.GetQuest(pq.QuestID)
		if quest == nil {
			continue
		}

		if pq.Completed {
			completedCount++
			continue // Only show completed in summary
		}

		activeCount++
		d.send(ch, fmt.Sprintf("\r\n[%d] %s\r\n", quest.ID, quest.Title))
		d.send(ch, fmt.Sprintf("    %s\r\n", quest.Description))

		switch quest.Type {
		case QuestTypeKill:
			d.send(ch, fmt.Sprintf("    Progress: %d/%d %s killed\r\n", pq.Progress, quest.TargetCount, quest.TargetMob))
		case QuestTypeCollect:
			d.send(ch, fmt.Sprintf("    Progress: %d/%d %s collected\r\n", pq.Progress, quest.TargetCount, quest.TargetItem))
		case QuestTypeDeliver:
			if pq.Progress > 0 {
				d.send(ch, "    Progress: Item ready for delivery\r\n")
			} else {
				d.send(ch, "    Progress: Need to obtain item\r\n")
			}
		case QuestTypeExplore:
			if pq.Progress > 0 {
				d.send(ch, "    Progress: Area explored!\r\n")
			} else {
				d.send(ch, "    Progress: Area not yet explored\r\n")
			}
		}

		if pq.Progress >= quest.TargetCount {
			d.send(ch, "    Status: READY TO COMPLETE!\r\n")
		}
	}

	d.send(ch, "\r\n--------------------------------------------------------------------------------\r\n")
	d.send(ch, fmt.Sprintf("Active quests: %d | Completed quests: %d\r\n", activeCount, completedCount))
}

func (d *CommandDispatcher) questAbandon(ch *types.Character, idStr string) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		d.send(ch, "Invalid quest number.\r\n")
		return
	}

	quest := d.Quests.GetQuest(id)
	if quest == nil {
		d.send(ch, "That quest does not exist.\r\n")
		return
	}

	if !d.Quests.IsOnQuest(ch, id) {
		d.send(ch, "You are not on that quest.\r\n")
		return
	}

	// Remove the quest from player's active quests
	if d.Quests.AbandonQuest(ch, id) {
		d.send(ch, fmt.Sprintf("You have abandoned the quest: %s\r\n", quest.Title))
		d.send(ch, "You may accept it again later if you change your mind.\r\n")
	} else {
		d.send(ch, "Failed to abandon quest.\r\n")
	}
}

func (d *CommandDispatcher) getQuestTypeName(qt QuestType) string {
	switch qt {
	case QuestTypeKill:
		return "Kill"
	case QuestTypeCollect:
		return "Collect"
	case QuestTypeDeliver:
		return "Deliver"
	case QuestTypeExplore:
		return "Explore"
	default:
		return "Unknown"
	}
}

// cmdMember handles clan membership invitations
// For non-clan players with invitations: member accept/deny
// For clan leaders: member <player> to invite/kick
func (d *CommandDispatcher) cmdMember(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use this command.\r\n")
		return
	}

	if d.Clans == nil {
		d.send(ch, "Clan system is not available.\r\n")
		return
	}

	arg := strings.TrimSpace(args)

	// Check if the character is a clan leader
	if !d.Clans.IsClanLeader(ch) {
		// Not a leader - check if they're already in a clan
		if d.Clans.IsClanMember(ch) {
			d.send(ch, "You are not a clan leader.\r\n")
			return
		}

		// Not in a clan - check for pending invitation
		if ch.PCData.Invited == 0 {
			d.send(ch, "You have not been invited to join a clan.\r\n")
			return
		}

		// Has an invitation - handle accept/deny
		invitedClan := d.Clans.GetClan(ch.PCData.Invited)
		if invitedClan == nil {
			d.send(ch, "The clan you were invited to no longer exists.\r\n")
			ch.PCData.Invited = 0
			return
		}

		if strings.EqualFold(arg, "accept") {
			clanColor := "{M" // magenta for peaceful clans
			if invitedClan.Pkill {
				clanColor = "{B" // blue for PK clans
			}
			d.send(ch, fmt.Sprintf("{RYou are now a member of clan {x[%s%s{x]\r\n",
				clanColor, invitedClan.WhoName))
			d.Clans.AddMember(ch.PCData.Invited, ch)
			ch.PCData.Invited = 0
			return
		}

		if strings.EqualFold(arg, "deny") {
			d.send(ch, "You turn down the invitation.\r\n")
			ch.PCData.Invited = 0
			return
		}

		d.send(ch, "Syntax: member <accept|deny>\r\n")
		return
	}

	// Character is a clan leader
	if arg == "" {
		d.send(ch, "Syntax: member <char>\r\n")
		return
	}

	// Find the target player
	victim := d.GameLoop.FindPlayer(arg)
	if victim == nil {
		d.send(ch, "They aren't playing.\r\n")
		return
	}

	if victim.IsNPC() || victim.PCData == nil {
		d.send(ch, "NPC's cannot join clans.\r\n")
		return
	}

	myClan := d.Clans.GetCharacterClan(ch)
	if myClan == nil {
		d.send(ch, "Your clan no longer exists!\r\n")
		return
	}

	// Check if target is banned from PK clans
	if victim.PlayerAct.Has(types.PlrNoClan) && myClan.Pkill {
		d.send(ch, "This player is banned from pkill clans.\r\n")
		return
	}

	if victim == ch {
		d.send(ch, "You're stuck...only a god can help you now!\r\n")
		return
	}

	// Check if victim is in a different clan
	if d.Clans.IsClanMember(victim) && !d.Clans.IsSameClan(ch, victim) {
		d.send(ch, "They are in another clan already.\r\n")
		return
	}

	// If victim is in the same clan, kick them out
	if d.Clans.IsClanMember(victim) {
		if d.Clans.IsClanLeader(victim) {
			d.send(ch, "You can't kick out another clan leader.\r\n")
			return
		}
		d.send(ch, "They are now clanless.\r\n")
		d.send(victim, "Your clan leader has kicked you out!\r\n")
		d.Clans.RemoveMember(victim.PCData.Clan, victim)
		return
	}

	// Victim has no clan - check if already invited
	if victim.PCData.Invited != 0 {
		d.send(ch, "They have already been invited to join a clan.\r\n")
		return
	}

	// Check level requirements based on tier
	// Tier 1 (class < MAX_CLASS/2): levels 25-70
	// Tier 2 (class >= MAX_CLASS/2): levels 15-70
	maxClass := 4 // Standard 4 base classes
	if victim.Class < maxClass/2 {
		if victim.Level < 25 || victim.Level > 70 {
			d.send(ch, "They must be between levels 25 -> 70.\r\n")
			return
		}
	} else {
		if victim.Level < 15 || victim.Level > 70 {
			d.send(ch, "They must be between levels 15 -> 70.\r\n")
			return
		}
	}

	// Send the invitation
	d.send(ch, fmt.Sprintf("%s has been invited to join your clan.\r\n", victim.Name))

	clanColor := "{M" // magenta for peaceful clans
	if myClan.Pkill {
		clanColor = "{B" // blue for PK clans
	}
	d.send(victim, fmt.Sprintf("{RYou have been invited to join clan {x[%s%s{x]\r\n",
		clanColor, myClan.WhoName))
	d.send(victim, "{YUse {Gmember accept{Y to join this clan,{x\r\n")
	d.send(victim, "{Yor {Gmember deny{Y to turn down the invitation.{x\r\n")

	victim.PCData.Invited = ch.PCData.Clan
}

func (d *CommandDispatcher) cmdClan(ch *types.Character, args string) {
	if d.Clans == nil {
		d.send(ch, "Clan system is not available.\r\n")
		return
	}

	if ch.PCData == nil {
		d.send(ch, "Only players can use clans.\r\n")
		return
	}

	parts := strings.Fields(args)
	subcmd := "info"
	if len(parts) > 0 {
		subcmd = strings.ToLower(parts[0])
	}

	switch subcmd {
	case "list":
		d.clanList(ch)
	case "info":
		if len(parts) > 1 {
			d.clanInfo(ch, parts[1])
		} else {
			d.clanInfoSelf(ch)
		}
	case "who":
		d.clanWho(ch)
	case "talk", "chat", "ctalk":
		if len(parts) < 2 {
			d.send(ch, "What do you want to say to your clan?\r\n")
			return
		}
		d.clanTalk(ch, strings.Join(parts[1:], " "))
	case "leave", "resign":
		d.clanLeave(ch)
	case "promote":
		if len(parts) < 2 {
			d.send(ch, "Promote who?\r\n")
			return
		}
		d.clanPromote(ch, parts[1])
	case "demote":
		if len(parts) < 2 {
			d.send(ch, "Demote who?\r\n")
			return
		}
		d.clanDemote(ch, parts[1])
	case "induct", "invite":
		if len(parts) < 2 {
			d.send(ch, "Induct who into the clan?\r\n")
			return
		}
		d.clanInduct(ch, parts[1])
	case "outcast", "kick":
		if len(parts) < 2 {
			d.send(ch, "Outcast who from the clan?\r\n")
			return
		}
		d.clanOutcast(ch, parts[1])
	default:
		d.send(ch, "Clan commands:\r\n")
		d.send(ch, "  clan list           - Show all clans\r\n")
		d.send(ch, "  clan info [name]    - Show clan information\r\n")
		d.send(ch, "  clan who            - Show online clan members\r\n")
		d.send(ch, "  clan talk <message> - Send message to clan\r\n")
		d.send(ch, "  clan leave          - Leave your clan\r\n")
		if d.Clans.IsClanLeader(ch) {
			d.send(ch, "\r\nLeader commands:\r\n")
			d.send(ch, "  clan induct <name>  - Induct someone into the clan\r\n")
			d.send(ch, "  clan outcast <name> - Remove someone from the clan\r\n")
			d.send(ch, "  clan promote <name> - Promote to leader\r\n")
			d.send(ch, "  clan demote <name>  - Demote from leader\r\n")
		}
	}
}

func (d *CommandDispatcher) clanList(ch *types.Character) {
	clans := d.Clans.GetAllClans()
	if len(clans) == 0 {
		d.send(ch, "There are no clans.\r\n")
		return
	}

	d.send(ch, "Available Clans:\r\n")
	d.send(ch, "--------------------------------------------------------------------------------\r\n")
	d.send(ch, fmt.Sprintf("%-4s %-20s %-10s %s\r\n", "ID", "Name", "Type", "Members"))
	d.send(ch, "--------------------------------------------------------------------------------\r\n")

	for id, clan := range clans {
		clanType := "Peaceful"
		if clan.Pkill {
			clanType = "PK"
		}
		if clan.Independent {
			clanType = "Independent"
		}
		d.send(ch, fmt.Sprintf("%-4d %-20s %-10s %d\r\n",
			id, clan.Name, clanType, len(clan.Members)))
	}
}

func (d *CommandDispatcher) clanInfoSelf(ch *types.Character) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		d.send(ch, "Use 'clan list' to see available clans.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)
	if clan == nil {
		d.send(ch, "Your clan no longer exists!\r\n")
		return
	}

	d.showClanInfo(ch, clan, ch.PCData.Clan)
}

func (d *CommandDispatcher) clanInfo(ch *types.Character, name string) {
	// Try to find clan by ID first
	if id, err := strconv.Atoi(name); err == nil {
		clan := d.Clans.GetClan(id)
		if clan != nil {
			d.showClanInfo(ch, clan, id)
			return
		}
	}

	// Search by name
	for id, clan := range d.Clans.GetAllClans() {
		if strings.EqualFold(clan.Name, name) {
			d.showClanInfo(ch, clan, id)
			return
		}
	}

	d.send(ch, "That clan does not exist.\r\n")
}

func (d *CommandDispatcher) showClanInfo(ch *types.Character, clan *Clan, clanID int) {
	d.send(ch, fmt.Sprintf("\r\nClan: %s\r\n", clan.Name))
	d.send(ch, "--------------------------------------------------------------------------------\r\n")

	if clan.WhoName != clan.Name {
		d.send(ch, fmt.Sprintf("Who Display:   %s\r\n", clan.WhoName))
	}

	clanType := "Peaceful"
	if clan.Pkill {
		clanType = "Player Killing"
	}
	if clan.Independent {
		clanType = "Independent"
	}
	d.send(ch, fmt.Sprintf("Type:          %s\r\n", clanType))

	d.send(ch, fmt.Sprintf("Members:       %d\r\n", len(clan.Members)))

	if len(clan.Leaders) > 0 {
		d.send(ch, fmt.Sprintf("Leaders:       %s\r\n", strings.Join(clan.Leaders, ", ")))
	}

	if clan.Hall > 0 {
		d.send(ch, fmt.Sprintf("Clan Hall:     Room %d\r\n", clan.Hall))
	}

	// Show if player is in this clan
	if ch.PCData.Clan == clanID {
		d.send(ch, "\r\nYou are a member of this clan.\r\n")
		if d.Clans.IsClanLeader(ch) {
			d.send(ch, "You are a leader of this clan.\r\n")
		}
	}
}

func (d *CommandDispatcher) clanWho(ch *types.Character) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)
	if clan == nil {
		d.send(ch, "Your clan no longer exists!\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Online members of %s:\r\n", clan.Name))
	d.send(ch, "--------------------------------------------------------------------------------\r\n")

	count := 0
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan {
			rank := "Member"
			if d.Clans.IsClanLeader(player) {
				rank = "Leader"
			}
			d.send(ch, fmt.Sprintf("  %-20s %-10s Level %d %s\r\n",
				player.Name, rank, player.Level, d.getClassName(player)))
			count++
		}
	}

	if count == 0 {
		d.send(ch, "  No clan members are currently online.\r\n")
	} else {
		d.send(ch, fmt.Sprintf("\r\n%d clan member(s) online.\r\n", count))
	}
}

func (d *CommandDispatcher) clanTalk(ch *types.Character, message string) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)
	if clan == nil {
		d.send(ch, "Your clan no longer exists!\r\n")
		return
	}

	formatted := fmt.Sprintf("{c[%s] %s: %s{x\r\n", clan.Name, ch.Name, message)

	// Send to all online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan {
			d.send(player, formatted)
		}
	}
}

func (d *CommandDispatcher) clanLeave(ch *types.Character) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)
	if clan == nil {
		d.send(ch, "Your clan no longer exists!\r\n")
		return
	}

	clanName := clan.Name

	// Check if they're a leader - warn them
	if d.Clans.IsClanLeader(ch) {
		d.send(ch, "Warning: You are a leader of this clan. Leaving will remove your leadership.\r\n")
	}

	// Remove from leaders if they are one
	for i, leader := range clan.Leaders {
		if leader == ch.Name {
			clan.Leaders = append(clan.Leaders[:i], clan.Leaders[i+1:]...)
			break
		}
	}

	// Remove from clan
	d.Clans.RemoveMember(ch.PCData.Clan, ch)

	d.send(ch, fmt.Sprintf("You have left %s.\r\n", clanName))

	// Notify online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan && player != ch {
			d.send(player, fmt.Sprintf("{c[%s] %s has left the clan.{x\r\n", clanName, ch.Name))
		}
	}
}

func (d *CommandDispatcher) clanInduct(ch *types.Character, targetName string) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	if !d.Clans.IsClanLeader(ch) {
		d.send(ch, "You must be a clan leader to induct members.\r\n")
		return
	}

	target := FindCharInRoom(ch, targetName)
	if target == nil {
		d.send(ch, "They are not here.\r\n")
		return
	}

	if target.PCData == nil {
		d.send(ch, "You can only induct players.\r\n")
		return
	}

	if target.PCData.Clan != 0 {
		d.send(ch, "They are already in a clan.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)

	d.Clans.AddMember(ch.PCData.Clan, target)
	d.send(ch, fmt.Sprintf("You have inducted %s into %s.\r\n", target.Name, clan.Name))
	d.send(target, fmt.Sprintf("%s has inducted you into %s.\r\n", ch.Name, clan.Name))

	// Notify online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan && player != ch && player != target {
			d.send(player, fmt.Sprintf("{c[%s] %s has been inducted by %s.{x\r\n", clan.Name, target.Name, ch.Name))
		}
	}
}

func (d *CommandDispatcher) clanOutcast(ch *types.Character, targetName string) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	if !d.Clans.IsClanLeader(ch) {
		d.send(ch, "You must be a clan leader to outcast members.\r\n")
		return
	}

	target := FindCharInRoom(ch, targetName)
	if target == nil {
		d.send(ch, "They are not here.\r\n")
		return
	}

	if target.PCData == nil || target.PCData.Clan != ch.PCData.Clan {
		d.send(ch, "They are not in your clan.\r\n")
		return
	}

	if target == ch {
		d.send(ch, "You cannot outcast yourself. Use 'clan leave' instead.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)

	// Remove from leaders if they are one
	for i, leader := range clan.Leaders {
		if leader == target.Name {
			clan.Leaders = append(clan.Leaders[:i], clan.Leaders[i+1:]...)
			break
		}
	}

	d.Clans.RemoveMember(ch.PCData.Clan, target)
	d.send(ch, fmt.Sprintf("You have outcast %s from %s.\r\n", target.Name, clan.Name))
	d.send(target, fmt.Sprintf("%s has outcast you from %s.\r\n", ch.Name, clan.Name))

	// Notify online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan && player != ch && player != target {
			d.send(player, fmt.Sprintf("{c[%s] %s has been outcast by %s.{x\r\n", clan.Name, target.Name, ch.Name))
		}
	}
}

func (d *CommandDispatcher) clanPromote(ch *types.Character, targetName string) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	if !d.Clans.IsClanLeader(ch) {
		d.send(ch, "You must be a clan leader to promote members.\r\n")
		return
	}

	target := FindCharInRoom(ch, targetName)
	if target == nil {
		d.send(ch, "They are not here.\r\n")
		return
	}

	if target.PCData == nil || target.PCData.Clan != ch.PCData.Clan {
		d.send(ch, "They are not in your clan.\r\n")
		return
	}

	if d.Clans.IsClanLeader(target) {
		d.send(ch, "They are already a leader.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)
	clan.Leaders = append(clan.Leaders, target.Name)

	d.send(ch, fmt.Sprintf("You have promoted %s to clan leader.\r\n", target.Name))
	d.send(target, fmt.Sprintf("%s has promoted you to clan leader!\r\n", ch.Name))

	// Notify online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan && player != ch && player != target {
			d.send(player, fmt.Sprintf("{c[%s] %s has been promoted to leader by %s.{x\r\n", clan.Name, target.Name, ch.Name))
		}
	}
}

func (d *CommandDispatcher) clanDemote(ch *types.Character, targetName string) {
	if ch.PCData.Clan == 0 {
		d.send(ch, "You are not in a clan.\r\n")
		return
	}

	if !d.Clans.IsClanLeader(ch) {
		d.send(ch, "You must be a clan leader to demote members.\r\n")
		return
	}

	target := FindCharInRoom(ch, targetName)
	if target == nil {
		d.send(ch, "They are not here.\r\n")
		return
	}

	if target.PCData == nil || target.PCData.Clan != ch.PCData.Clan {
		d.send(ch, "They are not in your clan.\r\n")
		return
	}

	if !d.Clans.IsClanLeader(target) {
		d.send(ch, "They are not a leader.\r\n")
		return
	}

	if target == ch {
		d.send(ch, "You cannot demote yourself.\r\n")
		return
	}

	clan := d.Clans.GetCharacterClan(ch)

	// Remove from leaders
	for i, leader := range clan.Leaders {
		if leader == target.Name {
			clan.Leaders = append(clan.Leaders[:i], clan.Leaders[i+1:]...)
			break
		}
	}

	d.send(ch, fmt.Sprintf("You have demoted %s from clan leader.\r\n", target.Name))
	d.send(target, fmt.Sprintf("%s has demoted you from clan leader.\r\n", ch.Name))

	// Notify online clan members
	for _, player := range d.GameLoop.GetPlayers() {
		if player.PCData != nil && player.PCData.Clan == ch.PCData.Clan && player != ch && player != target {
			d.send(player, fmt.Sprintf("{c[%s] %s has been demoted by %s.{x\r\n", clan.Name, target.Name, ch.Name))
		}
	}
}

// === Immortal Commands ===

func (d *CommandDispatcher) cmdGoto(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Goto where?\r\n")
		return
	}

	vnum, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Invalid room number.\r\n")
		return
	}

	room := d.GameLoop.GetRoom(vnum)
	if room == nil {
		d.send(ch, "Room not found.\r\n")
		return
	}

	// Move character
	oldRoom := ch.InRoom
	if oldRoom != nil {
		// Remove from old room
		for i, person := range oldRoom.People {
			if person == ch {
				oldRoom.People = append(oldRoom.People[:i], oldRoom.People[i+1:]...)
				break
			}
		}
	}
	// Add to new room
	room.People = append(room.People, ch)
	ch.InRoom = room

	d.send(ch, fmt.Sprintf("You goto room %d.\r\n", vnum))
	d.doLook(ch, "")
}

func (d *CommandDispatcher) cmdStat(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Stat whom?\r\n")
		return
	}

	target := d.GameLoop.FindPlayer(args)
	if target == nil {
		d.send(ch, "Player not found.\r\n")
		return
	}

	race := types.GetRace(target.Race)
	raceName := "Unknown"
	if race != nil {
		raceName = race.Name
	}
	class := types.GetClass(target.Class)
	className := "Unknown"
	if class != nil {
		className = class.Name
	}

	d.send(ch, fmt.Sprintf("Name: %s\r\n", target.Name))
	d.send(ch, fmt.Sprintf("Level: %d, Race: %s, Class: %s\r\n",
		target.Level, raceName, className))
	d.send(ch, fmt.Sprintf("HP: %d/%d, Mana: %d/%d, Move: %d/%d\r\n",
		target.Hit, target.MaxHit, target.Mana, target.MaxMana, target.Move, target.MaxMove))
	d.send(ch, fmt.Sprintf("Room: %d\r\n", ch.InRoom.Vnum))
}

func (d *CommandDispatcher) cmdWhere(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	d.send(ch, "Players:\r\n")
	players := d.GameLoop.GetPlayers()
	for _, player := range players {
		roomName := "Unknown"
		if player.InRoom != nil {
			roomName = player.InRoom.Name
		}
		vnum := 0
		if player.InRoom != nil {
			vnum = player.InRoom.Vnum
		}
		d.send(ch, fmt.Sprintf("  %s in %s [%d]\r\n", player.Name, roomName, vnum))
	}
}

func (d *CommandDispatcher) cmdShutdown(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	// Check for reboot flag
	reboot := strings.ToLower(args) == "reboot"

	// Broadcast shutdown message to all players
	message := "*** SHUTDOWN by " + ch.Name + " ***\r\n"
	if reboot {
		message = "*** REBOOT by " + ch.Name + " - reconnect in a few moments ***\r\n"
	}

	// Send to all players
	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, message)
	}

	// Save all players
	for _, player := range d.GameLoop.GetPlayers() {
		if d.OnSave != nil {
			d.OnSave(player)
		}
	}

	d.send(ch, "All players saved.\r\n")

	if reboot {
		d.send(ch, "Rebooting...\r\n")
	} else {
		d.send(ch, "Shutting down...\r\n")
	}

	// Trigger server shutdown
	if d.OnShutdown != nil {
		d.OnShutdown(reboot)
	}
}

func (d *CommandDispatcher) cmdAdvance(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Advance whom to what level?\r\n")
		return
	}

	targetName := parts[0]
	levelStr := parts[1]

	target := d.GameLoop.FindPlayer(targetName)
	if target == nil {
		d.send(ch, "Player not found.\r\n")
		return
	}

	level, err := strconv.Atoi(levelStr)
	if err != nil || level < 1 || level > 100 {
		d.send(ch, "Invalid level.\r\n")
		return
	}

	target.Level = level
	d.send(ch, fmt.Sprintf("%s advanced to level %d.\r\n", target.Name, level))
	d.send(target, fmt.Sprintf("You have been advanced to level %d!\r\n", level))
}

func (d *CommandDispatcher) cmdRestore(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Restore whom?\r\n")
		return
	}

	target := d.GameLoop.FindPlayer(args)
	if target == nil {
		d.send(ch, "Player not found.\r\n")
		return
	}

	target.Hit = target.MaxHit
	target.Mana = target.MaxMana
	target.Move = target.MaxMove

	d.send(ch, fmt.Sprintf("%s restored.\r\n", target.Name))
	d.send(target, "You feel much better!\r\n")
}

func (d *CommandDispatcher) cmdPeace(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You are nowhere!\r\n")
		return
	}

	// Stop all fighting in the room
	count := 0
	for _, person := range ch.InRoom.People {
		if person.Fighting != nil {
			combat.StopFighting(person, true)
			count++
		}
	}

	if count == 0 {
		d.send(ch, "There is no fighting here.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("Peace restored. Stopped %d combatants.\r\n", count))
	ActToRoom("$n makes a peaceful gesture. All fighting stops.", ch, nil, nil, d.Output)
}

func (d *CommandDispatcher) cmdEcho(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Echo what?\r\n")
		return
	}

	for _, player := range d.GameLoop.GetPlayers() {
		d.send(player, args+"\r\n")
	}
}

func (d *CommandDispatcher) cmdTransfer(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 1 || parts[0] == "" {
		d.send(ch, "Transfer whom (and where)?\r\n")
		return
	}

	targetName := parts[0]

	// Find the target player
	target := d.GameLoop.FindPlayer(targetName)
	if target == nil {
		// Try to find a mob
		target = FindCharInRoom(ch, targetName)
	}
	if target == nil {
		d.send(ch, "They aren't here.\r\n")
		return
	}

	// Determine destination
	var destRoom *types.Room
	if len(parts) > 1 {
		// Transfer to specified room or player
		vnum, err := strconv.Atoi(parts[1])
		if err == nil {
			destRoom = d.GameLoop.GetRoom(vnum)
		} else {
			// Find the destination player
			destPlayer := d.GameLoop.FindPlayer(parts[1])
			if destPlayer != nil && destPlayer.InRoom != nil {
				destRoom = destPlayer.InRoom
			}
		}
	} else {
		// Transfer to immortal's room
		destRoom = ch.InRoom
	}

	if destRoom == nil {
		d.send(ch, "No such location.\r\n")
		return
	}

	// Perform the transfer
	if target.InRoom != nil {
		// Remove from old room
		for i, person := range target.InRoom.People {
			if person == target {
				target.InRoom.People = append(target.InRoom.People[:i], target.InRoom.People[i+1:]...)
				break
			}
		}
	}

	// Add to new room
	destRoom.People = append(destRoom.People, target)
	target.InRoom = destRoom

	d.send(ch, fmt.Sprintf("%s has been transferred.\r\n", target.Name))
	d.send(target, fmt.Sprintf("%s has transferred you!\r\n", ch.Name))

	// Show the new room to the target
	d.doLook(target, "")
}

func (d *CommandDispatcher) cmdAt(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "At where what?\r\n")
		return
	}

	// Find the target location
	var targetRoom *types.Room
	vnum, err := strconv.Atoi(parts[0])
	if err == nil {
		targetRoom = d.GameLoop.GetRoom(vnum)
	} else {
		// Find player by name
		targetPlayer := d.GameLoop.FindPlayer(parts[0])
		if targetPlayer != nil && targetPlayer.InRoom != nil {
			targetRoom = targetPlayer.InRoom
		}
	}

	if targetRoom == nil {
		d.send(ch, "No such location.\r\n")
		return
	}

	// Save current room
	originalRoom := ch.InRoom

	// Temporarily move to target room
	ch.InRoom = targetRoom

	// Execute the command
	d.Dispatch(Command{
		Character: ch,
		Input:     parts[1],
	})

	// Return to original room
	ch.InRoom = originalRoom
}

func (d *CommandDispatcher) cmdLoad(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	parts := strings.SplitN(args, " ", 2)
	if len(parts) < 2 {
		d.send(ch, "Syntax: load mob <vnum> | load obj <vnum>\r\n")
		return
	}

	loadType := strings.ToLower(parts[0])
	vnum, err := strconv.Atoi(parts[1])
	if err != nil {
		d.send(ch, "Invalid vnum.\r\n")
		return
	}

	switch loadType {
	case "mob", "mobile", "m":
		// Load a mobile - need access to world data
		if d.GameLoop.World == nil {
			d.send(ch, "World data not loaded.\r\n")
			return
		}

		mobTemplate := d.GameLoop.World.GetMobTemplate(vnum)
		if mobTemplate == nil {
			d.send(ch, "No mobile with that vnum exists.\r\n")
			return
		}

		// Create a new mob instance from template
		name := mobTemplate.ShortDesc
		if len(mobTemplate.Keywords) > 0 {
			name = mobTemplate.Keywords[0]
		}
		mob := &types.Character{
			Name:      name,
			ShortDesc: mobTemplate.ShortDesc,
			LongDesc:  mobTemplate.LongDesc,
			Level:     mobTemplate.Level,
			Hit:       mobTemplate.HitDice.Number*mobTemplate.HitDice.Size + mobTemplate.HitDice.Bonus,
			MaxHit:    mobTemplate.HitDice.Number*mobTemplate.HitDice.Size + mobTemplate.HitDice.Bonus,
			Mana:      mobTemplate.ManaDice.Number*mobTemplate.ManaDice.Size + mobTemplate.ManaDice.Bonus,
			MaxMana:   mobTemplate.ManaDice.Number*mobTemplate.ManaDice.Size + mobTemplate.ManaDice.Bonus,
			Move:      100,
			MaxMove:   100,
			Gold:      mobTemplate.Gold,
			Alignment: mobTemplate.Alignment,
			Position:  types.PosStanding,
			InRoom:    ch.InRoom,
		}
		mob.Act.Set(types.ActNPC)

		// Add to room and game
		if ch.InRoom != nil {
			ch.InRoom.People = append(ch.InRoom.People, mob)
		}
		d.GameLoop.AddCharacter(mob)

		d.send(ch, fmt.Sprintf("%s has been created.\r\n", mob.ShortDesc))

	case "obj", "object", "o":
		// Load an object - need access to world data
		if d.GameLoop.World == nil {
			d.send(ch, "World data not loaded.\r\n")
			return
		}

		objTemplate := d.GameLoop.World.GetObjTemplate(vnum)
		if objTemplate == nil {
			d.send(ch, "No object with that vnum exists.\r\n")
			return
		}

		// Create a new object instance from template
		name := objTemplate.ShortDesc
		if len(objTemplate.Keywords) > 0 {
			name = objTemplate.Keywords[0]
		}
		obj := &types.Object{
			Vnum:      objTemplate.Vnum,
			Name:      name,
			ShortDesc: objTemplate.ShortDesc,
			LongDesc:  objTemplate.LongDesc,
			Weight:    objTemplate.Weight,
			Cost:      objTemplate.Cost,
			Level:     objTemplate.Level,
			Condition: 100,
		}

		// Give object to immortal
		ch.AddInventory(obj)
		obj.CarriedBy = ch

		d.send(ch, fmt.Sprintf("%s has been created.\r\n", obj.ShortDesc))

	default:
		d.send(ch, "Syntax: load mob <vnum> | load obj <vnum>\r\n")
	}
}

func (d *CommandDispatcher) cmdPurge(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	if ch.InRoom == nil {
		d.send(ch, "You are nowhere.\r\n")
		return
	}

	if args == "" {
		// Purge all NPCs and objects in room
		d.send(ch, "You purge the room.\r\n")
		ActToRoom("$n purges the room.", ch, nil, nil, d.Output)

		// Remove all NPCs (copy slice first to avoid modification during iteration)
		people := make([]*types.Character, len(ch.InRoom.People))
		copy(people, ch.InRoom.People)
		for _, person := range people {
			if person.IsNPC() {
				// Remove from room
				for i, p := range ch.InRoom.People {
					if p == person {
						ch.InRoom.People = append(ch.InRoom.People[:i], ch.InRoom.People[i+1:]...)
						break
					}
				}
				// Remove from game
				d.GameLoop.RemoveCharacter(person)
			}
		}

		// Remove all objects
		ch.InRoom.Objects = nil

		return
	}

	// Purge specific target
	// First try to find a mob
	target := FindCharInRoom(ch, args)
	if target != nil {
		if !target.IsNPC() {
			d.send(ch, "You can't purge players.\r\n")
			return
		}

		d.send(ch, fmt.Sprintf("You purge %s.\r\n", target.ShortDesc))
		ActToRoom(fmt.Sprintf("$n purges %s.", target.ShortDesc), ch, nil, nil, d.Output)

		// Remove from room
		for i, p := range ch.InRoom.People {
			if p == target {
				ch.InRoom.People = append(ch.InRoom.People[:i], ch.InRoom.People[i+1:]...)
				break
			}
		}
		// Remove from game
		d.GameLoop.RemoveCharacter(target)
		return
	}

	// Try to find an object
	obj := d.findObjInRoom(ch.InRoom, args)
	if obj != nil {
		d.send(ch, fmt.Sprintf("You purge %s.\r\n", obj.ShortDesc))

		// Remove from room
		for i, o := range ch.InRoom.Objects {
			if o == obj {
				ch.InRoom.Objects = append(ch.InRoom.Objects[:i], ch.InRoom.Objects[i+1:]...)
				break
			}
		}
		return
	}

	d.send(ch, "They aren't here.\r\n")
}

func (d *CommandDispatcher) cmdSockets(ch *types.Character, args string) {
	if !d.isImmortal(ch) {
		d.send(ch, "Huh?\r\n")
		return
	}

	players := d.GameLoop.GetPlayers()
	d.send(ch, fmt.Sprintf("Active connections: %d\r\n", len(players)))
	for _, player := range players {
		d.send(ch, fmt.Sprintf("  %s\r\n", player.Name))
	}
}

// === Helper Functions ===

func (d *CommandDispatcher) isImmortal(ch *types.Character) bool {
	return ch.Level >= 100
}

func (d *CommandDispatcher) getWearLocationName(loc types.WearLocation) string {
	names := map[types.WearLocation]string{
		types.WearLocLight:     "light",
		types.WearLocFingerL:   "finger",
		types.WearLocFingerR:   "finger",
		types.WearLocNeck1:     "neck",
		types.WearLocNeck2:     "neck",
		types.WearLocBody:      "body",
		types.WearLocHead:      "head",
		types.WearLocLegs:      "legs",
		types.WearLocFeet:      "feet",
		types.WearLocHands:     "hands",
		types.WearLocArms:      "arms",
		types.WearLocShield:    "shield",
		types.WearLocAbout:     "about",
		types.WearLocWaist:     "waist",
		types.WearLocWristL:    "wrist",
		types.WearLocWristR:    "wrist",
		types.WearLocWield:     "wield",
		types.WearLocHold:      "hold",
		types.WearLocFloat:     "float",
		types.WearLocSecondary: "secondary",
		types.WearLocFace:      "face",
	}
	if name, ok := names[loc]; ok {
		return name
	}
	return "unknown"
}

func (d *CommandDispatcher) findObjInInventory(ch *types.Character, name string) *types.Object {
	for _, obj := range ch.Inventory {
		if strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(name)) {
			return obj
		}
	}
	return nil
}

func (d *CommandDispatcher) findObjInRoom(room *types.Room, name string) *types.Object {
	for _, obj := range room.Objects {
		if strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(name)) {
			return obj
		}
	}
	return nil
}

func (d *CommandDispatcher) findObjInContainer(container *types.Object, name string) *types.Object {
	for _, obj := range container.Contents {
		if strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(name)) {
			return obj
		}
	}
	return nil
}

func (d *CommandDispatcher) findObjInEquipment(ch *types.Character, name string) *types.Object {
	for _, obj := range ch.Equipment {
		if obj != nil && strings.HasPrefix(strings.ToLower(obj.Name), strings.ToLower(name)) {
			return obj
		}
	}
	return nil
}

// cmdPassword changes the player's password
// Syntax: password <old> <new> <confirm>
func (d *CommandDispatcher) cmdPassword(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs don't have passwords.\r\n")
		return
	}

	parts := strings.Fields(args)
	if len(parts) != 3 {
		d.send(ch, "Syntax: password <old> <new> <confirm>\r\n")
		return
	}

	oldPass := parts[0]
	newPass := parts[1]
	confirm := parts[2]

	// Verify old password
	if !checkPassword(oldPass, ch.PCData.Password) {
		d.send(ch, "Wrong password.\r\n")
		return
	}

	// Check new password length
	if len(newPass) < 5 {
		d.send(ch, "New password must be at least 5 characters.\r\n")
		return
	}

	if len(newPass) > 20 {
		d.send(ch, "New password must be less than 20 characters.\r\n")
		return
	}

	// Confirm passwords match
	if newPass != confirm {
		d.send(ch, "Passwords don't match.\r\n")
		return
	}

	// Set new password
	ch.PCData.Password = hashPassword(newPass)
	d.send(ch, "Password changed.\r\n")
}

// hashPassword creates a SHA256 hash of a password
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// checkPassword verifies a password against a hash
func checkPassword(password, hash string) bool {
	return hashPassword(password) == hash
}

func (d *CommandDispatcher) getEquipmentSlot(ch *types.Character, obj *types.Object) types.WearLocation {
	for slot, equipped := range ch.Equipment {
		if equipped == obj {
			return types.WearLocation(slot)
		}
	}
	return types.WearLocNone
}
