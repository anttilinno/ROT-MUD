package game

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"rotmud/pkg/types"
)

// cmdNote handles the note command (general notes board)
func (d *CommandDispatcher) cmdNote(ch *types.Character, args string) {
	d.parseNote(ch, args, NoteNote)
}

// cmdIdea handles the idea command (suggestions board)
func (d *CommandDispatcher) cmdIdea(ch *types.Character, args string) {
	d.parseNote(ch, args, NoteIdea)
}

// cmdNews handles the news command (news board)
func (d *CommandDispatcher) cmdNews(ch *types.Character, args string) {
	d.parseNote(ch, args, NoteNews)
}

// cmdChanges handles the changes command (code changes board)
func (d *CommandDispatcher) cmdChanges(ch *types.Character, args string) {
	d.parseNote(ch, args, NoteChanges)
}

// parseNote is the common handler for all note types
func (d *CommandDispatcher) parseNote(ch *types.Character, args string, noteType NoteType) {
	if ch.IsNPC() {
		d.send(ch, "NPCs cannot use notes.\r\n")
		return
	}

	if d.Notes == nil {
		d.send(ch, "Note system is not available.\r\n")
		return
	}

	// Penalty notes are immortal only
	if noteType == NotePenalty && !ch.IsImmortal() {
		d.send(ch, "You don't have access to that board.\r\n")
		return
	}

	// Parse command
	parts := strings.SplitN(strings.TrimSpace(args), " ", 2)
	cmd := ""
	arg := ""
	if len(parts) > 0 {
		cmd = strings.ToLower(parts[0])
	}
	if len(parts) > 1 {
		arg = parts[1]
	}

	boardName := noteType.String() + "s"
	if noteType == NoteNews {
		boardName = "news"
	}

	switch cmd {
	case "", "read":
		d.noteRead(ch, arg, noteType, boardName)
	case "list":
		d.noteList(ch, noteType, boardName)
	case "write", "to":
		d.noteTo(ch, arg, noteType)
	case "subject":
		d.noteSubject(ch, arg)
	case "+":
		d.noteAddLine(ch, arg)
	case "-":
		d.noteRemoveLine(ch)
	case "clear":
		d.noteClear(ch)
	case "show":
		d.noteShow(ch)
	case "post", "send":
		d.notePost(ch, noteType)
	case "remove":
		d.noteRemove(ch, arg, noteType, boardName)
	case "catchup":
		d.noteCatchup(ch, noteType, boardName)
	default:
		d.noteHelp(ch, noteType)
	}
}

// noteHelp displays note command help
func (d *CommandDispatcher) noteHelp(ch *types.Character, noteType NoteType) {
	name := noteType.String()
	d.send(ch, fmt.Sprintf("Syntax: %s                 - read next unread %s\r\n", name, name))
	d.send(ch, fmt.Sprintf("        %s list            - list all %ss\r\n", name, name))
	d.send(ch, fmt.Sprintf("        %s read <number>   - read %s #\r\n", name, name))
	d.send(ch, fmt.Sprintf("        %s to <recipient>  - start writing to recipient\r\n", name))
	d.send(ch, fmt.Sprintf("        %s subject <text>  - set subject\r\n", name))
	d.send(ch, fmt.Sprintf("        %s + <line>        - add a line\r\n", name))
	d.send(ch, fmt.Sprintf("        %s -               - remove last line\r\n", name))
	d.send(ch, fmt.Sprintf("        %s show            - show note in progress\r\n", name))
	d.send(ch, fmt.Sprintf("        %s clear           - clear note in progress\r\n", name))
	d.send(ch, fmt.Sprintf("        %s post            - post the note\r\n", name))
	d.send(ch, fmt.Sprintf("        %s remove <number> - remove your own note\r\n", name))
	d.send(ch, fmt.Sprintf("        %s catchup         - mark all as read\r\n", name))
}

