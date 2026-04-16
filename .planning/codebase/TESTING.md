# Testing Patterns

**Analysis Date:** 2026-04-16

## Test Framework

**Runner:**
- Go's built-in `testing` package (no external test framework)
- Config: `/home/antti/Repos/Misc/ROT-MUD/.mise.toml` defines test task: `go test ./...`

**Assertion Library:**
- No external assertion library (stdlib `*testing.T` only)
- Manual assertions with `if` statements and `t.Error()`, `t.Errorf()`, `t.Fatal()`, `t.Fatalf()`

**Run Commands:**
```bash
go test ./...              # Run all tests in all packages
go test -v ./...           # Run with verbose output
go test -run TestName      # Run specific test
go test -race ./...        # Run with race detector
go test -cover ./...       # Show coverage
go test -coverprofile=coverage.out ./...  # Generate coverage report
```

## Test File Organization

**Location:**
- Co-located with implementation: test files in same package as code
- Pattern: `*.go` and `*_test.go` in same directory
- Examples:
  - `go/pkg/combat/combat.go` paired with `go/pkg/combat/combat_test.go`
  - `go/pkg/types/character.go` paired with `go/pkg/types/character_test.go`

**Naming:**
- Test files: `{module}_test.go`
- Test functions: `Test{FunctionName}` or `Test{TypeName}`
- Integration tests: `integration_test.go` (see `loader/integration_test.go`)

**Structure:**
```
go/pkg/
├── combat/
│   ├── combat.go
│   ├── combat_test.go           # Unit tests for combat system
│   ├── combat_sim_test.go        # Simulation/benchmarking tests
│   └── hit.go
├── types/
│   ├── character.go
│   ├── character_test.go         # Unit tests for character types
│   ├── affect_test.go
│   └── constants.go
├── loader/
│   ├── loader.go
│   ├── loader_test.go            # Unit tests for loader functions
│   └── integration_test.go        # Integration tests loading actual data files
└── game/
    ├── commands.go
    ├── commands_test.go
    └── handler_test.go
```

## Test Structure

**Suite Organization:**
```go
func TestCharacter(t *testing.T) {
	t.Run("NewCharacter creates character with correct values", func(t *testing.T) {
		ch := NewCharacter("Gandalf")
		if ch.Name != "Gandalf" {
			t.Errorf("expected name 'Gandalf', got '%s'", ch.Name)
		}
		if ch.Position != PosStanding {
			t.Errorf("expected position Standing, got %v", ch.Position)
		}
	})

	t.Run("Character stats work correctly", func(t *testing.T) {
		ch := NewCharacter("Test")
		ch.PermStats[StatStr] = 18
		ch.PermStats[StatInt] = 15
		ch.ModStats[StatStr] = 2 // Bonus from equipment/spell

		if ch.GetStat(StatStr) != 20 {
			t.Errorf("expected str 20 (18+2), got %d", ch.GetStat(StatStr))
		}
		if ch.GetStat(StatInt) != 15 {
			t.Errorf("expected int 15, got %d", ch.GetStat(StatInt))
		}
	})
}
```

**Patterns:**
- One test function per type/module with multiple `t.Run()` subtests
- Each subtest covers one behavior
- Descriptive subtest names: `"Dice returns value in range"`, `"Character stats work correctly"`, `"NewCharacter creates character with correct values"`
- Setup inline within test functions (minimal test fixtures)
- No test setup/teardown hooks; setup happens per-test

**Examples from test suite:**

Minimal assertion pattern from `combat_test.go`:
```go
func TestDice(t *testing.T) {
	t.Run("Dice returns value in range", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			result := Dice(2, 6)
			if result < 2 || result > 12 {
				t.Errorf("Dice(2,6) = %d, expected 2-12", result)
			}
		}
	})
}
```

