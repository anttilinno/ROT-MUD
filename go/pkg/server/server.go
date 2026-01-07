package server

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"
	"time"

	"rotmud/pkg/ai"
	"rotmud/pkg/combat"
	"rotmud/pkg/game"
	"rotmud/pkg/help"
	"rotmud/pkg/loader"
	"rotmud/pkg/magic"
	"rotmud/pkg/persistence"
	"rotmud/pkg/types"
)

// Room vnums for starting locations
const (
	RoomVnumMudSchool = 3700 // New characters start here
	RoomVnumTemple    = 3001 // Default recall/respawn point
)

// Session represents a connected client
type Session struct {
	conn       net.Conn
	reader     *bufio.Reader
	writer     *bufio.Writer
	Character  *types.Character
	Descriptor *types.Descriptor
	mu         sync.Mutex
}

// MetricsUpdaterFunc is a callback for updating metrics
type MetricsUpdaterFunc func(playerCount, npcCount, connCount int, commandCount int64)

// OnCommandFunc is a callback for when a command is processed
type OnCommandFunc func()

// Server handles TCP connections for the MUD
type Server struct {
	listener   net.Listener
	sessions   map[net.Conn]*Session
	wsSessions map[*types.Character]*WebSocketSession
	mu         sync.RWMutex
	logger     *slog.Logger
	GameLoop   *game.GameLoop
	Dispatcher *game.CommandDispatcher
	Clans      *game.ClanSystem
	Quests     *game.QuestSystem
	World      *loader.World
	DataPath   string // Path to data directory

	// Player persistence
	Persistence *persistence.PlayerPersistence
	Login       *LoginHandler

	// Statistics
	startTime    time.Time
	commandCount int64

	// Security
	apiKey string

	// Metrics
	metricsUpdater MetricsUpdaterFunc
	onCommand      OnCommandFunc

	// Shutdown control
	shutdownCh chan bool // true = reboot, false = shutdown
}

