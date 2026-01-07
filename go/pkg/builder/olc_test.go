package builder

import (
	"strings"
	"testing"

	"rotmud/pkg/types"
)

func TestOLCSystem(t *testing.T) {
	olc := NewOLCSystem()
	if olc == nil {
		t.Error("NewOLCSystem should return a non-nil system")
	}
}

func TestRoomEditor(t *testing.T) {
	olc := NewOLCSystem()

	var output strings.Builder
	olc.Output = func(ch *types.Character, msg string) {
		output.WriteString(msg)
	}

	room := types.NewRoom(3001, "Test Room", "This is a test room.")
	olc.GetRoom = func(vnum int) *types.Room {
		if vnum == 3001 {
			return room
		}
		return nil
	}

	state := &EditorState{
		Mode:     EditorRoom,
		EditVnum: 3001,
		Data:     room,
	}

	ch := types.NewCharacter("TestBuilder")
	ch.PCData = &types.PCData{Security: 9}

	// Test show command
	output.Reset()
	olc.processRoomCommand(ch, state, "show", "")
	if !strings.Contains(output.String(), "Test Room") {
		t.Error("show should display room name")
	}

	// Test name command
	olc.processRoomCommand(ch, state, "name", "New Room Name")
	if room.Name != "New Room Name" {
		t.Errorf("expected room name to be 'New Room Name', got %q", room.Name)
	}
	if !state.Modified {
		t.Error("state should be marked as modified")
	}

	// Test description command
	olc.processRoomCommand(ch, state, "desc", "A new description")
	if room.Description != "A new description" {
		t.Errorf("expected description to be 'A new description', got %q", room.Description)
	}

	// Test sector command
	olc.processRoomCommand(ch, state, "sector", "forest")
	if room.Sector != types.SectForest {
		t.Errorf("expected sector to be forest, got %v", room.Sector)
	}

	// Test help command
	output.Reset()
	olc.processRoomCommand(ch, state, "help", "")
	if !strings.Contains(output.String(), "Room editor commands") {
		t.Error("help should display command list")
	}
}

func TestRoomExitEditing(t *testing.T) {
	olc := NewOLCSystem()

	var output strings.Builder
	olc.Output = func(ch *types.Character, msg string) {
		output.WriteString(msg)
	}

	room := types.NewRoom(3001, "Test Room", "A test room.")
	destRoom := types.NewRoom(3002, "Destination", "Destination room.")

	olc.GetRoom = func(vnum int) *types.Room {
		switch vnum {
		case 3001:
			return room
		case 3002:
			return destRoom
		}
		return nil
	}

	state := &EditorState{
		Mode:     EditorRoom,
		EditVnum: 3001,
		Data:     room,
	}

	ch := types.NewCharacter("TestBuilder")
	ch.PCData = &types.PCData{Security: 9}

	// Set an exit
	olc.processRoomCommand(ch, state, "north", "3002")

	exit := room.GetExit(types.DirNorth)
	if exit == nil {
		t.Fatal("north exit should be set")
	}
	if exit.ToVnum != 3002 {
		t.Errorf("expected exit to vnum 3002, got %d", exit.ToVnum)
	}

	// Delete the exit
	olc.processRoomCommand(ch, state, "north", "delete")
	exit = room.GetExit(types.DirNorth)
	if exit != nil {
		t.Error("north exit should be deleted")
	}
}

func TestRoomFlagEditing(t *testing.T) {
	olc := NewOLCSystem()

	var output strings.Builder
	olc.Output = func(ch *types.Character, msg string) {
		output.WriteString(msg)
	}

	room := types.NewRoom(3001, "Test Room", "A test room.")

	state := &EditorState{
		Mode:     EditorRoom,
		EditVnum: 3001,
		Data:     room,
	}

	ch := types.NewCharacter("TestBuilder")
	ch.PCData = &types.PCData{Security: 9}

	// Show flags
	output.Reset()
	olc.processRoomCommand(ch, state, "flags", "")
	if !strings.Contains(output.String(), "Room flags") {
		t.Error("flags command should show room flags")
	}

	// Toggle safe flag
	olc.processRoomCommand(ch, state, "flags", "safe")
	if !room.Flags.Has(types.RoomSafe) {
		t.Error("room should have safe flag")
	}

	// Toggle it off
	olc.processRoomCommand(ch, state, "flags", "safe")
	if room.Flags.Has(types.RoomSafe) {
		t.Error("room should not have safe flag")
	}
}

