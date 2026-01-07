package game

import (
	"rotmud/pkg/types"
	"strings"
	"time"
)

// QuestType represents different types of quests
type QuestType int

const (
	QuestTypeKill QuestType = iota
	QuestTypeCollect
	QuestTypeDeliver
	QuestTypeExplore
)

// Quest represents a player quest
type Quest struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        QuestType `json:"type"`
	Level       int       `json:"level"`       // Minimum level to accept
	RewardXP    int       `json:"reward_xp"`   // XP reward
	RewardGold  int       `json:"reward_gold"` // Gold reward

	// Quest-specific data
	TargetMob   string `json:"target_mob,omitempty"`  // Mob to kill (for kill quests)
	TargetItem  string `json:"target_item,omitempty"` // Item to collect/deliver
	TargetCount int    `json:"target_count"`          // How many to kill/collect
	TargetRoom  int    `json:"target_room,omitempty"` // Room to explore

	GiverName string `json:"giver_name"` // NPC who gives the quest
}

// PlayerQuest tracks a player's progress on a quest
type PlayerQuest struct {
	QuestID     int       `json:"quest_id"`
	StartedAt   time.Time `json:"started_at"`
	Progress    int       `json:"progress"`  // Current progress (mobs killed, items collected, etc.)
	Completed   bool      `json:"completed"` // Whether quest is completed
	CompletedAt time.Time `json:"completed_at,omitempty"`
}

// QuestSystem manages all quests and player progress
type QuestSystem struct {
	quests       map[int]*Quest            // Quest ID to quest
	playerQuests map[string][]*PlayerQuest // Player name to their active/completed quests
}

// NewQuestSystem creates a new quest system
func NewQuestSystem() *QuestSystem {
	qs := &QuestSystem{
		quests:       make(map[int]*Quest),
		playerQuests: make(map[string][]*PlayerQuest),
	}

	qs.initializeDefaultQuests()
	return qs
}

// initializeDefaultQuests sets up some default quests
func (qs *QuestSystem) initializeDefaultQuests() {
	quests := []*Quest{
		{
			ID:          1,
			Title:       "Rat Extermination",
			Description: "Kill 5 rats in the sewers to help clean up the infestation.",
			Type:        QuestTypeKill,
			Level:       1,
			RewardXP:    100,
			RewardGold:  50,
			TargetMob:   "rat",
			TargetCount: 5,
			GiverName:   "Guard Captain",
		},
		{
			ID:          2,
			Title:       "Lost Ring",
			Description: "Find and return the lost wedding ring to the grieving widow.",
			Type:        QuestTypeCollect,
			Level:       1,
			RewardXP:    150,
			RewardGold:  75,
			TargetItem:  "wedding ring",
			TargetCount: 1,
			GiverName:   "Grieving Widow",
		},
		{
			ID:          3,
			Title:       "Spider Venom",
			Description: "Collect 3 vials of spider venom for the alchemist's research.",
			Type:        QuestTypeCollect,
			Level:       3,
			RewardXP:    200,
			RewardGold:  100,
			TargetItem:  "spider venom",
			TargetCount: 3,
			GiverName:   "Alchemist",
		},
		{
			ID:          4,
			Title:       "Explore the Caves",
			Description: "Venture into the dark caves and report back what you find.",
			Type:        QuestTypeExplore,
			Level:       2,
			RewardXP:    120,
			RewardGold:  60,
			TargetRoom:  100, // Example room vnum
			GiverName:   "Cave Explorer",
		},
		{
			ID:          5,
			Title:       "Bandit Troubles",
			Description: "Clear out 8 bandits who have been terrorizing the trade routes.",
			Type:        QuestTypeKill,
			Level:       3,
			RewardXP:    250,
			RewardGold:  150,
			TargetMob:   "bandit",
			TargetCount: 8,
			GiverName:   "Town Guard",
		},
		{
			ID:          6,
			Title:       "Magical Herbs",
			Description: "Collect 5 rare magical herbs for the alchemist's experiments.",
			Type:        QuestTypeCollect,
			Level:       4,
			RewardXP:    300,
			RewardGold:  200,
			TargetItem:  "magical herb",
			TargetCount: 5,
			GiverName:   "Master Alchemist",
		},
		{
			ID:          7,
			Title:       "Dragon Slayer",
			Description: "Slay the fearsome red dragon that has been burning villages.",
			Type:        QuestTypeKill,
			Level:       10,
			RewardXP:    2000,
			RewardGold:  1000,
			TargetMob:   "red dragon",
			TargetCount: 1,
			GiverName:   "King",
		},
	}

	for _, quest := range quests {
		qs.quests[quest.ID] = quest
	}
}

// GetQuest returns a quest by ID
func (qs *QuestSystem) GetQuest(id int) *Quest {
	return qs.quests[id]
}

// GetAllQuests returns all available quests
func (qs *QuestSystem) GetAllQuests() map[int]*Quest {
	return qs.quests
}

// GetAvailableQuests returns quests available to a character
func (qs *QuestSystem) GetAvailableQuests(ch *types.Character) []*Quest {
	var available []*Quest
	for _, quest := range qs.quests {
		if ch.Level >= quest.Level && !qs.HasCompletedQuest(ch, quest.ID) {
			available = append(available, quest)
		}
	}
	return available
}

// GetPlayerQuests returns all quests for a player
func (qs *QuestSystem) GetPlayerQuests(ch *types.Character) []*PlayerQuest {
	if ch.PCData == nil {
		return nil
	}
	return qs.playerQuests[ch.Name]
}

