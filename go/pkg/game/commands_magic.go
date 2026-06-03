package game

import (
	"fmt"
	"sort"
	"strings"

	"rotmud/pkg/types"
)

// Magic commands: cast, spells, etc.

func (d *CommandDispatcher) cmdCast(ch *types.Character, args string) {
	if args == "" {
		d.send(ch, "Cast what spell?\r\n")
		return
	}

	if d.Magic == nil {
		d.send(ch, "The magic system is not available.\r\n")
		return
	}

	// Parse spell name and target
	// Handle quoted spell names: cast 'magic missile' target
	// Or unquoted: cast detect invis (tries to find longest matching spell)
	var spellName, target string

	if args[0] == '\'' {
		// Quoted spell name
		endQuote := strings.Index(args[1:], "'")
		if endQuote == -1 {
			d.send(ch, "Spell name must be in quotes: cast 'spell name' [target]\r\n")
			return
		}
		spellName = args[1 : endQuote+1]
		remainder := strings.TrimSpace(args[endQuote+2:])
		if remainder != "" {
			target = remainder
		}
	} else {
		// Unquoted - try to find the longest matching spell name
		// Start with all words, then try removing words from the end
		words := strings.Fields(args)
		foundSpell := false
		for i := len(words); i >= 1; i-- {
			tryName := strings.ToLower(strings.Join(words[:i], " "))
			if d.Magic.Registry.FindByPrefix(tryName) != nil {
				spellName = tryName
				if i < len(words) {
					target = strings.Join(words[i:], " ")
				}
				foundSpell = true
				break
			}
		}
		if !foundSpell {
			// Fall back to first word only
			spellName = strings.ToLower(words[0])
			if len(words) > 1 {
				target = strings.Join(words[1:], " ")
			}
		}
	}

	// Create target finder callback
	targetFinder := func(caster *types.Character, name string, offensive bool) interface{} {
		victim := FindCharInRoom(caster, name)
		if victim != nil {
			return victim
		}
		return nil
	}

	// Cast the spell
	d.Magic.Cast(ch, spellName, target, targetFinder)
}

func (d *CommandDispatcher) cmdSpells(ch *types.Character, args string) {
	if ch.IsNPC() || ch.PCData == nil {
		d.send(ch, "NPCs don't have spells.\r\n")
		return
	}

	if d.Magic == nil || d.Magic.Registry == nil {
		d.send(ch, "The magic system is not available.\r\n")
		return
	}

	// Collect all spells for this class
	type spellInfo struct {
		name    string
		level   int
		mana    int
		learned int
		canUse  bool
	}

	// Max level a player can reach - spells with higher level requirements are not learnable
	const maxPlayerLevel = 51

	var classSpells []spellInfo
	for _, spell := range d.Magic.Registry.All() {
		reqLevel := spell.GetClassLevel(ch.Class)
		if reqLevel == 0 || reqLevel > maxPlayerLevel {
			continue // Class can't learn this spell (0 = not available, 53+ = class restriction)
		}

		learned := 0
		if ch.PCData.Learned != nil {
			learned = ch.PCData.Learned[spell.Name]
		}

		classSpells = append(classSpells, spellInfo{
			name:    spell.Name,
			level:   reqLevel,
			mana:    spell.ManaCost,
			learned: learned,
			canUse:  ch.Level >= reqLevel,
		})
	}

	// Sort by level requirement
	sort.Slice(classSpells, func(i, j int) bool {
		if classSpells[i].level != classSpells[j].level {
			return classSpells[i].level < classSpells[j].level
		}
		return classSpells[i].name < classSpells[j].name
	})

	if len(classSpells) == 0 {
		d.send(ch, "You have no spells available.\r\n")
		return
	}

	d.send(ch, "Spells available to your class:\r\n\r\n")
	d.send(ch, "Spell                Lv  Mana  Proficiency\r\n")
	d.send(ch, "-------------------------------------------\r\n")

	for _, s := range classSpells {
		var profStr string
		if s.canUse {
			rating := skillRating(s.learned)
			profStr = fmt.Sprintf("%3d%% %s", s.learned, rating)
		} else {
			profStr = "n/a"
		}
		d.send(ch, fmt.Sprintf("%-18s %3d  %4d  %s\r\n", s.name, s.level, s.mana, profStr))
	}

	d.send(ch, fmt.Sprintf("\r\nCurrent mana: %d/%d\r\n", ch.Mana, ch.MaxMana))
	d.send(ch, fmt.Sprintf("You have %d practice sessions left.\r\n", ch.Practice))
}

