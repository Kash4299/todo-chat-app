# Task Breakdown — KashFlow Todo Chat App
> Generated: 2026-04-25 | Solo Dev | Stack: Go + PostgreSQL + Kafka + WebSocket | Target: 1M concurrent users

---

## Legend

| Ký hiệu | Ý nghĩa |
|---|---|
| SP | Story Point (1 SP ≈ 1 ngày làm việc 4-6 tiếng) |
| Deps | Task phải hoàn thành TRƯỚC task này |
| 🔴 High Risk | Có thể gây block hoặc cần re-architecture |
| 🟡 Medium Risk | Cần chú ý, có thể phức tạp hơn dự kiến |
| 🟢 Low Risk | Straightforward, ít rủi ro |

**Tổng ước tính: ~104 SP (ngày). Với TDD x1.5 = ~150-160 ngày thực tế.**

---

## EPIC 1 — Authentication & Workspace (13 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T01 | Design DB schema (users, workspaces, roles, sessions) | Thiết kế nền dữ liệu ổn định cho identity, workspace, membership và audit để tránh migration đắt đỏ về sau. | 2 | — | 🔴 Schema sai sẽ tốn kém để migrate sau | Dùng migration files, nghĩ kỹ multi-tenant isolation |
| T02 | Integrate Auth0 JWT validation (RS256 + JWKS + issuer/audience) | Đảm bảo mọi request protected xác thực đúng chữ ký và claims theo Auth0, không dùng local token signing. | 2 | T01 | 🟡 Sai config sẽ khóa toàn bộ auth flow | API không tự ký HS256 token |
| T03 | Build identity sync from Auth0 claims (`sub`, `email`, `name`) to internal user profile | Đồng bộ identity ngoài vào user nội bộ để các service business dùng `user_id` ổn định. | 1 | T01, T02 | 🟢 | Tạo/cập nhật user nội bộ sau login thành công |
| T04 | Implement account linking design (email/password + Google cùng email) | Gộp nhiều phương thức login về một user domain model để tránh duplicate account và sai lệch quyền. | 2 | T01, T02, T03 | 🔴 Link sai gây account takeover | Bắt buộc verified email + explicit consent/step-up |
| T05 | Build Workspace Create & Manage API | Cung cấp tenant boundary qua workspace để các dữ liệu và quyền được cô lập đúng user scope. | 2 | T01 | 🟡 Multi-tenant isolation logic | Mỗi workspace = tenant riêng biệt |
| T06 | Build Member Invitation system (email + magic link) | Tạo quy trình thêm thành viên an toàn, có hạn dùng và khả năng truy vết để onboarding workspace. | 2 | T01, T05 | 🟡 Email delivery, token expiry | Token mời hết hạn sau 24h |
| T07 | Implement RBAC middleware (Admin / Member / Guest) | Chuẩn hoá phân quyền ở biên service để kiểm soát hành vi theo role một cách nhất quán. | 2 | T01, T05 | 🟡 Permission matrix phức tạp khi scale | Dùng middleware pattern, không hardcode |
| T08 | Build User Profile API (avatar, display name, status text) | Cho phép user tự quản thông tin hiển thị, tách khỏi logic authentication và giữ API profile độc lập. | 1 | T01 | 🟢 | Avatar lưu URL, chưa cần file upload |

---