// StartQuest starts a quest for a player
func (qs *QuestSystem) StartQuest(ch *types.Character, questID int) bool {
	if ch.PCData == nil {
		return false
	}

	quest := qs.GetQuest(questID)
	if quest == nil {
		return false
	}

	// Check if already has this quest
	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && !pq.Completed {
			return false // Already active
		}
	}

	// Check level requirement
	if ch.Level < quest.Level {
		return false
	}

	// Check if already completed
	if qs.HasCompletedQuest(ch, questID) {
		return false
	}

	// Start the quest
	playerQuest := &PlayerQuest{
		QuestID:   questID,
		StartedAt: time.Now(),
		Progress:  0,
		Completed: false,
	}

	qs.playerQuests[ch.Name] = append(qs.playerQuests[ch.Name], playerQuest)
	return true
}

// UpdateQuestProgress updates progress on a quest
func (qs *QuestSystem) UpdateQuestProgress(ch *types.Character, questID int, progress int) {
	if ch.PCData == nil {
		return
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && !pq.Completed {
			pq.Progress += progress

			quest := qs.GetQuest(questID)
			if quest != nil && pq.Progress >= quest.TargetCount {
				qs.CompleteQuest(ch, questID)
			}
			return
		}
	}
}

// CompleteQuest marks a quest as completed and gives rewards
func (qs *QuestSystem) CompleteQuest(ch *types.Character, questID int) bool {
	if ch.PCData == nil {
		return false
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && !pq.Completed {
			pq.Completed = true
			pq.CompletedAt = time.Now()

			// Give rewards
			quest := qs.GetQuest(questID)
			if quest != nil {
				ch.Exp += quest.RewardXP
				ch.Gold += quest.RewardGold
			}

			return true
		}
	}
	return false
}

// HasCompletedQuest checks if a player has completed a quest
func (qs *QuestSystem) HasCompletedQuest(ch *types.Character, questID int) bool {
	if ch.PCData == nil {
		return false
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && pq.Completed {
			return true
		}
	}
	return false
}

// IsOnQuest checks if a player is currently on a specific quest
func (qs *QuestSystem) IsOnQuest(ch *types.Character, questID int) bool {
	if ch.PCData == nil {
		return false
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && !pq.Completed {
			return true
		}
	}
	return false
}

// GetQuestProgress returns current progress on a quest
func (qs *QuestSystem) GetQuestProgress(ch *types.Character, questID int) int {
	if ch.PCData == nil {
		return 0
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.QuestID == questID && !pq.Completed {
			return pq.Progress
		}
	}
	return 0
}

// AbandonQuest removes an active quest from the player's quest log
func (qs *QuestSystem) AbandonQuest(ch *types.Character, questID int) bool {
	if ch.PCData == nil {
		return false
	}

	quests := qs.playerQuests[ch.Name]
	for i, pq := range quests {
		if pq.QuestID == questID && !pq.Completed {
			// Remove the quest from the slice
			qs.playerQuests[ch.Name] = append(quests[:i], quests[i+1:]...)
			return true
		}
	}
	return false
}

// OnMobKill is called when a player kills a mob to check for quest progress
// Returns true if any quest was updated
func (qs *QuestSystem) OnMobKill(ch *types.Character, victim *types.Character) bool {
	if ch.PCData == nil || victim == nil {
		return false
	}

	updated := false
	victimName := strings.ToLower(victim.ShortDesc)
	if victimName == "" {
		victimName = strings.ToLower(victim.Name)
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.Completed {
			continue
		}

		quest := qs.GetQuest(pq.QuestID)
		if quest == nil || quest.Type != QuestTypeKill {
			continue
		}

		// Check if the killed mob matches the quest target
		targetName := strings.ToLower(quest.TargetMob)
		if strings.Contains(victimName, targetName) || strings.Contains(strings.ToLower(victim.Name), targetName) {
			pq.Progress++
			updated = true

			// Check if quest is now complete
			if pq.Progress >= quest.TargetCount {
				pq.Completed = true
				pq.CompletedAt = time.Now()
			}
		}
	}

	return updated
}

// OnItemGet is called when a player gets an item to check for collect quest progress
// Returns true if any quest was updated
func (qs *QuestSystem) OnItemGet(ch *types.Character, item *types.Object) bool {
	if ch.PCData == nil || item == nil {
		return false
	}

	updated := false
	itemName := strings.ToLower(item.ShortDesc)
	if itemName == "" {
		itemName = strings.ToLower(item.Name)
	}

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.Completed {
			continue
		}

		quest := qs.GetQuest(pq.QuestID)
		if quest == nil || quest.Type != QuestTypeCollect {
			continue
		}

		// Check if the item matches the quest target
		targetItem := strings.ToLower(quest.TargetItem)
		if strings.Contains(itemName, targetItem) || strings.Contains(strings.ToLower(item.Name), targetItem) {
			pq.Progress++
			updated = true

			// Check if quest is now complete
			if pq.Progress >= quest.TargetCount {
				pq.Completed = true
				pq.CompletedAt = time.Now()
			}
		}
	}

	return updated
}

// OnRoomEnter is called when a player enters a room to check for explore quest progress
// Returns true if any quest was updated
func (qs *QuestSystem) OnRoomEnter(ch *types.Character, room *types.Room) bool {
	if ch.PCData == nil || room == nil {
		return false
	}

	updated := false

	for _, pq := range qs.playerQuests[ch.Name] {
		if pq.Completed {
			continue
		}

		quest := qs.GetQuest(pq.QuestID)
		if quest == nil || quest.Type != QuestTypeExplore {
			continue
		}

		// Check if this is the target room
		if room.Vnum == quest.TargetRoom && pq.Progress == 0 {
			pq.Progress = 1
			pq.Completed = true
			pq.CompletedAt = time.Now()
			updated = true
		}
	}

	return updated
}
