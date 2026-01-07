package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"rotmud/pkg/types"
)

// PlayerSave represents the JSON structure for player saves
type PlayerSave struct {
	Name      string `json:"name"`
	Level     int    `json:"level"`
	Class     int    `json:"class"`
	Race      int    `json:"race"`
	Sex       int    `json:"sex"`
	Alignment int    `json:"alignment"`

	// Vitals
	Hit     int `json:"hp"`
	MaxHit  int `json:"max_hp"`
	Mana    int `json:"mana"`
	MaxMana int `json:"max_mana"`
	Move    int `json:"move"`
	MaxMove int `json:"max_move"`

	// Stats
	Stats [5]int `json:"stats"`

	// Money
	Gold     int `json:"gold"`
	Silver   int `json:"silver"`
	Platinum int `json:"platinum"`
	BankGold int `json:"bank_gold,omitempty"`

	// Experience
	Exp   int `json:"exp"`
	Trust int `json:"trust"`

	// Play time
	Played int `json:"played"`

	// Position and location
	RoomVnum int `json:"room_vnum"`

	// Combat bonuses
	Hitroll int `json:"hitroll"`
	Damroll int `json:"damroll"`

	// Skills
	Skills map[string]int `json:"skills,omitempty"`

	// PC-specific data
	Password string `json:"password,omitempty"`
	Title    string `json:"title,omitempty"`
	Prompt   string `json:"prompt,omitempty"`
	Bamfin   string `json:"bamfin,omitempty"`
	Bamfout  string `json:"bamfout,omitempty"`

	// Clan and quest data
	Clan          int               `json:"clan,omitempty"`
	QuestProgress map[int]int       `json:"quest_progress,omitempty"`
	ForgetList    []string          `json:"forget_list,omitempty"`
	Aliases       map[string]string `json:"aliases,omitempty"`

	// Conditions (hunger, thirst, etc.)
	Conditions [4]int `json:"conditions,omitempty"`

	// Player flags (autoloot, autogold, autosac, etc.)
	PlayerAct uint64 `json:"player_act,omitempty"`

	// Training/Practice
	Train           int `json:"train,omitempty"`
	Practice        int `json:"practice,omitempty"`
	OverspentPoints int `json:"overspent_points,omitempty"`

	// Inventory and equipment
	Inventory []ObjectSave          `json:"inventory,omitempty"`
	Equipment map[string]ObjectSave `json:"equipment,omitempty"`

	// Affects
	Affects []AffectSave `json:"affects,omitempty"`
}

// ObjectSave represents a saved object
type ObjectSave struct {
	Vnum       int    `json:"vnum"`
	Name       string `json:"name,omitempty"`        // Keywords (if modified from template)
	ShortDesc  string `json:"short_desc,omitempty"`  // Short description
	LongDesc   string `json:"long_desc,omitempty"`   // Long description (on ground)
	Level      int    `json:"level,omitempty"`       // Object level
	Condition  int    `json:"condition,omitempty"`   // Condition 0-100
	Timer      int    `json:"timer,omitempty"`       // Decay timer (-1 = none)
	Enchanted  bool   `json:"enchanted,omitempty"`   // Has been enchanted
	ExtraFlags int    `json:"extra_flags,omitempty"` // Extra item flags
	WearFlags  int    `json:"wear_flags,omitempty"`  // Wear location flags
	WearLoc    int    `json:"wear_loc,omitempty"`    // Current wear location (for equipment)
	ItemType   int    `json:"item_type,omitempty"`   // Item type
	Weight     int    `json:"weight,omitempty"`      // Weight in pounds
	Cost       int    `json:"cost,omitempty"`        // Base value in gold
	Material   string `json:"material,omitempty"`    // Material type
	Owner      string `json:"owner,omitempty"`       // Player owner (for quest items)
	Values     [5]int `json:"values,omitempty"`      // Type-specific values

	// Object affects
	Affects []ObjectAffectSave `json:"affects,omitempty"`

	// Container contents (recursive for nested containers)
	Contents []ObjectSave `json:"contents,omitempty"`
}

// ObjectAffectSave represents an affect on an object
type ObjectAffectSave struct {
	Type      string `json:"type"`
	Level     int    `json:"level"`
	Duration  int    `json:"duration"`
	Location  int    `json:"location"`
	Modifier  int    `json:"modifier"`
	BitVector uint64 `json:"bitvector"`
}

