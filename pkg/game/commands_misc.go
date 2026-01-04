package game

import (
	"fmt"
	"math/rand"
	"strings"

	"rotmud/pkg/types"
)

// Voodoo doll vnum from C source (OBJ_VNUM_VOODOO = 51)
const VoodooDollVnum = 51

// cmdPlay handles the play command for jukeboxes
// Ported from do_play in music.c
func (d *CommandDispatcher) cmdPlay(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Play what?\r\n")
		return
	}

	// Find a jukebox in the room
	var jukebox *types.Object
	if ch.InRoom != nil {
		for _, obj := range ch.InRoom.Objects {
			if obj.ItemType == types.ItemTypeJukebox {
				jukebox = obj
				break
			}
		}
	}

	if jukebox == nil {
		d.send(ch, "You see nothing to play.\r\n")
		return
	}

	// Parse args
	parts := strings.SplitN(args, " ", 2)
	cmd := strings.ToLower(parts[0])
	rest := ""
	if len(parts) > 1 {
		rest = parts[1]
	}

	// Handle "play list" command
	if cmd == "list" {
		d.playList(ch, jukebox, rest)
		return
	}

	// Check if jukebox is full (values[1-4] are song queue, -1 = empty)
	// In C: juke->value[4] > -1 means full
	if jukebox.Values[4] >= 0 {
		d.send(ch, "The jukebox is full up right now.\r\n")
		return
	}

	// Find the song by name
	songIndex := d.findSong(args)
	if songIndex < 0 {
		d.send(ch, "That song isn't available.\r\n")
		return
	}

	d.send(ch, "Coming right up.\r\n")

	// Add song to queue
	for i := 1; i <= 4; i++ {
		if jukebox.Values[i] < 0 {
			if i == 1 {
				jukebox.Values[0] = -1 // Reset current line position
			}
			jukebox.Values[i] = songIndex
			return
		}
	}
}

// playList shows available songs on a jukebox
func (d *CommandDispatcher) playList(ch *types.Character, jukebox *types.Object, filter string) {
	songs := d.getAvailableSongs()
	if len(songs) == 0 {
		d.send(ch, fmt.Sprintf("%s has no songs available.\r\n", capitalizeFirst(jukebox.ShortDesc)))
		return
	}

	d.send(ch, fmt.Sprintf("%s has the following songs available:\r\n", capitalizeFirst(jukebox.ShortDesc)))

	// Parse filter for artist mode
	artistMode := false
	artistFilter := ""
	if filter != "" {
		parts := strings.SplitN(filter, " ", 2)
		if strings.ToLower(parts[0]) == "artist" {
			artistMode = true
			if len(parts) > 1 {
				artistFilter = strings.ToLower(parts[1])
			}
		}
	}

	filter = strings.ToLower(filter)
	col := 0
	for _, song := range songs {
		// Apply filter
		if artistMode {
			if artistFilter != "" && !strings.HasPrefix(strings.ToLower(song.Artist), artistFilter) {
				continue
			}
			d.send(ch, fmt.Sprintf("%-39s %-39s\r\n", song.Artist, song.Name))
		} else {
			if filter != "" && filter != "artist" && !strings.HasPrefix(strings.ToLower(song.Name), filter) {
				continue
			}
			d.send(ch, fmt.Sprintf("%-35s ", song.Name))
			col++
			if col%2 == 0 {
				d.send(ch, "\r\n")
			}
		}
	}

	if !artistMode && col%2 != 0 {
		d.send(ch, "\r\n")
	}
}

// Song represents a song in the jukebox
type Song struct {
	Name   string
	Artist string
	Lyrics []string
}

// getAvailableSongs returns the list of available songs
// In the full implementation, this would load from music.txt
func (d *CommandDispatcher) getAvailableSongs() []Song {
	// Placeholder - in the original C code, songs are loaded from music.txt
	return []Song{
		{Name: "Welcome to the Jungle", Artist: "Guns N' Roses"},
		{Name: "Bohemian Rhapsody", Artist: "Queen"},
		{Name: "Stairway to Heaven", Artist: "Led Zeppelin"},
	}
}

