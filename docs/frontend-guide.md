# Frontend Developer Guide — KashFlow
> **Audience**: AI code agent building the React/TypeScript frontend  
> **Backend**: Go + PostgreSQL + Kafka + WebSocket  
> **Date**: 2026-04-25 | Base URL (local): `http://localhost:8080`

---

## 1. Tech Stack & Project Setup

### Recommended Stack

| Concern | Library | Version |
|---|---|---|
| Framework | React | 18.x |
| Language | TypeScript | 5.x |
| Build tool | Vite | 5.x |
| Routing | React Router | 6.x |
| Server state | TanStack Query | 5.x |
| Client state | Zustand | 4.x |
| Styling | Tailwind CSS | 3.x |
| Components | shadcn/ui (Radix UI based) | latest |
| HTTP client | Axios | 1.x |
| Forms | React Hook Form + Zod | latest |
| Icons | Lucide React | latest |
| Drag & drop | @dnd-kit/core + @dnd-kit/sortable | latest |
| Virtualization | @tanstack/react-virtual | 3.x |
| Date utils | date-fns | 3.x |

### Project Scaffold

```bash
npm create vite@latest kashflow-web -- --template react-ts
cd kashflow-web
npm install axios react-router-dom @tanstack/react-query zustand
npm install react-hook-form @hookform/resolvers zod
npm install @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
npm install @tanstack/react-virtual date-fns lucide-react clsx tailwind-merge
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
# shadcn/ui setup
npx shadcn-ui@latest init
```

### Folder Structure

```
src/
├── api/              # Axios instance + all API call functions
│   ├── axios.ts      # Configured Axios instance with interceptors
│   ├── auth.ts       # Auth API calls (login, register, refresh, logout)
│   ├── user.ts       # User API calls
│   ├── task.ts       # Task API calls
│   ├── channel.ts    # Channel API calls
│   ├── message.ts    # Message API calls
│   └── notification.ts
├── components/       # Reusable UI components
│   ├── ui/           # shadcn/ui components (auto-generated)
│   ├── auth/         # Login form, Register form
│   ├── chat/         # MessageList, MessageInput, ChannelSidebar, etc.
│   ├── task/         # KanbanBoard, TaskCard, TaskModal, etc.
│   ├── layout/       # AppLayout, Sidebar, Header, NotificationBell
│   └── common/       # Avatar, Badge, EmptyState, LoadingSpinner, etc.
├── hooks/            # Custom hooks
│   ├── useAuth.ts    # Auth state from Zustand
│   ├── useWebSocket.ts  # WebSocket connection manager
│   └── useInfiniteMessages.ts
├── pages/            # Route-level components
│   ├── LoginPage.tsx
│   ├── RegisterPage.tsx
│   ├── WorkspacePage.tsx   # Main app page
│   └── ProfilePage.tsx
├── stores/           # Zustand stores
│   ├── authStore.ts
│   ├── wsStore.ts    # WebSocket messages + state
│   └── uiStore.ts    # Sidebar open/close, selected channel, etc.
├── types/            # TypeScript interfaces (all derived from Go models)
│   └── index.ts
├── lib/              # Utilities
│   ├── utils.ts      # cn(), formatDate(), etc.
│   └── constants.ts  # API_BASE_URL, WS_BASE_URL, etc.
├── router/           # React Router setup
│   └── index.tsx
└── main.tsx
```

---

## 2. TypeScript Interfaces

> These map 1:1 to the Go models in `internal/model/`. Use these types for ALL API calls and state.

