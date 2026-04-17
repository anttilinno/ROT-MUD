---
phase: 1
slug: golden-master-safety-net
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-04-17
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — stdlib only |
| **Quick run command** | `cd go && go test ./pkg/golden/... -run TestGoldenMaster -timeout 30s` |
| **Full suite command** | `cd go && go test ./... -timeout 120s` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd go && go test ./pkg/golden/... -run TestGoldenMaster -timeout 30s`
- **After every plan wave:** Run `cd go && go test ./... -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | MIGRATE-06 | — | N/A | unit | `cd go && go build ./pkg/combat/...` | ❌ W0 | ⬜ pending |
| 1-01-02 | 01 | 1 | MIGRATE-06 | — | N/A | unit | `cd go && go test ./pkg/golden/... -run TestGoldenMaster -count=2` | ❌ W0 | ⬜ pending |
| 1-01-03 | 01 | 1 | MIGRATE-06 | — | N/A | golden | `cd go && go test ./pkg/golden/... -run TestGoldenMaster -update && go test ./pkg/golden/... -run TestGoldenMaster -count=2` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `go/pkg/golden/` directory — create package skeleton
- [ ] `go/pkg/combat/dice.go` — add `SetRand()` hook for deterministic RNG
- [ ] `go/pkg/golden/golden_test.go` — stub `TestGoldenMaster` so `go test` resolves

*Existing `go test ./...` infrastructure covers the rest.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Byte-identical golden file on two successive runs | MIGRATE-06 | CI diff requires `-update` flag management | Run `go test ./pkg/golden/... -run TestGoldenMaster -count=2` and confirm no diff output |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
