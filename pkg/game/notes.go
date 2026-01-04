package game

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"rotmud/pkg/types"
)

// NoteType represents different note boards
type NoteType int

const (
	NoteNote    NoteType = iota // General notes
	NoteIdea                    // Ideas/suggestions
	NoteNews                    // News announcements
	NoteChanges                 // Code changes
	NotePenalty                 // Penalty notices (imm only)
)

// String returns the note type name
func (t NoteType) String() string {
	names := []string{"note", "idea", "news", "changes", "penalty"}
	if t >= 0 && int(t) < len(names) {
		return names[t]
	}
	return "note"
}

// Note represents a single note/message
type Note struct {
	ID        int       `json:"id"`
	Type      NoteType  `json:"type"`
	Sender    string    `json:"sender"`
	Date      time.Time `json:"date"`
	To        string    `json:"to"` // "all", "imm", player name, etc.
	Subject   string    `json:"subject"`
	Text      string    `json:"text"`
	DateStamp string    `json:"date_stamp"` // Formatted date for display
}

// NoteSystem manages all note boards
type NoteSystem struct {
	notes   map[NoteType][]*Note
	nextID  int
	dataDir string
	mu      sync.RWMutex
}

// NewNoteSystem creates a new note system
func NewNoteSystem(dataDir string) *NoteSystem {
	ns := &NoteSystem{
		notes:   make(map[NoteType][]*Note),
		nextID:  1,
		dataDir: dataDir,
	}

	// Initialize empty lists for each type
	for t := NoteNote; t <= NotePenalty; t++ {
		ns.notes[t] = make([]*Note, 0)
	}

	return ns
}

// Load loads notes from disk
func (ns *NoteSystem) Load() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for t := NoteNote; t <= NotePenalty; t++ {
		filename := ns.getFilename(t)
		data, err := os.ReadFile(filename)
		if err != nil {
			if os.IsNotExist(err) {
				continue // No notes file yet
			}
			return err
		}

		var notes []*Note
		if err := json.Unmarshal(data, &notes); err != nil {
			return err
		}

		ns.notes[t] = notes

		// Update nextID
		for _, n := range notes {
			if n.ID >= ns.nextID {
				ns.nextID = n.ID + 1
			}
		}
	}

	return nil
}

// Save saves notes to disk
func (ns *NoteSystem) Save() error {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// Create data directory if needed
	if ns.dataDir != "" {
		if err := os.MkdirAll(ns.dataDir, 0755); err != nil {
			return err
		}
	}

	for t := NoteNote; t <= NotePenalty; t++ {
		filename := ns.getFilename(t)
		data, err := json.MarshalIndent(ns.notes[t], "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(filename, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (ns *NoteSystem) getFilename(t NoteType) string {
	if ns.dataDir == "" {
		return t.String() + ".json"
	}
	return ns.dataDir + "/" + t.String() + ".json"
}

// Add adds a new note
func (ns *NoteSystem) Add(note *Note) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	note.ID = ns.nextID
	ns.nextID++
	note.Date = time.Now()
	note.DateStamp = note.Date.Format("Mon Jan 2 15:04:05 2006")

	ns.notes[note.Type] = append(ns.notes[note.Type], note)
}

// Remove removes a note by ID
func (ns *NoteSystem) Remove(t NoteType, id int) bool {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	notes := ns.notes[t]
	for i, n := range notes {
		if n.ID == id {
			ns.notes[t] = append(notes[:i], notes[i+1:]...)
			return true
		}
	}
	return false
}

// GetForPlayer returns notes visible to a player
func (ns *NoteSystem) GetForPlayer(t NoteType, ch *types.Character) []*Note {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	result := make([]*Note, 0)
	for _, n := range ns.notes[t] {
		if ns.canRead(ch, n) {
			result = append(result, n)
		}
	}

	// Sort by date (newest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.After(result[j].Date)
	})

	return result
}

// GetByID returns a note by ID if visible to player
func (ns *NoteSystem) GetByID(t NoteType, id int, ch *types.Character) *Note {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	for _, n := range ns.notes[t] {
		if n.ID == id && ns.canRead(ch, n) {
			return n
		}
	}
	return nil
}

// GetUnreadCount returns number of unread notes for a player
func (ns *NoteSystem) GetUnreadCount(t NoteType, ch *types.Character, lastRead time.Time) int {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	count := 0
	for _, n := range ns.notes[t] {
		if ns.canRead(ch, n) && n.Date.After(lastRead) {
			count++
		}
	}
	return count
}

// canRead checks if a character can read a note
func (ns *NoteSystem) canRead(ch *types.Character, n *Note) bool {
	to := strings.ToLower(n.To)

	// All can read "all"
	if to == "all" {
		return true
	}

	// Immortals can read "imm" or "immortal"
	if (to == "imm" || to == "immortal") && ch.IsImmortal() {
		return true
	}

	// Check if addressed to this player
	if strings.EqualFold(n.To, ch.Name) {
		return true
	}

	// Sender can always read their own notes
	if strings.EqualFold(n.Sender, ch.Name) {
		return true
	}

	// Immortals can read everything
	if ch.IsImmortal() {
		return true
	}

	return false
}

// NoteEditor manages note composition state per player
type NoteEditor struct {
	To      string
	Subject string
	Lines   []string
}

// NoteEditors tracks active note editors per character
var noteEditors = make(map[*types.Character]*NoteEditor)
var noteEditorsMu sync.RWMutex

// GetNoteEditor gets or creates a note editor for a character
func GetNoteEditor(ch *types.Character) *NoteEditor {
	noteEditorsMu.Lock()
	defer noteEditorsMu.Unlock()

	if ed, ok := noteEditors[ch]; ok {
		return ed
	}

	ed := &NoteEditor{
		Lines: make([]string, 0),
	}
	noteEditors[ch] = ed
	return ed
}

// ClearNoteEditor removes the note editor for a character
func ClearNoteEditor(ch *types.Character) {
	noteEditorsMu.Lock()
	defer noteEditorsMu.Unlock()
	delete(noteEditors, ch)
}

// HasNoteEditor checks if a character has an active note editor
func HasNoteEditor(ch *types.Character) bool {
	noteEditorsMu.RLock()
	defer noteEditorsMu.RUnlock()
	_, ok := noteEditors[ch]
	return ok
}

// AddLine adds a line to the note being composed
func (ed *NoteEditor) AddLine(line string) {
	ed.Lines = append(ed.Lines, line)
}

// GetText returns the full note text
func (ed *NoteEditor) GetText() string {
	return strings.Join(ed.Lines, "\r\n")
}

// Clear resets the editor
func (ed *NoteEditor) Clear() {
	ed.To = ""
	ed.Subject = ""
	ed.Lines = ed.Lines[:0]
}
