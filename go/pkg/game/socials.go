package game

import (
	"strings"

	"rotmud/pkg/types"
)

// Social represents a social action
type Social struct {
	Name        string // Command name (e.g., "smile", "laugh")
	CharNoArg   string // Message to character with no argument
	OthersNoArg string // Message to room with no argument
	CharFound   string // Message to character when target found
	OthersFound string // Message to room when target found (not target)
	VictFound   string // Message to target
	CharAuto    string // Message to character when targeting self
	OthersAuto  string // Message to room when targeting self
	NotFound    string // Message when target not found
}

// SocialRegistry holds all registered socials
type SocialRegistry struct {
	socials map[string]*Social
}

// NewSocialRegistry creates a new social registry with default socials
func NewSocialRegistry() *SocialRegistry {
	r := &SocialRegistry{
		socials: make(map[string]*Social),
	}
	r.registerDefaults()
	return r
}

// Register adds a social to the registry
func (r *SocialRegistry) Register(s *Social) {
	r.socials[strings.ToLower(s.Name)] = s
}

// Find looks up a social by name
func (r *SocialRegistry) Find(name string) *Social {
	name = strings.ToLower(name)

	// Exact match
	if s, ok := r.socials[name]; ok {
		return s
	}

	// Prefix match
	for socName, soc := range r.socials {
		if strings.HasPrefix(socName, name) {
			return soc
		}
	}

	return nil
}

// All returns all socials (for listing)
func (r *SocialRegistry) All() []*Social {
	result := make([]*Social, 0, len(r.socials))
	for _, s := range r.socials {
		result = append(result, s)
	}
	return result
}

