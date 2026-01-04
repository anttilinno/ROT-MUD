package game

import (
	"fmt"
	"strings"

	"rotmud/pkg/magic"
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
		// Unquoted - first word is spell name
		parts := strings.SplitN(args, " ", 2)
		spellName = parts[0]
		if len(parts) > 1 {
			target = parts[1]
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
	if d.Magic == nil || d.Magic.Registry == nil {
		d.send(ch, "The magic system is not available.\r\n")
		return
	}

	d.send(ch, "Available spells:\r\n")
	d.send(ch, "----------------\r\n")

	// Get all spells from registry
	allSpells := d.Magic.Registry.All()

	if len(allSpells) == 0 {
		d.send(ch, "No spells are available.\r\n")
		return
	}

	// Filter spells by class and level requirement
	spells := make([]*magic.Spell, 0)
	for _, spell := range allSpells {
		reqLevel := spell.GetClassLevel(ch.Class)
		// Only show spells the player can learn (reqLevel > 0) and has reached the level for
		if reqLevel > 0 && ch.Level >= reqLevel {
			spells = append(spells, spell)
		}
	}

	if len(spells) == 0 {
		d.send(ch, "You don't know any spells yet.\r\n")
		return
	}

	// Group by type for display
	offensive := make([]*magic.Spell, 0)
	defensive := make([]*magic.Spell, 0)
	other := make([]*magic.Spell, 0)

	for _, spell := range spells {
		switch spell.Target {
		case magic.TargetCharOffense:
			offensive = append(offensive, spell)
		case magic.TargetCharDefense, magic.TargetCharSelf:
			defensive = append(defensive, spell)
		default:
			other = append(other, spell)
		}
	}

	if len(offensive) > 0 {
		d.send(ch, "\r\n{cOffensive Spells:{x\r\n")
		for _, spell := range offensive {
			canCast := ""
			if ch.Mana >= spell.ManaCost {
				canCast = " *"
			}
			d.send(ch, fmt.Sprintf("  %-20s (mana: %3d)%s\r\n", spell.Name, spell.ManaCost, canCast))
		}
	}

	if len(defensive) > 0 {
		d.send(ch, "\r\n{gDefensive Spells:{x\r\n")
		for _, spell := range defensive {
			canCast := ""
			if ch.Mana >= spell.ManaCost {
				canCast = " *"
			}
			d.send(ch, fmt.Sprintf("  %-20s (mana: %3d)%s\r\n", spell.Name, spell.ManaCost, canCast))
		}
	}

	if len(other) > 0 {
		d.send(ch, "\r\n{yOther Spells:{x\r\n")
		for _, spell := range other {
			canCast := ""
			if ch.Mana >= spell.ManaCost {
				canCast = " *"
			}
			d.send(ch, fmt.Sprintf("  %-20s (mana: %3d)%s\r\n", spell.Name, spell.ManaCost, canCast))
		}
	}

	d.send(ch, "\r\n* = You have enough mana to cast\r\n")
	d.send(ch, fmt.Sprintf("Current mana: %d/%d\r\n", ch.Mana, ch.MaxMana))
}
