// Package server implements the network layer for the ROT MUD.
//
// This package handles TCP/telnet connections, WebSocket support,
// REST API endpoints, and Prometheus metrics. It is the modern
// replacement for the original comm.c networking code.
//
// # TCP/Telnet Server
//
// The main server accepts telnet connections on the configured port
// (default 4000). Each connection runs in its own goroutine and
// communicates with the game loop via channels.
//
// # WebSocket Support
//
// For web-based clients, the server provides WebSocket connections
// on a separate port (default 4001). This enables browser-based
// MUD clients without requiring telnet.
//
// # REST API
//
// Administrative endpoints include:
//
//   - GET /api/players: List online players
//   - GET /api/stats: Server statistics
//   - POST /api/shutdown: Graceful shutdown
//
// # Prometheus Metrics
//
// The server exposes metrics at /metrics:
//
//   - rotmud_players_online: Current player count (gauge)
//   - rotmud_commands_total: Commands processed (counter)
//   - rotmud_combat_damage: Damage dealt (histogram)
//   - rotmud_pulse_latency_seconds: Game loop timing (histogram)
//
// # Configuration
//
// Server configuration is loaded from data/config.toml:
//
//	[server]
//	telnet_port = 4000
//	websocket_port = 4001
//	pulse_ms = 250
//
//	[logging]
//	level = "info"
//	format = "json"
//
// # Connection Lifecycle
//
//  1. Client connects via TCP or WebSocket
//  2. Server creates Descriptor and sends greeting
//  3. Login/character creation state machine
//  4. Playing state: commands sent to game loop
//  5. Disconnect: cleanup and save
//
// # Usage Example
//
//	cfg := &server.Config{
//	    TelnetPort:    4000,
//	    WebSocketPort: 4001,
//	}
//
//	srv := server.New(cfg, gameLoop)
//	srv.Start()
//	defer srv.Stop()
//
//	// Server runs until stopped
//	<-ctx.Done()
package server