// registerDefaults adds common socials
func (r *SocialRegistry) registerDefaults() {
	// Basic positive socials
	r.Register(&Social{
		Name:        "smile",
		CharNoArg:   "You smile happily.",
		OthersNoArg: "$n smiles happily.",
		CharFound:   "You smile at $N.",
		OthersFound: "$n beams a smile at $N.",
		VictFound:   "$n smiles at you.",
		CharAuto:    "You smile at yourself.",
		OthersAuto:  "$n smiles at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "grin",
		CharNoArg:   "You grin evilly.",
		OthersNoArg: "$n grins evilly.",
		CharFound:   "You grin evilly at $N.",
		OthersFound: "$n grins evilly at $N.",
		VictFound:   "$n grins evilly at you.",
		CharAuto:    "You grin at yourself.",
		OthersAuto:  "$n grins at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "laugh",
		CharNoArg:   "You fall down laughing.",
		OthersNoArg: "$n falls down laughing.",
		CharFound:   "You laugh at $N mercilessly.",
		OthersFound: "$n laughs at $N mercilessly.",
		VictFound:   "$n laughs at you mercilessly.",
		CharAuto:    "You laugh at yourself.",
		OthersAuto:  "$n laughs at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "giggle",
		CharNoArg:   "You giggle.",
		OthersNoArg: "$n giggles.",
		CharFound:   "You giggle at $N.",
		OthersFound: "$n giggles at $N.",
		VictFound:   "$n giggles at you.",
		CharAuto:    "You giggle at yourself.",
		OthersAuto:  "$n giggles at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "wave",
		CharNoArg:   "You wave.",
		OthersNoArg: "$n waves happily.",
		CharFound:   "You wave at $N.",
		OthersFound: "$n waves at $N.",
		VictFound:   "$n waves at you.",
		CharAuto:    "You wave at yourself.",
		OthersAuto:  "$n waves at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "nod",
		CharNoArg:   "You nod solemnly.",
		OthersNoArg: "$n nods solemnly.",
		CharFound:   "You nod at $N.",
		OthersFound: "$n nods at $N.",
		VictFound:   "$n nods at you.",
		CharAuto:    "You nod at yourself.",
		OthersAuto:  "$n nods at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "shake",
		CharNoArg:   "You shake your head.",
		OthersNoArg: "$n shakes $s head.",
		CharFound:   "You shake $N's hand.",
		OthersFound: "$n shakes $N's hand.",
		VictFound:   "$n shakes your hand.",
		CharAuto:    "You shake your own hand.",
		OthersAuto:  "$n shakes $s own hand.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "bow",
		CharNoArg:   "You bow deeply.",
		OthersNoArg: "$n bows deeply.",
		CharFound:   "You bow before $N.",
		OthersFound: "$n bows before $N.",
		VictFound:   "$n bows before you.",
		CharAuto:    "You bow to yourself.",
		OthersAuto:  "$n bows to $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "hug",
		CharNoArg:   "You hug yourself.",
		OthersNoArg: "$n hugs $mself.",
		CharFound:   "You hug $N warmly.",
		OthersFound: "$n hugs $N warmly.",
		VictFound:   "$n hugs you warmly.",
		CharAuto:    "You hug yourself.",
		OthersAuto:  "$n hugs $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "poke",
		CharNoArg:   "You poke yourself in the ribs.",
		OthersNoArg: "$n pokes $mself in the ribs.",
		CharFound:   "You poke $N in the ribs.",
		OthersFound: "$n pokes $N in the ribs.",
		VictFound:   "$n pokes you in the ribs.",
		CharAuto:    "You poke yourself in the ribs.",
		OthersAuto:  "$n pokes $mself in the ribs.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "slap",
		CharNoArg:   "You slap yourself.",
		OthersNoArg: "$n slaps $mself.",
		CharFound:   "You slap $N.",
		OthersFound: "$n slaps $N.",
		VictFound:   "$n slaps you.",
		CharAuto:    "You slap yourself.",
		OthersAuto:  "$n slaps $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "cry",
		CharNoArg:   "You burst into tears.",
		OthersNoArg: "$n bursts into tears.",
		CharFound:   "You cry on $N's shoulder.",
		OthersFound: "$n cries on $N's shoulder.",
		VictFound:   "$n cries on your shoulder.",
		CharAuto:    "You cry on your own shoulder.",
		OthersAuto:  "$n cries on $s own shoulder.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "sigh",
		CharNoArg:   "You sigh.",
		OthersNoArg: "$n sighs loudly.",
		CharFound:   "You sigh at $N.",
		OthersFound: "$n sighs at $N.",
		VictFound:   "$n sighs at you.",
		CharAuto:    "You sigh at yourself.",
		OthersAuto:  "$n sighs at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "groan",
		CharNoArg:   "You groan loudly.",
		OthersNoArg: "$n groans loudly.",
		CharFound:   "You groan at $N.",
		OthersFound: "$n groans at $N.",
		VictFound:   "$n groans at you.",
		CharAuto:    "You groan at yourself.",
		OthersAuto:  "$n groans at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "cheer",
		CharNoArg:   "You cheer enthusiastically!",
		OthersNoArg: "$n cheers enthusiastically!",
		CharFound:   "You cheer for $N!",
		OthersFound: "$n cheers for $N!",
		VictFound:   "$n cheers for you!",
		CharAuto:    "You cheer for yourself!",
		OthersAuto:  "$n cheers for $mself!",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "dance",
		CharNoArg:   "You dance around wildly.",
		OthersNoArg: "$n dances around wildly.",
		CharFound:   "You lead $N across the dance floor.",
		OthersFound: "$n leads $N across the dance floor.",
		VictFound:   "$n leads you across the dance floor.",
		CharAuto:    "You dance by yourself.",
		OthersAuto:  "$n dances by $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "bounce",
		CharNoArg:   "You bounce around happily.",
		OthersNoArg: "$n bounces around happily.",
		CharFound:   "You bounce around $N happily.",
		OthersFound: "$n bounces around $N happily.",
		VictFound:   "$n bounces around you happily.",
		CharAuto:    "You bounce around.",
		OthersAuto:  "$n bounces around.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "cackle",
		CharNoArg:   "You cackle gleefully.",
		OthersNoArg: "$n throws back $s head and cackles with insane glee!",
		CharFound:   "You cackle gleefully at $N.",
		OthersFound: "$n cackles gleefully at $N.",
		VictFound:   "$n cackles gleefully at you.",
		CharAuto:    "You cackle at yourself.",
		OthersAuto:  "$n cackles at $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "thank",
		CharNoArg:   "You thank everyone.",
		OthersNoArg: "$n thanks everyone.",
		CharFound:   "You thank $N heartily.",
		OthersFound: "$n thanks $N heartily.",
		VictFound:   "$n thanks you heartily.",
		CharAuto:    "You thank yourself.",
		OthersAuto:  "$n thanks $mself.",
		NotFound:    "They aren't here.",
	})

	r.Register(&Social{
		Name:        "wink",
		CharNoArg:   "You wink suggestively.",
		OthersNoArg: "$n winks suggestively.",
		CharFound:   "You wink at $N.",
		OthersFound: "$n winks at $N.",
		VictFound:   "$n winks at you.",
		CharAuto:    "You wink at yourself.",
		OthersAuto:  "$n winks at $mself.",
		NotFound:    "They aren't here.",
	})
}

