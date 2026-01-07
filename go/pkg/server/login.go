package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"rotmud/pkg/help"
	"rotmud/pkg/persistence"
	"rotmud/pkg/types"
)

// LoginHandler manages the login and character creation process
type LoginHandler struct {
	Persistence *persistence.PlayerPersistence
	Output      func(session *Session, msg string)
	GetRoom     func(vnum int) *types.Room
	Help        *help.System // Help system for race/class info during creation

	// Temp data during creation
	tempName     string
	tempPassword string
	isNewPlayer  bool // True if this is a new character being created
}

// NewLoginHandler creates a new login handler
func NewLoginHandler(persistence *persistence.PlayerPersistence) *LoginHandler {
	return &LoginHandler{
		Persistence: persistence,
	}
}

// HandleInput processes input based on connection state
// Returns true if the player is now in playing state
func (h *LoginHandler) HandleInput(session *Session, input string) bool {
	input = strings.TrimSpace(input)
	desc := session.Descriptor

	switch desc.State {
	case types.ConGetName:
		return h.handleGetName(session, input)

	case types.ConGetOldPassword:
		return h.handleGetOldPassword(session, input)

	case types.ConConfirmNewName:
		return h.handleConfirmNewName(session, input)

	case types.ConGetNewPassword:
		return h.handleGetNewPassword(session, input)

	case types.ConConfirmNewPassword:
		return h.handleConfirmNewPassword(session, input)

	case types.ConGetNewRace:
		return h.handleGetNewRace(session, input)

	case types.ConGetNewSex:
		return h.handleGetNewSex(session, input)

	case types.ConGetNewClass:
		return h.handleGetNewClass(session, input)

	case types.ConGetAlignment:
		return h.handleGetAlignment(session, input)

	case types.ConDefaultChoice:
		return h.handleDefaultChoice(session, input)

	case types.ConGenGroups:
		return h.handleGenGroups(session, input)

	case types.ConPickWeapon:
		return h.handlePickWeapon(session, input)

	case types.ConReadMOTD:
		return h.handleReadMOTD(session, input)

	default:
		h.send(session, "Unknown state, starting over.\r\n")
		desc.State = types.ConGetName
		h.sendNamePrompt(session)
		return false
	}
}

// handleGetName handles the initial name entry
func (h *LoginHandler) handleGetName(session *Session, input string) bool {
	if input == "" {
		h.sendNamePrompt(session)
		return false
	}

	// Validate name
	if !h.isValidName(input) {
		h.send(session, "Illegal name, try another.\r\nBy what name do you wish to be known? ")
		return false
	}

	// Capitalize name
	name := h.capitalizeName(input)
	h.tempName = name

	// Check if player exists
	if h.Persistence.Exists(name) {
		h.send(session, fmt.Sprintf("Welcome back, %s. Password: ", name))
		session.Descriptor.State = types.ConGetOldPassword
	} else {
		h.send(session, fmt.Sprintf("Did I get that right, %s (Y/N)? ", name))
		session.Descriptor.State = types.ConConfirmNewName
	}

	return false
}

// handleGetOldPassword handles password verification for existing players
func (h *LoginHandler) handleGetOldPassword(session *Session, input string) bool {
	// Load the player to check password
	ch, err := h.Persistence.LoadPlayer(h.tempName)
	if err != nil {
		h.send(session, "\r\nError loading character. Please try again.\r\nBy what name do you wish to be known? ")
		session.Descriptor.State = types.ConGetName
		return false
	}

	// Check password
	if ch.PCData == nil || !h.checkPassword(input, ch.PCData.Password) {
		h.send(session, "\r\nWrong password.\r\nBy what name do you wish to be known? ")
		session.Descriptor.State = types.ConGetName
		return false
	}

	// Password correct - load character
	session.Character = ch
	session.Character.Descriptor = session.Descriptor
	session.Descriptor.Character = ch

	h.send(session, "\r\n\r\n")
	h.sendMOTD(session)
	session.Descriptor.State = types.ConReadMOTD

	return false
}

// handleConfirmNewName handles confirmation of a new character name
func (h *LoginHandler) handleConfirmNewName(session *Session, input string) bool {
	input = strings.ToLower(input)

	if input == "" || input[0] != 'y' {
		h.send(session, "Ok, what IS it, then? ")
		session.Descriptor.State = types.ConGetName
		return false
	}

	h.send(session, fmt.Sprintf("\r\nNew character.\r\n\r\nGive me a password for %s: ", h.tempName))
	session.Descriptor.State = types.ConGetNewPassword
	return false
}

// handleGetNewPassword handles new password entry
func (h *LoginHandler) handleGetNewPassword(session *Session, input string) bool {
	if len(input) < 5 {
		h.send(session, "\r\nPassword must be at least five characters long.\r\nPassword: ")
		return false
	}

	if len(input) > 20 {
		h.send(session, "\r\nPassword must be less than 20 characters.\r\nPassword: ")
		return false
	}

	h.tempPassword = h.hashPassword(input)
	h.send(session, "Please retype password: ")
	session.Descriptor.State = types.ConConfirmNewPassword
	return false
}

// handleConfirmNewPassword handles password confirmation
func (h *LoginHandler) handleConfirmNewPassword(session *Session, input string) bool {
	if !h.checkPassword(input, h.tempPassword) {
		h.send(session, "\r\nPasswords don't match.\r\nRetype password: ")
		session.Descriptor.State = types.ConGetNewPassword
		return false
	}

	// Mark as new player for MUD school placement
	h.isNewPlayer = true

	// Create the character
	ch := types.NewCharacter(h.tempName)
	ch.PCData = &types.PCData{
		Password: h.tempPassword,
		Learned:  make(map[string]int),
	}
	ch.Descriptor = session.Descriptor
	session.Descriptor.Character = ch
	session.Character = ch

	h.send(session, "\r\n")
	h.sendRaceMenu(session)
	session.Descriptor.State = types.ConGetNewRace
	return false
}

// handleGetNewRace handles race selection
func (h *LoginHandler) handleGetNewRace(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		h.sendRaceMenu(session)
		return false
	}

	// Check for help command
	if input == "help" {
		h.sendRaceMenu(session)
		return false
	}

	if strings.HasPrefix(input, "help ") {
		raceName := strings.TrimSpace(input[5:])
		h.sendRaceHelp(session, raceName)
		return false
	}

	// Find race by name prefix
	raceIndex := -1
	for i := 0; i < types.MaxRace; i++ {
		race := types.GetRace(i)
		if race != nil && strings.HasPrefix(strings.ToLower(race.Name), input) {
			raceIndex = i
			break
		}
	}

	if raceIndex < 0 {
		h.send(session, "That's not a valid race.\r\n")
		h.sendRaceMenu(session)
		return false
	}

	session.Character.Race = raceIndex

	// Apply race stats and creation point cost
	race := types.GetRace(raceIndex)
	if race != nil {
		for i := 0; i < types.MaxStats; i++ {
			session.Character.PermStats[i] = race.BaseStats[i]
		}
		// Race costs creation points
		if session.Character.PCData != nil {
			session.Character.PCData.Points = race.Points
		}
	}

	h.send(session, "\r\n")
	h.sendSexMenu(session)
	session.Descriptor.State = types.ConGetNewSex
	return false
}

// handleGetNewSex handles sex selection
func (h *LoginHandler) handleGetNewSex(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	switch {
	case strings.HasPrefix("male", input):
		session.Character.Sex = types.SexMale
	case strings.HasPrefix("female", input):
		session.Character.Sex = types.SexFemale
	case strings.HasPrefix("neutral", input):
		session.Character.Sex = types.SexNeutral
	default:
		h.send(session, "That's not a sex.\r\nWhat IS your sex? ")
		return false
	}

	h.send(session, "\r\n")
	h.sendClassMenu(session)
	session.Descriptor.State = types.ConGetNewClass
	return false
}

