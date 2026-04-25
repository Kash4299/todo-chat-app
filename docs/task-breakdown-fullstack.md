# Full-Stack Task Breakdown — KashFlow Chat + Task App
> Generated: 2026-04-25 | Stack: Go + React 18 + TypeScript | Target: 1M concurrent WebSocket users
> Coverage: Backend (BE) T01–T60 + Frontend (FE) FE01–FE45 across 13 sprints

---

## Legend

| Symbol | Meaning |
|---|---|
| SP | Story Point — 1 SP ≈ 1 focused workday (4–6 hrs). With TDD overhead, multiply actual calendar days by 1.5x. |
| BE | Backend task (Go service, database, infrastructure) |
| FE | Frontend task (React/TypeScript, UI components, state management) |
| Risk: High | Can block the project or require re-architecture if wrong |
| Risk: Medium | Needs careful attention; likely more complex than it looks |
| Risk: Low | Straightforward, well-understood implementation |
| (done) | Task is already completed as of 2026-04-25 |

**Total estimated effort: ~202 SP (BE: ~104 SP + FE: ~98 SP). With TDD x1.5 = ~300 real workdays.**

---

## Sprint 1 — Auth Foundation + FE Scaffold
**Goal: Running local dev environment, auth APIs complete, FE project boots and connects to backend.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T01 | BE | Design DB schema (users, workspaces, roles, sessions) via migration 002 | Define durable relational model for identity, tenancy, and permissions before feature growth. | 2 | — | High | Schema mistakes are expensive to fix later. Done via migration files. (done) |
| T02 | BE | Integrate Auth0 JWT validation — RS256 + JWKS + issuer/audience checks | Make API trust boundary explicit and standards-compliant with Auth0-issued tokens only. | 2 | T01 | Medium | No local HS256 signing in API; Auth0 is source of truth. |
| T03 | BE | Build identity-sync path from Auth0 token claims (`sub`, `email`, `name`) to internal `users` | Bridge external identity to internal domain user model for consistent authorization and ownership. | 1 | T01, T02 | Low | Create/update internal profile on first authenticated request. |
| T04 | BE | Implement account-linking design for same email (Auth0 DB + Google) | Prevent duplicate accounts by mapping multiple providers to one internal user safely. | 2 | T02, T03 | High | Add `user_identities` table + secure link policy (verified email + explicit consent/step-up). |
| T38 | BE | Set up Docker Compose (Go, PostgreSQL, Redis, Kafka, Zookeeper) | Provide reproducible local infra to test integration and scale-related behavior early. | 2 | — | Medium | Use `depends_on` with health checks to control startup order. |
| FE01 | FE | Scaffold Vite + React 18 + TypeScript project with strict tsconfig | Create strict frontend baseline to reduce runtime bugs and enforce type contracts. | 1 | — | Low | Use `npm create vite@latest`. Enable `strictNullChecks`, `noImplicitAny`. |
| FE02 | FE | Configure Tailwind CSS and shadcn/ui component library | Establish reusable UI foundation and tokenized styling for rapid feature delivery. | 1 | FE01 | Low | Run `shadcn-ui init`. Set up design tokens (colors, radius). |
| FE03 | FE | Establish folder structure: `features/`, `components/`, `hooks/`, `lib/`, `stores/` | Enforce modular boundaries so feature teams can scale without tangled imports. | 1 | FE01 | Low | Barrel exports per folder. No cross-feature imports without shared layer. |
| FE04 | FE | Build Axios HTTP client instance with base URL and default headers | Centralize transport concerns (base URL, headers, error mapping) for consistency. | 1 | FE01 | Low | Base URL from `VITE_API_URL` env var. Export typed `apiClient`. |
| FE05 | FE | Implement auth interceptor — attach Bearer token; auto-refresh on 401 response | Keep user session continuity while preventing retry storms and token race conditions. | 2 | FE04 | Medium | Queue concurrent 401 requests while refresh is in flight. Avoid infinite retry loop. |

**Sprint 1 Total: BE 8 SP + FE 6 SP = 14 SP**

---

