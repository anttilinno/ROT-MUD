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
