package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoad(t *testing.T) {
	t.Run("Load config from TOML", func(t *testing.T) {
		toml := `
[server]
telnet_port = 4000
websocket_port = 4001
pulse_ms = 250

[logging]
level = "info"
format = "json"
`
		cfg, err := LoadConfigFromString(toml)
		if err != nil {
			t.Fatalf("failed to load config: %v", err)
		}

		if cfg.Server.TelnetPort != 4000 {
			t.Errorf("expected telnet port 4000, got %d", cfg.Server.TelnetPort)
		}
		if cfg.Server.WebsocketPort != 4001 {
			t.Errorf("expected websocket port 4001, got %d", cfg.Server.WebsocketPort)
		}
		if cfg.Server.PulseMs != 250 {
			t.Errorf("expected pulse_ms 250, got %d", cfg.Server.PulseMs)
		}
		if cfg.Logging.Level != "info" {
			t.Errorf("expected log level 'info', got '%s'", cfg.Logging.Level)
		}
	})
}

func TestAreaMetadataLoad(t *testing.T) {
	t.Run("Load area metadata from TOML", func(t *testing.T) {
		toml := `
id = "midgaard"
name = "Midgaard"
credits = "Original ROM area"
reset_interval = 120

[vnum_range]
min = 3000
max = 3399

[level_range]
low = 1
high = 50
`
		area, err := LoadAreaMetadataFromString(toml)
		if err != nil {
			t.Fatalf("failed to load area: %v", err)
		}

		if area.ID != "midgaard" {
			t.Errorf("expected id 'midgaard', got '%s'", area.ID)
		}
		if area.Name != "Midgaard" {
			t.Errorf("expected name 'Midgaard', got '%s'", area.Name)
		}
		if area.VnumRange.Min != 3000 {
			t.Errorf("expected min vnum 3000, got %d", area.VnumRange.Min)
		}
		if area.VnumRange.Max != 3399 {
			t.Errorf("expected max vnum 3399, got %d", area.VnumRange.Max)
		}
	})
}

func TestRoomsLoad(t *testing.T) {
	t.Run("Load rooms from TOML", func(t *testing.T) {
		toml := `
[[rooms]]
vnum = 3001
name = "The Temple of Mota"
sector = "inside"
room_flags = ["safe"]
description = """
You are in the southern end of the temple hall."""

  [[rooms.exits]]
  direction = "north"
  to_vnum = 3005

[[rooms]]
vnum = 3005
name = "The Temple Altar"
sector = "inside"
room_flags = ["safe", "no_mob"]
description = "A large altar stands here."

  [[rooms.exits]]
  direction = "south"
  to_vnum = 3001
`
		rooms, err := LoadRoomsFromString(toml)
		if err != nil {
			t.Fatalf("failed to load rooms: %v", err)
		}

		if len(rooms.Rooms) != 2 {
			t.Fatalf("expected 2 rooms, got %d", len(rooms.Rooms))
		}

		room1 := rooms.Rooms[0]
		if room1.Vnum != 3001 {
			t.Errorf("expected vnum 3001, got %d", room1.Vnum)
		}
		if room1.Name != "The Temple of Mota" {
			t.Errorf("expected name 'The Temple of Mota', got '%s'", room1.Name)
		}
		if room1.Sector != "inside" {
			t.Errorf("expected sector 'inside', got '%s'", room1.Sector)
		}
		if len(room1.RoomFlags) != 1 || room1.RoomFlags[0] != "safe" {
			t.Errorf("expected room_flags ['safe'], got %v", room1.RoomFlags)
		}
		if len(room1.Exits) != 1 {
			t.Fatalf("expected 1 exit, got %d", len(room1.Exits))
		}
		if room1.Exits[0].Direction != "north" {
			t.Errorf("expected exit direction 'north', got '%s'", room1.Exits[0].Direction)
		}
		if room1.Exits[0].ToVnum != 3005 {
			t.Errorf("expected exit to_vnum 3005, got %d", room1.Exits[0].ToVnum)
		}
	})
}

