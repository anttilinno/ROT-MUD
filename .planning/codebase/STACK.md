# Technology Stack

**Analysis Date:** 2026-04-16

## Languages

**Primary:**
- Go 1.25.5 - Entire server codebase and utilities (`go/cmd/`, `go/pkg/`)

**Supporting:**
- TOML - Configuration and world data format
- JSON - Player persistence files

## Runtime

**Environment:**
- Go 1.23+ (specified in `.mise.toml`)
- Linux/Unix compatible (server uses `net.Listener`, pipes, POSIX signals)

**Package Manager:**
- Go Modules (go.mod/go.sum)
- Lockfile: Present at `/home/antti/Repos/Misc/ROT-MUD/go/go.sum`

## Frameworks

**Core:**
- Standard Library networking (`net`, `net/http`, `bufio`) - TCP server, WebSocket upgrade path, REST API server
- `log/slog` - Structured logging (configured in `go/cmd/rotmud/main.go`)

**Networking:**
- `github.com/gorilla/websocket` v1.5.3 - WebSocket support for real-time client connections (`go/pkg/server/websocket.go`)

**Configuration & Data:**
- `github.com/pelletier/go-toml/v2` v2.2.4 - TOML parsing for config and world data (`go/pkg/loader/loader.go`)

**Monitoring & Observability:**
- `github.com/prometheus/client_golang` v1.23.2 - Prometheus metrics collection (`go/pkg/server/metrics.go`)

## Key Dependencies

**Critical:**
- `github.com/gorilla/websocket` v1.5.3 - Required for WebSocket protocol support (dual protocol: Telnet + WebSocket)
- `github.com/pelletier/go-toml/v2` v2.2.4 - Required for TOML parsing of configuration and 61 game areas
- `github.com/prometheus/client_golang` v1.23.2 - Required for /metrics endpoint (production observability)

**Transitive (Monitoring):**
- `github.com/prometheus/common` v0.66.1 - Prometheus common utilities
- `github.com/prometheus/procfs` v0.16.1 - Process metrics for Prometheus
- `google.golang.org/protobuf` v1.36.8 - Protocol Buffers for metrics serialization
- `golang.org/x/sys` v0.35.0 - System calls (POSIX signal handling)

## Configuration

**Environment:**
- Single TOML file at `go/data/config.toml` (not environment variables)
- Key configuration values:
  - `telnet_port`: 4000 (Telnet protocol)
  - `websocket_port`: 4001 (WebSocket protocol)
  - `api_port`: 4002 (REST API)
  - `pulse_ms`: 250 (game loop cycle time in milliseconds)
  - `data_path`: "data" (relative path to world data)
  - `logging.level`: "info" or "debug"
  - `security.api_key`: Default "changeme" (hardcoded, needs production override)

**Build:**
- `.mise.toml` defines build tasks: `build`, `dev`, `test`, `clean`, `areconv`, `convert-areas`
- Go binary output: `go/rotmud` (12MB compiled, no runtime dependencies)

## Platform Requirements

**Development:**
- Go 1.23+
- `mise` (optional, for task runner)
- `make` (optional, for traditional builds)
- Standard POSIX toolchain

**Production:**
- Linux/Unix server with at least:
  - ~50MB RAM (game world + 61 areas, ~4,072 rooms)
  - 2+ CPU cores (game loop on main thread, concurrent client handling)
  - Port access: 4000 (Telnet), 4001 (WebSocket), 4002 (REST API)
  - Filesystem for player persistence (`data/players/`)

**Data:**
- 61 game areas in TOML format (`go/data/areas/`)
- Player save files in JSON format (`go/data/players/`)
- Configuration: `go/data/config.toml`
- Help files (text format, location: `go/data/help/`)

## Networking & Protocols

**Protocols Supported:**
- Telnet (RFC 854) on port 4000 - Traditional MUD access (`go/pkg/server/server.go`)
- WebSocket (RFC 6455) on port 4001 - Browser-based clients (`go/pkg/server/websocket.go`)
- HTTP/1.1 on port 4002 - REST API and Prometheus metrics (`go/pkg/server/api.go`)

**Connection Model:**
- TCP listener accepts both Telnet and WebSocket connections
- Each connection handled concurrently with `*bufio.Reader/*bufio.Writer`
- Session management in `Session` struct with `net.Conn` per client

## Testing Stack

**Framework:**
- Go built-in `testing` package
- Test files: 26 total across the codebase
- Run with: `go test ./...` (configured in `.mise.toml`)
- Popular test utilities: `github.com/stretchr/testify` v1.11.1 (available in dependencies)

**Build Artifacts:**
- Binary: `go/rotmud` (12.5MB when compiled)
- No external runtime dependencies (fully static binary)

---

*Stack analysis: 2026-04-16*
