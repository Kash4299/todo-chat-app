# Chat System Design — T09 (WebSocket Architecture)

> Sprint: 3 | Task: T09 (Design WebSocket architecture) | Status: Step 1 done
> Scale target: 1M concurrent users | Stack: Go + PostgreSQL + Kafka + Redis

---

## Design Process — 7-Step Framework

| Step | Topic | Status |
|---|---|---|
| 1 | Functional Requirements (events catalog) | ✅ Done |
| 2 | Non-functional Requirements + Scale Estimation | ⏳ Pending |
| 3 | Constraints & Assumptions | ⏳ Pending |
| 4 | Identify Hard Parts | ⏳ Pending |
| 5 | Design Options + Tradeoffs | ⏳ Pending |
| 6 | Failure Modes | ⏳ Pending |
| 7 | Diagrams + Final Doc | ⏳ Pending |

---

## Step 1 — Functional Requirements

### 1.1 Decision Criteria

Áp dụng nhất quán cho mọi event trong hệ thống:

1. **Pattern A (WebSocket)** dùng cho:
   - Hot-path messages (send, react, typing) — latency-critical
   - Connection lifecycle (auth, ping, error) — buộc qua WS theo định nghĩa
   - Real-time S→C broadcasts — raison d'être của WS

2. **REST** dùng cho:
   - Rare/idempotent CRUD (`message.edit`, `message.delete`)
   - Business resources có authorization phức tạp (channel CRUD, task CRUD)
   - Bulk fetches (message history, search, member list pagination)

3. **S→C events LUÔN qua WS**, kể cả khi command đi REST.
   → REST trả 200 OK cho caller; broadcast cho các subscribers KHÁC qua WS.

4. **Naming convention:**
   - C→S commands: imperative — `message.send`, `reaction.add`
   - S→C events: past-tense — `message.created`, `reaction.added`

---

### 1.2 Table A — WebSocket Protocol Events

#### A.1 Connection Lifecycle (5 events)

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 1 | `connection.auth` | C→S | once/connection | First frame sau WS handshake, gửi JWT. Server verify (T02 RS256 + JWKS). Fail → close với code 4401 |
| 2 | `connection.authenticated` | S→C | once | Confirm + trả `session_id`, `server_time` (cho clock skew). Hoặc close 4401 |
| 3 | `connection.ping` | C→S | 0.05 (~20s) | App-level heartbeat (không dùng WS RFC ping vì cần passing data: `last_seen_msg_id`) |
| 4 | `connection.pong` | S→C | 0.05 | Reply ping. Server detect disconnect khi miss 2-3 pings (40-60s) |
| 5 | `connection.error` | S→C | rare | Server push: rate limit, session expired, server overload. Client decide retry/relogin |

#### A.2 Message Hot-Path (3 events) — Pattern A pure

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 6 | `message.send` | C→S | 0.05 | Payload: `{ client_msg_id, channel_id, content, parent_id?, reply_to? }`. `client_msg_id` = idempotency key |
| 7 | `message.ack` | S→C | 0.05 | Map `client_msg_id` → `server_msg_id`. Client unbuffer pending message. Critical cho retry logic |
| 8 | `message.created` | S→C | 0.5 peak | Broadcast tới subscribers của channel |

#### A.3 Message CRUD broadcasts (2 events)

> Commands `PATCH /messages/:id`, `DELETE /messages/:id` đi REST (xem Table B).

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 9 | `message.updated` | S→C | 0.01 | Broadcast khi edit. Payload: `{ msg_id, new_content, edited_at, version }` |
| 10 | `message.deleted` | S→C | 0.005 | Broadcast khi delete. Tránh ghost message |

#### A.4 Reactions (4 events) — Pattern A

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 11 | `reaction.add` | C→S | 0.05 | Hot-path, latency thấp cho UX |
| 12 | `reaction.remove` | C→S | 0.02 | Same |
| 13 | `reaction.added` | S→C | 0.1 peak | Broadcast |
| 14 | `reaction.removed` | S→C | 0.05 | Broadcast |