## Sprint 2 — Workspace BE + Auth UI + Protected Routes
**Goal: Workspace APIs done. User can register, log in, and see a protected dashboard.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T05 | BE | Build Workspace CRUD API — create, update, get, list workspaces | Implement tenant core APIs with strict ownership and membership boundaries. | 2 | T01 | Medium | Multi-tenant isolation: every query scoped by workspace_id. |
| T06 | BE | Build Member Invitation system — generate invite token, POST /invitations/accept | Allow controlled workspace onboarding with expiring invites and auditable acceptance flow. | 2 | T01, T05 | Medium | Token expires 24h. Email delivery via SMTP (placeholder for now). |
| T07 | BE | Implement RBAC middleware — Admin / Member / Guest roles | Enforce role permissions consistently before business actions hit service logic. | 2 | T01, T05 | Medium | Middleware pattern. Permission matrix stored in config, not hardcoded per route. |
| T08 | BE | Build GET /users/me and PATCH /users/me (avatar URL, display name, status) | Provide self-service user profile management separate from auth identity provider data. | 1 | T01 | Low | Avatar is a URL string for now; no file upload yet. |
| T47 | BE | Set up GitHub Actions CI pipeline — lint → test → build | Automate quality gate so every push is validated for style, correctness, and buildability. | 1 | T01 | Low | `golangci-lint`, `go test -race`, `go build`. Fail fast on any step. |
| T48 | BE | Write multi-stage Dockerfile for Go service | Produce lightweight, secure, reproducible runtime image for deployment parity. | 1 | T38 | Low | Builder stage + distroless final image. Binary only, no source in image. |
| FE06 | FE | Set up TanStack Query (React Query) — QueryClient, QueryClientProvider, devtools | Standardize frontend data-fetching cache lifecycle and mutation handling. | 1 | FE01 | Low | Default staleTime 30s. Enable React Query devtools in dev mode only. |
| FE07 | FE | Build Zustand auth store — user state, tokens, login/logout actions, persist to localStorage | Centralize client auth state to keep route guards and API calls deterministic. | 2 | FE04 | Medium | Persist `accessToken` + `refreshToken` via `zustand/middleware/persist`. Clear on logout. |
| FE08 | FE | Build Auth0 login UX — Universal Login redirect/callback, store Auth0 tokens, redirect to /dashboard | Complete auth entrypoint UX for both email/password and social login via Auth0. | 2 | FE05, FE07 | Low | Support both Database Connection + Google button from Auth0 hosted page. |
| FE09 | FE | Build account-linking UX for same email identities | Provide explicit, safe user flow for linking identities that share email without takeover risk. | 2 | FE05, FE07, FE08 | Medium | Show explicit consent/step-up flow when Google login matches an existing email account. |
| FE10 | FE | Implement protected route guard — redirect to /login if no valid token | Ensure only authenticated sessions access app shell and protected resources. | 1 | FE07 | Low | HOC or wrapper component. Check token expiry client-side before redirect. |
| FE11 | FE | Build Profile page — GET /users/me, edit display name and avatar URL | Deliver first user-facing settings surface for identity/profile updates. | 2 | FE06, FE10 | Low | Optimistic update via TanStack Query `useMutation`. |

**Sprint 2 Total: BE 9 SP + FE 10 SP = 19 SP**

---

## Sprint 3 — WebSocket Architecture Spike + Kafka Design
**Goal: WebSocket server running, Kafka topics designed. This is the most critical design sprint — mistakes here require full rewrites.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T09 | BE | Design WebSocket architecture — connection manager, room routing, fan-out strategy | Draw and review the diagram BEFORE writing code. Wrong design = full rewrite. | 2 | T01 | High | Draw and review the diagram BEFORE writing code. Wrong design = full rewrite. |
| T10 | BE | Implement WebSocket server and goroutine-safe connection pool manager | Use sharded sync.Map to avoid lock contention. Watch for goroutine and memory leaks. | 3 | T09 | High | Use sharded sync.Map to avoid lock contention. Watch for goroutine and memory leaks. |
| T11 | BE | Design Kafka topic schema — `chat.messages`, `chat.presence`, `chat.notifications` | Topic design is hard to change in production. Decide partition count and retention now. | 1 | T09 | High | Topic design is hard to change in production. Decide partition count and retention now. |
| T23 | BE | Implement WebSocket heartbeat (ping/pong) and reconnect state sync | Client sends `last_message_id` on reconnect so server can replay missed events. | 2 | T10 | Medium | Client sends `last_message_id` on reconnect so server can replay missed events. |
| T39 | BE | Implement Redis client — session cache, rate-limit counters, presence TTL keys | Redis Cluster mode path must be viable. Use `go-redis/v9`. | 2 | T38 | Low | Redis Cluster mode path must be viable. Use `go-redis/v9`. |
| FE12 | FE | Build `useWebSocket` hook — connect, disconnect, auto-reconnect with exponential backoff | Reconnect on close codes 1001/1006. Send `last_message_id` on reconnect. Heartbeat ping every 25s. | 3 | FE07 | High | Reconnect on close codes 1001/1006. Send `last_message_id` on reconnect. Heartbeat ping every 25s. |
| FE13 | FE | Build WebSocket message dispatcher — route incoming WS events to Zustand stores by type | Event types: `message.new`, `typing`, `presence.update`, `reaction.add`, `notification`. | 2 | FE12 | Medium | Event types: `message.new`, `typing`, `presence.update`, `reaction.add`, `notification`. |

