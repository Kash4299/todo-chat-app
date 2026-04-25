# System Architecture — KashFlow
> Version: 1.0 | Date: 2026-04-25 | Scale target: 1M concurrent users

---

## PHẦN 1 — PHÂN TÍCH RỦI RO LỚN NHẤT

### Rủi ro #1: `BroadcastToRoom` là in-memory — hệ thống KHÔNG thể scale ngang

Đây là rủi ro lớn nhất, không phải vì khó, mà vì **nó ẩn trong code hiện tại và sẽ chỉ lộ ra khi deploy nhiều hơn 1 instance**.

**Vấn đề cụ thể trong code hiện tại:**

```
chat_handler.go → service.BroadcastToRoom(taskID, msg)
                       ↓
               In-memory room map
               [taskID] → [conn1, conn2, conn3]
```

Khi chạy 1 instance: hoạt động bình thường.

Khi chạy 2+ WS instances (bắt buộc để đạt 1M users):

```
Client A → WS Server #1 (room map: [conn_A])
Client B → WS Server #2 (room map: [conn_B])

Client A gửi tin nhắn →
  WS Server #1 broadcast → chỉ Client A nhận được
  WS Server #2 KHÔNG biết → Client B KHÔNG nhận được tin nhắn
```

**3 vấn đề phụ đi kèm:**

| Vấn đề | Vị trí trong code | Hậu quả |
|---|---|---|
| `sarama.SyncProducer` | `pkg/kafka/kafka.go:23` | Block goroutine mỗi lần publish → bottleneck nặng ở high load |
| `sarama.Consumer` thay vì `ConsumerGroup` | `pkg/kafka/kafka.go:36` | Kafka partitions không được chia đều giữa các WS instances → một instance xử lý tất cả |
| `task` = `room` conflation | schema, `chat_handler.go:64` | Không có channel độc lập (general, random) ngoài task → giới hạn product |

---

## PHẦN 2 — HAI PHƯƠNG ÁN GIẢM THIỂU RỦI RO

---

### PHƯƠNG ÁN A: Technical Spike (PoC 5 ngày)

**Ý tưởng**: Trước khi viết production code, bỏ ra 5 ngày xây một **throwaway prototype** để validate toàn bộ critical path.

**Những gì cần validate trong PoC:**

```
[Client x 10,000] → WS Server #1 → Kafka AsyncProducer
                                          ↓
                                    chat.messages topic
                                    (partition by channel_id)
                                          ↓
                         ┌────────────────┴────────────────┐
                    WS Server #1                      WS Server #2
                  ConsumerGroup #A                ConsumerGroup #A
                  (check local map)             (check local map)
                         ↓                               ↓
              Deliver to connected clients    Deliver to connected clients
```

**Metrics cần đo:**
- RAM per WebSocket connection (target: < 10KB/conn → 10GB cho 1M)
- Goroutine count per 1K connections
- Message end-to-end latency: Client A → Kafka → Client B (target: < 100ms)
- Kafka consumer lag khi tải cao

**Pros:**
- Kill unknown unknowns **trước khi** viết 27 SP production code
- 5 ngày đầu tư tránh được 2-3 tuần rewrite sau
- Biết chính xác điểm bottleneck là ở đâu

**Cons:**
- Code throwaway, không dùng được cho production
- 5 ngày "không tiến" về feature

**Khi nào nên chọn A**: Nếu anh chưa tự tin về WebSocket at scale trong Go và muốn học bằng cách thử nghiệm thật sự.

---

### PHƯƠNG ÁN B: Incremental Load Gate (tích hợp vào sprint)

**Ý tưởng**: Không làm throwaway. Thay vào đó, **fix 3 vấn đề kỹ thuật ngay trong Sprint 3**, rồi thêm load test checkpoint vào cuối mỗi sprint.

**Các thay đổi cần làm ngay trong Sprint 3:**

```
THAY ĐỔI 1: SyncProducer → AsyncProducer
pkg/kafka/kafka.go

THAY ĐỔI 2: Consumer → ConsumerGroup
pkg/kafka/kafka.go
→ Mỗi WS instance = 1 consumer trong group "ws-gateway"
→ Kafka tự động phân chia partitions

THAY ĐỔI 3: BroadcastToRoom → Publish to Kafka
chat_handler.go / chat_service.go
→ Nhận message từ client → publish Kafka
→ Kafka consumer nhận lại → check local conn map → deliver
```