func TestMobilesLoad(t *testing.T) {
	t.Run("Load mobiles from TOML", func(t *testing.T) {
		toml := `
[[mobiles]]
vnum = 3000
keywords = ["wizard", "old", "man"]
short_desc = "an old wizard"
long_desc = "An old wizard sits here, studying ancient tomes."
level = 46
sex = "male"
act_flags = ["npc", "sentinel", "stay_area"]
affected_by = ["detect_invis", "detect_hidden"]

  [mobiles.hit_dice]
  number = 10
  size = 10
  bonus = 500

  [mobiles.damage_dice]
  number = 5
  size = 6
  bonus = 10
`
		mobs, err := LoadMobilesFromString(toml)
		if err != nil {
			t.Fatalf("failed to load mobiles: %v", err)
		}

		if len(mobs.Mobiles) != 1 {
			t.Fatalf("expected 1 mobile, got %d", len(mobs.Mobiles))
		}

		mob := mobs.Mobiles[0]
		if mob.Vnum != 3000 {
			t.Errorf("expected vnum 3000, got %d", mob.Vnum)
		}
		if mob.Level != 46 {
			t.Errorf("expected level 46, got %d", mob.Level)
		}
		if mob.HitDice.Number != 10 {
			t.Errorf("expected hit dice number 10, got %d", mob.HitDice.Number)
		}
	})
}

func TestObjectsLoad(t *testing.T) {
	t.Run("Load objects from TOML", func(t *testing.T) {
		toml := `
[[objects]]
vnum = 3042
keywords = ["sword", "long", "steel"]
short_desc = "a long steel sword"
long_desc = "A finely crafted long sword lies here."
item_type = "weapon"
level = 10
weight = 5
cost = 300
wear_flags = ["take", "wield"]
extra_flags = ["glow", "magic"]

  [objects.weapon]
  weapon_type = "sword"
  dice_number = 2
  dice_size = 6
  damage_type = "slash"

[[objects]]
vnum = 3100
keywords = ["armor", "plate"]
short_desc = "a suit of plate armor"
item_type = "armor"
level = 15
weight = 50
cost = 1000
wear_flags = ["take", "body"]

  [objects.armor]
  ac_pierce = -30
  ac_bash = -30
  ac_slash = -30
  ac_exotic = -30
`
		objs, err := LoadObjectsFromString(toml)
		if err != nil {
			t.Fatalf("failed to load objects: %v", err)
		}

		if len(objs.Objects) != 2 {
			t.Fatalf("expected 2 objects, got %d", len(objs.Objects))
		}

		sword := objs.Objects[0]
		if sword.Vnum != 3042 {
			t.Errorf("expected vnum 3042, got %d", sword.Vnum)
		}
		if sword.ItemType != "weapon" {
			t.Errorf("expected item_type 'weapon', got '%s'", sword.ItemType)
		}
		if sword.Weapon == nil {
			t.Fatal("expected weapon data")
		}
		if sword.Weapon.DiceNumber != 2 {
			t.Errorf("expected dice number 2, got %d", sword.Weapon.DiceNumber)
		}

		armor := objs.Objects[1]
		if armor.Armor == nil {
			t.Fatal("expected armor data")
		}
		if armor.Armor.ACPierce != -30 {
			t.Errorf("expected ac_pierce -30, got %d", armor.Armor.ACPierce)
		}
	})
}