**Sprint 3 Total: BE 10 SP + FE 5 SP = 15 SP**

---

## Sprint 4 — Kafka Fan-out + Chat Core APIs + FE Chat Shell
**Goal: Kafka producers and consumers wired. Chat messages flow end-to-end across multiple server instances.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T12 | BE | Implement Kafka producer — publish `chat.messages` events asynchronously | Async producer with error callback. Do not block the HTTP handler. | 2 | T11 | Low | Async producer with error callback. Do not block the HTTP handler. |
| T13 | BE | Implement Kafka consumer — consume events and deliver to WebSocket clients | Consumer group per WebSocket server instance. At-least-once delivery + dedup by message_id. | 2 | T11, T12 | Medium | Consumer group per WebSocket server instance. At-least-once delivery + dedup by message_id. |
| T14 | BE | Build Channel management API — POST /channels, GET /channels, join/leave/list members | Public and private channel types. Member list with cursor pagination. | 2 | T01, T05 | Low | Public and private channel types. Member list with cursor pagination. |
| T15 | BE | Build Direct Message (DM) API — create DM channel between two users | DM is a private channel with exactly 2 participants. Idempotent create. | 2 | T14 | Low | DM is a private channel with exactly 2 participants. Idempotent create. |
| T16 | BE | Implement message persistence layer — INSERT to PostgreSQL on every send | Consider partitioning `messages` table by `channel_id` range for high write throughput. | 2 | T01 | Medium | Consider partitioning `messages` table by `channel_id` range for high write throughput. |
| T43 | BE | Document horizontal scaling architecture — WebSocket + Kafka fan-out diagram | Diagram: N WebSocket servers each consume all Kafka partitions, deliver locally. | 1 | T09, T11 | Low | Diagram: N WebSocket servers each consume all Kafka partitions, deliver locally. |
| FE14 | FE | Build app shell layout — sidebar + main content area + header, responsive | Use shadcn/ui Sidebar. Sidebar shows workspaces and channels. | 2 | FE10 | Low | Use shadcn/ui Sidebar. Sidebar shows workspaces and channels. |
| FE15 | FE | Build channel list sidebar — GET /channels, active state, unread badge | Real-time unread count updated via WS events. | 2 | FE06, FE14 | Low | Real-time unread count updated via WS events. |
| FE16 | FE | Build message list component — render messages, avatar, timestamp, author name | Virtualize with `@tanstack/react-virtual` for performance at 10K+ messages. | 2 | FE14 | Low | Virtualize with `@tanstack/react-virtual` for performance at 10K+ messages. |

**Sprint 4 Total: BE 11 SP + FE 6 SP = 17 SP**

---

## Sprint 5 — Chat Features BE + Message Input + Real-time FE
**Goal: Typing indicators, presence, read receipts done BE. FE can send and receive messages in real time.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T17 | BE | Build GET /channels/:id/messages with cursor-based pagination | Cursor = `message_id`. Return `next_cursor` in response. Never use offset. | 1 | T16 | Low | Cursor = `message_id`. Return `next_cursor` in response. Never use offset. |
| T18 | BE | Implement typing indicator broadcast — WS event, debounced 2s, not persisted | Broadcast `typing.start` / `typing.stop` events scoped to channel. | 1 | T10 | Low | Broadcast `typing.start` / `typing.stop` events scoped to channel. |
| T19 | BE | Build online/offline presence system — Redis TTL keys refreshed by heartbeat | Key: `presence:{user_id}`, TTL 30s. Heartbeat refreshes TTL. Broadcast on change. | 2 | T10, T39 | Medium | Key: `presence:{user_id}`, TTL 30s. Heartbeat refreshes TTL. Broadcast on change. |
| T20 | BE | Implement message read receipts — mark delivered and read, batch update | Batch update on focus event. Fan-out read status only to channel members, not globally. | 2 | T10, T16 | Medium | Batch update on focus event. Fan-out read status only to channel members, not globally. |
| T40 | BE | Implement Redis presence state management and GET /users/presence endpoint | Key: `presence:{user_id}`, TTL 30s. Heartbeat refresh. | 1 | T39, T19 | Low | Key: `presence:{user_id}`, TTL 30s. Heartbeat refresh. |
| T41 | BE | Implement API rate limiting middleware — sliding window per user and per IP | 60 req/min per user for REST; separate limit for WebSocket message sends. | 1 | T39 | Low | 60 req/min per user for REST; separate limit for WebSocket message sends. |
| FE17 | FE | Build message input — text area, Enter to send, Shift+Enter for newline, character limit | POST to /channels/:id/messages. Optimistic append before server confirm. | 2 | FE15 | Low | POST to /channels/:id/messages. Optimistic append before server confirm. |
| FE18 | FE | Implement infinite scroll / cursor pagination for message history | Load older messages on scroll-to-top. Use TanStack Query `useInfiniteQuery`. | 2 | FE16 | Medium | Load older messages on scroll-to-top. Use TanStack Query `useInfiniteQuery`. |
| FE19 | FE | Build typing indicator UI — show "User is typing…" below input with fade animation | Debounce WS send 400ms. Clear indicator after 3s of no event. | 1 | FE13, FE17 | Low | Debounce WS send 400ms. Clear indicator after 3s of no event. |
| FE20 | FE | Build presence badge — green/grey dot on user avatars, updated via WS events | Update Zustand `presenceStore` on `presence.update` WS event. | 1 | FE13 | Low | Update Zustand `presenceStore` on `presence.update` WS event. |
| FE21 | FE | Build read receipt UI — show "Seen by N" indicator under last read message | Only show for the last message the current user sent. | 1 | FE13 | Low | Only show for the last message the current user sent. |

