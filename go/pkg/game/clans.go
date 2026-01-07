package game

import (
	"rotmud/pkg/types"
)

// Clan represents a player clan/guild
type Clan struct {
	Name        string   // Clan name
	WhoName     string   // Display name in who list
	Pkill       bool     // Allows player killing
	Independent bool     // Independent clan (special rules)
	Hall        int      // Clan hall room vnum
	Entrance    int      // Clan entrance room vnum
	Pit         int      // Clan pit/donation room vnum
	Members     []string // Member names
	Leaders     []string // Leader names
}

// ClanSystem manages all clans
type ClanSystem struct {
	clans map[int]*Clan // Clan ID to clan
}

// NewClanSystem creates a new clan system
func NewClanSystem() *ClanSystem {
	cs := &ClanSystem{
		clans: make(map[int]*Clan),
	}

	// Initialize default clans
	cs.initializeDefaultClans()

	return cs
}

// initializeDefaultClans sets up the default clans
func (cs *ClanSystem) initializeDefaultClans() {
	// Add some basic clans
	clans := []Clan{
		{
			Name:        "Lone Wolf",
			WhoName:     "Lone Wolf",
			Pkill:       false,
			Independent: false,
			Hall:        0,
			Entrance:    0,
			Pit:         0,
			Members:     []string{},
			Leaders:     []string{"admin"},
		},
		{
			Name:        "Brigands",
			WhoName:     "{RB{r Brigand {x",
			Pkill:       true,
			Independent: false,
			Hall:        0,
			Entrance:    0,
			Pit:         0,
			Members:     []string{},
			Leaders:     []string{"admin"},
		},
		{
			Name:        "Knights",
			WhoName:     "{BM{b Knight {x",
			Pkill:       true,
			Independent: false,
			Hall:        0,
			Entrance:    0,
			Pit:         0,
			Members:     []string{},
			Leaders:     []string{"admin"},
		},
	}

	for i, clan := range clans {
		cs.clans[i+1] = &clan
	}
}

// GetClan returns a clan by ID
func (cs *ClanSystem) GetClan(id int) *Clan {
	return cs.clans[id]
}

// GetAllClans returns all clans
func (cs *ClanSystem) GetAllClans() map[int]*Clan {
	return cs.clans
}

// IsClanMember checks if a character is a member of any clan
func (cs *ClanSystem) IsClanMember(ch *types.Character) bool {
	if ch.PCData == nil {
		return false
	}
	return ch.PCData.Clan > 0
}

// GetCharacterClan returns the clan of a character
func (cs *ClanSystem) GetCharacterClan(ch *types.Character) *Clan {
	if ch.PCData == nil {
		return nil
	}
	return cs.GetClan(ch.PCData.Clan)
}

// IsSameClan checks if two characters are in the same clan
func (cs *ClanSystem) IsSameClan(ch1, ch2 *types.Character) bool {
	if ch1.PCData == nil || ch2.PCData == nil {
		return false
	}

	clan1 := ch1.PCData.Clan
	clan2 := ch2.PCData.Clan

	if clan1 == 0 || clan2 == 0 {
		return false
	}

	if clan1 != clan2 {
		return false
	}

	clan := cs.GetClan(clan1)
	if clan == nil {
		return false
	}

	// Independent clans don't count as "same clan" for some purposes
	return !clan.Independent
}

// CanPK checks if a character can attack another (PK rules)
func (cs *ClanSystem) CanPK(ch1, ch2 *types.Character) bool {
	if ch1.PCData == nil || ch2.PCData == nil {
		return false
	}

	clan1 := cs.GetCharacterClan(ch1)
	clan2 := cs.GetCharacterClan(ch2)

	// If either has no clan, allow PK (simplified)
	if clan1 == nil || clan2 == nil {
		return true
	}

	// Same clan can't fight (unless independent)
	if cs.IsSameClan(ch1, ch2) {
		return false
	}

	// Both clans must allow PK
	if !clan1.Pkill || !clan2.Pkill {
		return false
	}

	return true
}

// AddMember adds a character to a clan
func (cs *ClanSystem) AddMember(clanID int, ch *types.Character) bool {
	clan := cs.GetClan(clanID)
	if clan == nil || ch.PCData == nil {
		return false
	}

	// Remove from old clan if any
	if ch.PCData.Clan > 0 {
		cs.RemoveMember(ch.PCData.Clan, ch)
	}

	// Add to new clan
	ch.PCData.Clan = clanID
	clan.Members = append(clan.Members, ch.Name)

	return true
}

// RemoveMember removes a character from their clan
func (cs *ClanSystem) RemoveMember(clanID int, ch *types.Character) bool {
	clan := cs.GetClan(clanID)
	if clan == nil || ch.PCData == nil {
		return false
	}

	// Remove from member list
	for i, name := range clan.Members {
		if name == ch.Name {
			clan.Members = append(clan.Members[:i], clan.Members[i+1:]...)
			break
		}
	}

	ch.PCData.Clan = 0
	return true
}

// IsClanLeader checks if a character is a leader of their clan
func (cs *ClanSystem) IsClanLeader(ch *types.Character) bool {
	clan := cs.GetCharacterClan(ch)
	if clan == nil || ch.PCData == nil {
		return false
	}

	for _, leader := range clan.Leaders {
		if leader == ch.Name {
			return true
		}
	}
	return false
}

// CanUseClanItem checks if a character can use a clan-restricted item
func (cs *ClanSystem) CanUseClanItem(ch *types.Character, clanID int) bool {
	if clanID == 0 {
		return true // No clan restriction
	}

	if ch.PCData == nil {
		return false
	}

	return ch.PCData.Clan == clanID
}
