package game

import (
	"fmt"
	"strconv"
	"strings"

	"rotmud/pkg/combat"
	"rotmud/pkg/types"
)

// Configuration commands: wimpy, prompt, title, description, brief, compact, color

func (d *CommandDispatcher) cmdWimpy(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, fmt.Sprintf("Your current wimpy is %d hit points.\r\n", ch.Wimpy))
		return
	}

	wimpy, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "Wimpy must be a number.\r\n")
		return
	}

	if wimpy < 0 {
		d.send(ch, "Your wimpy must be positive.\r\n")
		return
	}

	if wimpy > ch.MaxHit/2 {
		d.send(ch, fmt.Sprintf("Your wimpy must be less than %d.\r\n", ch.MaxHit/2))
		return
	}

	ch.Wimpy = wimpy
	d.send(ch, fmt.Sprintf("Wimpy set to %d hit points.\r\n", wimpy))
}

func (d *CommandDispatcher) cmdTitle(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	if args == "" {
		if ch.PCData.Title != "" {
			d.send(ch, fmt.Sprintf("Your current title is: %s\r\n", ch.PCData.Title))
		} else {
			d.send(ch, "You have no title.\r\n")
		}
		return
	}

	// Check title length
	if len(args) > 45 {
		d.send(ch, "Title must be 45 characters or less.\r\n")
		return
	}

	// Check for profanity
	if containsProfanity(args) {
		d.send(ch, "That title is not allowed.\r\n")
		return
	}

	// Add space at start if doesn't start with punctuation
	if len(args) > 0 && args[0] != ',' && args[0] != '.' && args[0] != '!' && args[0] != '?' {
		args = " " + args
	}

	ch.PCData.Title = args
	d.send(ch, fmt.Sprintf("Title set to:%s\r\n", args))
}

func (d *CommandDispatcher) cmdDescription(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	if args == "" {
		if ch.Desc != "" {
			d.send(ch, "Your current description:\r\n")
			d.send(ch, ch.Desc+"\r\n")
		} else {
			d.send(ch, "You have no description.\r\n")
		}
		d.send(ch, "Use 'description <text>' to set a new description.\r\n")
		d.send(ch, "Use 'description clear' to clear your description.\r\n")
		return
	}

	if strings.ToLower(args) == "clear" {
		ch.Desc = ""
		d.send(ch, "Description cleared.\r\n")
		return
	}

	// Check for profanity
	if containsProfanity(args) {
		d.send(ch, "That description is not allowed.\r\n")
		return
	}

	ch.Desc = args
	d.send(ch, "Description set.\r\n")
}

func (d *CommandDispatcher) cmdPrompt(ch *types.Character, args string) {
	if ch.IsNPC() {
		return
	}

	if args == "" {
		if ch.Prompt != "" {
			d.send(ch, fmt.Sprintf("Your current prompt is: %s\r\n", ch.Prompt))
		} else {
			d.send(ch, "You have no custom prompt set.\r\n")
		}
		d.send(ch, "\r\nPrompt variables:\r\n")
		d.send(ch, "  %h - current hit points\r\n")
		d.send(ch, "  %H - maximum hit points\r\n")
		d.send(ch, "  %m - current mana\r\n")
		d.send(ch, "  %M - maximum mana\r\n")
		d.send(ch, "  %v - current movement\r\n")
		d.send(ch, "  %V - maximum movement\r\n")
		d.send(ch, "  %x - experience to level\r\n")
		d.send(ch, "  %g - gold on hand\r\n")
		d.send(ch, "  %a - alignment\r\n")
		d.send(ch, "  %r - room name\r\n")
		d.send(ch, "  %e - exits\r\n")
		d.send(ch, "  %c - carriage return\r\n")
		d.send(ch, "\r\nUse 'prompt default' for standard prompt.\r\n")
		d.send(ch, "Use 'prompt clear' to use no prompt.\r\n")
		return
	}

	if strings.ToLower(args) == "clear" {
		ch.Prompt = ""
		d.send(ch, "Prompt cleared.\r\n")
		return
	}

	if strings.ToLower(args) == "default" {
		ch.Prompt = "<%hhp %mm %vmv> "
		d.send(ch, "Prompt set to default.\r\n")
		return
	}

	ch.Prompt = args
	d.send(ch, fmt.Sprintf("Prompt set to: %s\r\n", args))
}

