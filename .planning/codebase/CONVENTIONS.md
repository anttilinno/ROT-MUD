# Coding Conventions

**Analysis Date:** 2026-04-16

## Naming Patterns

**Files:**
- Package files use lowercase with underscores: `combat.go`, `hit.go`, `commands.go`
- Test files use suffix `_test.go`: `combat_test.go`, `character_test.go`
- Doc files use `doc.go` inside packages for package-level documentation
- Constant/type definition files: `constants.go`, `flags.go`, `classes.go`, `races.go`

**Functions:**
- Public functions use PascalCase: `NewCharacter()`, `SetFighting()`, `IsAwake()`, `GetSkill()`
- Private functions use camelCase: `send()`, `sendPositionMessage()`, `formatObjectList()`
- Constructor functions use `New` prefix: `NewCharacter()`, `NewRoom()`, `NewObject()`, `NewAffect()`
- Getter methods use `Get` prefix: `GetStat()`, `GetEquipment()`, `GetExit()`, `GetRoom()`
- Boolean methods use `Is` or `Has` prefix: `IsNPC()`, `IsAwake()`, `IsExpired()`, `Has()`, `IsSafe()`
- Callback function types use `Func` suffix: `OutputFunc`, `RoomFinderFunc`, `CharMoverFunc`, `SkillGetterFunc`, `OnLevelUpFunc`, `OnDamageFunc`, `OnKillFunc`, `OnDeathFunc`

**Variables:**
- Local variables use camelCase: `ch`, `victim`, `damage`, `result`, `names`
- Character aliases follow MUD convention: `ch` (character), `victim`, `room`, `obj` (object), `items`, `people`
- Loop indices: `i`, `j` for simple numeric loops
- Config/struct fields use PascalCase: `Level`, `Name`, `Armor`, `MaxHit`, `Position`, `Fighting`

**Types:**
- Type names use PascalCase: `Character`, `Room`, `Object`, `Affect`, `Direction`, `Position`, `ItemType`
- Flag/enum types use PascalCase: `ActFlags`, `AffectFlags`, `ShieldFlags`, `CommFlags`, `PlayerFlags`, `ImmFlags`
- Constant enum values use Prefix convention: `DirNorth`, `DirEast`, `PosStanding`, `PosFighting`, `ItemTypeWeapon`, `WearLocWield`
- Type aliases use descriptive names: `Stat = int` for stat indices
- Struct field names match MUD domain: `PCData` (Player Character Data), `Hit` (HP), `Mana`, `Move`, `Alignment`, `Wimpy`, `Training`

## Code Style

**Formatting:**
- No explicit formatter configured (gofmt defaults apply)
- Line length: conventionally ~80-120 characters based on observed code
- Indentation: tabs (Go standard)
- Brace style: opening brace on same line (Go standard)

**Linting:**
- No `.golangci.yml` present; project follows Go conventions without additional linting
- Follows Go naming conventions strictly
- Unused variables are avoided or explicitly assigned to `_`

## Import Organization

**Order:**
1. Standard library imports: `fmt`, `os`, `testing`, `time`, `net`, `bufio`, `sync`, `crypto/sha256`, `encoding/hex`, `strings`, `strconv`, `path/filepath`, `log/slog`
2. Third-party imports: `github.com/gorilla/websocket`, `github.com/pelletier/go-toml/v2`, `github.com/prometheus/client_golang`
3. Local package imports: `rotmud/pkg/types`, `rotmud/pkg/combat`, `rotmud/pkg/game`, etc.

**Path Aliases:**
- Not used in observed code
- Full import paths always used: `"rotmud/pkg/types"`, `"rotmud/pkg/combat"`

**Examples:**
```go
import (
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pelletier/go-toml/v2"

	"rotmud/pkg/ai"
	"rotmud/pkg/builder"
	"rotmud/pkg/combat"
	"rotmud/pkg/game"
	"rotmud/pkg/types"
)
```

## Error Handling

**Patterns:**
- Functions that perform I/O or parsing return `(result, error)` tuple
- Error wrapping uses `fmt.Errorf()` with `%w` placeholder: `return nil, fmt.Errorf("parse config: %w", err)`
- Error messages are lowercase and context-prefixed: `"read config file: %w"`, `"parse area metadata: %w"`
- No panic usage in game logic
- Logging uses `slog` (structured logging): `logger.Error("server error", "error", err)`

**Examples from `loader.go`:**
```go
func LoadConfigFromString(data string) (*Config, error) {
	var cfg Config
	err := toml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func (l *AreaLoader) loadArea(areaPath string, world *World) error {
	return fmt.Errorf("read area.toml: %w", err)
}
```

## Logging

**Framework:** `log/slog` (Go 1.21+ structured logging)

**Patterns:**
- Server startup uses text format with `slog.NewTextHandler`
- Error logging includes context: `logger.Error("server error", "error", err)`
- Info level for normal operations: `logger.Info("client connected", "addr", conn.RemoteAddr())`
- Structured key-value pairs for context: `logger.Error("msg", "key", value, "key2", value2)`