```typescript
// src/types/index.ts

export interface User {
  id: string;           // UUID
  email: string;
  display_name: string;
  avatar_url?: string;
  status_text: string;
  is_active: boolean;
  created_at: string;   // ISO 8601
  updated_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
}

export interface Workspace {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  created_at: string;
  updated_at: string;
}

export interface WorkspaceMember {
  workspace_id: string;
  user_id: string;
  role: 'ADMIN' | 'MEMBER' | 'GUEST';
  joined_at: string;
}

export interface Channel {
  id: string;
  workspace_id: string;
  name: string;
  type: 'PUBLIC' | 'PRIVATE' | 'DM';
  created_by: string;
  created_at: string;
}

export interface ChannelMember {
  channel_id: string;
  user_id: string;
  last_read_at: string;
  joined_at: string;
}

export type TaskStatus   = 'TODO' | 'IN_PROGRESS' | 'REVIEW' | 'DONE';
export type TaskPriority = 'LOW' | 'MEDIUM' | 'HIGH' | 'URGENT';

export interface Task {
  id: string;
  workspace_id: string;
  channel_id?: string;
  title: string;
  description: string;
  status: TaskStatus;
  priority: TaskPriority;
  assignee_id?: string;
  created_by: string;
  parent_task_id?: string;
  position: number;
  due_date?: string;
  created_at: string;
  updated_at: string;
}

export type ActivityType =
  | 'CREATED' | 'STATUS_CHANGED' | 'PRIORITY_CHANGED'
  | 'ASSIGNED' | 'UNASSIGNED' | 'TITLE_CHANGED'
  | 'DESCRIPTION_CHANGED' | 'DUE_DATE_CHANGED';

export interface TaskActivity {
  id: string;
  task_id: string;
  user_id: string;
  activity_type: ActivityType;
  old_value?: string;
  new_value?: string;
  created_at: string;
}

export type MessageType = 'TEXT' | 'SYSTEM';

export interface Message {
  id: string;
  channel_id: string;
  user_id: string;
  message_type: MessageType;
  content: string;
  reply_to_id?: string;
  is_edited: boolean;
  created_at: string;
  updated_at: string;
}

export interface Reaction {
  message_id: string;
  user_id: string;
  emoji: string;
  created_at: string;
}

export type NotificationType = 'MENTION' | 'TASK_ASSIGNED' | 'TASK_DUE' | 'CHANNEL_INVITE';

export interface Notification {
  id: string;
  user_id: string;
  type: NotificationType;
  payload: Record<string, unknown>;
  is_read: boolean;
  created_at: string;
}

// API response wrappers
export interface ApiError {
  error: string;
}

export interface PaginatedMessages {
  messages: Message[];
  next_cursor?: string; // message ID to use as beforeID in next request
}
```

---

## 3. Authentication Flow

### Token Storage Strategy

```typescript
// Store in localStorage (persisted) via Zustand persist middleware
// NEVER store in cookies without httpOnly flag
// NEVER put tokens in URL query params (except for WS upgrade — backend accepts this)

const AUTH_KEYS = {
  accessToken:  'kf_access_token',
  refreshToken: 'kf_refresh_token',
  user:         'kf_user',
} as const;
```

### Zustand Auth Store

```typescript
// src/stores/authStore.ts
import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { User, TokenPair } from '../types';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  refreshToken: string | null;
  isAuthenticated: boolean;
  setAuth: (user: User, tokens: TokenPair) => void;
  setAccessToken: (token: string) => void;
  clearAuth: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      setAuth: (user, tokens) => set({
        user,
        accessToken: tokens.access_token,
        refreshToken: tokens.refresh_token,
        isAuthenticated: true,
      }),
      setAccessToken: (token) => set({ accessToken: token }),
      clearAuth: () => set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false }),
    }),
    { name: 'kf-auth' }
  )
);
```

### Axios Instance with Auto-Refresh

```typescript
// src/api/axios.ts
import axios from 'axios';
import { useAuthStore } from '../stores/authStore';
import { API_BASE_URL } from '../lib/constants';

export const api = axios.create({
  baseURL: API_BASE_URL, // 'http://localhost:8080/api/v1'
  headers: { 'Content-Type': 'application/json' },
});

// Attach access token to every request
api.interceptors.request.use((config) => {
  const token = useAuthStore.getState().accessToken;
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

// Auto-refresh on 401
let isRefreshing = false;
let failedQueue: Array<{ resolve: (t: string) => void; reject: (e: unknown) => void }> = [];

api.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config;
    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error);
    }

    if (isRefreshing) {
      return new Promise((resolve, reject) => {
        failedQueue.push({ resolve, reject });
      }).then((token) => {
        original.headers.Authorization = `Bearer ${token}`;
        return api(original);
      });
    }

    original._retry = true;
    isRefreshing = true;

    try {
      const refreshToken = useAuthStore.getState().refreshToken;
      if (!refreshToken) throw new Error('No refresh token');

      const { data } = await axios.post(`${API_BASE_URL}/auth/refresh`, {
        refresh_token: refreshToken,
      });

      const newAccess: string = data.access_token;
      useAuthStore.getState().setAccessToken(newAccess);

      failedQueue.forEach((p) => p.resolve(newAccess));
      failedQueue = [];

      original.headers.Authorization = `Bearer ${newAccess}`;
      return api(original);
    } catch (refreshError) {
      failedQueue.forEach((p) => p.reject(refreshError));
      failedQueue = [];
      useAuthStore.getState().clearAuth();
      window.location.href = '/login';
      return Promise.reject(refreshError);
    } finally {
      isRefreshing = false;
    }
  }
);
```