**Load test gates theo sprint:**

| Sprint | Load Target | Metric cần pass |
|---|---|---|
| Sprint 3 | 1.000 connections | Latency < 50ms, 0 message loss |
| Sprint 5 | 10.000 connections | Latency < 80ms, Memory < 500MB/instance |
| Sprint 7 | 100.000 connections | Latency < 100ms, Kafka lag < 1s |
| Sprint 9 | 1.000.000 connections | Latency < 150ms, System stable 5 phút |

**Pros:**
- Không mất time cho throwaway code
- Architecture vừa xây vừa validate liên tục
- Phát hiện vấn đề sớm nhưng vẫn tiến về feature

**Cons:**
- Nếu design WebSocket + Kafka sai ở Sprint 3, phải refactor trong lúc đang build features
- Không có "safe playground" để thử nghiệm

**Khi nào nên chọn B**: Nếu anh đã có hình dung rõ về kiến trúc Kafka fan-out và muốn tiến nhanh.

---

### KHUYẾN NGHỊ CỦA TÔI

**Chọn Option A (Spike 5 ngày)**, vì:

1. WebSocket ở quy mô 1M connections là thách thức mà rất ít developer đã làm — không nên underestimate
2. 5 ngày bây giờ rẻ hơn 3 tuần rewrite ở Sprint 6 khi đã có 40+ SP code trên đầu
3. PoC sẽ cho anh **con số thật** để tính toán infrastructure cost (RAM, instance count) — thứ bạn cần để deploy lên AWS

---

## PHẦN 3 — SYSTEM ARCHITECTURE DIAGRAM

### Diagram 1: Tổng quan hệ thống (System Overview)

```mermaid
graph TB
    subgraph CLIENT["🌐 Client Layer"]
        BROWSER["Web Browser\nReact + WS Client"]
    end

    subgraph EDGE["⚖️ Edge / Load Balancing"]
        ALB["AWS ALB\nHTTP / HTTPS\nREST API traffic"]
        WSALB["WebSocket LB\nIP Hash Routing\nSticky by user_id"]
    end

    subgraph SERVICES["⚙️ Service Layer — Go"]
        API["API Service\n─────────────\nAuth (JWT)\nTasks\nWorkspace\nSearch\nUsers"]
        WS1["WS Gateway #1\n─────────────\nconn pool\nlocal room map\nKafka producer\nKafka consumer"]
        WS2["WS Gateway #2\n─────────────\nconn pool\nlocal room map\nKafka producer\nKafka consumer"]
        WSN["WS Gateway #N\n─────────────\n(horizontal scale)"]
        NOTIF["Notification\nService\n─────────────\nKafka consumer\nemail fan-out"]
    end

    subgraph KAFKA["📨 Message Broker — Apache Kafka"]
        K1["chat.messages\npartition: channel_id"]
        K2["chat.presence\npartition: user_id"]
        K3["chat.notifications\npartition: user_id"]
        K4["task.events\npartition: workspace_id"]
    end

    subgraph DATA["🗄️ Data Layer"]
        PG_W[("PostgreSQL Primary\n── Write ──\nAll INSERT/UPDATE")]
        PG_R[("PostgreSQL Replica\n── Read ──\nChat history\nSearch\nTask list")]
        REDIS[("Redis Cluster\n─────────────\nsession cache\npresence TTL\nrate limiting\nmessage dedup")]
    end

    subgraph EXTERNAL["🔌 External Services"]
        SES["AWS SES\nEmail"]
        S3["AWS S3\nFiles (Future)"]
        SPOTIFY["Spotify API\n(Future)"]
    end

    subgraph OBS["📊 Observability"]
        PROM["Prometheus\nMetrics scrape"]
        GRAF["Grafana\nDashboards"]
        CW["CloudWatch\nLogs + Alerts"]
    end

    %% Client connections
    BROWSER -->|"REST HTTP/S"| ALB
    BROWSER -->|"WSS (TLS)"| WSALB

    %% Edge to Services
    ALB --> API
    WSALB --> WS1
    WSALB --> WS2
    WSALB --> WSN

    %% API → Data
    API -->|"Write"| PG_W
    API -->|"Read"| PG_R
    API -->|"Session / Cache"| REDIS
    API -->|"Publish"| K4

    %% WS → Kafka (publish)
    WS1 -->|"Publish msg"| K1
    WS1 -->|"Publish presence"| K2
    WS2 -->|"Publish msg"| K1
    WS2 -->|"Publish presence"| K2
    WSN -->|"Publish"| K1

    %% Kafka → WS (subscribe — ConsumerGroup)
    K1 -->|"Subscribe\nConsumerGroup: ws-gateway"| WS1
    K1 -->|"Subscribe"| WS2
    K1 -->|"Subscribe"| WSN
    K3 -->|"Subscribe"| WS1
    K3 -->|"Subscribe"| WS2

    %% WS → Data
    WS1 -->|"Persist message"| PG_W
    WS1 -->|"Presence TTL"| REDIS
    WS2 -->|"Persist message"| PG_W
    WS2 -->|"Presence TTL"| REDIS

    %% Kafka → Notification
    K3 -->|"Subscribe"| NOTIF
    K4 -->|"Subscribe"| NOTIF
    NOTIF -->|"Send email"| SES
    NOTIF -->|"Persist"| PG_W

    %% DB Replication
    PG_W -.->|"Streaming replication"| PG_R

    %% Observability
    API -.->|"metrics"| PROM
    WS1 -.->|"metrics"| PROM
    WS2 -.->|"metrics"| PROM
    NOTIF -.->|"metrics"| PROM
    PROM --> GRAF
    PG_W -.->|"slow query logs"| CW
    KAFKA -.->|"consumer lag"| CW
```

