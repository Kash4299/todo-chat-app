# Task Breakdown — KashFlow Todo Chat App
> Updated: 2026-06-20 (v2 — JD-fit roadmap merged) | Solo Dev | Target: 1M concurrent users
> Stack: Go + PostgreSQL + Redis + Kafka + MongoDB + WebSocket | Positioning: **Secure real-time collaboration platform**

---

## Legend

| Ký hiệu | Ý nghĩa |
|---|---|
| SP | Story Point (1 SP ≈ 1 ngày làm 4-6 tiếng) |
| Deps | Task phải xong TRƯỚC task này |
| 🔴 / 🟡 / 🟢 | High / Medium / Low risk |
| **M1 / M2 / M3** | Milestone (xem bảng dưới) — quyết định thứ tự làm |
| ✅ | Đã xong |
| ⛔ SUPERSEDED | Bị thay bởi hướng mới (Auth0 → self-hosted identity) |

---

## 🎯 MILESTONE MAP (ROI-first)

Thứ tự không theo "feature đầy đủ" mà theo **deploy được sớm + CV-impact cao nhất trước**. Quyết định nền tảng: **bỏ Auth0, tự host identity (RS256+JWKS), token RS256, JWKS**.

| Milestone | Mục tiêu | User story | Epics |
|---|---|---|---|
| **M1 — Deployable Secure Identity + Workspace** | Deploy được, đủ chiều sâu CV, **chưa cần chat** | register → verify → MFA → login → quản lý session → tạo/quản lý workspace → mời member → RBAC | EPIC 1 (harden), 10, 12, 13, 14, 15, 16 |
| **M2 — Real-time core** | Chat scale ngang qua Kafka | gửi/nhận message real-time đa-instance | EPIC 2, 5 (load gate), 7; audit chuyển sang Kafka |
| **M3 — Platform breadth** | Mở rộng identity + product | OAuth2/OIDC, tasks, search, liveops | EPIC 3, 4, 8, 11, 17, 9 (E2EE) |

> **M1 không dựng Kafka.** Audit ghi thẳng Mongo qua `AuditWriter` interface; M2 đổi implementation sang publish `audit.events`. Giảm moving part khi deploy lần đầu.

---

## EPIC 1 — Authentication & Workspace (13 SP) — **M1**

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| ✅ T01 | Design DB schema (users, workspaces, roles, sessions) | 2 | — | 🔴 | Dùng migration files, multi-tenant isolation |
| ⛔ T02 | ~~Integrate Auth0 JWT validation~~ | 2 | T01 | 🟡 | **SUPERSEDED** bởi EPIC 10 (self-hosted RS256+JWKS). Auth0 path sẽ bị gỡ (T80) |
| ✅ T03 | Identity sync (`sub`,`email`,`name`) → internal user | 1 | T01,T02 | 🟢 | Logic vẫn dùng cho social login, nhưng nguồn token là KashFlow |
| ✅ T04 | Account linking (email/password + Google cùng email) | 2 | T01-T03 | 🔴 | Giữ nguyên policy; provider 'AUTH0' → 'LOCAL'/'GOOGLE' |
| ✅ T05 | Workspace Create & Manage API | 2 | T01 | 🟡 | Mỗi workspace = tenant |
| ✅ T06 | Member Invitation (email + magic link) | 2 | T01,T05 | 🟡 | Token hết hạn 24h |
| ✅ T07 | RBAC middleware (Admin/Member/Guest) | 2 | T01,T05 | 🟡 | **Mở rộng M1**: thêm role OWNER, permission matrix, cache (xem T96) |
| ✅ T08 | User Profile API | 1 | T01 | 🟢 | Avatar = URL |

> EPIC 1 phần lớn đã xong. Việc còn lại cho M1 là **harden** (zero-UUID, pagination, ownership authz) — xem backend-review-schedule Phase 1.

---