#### A.5 Typing (2 events) — fire-and-forget

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 15 | `typing.start` | C→S | 0.2 burst | Client debounce 1-2s. Server auto-clear sau 5s không nhận thêm → không cần `typing.stop` |
| 16 | `typing.updated` | S→C | 0.2 peak | Broadcast: `{ user_id, channel_id, typing: bool, expires_at }` |

#### A.6 Read Receipts (2 events) — batched

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 17 | `message.read` | C→S | 0.02 | Cursor-based: `{ channel_id, last_read_msg_id }`. Batch theo time hoặc khi user idle, không gửi mỗi message (T20) |
| 18 | `message.read_by` | S→C | 0.05 | Broadcast: `{ user_id, channel_id, last_read_msg_id }`. Trong group lớn, fan-out nặng → cân nhắc throttle |

#### A.7 Channel Subscription (4 events) — WS-level subscription

> Subscription = "tôi muốn nhận events của channel này" (WS state).
> Khác với membership = ACL state (DB). Phải tách rõ — sẽ chi tiết ở Step 4 (room routing).

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 19 | `channel.subscribe` | C→S | 0.01 | Bắt đầu nhận broadcasts. Server check ACL trước khi add subscriber |
| 20 | `channel.unsubscribe` | C→S | 0.01 | Stop nhận. Khi user đóng tab/đổi channel |
| 21 | `channel.member_joined` | S→C | 0.01 | User mới join channel (membership thay đổi, khác subscription) |
| 22 | `channel.member_left` | S→C | 0.01 | Same |

#### A.8 Channel/Workspace metadata broadcasts (4 events)

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 23 | `channel.created` | S→C | rare | Workspace có channel mới → user trong workspace nhận để update sidebar |
| 24 | `channel.updated` | S→C | rare | Name, topic, settings thay đổi |
| 25 | `channel.deleted` | S→C | rare | Update sidebar, đóng channel nếu user đang mở |
| 26 | `workspace.updated` | S→C | very rare | Workspace settings, member list thay đổi |

#### A.9 Presence (1 event)

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 27 | `presence.updated` | S→C | 0.02 | Broadcast online/away/offline. Subscribed theo workspace hoặc "buddy list". Driven bởi ping timeout (T19/T40) |

#### A.10 Notifications (1 event, unified)

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 28 | `notification.created` | S→C | 0.1 peak | Unified channel cho mọi notification type (mention, task assigned, due soon). Payload có `type` field. T54 sẽ implement consumer |

#### A.11 Task broadcasts (5 events) — REST commands + WS broadcasts

> Task CRUD đi REST (xem Table B). Đây chỉ là S→C broadcasts.

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 29 | `task.assigned` | S→C | 0.005 | Notify assignee (T27) |
| 30 | `task.status_changed` | S→C | 0.05 peak | Sync Kanban board real-time (T31). Fan-out tới subscribers của project/board |
| 31 | `task.updated` | S→C | 0.02 | Field khác (priority, due_date, description) — gộp 1 event với `changed_fields` array để tránh event noise |
| 32 | `task.due_soon` | S→C | 0.01 | Server cron triggered (không từ user action) |
| 33 | `task.comment_created` | S→C | 0.05 | Notify task followers (T29) |

#### A.12 Multi-device sync (2 events)

| # | Event | Dir | Freq | Note |
|---|---|---|---|---|
| 34 | `session.synced` | S→C | 0.005 | Cross-device state: read state, draft sync giữa các device của cùng user |
| 35 | `conversation.synced` | S→C | 0.02 | Sau reconnect: server replay events đã miss kể từ `last_event_id` client gửi (T23) |

---

### 1.3 Table B — Paired REST Endpoints

