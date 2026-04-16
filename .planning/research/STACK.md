# Technology Stack — ROT-MUD Data-Driven Trait System

**Project:** ROT-MUD trait system refactor
**Researched:** 2026-04-16
**Scope:** Lua VM, TOML loading, trait composition/query API
**Baseline:** Go 1.25.5, `github.com/pelletier/go-toml/v2 v2.2.4` already in `go.mod`

---

## Executive Recommendation

| Concern | Pick | Version | Confidence |
|---------|------|---------|------------|
| Scripting VM | `github.com/yuin/gopher-lua` | v1.1.2 (2025-04-01) | HIGH |
| Lua↔Go reflection helper | **Do not use** gopher-luar | — | HIGH |
| TOML parser | `github.com/pelletier/go-toml/v2` | v2.3.0 (keep existing) | HIGH |
| Trait capability flags | `uint64` bitmask w/ `iota` constants | stdlib | HIGH |
| Trait modifiers/values | `map[TraitKey]int` (per-entity cache) | stdlib | HIGH |
| Trait behavior hooks | `map[HookEvent][]*ScriptRef` | stdlib | HIGH |
| ECS framework | **Do not introduce one** | — | HIGH |
| Bitset helper | `github.com/bits-and-blooms/bitset` (only if flag count > 64) | v1.24.4 | MEDIUM |

**One-liner:** Embed `gopher-lua` behind a thin hand-written binding layer, keep `pelletier/go-toml/v2`, model traits as three orthogonal stores (bitmask for capabilities, map for modifiers, slice for hooks) on the character struct. No ECS framework, no `gopher-luar` reflection wrapper.

---

## Core Dependencies

### Lua VM — `github.com/yuin/gopher-lua`

| Property | Value |
|----------|-------|
| Version | `v1.1.2` (2025-04-01) |
| Lua dialect | 5.1 + `goto` (from 5.2) |
| Go deps | None (pure Go, no cgo) |
| License | MIT |
| Stars / adoption | ~6k, used by Grafana Loki, Caddy, Hugo, Open Policy Agent |
| Maintenance | Active (v1.1.0 Jan 2024 → v1.1.1 Dec 2024 → v1.1.2 Apr 2025) |

**Why this:**
- **Pure Go, no cgo.** Cross-compilation to any Go target (Windows servers, ARM, etc.) stays trivial. This matters because Go MUD servers are often dropped onto VPS hosts where a C toolchain is painful.
- **Lua 5.1 is the MUD/gaming standard.** Mudlet, Aardwolf, Wireshark, World of Warcraft, LÖVE, Garry's Mod, Redis, Nginx, Neovim — all embed Lua 5.1-flavored engines. Scripts written by MUD builders online, tutorials, and existing snippets will mostly be 5.1-compatible. Sources: [lua-users wiki on versions](http://lua-users.org/wiki/LuaVersionCompatibility), [Aardwolf Lua docs](https://www.aardwolf.com/lua.html).
- **Fastest serious Go Lua VM.** ~20% faster than Shopify's `go-lua` and roughly on par with Python3 on micro-benchmarks. About 6-7x slower than C-Lua — fine for short hook scripts (OnDeath, OnAttack), not a bottleneck.
- **Context-based cancellation.** `L.SetContext(ctx)` checks `ctx.Done()` each bytecode iteration. Good enough for per-hook timeouts guarding against infinite loops in admin-authored scripts. Caveat: a single expensive op (huge string concat, tight C-level loop) can bypass the check — mitigate by disabling `string.rep`/pattern-matching for hooks.
- **Selective library loading.** `L.OpenBase()`, `L.OpenMath()`, etc. can be called individually instead of `L.OpenLibs()` — needed to keep hooks sandboxed (no `os`, no `io`, no `debug`, no `package`).
- **`LUserData` pattern is well documented** and has 10+ years of community examples for wrapping Go structs as Lua-addressable objects.