// New creates a new Server instance
func New(logger *slog.Logger) *Server {
	s := &Server{
		sessions:    make(map[net.Conn]*Session),
		wsSessions:  make(map[*types.Character]*WebSocketSession),
		logger:      logger,
		GameLoop:    game.NewGameLoop(),
		Dispatcher:  game.NewCommandDispatcher(),
		Clans:       game.NewClanSystem(),
		Quests:      game.NewQuestSystem(),
		Persistence: persistence.NewPlayerPersistence("data/players"),
		startTime:   time.Now(),
		apiKey:      "changeme", // TODO: Load from config
	}

	// Initialize login handler
	s.Login = NewLoginHandler(s.Persistence)
	s.Login.Output = func(session *Session, msg string) {
		session.Write(msg)
	}

	// Wire up command processing
	s.GameLoop.OnCommand = func(cmd game.Command) {
		s.commandCount++
		if s.onCommand != nil {
			s.onCommand()
		}
		s.Dispatcher.Dispatch(cmd)

		// Send prompt after command completes
		if cmd.Character != nil && !cmd.Character.IsNPC() {
			prompt := game.FormatPrompt(cmd.Character)
			if prompt == "" {
				prompt = "> "
			}
			s.SendToCharacter(cmd.Character, "\r\n"+prompt)
		}
	}

	// Wire up output - find the session for a character and send output
	s.Dispatcher.Output = func(ch *types.Character, msg string) {
		s.SendToCharacter(ch, msg)
	}

	// Give dispatcher access to game loop for character lookups (tell, etc.)
	s.Dispatcher.GameLoop = s.GameLoop

	// Wire up combat system output
	s.Dispatcher.Combat.Output = func(ch *types.Character, msg string) {
		s.SendToCharacter(ch, msg)
	}

	// Wire up violence update to combat system
	s.GameLoop.OnViolence = func() {
		s.Dispatcher.Combat.ViolenceUpdate(s.GameLoop.Characters)

		// Send combat prompts to all fighting players
		for _, ch := range s.GameLoop.Characters {
			if ch.Fighting != nil && !ch.IsNPC() {
				// Send combat status line
				combatPrompt := game.FormatCombatPrompt(ch)
				if combatPrompt != "" {
					s.SendToCharacter(ch, combatPrompt)
				}

				// Send regular prompt
				prompt := game.FormatPrompt(ch)
				if prompt == "" {
					prompt = "> "
				}
				s.SendToCharacter(ch, prompt)
			}
		}
	}

	// Wire up room finder for death respawn
	s.Dispatcher.Combat.RoomFinder = func(vnum int) *types.Room {
		if s.World != nil {
			return s.World.GetRoom(vnum)
		}
		return nil
	}

	// Wire up character mover for death respawn
	s.Dispatcher.Combat.CharMover = func(ch *types.Character, room *types.Room) {
		game.CharToRoom(ch, room)
	}

	// Wire up skill getter for combat passive skills (parry, dodge, etc.)
	s.Dispatcher.Combat.SkillGetter = func(ch *types.Character, skillName string) int {
		return s.Dispatcher.Skills.GetSkill(ch, skillName)
	}

	// Wire up level up callback to save character when they reach level 2
	s.Dispatcher.Combat.OnLevelUp = func(ch *types.Character, oldLevel, newLevel int) {
		// Save when reaching level 2 (first save for new characters)
		if oldLevel == 1 && newLevel >= 2 && ch.PCData != nil {
			if err := s.Persistence.SavePlayer(ch); err != nil {
				s.logger.Error("failed to save player on level up", "error", err, "name", ch.Name)
			} else {
				s.logger.Info("saved player on reaching level 2", "name", ch.Name)
				s.SendToCharacter(ch, "Your character has been saved for the first time!\r\n")
			}
		}
	}

	// Wire up death callback for autoloot/autosac
	s.Dispatcher.Combat.OnDeath = func(killer, victim *types.Character, corpse *types.Object) {
		s.handleAutolootAutosac(killer, victim, corpse)
	}

	// Wire up skills system in game loop for passive regen bonuses
	s.GameLoop.Skills = s.Dispatcher.Skills
	s.GameLoop.Combat = s.Dispatcher.Combat

	// Wire up area reset system - use the game's ResetSystem, not our own
	// The ResetSystem is already wired in GameLoop.Resets, so we don't need OnAreaReset
	s.GameLoop.OnAreaReset = nil

	// Wire up magic system output
	s.Dispatcher.Magic.Output = func(ch *types.Character, msg string) {
		s.SendToCharacter(ch, msg)
	}

	// Wire up object finder for locate object spell
	s.Dispatcher.Magic.ObjectFinder = func(name string, level int) []*types.Object {
		return s.findObjectsInWorld(name, level)
	}

	// Wire up object placement for create food/spring spells
	s.Dispatcher.Magic.ObjectToRoom = func(obj *types.Object, room *types.Room) {
		game.ObjToRoom(obj, room)
	}

	// Wire up weather control for control weather spell
	s.Dispatcher.Magic.WeatherControl = func(change int) {
		if s.GameLoop.WorldTime != nil {
			s.GameLoop.WorldTime.ControlWeather(change)
		}
	}

	// Wire up character movement for teleport/summon/gate spells
	s.Dispatcher.Magic.CharacterToRoom = func(ch *types.Character, room *types.Room) {
		game.CharToRoom(ch, room)
	}

	// Wire up room finder for teleport/summon/gate spells
	s.Dispatcher.Magic.RoomFinder = func(vnum int) *types.Room {
		if s.World != nil {
			return s.World.GetRoom(vnum)
		}
		return nil
	}

	// Wire up portal creator for portal/nexus spells
	s.Dispatcher.Magic.PortalCreator = func(srcRoom, destRoom *types.Room, duration int, twoWay bool) *types.Object {
		return s.createPortal(srcRoom, destRoom, duration, twoWay)
	}

	// Wire up pet creator for animate/resurrect/conjure spells
	s.Dispatcher.Magic.PetCreator = func(master *types.Character, petType string, sourceObj *types.Object) *types.Character {
		return s.createPet(master, petType, sourceObj)
	}

	// Wire up corpse finder for resurrect spell
	s.Dispatcher.Magic.CorpseFinder = func(ch *types.Character) *types.Object {
		return s.findCorpseInRoom(ch)
	}

	// Wire up object extraction for removing consumed objects
	s.Dispatcher.Magic.ExtractObject = func(obj *types.Object) {
		s.extractObject(obj)
	}

	// Wire up magic system in game loop for affect processing
	s.GameLoop.Magic = s.Dispatcher.Magic
	s.GameLoop.Output = func(ch *types.Character, msg string) {
		s.SendToCharacter(ch, msg)
	}

	// Wire up AI system for NPC specials (like spec_cast_adept)
	s.GameLoop.AI = ai.NewAISystem()
	s.GameLoop.AI.Magic = s.Dispatcher.Magic
	s.GameLoop.AI.Output = func(ch *types.Character, msg string) {
		s.SendToCharacter(ch, msg)
	}
	s.GameLoop.AI.ActToRoom = func(msg string, ch, victim *types.Character, output func(ch *types.Character, msg string)) {
		// Wrapper around game.ActToRoom (doesn't support object parameter from AI)
		game.ActToRoom(msg, ch, victim, nil, output)
	}
	s.GameLoop.AI.StartCombat = func(ch, victim *types.Character) {
		combat.SetFighting(ch, victim)
		combat.SetFighting(victim, ch)
	}
	s.GameLoop.AI.MoveChar = func(ch *types.Character, dir types.Direction) {
		// Move NPC in the direction
		if ch.InRoom == nil {
			return
		}
		exit := ch.InRoom.GetExit(dir)
		if exit == nil || exit.ToRoom == nil || exit.IsClosed() {
			return
		}
		// Announce departure
		game.ActToRoom("$n leaves "+dir.String()+".", ch, nil, nil, func(c *types.Character, m string) {
			s.SendToCharacter(c, m)
		})
		// Move to new room
		game.CharToRoom(ch, exit.ToRoom)
		// Announce arrival
		game.ActToRoom("$n has arrived.", ch, nil, nil, func(c *types.Character, m string) {
			s.SendToCharacter(c, m)
		})
	}

	// Wire up clan system
	s.Dispatcher.Clans = s.Clans

	// Wire up quest system
	s.Dispatcher.Quests = s.Quests

	// MOBprogs are already initialized in the dispatcher

	// Create shutdown channel and wire up OnShutdown
	s.shutdownCh = make(chan bool, 1)
	s.Dispatcher.OnShutdown = func(reboot bool) {
		select {
		case s.shutdownCh <- reboot:
		default:
			// Channel already has a shutdown signal
		}
	}

	// Wire up callback for removing NPCs from game loop (used by dismiss command)
	s.Dispatcher.OnRemoveCharacter = func(ch *types.Character) {
		s.GameLoop.RemoveCharacter(ch)
	}

	// Wire up save callback
	s.Dispatcher.OnSave = func(ch *types.Character) error {
		return s.Persistence.SavePlayer(ch)
	}

	// Wire up quit callback
	s.Dispatcher.OnQuit = func(ch *types.Character) {
		// Save and disconnect the player
		if ch.PCData != nil && ch.Level > 1 {
			s.Persistence.SavePlayer(ch)
		}
		// Remove from game and close connection
		s.dismissAllFollowers(ch)
		s.GameLoop.RemoveCharacter(ch)
		if ch.InRoom != nil {
			ch.InRoom.RemovePerson(ch)
		}
		// Close the connection - find the session
		s.mu.Lock()
		for conn, sess := range s.sessions {
			if sess.Character == ch {
				conn.Close()
				delete(s.sessions, conn)
				break
			}
		}
		s.mu.Unlock()
		// Also check websocket sessions
		s.mu.Lock()
		if ws, ok := s.wsSessions[ch]; ok {
			delete(s.wsSessions, ch)
			if ws.conn != nil {
				ws.conn.Close()
			}
		}
		s.mu.Unlock()
	}

	// Wire up delete callback
	s.Dispatcher.OnDelete = func(ch *types.Character) error {
		// Delete the player file
		if err := s.Persistence.Delete(ch.Name); err != nil {
			return err
		}
		// Mark as deleted so defer won't save
		ch.Deleted = true
		s.logger.Info("character deleted", "name", ch.Name)
		return nil
	}

	// Wire up disconnect callback (disconnect without saving)
	s.Dispatcher.DisconnectPlayer = func(ch *types.Character) {
		// Remove from game without saving
		s.dismissAllFollowers(ch)
		s.GameLoop.RemoveCharacter(ch)
		if ch.InRoom != nil {
			ch.InRoom.RemovePerson(ch)
		}
		// Close the connection - find the session
		s.mu.Lock()
		for conn, sess := range s.sessions {
			if sess.Character == ch {
				conn.Close()
				delete(s.sessions, conn)
				break
			}
		}
		s.mu.Unlock()
		// Also check websocket sessions
		s.mu.Lock()
		if ws, ok := s.wsSessions[ch]; ok {
			delete(s.wsSessions, ch)
			if ws.conn != nil {
				ws.conn.Close()
			}
		}
		s.mu.Unlock()
	}

	// Wire up tick update for affects and regeneration
	s.GameLoop.OnTick = func() {
		s.tickUpdate()
	}

	return s
}

