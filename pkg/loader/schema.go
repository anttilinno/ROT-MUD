package loader

// Config represents server configuration
type Config struct {
	Server   ServerConfig   `toml:"server"`
	Logging  LoggingConfig  `toml:"logging"`
	Security SecurityConfig `toml:"security"`
}

type ServerConfig struct {
	TelnetPort    int    `toml:"telnet_port"`
	WebsocketPort int    `toml:"websocket_port"`
	PulseMs       int    `toml:"pulse_ms"`
	DataPath      string `toml:"data_path"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

type SecurityConfig struct {
	APIKey string `toml:"api_key"`
}

// AreaMetadata represents area.toml
type AreaMetadata struct {
	ID            string     `toml:"id"`
	Name          string     `toml:"name"`
	Credits       string     `toml:"credits"`
	ResetInterval int        `toml:"reset_interval"`
	VnumRange     VnumRange  `toml:"vnum_range"`
	LevelRange    LevelRange `toml:"level_range"`
}

type VnumRange struct {
	Min int `toml:"min"`
	Max int `toml:"max"`
}

type LevelRange struct {
	Low  int `toml:"low"`
	High int `toml:"high"`
}

// RoomsFile represents a rooms/*.toml file
type RoomsFile struct {
	Rooms []RoomData `toml:"rooms"`
}

type RoomData struct {
	Vnum        int             `toml:"vnum"`
	Name        string          `toml:"name"`
	Description string          `toml:"description"`
	Sector      string          `toml:"sector"`
	RoomFlags   []string        `toml:"room_flags"`
	HealRate    int             `toml:"heal_rate"`
	ManaRate    int             `toml:"mana_rate"`
	Exits       []ExitData      `toml:"exits"`
	ExtraDescs  []ExtraDescData `toml:"extra_descs"`
	MobResets   []MobResetData  `toml:"mob_resets"`
	ObjResets   []ObjResetData  `toml:"obj_resets"`
}

// MobResetData defines a mob that spawns in a room
type MobResetData struct {
	Vnum  int `toml:"vnum"`  // Mob template vnum
	Max   int `toml:"max"`   // Max number in world (0 = unlimited)
	Count int `toml:"count"` // Number to spawn (default 1)
}

// ObjResetData defines an object that spawns in a room
type ObjResetData struct {
	Vnum  int `toml:"vnum"`  // Object template vnum
	Max   int `toml:"max"`   // Max number in world (0 = unlimited)
	Count int `toml:"count"` // Number to spawn (default 1)
}

type ExitData struct {
	Direction   string   `toml:"direction"`
	ToVnum      int      `toml:"to_vnum"`
	Key         int      `toml:"key"`
	Keywords    string   `toml:"keywords"`
	Description string   `toml:"description"`
	Flags       []string `toml:"flags"`
}

type ExtraDescData struct {
	Keywords    []string `toml:"keywords"`
	Description string   `toml:"description"`
}

// MobilesFile represents a mobs/*.toml file
type MobilesFile struct {
	Mobiles []MobileData `toml:"mobiles"`
}

type MobileData struct {
	Vnum        int       `toml:"vnum"`
	Keywords    []string  `toml:"keywords"`
	ShortDesc   string    `toml:"short_desc"`
	LongDesc    string    `toml:"long_desc"`
	Description string    `toml:"description"`
	Level       int       `toml:"level"`
	Sex         string    `toml:"sex"`
	Race        string    `toml:"race"`
	Alignment   int       `toml:"alignment"`
	ActFlags    []string  `toml:"act_flags"`
	AffectedBy  []string  `toml:"affected_by"`
	OffFlags    []string  `toml:"off_flags"`
	ImmFlags    []string  `toml:"imm_flags"`
	ResFlags    []string  `toml:"res_flags"`
	VulnFlags   []string  `toml:"vuln_flags"`
	HitDice     DiceData  `toml:"hit_dice"`
	ManaDice    DiceData  `toml:"mana_dice"`
	DamageDice  DiceData  `toml:"damage_dice"`
	DamageType  string    `toml:"damage_type"`
	AC          [4]int    `toml:"ac"`
	Hitroll     int       `toml:"hitroll"`
	Gold        int       `toml:"gold"`
	StartPos    string    `toml:"start_pos"`
	DefaultPos  string    `toml:"default_pos"`
	Special     string    `toml:"special"` // Special behavior function name (e.g. "spec_cast_mage")
	Shop        *ShopData `toml:"shop"`

	// MOBprogs references mobprog files by name (loaded from mobprogs/ directory)
	MOBprogs []string `toml:"mobprogs"`

	// LoadedProgs holds the actual loaded mobprog data (populated during area loading)
	LoadedProgs []*MOBprogData `toml:"-"`
}

// ShopData defines a shop for a mobile
type ShopData struct {
	BuyTypes   []string `toml:"buy_types"`   // Item types the shop buys
	ProfitBuy  int      `toml:"profit_buy"`  // Buy profit margin (100 = base)
	ProfitSell int      `toml:"profit_sell"` // Sell profit margin (100 = base)
	OpenHour   int      `toml:"open_hour"`   // Hour shop opens
	CloseHour  int      `toml:"close_hour"`  // Hour shop closes
}

type DiceData struct {
	Number int `toml:"number"`
	Size   int `toml:"size"`
	Bonus  int `toml:"bonus"`
}

// ObjectsFile represents an objects/*.toml file
type ObjectsFile struct {
	Objects []ObjectData `toml:"objects"`
}

type ObjectData struct {
	Vnum       int             `toml:"vnum"`
	Keywords   []string        `toml:"keywords"`
	ShortDesc  string          `toml:"short_desc"`
	LongDesc   string          `toml:"long_desc"`
	ItemType   string          `toml:"item_type"`
	Level      int             `toml:"level"`
	Weight     int             `toml:"weight"`
	Cost       int             `toml:"cost"`
	Condition  int             `toml:"condition"`
	Material   string          `toml:"material"`
	WearFlags  []string        `toml:"wear_flags"`
	ExtraFlags []string        `toml:"extra_flags"`
	Weapon     *WeaponData     `toml:"weapon"`
	Armor      *ArmorData      `toml:"armor"`
	Container  *ContainerData  `toml:"container"`
	Light      *LightData      `toml:"light"`
	Affects    []AffectData    `toml:"affects"`
	ExtraDescs []ExtraDescData `toml:"extra_descs"`
}

type WeaponData struct {
	WeaponType string   `toml:"weapon_type"`
	DiceNumber int      `toml:"dice_number"`
	DiceSize   int      `toml:"dice_size"`
	DamageType string   `toml:"damage_type"`
	Flags      []string `toml:"flags"`
}

type ArmorData struct {
	ACPierce int `toml:"ac_pierce"`
	ACBash   int `toml:"ac_bash"`
	ACSlash  int `toml:"ac_slash"`
	ACExotic int `toml:"ac_exotic"`
}

type ContainerData struct {
	Capacity  int      `toml:"capacity"`
	Flags     []string `toml:"flags"`
	Key       int      `toml:"key"`
	MaxWeight int      `toml:"max_weight"`
}

type LightData struct {
	Duration int `toml:"duration"`
}

type AffectData struct {
	Location string `toml:"location"`
	Modifier int    `toml:"modifier"`
}

// MOBprogData represents a single mobprog definition in TOML
// Stored in mobprogs/<name>.toml files within an area directory
type MOBprogData struct {
	// Name is the unique identifier for this mobprog within the area
	// Derived from filename if not specified
	Name string `toml:"name"`

	// Trigger type: speech, act, fight, death, entry, greet, give, bribe, etc.
	Trigger string `toml:"trigger"`

	// TriggerArg is the argument for the trigger:
	// - For speech/act: phrase to match
	// - For bribe: minimum amount
	// - For give: object name or vnum
	// - For percentage-based triggers (greet, entry, fight, death): percent chance (e.g. "25" for 25%)
	TriggerArg string `toml:"trigger_arg"`

	// Commands to execute when the trigger fires
	// These can include MUD commands and special MOBprog commands:
	// - say, emote, yell, etc. - normal commands
	// - mpecho - echo to room
	// - mptransfer - transfer player
	// - mpforce - force player to do something
	// - mpkill - attack target
	// - mpgoto - go to room
	// - mpjunk - destroy item
	// - mppurge - purge mob/object
	// - mpat - execute at another location
	Commands []string `toml:"commands"`
}

// MOBprogFile represents a mobprogs/*.toml file that can contain multiple progs
type MOBprogFile struct {
	Progs []MOBprogData `toml:"progs"`
}

// MobProgRef references a mobprog by name for association with a mob
type MobProgRef struct {
	Name string `toml:"name"` // Reference to a mobprog file (without .toml extension)
}
