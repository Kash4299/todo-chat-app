# Todo Chat App - JD Fit Update Roadmap

## Goal

Turn `todo-chat-app` from a CRUD/chat learning project into a portfolio project that matches both:

- Security / Identity Platform backend roles: OAuth2, OIDC, JWT, MFA, Redis, Kafka, MongoDB, OWASP, observability.
- High-traffic backend / game platform roles: real-time systems, player-facing services, reliability, scalability, CI/CD, cloud/container, monitoring.

Target positioning:

> Secure real-time collaboration platform with identity, event-driven backend, observability, and deployable infrastructure.

---

## Priority 0 - Deployment Readiness

### What To Add

- Update Dockerfile Go version to match `go.mod`.
- Add `.dockerignore`.
- Complete `docker-compose.yaml` with:
  - PostgreSQL
  - Redis
  - Kafka
  - Mailhog or another local SMTP mock
  - Prometheus
  - Grafana
- Add health endpoints:
  - `/healthz`: process is alive
  - `/readyz`: DB/Redis/Kafka readiness
- Add GitHub Actions:
  - run tests
  - build binary
  - build Docker image
- Improve Makefile:
  - `make test`
  - `make build`
  - `make docker-up`
  - `make docker-down`
  - `make migrate-up`

### Why It Matters

Both JDs expect practical Docker, CI/CD, deployment, and production-readiness awareness.

### CV Bullet

```text
Containerized a Go backend service with Docker Compose, PostgreSQL, Redis, Kafka, SMTP mock, health checks, database migrations, and GitHub Actions for automated testing and image builds.
```

---

## Priority 1 - Auth / Identity Module

### What To Add

- JWT access token and refresh token flow.
- Refresh token rotation.
- Token revocation.
- Logout current device.
- Logout all devices.
- Redis-backed session store.
- Device/session management:
  - list active sessions
  - revoke session
- MFA:
  - TOTP or email OTP
- Password reset flow.
- Login rate limiting.
- Audit log for:
  - login
  - logout
  - failed login
  - token refresh
  - MFA enabled/disabled
  - MFA failure

### Why It Matters

This directly matches Identity Platform requirements: JWT, token lifecycle, MFA, session management, Redis, and secure account flows.

### CV Bullet

```text
Built an identity module in Go with JWT access/refresh token lifecycle, Redis-backed session storage, token revocation, MFA, login rate limiting, and audit logging.
```

---

## Priority 2 - OAuth2 / OIDC Learning Prototype

### What To Add

Build a small internal IdP-style module:

- `/oauth/authorize`
- `/oauth/token`
- Authorization Code Flow with PKCE.
- JWKS endpoint.
- ID token generation.
- Client registration table.
- Scope model.
- Consent page or API-level consent record.

### Why It Matters

Silver Tiger's JD treats OAuth2, OpenID Connect, and JWT as mandatory. A focused prototype gives you concrete experience to discuss in interviews.

### CV Bullet

```text
Implemented an OAuth2/OIDC learning prototype supporting Authorization Code Flow with PKCE, JWKS, ID token generation, client registration, and scoped access control.
```

---

## Priority 3 - RBAC / Permission System

### What To Add

- Roles:
  - owner
  - admin
  - member
  - guest
- Permission matrix:
  - `task:create`
  - `task:update`
  - `task:delete`
  - `channel:read`
  - `channel:write`
  - `workspace:invite`
  - `workspace:manage_members`
- Keep authentication in middleware.
- Keep authorization decisions in service layer.
- Cache permissions in Redis.
- Invalidate permission cache when role/member changes.

### Why It Matters

This demonstrates clean backend architecture and real authorization design, not just JWT verification.

### CV Bullet

```text
Designed workspace-level RBAC with a permission matrix, service-layer authorization checks, Redis permission caching, and cache invalidation on role changes.
```

---

## Priority 4 - Kafka Event-Driven Pipeline

### What To Add

Create real Kafka topics and consumers:

```text
chat.messages
audit.events
notification.events
email.jobs
workspace.events
```

Implement:

- Producer when a chat message is sent.
- Consumer to persist messages.
- Consumer to send notification/email jobs.
- Retry with backoff.
- Dead-letter topic.
- Consumer groups.
- Idempotency key for event processing.

### Why It Matters

Both JDs value Kafka and event-driven systems. VNGGames especially cares about high-throughput, player-facing backend systems.

### CV Bullet

```text
Implemented Kafka-based asynchronous workflows with producer/consumer groups, retry handling, dead-letter topics, and idempotent event processing for chat, audit, and notification events.
```

---

## Priority 5 - Redis Production Patterns

### What To Add

- Cache user profile by ID.
- Cache workspace membership.
- Cache permission checks.
- Login rate limiting.
- Distributed lock for:
  - resend email
  - invitation resend
  - idempotent sensitive actions
- Idempotency key storage.
- Cache stampede protection with `singleflight` or Redis lock.
- Clear TTL strategy.

### Why It Matters

This turns Redis from a checkbox skill into real production experience.

### CV Bullet

```text
Applied Redis production patterns including cache-aside, permission caching, login rate limiting, distributed locks, idempotency keys, TTL strategy, and cache stampede protection.
```

---

## Priority 6 - MongoDB Audit Log Module

### What To Add

Use MongoDB for append-heavy audit/activity storage:

- Collection: `security_audit_logs`
- Collection: `user_activity_logs`
- Fields:
  - user_id
  - event_type
  - ip_address
  - user_agent
  - metadata
  - created_at
- Indexes:
  - `{ user_id: 1, created_at: -1 }`
  - `{ event_type: 1, created_at: -1 }`
  - TTL index for retention if needed
- Query API with pagination.

### Why It Matters