### Protected Route Component

```typescript
// src/router/index.tsx
import { Navigate, Outlet } from 'react-router-dom';
import { useAuthStore } from '../stores/authStore';

export function ProtectedRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />;
}

export function PublicOnlyRoute() {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  return isAuthenticated ? <Navigate to="/app" replace /> : <Outlet />;
}
```

---

## 4. API Reference

> **Base URL (local):** `http://localhost:8080/api/v1`  
> **Auth:** All protected endpoints require `Authorization: Bearer <access_token>` header  
> **Content-Type:** `application/json` for all requests

### 4.1 Auth Endpoints (Public)

---

#### `POST /auth/register`

Register a new user account.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "display_name": "Nguyen Van A"
}
```

**Validation rules:**
- `email`: required, valid email format
- `password`: required, minimum 8 characters
- `display_name`: required

**Response `201`:**
```json
{
  "user": {
    "id": "550e8400-e29b-41d4-a716-446655440000",
    "email": "user@example.com",
    "display_name": "Nguyen Van A",
    "avatar_url": "",
    "status_text": "",
    "is_active": true,
    "created_at": "2026-04-25T10:00:00Z",
    "updated_at": "2026-04-25T10:00:00Z"
  },
  "tokens": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "a3f1b2c4d5e6f7a8b9c0d1e2f3a4b5c6..."
  }
}
```

**Error responses:**
| Code | Body | When |
|---|---|---|
| 400 | `{"error": "Key: 'email' Error:Field validation..."}` | Validation failed |
| 409 | `{"error": "email already registered"}` | Duplicate email |
| 500 | `{"error": "internal server error"}` | Server error |

---

#### `POST /auth/login`

Authenticate with email and password.

**Request body:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response `200`:**
```json
{
  "user": { /* same User object as register */ },
  "tokens": {
    "access_token": "eyJhbGci...",
    "refresh_token": "a3f1b2c4..."
  }
}
```

**Error responses:**
| Code | Body | When |
|---|---|---|
| 400 | `{"error": "..."}` | Validation failed |
| 401 | `{"error": "invalid credentials"}` | Wrong email or password |

---

#### `POST /auth/refresh`

Exchange a refresh token for a new access token + refresh token pair.  
**Important:** Each refresh token can only be used ONCE (rotation). Store the new tokens immediately.

**Request body:**
```json
{
  "refresh_token": "a3f1b2c4d5e6f7a8b9c0d1e2f3a4b5c6..."
}
```

**Response `200`:**
```json
{
  "access_token": "eyJhbGci...",
  "refresh_token": "x7y8z9a1b2c3..."
}
```

**Error responses:**
| Code | Body | When |
|---|---|---|
| 401 | `{"error": "invalid or expired refresh token"}` | Token not found or expired |
| 401 | `{"error": "refresh token expired"}` | TTL exceeded (7 days) |

---

### 4.2 Auth Endpoints (Protected)

---

#### `POST /auth/logout`

Invalidate the current refresh token.

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "refresh_token": "a3f1b2c4..."
}
```

**Response `200`:**
```json
{ "message": "logged out" }
```

---

### 4.3 User Endpoints (Protected)

---

#### `GET /users/me`

Get the currently authenticated user's profile.

**Headers:** `Authorization: Bearer <access_token>`

