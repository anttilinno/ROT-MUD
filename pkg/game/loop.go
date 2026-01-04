package game

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"rotmud/pkg/ai"
	"rotmud/pkg/combat"
	"rotmud/pkg/loader"
	"rotmud/pkg/magic"
	"rotmud/pkg/skills"
	"rotmud/pkg/types"
)

// Pulse timing constants (matching ROT's merc.h)
const (
	PulsePerSecond = 4   // 4 pulses per second = 250ms per pulse
	PulseViolence  = 3   // Combat update every 3 pulses (750ms)
	PulseMobile    = 4   // Mobile AI update every 4 pulses (1s)
	PulseMusic     = 6   // Music update every 6 pulses (1.5s)
	PulseTick      = 60  // Character tick every 60 pulses (15s)
	PulseArea      = 120 // Area reset every 120 pulses (30s)
)

// Command represents a queued player command
type Command struct {
	Character *types.Character
	Input     string
	Timestamp time.Time
}

// GameLoop manages the main game loop with pulse-based timing
type GameLoop struct {
	// Configuration
	PulseRate time.Duration // Time between pulses (default 250ms)
	Logger    *slog.Logger

	// State
	PulseCount uint64 // Total pulses since start
	Running    bool
	Wizlock    bool // If true, no new connections allowed
	Newlock    bool // If true, no new characters can be created

	// Channels
	commands chan Command
	done     chan struct{}

	// Callbacks for update events
	OnPulse     func()        // Called every pulse
	OnViolence  func()        // Called every PulseViolence pulses
	OnMobile    func()        // Called every PulseMobile pulses
	OnTick      func()        // Called every PulseTick pulses
	OnAreaReset func()        // Called every PulseArea pulses
	OnCommand   func(Command) // Called when a command is processed

	// Game state references
	Characters []*types.Character  // All active characters
	Rooms      map[int]*types.Room // All rooms by vnum
	Magic      *magic.MagicSystem  // Magic system for spells/affects
	MOBprogs   *MOBprogSystem      // MOB program system
	Output     OutputFunc          // Function to send output to characters

	// Systems
	AI        *ai.AISystem         // NPC AI system
	World     *loader.World        // World data for resets
	Resets    *ResetSystem         // Area reset system
	Areas     []*types.Area        // All areas
	Combat    *combat.CombatSystem // Combat system for violence updates
	Skills    *skills.SkillSystem  // Skills system for passive skills
	WorldTime *WorldTime           // Time and weather system

	mu sync.RWMutex
}

// NewGameLoop creates a new game loop with default settings
func NewGameLoop() *GameLoop {
	return &GameLoop{
		PulseRate:  250 * time.Millisecond,
		commands:   make(chan Command, 256), // Buffered channel
		done:       make(chan struct{}),
		Characters: make([]*types.Character, 0),
		Rooms:      make(map[int]*types.Room),
		WorldTime:  NewWorldTime(),
	}
}

// Start begins the game loop in a goroutine
func (g *GameLoop) Start() {
	g.mu.Lock()
	if g.Running {
		g.mu.Unlock()
		return
	}
	g.Running = true
	g.done = make(chan struct{})
	g.mu.Unlock()

	go g.run()
}

// Stop gracefully stops the game loop
func (g *GameLoop) Stop() {
	g.mu.Lock()
	if !g.Running {
		g.mu.Unlock()
		return
	}
	g.Running = false
	g.mu.Unlock()

	close(g.done)
}

// run is the main game loop goroutine
func (g *GameLoop) run() {
	ticker := time.NewTicker(g.PulseRate)
	defer ticker.Stop()

	for {
		select {
		case <-g.done:
			return

		case <-ticker.C:
			g.pulse()

		case cmd := <-g.commands:
			g.processCommand(cmd)
		}
	}
}