| Method | Path | Pairs với WS event | Note |
|---|---|---|---|
| `PATCH` | `/messages/:id` | → `message.updated` | Edit rare, idempotent |
| `DELETE` | `/messages/:id` | → `message.deleted` | Rare |
| `GET` | `/channels/:id/messages?cursor=...` | n/a | Bulk fetch, cursor pagination (T17) |
| `POST` | `/channels` | → `channel.created` | Resource creation |
| `PATCH` | `/channels/:id` | → `channel.updated` | Settings update |
| `DELETE` | `/channels/:id` | → `channel.deleted` | |
| `POST` | `/channels/:id/members` | → `channel.member_joined` | Add member (RBAC check) |
| `DELETE` | `/channels/:id/members/:user_id` | → `channel.member_left` | Remove member |
| `POST` | `/tasks` | → `task.created` (notification) | Task creation |
| `PATCH` | `/tasks/:id` | → `task.updated` / `task.assigned` / `task.status_changed` | Routes về event tương ứng theo `changed_fields` |
| `DELETE` | `/tasks/:id` | → `task.deleted` (notification) | |
| `POST` | `/tasks/:id/comments` | → `task.comment_created` | |
| `POST` | `/auth/refresh` | n/a | Token refresh, security tách riêng (không đụng WS) |
| `GET` | `/search?q=...` | n/a | Search REST (T34) |

---

### 1.4 Decisions có chủ ý + Rationale

| Decision | Rationale |
|---|---|
| **Hybrid Pattern (A + REST)** | Pattern A pure đơn giản nhưng REST middleware ecosystem (auth, validation, rate-limit, audit log) trưởng thành hơn cho business CRUD. Industry-standard (Slack/Discord). |
| **`message.send` qua WS, KHÔNG REST** | Hot-path. Latency budget 50-100ms end-to-end. HTTP POST + JWT verify mỗi request mất 20-50ms. WS đã established → chỉ frame send. |
| **`task.update_status` qua REST** | Status change ít hơn message bằng nhiều order of magnitude. REST cho phép strong validation (state machine — T28), audit log dễ. Trade 5-20ms latency = chấp nhận được. |
| **`message.read` batch không per-message** | T20 ghi rõ. Per-message read receipt trong group 100 user = 100 fan-out events / 1 message read = bùng nổ. |
| **`typing.stop` bị loại** | Server-side timeout 5s đơn giản hơn + robust hơn (client crash không gửi stop = bug). Industry standard. |
| **`reaction.*` qua WS dù tần suất "rare"** | Frequency 0.05 không thực sự rare. UX expectation: react thấy ngay. Latency budget 100ms. |
| **`notification.created` unified với `type` field** | Mention, task assign, due soon, system message gộp 1 event. Đỡ event proliferation, client switch theo type. |
| **`channel.subscribe` ≠ `channel.member_joined`** | Subscription = WS state ("muốn nhận broadcasts"). Membership = ACL state (DB). Hai khái niệm phải tách rõ ngay từ đầu — sẽ đào sâu ở Step 4 (room routing). |
| **Auth via `connection.auth` event, không qua URL query param** | URL params bị log trong access log → JWT leak. First-frame auth là pattern an toàn. |

---

### 1.5 Tổng kết Step 1

- **35 WebSocket events** chia 12 nhóm
- **14 REST endpoints** kèm theo
- Decision criteria + rationale articulate xong
- Naming convention cố định

---

## Step 2 — Non-functional Requirements + Scale Estimation

> **Quy tắc**: mọi số phải có derivation chain. Format `metric = input₁ × input₂ × ... = result`.
> Assumption ghi rõ để sau review lại.

### 2.1 Target Scale Metrics

**Architectural target**: hỗ trợ **1,000,000 concurrent connections** ở peak (theo spec T09 + architecture.md).

```
Derivation (top-down từ MAU):

MAU                        = 5,000,000        (assumption baseline)
DAU/MAU ratio              = 50%              (chat/collab benchmark: Slack 50%, Discord 50%, Teams 60%, WhatsApp 70%)
DAU                        = 5M × 0.50        = 2,500,000
Peak concurrent factor     = 40%              (work hour spike: 9-11AM + 2-4PM)
Concurrent users (peak)    = 2.5M × 0.40      = 1,000,000
Multi-device factor        = 1.0              (assume 1 active device/user — phone OR laptop. Multi-device login sync qua S→C event, không multiply connection count)
Concurrent connections     = 1M × 1.0         = 1,000,000  ← architecture target
```

**Connection state breakdown:**