// FormatPrompt formats a character's prompt with variable substitution
func FormatPrompt(ch *types.Character) string {
	if ch.Prompt == "" {
		return ""
	}

	prompt := ch.Prompt

	// Replace variables
	prompt = strings.ReplaceAll(prompt, "%h", fmt.Sprintf("%d", ch.Hit))
	prompt = strings.ReplaceAll(prompt, "%H", fmt.Sprintf("%d", ch.MaxHit))
	prompt = strings.ReplaceAll(prompt, "%m", fmt.Sprintf("%d", ch.Mana))
	prompt = strings.ReplaceAll(prompt, "%M", fmt.Sprintf("%d", ch.MaxMana))
	prompt = strings.ReplaceAll(prompt, "%v", fmt.Sprintf("%d", ch.Move))
	prompt = strings.ReplaceAll(prompt, "%V", fmt.Sprintf("%d", ch.MaxMove))
	prompt = strings.ReplaceAll(prompt, "%g", fmt.Sprintf("%d", ch.Gold))
	prompt = strings.ReplaceAll(prompt, "%a", fmt.Sprintf("%d", ch.Alignment))
	prompt = strings.ReplaceAll(prompt, "%c", "\r\n")

	// Experience to level (with creation point overspend penalty)
	overspent := 0
	if ch.PCData != nil {
		overspent = ch.PCData.OverspentPoints
	}
	expToLevel := combat.ExpToLevelWithPenalty(ch.Level+1, overspent) - ch.Exp
	prompt = strings.ReplaceAll(prompt, "%x", fmt.Sprintf("%d", expToLevel))

	// Room name
	roomName := "Nowhere"
	if ch.InRoom != nil {
		roomName = ch.InRoom.Name
	}
	prompt = strings.ReplaceAll(prompt, "%r", roomName)

	// Exits
	exits := ""
	if ch.InRoom != nil {
		for dir := types.Direction(0); dir < types.DirMax; dir++ {
			if ch.InRoom.GetExit(dir) != nil {
				exits += string(dir.String()[0])
			}
		}
	}
	if exits == "" {
		exits = "none"
	}
	prompt = strings.ReplaceAll(prompt, "%e", exits)

	return prompt
}

// conditionString returns a text description of a character's health condition
func conditionString(ch *types.Character) string {
	percent := 100
	if ch.MaxHit > 0 {
		percent = ch.Hit * 100 / ch.MaxHit
	}

	switch {
	case percent >= 100:
		return "is in excellent condition"
	case percent >= 90:
		return "has a few scratches"
	case percent >= 75:
		return "has some small wounds and bruises"
	case percent >= 50:
		return "has quite a few wounds"
	case percent >= 30:
		return "has some big nasty wounds and scratches"
	case percent >= 15:
		return "looks pretty hurt"
	case percent >= 0:
		return "is in awful condition"
	default:
		return "is bleeding to death"
	}
}

// healthBar returns a visual health bar like [********  ] (82%)
func healthBar(ch *types.Character) string {
	percent := 100
	if ch.MaxHit > 0 {
		percent = ch.Hit * 100 / ch.MaxHit
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// 10 character bar
	filled := percent / 10
	empty := 10 - filled

	bar := strings.Repeat("*", filled) + strings.Repeat(" ", empty)
	return fmt.Sprintf("[%s] (%d%%)", bar, percent)
}

// FormatCombatPrompt formats a combat status showing opponent health
// Format: The horned lizard: [********  ] (82%)
func FormatCombatPrompt(ch *types.Character) string {
	if ch.Fighting == nil {
		return ""
	}

	victim := ch.Fighting

	// Get victim's name for display
	victimName := victim.Name
	if victim.ShortDesc != "" {
		victimName = victim.ShortDesc
	}

	// Build the combat status line with health bar
	return fmt.Sprintf("%s: %s\r\n", victimName, healthBar(victim))
}

func (d *CommandDispatcher) cmdBrief(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommBrief) {
		ch.Comm.Remove(types.CommBrief)
		d.send(ch, "Full descriptions activated.\r\n")
	} else {
		ch.Comm.Set(types.CommBrief)
		d.send(ch, "Short descriptions activated.\r\n")
	}
}

