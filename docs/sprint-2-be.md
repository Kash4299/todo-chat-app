# Sprint 2 — Backend Implementation Spec

> Tasks: T05, T06, T07, T08, T47 | T48 đã hoàn thành | T25-partial + T32-partial pulled in to unblock FE
>
> **Trạng thái:** T05 ✅ | T06 ✅ | T07 ✅ | T08 ✅ | T25-partial ✅ | T32-partial ✅ | T47 ⏭️ bỏ qua sprint này

---

## Files cần tạo / sửa

**Tạo mới:**
```
internal/repository/workspace/repository.go
internal/repository/workspaceinvitation/repository.go
internal/service/workspace_service.go
internal/service/workspace_invitation_service.go
internal/service/email_service.go
internal/handler/workspace_handler.go
internal/middleware/rbac.go
.github/workflows/ci.yml
```

**Sửa:**
```
internal/repository/workspacemember/repository.go  — thêm AddMember, GetRole
internal/service/user_service.go                   — thêm UpdateProfile
internal/handler/user_handler.go                   — thêm UpdateProfile handler
internal/repository/init.go                        — register repos mới
internal/service/init.go                           — register services mới
internal/handler/init.go                           — register handlers mới
internal/route/route.go                            — thêm routes
```

---

## T05 — Workspace API

### Repository: `internal/repository/workspace/repository.go`

```go
type IWorkspaceRepository interface {
    Create(ws *model.Workspace) error
    FindByID(id uuid.UUID) (*model.Workspace, error)
    FindByUserID(userID uuid.UUID) ([]model.Workspace, error)
    Delete(id uuid.UUID) error
}
```

`FindByUserID` query:
```sql
SELECT workspaces.* FROM workspaces
JOIN workspace_members ON workspaces.id = workspace_members.workspace_id
WHERE workspace_members.user_id = ?
```

### Repository: `internal/repository/workspacemember/repository.go` — thêm 2 methods

```go
type IWorkspaceMemberRepository interface {
    IsMember(workspaceID, userID uuid.UUID) (bool, error)       // đã có
    AddMember(workspaceID, userID uuid.UUID, role string) error
    GetRole(workspaceID, userID uuid.UUID) (string, error)      // ErrRecordNotFound nếu không phải member
}
```

### Service: `internal/service/workspace_service.go`

```go
var ErrWorkspaceNotFound     = errors.New("workspace not found")
var ErrForbidden    = errors.New("forbidden")
var ErrWorkspaceInvalidInput = errors.New("invalid workspace input")

type IWorkspaceService interface {
    Create(ownerID uuid.UUID, name string) (*model.Workspace, error)
    GetByID(actorID, workspaceID uuid.UUID) (*model.Workspace, error)
    ListByUser(userID uuid.UUID) ([]model.Workspace, error)
    Delete(actorID, workspaceID uuid.UUID) error
}
```

**Business rules:**

- `Create`: trim name, max 100 chars. Slug = `generateSlug(name)`. Transaction: insert workspace → insert workspace_member với role `ADMIN`.
- `GetByID`: fetch workspace → verify `IsMember(workspaceID, actorID)` → nếu không phải member trả `ErrForbidden`.
- `Delete`: fetch workspace → verify `workspace.OwnerID == actorID` (phải là owner, không chỉ admin) → delete. Cascade DB tự xử lý workspace_members.

**Slug helper** (private, trong `workspace_service.go`):

```go
// generateSlug tạo URL-safe slug từ name với 4-char random suffix.
// Dùng crypto/rand, không dùng math/rand.
// Ví dụ: "My Team!" → "my-team-x4k2"
func generateSlug(name string) string {
    // 1. lowercase + trim
    // 2. replace khoảng trắng → "-"
    // 3. giữ lại [a-z0-9-], bỏ ký tự đặc biệt
    // 4. collapse multiple dashes
    // 5. append "-" + 4 random chars từ [a-z0-9]
}
```

### Handler: `internal/handler/workspace_handler.go`

| Method | Route | Body | Response |
|--------|-------|------|----------|
| `Create` | `POST /workspaces` | `{"name": "string"}` | 201 + workspace |
| `ListByUser` | `GET /workspaces` | — | 200 + `[]workspace` |
| `GetByID` | `GET /workspaces/:id` | — | 200 + workspace |
| `Delete` | `DELETE /workspaces/:id` | — | 204 |

Tất cả protected. `userID` lấy từ `middleware.UserIDContextKey`.

---

## T06 — Invitation API

> Migration: `000004_add_workspace_invitations` đã có.

### Repository: `internal/repository/workspaceinvitation/repository.go`

```go
type IWorkspaceInvitationRepository interface {
    Create(inv *model.WorkspaceInvitation) error
    FindByToken(token string) (*model.WorkspaceInvitation, error)
    FindByWorkspaceAndEmail(workspaceID uuid.UUID, email string) (*model.WorkspaceInvitation, error)
    MarkUsed(id uuid.UUID, usedAt time.Time) error
    Update(inv *model.WorkspaceInvitation) error
    ListPendingByWorkspace(workspaceID uuid.UUID) ([]*model.WorkspaceInvitation, error)
}
```

