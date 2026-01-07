package game

import (
	"fmt"
	"math/rand"
	"strings"

	"rotmud/pkg/types"
)

// Additional communication commands: shout, yell, afk

func (d *CommandDispatcher) cmdShout(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommNoShout) {
		d.send(ch, "You can't shout.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Shout what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	// Check mana cost (shouting costs some mana in ROM/ROT)
	manaCost := 10
	if ch.Mana < manaCost {
		d.send(ch, "You don't have enough energy to shout.\r\n")
		return
	}
	ch.Mana -= manaCost

	// Add lag
	ch.Wait += 12 // 3 seconds

	d.send(ch, fmt.Sprintf("You shout '%s'\r\n", args))

	// Send to all players in the same area
	for _, player := range d.GameLoop.GetPlayers() {
		if player != ch && !player.Comm.Has(types.CommShoutsOff) {
			// In same area check
			if player.InRoom != nil && ch.InRoom != nil {
				if player.InRoom.Area == ch.InRoom.Area {
					d.send(player, fmt.Sprintf("%s shouts '%s'\r\n", ch.Name, args))
				} else {
					// Distant shout
					d.send(player, fmt.Sprintf("Someone shouts '%s' in the distance.\r\n", args))
				}
			}
		}
	}
}

func (d *CommandDispatcher) cmdYell(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommNoShout) {
		d.send(ch, "You can't yell.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Yell what?\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed!\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("You yell '%s'\r\n", args))

	// Send to all players in the same area (no mana cost unlike shout)
	for _, player := range d.GameLoop.GetPlayers() {
		if player != ch && !player.Comm.Has(types.CommShoutsOff) {
			// In same area check
			if player.InRoom != nil && ch.InRoom != nil {
				if player.InRoom.Area == ch.InRoom.Area {
					d.send(player, fmt.Sprintf("%s yells '%s'\r\n", ch.Name, args))
				}
			}
		}
	}
}

func (d *CommandDispatcher) cmdAFK(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommAFK) {
		ch.Comm.Remove(types.CommAFK)
		d.send(ch, "AFK mode removed. Type 'replay' to see tells.\r\n")
	} else {
		ch.Comm.Set(types.CommAFK)
		d.send(ch, "You are now in AFK mode.\r\n")
	}
}

func (d *CommandDispatcher) cmdQuiet(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommQuiet) {
		ch.Comm.Remove(types.CommQuiet)
		d.send(ch, "Quiet mode removed.\r\n")
	} else {
		ch.Comm.Set(types.CommQuiet)
		d.send(ch, "You are now in quiet mode (no channels).\r\n")
	}
}

func (d *CommandDispatcher) cmdDeaf(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommDeaf) {
		ch.Comm.Remove(types.CommDeaf)
		d.send(ch, "You can now hear channel communications.\r\n")
	} else {
		ch.Comm.Set(types.CommDeaf)
		d.send(ch, "You can no longer hear channel communications.\r\n")
	}
}