**Response `200`:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "email": "user@example.com",
  "display_name": "Nguyen Van A",
  "avatar_url": "https://...",
  "status_text": "Working on something cool",
  "is_active": true,
  "created_at": "2026-04-25T10:00:00Z",
  "updated_at": "2026-04-25T10:00:00Z"
}
```

**Error responses:**
| Code | Body | When |
|---|---|---|
| 401 | `{"error": "missing token"}` | No token |
| 401 | `{"error": "invalid token"}` | Invalid/expired token |
| 404 | `{"error": "user not found"}` | User deleted |

---

### 4.4 Task Endpoints (Protected)

---

#### `POST /tasks`

Create a new task.

**Headers:** `Authorization: Bearer <access_token>`

**Request body:**
```json
{
  "workspace_id": "uuid-of-workspace",
  "title": "Fix login bug",
  "description": "The login form doesn't validate email properly",
  "priority": "HIGH",
  "due_date": "2026-05-01T17:00:00Z"
}
```

**Field rules:**
- `workspace_id`: required, valid UUID
- `title`: required
- `description`: optional (defaults to `""`)
- `priority`: optional, one of `LOW | MEDIUM | HIGH | URGENT` (defaults to `MEDIUM`)
- `due_date`: optional, RFC3339 format

**Response `201`:**
```json
{
  "id": "uuid",
  "workspace_id": "uuid-of-workspace",
  "channel_id": null,
  "title": "Fix login bug",
  "description": "The login form doesn't validate email properly",
  "status": "TODO",
  "priority": "HIGH",
  "assignee_id": null,
  "created_by": "uuid-of-creator",
  "parent_task_id": null,
  "position": 0,
  "due_date": "2026-05-01T17:00:00Z",
  "created_at": "2026-04-25T10:00:00Z",
  "updated_at": "2026-04-25T10:00:00Z"
}
```

---

#### `GET /tasks/:id`

Get a single task by ID.

**Headers:** `Authorization: Bearer <access_token>`

**Response `200`:** Task object (same as create response)

**Error responses:**
| Code | Body |
|---|---|
| 400 | `{"error": "invalid task id"}` |
| 404 | `{"error": "task not found"}` |

---

### 4.5 WebSocket — Real-time Chat

---

#### Connection

```
wss://localhost:8080/api/v1/ws/channels/:channelID?token=<access_token>
```

> **Why query param?** WebSocket upgrade requests cannot set custom headers in browsers. The backend reads `?token=` for WebSocket auth specifically.

**Channel ID:** UUID of the channel to join. The user must be a member of the channel (enforced server-side). If not a member, the server closes the connection immediately after upgrade.

**Reconnection strategy:** Use exponential backoff: 1s → 2s → 4s → 8s → max 30s. Reset counter on successful connection.

---

#### Client → Server: Send a message

Send a JSON text frame:

```json
{
  "message_type": "TEXT",
  "content": "Hello everyone!"
}
```

| Field | Type | Required | Values |
|---|---|---|---|
| `message_type` | string | yes | `"TEXT"` |
| `content` | string | yes | Non-empty string |

**Important:** The server does NOT send an acknowledgement frame. The message will be echoed back to all connected clients (including the sender) via the broadcast mechanism.

---

#### Server → Client: Receive a message

The server broadcasts a full `Message` object as a JSON text frame when any client sends a message:

```json
{
  "id": "uuid-of-message",
  "channel_id": "uuid-of-channel",
  "user_id": "uuid-of-sender",
  "message_type": "TEXT",
  "content": "Hello everyone!",
  "reply_to_id": null,
  "is_edited": false,
  "created_at": "2026-04-25T10:05:00Z",
  "updated_at": "2026-04-25T10:05:00Z"
}
```

> **Note:** Future server-pushed events (typing indicator, presence, reactions, notifications) will use the same channel with a different `message_type` or a wrapper event object. The format will be documented when implemented.

---

#### WebSocket Client Hook

```typescript
// src/hooks/useWebSocket.ts
import { useEffect, useRef, useCallback } from 'react';
import { useAuthStore } from '../stores/authStore';
import { WS_BASE_URL } from '../lib/constants';
import type { Message } from '../types';

interface UseWebSocketOptions {
  channelId: string | null;
  onMessage: (msg: Message) => void;
}