## EPIC 10 — Identity Platform Hardening (self-hosted) — **M1** ⭐ CV centerpiece (16 SP)

> Đây là phần thay thế Auth0 và là "ngôi sao" CV cho Identity Platform role. Code đã có local auth (password + verify + HS256 + refresh + linking); epic này nâng nó lên chuẩn production IdP.

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T71 | RSA keypair + `/.well-known/jwks.json` + key rotation (`kid`, 2-key overlap) | 2 | — | 🔴 | Private key từ secret store, không log/commit |
| T72 | Migrate access/ID token HS256 → RS256; verifier dùng cached JWKS | 2 | T71 | 🟡 | In-proc JWKS cache + `singleflight` refresh |
| T73 | Refresh rotation + **reuse detection** (token family → revoke family) | 2 | — | 🔴 | Refresh đã-rotate bị dùng lại = trộm token |
| T74 | Session/device management (list / revoke one / revoke all) | 2 | T73 | 🟡 | `user_sessions` + family_id |
| T75 | MFA TOTP enroll/verify + recovery codes | 3 | T72 | 🔴 | Challenge nằm GIỮA password-ok và token-issue |
| T76 | Email OTP fallback | 1 | T75 | 🟢 | Dùng SMTP/Mailhog |
| T77 | Password reset flow (token hashed, one-time, TTL ngắn) | 1 | — | 🟢 | Không tiết lộ email tồn tại hay không |
| T78 | Login rate limiting + lockout (Redis sliding window per email/IP) | 1 | T95-style Redis | 🟡 | Fail-closed |
| T79 | Identity audit events → `AuditWriter` (login/logout/refresh/mfa/reset) | 1 | T92 | 🟢 | Ghi Mongo (M1), Kafka (M2) |
| T80 | Remove Auth0 path; identity = single source of truth | 1 | T72 | 🟡 | Gỡ Auth0 middleware/config; update AGENTS.md |

---

## EPIC 14 — Redis Production Patterns — **M1** (5 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T95 | Cache-aside user profile + workspace membership (TTL) | 1 | T39 | 🟢 | Hot read trong auth/authorization path |
| T96 | Permission/RBAC cache + **invalidate** khi role/member đổi | 1 | T07,T95 | 🟡 | Stale permission = lỗ hổng authz |
| T97 | Distributed lock (resend email, invite resend, idempotent sensitive actions) | 1 | T39 | 🟡 | `lock:{resource}`, TTL ngắn |
| T98 | Idempotency key storage cho sensitive POST | 1 | T39 | 🟢 | `idem:{key}` |
| T99 | Cache stampede protection (`singleflight` + Redis lock) | 1 | T95 | 🟡 | Key nóng của user active |

---

## EPIC 13 — MongoDB Audit / Activity Log — **M1** (3 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T92 | Mongo integration + `AuditWriter` interface (direct write ở M1) | 1 | T38 | 🟢 | fx lifecycle; OnStop disconnect |
| T93 | `security_audit_logs` + `user_activity_logs` + compound indexes | 1 | T92 | 🟡 | `{user_id,created_at}`, `{event_type,created_at}`, TTL optional |
| T94 | Audit query API (cursor pagination, filter by user/type/time) | 1 | T93 | 🟢 | Index-backed, admin-only |

---

## EPIC 12 — Observability — **M1** (7 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T87 | Structured logging (`slog`/`zap`) + request_id/trace_id, no secrets | 1 | — | 🟢 | Swap `log.Printf` dần |
| T88 | Prometheus metrics + `/metrics` (req/err/latency/ws conn/db) | 2 | — | 🟢 | Histogram cho latency |
| T89 | Grafana dashboard (latency, error rate, conn, db) | 1 | T88 | 🟢 | Provision qua compose |
| T90 | OpenTelemetry tracing (HTTP → service → DB → Redis → Kafka) | 2 | T88 | 🟡 | Propagate trace qua context + Kafka headers |
| T91 | Alert rules (p95>300ms, error>1%, kafka lag) | 1 | T89 | 🟢 | Bắt đầu với 3 alert |