**Sprint 5 Total: BE 8 SP + FE 7 SP = 15 SP**

---

## Sprint 6 — Chat Polish BE + Reactions + Thread UI
**Goal: Emoji reactions, reply threads, read replica routing. Chat feature-complete on both layers.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T21 | BE | Build Emoji Reaction API — POST/DELETE /messages/:id/reactions, broadcast via WS | Limit to standard emoji set. Upsert by (message_id, user_id, emoji). | 1 | T16, T10 | Low | Limit to standard emoji set. Upsert by (message_id, user_id, emoji). |
| T22 | BE | Build Reply / Thread API — POST /messages/:id/replies, GET /messages/:id/thread | `parent_message_id` on message row. Cache thread reply count in Redis. | 2 | T16 | Medium | `parent_message_id` on message row. Cache thread reply count in Redis. |
| T42 | BE | Set up PostgreSQL read replica routing — writes to primary, reads to replica | Use two connection pools in `pgx`. Route read-only queries to replica. Warn: replication lag. | 2 | T38 | Medium | Use two connection pools in `pgx`. Route read-only queries to replica. Warn: replication lag. |
| FE22 | FE | Build emoji reaction UI — reaction picker (emoji-mart), reaction bar per message | Show count per emoji. Highlight reactions the current user added. Toggle on click. | 2 | FE16, FE13 | Low | Show count per emoji. Highlight reactions the current user added. Toggle on click. |
| FE23 | FE | Build reply / thread UI — inline thread count, slide-in thread panel | Thread panel loads /messages/:id/thread with its own pagination. No nesting beyond 1 level. | 3 | FE16, FE18 | Medium | Thread panel loads /messages/:id/thread with its own pagination. No nesting beyond 1 level. |
| FE24 | FE | Build Workspace management UI — create workspace, invite members, member list | POST /workspaces, POST /invitations. Show pending invite badge. | 2 | FE06, FE10 | Low | POST /workspaces, POST /invitations. Show pending invite badge. |
| FE25 | FE | Build channel management UI — create channel (public/private), join/leave, settings panel | Modal form with shadcn/ui Dialog. Channel privacy toggle. | 2 | FE15 | Low | Modal form with shadcn/ui Dialog. Channel privacy toggle. |

**Sprint 6 Total: BE 5 SP + FE 9 SP = 14 SP**

---

