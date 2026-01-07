package skills

import (
	"rotmud/pkg/types"
)

// SkillType distinguishes between skills and spells
type SkillType int

const (
	TypeSkill SkillType = iota
	TypeSpell
)

// Skill represents a learnable skill or spell
type Skill struct {
	Name        string         // Skill name
	Type        SkillType      // Skill or spell
	Levels      map[int]int    // Class index -> level required (0 = can't learn)
	Rating      map[int]int    // Class index -> difficulty rating (for improvement)
	MinPosition types.Position // Minimum position to use
	MinMana     int            // Mana cost (for spells)
	Beats       int            // Lag after use (in pulses)
	NounDamage  string         // Damage message noun
	WearOff     string         // Message when affect wears off
}

// NewSkill creates a new skill
func NewSkill(name string) *Skill {
	return &Skill{
		Name:        name,
		Type:        TypeSkill,
		Levels:      make(map[int]int),
		Rating:      make(map[int]int),
		MinPosition: types.PosStanding,
		Beats:       12, // 3 seconds default lag
	}
}

// SetClassLevel sets the level requirement for a class
func (s *Skill) SetClassLevel(classIndex, level, rating int) *Skill {
	s.Levels[classIndex] = level
	s.Rating[classIndex] = rating
	return s
}

// GetLevel returns the level requirement for a class
func (s *Skill) GetLevel(classIndex int) int {
	if level, ok := s.Levels[classIndex]; ok {
		return level
	}
	return 0 // Can't learn
}

// GetRating returns the difficulty rating for a class
func (s *Skill) GetRating(classIndex int) int {
	if rating, ok := s.Rating[classIndex]; ok {
		return rating
	}
	return 0
}

// CanLearn checks if a class can learn this skill at any level
func (s *Skill) CanLearn(classIndex int) bool {
	level := s.GetLevel(classIndex)
	return level > 0 && level <= types.MaxLevel
}

// SkillRegistry holds all registered skills
type SkillRegistry struct {
	byName  map[string]*Skill
	byIndex map[int]*Skill
	skills  []*Skill
}

// NewSkillRegistry creates a new skill registry
func NewSkillRegistry() *SkillRegistry {
	return &SkillRegistry{
		byName:  make(map[string]*Skill),
		byIndex: make(map[int]*Skill),
		skills:  make([]*Skill, 0),
	}
}

// Register adds a skill to the registry
func (r *SkillRegistry) Register(skill *Skill) int {
	index := len(r.skills)
	r.skills = append(r.skills, skill)
	r.byName[skill.Name] = skill
	r.byIndex[index] = skill
	return index
}

// FindByName finds a skill by name (exact match only)
func (r *SkillRegistry) FindByName(name string) *Skill {
	if skill, ok := r.byName[name]; ok {
		return skill
	}
	return nil
}

// FindByPrefix finds a skill by name prefix (for command parsing)
func (r *SkillRegistry) FindByPrefix(name string) *Skill {
	// Exact match first
	if skill, ok := r.byName[name]; ok {
		return skill
	}

	// Prefix match - find the best (shortest) matching skill name
	var bestMatch *Skill
	for skillName, skill := range r.byName {
		if len(name) <= len(skillName) && skillName[:len(name)] == name {
			// Prefer shorter skill names (more specific match)
			if bestMatch == nil || len(skillName) < len(bestMatch.Name) {
				bestMatch = skill
			}
		}
	}
	return bestMatch
}

// FindByIndex finds a skill by index
func (r *SkillRegistry) FindByIndex(index int) *Skill {
	return r.byIndex[index]
}

// GetIndex returns the index for a skill name
func (r *SkillRegistry) GetIndex(name string) int {
	for i, skill := range r.skills {
		if skill.Name == name {
			return i
		}
	}
	return -1
}

// All returns all registered skills
func (r *SkillRegistry) All() []*Skill {
	return r.skills
}

// Count returns the number of registered skills
func (r *SkillRegistry) Count() int {
	return len(r.skills)
}