**Example from `main.go`:**
```go
logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelInfo,
}))
srv := server.New(logger)
if err := srv.Start(4000); err != nil {
	logger.Error("server error", "error", err)
	os.Exit(1)
}
```

## Comments

**When to Comment:**
- Type definitions with domain-specific meaning include doc comments
- Public functions have one-line doc comments: `// NewCharacter creates a new character`
- Complex algorithms or non-obvious logic include inline comments
- References to original C source included for maintainability: `// Based on CHAR_DATA from merc.h:1585-1720`

**JSDoc/TSDoc:**
- Not applicable (Go uses different conventions)
- Package-level documentation uses `package` statement with comment block
- Doc comments appear immediately before the declaration they document

**Examples:**
```go
// Affect represents a temporary effect on a character
// Based on AFFECT_DATA from merc.h:566-577
type Affect struct {
	Type         string      // Spell/skill name
	Level        int         // Caster level
	Duration     int         // Ticks remaining (-1 for permanent)
	Location     ApplyType   // What stat to modify
	Modifier     int         // How much to modify
	BitVector    AffectFlags // Flags to set (e.g., AffSanctuary)
	ShieldVector ShieldFlags // Shield flags to set (e.g., ShdFire, ShdIce)
}

// NewAffect creates a new affect
func NewAffect(spellType string, level, duration int, location ApplyType, modifier int, bits AffectFlags) *Affect {
	return &Affect{...}
}

// IsExpired returns true if the affect has expired
// Permanent affects (duration -1) never expire
func (a *Affect) IsExpired() bool {
	return a.Duration == 0
}
```

## Function Design

**Size:** Functions are generally 10-50 lines; larger functions split into helpers. See `combat.go` for combat calculation functions (~50 lines), `commands.go` for command handlers (20-100 lines depending on complexity).

**Parameters:**
- Functions accept specific types rather than interfaces where possible (e.g., `func SetFighting(ch, victim *types.Character)`)
- Callback functions injected via struct fields for dependency injection: `CombatSystem` struct has `Output`, `RoomFinder`, `CharMover`, `SkillGetter`, `OnLevelUp`, `OnDamage`, `OnKill`, `OnDeath` fields
- Context not used (game is synchronous)

**Return Values:**
- Single value for pure functions: `func (d Direction) String() string`
- Tuple `(value, error)` for fallible operations: `func LoadConfigFromFile(path string) (*Config, error)`
- Pointer receivers for mutating methods: `func (f *ActFlags) Set(flag ActFlags)`
- No error-as-second-return for purely informational failures (e.g., finding character by name returns `*Character, bool` or just `*Character`)

**Examples:**
```go
// Pure function
func (d Direction) Reverse() Direction {
	switch d {
	case DirNorth: return DirSouth
	case DirSouth: return DirNorth
	// ...
	}
	return d
}

// Fallible I/O operation
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return LoadConfigFromString(string(data))
}

// Mutating method (pointer receiver)
func (f *ActFlags) Set(flag ActFlags) {
	*f |= flag
}

// Callback injection
func (c *CombatSystem) GetSkill(ch *types.Character, skillName string) int {
	if c.SkillGetter != nil {
		return c.SkillGetter(ch, skillName)
	}
	return 20 + ch.Level*2
}
```

## Module Design

**Exports:**
- Packages export types and functions needed by other packages (public with PascalCase)
- Internal functions start with lowercase (private)
- Struct fields that are part of public API use PascalCase
- Unexported fields use lowercase: `type AffectList struct { affects []*Affect }`

**Barrel Files:**
- Not used; each package is independently importable
- `doc.go` files provide package-level documentation

**Package Organization:**
- `rotmud/pkg/types/` - Core types (Character, Room, Object, Affect, flags, constants)
- `rotmud/pkg/combat/` - Combat system logic
- `rotmud/pkg/game/` - Game loop, commands, command dispatcher
- `rotmud/pkg/magic/` - Spell system
- `rotmud/pkg/skills/` - Skill system
- `rotmud/pkg/loader/` - TOML data loading
- `rotmud/pkg/persistence/` - Player save/load
- `rotmud/pkg/server/` - TCP/WebSocket server
- `rotmud/pkg/help/` - Help system
- `rotmud/pkg/builder/` - In-game building commands (OLC)
- `rotmud/pkg/ai/` - NPC AI
- `rotmud/pkg/shops/` - Shop system

**Example exports from `types/character.go`:**
```go
// Character represents a player or NPC (exported)
type Character struct {
	Name      string // Exported field
	Level     int
	Class     int
	// ...
	PCData     *PCData     // Exported field
	Descriptor *Descriptor // Exported field
	Deleted    bool        // Exported field
}

// NewCharacter creates a new character (exported function)
func NewCharacter(name string) *Character {
	return &Character{...}
}

// IsNPC returns true if character is a mobile (exported function)
func (c *Character) IsNPC() bool {
	return c.Act.Has(ActNPC)
}
```

---

*Convention analysis: 2026-04-16*