// LoadWorld loads the game world from the data directory
func (s *Server) LoadWorld() error {
	if s.DataPath == "" {
		s.DataPath = "data"
	}

	areasPath := s.DataPath + "/areas"
	areaLoader := loader.NewAreaLoader(areasPath)

	world, err := areaLoader.LoadAll()
	if err != nil {
		return fmt.Errorf("load world: %w", err)
	}

	s.World = world
	s.logger.Info("world loaded",
		"areas", len(world.Areas),
		"rooms", len(world.Rooms),
		"mob_templates", len(world.MobTemplates),
		"obj_templates", len(world.ObjTemplates))

	// Copy rooms to game loop
	for vnum, room := range world.Rooms {
		s.GameLoop.Rooms[vnum] = room
	}

	// Initialize the reset system for periodic resets
	s.GameLoop.Resets = game.NewResetSystem(world)
	s.GameLoop.Areas = world.Areas
	s.GameLoop.World = world

	// Load help files
	helpPath := s.DataPath + "/help"
	helpSystem := help.NewSystem()
	if err := helpSystem.LoadDir(helpPath); err != nil {
		s.logger.Warn("failed to load help files", "path", helpPath, "error", err)
	} else {
		s.logger.Info("help system loaded", "entries", helpSystem.Count())
	}
	// Wire up help to dispatcher and login handler
	s.Dispatcher.Help = helpSystem
	s.Login.Help = helpSystem

	// Reset the world (spawn mobs and objects)
	s.resetWorld()

	return nil
}

// resetWorld spawns mobs and objects based on room reset data
func (s *Server) resetWorld() {
	if s.World == nil {
		return
	}

	mobCount := 0
	objCount := 0

	for _, room := range s.World.Rooms {
		// Spawn mobs
		for _, reset := range room.MobResets {
			// Check if mob already exists in this room
			mobExists := false
			for _, ch := range room.People {
				if ch.IsNPC() && ch.MobVnum == reset.Vnum {
					mobExists = true
					break
				}
			}
			if mobExists {
				continue
			}

			// Check max in world
			if reset.Max > 0 {
				count := s.countMobInWorld(reset.Vnum)
				if count >= reset.Max {
					continue
				}
			}

			for i := 0; i < reset.Count; i++ {
				mob := s.World.CreateMobFromTemplate(reset.Vnum)
				if mob == nil {
					s.logger.Warn("mob template not found", "vnum", reset.Vnum)
					continue
				}
				game.CharToRoom(mob, room)
				s.GameLoop.AddCharacter(mob)
				mobCount++

				// Process equipment resets for this mob
				for _, equipReset := range reset.Equips {
					s.equipMob(mob, &equipReset)
				}
			}
		}

		// Spawn objects
		for _, reset := range room.ObjResets {
			// Check if object already exists in this room
			objExists := false
			for _, obj := range room.Objects {
				if obj.Vnum == reset.Vnum {
					objExists = true
					break
				}
			}
			if objExists {
				continue
			}

			// Check max in world
			if reset.Max > 0 {
				count := s.countObjInWorld(reset.Vnum)
				if count >= reset.Max {
					continue
				}
			}

			for i := 0; i < reset.Count; i++ {
				tmpl := s.World.GetObjTemplate(reset.Vnum)
				if tmpl == nil {
					s.logger.Warn("object template not found", "vnum", reset.Vnum)
					continue
				}
				obj := s.createObjectFromTemplate(tmpl)
				game.ObjToRoom(obj, room)
				objCount++
			}
		}
	}

	s.logger.Info("world reset complete", "mobs", mobCount, "objects", objCount)
}

// equipMob gives an object to a mob based on equip reset data
func (s *Server) equipMob(mob *types.Character, reset *types.EquipReset) {
	if s.World == nil {
		return
	}

	// Get the object template
	tmpl := s.World.GetObjTemplate(reset.Vnum)
	if tmpl == nil {
		s.logger.Warn("equip reset: object template not found", "vnum", reset.Vnum)
		return
	}

	// Create the object
	obj := s.createObjectFromTemplate(tmpl)
	if obj == nil {
		return
	}

	// Either put in inventory or equip
	if reset.InvOnly {
		obj.CarriedBy = mob
		mob.Inventory = append(mob.Inventory, obj)
	} else {
		// Equip at the specified wear location
		if reset.WearLoc >= 0 && int(reset.WearLoc) < len(mob.Equipment) {
			mob.Equipment[reset.WearLoc] = obj
			obj.CarriedBy = mob
			obj.WearLoc = reset.WearLoc
		} else {
			// Invalid wear location, put in inventory
			obj.CarriedBy = mob
			mob.Inventory = append(mob.Inventory, obj)
		}
	}
}

// handleAutolootAutosac processes autoloot and autosac after a kill
func (s *Server) handleAutolootAutosac(killer, victim *types.Character, corpse *types.Object) {
	// Only process for player killers and NPC victims
	if killer == nil || killer.IsNPC() || !victim.IsNPC() || corpse == nil {
		return
	}

	// Check for autoloot
	if killer.PlayerAct.Has(types.PlrAutoLoot) {
		// Get all items from corpse
		for len(corpse.Contents) > 0 {
			obj := corpse.Contents[0]
			corpse.RemoveContent(obj)
			game.ObjToChar(obj, killer)
			s.SendToCharacter(killer, fmt.Sprintf("You get %s from %s.\r\n", obj.ShortDesc, corpse.ShortDesc))
		}
	}

	// Check for autosac (only if corpse is empty or autoloot got everything)
	if killer.PlayerAct.Has(types.PlrAutoSac) && len(corpse.Contents) == 0 {
		// Remove corpse from room
		if corpse.InRoom != nil {
			corpse.InRoom.RemoveObject(corpse)
		}
		// Give a small amount of gold as sacrifice reward
		gold := 1
		killer.Gold += gold
		s.SendToCharacter(killer, fmt.Sprintf("The gods give you %d gold coin for your sacrifice.\r\n", gold))
	}
}

// countMobInWorld counts how many mobs of a given vnum exist
func (s *Server) countMobInWorld(vnum int) int {
	count := 0
	for _, ch := range s.GameLoop.Characters {
		if ch.IsNPC() && ch.MobVnum == vnum {
			count++
		}
	}
	return count
}