// findSong finds a song by name prefix match
func (d *CommandDispatcher) findSong(name string) int {
	name = strings.ToLower(name)
	songs := d.getAvailableSongs()
	for i, song := range songs {
		if strings.HasPrefix(strings.ToLower(song.Name), name) {
			return i
		}
	}
	return -1
}

// cmdVoodoo handles the voodoo command for using voodoo dolls
// Ported from do_voodoo, do_vdpi, do_vdtr, do_vdth in fight.c
func (d *CommandDispatcher) cmdVoodoo(ch *types.Character, args string) {
	// NPCs can't use voodoo
	if ch.IsNPC() {
		return
	}

	// Must be holding a voodoo doll
	doll := ch.GetEquipment(types.WearLocHold)
	if doll == nil || doll.Vnum != VoodooDollVnum {
		d.send(ch, "You are not holding a voodoo doll.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Syntax: voodoo <action>\r\n")
		d.send(ch, "Actions: pin trip throw\r\n")
		return
	}

	// The voodoo doll's name field contains the target's name
	targetName := doll.Name

	action := strings.ToLower(strings.TrimSpace(args))
	switch action {
	case "pin":
		d.voodooPin(ch, targetName)
	case "trip":
		d.voodooTrip(ch, targetName)
	case "throw":
		d.voodooThrow(ch, targetName)
	default:
		d.send(ch, "Syntax: voodoo <action>\r\n")
		d.send(ch, "Actions: pin trip throw\r\n")
	}
}

// voodooPin sticks a pin in the voodoo doll, causing gut pain to the victim
// Ported from do_vdpi in fight.c
func (d *CommandDispatcher) voodooPin(ch *types.Character, targetName string) {
	victim := d.findVoodooVictim(ch, targetName)
	if victim == nil {
		d.send(ch, "Your victim doesn't seem to be in the realm.\r\n")
		return
	}

	if !d.canVoodoo(ch, victim) {
		return
	}

	// Perform the voodoo
	d.send(ch, "You stick a pin into your voodoo doll.\r\n")
	ActToRoom("$n sticks a pin into a voodoo doll.", ch, nil, nil, d.Output)

	// Victim feels pain
	d.send(victim, "{RYou double over with a sudden pain in your gut!{x\r\n")
	ActToRoom("$n suddenly doubles over with a look of extreme pain!", victim, nil, nil, d.Output)

	// Apply voodoo protection to prevent spam
	d.applyVoodooProtection(victim)
}

// voodooTrip slams the voodoo doll against the ground, tripping the victim
// Ported from do_vdtr in fight.c
func (d *CommandDispatcher) voodooTrip(ch *types.Character, targetName string) {
	victim := d.findVoodooVictim(ch, targetName)
	if victim == nil {
		d.send(ch, "Your victim doesn't seem to be in the realm.\r\n")
		return
	}

	if !d.canVoodoo(ch, victim) {
		return
	}

	// Perform the voodoo
	d.send(ch, "You slam your voodoo doll against the ground.\r\n")
	ActToRoom("$n slams a voodoo doll against the ground.", ch, nil, nil, d.Output)

	// Victim trips
	d.send(victim, "{RYour feet slide out from under you!{x\r\n")
	d.send(victim, "{RYou hit the ground face first!{x\r\n")
	ActToRoom("$n trips over $s own feet, and does a nose dive into the ground!", victim, nil, nil, d.Output)

	// Apply voodoo protection
	d.applyVoodooProtection(victim)
}

