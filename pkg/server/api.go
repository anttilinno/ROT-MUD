package server

import (
	"encoding/json"
	"net/http"
	"time"

	"rotmud/pkg/types"
)

// API response types

// PlayerInfo represents player data for API responses
type PlayerInfo struct {
	Name     string `json:"name"`
	Level    int    `json:"level"`
	Class    string `json:"class"`
	Room     string `json:"room"`
	RoomVnum int    `json:"room_vnum"`
	HP       int    `json:"hp"`
	MaxHP    int    `json:"max_hp"`
	Fighting string `json:"fighting,omitempty"`
}

// ServerStats represents server statistics
type ServerStats struct {
	Uptime        string `json:"uptime"`
	UptimeSeconds int64  `json:"uptime_seconds"`
	PlayersOnline int    `json:"players_online"`
	NPCsActive    int    `json:"npcs_active"`
	RoomsLoaded   int    `json:"rooms_loaded"`
	AreasLoaded   int    `json:"areas_loaded"`
	PulseCount    uint64 `json:"pulse_count"`
	CommandsTotal int64  `json:"commands_total"`
}

// API handlers

// handleAPIPlayers returns a list of online players
func (h *HTTPServer) handleAPIPlayers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	players := make([]PlayerInfo, 0)

	h.server.GameLoop.ForEachCharacter(func(ch *types.Character) {
		if ch.IsNPC() {
			return
		}

		info := PlayerInfo{
			Name:  ch.Name,
			Level: ch.Level,
			Class: getClassName(ch.Class),
			HP:    ch.Hit,
			MaxHP: ch.MaxHit,
		}

		if ch.InRoom != nil {
			info.Room = ch.InRoom.Name
			info.RoomVnum = ch.InRoom.Vnum
		}

		if ch.Fighting != nil {
			info.Fighting = ch.Fighting.Name
		}

		players = append(players, info)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"players": players,
		"count":   len(players),
	})
}

// handleAPIStats returns server statistics
func (h *HTTPServer) handleAPIStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Calculate uptime
	uptime := time.Since(h.server.startTime)

	// Count players and NPCs
	playerCount := 0
	npcCount := 0
	h.server.GameLoop.ForEachCharacter(func(ch *types.Character) {
		if ch.IsNPC() {
			npcCount++
		} else {
			playerCount++
		}
	})

	// Count rooms and areas
	roomCount := len(h.server.GameLoop.Rooms)
	areaCount := 0
	if h.server.World != nil {
		areaCount = len(h.server.World.Areas)
	}

	stats := ServerStats{
		Uptime:        formatDuration(uptime),
		UptimeSeconds: int64(uptime.Seconds()),
		PlayersOnline: playerCount,
		NPCsActive:    npcCount,
		RoomsLoaded:   roomCount,
		AreasLoaded:   areaCount,
		PulseCount:    h.server.GameLoop.GetPulseCount(),
		CommandsTotal: h.server.commandCount,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleAPIShutdown initiates a graceful shutdown
func (h *HTTPServer) handleAPIShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check for API key (basic security)
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" || apiKey != h.server.apiKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	h.logger.Info("Shutdown requested via API")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "shutting_down",
		"message": "Server shutdown initiated",
	})

	// Trigger shutdown in background
	go func() {
		time.Sleep(1 * time.Second) // Give time for response to be sent
		h.server.Stop()
	}()
}

// handleHealth returns a simple health check
func (h *HTTPServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
	})
}

// Helper functions

func getClassName(classIndex int) string {
	names := []string{"mage", "cleric", "thief", "warrior"}
	if classIndex >= 0 && classIndex < len(names) {
		return names[classIndex]
	}
	return "unknown"
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if days > 0 {
		return itoa(days) + "d " + itoa(hours) + "h " + itoa(minutes) + "m"
	}
	if hours > 0 {
		return itoa(hours) + "h " + itoa(minutes) + "m " + itoa(seconds) + "s"
	}
	if minutes > 0 {
		return itoa(minutes) + "m " + itoa(seconds) + "s"
	}
	return itoa(seconds) + "s"
}
