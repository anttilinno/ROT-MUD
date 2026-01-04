package server

import (
	"bufio"
	"bytes"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"rotmud/pkg/types"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins for now (in production, restrict this)
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocketLoginHandler wraps the standard login handler for WebSocket sessions
type WebSocketLoginHandler struct {
	handler   *LoginHandler
	wsSession *WebSocketSession
	isNew     bool
}

// NewWebSocketLoginHandler creates a login handler for a WebSocket session
func NewWebSocketLoginHandler(handler *LoginHandler, ws *WebSocketSession) *WebSocketLoginHandler {
	return &WebSocketLoginHandler{
		handler:   handler,
		wsSession: ws,
	}
}

// HandleInput processes input during login
func (wl *WebSocketLoginHandler) HandleInput(input string) bool {
	// Create a temporary session wrapper for the login handler
	tempSession := &Session{
		Descriptor: wl.wsSession.Descriptor,
		Character:  wl.wsSession.Character,
	}

	// Set up output to go to WebSocket
	wl.handler.Output = func(sess *Session, msg string) {
		wl.wsSession.Write(msg)
	}

	// Process the input
	result := wl.handler.HandleInput(tempSession, input)

	// Copy character back if created
	if tempSession.Character != nil {
		wl.wsSession.Character = tempSession.Character
		wl.wsSession.Character.Descriptor = wl.wsSession.Descriptor
		wl.wsSession.Descriptor.Character = wl.wsSession.Character
	}

	// Track if new player
	if result {
		wl.isNew = wl.handler.IsNewPlayer()
	}

	return result
}

// IsNewPlayer returns true if this is a new character
func (wl *WebSocketLoginHandler) IsNewPlayer() bool {
	return wl.isNew
}

// Reset clears the login state
func (wl *WebSocketLoginHandler) Reset() {
	wl.handler.ResetState()
	wl.isNew = false
}

// WebSocketSession wraps a WebSocket connection as a session
type WebSocketSession struct {
	conn       *websocket.Conn
	Character  *types.Character
	Descriptor *types.Descriptor
	server     *Server
	mu         sync.Mutex
	writeChan  chan string
	done       chan struct{}
	login      *WebSocketLoginHandler
}

// NewWebSocketSession creates a new WebSocket session
func NewWebSocketSession(conn *websocket.Conn, server *Server) *WebSocketSession {
	return &WebSocketSession{
		conn:      conn,
		server:    server,
		writeChan: make(chan string, 64),
		done:      make(chan struct{}),
	}
}

// Write sends text to the WebSocket client
func (ws *WebSocketSession) Write(text string) {
	select {
	case ws.writeChan <- text:
	default:
		// Channel full, drop message
	}
}

// WriteLine sends text with a newline
func (ws *WebSocketSession) WriteLine(text string) {
	ws.Write(text + "\r\n")
}

// writeLoop handles outgoing messages
func (ws *WebSocketSession) writeLoop() {
	for {
		select {
		case <-ws.done:
			return
		case msg := <-ws.writeChan:
			ws.mu.Lock()
			err := ws.conn.WriteMessage(websocket.TextMessage, []byte(msg))
			ws.mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}

// readLoop handles incoming messages
func (ws *WebSocketSession) readLoop() {
	defer func() {
		close(ws.done)
		ws.conn.Close()
	}()

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			return
		}

		// Process each line
		reader := bufio.NewReader(bytes.NewReader(message))
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				// Process any remaining text
				line = strings.TrimSpace(string(message))
				if line != "" {
					ws.processInput(line)
				}
				break
			}
			line = strings.TrimSpace(line)
			if line != "" {
				ws.processInput(line)
			}
		}
	}
}

// readLoopWithLogin handles incoming messages during login/creation and playing
func (ws *WebSocketSession) readLoopWithLogin(onLoggedIn func()) {
	defer func() {
		close(ws.done)
		ws.conn.Close()
	}()

	loggedIn := false

	for {
		_, message, err := ws.conn.ReadMessage()
		if err != nil {
			return
		}

		line := strings.TrimSpace(string(message))
		if line == "" {
			continue
		}

		if !loggedIn && ws.login != nil {
			// Still in login/creation phase
			if ws.login.HandleInput(line) {
				loggedIn = true
				if onLoggedIn != nil {
					onLoggedIn()
				}
				ws.Write("\r\n> ")
			}
		} else {
			// Playing phase
			ws.processInput(line)
		}
	}
}

