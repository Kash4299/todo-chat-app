# Backend Review Schedule
> Created: 2026-05-23 | Updated: 2026-06-20 (v2 — Milestone 1 deploy-first) | Owner: Kash | Reviewer: Codex/Claude
> Goal: chia BE thành task nhỏ, có thứ tự, test trước, có review gate rõ ràng.
> **v2 change**: bám theo Milestone Map trong `task-breakdown.md`. Mục tiêu gần nhất = **deploy Milestone 1** (login + workspace, đủ chiều sâu CV). Auth0 bị bỏ, identity tự host (RS256+JWKS).

---

## Cách làm mỗi task

Mỗi task BE đi theo cùng một vòng:

1. Viết test fail trước.
2. Chạy đúng package test, xác nhận fail vì behavior chưa có.
3. Implement tối thiểu để pass.
4. Chạy test package liên quan, rồi `go test ./...`.
5. Gửi diff để review.

Khi gửi review, ghi ngắn:

```text
Task: T<NN>
What changed:
- ...
Tests:
- go test ./internal/service/...
- go test ./...
Notes/Risks:
- ...
```

### Review reject ngay nếu (mở rộng cho identity platform)

Các gate cũ (vẫn áp dụng):
- Handler gọi repository trực tiếp.
- Identity lấy từ body/query/param thay vì auth context.
- Serialize `err.Error()` ra response.
- UUID zero đi vào service/repository mà không validate.
- List query không pagination/limit.
- Repository test mock database.
- Goroutine/connection không có đường cleanup.

Gate mới (identity / security — Milestone 1):
- **Secret bị log** (token, password, RS256 private key, JWT secret) → reject.
- **RS256 private key** không đọc từ secret store / bị commit → reject.
- **Refresh token** lưu plaintext (không hash) hoặc không có reuse detection → reject.
- **MFA challenge** nằm SAU token-issue thay vì giữa password-ok và token-issue → reject.
- **Sensitive action** (login, password reset, MFA toggle, session revoke) không ghi audit → reject.
- **Login** không rate limit → reject.
- Cache permission/membership không invalidate khi role/member đổi → reject (stale authz).

---

## Review Cadence

| Nhịp | Việc cần làm |
|---|---|
| Mỗi task nhỏ | Làm xong, chạy test, gửi diff review ngay. |
| Cuối mỗi phase | `go test ./...`, update status file này. |
| Trước task có RS256/MFA/session | Review design trước khi code — đây là phần dễ sai bảo mật nhất. |
| Trước task WebSocket/Kafka (M2) | Review design trước khi code — phần dễ phải rewrite nhất. |
| Sau mỗi feature endpoint | Có handler unauthorized test, service business-rule test, repo DB test nếu có SQL mới. |

> Lưu ý: ID task giờ thống nhất theo `T##` trong `task-breakdown.md`. BE-## của v1 được gộp vào các phase dưới.

---

# MILESTONE 1 — Deployable Secure Identity + Workspace

> Exit của M1 = deploy được một slice login + workspace, có observability, đủ chiều sâu CV. KHÔNG có chat/Kafka.

## Phase M1.0 — Audit & baseline

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| BE-00 | Baseline `go test ./...` + `-race`, ghi pass/fail | Không sửa code trước khi biết baseline | 0.5 |
| T101 | Audit body-size limit + error mapping toàn handler | Không endpoint nhận body unbounded; không raw error ra client | 1 |

Exit: có baseline + issue list debt tách khỏi feature.

## Phase M1.1 — Deployment env first (deploy-as-you-build)

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T104 | Fix Dockerfile (Go khớp go.mod) + `.dockerignore` | multi-stage, image nhỏ | 1 |
| T105 | Full docker-compose (pg, redis, mongo, mailhog, prometheus, grafana) | health checks + `depends_on` | 2 |
| T106 | `/healthz` + `/readyz` (DB/Redis/Mongo) | readyz fail khi dependency down | 1 |
| T107 | GitHub Actions CI (lint → test -race → build → image) | CI xanh trên PR | 1 |
| T108 | Makefile (test/build/docker-up/down/migrate-up) | reproducible local | 1 |

Exit: `make docker-up` dựng full stack; CI chạy mỗi PR.

## Phase M1.2 — Identity Platform core ⭐ (CV centerpiece)