// pulse processes a single game pulse
func (g *GameLoop) pulse() {
	g.mu.Lock()
	g.PulseCount++
	pulseNum := g.PulseCount
	g.mu.Unlock()

	// Call generic pulse handler
	if g.OnPulse != nil {
		g.OnPulse()
	}

	// Violence update (combat) - every 3 pulses
	if pulseNum%PulseViolence == 0 {
		g.violenceUpdate()
	}

	// Mobile update (NPC AI) - every 4 pulses
	if pulseNum%PulseMobile == 0 {
		g.mobileUpdate()
	}

	// Character tick (regen, affects) - every 60 pulses
	if pulseNum%PulseTick == 0 {
		g.tickUpdate()
	}

	// Area reset - every 120 pulses
	if pulseNum%PulseArea == 0 {
		g.areaUpdate()
	}
}

// violenceUpdate processes combat for all fighting characters
func (g *GameLoop) violenceUpdate() {
	if g.OnViolence != nil {
		g.OnViolence()
	}

	// Process wait/daze timers
	for _, ch := range g.Characters {
		if ch.Wait > 0 {
			ch.Wait--
		}
		if ch.Daze > 0 {
			ch.Daze--
		}
	}

	// Process combat for all fighting characters
	if g.Combat != nil {
		g.Combat.ViolenceUpdate(g.Characters)
	}
}

// mobileUpdate processes NPC AI
func (g *GameLoop) mobileUpdate() {
	if g.OnMobile != nil {
		g.OnMobile()
	}

	// Process AI for all NPCs
	if g.AI != nil {
		g.AI.ProcessAllMobiles(g.Characters)
	}
}

// weatherUpdate advances game time and broadcasts weather changes
func (g *GameLoop) weatherUpdate() {
	if g.WorldTime == nil {
		return
	}

	message := g.WorldTime.Tick()
	if message != "" && g.Output != nil {
		// Broadcast weather changes to all outdoor, awake characters
		for _, ch := range g.Characters {
			if ch.Position > types.PosSleeping && IsOutside(ch) {
				g.Output(ch, message)
			}
		}
	}
}

// tickUpdate processes character regeneration and affect decay
func (g *GameLoop) tickUpdate() {
	if g.OnTick != nil {
		g.OnTick()
	}

	// Update weather and time (once per tick = once per game hour)
	g.weatherUpdate()

	// Process hunger/thirst for players
	g.processConditions()

	// Process regeneration for all characters
	g.processRegeneration()

	// Process affect decay and wear-off
	g.processAffectDecay()
}

// processConditions handles hunger, thirst, and drunk decay for players
func (g *GameLoop) processConditions() {
	for _, ch := range g.Characters {
		// Only process conditions for players
		if ch.IsNPC() || ch.PCData == nil {
			continue
		}

		// Immortals don't get hungry/thirsty
		if ch.IsImmortal() {
			continue
		}

		// Decrease hunger (-1 per tick)
		g.gainCondition(ch, types.CondHunger, -1)

		// Decrease thirst (-1 per tick)
		g.gainCondition(ch, types.CondThirst, -1)

		// Decrease fullness (-1 per tick)
		g.gainCondition(ch, types.CondFull, -1)

		// Decrease drunkenness (-1 per tick)
		if ch.PCData.Condition[types.CondDrunk] > 0 {
			g.gainCondition(ch, types.CondDrunk, -1)
			if ch.PCData.Condition[types.CondDrunk] == 0 {
				g.sendOutput(ch, "You are sober.\r\n")
			}
		}
	}
}

// gainCondition applies a condition change and sends messages
func (g *GameLoop) gainCondition(ch *types.Character, cond int, value int) {
	if ch.IsNPC() || ch.PCData == nil || ch.IsImmortal() {
		return
	}

	oldValue := ch.PCData.Condition[cond]
	newValue := oldValue + value

	// Bound the value
	if newValue < -48 {
		newValue = -48
	}
	if newValue > 48 {
		newValue = 48
	}

	ch.PCData.Condition[cond] = newValue

	// Send messages when conditions hit critical levels
	if oldValue > 0 && newValue <= 0 {
		switch cond {
		case types.CondHunger:
			g.sendOutput(ch, "You are hungry.\r\n")
		case types.CondThirst:
			g.sendOutput(ch, "You are thirsty.\r\n")
		}
	}

	// Send messages when starving/dehydrated
	if newValue < -16 && (oldValue >= -16 || value < 0) {
		switch cond {
		case types.CondHunger:
			g.sendOutput(ch, "You are STARVING!\r\n")
		case types.CondThirst:
			g.sendOutput(ch, "You are DYING of THIRST!\r\n")
		}
	}
}