// PerformSocial executes a social action
func PerformSocial(ch *types.Character, social *Social, target *types.Character, output func(ch *types.Character, msg string)) {
	if output == nil {
		return
	}

	if target == nil {
		// No target specified
		output(ch, formatSocial(social.CharNoArg, ch, nil)+"\r\n")
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if person != ch {
					output(person, formatSocial(social.OthersNoArg, ch, nil)+"\r\n")
				}
			}
		}
	} else if target == ch {
		// Self-target
		output(ch, formatSocial(social.CharAuto, ch, ch)+"\r\n")
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if person != ch {
					output(person, formatSocial(social.OthersAuto, ch, ch)+"\r\n")
				}
			}
		}
	} else {
		// Targeting someone else
		output(ch, formatSocial(social.CharFound, ch, target)+"\r\n")
		output(target, formatSocial(social.VictFound, ch, target)+"\r\n")
		if ch.InRoom != nil {
			for _, person := range ch.InRoom.People {
				if person != ch && person != target {
					output(person, formatSocial(social.OthersFound, ch, target)+"\r\n")
				}
			}
		}
	}
}

// formatSocial replaces social message placeholders
// $n = character name, $N = victim name, $s = his/her, $m = him/her, $e = he/she
func formatSocial(msg string, ch, victim *types.Character) string {
	if ch == nil {
		return msg
	}

	result := strings.Builder{}
	for i := 0; i < len(msg); i++ {
		if i < len(msg)-1 && msg[i] == '$' {
			switch msg[i+1] {
			case 'n':
				result.WriteString(ch.Name)
				i++
				continue
			case 'N':
				if victim != nil {
					result.WriteString(victim.Name)
				} else {
					result.WriteString("someone")
				}
				i++
				continue
			case 's':
				result.WriteString(hisHer(ch))
				i++
				continue
			case 'S':
				if victim != nil {
					result.WriteString(hisHer(victim))
				} else {
					result.WriteString("their")
				}
				i++
				continue
			case 'm':
				result.WriteString(himHer(ch))
				i++
				continue
			case 'M':
				if victim != nil {
					result.WriteString(himHer(victim))
				} else {
					result.WriteString("them")
				}
				i++
				continue
			case 'e':
				result.WriteString(heShe(ch))
				i++
				continue
			case 'E':
				if victim != nil {
					result.WriteString(heShe(victim))
				} else {
					result.WriteString("they")
				}
				i++
				continue
			}
		}
		result.WriteByte(msg[i])
	}
	return result.String()
}

func hisHer(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "his"
	case types.SexFemale:
		return "her"
	default:
		return "its"
	}
}

func himHer(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "him"
	case types.SexFemale:
		return "her"
	default:
		return "it"
	}
}

func heShe(ch *types.Character) string {
	switch ch.Sex {
	case types.SexMale:
		return "he"
	case types.SexFemale:
		return "she"
	default:
		return "it"
	}
}
