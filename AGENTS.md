# GroovyMud — Architectural Decisions

This document records the key architectural decisions made during the Go rewrite of GroovyMud (`gomud/`).
It is intended for future contributors and AI coding agents working in this repository.

---

## 1. Clean-room Go rewrite under `gomud/`

**Decision**: New code lives entirely under `gomud/`. The legacy Java/Groovy source tree (`src/`) is left in place but is not touched by the Go work.

**Rationale**: Keeps the two stacks completely isolated. The legacy code can be removed in a later PR once the Go rewrite reaches feature-parity.

---

## 2. Pure-Go SQLite (`modernc.org/sqlite`) — no CGo

**Decision**: Player persistence uses SQLite via `modernc.org/sqlite`, a pure-Go SQLite port that requires no C compiler or CGo.

**Rationale**: The original stack used XStream XML files for player serialisation, which couples persistence to Java object graphs. SQLite gives us relational queries, ACID guarantees and a schema that can be version-migrated. The pure-Go driver means the binary cross-compiles without a C toolchain and works in musl/Alpine containers without any extra configuration.

**Trade-offs**: `modernc.org/sqlite` is slightly larger than a CGo wrapper but eliminates all cgo complexity.

---

## 3. bcrypt + role column instead of JAAS

**Decision**: Authentication is `bcrypt.GenerateFromPassword` / `bcrypt.CompareHashAndPassword` from `golang.org/x/crypto`. Authorisation is a `role TEXT` column (`player`, `creator`, `god`) on the `players` table.

**Rationale**: Java's JAAS/SecurityManager required policy files and privileged execution blocks; it was famously removed from the JDK in Java 17. bcrypt is the industry standard for offline password hashing. A single column is sufficient for coarse-grained role checks at the Go layer.

---

## 4. TOML world files instead of Groovy BeanBuilder DSL

**Decision**: World data (rooms, exits, items, MOBs) is described in TOML files under `gomud/world/`. The loader lives in `internal/loader/`.

**Rationale**: The Groovy BeanBuilder DSL was embedded Spring XML configuration in disguise — powerful, but tightly coupled to the Java classpath. TOML is human-readable, version-control friendly, and has excellent Go library support (`github.com/BurntSushi/toml`). World files can be edited by a content creator without knowing Go.

---

## 5. Explicit constructor wiring in `main.go` — no DI framework

**Decision**: All dependencies (`World`, `Engine`, `Server`, `Registry`) are wired by hand in `cmd/gomud/main.go`.

**Rationale**: Spring XML DI adds a large runtime dependency and obscures the object graph. Go's explicit constructors (`engine.New(w)`) make the call graph obvious and traceable by IDEs and static analysers. There is no magic.

---

## 6. Vertical-slice delivery

**Decision**: The rewrite is delivered as ten vertical slices, each producing a working, demonstrable subset of the game.

| Slice | Feature |
|---|---|
| 1 | TCP server + splash banner |
| 2 | Registration + login (bcrypt + SQLite) |
| 3 | World loader + `look` command |
| 4 | Movement (`go <direction>`, direction aliases) |
| 5 | Items (`get`, `drop`, `inventory`, `put`) |
| 6 | Engine heartbeat + MOBs |
| 7 | `say`, `tell`, `who` |
| 8 | Autosave + state-preserving reconnect |
| 9 | Starlark scripted commands + behaviours |
| 10 | Full Minnovar demo, Dockerfile, full E2E harness |

**Rationale**: Avoids a big-bang rewrite. Each slice can be reviewed, tested and deployed independently.

---

## 7. Package structure

```
gomud/
  cmd/gomud/          — binary entry point; wires all dependencies
  internal/
    auth/             — bcrypt Register/Login; thin layer over store
    command/          — command.Registry + all built-in handlers
    engine/           — wires Server ↔ World ↔ Registry; routes events
    event/            — publish/subscribe event bus (Global/Container/Local scopes)
    loader/           — TOML → World hydration
    net/              — TCP server, per-session goroutines, telnet IAC stripping, login flow
    store/            — SQLite migrations + CRUD (players, player_inventory)
    world/            — Room/Item/Player/World structs; mutex-safe collections
  test/
    e2e/              — outside-in approval tests (full server round-trips)
    approvals/        — *.approved.txt approval baseline files
  world/              — TOML data files (minnovar.toml, …)
```

**Rule**: `internal/` packages may only depend downward. `engine` depends on `net`, `world`, `command`, `event`. `command` depends on `world` and `event`. `net` depends on `auth`. `auth` depends on `store`. Nobody imports `engine` — it is the root of the dependency graph.

---

## 8. TCP + telnet IAC stripping

**Decision**: The server is a raw TCP listener (`net.Listen("tcp", …)`). IAC telnet negotiation bytes (0xFF prefix sequences) are stripped in `session.go` before commands are parsed. The server sends `WILL ECHO` / `WONT ECHO` to suppress client-side echoing during password entry.

**Rationale**: Using a full telnet library adds complexity without benefit for a MUD. Most clients send IAC sequences on connect; stripping them is a one-function concern.

---

## 9. Scoped event bus

**Decision**: `internal/event` provides a simple pub/sub bus with three scopes: `GlobalScope`, `ContainerScope`, `LocalScope`. All subscribers receive all events and filter by scope themselves.

**Rationale**: A MUD needs room-level broadcast (departure/arrival messages), area-level broadcast (future: zone shouts), and global broadcast (future: `tell`, server announcements). Embedding scope in the event struct avoids multiple separate channels while keeping the API simple. Subscribers filter once.

---

## 10. Per-connection goroutines

**Decision**: Each accepted TCP connection runs in its own goroutine (`go srv.handleSession(sess)`). There is no goroutine pool or connection limit (for now).

**Rationale**: Go goroutines are cheap (~2 KB stack) and this model is idiomatic for Go network servers. A MUD is unlikely to need hundreds of thousands of concurrent connections. A connection pool or semaphore can be added in a later slice if needed.

---

## 11. `slog` for structured logging

**Decision**: All logging uses `log/slog` (Go 1.21+) via the default logger, falling back to `log.Printf` in a few legacy locations.

**Rationale**: `slog` provides structured, levelled logging without an external dependency. It is replaceable at runtime (e.g., switch to JSON for production containers) by setting the default handler.

---

## 12. Approval-test strategy for outside-in tests

**Decision**: Outside-in tests live in `gomud/test/e2e/`. Each scenario drives a real TCP connection against an in-memory server (in-memory SQLite, TOML world loaded from `gomud/world/`). The textual output is compared against committed `*.approved.txt` files in `gomud/test/approvals/`. A mismatch writes a `*.received.txt` alongside the approved file and fails the test.

**Rationale**: Approval tests act as living documentation of the expected player experience. The `.approved.txt` files are human-readable and can be reviewed alongside code changes in pull requests. The CI workflow posts a diff of any changed approvals as a PR comment so reviewers can see the UX impact of every change without connecting to a server.

---

## 13. CI workflow

**Decision**: `.github/workflows/gomud-ci.yml` runs on every push and pull-request event targeting the Go sources (`gomud/**`). Steps: `go vet`, `go test ./...`, approval diff detection, PR comment creation/update.

**Rationale**: Fast feedback loop. Approval diffs are posted to the PR (one persistent comment, updated in place) so only changed approvals are visible without trawling CI logs.