---

## EPIC 15 — Security Hardening & Docs — **M1** (4 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T100 | Security headers + CORS allowlist + WS origin validation | 1 | — | 🟢 | Không `*` ở prod |
| T101 | Audit body-size limit + error mapping (no raw `err.Error()`) toàn handler | 1 | — | 🟢 | Khớp AGENTS rules |
| T102 | `docs/security-checklist.md` | 1 | — | 🟢 | OWASP-based |
| T103 | `docs/threat-model.md` | 1 | — | 🟢 | Tập trung identity attack surface |

---

## EPIC 16 — Deployment & DevOps Expansion — **M1** (12 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T104 | Fix Dockerfile (Go version khớp go.mod) + `.dockerignore` | 1 | — | 🟢 | multi-stage build |
| T105 | Full docker-compose (pg, redis, kafka, mongo, mailhog, prometheus, grafana) | 2 | T104 | 🟡 | `depends_on` + health checks |
| T106 | `/healthz` (alive) + `/readyz` (DB/Redis/Mongo ready) | 1 | — | 🟢 | M1 chưa cần Kafka readiness |
| T107 | GitHub Actions CI (lint → test -race → build → docker image) | 1 | — | 🟢 | golangci-lint |
| T108 | Makefile (test/build/docker-up/docker-down/migrate-up) | 1 | T105 | 🟢 | |
| T109 | K8s manifests / Helm (deploy/svc/ingress/configmap/secret/probes/HPA/limits) | 3 | T106,T107 | 🟡 | Probes → `/healthz`+`/readyz` |
| T110 | Terraform skeleton (VPC, RDS, ElastiCache, ECS/EKS, Mongo) | 2 | — | 🟡 | smallest instance, reproducible |
| T111 | GitHub Actions CD → staging deploy | 1 | T107,T109 | 🟡 | Có thể deploy lên managed container platform để rẻ |

> **Deploy M1**: chọn 1 — (a) managed container platform (Render/Fly/Railway) cho nhanh+rẻ, hoặc (b) managed k8s (GKE/EKS) cho CV "deployed to Kubernetes". K8s manifests (T109) là artifact CV dù live deploy chạy ở đâu.

---

## EPIC 11 — OAuth2 / OIDC Provider — **M3** (9 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T81 | `oauth_clients` + client registration API | 1 | T01 | 🟢 | client_secret hashed |
| T82 | `GET /oauth/authorize` — Authorization Code + PKCE (S256) | 2 | T81 | 🔴 | Verify `S256(verifier)==challenge` |
| T83 | `POST /oauth/token` — code→token, refresh→token | 2 | T82 | 🟡 | Reuse RS256/JWKS từ T71 |
| T84 | ID token + `/userinfo` + `/.well-known/openid-configuration` | 1 | T83 | 🟡 | OIDC discovery |
| T85 | Scope model + consent record | 1 | T82 | 🟢 | `oauth_consents` |
| T86 | Google login = direct OAuth2 client (reframe T58) | 2 | T72 | 🟡 | KashFlow là client của Google, không qua Auth0 |

---

## EPIC 17 — LiveOps — **M3** (6 SP)

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T112 | Announcement API + segmented broadcast | 2 | T54 | 🟡 | Audience: ALL/workspace/role |
| T113 | Feature flags endpoint + Redis cache | 1 | T39 | 🟢 | rollout %/segment rules |
| T114 | Scheduled campaigns (leader via Redis lock, idempotent) | 1 | T97 | 🟡 | |
| T115 | User segments (active/members/role) | 1 | T40 | 🟢 | active = presence |
| T116 | Rate-limited broadcast delivery | 1 | T54 | 🟡 | tránh fan-out storm |

---