// processInput handles a line of input from the WebSocket client
func (ws *WebSocketSession) processInput(line string) {
	if ws.Character == nil {
		return
	}

	// Handle quit command specially
	if strings.ToLower(line) == "quit" {
		ws.WriteLine("Goodbye!")
		ws.conn.Close()
		return
	}

	// Queue command for game loop processing
	ws.server.GameLoop.QueueCommand(ws.Character, line)
}

// Close closes the WebSocket session
func (ws *WebSocketSession) Close() {
	select {
	case <-ws.done:
		// Already closed
	default:
		close(ws.done)
	}
	ws.conn.Close()
}

// HTTPServer handles HTTP and WebSocket connections
type HTTPServer struct {
	server     *Server
	logger     *slog.Logger
	httpServer *http.Server
	mux        *http.ServeMux
	wsSessions map[*websocket.Conn]*WebSocketSession
	mu         sync.RWMutex
	Metrics    *Metrics
}

// NewHTTPServer creates a new HTTP server
func NewHTTPServer(server *Server, logger *slog.Logger) *HTTPServer {
	h := &HTTPServer{
		server:     server,
		logger:     logger,
		mux:        http.NewServeMux(),
		wsSessions: make(map[*websocket.Conn]*WebSocketSession),
		Metrics:    NewMetrics(),
	}

	// Register handlers
	h.mux.HandleFunc("/ws", h.handleWebSocket)
	h.mux.HandleFunc("/api/players", h.handleAPIPlayers)
	h.mux.HandleFunc("/api/stats", h.handleAPIStats)
	h.mux.HandleFunc("/api/shutdown", h.handleAPIShutdown)
	h.mux.HandleFunc("/health", h.handleHealth)
	h.mux.Handle("/metrics", h.Metrics.Handler())

	return h
}

// Start starts the HTTP server
func (h *HTTPServer) Start(port int) error {
	addr := ":" + itoa(port)
	h.httpServer = &http.Server{
		Addr:    addr,
		Handler: h.mux,
	}

	h.logger.Info("HTTP/WebSocket server starting", "port", port)
	return h.httpServer.ListenAndServe()
}

// Stop stops the HTTP server
func (h *HTTPServer) Stop() error {
	if h.httpServer != nil {
		return h.httpServer.Close()
	}
	return nil
}

// handleWebSocket upgrades HTTP to WebSocket and handles the connection
func (h *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("WebSocket upgrade failed", "error", err)
		return
	}

	remoteAddr := r.RemoteAddr
	h.logger.Info("WebSocket connection", "addr", remoteAddr)

	// Create WebSocket session
	wsSession := NewWebSocketSession(conn, h.server)

	// Create descriptor (starts in login state)
	desc := types.NewDescriptor(remoteAddr)
	desc.State = types.ConGetName
	wsSession.Descriptor = desc

	// Register session
	h.mu.Lock()
	h.wsSessions[conn] = wsSession
	h.mu.Unlock()

	// Create login handler for this session
	wsSession.login = NewWebSocketLoginHandler(h.server.Login, wsSession)

	// Send greeting and name prompt
	wsSession.Write(h.getGreeting())
	wsSession.Write("\r\nBy what name do you wish to be known? ")

	// Start write loop
	go wsSession.writeLoop()

	// Read loop with login - blocks until connection closes
	wsSession.readLoopWithLogin(func() {
		// Called when login is complete
		h.enterGameWS(wsSession)
	})

	// Cleanup
	h.mu.Lock()
	delete(h.wsSessions, conn)
	h.mu.Unlock()

	// Save character if logged in (but not level 1 - avoid abandoned characters)
	if wsSession.Character != nil && wsSession.Character.PCData != nil {
		// Dismiss any pets/followers before disconnecting
		h.server.dismissAllFollowers(wsSession.Character)

		if wsSession.Character.Level > 1 {
			if err := h.server.Persistence.SavePlayer(wsSession.Character); err != nil {
				h.logger.Error("failed to save player on WS disconnect", "error", err, "name", wsSession.Character.Name)
			}
		}
		h.server.GameLoop.RemoveCharacter(wsSession.Character)
		if wsSession.Character.InRoom != nil {
			wsSession.Character.InRoom.RemovePerson(wsSession.Character)
		}
	}

	// Clean up WebSocket output registration
	h.server.unregisterWebSocketOutput(wsSession)

	h.logger.Info("WebSocket disconnected", "addr", remoteAddr)
}