// AffectSave represents a saved affect
type AffectSave struct {
	Type      string `json:"type"`
	Level     int    `json:"level"`
	Duration  int    `json:"duration"`
	Location  int    `json:"location"`
	Modifier  int    `json:"modifier"`
	BitVector uint64 `json:"bitvector"`
}

// PlayerPersistence handles saving and loading players
type PlayerPersistence struct {
	BasePath string
}

// NewPlayerPersistence creates a new persistence handler
func NewPlayerPersistence(basePath string) *PlayerPersistence {
	return &PlayerPersistence{BasePath: basePath}
}

// SavePlayer saves a character to disk
func (p *PlayerPersistence) SavePlayer(ch *types.Character) error {
	save := p.characterToSave(ch)

	data, err := json.MarshalIndent(save, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal player: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(p.BasePath, 0755); err != nil {
		return fmt.Errorf("create players directory: %w", err)
	}

	path := p.playerPath(ch.Name)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write player file: %w", err)
	}

	return nil
}

// LoadPlayer loads a character from disk
func (p *PlayerPersistence) LoadPlayer(name string) (*types.Character, error) {
	path := p.playerPath(name)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read player file: %w", err)
	}

	var save PlayerSave
	if err := json.Unmarshal(data, &save); err != nil {
		return nil, fmt.Errorf("unmarshal player: %w", err)
	}

	return p.saveToCharacter(&save), nil
}

// Exists checks if a player file exists
func (p *PlayerPersistence) Exists(name string) bool {
	path := p.playerPath(name)
	_, err := os.Stat(path)
	return err == nil
}

// Delete removes a player file
func (p *PlayerPersistence) Delete(name string) error {
	path := p.playerPath(name)
	return os.Remove(path)
}

// playerPath returns the file path for a player
func (p *PlayerPersistence) playerPath(name string) string {
	// Sanitize name - capitalize first letter
	if name == "" {
		return filepath.Join(p.BasePath, "unknown.json")
	}
	lower := strings.ToLower(name)
	safeName := strings.ToUpper(lower[:1]) + lower[1:]
	return filepath.Join(p.BasePath, safeName+".json")
}

// characterToSave converts a Character to a PlayerSave
func (p *PlayerPersistence) characterToSave(ch *types.Character) *PlayerSave {
	save := &PlayerSave{
		Name:      ch.Name,
		Level:     ch.Level,
		Class:     ch.Class,
		Race:      ch.Race,
		Sex:       int(ch.Sex),
		Alignment: ch.Alignment,
		Hit:       ch.Hit,
		MaxHit:    ch.MaxHit,
		Mana:      ch.Mana,
		MaxMana:   ch.MaxMana,
		Move:      ch.Move,
		MaxMove:   ch.MaxMove,
		Stats:     ch.PermStats,
		Gold:      int(ch.Gold),
		Silver:    int(ch.Silver),
		Platinum:  int(ch.Platinum),
		Exp:       int(ch.Exp),
		Trust:     ch.Trust,
		Played:    ch.Played,
		Hitroll:   ch.HitRoll,
		Damroll:   ch.DamRoll,
	}

	// Save room vnum
	if ch.InRoom != nil {
		save.RoomVnum = ch.InRoom.Vnum
	}

	// Save prompt
	save.Prompt = ch.Prompt

	// Save skills from PCData
	if ch.PCData != nil {
		save.Password = ch.PCData.Password
		save.Skills = ch.PCData.Learned
		save.Title = ch.PCData.Title
		save.Bamfin = ch.PCData.Bamfin
		save.Bamfout = ch.PCData.Bamfout
		save.Clan = ch.PCData.Clan
		save.BankGold = ch.PCData.BankGold
		save.QuestProgress = ch.PCData.QuestProgress
		save.ForgetList = ch.PCData.ForgetList
		save.Aliases = ch.PCData.Aliases
		save.Conditions = ch.PCData.Condition
	}

	// Save train/practice
	save.Train = ch.Train
	save.Practice = ch.Practice

	// Save player flags (autoloot, autogold, autosac, etc.)
	save.PlayerAct = uint64(ch.PlayerAct)

	// Save overspent points (affects XP per level)
	if ch.PCData != nil {
		save.OverspentPoints = ch.PCData.OverspentPoints
	}

	// Save inventory
	for _, obj := range ch.Inventory {
		save.Inventory = append(save.Inventory, p.objectToSave(obj))
	}

	// Save equipment
	save.Equipment = make(map[string]ObjectSave)
	for loc := types.WearLocation(0); loc < types.WearLocMax; loc++ {
		obj := ch.GetEquipment(loc)
		if obj != nil {
			locName := wearLocationName(loc)
			save.Equipment[locName] = p.objectToSave(obj)
		}
	}

	// Save affects
	for _, aff := range ch.Affected.All() {
		save.Affects = append(save.Affects, AffectSave{
			Type:      aff.Type,
			Level:     aff.Level,
			Duration:  aff.Duration,
			Location:  int(aff.Location),
			Modifier:  aff.Modifier,
			BitVector: uint64(aff.BitVector),
		})
	}

	return save
}