> Phần rủi ro bảo mật cao nhất. Review design trước mỗi task RS256/MFA/session.

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T71 | RSA keypair + `/.well-known/jwks.json` + rotation (`kid`, 2-key overlap) | private key từ secret store, không log | 2 |
| T72 | Access/ID token HS256 → RS256; verifier dùng JWKS cache + singleflight | không còn HS256 cho app auth | 2 |
| T80 | Gỡ Auth0 path; identity = single source of truth | không còn import/route Auth0; update AGENTS.md | 1 |
| T73 | Refresh rotation + reuse detection (token family) | refresh dùng lại → revoke family + audit | 2 |
| T74 | Session/device mgmt (list/revoke one/revoke all) | revoke session → refresh family chết | 2 |
| T77 | Password reset (token hashed, one-time, TTL ngắn) | không tiết lộ email tồn tại | 1 |
| T75 | MFA TOTP enroll/verify + recovery codes | challenge giữa password-ok và token-issue | 3 |
| T76 | Email OTP fallback | dùng Mailhog test | 1 |
| T78 | Login rate limit + lockout (Redis sliding window) | fail-closed | 1 |

Exit: `go test -race ./internal/service/... ./internal/handler/...` xanh; mọi flow nhạy cảm có test happy + abuse path.

## Phase M1.3 — Redis production patterns + MongoDB audit

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T92 | Mongo integration + `AuditWriter` interface (direct write M1) | OnStop disconnect; interface để M2 đổi sang Kafka | 1 |
| T93 | `security_audit_logs`+`user_activity_logs` + compound indexes | index-backed query, append-only | 1 |
| T79 | Wire identity audit events vào `AuditWriter` | login/logout/refresh/mfa/reset đều ghi | 1 |
| T94 | Audit query API (cursor pagination, filters) | admin-only, không seq scan | 1 |
| T95 | Cache-aside user profile + membership | TTL rõ ràng | 1 |
| T96 | Permission/RBAC cache + invalidate on role/member change | stale permission = reject | 1 |
| T97 | Distributed lock (resend email, invite resend) | TTL ngắn, không deadlock | 1 |
| T98 | Idempotency key storage cho sensitive POST | `idem:{key}` | 1 |
| T99 | Cache stampede protection (singleflight + lock) | key nóng không thundering herd | 1 |

Exit: audit ghi đủ; Redis patterns có test.

## Phase M1.4 — Observability

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T87 | Structured logging (slog/zap) + request_id/trace_id | không log secret | 1 |
| T88 | Prometheus metrics + `/metrics` | req/err/latency histogram | 2 |
| T89 | Grafana dashboard | provision qua compose | 1 |
| T90 | OpenTelemetry tracing (HTTP→service→DB→Redis) | trace_id propagate qua context | 2 |
| T91 | Alert rules (p95>300ms, error>1%) | alert không noise | 1 |

Exit: dashboard hiển thị latency/error/conn; trace 1 request đi xuyên layer.

## Phase M1.5 — Workspace/RBAC hardening + security baseline

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T05-T08 harden | Zero-UUID validation, pagination contract, ownership authz | authz ở service, không repo | 3 |
| T07 mở rộng | Role OWNER + permission matrix + cache (T96) | matrix không hardcode rải rác | 1 |
| T100 | Security headers + CORS allowlist + (WS origin để M2) | không `*` ở prod | 1 |
| T102 | `docs/security-checklist.md` | OWASP-based | 1 |
| T103 | `docs/threat-model.md` | tập trung identity attack surface | 1 |

Exit: workspace/invitation/profile đủ test happy + forbidden; identity không lấy từ body.

## Phase M1.6 — Deploy M1 (LIVE)

| Task | Deliverable | Review Gate | Est |
|---|---|---|---|
| T109 | K8s manifests / Helm (deploy/svc/ingress/configmap/secret/probes/HPA/limits) | probes → healthz/readyz; secret không plaintext | 3 |
| T110 | Terraform skeleton (VPC/RDS/ElastiCache/Mongo/ECS hoặc EKS) | reproducible, smallest instance | 2 |
| T111 | GitHub Actions CD → staging deploy | rollback path | 1 |
| T44 (smoke) | k6 smoke cho login + workspace API | threshold latency/error rate | 1 |

