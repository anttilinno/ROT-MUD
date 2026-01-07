package game

import (
	"rotmud/pkg/types"
	"strings"
)

// MOBprogTrigger represents different triggers for MOB programs
type MOBprogTrigger int

const (
	TriggerSpeech MOBprogTrigger = iota
	TriggerAct
	TriggerFight
	TriggerDeath
	TriggerEntry
	TriggerGreet
	TriggerGive
	TriggerBribe
)

// MOBprog represents a mobile program/script
type MOBprog struct {
	ID       int            `json:"id"`
	MobVnum  int            `json:"mob_vnum"` // Which mob this program is for
	Trigger  MOBprogTrigger `json:"trigger"`
	Phrase   string         `json:"phrase,omitempty"` // Trigger phrase for speech/act triggers
	Commands []string       `json:"commands"`         // Commands to execute
}

// CommandExecutor is a function that can execute commands for a mob
type CommandExecutor func(mob *types.Character, command string, args string)

// RoomMessageSender is a function that sends messages to everyone in a room
type RoomMessageSender func(room *types.Room, message string)

// MOBprogSystem manages all MOB programs
type MOBprogSystem struct {
	programs    map[int][]*MOBprog // Mob vnum to list of programs
	executeCmd  CommandExecutor
	sendRoomMsg RoomMessageSender
}

// NewMOBprogSystem creates a new MOB program system
func NewMOBprogSystem() *MOBprogSystem {
	mps := &MOBprogSystem{
		programs: make(map[int][]*MOBprog),
	}

	mps.initializeDefaultPrograms()
	return mps
}

// SetCommandExecutor sets the function used to execute commands
func (mps *MOBprogSystem) SetCommandExecutor(executor CommandExecutor) {
	mps.executeCmd = executor
}

// SetRoomMessageSender sets the function used to send room messages
func (mps *MOBprogSystem) SetRoomMessageSender(sender RoomMessageSender) {
	mps.sendRoomMsg = sender
}

// initializeDefaultPrograms sets up some default MOB programs
func (mps *MOBprogSystem) initializeDefaultPrograms() {
	programs := []*MOBprog{
		// Guard that responds to "hail"
		{
			ID:      1,
			MobVnum: 100, // Example guard vnum
			Trigger: TriggerSpeech,
			Phrase:  "hail",
			Commands: []string{
				"say Greetings, adventurer! The city is safe under my watch.",
				"emote stands at attention.",
			},
		},
		// Shopkeeper that responds to "buy"
		{
			ID:      2,
			MobVnum: 200, // Example shopkeeper vnum
			Trigger: TriggerSpeech,
			Phrase:  "buy",
			Commands: []string{
				"say Take a look at my wares, friend!",
				"emote gestures to the merchandise.",
			},
		},
		// Aggressive mob that taunts in combat
		{
			ID:      3,
			MobVnum: 300, // Example monster vnum
			Trigger: TriggerFight,
			Commands: []string{
				"say You'll regret challenging me!",
				"emote snarls menacingly.",
			},
		},
		// Mob that drops a key on death
		{
			ID:      4,
			MobVnum: 400, // Example boss vnum
			Trigger: TriggerDeath,
			Commands: []string{
				"echo The defeated foe drops a glowing key!",
				// In a full implementation, this would spawn an item
			},
		},
		// Bandit leader who taunts players
		{
			ID:      5,
			MobVnum: 500, // Bandit leader
			Trigger: TriggerSpeech,
			Phrase:  "surrender",
			Commands: []string{
				"say Ha! I don't surrender to weaklings like you!",
				"emote laughs maniacally.",
			},
		},
		// Merchant who responds to "trade"
		{
			ID:      6,
			MobVnum: 600, // Merchant
			Trigger: TriggerSpeech,
			Phrase:  "trade",
			Commands: []string{
				"say I have fine wares for sale! Take a look.",
				"emote opens his merchant pack.",
			},
		},
		// Guard who warns about danger
		{
			ID:      7,
			MobVnum: 100, // Guard
			Trigger: TriggerEntry,
			Commands: []string{
				"say Beware, adventurer! Dangerous creatures lurk in these parts.",
				"emote points to the shadows.",
			},
		},
		// Innkeeper who offers services
		{
			ID:      8,
			MobVnum: 700, // Innkeeper
			Trigger: TriggerSpeech,
			Phrase:  "stay",
			Commands: []string{
				"say Welcome to my inn! Rooms are 10 gold per night.",
				"say Just say 'rent room' to get a room.",
			},
		},
	}

	for _, prog := range programs {
		mps.AddProgram(prog)
	}
}