## EPIC 2 — Real-time Chat System (27 SP) — **M2**

> WebSocket scalability (roadmap P8) + Kafka pipeline (P4) được merge vào đây dưới dạng ghi chú "P8/P4".

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| ✅ T09 | Design WebSocket architecture | 2 | T01 | 🔴 | Vẽ diagram trước (đã có chat-system-design.md) |
| T10 | WebSocket server & connection pool manager | 3 | T09 | 🔴 | **P8**: sharded room map (room_manager.go đã bắt đầu), bounded send queue per-conn |
| T11 | Kafka topic schema (messages, presence, notifications, audit, email, DLQ) | 1 | T09 | 🔴 | **P4**: thêm `audit.events`,`email.jobs`,`*.DLQ` |
| T12 | Kafka **AsyncProducer** (publish message events) | 2 | T11 | 🟡 | Fix SyncProducer debt |
| T13 | Kafka **ConsumerGroup** (deliver → WS) + idempotency + DLQ | 2 | T11,T12 | 🟡 | **P4**: retry+backoff, dedup, consumer groups |
| T14 | Channel management API | 2 | T01,T05 | 🟢 | Public/Private, member pagination |
| T15 | Direct Message logic | 2 | T14 | 🟢 | DM = private channel 2 users |
| T16 | Message persistence (PostgreSQL) | 2 | T01 | 🟡 | Partition by channel_id/time |
| T17 | Message history API (cursor pagination) | 1 | T16 | 🟢 | Không OFFSET trên messages |
| T18 | Typing indicator | 1 | T10 | 🟢 | Debounce 2s, không persist |
| T19 | Presence (Redis + WS) | 2 | T10,T39 | 🟡 | **P8**: presence tracking Redis |
| T20 | Read status (Delivered/Read) | 2 | T10,T16 | 🟡 | Batch update |
| T21 | Emoji reaction + broadcast | 1 | T16,T10 | 🟢 | |
| T22 | Reply / Thread | 2 | T16 | 🟡 | parent_message_id |
| T23 | WS reconnection & heartbeat | 2 | T10 | 🟡 | **P8**: reconnect-resync by cursor/event_id, slow-consumer disconnect, cross-node fan-out qua Kafka |

---

## EPIC 3 — Task Management (14 SP) — **M3**

| ID | Task | SP | Deps | Risk |
|---|---|---|---|---|
| T24 | Task schema | 1 | T01 | 🟡 |
| T25 | Task CRUD API | 2 | T24 | 🟢 |
| T26 | Subtask breakdown | 1 | T25 | 🟡 |
| T27 | Task Assignment | 1 | T25 | 🟢 |
| T28 | Status Workflow engine | 2 | T25 | 🟡 |
| T29 | Task Comments (link channel) | 2 | T25,T14 | 🟡 |
| T30 | Task Activity Log | 2 | T25 | 🟡 |
| T31 | Kanban Board view API | 2 | T25,T28 | 🟢 |
| T32 | List view API + filters | 1 | T25 | 🟢 |

---

## EPIC 4 — Search (8 SP) — **M3**

| ID | Task | SP | Deps | Risk |
|---|---|---|---|---|
| T33 | PostgreSQL full-text (tsvector + GIN) | 2 | T16,T24 | 🟡 |
| T34 | Message search endpoint | 2 | T33 | 🟢 |
| T35 | Task search endpoint | 1 | T33 | 🟢 |
| T36 | User search within workspace | 1 | T01 | 🟢 |
| T37 | Result ranking + highlight | 2 | T34,T35 | 🟡 |

---

## EPIC 5 — Infrastructure & Scalability (17 SP) — **M1 (base) / M2 (load gates)**

