package types

// Exit represents a room exit/door
// Based on EXIT_DATA from merc.h:1876-1890
type Exit struct {
	Direction   Direction // Which direction this exit leads
	ToVnum      int       // Target room vnum
	ToRoom      *Room     // Resolved room pointer (set during loading)
	Flags       ExitFlags // Exit flags (door, closed, locked, etc.)
	Key         int       // Key vnum required to unlock
	Keywords    string    // Keywords for the door (e.g., "door gate")
	Description string    // What you see when you look at the exit
}

// NewExit creates a new exit
func NewExit(dir Direction, toVnum int) *Exit {
	return &Exit{
		Direction: dir,
		ToVnum:    toVnum,
	}
}

// IsDoor returns true if this exit has a door
func (e *Exit) IsDoor() bool {
	return e.Flags.Has(ExitIsDoor)
}

// IsClosed returns true if the door is closed
func (e *Exit) IsClosed() bool {
	return e.Flags.Has(ExitClosed)
}

// IsLocked returns true if the door is locked
func (e *Exit) IsLocked() bool {
	return e.Flags.Has(ExitLocked)
}

// Open opens the door (removes closed flag)
func (e *Exit) Open() {
	e.Flags.Remove(ExitClosed)
}

// Close closes the door (sets closed flag)
func (e *Exit) Close() {
	e.Flags.Set(ExitClosed)
}

// Lock locks the door
func (e *Exit) Lock() {
	e.Flags.Set(ExitLocked)
}

// Unlock unlocks the door
func (e *Exit) Unlock() {
	e.Flags.Remove(ExitLocked)
}

// ExtraDescription represents additional descriptions that can be looked at
// Based on EXTRA_DESCR_DATA from merc.h:1793-1799
type ExtraDescription struct {
	Keywords    []string // Keywords to match (e.g., ["statue", "marble"])
	Description string   // What you see when you look at it
}

// MobReset defines a mob that spawns in a room
type MobReset struct {
	Vnum   int          // Mob template vnum
	Max    int          // Max number in world (0 = unlimited)
	Count  int          // Number to spawn (default 1)
	Equips []EquipReset // Equipment to give to this mob
}

// EquipReset defines an object to equip on a mob
type EquipReset struct {
	Vnum    int          // Object template vnum
	WearLoc WearLocation // Where to equip the item
	Limit   int          // Max number in world (0 = unlimited)
	InvOnly bool         // Put in inventory instead of equipping
}

// ObjReset defines an object that spawns in a room
type ObjReset struct {
	Vnum  int // Object template vnum
	Max   int // Max number in world (0 = unlimited)
	Count int // Number to spawn (default 1)
}

// Room represents a room in the world
// Based on ROOM_INDEX_DATA from merc.h:1949-1970
type Room struct {
	Vnum              int                 // Virtual number (unique ID)
	Name              string              // Room name shown in look
	Description       string              // Full room description
	Flags             RoomFlags           // Room flags
	Sector            Sector              // Terrain type
	Exits             [6]*Exit            // Exits (indexed by Direction)
	ExtraDescriptions []*ExtraDescription // Extra look targets
	HealRate          int                 // HP regen rate modifier (100 = normal)
	ManaRate          int                 // Mana regen rate modifier (100 = normal)
	Clan              int                 // Clan that owns this room (0 = none)
	Owner             string              // Player owner (for houses)
	Light             int                 // Light level

	// Reset data (loaded from TOML)
	MobResets []MobReset // Mobs that spawn here
	ObjResets []ObjReset // Objects that spawn here

	// Runtime data (not persisted)
	Area    *Area        // Area this room belongs to
	People  []*Character // Characters in the room
	Objects []*Object    // Objects on the floor
}

// NewRoom creates a new room
func NewRoom(vnum int, name, description string) *Room {
	return &Room{
		Vnum:        vnum,
		Name:        name,
		Description: description,
		Sector:      SectInside,
		HealRate:    100,
		ManaRate:    100,
		People:      make([]*Character, 0),
		Objects:     make([]*Object, 0),
	}
}

// GetExit returns the exit in the given direction, or nil
func (r *Room) GetExit(dir Direction) *Exit {
	if dir < 0 || dir >= DirMax {
		return nil
	}
	return r.Exits[dir]
}

// SetExit sets an exit in the given direction
func (r *Room) SetExit(dir Direction, exit *Exit) {
	if dir >= 0 && dir < DirMax {
		r.Exits[dir] = exit
	}
}

// ExitDirections returns a slice of directions that have exits
func (r *Room) ExitDirections() []Direction {
	var dirs []Direction
	for dir, exit := range r.Exits {
		if exit != nil {
			dirs = append(dirs, Direction(dir))
		}
	}
	return dirs
}

// IsDark returns true if the room is dark
func (r *Room) IsDark() bool {
	if r.Light > 0 {
		return false
	}
	return r.Flags.Has(RoomDark)
}

// IsSafe returns true if combat is not allowed
func (r *Room) IsSafe() bool {
	return r.Flags.Has(RoomSafe)
}

// IsPrivate returns true if the room has limited occupancy
func (r *Room) IsPrivate() bool {
	return r.Flags.Has(RoomPrivate)
}

// PeopleCount returns the number of characters in the room
func (r *Room) PeopleCount() int {
	return len(r.People)
}

// AddPerson adds a character to the room
func (r *Room) AddPerson(ch *Character) {
	r.People = append(r.People, ch)
}

// RemovePerson removes a character from the room
func (r *Room) RemovePerson(ch *Character) {
	for i, p := range r.People {
		if p == ch {
			r.People = append(r.People[:i], r.People[i+1:]...)
			return
		}
	}
}

// AddObject adds an object to the room floor
func (r *Room) AddObject(obj *Object) {
	r.Objects = append(r.Objects, obj)
}

// RemoveObject removes an object from the room floor
func (r *Room) RemoveObject(obj *Object) {
	for i, o := range r.Objects {
		if o == obj {
			r.Objects = append(r.Objects[:i], r.Objects[i+1:]...)
			return
		}
	}
}

// Area represents a game area/zone
// Based on AREA_DATA from merc.h:1923-1942
type Area struct {
	Name       string // Area name
	Filename   string // Source filename
	Credits    string // Builder credits
	MinVnum    int    // Minimum vnum in area
	MaxVnum    int    // Maximum vnum in area
	LowRange   int    // Suggested level range (low)
	HighRange  int    // Suggested level range (high)
	Age        int    // Ticks since last reset
	NumPlayers int    // Players currently in area
	Empty      bool   // True if area is empty of players

	// Runtime data
	Rooms map[int]*Room // Rooms in this area (keyed by vnum)
}

// NewArea creates a new area
func NewArea(name string, minVnum, maxVnum int) *Area {
	return &Area{
		Name:    name,
		MinVnum: minVnum,
		MaxVnum: maxVnum,
		Rooms:   make(map[int]*Room),
	}
}