// handleGetNewClass handles class selection
func (h *LoginHandler) handleGetNewClass(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		h.sendClassMenu(session)
		return false
	}

	// Check for help command
	if input == "help" {
		h.sendClassMenu(session)
		return false
	}

	if strings.HasPrefix(input, "help ") {
		className := strings.TrimSpace(input[5:])
		h.sendClassHelp(session, className)
		return false
	}

	// Find class by name prefix (only tier 1 classes)
	classIndex := -1
	for i := 0; i < types.ClassWizard; i++ { // Only tier 1 classes
		class := types.GetClass(i)
		if class != nil && strings.HasPrefix(strings.ToLower(class.Name), input) {
			classIndex = i
			break
		}
	}

	if classIndex < 0 {
		h.send(session, "That's not a valid class.\r\n")
		h.sendClassMenu(session)
		return false
	}

	session.Character.Class = classIndex

	// Next: alignment selection
	h.send(session, "\r\n")
	h.sendAlignmentMenu(session)
	session.Descriptor.State = types.ConGetAlignment
	return false
}

// handleGetAlignment handles alignment selection
func (h *LoginHandler) handleGetAlignment(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		h.sendAlignmentMenu(session)
		return false
	}

	switch {
	case strings.HasPrefix("good", input) || input == "g":
		session.Character.Alignment = 750
	case strings.HasPrefix("neutral", input) || input == "n":
		session.Character.Alignment = 0
	case strings.HasPrefix("evil", input) || input == "e":
		session.Character.Alignment = -750
	default:
		h.send(session, "That's not a valid alignment.\r\n")
		h.sendAlignmentMenu(session)
		return false
	}

	// Ask about customization
	h.send(session, "\r\n")
	h.sendCustomizePrompt(session)
	session.Descriptor.State = types.ConDefaultChoice
	return false
}

// handleDefaultChoice handles the customize yes/no question
func (h *LoginHandler) handleDefaultChoice(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		h.sendCustomizePrompt(session)
		return false
	}

	ch := session.Character

	switch {
	case strings.HasPrefix("yes", input) || input == "y":
		// Initialize with basic skills only
		h.initializeNewCharacter(session.Character)

		// Enter customization mode - show the list immediately
		h.send(session, "\r\n")
		h.sendGroupList(session)
		h.sendGroupPrompt(session)
		session.Descriptor.State = types.ConGenGroups
		return false

	case strings.HasPrefix("no", input) || input == "n":
		// Initialize with default group (full set of class skills)
		h.initializeNewCharacter(session.Character)
		h.giveDefaultGroup(ch)

		// Proceed to weapon selection
		h.send(session, "\r\n")
		h.sendWeaponMenu(session)
		session.Descriptor.State = types.ConPickWeapon
		return false

	default:
		h.send(session, "Please answer (Y/N)? ")
		return false
	}
}

// handleGenGroups handles skill group customization
func (h *LoginHandler) handleGenGroups(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))
	ch := session.Character

	if input == "" {
		h.sendGroupMenu(session)
		return false
	}

	args := strings.Fields(input)
	cmd := args[0]

	switch {
	case strings.HasPrefix("done", cmd):
		// Finish customization, proceed to weapon
		h.send(session, "\r\n")
		h.sendWeaponMenu(session)
		session.Descriptor.State = types.ConPickWeapon
		return false

	case strings.HasPrefix("list", cmd):
		// List available groups
		h.sendGroupList(session)
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("learned", cmd):
		// Show current skills
		h.sendLearnedSkills(session)
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("add", cmd):
		if len(args) < 2 {
			h.send(session, "Add which group or skill? Type 'list' to see options.\r\n")
			h.sendGroupPrompt(session)
			return false
		}
		name := strings.Join(args[1:], " ")

		// Try adding as group first
		result := h.addGroup(ch, name)
		switch result {
		case 0:
			expPerLevel := h.getExpPerLevel(ch)
			h.send(session, fmt.Sprintf("Group '%s' added. (%d exp/level)\r\n", name, expPerLevel))
			h.sendGroupPrompt(session)
			return false
		case 2:
			h.send(session, "You already have that group.\r\n")
			h.sendGroupPrompt(session)
			return false
		}

		// Group not found (result == 1), try adding as individual skill
		skillResult := h.addIndividualSkill(ch, name)
		switch skillResult {
		case 0:
			expPerLevel := h.getExpPerLevel(ch)
			h.send(session, fmt.Sprintf("Skill '%s' added. (%d exp/level)\r\n", name, expPerLevel))
		case 1:
			h.send(session, "That group or skill is not available. Type 'list' to see options.\r\n")
		case 2:
			h.send(session, "You already have that skill.\r\n")
		}
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("drop", cmd):
		if len(args) < 2 {
			h.send(session, "Drop which group or skill?\r\n")
			h.sendGroupPrompt(session)
			return false
		}
		name := strings.Join(args[1:], " ")

		// Try dropping as group first
		if h.dropGroup(ch, name) {
			expPerLevel := h.getExpPerLevel(ch)
			h.send(session, fmt.Sprintf("Group '%s' dropped. (%d exp/level)\r\n", name, expPerLevel))
			h.sendGroupPrompt(session)
			return false
		}

		// Try dropping as individual skill
		if h.dropIndividualSkill(ch, name) {
			expPerLevel := h.getExpPerLevel(ch)
			h.send(session, fmt.Sprintf("Skill '%s' dropped. (%d exp/level)\r\n", name, expPerLevel))
		} else {
			h.send(session, "You don't have that group or skill.\r\n")
		}
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("info", cmd):
		if len(args) < 2 {
			h.send(session, "Info on which group? Type 'list' to see available groups.\r\n")
			h.sendGroupPrompt(session)
			return false
		}
		groupName := strings.Join(args[1:], " ")
		h.send(session, h.getGroupInfo(ch, groupName))
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("skills", cmd):
		h.sendSkillsTable(session)
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("spells", cmd):
		h.sendSpellsTable(session)
		h.sendGroupPrompt(session)
		return false

	case strings.HasPrefix("help", cmd):
		if len(args) < 2 {
			// No argument - show command help
			h.send(session, `Customization commands:
  list              - show available groups and skills with costs
  info <group>      - show details about a group
  add <name>        - add a skill group or individual skill
  drop <name>       - remove a group or skill (refunds points)
  learned           - show your current skills/spells with proficiency
  skills            - show your learned skills and at what level they unlock
  spells            - show your learned spells and at what level they unlock
  help <skill>      - get help on a skill or spell
  done              - finish and continue to weapon selection

Groups cost multiple points but give several related skills.
Individual skills can be added one at a time for more control.
`)
		} else {
			// Look up help for the skill/spell
			skillName := strings.Join(args[1:], " ")
			h.sendSkillHelp(session, skillName)
		}
		h.sendGroupPrompt(session)
		return false

	default:
		h.send(session, "Commands: list, info, add, drop, learned, help, done\r\n")
		h.sendGroupPrompt(session)
		return false
	}
}

// handlePickWeapon handles starting weapon selection
func (h *LoginHandler) handlePickWeapon(session *Session, input string) bool {
	input = strings.ToLower(strings.TrimSpace(input))

	ch := session.Character

	// Get available weapons (ones the character has learned)
	weapons := h.getAvailableWeapons(ch)

	// Handle 'back' to return to customization
	if input == "back" {
		h.send(session, "\r\n")
		h.sendGroupList(session)
		h.sendGroupPrompt(session)
		session.Descriptor.State = types.ConGenGroups
		return false
	}

	// Handle no weapons learned - allow continuing or going back
	if len(weapons) == 0 {
		if input == "" {
			// Continue without weapon skill
			h.send(session, "\r\nYou have no weapon proficiency.\r\n")
			h.finishCharacterCreation(session)
			return false
		}
		h.sendWeaponMenu(session)
		return false
	}

	if input == "" {
		h.sendWeaponMenu(session)
		return false
	}

	// Find selected weapon
	selectedWeapon := ""
	for _, w := range weapons {
		if strings.HasPrefix(strings.ToLower(w), input) {
			selectedWeapon = w
			break
		}
	}

	if selectedWeapon == "" {
		h.send(session, "That's not a valid weapon choice.\r\n")
		h.sendWeaponMenu(session)
		return false
	}

	// Set the chosen weapon skill to 40%
	if ch.PCData != nil {
		ch.PCData.Learned[selectedWeapon] = 40
	}

	h.send(session, fmt.Sprintf("\r\nYou will start with skill in %s.\r\n", selectedWeapon))
	h.finishCharacterCreation(session)
	return false
}