// countObjInWorld counts how many objects of a given vnum exist
func (s *Server) countObjInWorld(vnum int) int {
	count := 0
	for _, room := range s.World.Rooms {
		for _, obj := range room.Objects {
			if obj.Vnum == vnum {
				count++
			}
		}
	}
	return count
}

// createObjectFromTemplate creates an Object instance from a template
func (s *Server) createObjectFromTemplate(tmpl *loader.ObjectData) *types.Object {
	obj := types.NewObject(tmpl.Vnum, tmpl.ShortDesc, parseItemType(tmpl.ItemType))
	obj.Name = strings.Join(tmpl.Keywords, " ")
	obj.LongDesc = tmpl.LongDesc
	obj.Level = tmpl.Level
	obj.Weight = tmpl.Weight
	obj.Cost = tmpl.Cost

	// Parse wear flags
	for _, flag := range tmpl.WearFlags {
		switch strings.ToLower(flag) {
		case "take":
			obj.WearFlags.Set(types.WearTake)
		case "wield":
			obj.WearFlags.Set(types.WearWield)
		case "hold":
			obj.WearFlags.Set(types.WearHold)
		case "body":
			obj.WearFlags.Set(types.WearBody)
		case "head":
			obj.WearFlags.Set(types.WearHead)
		case "legs":
			obj.WearFlags.Set(types.WearLegs)
		case "feet":
			obj.WearFlags.Set(types.WearFeet)
		case "hands":
			obj.WearFlags.Set(types.WearHands)
		case "arms":
			obj.WearFlags.Set(types.WearArms)
		case "shield":
			obj.WearFlags.Set(types.WearShield)
		case "about":
			obj.WearFlags.Set(types.WearAbout)
		case "waist":
			obj.WearFlags.Set(types.WearWaist)
		case "wrist":
			obj.WearFlags.Set(types.WearWrist)
		case "finger":
			obj.WearFlags.Set(types.WearFinger)
		case "neck":
			obj.WearFlags.Set(types.WearNeck)
		}
	}

	// Parse extra flags
	for _, flag := range tmpl.ExtraFlags {
		switch strings.ToLower(flag) {
		case "glow":
			obj.ExtraFlags.Set(types.ItemGlow)
		case "hum":
			obj.ExtraFlags.Set(types.ItemHum)
		case "magic":
			obj.ExtraFlags.Set(types.ItemMagic)
		case "bless":
			obj.ExtraFlags.Set(types.ItemBless)
		case "nodrop":
			obj.ExtraFlags.Set(types.ItemNoDrop)
		case "noremove":
			obj.ExtraFlags.Set(types.ItemNoRemove)
		}
	}

	// Weapon-specific values
	if tmpl.Weapon != nil {
		obj.Values[0] = parseWeaponType(tmpl.Weapon.WeaponType, tmpl.Keywords)
		obj.Values[1] = tmpl.Weapon.DiceNumber
		obj.Values[2] = tmpl.Weapon.DiceSize
	}

	// Parse affects (hitroll, damroll, stat bonuses, etc.)
	for _, affData := range tmpl.Affects {
		aff := &types.Affect{
			Type:     "equipment",
			Level:    obj.Level,
			Duration: -1, // Permanent while equipped
			Location: parseApplyType(affData.Location),
			Modifier: affData.Modifier,
		}
		obj.Affects.Add(aff)
	}

	return obj
}

// parseApplyType converts a location string to an ApplyType constant
func parseApplyType(s string) types.ApplyType {
	switch strings.ToLower(s) {
	case "str", "strength":
		return types.ApplyStr
	case "dex", "dexterity":
		return types.ApplyDex
	case "int", "intelligence":
		return types.ApplyInt
	case "wis", "wisdom":
		return types.ApplyWis
	case "con", "constitution":
		return types.ApplyCon
	case "mana":
		return types.ApplyMana
	case "hp", "hit", "hitpoints":
		return types.ApplyHit
	case "move", "movement":
		return types.ApplyMove
	case "ac", "armor":
		return types.ApplyAC
	case "hitroll":
		return types.ApplyHitroll
	case "damroll":
		return types.ApplyDamroll
	case "saves", "saving_throw":
		return types.ApplySaves
	}
	return types.ApplyNone
}

// parseItemType converts item type string to ItemType constant
func parseItemType(s string) types.ItemType {
	switch strings.ToLower(s) {
	case "weapon":
		return types.ItemTypeWeapon
	case "armor":
		return types.ItemTypeArmor
	case "scroll":
		return types.ItemTypeScroll
	case "wand":
		return types.ItemTypeWand
	case "staff":
		return types.ItemTypeStaff
	case "potion":
		return types.ItemTypePotion
	case "container":
		return types.ItemTypeContainer
	case "key":
		return types.ItemTypeKey
	case "food":
		return types.ItemTypeFood
	case "money":
		return types.ItemTypeMoney
	case "light":
		return types.ItemTypeLight
	case "fountain":
		return types.ItemTypeFountain
	case "drink", "drink_con":
		return types.ItemTypeDrinkCon
	case "pill":
		return types.ItemTypePill
	case "treasure":
		return types.ItemTypeTreasure
	case "furniture":
		return types.ItemTypeFurniture
	case "trash":
		return types.ItemTypeTrash
	case "boat":
		return types.ItemTypeBoat
	case "jewelry":
		return types.ItemTypeJewelry
	case "clothing":
		return types.ItemTypeClothing
	case "map":
		return types.ItemTypeMap
	case "portal":
		return types.ItemTypePortal
	case "corpse_npc":
		return types.ItemTypeCorpseNPC
	case "corpse_pc":
		return types.ItemTypeCorpsePC
	default:
		return types.ItemTypeTrash
	}
}

// parseWeaponType converts weapon type string or keywords to weapon class constant
// Returns weapon class: 0=exotic, 1=sword, 2=dagger, 3=spear, 4=mace, 5=axe, 6=flail, 7=whip, 8=polearm
func parseWeaponType(weaponType string, keywords []string) int {
	// First check explicit weapon type
	switch strings.ToLower(weaponType) {
	case "sword":
		return 1
	case "dagger":
		return 2
	case "spear":
		return 3
	case "mace":
		return 4
	case "axe":
		return 5
	case "flail":
		return 6
	case "whip":
		return 7
	case "polearm", "glaive":
		return 8
	}

	// If no explicit type, try to infer from keywords
	for _, kw := range keywords {
		switch strings.ToLower(kw) {
		case "sword", "blade", "longsword", "shortsword", "broadsword":
			return 1
		case "dagger", "knife", "dirk", "stiletto":
			return 2
		case "spear", "lance", "javelin", "pike":
			return 3
		case "mace", "club", "hammer", "maul", "morning star", "morningstar":
			return 4
		case "axe", "hatchet", "battleaxe":
			return 5
		case "flail", "nunchaku":
			return 6
		case "whip", "lash", "bullwhip":
			return 7
		case "polearm", "glaive", "halberd", "bardiche", "staff":
			return 8
		}
	}

	// Default to exotic (uses sword skill as fallback in combat code)
	return 0
}