// noteRead reads a specific note or the next unread one
func (d *CommandDispatcher) noteRead(ch *types.Character, arg string, noteType NoteType, boardName string) {
	notes := d.Notes.GetForPlayer(noteType, ch)

	if len(notes) == 0 {
		d.send(ch, fmt.Sprintf("There are no %s for you.\r\n", boardName))
		return
	}

	// If no argument or "next", read next unread
	if arg == "" || strings.ToLower(arg) == "next" {
		lastRead := d.getLastRead(ch, noteType)
		for i, note := range notes {
			if note.Date.After(lastRead) {
				d.displayNote(ch, note, i)
				d.updateLastRead(ch, noteType, note.Date)
				return
			}
		}
		d.send(ch, fmt.Sprintf("You have no unread %s.\r\n", boardName))
		return
	}

	// Read by number
	num, err := strconv.Atoi(arg)
	if err != nil {
		d.send(ch, "Read which number?\r\n")
		return
	}

	if num < 0 || num >= len(notes) {
		d.send(ch, fmt.Sprintf("There aren't that many %s.\r\n", boardName))
		return
	}

	note := notes[num]
	d.displayNote(ch, note, num)
	d.updateLastRead(ch, noteType, note.Date)
}

// displayNote shows a note to the character
func (d *CommandDispatcher) displayNote(ch *types.Character, note *Note, num int) {
	d.send(ch, fmt.Sprintf("[%3d] %s: %s\r\n", num, note.Sender, note.Subject))
	d.send(ch, note.DateStamp+"\r\n")
	d.send(ch, fmt.Sprintf("To: %s\r\n", note.To))
	d.send(ch, strings.Repeat("-", 40)+"\r\n")
	d.send(ch, note.Text+"\r\n")
}

// noteList lists all notes visible to the character
func (d *CommandDispatcher) noteList(ch *types.Character, noteType NoteType, boardName string) {
	notes := d.Notes.GetForPlayer(noteType, ch)

	if len(notes) == 0 {
		d.send(ch, fmt.Sprintf("There are no %s for you.\r\n", boardName))
		return
	}

	lastRead := d.getLastRead(ch, noteType)

	for i, note := range notes {
		unreadMarker := " "
		if note.Date.After(lastRead) {
			unreadMarker = "N"
		}
		d.send(ch, fmt.Sprintf("[%3d%s] %s: %s\r\n", i, unreadMarker, note.Sender, note.Subject))
	}
}

// noteTo starts a new note or sets the recipient
func (d *CommandDispatcher) noteTo(ch *types.Character, arg string, noteType NoteType) {
	if arg == "" {
		d.send(ch, "Write to whom?\r\n")
		return
	}

	ed := GetNoteEditor(ch)
	ed.To = arg
	d.send(ch, fmt.Sprintf("Note to: %s\r\n", arg))
	d.send(ch, "Use 'note subject <text>' to set the subject.\r\n")
}

// noteSubject sets the subject of the note being composed
func (d *CommandDispatcher) noteSubject(ch *types.Character, arg string) {
	if arg == "" {
		d.send(ch, "What subject?\r\n")
		return
	}

	ed := GetNoteEditor(ch)
	if ed.To == "" {
		d.send(ch, "You need to specify a recipient first with 'note to <name>'.\r\n")
		return
	}

	ed.Subject = arg
	d.send(ch, fmt.Sprintf("Subject: %s\r\n", arg))
	d.send(ch, "Use 'note + <line>' to add lines to your note.\r\n")
}

// noteAddLine adds a line to the note being composed
func (d *CommandDispatcher) noteAddLine(ch *types.Character, arg string) {
	ed := GetNoteEditor(ch)
	if ed.To == "" {
		d.send(ch, "You need to start a note first with 'note to <name>'.\r\n")
		return
	}

	ed.AddLine(arg)
	d.send(ch, "Line added.\r\n")
}

// noteRemoveLine removes the last line from the note being composed
func (d *CommandDispatcher) noteRemoveLine(ch *types.Character) {
	ed := GetNoteEditor(ch)
	if len(ed.Lines) == 0 {
		d.send(ch, "There are no lines to remove.\r\n")
		return
	}

	ed.Lines = ed.Lines[:len(ed.Lines)-1]
	d.send(ch, "Line removed.\r\n")
}

// noteClear clears the note being composed
func (d *CommandDispatcher) noteClear(ch *types.Character) {
	ClearNoteEditor(ch)
	d.send(ch, "Note cleared.\r\n")
}