```
Active fraction   = 30%      (typing/reading/scrolling actively)  → 300K active
Idle fraction     = 70%      (logged in, app background/inactive) → 700K idle
```

> **Tại sao tách active/idle quan trọng**: idle connection chỉ tốn RAM (constant), không tốn CPU. Active connection tốn cả RAM + CPU (process events). Sizing CPU dựa trên active count, sizing RAM dựa trên total count.

---

### 2.2 Latency Budget

Per event class. Latency budget = **end-to-end** (client A action → client B nhận event).

#### Hot-path: `message.send` → `message.created` (peer nhận)

```
Target: p50 < 100ms, p99 < 250ms

Breakdown:
  Client A → Server WS frame    : 10-20ms   (network RTT/2 + frame parse)
  Server auth + validate        : 5-10ms    (JWT cached after connection.auth, idempotency check Redis)
  DB write (async, fire+forget) : 0ms       (acked qua kafka, persist async)
  Kafka publish                 : 1-5ms     (async producer, local buffer)
  Cross-node Kafka consume      : 5-20ms    (consumer group lag)
  Server B fan-out + WS frame   : 10-20ms

Sum (median path): 30-75ms  → fits p50 100ms
Sum (tail path):   ~200ms   → fits p99 250ms
```

> **Decision**: DB write KHÔNG nằm trong critical path. Server publish Kafka → ACK ngay. Persistence async (T16). Nếu DB write fail, retry qua Kafka consumer + reconcile. Latency win lớn.

#### Connection establishment (cold start)

```
Target: p50 < 500ms, p99 < 1000ms

Breakdown:
  TLS handshake (1.3 with 0-RTT)    : 50-150ms
  WS upgrade (HTTP 101 + 1 RTT)     : 30-100ms
  connection.auth (JWT verify)      : 10-30ms
  conversation.synced (delta replay): 50-500ms (tùy missed events)
```

#### Reconnect (warm, sau network blip)

```
Target: p50 < 300ms, p99 < 800ms

Branched từ cold:
  TLS resumption (session ticket)   : 1 RTT (~30-50ms)
  WS upgrade + auth                 : ~100ms
  conversation.synced               : variable (10ms nếu 0 events, 500ms+ nếu nhiều)
```

#### Other event classes

| Event class | Target | Lý do |
|---|---|---|
| `typing.updated` | < 200ms p99 | Forgiving — UX vẫn OK với 200ms |
| `reaction.added` | < 200ms p99 | Same |
| `presence.updated` | < 2s p99 | Eventual consistency OK. User không cần biết bạn online ngay |
| `message.read_by` | < 5s p99 | Batched, không interactive |
| `notification.created` | p50 < 300ms, p99 < 1s | Mention/task: tương đối quan trọng nhưng không hot-path |

---

### 2.3 Throughput Estimation

#### Send rate (C→S `message.send`)

```
Sustained avg = 0.02 msg/s/concurrent
Peak burst    = 0.05 msg/s/concurrent  (3x spike: lunch, breaking news, work standup)

Sustained C→S send = 1M × 0.02 = 20,000 msg/sec
Peak C→S send      = 1M × 0.05 = 50,000 msg/sec
```

> **Lưu ý**: 0.02 sustained là rate **tính trên CONCURRENT** (đã filter idle). Không phải trên DAU.

#### Fan-out factor (broadcast to subscribers)

```
Mix breakdown (assumption):
  - 40% messages trong DM (2 user)         → fan-out 1 (sender không cần broadcast)
  - 40% messages trong small channel (5-15 user) → avg fan-out 8
  - 15% messages trong medium channel (50-200 user) → avg fan-out 100
  - 5%  messages trong large channel (500-5000 user) → avg fan-out 2000

Weighted avg fan-out = 0.4×1 + 0.4×8 + 0.15×100 + 0.05×2000
                     = 0.4 + 3.2 + 15 + 100
                     = 118.6 ≈ ~120 fan-out/message average

Effective design fan-out (capped, vì large channel có "lazy delivery" tier):
  Hot subscribers (recently active): cap fan-out at ~50 per message
  Lazy delivery cho rest qua background sync
```