// enterGameWS brings a WebSocket player into the game world
func (h *HTTPServer) enterGameWS(ws *WebSocketSession) {
	ch := ws.Character
	if ch == nil {
		return
	}

	// Find starting room
	var startRoom *types.Room

	// New players start in MUD School (if it exists)
	if ws.login != nil && ws.login.IsNewPlayer() {
		// Give starting equipment to new characters
		h.server.giveStartingEquipment(ch)

		if h.server.World != nil {
			startRoom = h.server.World.GetRoom(RoomVnumMudSchool)
		}
		if startRoom != nil {
			ws.WriteLine("")
			ws.WriteLine("Welcome to the MUD School! Here you can learn the basics of the game.")
			ws.WriteLine("Type 'help' to see available commands, and 'look' to see the room.")
			ws.WriteLine("")
		}
	}

	// Returning players go to their recall point or saved room
	if startRoom == nil && ch.PCData != nil && ch.PCData.Recall != 0 && h.server.World != nil {
		startRoom = h.server.World.GetRoom(ch.PCData.Recall)
	}

	// Otherwise use default temple
	if startRoom == nil {
		startRoom = h.server.getOrCreateStartRoom()
	}

	// Place character in room
	ch.InRoom = startRoom
	startRoom.AddPerson(ch)

	// Add to game loop
	h.server.GameLoop.AddCharacter(ch)

	// Register for output
	h.server.registerWebSocketOutput(ws)

	// Welcome message
	ws.WriteLine("")
	ws.WriteLine("Welcome to Rivers of Time, " + ch.Name + "!")
	ws.WriteLine("")

	// Show the room
	h.server.Dispatcher.Registry.Execute("look", ch, "")

	// Reset login state
	if ws.login != nil {
		ws.login.Reset()
	}
}

// getGreeting returns the MUD greeting
func (h *HTTPServer) getGreeting() string {
	return `
 ____  _                             __   _____ _                
|  _ \(_)_   _____ _ __ ___    ___  / _| |_   _(_)_ __ ___   ___ 
| |_) | \ \ / / _ \ '__/ __|  / _ \| |_    | | | | '_ ` + "`" + ` _ \ / _ \
|  _ <| |\ V /  __/ |  \__ \ | (_) |  _|   | | | | | | | | |  __/
|_| \_\_| \_/ \___|_|  |___/  \___/|_|     |_| |_|_| |_| |_|\___|

===============================================================================

                Original DikuMUD by Hans Staerfeldt, Katja Nyboe,
                Tom Madsen, Michael Seifert, and Sebastian Hammer
                Based on MERC 2.1 code by Hatchet, Furey, and Kahn
                ROM 2.4 copyright (c) 1993-1995 Russ Taylor
                ROT 1.4 copyright (c) 1996-1997 Russ Walsh
                Go port by the ROT team

===============================================================================
`
}

// registerWebSocketOutput registers a WebSocket session for output
func (s *Server) registerWebSocketOutput(ws *WebSocketSession) {
	// Store in a map for SendToCharacter to find
	s.mu.Lock()
	if s.wsSessions == nil {
		s.wsSessions = make(map[*types.Character]*WebSocketSession)
	}
	s.wsSessions[ws.Character] = ws
	s.mu.Unlock()
}

// unregisterWebSocketOutput removes a WebSocket session from output registration
func (s *Server) unregisterWebSocketOutput(ws *WebSocketSession) {
	s.mu.Lock()
	if s.wsSessions != nil && ws.Character != nil {
		delete(s.wsSessions, ws.Character)
	}
	s.mu.Unlock()
}

// Helper for int to string
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
