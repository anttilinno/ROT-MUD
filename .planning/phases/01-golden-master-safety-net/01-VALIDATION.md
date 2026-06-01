---
phase: 1
slug: golden-master-safety-net
status: validated
nyquist_compliant: true
wave_0_complete: true
created: 2026-04-17
validated: 2026-06-01
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | go test |
| **Config file** | none — stdlib only |
| **Quick run command** | `cd go && go test ./pkg/golden/... -run TestGolden -timeout 30s` |
| **Full suite command** | `cd go && go test ./... -timeout 120s` |
| **Estimated runtime** | ~10 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd go && go test ./pkg/golden/... -run TestGolden -timeout 30s`
- **After every plan wave:** Run `cd go && go test ./... -timeout 120s`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|-----------|-------------------|-------------|--------|
| 1-01-01 | 01 | 1 | MIGRATE-06 | unit | `cd go && go test ./pkg/combat/ -run 'TestSetRandDeterministic\|TestSetRandRestore\|TestDefaultRandFallsThroughToGlobal\|TestEdgeCasesPreserved' -count=2 -timeout 30s` | ✅ pkg/combat/dice_test.go | ✅ green |
| 1-01-02 | 01 | 1 | MIGRATE-06 | unit | `cd go && go test ./pkg/combat/ -run 'TestCombatSystemRandField\|TestCombatSystemRandFieldTypeByReflection' -timeout 10s` | ✅ pkg/combat/dice_test.go | ✅ green |
| 1-02-01 | 02 | 2 | MIGRATE-06 | golden | `cd go && go build ./pkg/golden/... && go vet ./pkg/golden/...` | ✅ pkg/golden/fixture.go, doc.go | ✅ green |
| 1-02-02 | 02 | 2 | MIGRATE-06 | golden | `cd go && go test ./pkg/golden/ -run TestGolden -timeout 30s` | ✅ pkg/golden/golden_test.go | ✅ green |
| 1-03-01 | 03 | 3 | MIGRATE-06 | golden | `cd go && go test ./pkg/golden/ -run TestGolden -count=2 -timeout 60s` | ✅ pkg/golden/testdata/entities.golden | ✅ green |
| 1-04-01 | 04 | 4 | MIGRATE-06 | golden | `cd go && go test ./pkg/golden/ -run TestGolden -count=2 -timeout 60s` (MOB TEMPLATES section) | ✅ pkg/golden/testdata/entities.golden | ✅ green |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [x] `go/pkg/golden/` directory — package created (doc.go, fixture.go, golden_test.go)
- [x] `go/pkg/combat/dice.go` — `SetRand()` hook added for deterministic RNG
- [x] `go/pkg/golden/golden_test.go` — `TestGolden` implemented and green

*Existing `go test ./...` infrastructure covers the rest.*

---

## Manual-Only Verifications

None. Determinism (byte-identical golden on two successive runs) is fully automated via
`go test ./pkg/golden/ -run TestGolden -count=2`, which the standard `go test ./...` CI path
executes. The `-update` regeneration discipline is enforced by commit-message visibility +
PR review (threat T-01-03-01), not a separate test, and does not count as a manual-only gap.

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 30s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** validated 2026-06-01 — all 6 tasks have automated verification, full suite green.

---

## Validation Audit 2026-06-01

| Metric | Count |
|--------|-------|
| Gaps found | 0 |
| Resolved | 0 |
| Escalated | 0 |

Audit reconciled the stale pre-execution draft against shipped artifacts. All 4 plans
(01-04) executed; sole requirement MIGRATE-06 is fully COVERED by automated tests:
`pkg/combat/dice_test.go` (RNG determinism, 6 tests) and `pkg/golden/golden_test.go`
(`TestGolden` parity gate, `-count=2` byte-identical). No auditor spawn required — zero gaps.
Cross-checked against 01-VERIFICATION.md (status passed, 5/5, SC #3 closed by 01-04).