func TestMobileEditor(t *testing.T) {
	olc := NewOLCSystem()

	var output strings.Builder
	olc.Output = func(ch *types.Character, msg string) {
		output.WriteString(msg)
	}

	mob := &MobileTemplate{
		Vnum:      3001,
		Keywords:  []string{"goblin"},
		ShortDesc: "a goblin",
		Level:     5,
	}

	state := &EditorState{
		Mode:     EditorMobile,
		EditVnum: 3001,
		Data:     mob,
	}

	ch := types.NewCharacter("TestBuilder")
	ch.PCData = &types.PCData{Security: 9}

	// Test show command
	output.Reset()
	olc.processMobileCommand(ch, state, "show", "")
	if !strings.Contains(output.String(), "goblin") {
		t.Error("show should display mobile info")
	}

	// Test level command
	olc.processMobileCommand(ch, state, "level", "10")
	if mob.Level != 10 {
		t.Errorf("expected level 10, got %d", mob.Level)
	}

	// Test short command
	olc.processMobileCommand(ch, state, "short", "a fierce goblin")
	if mob.ShortDesc != "a fierce goblin" {
		t.Errorf("expected short desc 'a fierce goblin', got %q", mob.ShortDesc)
	}

	// Test special command
	olc.processMobileCommand(ch, state, "special", "spec_cast_mage")
	if mob.Special != "spec_cast_mage" {
		t.Errorf("expected special 'spec_cast_mage', got %q", mob.Special)
	}
}

func TestObjectEditor(t *testing.T) {
	olc := NewOLCSystem()

	var output strings.Builder
	olc.Output = func(ch *types.Character, msg string) {
		output.WriteString(msg)
	}

	obj := &ObjectTemplate{
		Vnum:      3001,
		Keywords:  []string{"sword"},
		ShortDesc: "a sword",
		ItemType:  types.ItemTypeWeapon,
		Level:     5,
		Cost:      100,
	}

	state := &EditorState{
		Mode:     EditorObject,
		EditVnum: 3001,
		Data:     obj,
	}

	ch := types.NewCharacter("TestBuilder")
	ch.PCData = &types.PCData{Security: 9}

	// Test show command
	output.Reset()
	olc.processObjectCommand(ch, state, "show", "")
	if !strings.Contains(output.String(), "sword") {
		t.Error("show should display object info")
	}

	// Test cost command
	olc.processObjectCommand(ch, state, "cost", "500")
	if obj.Cost != 500 {
		t.Errorf("expected cost 500, got %d", obj.Cost)
	}

	// Test weight command
	olc.processObjectCommand(ch, state, "weight", "10")
	if obj.Weight != 10 {
		t.Errorf("expected weight 10, got %d", obj.Weight)
	}
}

func TestParseHelpers(t *testing.T) {
	// Test parseDirection
	tests := []struct {
		input    string
		expected int
	}{
		{"north", int(types.DirNorth)},
		{"n", int(types.DirNorth)},
		{"south", int(types.DirSouth)},
		{"east", int(types.DirEast)},
		{"west", int(types.DirWest)},
		{"up", int(types.DirUp)},
		{"down", int(types.DirDown)},
		{"invalid", -1},
	}

	for _, tt := range tests {
		result := parseDirection(tt.input)
		if result != tt.expected {
			t.Errorf("parseDirection(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}

	// Test parseSector
	sectorTests := []struct {
		input    string
		expected int
	}{
		{"inside", int(types.SectInside)},
		{"city", int(types.SectCity)},
		{"forest", int(types.SectForest)},
		{"invalid", -1},
	}

	for _, tt := range sectorTests {
		result := parseSector(tt.input)
		if result != tt.expected {
			t.Errorf("parseSector(%q) = %d, expected %d", tt.input, result, tt.expected)
		}
	}
}