// healOffer is one purchasable service sold by a temple healer (spec_cast_adept).
// Cost is the price in copper (the economy's base unit; historical "gold" numbers
// in area data are read as copper). Spell is the magic-registry spell name, or ""
// for the special "mana" restore.
type healOffer struct {
	Keyword string
	Spell   string
	Word    string // magic word the healer utters
	Cost    int64  // copper
	Desc    string
}

// healMenu mirrors the classic ROM healer (do_heal in healer.c). Prices are in
// copper, scaled to this server's economy.
var healMenu = []healOffer{
	{"light", "cure light", "judicandus dies", 10, "cure light wounds"},
	{"serious", "cure serious", "judicandus med", 15, "cure serious wounds"},
	{"critic", "cure critical", "judicandus maximus", 25, "cure critical wounds"},
	{"heal", "heal", "pumo", 50, "healing spell"},
	{"blind", "cure blindness", "judicandus noselacri", 20, "cure blindness"},
	{"disease", "cure disease", "judicandus eugzagz", 15, "cure disease"},
	{"poison", "cure poison", "judicandus sausabru", 25, "cure poison"},
	{"uncurse", "remove curse", "candusima", 50, "remove curse"},
	{"refresh", "refresh", "candusima", 5, "restore movement"},
	{"mana", "", "energizer", 10, "restore mana"},
}

// healerOrder names the faction a healer collects donations for. There is no
// structured faction field on mobs yet, so this is a sensible generic default;
// it is the single place to wire a per-mob faction later.
func healerOrder(healer *types.Character) string {
	return "the temple"
}

// findHealer returns the first visible NPC in the room running the healer
// special function, or nil if none is present.
func (d *CommandDispatcher) findHealer(ch *types.Character) *types.Character {
	if ch.InRoom == nil {
		return nil
	}
	for _, mob := range ch.InRoom.People {
		if mob.IsNPC() && mob.Special == "spec_cast_adept" {
			return mob
		}
	}
	return nil
}

// cmdHeal implements the healer shop: with no argument it prints the menu of
// services and prices; with a keyword it charges the player and has the healer
// cast the corresponding spell. Ported from ROM's do_heal.
func (d *CommandDispatcher) cmdHeal(ch *types.Character, args string) {
	healer := d.findHealer(ch)
	if healer == nil {
		d.send(ch, "You can't do that here.\r\n")
		return
	}

	order := healerOrder(healer)
	arg := strings.ToLower(strings.TrimSpace(args))
	if arg == "" {
		d.send(ch, fmt.Sprintf("In return for a donation to %s, the healer offers:\r\n\r\n", order))
		for _, o := range healMenu {
			d.send(ch, fmt.Sprintf("  %-8s - %-22s suggested donation: %s\r\n",
				o.Keyword, o.Desc, types.FormatCoin(o.Cost)))
		}
		d.send(ch, "\r\nType heal <type> to make a donation and receive a blessing.\r\n")
		return
	}

	var offer *healOffer
	for i := range healMenu {
		if strings.HasPrefix(healMenu[i].Keyword, arg) {
			offer = &healMenu[i]
			break
		}
	}
	if offer == nil {
		ActToChar("$N tells you 'Type 'heal' to see the blessings I offer for a donation.'", ch, healer, nil, d.Output)
		return
	}

	cost := offer.Cost
	if ch.Coin < cost {
		ActToChar("$N tells you 'That donation is beyond your means right now, child.'", ch, healer, nil, d.Output)
		return
	}
	ch.Coin -= cost
	ActToChar(fmt.Sprintf("$N accepts your donation of %s to %s.", types.FormatCoin(cost), order), ch, healer, nil, d.Output)

	ActToRoom("$n utters the words '"+offer.Word+"'.", healer, nil, nil, d.Output)

	if offer.Spell == "" {
		// "mana" — not a spell; restore the buyer's mana.
		ch.Mana = ch.MaxMana
		d.send(ch, "A warm glow passes through you; your mana is restored.\r\n")
		return
	}

	if d.Magic == nil || d.Magic.Registry == nil {
		d.send(ch, "The magic system is not available.\r\n")
		return
	}
	spell := d.Magic.Registry.FindByName(offer.Spell)
	if spell == nil || spell.Func == nil {
		d.send(ch, "Something goes wrong with the spell.\r\n")
		return
	}
	spell.Func(healer, healer.Level, ch)
}