func TestAreaLoader(t *testing.T) {
	// Create a temporary test area
	tmpDir := t.TempDir()
	areaDir := filepath.Join(tmpDir, "areas", "testarea")
	os.MkdirAll(filepath.Join(areaDir, "rooms"), 0755)
	os.MkdirAll(filepath.Join(areaDir, "mobs"), 0755)
	os.MkdirAll(filepath.Join(areaDir, "objects"), 0755)

	// Write area.toml
	areaToml := `
id = "testarea"
name = "Test Area"
reset_interval = 60

[vnum_range]
min = 1000
max = 1099
`
	os.WriteFile(filepath.Join(areaDir, "area.toml"), []byte(areaToml), 0644)

	// Write rooms
	roomsToml := `
[[rooms]]
vnum = 1000
name = "Test Room"
sector = "inside"
description = "A test room."

  [[rooms.exits]]
  direction = "north"
  to_vnum = 1001

[[rooms]]
vnum = 1001
name = "North Room"
sector = "inside"
description = "The north room."

  [[rooms.exits]]
  direction = "south"
  to_vnum = 1000
`
	os.WriteFile(filepath.Join(areaDir, "rooms", "test.toml"), []byte(roomsToml), 0644)

	// Write mobs
	mobsToml := `
[[mobiles]]
vnum = 1000
keywords = ["guard"]
short_desc = "a guard"
level = 10
act_flags = ["npc", "sentinel"]
`
	os.WriteFile(filepath.Join(areaDir, "mobs", "guards.toml"), []byte(mobsToml), 0644)

	// Write objects
	objsToml := `
[[objects]]
vnum = 1000
keywords = ["key"]
short_desc = "a brass key"
item_type = "key"
level = 1
weight = 1
cost = 10
wear_flags = ["take"]
`
	os.WriteFile(filepath.Join(areaDir, "objects", "keys.toml"), []byte(objsToml), 0644)

	t.Run("Load complete area", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check area was loaded
		if len(world.Areas) != 1 {
			t.Fatalf("expected 1 area, got %d", len(world.Areas))
		}

		// Check rooms
		room := world.GetRoom(1000)
		if room == nil {
			t.Fatal("expected room 1000")
		}
		if room.Name != "Test Room" {
			t.Errorf("expected room name 'Test Room', got '%s'", room.Name)
		}

		// Check room connections resolved
		exit := room.GetExit(0) // north
		if exit == nil {
			t.Fatal("expected north exit")
		}
		if exit.ToRoom == nil {
			t.Fatal("expected exit to resolve to room")
		}
		if exit.ToRoom.Vnum != 1001 {
			t.Errorf("expected exit to room 1001, got %d", exit.ToRoom.Vnum)
		}

		// Check mob template
		mob := world.GetMobTemplate(1000)
		if mob == nil {
			t.Fatal("expected mob template 1000")
		}
		if mob.Level != 10 {
			t.Errorf("expected mob level 10, got %d", mob.Level)
		}

		// Check object template
		obj := world.GetObjTemplate(1000)
		if obj == nil {
			t.Fatal("expected object template 1000")
		}
		if obj.ItemType != "key" {
			t.Errorf("expected item_type 'key', got '%s'", obj.ItemType)
		}
	})
}