## Sprint 7 — Task Management BE + Kanban Board FE
**Goal: Task CRUD and status workflow done BE. Kanban board renders and updates tasks FE.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T24 | BE | Design task schema — tasks, subtasks, priority ENUM, status ENUM, assignee, tags array | Schema must be flexible for custom workflow stages later. Tags as `text[]`. | 1 | T01 | Medium | Schema must be flexible for custom workflow stages later. Tags as `text[]`. |
| T25 | BE | Build Task CRUD API — POST /tasks, GET /tasks/:id, PATCH /tasks/:id, DELETE /tasks/:id | Business rule validation: due_date must be in future; priority required. | 2 | T24 | Low | Business rule validation: due_date must be in future; priority required. |
| T26 | BE | Build Subtask API — create subtask with parent_task_id, GET /tasks/:id/subtasks | Limit depth to 2 levels to avoid recursive complexity. | 1 | T25 | Medium | Limit depth to 2 levels to avoid recursive complexity. |
| T27 | BE | Build Task Assignment API — PUT /tasks/:id/assignee, trigger notification event | Emit Kafka `task.assigned` event on change. Connect to notification pipeline (T53). | 1 | T25 | Low | Emit Kafka `task.assigned` event on change. Connect to notification pipeline (T53). |
| T28 | BE | Build Status Workflow engine — state machine transitions, log each transition | States: Todo → In Progress → Review → Done. Log old/new state with timestamp. | 2 | T25 | Medium | States: Todo → In Progress → Review → Done. Log old/new state with timestamp. |
| T31 | BE | Build Kanban Board API — GET /boards/:id/tasks grouped by status, with position field | `position` float field for drag-drop reorder. Sparse update on reorder. | 2 | T25, T28 | Low | `position` float field for drag-drop reorder. Sparse update on reorder. |
| FE26 | FE | Build Kanban board view — columns per status, task cards with assignee and priority | Use `@dnd-kit/core` for drag-and-drop. Optimistic reorder on drop. | 3 | FE06, FE10 | Medium | Use `@dnd-kit/core` for drag-and-drop. Optimistic reorder on drop. |
| FE27 | FE | Implement drag-and-drop task reorder — PATCH /tasks/:id with new status and position | Rollback optimistic update on API error. Debounce PATCH 300ms during rapid drags. | 2 | FE26 | Medium | Rollback optimistic update on API error. Debounce PATCH 300ms during rapid drags. |
| FE28 | FE | Build Task creation modal — form with title, description, priority, due date, assignee | shadcn/ui Dialog + Form. Assignee picker via GET /workspaces/:id/members. | 2 | FE26 | Low | shadcn/ui Dialog + Form. Assignee picker via GET /workspaces/:id/members. |

**Sprint 7 Total: BE 9 SP + FE 7 SP = 16 SP**

---

## Sprint 8 — Task Features BE + Task Detail + List View FE
**Goal: Task comments, activity log, list view done. FE has full task detail experience.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T29 | BE | Build Task Comments API — POST /tasks/:id/comments, linked to chat channel | Each task auto-creates or links to a dedicated channel. Comments fan-out to channel. | 2 | T25, T14 | Medium | Each task auto-creates or links to a dedicated channel. Comments fan-out to channel. |
| T30 | BE | Build Task Activity Log — event sourcing lite: record every field change with old/new value | Store event_type + old_value + new_value + actor_id + timestamp. Append-only table. | 2 | T25 | Medium | Store event_type + old_value + new_value + actor_id + timestamp. Append-only table. |
| T32 | BE | Build Task List API — GET /tasks with filters (assignee, status, priority, date range) | Query builder approach. Support multi-value filters. Cursor pagination. | 1 | T25 | Low | Query builder approach. Support multi-value filters. Cursor pagination. |
| T44 | BE | Write k6 load test scripts — WebSocket connections + REST API scenarios | Scenarios: 100 → 1K → 10K → 100K → 1M connections. Ramp up in stages. | 2 | T10, T16 | Low | Scenarios: 100 → 1K → 10K → 100K → 1M connections. Ramp up in stages. |
| FE29 | FE | Build Task detail view — full page with description, comments, activity log, subtasks | Side panel or dedicated route. Load comments via GET /tasks/:id/comments. | 3 | FE28 | Medium | Side panel or dedicated route. Load comments via GET /tasks/:id/comments. |
| FE30 | FE | Build Task assignment UI — assignee picker dropdown, show avatar + name, clear button | Filter workspace members. Show current assignee prominently. | 1 | FE28 | Low | Filter workspace members. Show current assignee prominently. |
| FE31 | FE | Build status change UI — status badge as dropdown, PUT /tasks/:id/status on select | Optimistic UI. Reflect new status color immediately before API confirms. | 1 | FE29 | Low | Optimistic UI. Reflect new status color immediately before API confirms. |
| FE32 | FE | Build Task list view — table layout with column sort, filter bar (status, priority, date) | Toggle between kanban and list view. Persist view preference in localStorage. | 3 | FE06 | Low | Toggle between kanban and list view. Persist view preference in localStorage. |
| FE33 | FE | Build subtask UI — expandable subtask list inside task detail, inline add subtask | Checkbox to mark subtask done. Show progress bar (X/Y complete) on parent card. | 2 | FE29 | Low | Checkbox to mark subtask done. Show progress bar (X/Y complete) on parent card. |