// sendOutput sends a message to a character if output callback is set
func (g *GameLoop) sendOutput(ch *types.Character, msg string) {
	if g.Output != nil {
		g.Output(ch, msg)
	}
}

// processRegeneration handles HP, mana, and move regeneration for all characters
func (g *GameLoop) processRegeneration() {
	for _, ch := range g.Characters {
		if ch.Position >= types.PosStunned {
			// HP regeneration
			if ch.Hit < ch.MaxHit {
				ch.Hit += g.hitGain(ch)
				if ch.Hit > ch.MaxHit {
					ch.Hit = ch.MaxHit
				}
			}

			// Mana regeneration
			if ch.Mana < ch.MaxMana {
				ch.Mana += g.manaGain(ch)
				if ch.Mana > ch.MaxMana {
					ch.Mana = ch.MaxMana
				}
			}

			// Move regeneration
			if ch.Move < ch.MaxMove {
				ch.Move += g.moveGain(ch)
				if ch.Move > ch.MaxMove {
					ch.Move = ch.MaxMove
				}
			}
		}
	}
}

// getSkill returns a character's skill level (0-100)
func (g *GameLoop) getSkill(ch *types.Character, skillName string) int {
	if g.Skills != nil {
		return g.Skills.GetSkill(ch, skillName)
	}
	// Fallback for NPCs
	if ch.IsNPC() {
		return 40 + ch.Level/2
	}
	return 0
}

// hitGain calculates HP regeneration for a character
func (g *GameLoop) hitGain(ch *types.Character) int {
	var gain int

	if ch.IsNPC() {
		gain = 5 + ch.Level
		if ch.IsAffected(types.AffRegeneration) {
			gain *= 2
		}

		switch ch.Position {
		default:
			gain /= 2
		case types.PosSleeping:
			gain = 3 * gain / 2
		case types.PosResting:
			// No change
		case types.PosFighting:
			gain /= 3
		}
	} else {
		gain = max(3, ch.GetStat(types.StatCon)-3+ch.Level/2)

		// Fast healing skill bonus
		fastHealing := g.getSkill(ch, "fast healing")
		if fastHealing > 0 {
			gain += fastHealing * gain / 100 // Up to 100% bonus at skill 100
		}

		switch ch.Position {
		default:
			gain /= 4
		case types.PosSleeping:
			// No change
		case types.PosResting:
			gain /= 2
		case types.PosFighting:
			gain /= 6
		}

		// Apply hunger/thirst penalties
		if ch.PCData != nil {
			if ch.PCData.Condition[types.CondHunger] <= 0 {
				gain /= 2 // Half regen when hungry
			}
			if ch.PCData.Condition[types.CondThirst] <= 0 {
				gain /= 2 // Half regen when thirsty
			}
		}
	}

	// Room heal rate (default 90 = 100%)
	if ch.InRoom != nil {
		gain = gain * ch.InRoom.HealRate / 90
	}

	// TODO: Apply furniture bonuses

	// Apply status penalties
	if ch.IsAffected(types.AffPoison) {
		gain /= 4
	}
	if ch.IsAffected(types.AffPlague) {
		gain /= 8
	}
	if ch.IsAffected(types.AffHaste) || ch.IsAffected(types.AffSlow) {
		gain /= 2
	}

	return min(gain, ch.MaxHit-ch.Hit)
}