func TestShopLoading(t *testing.T) {
	// Create a temporary test area with shopkeeper
	tmpDir := t.TempDir()
	areaDir := filepath.Join(tmpDir, "areas", "shoparea")
	os.MkdirAll(filepath.Join(areaDir, "rooms"), 0755)
	os.MkdirAll(filepath.Join(areaDir, "mobs"), 0755)

	// Write area.toml
	areaToml := `
id = "shoparea"
name = "Shop Area"
reset_interval = 60

[vnum_range]
min = 2000
max = 2099
`
	os.WriteFile(filepath.Join(areaDir, "area.toml"), []byte(areaToml), 0644)

	// Write rooms
	roomsToml := `
[[rooms]]
vnum = 2000
name = "The Weapon Shop"
sector = "inside"
description = "A weapons shop."
`
	os.WriteFile(filepath.Join(areaDir, "rooms", "shop.toml"), []byte(roomsToml), 0644)

	// Write mobs with shop data
	mobsToml := `
[[mobiles]]
vnum = 2000
keywords = ["weaponsmith", "smith"]
short_desc = "the weaponsmith"
long_desc = "A burly weaponsmith stands here."
level = 45
sex = "male"
act_flags = ["npc", "sentinel", "no_purge"]

  [mobiles.shop]
  buy_types = ["weapon"]
  profit_buy = 50
  profit_sell = 150
  open_hour = 6
  close_hour = 22

[[mobiles]]
vnum = 2001
keywords = ["armorer"]
short_desc = "the armorer"
long_desc = "An armorer is here."
level = 45
sex = "male"
act_flags = ["npc", "sentinel"]

  [mobiles.shop]
  buy_types = ["armor", "clothing"]
  profit_buy = 75
  profit_sell = 125
  open_hour = 0
  close_hour = 24

[[mobiles]]
vnum = 2002
keywords = ["guard"]
short_desc = "the guard"
level = 30
act_flags = ["npc", "sentinel"]
`
	os.WriteFile(filepath.Join(areaDir, "mobs", "shopkeepers.toml"), []byte(mobsToml), 0644)

	t.Run("Load shops from mobile data", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check weaponsmith has shop
		shop1 := world.GetShop(2000)
		if shop1 == nil {
			t.Fatal("expected shop for mob 2000")
		}
		if len(shop1.BuyTypes) != 1 || shop1.BuyTypes[0] != "weapon" {
			t.Errorf("expected buy_types ['weapon'], got %v", shop1.BuyTypes)
		}
		if shop1.ProfitBuy != 50 {
			t.Errorf("expected profit_buy 50, got %d", shop1.ProfitBuy)
		}
		if shop1.ProfitSell != 150 {
			t.Errorf("expected profit_sell 150, got %d", shop1.ProfitSell)
		}
		if shop1.OpenHour != 6 {
			t.Errorf("expected open_hour 6, got %d", shop1.OpenHour)
		}
		if shop1.CloseHour != 22 {
			t.Errorf("expected close_hour 22, got %d", shop1.CloseHour)
		}

		// Check armorer has shop with multiple buy types
		shop2 := world.GetShop(2001)
		if shop2 == nil {
			t.Fatal("expected shop for mob 2001")
		}
		if len(shop2.BuyTypes) != 2 {
			t.Errorf("expected 2 buy types, got %d", len(shop2.BuyTypes))
		}
		// Check it's 24/7
		if shop2.OpenHour != 0 || shop2.CloseHour != 24 {
			t.Errorf("expected 24/7 shop, got %d-%d", shop2.OpenHour, shop2.CloseHour)
		}

		// Check guard has no shop
		shop3 := world.GetShop(2002)
		if shop3 != nil {
			t.Error("expected no shop for guard")
		}

		// Check all shops count
		allShops := world.GetAllShops()
		if len(allShops) != 2 {
			t.Errorf("expected 2 shops total, got %d", len(allShops))
		}
	})

	t.Run("Shops accessible via mob template", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Get mob template and verify shop is attached
		mob := world.GetMobTemplate(2000)
		if mob == nil {
			t.Fatal("expected mob template 2000")
		}
		if mob.Shop == nil {
			t.Fatal("expected shop attached to mob template")
		}
		if mob.Shop.ProfitBuy != 50 {
			t.Errorf("expected profit_buy 50, got %d", mob.Shop.ProfitBuy)
		}
	})
}

