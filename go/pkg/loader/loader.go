package loader

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"rotmud/pkg/types"
)

// mobCoinDrop computes the copper-piece purse a mob carries.
//
//	The TOML 'gold' field is the mob's wealth in copper, restored from the
//	ROT area files (ROT counted it in silver; reading it as copper keeps
//	purses small). 0 means the mob carries nothing. The purse is the wealth
//	randomised ±50%.
//
//	Mobs with no 'gold' field (new content) fall back to level × 10 copper,
//	casters 25% more, plain warriors 33% less.
func mobCoinDrop(tmpl *MobileData) int64 {
	var base int64
	if tmpl.Gold != nil {
		base = int64(*tmpl.Gold)
	} else {
		base = int64(tmpl.Level) * 10
		for _, f := range tmpl.ActFlags {
			switch strings.ToLower(f) {
			case "mage", "cleric":
				base = base * 5 / 4
			case "warrior":
				base = base * 2 / 3
			}
		}
	}
	if base <= 0 {
		return 0
	}
	// Random in [50%, 150%].
	lo := base / 2
	hi := base * 3 / 2
	if hi <= lo {
		return base
	}
	return lo + rand.Int63n(hi-lo+1)
}

// LoadConfigFromString parses config from TOML string
func LoadConfigFromString(data string) (*Config, error) {
	var cfg Config
	err := toml.Unmarshal([]byte(data), &cfg)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

// LoadConfigFromFile loads config from a file
func LoadConfigFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}
	return LoadConfigFromString(string(data))
}

// LoadAreaMetadataFromString parses area metadata from TOML string
func LoadAreaMetadataFromString(data string) (*AreaMetadata, error) {
	var area AreaMetadata
	err := toml.Unmarshal([]byte(data), &area)
	if err != nil {
		return nil, fmt.Errorf("parse area metadata: %w", err)
	}
	return &area, nil
}

// LoadRoomsFromString parses rooms from TOML string
func LoadRoomsFromString(data string) (*RoomsFile, error) {
	var rooms RoomsFile
	err := toml.Unmarshal([]byte(data), &rooms)
	if err != nil {
		return nil, fmt.Errorf("parse rooms: %w", err)
	}
	return &rooms, nil
}

// LoadMobilesFromString parses mobiles from TOML string
func LoadMobilesFromString(data string) (*MobilesFile, error) {
	var mobs MobilesFile
	err := toml.Unmarshal([]byte(data), &mobs)
	if err != nil {
		return nil, fmt.Errorf("parse mobiles: %w", err)
	}
	return &mobs, nil
}

// LoadObjectsFromString parses objects from TOML string
func LoadObjectsFromString(data string) (*ObjectsFile, error) {
	var objs ObjectsFile
	err := toml.Unmarshal([]byte(data), &objs)
	if err != nil {
		return nil, fmt.Errorf("parse objects: %w", err)
	}
	return &objs, nil
}

// LoadMOBprogFromString parses a mobprog from TOML string
func LoadMOBprogFromString(data string) (*MOBprogData, error) {
	var prog MOBprogData
	err := toml.Unmarshal([]byte(data), &prog)
	if err != nil {
		return nil, fmt.Errorf("parse mobprog: %w", err)
	}
	return &prog, nil
}

// LoadMOBprogsFromString parses multiple mobprogs from TOML string
func LoadMOBprogsFromString(data string) (*MOBprogFile, error) {
	var progs MOBprogFile
	err := toml.Unmarshal([]byte(data), &progs)
	if err != nil {
		return nil, fmt.Errorf("parse mobprogs file: %w", err)
	}
	return &progs, nil
}

// World holds all loaded game data
type World struct {
	Areas        []*types.Area
	Rooms        map[int]*types.Room
	MobTemplates map[int]*MobileData
	ObjTemplates map[int]*ObjectData
	Shops        map[int]*ShopData       // Shops by keeper mob vnum
	Forge        *ForgeConfig            // Weapon commission tiers (nil if no forge.toml)
	MOBprogs     map[string]*MOBprogData // MOBprogs by "area/name" key
}

// NewWorld creates a new empty world
func NewWorld() *World {
	return &World{
		Areas:        make([]*types.Area, 0),
		Rooms:        make(map[int]*types.Room),
		MobTemplates: make(map[int]*MobileData),
		ObjTemplates: make(map[int]*ObjectData),
		Shops:        make(map[int]*ShopData),
		MOBprogs:     make(map[string]*MOBprogData),
	}
}

// GetRoom returns a room by vnum
func (w *World) GetRoom(vnum int) *types.Room {
	return w.Rooms[vnum]
}

// GetMobTemplate returns a mob template by vnum
func (w *World) GetMobTemplate(vnum int) *MobileData {
	return w.MobTemplates[vnum]
}

// GetObjTemplate returns an object template by vnum
func (w *World) GetObjTemplate(vnum int) *ObjectData {
	return w.ObjTemplates[vnum]
}

// GetShop returns shop data for a mob by vnum
func (w *World) GetShop(keeperVnum int) *ShopData {
	return w.Shops[keeperVnum]
}

// GetAllShops returns all shops in the world
func (w *World) GetAllShops() map[int]*ShopData {
	return w.Shops
}

