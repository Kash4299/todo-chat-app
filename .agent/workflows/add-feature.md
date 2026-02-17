---
description: How to add a new function or service following TDD in this project
---

# Adding a New Feature (TDD Flow)

This guide covers two scenarios:
- **A) Adding a new function** to an existing service (e.g. `GetTodoByID`)
- **B) Adding an entirely new service** (e.g. `CommentService`)

---

## A) Adding a New Function to an Existing Service

> Example: adding `GetTodoByID` to the Todo domain.

### Step 1: Define the repository method in the interface

```go
// internal/repositories/todo/interfaces.go
type ITodoRepository interface {
    GetAllTodoList(ctx context.Context) (*[]model.Todo, error)
    CreateTodo(ctx context.Context, in *model.Todo) error
    GetTodoByID(ctx context.Context, id uuid.UUID) (*model.Todo, error)  // ← NEW
}
```

### Step 2: Define the service method in the interface

```go
// internal/services/interfaces.go
type ITodoService interface {
    GetAllTodos(ctx context.Context) (*[]model.Todo, error)
    CreateTodo(ctx context.Context, req *request.CreateTodoRequest) error
    GetTodoByID(ctx context.Context, id uuid.UUID) (*model.Todo, error)  // ← NEW
}
```

### Step 3: Write the test FIRST (Red phase)

```go
// internal/services/todo_service_test.go — add a new test function

func TestGetTodoByID(t *testing.T) {
    expectedID := uuid.New()

    tests := []struct {
        name      string
        id        uuid.UUID
        mockSetup func(m *MockTodoRepository)
        wantErr   bool
    }{
        {
            name: "success - returns todo",
            id:   expectedID,
            mockSetup: func(m *MockTodoRepository) {
                m.On("GetTodoByID", mock.Anything, expectedID).Return(&model.Todo{
                    BaseModel: model.BaseModel{ID: expectedID},
                    Title:     "Found Todo",
                }, nil)
            },
            wantErr: false,
        },
        {
            name: "error - not found",
            id:   uuid.New(),
            mockSetup: func(m *MockTodoRepository) {
                m.On("GetTodoByID", mock.Anything, mock.Anything).
                    Return(nil, errors.New("not found"))
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockRepo := new(MockTodoRepository)
            tt.mockSetup(mockRepo)
            svc := newTestTodoService(mockRepo)

            result, err := svc.GetTodoByID(context.Background(), tt.id)

            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.id, result.ID)
            }
            mockRepo.AssertExpectations(t)
        })
    }
}
```

> **Don't forget**: update `MockTodoRepository` in the test file to add the new method.

### Step 4: Implement the repository (Green phase)

```go
// internal/repositories/todo/get_todo_by_id.go  ← NEW FILE
package todo

import (
    "context"
    "todo/internal/model"
    "github.com/google/uuid"
)

func (r *TodoRepository) GetTodoByID(ctx context.Context, id uuid.UUID) (*model.Todo, error) {
    var todo model.Todo
    if err := r.db.DB.WithContext(ctx).First(&todo, "id = ?", id).Error; err != nil {
        return nil, err
    }
    return &todo, nil
}
```

### Step 5: Implement the service

```go
// internal/services/todo.go — add method
func (s *TodoService) GetTodoByID(ctx context.Context, id uuid.UUID) (*model.Todo, error) {
    return s.hub.TodoRepository.GetTodoByID(ctx, id)
}
```

### Step 6: Expose via HTTP handler

```go
// internal/handlers/todo/todo_handler.go — add method
func (h *TodoHandler) GetByID(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        c.JSON(http.StatusBadRequest, common.BadRequestResponse("invalid id"))
        return
    }

    todo, err := h.service.GetTodoByID(c.Request.Context(), id)
    if err != nil {
        c.JSON(http.StatusNotFound, common.NotFoundResponse("todo not found"))
        return
    }

    c.JSON(http.StatusOK, common.SuccessResponse(todo))
}
```

### Step 7: Add the route

```go
// internal/server/router.go — uncomment or add
todos.GET("/:id", middleware.AuthenticateMiddleware, r.handlers.TodoHandler.GetByID)
```

### Step 8: Expose via gRPC (optional)

1. Add the RPC to `proto/todo.proto`:
   ```protobuf
   rpc GetTodoByID (GetTodoByIDRequest) returns (TodoItem);
   ```
2. Run `make proto-gen`
3. Implement the handler in `internal/grpc/todo_grpc.go`
4. Write the gRPC test in `internal/grpc/todo_grpc_test.go`

