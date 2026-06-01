---
phase: 02
slug: trait-type-system
status: verified
threats_open: 0
asvs_level: 1
created: 2026-06-01
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| (none this phase) | Pure-Go in-memory data-structure package (`go/pkg/traits/`). No network I/O, no user-input parsing, no persistence, no auth. Capability-string interning is fed only by trusted in-process callers this phase. | None — no untrusted input crosses a boundary in Phase 2. |

*Forward-looking: P3 will feed TOML-sourced capability strings into `internCapability`/`Resolve`. The 256-bit ceiling and per-axis clamps in this phase are the bounded-growth controls carried into that future input boundary.*

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-02-01 | Denial of Service | capability registry (`internCapability`) | mitigate | 256-bit ceiling returns `(0,false)` on overflow, never panics or grows unbounded (D-06). `capability.go:79-81`; test `capability_test.go:96-120` (257th-key overflow, recover-guarded). | closed |
| T-02-02 | Denial of Service | per-axis RIS sum (`Resolve`) | mitigate | Per-axis `[-CAP,+CAP]` clamp (CAP=100, D-03) applied incrementally — bounded magnitude regardless of source count (TRAIT-02). Stat sums clamped `±ModCap=25`. `resolve.go:13,68-76,80-85`; test `resolve_test.go:10-34` (+60+60 Fire cap boundary). | closed |
| T-02-03 | Denial of Service | capability interning in `Resolve` | accept | 256-bit ceiling from registry bounds growth; `Resolve` skips overflow via `(_,false)` without panic. `resolve.go:93-100`. See Accepted Risks Log. | closed |
| T-02-SC | Tampering | npm/pip/cargo installs | accept | No package-manager installs this phase; pure stdlib + existing `rotmud/pkg/types`. `go.mod`/`go.sum` untouched (verified `git diff`). See Accepted Risks Log. | closed |

*Status: open · closed*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-01 | T-02-03 | Capability interning overflow in `Resolve` is bounded by the 256-bit registry ceiling (T-02-01) and skipped without panic. No untrusted input crosses a boundary this phase; interning is fed only by trusted in-process callers. Forward-looking note: P3 feeds TOML-sourced strings into this path — the ceiling is the carried-forward control. | gsd-security-auditor (verified in code) | 2026-06-01 |
| AR-02-02 | T-02-SC | No package-manager installs in Phase 2 — pure stdlib + existing `rotmud/pkg/types`. Zero new Go modules added; no supply-chain attack surface. Re-confirmed against `go.mod`/`go.sum` (unchanged). | gsd-security-auditor (verified in code) | 2026-06-01 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-01 | 4 | 4 | 0 | gsd-security-auditor (opus) |

**Informational (not blockers, no new boundary):** review-added `sync.RWMutex` (`capability.go:54`) guards the package-level registry against a concurrent map read/write race while preserving the SC#4 zero-alloc query path; `slog.Warn` on dropped capability (`resolve.go:97`) replaced a silent drop. Both fed only by trusted in-process callers this phase.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-01