// finishCharacterCreation completes character creation and shows MOTD
func (h *LoginHandler) finishCharacterCreation(session *Session) {
	ch := session.Character

	// Calculate overspent creation points (affects XP per level)
	remaining := h.getCreationPoints(ch)
	if remaining < 0 {
		ch.PCData.OverspentPoints = -remaining
	}

	// Show final exp per level
	expPerLevel := h.getExpPerLevel(ch)
	h.send(session, fmt.Sprintf("Your experience per level: %d\r\n", expPerLevel))

	// Note: We don't save level 1 characters to avoid abandoned character files
	// The character will be saved when they reach level 2

	h.send(session, "\r\n")
	h.sendMOTD(session)
	session.Descriptor.State = types.ConReadMOTD
}

// handleReadMOTD handles MOTD display and enters the game
func (h *LoginHandler) handleReadMOTD(session *Session, input string) bool {
	session.Descriptor.State = types.ConPlaying
	return true // Ready to play
}

// IsNewPlayer returns true if this login session created a new character
func (h *LoginHandler) IsNewPlayer() bool {
	return h.isNewPlayer
}

// ResetState clears temporary login state for the next connection
func (h *LoginHandler) ResetState() {
	h.tempName = ""
	h.tempPassword = ""
	h.isNewPlayer = false
}

// initializeNewCharacter sets up a new character with starting stats
func (h *LoginHandler) initializeNewCharacter(ch *types.Character) {
	ch.Level = 1
	ch.Exp = 0

	// Get class info
	class := types.GetClass(ch.Class)
	if class == nil {
		class = types.GetClass(types.ClassWarrior)
	}

	// Set HP based on class
	ch.MaxHit = class.HPMax + ch.GetStat(types.StatCon)
	ch.Hit = ch.MaxHit

	// Set Mana based on class
	if class.FreesMana {
		ch.MaxMana = 100 + ch.GetStat(types.StatInt)*2
	} else {
		ch.MaxMana = 50
	}
	ch.Mana = ch.MaxMana

	// Set Movement
	ch.MaxMove = 100 + ch.GetStat(types.StatCon) + ch.GetStat(types.StatDex)
	ch.Move = ch.MaxMove

	// Set armor
	for i := 0; i < 4; i++ {
		ch.Armor[i] = 100
	}

	// Set base hitroll/damroll based on class
	// Warriors get better base combat stats
	switch ch.Class {
	case types.ClassWarrior:
		ch.HitRoll = 5
		ch.DamRoll = 5
	case types.ClassRanger:
		ch.HitRoll = 4
		ch.DamRoll = 4
	case types.ClassThief:
		ch.HitRoll = 3
		ch.DamRoll = 3
	default:
		ch.HitRoll = 2
		ch.DamRoll = 2
	}

	// Initialize conditions
	if ch.PCData != nil {
		ch.PCData.Condition[types.CondHunger] = 48
		ch.PCData.Condition[types.CondThirst] = 48
		ch.PCData.Condition[types.CondFull] = 48
		ch.PCData.Condition[types.CondDrunk] = 0
	}

	// Starting gold
	ch.Gold = 20
	ch.Silver = 0

	// Alignment (neutral by default, may be set by alignment choice)
	if ch.Alignment == 0 {
		ch.Alignment = 0
	}

	// Position
	ch.Position = types.PosStanding

	// Starting training and practice sessions (matches original C code)
	ch.Train = 3
	ch.Practice = 25

	// Set title
	if ch.PCData != nil {
		ch.PCData.Title = fmt.Sprintf(" the %s", types.RaceName(ch.Race))
	}

	// Give starting skills/spells based on class
	// Skills available at level 1 start at 1% proficiency
	h.giveStartingSkills(ch)
}

// giveStartingSkills grants level 1 skills to a new character
func (h *LoginHandler) giveStartingSkills(ch *types.Character) {
	if ch.PCData == nil {
		return
	}

	// Skills that each class gets at level 1 (based on defaults.go)
	// Format: skill name -> list of classes that get it at level 1
	level1Skills := map[string][]int{
		// Combat skills
		"parry":           {types.ClassWarrior},
		"shield block":    {types.ClassWarrior},
		"enhanced damage": {types.ClassWarrior},
		"bash":            {types.ClassWarrior},
		"rescue":          {types.ClassWarrior},
		"dodge":           {types.ClassThief},
		"trip":            {types.ClassThief},
		"backstab":        {types.ClassThief},
		"hide":            {types.ClassThief},
		"peek":            {types.ClassThief},

		// Weapon skills - one primary weapon per class
		"sword":  {types.ClassWarrior},
		"dagger": {types.ClassMage, types.ClassThief},
		"mace":   {types.ClassCleric},

		// Utility skills
		"scrolls": {types.ClassMage, types.ClassCleric},
		"staves":  {types.ClassMage, types.ClassCleric},
		"wands":   {types.ClassMage, types.ClassCleric},
		"haggle":  {types.ClassThief},
	}

	for skillName, classes := range level1Skills {
		for _, classID := range classes {
			if ch.Class == classID {
				ch.PCData.Learned[skillName] = 1 // Start at 1%, will improve with practice
				break
			}
		}
	}

	// Give mages and clerics their first spells
	level1Spells := map[string][]int{
		// Mage spells
		"magic missile": {types.ClassMage},
		"detect magic":  {types.ClassMage},
		"armor":         {types.ClassMage},

		// Cleric spells
		"cure light":   {types.ClassCleric},
		"bless":        {types.ClassCleric},
		"detect evil":  {types.ClassCleric},
		"create water": {types.ClassCleric},
	}

	for spellName, classes := range level1Spells {
		for _, classID := range classes {
			if ch.Class == classID {
				ch.PCData.Learned[spellName] = 1
				break
			}
		}
	}

	// Everyone gets recall at 40% - it's a universal skill
	ch.PCData.Learned["recall"] = 40
}

// Helper functions

func (h *LoginHandler) send(session *Session, msg string) {
	if h.Output != nil {
		h.Output(session, msg)
	} else {
		session.Write(msg)
	}
}

func (h *LoginHandler) sendNamePrompt(session *Session) {
	h.send(session, "By what name do you wish to be known? ")
}

func (h *LoginHandler) sendRaceMenu(session *Session) {
	var sb strings.Builder
	sb.WriteString("The following races are available:\r\n\r\n")

	count := 0
	for i := 0; i < types.MaxRace; i++ {
		race := types.GetRace(i)
		if race != nil {
			// Pad race names to fixed width for alignment
			name := fmt.Sprintf("%-12s", race.Name)
			sb.WriteString("  ")
			sb.WriteString(name)
			count++
			if count%4 == 0 {
				sb.WriteString("\r\n")
			}
		}
	}
	if count%4 != 0 {
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\nWhat is your race (type 'help <race>' for info)? ")
	h.send(session, sb.String())
}

func (h *LoginHandler) sendRaceHelp(session *Session, raceName string) {
	// Try to find help from the help system
	if h.Help != nil {
		entry := h.Help.Find(raceName)
		if entry != nil {
			h.send(session, "\r\n"+entry.Format()+"\r\n")
			h.send(session, "What is your race (type 'help <race>' for info)? ")
			return
		}
	}

	// Fall back to basic race info from the race table
	for i := 0; i < types.MaxRace; i++ {
		race := types.GetRace(i)
		if race != nil && strings.HasPrefix(strings.ToLower(race.Name), strings.ToLower(raceName)) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\r\n%s\r\n", strings.ToUpper(race.Name)))
			sb.WriteString(strings.Repeat("-", len(race.Name)))
			sb.WriteString("\r\n\r\n")
			sb.WriteString(fmt.Sprintf("Size: %s\r\n", sizeName(race.Size)))
			sb.WriteString("\r\nBase Stats:\r\n")
			sb.WriteString(fmt.Sprintf("  STR: %d  INT: %d  WIS: %d  DEX: %d  CON: %d\r\n",
				race.BaseStats[types.StatStr],
				race.BaseStats[types.StatInt],
				race.BaseStats[types.StatWis],
				race.BaseStats[types.StatDex],
				race.BaseStats[types.StatCon]))
			sb.WriteString("\r\n")
			h.send(session, sb.String())
			h.send(session, "What is your race (type 'help <race>' for info)? ")
			return
		}
	}

	h.send(session, "No help available for that race.\r\n")
	h.send(session, "What is your race (type 'help <race>' for info)? ")
}