### Step 9: Verify

```bash
make test          # All tests should pass
make test-cover    # Check coverage
```

---

## B) Adding an Entirely New Service

> Example: adding a `Comment` service.

### Step 1: Create the model

```go
// internal/model/comment.go  ← NEW FILE
package model

import "github.com/google/uuid"

type Comment struct {
    BaseModel
    TodoID  uuid.UUID `gorm:"column:todo_id" json:"todo_id"`
    UserID  uuid.UUID `gorm:"column:user_id" json:"user_id"`
    Content string    `gorm:"column:content" json:"content"`
}
```

### Step 2: Create the DTO

```go
// internal/dtos/request/comment.go  ← NEW FILE
package request

type CreateCommentRequest struct {
    TodoID  string `json:"todo_id"`
    Content string `json:"content"`
}
```

### Step 3: Create the repository + interface

```go
// internal/repositories/comment/interfaces.go  ← NEW FILE
package comment

import (
    "context"
    "todo/internal/model"
    "github.com/google/uuid"
)

type ICommentRepository interface {
    CreateComment(ctx context.Context, in *model.Comment) error
    GetCommentsByTodoID(ctx context.Context, todoID uuid.UUID) (*[]model.Comment, error)
}
```

```go
// internal/repositories/comment/new.go  ← NEW FILE
package comment

import "todo/internal/database"

type CommentRepository struct {
    db *database.Database
}

func NewCommentRepository(db *database.Database) *CommentRepository {
    return &CommentRepository{db: db}
}
```

```go
// internal/repositories/comment/create_comment.go  ← NEW FILE
// (implement the actual DB logic)
```

### Step 4: Register repository in Fx module

```go
// internal/repositories/init.go — add import and provider
import "todo/internal/repositories/comment"

var Module = fx.Options(
    fx.Provide(
        todo.NewTodoRepository,
        user.NewUserRepository,
        comment.NewCommentRepository,  // ← NEW
    ),
)
```

### Step 5: Add to Hub + service interface

```go
// internal/services/hub.go — add field
type Hub struct {
    TodoRepository    todorepo.ITodoRepository
    UserRepository    userrepo.IUserRepository
    CommentRepository commentrepo.ICommentRepository  // ← NEW
}
```

```go
// internal/services/interfaces.go — add interface
type ICommentService interface {
    CreateComment(ctx context.Context, req *request.CreateCommentRequest, userID string) error
    GetCommentsByTodoID(ctx context.Context, todoID uuid.UUID) (*[]model.Comment, error)
}
```

### Step 6: Create the service + write tests FIRST

```go
// internal/services/comment.go  ← NEW FILE
// internal/services/comment_service_test.go  ← NEW TEST FILE (write tests first!)
```

### Step 7: Register service in Fx module

```go
// internal/services/init.go — add provider
var Module = fx.Options(
    fx.Provide(
        NewHub,
        NewTodoService,
        NewUserService,
        NewCommentService,  // ← NEW
    ),
)
```

### Step 8: Create handler, register in Fx, add routes

Follow the same pattern as `internal/handlers/todo/`.

### Step 9: Create gRPC handler (optional)

1. Create `proto/comment.proto`
2. Run `make proto-gen`
3. Create `internal/grpc/comment_grpc.go`

### Step 10: Verify

```bash
make test
make test-cover
```

---

## Quick Checklist

For any new feature, make sure you touch these layers:

| Layer | File Pattern | Purpose |
|-------|-------------|---------|
| Model | `internal/model/<name>.go` | Database schema |
| DTO | `internal/dtos/request/<name>.go` | Request validation |
| Repo Interface | `internal/repositories/<name>/interfaces.go` | Contract |
| Repo Impl | `internal/repositories/<name>/<method>.go` | DB logic |
| Service Interface | `internal/services/interfaces.go` | Contract |
| Service Impl | `internal/services/<name>.go` | Business logic |
| Service Test | `internal/services/<name>_service_test.go` | **Write FIRST** |
| Handler | `internal/handlers/<name>/` | HTTP layer |
| Router | `internal/server/router.go` | Route registration |
| gRPC Proto | `proto/<name>.proto` | gRPC definition |
| gRPC Handler | `internal/grpc/<name>_grpc.go` | gRPC layer |
| gRPC Test | `internal/grpc/<name>_grpc_test.go` | **Write FIRST** |
| Fx Modules | `init.go` files | Dependency wiring |