// voodooThrow tosses the voodoo doll, throwing the victim through the air
// Ported from do_vdth in fight.c
func (d *CommandDispatcher) voodooThrow(ch *types.Character, targetName string) {
	victim := d.findVoodooVictim(ch, targetName)
	if victim == nil {
		d.send(ch, "Your victim doesn't seem to be in the realm.\r\n")
		return
	}

	if !d.canVoodoo(ch, victim) {
		return
	}

	// Perform the voodoo
	d.send(ch, "You toss your voodoo doll into the air.\r\n")
	ActToRoom("$n tosses a voodoo doll into the air.", ch, nil, nil, d.Output)

	// Apply voodoo protection first
	d.applyVoodooProtection(victim)

	// If victim is fighting or 25% chance, just slam into wall
	if victim.Fighting != nil || rand.Intn(100) < 25 {
		d.send(victim, "{RA sudden gust of wind throws you through the air!{x\r\n")
		d.send(victim, "{RYou slam face first into the nearest wall!{x\r\n")
		ActToRoom("A sudden gust of wind picks up $n and throws $m into a wall!", victim, nil, nil, d.Output)
		return
	}

	// Try to move victim to adjacent room
	victim.Position = types.PosStanding
	wasInRoom := victim.InRoom

	// Try up to 6 random directions
	for attempt := 0; attempt < 6; attempt++ {
		dir := types.Direction(rand.Intn(int(types.DirMax)))

		if wasInRoom == nil {
			break
		}

		exit := wasInRoom.GetExit(dir)
		if exit == nil || exit.ToRoom == nil {
			continue
		}

		// Can't go through closed doors
		if exit.Flags.Has(types.ExitClosed) {
			continue
		}

		// NPCs can't enter NO_MOB rooms
		if victim.IsNPC() && exit.ToRoom.Flags.Has(types.RoomNoMob) {
			continue
		}

		// Move the victim
		nowInRoom := exit.ToRoom

		// Show departure message
		victim.InRoom = wasInRoom
		ActToRoom(fmt.Sprintf("A sudden gust of wind picks up $n and throws $m to the %s.", dir.String()), victim, nil, nil, d.Output)

		// Move character
		CharFromRoom(victim)
		CharToRoom(victim, nowInRoom)

		// Arrival message and look
		ActToRoom("$n sails into the room and slams face first into a wall!", victim, nil, nil, d.Output)
		d.cmdLook(victim, "auto")
		d.send(victim, "{RA sudden gust of wind throws you through the air!{x\r\n")
		d.send(victim, "{RYou slam face first into the nearest wall!{x\r\n")
		return
	}

	// Couldn't move, just slam into wall
	d.send(victim, "{RA sudden gust of wind throws you through the air!{x\r\n")
	d.send(victim, "{RYou slam face first into the nearest wall!{x\r\n")
	ActToRoom("A sudden gust of wind picks up $n and throws $m into a wall!", victim, nil, nil, d.Output)
}

// findVoodooVictim finds a player matching the voodoo doll's name
func (d *CommandDispatcher) findVoodooVictim(ch *types.Character, targetName string) *types.Character {
	targetName = strings.ToLower(targetName)

	// Search all connected players
	if d.GameLoop == nil {
		return nil
	}

	for _, victim := range d.GameLoop.GetPlayers() {
		// Must be able to see them (simplified - in C uses can_see)
		if victim.IsNPC() {
			continue
		}

		// Match by exact name (doll is named for specific person)
		if strings.EqualFold(victim.Name, targetName) {
			return victim
		}

		// Also allow keyword matching (the doll's name might contain keywords)
		if nameMatches(targetName, strings.ToLower(victim.Name)) {
			return victim
		}
	}

	return nil
}

// canVoodoo checks if the voodoo action can be performed
func (d *CommandDispatcher) canVoodoo(ch *types.Character, victim *types.Character) bool {
	// Can't voodoo immortals of higher level
	if victim.IsImmortal() && victim.Level > ch.Level {
		d.send(ch, "That's not a good idea.\r\n")
		return false
	}

	// Can't voodoo players below level 20 (unless immortal)
	if victim.Level < 20 && !ch.IsImmortal() {
		d.send(ch, "They are a little too young for that.\r\n")
		return false
	}

	// Can't voodoo someone already protected
	if victim.IsShielded(types.ShdProtectVoodoo) {
		d.send(ch, "They are still reeling from a previous voodoo.\r\n")
		return false
	}

	return true
}

// applyVoodooProtection applies the voodoo protection shield to prevent spam
func (d *CommandDispatcher) applyVoodooProtection(victim *types.Character) {
	aff := &types.Affect{
		Type:         "protection voodoo",
		Level:        victim.Level,
		Duration:     1, // 1 tick
		Location:     types.ApplyNone,
		Modifier:     0,
		ShieldVector: types.ShdProtectVoodoo,
	}
	victim.AddAffect(aff)
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