// AddProgram adds a MOB program
func (mps *MOBprogSystem) AddProgram(prog *MOBprog) {
	mps.programs[prog.MobVnum] = append(mps.programs[prog.MobVnum], prog)
}

// AddProgramFromData creates and adds a MOB program from loader data
// trigger: speech, act, fight, death, entry, greet, give, bribe
// triggerArg: phrase for speech/act, amount for bribe, percent for percentage triggers
func (mps *MOBprogSystem) AddProgramFromData(mobVnum int, trigger string, triggerArg string, commands []string) {
	trigType := ParseTriggerType(trigger)
	prog := &MOBprog{
		MobVnum:  mobVnum,
		Trigger:  trigType,
		Phrase:   triggerArg,
		Commands: commands,
	}
	mps.AddProgram(prog)
}

// ParseTriggerType converts a trigger string to MOBprogTrigger
func ParseTriggerType(trigger string) MOBprogTrigger {
	switch strings.ToLower(trigger) {
	case "speech":
		return TriggerSpeech
	case "act":
		return TriggerAct
	case "fight":
		return TriggerFight
	case "death":
		return TriggerDeath
	case "entry":
		return TriggerEntry
	case "greet":
		return TriggerGreet
	case "give":
		return TriggerGive
	case "bribe":
		return TriggerBribe
	default:
		return TriggerSpeech
	}
}

// TriggerTypeString returns the string representation of a trigger type
func TriggerTypeString(trigger MOBprogTrigger) string {
	switch trigger {
	case TriggerSpeech:
		return "speech"
	case TriggerAct:
		return "act"
	case TriggerFight:
		return "fight"
	case TriggerDeath:
		return "death"
	case TriggerEntry:
		return "entry"
	case TriggerGreet:
		return "greet"
	case TriggerGive:
		return "give"
	case TriggerBribe:
		return "bribe"
	default:
		return "unknown"
	}
}

// ClearPrograms removes all programs (useful before reloading from files)
func (mps *MOBprogSystem) ClearPrograms() {
	mps.programs = make(map[int][]*MOBprog)
}

// GetPrograms returns all programs for a specific mob
func (mps *MOBprogSystem) GetPrograms(mobVnum int) []*MOBprog {
	return mps.programs[mobVnum]
}

// ExecuteTrigger executes programs for a trigger on a specific mob
func (mps *MOBprogSystem) ExecuteTrigger(trigger MOBprogTrigger, mob *types.Character, args ...interface{}) {
	programs := mps.GetPrograms(mob.MobVnum)
	if len(programs) == 0 {
		return
	}

	for _, prog := range programs {
		if prog.Trigger != trigger {
			continue
		}

		// Check trigger-specific conditions
		switch trigger {
		case TriggerSpeech:
			if len(args) > 0 {
				if phrase, ok := args[0].(string); ok {
					if !strings.Contains(strings.ToLower(phrase), strings.ToLower(prog.Phrase)) {
						continue
					}
				}
			}
		case TriggerAct:
			// Check if the act contains the phrase
			if len(args) > 0 {
				if actText, ok := args[0].(string); ok {
					if !strings.Contains(strings.ToLower(actText), strings.ToLower(prog.Phrase)) {
						continue
					}
				}
			}
		case TriggerGive:
			// Check if the given item matches
			if len(args) > 0 {
				if item, ok := args[0].(*types.Object); ok {
					if !strings.Contains(strings.ToLower(item.Name), strings.ToLower(prog.Phrase)) {
						continue
					}
				}
			}
		case TriggerBribe:
			// Check bribe amount
			if len(args) > 0 {
				if amount, ok := args[0].(int); ok {
					// Could check if amount matches some condition
					_ = amount
				}
			}
		}

		// Execute the commands
		mps.executeCommands(prog.Commands, mob, args...)
	}
}

