package task_test

import (
	"os"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	taskrepo "github.com/Kash4299/todo-chat-app/internal/repository/task"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}

	if err := db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`).Error; err != nil {
		t.Fatalf("create extension: %v", err)
	}
	// AutoMigrate does not add FK constraints (no constraint: tags on model.Task),
	// so we can insert tasks with arbitrary workspace/user UUIDs.
	if err := db.AutoMigrate(&model.Task{}); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	if err := db.Exec("TRUNCATE TABLE tasks RESTART IDENTITY CASCADE").Error; err != nil {
		t.Fatalf("truncate tasks: %v", err)
	}

	return db
}

func makeTask(workspaceID, creatorID uuid.UUID, position int) *model.Task {
	return &model.Task{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		Title:       "task",
		Status:      "TODO",
		Priority:    "MEDIUM",
		CreatedBy:   creatorID,
		Position:    position,
		CreatedAt:   time.Now(),
	}
}

// ── FindByWorkspace ───────────────────────────────────────────────────────────

func TestTaskRepository_FindByWorkspace_ReturnsAllInOrder(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	wsID := uuid.New()
	creator := uuid.New()

	// Insert 3 tasks with explicit positions so order is deterministic.
	for i := range 3 {
		task := makeTask(wsID, creator, i)
		if err := db.Create(task).Error; err != nil {
			t.Fatalf("seed task: %v", err)
		}
	}

	got, err := repo.FindByWorkspace(wsID, 0, 10)
	if err != nil {
		t.Fatalf("FindByWorkspace: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(got))
	}
	// Ordered by position ASC — first task should have position 0.
	if got[0].Position != 0 {
		t.Fatalf("expected first task position=0, got %d", got[0].Position)
	}
}

func TestTaskRepository_FindByWorkspace_Pagination(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	wsID := uuid.New()
	creator := uuid.New()

	// 5 tasks with distinct positions.
	for i := range 5 {
		task := makeTask(wsID, creator, i)
		if err := db.Create(task).Error; err != nil {
			t.Fatalf("seed task: %v", err)
		}
	}

	// Page 1: offset=0, limit=2 → positions 0,1
	page1, err := repo.FindByWorkspace(wsID, 0, 2)
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1: expected 2, got %d", len(page1))
	}

	// Page 2: offset=2, limit=2 → positions 2,3
	page2, err := repo.FindByWorkspace(wsID, 2, 2)
	if err != nil {
		t.Fatalf("page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Fatalf("page 2: expected 2, got %d", len(page2))
	}
	if page1[0].ID == page2[0].ID {
		t.Fatal("page 1 and page 2 returned the same first task")
	}

	// Page 3: offset=4, limit=2 → only position 4 remains
	page3, err := repo.FindByWorkspace(wsID, 4, 2)
	if err != nil {
		t.Fatalf("page 3: %v", err)
	}
	if len(page3) != 1 {
		t.Fatalf("page 3: expected 1, got %d", len(page3))
	}
}

func TestTaskRepository_FindByWorkspace_FiltersByWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	ws1 := uuid.New()
	ws2 := uuid.New()
	creator := uuid.New()

	task1 := makeTask(ws1, creator, 0)
	task2 := makeTask(ws2, creator, 0)
	for _, tk := range []*model.Task{task1, task2} {
		if err := db.Create(tk).Error; err != nil {
			t.Fatalf("seed task: %v", err)
		}
	}

	got, err := repo.FindByWorkspace(ws1, 0, 10)
	if err != nil {
		t.Fatalf("FindByWorkspace: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 task for ws1, got %d", len(got))
	}
	if got[0].WorkspaceID != ws1 {
		t.Fatalf("expected workspace_id %s, got %s", ws1, got[0].WorkspaceID)
	}
}

func TestTaskRepository_FindByWorkspace_EmptyWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)

	got, err := repo.FindByWorkspace(uuid.New(), 0, 10)
	if err != nil {
		t.Fatalf("expected nil error for empty workspace, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 tasks, got %d", len(got))
	}
}

// ── CountByWorkspace ──────────────────────────────────────────────────────────

func TestTaskRepository_CountByWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	wsID := uuid.New()
	creator := uuid.New()

	for i := range 4 {
		if err := db.Create(makeTask(wsID, creator, i)).Error; err != nil {
			t.Fatalf("seed task: %v", err)
		}
	}

	count, err := repo.CountByWorkspace(wsID)
	if err != nil {
		t.Fatalf("CountByWorkspace: %v", err)
	}
	if count != 4 {
		t.Fatalf("expected count=4, got %d", count)
	}
}

func TestTaskRepository_CountByWorkspace_FiltersByWorkspace(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	ws1 := uuid.New()
	ws2 := uuid.New()
	creator := uuid.New()

	// 3 tasks in ws1, 2 tasks in ws2.
	for i := range 3 {
		if err := db.Create(makeTask(ws1, creator, i)).Error; err != nil {
			t.Fatalf("seed ws1 task: %v", err)
		}
	}
	for i := range 2 {
		if err := db.Create(makeTask(ws2, creator, i)).Error; err != nil {
			t.Fatalf("seed ws2 task: %v", err)
		}
	}

	count1, err := repo.CountByWorkspace(ws1)
	if err != nil {
		t.Fatalf("CountByWorkspace ws1: %v", err)
	}
	if count1 != 3 {
		t.Fatalf("expected count=3 for ws1, got %d", count1)
	}

	count2, err := repo.CountByWorkspace(ws2)
	if err != nil {
		t.Fatalf("CountByWorkspace ws2: %v", err)
	}
	if count2 != 2 {
		t.Fatalf("expected count=2 for ws2, got %d", count2)
	}
}

func TestTaskRepository_CountByWorkspace_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)

	count, err := repo.CountByWorkspace(uuid.New())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if count != 0 {
		t.Fatalf("expected count=0 for empty workspace, got %d", count)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestTaskRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	wsID := uuid.New()
	creator := uuid.New()

	task := makeTask(wsID, creator, 0)
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}

	task.Title = "updated title"
	task.Status = "DONE"
	if err := repo.Update(task); err != nil {
		t.Fatalf("Update: %v", err)
	}

	var got model.Task
	if err := db.First(&got, task.ID).Error; err != nil {
		t.Fatalf("fetch updated task: %v", err)
	}
	if got.Title != "updated title" {
		t.Fatalf("expected title 'updated title', got %s", got.Title)
	}
	if got.Status != "DONE" {
		t.Fatalf("expected status DONE, got %s", got.Status)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestTaskRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := taskrepo.NewTaskRepository(db)
	wsID := uuid.New()
	creator := uuid.New()

	task := makeTask(wsID, creator, 0)
	if err := db.Create(task).Error; err != nil {
		t.Fatalf("seed task: %v", err)
	}

	if err := repo.Delete(task.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := db.First(&model.Task{}, task.ID).Error; err == nil {
		t.Fatal("expected task to be deleted")
	}
}