> Docker/Redis nền nằm M1. Load test 1M (roadmap P11) là M2 sau khi có WS/Kafka.

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T38 | Docker Compose (base) | 2 | — | 🟡 | Mở rộng ở T105 (mongo/mailhog/prometheus/grafana) |
| T39 | Redis integration (session cache, rate limit) | 2 | T38 | 🟢 | Nền cho EPIC 14 |
| T40 | Redis presence state | 1 | T39,T19 | 🟢 | **M2** |
| T41 | ✅ API rate limiting middleware | 1 | T39 | 🟢 | Pool config đã có; rate limit middleware cần wire |
| T42 | PostgreSQL read replica routing | 2 | T38 | 🟡 | **M2** |
| T43 | Document horizontal scaling (WS+Kafka fan-out) | 1 | T09,T11 | 🟢 | ✅ đã có trong architecture.md |
| T44 | k6 load test scripts (WS + REST) | 2 | T10,T16 | 🟢 | **P11**: thêm `loadtests/`; M1 chạy smoke cho auth/workspace |
| T45 | Run load test + profile (pprof, slow query) + **performance report** (p50/p95/p99, throughput, bottleneck, before/after, EXPLAIN ANALYZE, Redis hit ratio) | 3 | T44 | 🔴 | **P11** |
| T46 | Optimize 1M conn (OS tuning, backpressure, circuit breaker) | 3 | T45 | 🔴 | **M2** |

---

## EPIC 6 — DevOps & CI/CD (11 SP) — **M1 (CI) / M3 (AWS)**

> CI/Docker nền chuyển sang EPIC 16 (M1). AWS provisioning ở M3.

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| T47 | GitHub Actions CI | 1 | T01 | 🟢 | → gộp/đồng bộ với T107 |
| T48 | Dockerfile multi-stage | 1 | T38 | 🟢 | → T104 |
| T49 | AWS account/IAM/VPC | 2 | — | 🟡 | **M3** (hoặc Terraform T110) |
| T50 | Provision AWS (ECS/RDS/ElastiCache/MSK/Mongo) | 3 | T49 | 🔴 | **M3** |
| T51 | GitHub Actions CD → ECS | 2 | T47,T48,T50 | 🟡 | **M3**; M1 dùng T111 |
| T52 | Monitoring + Alerting (CloudWatch) | 2 | T50 | 🟡 | **M3**; M1 dùng Prometheus (EPIC 12) |

---

## EPIC 7 — Notifications (7 SP) — **M2**

| ID | Task | SP | Deps | Risk |
|---|---|---|---|---|
| T53 | Notification schema + Kafka topic | 1 | T11 | 🟢 |
| T54 | Notification service (consume → persist → WS fan-out) | 2 | T53 | 🟡 |
| T55 | Mention detection (`@username`) | 1 | T16,T53 | 🟢 |
| T56 | Notification Center API | 1 | T54 | 🟢 |
| T57 | Email notification (queue via `email.jobs`) | 2 | T54 | 🟡 |

---

## EPIC 8 — Third-party Integrations (7 SP) — **M3**

| ID | Task | SP | Deps | Risk | Ghi chú |
|---|---|---|---|---|---|
| ⛔ T58 | ~~Google OAuth (thay email/password)~~ | 2 | T02 | 🟡 | **SUPERSEDED/RE-FRAMED** → T86 (direct OAuth2 client) |
| T59 | Spotify integration (now-playing) | 3 | T01 | 🟡 | Token refresh 1h |
| T60 | Outgoing Webhook (HMAC sign, SSRF guard) | 2 | T25,T14 | 🟡 | |

---

## EPIC 9 — End-to-End Encryption / E2EE (22 SP) — **M3 (last)**

> Signal Protocol; chi tiết `docs/e2ee-plan.md`. Prerequisite: frontend + core ổn định.