## EPIC 2 — Real-time Chat System (27 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T09 | Design WebSocket architecture (connection manager, room routing) | Vẽ diagram trước, review kỹ trước khi code | 2 | T01 | 🔴 Sai design = phải rewrite toàn bộ | Vẽ diagram trước, review kỹ trước khi code |
| T10 | Implement WebSocket server & connection pool manager | Dùng sync.Map hoặc sharded map cho connection pool | 3 | T09 | 🔴 Goroutine leak, memory leak | Dùng sync.Map hoặc sharded map cho connection pool |
| T11 | Design Kafka topic schema (messages, presence, notifications) | Topic: `chat.messages`, `chat.presence`, `chat.notifications` | 1 | T09 | 🔴 Topic design khó thay đổi sau khi production | Topic: `chat.messages`, `chat.presence`, `chat.notifications` |
| T12 | Implement Kafka producer (publish message events) | Dùng async producer, handle errors | 2 | T11 | 🟢 | Dùng async producer, handle errors |
| T13 | Implement Kafka consumer (deliver events tới WebSocket clients) | Consumer group per WebSocket server instance | 2 | T11, T12 | 🟡 At-least-once delivery + dedup logic | Consumer group per WebSocket server instance |
| T14 | Build Channel management API (create, join, leave, list members) | Public/Private channel, member list pagination | 2 | T01, T05 | 🟢 | Public/Private channel, member list pagination |
| T15 | Build Direct Message (DM) logic | DM = private channel giữa 2 users | 2 | T14 | 🟢 | DM = private channel giữa 2 users |
| T16 | Implement Message persistence layer (PostgreSQL) | Xem xét partitioning theo channel_id hoặc time | 2 | T01 | 🟡 Write throughput cao khi scale | Xem xét partitioning theo channel_id hoặc time |
| T17 | Build Message history API với cursor-based pagination | Dùng cursor (message_id) thay offset pagination | 1 | T16 | 🟢 | Dùng cursor (message_id) thay offset pagination |
| T18 | Implement Typing indicator (WebSocket broadcast event) | Debounce 2s, không persist vào DB | 1 | T10 | 🟢 | Debounce 2s, không persist vào DB |
| T19 | Build Online/Offline Presence system (Redis + WebSocket) | TTL-based presence trong Redis | 2 | T10, T39 | 🟡 Consistency khi user có nhiều tab/device | TTL-based presence trong Redis |
| T20 | Implement Message read status (Delivered / Read receipts) | Batch update, không per-message event | 2 | T10, T16 | 🟡 Fan-out phức tạp trong group message lớn | Batch update, không per-message event |
| T21 | Build Emoji Reaction API + real-time broadcast | Giới hạn N loại emoji | 1 | T16, T10 | 🟢 | Giới hạn N loại emoji |
| T22 | Build Reply / Thread system | parent_message_id trên message, thread count cache | 2 | T16 | 🟡 Thread pagination UX phức tạp | parent_message_id trên message, thread count cache |
| T23 | Implement WebSocket reconnection & heartbeat (ping/pong) | Client gửi last_message_id để server resync | 2 | T10 | 🟡 State sync khi reconnect | Client gửi last_message_id để server resync |

---

## EPIC 3 — Task Management (14 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T24 | Design Task schema (task, subtask, priority, status, assignee) | Dùng ENUM cho status, array cho tags | 1 | T01 | 🟡 Schema phải flexible cho workflow sau | Dùng ENUM cho status, array cho tags |
| T25 | Build Task CRUD API (Create, Read, Update, Delete) | Validate business rules (due date, priority) | 2 | T24 | 🟢 | Validate business rules (due date, priority) |
| T26 | Build Subtask breakdown API (parent-child relationship) | parent_task_id nullable | 1 | T25 | 🟡 Giới hạn depth (max 2-3 level) để tránh recursion phức tạp | parent_task_id nullable |
| T27 | Implement Task Assignment API (assign, reassign, unassign) | Trigger notification khi assign (kết nối T53) | 1 | T25 | 🟢 | Trigger notification khi assign (kết nối T53) |
| T28 | Build Status Workflow engine (Todo → In Progress → Review → Done) | State machine pattern, log mỗi transition | 2 | T25 | 🟡 Custom workflow sẽ phức tạp hơn | State machine pattern, log mỗi transition |
| T29 | Build Task Comments API (liên kết với channel chat) | Task tự động tạo channel riêng, hoặc embed comment | 2 | T25, T14 | 🟡 Sync giữa task comment và chat message | Task tự động tạo channel riêng, hoặc embed comment |
| T30 | Build Task Activity Log (ai thay đổi gì, khi nào) | Event sourcing lite: lưu event type + old/new value | 2 | T25 | 🟡 Storage growth theo thời gian | Event sourcing lite: lưu event type + old/new value |
| T31 | Build Kanban Board view API (group by status, drag-drop order) | Thêm `position` field cho card ordering | 2 | T25, T28 | 🟢 | Thêm `position` field cho card ordering |
| T32 | Build List view API với filters (assignee, status, priority, date) | Dùng query builder, support multiple filters | 1 | T25 | 🟢 | Dùng query builder, support multiple filters |