// Start begins listening for connections on the specified port
func (s *Server) Start(port int) error {
	// Load world data
	if err := s.LoadWorld(); err != nil {
		s.logger.Warn("failed to load world, using fallback rooms", "error", err)
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}
	s.listener = listener
	s.logger.Info("server started", "port", port)

	// Start the game loop
	s.GameLoop.Start()
	s.logger.Info("game loop started")

	// Accept connections in a goroutine so we can handle shutdown
	acceptDone := make(chan struct{})
	go func() {
		defer close(acceptDone)
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if we're shutting down
				select {
				case <-s.shutdownCh:
					// Put back the signal for the main loop to handle
					s.shutdownCh <- false
					return
				default:
					s.logger.Error("accept error", "error", err)
					continue
				}
			}
			go s.handleConnection(conn)
		}
	}()

	// Wait for shutdown signal
	reboot := <-s.shutdownCh

	s.logger.Info("shutdown signal received", "reboot", reboot)

	// Close listener to stop accepting new connections
	listener.Close()

	// Wait for accept goroutine to finish
	<-acceptDone

	// Stop the game loop
	s.GameLoop.Stop()

	// Close all existing connections
	s.mu.Lock()
	for conn := range s.sessions {
		conn.Close()
	}
	s.mu.Unlock()

	if reboot {
		s.logger.Info("server ready for restart")
		return fmt.Errorf("reboot requested")
	}

	s.logger.Info("server shutdown complete")
	return nil
}

// handleConnection manages a single client connection
func (s *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	// Create descriptor
	desc := types.NewDescriptor(remoteAddr)

	// Create session
	session := &Session{
		conn:       conn,
		reader:     bufio.NewReader(conn),
		writer:     bufio.NewWriter(conn),
		Descriptor: desc,
	}

	s.mu.Lock()
	s.sessions[conn] = session
	s.mu.Unlock()

	s.logger.Info("new connection", "addr", remoteAddr)

	defer func() {
		// Save character if logged in (but not level 1 - avoid abandoned characters)
		// Don't save if character was deleted
		if session.Character != nil && session.Character.PCData != nil && !session.Character.Deleted {
			// Dismiss any pets/followers before disconnecting
			s.dismissAllFollowers(session.Character)

			if session.Character.Level > 1 {
				if err := s.Persistence.SavePlayer(session.Character); err != nil {
					s.logger.Error("failed to save player on disconnect", "error", err, "name", session.Character.Name)
				}
			} else {
				s.logger.Info("not saving level 1 character", "name", session.Character.Name)
			}
			s.GameLoop.RemoveCharacter(session.Character)
		}

		s.mu.Lock()
		delete(s.sessions, conn)
		s.mu.Unlock()
		conn.Close()
		s.logger.Info("connection closed", "addr", remoteAddr)
	}()

	// Send greeting
	s.sendGreeting(session)

	// Send initial prompt for name
	session.Write("By what name do you wish to be known? ")

	// Login/creation loop
	for desc.State != types.ConPlaying {
		line, err := session.reader.ReadString('\n')
		if err != nil {
			return
		}

		if s.Login.HandleInput(session, line) {
			// Login complete, enter the game
			break
		}
	}

	// Player is now logged in - enter the game
	s.enterGame(session)

	// Send initial prompt
	prompt := game.FormatPrompt(session.Character)
	if prompt == "" {
		prompt = "> "
	}
	session.Write("\r\n" + prompt)

	// Main input loop
	for {
		line, err := session.reader.ReadString('\n')
		if err != nil {
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			// Empty input - just show prompt again
			prompt := game.FormatPrompt(session.Character)
			if prompt == "" {
				prompt = "> "
			}
			session.Write("\r\n" + prompt)
			continue
		}

		// Handle quit command specially (bypasses game loop)
		if strings.ToLower(line) == "quit" {
			// Save character before quitting (but not level 1 - avoid abandoned characters)
			if session.Character != nil && session.Character.PCData != nil {
				if session.Character.Level > 1 {
					if err := s.Persistence.SavePlayer(session.Character); err != nil {
						s.logger.Error("failed to save player on quit", "error", err)
					}
					session.WriteLine("Your character has been saved.")
				} else {
					session.WriteLine("Level 1 characters are not saved. Gain a level to save your progress!")
				}
			}
			session.WriteLine("Goodbye!")
			return
		}

		// Queue command for game loop processing
		// Prompt will be sent after command completes via OnCommand callback
		s.GameLoop.QueueCommand(session.Character, line)
	}
}

// enterGame brings a character into the game world
func (s *Server) enterGame(session *Session) {
	ch := session.Character

	// Find starting room
	var startRoom *types.Room

	// New players start in MUD School (if it exists)
	if s.Login.IsNewPlayer() {
		// Give starting equipment to new characters
		s.giveStartingEquipment(ch)

		if s.World != nil {
			startRoom = s.World.GetRoom(RoomVnumMudSchool)
		}
		if startRoom != nil {
			session.WriteLine("")
			session.WriteLine("Welcome to the MUD School! Here you can learn the basics of the game.")
			session.WriteLine("Type 'help' to see available commands, and 'look' to see the room.")
			session.WriteLine("")
		} else {
			// MUD school not loaded, give basic tips anyway
			session.WriteLine("")
			session.WriteLine("Welcome! Here are some basic commands to get started:")
			session.WriteLine("  look     - see the room around you")
			session.WriteLine("  score    - see your character stats")
			session.WriteLine("  help     - get help on commands")
			session.WriteLine("  north/south/east/west/up/down - move around")
			session.WriteLine("")
		}
	}

	// Returning players go to their saved room first, then recall point
	if startRoom == nil && ch.PCData != nil && s.World != nil {
		// First try saved room (where they quit)
		if ch.PCData.SavedRoom != 0 {
			startRoom = s.World.GetRoom(ch.PCData.SavedRoom)
		}
		// Fall back to recall point
		if startRoom == nil && ch.PCData.Recall != 0 {
			startRoom = s.World.GetRoom(ch.PCData.Recall)
		}
	}

	// Reapply equipment affects for returning players
	if !s.Login.IsNewPlayer() {
		s.applyEquipmentAffects(ch)
	}

	// Otherwise use default temple
	if startRoom == nil {
		startRoom = s.getOrCreateStartRoom()
	}

	// Place character in room
	ch.InRoom = startRoom
	startRoom.AddPerson(ch)

	// Add to game loop
	s.GameLoop.AddCharacter(ch)

	// Welcome message
	session.WriteLine("")
	session.WriteLine(fmt.Sprintf("Welcome to Rivers of Time, %s!", ch.Name))
	session.WriteLine("")

	// Show the room
	s.Dispatcher.Registry.Execute("look", ch, "")

	// Reset login state for next connection
	s.Login.ResetState()
}

