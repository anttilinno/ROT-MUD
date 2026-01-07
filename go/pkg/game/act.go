package game

import (
	"strings"

	"rotmud/pkg/types"
)

// OutputFunc is a function that sends output to a character
type OutputFunc func(ch *types.Character, msg string)

// ActFormat formats a message using MUD-style tokens
// Tokens:
//
//	$n - actor name
//	$N - victim name
//	$e - he/she/it (actor)
//	$E - he/she/it (victim)
//	$m - him/her/it (actor)
//	$M - him/her/it (victim)
//	$s - his/her/its (actor)
//	$S - his/her/its (victim)
//	$p - object short description
//	$P - second object short description
func ActFormat(format string, ch *types.Character, victim *types.Character, obj *types.Object) string {
	var result strings.Builder
	result.Grow(len(format) * 2)

	for i := 0; i < len(format); i++ {
		if format[i] != '$' || i+1 >= len(format) {
			result.WriteByte(format[i])
			continue
		}

		i++
		switch format[i] {
		case 'n':
			if ch != nil {
				result.WriteString(ch.Name)
			}
		case 'N':
			if victim != nil {
				result.WriteString(victim.Name)
			}
		case 'e':
			if ch != nil {
				result.WriteString(heSheIt(ch))
			}
		case 'E':
			if victim != nil {
				result.WriteString(heSheIt(victim))
			}
		case 'm':
			if ch != nil {
				result.WriteString(himHerIt(ch))
			}
		case 'M':
			if victim != nil {
				result.WriteString(himHerIt(victim))
			}
		case 's':
			if ch != nil {
				result.WriteString(hisHerIts(ch))
			}
		case 'S':
			if victim != nil {
				result.WriteString(hisHerIts(victim))
			}
		case 'p':
			if obj != nil {
				result.WriteString(obj.ShortDesc)
			}
		case 'P':
			// Second object - not commonly used
			if obj != nil {
				result.WriteString(obj.ShortDesc)
			}
		default:
			// Unknown token, pass through
			result.WriteByte('$')
			result.WriteByte(format[i])
		}
	}

	return result.String()
}

// heSheIt returns the subjective pronoun for a character
func heSheIt(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "he"
	case types.SexFemale:
		return "she"
	default:
		return "it"
	}
}

// himHerIt returns the objective pronoun for a character
func himHerIt(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "him"
	case types.SexFemale:
		return "her"
	default:
		return "it"
	}
}

// hisHerIts returns the possessive pronoun for a character
func hisHerIts(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "his"
	case types.SexFemale:
		return "her"
	default:
		return "its"
	}
}

// ActToRoom sends a formatted message to everyone in the room except the actor
func ActToRoom(format string, ch *types.Character, victim *types.Character, obj *types.Object, output OutputFunc) {
	if ch == nil || ch.InRoom == nil {
		return
	}

	msg := ActFormat(format, ch, victim, obj) + "\r\n"

	for _, person := range ch.InRoom.People {
		if person == ch {
			continue
		}
		output(person, msg)
	}
}

// ActToChar sends a formatted message to the actor only
func ActToChar(format string, ch *types.Character, victim *types.Character, obj *types.Object, output OutputFunc) {
	if ch == nil {
		return
	}
	msg := ActFormat(format, ch, victim, obj) + "\r\n"
	output(ch, msg)
}

// ActToVict sends a formatted message to the victim only
func ActToVict(format string, ch *types.Character, victim *types.Character, obj *types.Object, output OutputFunc) {
	if victim == nil {
		return
	}
	msg := ActFormat(format, ch, victim, obj) + "\r\n"
	output(victim, msg)
}

// ActToNotVict sends a formatted message to everyone in the room except actor and victim
func ActToNotVict(format string, ch *types.Character, victim *types.Character, obj *types.Object, output OutputFunc) {
	if ch == nil || ch.InRoom == nil {
		return
	}

	msg := ActFormat(format, ch, victim, obj) + "\r\n"

	for _, person := range ch.InRoom.People {
		if person == ch || person == victim {
			continue
		}
		output(person, msg)
	}
}

// ActToAll sends a formatted message to everyone in the room including the actor
func ActToAll(format string, ch *types.Character, victim *types.Character, obj *types.Object, output OutputFunc) {
	if ch == nil || ch.InRoom == nil {
		return
	}

	msg := ActFormat(format, ch, victim, obj) + "\r\n"

	for _, person := range ch.InRoom.People {
		output(person, msg)
	}
}