// GetMOBprog returns a mobprog by its full key (area/name)
func (w *World) GetMOBprog(key string) *MOBprogData {
	return w.MOBprogs[key]
}

// GetMOBprogsForMob returns all mobprogs associated with a mob template
func (w *World) GetMOBprogsForMob(mobVnum int) []*MOBprogData {
	tmpl := w.GetMobTemplate(mobVnum)
	if tmpl == nil {
		return nil
	}
	return tmpl.LoadedProgs
}

// AreaLoader loads areas from disk
type AreaLoader struct {
	BasePath string
}

// NewAreaLoader creates a new area loader
func NewAreaLoader(basePath string) *AreaLoader {
	return &AreaLoader{BasePath: basePath}
}

// LoadAll loads all areas from the base path
func (l *AreaLoader) LoadAll() (*World, error) {
	world := NewWorld()

	// Find all area directories
	entries, err := os.ReadDir(l.BasePath)
	if err != nil {
		return nil, fmt.Errorf("read areas directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		areaPath := filepath.Join(l.BasePath, entry.Name())
		if err := l.loadArea(areaPath, world); err != nil {
			return nil, fmt.Errorf("load area %s: %w", entry.Name(), err)
		}
	}

	// Resolve room exits
	l.resolveExits(world)

	// Price teleport items by the danger of their destination area.
	priceTeleportItems(world)

	// Weapon commission tiers live next to the areas directory.
	forgePath := filepath.Join(filepath.Dir(l.BasePath), "forge.toml")
	if data, err := os.ReadFile(forgePath); err == nil {
		var forge ForgeConfig
		if err := toml.Unmarshal(data, &forge); err != nil {
			return nil, fmt.Errorf("parse forge.toml: %w", err)
		}
		world.Forge = &forge
	}

	return world, nil
}

// priceTeleportItems sets the Cost of every portal/teleport object template from
// the average mob level of its destination area, so far-off or dangerous
// destinations cost more. Cost is in copper (base + level scaled).
func priceTeleportItems(world *World) {
	const base, perLevel = 50, 20
	for _, tmpl := range world.ObjTemplates {
		if tmpl.Portal == nil || tmpl.Portal.DestVnum <= 0 {
			continue
		}
		room := world.Rooms[tmpl.Portal.DestVnum]
		if room == nil || room.Area == nil {
			continue
		}
		var total, n int
		for v := room.Area.MinVnum; v <= room.Area.MaxVnum; v++ {
			if m := world.MobTemplates[v]; m != nil {
				total += m.Level
				n++
			}
		}
		avg := 0
		if n > 0 {
			avg = total / n
		}
		tmpl.Cost = base + avg*perLevel
	}
}

// loadArea loads a single area directory
func (l *AreaLoader) loadArea(areaPath string, world *World) error {
	// Load area metadata
	metaPath := filepath.Join(areaPath, "area.toml")
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read area.toml: %w", err)
	}

	meta, err := LoadAreaMetadataFromString(string(metaData))
	if err != nil {
		return err
	}

	area := types.NewArea(meta.Name, meta.VnumRange.Min, meta.VnumRange.Max)
	area.Filename = filepath.Base(areaPath)
	area.Credits = meta.Credits
	area.MaxLevel = meta.MaxLevel
	world.Areas = append(world.Areas, area)

	// Load mobprogs first (so they're available when loading mobs)
	mobprogsPath := filepath.Join(areaPath, "mobprogs")
	if err := l.loadMOBprogs(mobprogsPath, area.Filename, world); err != nil {
		return err
	}

	// Load rooms
	roomsPath := filepath.Join(areaPath, "rooms")
	if err := l.loadRooms(roomsPath, area, world); err != nil {
		return err
	}

	// Load mobs (and resolve mobprog references)
	mobsPath := filepath.Join(areaPath, "mobs")
	if err := l.loadMobs(mobsPath, world); err != nil {
		return err
	}

	// Resolve mobprog references for mobs in this area
	l.resolveMOBprogRefs(area.Filename, world)

	// Load objects
	objsPath := filepath.Join(areaPath, "objects")
	if err := l.loadObjects(objsPath, world); err != nil {
		return err
	}

	return nil
}

// loadRooms loads all room files from a directory
func (l *AreaLoader) loadRooms(roomsPath string, area *types.Area, world *World) error {
	entries, err := os.ReadDir(roomsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No rooms directory is ok
		}
		return fmt.Errorf("read rooms directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(roomsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		roomsFile, err := LoadRoomsFromString(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		for _, rd := range roomsFile.Rooms {
			room := l.convertRoom(&rd, area)
			world.Rooms[room.Vnum] = room
			area.Rooms[room.Vnum] = room
		}
	}

	return nil
}

