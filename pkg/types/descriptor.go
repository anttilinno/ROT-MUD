package types

// EditorType represents OLC editor modes
type EditorType int

const (
	EditorNone EditorType = iota
	EditorArea
	EditorRoom
	EditorObject
	EditorMobile
	EditorMPCode
	EditorHelp
)

// Descriptor represents a network connection
// Based on DESCRIPTOR_DATA from merc.h:340-363
type Descriptor struct {
	// Connection info
	Host  string    // Hostname/IP of connection
	State ConnState // Current connection state

	// Character association
	Character *Character // Character being played
	Original  *Character // Original character (for switched immortals)

	// Snooping
	SnoopedBy *Descriptor // Who is snooping this connection

	// Input handling
	LastCommand string // Last command entered
	RepeatCount int    // How many times command repeated

	// OLC (Online Creation)
	Editor     EditorType  // Current OLC editor mode
	EditData   interface{} // Data being edited (Room, Object, etc.)
	EditString *string     // String being edited

	// Output paging
	ShowstrHead  string // Full text being paged
	ShowstrPoint int    // Current position in paged text
}

// NewDescriptor creates a new descriptor
func NewDescriptor(host string) *Descriptor {
	return &Descriptor{
		Host:  host,
		State: ConGetName,
	}
}

// IsPlaying returns true if the connection is in playing state
func (d *Descriptor) IsPlaying() bool {
	return d.State == ConPlaying
}

// IsSnooped returns true if someone is snooping this connection
func (d *Descriptor) IsSnooped() bool {
	return d.SnoopedBy != nil
}

// InEditor returns true if in an OLC editor
func (d *Descriptor) InEditor() bool {
	return d.Editor != EditorNone
}

// HasCharacter returns true if a character is associated
func (d *Descriptor) HasCharacter() bool {
	return d.Character != nil
}