// objectToSave converts an Object to an ObjectSave
func (p *PlayerPersistence) objectToSave(obj *types.Object) ObjectSave {
	save := ObjectSave{
		Vnum:       obj.Vnum,
		Name:       obj.Name,
		ShortDesc:  obj.ShortDesc,
		LongDesc:   obj.LongDesc,
		Level:      obj.Level,
		Condition:  obj.Condition,
		Timer:      obj.Timer,
		Enchanted:  obj.Enchanted,
		ExtraFlags: int(obj.ExtraFlags),
		WearFlags:  int(obj.WearFlags),
		WearLoc:    int(obj.WearLoc),
		ItemType:   int(obj.ItemType),
		Weight:     obj.Weight,
		Cost:       obj.Cost,
		Material:   obj.Material,
		Owner:      obj.Owner,
		Values:     obj.Values,
	}

	// Save object affects
	for _, aff := range obj.Affects.All() {
		save.Affects = append(save.Affects, ObjectAffectSave{
			Type:      aff.Type,
			Level:     aff.Level,
			Duration:  aff.Duration,
			Location:  int(aff.Location),
			Modifier:  aff.Modifier,
			BitVector: uint64(aff.BitVector),
		})
	}

	// Save container contents recursively (handles nested containers)
	for _, contained := range obj.Contents {
		save.Contents = append(save.Contents, p.objectToSave(contained))
	}

	return save
}

// saveToCharacter converts a PlayerSave to a Character
func (p *PlayerPersistence) saveToCharacter(save *PlayerSave) *types.Character {
	ch := types.NewCharacter(save.Name)

	ch.Level = save.Level
	ch.Class = save.Class
	ch.Race = save.Race
	ch.Sex = types.Sex(save.Sex)
	ch.Alignment = save.Alignment
	ch.Hit = save.Hit
	ch.MaxHit = save.MaxHit
	ch.Mana = save.Mana
	ch.MaxMana = save.MaxMana
	ch.Move = save.Move
	ch.MaxMove = save.MaxMove
	ch.PermStats = save.Stats
	ch.Gold = save.Gold
	ch.Silver = save.Silver
	ch.Platinum = save.Platinum
	ch.Exp = int(save.Exp)
	ch.Trust = save.Trust
	ch.Played = save.Played
	ch.HitRoll = save.Hitroll
	ch.DamRoll = save.Damroll

	// Migration: fix old characters with 0 hitroll/damroll
	// Set base values based on class if they're missing
	if ch.HitRoll == 0 && ch.DamRoll == 0 {
		switch ch.Class {
		case types.ClassWarrior:
			ch.HitRoll = 5
			ch.DamRoll = 5
		case types.ClassRanger:
			ch.HitRoll = 4
			ch.DamRoll = 4
		case types.ClassThief:
			ch.HitRoll = 3
			ch.DamRoll = 3
		default:
			ch.HitRoll = 2
			ch.DamRoll = 2
		}
	}

	// Restore PCData
	if save.Password != "" || save.Skills != nil || save.Title != "" {
		ch.PCData = &types.PCData{
			Password:      save.Password,
			Learned:       save.Skills,
			Title:         save.Title,
			Bamfin:        save.Bamfin,
			Bamfout:       save.Bamfout,
			Clan:          save.Clan,
			BankGold:      save.BankGold,
			QuestProgress: save.QuestProgress,
			ForgetList:    save.ForgetList,
			Aliases:       save.Aliases,
			Condition:     save.Conditions,
		}
	}

	// Restore train/practice
	ch.Train = save.Train
	ch.Practice = save.Practice

	// Restore player flags (autoloot, autogold, autosac, etc.)
	ch.PlayerAct = types.PlayerFlags(save.PlayerAct)

	// Restore overspent points and saved room
	if ch.PCData != nil {
		ch.PCData.OverspentPoints = save.OverspentPoints
		ch.PCData.SavedRoom = save.RoomVnum
	}

	// Restore prompt
	ch.Prompt = save.Prompt

	// Restore inventory
	for _, objSave := range save.Inventory {
		obj := p.saveToObject(&objSave)
		ch.AddInventory(obj)
	}

	// Restore equipment
	for locName, objSave := range save.Equipment {
		loc := parseWearLocation(locName)
		if loc != types.WearLocNone {
			obj := p.saveToObject(&objSave)
			ch.Equip(obj, loc)
		}
	}

	// Restore affects
	for _, affSave := range save.Affects {
		aff := types.NewAffect(
			affSave.Type,
			affSave.Level,
			affSave.Duration,
			types.ApplyType(affSave.Location),
			affSave.Modifier,
			types.AffectFlags(affSave.BitVector),
		)
		ch.AddAffect(aff)
	}

	return ch
}