---

### Diagram 2: WebSocket Message Fan-out Flow (Critical Path)

> Đây là luồng quan trọng nhất — giải thích cách 1M users nhận được message cùng lúc.

```mermaid
sequenceDiagram
    participant CA as Client A
    participant WS1 as WS Gateway #1
    participant PG as PostgreSQL
    participant KF as Kafka (chat.messages)
    participant WS2 as WS Gateway #2
    participant WS3 as WS Gateway #3
    participant CB as Client B (on WS2)
    participant CC as Client C (on WS3)

    Note over CA,CC: Client A gửi message vào channel "general"

    CA->>WS1: WS frame: {channel_id, content}
    
    par Persist & Publish (parallel)
        WS1->>PG: INSERT INTO messages (async write)
        WS1->>KF: AsyncProducer.Input() <- msg\n(key = channel_id, no blocking)
    end

    Note over KF: Kafka phân phối đến tất cả\nConsumerGroup "ws-gateway"

    par Fan-out to all WS servers
        KF->>WS1: ConsumerGroup receives msg
        KF->>WS2: ConsumerGroup receives msg
        KF->>WS3: ConsumerGroup receives msg
    end

    Note over WS1: Check local conn map:\nchannel "general" → [conn_A]\nDeliver to conn_A (echo back)
    WS1-->>CA: Delivered ✓ (with server timestamp + msg_id)

    Note over WS2: Check local conn map:\nchannel "general" → [conn_B]\nDeliver to conn_B
    WS2-->>CB: Delivered ✓

    Note over WS3: Check local conn map:\nchannel "general" → [conn_C]\nDeliver to conn_C
    WS3-->>CC: Delivered ✓
```

---

### Diagram 3: Authentication Flow