State mutation pattern from `character_test.go`:
```go
func TestCharacter(t *testing.T) {
	t.Run("Character equipment slots", func(t *testing.T) {
		ch := NewCharacter("Test")
		sword := NewObject(3042, "a sword", ItemTypeWeapon)

		ch.Equip(sword, WearLocWield)
		if ch.GetEquipment(WearLocWield) != sword {
			t.Error("expected sword to be equipped in wield slot")
		}

		ch.Unequip(WearLocWield)
		if ch.GetEquipment(WearLocWield) != nil {
			t.Error("expected wield slot to be empty after unequip")
		}
	})
}
```

## Mocking

**Framework:** No external mocking library

**Patterns:**
- Dependency injection via function callbacks (for units that need mocking)
- Pass mock implementations as parameters
- Example from `combat.go`:
```go
type CombatSystem struct {
	Output      OutputFunc              // Callback for sending messages
	RoomFinder  RoomFinderFunc         // Callback to find rooms
	CharMover   CharMoverFunc          // Callback to move characters
	SkillGetter SkillGetterFunc        // Callback to get skill levels
	OnLevelUp   OnLevelUpFunc          // Callback when char levels up
	OnDamage    OnDamageFunc           // Callback when damage dealt
	OnKill      OnKillFunc             // Callback when kill happens
	OnDeath     OnDeathFunc            // Callback after death
}
```

- Tests create instances with nil callbacks or minimal implementations
- Example mock patterns from tests:
```go
// Combat system with nil callbacks (no output expected)
cs := combat.NewCombatSystem()
// cs.Output is nil - combat functions check if callbacks exist

// Or provide mock implementations:
var outputs []string
cs := combat.NewCombatSystem()
cs.Output = func(ch *types.Character, msg string) {
	outputs = append(outputs, msg)
}
// Now assertions can check outputs
```

**What to Mock:**
- I/O operations (file reads, network calls, database queries)
- Time-dependent operations (use fake time or time.Now() mocking)
- Callbacks that are optional for testing

**What NOT to Mock:**
- Core game logic types (Character, Room, Object, Affect)
- Pure computational functions (damage calculation, stat modifiers)
- Everything in `types/` package - these are domain objects, not mocks

## Fixtures and Factories

**Test Data:**
- No separate fixture files (inline in tests)
- Constructor patterns from `types/` used to build test objects:
```go
ch := NewCharacter("Test")
room := NewRoom(1, "Test", "Test room")
obj := NewObject(3042, "a sword", ItemTypeWeapon)
aff := NewAffect("sanctuary", 50, 10, ApplyNone, 0, AffSanctuary)
```

- Multi-object test setup:
```go
func TestStopFighting(t *testing.T) {
	t.Run("StopFighting with allInRoom clears others", func(t *testing.T) {
		room := types.NewRoom(1, "Test", "Test room")
		ch := types.NewCharacter("Attacker")
		victim := types.NewCharacter("Victim")

		ch.InRoom = room
		victim.InRoom = room
		room.AddPerson(ch)
		room.AddPerson(victim)

		SetFighting(ch, victim)
		SetFighting(victim, ch)

		StopFighting(ch, true)

		if victim.Fighting != nil {
			t.Error("Expected victim.Fighting to be nil")
		}
	})
}
```

**Location:**
- Test data created inline in tests (no separate factory modules)
- TOML fixtures used for config/data loading tests (see `help_test.go`):
```go
func TestSystemLoadFile(t *testing.T) {
	tmpDir := t.TempDir()
	helpFile := filepath.Join(tmpDir, "commands.toml")

	content := `
[[help]]
keywords = ["north", "n"]
syntax = "north"
description = "Move north to the next room."
`
	if err := os.WriteFile(helpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	// ... load and test
}
```

## Coverage

**Requirements:** No explicit coverage target (enforced via linting or CI checks)

**View Coverage:**
```bash
go test -cover ./...                    # Summary per package
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # HTML report in browser
go tool cover -func=coverage.out        # Function-level coverage
```

## Test Types