> **Đây là điểm shape architecture lớn nhất.** Large channel fan-out = thundering herd. Cần lazy delivery + tiered subscriber list (sẽ chi tiết ở Step 4 và Step 5).

Sử dụng **effective fan-out = 50** cho throughput estimation:

#### S→C broadcast rate

```
message.created peak:
  = 50K send/sec × 50 fan-out = 2,500,000 events/sec broadcast
  → too high. Phải dùng lazy delivery cho non-active subscribers.

  Realistic broadcast (chỉ active subscribers):
  active_subscribers_per_msg = 50 × 0.3 (active fraction) = 15
  = 50K × 15 = 750,000 events/sec
```

#### Other S→C event classes (peak)

| Event | Derivation | Peak ev/s |
|---|---|---|
| `message.created` | 50K send × 15 active fan-out | 750K |
| `typing.updated` | 300K active × 0.05 typing × 8 group fan-out | 120K |
| `reaction.added` | 1M × 0.02 react × 8 fan-out × 0.3 active | 48K |
| `message.read_by` | 1M × 0.01 read × 8 × 0.3 active | 24K |
| `notification.created` | 1M × 0.02 notif (1:1, không fan-out) | 20K |
| `presence.updated` | 1M × 0.005 (low rate, throttled) × 30 buddy fan-out | 150K |
| `message.ack` | 50K send (1:1 cho sender) | 50K |
| `connection.pong` | 1M × 0.05 (every 20s) | 50K |
| **Total S→C peak** | | **~1.2M events/sec** |
| **Total C→S peak** | sends + reactions + typing + ping | **~150K events/sec** |

> **Numbers shape architecture**:
> - 1.2M events/sec broadcast = bắt buộc multi-node + cross-node fan-out qua Kafka.
> - Single Go server max ~50-100K events/sec realistic. → cần ít nhất 12-24 nodes.

---

### 2.4 Message Payload + Bandwidth

#### Payload sizes

```
message text content:
  avg = 300 B
  p99 = 2 KB
  max = 10 KB (hard limit, enforced server-side)

Media (image, file, video):
  KHÔNG inline trong WS frame.
  Upload qua REST → S3/CDN. WS frame chỉ chứa URL ref + thumbnail metadata (~500 B).

Full WS event sizes (text content + JSON envelope + sender metadata):
  message.created    : 500 B avg
  message.ack        : 100 B
  typing.updated     : 100 B
  reaction.added     : 150 B
  presence.updated   : 80 B
  message.read_by    : 120 B
  notification.created: 300 B
```

#### Peak outbound bandwidth (server → all clients)

```
Per-event bytes/sec:
  message.created    : 750K × 500 B = 375 MB/s
  typing.updated     : 120K × 100 B = 12 MB/s
  reaction.added     : 48K × 150 B  = 7.2 MB/s
  presence.updated   : 150K × 80 B  = 12 MB/s
  message.read_by    : 24K × 120 B  = 2.9 MB/s
  notification       : 20K × 300 B  = 6 MB/s
  ack/pong/other     : ~5 MB/s

Total peak outbound: ~420 MB/s = 3.4 Gbps
```

#### Per-user bandwidth

```
Active user peak receive: ~50 events/sec × 400 B avg = 20 KB/s = 160 kbps
Idle user receive: ~5 events/sec × 200 B = 1 KB/s = 8 kbps

Mobile-friendly: ✓ (160 kbps là 1/6 video call quality)
```

#### Daily bandwidth (sustained avg ≈ 30% of peak)

```
Sustained outbound = ~125 MB/s
Daily = 125 MB/s × 86400 = 10.8 TB/day outbound
Monthly = ~325 TB/month

AWS data transfer cost:
  CloudFront: $0.085/GB
  Direct EC2 internet: $0.09/GB
  
325 TB × $0.09 = ~$29,000/month  ← data transfer alone

Mitigation:
  - WS compression (permessage-deflate): 30-60% reduction cho text → ~$12-20K/month
  - CloudFront edge: chỉ giúp HTTP, không giúp WS (WS không cache được)
  - Regional clustering: route user → nearest region
```

