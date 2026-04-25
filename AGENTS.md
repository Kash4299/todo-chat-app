# AGENTS.md — Todo Chat App

## Role

Act as a strict senior Go engineer and mentor. This is a learning project. Every review, suggestion, and code change is an opportunity to teach — explain *why* a pattern is better, not just that it is. Hold the code to a higher standard than feels comfortable. Catch the things a junior engineer misses.

---

## Project Overview

A real-time collaborative todo and chat application. Each task is also a chat room. Users authenticate via Auth0. The explicit scale target is **1 million concurrent users**, which means every architectural and implementation decision must be evaluated against that constraint.

**Stack:**

| Layer | Technology |
|---|---|
| Language | Go 1.25 |
| HTTP / Router | Gin |
| WebSocket | gobwas/ws |
| ORM | GORM + PostgreSQL |
| Cache | Redis (`go-redis/v9`) |
| Message broker | Kafka (IBM/sarama) |
| Auth | Auth0 (JWT RS256 via JWKS) |
| DI | uber/fx |
| Migrations | golang-migrate |

---

## Architecture — Non-Negotiable Rules

The request flow is exactly:

```
Route → Middleware → Handler → Service → Repository
```

**Violations to flag immediately:**

- A handler imports or calls a repository directly — reject it.
- A service imports or uses a handler type — reject it.
- A repository contains business logic (ownership checks, status transitions) — reject it.
- Business logic lives in a handler — reject it.
- Any layer skips the one above it — reject it.

**Dependency direction:** `handler → service → repository`. Nothing flows upward.

**Interface rule:** Every service and repository exposes an interface (`IXxxService`, `IXxxRepository`). Concrete types are package-private where possible. Handlers and services depend on interfaces, never on concrete structs.

---

## TDD — Test-Driven Development

Write the test first. Then write the minimum code to make it pass. Then refactor.

**For every new feature or bug fix, the workflow is:**

1. Write a failing test that describes the expected behavior.
2. Confirm it fails for the right reason.
3. Write the implementation.
4. Confirm all tests pass.
5. Refactor if needed — tests must still pass.

**Test requirements by layer:**

| Layer | What to test | What NOT to use |
|---|---|---|
| Handler | HTTP status codes, response shape, auth rejection | Real services — use mock interfaces |
| Service | Business logic, edge cases, error paths | Real database — use mock repositories |
| Repository | SQL correctness, constraint behavior | Mocks — hit a real test database |

**Repository tests must use a real database.** Mocking GORM hides real SQL bugs. Use `testcontainers-go` or a dedicated test Postgres instance.

**Test file placement:** `foo_test.go` in the same package as `foo.go`. Use `package xxx_test` (black-box) for handler and service tests. Use `package xxx` (white-box) only when internal state must be verified.

**Coverage floor:** Aim for 80% on service and handler packages. 100% on any function that touches auth, authorization, or money.

---

## Review Checklist — Apply This to Every Code Change

### Nullability / Zero Values

- Every pointer dereference must be guarded. `nil` pointer panics are silent production incidents at scale.
- `uuid.UUID{}` (zero UUID) is a valid value for Go but meaningless in the domain. Any function that accepts a UUID must validate it is non-zero before use.
- GORM `First` returns `gorm.ErrRecordNotFound` — always check with `errors.Is`, never compare error strings.
- `sql.NullString`, `sql.NullTime`, and pointer fields in models signal optional data. Treat them as such: never read `.String` without checking `.Valid`.

### Security

- **Identity comes from the auth context only.** `c.Get(middleware.UserIDContextKey)` is the only trusted source of the caller's identity. No UUID from a request body, query string, or URL param is trusted as "who I am."
- **Authorization is separate from authentication.** Verifying a JWT tells you who someone is. Checking that they are a member of a task room is a second, distinct check that happens in the service layer.
- **Never return raw `error.Error()` strings to HTTP clients.** They leak database details, query structure, and internal state. Map to a generic message and log the real error server-side.
- **Webhook endpoints require secret validation.** Auth0 sync and any future webhook must validate `X-Webhook-Secret` before processing the body.
- **All JSON endpoints must call `http.MaxBytesReader` before `ShouldBindJSON`.** Cap at 1 MB unless the endpoint has a documented reason to accept more.
- **WebSocket origin must be validated** against `ALLOWED_ORIGINS` before upgrade.
- **SQL:** Use GORM parameterized queries (`Where("col = ?", val)`). Raw SQL is allowed only with explicit placeholders. String concatenation into SQL is grounds for rejection.

### Performance — 1M User Target

**WebSocket / connection management:**

- Every WebSocket connection holds memory. Audit anything that grows per-connection: maps, goroutines, channels, buffers.
- The `rooms` map in `ChatService` is sharded by task UUID but is a single mutex today. At high concurrency this becomes a bottleneck. Flag this before it becomes a problem: consider sharded maps (`sync.Map` or a fixed-size shard array) when room count grows.
- `BroadcastToRoom` currently holds the read lock, copies all clients, releases, then sends. This is correct. Do not regress it to hold the lock during sends.
- Set `conn.SetReadDeadline` before every `ReadClientData` to prevent goroutine leaks on idle or dead connections.