```mermaid
sequenceDiagram
    participant C as Client
    participant A0 as Auth0
    participant API as API Service
    participant PG as PostgreSQL

    Note over C,PG: Login with Email/Password (Auth0 Database Connection)
    C->>A0: Universal Login (email + password)
    A0-->>C: id_token/access_token (sub=auth0|...)
    C->>API: GET /users/me (Bearer access_token)
    API->>API: Verify RS256 JWT via JWKS (issuer + audience)
    API->>PG: SELECT user_identity WHERE provider_subject = sub
    alt identity exists
        PG-->>API: identity -> user_id
        API-->>C: user profile
    else first login
        API->>PG: create users + user_identities row
        API-->>C: user profile
    end

    Note over C,PG: Login with Google (same email as existing account)
    C->>A0: Universal Login (Google)
    A0-->>C: id_token/access_token (sub=google-oauth2|...)
    C->>API: GET /users/me (Bearer access_token)
    API->>API: Verify JWT + read {sub,email,email_verified}
    API->>PG: SELECT user_identity WHERE provider_subject = sub
    alt google identity already linked
        API-->>C: existing user profile
    else link candidate
        API->>PG: find existing user by same verified email
        API->>API: enforce linking policy (email_verified + recent auth + explicit consent/step-up)
        API->>PG: INSERT user_identities(user_id, provider_subject='google-oauth2|...')
        API-->>C: linked profile (single internal user)
    end

    Note over C,PG: Authenticated Request
    C->>API: GET /tasks (Authorization: Bearer {access_token})
    API->>API: Verify JWT signature (JWKS)
    API->>PG: SELECT tasks WHERE workspace_id = ?
    API-->>C: tasks[]

    Note over C,PG: WebSocket Auth
    C->>API: WSS /ws/channels/{id}?token={access_token}
    API->>API: Verify JWT in query param
    API->>PG: resolve identity (sub -> user_id)
    API-->>C: 101 Switching Protocols
```

---

### Diagram 4: Data Schema — Identity Linking Ready

> Auth0 là nguồn identity duy nhất. App DB không lưu password hash.  
> Để hỗ trợ "đăng ký email/password trước, login Google sau" cần bảng `user_identities`.

```mermaid
erDiagram
    USERS {
        uuid id PK
        string primary_auth0_user_id UK
        string email UK
        string display_name
        string avatar_url
        string status_text
        boolean is_active
        timestamp created_at
        timestamp updated_at
    }

    USER_IDENTITIES {
        uuid id PK
        uuid user_id FK
        string provider "AUTH0_DB|GOOGLE_OAUTH2|..."
        string provider_subject UK "full Auth0 sub"
        string email_at_link_time
        boolean is_primary
        timestamp linked_at
        timestamp created_at
    }

    WORKSPACES {
        uuid id PK
        string name
        string slug UK
        uuid owner_id FK
        timestamp created_at
    }

    WORKSPACE_MEMBERS {
        uuid workspace_id FK
        uuid user_id FK
        string role "ADMIN|MEMBER|GUEST"
        timestamp joined_at
    }

    CHANNELS {
        uuid id PK
        uuid workspace_id FK
        string name
        string type "PUBLIC|PRIVATE|DM"
        uuid created_by FK
        timestamp created_at
    }

    CHANNEL_MEMBERS {
        uuid channel_id FK
        uuid user_id FK
        timestamp last_read_at
        timestamp joined_at
    }

    TASKS {
        uuid id PK
        uuid workspace_id FK
        uuid channel_id FK "optional: task thread"
        string title
        text description
        string status "TODO|IN_PROGRESS|REVIEW|DONE"
        string priority "LOW|MEDIUM|HIGH|URGENT"
        uuid created_by FK
        uuid assignee_id FK "nullable"
        uuid parent_task_id FK "nullable: subtask"
        timestamp due_date
        timestamp created_at
        timestamp updated_at
    }

    MESSAGES {
        uuid id PK
        uuid channel_id FK
        uuid user_id FK
        string message_type "TEXT|SYSTEM"
        text content
        uuid reply_to_id FK "nullable: thread"
        jsonb attachments "[] future"
        tsvector search_vector "GENERATED"
        timestamp created_at
    }

    NOTIFICATIONS {
        uuid id PK
        uuid user_id FK
        string type "MENTION|TASK_ASSIGNED|DUE_DATE"
        jsonb payload
        boolean is_read
        timestamp created_at
    }

    USERS ||--o{ USER_IDENTITIES : "has login identities"
    USERS ||--o{ WORKSPACE_MEMBERS : "belongs to"
    WORKSPACES ||--o{ WORKSPACE_MEMBERS : "has"
    WORKSPACES ||--o{ CHANNELS : "has"
    WORKSPACES ||--o{ TASKS : "has"
    CHANNELS ||--o{ CHANNEL_MEMBERS : "has"
    CHANNELS ||--o{ MESSAGES : "contains"
    USERS ||--o{ MESSAGES : "sends"
    USERS ||--o{ TASKS : "creates"
    TASKS ||--o{ TASKS : "parent-child"
```