// convertRoom converts TOML room data to a Room type
func (l *AreaLoader) convertRoom(rd *RoomData, area *types.Area) *types.Room {
	room := types.NewRoom(rd.Vnum, rd.Name, rd.Description)
	room.Area = area
	room.Sector = parseSector(rd.Sector)
	room.HealRate = rd.HealRate
	if room.HealRate == 0 {
		room.HealRate = 100
	}
	room.ManaRate = rd.ManaRate
	if room.ManaRate == 0 {
		room.ManaRate = 100
	}

	// Parse room flags
	for _, flag := range rd.RoomFlags {
		for _, rf := range roomFlagNames {
			if strings.EqualFold(flag, rf.name) {
				room.Flags.Set(rf.flag)
			}
		}
	}

	// Parse exits (ToRoom resolved later)
	for _, ed := range rd.Exits {
		dir := parseDirection(ed.Direction)
		if dir < 0 {
			continue
		}

		exit := types.NewExit(dir, ed.ToVnum)
		exit.Key = ed.Key
		exit.Keywords = ed.Keywords
		exit.Description = ed.Description

		for _, flag := range ed.Flags {
			switch strings.ToLower(flag) {
			case "door":
				exit.Flags.Set(types.ExitIsDoor) // open unless "closed" is also set
			case "closed":
				exit.Flags.Set(types.ExitIsDoor) // Closed implies door
				exit.Flags.Set(types.ExitClosed)
			case "locked":
				exit.Flags.Set(types.ExitIsDoor) // Locked implies door
				exit.Flags.Set(types.ExitClosed) // Locked implies closed
				exit.Flags.Set(types.ExitLocked)
			case "pickproof":
				exit.Flags.Set(types.ExitPickproof)
			case "open":
				// Explicit open flag removes closed state
				exit.Flags.Remove(types.ExitClosed)
			}
		}

		exit.ResetFlags = exit.Flags
		room.SetExit(dir, exit)
	}

	// Parse extra descriptions
	for _, ed := range rd.ExtraDescs {
		room.ExtraDescriptions = append(room.ExtraDescriptions, &types.ExtraDescription{
			Keywords:    ed.Keywords,
			Description: ed.Description,
		})
	}

	// Parse mob resets
	for _, mr := range rd.MobResets {
		count := mr.Count
		if count == 0 {
			count = 1
		}
		mobReset := types.MobReset{
			Vnum:  mr.Vnum,
			Max:   mr.Max,
			Count: count,
		}

		// Parse equipment resets for this mob
		for _, er := range mr.Equips {
			mobReset.Equips = append(mobReset.Equips, types.EquipReset{
				Vnum:     er.Vnum,
				WearLoc:  parseWearLocation(er.WearLoc),
				Limit:    er.Limit,
				InvOnly:  er.InvOnly,
				Contents: parseObjResets(er.Contents),
			})
		}

		room.MobResets = append(room.MobResets, mobReset)
	}

	room.ObjResets = parseObjResets(rd.ObjResets)
	room.RandomExits = rd.RandomExits

	return room
}

// parseObjResets converts object resets, and the resets for their contents.
func parseObjResets(data []ObjResetData) []types.ObjReset {
	var out []types.ObjReset
	for _, or := range data {
		out = append(out, types.ObjReset{
			Vnum:     or.Vnum,
			Max:      or.Max,
			Count:    max(or.Count, 1),
			Contents: parseObjResets(or.Contents),
		})
	}
	return out
}

// formatObjResets is parseObjResets in reverse, for saving rooms.
func formatObjResets(resets []types.ObjReset) []ObjResetData {
	var out []ObjResetData
	for _, or := range resets {
		out = append(out, ObjResetData{
			Vnum:     or.Vnum,
			Max:      or.Max,
			Count:    or.Count,
			Contents: formatObjResets(or.Contents),
		})
	}
	return out
}

