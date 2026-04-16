# External Integrations

**Analysis Date:** 2026-04-16

## APIs & External Services

**Not Used:**
This is a self-contained MUD server with no external API integrations. All game data, world content, NPCs, spells, and skills are embedded in the binary or loaded from local TOML/JSON files.

## Data Storage

**Databases:**
- Not used - No relational database or NoSQL integration
- State: All game world (61 areas, 4,072 rooms, 1,341 NPCs, 1,677 objects) loaded into memory at startup from TOML files
- Player persistence: File-based JSON format

**Player Persistence:**
- Location: `go/data/players/`
- Format: JSON files (one per player, named `{PlayerName}.json`)
- Client: Custom JSON marshaling in `go/pkg/persistence/player.go`
- Example: `Aldor.json` (level 60 mage, 17KB file)
- Schema: `PlayerSave` struct in `go/pkg/persistence/player.go` includes:
  - Character vitals (HP, mana, move)
  - Stats, alignment, level, class, race
  - Inventory and equipment
  - Skills and spell knowledge
  - Clan membership and quest progress
  - Play time and experience

**World Data:**
- Format: TOML (Topic in Configuration Format)
- Location: `go/data/areas/{area_name}/`
- Structure per area:
  - `area.toml` - Area metadata (name, credits, reset interval, vnum range)
  - `rooms/rooms.toml` - Room descriptions and connections
  - `mobs/mobs.toml` - NPC definitions (mobiles)
  - `objects/objects.toml` - Object definitions
- Parser: `github.com/pelletier/go-toml/v2` v2.2.4 in `go/pkg/loader/loader.go`
- Loader: `LoadRoomsFromString()`, `LoadMobsFromString()`, `LoadObjectsFromString()` functions
- Legacy ROM format: Original `.are` files (C source format) are converted to TOML via `go/cmd/areconv/` tool

**File Storage:**
- Local filesystem only
- Directories managed by `go/pkg/persistence/`:
  - Player saves: `data/players/`
  - World data: `data/areas/`
  - Help files: `data/help/`

**Caching:**
- In-memory cache only (no Redis, Memcached, or similar)
- Game world loaded entirely into memory at server startup
- Session state cached in `Server.sessions` map

## Authentication & Identity

**Auth Provider:**
- Custom implementation (not OAuth/SAML/external)
- Location: `go/pkg/server/login.go` (`LoginHandler` struct)
- Mechanism:
  - Username/password flow (text-based, standard MUD authentication)
  - Password stored as plaintext in player JSON (no hashing) - **security concern**
  - No token-based auth (pure session-based)
  - No external identity provider

**API Security:**
- Simple API key validation in `go/pkg/server/api.go`
- API key: Hardcoded in `go/pkg/server/server.go` line 89 as "changeme"
- Used for REST API endpoints (authentication weak, not production-ready)

## Monitoring & Observability

**Error Tracking:**
- Not used - No integration with Sentry, Rollbar, Datadog, etc.
- Errors logged via `slog.Logger.Error()` to stdout

**Logs:**
- Standard structured logging with `log/slog`
- Format: Text (configurable to JSON in `go/data/config.toml`)
- Destination: stdout (application responsible for log aggregation if needed)
- Log level: Configurable (default: "info")
- No external log aggregation service (ELK, Splunk, etc.)

**Metrics:**
- Prometheus metrics exposed on port 4002
- Endpoint: `http://localhost:4002/metrics`
- Handler: `promhttp.Handler()` from `github.com/prometheus/client_golang`
- Metrics tracked:
  - `rotmud_players_online` - Current player count (gauge)
  - `rotmud_npcs_active` - Active NPC count (gauge)
  - `rotmud_commands_total` - Total commands processed (counter)
  - `rotmud_combat_damage` - Combat damage histogram (buckets: 1-1000)
  - `rotmud_pulse_latency_seconds` - Game loop cycle time (histogram, 1ms-1s)
  - `rotmud_spells_cast_total` - Total spells cast (counter)
  - `rotmud_connections_open` - Open connections (Telnet + WebSocket, gauge)
- Metric registration: `go/pkg/server/metrics.go` (NewMetrics function, lines 22-68)
- Metrics update: Callback-based via `Server.metricsUpdater` function

## CI/CD & Deployment

**Hosting:**
- Not specified - Server is self-contained and can run on any Linux/Unix system
- No cloud provider integration (AWS, Google Cloud, Azure, Heroku, etc.)
- Deployment: Binary + data directory

**CI Pipeline:**
- Not configured - No `.github/workflows/`, `.gitlab-ci.yml`, Jenkinsfile, or similar
- Build: Manual via `mise run build` or `go build`
- Tests: Manual via `mise run test` or `go test ./...`

**Deployment Strategy:**
- Single binary (`go/rotmud`) executable
- Data directory (`go/data/`) must be in same location or path configured
- Startup: `./rotmud` (listening on ports 4000, 4001, 4002)
- Graceful shutdown: Controlled via `server.shutdownCh`
- Reboot support: `shutdownCh` can signal reboot vs. shutdown (line 74 in server.go)

## Environment Configuration

**Required env vars:**
- None - All configuration via `go/data/config.toml`
- Application defaults if file missing:
  - Telnet port: 4000
  - WebSocket port: 4001
  - Pulse: 250ms
  - Data path: "data"

**Optional env vars:**
- None detected - Configuration is entirely file-based

**Secrets location:**
- `.env` file: Not used
- Secrets in config: Hardcoded in source:
  - API key: `go/pkg/server/server.go` line 89 ("changeme")
  - TODO comment: "Load from config" suggests future improvement
- Production concern: Secrets should be externalized before production use

## Webhooks & Callbacks

**Incoming Webhooks:**
- None - Server does not accept external webhook events

**Outgoing Webhooks:**
- None - Server does not call external webhooks

**Event Callbacks:**
- Internal use only:
  - `MetricsUpdaterFunc` - Called to update metrics
  - `OnCommandFunc` - Called when command processes
  - Combat callbacks (`OnDeath`, `OnLevelUp`)
  - Area reset callbacks
  - Character output routing callbacks
- All callbacks internal to server (for wiring game systems together)

## Network Architecture

**Protocols:**
- Telnet (TCP port 4000) - Raw TCP with \r\n line endings
- WebSocket (TCP port 4001) - HTTP upgrade to WebSocket (RFC 6455)
- HTTP (TCP port 4002) - REST API + Prometheus metrics

**Concurrent Connections:**
- Unlimited (no hardcoded limit in code review)
- Each connection spawned as goroutine
- Session state: `map[net.Conn]*Session` managed by `Server.mu` (RWMutex)
- WebSocket sessions: Separate tracking in `map[*types.Character]*WebSocketSession`

**Message Protocol:**
- Telnet: Text commands with \r\n terminators
- WebSocket: JSON message format (custom, not STOMP/AMQP)
- API: REST endpoints returning JSON

---

*Integration audit: 2026-04-16*