**Sprint 8 Total: BE 7 SP + FE 10 SP = 17 SP**

---

## Sprint 9 — Search BE + Search UI FE
**Goal: Full-text search working across messages, tasks, and users. FE search bar integrated.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T33 | BE | Set up PostgreSQL full-text search — `tsvector` column + GIN index on messages and tasks | Add `unaccent` extension. Trigger auto-updates `search_vector` on insert/update. | 2 | T16, T24 | Medium | Add `unaccent` extension. Trigger auto-updates `search_vector` on insert/update. |
| T34 | BE | Build GET /search/messages — keyword search scoped by channel, with term highlight | Use `ts_rank` for ranking. Highlight via `ts_headline`. Cursor pagination. | 2 | T33 | Low | Use `ts_rank` for ranking. Highlight via `ts_headline`. Cursor pagination. |
| T35 | BE | Build GET /search/tasks — title search + multi-filter (status, assignee, date range) | Combine `@@` full-text with regular WHERE filters in one query. | 1 | T33 | Low | Combine `@@` full-text with regular WHERE filters in one query. |
| T36 | BE | Build GET /search/users — ILIKE search on display_name + email within workspace | Limit results to 20. Used for @mention autocomplete as well. | 1 | T01 | Low | Limit results to 20. Used for @mention autocomplete as well. |
| T37 | BE | Implement search result ranking and keyword highlight | Relevance tuning takes time. Budget iterations. Use `ts_rank_cd` for proximity. | 2 | T34, T35 | Medium | Relevance tuning takes time. Budget iterations. Use `ts_rank_cd` for proximity. |
| FE34 | FE | Build search bar component — global command palette (Cmd+K), debounced input | shadcn/ui CommandDialog. Debounce 300ms. Show recent searches from localStorage. | 2 | FE06 | Low | shadcn/ui CommandDialog. Debounce 300ms. Show recent searches from localStorage. |
| FE35 | FE | Build search results panel — tabs for Messages / Tasks / People, highlighted snippets | Highlight matched terms with `<mark>`. Clicking a message result scrolls to it in channel. | 3 | FE34 | Low | Highlight matched terms with `<mark>`. Clicking a message result scrolls to it in channel. |
| FE36 | FE | Build @mention autocomplete — dropdown on `@` key in message input, query /search/users | Keyboard-navigable dropdown. Insert `@username` token into message on select. | 1 | FE17 | Low | Keyboard-navigable dropdown. Insert `@username` token into message on select. |

**Sprint 9 Total: BE 8 SP + FE 6 SP = 14 SP**

---

## Sprint 10 — Load Testing + Performance Optimization (BE)
**Goal: System validated at 1M concurrent connections. Bottlenecks identified and resolved.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T45 | BE | Run k6 load tests + profile bottlenecks — pprof CPU/heap, PostgreSQL EXPLAIN ANALYZE | May uncover deep architectural issues. Allocate full sprint if needed. | 3 | T44, full system | High | May uncover deep architectural issues. Allocate full sprint if needed. |
| T46 | BE | Optimize for 1M concurrent connections — OS tuning (ulimit, TCP buffers), code fixes | `sysctl` tuning, file descriptor limits, connection backpressure, circuit breakers. | 3 | T45 | High | `sysctl` tuning, file descriptor limits, connection backpressure, circuit breakers. |

**Sprint 10 Total: BE 6 SP + FE 0 SP = 6 SP**

---

## Sprint 11 — AWS Deployment + FE Build Pipeline
**Goal: App live on AWS. CI/CD pipeline deploys both backend and frontend on merge.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T49 | BE | Provision AWS account, IAM roles, VPC, subnets, security groups via Terraform | Principle of least privilege. Terraform state in S3 with DynamoDB lock. | 2 | — | Medium | Principle of least privilege. Terraform state in S3 with DynamoDB lock. |
| T50 | BE | Provision AWS services — ECS Fargate, RDS PostgreSQL, ElastiCache Redis, MSK Kafka | MSK config is complex and expensive. Start with smallest instances. Scale after load tests. | 3 | T49 | High | MSK config is complex and expensive. Start with smallest instances. Scale after load tests. |
| T51 | BE | Set up GitHub Actions CD — build → push ECR → rolling deploy to ECS | Blue/green deployment via ECS CodeDeploy for zero downtime. | 2 | T47, T48, T50 | Medium | Blue/green deployment via ECS CodeDeploy for zero downtime. |
| T52 | BE | Set up CloudWatch monitoring + alarms — CPU, memory, WebSocket conn count, Kafka lag | Alert on Kafka consumer lag > 1000 and WebSocket error rate > 1%. Avoid alert fatigue. | 2 | T50 | Medium | Alert on Kafka consumer lag > 1000 and WebSocket error rate > 1%. Avoid alert fatigue. |
| FE37 | FE | Configure Vite production build — env vars, tree-shaking, chunk splitting, asset hashing | `VITE_API_URL` and `VITE_WS_URL` from CI environment. | 1 | FE01 | Low | `VITE_API_URL` and `VITE_WS_URL` from CI environment. |
| FE38 | FE | Set up GitHub Actions FE pipeline — type-check → lint → test → build → deploy to S3+CloudFront | Invalidate CloudFront on deploy. Cache static assets with long max-age. | 2 | FE37, T49 | Low | Invalidate CloudFront on deploy. Cache static assets with long max-age. |