// applyEquipmentAffects applies all stat modifiers from equipped items
// This is called when a player logs in to restore equipment bonuses
func (s *Server) applyEquipmentAffects(ch *types.Character) {
	for _, obj := range ch.Equipment {
		if obj == nil {
			continue
		}
		for _, af := range obj.Affects.All() {
			magic.ApplyModifier(ch, af)
		}
	}
}

// getOrCreateStartRoom returns the starting room
func (s *Server) getOrCreateStartRoom() *types.Room {
	// Check if we have loaded rooms - prefer Temple of Mota (3001)
	if room, ok := s.GameLoop.Rooms[3001]; ok {
		return room
	}

	// Fallback: Create a simple starting room
	s.logger.Warn("creating fallback starting room")
	room := types.NewRoom(3001, "The Temple of Mota",
		`You are in the southern end of the temple hall in the Temple of Mota.
The temple has been constructed from giant marble blocks, eternal in
appearance, and most of the walls are covered by ancient wall paintings.`)
	room.Sector = types.SectInside
	room.Flags.Set(types.RoomSafe)

	// Add some exits
	northRoom := types.NewRoom(3005, "The Temple Altar",
		`A large altar stands in the center of this sacred chamber.
Candles flicker on either side, casting dancing shadows on the walls.`)
	northRoom.Sector = types.SectInside
	northRoom.Flags.Set(types.RoomSafe)

	// Link rooms
	room.SetExit(types.DirNorth, types.NewExit(types.DirNorth, 3005))
	room.Exits[types.DirNorth].ToRoom = northRoom

	northRoom.SetExit(types.DirSouth, types.NewExit(types.DirSouth, 3001))
	northRoom.Exits[types.DirSouth].ToRoom = room

	// Store rooms
	s.GameLoop.Rooms[3001] = room
	s.GameLoop.Rooms[3005] = northRoom

	return room
}

// giveStartingEquipment gives new characters their starting gear
func (s *Server) giveStartingEquipment(ch *types.Character) {
	if s.World == nil {
		return
	}

	// School equipment vnums based on weapon choice
	// The weapon they chose during creation is already set at 40%
	weaponVnums := map[string]int{
		"sword":   3702, // sub issue sword
		"mace":    3700, // sub issue mace
		"dagger":  3701, // sub issue dagger
		"axe":     3719, // sub issue axe
		"spear":   3717, // sub issue spear
		"flail":   3720, // sub issue flail
		"whip":    3721, // sub issue whip
		"polearm": 3722, // sub issue polearm
	}

	// Find their chosen weapon (the one at 40%)
	chosenWeapon := "sword" // default
	if ch.PCData != nil {
		for skill, level := range ch.PCData.Learned {
			if level == 40 {
				if _, ok := weaponVnums[skill]; ok {
					chosenWeapon = skill
					break
				}
			}
		}
	}

	// Helper to create and give an object
	giveObject := func(vnum int) {
		if tmpl := s.World.GetObjTemplate(vnum); tmpl != nil {
			obj := s.createObjectFromTemplate(tmpl)
			obj.Level = 1
			ch.Inventory = append(ch.Inventory, obj)
		}
	}

	// Give the weapon
	if weaponVnum, ok := weaponVnums[chosenWeapon]; ok {
		giveObject(weaponVnum)
	}

	// Give basic starting equipment - more can be earned from mud school monsters
	giveObject(3703) // vest (body)
	giveObject(3704) // shield

	// Give a light source - a torch (vnum 3030)
	giveObject(3030)

	// Give some bread/food - vnum 3011
	giveObject(3011)
	giveObject(3011)

	// Give a bottle of ale - vnum 3002
	giveObject(3002)
}

// sendGreeting sends the MUD welcome message
func (s *Server) sendGreeting(session *Session) {
	greeting := `
 ____  _                             __   _____ _                
|  _ \(_)_   _____ _ __ ___    ___  / _| |_   _(_)_ __ ___   ___ 
| |_) | \ \ / / _ \ '__/ __|  / _ \| |_    | | | | '_ ` + "`" + ` _ \ / _ \
|  _ <| |\ V /  __/ |  \__ \ | (_) |  _|   | | | | | | | | |  __/
|_| \_\_| \_/ \___|_|  |___/  \___/|_|     |_| |_|_| |_| |_|\___|

===============================================================================

                Original DikuMUD by Hans Staerfeldt, Katja Nyboe,
                Tom Madsen, Michael Seifert, and Sebastian Hammer
                Based on MERC 2.1 code by Hatchet, Furey, and Kahn
                ROM 2.4 copyright (c) 1993-1995 Russ Taylor
                ROT 1.4 copyright (c) 1996-1997 Russ Walsh
                Go port by the ROT team

===============================================================================

`
	session.Write(greeting)
}

// Write sends a string to the session
func (sess *Session) Write(text string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if _, err := sess.writer.WriteString(text); err != nil {
		slog.Debug("session write error", "error", err)
		return
	}
	if err := sess.writer.Flush(); err != nil {
		slog.Debug("session flush error", "error", err)
	}
}

// WriteLine sends a string with a newline to the session
func (sess *Session) WriteLine(text string) {
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if _, err := sess.writer.WriteString(text + "\r\n"); err != nil {
		slog.Debug("session write error", "error", err)
		return
	}
	if err := sess.writer.Flush(); err != nil {
		slog.Debug("session flush error", "error", err)
	}
}

// SendToCharacter finds the session for a character and sends output
func (s *Server) SendToCharacter(ch *types.Character, msg string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.sessions {
		if session.Character == ch {
			session.Write(msg)

			// If this character is being snooped, forward output to snooper
			if session.Descriptor != nil && session.Descriptor.SnoopedBy != nil {
				snooper := session.Descriptor.SnoopedBy
				if snooper.Character != nil {
					// Find snooper's session and send with prefix
					for _, snooperSession := range s.sessions {
						if snooperSession.Character == snooper.Character {
							snooperSession.Write(msg)
							break
						}
					}
				}
			}
			return
		}
	}
}