func sizeName(size types.Size) string {
	switch size {
	case types.SizeTiny:
		return "Tiny"
	case types.SizeSmall:
		return "Small"
	case types.SizeMedium:
		return "Medium"
	case types.SizeLarge:
		return "Large"
	case types.SizeHuge:
		return "Huge"
	case types.SizeGiant:
		return "Giant"
	default:
		return "Medium"
	}
}

func (h *LoginHandler) sendSexMenu(session *Session) {
	h.send(session, "What is your sex (M)ale, (F)emale, or (N)eutral? ")
}

func (h *LoginHandler) sendClassMenu(session *Session) {
	var sb strings.Builder
	sb.WriteString("The following classes are available:\r\n\r\n")

	count := 0
	for i := 0; i < types.ClassWizard; i++ { // Only tier 1 classes
		class := types.GetClass(i)
		if class != nil {
			name := fmt.Sprintf("%-12s", class.Name)
			sb.WriteString("  ")
			sb.WriteString(name)
			count++
			if count%4 == 0 {
				sb.WriteString("\r\n")
			}
		}
	}
	if count%4 != 0 {
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\nWhat is your class (type 'help <class>' for info)? ")
	h.send(session, sb.String())
}

func (h *LoginHandler) sendClassHelp(session *Session, className string) {
	// Try to find help from the help system
	if h.Help != nil {
		entry := h.Help.Find(className)
		if entry != nil {
			h.send(session, "\r\n"+entry.Format()+"\r\n")
			h.send(session, "What is your class (type 'help <class>' for info)? ")
			return
		}
	}

	// Fall back to basic class info from the class table
	for i := 0; i < types.ClassWizard; i++ {
		class := types.GetClass(i)
		if class != nil && strings.HasPrefix(strings.ToLower(class.Name), strings.ToLower(className)) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\r\n%s\r\n", strings.ToUpper(class.Name)))
			sb.WriteString(strings.Repeat("-", len(class.Name)))
			sb.WriteString("\r\n\r\n")
			sb.WriteString(fmt.Sprintf("Prime Attribute: %s\r\n", statName(class.PrimeStat)))
			sb.WriteString(fmt.Sprintf("HP Per Level: %d\r\n", class.HPMax))
			if class.FreesMana {
				sb.WriteString("Uses Mana: Yes\r\n")
			} else {
				sb.WriteString("Uses Mana: No\r\n")
			}
			sb.WriteString("\r\n")
			h.send(session, sb.String())
			h.send(session, "What is your class (type 'help <class>' for info)? ")
			return
		}
	}

	h.send(session, "No help available for that class.\r\n")
	h.send(session, "What is your class (type 'help <class>' for info)? ")
}

func statName(stat int) string {
	switch stat {
	case types.StatStr:
		return "Strength"
	case types.StatInt:
		return "Intelligence"
	case types.StatWis:
		return "Wisdom"
	case types.StatDex:
		return "Dexterity"
	case types.StatCon:
		return "Constitution"
	default:
		return "Unknown"
	}
}

func (h *LoginHandler) sendSkillHelp(session *Session, skillName string) {
	// Try to find help from the help system
	if h.Help != nil {
		entry := h.Help.Find(skillName)
		if entry != nil {
			h.send(session, "\r\n"+entry.Format()+"\r\n")
			return
		}
	}

	// Check if it's a group and show group info
	ch := session.Character
	groups := h.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(skillName)) {
			h.send(session, h.getGroupInfo(ch, skillName))
			return
		}
	}

	// Check if it's a skill and show basic info
	skills := h.getAvailableSkills(ch)
	for _, s := range skills {
		if strings.HasPrefix(strings.ToLower(s.Name), strings.ToLower(skillName)) {
			h.send(session, fmt.Sprintf("\r\n%s (cost: %d points)\r\n\r\n", strings.ToUpper(s.Name), s.Cost))
			return
		}
	}

	h.send(session, fmt.Sprintf("No help available for '%s'.\r\n", skillName))
}

func (h *LoginHandler) sendAlignmentMenu(session *Session) {
	msg := `You may be good, neutral, or evil.

  Good    - Follow the path of righteousness and virtue
  Neutral - Walk the balance between light and dark  
  Evil    - Embrace power at any cost

Which alignment (G/N/E)? `
	h.send(session, msg)
}

func (h *LoginHandler) sendCustomizePrompt(session *Session) {
	msg := `Do you wish to customize this character?
Customization takes time, but allows a wider range of skills and abilities.
Customize (Y/N)? `
	h.send(session, msg)
}

func (h *LoginHandler) sendGroupMenu(session *Session) {
	// Show the full list and prompt
	h.sendGroupList(session)
	h.sendGroupPrompt(session)
}

func (h *LoginHandler) sendGroupPrompt(session *Session) {
	ch := session.Character
	pointsSpent := h.getPointsSpent(ch)
	expPerLevel := h.getExpPerLevel(ch)
	h.send(session, fmt.Sprintf("Creation points: %d/40  Experience per level: %d\r\nChoice (add/drop/list/learned/skills/spells/help/done): ", pointsSpent, expPerLevel))
}

func (h *LoginHandler) sendGroupList(session *Session) {
	ch := session.Character
	groups := h.getAvailableGroups(ch)
	skills := h.getAvailableSkills(ch)

	var sb strings.Builder

	// Show groups in 3 columns
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("%-18s %-5s %-18s %-5s %-18s %-5s\r\n", "group", "cp", "group", "cp", "group", "cp"))
	col := 0
	for _, g := range groups {
		// Skip if already have ALL skills from the group
		hasAllSkills := true
		for _, skill := range g.Skills {
			if ch.PCData == nil || ch.PCData.Learned[skill] == 0 {
				hasAllSkills = false
				break
			}
		}
		if hasAllSkills {
			continue
		}
		sb.WriteString(fmt.Sprintf("%-18s %-5d ", g.Name, g.Cost))
		col++
		if col%3 == 0 {
			sb.WriteString("\r\n")
		}
	}
	if col%3 != 0 {
		sb.WriteString("\r\n")
	}

	// Show individual skills in 3 columns
	sb.WriteString("\r\n")
	sb.WriteString(fmt.Sprintf("%-18s %-5s %-18s %-5s %-18s %-5s\r\n", "skill", "cp", "skill", "cp", "skill", "cp"))
	col = 0
	for _, s := range skills {
		// Skip if already learned
		if ch.PCData != nil && ch.PCData.Learned[s.Name] > 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("%-18s %-5d ", s.Name, s.Cost))
		col++
		if col%3 == 0 {
			sb.WriteString("\r\n")
		}
	}
	if col%3 != 0 {
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\n")
	h.send(session, sb.String())
}

func (h *LoginHandler) sendLearnedSkills(session *Session) {
	ch := session.Character
	if ch.PCData == nil {
		h.send(session, "No skills learned yet.\r\n")
		return
	}

	var sb strings.Builder
	sb.WriteString("\r\nYour current skills:\r\n")
	count := 0
	for skill, level := range ch.PCData.Learned {
		if level > 0 {
			sb.WriteString(fmt.Sprintf("  %-20s %3d%%", skill, level))
			count++
			if count%2 == 0 {
				sb.WriteString("\r\n")
			}
		}
	}
	if count%2 != 0 {
		sb.WriteString("\r\n")
	}
	if count == 0 {
		sb.WriteString("  (none)\r\n")
	}
	sb.WriteString("\r\n")
	h.send(session, sb.String())
}