// loadMobs loads all mobile files from a directory
func (l *AreaLoader) loadMobs(mobsPath string, world *World) error {
	entries, err := os.ReadDir(mobsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read mobs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(mobsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		mobsFile, err := LoadMobilesFromString(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		for i := range mobsFile.Mobiles {
			mob := &mobsFile.Mobiles[i]
			world.MobTemplates[mob.Vnum] = mob

			// Register shop if the mobile has shop data
			if mob.Shop != nil {
				world.Shops[mob.Vnum] = mob.Shop
			}
		}
	}

	return nil
}

// loadObjects loads all object files from a directory
func (l *AreaLoader) loadObjects(objsPath string, world *World) error {
	entries, err := os.ReadDir(objsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read objects directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(objsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		objsFile, err := LoadObjectsFromString(string(data))
		if err != nil {
			return fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		for i := range objsFile.Objects {
			obj := &objsFile.Objects[i]
			capGear(obj)
			world.ObjTemplates[obj.Vnum] = obj
		}
	}

	return nil
}

// loadMOBprogs loads all mobprog files from a directory
func (l *AreaLoader) loadMOBprogs(mobprogsPath string, areaName string, world *World) error {
	entries, err := os.ReadDir(mobprogsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No mobprogs directory is ok
		}
		return fmt.Errorf("read mobprogs directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		filePath := filepath.Join(mobprogsPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}

		// Get the prog name from the filename (without .toml extension)
		progName := strings.TrimSuffix(entry.Name(), ".toml")

		// Try parsing as a single prog first (simpler format)
		prog, err := LoadMOBprogFromString(string(data))
		if err == nil && prog.Trigger != "" {
			// Single prog format - use filename as the prog name if not specified
			if prog.Name == "" {
				prog.Name = progName
			}
			// Store with area-qualified key
			key := areaName + "/" + prog.Name
			world.MOBprogs[key] = prog
			continue
		}

		// Try parsing as a multi-prog file
		progsFile, err := LoadMOBprogsFromString(string(data))
		if err != nil {
			return fmt.Errorf("parse mobprog %s: %w", entry.Name(), err)
		}

		// Store each prog with its qualified key
		for i := range progsFile.Progs {
			p := &progsFile.Progs[i]
			if p.Name == "" {
				// Generate a name from filename + index
				p.Name = fmt.Sprintf("%s_%d", progName, i)
			}
			key := areaName + "/" + p.Name
			world.MOBprogs[key] = p
		}
	}

	return nil
}

// resolveMOBprogRefs links mob templates to their referenced mobprogs
func (l *AreaLoader) resolveMOBprogRefs(areaName string, world *World) {
	for _, mob := range world.MobTemplates {
		if len(mob.MOBprogs) == 0 {
			continue
		}

		mob.LoadedProgs = make([]*MOBprogData, 0, len(mob.MOBprogs))
		for _, progRef := range mob.MOBprogs {
			// Try area-qualified name first
			var prog *MOBprogData
			if strings.Contains(progRef, "/") {
				prog = world.MOBprogs[progRef]
			} else {
				// Try with current area prefix
				prog = world.MOBprogs[areaName+"/"+progRef]
			}

			if prog != nil {
				mob.LoadedProgs = append(mob.LoadedProgs, prog)
			}
		}
	}
}

// resolveExits links room exits to their target rooms
func (l *AreaLoader) resolveExits(world *World) {
	for _, room := range world.Rooms {
		for dir := types.Direction(0); dir < types.DirMax; dir++ {
			exit := room.GetExit(dir)
			if exit == nil {
				continue
			}
			exit.ToRoom = world.Rooms[exit.ToVnum]
		}
	}
}

// CreateMobFromTemplate creates a Character instance from a mob template
func (w *World) CreateMobFromTemplate(vnum int) *types.Character {
	tmpl := w.GetMobTemplate(vnum)
	if tmpl == nil {
		return nil
	}

	ch := types.NewNPC(vnum, strings.Join(tmpl.Keywords, " "), tmpl.Level)
	ch.ShortDesc = tmpl.ShortDesc
	ch.LongDesc = tmpl.LongDesc
	ch.Desc = tmpl.Description
	ch.Alignment = tmpl.Alignment
	ch.Coin = mobCoinDrop(tmpl)
	ch.HitRoll = tmpl.Hitroll

	// Shopkeepers can't be attacked: they carry large purses to buy stock,
	// and killing them would farm money.
	if w.Shops[vnum] != nil {
		ch.Act.Set(types.ActProtected)
	}

	// Parse sex
	switch strings.ToLower(tmpl.Sex) {
	case "male":
		ch.Sex = types.SexMale
	case "female":
		ch.Sex = types.SexFemale
	default:
		ch.Sex = types.SexNeutral
	}

	// Set HP from dice
	if tmpl.HitDice.Number > 0 && tmpl.HitDice.Size > 0 {
		baseHP := tmpl.HitDice.Number*(tmpl.HitDice.Size/2) + tmpl.HitDice.Bonus
		ch.MaxHit = baseHP
		ch.Hit = baseHP
	} else {
		// Default HP based on level
		ch.MaxHit = 20 + tmpl.Level*8
		ch.Hit = ch.MaxHit
	}

	// ROM create_mobile rolls mana; like HP above, take the dice average.
	if tmpl.ManaDice.Number > 0 && tmpl.ManaDice.Size > 0 {
		ch.MaxMana = tmpl.ManaDice.Number*(tmpl.ManaDice.Size/2) + tmpl.ManaDice.Bonus
		ch.Mana = ch.MaxMana
	}

	// Set damage dice
	if tmpl.DamageDice.Number > 0 {
		ch.Damage[0] = tmpl.DamageDice.Number
		ch.Damage[1] = tmpl.DamageDice.Size
		ch.Damage[2] = tmpl.DamageDice.Bonus
	} else {
		// Default damage
		ch.Damage[0] = 1 + tmpl.Level/5
		ch.Damage[1] = 4
		ch.Damage[2] = tmpl.Level / 3
	}

	for _, flag := range tmpl.ActFlags {
		ch.Act.Set(actFlagNames[strings.ToLower(flag)])
	}
	for _, flag := range tmpl.AffectedBy {
		ch.AffectedBy.Set(affFlagNames[strings.ToLower(flag)])
	}
	for _, flag := range tmpl.ShieldFlags {
		ch.ShieldedBy.Set(shieldFlagNames[strings.ToLower(flag)])
	}

	ch.Imm = parseImmFlags(tmpl.ImmFlags)
	ch.Res = parseImmFlags(tmpl.ResFlags)
	ch.Vuln = parseImmFlags(tmpl.VulnFlags)
	ch.DamType = types.DamageType(parseMobDamageType(tmpl.DamageType))
	ch.Size = parseSize(tmpl.Size)

	// Armour class: from the template (already in ROM's x10 units), else
	// ROM's old-format default by level.
	if tmpl.AC != [4]int{} {
		ch.Armor = tmpl.AC
	} else {
		for i := range 3 {
			ch.Armor[i] = interpolate(tmpl.Level, 100, -100)
		}
		ch.Armor[types.ACExotic] = interpolate(tmpl.Level, 100, 0)
	}

	ch.Off = parseOffFlags(tmpl.OffFlags)
	setMobStats(ch, ch.Off.Has(types.OffFast))
	applyMobFloor(ch)

	ch.StartPos = parseMobPosition(tmpl.StartPos)
	ch.DefaultPos = parseMobPosition(tmpl.DefaultPos)
	ch.Position = ch.StartPos

	// Set special function (e.g. spec_cast_adept)
	ch.Special = tmpl.Special

	// LLM dialog opt-in (Tier 1)
	ch.LLMEnabled = tmpl.LLMEnabled
	ch.LLMPersona = tmpl.LLMPersona

	return ch
}

// setMobStats gives a mob ROM create_mobile's stats: 11 + level/4 (max 25),
// adjusted for class act flags, OFF_FAST and size.
func setMobStats(ch *types.Character, fast bool) {
	base := min(25, 11+ch.Level/4)
	for i := range ch.PermStats {
		ch.PermStats[i] = base
	}
	adj := func(flag types.ActFlags, str, intel, wis, dex, con int) {
		if ch.Act.Has(flag) {
			ch.PermStats[types.StatStr] += str
			ch.PermStats[types.StatInt] += intel
			ch.PermStats[types.StatWis] += wis
			ch.PermStats[types.StatDex] += dex
			ch.PermStats[types.StatCon] += con
		}
	}
	adj(types.ActWarrior, 3, -1, 0, 0, 2)
	adj(types.ActThief, 0, 1, -1, 3, 0)
	adj(types.ActCleric, 1, 0, 3, -1, 0)
	adj(types.ActMage, -1, 3, 0, 1, 0)
	adj(types.ActRanger, 3, 1, 0, 0, -1)
	adj(types.ActDruid, -1, 0, 3, 1, 0)
	adj(types.ActVampire, 1, 0, -1, 0, 3)
	if fast {
		ch.PermStats[types.StatDex] += 2
	}
	ch.PermStats[types.StatStr] += int(ch.Size - types.SizeMedium)
	ch.PermStats[types.StatCon] += int(ch.Size-types.SizeMedium) / 2
}

// interpolate is ROM's linear level interpolation between level 0 and 32.
func interpolate(level, v00, v32 int) int {
	return v00 + level*(v32-v00)/32
}

var immFlagNames = map[string]types.ImmFlags{
	"summon": types.ImmSummon, "charm": types.ImmCharm, "magic": types.ImmMagic,
	"weapon": types.ImmWeapon, "bash": types.ImmBash, "pierce": types.ImmPierce,
	"slash": types.ImmSlash, "fire": types.ImmFire, "cold": types.ImmCold,
	"lightning": types.ImmLightning, "acid": types.ImmAcid, "poison": types.ImmPoison,
	"negative": types.ImmNegative, "holy": types.ImmHoly, "energy": types.ImmEnergy,
	"mental": types.ImmMental, "disease": types.ImmDisease, "drowning": types.ImmDrowning,
	"light": types.ImmLight, "sound": types.ImmSound, "silver": types.ImmSilver,
}

var actFlagNames = map[string]types.ActFlags{
	"npc": types.ActNPC, "sentinel": types.ActSentinel, "scavenger": types.ActScavenger,
	"key": types.ActKey, "ranger": types.ActRanger, "aggressive": types.ActAggressive,
	"stay_area": types.ActStayArea, "wimpy": types.ActWimpy, "pet": types.ActPet,
	"train": types.ActTrain, "practice": types.ActPractice, "druid": types.ActDruid,
	"no_body": types.ActNoBody, "nb_drop": types.ActNBDrop, "undead": types.ActUndead,
	"vampire": types.ActVampire, "cleric": types.ActCleric, "mage": types.ActMage,
	"thief": types.ActThief, "warrior": types.ActWarrior, "no_align": types.ActNoAlign,
	"no_purge": types.ActNoPurge, "outdoors": types.ActOutdoors, "is_satan": types.ActIsSatan,
	"indoors": types.ActIndoors, "is_priest": types.ActIsPriest, "healer": types.ActIsHealer,
	"is_healer": types.ActIsHealer, "gain": types.ActGain, "update_always": types.ActUpdateAlways,
	"changer": types.ActIsChanger, "banker": types.ActBanker, "protected": types.ActProtected,
}

var affFlagNames = map[string]types.AffectFlags{
	"blind": types.AffBlind, "invisible": types.AffInvisible, "detect_evil": types.AffDetectEvil,
	"detect_invis": types.AffDetectInvis, "detect_magic": types.AffDetectMagic,
	"detect_hidden": types.AffDetectHidden, "detect_good": types.AffDetectGood,
	"sanctuary": types.AffSanctuary, "faerie_fire": types.AffFaerieFire, "infrared": types.AffInfrared,
	"curse": types.AffCurse, "farsight": types.AffFarsight, "poison": types.AffPoison,
	"protect_evil": types.AffProtectEvil, "protect_good": types.AffProtectGood, "sneak": types.AffSneak,
	"hide": types.AffHide, "sleep": types.AffSleep, "charm": types.AffCharm, "flying": types.AffFlying,
	"pass_door": types.AffPassDoor, "haste": types.AffHaste, "calm": types.AffCalm,
	"plague": types.AffPlague, "weaken": types.AffWeaken, "dark_vision": types.AffDarkVision,
	"berserk": types.AffBerserk, "swim": types.AffSwim, "regeneration": types.AffRegeneration,
	"slow": types.AffSlow,
}

var shieldFlagNames = map[string]types.ShieldFlags{
	"ice": types.ShdIce, "fire": types.ShdFire, "shock": types.ShdShock,
}

// parseMobPosition reads a template position; a mob never spawns fighting.
func parseMobPosition(s string) types.Position {
	switch s {
	case "sleeping":
		return types.PosSleeping
	case "resting":
		return types.PosResting
	case "sitting":
		return types.PosSitting
	}
	return types.PosStanding
}

var offFlagNames = map[string]types.OffFlags{
	"area_attack": types.OffAreaAttack, "backstab": types.OffBackstab, "bash": types.OffBash,
	"berserk": types.OffBerserk, "disarm": types.OffDisarm, "dodge": types.OffDodge,
	"fade": types.OffFade, "fast": types.OffFast, "kick": types.OffKick, "kick_dirt": types.OffKickDirt,
	"parry": types.OffParry, "rescue": types.OffRescue, "tail": types.OffTail, "trip": types.OffTrip,
	"crush": types.OffCrush,
}

func parseOffFlags(names []string) (flags types.OffFlags) {
	for _, n := range names {
		flags.Set(offFlagNames[strings.ToLower(n)])
	}
	return flags
}

func parseImmFlags(names []string) (flags types.ImmFlags) {
	for _, n := range names {
		flags |= immFlagNames[n]
	}
	return flags
}

var mobDamageTypes = map[string]types.DamageType{
	"bash": types.DamBash, "pierce": types.DamPierce, "slash": types.DamSlash,
	"fire": types.DamFire, "cold": types.DamCold, "lightning": types.DamLightning,
	"acid": types.DamAcid, "poison": types.DamPoison, "negative": types.DamNegative,
	"holy": types.DamHoly, "energy": types.DamEnergy,
}

func parseMobDamageType(s string) types.DamageType {
	if t, ok := mobDamageTypes[s]; ok {
		return t
	}
	return types.DamNone
}

func parseSize(s string) types.Size {
	for i, n := range []string{"tiny", "small", "medium", "large", "huge", "giant"} {
		if s == n {
			return types.Size(i)
		}
	}
	return types.SizeMedium
}

// Helper functions

func parseDirection(s string) types.Direction {
	switch strings.ToLower(s) {
	case "north", "n":
		return types.DirNorth
	case "east", "e":
		return types.DirEast
	case "south", "s":
		return types.DirSouth
	case "west", "w":
		return types.DirWest
	case "up", "u":
		return types.DirUp
	case "down", "d":
		return types.DirDown
	}
	return -1
}

func parseSector(s string) types.Sector {
	switch strings.ToLower(s) {
	case "inside":
		return types.SectInside
	case "city":
		return types.SectCity
	case "field":
		return types.SectField
	case "forest":
		return types.SectForest
	case "hills":
		return types.SectHills
	case "mountain":
		return types.SectMountain
	case "water_swim", "swim":
		return types.SectWaterSwim
	case "water_noswim", "noswim":
		return types.SectWaterNoSwim
	case "air":
		return types.SectAir
	case "desert":
		return types.SectDesert
	}
	return types.SectInside
}

// parseWearLocation converts a wear location string to a WearLocation constant
func parseWearLocation(s string) types.WearLocation {
	switch strings.ToLower(s) {
	case "light":
		return types.WearLocLight
	case "finger_l", "finger1", "lfinger":
		return types.WearLocFingerL
	case "finger_r", "finger2", "rfinger":
		return types.WearLocFingerR
	case "neck1", "neck":
		return types.WearLocNeck1
	case "neck2":
		return types.WearLocNeck2
	case "body", "torso":
		return types.WearLocBody
	case "head":
		return types.WearLocHead
	case "legs":
		return types.WearLocLegs
	case "feet":
		return types.WearLocFeet
	case "hands":
		return types.WearLocHands
	case "arms":
		return types.WearLocArms
	case "shield", "offhand":
		return types.WearLocShield
	case "about", "cloak":
		return types.WearLocAbout
	case "waist", "belt":
		return types.WearLocWaist
	case "wrist_l", "wrist1", "lwrist":
		return types.WearLocWristL
	case "wrist_r", "wrist2", "rwrist":
		return types.WearLocWristR
	case "wield", "weapon":
		return types.WearLocWield
	case "hold", "held":
		return types.WearLocHold
	case "float", "floating":
		return types.WearLocFloat
	case "secondary", "dual":
		return types.WearLocSecondary
	case "face":
		return types.WearLocFace
	}
	return types.WearLocNone
}

// formatWearLocation converts a WearLocation constant to a string
func formatWearLocation(loc types.WearLocation) string {
	switch loc {
	case types.WearLocLight:
		return "light"
	case types.WearLocFingerL:
		return "finger_l"
	case types.WearLocFingerR:
		return "finger_r"
	case types.WearLocNeck1:
		return "neck1"
	case types.WearLocNeck2:
		return "neck2"
	case types.WearLocBody:
		return "body"
	case types.WearLocHead:
		return "head"
	case types.WearLocLegs:
		return "legs"
	case types.WearLocFeet:
		return "feet"
	case types.WearLocHands:
		return "hands"
	case types.WearLocArms:
		return "arms"
	case types.WearLocShield:
		return "shield"
	case types.WearLocAbout:
		return "about"
	case types.WearLocWaist:
		return "waist"
	case types.WearLocWristL:
		return "wrist_l"
	case types.WearLocWristR:
		return "wrist_r"
	case types.WearLocWield:
		return "wield"
	case types.WearLocHold:
		return "hold"
	case types.WearLocFloat:
		return "float"
	case types.WearLocSecondary:
		return "secondary"
	case types.WearLocFace:
		return "face"
	}
	return ""
}

// === TOML Save Functions for OLC Persistence ===

// SaveRoom saves a room to its area's rooms TOML file
func (w *World) SaveRoom(room *types.Room, basePath string) error {
	if room == nil {
		return fmt.Errorf("nil room")
	}

	// Find the area this room belongs to
	var areaDir string
	for _, area := range w.Areas {
		if room.Vnum >= area.MinVnum && room.Vnum <= area.MaxVnum {
			areaDir = filepath.Join(basePath, area.Filename)
			break
		}
	}
	if areaDir == "" {
		return fmt.Errorf("no area found for room vnum %d", room.Vnum)
	}

	// Create rooms directory if needed
	roomsDir := filepath.Join(areaDir, "rooms")
	if err := os.MkdirAll(roomsDir, 0755); err != nil {
		return fmt.Errorf("create rooms directory: %w", err)
	}

	// Convert room to RoomData
	rd := RoomData{
		Vnum:        room.Vnum,
		Name:        room.Name,
		Description: room.Description,
		Sector:      formatSector(room.Sector),
		RoomFlags:   formatRoomFlags(room.Flags),
		HealRate:    room.HealRate,
		ManaRate:    room.ManaRate,
	}

	// Convert exits
	for dir := types.Direction(0); dir < types.DirMax; dir++ {
		exit := room.GetExit(dir)
		if exit == nil {
			continue
		}
		ed := ExitData{
			Direction: formatDirection(dir),
			ToVnum:    exit.ToVnum,
			Key:       exit.Key,
			Keywords:  exit.Keywords,
			Flags:     formatExitFlags(exit.Flags),
		}
		rd.Exits = append(rd.Exits, ed)
	}

	// Convert extra descriptions
	for _, ed := range room.ExtraDescriptions {
		rd.ExtraDescs = append(rd.ExtraDescs, ExtraDescData{
			Keywords:    ed.Keywords,
			Description: ed.Description,
		})
	}

	// Convert mob resets
	for _, mr := range room.MobResets {
		mobResetData := MobResetData{
			Vnum:  mr.Vnum,
			Max:   mr.Max,
			Count: mr.Count,
		}

		// Convert equip resets for this mob
		for _, er := range mr.Equips {
			mobResetData.Equips = append(mobResetData.Equips, EquipResetData{
				Vnum:     er.Vnum,
				WearLoc:  formatWearLocation(er.WearLoc),
				Limit:    er.Limit,
				InvOnly:  er.InvOnly,
				Contents: formatObjResets(er.Contents),
			})
		}

		rd.MobResets = append(rd.MobResets, mobResetData)
	}

	rd.ObjResets = formatObjResets(room.ObjResets)
	rd.RandomExits = room.RandomExits

	// Determine filename based on room vnum
	filename := fmt.Sprintf("room_%d.toml", room.Vnum)
	filePath := filepath.Join(roomsDir, filename)

	// Check if file exists - if so, read existing content and update
	roomsFile := &RoomsFile{Rooms: []RoomData{rd}}

	// Write TOML
	data, err := toml.Marshal(roomsFile)
	if err != nil {
		return fmt.Errorf("marshal room: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write room file: %w", err)
	}

	return nil
}

// SaveMobile saves a mobile template to its area's mobs TOML file
func (w *World) SaveMobile(vnum int, template *MobileData, basePath string) error {
	if template == nil {
		return fmt.Errorf("nil template")
	}

	// Find the area this mob belongs to
	var areaDir string
	for _, area := range w.Areas {
		if vnum >= area.MinVnum && vnum <= area.MaxVnum {
			areaDir = filepath.Join(basePath, area.Filename)
			break
		}
	}
	if areaDir == "" {
		return fmt.Errorf("no area found for mob vnum %d", vnum)
	}

	// Create mobs directory if needed
	mobsDir := filepath.Join(areaDir, "mobs")
	if err := os.MkdirAll(mobsDir, 0755); err != nil {
		return fmt.Errorf("create mobs directory: %w", err)
	}

	// Determine filename
	filename := fmt.Sprintf("mob_%d.toml", vnum)
	filePath := filepath.Join(mobsDir, filename)

	// Create mobs file with this single mobile
	mobsFile := &MobilesFile{Mobiles: []MobileData{*template}}

	// Write TOML
	data, err := toml.Marshal(mobsFile)
	if err != nil {
		return fmt.Errorf("marshal mobile: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write mobile file: %w", err)
	}

	// Update in-memory template
	w.MobTemplates[vnum] = template

	return nil
}

// SaveObject saves an object template to its area's objects TOML file
func (w *World) SaveObject(vnum int, template *ObjectData, basePath string) error {
	if template == nil {
		return fmt.Errorf("nil template")
	}

	// Find the area this object belongs to
	var areaDir string
	for _, area := range w.Areas {
		if vnum >= area.MinVnum && vnum <= area.MaxVnum {
			areaDir = filepath.Join(basePath, area.Filename)
			break
		}
	}
	if areaDir == "" {
		return fmt.Errorf("no area found for object vnum %d", vnum)
	}

	// Create objects directory if needed
	objsDir := filepath.Join(areaDir, "objects")
	if err := os.MkdirAll(objsDir, 0755); err != nil {
		return fmt.Errorf("create objects directory: %w", err)
	}

	// Determine filename
	filename := fmt.Sprintf("obj_%d.toml", vnum)
	filePath := filepath.Join(objsDir, filename)

	// Create objects file with this single object
	objsFile := &ObjectsFile{Objects: []ObjectData{*template}}

	// Write TOML
	data, err := toml.Marshal(objsFile)
	if err != nil {
		return fmt.Errorf("marshal object: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write object file: %w", err)
	}

	// Update in-memory template
	w.ObjTemplates[vnum] = template

	return nil
}

// SaveArea saves area metadata
func (w *World) SaveArea(area *types.Area, basePath string) error {
	if area == nil {
		return fmt.Errorf("nil area")
	}

	areaDir := filepath.Join(basePath, area.Filename)

	// Create area directory if needed
	if err := os.MkdirAll(areaDir, 0755); err != nil {
		return fmt.Errorf("create area directory: %w", err)
	}

	// Create metadata
	meta := &AreaMetadata{
		ID:            area.Filename,
		Name:          area.Name,
		Credits:       area.Credits,
		ResetInterval: 120, // Default
		VnumRange: VnumRange{
			Min: area.MinVnum,
			Max: area.MaxVnum,
		},
	}

	// Write TOML
	filePath := filepath.Join(areaDir, "area.toml")
	data, err := toml.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal area: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write area file: %w", err)
	}

	return nil
}

// Helper functions for TOML formatting

func formatSector(s types.Sector) string {
	switch s {
	case types.SectInside:
		return "inside"
	case types.SectCity:
		return "city"
	case types.SectField:
		return "field"
	case types.SectForest:
		return "forest"
	case types.SectHills:
		return "hills"
	case types.SectMountain:
		return "mountain"
	case types.SectWaterSwim:
		return "water_swim"
	case types.SectWaterNoSwim:
		return "water_noswim"
	case types.SectAir:
		return "air"
	case types.SectDesert:
		return "desert"
	default:
		return "inside"
	}
}

func formatDirection(d types.Direction) string {
	switch d {
	case types.DirNorth:
		return "north"
	case types.DirEast:
		return "east"
	case types.DirSouth:
		return "south"
	case types.DirWest:
		return "west"
	case types.DirUp:
		return "up"
	case types.DirDown:
		return "down"
	default:
		return "north"
	}
}

// roomFlagNames maps room flags to their TOML names (ROT's room_flags table).
var roomFlagNames = []struct {
	name string
	flag types.RoomFlags
}{
	{"dark", types.RoomDark}, {"no_mob", types.RoomNoMob}, {"indoors", types.RoomIndoors},
	{"private", types.RoomPrivate}, {"safe", types.RoomSafe}, {"solitary", types.RoomSolitary},
	{"pet_shop", types.RoomPetShop}, {"no_recall", types.RoomNoRecall}, {"imp_only", types.RoomImpOnly},
	{"gods_only", types.RoomGodsOnly}, {"heroes_only", types.RoomHeroesOnly},
	{"newbies_only", types.RoomNewbiesOnly}, {"law", types.RoomLaw}, {"nowhere", types.RoomNowhere},
}

func formatRoomFlags(flags types.RoomFlags) []string {
	var result []string
	for _, rf := range roomFlagNames {
		if flags.Has(rf.flag) {
			result = append(result, rf.name)
		}
	}
	return result
}

func formatExitFlags(flags types.ExitFlags) []string {
	var result []string
	if flags.Has(types.ExitIsDoor) {
		result = append(result, "door")
	}
	if flags.Has(types.ExitClosed) {
		result = append(result, "closed")
	}
	if flags.Has(types.ExitLocked) {
		result = append(result, "locked")
	}
	if flags.Has(types.ExitPickproof) {
		result = append(result, "pickproof")
	}
	return result
}