`ListPendingByWorkspace` query:
```sql
WHERE workspace_id = ? AND used_at IS NULL ORDER BY created_at DESC
```

### Service: `internal/service/workspace_invitation_service.go`

```go
var ErrInvitationNotFound = errors.New("invitation not found")
var ErrInvitationExpired  = errors.New("invitation expired")
var ErrInvitationUsed     = errors.New("invitation already used")
var ErrAlreadyMember      = errors.New("user is already a member")

type IWorkspaceInvitationService interface {
    Invite(actorID, workspaceID uuid.UUID, email string) error
    ResendInvite(actorID, workspaceID uuid.UUID, email string) error
    AcceptInvite(actorID uuid.UUID, token string) error
    ListPending(actorID, workspaceID uuid.UUID) ([]*model.WorkspaceInvitation, error)
}
```

**Business rules:**

`Invite`:
1. Verify actorID là ADMIN của workspace (dùng `GetRole`)
2. `FindByWorkspaceAndEmail` → nếu tìm thấy, `used_at IS NULL`, chưa expire → error "đã được mời"
3. Nếu tìm thấy nhưng đã expired → tự động resend (gọi update thay vì create)
4. Generate token: `crypto/rand` 32 bytes → hex encode → 64 chars
5. `expires_at = now + 7 ngày`
6. Create invitation record
7. `emailService.SendInvitation(...)` — Sprint 2: log, không send thật

`ResendInvite`:
1. Verify actorID là ADMIN
2. `FindByWorkspaceAndEmail` → không tìm thấy → error
3. Generate token mới, `expires_at` mới, `used_at = nil`
4. `Update(inv)`

`AcceptInvite`:
1. `FindByToken(token)` → không tìm thấy → `ErrInvitationNotFound`
2. `used_at != nil` → `ErrInvitationUsed`
3. `expires_at < now` → `ErrInvitationExpired`
4. actorID email không match `invitation.Email` → `ErrInvitationForbidden`
5. `IsMember(workspaceID, actorID)` → đã là member → `ErrAlreadyMember`
6. Transaction: `AddMember(workspaceID, actorID, "MEMBER")` + `MarkUsed(inv.ID, now)`

`ListPending`:
1. Verify actorID là ADMIN của workspace
2. `ListPendingByWorkspace(workspaceID)`

### Email Service: `internal/service/email_service.go`

```go
type IEmailService interface {
    SendInvitation(toEmail, workspaceName, inviteLink string) error
}

// NoOpEmailService — Sprint 2 stub. TODO Sprint X: wire SendGrid/SES.
type NoOpEmailService struct{}

func (s *NoOpEmailService) SendInvitation(toEmail, workspaceName, inviteLink string) error {
    log.Printf("[EMAIL STUB] invite to %s link: %s", toEmail, inviteLink)
    return nil
}
```

### Handler routes — thêm vào `WorkspaceHandler`

| Method | Route | Body | Response |
|--------|-------|------|----------|
| `Invite` | `POST /workspaces/:id/invitations` | `{"email": "string"}` | 201 |
| `ResendInvite` | `POST /workspaces/:id/invitations/resend` | `{"email": "string"}` | 200 |
| `ListPending` | `GET /workspaces/:id/invitations` | — | 200 + list |
| `AcceptInvite` | `POST /invitations/accept` | `{"token": "string"}` | 200 |

`AcceptInvite` không có `:id` vì token đã chứa workspace context.

---

## T07 — RBAC Middleware

### `internal/middleware/rbac.go`

```go
type RBACMiddleware struct {
    workspaceMemberRepo workspacemember.IWorkspaceMemberRepository
}

func NewRBACMiddleware(repo workspacemember.IWorkspaceMemberRepository) *RBACMiddleware

// RequireWorkspaceRole extract workspaceID từ :id param, userID từ context,
// query GetRole, check role nằm trong allowed list.
func (m *RBACMiddleware) RequireWorkspaceRole(roles ...string) gin.HandlerFunc
```

**Logic middleware:**
1. Extract `userID` từ `middleware.UserIDContextKey`
2. Parse `workspaceID` từ `c.Param("id")`
3. `GetRole(workspaceID, userID)` → `ErrRecordNotFound` → 403
4. Role không nằm trong `roles` list → 403

**Áp dụng trong routes:**
```go
// ADMIN only
workspaces.POST("/:id/invitations",        rbac.RequireWorkspaceRole("ADMIN"), ...)
workspaces.POST("/:id/invitations/resend", rbac.RequireWorkspaceRole("ADMIN"), ...)
workspaces.GET("/:id/invitations",         rbac.RequireWorkspaceRole("ADMIN"), ...)
workspaces.DELETE("/:id",                  rbac.RequireWorkspaceRole("ADMIN"), ...)

// ADMIN + MEMBER
workspaces.GET("/:id", rbac.RequireWorkspaceRole("ADMIN", "MEMBER"), ...)
```