func TestMOBprogLoading(t *testing.T) {
	// Create a temporary test area with mobprogs
	tmpDir := t.TempDir()
	areaDir := filepath.Join(tmpDir, "areas", "testarea")
	os.MkdirAll(filepath.Join(areaDir, "rooms"), 0755)
	os.MkdirAll(filepath.Join(areaDir, "mobs"), 0755)
	os.MkdirAll(filepath.Join(areaDir, "mobprogs"), 0755)

	// Write area.toml
	areaToml := `
id = "testarea"
name = "Test Area"
reset_interval = 60

[vnum_range]
min = 5000
max = 5099
`
	os.WriteFile(filepath.Join(areaDir, "area.toml"), []byte(areaToml), 0644)

	// Write rooms
	roomsToml := `
[[rooms]]
vnum = 5000
name = "Test Room"
sector = "inside"
description = "A test room."
`
	os.WriteFile(filepath.Join(areaDir, "rooms", "test.toml"), []byte(roomsToml), 0644)

	// Write a simple single-prog mobprog file
	speechProgToml := `
name = "guard_greeting"
trigger = "speech"
trigger_arg = "hello"
commands = [
  "say Greetings, traveler!",
  "emote nods respectfully."
]
`
	os.WriteFile(filepath.Join(areaDir, "mobprogs", "guard_greeting.toml"), []byte(speechProgToml), 0644)

	// Write a multi-prog mobprog file
	multiProgToml := `
[[progs]]
name = "greet_player"
trigger = "greet"
trigger_arg = "50"
commands = [
  "say Welcome to my shop!",
  "emote smiles warmly."
]

[[progs]]
name = "fight_taunt"
trigger = "fight"
trigger_arg = "25"
commands = [
  "say You cannot defeat me!",
  "emote laughs menacingly."
]
`
	os.WriteFile(filepath.Join(areaDir, "mobprogs", "shopkeeper_progs.toml"), []byte(multiProgToml), 0644)

	// Write a death trigger prog
	deathProgToml := `
trigger = "death"
trigger_arg = "100"
commands = [
  "mpecho The beast lets out a final roar!",
  "mpecho A mysterious key drops from its grasp."
]
`
	os.WriteFile(filepath.Join(areaDir, "mobprogs", "boss_death.toml"), []byte(deathProgToml), 0644)

	// Write mobs with mobprog references
	mobsToml := `
[[mobiles]]
vnum = 5000
keywords = ["guard"]
short_desc = "the guard"
long_desc = "A guard stands here."
level = 30
act_flags = ["npc", "sentinel"]
mobprogs = ["guard_greeting"]

[[mobiles]]
vnum = 5001
keywords = ["shopkeeper"]
short_desc = "the shopkeeper"
long_desc = "A shopkeeper is here."
level = 45
act_flags = ["npc", "sentinel"]
mobprogs = ["greet_player", "fight_taunt"]

[[mobiles]]
vnum = 5002
keywords = ["boss", "dragon"]
short_desc = "the dragon boss"
long_desc = "A fearsome dragon guards the treasure."
level = 60
act_flags = ["npc", "aggressive"]
mobprogs = ["boss_death"]

[[mobiles]]
vnum = 5003
keywords = ["citizen"]
short_desc = "a citizen"
long_desc = "A citizen walks by."
level = 5
act_flags = ["npc"]
`
	os.WriteFile(filepath.Join(areaDir, "mobs", "mobs.toml"), []byte(mobsToml), 0644)

	t.Run("Load single mobprog file", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check mobprog was loaded with correct key
		prog := world.GetMOBprog("testarea/guard_greeting")
		if prog == nil {
			t.Fatal("expected mobprog testarea/guard_greeting")
		}
		if prog.Trigger != "speech" {
			t.Errorf("expected trigger 'speech', got '%s'", prog.Trigger)
		}
		if prog.TriggerArg != "hello" {
			t.Errorf("expected trigger_arg 'hello', got '%s'", prog.TriggerArg)
		}
		if len(prog.Commands) != 2 {
			t.Errorf("expected 2 commands, got %d", len(prog.Commands))
		}
	})

	t.Run("Load multi-prog file", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check first prog
		greetProg := world.GetMOBprog("testarea/greet_player")
		if greetProg == nil {
			t.Fatal("expected mobprog testarea/greet_player")
		}
		if greetProg.Trigger != "greet" {
			t.Errorf("expected trigger 'greet', got '%s'", greetProg.Trigger)
		}

		// Check second prog
		fightProg := world.GetMOBprog("testarea/fight_taunt")
		if fightProg == nil {
			t.Fatal("expected mobprog testarea/fight_taunt")
		}
		if fightProg.Trigger != "fight" {
			t.Errorf("expected trigger 'fight', got '%s'", fightProg.Trigger)
		}
	})

	t.Run("Load mobprog with name from filename", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// This prog had no name field, should use filename
		prog := world.GetMOBprog("testarea/boss_death")
		if prog == nil {
			t.Fatal("expected mobprog testarea/boss_death")
		}
		if prog.Trigger != "death" {
			t.Errorf("expected trigger 'death', got '%s'", prog.Trigger)
		}
	})

	t.Run("Mob with mobprog reference resolved", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check guard has its mobprog
		guard := world.GetMobTemplate(5000)
		if guard == nil {
			t.Fatal("expected mob template 5000")
		}
		if len(guard.LoadedProgs) != 1 {
			t.Fatalf("expected 1 loaded prog, got %d", len(guard.LoadedProgs))
		}
		if guard.LoadedProgs[0].Name != "guard_greeting" {
			t.Errorf("expected prog name 'guard_greeting', got '%s'", guard.LoadedProgs[0].Name)
		}
	})

	t.Run("Mob with multiple mobprog references", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check shopkeeper has both mobprogs
		shopkeeper := world.GetMobTemplate(5001)
		if shopkeeper == nil {
			t.Fatal("expected mob template 5001")
		}
		if len(shopkeeper.LoadedProgs) != 2 {
			t.Fatalf("expected 2 loaded progs, got %d", len(shopkeeper.LoadedProgs))
		}

		// Verify both progs are present
		progNames := make(map[string]bool)
		for _, prog := range shopkeeper.LoadedProgs {
			progNames[prog.Name] = true
		}
		if !progNames["greet_player"] {
			t.Error("expected greet_player prog")
		}
		if !progNames["fight_taunt"] {
			t.Error("expected fight_taunt prog")
		}
	})

	t.Run("Mob without mobprogs", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Check citizen has no mobprogs
		citizen := world.GetMobTemplate(5003)
		if citizen == nil {
			t.Fatal("expected mob template 5003")
		}
		if len(citizen.LoadedProgs) != 0 {
			t.Errorf("expected no loaded progs, got %d", len(citizen.LoadedProgs))
		}
	})

	t.Run("GetMOBprogsForMob helper", func(t *testing.T) {
		loader := NewAreaLoader(filepath.Join(tmpDir, "areas"))
		world, err := loader.LoadAll()
		if err != nil {
			t.Fatalf("failed to load areas: %v", err)
		}

		// Test the helper function
		progs := world.GetMOBprogsForMob(5001)
		if len(progs) != 2 {
			t.Fatalf("expected 2 progs, got %d", len(progs))
		}

		// Non-existent mob should return nil
		noProgs := world.GetMOBprogsForMob(99999)
		if noProgs != nil {
			t.Error("expected nil for non-existent mob")
		}
	})
}