// manaGain calculates mana regeneration for a character
func (g *GameLoop) manaGain(ch *types.Character) int {
	var gain int

	if ch.IsNPC() {
		gain = 5 + ch.Level
		switch ch.Position {
		default:
			gain /= 2
		case types.PosSleeping:
			gain = 3 * gain / 2
		case types.PosResting:
			// No change
		case types.PosFighting:
			gain /= 3
		}
	} else {
		gain = (ch.GetStat(types.StatWis) + ch.GetStat(types.StatInt) + ch.Level) / 3 * 2

		// Meditation skill bonus
		meditation := g.getSkill(ch, "meditation")
		if meditation > 0 {
			gain += meditation * gain / 100 // Up to 100% bonus at skill 100
		}

		if class := types.GetClass(ch.Class); class != nil && !class.FreesMana {
			gain /= 2
		}

		switch ch.Position {
		default:
			gain /= 4
		case types.PosSleeping:
			// No change
		case types.PosResting:
			gain /= 2
		case types.PosFighting:
			gain /= 6
		}

		// Apply hunger/thirst penalties
		if ch.PCData != nil {
			if ch.PCData.Condition[types.CondHunger] <= 0 {
				gain /= 2
			}
			if ch.PCData.Condition[types.CondThirst] <= 0 {
				gain /= 2
			}
		}
	}

	// Room mana rate
	if ch.InRoom != nil {
		gain = gain * ch.InRoom.ManaRate / 90
	}

	// TODO: Apply furniture bonuses

	return min(gain, ch.MaxMana-ch.Mana)
}

// moveGain calculates move regeneration for a character
func (g *GameLoop) moveGain(ch *types.Character) int {
	var gain int

	if ch.IsNPC() {
		gain = ch.Level
	} else {
		gain = max(15, ch.Level)

		switch ch.Position {
		case types.PosSleeping:
			gain += ch.GetStat(types.StatDex)
		case types.PosResting:
			gain += ch.GetStat(types.StatDex) / 2
		}

		// Apply hunger/thirst penalties
		if ch.PCData != nil {
			if ch.PCData.Condition[types.CondHunger] <= 0 {
				gain /= 2
			}
			if ch.PCData.Condition[types.CondThirst] <= 0 {
				gain /= 2
			}
		}
	}

	// Room heal rate
	if ch.InRoom != nil {
		gain = gain * ch.InRoom.HealRate / 90
	}

	// Apply status penalties
	if ch.IsAffected(types.AffPoison) {
		gain /= 4
	}
	if ch.IsAffected(types.AffPlague) {
		gain /= 8
	}
	if ch.IsAffected(types.AffHaste) || ch.IsAffected(types.AffSlow) {
		gain /= 2
	}

	return min(gain, ch.MaxMove-ch.Move)
}

// processAffectDecay handles affect duration decrement and wear-off
func (g *GameLoop) processAffectDecay() {
	if g.Magic != nil {
		g.Magic.ProcessAffectTick(g.Characters)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// areaUpdate processes area resets
func (g *GameLoop) areaUpdate() {
	if g.OnAreaReset != nil {
		g.OnAreaReset()
	}

	// Process area resets
	if g.Resets != nil && g.Areas != nil {
		g.Resets.ProcessAreaResets(g.Areas)
	}
}

// processCommand handles a single player command
func (g *GameLoop) processCommand(cmd Command) {
	if g.OnCommand != nil {
		g.OnCommand(cmd)
	}

	// TODO: Parse and execute the command
}

// QueueCommand adds a command to the processing queue
func (g *GameLoop) QueueCommand(ch *types.Character, input string) {
	cmd := Command{
		Character: ch,
		Input:     input,
		Timestamp: time.Now(),
	}

	select {
	case g.commands <- cmd:
		// Command queued successfully
	default:
		// Channel full, command dropped
		if g.Logger != nil {
			g.Logger.Warn("command queue full, dropping command",
				"character", ch.Name,
				"input", input)
		}
	}
}

// AddCharacter adds a character to the game
func (g *GameLoop) AddCharacter(ch *types.Character) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.Characters = append(g.Characters, ch)
}

// RemoveCharacter removes a character from the game
func (g *GameLoop) RemoveCharacter(ch *types.Character) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for i, c := range g.Characters {
		if c == ch {
			g.Characters = append(g.Characters[:i], g.Characters[i+1:]...)
			return
		}
	}
}