export function useWebSocket({ channelId, onMessage }: UseWebSocketOptions) {
  const ws = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>();
  const reconnectDelay = useRef(1000);
  const accessToken = useAuthStore((s) => s.accessToken);

  const connect = useCallback(() => {
    if (!channelId || !accessToken) return;

    const url = `${WS_BASE_URL}/ws/channels/${channelId}?token=${accessToken}`;
    ws.current = new WebSocket(url);

    ws.current.onopen = () => {
      reconnectDelay.current = 1000; // reset backoff on success
    };

    ws.current.onmessage = (event) => {
      try {
        const msg: Message = JSON.parse(event.data);
        onMessage(msg);
      } catch {
        // ignore malformed frames
      }
    };

    ws.current.onclose = () => {
      // Reconnect with exponential backoff
      reconnectTimer.current = setTimeout(() => {
        reconnectDelay.current = Math.min(reconnectDelay.current * 2, 30000);
        connect();
      }, reconnectDelay.current);
    };

    ws.current.onerror = () => {
      ws.current?.close();
    };
  }, [channelId, accessToken, onMessage]);

  useEffect(() => {
    connect();
    return () => {
      clearTimeout(reconnectTimer.current);
      ws.current?.close();
    };
  }, [connect]);

  const sendMessage = useCallback((content: string) => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({ message_type: 'TEXT', content }));
    }
  }, []);

  return { sendMessage };
}
```

---

### 4.6 Planned Future Endpoints

> These endpoints do NOT exist yet. They will be added in upcoming sprints. Build the FE components against these contracts — they are final.

#### Workspace

| Method | Path | Description |
|---|---|---|
| POST | `/workspaces` | Create workspace |
| GET | `/workspaces` | List user's workspaces |
| GET | `/workspaces/:id` | Get workspace details |
| POST | `/workspaces/:id/members` | Invite member |
| DELETE | `/workspaces/:id/members/:userId` | Remove member |

#### Channel

| Method | Path | Description |
|---|---|---|
| POST | `/workspaces/:id/channels` | Create channel |
| GET | `/workspaces/:id/channels` | List channels in workspace |
| POST | `/channels/:id/members/:userId` | Add member to channel |
| DELETE | `/channels/:id/members/:userId` | Remove member |

#### Messages

| Method | Path | Description |
|---|---|---|
| GET | `/channels/:id/messages?limit=50&before=<message_uuid>` | Message history (cursor pagination) |
| GET | `/channels/:id/messages/search?q=<query>&limit=20` | Full-text search |

#### Tasks (extended)

| Method | Path | Description |
|---|---|---|
| GET | `/workspaces/:id/tasks` | List tasks (filter: status, priority, assignee) |
| PATCH | `/tasks/:id` | Update task fields |
| DELETE | `/tasks/:id` | Delete task |
| POST | `/tasks/:id/assign` | Assign task to user |
| GET | `/tasks/:id/activities` | Task activity log |

#### Notifications

| Method | Path | Description |
|---|---|---|
| GET | `/notifications?unread_only=true&limit=20` | List notifications |
| PATCH | `/notifications/:id/read` | Mark notification as read |
| POST | `/notifications/read-all` | Mark all as read |

---

## 5. Error Handling Convention

All API errors follow this format:
```json
{ "error": "human-readable error message" }
```

**Standard HTTP status codes used:**

| Code | Meaning | FE action |
|---|---|---|
| 400 | Bad request (validation) | Show field errors |
| 401 | Unauthorized | Try refresh → if fails, redirect to `/login` |
| 403 | Forbidden | Show "access denied" toast |
| 404 | Not found | Show empty state or redirect |
| 409 | Conflict (e.g. duplicate) | Show specific error message |
| 500 | Server error | Show generic "something went wrong" toast |

**Recommended pattern:**
```typescript
import { AxiosError } from 'axios';

function getErrorMessage(error: unknown): string {
  if (error instanceof AxiosError) {
    return error.response?.data?.error ?? error.message;
  }
  return 'Something went wrong';
}
```

---

## 6. State Management Architecture

### Separation of concerns

| State type | Where | Examples |
|---|---|---|
| Server state | TanStack Query | User profile, tasks list, message history |
| Real-time state | Zustand `wsStore` | Incoming WebSocket messages |
| Auth state | Zustand `authStore` (persisted) | User, tokens |
| UI state | Zustand `uiStore` | Selected channel, sidebar open |

### TanStack Query setup

```typescript
// src/main.tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60,     // 1 minute
      retry: (count, error) => {
        if (error instanceof AxiosError && error.response?.status === 401) return false;
        return count < 2;
      },
    },
  },
});
```

### Query key conventions

```typescript
// Consistent query key factory — prevents cache invalidation bugs
export const queryKeys = {
  user:       { me: ['user', 'me'] },
  workspace:  { all: ['workspaces'], one: (id: string) => ['workspaces', id] },
  channels:   { all: (wsId: string) => ['channels', wsId] },
  messages:   { list: (cId: string) => ['messages', cId] },
  tasks:      { workspace: (wsId: string) => ['tasks', wsId] },
  notifications: { all: ['notifications'] },
} as const;
```

---

## 7. Message List: Infinite Scroll Pattern

Messages load newest-first with cursor pagination. Scroll up to load older messages.

```typescript
// src/hooks/useInfiniteMessages.ts
import { useInfiniteQuery } from '@tanstack/react-query';
import { fetchMessages } from '../api/message';
import { queryKeys } from '../lib/queryKeys';