**Exit M1 = deployed + CV-ready**:
> "Built and deployed a self-hosted identity platform in Go (RS256/JWKS, refresh rotation w/ reuse detection, TOTP MFA, session/device management, Redis-backed login rate limiting, MongoDB audit logging) with workspace RBAC, Prometheus/Grafana/OpenTelemetry observability, Dockerized with CI/CD and Kubernetes deployment."

---

# MILESTONE 2 — Real-time core (sau khi M1 live)

> Đây là lúc rủi ro #2 (in-memory broadcast) được giải quyết. Review design trước khi code.

| Phase | Tasks | Mục tiêu |
|---|---|---|
| M2.1 WS design spike | T09 (đã có), thiết kế sharded room/connection, backpressure, slow-consumer policy | Không single global mutex, không hold lock khi write socket |
| M2.2 WS core | T10, T23 lifecycle/heartbeat/reconnect-resync | `SetReadDeadline` trước mỗi read; cleanup rõ |
| M2.3 Kafka fan-out | T11, T12 (AsyncProducer), T13 (ConsumerGroup + idempotency + DLQ) | Cross-node fan-out qua Kafka; OnStop đóng producer/consumer; audit chuyển sang `audit.events` |
| M2.4 Chat persistence | T14, T15, T16, T17 (cursor), T18, T19 (presence Redis), T20 | Membership check ở service; cursor pagination |
| M2.5 Notifications | T53-T57 | Fan-out rate-limited; email qua `email.jobs` |
| M2.6 Load gate | T44, T45 (perf report), T46 | Số liệu baseline; bottleneck có evidence |

---

# MILESTONE 3 — Platform breadth

| Phase | Tasks | Mục tiêu |
|---|---|---|
| M3.1 OAuth2/OIDC provider | T81-T86 | authorize/token/PKCE/JWKS/userinfo/consent; Google direct |
| M3.2 Task management | T24-T32 | CRUD/assignment/workflow/kanban, không lấy identity từ body |
| M3.3 Search | T33-T37 | scoped theo tenant, GIN index |
| M3.4 LiveOps | T112-T116 | announcement/feature flag/segment/scheduled |
| M3.5 AWS prod + integrations | T49-T52, T59, T60 | managed services; webhook SSRF guard |
| M3.6 E2EE (last) | T61-T70 | sau frontend; security audit bắt buộc |

---

## Suggested Weekly Schedule — Milestone 1 (~4-6 tiếng/ngày)

| Week | Focus | Tasks |
|---|---|---|
| W1 | Audit + deploy env | BE-00, T101, T104-T106, T108 |
| W2 | CI + RS256/JWKS + remove Auth0 | T107, T71, T72, T80 |
| W3 | Refresh rotation + sessions + reset | T73, T74, T77 |
| W4 | MFA + login rate limit | T75, T76, T78 |
| W5 | Mongo audit + audit events | T92, T93, T79, T94 |
| W6 | Redis production patterns | T95-T99 |
| W7 | Observability | T87-T91 |
| W8 | Workspace/RBAC harden + security docs | T05-T08 harden, T96, T100, T102, T103 |
| W9 | K8s/Helm/Terraform + deploy | T109, T110, T111, T44 smoke → **M1 LIVE** |

---

## Current Status

| Milestone / Phase | Status | Notes |
|---|---|---|
| M1.0 Audit | Not started | Start here: BE-00. |
| M1.1 Deploy env | Not started | Docker/compose/health/CI/Makefile. |
| M1.2 Identity core | Not started | ⭐ CV centerpiece; design review trước RS256/MFA/session. |
| M1.3 Redis + Mongo | Not started | Audit interface để M2 đổi sang Kafka. |
| M1.4 Observability | Not started | Prometheus/Grafana/OTel. |
| M1.5 Workspace/RBAC harden | Partial | Code workspace/member đã có, cần hardening + cache. |
| M1.6 Deploy M1 | Not started | Exit = LIVE + CV-ready. |
| M2 Real-time core | Not started | WebSocket design review required trước implement. |
| M3 Platform breadth | Not started | OAuth2/OIDC, tasks, search, liveops, E2EE. |

---

## First Task To Do

`BE-00` — baseline.

```bash
go test ./...
go test -race ./internal/service/... ./internal/handler/...
```

Expected output for review:

```text
Task: BE-00
Baseline:
- go test ./...: pass/fail
- go test -race ...: pass/fail
Failures:
- package: reason
Next proposed task:
- T104 (Dockerfile fix) hoặc T101 (error-mapping audit)
```