**Sprint 11 Total: BE 9 SP + FE 3 SP = 12 SP**

---

## Sprint 12 — Notifications BE + Notification UI FE
**Goal: In-app notification system working. Users notified of mentions and task assignments.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T53 | BE | Design notification schema + Kafka `chat.notifications` topic | Notification types: `mention`, `task_assigned`, `task_due`, `reaction`. | 1 | T11 | Low | Notification types: `mention`, `task_assigned`, `task_due`, `reaction`. |
| T54 | BE | Build notification service — consume Kafka, persist, fan-out to WebSocket | Batch fan-out. Rate limit per user to prevent notification storm in large groups. | 2 | T53 | Medium | Batch fan-out. Rate limit per user to prevent notification storm in large groups. |
| T55 | BE | Build @mention detection — parse `@username` in message body, emit notification event | Detect on message save. Resolve username to user_id at write time. | 1 | T16, T53 | Low | Detect on message save. Resolve username to user_id at write time. |
| T56 | BE | Build GET /notifications — list unread, PATCH /notifications/:id/read, DELETE all | Cursor pagination. Filter by notification type. | 1 | T54 | Low | Cursor pagination. Filter by notification type. |
| T57 | BE | Build email notification service for task assigned and @mention events | Use AWS SES. Queue delivery with retry. Respect user notification preferences. | 2 | T54 | Medium | Use AWS SES. Queue delivery with retry. Respect user notification preferences. |
| FE39 | FE | Build notification bell icon + unread badge in header | Increment badge count on `notification` WS event. Zero on panel open. | 1 | FE13 | Low | Increment badge count on `notification` WS event. Zero on panel open. |
| FE40 | FE | Build notification panel — list notifications, mark read on click, mark all read button | GET /notifications with infinite scroll. Clicking a message notification jumps to it. | 2 | FE39, FE06 | Low | GET /notifications with infinite scroll. Clicking a message notification jumps to it. |
| FE41 | FE | Build notification preference settings — toggle email/in-app per notification type | PATCH /users/me/notification-preferences. Persist to backend, not just localStorage. | 1 | FE40 | Low | PATCH /users/me/notification-preferences. Persist to backend, not just localStorage. |

**Sprint 12 Total: BE 7 SP + FE 4 SP = 11 SP**

---

## Sprint 13 — Integrations (Stretch Goal)
**Goal: Google OAuth, Spotify widget, outgoing webhooks. All are nice-to-have, not blocking launch.**

| ID | Layer | Task Name | Description | SP | Dependencies | Risk | Notes |
|---|---|---|---|---|---|---|---|
| T58 | BE | Implement Google OAuth 2.0 login — callback handler, upsert user, issue JWT | Add `provider` and `provider_id` columns to users. Handle existing email conflicts. | 2 | T02 | Medium | Add `provider` and `provider_id` columns to users. Handle existing email conflicts. |
| T59 | BE | Build Spotify integration — link account, display now-playing on user profile | Spotify access token expires every 1h; implement refresh. Poll every 30s for now-playing. | 3 | T01 | Medium | Spotify access token expires every 1h; implement refresh. Poll every 30s for now-playing. |
| T60 | BE | Build outgoing Webhook system — register URL, sign payload with HMAC, retry with backoff | Validate target URL (SSRF: whitelist HTTPS only). Exponential backoff on failure. | 2 | T25, T14 | Medium | Validate target URL (SSRF: whitelist HTTPS only). Exponential backoff on failure. |
| FE42 | FE | Build Google OAuth login button — redirect to /auth/google, handle callback token | Use a single sign-in button on the Login page. No popup; redirect flow only. | 1 | FE08, T58 | Low | Use a single sign-in button on the Login page. No popup; redirect flow only. |
| FE43 | FE | Build Spotify widget on profile page — show album art, track name, artist, link | Poll GET /users/:id/spotify-status every 30s. Show "Not playing" when idle. | 2 | FE11, T59 | Low | Poll GET /users/:id/spotify-status every 30s. Show "Not playing" when idle. |
| FE44 | FE | Build Webhook management UI — list webhooks, create/delete, show delivery logs | Table view with status badge (active / failed). Show last N delivery attempts. | 2 | FE06, T60 | Low | Table view with status badge (active / failed). Show last N delivery attempts. |
| FE45 | FE | Build Notification email preference toggle UI — per-event toggle connected to PATCH /users/me | Reuse the settings panel from FE41. Group by category (Chat, Tasks, System). | 1 | FE41 | Low | Reuse the settings panel from FE41. Group by category (Chat, Tasks, System). |