export function useInfiniteMessages(channelId: string) {
  return useInfiniteQuery({
    queryKey: queryKeys.messages.list(channelId),
    queryFn: ({ pageParam }) => fetchMessages(channelId, 50, pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => lastPage.next_cursor,
    select: (data) => ({
      pages: [...data.pages].reverse(),  // oldest page first for display
      pageParams: data.pageParams,
    }),
  });
}
```

**Corresponding API call:**
```typescript
// src/api/message.ts
export async function fetchMessages(
  channelId: string,
  limit = 50,
  beforeId?: string
): Promise<PaginatedMessages> {
  const params = new URLSearchParams({ limit: String(limit) });
  if (beforeId) params.append('before', beforeId);
  const { data } = await api.get(`/channels/${channelId}/messages?${params}`);
  return data;
}
```

---

## 8. Environment Variables

```bash
# .env.local (frontend project root)
VITE_API_BASE_URL=http://localhost:8080/api/v1
VITE_WS_BASE_URL=ws://localhost:8080/api/v1
```

```typescript
// src/lib/constants.ts
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL;
export const WS_BASE_URL  = import.meta.env.VITE_WS_BASE_URL;
```

---

## 9. Routing Structure

```typescript
// src/router/index.tsx
import { createBrowserRouter } from 'react-router-dom';
import { ProtectedRoute, PublicOnlyRoute } from './guards';

export const router = createBrowserRouter([
  {
    element: <PublicOnlyRoute />,
    children: [
      { path: '/login',    element: <LoginPage /> },
      { path: '/register', element: <RegisterPage /> },
    ],
  },
  {
    element: <ProtectedRoute />,
    children: [
      {
        path: '/app',
        element: <AppLayout />,
        children: [
          { index: true, element: <Navigate to="/app/workspaces" replace /> },
          { path: 'workspaces',              element: <WorkspaceListPage /> },
          { path: 'workspaces/:workspaceId', element: <WorkspacePage />,
            children: [
              { path: 'channels/:channelId', element: <ChannelPage /> },
              { path: 'tasks',               element: <KanbanPage /> },
              { path: 'tasks/:taskId',       element: <TaskDetailPage /> },
            ],
          },
          { path: 'profile', element: <ProfilePage /> },
        ],
      },
    ],
  },
  { path: '/', element: <Navigate to="/app" replace /> },
]);
```

---

## 10. Important Rules for AI Agent Building the Frontend

1. **Never put tokens in sessionStorage** — use `localStorage` via Zustand persist.

2. **Always use the Axios instance from `src/api/axios.ts`** — never call `fetch()` or `axios.create()` again. The interceptor handles auth automatically.

3. **Token refresh is handled by the interceptor** — do NOT manually call `/auth/refresh` in component code.

4. **WebSocket messages are additive** — when a new message arrives via WebSocket, ADD it to the TanStack Query cache, do NOT refetch:
   ```typescript
   queryClient.setQueryData(queryKeys.messages.list(channelId), (old) => ({
     ...old,
     pages: [[newMessage, ...old.pages[0]], ...old.pages.slice(1)],
   }));
   ```

5. **Identity comes from the auth store** — `useAuthStore((s) => s.user)`. Never read `created_by` from a response to determine "is this my message".

6. **All UUIDs are strings** in TypeScript (the Go backend uses UUID type which serializes to string).

7. **All timestamps are ISO 8601 strings** — use `date-fns` to format them: `format(parseISO(msg.created_at), 'HH:mm')`.

8. **Error messages come from `response.data.error`** — always extract from the AxiosError response body.

9. **`null` vs `undefined`** — Go omits null UUID fields with `omitempty`. Optional UUID fields may be `undefined` in TypeScript after JSON parse. Always use optional chaining: `task.assignee_id?.substring(0, 8)`.

10. **WebSocket reconnects automatically** — the `useWebSocket` hook handles reconnection. Components just pass `onMessage` and call `sendMessage`. Never manage WebSocket state in component local state.