// sendSkillsTable displays learned skills with the level at which they become available
func (h *LoginHandler) sendSkillsTable(session *Session) {
	ch := session.Character
	if ch.PCData == nil {
		h.send(session, "\r\nNo skills learned yet.\r\n\r\n")
		return
	}

	// Skill data: skill name -> level required per class (mage, cleric, thief, warrior, ranger, druid, vampire)
	// 0 means cannot learn
	skillLevels := map[string][7]int{
		// Combat skills
		"second attack":   {30, 24, 12, 5, 15, 20, 20},
		"third attack":    {0, 0, 0, 25, 30, 0, 0},
		"fourth attack":   {0, 0, 0, 35, 0, 0, 0},
		"fifth attack":    {0, 0, 0, 45, 0, 0, 0},
		"dual wield":      {0, 0, 15, 20, 20, 0, 0},
		"dodge":           {20, 22, 1, 13, 10, 15, 10},
		"parry":           {22, 20, 13, 1, 8, 15, 15},
		"shield block":    {0, 10, 0, 1, 5, 10, 0},
		"enhanced damage": {0, 30, 25, 1, 15, 0, 0},
		"grip":            {0, 0, 0, 15, 0, 0, 0},
		"kick":            {0, 12, 14, 8, 12, 15, 0},
		"bash":            {0, 0, 0, 1, 0, 0, 0},
		"trip":            {0, 0, 1, 15, 10, 0, 0},
		"dirt kicking":    {0, 0, 3, 3, 5, 0, 0},
		"disarm":          {0, 0, 12, 11, 15, 0, 0},
		"gouge":           {0, 0, 10, 0, 0, 0, 0},
		"stun":            {0, 0, 0, 20, 0, 0, 0},
		"backstab":        {0, 0, 1, 0, 0, 0, 10},
		"circle":          {0, 0, 15, 0, 0, 0, 0},
		"berserk":         {0, 0, 0, 18, 0, 0, 0},
		"rescue":          {0, 12, 0, 1, 10, 0, 0},
		"hand to hand":    {25, 10, 15, 6, 15, 20, 15},
		// Thief skills
		"sneak":     {0, 0, 4, 0, 10, 0, 5},
		"hide":      {0, 0, 1, 0, 8, 0, 5},
		"steal":     {0, 0, 5, 0, 0, 0, 0},
		"pick lock": {0, 0, 7, 0, 0, 0, 0},
		"peek":      {0, 0, 1, 0, 0, 0, 0},
		"envenom":   {0, 0, 10, 0, 15, 0, 0},
		"track":     {0, 0, 20, 0, 10, 0, 0},
		// Weapon skills
		"sword":   {20, 15, 6, 1, 1, 10, 5},
		"dagger":  {1, 10, 1, 5, 5, 5, 1},
		"spear":   {0, 15, 0, 5, 1, 0, 0},
		"mace":    {0, 1, 15, 6, 10, 0, 0},
		"axe":     {0, 0, 0, 1, 5, 0, 0},
		"flail":   {0, 1, 0, 10, 0, 0, 0},
		"whip":    {0, 0, 10, 0, 0, 0, 0},
		"polearm": {0, 0, 0, 10, 0, 1, 0},
		// Utility skills
		"meditation":   {6, 6, 0, 0, 0, 10, 0},
		"fast healing": {15, 9, 16, 6, 10, 15, 10},
		"haggle":       {15, 15, 1, 18, 10, 15, 0},
		"lore":         {0, 0, 6, 15, 5, 5, 0}, // Casters use identify spell instead
		"recall":       {1, 1, 1, 1, 1, 1, 1},
		// Magic item skills
		"scrolls": {1, 1, 10, 20, 15, 10, 10},
		"staves":  {1, 1, 15, 25, 20, 15, 15},
		"wands":   {1, 1, 12, 25, 20, 15, 15},
	}

	classIndex := ch.Class
	if classIndex >= 7 {
		classIndex = 0 // Default for tier 2 classes
	}

	// Collect learned skills with their levels
	type skillEntry struct {
		name  string
		level int
	}
	var learned []skillEntry

	for name := range ch.PCData.Learned {
		if levels, ok := skillLevels[name]; ok {
			level := levels[classIndex]
			if level > 0 && level <= 101 {
				learned = append(learned, skillEntry{name, level})
			}
		}
	}

	// Sort by level, then by name
	for i := 0; i < len(learned)-1; i++ {
		for j := i + 1; j < len(learned); j++ {
			if learned[i].level > learned[j].level ||
				(learned[i].level == learned[j].level && learned[i].name > learned[j].name) {
				learned[i], learned[j] = learned[j], learned[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("\r\nYour learned skills:\r\n")
	sb.WriteString(fmt.Sprintf("%-20s %s\r\n", "Skill", "Level"))
	sb.WriteString(strings.Repeat("-", 30) + "\r\n")

	for _, skill := range learned {
		sb.WriteString(fmt.Sprintf("%-20s %3d\r\n", skill.name, skill.level))
	}

	if len(learned) == 0 {
		sb.WriteString("  (no skills learned)\r\n")
	}

	sb.WriteString("\r\n")
	h.send(session, sb.String())
}

// sendSpellsTable displays learned spells with the level at which they become available
func (h *LoginHandler) sendSpellsTable(session *Session) {
	ch := session.Character
	if ch.PCData == nil {
		h.send(session, "\r\nNo spells learned yet.\r\n\r\n")
		return
	}

	// Spell data from magic/spells_data.go
	// Format: spell name -> levels per class [mage, cleric, thief, warrior, ranger, druid, vampire]
	// 102/103 means cannot learn
	spellLevels := map[string][7]int{
		// Damage spells
		"acid blast":      {55, 93, 103, 103, 100, 45, 65},
		"burning hands":   {13, 87, 103, 103, 103, 23, 103},
		"call lightning":  {103, 36, 103, 103, 30, 48, 103},
		"cause critical":  {45, 45, 103, 103, 103, 103, 21},
		"cause light":     {5, 2, 103, 103, 103, 103, 5},
		"cause serious":   {20, 23, 103, 103, 103, 103, 11},
		"chain lightning": {66, 103, 103, 103, 103, 103, 103},
		"chill touch":     {12, 103, 103, 103, 103, 22, 40},
		"colour spray":    {32, 103, 103, 103, 103, 20, 103},
		"demonfire":       {103, 60, 103, 103, 103, 103, 40},
		"dispel evil":     {103, 25, 103, 103, 35, 103, 103},
		"dispel good":     {103, 25, 103, 103, 35, 103, 103},
		"earthquake":      {103, 20, 103, 103, 20, 10, 103},
		"energy drain":    {39, 45, 103, 103, 103, 103, 19},
		"fireball":        {45, 103, 103, 103, 103, 45, 103},
		"flamestrike":     {103, 40, 103, 103, 103, 40, 103},
		"harm":            {103, 50, 103, 103, 103, 103, 45},
		"heat metal":      {103, 32, 103, 103, 103, 27, 103},
		"holy word":       {103, 72, 103, 103, 103, 103, 103},
		"lightning bolt":  {25, 50, 103, 103, 60, 35, 103},
		"magic missile":   {1, 103, 103, 103, 103, 11, 103},
		"ray of truth":    {103, 55, 103, 103, 103, 103, 103},
		"shocking grasp":  {20, 103, 103, 103, 103, 30, 103},
		// Healing spells
		"cure blindness": {103, 11, 103, 103, 25, 16, 103},
		"cure critical":  {103, 23, 103, 103, 103, 32, 103},
		"cure disease":   {103, 23, 103, 103, 30, 103, 103},
		"cure light":     {103, 3, 103, 103, 8, 13, 103},
		"cure poison":    {103, 28, 103, 103, 20, 27, 103},
		"cure serious":   {103, 13, 103, 103, 20, 23, 103},
		"heal":           {103, 42, 103, 103, 103, 50, 103},
		"mass healing":   {103, 75, 103, 103, 103, 103, 103},
		"refresh":        {15, 10, 103, 103, 5, 8, 103},
		"remove curse":   {103, 36, 103, 103, 103, 35, 103},
		// Buff spells
		"armor":           {13, 2, 102, 102, 102, 23, 102},
		"bless":           {102, 13, 102, 102, 102, 16, 102},
		"calm":            {95, 32, 102, 102, 90, 46, 59},
		"detect evil":     {21, 8, 103, 103, 20, 14, 103},
		"detect good":     {21, 8, 103, 103, 20, 14, 103},
		"detect hidden":   {25, 20, 20, 103, 15, 15, 103},
		"detect invis":    {8, 17, 103, 103, 15, 10, 40},
		"detect magic":    {5, 7, 103, 103, 12, 3, 103},
		"detect poison":   {25, 13, 103, 103, 10, 10, 103},
		"faerie fire":     {11, 7, 103, 103, 1, 6, 103},
		"faerie fog":      {28, 34, 103, 103, 21, 25, 103},
		"fireproof":       {23, 16, 103, 103, 103, 12, 103},
		"fireshield":      {70, 103, 103, 103, 103, 68, 103},
		"fly":             {20, 35, 103, 103, 25, 30, 103},
		"frenzy":          {103, 44, 103, 103, 103, 103, 103},
		"giant strength":  {21, 103, 103, 103, 103, 21, 103},
		"haste":           {41, 103, 103, 103, 103, 41, 103},
		"iceshield":       {70, 103, 103, 103, 103, 68, 103},
		"infravision":     {14, 22, 103, 103, 5, 11, 5},
		"invisibility":    {10, 103, 103, 103, 25, 103, 10},
		"know alignment":  {22, 14, 103, 103, 20, 16, 103},
		"mass invis":      {45, 103, 103, 103, 103, 103, 30},
		"pass door":       {44, 52, 103, 103, 103, 45, 48},
		"protection evil": {22, 14, 103, 103, 25, 14, 103},
		"protection good": {22, 14, 103, 103, 25, 14, 103},
		"sanctuary":       {72, 40, 103, 103, 103, 75, 103},
		"shield":          {40, 55, 103, 103, 103, 45, 103},
		"shockshield":     {70, 103, 103, 103, 103, 68, 103},
		"slow":            {43, 103, 103, 103, 103, 51, 103},
		"stone skin":      {50, 70, 103, 103, 103, 55, 103},
		"weaken":          {21, 103, 103, 103, 103, 35, 31},
		// Debuff spells
		"blindness":    {24, 15, 103, 103, 65, 14, 34},
		"charm person": {40, 30, 50, 103, 50, 103, 43},
		"curse":        {36, 36, 103, 103, 103, 43, 25},
		"plague":       {43, 34, 103, 103, 103, 43, 36},
		"poison":       {33, 24, 103, 103, 40, 38, 23},
		"sleep":        {20, 103, 103, 103, 103, 25, 40},
		// Utility spells
		"animate":         {103, 103, 103, 103, 103, 103, 45},
		"cancellation":    {51, 51, 102, 102, 102, 46, 102},
		"conjure":         {85, 102, 102, 102, 102, 95, 102},
		"continual light": {11, 8, 102, 102, 7, 5, 102},
		"control weather": {102, 48, 102, 102, 32, 20, 102},
		"create food":     {19, 10, 102, 102, 6, 7, 102},
		"create rose":     {32, 21, 102, 102, 16, 11, 102},
		"create spring":   {28, 34, 102, 102, 26, 24, 102},
		"create water":    {15, 6, 102, 102, 5, 3, 102},
		"dispel magic":    {32, 32, 103, 103, 45, 40, 103},
		"enchant armor":   {32, 103, 103, 103, 103, 103, 103},
		"enchant weapon":  {34, 103, 103, 103, 103, 103, 103},
		"floating disc":   {8, 20, 103, 103, 10, 14, 103},
		"gate":            {46, 59, 103, 103, 103, 55, 103},
		"identify":        {25, 25, 103, 103, 30, 30, 103},
		"locate object":   {14, 30, 103, 103, 103, 20, 103},
		"nexus":           {75, 85, 103, 103, 103, 103, 103},
		"portal":          {55, 65, 103, 103, 103, 103, 103},
		"recharge":        {14, 103, 103, 103, 103, 103, 103},
		"summon":          {44, 34, 103, 103, 50, 45, 103},
		"teleport":        {23, 37, 103, 103, 35, 25, 103},
		"ventriloquate":   {3, 103, 15, 103, 6, 4, 103},
		"word of recall":  {52, 36, 103, 103, 103, 46, 103},
	}

	classIndex := ch.Class
	if classIndex >= 7 {
		classIndex = 0 // Default for tier 2 classes
	}

	// Collect learned spells with their levels
	type spellEntry struct {
		name  string
		level int
	}
	var learned []spellEntry

	for name := range ch.PCData.Learned {
		if levels, ok := spellLevels[name]; ok {
			level := levels[classIndex]
			if level > 0 && level <= 101 {
				learned = append(learned, spellEntry{name, level})
			}
		}
	}

	// Sort by level, then by name
	for i := 0; i < len(learned)-1; i++ {
		for j := i + 1; j < len(learned); j++ {
			if learned[i].level > learned[j].level ||
				(learned[i].level == learned[j].level && learned[i].name > learned[j].name) {
				learned[i], learned[j] = learned[j], learned[i]
			}
		}
	}

	var sb strings.Builder
	sb.WriteString("\r\nYour learned spells:\r\n")
	sb.WriteString(fmt.Sprintf("%-20s %s\r\n", "Spell", "Level"))
	sb.WriteString(strings.Repeat("-", 30) + "\r\n")

	for _, spell := range learned {
		sb.WriteString(fmt.Sprintf("%-20s %3d\r\n", spell.name, spell.level))
	}

	if len(learned) == 0 {
		sb.WriteString("  (no spells learned)\r\n")
	}

	sb.WriteString("\r\n")
	h.send(session, sb.String())
}

// SkillGroup represents a bundle of skills that can be learned together
type SkillGroup struct {
	Name   string
	Cost   int
	Skills []string
}

// IndividualSkill represents a single skill that can be purchased
type IndividualSkill struct {
	Name  string
	Cost  int
	Level int // Level at which available
}

// getAvailableSkills returns individual skills available for purchase
func (h *LoginHandler) getAvailableSkills(ch *types.Character) []IndividualSkill {
	// Skills available based on class, with diversified costs
	// Format: skill name, cost for each class (mage, cleric, thief, warrior)
	// Costs range from 1-10 based on power and class appropriateness
	skillCosts := map[string][]int{
		// Combat skills - extra attacks are very powerful
		"second attack":   {6, 5, 4, 2},
		"third attack":    {0, 0, 0, 4},
		"fourth attack":   {0, 0, 0, 6},
		"dual wield":      {0, 0, 5, 5},
		"dodge":           {7, 6, 2, 5},
		"parry":           {7, 5, 5, 2},
		"shield block":    {0, 4, 0, 2},
		"enhanced damage": {0, 7, 6, 3},
		"grip":            {0, 0, 0, 3},
		"kick":            {0, 3, 5, 2},
		"bash":            {0, 0, 0, 2},
		"trip":            {0, 0, 3, 5},
		"dirt kicking":    {0, 0, 2, 3},
		"disarm":          {0, 0, 4, 3},
		"gouge":           {0, 0, 3, 0},
		"stun":            {0, 0, 0, 5},
		"backstab":        {0, 0, 6, 0},
		"circle":          {0, 0, 5, 0},
		"berserk":         {0, 0, 0, 6},
		"rescue":          {0, 4, 0, 3},
		"hand to hand":    {6, 4, 5, 3},
		// Thief skills - core skills cheaper, advanced more expensive
		"sneak":     {0, 0, 3, 0},
		"hide":      {0, 0, 2, 0},
		"steal":     {0, 0, 4, 0},
		"pick lock": {0, 0, 3, 0},
		"peek":      {0, 0, 1, 0},
		"envenom":   {0, 0, 5, 0},
		"track":     {0, 0, 4, 0},
		// Weapon skills - primary weapons cheaper for each class
		"sword":   {5, 4, 3, 2},
		"dagger":  {2, 4, 2, 3},
		"spear":   {0, 4, 0, 3},
		"mace":    {0, 2, 5, 3},
		"axe":     {0, 0, 0, 2},
		"flail":   {0, 2, 0, 3},
		"whip":    {0, 0, 4, 0},
		"polearm": {0, 0, 0, 3},
		// Utility skills
		"meditation":   {3, 3, 0, 0},
		"fast healing": {6, 4, 5, 3},
		"haggle":       {4, 4, 2, 6},
		"lore":         {0, 0, 4, 5}, // Casters use identify spell instead
		// recall is not purchasable - everyone gets it at 40% by default
		// Magic item skills - easier for casters
		"scrolls": {1, 1, 4, 6},
		"staves":  {1, 1, 5, 7},
		"wands":   {1, 1, 4, 6},
	}

	result := make([]IndividualSkill, 0)
	for name, costs := range skillCosts {
		cost := 0
		switch ch.Class {
		case types.ClassMage:
			cost = costs[0]
		case types.ClassCleric:
			cost = costs[1]
		case types.ClassThief:
			cost = costs[2]
		case types.ClassWarrior:
			cost = costs[3]
		}
		if cost > 0 { // Only add if available to this class
			result = append(result, IndividualSkill{Name: name, Cost: cost})
		}
	}

	// Sort by name for consistent display
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			if result[i].Name > result[j].Name {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result
}

// getAvailableGroups returns skill groups available to a character's class
func (h *LoginHandler) getAvailableGroups(ch *types.Character) []SkillGroup {
	// Class-specific groups with diversified costs based on power level
	classGroups := map[int][]SkillGroup{
		types.ClassWarrior: {
			{"weaponsmaster", 10, []string{"axe", "dagger", "flail", "mace", "polearm", "spear", "sword", "whip"}},
		},
		types.ClassThief: {
			{"stealth", 6, []string{"sneak", "hide", "backstab", "circle"}},
		},
		types.ClassMage: {
			{"attack", 3, []string{"magic missile", "burning hands", "chill touch", "colour spray"}},
			{"beguiling", 4, []string{"charm person", "sleep"}},
			{"combat", 7, []string{"acid blast", "fireball", "lightning bolt"}},
			{"detection", 5, []string{"detect evil", "detect good", "detect hidden", "detect magic", "detect invis", "identify"}},
			{"enhancement", 6, []string{"giant strength", "haste", "infravision"}},
			{"illusion", 4, []string{"invisibility", "mass invis", "ventriloquate"}},
			{"maladictions", 5, []string{"blindness", "curse", "poison", "plague", "weaken"}},
			{"protective", 6, []string{"armor", "shield", "stone skin"}},
			{"transportation", 5, []string{"fly", "pass door", "teleport", "gate"}},
		},
		types.ClassCleric: {
			{"attack", 3, []string{"cause light", "cause serious", "cause critical", "flamestrike"}},
			{"benedictions", 5, []string{"bless", "calm", "holy word", "remove curse"}},
			{"creation", 2, []string{"create food", "create water", "create spring"}},
			{"curative", 4, []string{"cure blindness", "cure disease", "cure poison"}},
			{"detection", 5, []string{"detect evil", "detect good", "detect hidden", "detect magic", "detect invis", "identify"}},
			{"healing", 7, []string{"cure light", "cure serious", "cure critical", "heal", "mass healing"}},
			{"protective", 6, []string{"armor", "sanctuary", "shield"}},
			{"transportation", 4, []string{"fly", "word of recall", "summon"}},
			{"weather", 3, []string{"call lightning", "control weather"}},
		},
		types.ClassDruid: {
			{"curative", 4, []string{"cure blindness", "cure disease", "cure poison"}},
			{"detection", 4, []string{"detect evil", "detect good", "detect hidden", "detect magic", "identify"}},
			{"healing", 6, []string{"cure light", "cure serious", "cure critical", "heal"}},
			{"protective", 6, []string{"armor", "sanctuary", "shield"}},
			{"weather", 5, []string{"call lightning", "control weather", "earthquake"}},
		},
	}

	result := make([]SkillGroup, 0)

	// Add class-specific groups
	if cg, ok := classGroups[ch.Class]; ok {
		result = append(result, cg...)
	}

	return result
}

// giveDefaultGroup gives the default skill group for a class
func (h *LoginHandler) giveDefaultGroup(ch *types.Character) {
	if ch.PCData == nil {
		return
	}

	// Default groups per class - one primary weapon per class
	defaultSkills := map[int][]string{
		types.ClassMage: {
			"detect magic", "detect invis", "detect hidden",
			"armor", "shield", "stone skin",
			"magic missile", "burning hands", "fireball", "lightning bolt",
			"invisibility", "fly", "pass door",
			"charm person", "sleep",
			"blindness", "curse", "poison",
			"dagger", // Primary weapon: dagger
		},
		types.ClassCleric: {
			"cure light", "cure serious", "cure critical", "heal", "mass healing",
			"cure blindness", "cure disease", "cure poison", "remove curse",
			"bless", "sanctuary", "armor",
			"detect evil", "detect good", "detect magic",
			"cause light", "cause serious", "cause critical", "flamestrike",
			"fly", "word of recall",
			"mace", // Primary weapon: mace
		},
		types.ClassThief: {
			"backstab", "circle", "sneak", "hide", "steal", "peek", "pick lock",
			"dodge", "trip", "disarm", "second attack",
			"envenom", "haggle",
			"dagger", // Primary weapon: dagger
		},
		types.ClassWarrior: {
			"bash", "rescue", "disarm", "kick",
			"parry", "shield block", "dodge", "enhanced damage", "grip",
			"second attack", "third attack", "fourth attack", "dual wield",
			"sword", // Primary weapon: sword
		},
		types.ClassRanger: {
			"dirt kick", "track", "envenom", "kick",
			"parry", "shield block", "enhanced damage",
			"second attack", "third attack",
			"cure light", "cure serious",
			"spear", // Primary weapon: spear
		},
		types.ClassDruid: {
			"shield block", "second attack",
			"cure light", "cure serious", "cure critical", "heal",
			"bless", "armor", "sanctuary",
			"call lightning", "earthquake",
			"polearm", // Primary weapon: polearm
		},
		types.ClassVampire: {
			"backstab", "sneak", "hide", "dodge", "disarm",
			"fast healing", "hand to hand", "shield block",
			"detect invis", "detect hidden",
			"invisibility", "charm person", "sleep",
			"dagger", // Primary weapon: dagger
		},
	}

	skills, ok := defaultSkills[ch.Class]
	if !ok {
		return
	}

	for _, skill := range skills {
		if ch.PCData.Learned[skill] < 1 {
			ch.PCData.Learned[skill] = 1
		}
	}

	// Everyone gets recall at 40% - it's a universal skill
	if ch.PCData.Learned["recall"] < 40 {
		ch.PCData.Learned["recall"] = 40
	}
}

// addGroup adds a skill group to a character
// Returns: 0 = success, 1 = not found, 2 = already have all skills
func (h *LoginHandler) addGroup(ch *types.Character, groupName string) int {
	if ch.PCData == nil {
		return 1
	}

	groups := h.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(groupName)) {
			// Check if already have ALL skills from the group
			hasAllSkills := true
			for _, skill := range g.Skills {
				if ch.PCData.Learned[skill] == 0 {
					hasAllSkills = false
					break
				}
			}
			if hasAllSkills {
				return 2 // Already have all skills
			}

			// Add skills from the group that we don't already have
			for _, skill := range g.Skills {
				if ch.PCData.Learned[skill] == 0 {
					ch.PCData.Learned[skill] = 1
				}
			}

			// Spend points (can go over 40, increases exp per level)
			ch.PCData.Points += g.Cost
			return 0 // Success
		}
	}
	return 1 // Not found
}

// dropGroup removes a skill group from a character
func (h *LoginHandler) dropGroup(ch *types.Character, groupName string) bool {
	if ch.PCData == nil {
		return false
	}

	groups := h.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(groupName)) {
			// Check if we have any skills from this group
			hasGroup := false
			for _, skill := range g.Skills {
				if ch.PCData.Learned[skill] > 0 {
					hasGroup = true
					break
				}
			}
			if !hasGroup {
				return false
			}

			// Remove all skills from the group
			for _, skill := range g.Skills {
				delete(ch.PCData.Learned, skill)
			}

			// Refund points
			if ch.PCData != nil {
				ch.PCData.Points -= g.Cost
				if ch.PCData.Points < 0 {
					ch.PCData.Points = 0
				}
			}
			return true
		}
	}
	return false
}

// addIndividualSkill adds a single skill to a character
// Returns: 0 = success, 1 = not found, 2 = already have
func (h *LoginHandler) addIndividualSkill(ch *types.Character, skillName string) int {
	if ch.PCData == nil {
		return 1
	}

	skills := h.getAvailableSkills(ch)
	for _, s := range skills {
		if strings.HasPrefix(strings.ToLower(s.Name), strings.ToLower(skillName)) {
			// Check if already have this skill
			if ch.PCData.Learned[s.Name] > 0 {
				return 2 // Already have it
			}

			// Add the skill at 1%
			ch.PCData.Learned[s.Name] = 1

			// Spend points (can go over 40, increases exp per level)
			ch.PCData.Points += s.Cost
			return 0 // Success
		}
	}
	return 1 // Not found
}

// dropIndividualSkill removes a single skill from a character
func (h *LoginHandler) dropIndividualSkill(ch *types.Character, skillName string) bool {
	if ch.PCData == nil {
		return false
	}

	skills := h.getAvailableSkills(ch)
	for _, s := range skills {
		if strings.HasPrefix(strings.ToLower(s.Name), strings.ToLower(skillName)) {
			// Check if we have this skill
			if ch.PCData.Learned[s.Name] == 0 {
				return false
			}

			// Remove the skill
			delete(ch.PCData.Learned, s.Name)

			// Refund points
			ch.PCData.Points -= s.Cost
			if ch.PCData.Points < 0 {
				ch.PCData.Points = 0
			}
			return true
		}
	}
	return false
}

// BaseCreationPoints is the starting creation points for all classes
const BaseCreationPoints = 40

// BaseExpPerLevel is the base experience needed per level
const BaseExpPerLevel = 1000

// ExpPerPoint is the additional exp per level for each point spent over base
const ExpPerPoint = 100

// getCreationPoints returns remaining creation points for a character
// Can be negative if overspent (which increases XP required per level)
func (h *LoginHandler) getCreationPoints(ch *types.Character) int {
	if ch.PCData == nil {
		return 0
	}
	return BaseCreationPoints - ch.PCData.Points
}

// getPointsSpent returns total creation points spent (including race cost)
func (h *LoginHandler) getPointsSpent(ch *types.Character) int {
	if ch.PCData == nil {
		return 0
	}
	return ch.PCData.Points
}

// getExpPerLevel returns the experience required per level based on points spent
// Base is 1000, plus 100 for each point spent over 40
func (h *LoginHandler) getExpPerLevel(ch *types.Character) int {
	if ch.PCData == nil {
		return BaseExpPerLevel
	}
	// Points spent over 40 add 100 exp per level each
	overSpent := ch.PCData.Points - BaseCreationPoints
	if overSpent > 0 {
		return BaseExpPerLevel + (overSpent * ExpPerPoint)
	}
	return BaseExpPerLevel
}

// getGroupInfo returns detailed info about a skill group
func (h *LoginHandler) getGroupInfo(ch *types.Character, groupName string) string {
	groups := h.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(groupName)) {
			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("\r\nGroup: %s (cost: %d points)\r\n", g.Name, g.Cost))
			sb.WriteString("Skills/Spells included:\r\n")
			for _, skill := range g.Skills {
				status := " "
				if ch.PCData != nil && ch.PCData.Learned[skill] > 0 {
					status = "*"
				}
				sb.WriteString(fmt.Sprintf("  %s %s\r\n", status, skill))
			}
			sb.WriteString("(* = already learned)\r\n")
			return sb.String()
		}
	}
	return "That group doesn't exist.\r\n"
}