// tickUpdate handles periodic character updates (regeneration, affects)
func (s *Server) tickUpdate() {
	// Use GetCharacters() for thread-safe iteration
	for _, ch := range s.GameLoop.GetCharacters() {
		// HP regeneration
		if ch.Hit < ch.MaxHit {
			// Base regen based on position
			var gain int
			switch ch.Position {
			case types.PosSleeping:
				gain = ch.MaxHit / 8
			case types.PosResting:
				gain = ch.MaxHit / 12
			case types.PosSitting:
				gain = ch.MaxHit / 16
			default:
				gain = ch.MaxHit / 24
			}

			// Room bonus
			if ch.InRoom != nil {
				gain = gain * ch.InRoom.HealRate / 100
			}

			// Minimum gain
			if gain < 1 {
				gain = 1
			}

			ch.Hit = min(ch.Hit+gain, ch.MaxHit)
		}

		// Mana regeneration
		if ch.Mana < ch.MaxMana {
			var gain int
			switch ch.Position {
			case types.PosSleeping:
				gain = ch.MaxMana / 6
			case types.PosResting:
				gain = ch.MaxMana / 10
			case types.PosSitting:
				gain = ch.MaxMana / 14
			default:
				gain = ch.MaxMana / 20
			}

			// Intelligence bonus
			gain += ch.GetStat(types.StatInt) / 2

			// Room bonus
			if ch.InRoom != nil {
				gain = gain * ch.InRoom.ManaRate / 100
			}

			if gain < 1 {
				gain = 1
			}

			ch.Mana = min(ch.Mana+gain, ch.MaxMana)
		}

		// Move regeneration
		if ch.Move < ch.MaxMove {
			var gain int
			switch ch.Position {
			case types.PosSleeping:
				gain = ch.MaxMove / 4
			case types.PosResting:
				gain = ch.MaxMove / 8
			case types.PosSitting:
				gain = ch.MaxMove / 12
			default:
				gain = ch.MaxMove / 16
			}

			if gain < 1 {
				gain = 1
			}

			ch.Move = min(ch.Move+gain, ch.MaxMove)
		}

		// Process poison damage
		if ch.IsAffected(types.AffPoison) && !ch.IsAffected(types.AffSlow) {
			// Find the poison affect to get its level
			poisonLevel := 10 // Default level
			for _, aff := range ch.Affected.All() {
				if aff.Type == "poison" {
					poisonLevel = aff.Level
					break
				}
			}

			s.SendToCharacter(ch, "You shiver and suffer.\r\n")
			if ch.InRoom != nil {
				for _, person := range ch.InRoom.People {
					if person != ch {
						s.SendToCharacter(person, ch.Name+" shivers and suffers.\r\n")
					}
				}
			}

			// Deal poison damage (level/10 + 1)
			dam := poisonLevel/10 + 1
			s.Dispatcher.Combat.Damage(ch, ch, dam, types.DamPoison, false)
		}

		// Process plague damage
		if ch.IsAffected(types.AffPlague) {
			// Find the plague affect
			plagueLevel := 10 // Default
			for _, aff := range ch.Affected.All() {
				if aff.Type == "plague" {
					plagueLevel = aff.Level
					break
				}
			}

			s.SendToCharacter(ch, "You writhe in agony from the plague.\r\n")
			if ch.InRoom != nil {
				for _, person := range ch.InRoom.People {
					if person != ch {
						s.SendToCharacter(person, ch.Name+" writhes in agony as plague sores erupt from their skin.\r\n")
					}
				}
			}

			// Plague drains mana and move, and deals damage
			dam := plagueLevel/5 + 1
			if dam > ch.Level {
				dam = ch.Level
			}
			ch.Mana -= dam
			ch.Move -= dam
			if ch.Mana < 0 {
				ch.Mana = 0
			}
			if ch.Move < 0 {
				ch.Move = 0
			}

			s.Dispatcher.Combat.Damage(ch, ch, dam, types.DamDisease, false)

			// Plague can spread to others in the room
			if ch.InRoom != nil {
				for _, victim := range ch.InRoom.People {
					if victim == ch || victim.IsImmortal() {
						continue
					}
					if victim.IsAffected(types.AffPlague) {
						continue
					}
					// Small chance to spread (1 in 16)
					if combat.NumberRange(0, 15) == 0 {
						// Create plague affect on victim
						aff := types.NewAffect("plague", plagueLevel-1, combat.NumberRange(1, 2*(plagueLevel-1)), types.ApplyStr, -5, types.AffPlague)
						victim.AddAffect(aff)
						s.SendToCharacter(victim, "You feel hot and feverish.\r\n")
						if ch.InRoom != nil {
							for _, person := range ch.InRoom.People {
								if person != victim {
									s.SendToCharacter(person, victim.Name+" shivers and looks very ill.\r\n")
								}
							}
						}
					}
				}
			}
		}

		// Process incapacitated/mortal wound damage
		if ch.Position == types.PosIncap {
			if combat.NumberRange(0, 1) == 0 {
				s.Dispatcher.Combat.Damage(ch, ch, 1, types.DamNone, false)
			}
		} else if ch.Position == types.PosMortal {
			s.Dispatcher.Combat.Damage(ch, ch, 1, types.DamNone, false)
		}
	}

	// Process affect tick for all characters
	s.Dispatcher.Magic.ProcessAffectTick(s.GameLoop.Characters)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Stop gracefully shuts down the server
func (s *Server) Stop() {
	s.GameLoop.Stop()

	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for conn := range s.sessions {
		conn.Close()
	}
	s.mu.Unlock()
}

// GetSessionCount returns the number of active sessions
func (s *Server) GetSessionCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sessions)
}

// SetAPIKey sets the API key for authentication
func (s *Server) SetAPIKey(key string) {
	s.apiKey = key
}

// SetOnCommand sets the callback for when a command is processed
func (s *Server) SetOnCommand(callback OnCommandFunc) {
	s.onCommand = callback
}

// SetMetricsUpdater sets the metrics callback and wires it into the game loop
func (s *Server) SetMetricsUpdater(updater MetricsUpdaterFunc) {
	s.metricsUpdater = updater

	// Wire up to game loop pulse
	s.GameLoop.OnPulse = func() {
		if s.metricsUpdater == nil {
			return
		}

		// Count players and NPCs
		playerCount := 0
		npcCount := 0
		s.GameLoop.ForEachCharacter(func(ch *types.Character) {
			if ch.IsNPC() {
				npcCount++
			} else {
				playerCount++
			}
		})

		// Count connections
		connCount := s.GetSessionCount()

		// Update metrics
		s.metricsUpdater(playerCount, npcCount, connCount, s.commandCount)
	}
}