func TestMOBprogParsing(t *testing.T) {
	t.Run("Parse single mobprog", func(t *testing.T) {
		toml := `
name = "test_prog"
trigger = "speech"
trigger_arg = "help"
commands = [
  "say How can I help you?",
  "emote looks at you curiously."
]
`
		prog, err := LoadMOBprogFromString(toml)
		if err != nil {
			t.Fatalf("failed to parse mobprog: %v", err)
		}

		if prog.Name != "test_prog" {
			t.Errorf("expected name 'test_prog', got '%s'", prog.Name)
		}
		if prog.Trigger != "speech" {
			t.Errorf("expected trigger 'speech', got '%s'", prog.Trigger)
		}
		if prog.TriggerArg != "help" {
			t.Errorf("expected trigger_arg 'help', got '%s'", prog.TriggerArg)
		}
		if len(prog.Commands) != 2 {
			t.Errorf("expected 2 commands, got %d", len(prog.Commands))
		}
	})

	t.Run("Parse mobprog file with multiple progs", func(t *testing.T) {
		toml := `
[[progs]]
name = "prog1"
trigger = "greet"
trigger_arg = "100"
commands = ["say Hello!"]

[[progs]]
name = "prog2"
trigger = "death"
trigger_arg = "50"
commands = ["mpecho It dies!", "mppurge"]
`
		progs, err := LoadMOBprogsFromString(toml)
		if err != nil {
			t.Fatalf("failed to parse mobprogs: %v", err)
		}

		if len(progs.Progs) != 2 {
			t.Fatalf("expected 2 progs, got %d", len(progs.Progs))
		}

		if progs.Progs[0].Name != "prog1" {
			t.Errorf("expected first prog name 'prog1', got '%s'", progs.Progs[0].Name)
		}
		if progs.Progs[1].Name != "prog2" {
			t.Errorf("expected second prog name 'prog2', got '%s'", progs.Progs[1].Name)
		}
	})
}