**Sprint 13 Total: BE 7 SP + FE 6 SP = 13 SP**

---

## Summary Table — Total SP by Layer and Sprint

| Sprint | Focus | BE SP | FE SP | Sprint Total |
|---|---|---|---|---|
| Sprint 1 | Auth Foundation + FE Scaffold | 8 | 6 | 14 |
| Sprint 2 | Workspace BE + Auth UI + Routes | 9 | 10 | 19 |
| Sprint 3 | WebSocket Architecture + Kafka Design | 10 | 5 | 15 |
| Sprint 4 | Kafka Fan-out + Chat Core + FE Shell | 11 | 6 | 17 |
| Sprint 5 | Chat Features + Message Input + Real-time | 8 | 7 | 15 |
| Sprint 6 | Chat Polish + Reactions + Thread UI | 5 | 9 | 14 |
| Sprint 7 | Task Management BE + Kanban Board FE | 9 | 7 | 16 |
| Sprint 8 | Task Features BE + Task Detail + List FE | 7 | 10 | 17 |
| Sprint 9 | Search BE + Search UI FE | 8 | 6 | 14 |
| Sprint 10 | Load Testing + Optimization (BE only) | 6 | 0 | 6 |
| Sprint 11 | AWS Deployment + FE Build Pipeline | 9 | 3 | 12 |
| Sprint 12 | Notifications BE + Notification UI FE | 7 | 4 | 11 |
| Sprint 13 | Integrations (Stretch Goal) | 7 | 6 | 13 |
| **TOTAL** | | **104 SP** | **79 SP** | **183 SP** |

> Note: Sprint 13 FE tasks FE42–FE45 add 10 SP to FE total. The 79 SP figure above covers FE01–FE41 (core).
> Including all stretch FE tasks: **FE total = 89 SP. Grand total = 193 SP.**

---

## Task ID Reference Index

### Done (as of 2026-04-25)
- T01 — DB schema migration 002
- T02 — Auth0 JWT validation baseline (RS256/JWKS)
- T03 — initial identity sync path
- T04 — account-linking design baseline

### Backend Tasks (T05–T60)
| Range | Epic |
|---|---|
| T05–T08 | Auth & Workspace |
| T09–T23 | Real-time Chat System |
| T24–T32 | Task Management |
| T33–T37 | Search |
| T38–T46 | Infrastructure & Scalability |
| T47–T52 | DevOps & CI/CD |
| T53–T57 | Notifications |
| T58–T60 | Third-party Integrations |

### Frontend Tasks (FE01–FE45)
| Range | Area |
|---|---|
| FE01–FE03 | Project scaffold and structure |
| FE04–FE05 | HTTP client and auth interceptor |
| FE06–FE07 | TanStack Query and Zustand setup |
| FE08–FE11 | Auth pages and protected routes |
| FE12–FE13 | WebSocket client and event dispatcher |
| FE14–FE16 | App shell and chat layout |
| FE17–FE21 | Chat messaging and real-time features |
| FE22–FE25 | Chat polish — reactions, threads, workspace/channel UI |
| FE26–FE28 | Kanban board and task creation |
| FE29–FE33 | Task detail, list view, subtasks |
| FE34–FE36 | Search and @mention autocomplete |
| FE37–FE38 | Production build and deployment pipeline |
| FE39–FE41 | Notification bell, panel, preferences |
| FE42–FE45 | Integrations — OAuth, Spotify, Webhooks |

---

> **Critical path warning**: Sprint 3 (WebSocket + Kafka design) is the highest-risk point. If the Kafka fan-out architecture is wrong, every subsequent sprint that touches real-time features will need rework. Do not rush T09–T11. Budget time for a written design review before writing code.
