package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"rotmud/pkg/types"
)

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

	return world, nil
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
		switch strings.ToLower(flag) {
		case "dark":
			room.Flags.Set(types.RoomDark)
		case "no_mob":
			room.Flags.Set(types.RoomNoMob)
		case "indoors":
			room.Flags.Set(types.RoomIndoors)
		case "private":
			room.Flags.Set(types.RoomPrivate)
		case "safe":
			room.Flags.Set(types.RoomSafe)
		case "solitary":
			room.Flags.Set(types.RoomSolitary)
		case "no_recall":
			room.Flags.Set(types.RoomNoRecall)
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
				exit.Flags.Set(types.ExitIsDoor)
			case "closed":
				exit.Flags.Set(types.ExitClosed)
			case "locked":
				exit.Flags.Set(types.ExitLocked)
			case "pickproof":
				exit.Flags.Set(types.ExitPickproof)
			}
		}

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
		room.MobResets = append(room.MobResets, types.MobReset{
			Vnum:  mr.Vnum,
			Max:   mr.Max,
			Count: count,
		})
	}

	// Parse object resets
	for _, or := range rd.ObjResets {
		count := or.Count
		if count == 0 {
			count = 1
		}
		room.ObjResets = append(room.ObjResets, types.ObjReset{
			Vnum:  or.Vnum,
			Max:   or.Max,
			Count: count,
		})
	}

	return room
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
	ch.Gold = tmpl.Gold
	ch.HitRoll = tmpl.Hitroll

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

	// Parse act flags
	for _, flag := range tmpl.ActFlags {
		switch strings.ToLower(flag) {
		case "sentinel":
			ch.Act.Set(types.ActSentinel)
		case "scavenger":
			ch.Act.Set(types.ActScavenger)
		case "aggressive":
			ch.Act.Set(types.ActAggressive)
		case "stay_area":
			ch.Act.Set(types.ActStayArea)
		case "wimpy":
			ch.Act.Set(types.ActWimpy)
		case "warrior":
			ch.Act.Set(types.ActWarrior)
		case "mage":
			ch.Act.Set(types.ActMage)
		case "cleric":
			ch.Act.Set(types.ActCleric)
		case "thief":
			ch.Act.Set(types.ActThief)
		case "undead":
			ch.Act.Set(types.ActUndead)
		case "train":
			ch.Act.Set(types.ActTrain)
		case "practice":
			ch.Act.Set(types.ActPractice)
		case "pet":
			ch.Act.Set(types.ActPet)
		case "healer", "is_healer":
			ch.Act.Set(types.ActIsHealer)
		case "gain":
			ch.Act.Set(types.ActGain)
		case "update_always":
			ch.Act.Set(types.ActUpdateAlways)
		case "no_align":
			ch.Act.Set(types.ActNoAlign)
		case "no_purge":
			ch.Act.Set(types.ActNoPurge)
		}
	}

	// Parse affected_by flags
	for _, flag := range tmpl.AffectedBy {
		switch strings.ToLower(flag) {
		case "sanctuary":
			ch.AffectedBy.Set(types.AffSanctuary)
		case "detect_invis":
			ch.AffectedBy.Set(types.AffDetectInvis)
		case "detect_hidden":
			ch.AffectedBy.Set(types.AffDetectHidden)
		case "detect_evil":
			ch.AffectedBy.Set(types.AffDetectEvil)
		case "detect_good":
			ch.AffectedBy.Set(types.AffDetectGood)
		case "invisible":
			ch.AffectedBy.Set(types.AffInvisible)
		case "hide":
			ch.AffectedBy.Set(types.AffHide)
		case "flying":
			ch.AffectedBy.Set(types.AffFlying)
		case "haste":
			ch.AffectedBy.Set(types.AffHaste)
		}
	}

	// Set default position
	ch.Position = types.PosStanding
	ch.DefaultPos = types.PosStanding

	return ch
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
		rd.MobResets = append(rd.MobResets, MobResetData{
			Vnum:  mr.Vnum,
			Max:   mr.Max,
			Count: mr.Count,
		})
	}

	// Convert object resets
	for _, or := range room.ObjResets {
		rd.ObjResets = append(rd.ObjResets, ObjResetData{
			Vnum:  or.Vnum,
			Max:   or.Max,
			Count: or.Count,
		})
	}

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

func formatRoomFlags(flags types.RoomFlags) []string {
	var result []string
	if flags.Has(types.RoomDark) {
		result = append(result, "dark")
	}
	if flags.Has(types.RoomNoMob) {
		result = append(result, "no_mob")
	}
	if flags.Has(types.RoomIndoors) {
		result = append(result, "indoors")
	}
	if flags.Has(types.RoomPrivate) {
		result = append(result, "private")
	}
	if flags.Has(types.RoomSafe) {
		result = append(result, "safe")
	}
	if flags.Has(types.RoomSolitary) {
		result = append(result, "solitary")
	}
	if flags.Has(types.RoomNoRecall) {
		result = append(result, "no_recall")
	}
	if flags.Has(types.RoomLaw) {
		result = append(result, "law")
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