func (d *CommandDispatcher) cmdChannels(ch *types.Character, args string) {
	d.send(ch, "Channel settings:\r\n")
	d.send(ch, fmt.Sprintf("  gossip:  %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoGossip))))
	d.send(ch, fmt.Sprintf("  music:   %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoMusic))))
	d.send(ch, fmt.Sprintf("  grats:   %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoGrats))))
	d.send(ch, fmt.Sprintf("  ask:     %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoAsk))))
	d.send(ch, fmt.Sprintf("  quote:   %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoQuote))))
	d.send(ch, fmt.Sprintf("  shouts:  %s\r\n", channelOnOff(!ch.Comm.Has(types.CommShoutsOff))))
	d.send(ch, fmt.Sprintf("  cgossip: %s\r\n", channelOnOff(!ch.Comm.Has(types.CommNoCGossip))))
	d.send(ch, fmt.Sprintf("  quiet:   %s\r\n", channelOnOff(ch.Comm.Has(types.CommQuiet))))
	d.send(ch, fmt.Sprintf("  deaf:    %s\r\n", channelOnOff(ch.Comm.Has(types.CommDeaf))))
}

func channelOnOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

// cmdPmote - Personal emote that replaces character names with "you"
func (d *CommandDispatcher) cmdPmote(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommNoEmote) {
		d.send(ch, "You can't show your emotions.\r\n")
		return
	}

	if args == "" {
		d.send(ch, "Emote what?\r\n")
		return
	}

	// Send to self
	d.send(ch, fmt.Sprintf("%s %s\r\n", ch.Name, args))

	// Send to others in room, replacing their name with "you"
	if ch.InRoom != nil {
		for _, vch := range ch.InRoom.People {
			if vch == ch {
				continue
			}

			// Check if the victim's name appears in the emote
			personalizedMsg := replaceNameWithYou(args, vch.Name)
			d.send(vch, fmt.Sprintf("%s %s\r\n", ch.Name, personalizedMsg))
		}
	}
}

// replaceNameWithYou replaces a character's name with "you" in text
func replaceNameWithYou(text, name string) string {
	// Simple case-insensitive replacement
	result := text
	lowerText := strings.ToLower(text)
	lowerName := strings.ToLower(name)

	startIdx := 0
	for {
		idx := strings.Index(lowerText[startIdx:], lowerName)
		if idx == -1 {
			break
		}

		actualIdx := startIdx + idx
		// Replace name with "you"
		result = result[:actualIdx] + "you" + result[actualIdx+len(name):]
		lowerText = strings.ToLower(result)
		startIdx = actualIdx + 3 // len("you")
	}

	return result
}

// cmdPose - Shows a class-specific pose message
func (d *CommandDispatcher) cmdPose(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	// Get class name from class index
	className := types.ClassName(ch.Class)

	// Pose messages based on class
	poses := getPoseMessages(className)
	if len(poses) == 0 {
		d.send(ch, "You strike a heroic pose!\r\n")
		ActToRoom("$n strikes a heroic pose!", ch, nil, nil, d.Output)
		return
	}

	// Pick a random pose
	idx := rand.Intn(len(poses))
	pose := poses[idx]

	d.send(ch, pose.ToChar+"\r\n")
	ActToRoom(pose.ToRoom, ch, nil, nil, d.Output)
}

// PoseMessage contains the message shown to character and room
type PoseMessage struct {
	ToChar string
	ToRoom string
}

// getPoseMessages returns pose messages for a class
func getPoseMessages(class string) []PoseMessage {
	// Class-specific poses
	switch strings.ToLower(class) {
	case "mage":
		return []PoseMessage{
			{"Blue sparks fly from your fingers.", "Blue sparks fly from $n's fingers."},
			{"A small fire crackles in your palm.", "A small fire crackles in $n's palm."},
			{"You shimmer with magical energy.", "$n shimmers with magical energy."},
			{"Your eyes glow briefly with power.", "$n's eyes glow briefly with power."},
			{"The air crackles with electricity around you.", "The air crackles with electricity around $n."},
		}
	case "cleric":
		return []PoseMessage{
			{"You are surrounded by a holy aura.", "$n is surrounded by a holy aura."},
			{"A beam of light shines down upon you.", "A beam of light shines down upon $n."},
			{"You recite a prayer to the gods.", "$n recites a prayer to the gods."},
			{"Your holy symbol glows warmly.", "$n's holy symbol glows warmly."},
			{"Divine energy swirls around you.", "Divine energy swirls around $n."},
		}
	case "thief":
		return []PoseMessage{
			{"You flip a coin deftly.", "$n flips a coin deftly."},
			{"You check your daggers.", "$n checks $s daggers."},
			{"You produce a lockpick from nowhere.", "$n produces a lockpick from nowhere."},
			{"Your hands move with blinding speed.", "$n's hands move with blinding speed."},
			{"You vanish into the shadows momentarily.", "$n vanishes into the shadows momentarily."},
		}
	case "warrior":
		return []PoseMessage{
			{"Your muscles ripple impressively.", "$n's muscles ripple impressively."},
			{"You crack your knuckles menacingly.", "$n cracks $s knuckles menacingly."},
			{"You flex your battle-hardened arms.", "$n flexes $s battle-hardened arms."},
			{"A scar on your face catches the light.", "A scar on $n's face catches the light."},
			{"You grunt with barely contained strength.", "$n grunts with barely contained strength."},
		}
	default:
		return []PoseMessage{
			{"You strike a heroic pose!", "$n strikes a heroic pose!"},
			{"You look around confidently.", "$n looks around confidently."},
			{"You stand tall and proud.", "$n stands tall and proud."},
		}
	}
}

// randomRange returns a random number between min and max (inclusive)
func randomRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + rand.Intn(max-min+1)
}

// cmdQuote - Quote channel
func (d *CommandDispatcher) cmdQuote(ch *types.Character, args string) {
	if args == "" {
		// Toggle quote channel
		if ch.Comm.Has(types.CommNoQuote) {
			ch.Comm.Remove(types.CommNoQuote)
			d.send(ch, "Quote channel is now ON.\r\n")
		} else {
			ch.Comm.Set(types.CommNoQuote)
			d.send(ch, "Quote channel is now OFF.\r\n")
		}
		return
	}

	if ch.Comm.Has(types.CommQuiet) {
		d.send(ch, "You must turn off quiet mode first.\r\n")
		return
	}

	if ch.Comm.Has(types.CommNoChannels) {
		d.send(ch, "The gods have revoked your channel privileges.\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed on channels!\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("You quote '%s'\r\n", args))

	// Send to all players who can hear quote channel
	for _, player := range d.GameLoop.GetPlayers() {
		if player != ch && !player.Comm.Has(types.CommNoQuote) && !player.Comm.Has(types.CommQuiet) {
			d.send(player, fmt.Sprintf("%s quotes '%s'\r\n", ch.Name, args))
		}
	}
}

// cmdReplay - Replay buffered tells
func (d *CommandDispatcher) cmdReplay(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "You can't replay.\r\n")
		return
	}

	// Check if there are buffered tells
	if len(ch.PCData.TellBuffer) == 0 {
		d.send(ch, "You have no tells to replay.\r\n")
		return
	}

	d.send(ch, "Replaying tells:\r\n")
	for _, tell := range ch.PCData.TellBuffer {
		d.send(ch, tell+"\r\n")
	}

	// Clear the buffer
	ch.PCData.TellBuffer = nil
}

// cmdQgossip - Quest gossip channel
func (d *CommandDispatcher) cmdQgossip(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Quest gossip what?\r\n")
		return
	}

	if ch.Comm.Has(types.CommQuiet) {
		d.send(ch, "You must turn off quiet mode first.\r\n")
		return
	}

	if ch.Comm.Has(types.CommNoChannels) {
		d.send(ch, "The gods have revoked your channel privileges.\r\n")
		return
	}

	if containsProfanity(args) {
		d.send(ch, "Profanity is not allowed on channels!\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("[Quest] You: %s\r\n", args))

	// Send to all players who are on a quest
	for _, player := range d.GameLoop.GetPlayers() {
		if player != ch && !player.Comm.Has(types.CommQuiet) {
			// Check if player has any active quests
			if player.PCData != nil && len(player.PCData.QuestProgress) > 0 {
				d.send(player, fmt.Sprintf("[Quest] %s: %s\r\n", ch.Name, args))
			}
		}
	}
}

// MaxForgetList is the maximum number of players that can be ignored
const MaxForgetList = 10

// cmdForget - Add a player to ignore list
func (d *CommandDispatcher) cmdForget(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	args = strings.TrimSpace(args)

	// No argument - show current forget list
	if args == "" {
		if len(ch.PCData.ForgetList) == 0 {
			d.send(ch, "You are not forgetting anyone.\r\n")
			return
		}
		d.send(ch, "You are currently forgetting:\r\n")
		for _, name := range ch.PCData.ForgetList {
			d.send(ch, fmt.Sprintf("    %s\r\n", name))
		}
		return
	}

	// Normalize the name
	targetName := strings.ToLower(args)

	// Check if already in forget list
	for _, name := range ch.PCData.ForgetList {
		if strings.ToLower(name) == targetName {
			d.send(ch, "You have already forgotten that person.\r\n")
			return
		}
	}

	// Can't forget yourself
	if strings.ToLower(ch.Name) == targetName {
		d.send(ch, "You forget yourself for a moment, but it passes.\r\n")
		return
	}

	// Check if the player exists and is online
	var targetPlayer *types.Character
	for _, player := range d.GameLoop.GetPlayers() {
		if strings.ToLower(player.Name) == targetName {
			targetPlayer = player
			break
		}
	}

	if targetPlayer == nil {
		d.send(ch, "No one by that name is playing.\r\n")
		return
	}

	// Can't forget immortals
	if targetPlayer.Level >= 51 {
		d.send(ch, "That person is very hard to forget.\r\n")
		return
	}

	// Check forget limit
	if len(ch.PCData.ForgetList) >= MaxForgetList {
		d.send(ch, "Sorry, you have reached the forget limit.\r\n")
		return
	}

	// Add to forget list (store with proper capitalization)
	ch.PCData.ForgetList = append(ch.PCData.ForgetList, targetPlayer.Name)
	d.send(ch, fmt.Sprintf("You are now deaf to %s.\r\n", targetPlayer.Name))
}

// cmdRemember - Remove a player from ignore list
func (d *CommandDispatcher) cmdRemember(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	args = strings.TrimSpace(args)

	// No argument - show current forget list
	if args == "" {
		if len(ch.PCData.ForgetList) == 0 {
			d.send(ch, "You are not forgetting anyone.\r\n")
			return
		}
		d.send(ch, "You are currently forgetting:\r\n")
		for _, name := range ch.PCData.ForgetList {
			d.send(ch, fmt.Sprintf("    %s\r\n", name))
		}
		return
	}

	// Normalize the name
	targetName := strings.ToLower(args)

	// Find and remove from forget list
	found := false
	newList := make([]string, 0, len(ch.PCData.ForgetList))
	for _, name := range ch.PCData.ForgetList {
		if strings.ToLower(name) == targetName {
			found = true
		} else {
			newList = append(newList, name)
		}
	}

	if !found {
		d.send(ch, "No one by that name is forgotten.\r\n")
		return
	}

	ch.PCData.ForgetList = newList
	d.send(ch, "Forget removed.\r\n")
}

// IsForget checks if a character is on another's forget list
func IsForget(ch, target *types.Character) bool {
	if ch == nil || target == nil || ch.PCData == nil {
		return false
	}

	targetName := strings.ToLower(target.Name)
	for _, name := range ch.PCData.ForgetList {
		if strings.ToLower(name) == targetName {
			return true
		}
	}
	return false
}

// cmdForge - Stub to prevent confusion with 'forget'
func (d *CommandDispatcher) cmdForge(ch *types.Character, args string) {
	d.send(ch, "I'm sorry, forget must be entered in full.\r\n")
}