> **Cost flag**: bandwidth là cost driver lớn ở scale này. Phải có item trong cost model.

---

### 2.5 Connection Memory Budget — quyết định kiến trúc

```
Per-connection state (Go + gorilla/websocket):
  goroutine stack (read + write loop)  : 2 × 8 KB = 16 KB
  read buffer                          : 4 KB
  write buffer                         : 4 KB
  connection metadata struct           : 2 KB
    {user_id, session_id, channel_subs[], last_pong, presence_state, ...}
  TLS state (if direct TLS termination): 2 KB
  Outbound message channel buffer (16 msg × 500 B): 8 KB
                                         ─────────
  Total per-connection                 : ~36 KB

For 1M connections:
  1M × 36 KB = 36 GB RAM total cho connection state
```

#### Implication: PHẢI multi-node

```
Single AWS r6i.16xlarge: 512 GB RAM
  Nominally 1 box hold 1M conns. NHƯNG:
  - 1 box failure = 1M user disconnect đồng thời = thundering herd reconnect
  - Deploy/restart = 1M reconnect = ngừng dịch vụ
  - Vertical scaling reach limit nhanh
  → KHÔNG ACCEPTABLE.

Sizing: nhiều node nhỏ thay vì 1 node lớn.

Recommended layout:
  30 WS nodes × 35K connections/node
  = 1.05M capacity (5% headroom)
  
  Per node:
    35K × 36 KB = 1.26 GB conn state
    + app working memory (handlers, caches): 2 GB
    + OS + kernel buffers: 1 GB
    = ~4.3 GB
    
  Recommended instance: c6i.large (2 vCPU, 4 GB) hoặc m6i.large (2 vCPU, 8 GB safer)
  
  Failure blast radius: 1 node down = 35K user reconnect (3.5% of total)
                                    → manageable
```

> **Đây là số driver chính của Step 4 (hard parts)**: connection state đã distributed → subscription state cho room cũng phải distributed → đó là vấn đề khó nhất của T09.

---

### 2.6 Storage Growth

```
Daily message volume:
  DAU × avg_msg/user/day
  = 2.5M × 30 msg/user/day  (Slack benchmark ~30, conservative)
  = 75M messages/day

Storage per message row (PostgreSQL):
  content (text, avg 300 B)      : 300 B
  row metadata                    : 200 B
    {id uuid, sender_id, channel_id, parent_id, created_at, edited_at,
     status, version, deleted_at, search_vector}
  ────────────────────────────────────
  Per-row raw                     : 500 B
  Index overhead (40%)            : 200 B
  Per-row total                   : 700 B

Daily storage growth:
  75M × 700 B = 52.5 GB/day  (raw + indexes)
  + WAL (replication, PITR): 1.5x = 78 GB/day

Yearly:
  78 × 365 = 28.5 TB/year

5-year projection: ~140 TB
```

#### Implication: bắt buộc partitioning (T16)

```
Single PostgreSQL table tới 30 TB = unmanageable:
  - VACUUM time → giờ
  - Backup time → giờ
  - Index size → không fit RAM cache
  
Strategy:
  Partition by created_at (monthly), within partition shard by channel_id.
  Old partitions → cold storage (S3 + restore on demand).
  Hot working set: current month + previous month = ~5 TB → SSD-friendly.
```

---

### 2.7 Availability Targets

```
Overall SLA: 99.95% (21.6 min/month downtime)

Per-feature SLA (degraded mode aware):
```

| Feature class | SLA | Downtime/month | Degradation |
|---|---|---|---|
| Send/receive message | 99.99% | 4.3 min | Core. Nếu down = sản phẩm không hoạt động |
| Real-time typing/presence | 99.9% | 43 min | UX nice-to-have. Down = chat vẫn hoạt động |
| Read receipts | 99.5% | 3.6h | Có thể batch retry |
| Search | 99.5% | 3.6h | History fetch vẫn dùng được |
| Notifications (push) | 99.9% | 43 min | Alternative: email fallback |
| Task management | 99.9% | 43 min | Có thể eventual consistency |