---

## EPIC 4 — Search (8 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T33 | Setup PostgreSQL Full-text search (tsvector + GIN index) | Thêm `search_vector` column, trigger tự update | 2 | T16, T24 | 🟡 Vietnamese tokenization cần `unaccent` extension | Thêm `search_vector` column, trigger tự update |
| T34 | Build Message search endpoint (keyword, scope by channel) | Highlight matched terms, cursor pagination | 2 | T33 | 🟢 | Highlight matched terms, cursor pagination |
| T35 | Build Task search endpoint (title, status, assignee, date range) | Combine full-text + filter queries | 1 | T33 | 🟢 | Combine full-text + filter queries |
| T36 | Build User search within workspace | ILIKE search trên display_name + email | 1 | T01 | 🟢 | ILIKE search trên display_name + email |
| T37 | Implement search result ranking & keyword highlight | Dùng `ts_rank`, highlight dùng `ts_headline` | 2 | T34, T35 | 🟡 Relevance tuning mất nhiều thời gian | Dùng `ts_rank`, highlight dùng `ts_headline` |

---

## EPIC 5 — Infrastructure & Scalability (17 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T38 | Setup Docker Compose (Go, PostgreSQL, Redis, Kafka, Zookeeper) | Dùng `depends_on` với health checks | 2 | — | 🟡 Port conflicts, health check ordering | Dùng `depends_on` với health checks |
| T39 | Implement Redis integration (session cache, rate limiting) | Redis Cluster mode khi scale | 2 | T38 | 🟢 | Redis Cluster mode khi scale |
| T40 | Implement Redis presence state management | Key: `presence:{user_id}`, TTL 30s, heartbeat refresh | 1 | T39, T19 | 🟢 | Key: `presence:{user_id}`, TTL 30s, heartbeat refresh |
| T41 | Implement API Rate Limiting middleware (per user, per IP) | Sliding window counter trong Redis | 1 | T39 | 🟢 | Sliding window counter trong Redis |
| T42 | Setup PostgreSQL read replica routing (writes → primary, reads → replica) | Dùng `pgx` connection pool với 2 connection strings | 2 | T38 | 🟡 Replication lag ảnh hưởng reads | Dùng `pgx` connection pool với 2 connection strings |
| T43 | Document horizontal scaling architecture (WebSocket + Kafka fan-out) | Diagram: nhiều WebSocket servers → share qua Kafka | 1 | T09, T11 | 🟢 | Diagram: nhiều WebSocket servers → share qua Kafka |
| T44 | Write k6 load test scripts (WebSocket connections + REST APIs) | Test scenarios: 100 → 1000 → 10K → 100K → 1M | 2 | T10, T16 | 🟢 | Test scenarios: 100 → 1000 → 10K → 100K → 1M |
| T45 | Run load test + profile bottlenecks (pprof, DB slow queries) | Dùng Go pprof, PostgreSQL EXPLAIN ANALYZE | 3 | T44, toàn bộ system | 🔴 Có thể phát hiện vấn đề kiến trúc sâu | Dùng Go pprof, PostgreSQL EXPLAIN ANALYZE |
| T46 | Optimize cho 1M concurrent connections (OS tuning, code fix) | `sysctl` tuning, connection backpressure, circuit breaker | 3 | T45 | 🔴 OS-level: ulimit, TCP buffer, file descriptors | `sysctl` tuning, connection backpressure, circuit breaker |

---

