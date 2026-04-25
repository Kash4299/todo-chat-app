package service_test

import (
	"errors"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockTaskRepo struct {
	createErr            error
	createCalls          int
	createdTask          *model.Task
	findByIDTask         *model.Task
	findByIDErr          error
	findByWorkspaceTasks []model.Task
	findByWorkspaceErr   error
	updateErr            error
	deleteErr            error
}

func (m *mockTaskRepo) Create(task *model.Task) error {
	m.createCalls++
	m.createdTask = task
	return m.createErr
}

func (m *mockTaskRepo) FindByID(id uuid.UUID) (*model.Task, error) {
	return m.findByIDTask, m.findByIDErr
}

func (m *mockTaskRepo) FindByWorkspace(workspaceID uuid.UUID) ([]model.Task, error) {
	return m.findByWorkspaceTasks, m.findByWorkspaceErr
}

func (m *mockTaskRepo) Update(task *model.Task) error {
	return m.updateErr
}

func (m *mockTaskRepo) Delete(id uuid.UUID) error {
	return m.deleteErr
}

type mockWorkspaceMemberRepo struct {
	isMember bool
	err      error
}

func (m *mockWorkspaceMemberRepo) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	return m.isMember, m.err
}

func TestTaskService_CreateRejectsInvalidInput(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})

	err := svc.Create(uuid.Nil, &model.Task{WorkspaceID: uuid.New()})
	if !errors.Is(err, service.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_CreateRejectsNonMember(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: false})
	err := svc.Create(uuid.New(), &model.Task{WorkspaceID: uuid.New(), Title: "t"})
	if !errors.Is(err, service.ErrTaskForbidden) {
		t.Fatalf("expected ErrTaskForbidden, got %v", err)
	}
}

func TestTaskService_CreateAppliesDefaultsAndSaves(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})

	actorID := uuid.New()
	task := &model.Task{
		WorkspaceID: uuid.New(),
		Title:       "Task",
		Priority:    "",
	}
	err := svc.Create(actorID, task)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", repo.createCalls)
	}
	if task.CreatedBy != actorID {
		t.Fatal("expected CreatedBy to be set from actor")
	}
	if task.Status != "TODO" {
		t.Fatalf("expected TODO status, got %s", task.Status)
	}
	if task.Priority != "MEDIUM" {
		t.Fatalf("expected MEDIUM priority, got %s", task.Priority)
	}
}

func TestTaskService_GetByIDRejectsForbidden(t *testing.T) {
	repo := &mockTaskRepo{
		findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()},
	}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: false})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrTaskForbidden) {
		t.Fatalf("expected ErrTaskForbidden, got %v", err)
	}
}

func TestTaskService_GetByIDMapsNotFound(t *testing.T) {
	repo := &mockTaskRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, service.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskService_CreateMembershipRepoError(t *testing.T) {
	expected := errors.New("membership check failed")
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{err: expected})

	err := svc.Create(uuid.New(), &model.Task{WorkspaceID: uuid.New(), Title: "x"})
	if !errors.Is(err, expected) {
		t.Fatalf("expected membership repo error, got %v", err)
	}
}

func TestTaskService_CreateRejectsInvalidPriority(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})

	err := svc.Create(uuid.New(), &model.Task{
		WorkspaceID: uuid.New(),
		Title:       "x",
		Priority:    "invalid",
	})
	if !errors.Is(err, service.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput for bad priority, got %v", err)
	}
}

func TestTaskService_CreateUppercasesPriority(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})

	task := &model.Task{
		WorkspaceID: uuid.New(),
		Title:       "x",
		Priority:    "high",
	}
	err := svc.Create(uuid.New(), task)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if task.Priority != "HIGH" {
		t.Fatalf("expected HIGH priority, got %s", task.Priority)
	}
}

func TestTaskService_GetByIDRejectsInvalidInput(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.GetByID(uuid.Nil, uuid.New())
	if !errors.Is(err, service.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_GetByIDReturnsRepoError(t *testing.T) {
	expected := errors.New("db exploded")
	repo := &mockTaskRepo{findByIDErr: expected}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, expected) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestTaskService_GetByIDMembershipRepoError(t *testing.T) {
	expected := errors.New("membership query failed")
	repo := &mockTaskRepo{
		findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()},
	}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{err: expected})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, expected) {
		t.Fatalf("expected membership error, got %v", err)
	}
}

func TestTaskService_GetByIDSuccess(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, err := svc.GetByID(uuid.New(), uuid.New())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != task {
		t.Fatal("expected same task pointer")
	}
}

func TestTaskService_GetByWorkspaceProxy(t *testing.T) {
	expected := []model.Task{{ID: uuid.New()}, {ID: uuid.New()}}
	repo := &mockTaskRepo{findByWorkspaceTasks: expected}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, err := svc.GetByWorkspace(uuid.New())
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
}

func TestTaskService_UpdateDeleteProxyErrors(t *testing.T) {
	updateErr := errors.New("update failed")
	deleteErr := errors.New("delete failed")
	repo := &mockTaskRepo{updateErr: updateErr, deleteErr: deleteErr}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})

	if err := svc.Update(&model.Task{}); !errors.Is(err, updateErr) {
		t.Fatalf("expected updateErr, got %v", err)
	}
	if err := svc.Delete(uuid.New()); !errors.Is(err, deleteErr) {
		t.Fatalf("expected deleteErr, got %v", err)
	}
}
