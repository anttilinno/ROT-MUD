package combat

import "rotmud/pkg/types"

// IsSameGroup is ROM is_same_group: both follow the same leader.
func IsSameGroup(a, b *types.Character) bool {
	if a == nil || b == nil {
		return false
	}
	if a.Leader != nil {
		a = a.Leader
	}
	if b.Leader != nil {
		b = b.Leader
	}
	return a == b
}

// checkAssist is ROT check_assist (fight.c): after ch's round against
// victim, idle bystanders in the room may join in.
func (c *CombatSystem) checkAssist(ch, victim *types.Character) {
	room := ch.InRoom
	for _, rch := range append([]*types.Character(nil), room.People...) {
		if victim.Position == types.PosDead || victim.InRoom != room {
			return // an earlier helper finished the fight
		}
		if !IsAwake(rch) || rch.Fighting != nil || rch.InRoom != room {
			continue
		}

		// ASSIST_PLAYERS: mobs that side with players against tougher mobs.
		if !ch.IsNPC() && rch.IsNPC() && rch.Off.Has(types.OffAssistPlayers) &&
			rch.Level+6 > victim.Level {
			c.screamAndAttack(rch, victim)
			continue
		}

		// Players and charmies: autoassist and pets help their group.
		if !ch.IsNPC() || ch.IsAffected(types.AffCharm) {
			if ((!rch.IsNPC() && rch.PlayerAct.Has(types.PlrAutoAssist)) || rch.IsAffected(types.AffCharm)) &&
				IsSameGroup(ch, rch) && !IsSafe(rch, victim) {
				c.MultiHit(rch, victim)
			}
			continue
		}

		if !rch.IsNPC() || !mobAssists(rch, ch) || NumberBits(1) == 0 {
			continue
		}
		// Pick a random member of the victim's group the helper can see.
		// ponytail: ROT walks vch->next (char_list) here, a ROM bug that can
		// pick targets outside the room; the room is what it meant.
		var target *types.Character
		n := 0
		for _, vch := range room.People {
			if CanSee(rch, vch) && IsSameGroup(vch, victim) && NumberRange(0, n) == 0 {
				target = vch
				n++
			}
		}
		if target != nil {
			c.screamAndAttack(rch, target)
		}
	}
}

// mobAssists reports whether mob rch takes mob ch's side by its ASSIST_*
// off flags or mob group.
func mobAssists(rch, ch *types.Character) bool {
	switch {
	case rch.Off.Has(types.OffAssistAll):
		return true
	case rch.MobGroup != 0 && rch.MobGroup == ch.MobGroup:
		return true
	case rch.Off.Has(types.OffAssistRace) && rch.MobRace == ch.MobRace:
		return true
	case rch.Off.Has(types.OffAssistAlign) &&
		(rch.IsGood() && ch.IsGood() || rch.IsEvil() && ch.IsEvil() || rch.IsNeutral() && ch.IsNeutral()):
		return true
	}
	return rch.Off.Has(types.OffAssistVnum) && rch.MobVnum == ch.MobVnum
}

func (c *CombatSystem) screamAndAttack(rch, victim *types.Character) {
	if c.Output != nil {
		msg := formatDeathCry("{R$n screams and attacks!{x", rch) + "\r\n"
		for _, p := range rch.InRoom.People {
			c.Output(p, msg)
		}
	}
	c.MultiHit(rch, victim)
}