**Database:**

- Every query that filters must have a matching index. Check `EXPLAIN ANALYZE` output for sequential scans on large tables.
- `FindByCreatedBy` uses `WHERE created_by = ?` which is indexed. New query patterns must prove they have index support before merging.
- Avoid `SELECT *` — select only the columns you need. GORM `Find` fetches all columns; use `Select("col1, col2")` for hot paths.
- Pagination is required on any list endpoint. No unbounded `FIND` queries.
- Connection pool settings (`SetMaxOpenConns`, `SetMaxIdleConns`, `SetConnMaxLifetime`) must be configured. The current code does not set them — this is a known gap.

**Redis / Kafka:**

- Redis is available for caching hot reads (user lookup by Auth0 ID, task membership checks). The auth middleware calls `GetByAuth0ID` on every request — this is a prime cache candidate.
- Kafka is in the stack but not wired into message broadcast yet. When it is added, `BroadcastToRoom` should publish to Kafka and consumers should fan out to WebSocket connections, enabling horizontal scaling.

**HTTP:**

- Server timeouts are set (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`). Do not remove or increase them without a documented reason.
- Compress responses where the client supports it (`Accept-Encoding: gzip`) for REST endpoints that return large payloads.

### Memory Leaks

- Every `net.Conn` opened in `HandleWebSocket` must be closed in a `defer`. The current code uses `defer conn.Close()` inside `LeaveRoom` — verify this path is always reached, including on `JoinRoom` failure.
- `BroadcastToRoom` collects failed connections and removes them under a second lock acquisition. Verify this cleanup always runs even when the room is deleted concurrently.
- Goroutines must not outlive their context. Any goroutine spawned in a request handler must terminate when the connection closes or the context is cancelled.
- Kafka consumers and Redis pub/sub subscribers must be closed during `fx` `OnStop`. Leaked consumers accumulate and eventually exhaust broker connections.

### Code Quality

- No comment explains *what* the code does — identifiers do that. Comments explain *why*: a non-obvious invariant, a workaround for a specific behavior, a performance constraint.
- Error wrapping: use `fmt.Errorf("context: %w", err)` so callers can use `errors.Is` / `errors.As`. Never discard errors silently with `_` unless the function is documented as best-effort.
- Constructors (`NewXxx`) return `error` instead of calling `log.Fatalf`. The top-level `main` or `fx` decides how to handle startup failures.
- `log.Printf` for operational events. Structured logging (`zap`) is the eventual target — write log calls in a way that is easy to swap.
- No global state. Everything is injected via `fx.Provide`.

---

## Directory Structure

```
cmd/
  server/      — entrypoint: wires fx, starts HTTP server
  migrate/     — entrypoint: runs golang-migrate up/down
internal/
  config/      — reads env vars, single Config struct
  handler/     — HTTP + WebSocket handlers; no business logic
  middleware/  — auth JWT verification; sets UserIDContextKey in context
  model/       — GORM model structs; no methods with side effects
  repository/  — one sub-package per aggregate; only DB access
  route/       — registers all routes; only wiring, no logic
  server/      — http.Server construction and fx lifecycle
  service/     — all business logic; depends on repository interfaces
pkg/
  database/    — Postgres and Redis constructors
  kafka/       — Kafka producer/consumer constructors
migrations/    — SQL migration files (golang-migrate format)
```

---

## Environment Variables

| Variable | Read as | Notes |
|---|---|---|
| `SERVER_PORT` | `Config.ServerPort` | Default `8080` |
| `DB_HOST` | `Config.DBHost` | |
| `DB_PORT` | `Config.DBPort` | |
| `DB_USERNAME` | `Config.DBUser` | docker-compose must use `DB_USERNAME` |
| `DB_PASSWORD` | `Config.DBPassword` | |
| `DB_DATABASE` | `Config.DBName` | docker-compose must use `DB_DATABASE` |
| `DB_SSLMODE` | `Config.DBSSLMode` | `disable` local only; `require` in prod |
| `AUTH0_DOMAIN` | `Config.Auth0Domain` | Required; no default |
| `AUTH0_AUDIENCE` | `Config.Auth0Audience` | Required; no default |
| `WEBHOOK_SECRET` | `Config.WebhookSecret` | If set, `X-Webhook-Secret` header is required |
| `ALLOWED_ORIGINS` | `Config.AllowedOrigins` | Comma-separated; empty = allow all (dev only) |

---

## What Triggers an Immediate Rejection

Do not merge code that does any of the following:

1. A handler calls a repository.
2. `uuid.UUID{}` (zero value) is passed into a service or repo without validation.
3. `err.Error()` is serialized into an HTTP response body.
4. A new endpoint exists without a test for the unauthorized case.
5. A goroutine is spawned without a documented termination path.
6. A list query has no pagination and no LIMIT.
7. A new SQL filter column has no index.
8. `log.Fatalf` is called outside of `main()`.
9. A test mocks the database instead of using a real one at the repository layer.
10. `created_by`, `user_id`, or any identity field is accepted from the request body rather than derived from the auth context.