**Unit Tests:**
- Scope: Individual functions and methods
- Approach: Fast, isolated, no I/O or network
- Examples:
  - `combat_test.go`: Tests `Dice()`, `NumberRange()`, `SetFighting()`, `IsSafe()`
  - `character_test.go`: Tests character creation, stat calculation, equipment management
  - `affect_test.go`: Tests affect duration/expiration, affect list management
  - `types/flags_test.go`: Tests flag operations (Has, Set, Remove, Toggle)

**Integration Tests:**
- Scope: Multiple components working together
- Approach: Load actual TOML data files, verify connections
- Files: `*integration_test.go` (e.g., `loader/integration_test.go`)
- Example from `loader/integration_test.go`:
```go
func TestLoadMidgaard(t *testing.T) {
	dataPath := filepath.Join(filepath.Dir(filename), "..", "..", "data", "areas")

	t.Run("Load midgaard area from disk", func(t *testing.T) {
		loader := NewAreaLoader(dataPath)
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Verify areas, rooms, mobs, objects all loaded and linked
		if len(world.Rooms) < 1000 {
			t.Errorf("expected at least 1000 rooms, got %d", len(world.Rooms))
		}

		temple := world.GetRoom(3001)
		if temple == nil {
			t.Fatal("expected room 3001 (Temple of Thoth)")
		}

		northExit := temple.GetExit(0) // DirNorth
		if northExit == nil {
			t.Fatal("expected north exit from temple")
		}
		if northExit.ToRoom == nil {
			t.Fatal("expected north exit to resolve to room")
		}
		if northExit.ToRoom.Vnum != 3054 {
			t.Errorf("expected north exit to lead to 3054, got %d", northExit.ToRoom.Vnum)
		}
	})
}
```

**E2E Tests:**
- Not present in codebase
- Framework: None

## Common Patterns

**Async Testing:**
Not applicable (Go MUD is synchronous event loop)

**Error Testing:**
```go
func TestSystemLoadFileInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	badFile := filepath.Join(tmpDir, "bad.toml")

	// Invalid TOML
	if err := os.WriteFile(badFile, []byte("not valid { toml"), 0644); err != nil {
		t.Fatal(err)
	}

	sys := NewSystem()
	err := sys.LoadFile(badFile)
	if err == nil {
		t.Error("expected error loading invalid TOML")
	}
}
```

**Table-driven tests:**
Not observed in codebase; tests use individual `t.Run()` subtests instead

**Randomized property testing:**
Not used; deterministic tests only

**Looping for probabilistic functions:**
```go
func TestDice(t *testing.T) {
	t.Run("Dice returns value in range", func(t *testing.T) {
		for i := 0; i < 100; i++ {  // Run 100 times to catch occasional failures
			result := Dice(2, 6)
			if result < 2 || result > 12 {
				t.Errorf("Dice(2,6) = %d, expected 2-12", result)
			}
		}
	})
}
```

**File I/O testing:**
Uses `t.TempDir()` to create isolated temporary directories:
```go
func TestSystemLoadDir(t *testing.T) {
	tmpDir := t.TempDir()  // Auto-cleaned after test

	// Create multiple help files
	content1 := `[[help]]...`
	if err := os.WriteFile(filepath.Join(tmpDir, "movement.toml"), []byte(content1), 0644); err != nil {
		t.Fatal(err)
	}
	// ...
}
```

## Test File Coverage

**Current test files (26 total):**
- `combat/`: combat_test.go, combat_sim_test.go
- `types/`: affect_test.go, character_test.go, descriptor_test.go, flags_test.go, object_test.go, room_test.go
- `game/`: act_test.go, commands_test.go, handler_test.go, loop_test.go, mobprogs_test.go, notes_test.go, pets_test.go, stacking_test.go
- `loader/`: loader_test.go, integration_test.go
- `magic/`: magic_test.go
- `skills/`: skills_test.go
- `persistence/`: player_test.go
- `server/`: server_test.go
- `shops/`: shop_test.go
- `help/`: help_test.go
- `ai/`: specials_test.go
- `builder/`: olc_test.go

**Coverage gaps (untested packages):**
- No tests for: `ai/`, `builder/` - minimal test coverage relative to code complexity

---

*Testing analysis: 2026-04-16*