// noteShow shows the note being composed
func (d *CommandDispatcher) noteShow(ch *types.Character) {
	ed := GetNoteEditor(ch)

	if ed.To == "" && ed.Subject == "" && len(ed.Lines) == 0 {
		d.send(ch, "You have no note in progress.\r\n")
		return
	}

	d.send(ch, fmt.Sprintf("To:      %s\r\n", ed.To))
	d.send(ch, fmt.Sprintf("Subject: %s\r\n", ed.Subject))
	d.send(ch, strings.Repeat("-", 40)+"\r\n")
	for i, line := range ed.Lines {
		d.send(ch, fmt.Sprintf("%2d> %s\r\n", i+1, line))
	}
}

// notePost posts the composed note
func (d *CommandDispatcher) notePost(ch *types.Character, noteType NoteType) {
	ed := GetNoteEditor(ch)

	if ed.To == "" {
		d.send(ch, "You need to specify a recipient with 'note to <name>'.\r\n")
		return
	}

	if ed.Subject == "" {
		d.send(ch, "You need to specify a subject with 'note subject <text>'.\r\n")
		return
	}

	if len(ed.Lines) == 0 {
		d.send(ch, "You need to add some text with 'note + <line>'.\r\n")
		return
	}

	note := &Note{
		Type:    noteType,
		Sender:  ch.Name,
		To:      ed.To,
		Subject: ed.Subject,
		Text:    ed.GetText(),
	}

	d.Notes.Add(note)

	// Save notes to disk
	if err := d.Notes.Save(); err != nil {
		d.send(ch, "Note posted, but failed to save to disk.\r\n")
	} else {
		d.send(ch, "Note posted.\r\n")
	}

	ClearNoteEditor(ch)
}

// noteRemove removes a note by number
func (d *CommandDispatcher) noteRemove(ch *types.Character, arg string, noteType NoteType, boardName string) {
	if arg == "" {
		d.send(ch, "Remove which note?\r\n")
		return
	}

	num, err := strconv.Atoi(arg)
	if err != nil {
		d.send(ch, "Remove which number?\r\n")
		return
	}

	notes := d.Notes.GetForPlayer(noteType, ch)

	if num < 0 || num >= len(notes) {
		d.send(ch, fmt.Sprintf("There aren't that many %s.\r\n", boardName))
		return
	}

	note := notes[num]

	// Only sender or immortals can remove notes
	if !strings.EqualFold(note.Sender, ch.Name) && !ch.IsImmortal() {
		d.send(ch, "You can only remove your own notes.\r\n")
		return
	}

	if d.Notes.Remove(noteType, note.ID) {
		if err := d.Notes.Save(); err != nil {
			d.send(ch, "Note removed, but failed to save to disk.\r\n")
		} else {
			d.send(ch, "Note removed.\r\n")
		}
	} else {
		d.send(ch, "Failed to remove note.\r\n")
	}
}

// noteCatchup marks all notes of a type as read
func (d *CommandDispatcher) noteCatchup(ch *types.Character, noteType NoteType, boardName string) {
	d.updateLastRead(ch, noteType, time.Now())
	d.send(ch, fmt.Sprintf("All %s marked as read.\r\n", boardName))
}

// getLastRead returns when the character last read notes of this type
func (d *CommandDispatcher) getLastRead(ch *types.Character, noteType NoteType) time.Time {
	if ch.PCData == nil {
		return time.Time{}
	}

	switch noteType {
	case NoteNote:
		return ch.PCData.LastNote
	case NoteIdea:
		return ch.PCData.LastIdea
	case NoteNews:
		return ch.PCData.LastNews
	case NoteChanges:
		return ch.PCData.LastChanges
	case NotePenalty:
		return ch.PCData.LastPenalty
	}
	return time.Time{}
}

// updateLastRead updates when the character last read notes of this type
func (d *CommandDispatcher) updateLastRead(ch *types.Character, noteType NoteType, t time.Time) {
	if ch.PCData == nil {
		return
	}

	switch noteType {
	case NoteNote:
		ch.PCData.LastNote = t
	case NoteIdea:
		ch.PCData.LastIdea = t
	case NoteNews:
		ch.PCData.LastNews = t
	case NoteChanges:
		ch.PCData.LastChanges = t
	case NotePenalty:
		ch.PCData.LastPenalty = t
	}
}