### Linking Policy (Bắt buộc)

1. Không auto-link nếu `email_verified = false`.
2. Không link chỉ dựa vào email; cần bước xác nhận từ user (explicit consent) hoặc step-up auth.
3. `provider_subject` là duy nhất toàn hệ thống.
4. Mỗi login đều resolve theo `provider_subject` trước, chỉ fallback email khi chưa có identity row.
5. Sau khi link thành công, mọi provider của user phải map về cùng một `users.id`.

---

## PHẦN 4 — REDIS KEY DESIGN

> Redis là lớp "fast state" — mọi thứ cần đọc/ghi nhanh mà không cần persist lâu dài.

| Key Pattern | Value | TTL | Mục đích |
|---|---|---|---|
| `idmap:{provider_subject}` | `user_id` | 1 giờ | Cache mapping `sub` → internal user to giảm DB read |
| `presence:{user_id}` | `1` | 30 giây | Online/offline status |
| `ratelimit:{user_id}:{endpoint}` | counter | 1 phút | Rate limiting sliding window |
| `dedup:{message_id}` | `1` | 5 phút | Tránh Kafka at-least-once duplicate |
| `channel:unread:{user_id}:{channel_id}` | count | No TTL | Unread message counter |

---

## PHẦN 5 — KAFKA TOPIC DESIGN

| Topic | Partition Key | Retention | Consumer Groups |
|---|---|---|---|
| `chat.messages` | `channel_id` | 7 ngày | `ws-gateway` (all WS instances) |
| `chat.presence` | `user_id` | 1 phút | `ws-gateway` |
| `chat.notifications` | `user_id` | 24 giờ | `ws-gateway`, `notification-service` |
| `task.events` | `workspace_id` | 7 ngày | `notification-service` |

**Lý do partition by `channel_id` cho `chat.messages`:**
- Messages trong cùng channel đảm bảo ordering (FIFO)
- Kafka partition = đơn vị parallelism → nhiều channel = nhiều partitions xử lý song song
- Số partitions recommend: `expected_peak_channels_active_concurrent × 2` (bắt đầu với 32)

---

## PHẦN 6 — INFRASTRUCTURE (Local → AWS)

### Local Development (hiện tại)
```
docker-compose.yaml
├── go-api          (port 8080)
├── postgres        (port 5432)
├── redis           (port 6379)
├── kafka           (port 9092)
└── zookeeper       (port 2181)
```

### AWS Production Target
```
AWS VPC (private subnets)
├── ECS Fargate
│   ├── api-service          (2 tasks minimum)
│   ├── ws-gateway           (N tasks, scale by CPU/conn count)
│   └── notification-service (1-2 tasks)
├── RDS PostgreSQL           (db.r6g.large + 1 read replica)
├── ElastiCache Redis        (cache.r6g.large, cluster mode)
├── MSK (Managed Kafka)      (kafka.m5.large × 3 brokers)
└── ALB                      (HTTP + WebSocket listeners)
```

---

## PHẦN 7 — ĐIỂM CẦN LÀM NGAY (Technical Debt trong code hiện tại)

> Những vấn đề này cần fix TRƯỚC khi build thêm feature, không phải sau.

| # | Vấn đề | File | Fix cần làm |
|---|---|---|---|
| 1 | `SyncProducer` block goroutine | `pkg/kafka/kafka.go:23` | Chuyển sang `AsyncProducer` |
| 2 | `sarama.Consumer` thay vì `ConsumerGroup` | `pkg/kafka/kafka.go:36` | Dùng `sarama.NewConsumerGroup` |
| 3 | `BroadcastToRoom` in-memory | `chat_service.go` | Route qua Kafka fan-out |
| 4 | Identity model chưa hỗ trợ account linking | `users` schema + auth service | Thêm `user_identities`, policy link an toàn theo email verified |
| 5 | Thiếu `workspace` + `channel` concept | DB schema | Viết migration mới trước Sprint 2 |
| 6 | `task` conflated với `chat room` | Toàn bộ codebase | Tách `channel` thành entity riêng |