func (d *CommandDispatcher) cmdCompact(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommCompact) {
		ch.Comm.Remove(types.CommCompact)
		d.send(ch, "Compact mode removed.\r\n")
	} else {
		ch.Comm.Set(types.CommCompact)
		d.send(ch, "Compact mode set.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutolist(ch *types.Character, args string) {
	d.send(ch, "Automatic actions:\r\n")
	d.send(ch, fmt.Sprintf("  autoexit   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoExit))))
	d.send(ch, fmt.Sprintf("  autogold   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoGold))))
	d.send(ch, fmt.Sprintf("  autoloot   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoLoot))))
	d.send(ch, fmt.Sprintf("  autosac    %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoSac))))
	d.send(ch, fmt.Sprintf("  autoassist %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoAssist))))
	d.send(ch, fmt.Sprintf("  autosplit  %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoSplit))))
	d.send(ch, fmt.Sprintf("  autostore  %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoStore))))
	d.send(ch, fmt.Sprintf("  autopeek   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrAutoPeek))))
	d.send(ch, "\r\nProtection settings:\r\n")
	d.send(ch, fmt.Sprintf("  nofollow   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrNoFollow))))
	d.send(ch, fmt.Sprintf("  nosummon   %s\r\n", onOff(ch.PlayerAct.Has(types.PlrNoSummon))))
	d.send(ch, fmt.Sprintf("  noloot     %s\r\n", onOff(ch.PlayerAct.Has(types.PlrNoLoot))))
	d.send(ch, "\r\nDisplay settings:\r\n")
	d.send(ch, fmt.Sprintf("  brief      %s\r\n", onOff(ch.Comm.Has(types.CommBrief))))
	d.send(ch, fmt.Sprintf("  compact    %s\r\n", onOff(ch.Comm.Has(types.CommCompact))))
	d.send(ch, fmt.Sprintf("  color      %s\r\n", onOff(ch.Comm.Has(types.CommColor))))
}

func onOff(value bool) string {
	if value {
		return "ON"
	}
	return "OFF"
}

func (d *CommandDispatcher) cmdAutoexit(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoExit != 0 {
		ch.PlayerAct &^= types.PlrAutoExit
		d.send(ch, "Exits will no longer be displayed automatically.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoExit
		d.send(ch, "Exits will now be displayed automatically.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutogold(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoGold != 0 {
		ch.PlayerAct &^= types.PlrAutoGold
		d.send(ch, "Autogold removed.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoGold
		d.send(ch, "Automatic gold looting set.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutoloot(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoLoot != 0 {
		ch.PlayerAct &^= types.PlrAutoLoot
		d.send(ch, "Autoloot removed.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoLoot
		d.send(ch, "Automatic corpse looting set.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutosac(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoSac != 0 {
		ch.PlayerAct &^= types.PlrAutoSac
		d.send(ch, "Autosacrifice removed.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoSac
		d.send(ch, "Automatic corpse sacrifice set.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutoassist(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoAssist != 0 {
		ch.PlayerAct &^= types.PlrAutoAssist
		d.send(ch, "Autoassist removed.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoAssist
		d.send(ch, "You will now assist group members in combat.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutosplit(ch *types.Character, args string) {
	if ch.PlayerAct&types.PlrAutoSplit != 0 {
		ch.PlayerAct &^= types.PlrAutoSplit
		d.send(ch, "Autosplit removed.\r\n")
	} else {
		ch.PlayerAct |= types.PlrAutoSplit
		d.send(ch, "Automatic gold splitting set.\r\n")
	}
}

// MaxAliases is the maximum number of aliases a player can have
const MaxAliases = 30

func (d *CommandDispatcher) cmdAlias(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	// Initialize aliases map if nil
	if ch.PCData.Aliases == nil {
		ch.PCData.Aliases = make(map[string]string)
	}

	// No arguments - list all aliases
	if args == "" {
		if len(ch.PCData.Aliases) == 0 {
			d.send(ch, "You have no aliases defined.\r\n")
			return
		}
		d.send(ch, "Your current aliases:\r\n")
		for alias, substitution := range ch.PCData.Aliases {
			d.send(ch, fmt.Sprintf("  %s:  %s\r\n", alias, substitution))
		}
		return
	}

	// Parse the alias and substitution
	parts := strings.SplitN(args, " ", 2)
	aliasName := strings.ToLower(parts[0])

	// Just an alias name - show that specific alias
	if len(parts) == 1 {
		if substitution, ok := ch.PCData.Aliases[aliasName]; ok {
			d.send(ch, fmt.Sprintf("%s is aliased to: %s\r\n", aliasName, substitution))
		} else {
			d.send(ch, fmt.Sprintf("'%s' is not an alias.\r\n", aliasName))
		}
		return
	}

	substitution := parts[1]

	// Check if it's a builtin command (can't override)
	if d.Registry.Find(aliasName) != nil {
		d.send(ch, "You can't alias a builtin command.\r\n")
		return
	}

	// Check alias length
	if len(aliasName) > 20 {
		d.send(ch, "Alias name too long (max 20 characters).\r\n")
		return
	}

	if len(substitution) > 100 {
		d.send(ch, "Alias substitution too long (max 100 characters).\r\n")
		return
	}

	// Check for recursive aliases
	firstWord := strings.ToLower(strings.Fields(substitution)[0])
	if firstWord == aliasName {
		d.send(ch, "Aliases cannot be recursive.\r\n")
		return
	}

	// Check max aliases
	if _, exists := ch.PCData.Aliases[aliasName]; !exists && len(ch.PCData.Aliases) >= MaxAliases {
		d.send(ch, fmt.Sprintf("You have reached the maximum of %d aliases.\r\n", MaxAliases))
		return
	}

	// Set the alias
	ch.PCData.Aliases[aliasName] = substitution
	d.send(ch, fmt.Sprintf("Alias '%s' set to: %s\r\n", aliasName, substitution))
}

func (d *CommandDispatcher) cmdUnalias(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		return
	}

	if args == "" {
		d.send(ch, "Unalias what?\r\n")
		return
	}

	if ch.PCData.Aliases == nil {
		d.send(ch, "You have no aliases defined.\r\n")
		return
	}

	aliasName := strings.ToLower(args)

	if _, ok := ch.PCData.Aliases[aliasName]; !ok {
		d.send(ch, fmt.Sprintf("'%s' is not an alias.\r\n", aliasName))
		return
	}

	delete(ch.PCData.Aliases, aliasName)
	d.send(ch, fmt.Sprintf("Alias '%s' removed.\r\n", aliasName))
}

func (d *CommandDispatcher) cmdNosummon(ch *types.Character, args string) {
	if ch.PlayerAct.Has(types.PlrNoSummon) {
		ch.PlayerAct.Remove(types.PlrNoSummon)
		d.send(ch, "You may now be summoned.\r\n")
	} else {
		ch.PlayerAct.Set(types.PlrNoSummon)
		d.send(ch, "You may no longer be summoned.\r\n")
	}
}

func (d *CommandDispatcher) cmdNoloot(ch *types.Character, args string) {
	if ch.PlayerAct.Has(types.PlrNoLoot) {
		ch.PlayerAct.Remove(types.PlrNoLoot)
		d.send(ch, "Your corpse may now be looted.\r\n")
	} else {
		ch.PlayerAct.Set(types.PlrNoLoot)
		d.send(ch, "Your corpse may no longer be looted by others.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutostore(ch *types.Character, args string) {
	if ch.PlayerAct.Has(types.PlrAutoStore) {
		ch.PlayerAct.Remove(types.PlrAutoStore)
		d.send(ch, "Autostore removed.\r\n")
	} else {
		ch.PlayerAct.Set(types.PlrAutoStore)
		d.send(ch, "Automatic item storing set.\r\n")
	}
}

func (d *CommandDispatcher) cmdAutopeek(ch *types.Character, args string) {
	if ch.PlayerAct.Has(types.PlrAutoPeek) {
		ch.PlayerAct.Remove(types.PlrAutoPeek)
		d.send(ch, "Autopeek removed.\r\n")
	} else {
		ch.PlayerAct.Set(types.PlrAutoPeek)
		d.send(ch, "Automatic peeking at characters set.\r\n")
	}
}

func (d *CommandDispatcher) cmdColor(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommColor) {
		ch.Comm.Remove(types.CommColor)
		d.send(ch, "Color is now OFF.\r\n")
	} else {
		ch.Comm.Set(types.CommColor)
		d.send(ch, "\033[1;31mC\033[1;33mo\033[1;32ml\033[1;36mo\033[1;34mr\033[0m is now ON.\r\n")
	}
}

func (d *CommandDispatcher) cmdOutfit(ch *types.Character, args string) {
	if ch.PCData == nil {
		d.send(ch, "Only players can use outfit.\r\n")
		return
	}

	if ch.Level > 5 {
		d.send(ch, "You are too experienced to need an outfit.\r\n")
		return
	}

	// Check if player already has basic equipment
	hasLight := ch.GetEquipment(types.WearLocLight) != nil
	hasVest := ch.GetEquipment(types.WearLocBody) != nil
	hasWeapon := ch.GetEquipment(types.WearLocWield) != nil
	hasShield := ch.GetEquipment(types.WearLocShield) != nil

	if hasLight && hasVest && hasWeapon && hasShield {
		d.send(ch, "You already have all the basic equipment you need.\r\n")
		return
	}

	d.send(ch, "You receive a set of basic equipment:\r\n")

	// Give a torch if no light
	if !hasLight {
		torch := types.NewObject(0, "a small torch", types.ItemTypeLight)
		torch.Name = "torch"
		torch.LongDesc = "A small torch flickers here."
		torch.Level = 1
		torch.Weight = 5
		torch.WearFlags.Set(types.WearTake)
		torch.WearFlags.Set(types.WearHold)
		torch.Values[2] = 24 // Hours of light

		ObjToChar(torch, ch)
		d.send(ch, "  - a small torch\r\n")
	}

	// Give a vest if no body armor
	if !hasVest {
		vest := types.NewObject(0, "a cloth vest", types.ItemTypeArmor)
		vest.Name = "vest cloth"
		vest.LongDesc = "A simple cloth vest lies here."
		vest.Level = 1
		vest.Weight = 10
		vest.WearFlags.Set(types.WearTake)
		vest.WearFlags.Set(types.WearBody)
		vest.Values[0] = 3 // AC bonus

		ObjToChar(vest, ch)
		d.send(ch, "  - a cloth vest\r\n")
	}

	// Give a sword if no weapon
	// Based on school sword (vnum 3702): 1d6 slash with +1 hitroll
	if !hasWeapon {
		sword := types.NewObject(3702, "a sub issue sword", types.ItemTypeWeapon)
		sword.Name = "sword sub merc"
		sword.LongDesc = "You see a sub issue sword here."
		sword.Level = 1
		sword.Weight = 30
		sword.Cost = 360
		sword.WearFlags.Set(types.WearTake)
		sword.WearFlags.Set(types.WearWield)
		sword.Values[0] = 1 // Weapon type: sword
		sword.Values[1] = 1 // Dice number
		sword.Values[2] = 6 // Dice size (1d6)
		sword.Values[3] = 3 // Damage type: slash

		// Add +1 hitroll affect (like school sword APPLY_HITROLL 1)
		sword.Affects.Add(&types.Affect{
			Type:     "object",
			Level:    1,
			Duration: -1, // Permanent
			Location: types.ApplyHitroll,
			Modifier: 1,
		})

		ObjToChar(sword, ch)
		d.send(ch, "  - a sub issue sword\r\n")
	}

	// Give a shield if no shield
	if !hasShield {
		shield := types.NewObject(0, "a wooden shield", types.ItemTypeArmor)
		shield.Name = "shield wooden"
		shield.LongDesc = "A wooden shield lies here."
		shield.Level = 1
		shield.Weight = 20
		shield.WearFlags.Set(types.WearTake)
		shield.WearFlags.Set(types.WearShield)
		shield.Values[0] = 2 // AC bonus

		ObjToChar(shield, ch)
		d.send(ch, "  - a wooden shield\r\n")
	}

	d.send(ch, "Use 'wear all' to equip your new gear.\r\n")
}

// cmdCombine - Toggle combined inventory display
func (d *CommandDispatcher) cmdCombine(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommCombine) {
		ch.Comm.Remove(types.CommCombine)
		d.send(ch, "Long inventory selected.\r\n")
	} else {
		ch.Comm.Set(types.CommCombine)
		d.send(ch, "Combined inventory selected.\r\n")
	}
}

// cmdScroll - Set lines per page for scrolling
func (d *CommandDispatcher) cmdScroll(ch *types.Character, args string) {
	if ch.PCData == nil {
		d.send(ch, "Only players can set scroll.\r\n")
		return
	}

	if args == "" {
		if ch.Lines == 0 {
			d.send(ch, "You do not page long messages.\r\n")
		} else {
			d.send(ch, fmt.Sprintf("You currently display %d lines per page.\r\n", ch.Lines+2))
		}
		return
	}

	lines, err := strconv.Atoi(args)
	if err != nil {
		d.send(ch, "You must provide a number.\r\n")
		return
	}

	if lines == 0 {
		d.send(ch, "Paging disabled.\r\n")
		ch.Lines = 0
		return
	}

	if lines < 10 || lines > 100 {
		d.send(ch, "You must provide a reasonable number (10-100).\r\n")
		return
	}

	ch.Lines = lines - 2
	d.send(ch, fmt.Sprintf("Scroll set to %d lines.\r\n", lines))
}

// cmdLong - Toggle long descriptions (opposite of brief)
func (d *CommandDispatcher) cmdLong(ch *types.Character, args string) {
	if ch.Comm.Has(types.CommLong) {
		ch.Comm.Remove(types.CommLong)
		d.send(ch, "Short descriptions activated.\r\n")
	} else {
		ch.Comm.Set(types.CommLong)
		d.send(ch, "Long descriptions activated.\r\n")
	}
}

// cmdNotran - Toggle allowing transfers (immortal only)
func (d *CommandDispatcher) cmdNotran(ch *types.Character, args string) {
	if ch.PlayerAct.Has(types.PlrNoTrans) {
		ch.PlayerAct.Remove(types.PlrNoTrans)
		d.send(ch, "You may now be transferred.\r\n")
	} else {
		ch.PlayerAct.Set(types.PlrNoTrans)
		d.send(ch, "You may no longer be transferred.\r\n")
	}
}

// cmdPrefix - Set command prefix (for immortals)
func (d *CommandDispatcher) cmdPrefix(ch *types.Character, args string) {
	if ch.PCData == nil {
		return
	}

	if args == "" {
		if ch.Prefix == "" {
			d.send(ch, "You have no prefix to clear.\r\n")
		} else {
			ch.Prefix = ""
			d.send(ch, "Prefix removed.\r\n")
		}
		return
	}

	if ch.Prefix != "" {
		d.send(ch, fmt.Sprintf("Prefix changed to %s.\r\n", args))
	} else {
		d.send(ch, fmt.Sprintf("Prefix set to %s.\r\n", args))
	}
	ch.Prefix = args
}