func (h *LoginHandler) sendWeaponMenu(session *Session) {
	ch := session.Character
	weapons := h.getAvailableWeapons(ch)

	if len(weapons) == 0 {
		h.send(session, "You haven't learned any weapon skills!\r\nYou should go back and add at least one weapon skill.\r\n")
		h.send(session, "Press ENTER to continue anyway, or type 'back' to return to customization: ")
		return
	}

	var sb strings.Builder
	sb.WriteString("Please pick a weapon from the following choices:\r\n  ")
	for i, w := range weapons {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(w)
	}
	sb.WriteString("\r\nYour choice? ")
	h.send(session, sb.String())
}

// getAvailableWeapons returns the list of weapon skills the character has learned
func (h *LoginHandler) getAvailableWeapons(ch *types.Character) []string {
	if ch.PCData == nil {
		return []string{}
	}

	// All possible weapon skills
	allWeapons := []string{"sword", "axe", "dagger", "mace", "flail", "spear", "polearm", "whip"}

	// Return only the weapons the character has learned
	var learned []string
	for _, weapon := range allWeapons {
		if ch.PCData.Learned[weapon] > 0 {
			learned = append(learned, weapon)
		}
	}

	return learned
}

func (h *LoginHandler) sendMOTD(session *Session) {
	motd := `
* You are responsible for knowing the rules (type 'rules'). Ignorance of 
  the rules is no excuse.
* Important commands: help, commands, news, notes
* Use the delete command to erase unwanted characters.
* Use COLOR to turn on the color codes.
* Use SECOND to wield a second weapon.
* Use HELP FLAGS to get a list of the affect flags.

[Press ENTER to continue]
`
	h.send(session, motd)
}