// executeCommands executes a list of commands as if the mob was performing them
func (mps *MOBprogSystem) executeCommands(commands []string, mob *types.Character, args ...interface{}) {
	if mob == nil || mob.InRoom == nil {
		return
	}

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		if cmd == "" {
			continue
		}

		parts := strings.SplitN(cmd, " ", 2)
		command := strings.ToLower(parts[0])
		arg := ""
		if len(parts) > 1 {
			arg = parts[1]
		}

		switch command {
		case "echo":
			// Send message directly to room
			if mps.sendRoomMsg != nil {
				mps.sendRoomMsg(mob.InRoom, arg+"\r\n")
			}

		case "say", "emote", "yell", "shout", "give", "drop", "get", "wear", "wield",
			"remove", "kill", "attack", "flee", "cast", "look", "open", "close",
			"lock", "unlock", "north", "south", "east", "west", "up", "down",
			"follow", "group":
			// Execute as a regular command through the command dispatcher
			if mps.executeCmd != nil {
				mps.executeCmd(mob, command, arg)
			}

		case "mpecho":
			// Echo to room without mob name
			if mps.sendRoomMsg != nil {
				mps.sendRoomMsg(mob.InRoom, arg+"\r\n")
			}

		case "mptransfer":
			// Transfer a character to a room
			// Would need more complex implementation
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "transfer", arg)
			}

		case "mpforce":
			// Force a character to do something
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "force", arg)
			}

		case "mpkill":
			// Kill a target
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "kill", arg)
			}

		case "mpgoto":
			// Go to a room or character
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "goto", arg)
			}

		case "mpjunk":
			// Destroy an object
			// Implementation would remove object from inventory

		case "mppurge":
			// Purge a mob or object from room
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "purge", arg)
			}

		case "mpat":
			// Execute command at another location
			if mps.executeCmd != nil {
				mps.executeCmd(mob, "at", arg)
			}

		default:
			// Try to execute as a regular command
			if mps.executeCmd != nil {
				mps.executeCmd(mob, command, arg)
			}
		}
	}
}

// CheckSpeechTrigger checks if any MOB programs should trigger on speech
func (mps *MOBprogSystem) CheckSpeechTrigger(speaker *types.Character, speech string) {
	// Check all mobs in the room
	for _, mob := range speaker.InRoom.People {
		if mob.IsNPC() {
			mps.ExecuteTrigger(TriggerSpeech, mob, speech)
		}
	}
}

// CheckActTrigger checks if any MOB programs should trigger on acts/emotes
func (mps *MOBprogSystem) CheckActTrigger(actor *types.Character, actText string) {
	// Check all mobs in the room
	for _, mob := range actor.InRoom.People {
		if mob != actor && mob.IsNPC() {
			mps.ExecuteTrigger(TriggerAct, mob, actText)
		}
	}
}

// CheckFightTrigger checks fight triggers for a mob
func (mps *MOBprogSystem) CheckFightTrigger(fighter *types.Character) {
	if fighter.IsNPC() {
		mps.ExecuteTrigger(TriggerFight, fighter)
	}
}

// CheckDeathTrigger checks death triggers for a mob
func (mps *MOBprogSystem) CheckDeathTrigger(dying *types.Character) {
	if dying.IsNPC() {
		mps.ExecuteTrigger(TriggerDeath, dying)
	}
}

// CheckEntryTrigger checks entry triggers when someone enters a room
func (mps *MOBprogSystem) CheckEntryTrigger(entering *types.Character) {
	// Check all mobs in the room
	for _, mob := range entering.InRoom.People {
		if mob != entering && mob.IsNPC() {
			mps.ExecuteTrigger(TriggerEntry, mob, entering)
		}
	}
}

// CheckGreetTrigger checks greet triggers when someone enters a room (from a different direction)
func (mps *MOBprogSystem) CheckGreetTrigger(entering *types.Character) {
	// Similar to entry but might have different conditions
	mps.CheckEntryTrigger(entering)
}

// CheckGiveTrigger checks give triggers when an item is given to a mob
func (mps *MOBprogSystem) CheckGiveTrigger(receiver *types.Character, item *types.Object) {
	if receiver.IsNPC() {
		mps.ExecuteTrigger(TriggerGive, receiver, item)
	}
}

// CheckBribeTrigger checks bribe triggers when money is given to a mob
func (mps *MOBprogSystem) CheckBribeTrigger(receiver *types.Character, amount int) {
	if receiver.IsNPC() {
		mps.ExecuteTrigger(TriggerBribe, receiver, amount)
	}
}