**What you give up:**
- Lua 5.1 feature set (no integer type, no `//` operator, no `goto` in 5.1 proper — but gopher-lua backports `goto`). Fine for hooks.
- No bitwise operators in 5.1 (they're in 5.3). If a hook needs bitops, expose a Go helper function. Unlikely for OnDeath/OnAttack.
- Sandboxing is cooperative, not enforced. No memory limits, no CPU accounting. **Acceptable for admin-authored scripts**, explicitly unacceptable for untrusted input — matches your out-of-scope list ("admin-defined data files only").

Sources:
- [yuin/gopher-lua GitHub](https://github.com/yuin/gopher-lua)
- [Release notes v1.1.2](https://github.com/yuin/gopher-lua/releases)
- [Benchmarks wiki](https://github.com/yuin/gopher-lua/wiki/Benchmarks)
- [Issue #521 on SetContext timeouts](https://github.com/yuin/gopher-lua/issues/521)
- [pkg.go.dev](https://pkg.go.dev/github.com/yuin/gopher-lua)

---

### TOML Parser — `github.com/pelletier/go-toml/v2` (keep)

| Property | Value |
|----------|-------|
| Version | `v2.3.0` (Mar 2026); currently pinned at `v2.2.4` — upgrade safe |
| Go deps | None |
| License | MIT |
| Performance | 1.8x–5.3x faster than BurntSushi/toml for unmarshal |
| Maintenance | Active (55 releases, latest Mar 2026) |

**Why this:** Already in `go.mod` and used by `pkg/loader`. Fastest mainstream option (5.3x faster than BurntSushi for struct unmarshal). Behaves like `encoding/json` for embedded struct fields, supports strict mode (flag unknown keys — catches typos in race/class TOML). [BurntSushi/toml is effectively unsupported](https://github.com/pelletier/go-toml/discussions/471); migration away from it is the trend.

**Recommend:** Bump to `v2.3.0` during this milestone — same API, bug fixes, no breaking changes in 2.x line.

**Known constraint — polymorphic unmarshaling is NOT supported.** This affects trait design directly. go-toml v2 explicitly dropped the polymorphic plans that were floated for v2.0 ([discussion #488](https://github.com/pelletier/go-toml/discussions/488), [issue #331](https://github.com/pelletier/go-toml/issues/331)). You cannot unmarshal

```toml
[[traits]]
kind = "resistance"
element = "fire"
amount = 50

[[traits]]
kind = "hook"
event = "ondeath"
script = "vampire_mist.lua"
```

directly into a `[]Trait` interface via struct tags alone. Options:

1. **Homogeneous sections (recommended).** Use separate TOML arrays per trait kind — one schema per section, no polymorphism needed:
   ```toml
   [stats]
   str = 15
   con = 17

   [[resistances]]
   element = "fire"
   amount = 50

   [[capabilities]]
   name = "darkvision"

   [[hooks]]
   event = "ondeath"
   script = "vampire_mist"
   ```
   Pros: Clean Go structs, strict mode works, trait kinds self-documenting in data files. Cons: Adding a new trait category requires a new TOML section and a new Go struct (but zero code outside the loader).

2. **Two-pass unmarshal.** Unmarshal into `[]map[string]any` first, read `kind` field, dispatch to concrete struct per element. Common pattern for tagged unions in Go ([Daniel Schmidt's blog](https://danielmschmidt.de/posts/2024-07-22-discriminated-union-pattern-in-go/), [Nicolas Hery's JSON sum types post](https://nicolashery.com/decoding-json-sum-types-in-go/)). Works but adds ~30 lines of loader boilerplate per new trait type.

3. **`UnmarshalTOML` on a wrapper type.** Similar to option 2, but localized in a custom unmarshaller. Most flexible, most code.

**Decision:** Option 1 (homogeneous sections). Matches how `pkg/loader/schema.go` already structures world data (`Rooms []RoomData`, `Mobs []MobData`). Keeps PITFALLS small, roadmap simple.

Sources:
- [pelletier/go-toml GitHub](https://github.com/pelletier/go-toml)
- [pkg.go.dev/pelletier/go-toml/v2](https://pkg.go.dev/github.com/pelletier/go-toml/v2)
- [Discussion #488 on polymorphic unmarshal](https://github.com/pelletier/go-toml/discussions/488)

---

### Trait Storage Pattern — stdlib only

Three orthogonal stores on `Character`, each optimized for its access pattern:

```go
type Character struct {
    // existing fields...
    Capabilities TraitFlags            // uint64 bitmask — hot path checks
    Modifiers    map[ModifierKey]int   // resist/vuln/stat amounts
    Hooks        map[HookEvent][]Script // behavior scripts
}

type TraitFlags uint64

const (
    CapDarkvision TraitFlags = 1 << iota
    CapVulnerableSilver
    CapImmuneCharm
    CapImmuneSleep
    CapFlight
    CapSwim
    // ... up to 64 slots
)

func (c *Character) HasCap(f TraitFlags) bool { return c.Capabilities&f != 0 }
```

**Why three stores instead of one generic `[]Trait`:**

| Access pattern | Example | Store | Why |
|----------------|---------|-------|-----|
| "Does X have capability Y?" | `if ch.HasCap(CapVulnerableSilver)` | `uint64` bitmask | O(1), single AND, zero allocs. Called every weapon swing. |
| "How much fire resistance does X have?" | `dmg -= ch.Modifier(ModResistFire)` | `map[Key]int` | O(1) hash, sparse (most chars don't resist most elements). |
| "Run OnDeath hooks for X" | `for _, h := range ch.Hooks[HookOnDeath]` | `map[Event][]Script` | Iterated once per event, rare compared to capability checks. |

Sources confirm this layering: [YourBasic Go on bitmasks](https://yourbasic.org/golang/bitmask-flag-set-clear/), [Go Wiki: Iota](https://go.dev/wiki/Iota), [Devtrovert on flag enums](https://blog.devtrovert.com/p/go-what-is-flags-enum-and-how-to). For the "at what point do maps beat slices" question — [Graham King's benchmarks](https://darkcoding.net/software/go-slice-search-vs-map-lookup/) and [fujiwaram/slice-vs-map](https://github.com/fujiwaram/slice-vs-map) put the crossover at ~5-10 elements. You'll have 30+ modifier keys, so map wins.

**If > 64 capability flags:** switch `TraitFlags` to `github.com/bits-and-blooms/bitset` (v1.24.4, used by beego / CubeFS / Amazon EKS). Keep the API identical. Don't pre-adopt — 64 capability slots is more than you currently need (survey your combat code: ~15-20 flag-style checks today).

---

## Libraries NOT to Use (and Why)

### `layeh.com/gopher-luar` — reject

Reflection-based auto-binding for gopher-lua. Tempting because it exposes Go structs to Lua "for free."

**Reasons to avoid:**
1. **Unmaintained.** Latest release `v1.0.11` is from **June 2023**. pkg.go.dev displays the warning *"This package is not in the latest version of its module."* The repo has 284 stars and very limited recent activity.
2. **Reflection surface is the whole binding.** Every field on `*Character` becomes reachable from Lua — including fields that should not be (password hash, socket handle, internal timers). Hard to narrow the attack surface.
3. **Hand-written bindings are cheap.** You only need to expose a handful of character/combat fields and methods to hooks. 50-100 lines of `LUserData` wrapping gives you a curated, reviewable API.
4. **Debuggability.** Stack traces from hand-written bindings point at your code. Luar stack traces point into reflection internals.

### `Shopify/go-lua` — reject

Lua 5.2, about 20% slower than gopher-lua, lacks coroutines and `string.dump`. Used in production at Shopify (Genghis load generator) since 2014 but sees less community activity than gopher-lua. No compelling reason over gopher-lua for this use case.

Source: [Shopify engineering announcement](https://shopify.engineering/announcing-go-lua)

### `d5/tengo` — reject

Fast (~2x gopher-lua for fib(35)), pure Go, good sandboxing. But it's a custom Go-adjacent language, not Lua. MUD builders will not know it. Zero ecosystem of existing scripts. Kills the "standard MUD scripting language" goal in your PROJECT.md.

### `mattn/anko` — reject

~25x slower than tengo per benchmarks. Also a custom language. No advantage over Lua.

### `google/starlark-go` — reject

Python-dialect, deterministic, hermetic. Excellent sandboxing (no FS/net/time access). But: not Lua, slower than tengo, awkward syntax for imperative scripts like combat hooks. Great for configs, wrong for this.

### `dop251/goja` (JavaScript) — reject

High-quality, maintained ECMAScript 5.1 engine. About 6-7x faster than `otto`. But:
1. Not the MUD scripting standard — you'd be swimming upstream against the entire ecosystem.
2. **No memory limits on scripts** (per the goja README caveat) — same sandboxing weakness as gopher-lua, no advantage.
3. Larger binary footprint than gopher-lua.
4. QuickJS-via-Wazero (`QJS`) is a 2025 contender with real sandbox limits, but it's new, cgo-adjacent via WASM, and overkill for admin-authored hooks.

Source: [goja GitHub](https://github.com/dop251/goja), [InfoQ on QJS/WASM](https://www.infoq.com/news/2025/12/javascript-golang-wasm/)

### Any Go ECS library (`donburi`, `mlange-42/ark`) — reject

Both are competent archetype-based ECS frameworks aimed at Ebitengine games. Not appropriate here:
1. **Architectural mismatch.** Your characters aren't entities-with-components — they're structs with a trait overlay. Introducing an ECS forces a rewrite of combat, magic, and socket code far beyond the trait-system scope.
2. **PROJECT.md explicitly bounds scope to races/classes.** ECS is a whole-engine decision.
3. **Zero payoff.** You'd get archetype iteration (query all entities with component X), but your lookups are always "does *this one* character have trait Y" — ECS solves the opposite problem.

Go ECS libraries are real and maintained ([Ark v0.5.0 release note](https://forum.golangbridge.org/t/ark-v0-5-0-released-a-minimal-high-performance-entity-component-system-ecs-for-go/40877), [donburi on pkg.go.dev](https://pkg.go.dev/github.com/yohamta/donburi)). Just not for this project.

---

## Installation / `go.mod` Diff

```bash
# Add
go get github.com/yuin/gopher-lua@v1.1.2

# Upgrade (safe, same major)
go get github.com/pelletier/go-toml/v2@latest

# Optional, only if > 64 capability flags
# go get github.com/bits-and-blooms/bitset@v1.24.4
```

No cgo. No new transitive deps of note (`gopher-lua` has zero Go-module deps, just stdlib).

---

## Architectural Constraints This Stack Imposes

1. **Hooks run synchronously inside the game tick.** gopher-lua is single-threaded per `LState`. Share bytecode across multiple `LStates` (one per hook invocation, or a small pool) if you want parallel hook execution — otherwise serialize through one state. For a MUD at MUD scale (dozens to hundreds of players), one state with sequential dispatch is fine and much simpler.

2. **Hook scripts must be pre-compiled once at startup.** `L.LoadFile` → `*FunctionProto`, cache in the loader, `L.Push(L.NewFunctionFromProto(p))` per invocation. Don't parse Lua source on every OnAttack.

3. **Timeout all hook invocations.** Always run hooks under a `context.WithTimeout`, e.g. 50ms. A runaway script must not stall the pulse loop. See the caveat above — string/pattern ops can still stall; disable `string.rep`, `string.gmatch`, and any lib you don't need.

4. **TOML files are read-only at startup.** Matches PROJECT.md out-of-scope ("hot reloading is out of scope"). `pelletier/go-toml/v2` fits this trivially — no runtime file watching, no encoder needed.

5. **Capability flag budget is 64 (one `uint64`).** More than enough for race+class traits in ROM lineage. If you blow that budget, switch storage without changing the `HasCap` API.

---

## Confidence Assessment

| Decision | Confidence | Basis |
|----------|------------|-------|
| gopher-lua v1.1.2 | HIGH | Current release verified on GitHub releases page; API stable since v1.0.0 |
| Lua 5.1 dialect choice | HIGH | Every major MUD client/server uses 5.1 family; ecosystem alignment is the whole point |
| Reject gopher-luar | HIGH | Verified last release 2023-06-05 on pkg.go.dev with explicit "not latest" warning |
| Keep pelletier/go-toml v2 | HIGH | Already in use, fastest benchmarked option, actively maintained (v2.3.0 Mar 2026) |
| Homogeneous TOML sections over polymorphism | HIGH | Polymorphic unmarshal explicitly unsupported and won't be added, confirmed in maintainer discussion |
| Bitmask + map + slice trait layout | HIGH | Direct consequence of Go hash-map vs bitmask benchmarks; no controversy |
| Reject ECS | HIGH | Scope mismatch, PROJECT.md explicitly bounds to races/classes |
| Reject goja / tengo / starlark / anko | HIGH | Each fails on at least one hard constraint (standard MUD scripting language, performance, sandbox story) |
| 64-flag capability budget | MEDIUM | Based on quick scan of combat.go / magic.go idioms; formal audit would tighten this. If exceeded, swap to `bits-and-blooms/bitset` — same API shape. |
| `v2.3.0` upgrade path is safe | MEDIUM | Reviewed release notes topically, not line-by-line. Run existing loader tests after bump. |

---

## Sources

- [yuin/gopher-lua](https://github.com/yuin/gopher-lua) — primary VM
- [gopher-lua releases](https://github.com/yuin/gopher-lua/releases) — v1.1.2 confirmed Apr 1 2025
- [gopher-lua benchmarks wiki](https://github.com/yuin/gopher-lua/wiki/Benchmarks)
- [gopher-lua issue #521 on context timeouts](https://github.com/yuin/gopher-lua/issues/521)
- [gopher-lua issue #89 on state termination](https://github.com/yuin/gopher-lua/issues/89)
- [pkg.go.dev gopher-lua](https://pkg.go.dev/github.com/yuin/gopher-lua)
- [pelletier/go-toml](https://github.com/pelletier/go-toml)
- [pelletier/go-toml v2 README](https://github.com/pelletier/go-toml/blob/v2/README.md)
- [pelletier/go-toml discussion #488 (unmarshal polymorphism)](https://github.com/pelletier/go-toml/discussions/488)
- [pelletier/go-toml discussion #471 (BurntSushi status)](https://github.com/pelletier/go-toml/discussions/471)
- [layeh/gopher-luar](https://github.com/layeh/gopher-luar) — reviewed and rejected
- [pkg.go.dev gopher-luar](https://pkg.go.dev/layeh.com/gopher-luar) — confirms unmaintained warning
- [Shopify engineering: announcing go-lua](https://shopify.engineering/announcing-go-lua)
- [dop251/goja](https://github.com/dop251/goja)
- [d5/tengo](https://github.com/d5/tengo)
- [google/starlark-go](https://starlark-lang.org/)
- [mlange-42/ark ECS](https://github.com/mlange-42/ark)
- [yohamta/donburi ECS](https://pkg.go.dev/github.com/yohamta/donburi)
- [bits-and-blooms/bitset](https://github.com/bits-and-blooms/bitset)
- [YourBasic: bitmasks, bitsets and flags in Go](https://yourbasic.org/golang/bitmask-flag-set-clear/)
- [Go Wiki: Iota](https://go.dev/wiki/Iota)
- [Daniel Schmidt: discriminated unions in Go](https://danielmschmidt.de/posts/2024-07-22-discriminated-union-pattern-in-go/)
- [Nicolas Hery: decoding JSON sum types in Go](https://nicolashery.com/decoding-json-sum-types-in-go/)
- [Graham King: slice search vs map lookup](https://darkcoding.net/software/go-slice-search-vs-map-lookup/)
- [lua-users wiki: Lua Version Compatibility](http://lua-users.org/wiki/LuaVersionCompatibility)
- [Aardwolf MUD Lua scripting](https://www.aardwolf.com/lua.html)
- [Embedded scripting languages survey (dbohdan)](http://dbohdan.github.io/embedded-scripting-languages/)