// GetCharacterCount returns the number of active characters
func (g *GameLoop) GetCharacterCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.Characters)
}

// GetPulseCount returns the current pulse count
func (g *GameLoop) GetPulseCount() uint64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.PulseCount
}

// FindCharacterByName finds an online character by name (case-insensitive prefix match)
func (g *GameLoop) FindCharacterByName(name string) *types.Character {
	g.mu.RLock()
	defer g.mu.RUnlock()

	name = strings.ToLower(name)
	for _, ch := range g.Characters {
		if strings.HasPrefix(strings.ToLower(ch.Name), name) {
			return ch
		}
	}
	return nil
}

// IsRunning returns true if the game loop is running
func (g *GameLoop) IsRunning() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Running
}

// GetCharacters returns a copy of the characters list (thread-safe)
func (g *GameLoop) GetCharacters() []*types.Character {
	g.mu.RLock()
	defer g.mu.RUnlock()
	result := make([]*types.Character, len(g.Characters))
	copy(result, g.Characters)
	return result
}

// GetRoom returns the room with the given vnum
func (g *GameLoop) GetRoom(vnum int) *types.Room {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Rooms[vnum]
}

// GetAllRooms returns all rooms in the game
func (g *GameLoop) GetAllRooms() map[int]*types.Room {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.Rooms
}

// GetPlayers returns all player characters (non-NPCs)
func (g *GameLoop) GetPlayers() []*types.Character {
	g.mu.RLock()
	defer g.mu.RUnlock()
	var players []*types.Character
	for _, ch := range g.Characters {
		if !ch.IsNPC() {
			players = append(players, ch)
		}
	}
	return players
}

// FindPlayer finds a player character by name (case-insensitive prefix match)
func (g *GameLoop) FindPlayer(name string) *types.Character {
	g.mu.RLock()
	defer g.mu.RUnlock()
	name = strings.ToLower(name)
	for _, ch := range g.Characters {
		if !ch.IsNPC() && strings.HasPrefix(strings.ToLower(ch.Name), name) {
			return ch
		}
	}
	return nil
}

// GetTime returns current game time
func (g *GameLoop) GetTime() map[string]interface{} {
	if g.WorldTime == nil {
		return map[string]interface{}{
			"hour":   12,
			"suffix": "pm",
			"day":    "the Moon",
		}
	}
	hour12 := g.WorldTime.Time.Hour % 12
	if hour12 == 0 {
		hour12 = 12
	}
	suffix := "am"
	if g.WorldTime.Time.Hour >= 12 {
		suffix = "pm"
	}
	day := g.WorldTime.Time.Day + 1
	dayName := dayNames[day%7]

	return map[string]interface{}{
		"hour":   hour12,
		"suffix": suffix,
		"day":    dayName,
	}
}

// GetWeather returns current weather
func (g *GameLoop) GetWeather() map[string]interface{} {
	if g.WorldTime == nil {
		return map[string]interface{}{
			"description": "The weather is clear and pleasant.",
		}
	}
	return map[string]interface{}{
		"description": g.WorldTime.GetWeatherString(),
	}
}

// ForEachCharacter calls fn for each character (thread-safe)
func (g *GameLoop) ForEachCharacter(fn func(*types.Character)) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	for _, ch := range g.Characters {
		fn(ch)
	}
}

// RunWithContext runs the game loop until the context is cancelled
func (g *GameLoop) RunWithContext(ctx context.Context) {
	g.mu.Lock()
	g.Running = true
	g.mu.Unlock()

	ticker := time.NewTicker(g.PulseRate)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			g.mu.Lock()
			g.Running = false
			g.mu.Unlock()
			return

		case <-ticker.C:
			g.pulse()

		case cmd := <-g.commands:
			g.processCommand(cmd)
		}
	}
}