#### Degraded modes

| Failure | Degradation strategy |
|---|---|
| WS down, REST up | Client fallback: send qua `POST /messages` (REST), poll `/messages?since=X` để nhận. Latency 5-10s thay vì 100ms |
| Kafka cluster down | WS server chuyển sang local in-memory pub/sub (chỉ trong cùng node). Cross-node fan-out tạm dừng. Reconcile khi Kafka recover |
| 1 WS node down | Clients reconnect tới node khác (load balancer). `conversation.synced` replay events đã miss qua Kafka offset. ~30s gián đoạn |
| PostgreSQL primary down | Read-only mode cho ~30s (failover). Send queue ở Kafka, retry sau khi replica promoted |
| Redis presence down | Presence stale (last-known state). Không impact send/receive |

---

### 2.8 Numbers shape architecture (input cho Step 4)

| # | Number | Architectural implication |
|---|---|---|
| 1 | 1M concurrent connections | **Multi-node WS tier bắt buộc** (~30 nodes × 35K conn/node) |
| 2 | 36 GB total connection RAM | Phải distribute. Single-box vertical scaling không khả thi |
| 3 | 750K message broadcasts/sec peak | **Cross-node fan-out qua Kafka bắt buộc**. Single-node max ~50-100K ev/s. Cần consumer group per-node |
| 4 | Effective fan-out 50 per message (large channel up to 5000) | **Lazy delivery + tiered subscribers cần thiết**. Naive fan-out = thundering herd |
| 5 | 3.4 Gbps peak outbound, ~325 TB/month | Cost concern. WS compression bắt buộc (`permessage-deflate`) |
| 6 | 78 GB/day DB write | **PostgreSQL partitioning** (T16). Async write qua Kafka để giữ latency budget |
| 7 | DB write KHÔNG trong critical path | Architecture: WS server → Kafka publish → ACK. Persist worker consume Kafka → DB |
| 8 | Connection state distributed | → **Subscription state cũng phải distributed** (chính là hard part #1 của T09) |

### 2.9 Assumptions to revisit

Mọi assumption phải re-validate sau load test (T44, T45):

1. DAU/MAU = 50% — measure thực tế sau khi launch
2. Peak concurrent factor = 40% of DAU — đo qua connection metrics
3. Send rate 0.02 msg/s/concurrent sustained — đo qua Kafka producer rate
4. Avg fan-out 50 effective — đo qua channel size distribution
5. Connection memory 36 KB — profile qua pprof
6. Avg message size 300 B — đo qua DB column stats

---

## Step 3 — Constraints & Assumptions

### 3.1 Strategic Posture

**Design target: 1M concurrent** (per spec). **Test capacity: ~3K concurrent** (VPS budget).
**Decision**: design cho 1M kể cả khi không test được full scale.

Lý do:
- Architecture cho 3K ≠ architecture cho 1M (multi-node, Kafka fan-out, sharded state). Design cho 3K → rewrite khi cần 1M.
- Architecture cho 1M là **superset** — chạy được trên 3K (overkill nhưng OK), chạy được lên 1M không đổi code.
- T09 risk 🔴 ghi rõ: "Sai design = rewrite toàn bộ". Mục tiêu T09 = không phải rewrite khi scale.

### 3.2 Stack Constraints (per spec)

| Component | Choice | Rationale |
|---|---|---|
| Backend language | Go | Per project setup. Goroutine model phù hợp WS, ecosystem ổn |
| WS library | `gorilla/websocket` | Idiomatic Go, đủ cho 3K test + 35K/node production. Switch sang `gobwas/ws` chỉ khi load test (T45) chứng minh RAM bound |
| Message broker | Kafka | Per spec T11. Hỗ trợ cross-node fan-out + replay (T23 reconnect-resync) |
| Primary DB | PostgreSQL | Per spec. Partitioning cho message storage (T16) |
| Cache + presence | Redis | Per spec T39, T40. Cluster mode khi production |
| Auth | Auth0 (RS256 + JWKS) | Per T02. WS connection auth qua first-frame JWT |

### 3.3 Resource Constraints

| Phase | Environment | Capacity |
|---|---|---|
| **Dev local** | Docker Compose (T38) | 1 box, 100-500 conn |
| **Sprint test** | 2-3 VPS (3 GB RAM, 200 Mbps mỗi cái) | ~3K conn total, **multi-node verifiable** |
| **Load test (T44, T45)** | Same VPS cluster, scale-down scenarios | Test invariants, không test full load |
| **Production** | AWS (T49, T50) | 1M concurrent target. ~30 WS nodes + DB + Redis + Kafka MSK |

### 3.4 Code-Architecture Invariants

Quy tắc bắt buộc — vi phạm = đã rewrite trá hình:

1. **Code chạy 1 node hay 30 node phải GIỐNG NHAU.** Không có `if single_node { ... } else { ... }`.
2. **Node count = config var**, không hardcode.
3. **Mọi node treat itself = 1 trong N**. Không có "primary" hay "leader" node trong WS tier.
4. **Cross-node communication = qua Kafka và Redis**, không direct node-to-node.
5. **Subscription state = distributed** (Redis hoặc consistent-hashing), không in-memory single source.

### 3.5 Validation Strategy (verify 1M trên 3K)

Test **invariants** ở scale nhỏ → induction → confidence ở scale lớn.

| Invariant | Verify cách nào ở scale nhỏ |
|---|---|
| Cross-node fan-out hoạt động | 2 nodes × 100 conn. Send node A. Verify nhận node B trong <200ms |
| Subscription routing distributed correctly | 3 nodes. Subscribe channel X từ node 1. Verify message từ node 2/3 routed về node 1 |
| Reconnect-resync correctness | Restart 1 node. Client reconnect, verify không miss event qua `conversation.synced` |
| Failure isolation | Kill 1/3 nodes. Verify 2/3 conn unaffected, 1/3 reconnect tới node khác <30s |
| Memory linearity | Profile per-conn @ 100, 500, 1K, 3K. Verify linear (không leak, không quadratic) |
| Latency stability | p50/p99 @ 100/1K/3K. Verify p99 không spike đột ngột |
| Kafka backpressure | Stress publish rate. Verify producer không block/crash khi consumer lag |

Rule: invariant đúng @ N=2 và N=3 → induction → đúng @ N=30.

### 3.6 Risks Accepted (không verify được tới production)

| Risk | Lý do | Mitigation |
|---|---|---|
| Kafka broker overload @ 1M | VPS không đủ throughput stress | Pre-deploy: Kafka benchmark riêng. Production: monitor lag (T52) + alert |
| Hot channel fan-out (5000+ subscribers) | Test bed không tạo nổi | Math defendable trên paper. Production: lazy delivery + load shed |
| GC pause @ 1M conn | 3K không tạo đủ pressure | pprof profile @ 3K, tune GOGC. Monitor production |
| DB write throughput @ 75M msg/day | RDS spec trên paper | pgbench synthetic load. Monitor write latency production |
| Multi-region network latency | Single-region VPS không simulate | Design region-aware. Multi-region staged rollout |

### 3.7 Solo Dev Constraints

- **1 dev, 4-6h/ngày** — feature scope phải minimal viable
- **TDD required** (per memory) — multiplier 1.5x time/feature
- **Sprint 3 budget**: 6 SP (T09=2 + T10=3 + T11=1) ≈ 9-10 ngày thực tế
- **Không có ops team** — automation + observability từ ngày 1 (T47, T48, T52)
- **No on-call** — degraded mode + alerting > heroics

---

## Step 4 — Hard Parts

> ⏳ Pending. Predicted:
> - Connection state at 1M scale (memory, FD limits)
> - Fan-out trong group lớn
> - Cross-node routing (subscription state shared via Kafka/Redis)
> - Reconnect-resync ordering
> - Hot rooms (room có nhiều subscribers cùng lúc)

---

## Step 5 — Design Options + Tradeoffs

> ⏳ Pending.

---

## Step 6 — Failure Modes

> ⏳ Pending.

---

## Step 7 — Diagrams + Final Doc

> ⏳ Pending.
