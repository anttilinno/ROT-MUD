package help

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Entry represents a single help topic
type Entry struct {
	Keywords    []string `toml:"keywords"`
	Level       int      `toml:"level"`       // Minimum level to see
	SeeAlso     []string `toml:"see_also"`    // Related topics
	Syntax      string   `toml:"syntax"`      // Command syntax
	Description string   `toml:"description"` // Full help text
}

// HelpFile represents a TOML file containing help entries
type HelpFile struct {
	Entries []Entry `toml:"help"`
}

// System manages help topics
type System struct {
	entries map[string]*Entry // keyword -> entry
}

// NewSystem creates a new help system
func NewSystem() *System {
	return &System{
		entries: make(map[string]*Entry),
	}
}

// LoadFile loads help entries from a single TOML file
func (s *System) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var hf HelpFile
	if err := toml.Unmarshal(data, &hf); err != nil {
		return err
	}

	for i := range hf.Entries {
		entry := &hf.Entries[i]
		for _, keyword := range entry.Keywords {
			s.entries[strings.ToLower(keyword)] = entry
		}
	}

	return nil
}

// LoadDir loads all help files from a directory
func (s *System) LoadDir(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".toml") {
			return nil
		}
		return s.LoadFile(path)
	})
}

// Register adds a help entry programmatically
func (s *System) Register(entry *Entry) {
	for _, keyword := range entry.Keywords {
		s.entries[strings.ToLower(keyword)] = entry
	}
}

// Find looks up a help topic by keyword
func (s *System) Find(keyword string) *Entry {
	keyword = strings.ToLower(keyword)

	// Exact match first
	if entry, ok := s.entries[keyword]; ok {
		return entry
	}

	// Prefix match
	for key, entry := range s.entries {
		if strings.HasPrefix(key, keyword) {
			return entry
		}
	}

	return nil
}

// FindAll returns all entries matching a keyword (for disambiguation)
func (s *System) FindAll(keyword string) []*Entry {
	keyword = strings.ToLower(keyword)
	seen := make(map[*Entry]bool)
	var results []*Entry

	for key, entry := range s.entries {
		if strings.HasPrefix(key, keyword) && !seen[entry] {
			seen[entry] = true
			results = append(results, entry)
		}
	}

	return results
}

// Count returns the number of unique help entries
func (s *System) Count() int {
	seen := make(map[*Entry]bool)
	for _, entry := range s.entries {
		seen[entry] = true
	}
	return len(seen)
}

// All returns all keywords
func (s *System) AllKeywords() []string {
	var keywords []string
	for keyword := range s.entries {
		keywords = append(keywords, keyword)
	}
	return keywords
}

// Format returns a formatted help text for display
func (e *Entry) Format() string {
	var sb strings.Builder

	// Title
	sb.WriteString(strings.ToUpper(e.Keywords[0]))
	sb.WriteString("\r\n")
	sb.WriteString(strings.Repeat("-", len(e.Keywords[0])))
	sb.WriteString("\r\n\r\n")

	// Syntax
	if e.Syntax != "" {
		sb.WriteString("Syntax: ")
		sb.WriteString(e.Syntax)
		sb.WriteString("\r\n\r\n")
	}

	// Description
	sb.WriteString(e.Description)
	if !strings.HasSuffix(e.Description, "\n") {
		sb.WriteString("\r\n")
	}

	// See also
	if len(e.SeeAlso) > 0 {
		sb.WriteString("\r\nSee also: ")
		sb.WriteString(strings.Join(e.SeeAlso, ", "))
		sb.WriteString("\r\n")
	}

	return sb.String()
}