## EPIC 6 — DevOps & CI/CD (11 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T47 | Setup GitHub Actions CI pipeline (lint → test → build) | golangci-lint, go test -race, go build | 1 | T01 | 🟢 | golangci-lint, go test -race, go build |
| T48 | Write Dockerfile cho Go service (multi-stage build) | Builder stage + distroless/alpine final image | 1 | T38 | 🟢 | Builder stage + distroless/alpine final image |
| T49 | Setup AWS account, IAM roles, VPC, subnets, security groups | Dùng Terraform hoặc AWS CDK để reproducible | 2 | — | 🟡 AWS cost nếu misconfigure, IAM tối thiểu principle | Dùng Terraform hoặc AWS CDK để reproducible |
| T50 | Provision AWS services (ECS Fargate, RDS, ElastiCache, MSK) | Bắt đầu với smallest instance, scale sau | 3 | T49 | 🔴 MSK (managed Kafka) config phức tạp và tốn kém | Bắt đầu với smallest instance, scale sau |
| T51 | Setup GitHub Actions CD pipeline (build → push ECR → deploy ECS) | Blue/green deployment để zero downtime | 2 | T47, T48, T50 | 🟡 ECS task definition + service update phức tạp | Blue/green deployment để zero downtime |
| T52 | Setup Monitoring + Alerting (CloudWatch metrics + alarms) | Metrics: CPU, memory, WebSocket conn count, Kafka lag | 2 | T50 | 🟡 Alert fatigue nếu threshold sai | Metrics: CPU, memory, WebSocket conn count, Kafka lag |

---

## EPIC 7 — Notifications (7 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T53 | Design Notification schema + Kafka topic (`chat.notifications`) | Notification types: mention, task_assigned, due_date | 1 | T11 | 🟢 | Notification types: mention, task_assigned, due_date |
| T54 | Build Notification service (consume Kafka → persist → fan-out to WS) | Batch fan-out, rate limit per user | 2 | T53 | 🟡 Fan-out storm khi group lớn | Batch fan-out, rate limit per user |
| T55 | Build Mention detection trong messages (`@username` parsing) | Parse khi message được save, trigger notification event | 1 | T16, T53 | 🟢 | Parse khi message được save, trigger notification event |
| T56 | Build Notification Center API (list unread, mark read, clear all) | Cursor pagination, filter by type | 1 | T54 | 🟢 | Cursor pagination, filter by type |
| T57 | Build Email notification service (task assigned, @mention) | Dùng SendGrid hoặc AWS SES, queue email delivery | 2 | T54 | 🟡 Email deliverability, SMTP config | Dùng SendGrid hoặc AWS SES, queue email delivery |

---

## EPIC 8 — Third-party Integrations (7 SP)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T58 | Implement Google OAuth login (thay thế email/password) | Thêm provider field vào users table | 2 | T02 | 🟡 OAuth flow dễ sai ở callback handling | Thêm provider field vào users table |
| T59 | Spotify integration (link account, display now-playing trên profile) | Poll hoặc dùng Spotify webhook nếu available | 3 | T01 | 🟡 Spotify API rate limits, access token refresh mỗi 1h | Poll hoặc dùng Spotify webhook nếu available |
| T60 | Build Outgoing Webhook system (notify external services khi có event) | Validate URL (whitelist scheme), sign payload bằng HMAC | 2 | T25, T14 | 🟡 SSRF vulnerability, retry với exponential backoff | Validate URL (whitelist scheme), sign payload bằng HMAC |

---

## EPIC 9 — End-to-End Encryption / E2EE (22 SP)

> Approach: Signal Protocol (Messenger-style) — chi tiết xem `docs/e2ee-plan.md`
> Prerequisite: Frontend hoàn thiện + core app ổn định (sau Sprint 13)