// saveToObject converts an ObjectSave to an Object
func (p *PlayerPersistence) saveToObject(save *ObjectSave) *types.Object {
	obj := types.NewObject(save.Vnum, save.ShortDesc, types.ItemType(save.ItemType))

	// Restore all saved fields
	if save.Name != "" {
		obj.Name = save.Name
	}
	if save.LongDesc != "" {
		obj.LongDesc = save.LongDesc
	}
	obj.Level = save.Level
	obj.Condition = save.Condition
	obj.Timer = save.Timer
	obj.Enchanted = save.Enchanted
	obj.ExtraFlags = types.ItemFlags(save.ExtraFlags)
	obj.WearFlags = types.WearFlags(save.WearFlags)
	obj.WearLoc = types.WearLocation(save.WearLoc)
	obj.Weight = save.Weight
	obj.Cost = save.Cost
	if save.Material != "" {
		obj.Material = save.Material
	}
	if save.Owner != "" {
		obj.Owner = save.Owner
	}
	obj.Values = save.Values

	// Restore object affects
	for _, affSave := range save.Affects {
		aff := types.NewAffect(
			affSave.Type,
			affSave.Level,
			affSave.Duration,
			types.ApplyType(affSave.Location),
			affSave.Modifier,
			types.AffectFlags(affSave.BitVector),
		)
		obj.Affects.Add(aff)
	}

	// Restore container contents recursively (handles nested containers)
	for _, contained := range save.Contents {
		contentObj := p.saveToObject(&contained)
		obj.AddContent(contentObj)
	}

	return obj
}

// Helper functions

func wearLocationName(loc types.WearLocation) string {
	names := map[types.WearLocation]string{
		types.WearLocLight:     "light",
		types.WearLocFingerL:   "finger_l",
		types.WearLocFingerR:   "finger_r",
		types.WearLocNeck1:     "neck_1",
		types.WearLocNeck2:     "neck_2",
		types.WearLocBody:      "body",
		types.WearLocHead:      "head",
		types.WearLocLegs:      "legs",
		types.WearLocFeet:      "feet",
		types.WearLocHands:     "hands",
		types.WearLocArms:      "arms",
		types.WearLocShield:    "shield",
		types.WearLocAbout:     "about",
		types.WearLocWaist:     "waist",
		types.WearLocWristL:    "wrist_l",
		types.WearLocWristR:    "wrist_r",
		types.WearLocWield:     "wield",
		types.WearLocHold:      "hold",
		types.WearLocFloat:     "float",
		types.WearLocSecondary: "secondary",
		types.WearLocFace:      "face",
	}
	if name, ok := names[loc]; ok {
		return name
	}
	return ""
}

func parseWearLocation(name string) types.WearLocation {
	locations := map[string]types.WearLocation{
		"light":     types.WearLocLight,
		"finger_l":  types.WearLocFingerL,
		"finger_r":  types.WearLocFingerR,
		"neck_1":    types.WearLocNeck1,
		"neck_2":    types.WearLocNeck2,
		"body":      types.WearLocBody,
		"head":      types.WearLocHead,
		"legs":      types.WearLocLegs,
		"feet":      types.WearLocFeet,
		"hands":     types.WearLocHands,
		"arms":      types.WearLocArms,
		"shield":    types.WearLocShield,
		"about":     types.WearLocAbout,
		"waist":     types.WearLocWaist,
		"wrist_l":   types.WearLocWristL,
		"wrist_r":   types.WearLocWristR,
		"wield":     types.WearLocWield,
		"hold":      types.WearLocHold,
		"float":     types.WearLocFloat,
		"secondary": types.WearLocSecondary,
		"face":      types.WearLocFace,
	}
	if loc, ok := locations[name]; ok {
		return loc
	}
	return types.WearLocNone
}
