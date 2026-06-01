# Security Audit — Phase 02: Trait Type System

**Audited:** 2026-06-01
**ASVS Level:** 1
**block_on:** high
**Result:** SECURED — 4/4 threats closed (2 mitigate verified in code, 2 accept verified)

This audit verifies each declared threat in the Phase 02 threat register against the
implemented code (`go/pkg/traits/`). Evidence is a concrete `file:line` match, not
documentation or intent. Implementation files were not modified.

---

## Threat Verification

| Threat ID | Category | Component | Disposition | Status | Evidence |
|-----------|----------|-----------|-------------|--------|----------|
| T-02-01 | Denial of Service | capability registry (`internCapability`) | mitigate | CLOSED | `go/pkg/traits/capability.go:79-81` — 256-bit ceiling returns `(0, false)` on overflow, no panic. Test: `go/pkg/traits/capability_test.go:96-120` (257th-key overflow without panic). |
| T-02-02 | Denial of Service | per-axis RIS sum (`Resolve`) | mitigate | CLOSED | `go/pkg/traits/resolve.go:68-76` — per-axis `clamp()` to `[-CAP,+CAP]` (CAP=100, `resolve.go:13`), applied incrementally. Stat sums clamped to `±ModCap=25` (`resolve.go:80-85`). Test: `go/pkg/traits/resolve_test.go:10-34` (+60+60 Fire clamps at cap). |
| T-02-03 | Denial of Service | capability interning in `Resolve` | accept | CLOSED | Acceptance rationale holds: `go/pkg/traits/resolve.go:93-100` skips overflow without panic via `(_, false)` signal. Bounded by the 256-bit ceiling (T-02-01). Documented as accepted risk below. |
| T-02-SC | Tampering | npm/pip/cargo installs | accept | CLOSED | `go.mod`/`go.sum` untouched in phase (`git diff --name-only 96bcc17^..HEAD` shows only `go/pkg/traits/*` + planning docs). `require` block unchanged: pure stdlib + existing `rotmud/pkg/types`. Documented as accepted risk below. |

---

## Mitigation Detail

### T-02-01 — capability registry overflow (mitigate, CLOSED)

`internCapability` (`capability.go:63-86`) enforces the 256-bit ceiling: when
`capNextBit >= capBitsCeiling` (256) and the key is not yet registered, it returns
`(0, false)` rather than panicking or growing the registry. `CapBits.Has`/`Set`
(`capability.go:20-35`) additionally bounds-guard out-of-range bit indices, so an
exported caller passing an arbitrary `bit` cannot index out of bounds.

Verified by `TestCapabilityOverflow` (`capability_test.go:96`): registers 256 keys,
asserts the 257th returns the overflow signal inside a `recover()` guard that fails
the test on panic.

### T-02-02 — per-axis RIS additive stacking (mitigate, CLOSED)

`Resolve()` (`resolve.go:56-103`) sums each damage axis and applies `clamp()` to
`[-CAP, +CAP]` **incrementally** after every addition (`resolve.go:69,72,75`), and
also clamps each individual magnitude before adding. This makes int overflow
impossible regardless of source count or magnitude size (relevant for P3 TOML-sourced
magnitudes). Stat modifiers are clamped identically to `±ModCap` (`resolve.go:84`)
with an out-of-range stat-index bounds guard (`resolve.go:81-83`).

Verified by `resolve_test.go:10-34` (two +60 Fire entries clamp to the cap boundary
for both vulnerability and resistance directions).

---

## Accepted Risks Log

### T-02-03 — capability interning in Resolve (DoS, accepted)

**Accepted.** Capability interning during `Resolve()` is bounded by the 256-bit
registry ceiling (T-02-01). On overflow, `Resolve()` (`resolve.go:93-100`) skips the
unregistrable key without panicking and emits an `slog.Warn` identifying the dropped
key. Rationale still holds in code: no unbounded growth, no panic. Forward-looking —
P3 will feed TOML-sourced capability strings into this path; the ceiling is the
defensive bound carried forward. No untrusted input crosses a boundary in Phase 02.

### T-02-SC — supply chain / package-manager installs (Tampering, accepted)

**Accepted.** No new Go modules were added this phase. `go.mod` and `go.sum` are
untouched (verified by `git diff --name-only 96bcc17^..HEAD`: only `go/pkg/traits/*`
and planning docs changed). The package uses pure stdlib (`sync`, `log/slog`) plus the
pre-existing `rotmud/pkg/types`. No supply-chain surface introduced.

---

## Unregistered Flags

Neither `02-01-SUMMARY.md` nor `02-02-SUMMARY.md` contains a `## Threat Flags`
section. No new attack surface was declared by the executor.

**Informational (not blockers, no new untrusted-input boundary):**

- A `sync.RWMutex` (`capMu`, `capability.go:54`) was added during code review
  (WR-01..WR-05 / commits `5f25fc1`..`f50c01e`) to guard the package-level registry
  against concurrent map read/write. This is a concurrency hardening that *closes* a
  latent data-race gap; it does not introduce new attack surface. The uncontended
  `RLock` does not allocate, preserving the SC#4 zero-alloc `HasCapability` path
  (verified by `query_test.go:143` `AllocsPerRun == 0`).
- `slog.Warn` on dropped capability (`resolve.go:97`) replaced a silent drop — an
  observability improvement, no new boundary.

These remain pure in-memory data-structure operations fed only by trusted in-process
callers in Phase 02. No threat-register entry required.

---

## Verification Commands Run

- `git diff --name-only 96bcc17^..HEAD` — phase touched only `go/pkg/traits/*` + planning docs; `go.mod`/`go.sum` not in the list (T-02-SC).
- `grep -n` over `capability.go`, `resolve.go`, `query.go` — located each mitigation at the cited lines.
- `grep -n` over the `*_test.go` files — confirmed the 257th-key overflow test, the +60+60 Fire cap-boundary test, and `AllocsPerRun == 0` zero-alloc test cited in the register exist.

No implementation files were modified during this audit.