Silver Tiger requires MongoDB schema design, indexing, and performance tuning. This is a bounded way to add MongoDB without rewriting the whole app.

### CV Bullet

```text
Added MongoDB-backed audit log storage with compound indexes, pagination, and retention strategy for security and user activity events.
```

---

## Priority 7 - Observability

### What To Add

- Structured logging with `slog` or `zap`.
- Prometheus metrics:
  - request count
  - error count
  - latency histogram
  - active WebSocket connections
  - Kafka consumer lag
  - Redis operation errors
  - DB query latency
- Grafana dashboard.
- OpenTelemetry tracing:
  - HTTP request
  - service method
  - DB call
  - Redis call
  - Kafka publish/consume
- Alert examples:
  - p95 latency > 300ms
  - error rate > 1%
  - Kafka lag above threshold

### Why It Matters

This matches production backend expectations: reliability, debugging, monitoring, incident response, and system ownership.

### CV Bullet

```text
Implemented production observability with Prometheus metrics, Grafana dashboards, structured logs, and OpenTelemetry tracing across HTTP, database, Redis, and Kafka flows.
```

---

## Priority 8 - WebSocket Scalability

### What To Add

- Sharded room map.
- Per-connection bounded send queue.
- Slow-consumer detection and disconnect.
- Reconnect-resync with cursor/event ID.
- Presence tracking with Redis.
- Cross-node fan-out through Kafka.
- WebSocket load test with k6 or custom Go client.

### Why It Matters

This is the strongest bridge to VNGGames because it demonstrates real-time, player-facing, high-concurrency backend thinking.

### CV Bullet

```text
Designed and implemented scalable WebSocket room management with sharded connection maps, bounded outbound queues, slow-consumer isolation, presence tracking, and reconnect-resync by event cursor.
```

---

## Priority 9 - LiveOps / Game-Like Backend Feature

### What To Add

Add a LiveOps-style module:

- Admin announcement API.
- Broadcast announcement to workspace/user segment.
- Scheduled campaign.
- Feature flag/config endpoint.
- User segments:
  - active users
  - workspace members
  - role-based audience
- Rate-limited broadcast delivery.

### Why It Matters

This maps `todo-chat-app` to game backend requirements: live operations, traffic spikes, segmentation, and real-time user-facing communication.

### CV Bullet

```text
Built a LiveOps-style announcement and feature flag module supporting segmented real-time broadcasts, scheduled campaigns, and rate-limited delivery.
```

---

## Priority 10 - Security Hardening

### What To Add

- Request body size limit.
- CORS allowlist.
- WebSocket origin validation.
- Secure password hashing.
- CSRF strategy if cookies are used.
- Parameterized SQL only.
- No raw internal errors returned to clients.
- Security headers.
- Brute-force protection.
- Audit trail for sensitive actions.
- Documentation:
  - `docs/security-checklist.md`
  - `docs/threat-model.md`

### Why It Matters

This matches OWASP, secure coding, and identity/security platform expectations.

### CV Bullet

```text
Hardened backend security with request limits, CORS allowlist, WebSocket origin validation, brute-force protection, safe error handling, and documented OWASP-based threat modeling.
```

---

## Priority 11 - Load Testing / Performance Report

### What To Add

- `loadtests/` directory.
- k6 tests for:
  - login
  - task APIs
  - workspace APIs
  - chat send
  - WebSocket connect/message flow
- Report:
  - p50
  - p95
  - p99
  - throughput
  - error rate
  - bottleneck found
  - before/after optimization
- DB query plan notes with `EXPLAIN ANALYZE`.
- Redis hit ratio notes.

### Why It Matters

This proves performance thinking instead of only claiming scalability.

### CV Bullet

```text
Created k6 load tests and performance reports covering API latency, WebSocket connection behavior, p95/p99 latency, database query plans, Redis cache hit ratio, and optimization results.
```

---

## Priority 12 - DevOps / Infrastructure

### What To Add

- Kubernetes manifests or Helm chart:
  - Deployment
  - Service
  - Ingress
  - ConfigMap
  - Secret
  - readiness/liveness probes
  - resource requests/limits
  - HPA
- Terraform basics:
  - ECS/Fargate or EKS skeleton
  - RDS
  - ElastiCache
  - MSK or SQS optional
- GitHub Actions deploy workflow for staging.

### Why It Matters

This supports your plan to grow stronger in DevOps while still staying backend-focused.

### CV Bullet

```text
Prepared Kubernetes and Terraform deployment templates with health probes, resource limits, HPA, managed database/cache dependencies, and CI/CD-based staging deployment.
```

---

## Recommended Build Order

1. Deployment readiness: Docker, Compose, CI, health checks.
2. Auth/Identity: refresh token, session, MFA, rate limit.
3. Redis production patterns.
4. Kafka event pipeline with DLQ.
5. Observability: Prometheus, Grafana, OpenTelemetry.
6. MongoDB audit log.
7. WebSocket scalability.
8. Load testing and performance report.
9. OAuth2/OIDC prototype.
10. LiveOps module.
11. Kubernetes/Helm/Terraform.

---

## Highest ROI For CV

If time is limited, prioritize these five:

1. JWT/refresh token/session/MFA/rate limit.
2. Kafka event pipeline with retry and DLQ.
3. Prometheus/Grafana/OpenTelemetry.
4. MongoDB audit log with indexes.
5. Docker Compose + CI/CD + Kubernetes basic deploy.

After these are done, the project can be presented as:

```text
A production-style secure real-time backend platform built with Go, PostgreSQL, Redis, Kafka, MongoDB, Docker, Kubernetes, Prometheus, Grafana, and OpenTelemetry, focused on identity, event-driven architecture, observability, and high-concurrency WebSocket communication.
```
