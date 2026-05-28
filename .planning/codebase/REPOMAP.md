──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
Using openrouter/deepseek/deepseek-r1:free model with API key from environment.
Aider v0.86.2
Model: openrouter/deepseek/deepseek-r1:free with diff edit format, prompt cache, infinite output
Git repo: .git with 368 files
Repo-map: using 4096 tokens, auto refresh
Here are summaries of some files present in my git repository.
Do not propose changes to these files, treat them as *read-only*.
If you need to edit any of these files, ask me to *add them to the chat* first.

go/pkg/ai/specials.go:
⋮
│type SpecialContext struct {
│       Magic            *magic.MagicSystem
│       Output           func(ch *types.Character, msg string)
│       ActToRoom        func(msg string, ch, victim *types.Character, output func(ch *types.Character, ms
│       StartCombat      func(ch, victim *types.Character)
│       CastSpell        func(ch *types.Character, spellName string, victim *types.Character) bool
│       FindPlayerInRoom func(ch *types.Character) *types.Character
│       MoveChar         func(ch *types.Character, dir types.Direction)
⋮

go/pkg/combat/combat.go:
⋮
│type OutputFunc func(ch *types.Character, msg string)
│
⋮
│type CombatSystem struct {
│       Output      OutputFunc
│       RoomFinder  RoomFinderFunc  // For finding recall room on death
│       CharMover   CharMoverFunc   // For moving characters to rooms
│       SkillGetter SkillGetterFunc // For checking skill levels
│       OnLevelUp   OnLevelUpFunc   // Called when a character levels up
│       OnDamage    OnDamageFunc    // Called when damage is dealt (for metrics)
│       OnKill      OnKillFunc      // Called when a character is killed (for quests)
│       OnDeath     OnDeathFunc     // Called after death processing (for autoloot/autosac)
⋮

go/pkg/combat/dice.go:
⋮
│func NumberRange(low, high int) int {
│       if low >= high {
│               return low
│       }
│       return low + rand.Intn(high-low+1)
⋮
│func NumberPercent() int {
│       return rand.Intn(100) + 1
⋮
│func Min(a, b int) int {
│       if a < b {
│               return a
│       }
│       return b
⋮
│func Clamp(val, min, max int) int {
│       if val < min {
│               return min
│       }
│       if val > max {
│               return max
│       }
│       return val
⋮

go/pkg/game/act.go:
⋮
│type OutputFunc func(ch *types.Character, msg string)
│
⋮
│func ActFormat(format string, ch *types.Character, victim *types.Character, obj *types.Object) stri
│       var result strings.Builder
│       result.Grow(len(format) * 2)
│
│       for i := 0; i < len(format); i++ {
│               if format[i] != '$' || i+1 >= len(format) {
│                       result.WriteByte(format[i])
│                       continue
│               }
│
⋮
│func himHerIt(ch *types.Character) string {
│       switch ch.Sex {
│       case types.SexMale:
│               return "him"
│       case types.SexFemale:
│               return "her"
│       default:
│               return "it"
│       }
⋮
│func hisHerIts(ch *types.Character) string {
│       switch ch.Sex {
│       case types.SexMale:
│               return "his"
│       case types.SexFemale:
│               return "her"
│       default:
│               return "its"
│       }
⋮
│func ActToRoom(format string, ch *types.Character, victim *types.Character, obj *types.Object, outp
│       if ch == nil || ch.InRoom == nil {
│               return
│       }
│
│       msg := ActFormat(format, ch, victim, obj) + "\r\n"
│
│       for _, person := range ch.InRoom.People {
│               if person == ch {
│                       continue
⋮

go/pkg/game/clans.go:
⋮
│type ClanSystem struct {
│       clans map[int]*Clan // Clan ID to clan
⋮
│func (cs *ClanSystem) GetCharacterClan(ch *types.Character) *Clan {
│       if ch.PCData == nil {
│               return nil
│       }
│       return cs.GetClan(ch.PCData.Clan)
⋮

go/pkg/game/commands.go:
⋮
│func (d *CommandDispatcher) send(ch *types.Character, msg string) {
│       if d.Output != nil {
│               d.Output(ch, msg)
│       }
⋮
│type CommandDispatcher struct {
│       Registry *CommandRegistry
│       Output   func(ch *types.Character, msg string) // Output callback
│       GameLoop *GameLoop                             // Reference to game loop for looking up characters
│       Combat   *combat.CombatSystem                  // Combat system
│       Magic    *magic.MagicSystem                    // Magic system
│       Skills   *skills.SkillSystem                   // Skills system
│       Shops    *shops.ShopHandler                    // Shop system
│       Socials  *SocialRegistry                       // Social commands
│       Notes    *NoteSystem                           // Note/board system
⋮

go/pkg/game/handler.go:
⋮
│func FindCharInRoom(ch *types.Character, name string) *types.Character {
│       if ch == nil || ch.InRoom == nil {
│               return nil
│       }
│
│       // Check "self" or "me"
│       lowerName := strings.ToLower(name)
│       if lowerName == "self" || lowerName == "me" {
│               return ch
│       }
│
⋮
│func parseNumberPrefix(name string) (int, string) {
│       // Check for N.item format
│       if idx := strings.Index(name, "."); idx > 0 {
│               numStr := name[:idx]
│               if num, err := strconv.Atoi(numStr); err == nil && num > 0 {
│                       return num, name[idx+1:]
│               }
│       }
│       return 1, name
⋮
│func nameMatches(fullName, search string) bool {
│       fullName = strings.ToLower(fullName)
│       search = strings.ToLower(search)
│
│       // Check exact match
│       if fullName == search {
│               return true
│       }
│
│       // Check prefix match
⋮
│func keywordsMatch(desc, search string) bool {
│       desc = strings.ToLower(desc)
│       search = strings.ToLower(search)
│
│       for _, word := range strings.Fields(desc) {
│               // Remove punctuation
│               word = strings.Trim(word, ".,!?;:'\"")
│               if strings.HasPrefix(word, search) {
│                       return true
│               }
⋮

go/pkg/game/loop.go:
⋮
│type GameLoop struct {
│       // Configuration
│       PulseRate time.Duration // Time between pulses (default 250ms)
│       Logger    *slog.Logger
│
│       // State
│       PulseCount uint64 // Total pulses since start
│       Running    bool
│       Wizlock    bool // If true, no new connections allowed
│       Newlock    bool // If true, no new characters can be created
│
⋮

go/pkg/game/mobprogs.go:
⋮
│type MOBprogTrigger int
│
⋮
│type MOBprogSystem struct {
│       programs    map[int][]*MOBprog // Mob vnum to list of programs
│       executeCmd  CommandExecutor
│       sendRoomMsg RoomMessageSender
⋮
│func (mps *MOBprogSystem) ExecuteTrigger(trigger MOBprogTrigger, mob *types.Character, args ...inte
│       programs := mps.GetPrograms(mob.MobVnum)
│       if len(programs) == 0 {
│               return
│       }
│
│       for _, prog := range programs {
│               if prog.Trigger != trigger {
│                       continue
│               }
│
⋮

go/pkg/game/notes.go:
⋮
│type NoteType int
│
⋮
│func (t NoteType) String() string {
│       names := []string{"note", "idea", "news", "changes", "penalty"}
│       if t >= 0 && int(t) < len(names) {
│               return names[t]
│       }
│       return "note"
⋮
│type NoteSystem struct {
│       notes   map[NoteType][]*Note
│       nextID  int
│       dataDir string
│       mu      sync.RWMutex
⋮
│func (ns *NoteSystem) getFilename(t NoteType) string {
│       if ns.dataDir == "" {
│               return t.String() + ".json"
│       }
│       return ns.dataDir + "/" + t.String() + ".json"
⋮
│type NoteEditor struct {
│       To      string
│       Subject string
│       Lines   []string
⋮

go/pkg/game/quests.go:
⋮
│type QuestSystem struct {
│       quests       map[int]*Quest            // Quest ID to quest
│       playerQuests map[string][]*PlayerQuest // Player name to their active/completed quests
⋮
│func (qs *QuestSystem) GetQuest(id int) *Quest {
│       return qs.quests[id]
⋮

go/pkg/game/weather.go:
⋮
│type TimeInfo struct {
│       Hour  int // 0-23
│       Day   int // 0-34
│       Month int // 0-16
│       Year  int
⋮
│type WeatherInfo struct {
│       Mmhg     int // Pressure (960-1040)
│       Change   int // Pressure change rate (-12 to +12)
│       Sky      int // Sky condition (cloudless, cloudy, raining, lightning)
│       Sunlight int // Time of day (dark, rise, light, set)
⋮
│type WorldTime struct {
│       Time    TimeInfo
│       Weather WeatherInfo
⋮
│func (w *WorldTime) updateWeather() string {
│       // Seasonal pressure bias
│       var diff int
│       if w.Time.Month >= 9 && w.Time.Month <= 16 {
│               // Winter months - low pressure more likely
│               if w.Weather.Mmhg > 985 {
│                       diff = -2
│               } else {
│                       diff = 2
│               }
⋮

go/pkg/help/help.go:
⋮
│type Entry struct {
│       Keywords    []string `toml:"keywords"`
│       Level       int      `toml:"level"`       // Minimum level to see
│       SeeAlso     []string `toml:"see_also"`    // Related topics
│       Syntax      string   `toml:"syntax"`      // Command syntax
│       Description string   `toml:"description"` // Full help text
⋮
│type HelpFile struct {
│       Entries []Entry `toml:"help"`
⋮
│type System struct {
│       entries map[string]*Entry // keyword -> entry
⋮
│func (s *System) LoadFile(path string) error {
│       data, err := os.ReadFile(path)
│       if err != nil {
│               return err
│       }
│
│       var hf HelpFile
│       if err := toml.Unmarshal(data, &hf); err != nil {
│               return err
│       }
│
⋮

go/pkg/loader/loader.go:
⋮
│func (w *World) GetMobTemplate(vnum int) *MobileData {
│       return w.MobTemplates[vnum]
⋮

go/pkg/loader/schema.go:
⋮
│type ServerConfig struct {
│       TelnetPort    int    `toml:"telnet_port"`
│       WebsocketPort int    `toml:"websocket_port"`
│       PulseMs       int    `toml:"pulse_ms"`
│       DataPath      string `toml:"data_path"`
⋮
│type LoggingConfig struct {
│       Level  string `toml:"level"`
│       Format string `toml:"format"`
⋮
│type SecurityConfig struct {
│       APIKey string `toml:"api_key"`
⋮
│type AreaMetadata struct {
│       ID            string     `toml:"id"`
│       Name          string     `toml:"name"`
│       Credits       string     `toml:"credits"`
│       ResetInterval int        `toml:"reset_interval"`
│       VnumRange     VnumRange  `toml:"vnum_range"`
│       LevelRange    LevelRange `toml:"level_range"`
⋮
│type VnumRange struct {
│       Min int `toml:"min"`
│       Max int `toml:"max"`
⋮
│type LevelRange struct {
│       Low  int `toml:"low"`
│       High int `toml:"high"`
⋮
│type RoomsFile struct {
│       Rooms []RoomData `toml:"rooms"`
⋮
│type RoomData struct {
│       Vnum        int             `toml:"vnum"`
│       Name        string          `toml:"name"`
│       Description string          `toml:"description"`
│       Sector      string          `toml:"sector"`
│       RoomFlags   []string        `toml:"room_flags"`
│       HealRate    int             `toml:"heal_rate"`
│       ManaRate    int             `toml:"mana_rate"`
│       Exits       []ExitData      `toml:"exits"`
│       ExtraDescs  []ExtraDescData `toml:"extra_descs"`
⋮
│type MobResetData struct {
│       Vnum   int              `toml:"vnum"`   // Mob template vnum
│       Max    int              `toml:"max"`    // Max number in world (0 = unlimited)
│       Count  int              `toml:"count"`  // Number to spawn (default 1)
│       Equips []EquipResetData `toml:"equips"` // Equipment to give to this mob
⋮
│type EquipResetData struct {
│       Vnum    int    `toml:"vnum"`     // Object template vnum
│       WearLoc string `toml:"wear_loc"` // Wear location (wield, body, head, etc.)
│       Limit   int    `toml:"limit"`    // Max number in world (0 = unlimited)
│       InvOnly bool   `toml:"inv_only"` // Put in inventory instead of equipping
⋮
│type ObjResetData struct {
│       Vnum  int `toml:"vnum"`  // Object template vnum
│       Max   int `toml:"max"`   // Max number in world (0 = unlimited)
│       Count int `toml:"count"` // Number to spawn (default 1)
⋮
│type ExitData struct {
│       Direction   string   `toml:"direction"`
│       ToVnum      int      `toml:"to_vnum"`
│       Key         int      `toml:"key"`
│       Keywords    string   `toml:"keywords"`
│       Description string   `toml:"description"`
│       Flags       []string `toml:"flags"`
⋮
│type ExtraDescData struct {
│       Keywords    []string `toml:"keywords"`
│       Description string   `toml:"description"`
⋮
│type MobilesFile struct {
│       Mobiles []MobileData `toml:"mobiles"`
⋮
│type MobileData struct {
│       Vnum        int       `toml:"vnum"`
│       Keywords    []string  `toml:"keywords"`
│       ShortDesc   string    `toml:"short_desc"`
│       LongDesc    string    `toml:"long_desc"`
│       Description string    `toml:"description"`
│       Level       int       `toml:"level"`
│       Sex         string    `toml:"sex"`
│       Race        string    `toml:"race"`
│       Alignment   int       `toml:"alignment"`
⋮
│type ShopData struct {
│       BuyTypes   []string `toml:"buy_types"`   // Item types the shop buys
│       ProfitBuy  int      `toml:"profit_buy"`  // Buy profit margin (100 = base)
│       ProfitSell int      `toml:"profit_sell"` // Sell profit margin (100 = base)
│       OpenHour   int      `toml:"open_hour"`   // Hour shop opens
│       CloseHour  int      `toml:"close_hour"`  // Hour shop closes
⋮
│type DiceData struct {
│       Number int `toml:"number"`
│       Size   int `toml:"size"`
│       Bonus  int `toml:"bonus"`
⋮
│type ObjectsFile struct {
│       Objects []ObjectData `toml:"objects"`
⋮
│type ObjectData struct {
│       Vnum       int             `toml:"vnum"`
│       Keywords   []string        `toml:"keywords"`
│       ShortDesc  string          `toml:"short_desc"`
│       LongDesc   string          `toml:"long_desc"`
│       ItemType   string          `toml:"item_type"`
│       Level      int             `toml:"level"`
│       Weight     int             `toml:"weight"`
│       Cost       int             `toml:"cost"`
│       Condition  int             `toml:"condition"`
⋮
│type WeaponData struct {
│       WeaponType string   `toml:"weapon_type"`
│       DiceNumber int      `toml:"dice_number"`
│       DiceSize   int      `toml:"dice_size"`
│       DamageType string   `toml:"damage_type"`
│       Flags      []string `toml:"flags"`
⋮
│type ArmorData struct {
│       ACPierce int `toml:"ac_pierce"`
│       ACBash   int `toml:"ac_bash"`
│       ACSlash  int `toml:"ac_slash"`
│       ACExotic int `toml:"ac_exotic"`
⋮
│type ContainerData struct {
│       Capacity  int      `toml:"capacity"`
│       Flags     []string `toml:"flags"`
│       Key       int      `toml:"key"`
│       MaxWeight int      `toml:"max_weight"`
⋮
│type LightData struct {
│       Duration int `toml:"duration"`
⋮
│type AffectData struct {
│       Location string `toml:"location"`
│       Modifier int    `toml:"modifier"`
⋮
│type MOBprogData struct {
│       // Name is the unique identifier for this mobprog within the area
│       // Derived from filename if not specified
│       Name string `toml:"name"`
│
│       // Trigger type: speech, act, fight, death, entry, greet, give, bribe, etc.
│       Trigger string `toml:"trigger"`
│
│       // TriggerArg is the argument for the trigger:
│       // - For speech/act: phrase to match
⋮
│type MOBprogFile struct {
│       Progs []MOBprogData `toml:"progs"`
⋮

go/pkg/magic/affect.go:
⋮
│func AddAffect(ch *types.Character, af *types.Affect) {
│       // Check for existing affect of same type
│       existing := ch.Affected.FindByType(af.Type)
│       if existing != nil {
│               // Stack duration (up to a limit)
│               existing.Duration += af.Duration
│               if existing.Duration > 100 {
│                       existing.Duration = 100
│               }
│               // Update level if stronger
⋮
│func NewAffect(affectType string, level, duration int, location types.ApplyType, modifier int) *typ
│       return &types.Affect{
│               Type:     affectType,
│               Level:    level,
│               Duration: duration,
│               Location: location,
│               Modifier: modifier,
│       }
⋮

go/pkg/magic/spell.go:
⋮
│type TargetType int
│
⋮
│type SpellFunc func(caster *types.Character, level int, target interface{}) bool
│
⋮
│func (s *Spell) GetClassLevel(classIndex int) int {
│       // Map class index to name and look up
│       className := types.ClassName(classIndex)
│       if reqLevel, ok := s.Levels[className]; ok {
│               return reqLevel
│       }
│       return 0 // Can't cast
⋮
│type SpellRegistry struct {
│       byName map[string]*Spell
│       bySlot map[int]*Spell
⋮
│func (r *SpellRegistry) FindByName(name string) *Spell {
│       if spell, ok := r.byName[name]; ok {
│               return spell
│       }
│       return nil
⋮

go/pkg/magic/spells_data.go:
⋮
│type SpellData struct {
│       Name       string
│       Slot       int
│       Target     TargetType
│       MinMana    int
│       Beats      int // Lag after casting
│       DamageNoun string
│       WearOff    string
│       WearOffObj string
│       // Class levels: indexed by class constant
⋮

go/pkg/server/metrics.go:
⋮
│type Metrics struct {
│       PlayersOnline   prometheus.Gauge
│       NPCsActive      prometheus.Gauge
│       CommandsTotal   prometheus.Counter
│       CombatDamage    prometheus.Histogram
│       PulseLatency    prometheus.Histogram
│       SpellsCast      prometheus.Counter
│       ConnectionsOpen prometheus.Gauge
⋮
│func (m *Metrics) Handler() http.Handler {
│       return promhttp.Handler()
⋮

go/pkg/shops/commands.go:
⋮
│func (h *ShopHandler) sendToChar(ch *types.Character, format string, args ...interface{}) {
│       if h.Output != nil {
│               h.Output(ch, fmt.Sprintf(format, args...))
│       }
⋮

go/pkg/shops/shop.go:
⋮
│func getHaggleBonus(ch *types.Character) int {
│       if ch == nil || ch.IsNPC() || ch.PCData == nil || ch.PCData.Learned == nil {
│               return 0
│       }
│
│       haggle := ch.PCData.Learned["haggle"]
│       if haggle <= 0 {
│               return 0
│       }
│
⋮
│type ShopRegistry struct {
│       byKeeperVnum map[int]*Shop
⋮

go/pkg/skills/skill.go:
⋮
│type SkillType int
│
⋮
│type Skill struct {
│       Name        string         // Skill name
│       Type        SkillType      // Skill or spell
│       Levels      map[int]int    // Class index -> level required (0 = can't learn)
│       Rating      map[int]int    // Class index -> difficulty rating (for improvement)
│       MinPosition types.Position // Minimum position to use
│       MinMana     int            // Mana cost (for spells)
│       Beats       int            // Lag after use (in pulses)
│       NounDamage  string         // Damage message noun
│       WearOff     string         // Message when affect wears off
⋮
│func (s *Skill) SetClassLevel(classIndex, level, rating int) *Skill {
│       s.Levels[classIndex] = level
│       s.Rating[classIndex] = rating
│       return s
⋮
│func (s *Skill) GetLevel(classIndex int) int {
│       if level, ok := s.Levels[classIndex]; ok {
│               return level
│       }
│       return 0 // Can't learn
⋮
│type SkillRegistry struct {
│       byName  map[string]*Skill
│       byIndex map[int]*Skill
│       skills  []*Skill
⋮

go/pkg/types/affect.go:
⋮
│type Affect struct {
│       Type         string      // Spell/skill name
│       Level        int         // Caster level
│       Duration     int         // Ticks remaining (-1 for permanent)
│       Location     ApplyType   // What stat to modify
│       Modifier     int         // How much to modify
│       BitVector    AffectFlags // Flags to set (e.g., AffSanctuary)
│       ShieldVector ShieldFlags // Shield flags to set (e.g., ShdFire, ShdIce)
⋮
│func (a *Affect) IsExpired() bool {
│       return a.Duration == 0
⋮
│type AffectList struct {
│       affects []*Affect
⋮
│func (l *AffectList) FindByType(spellType string) *Affect {
│       for _, a := range l.affects {
│               if a.Type == spellType {
│                       return a
│               }
│       }
│       return nil
⋮
│func (l *AffectList) GetBitVector() AffectFlags {
│       var bits AffectFlags
│       for _, a := range l.affects {
│               bits |= a.BitVector
│       }
│       return bits
⋮
│func (l *AffectList) GetShieldVector() ShieldFlags {
│       var bits ShieldFlags
│       for _, a := range l.affects {
│               bits |= a.ShieldVector
│       }
│       return bits
⋮

go/pkg/types/character.go:
⋮
│type Character struct {
│       // Identity
│       Name      string // Character name
│       ShortDesc string // Short description (for NPCs)
│       LongDesc  string // Long description (when seen in room)
│       Desc      string // Full description (examine)
│
│       // Core attributes
│       Level     int  // Character level
│       Class     int  // Class index
⋮
│func NewCharacter(name string) *Character {
│       return &Character{
│               Name:      name,
│               Level:     1,
│               Position:  PosStanding,
│               Alignment: 0,
│               Size:      SizeMedium,
│               Sex:       SexNeutral,
│               Hit:       20,
│               MaxHit:    20,
⋮
│func (ch *Character) IsNPC() bool {
│       return ch.Act.Has(ActNPC)
⋮
│func (ch *Character) IsImmortal() bool {
│       return ch.Level >= LevelImmortal
⋮
│func (ch *Character) IsAffected(flag AffectFlags) bool {
│       return ch.AffectedBy.Has(flag)
⋮
│func (ch *Character) GetEquipment(loc WearLocation) *Object {
│       if loc < 0 || loc >= WearLocMax {
│               return nil
│       }
│       return ch.Equipment[loc]
⋮
│func (ch *Character) AddInventory(obj *Object) {
│       obj.CarriedBy = ch
│       obj.InRoom = nil
│       obj.InObject = nil
│       ch.Inventory = append(ch.Inventory, obj)
⋮
│func (ch *Character) HasPenalty(flag PlayerFlags) bool {
│       return ch.PlayerAct.Has(flag)
⋮

go/pkg/types/classes.go:
⋮
│type Class struct {
│       Name         string // Class name
│       ShortName    string // Who list abbreviation
│       PrimeStat    int    // Primary attribute (StatStr, StatInt, etc.)
│       StartWeapon  int    // Starting weapon vnum
│       Guilds       [3]int // Guild room vnums
│       Thac0_00     int    // THAC0 at level 0
│       Thac0_32     int    // THAC0 at level 32
│       HPMin        int    // Minimum HP gain per level
│       HPMax        int    // Maximum HP gain per level
⋮
│func GetClass(index int) *Class {
│       if index >= 0 && index < len(ClassTable) {
│               return &ClassTable[index]
│       }
│       return nil
⋮
│func ClassName(classIndex int) string {
│       if c := GetClass(classIndex); c != nil {
│               return c.Name
│       }
│       return "unknown"
⋮

go/pkg/types/constants.go:
⋮
│type Direction int
│
⋮
│type Position int
│
⋮
│type Sex int
│
⋮
│type Size int
│
⋮
│type Sector int
│
⋮
│type ItemType int
│
⋮
│type WearLocation int
│
⋮
│type ApplyType int
│
⋮
│type DamageType int
│
⋮
│type WeaponClass int
│
⋮
│type ConnState int
│
⋮

go/pkg/types/descriptor.go:
⋮
│type EditorType int
│
⋮
│type Descriptor struct {
│       // Connection info
│       Host  string    // Hostname/IP of connection
│       State ConnState // Current connection state
│
│       // Character association
│       Character *Character // Character being played
│       Original  *Character // Original character (for switched immortals)
│
│       // Snooping
⋮
│func NewDescriptor(host string) *Descriptor {
│       return &Descriptor{
│               Host:  host,
│               State: ConGetName,
│       }
⋮

go/pkg/types/flags.go:
⋮
│type ActFlags uint64
│
⋮
│func (f ActFlags) Has(flag ActFlags) bool {
│       return f&flag != 0
⋮
│func (f *ActFlags) Set(flag ActFlags) {
│       *f |= flag
⋮
│type AffectFlags uint64
│
⋮
│func (f AffectFlags) Has(flag AffectFlags) bool {
│       return f&flag != 0
⋮
│func (f *AffectFlags) Set(flag AffectFlags) {
│       *f |= flag
⋮
│type ShieldFlags uint64
│
⋮
│func (f ShieldFlags) Has(flag ShieldFlags) bool {
│       return f&flag != 0
⋮
│func (f *ShieldFlags) Set(flag ShieldFlags) {
│       *f |= flag
⋮
│type RoomFlags uint32
│
⋮
│func (f RoomFlags) Has(flag RoomFlags) bool {
│       return f&flag != 0
⋮
│func (f *RoomFlags) Set(flag RoomFlags) {
│       *f |= flag
⋮
│type ItemFlags uint32
│
⋮
│func (f ItemFlags) Has(flag ItemFlags) bool {
│       return f&flag != 0
⋮
│func (f *ItemFlags) Set(flag ItemFlags) {
│       *f |= flag
⋮
│type WearFlags uint32
│
⋮
│func (f WearFlags) Has(flag WearFlags) bool {
│       return f&flag != 0
⋮
│func (f *WearFlags) Set(flag WearFlags) {
│       *f |= flag
⋮
│type ExitFlags uint32
│
⋮
│func (f ExitFlags) Has(flag ExitFlags) bool {
│       return f&flag != 0
⋮
│func (f *ExitFlags) Set(flag ExitFlags) {
│       *f |= flag
⋮
│type OffFlags uint32
│
⋮
│func (f OffFlags) Has(flag OffFlags) bool {
│       return f&flag != 0
⋮
│func (f *OffFlags) Set(flag OffFlags) {
│       *f |= flag
⋮
│type ImmFlags uint32
│
⋮
│func (f ImmFlags) Has(flag ImmFlags) bool {
│       return f&flag != 0
⋮
│func (f *ImmFlags) Set(flag ImmFlags) {
│       *f |= flag
⋮
│type CommFlags uint64
│
⋮
│type PlayerFlags uint64
│
⋮
│func (f CommFlags) Has(flag CommFlags) bool {
│       return f&flag != 0
⋮
│func (f *CommFlags) Set(flag CommFlags) {
│       *f |= flag
⋮
│func (f PlayerFlags) Has(flag PlayerFlags) bool {
│       return f&flag != 0
⋮
│func (f *PlayerFlags) Set(flag PlayerFlags) {
│       *f |= flag
⋮

go/pkg/types/object.go:
⋮
│type Object struct {
│       Vnum       int          // Virtual number (template ID)
│       Name       string       // Keywords for targeting
│       ShortDesc  string       // Short description (in inventory)
│       LongDesc   string       // Long description (on ground)
│       ItemType   ItemType     // Type of item (weapon, armor, etc.)
│       ExtraFlags ItemFlags    // Extra flags (glow, hum, magic, etc.)
│       WearFlags  WearFlags    // Where it can be worn/wielded
│       WearLoc    WearLocation // Current wear location (if equipped)
│       Weight     int          // Weight in pounds
⋮
│func NewObject(vnum int, shortDesc string, itemType ItemType) *Object {
│       return &Object{
│               Vnum:      vnum,
│               ShortDesc: shortDesc,
│               ItemType:  itemType,
│               Condition: 100,
│               Timer:     -1, // No timer by default
│               WearLoc:   WearLocNone,
│               Contents:  make([]*Object, 0),
│       }
⋮
│func (o *Object) IsExpired() bool {
│       return o.Timer == 0
⋮
│func (o *Object) DamageType() DamageType {
│       return DamageType(o.Values[3])
⋮
│func (o *Object) ContentsWeight() int {
│       total := 0
│       for _, item := range o.Contents {
│               total += item.Weight
│       }
│       return total
⋮
│func (o *Object) TotalWeight() int {
│       return o.Weight + o.ContentsWeight()
⋮

go/pkg/types/races.go:
⋮
│type Race struct {
│       Name            string        // Race name
│       ShortName       string        // Who list abbreviation
│       Points          int           // Creation point cost
│       ClassMultiplier [MaxClass]int // XP multiplier per class (100 = normal)
│       BonusSkills     []string      // Skills gained at creation
│       BaseStats       [MaxStats]int // Starting stats
│       MaxStats        [MaxStats]int // Maximum stats
│       Size            Size          // Race size
⋮
│func GetRace(index int) *Race {
│       if index >= 0 && index < len(RaceTable) {
│               return &RaceTable[index]
│       }
│       return nil
⋮

go/pkg/types/room.go:
⋮
│type ExtraDescription struct {
│       Keywords    []string // Keywords to match (e.g., ["statue", "marble"])
│       Description string   // What you see when you look at it
⋮
│type MobReset struct {
│       Vnum   int          // Mob template vnum
│       Max    int          // Max number in world (0 = unlimited)
│       Count  int          // Number to spawn (default 1)
│       Equips []EquipReset // Equipment to give to this mob
⋮
│type EquipReset struct {
│       Vnum    int          // Object template vnum
│       WearLoc WearLocation // Where to equip the item
│       Limit   int          // Max number in world (0 = unlimited)
│       InvOnly bool         // Put in inventory instead of equipping
⋮
│type ObjReset struct {
│       Vnum  int // Object template vnum
│       Max   int // Max number in world (0 = unlimited)
│       Count int // Number to spawn (default 1)
⋮
│type Room struct {
│       Vnum              int                 // Virtual number (unique ID)
│       Name              string              // Room name shown in look
│       Description       string              // Full room description
│       Flags             RoomFlags           // Room flags
│       Sector            Sector              // Terrain type
│       Exits             [6]*Exit            // Exits (indexed by Direction)
│       ExtraDescriptions []*ExtraDescription // Extra look targets
│       HealRate          int                 // HP regen rate modifier (100 = normal)
│       ManaRate          int                 // Mana regen rate modifier (100 = normal)
⋮
│func (r *Room) AddPerson(ch *Character) {
│       r.People = append(r.People, ch)
⋮
│func (r *Room) RemovePerson(ch *Character) {
│       for i, p := range r.People {
│               if p == ch {
│                       r.People = append(r.People[:i], r.People[i+1:]...)
│                       return
│               }
│       }
⋮