| ID | Task | SP | Deps | Risk |
|---|---|---|---|---|
| T61 | Schema: key bundles + one-time prekeys | 1 | T01 | 🟢 |
| T62 | Key management API | 2 | T61 | 🟢 |
| T63 | Message schema (is_encrypted, sender_device_id) | 1 | T16,T61 | 🟡 |
| T64 | FE: libsignal + keystore | 3 | T62 | 🔴 |
| T65 | FE: X3DH key exchange | 3 | T64 | 🔴 |
| T66 | FE: Double Ratchet | 3 | T65 | 🔴 |
| T67 | DM E2EE opt-in | 2 | T66,T15 | 🟡 |
| T68 | Group E2EE (Sender Keys) | 3 | T66 | 🔴 |
| T69 | Client-side search for E2EE | 2 | T67 | 🟡 |
| T70 | Security audit + pentest | 2 | T67,T68 | 🔴 |

---

## Tổng kết Story Points

| Epic | SP | Milestone |
|---|---|---|
| EPIC 1 — Auth & Workspace | 13 | M1 (harden) |
| EPIC 10 — Identity Platform Hardening ⭐ | 16 | M1 |
| EPIC 14 — Redis Production Patterns | 5 | M1 |
| EPIC 13 — MongoDB Audit Log | 3 | M1 |
| EPIC 12 — Observability | 7 | M1 |
| EPIC 15 — Security Hardening & Docs | 4 | M1 |
| EPIC 16 — Deployment & DevOps Expansion | 12 | M1 |
| EPIC 2 — Real-time Chat (incl. P8/P4) | 27 | M2 |
| EPIC 5 — Infrastructure | 17 | M1/M2 |
| EPIC 7 — Notifications | 7 | M2 |
| EPIC 6 — DevOps (AWS) | 11 | M1/M3 |
| EPIC 11 — OAuth2/OIDC Provider | 9 | M3 |
| EPIC 3 — Task Management | 14 | M3 |
| EPIC 4 — Search | 8 | M3 |
| EPIC 17 — LiveOps | 6 | M3 |
| EPIC 8 — Integrations | 7 | M3 |
| EPIC 9 — E2EE | 22 | M3 |
| **TOTAL** | **~188 SP** | |

> Với TDD (~1.5x): tổng thực tế ≈ **280 ngày**. **Milestone 1 ≈ 50-55 SP (~75-80 ngày)** — đây là phần bạn deploy + đưa lên CV trước.

---

## Sprint Order — Milestone 1 (deploy-first, ROI-first)

> Mỗi sprint 2 tuần. Đây chỉ là M1; M2/M3 lập kế hoạch lại sau khi M1 deploy.

| Sprint | Tasks | Goal | CV unlock |
|---|---|---|---|
| S1 | BE-00 audit, T104, T105, T106, T108 | Docker env đầy đủ + health endpoints + Makefile | "Containerized, health-checked Go service" |
| S2 | T71, T72, T80 | RS256 + JWKS + gỡ Auth0 | "Self-hosted JWT with RS256/JWKS" |
| S3 | T73, T74, T77 | Refresh rotation+reuse detection, sessions, password reset | "Refresh rotation, session/device mgmt" |
| S4 | T75, T76, T78, T79 | MFA + login rate limit + audit events | "TOTP MFA, brute-force protection, audit log" |
| S5 | T92, T93, T94, T95-T99 | Mongo audit + Redis production patterns | "MongoDB audit w/ indexes; Redis cache/lock/idempotency" |
| S6 | T87, T88, T89, T90, T91 | Observability stack | "Prometheus/Grafana/OpenTelemetry" |
| S7 | EPIC 1 harden (BE-05→09), T07 RBAC matrix+cache (T96), T100, T101 | Workspace/RBAC hardening + security headers | "Workspace RBAC w/ permission cache" |
| S8 | T102, T103, T107, T109, T110, T111 | Threat model + CI/CD + K8s/Helm/Terraform + deploy | "CI/CD + Kubernetes deploy" → **M1 LIVE** |

After S8: M1 deployed, CV-ready. Then plan M2 (chat/WS/Kafka) starting at T09→T23 + EPIC 5 load gates.