Note: `DELETE /:id` dùng RBAC check role ADMIN, nhưng service layer còn check thêm `workspace.OwnerID == actorID`.

---

## T08 — User Profile Update

### Service — thêm vào `IUserService`

```go
UpdateProfile(userID uuid.UUID, displayName, avatarURL, statusText string) (*model.User, error)
```

Không cần repo method mới — `user.IUserRepository.Update` đã có.

**Business rules:**
- `displayName`: trim, max 100 chars, không empty nếu được truyền
- `avatarURL`: trim, validate URL hợp lệ nếu không empty
- `statusText`: trim, max 150 chars
- Chỉ update fields không empty. Empty string = giữ nguyên giá trị cũ.

### Handler — thêm vào `UserHandler`

| Method | Route | Body | Response |
|--------|-------|------|----------|
| `UpdateProfile` | `PATCH /users/me` | `{"display_name"?, "avatar_url"?, "status_text"?}` | 200 + user |

---

## T47 — GitHub Actions CI

### `.github/workflows/ci.yml`

```yaml
name: CI

on:
  push:
    branches: [master, develop]
  pull_request:
    branches: [master]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:15
        env:
          POSTGRES_USER: postgres
          POSTGRES_PASSWORD: postgres
          POSTGRES_DB: todo_chat_test
        ports:
          - 5432:5432
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    env:
      DB_HOST: localhost
      DB_PORT: 5432
      DB_USERNAME: postgres
      DB_PASSWORD: postgres
      DB_DATABASE: todo_chat_test
      DB_SSLMODE: disable
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: go test -race -count=1 ./...

  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.25'
      - run: |
          go build ./cmd/server
          go build ./cmd/migrate
```

---

## Checklist implement

Làm theo thứ tự để tránh dependency lỗi:

```
[x] 1.  workspacemember repo     — AddMember, GetRole, WithTx
[x] 2.  workspace repo           — Create, FindByID, FindByUserID, Delete, WithTx
[x] 3.  workspaceinvitation repo — 6 methods, WithTx
[x] 4.  workspace service        — Create, GetByID, ListByUser, Delete + generateSlug
[x] 5.  workspace handler        — Create, GetByID, ListByUser, Delete
[x] 6.  init.go / route.go       — workspace repos, service, handler wired
[x] 7.  email service            — SMTP (vượt spec NoOp): SendVerificationEmail + SendWorkspaceInvitationEmail
[x] 8.  workspace invitation service — Invite, ResendInvitation, AcceptInvitation, GetListInvitations
[x] 9.  workspace invitation handler — 4 endpoints: Invite, ResendInvitation, GetListInvitations, AcceptInvitation
[x] 10. RBAC middleware          — RequireWorkspaceRole wired cho ADMIN routes
[x] 11. init.go / route.go       — invitation handler + RBAC wired
[x] 14. Migration 000004         — workspace_invitations table tồn tại
[x] 12. user service             — UpdateProfile (validate displayName ≤100, statusText ≤150, avatarURL http/https)
[x] 13. user handler + route     — PATCH /users/me → 200 + user
[x] 16. task service             — ListByWorkspace (auth + pagination), UpdateTask (auth + validation), DeleteTask (auth) [T25-partial + T32-partial]
[x] 17. task handler + routes    — GET /tasks?workspace_id&page&page_size, PATCH /tasks/:id, DELETE /tasks/:id [FE08, FE09, FE10]
[x] 18. task repo                — FindByWorkspace(offset,limit) + CountByWorkspace for OFFSET pagination
[ ] 15. GitHub Actions           — bỏ qua, chưa có VPS  ← T47
```

### Lệch spec cần lưu ý

- **AcceptInvitation route**: đã sửa thành `POST /invitations/accept` (token trong body, trả về workspace_id để FE redirect). ✅
- **Email service**: spec yêu cầu NoOpEmailService (log only) cho Sprint 2, nhưng đã implement SMTP thật. Không ảnh hưởng correctness nhưng cần SMTP config trong env.
- **Task Update/Delete/List (FE08/09/10)**: pulled từ T25 (Sprint 7) và T32 (Sprint 8) vào Sprint 2 để unblock FE. Các method cũ `Update(t *model.Task)` và `Delete(id uuid.UUID)` không có authorization check — đã fix, interface giờ là `UpdateTask(actorID, taskID, input)` và `DeleteTask(actorID, taskID)`.
- **Task pagination**: `GET /tasks?workspace_id=&page=&page_size=` dùng OFFSET pagination (page/page_size) — đúng theo strategy (cursor-based chỉ dành cho message feed).

---

## Rules không được vi phạm

- `userID` phải lấy từ auth context, không bao giờ từ request body/param
- Handler không gọi repo trực tiếp — phải qua service
- RBAC middleware check role — authorization chi tiết (owner check) nằm trong service
- Mọi service method phải có test file tương ứng
- `uuid.Nil` phải validate ngay đầu method
- Token generation dùng `crypto/rand`, không dùng `math/rand`