// findObjectsInWorld searches for objects matching the given name
func (s *Server) findObjectsInWorld(name string, level int) []*types.Object {
	var results []*types.Object
	nameLower := strings.ToLower(name)

	// Search objects in rooms
	for _, room := range s.GameLoop.Rooms {
		for _, obj := range room.Objects {
			if strings.Contains(strings.ToLower(obj.Name), nameLower) ||
				strings.Contains(strings.ToLower(obj.ShortDesc), nameLower) {
				results = append(results, obj)
			}
		}
	}

	// Search objects carried by characters
	for _, ch := range s.GameLoop.GetCharacters() {
		// Check inventory
		for _, obj := range ch.Inventory {
			if strings.Contains(strings.ToLower(obj.Name), nameLower) ||
				strings.Contains(strings.ToLower(obj.ShortDesc), nameLower) {
				results = append(results, obj)
			}
		}

		// Check equipment
		for i := types.WearLocation(0); i < types.WearLocMax; i++ {
			obj := ch.GetEquipment(i)
			if obj != nil {
				if strings.Contains(strings.ToLower(obj.Name), nameLower) ||
					strings.Contains(strings.ToLower(obj.ShortDesc), nameLower) {
					results = append(results, obj)
				}
			}
		}
	}

	return results
}

// createPortal creates a portal object linking two rooms
func (s *Server) createPortal(srcRoom, destRoom *types.Room, duration int, twoWay bool) *types.Object {
	if srcRoom == nil || destRoom == nil {
		return nil
	}

	// Create the portal object
	portal := types.NewObject(26, "a shimmering portal", types.ItemTypePortal)
	portal.Name = "portal shimmering"
	portal.LongDesc = "A shimmering portal floats here."
	portal.Timer = duration
	portal.Values[0] = 0             // Charges (0 = unlimited)
	portal.Values[1] = 0             // Exit flags
	portal.Values[2] = 0             // Gate flags
	portal.Values[3] = destRoom.Vnum // Destination room vnum

	// Place portal in source room
	game.ObjToRoom(portal, srcRoom)

	// If two-way (nexus), create return portal in destination room
	if twoWay {
		returnPortal := types.NewObject(27, "a shimmering nexus", types.ItemTypePortal)
		returnPortal.Name = "nexus portal shimmering"
		returnPortal.LongDesc = "A shimmering nexus floats here."
		returnPortal.Timer = duration
		returnPortal.Values[0] = 0
		returnPortal.Values[1] = 0
		returnPortal.Values[2] = 0
		returnPortal.Values[3] = srcRoom.Vnum

		game.ObjToRoom(returnPortal, destRoom)
	}

	return portal
}

// createPet creates a pet from a spell
// petType: "zombie" (resurrect), "animate" (animate body part), or "demon" (conjure)
// sourceObj: the corpse or body part used (nil for demon)
func (s *Server) createPet(master *types.Character, petType string, sourceObj *types.Object) *types.Character {
	if master == nil || master.InRoom == nil {
		return nil
	}

	var pet *types.Character

	switch petType {
	case "zombie":
		if sourceObj == nil {
			return nil
		}
		// Get corpse level from object's value (or use master's level / 2)
		corpseLevel := sourceObj.Level
		if corpseLevel < 1 {
			corpseLevel = master.Level / 2
		}
		pet = game.CreateZombiePet(master, master.Level, corpseLevel, sourceObj.ShortDesc)

	case "animate":
		if sourceObj == nil {
			return nil
		}
		// Animate creates a follower, not a pet (doesn't set master.Pet)
		pet = game.CreateAnimateFollower(master, sourceObj)

	case "demon":
		pet = game.CreateDemonPet(master)

	default:
		return nil
	}

	if pet == nil {
		return nil
	}

	// Place pet in the room with master
	game.CharToRoom(pet, master.InRoom)

	// Add pet to game loop
	s.GameLoop.AddCharacter(pet)

	return pet
}

// findCorpseInRoom finds a corpse object in the character's room
func (s *Server) findCorpseInRoom(ch *types.Character) *types.Object {
	if ch == nil || ch.InRoom == nil {
		return nil
	}

	// Search for corpse objects
	for _, obj := range ch.InRoom.Objects {
		if obj.ItemType == types.ItemTypeCorpseNPC || obj.ItemType == types.ItemTypeCorpsePC {
			return obj
		}
		// Also check by name/keywords
		nameLower := strings.ToLower(obj.Name)
		if strings.Contains(nameLower, "corpse") {
			return obj
		}
	}

	return nil
}

// extractObject removes an object from the game
// If it's a container (like a corpse), drop contents to the room first
func (s *Server) extractObject(obj *types.Object) {
	if obj == nil {
		return
	}

	// If it's a container, drop contents to the room
	if len(obj.Contents) > 0 && obj.InRoom != nil {
		for _, item := range obj.Contents {
			obj.RemoveContent(item)
			game.ObjToRoom(item, obj.InRoom)
		}
	}

	// Remove from inventory if carried
	if obj.CarriedBy != nil {
		game.ObjFromChar(obj)
	}

	// Remove from room if on ground
	if obj.InRoom != nil {
		game.ObjFromRoom(obj)
	}

	// Remove from container if inside one
	if obj.InObject != nil {
		obj.InObject.RemoveContent(obj)
	}
}

// dismissAllFollowers removes all pets and charmed followers when a player disconnects
// This prevents orphaned NPCs in the game world
func (s *Server) dismissAllFollowers(ch *types.Character) {
	if ch == nil {
		return
	}

	// First dismiss the primary pet
	if ch.Pet != nil {
		pet := ch.Pet
		game.DismissPet(ch)
		s.GameLoop.RemoveCharacter(pet)
	}

	// Then remove any other charmed followers in the same room
	if ch.InRoom != nil {
		// Collect followers first to avoid modifying slice while iterating
		var followers []*types.Character
		for _, person := range ch.InRoom.People {
			if person.Master == ch && person.IsAffected(types.AffCharm) && person != ch {
				followers = append(followers, person)
			}
		}

		// Remove each follower
		for _, follower := range followers {
			game.StopFollower(follower)
			if follower.InRoom != nil {
				follower.InRoom.RemovePerson(follower)
				follower.InRoom = nil
			}
			s.GameLoop.RemoveCharacter(follower)
		}
	}
}