// SendGreeting sends the login greeting screen
func (h *LoginHandler) SendGreeting(session *Session) {
	greeting := `

        ____  _                          __   _____ _
       / __ \(_)   _____  __________    / /_ / __  (_)___ ___  ___
      / /_/ / / | / / _ \/ ___/ ___/   / __ \/ /_/ / / __ ` + "`" + `__ \/ _ \
     / _, _/ /| |/ /  __/ /  (__  )   / /_/ / /_  / / / / / / /  __/
    /_/ |_/_/ |___/\___/_/  /____/    \____/_/ /_/_/_/ /_/ /_/\___/

                     R I V E R S   O F   T I M E

===============================================================================

                Original DikuMUD by Hans Staerfeldt, Katja Nyboe,
                Tom Madsen, Michael Seifert, and Sebastian Hammer
                Based on MERC 2.1 code by Hatchet, Furey, and Kahn
                ROM 2.4 copyright (c) 1993-1995 Russ Taylor
                ROT 1.4 copyright (c) 1996-1997 Russ Walsh
                Go port by the ROT team

===============================================================================

By what name do you wish to be known? `
	h.send(session, greeting)
}

// isValidName checks if a name is valid
func (h *LoginHandler) isValidName(name string) bool {
	if len(name) < 2 || len(name) > 12 {
		return false
	}

	// Must start with a letter
	if !unicode.IsLetter(rune(name[0])) {
		return false
	}

	// Must be all letters
	for _, r := range name {
		if !unicode.IsLetter(r) {
			return false
		}
	}

	// Check for reserved names
	reserved := []string{"all", "auto", "immortal", "self", "someone", "something", "god", "admin"}
	lower := strings.ToLower(name)
	for _, r := range reserved {
		if lower == r {
			return false
		}
	}

	return true
}

// capitalizeName properly capitalizes a name
func (h *LoginHandler) capitalizeName(name string) string {
	if name == "" {
		return name
	}
	lower := strings.ToLower(name)
	return strings.ToUpper(string(lower[0])) + lower[1:]
}

// hashPassword creates a SHA256 hash of a password
func (h *LoginHandler) hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// checkPassword verifies a password against a hash
func (h *LoginHandler) checkPassword(password, hash string) bool {
	return h.hashPassword(password) == hash
}
