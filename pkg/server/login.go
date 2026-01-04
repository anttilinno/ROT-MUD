package server

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"unicode"

	"rotmud/pkg/persistence"
	"rotmud/pkg/types"
)

// LoginHandler manages the login and character creation process
type LoginHandler struct {
	Persistence *persistence.PlayerPersistence
	Output      func(session *Session, msg string)
	GetRoom     func(vnum int) *types.Room

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

	h.send(session, fmt.Sprintf("\r\nNew character.\r\nGive me a password for %s: ", h.tempName))
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
	h.send(session, "\r\nPlease retype password: ")
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

	if input == "" || input == "help" {
		h.sendRaceMenu(session)
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

	// Apply race stats
	race := types.GetRace(raceIndex)
	if race != nil {
		for i := 0; i < types.MaxStats; i++ {
			session.Character.PermStats[i] = race.BaseStats[i]
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

	if input == "" || input == "help" {
		h.sendClassMenu(session)
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
			points := h.getCreationPoints(ch)
			h.send(session, fmt.Sprintf("Group '%s' added. (%d points remaining)\r\n", name, points))
			h.sendGroupPrompt(session)
			return false
		case 2:
			h.send(session, "You already have that group.\r\n")
			h.sendGroupPrompt(session)
			return false
		case 3:
			h.send(session, "You don't have enough creation points for that group.\r\n")
			h.sendGroupPrompt(session)
			return false
		}

		// Group not found (result == 1), try adding as individual skill
		skillResult := h.addIndividualSkill(ch, name)
		switch skillResult {
		case 0:
			points := h.getCreationPoints(ch)
			h.send(session, fmt.Sprintf("Skill '%s' added. (%d points remaining)\r\n", name, points))
		case 1:
			h.send(session, "That group or skill is not available. Type 'list' to see options.\r\n")
		case 2:
			h.send(session, "You already have that skill.\r\n")
		case 3:
			h.send(session, "You don't have enough creation points.\r\n")
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
			points := h.getCreationPoints(ch)
			h.send(session, fmt.Sprintf("Group '%s' dropped. (%d points remaining)\r\n", name, points))
			h.sendGroupPrompt(session)
			return false
		}

		// Try dropping as individual skill
		if h.dropIndividualSkill(ch, name) {
			points := h.getCreationPoints(ch)
			h.send(session, fmt.Sprintf("Skill '%s' dropped. (%d points remaining)\r\n", name, points))
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

	case strings.HasPrefix("help", cmd):
		h.send(session, `Customization commands:
  list              - show available groups and skills with costs
  info <group>      - show details about a group
  add <name>        - add a skill group or individual skill
  drop <name>       - remove a group or skill (refunds points)
  learned           - show your current skills
  done              - finish and continue to weapon selection

Groups cost multiple points but give several related skills.
Individual skills can be added one at a time for more control.
`)
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

	if input == "" {
		h.sendWeaponMenu(session)
		return false
	}

	ch := session.Character

	// Get available weapons for this class
	weapons := h.getAvailableWeapons(ch)

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

	// Note: We don't save level 1 characters to avoid abandoned character files
	// The character will be saved when they reach level 2

	h.send(session, "\r\n")
	h.sendMOTD(session)
	session.Descriptor.State = types.ConReadMOTD
	return false
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

		// Weapon skills
		"sword":  {types.ClassWarrior},
		"axe":    {types.ClassWarrior},
		"dagger": {types.ClassMage, types.ClassThief},
		"mace":   {types.ClassCleric},
		"flail":  {types.ClassCleric},

		// Utility skills
		"recall":  {types.ClassMage, types.ClassCleric, types.ClassThief, types.ClassWarrior},
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
	sb.WriteString("The following races are available:\r\n")
	sb.WriteString("  ")

	count := 0
	for i := 0; i < types.MaxRace; i++ {
		race := types.GetRace(i)
		if race != nil {
			if count > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(race.Name)
			count++
			if count%6 == 0 {
				sb.WriteString("\r\n  ")
			}
		}
	}
	sb.WriteString("\r\n\r\nWhat is your race? ")
	h.send(session, sb.String())
}

func (h *LoginHandler) sendSexMenu(session *Session) {
	h.send(session, "What is your sex (M/F/N)? ")
}

func (h *LoginHandler) sendClassMenu(session *Session) {
	var sb strings.Builder
	sb.WriteString("The following classes are available:\r\n")
	sb.WriteString("  ")

	count := 0
	for i := 0; i < types.ClassWizard; i++ { // Only tier 1 classes
		class := types.GetClass(i)
		if class != nil {
			if count > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(class.Name)
			count++
		}
	}
	sb.WriteString("\r\n\r\nWhat is your class? ")
	h.send(session, sb.String())
}

func (h *LoginHandler) sendAlignmentMenu(session *Session) {
	msg := `You may be good, neutral, or evil.

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
	points := h.getCreationPoints(ch)
	h.send(session, fmt.Sprintf("Creation points: %d\r\nChoice (add/drop/list/learned/help/done): ", points))
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
		// Skip if already have any skill from the group
		hasGroup := false
		for _, skill := range g.Skills {
			if ch.PCData != nil && ch.PCData.Learned[skill] > 0 {
				hasGroup = true
				break
			}
		}
		if hasGroup {
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
	// Skills available based on class, with costs
	// Format: skill name, cost for each class (mage, cleric, thief, warrior)
	skillCosts := map[string][]int{
		// Combat skills
		"second attack":   {5, 5, 5, 3},
		"third attack":    {0, 0, 0, 4},
		"fourth attack":   {0, 0, 0, 5},
		"dual wield":      {0, 0, 4, 4},
		"dodge":           {8, 6, 4, 6},
		"parry":           {8, 6, 6, 4},
		"shield block":    {0, 5, 0, 3},
		"enhanced damage": {0, 6, 5, 3},
		"grip":            {0, 0, 0, 4},
		"kick":            {0, 4, 6, 4},
		"bash":            {0, 0, 0, 3},
		"trip":            {0, 0, 4, 6},
		"dirt kicking":    {0, 0, 4, 4},
		"disarm":          {0, 0, 5, 4},
		"gouge":           {0, 0, 4, 0},
		"stun":            {0, 0, 0, 5},
		"backstab":        {0, 0, 5, 0},
		"circle":          {0, 0, 4, 0},
		"berserk":         {0, 0, 0, 5},
		"rescue":          {0, 5, 0, 4},
		"hand to hand":    {8, 5, 6, 4},
		// Thief skills
		"sneak":     {0, 0, 4, 0},
		"hide":      {0, 0, 4, 0},
		"steal":     {0, 0, 4, 0},
		"pick lock": {0, 0, 4, 0},
		"peek":      {0, 0, 4, 0},
		"envenom":   {0, 0, 4, 0},
		"track":     {0, 0, 5, 0},
		// Weapon skills
		"sword":   {6, 5, 4, 3},
		"dagger":  {4, 5, 3, 4},
		"spear":   {0, 5, 0, 4},
		"mace":    {0, 3, 6, 4},
		"axe":     {0, 0, 0, 3},
		"flail":   {0, 3, 0, 4},
		"whip":    {0, 0, 5, 0},
		"polearm": {0, 0, 0, 4},
		// Utility skills
		"meditation":   {5, 5, 0, 0},
		"fast healing": {8, 5, 6, 4},
		"haggle":       {5, 5, 3, 8},
		"lore":         {5, 4, 6, 8},
		"recall":       {2, 2, 2, 2},
		// Magic item skills
		"scrolls": {2, 2, 5, 8},
		"staves":  {2, 2, 6, 8},
		"wands":   {2, 2, 5, 8},
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
	// Groups available to all classes
	commonGroups := []SkillGroup{
		{"attack", 5, []string{"magic missile", "cause light", "burning hands"}},
		{"benedictions", 4, []string{"bless", "calm", "holy word"}},
		{"combat", 6, []string{"acid blast", "fireball", "lightning bolt"}},
		{"creation", 4, []string{"create food", "create water", "create spring"}},
		{"curative", 4, []string{"cure blindness", "cure disease", "cure poison"}},
		{"detection", 4, []string{"detect evil", "detect good", "detect hidden", "detect magic", "detect invis"}},
		{"enhancement", 5, []string{"giant strength", "haste", "infravision"}},
		{"healing", 6, []string{"cure light", "cure serious", "cure critical", "heal", "mass healing"}},
		{"maladictions", 5, []string{"blindness", "curse", "poison", "plague", "weaken"}},
		{"protective", 4, []string{"armor", "sanctuary", "shield", "stone skin"}},
		{"transportation", 4, []string{"fly", "pass door", "word of recall"}},
		{"weather", 4, []string{"call lightning", "earthquake"}},
	}

	// Class-specific groups
	classGroups := map[int][]SkillGroup{
		types.ClassWarrior: {
			{"weaponsmaster", 8, []string{"axe", "dagger", "flail", "mace", "polearm", "spear", "sword", "whip"}},
		},
		types.ClassThief: {
			{"stealth", 4, []string{"sneak", "hide", "backstab", "circle"}},
		},
		types.ClassMage: {
			{"beguiling", 5, []string{"charm person", "sleep"}},
			{"illusion", 4, []string{"invisibility", "mass invis", "ventriloquate"}},
		},
		types.ClassCleric: {
			{"attack", 4, []string{"cause light", "cause serious", "cause critical", "flamestrike"}},
		},
	}

	result := make([]SkillGroup, 0)
	result = append(result, commonGroups...)
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

	// Default groups per class - these are the comprehensive defaults
	defaultSkills := map[int][]string{
		types.ClassMage: {
			"detect magic", "detect invis", "detect hidden",
			"armor", "shield", "stone skin",
			"magic missile", "burning hands", "fireball", "lightning bolt",
			"invisibility", "fly", "pass door",
			"charm person", "sleep",
			"blindness", "curse", "poison",
		},
		types.ClassCleric: {
			"cure light", "cure serious", "cure critical", "heal", "mass healing",
			"cure blindness", "cure disease", "cure poison", "remove curse",
			"bless", "sanctuary", "armor",
			"detect evil", "detect good", "detect magic",
			"cause light", "cause serious", "cause critical", "flamestrike",
			"fly", "word of recall",
		},
		types.ClassThief: {
			"backstab", "circle", "sneak", "hide", "steal", "peek", "pick lock",
			"dodge", "trip", "disarm", "second attack",
			"dagger", "sword", "mace",
			"envenom", "haggle",
		},
		types.ClassWarrior: {
			"bash", "rescue", "disarm", "kick",
			"parry", "shield block", "dodge", "enhanced damage", "grip",
			"second attack", "third attack", "fourth attack", "dual wield",
			"sword", "axe", "dagger", "mace", "flail", "polearm", "spear", "whip",
		},
		types.ClassRanger: {
			"dirt kick", "track", "envenom", "kick",
			"parry", "shield block", "enhanced damage",
			"second attack", "third attack",
			"spear", "sword", "axe",
			"cure light", "cure serious",
		},
		types.ClassDruid: {
			"shield block", "second attack",
			"cure light", "cure serious", "cure critical", "heal",
			"bless", "armor", "sanctuary",
			"call lightning", "earthquake",
			"polearm", "dagger",
		},
		types.ClassVampire: {
			"backstab", "sneak", "hide", "dodge", "disarm",
			"fast healing", "hand to hand", "shield block",
			"detect invis", "detect hidden",
			"invisibility", "charm person", "sleep",
			"dagger", "sword",
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
}

// addGroup adds a skill group to a character
// Returns: 0 = success, 1 = not found, 2 = already have, 3 = not enough points
func (h *LoginHandler) addGroup(ch *types.Character, groupName string) int {
	if ch.PCData == nil {
		return 1
	}

	groups := h.getAvailableGroups(ch)
	for _, g := range groups {
		if strings.HasPrefix(strings.ToLower(g.Name), strings.ToLower(groupName)) {
			// Check if already have this group (any skill from it)
			hasGroup := false
			for _, skill := range g.Skills {
				if ch.PCData.Learned[skill] > 0 {
					hasGroup = true
					break
				}
			}
			if hasGroup {
				return 2 // Already have it
			}

			// Check if enough points
			remaining := h.getCreationPoints(ch)
			if remaining < g.Cost {
				return 3 // Not enough points
			}

			// Add all skills from the group at 1%
			for _, skill := range g.Skills {
				ch.PCData.Learned[skill] = 1
			}

			// Spend points
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
// Returns: 0 = success, 1 = not found, 2 = already have, 3 = not enough points
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

			// Check if enough points
			remaining := h.getCreationPoints(ch)
			if remaining < s.Cost {
				return 3 // Not enough points
			}

			// Add the skill at 1%
			ch.PCData.Learned[s.Name] = 1

			// Spend points
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

// getCreationPoints returns remaining creation points for a character
func (h *LoginHandler) getCreationPoints(ch *types.Character) int {
	if ch.PCData == nil {
		return 0
	}

	// Starting points depend on class
	basePoints := 40

	// Class adjustments
	switch ch.Class {
	case types.ClassMage, types.ClassCleric:
		basePoints = 50 // Spellcasters get more for spell groups
	case types.ClassWarrior:
		basePoints = 45 // Warriors need weapon skills
	}

	return basePoints - ch.PCData.Points
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

// getAvailableWeapons returns the list of weapon skills available to a character
func (h *LoginHandler) getAvailableWeapons(ch *types.Character) []string {
	if ch.PCData == nil {
		return []string{"sword"}
	}

	// Weapons available depend on class
	// Each class gets certain weapons in their basics/defaults
	classWeapons := map[int][]string{
		types.ClassMage:    {"dagger"},
		types.ClassCleric:  {"mace", "flail"},
		types.ClassThief:   {"dagger", "sword"},
		types.ClassWarrior: {"sword", "axe", "dagger", "mace", "flail", "spear", "polearm", "whip"},
		types.ClassRanger:  {"spear", "sword", "axe"},
		types.ClassDruid:   {"polearm", "dagger"},
		types.ClassVampire: {"dagger", "sword"},
	}

	if weapons, ok := classWeapons[ch.Class]; ok {
		return weapons
	}

	// Default fallback
	return []string{"sword", "dagger"}
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