| ID | Task Name | Description | SP | Dependencies | Risk | Ghi chú |
|---|---|---|---|---|---|---|
| T61 | DB schema: user_key_bundles + user_one_time_prekeys | Lưu public keys của từng device | 1 | T01 | 🟢 | Private keys không bao giờ lên server |
| T62 | Key management API (upload bundle, fetch bundle, replenish OPKs) | Server relay public keys, không xử lý crypto | 2 | T61 | 🟢 | GET /users/:id/keys, POST /users/me/keys |
| T63 | Message schema changes (is_encrypted, sender_device_id) | Disable search_vector cho encrypted messages | 1 | T16, T61 | 🟡 | Search không hoạt động với E2EE messages |
| T64 | Frontend: libsignal setup + keystore (IndexedDB + WebCrypto) | Private keys lưu trên device, không rời browser | 3 | T62 | 🔴 | Dùng @signalapp/libsignal-client, không tự implement |
| T65 | Frontend: X3DH initial key exchange | Alice fetch Bob's key bundle → derive shared secret | 3 | T64 | 🔴 | Sai X3DH = toàn bộ session insecure |
| T66 | Frontend: Double Ratchet integration | Forward secrecy cho ongoing messages | 3 | T65 | 🔴 | Mỗi message dùng key mới |
| T67 | DM E2EE — opt-in phase | Chỉ DM mới tạo sau ngày X mới encrypted | 2 | T66, T15 | 🟡 | Existing DMs giữ nguyên plaintext |
| T68 | Group E2EE — Sender Keys | Một Sender Key per group, distribute qua X3DH | 3 | T66 | 🔴 | Phức tạp hơn DM, member join/leave cần re-key |
| T69 | Client-side search (replace server search cho E2EE channels) | Index encrypted messages locally trên device | 2 | T67 | 🟡 | Server không thể search ciphertext |
| T70 | Security audit + penetration testing | Bắt buộc trước khi ship E2EE ra production | 2 | T67, T68 | 🔴 | Không audit = không ship |

---

## Tổng kết

| Epic | Story Points | Ưu tiên |
|---|---|---|
| EPIC 1 — Auth & Workspace | 13 SP | P0 — Phải làm đầu tiên |
| EPIC 2 — Real-time Chat | 27 SP | P0 — Core của sản phẩm |
| EPIC 5 — Infrastructure | 17 SP | P0 — Song song với Epic 2 |
| EPIC 3 — Task Management | 14 SP | P1 |
| EPIC 6 — DevOps & CI/CD | 11 SP | P1 — Bắt đầu sớm, làm dần |
| EPIC 4 — Search | 8 SP | P2 |
| EPIC 7 — Notifications | 7 SP | P2 |
| EPIC 8 — Integrations | 7 SP | P3 — Nice-to-have |
| EPIC 9 — E2EE Encryption | 22 SP | P3 — Sau Sprint 13, frontend required |
| **TOTAL** | **126 SP** | |

> **Lưu ý thực tế**: Với TDD (viết test trước), mỗi task thực tế tốn ~1.5x. Tổng thực tế ≈ **185-200 ngày làm việc**.
> Với 4-6 tiếng/ngày, nên ưu tiên P0 tasks để có demo được trước tháng 6, P1-P2 hoàn thiện trước tháng 8.

---

## Recommended Sprint Order (2-week sprints)

| Sprint | Tasks | Goal |
|---|---|---|
| Sprint 1 | T01, T02, T03, T04, T38 | Auth foundation + Docker local env |
| Sprint 2 | T05, T06, T07, T08, T47, T48 | Workspace + CI pipeline |
| Sprint 3 | T09, T10, T11 | WebSocket + Kafka architecture (thiết kế kỹ nhất) |
| Sprint 4 | T12, T13, T14, T15, T16 | Chat core: messages + channels |
| Sprint 5 | T17, T18, T19, T20, T39, T40, T41 | Chat features + Redis |
| Sprint 6 | T21, T22, T23, T42, T43 | Chat polish + DB replica |
| Sprint 7 | T24, T25, T26, T27, T28 | Task management core |
| Sprint 8 | T29, T30, T31, T32, T44 | Task features + load test scripts |
| Sprint 9 | T33, T34, T35, T36, T37 | Search |
| Sprint 10 | T45, T46 | Load test + optimize (1M target) |
| Sprint 11 | T49, T50, T51, T52 | AWS deployment |
| Sprint 12 | T53, T54, T55, T56, T57 | Notifications |
| Sprint 13 | T58, T59, T60 | Integrations (stretch goal) |
| Sprint 14 | T61, T62, T63 | E2EE: backend schema + key API |
| Sprint 15 | T64, T65, T66 | E2EE: frontend libsignal + X3DH + Double Ratchet |
| Sprint 16 | T67, T68 | E2EE: DM + Group encryption |
| Sprint 17 | T69, T70 | E2EE: client-side search + security audit |
