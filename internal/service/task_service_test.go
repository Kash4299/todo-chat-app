package service_test

import (
	"errors"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── mocks ─────────────────────────────────────────────────────────────────────

type mockTaskRepo struct {
	createErr             error
	createCalls           int
	createdTask           *model.Task
	findByIDTask          *model.Task
	findByIDErr           error
	findByWorkspaceTasks  []model.Task
	findByWorkspaceErr    error
	countByWorkspaceTotal int64
	countByWorkspaceErr   error
	updateErr             error
	deleteErr             error
}

func (m *mockTaskRepo) Create(task *model.Task) error {
	m.createCalls++
	m.createdTask = task
	return m.createErr
}

func (m *mockTaskRepo) FindByID(id uuid.UUID) (*model.Task, error) {
	return m.findByIDTask, m.findByIDErr
}

func (m *mockTaskRepo) FindByWorkspace(workspaceID uuid.UUID, offset, limit int) ([]model.Task, error) {
	return m.findByWorkspaceTasks, m.findByWorkspaceErr
}

func (m *mockTaskRepo) CountByWorkspace(workspaceID uuid.UUID) (int64, error) {
	return m.countByWorkspaceTotal, m.countByWorkspaceErr
}

func (m *mockTaskRepo) Update(task *model.Task) error {
	return m.updateErr
}

func (m *mockTaskRepo) Delete(id uuid.UUID) error {
	return m.deleteErr
}

type mockWorkspaceMemberRepo struct {
	isMember         bool
	role             string
	err              error
	addedWorkspaceID uuid.UUID
	addedUserID      uuid.UUID
	addedRole        string
}

func (m *mockWorkspaceMemberRepo) WithTx(_ *gorm.DB) workspacemember.IWorkspaceMemberRepository {
	return m
}

func (m *mockWorkspaceMemberRepo) IsMember(workspaceID, userID uuid.UUID) (bool, error) {
	return m.isMember, m.err
}

func (m *mockWorkspaceMemberRepo) AddMember(workspaceID, userID uuid.UUID, role string) error {
	m.addedWorkspaceID = workspaceID
	m.addedUserID = userID
	m.addedRole = role
	return m.err
}

func (m *mockWorkspaceMemberRepo) GetRole(workspaceID, userID uuid.UUID) (string, error) {
	return m.role, m.err
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestTaskService_CreateRejectsInvalidInput(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})

	err := svc.Create(uuid.Nil, &model.Task{WorkspaceID: uuid.New()})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_CreateRejectsNonMember(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: false})
	err := svc.Create(uuid.New(), &model.Task{WorkspaceID: uuid.New(), Title: "t"})
	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTaskService_CreateAppliesDefaultsAndSaves(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})

	actorID := uuid.New()
	task := &model.Task{WorkspaceID: uuid.New(), Title: "Task", Priority: ""}
	if err := svc.Create(actorID, task); err != nil {
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

	err := svc.Create(uuid.New(), &model.Task{WorkspaceID: uuid.New(), Title: "x", Priority: "invalid"})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput for bad priority, got %v", err)
	}
}

func TestTaskService_CreateUppercasesPriority(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})

	task := &model.Task{WorkspaceID: uuid.New(), Title: "x", Priority: "high"}
	if err := svc.Create(uuid.New(), task); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if task.Priority != "HIGH" {
		t.Fatalf("expected HIGH priority, got %s", task.Priority)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestTaskService_GetByIDRejectsInvalidInput(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.GetByID(uuid.Nil, uuid.New())
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_GetByIDMapsNotFound(t *testing.T) {
	repo := &mockTaskRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, constants.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskService_GetByIDRejectsForbidden(t *testing.T) {
	repo := &mockTaskRepo{findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: false})
	_, err := svc.GetByID(uuid.New(), uuid.New())
	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
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
	repo := &mockTaskRepo{findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}}
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

// ── ListByWorkspace ───────────────────────────────────────────────────────────

func TestTaskService_ListByWorkspace_RejectsNilActorID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, _, err := svc.ListByWorkspace(uuid.Nil, uuid.New(), 1, 20)
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_RejectsNilWorkspaceID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.Nil, 1, 20)
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_RejectsForbidden(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: false})
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 20)
	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_MembershipError(t *testing.T) {
	expected := errors.New("membership failed")
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{err: expected})
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 20)
	if !errors.Is(err, expected) {
		t.Fatalf("expected membership error, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_CountError(t *testing.T) {
	expected := errors.New("count failed")
	repo := &mockTaskRepo{countByWorkspaceErr: expected}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 20)
	if !errors.Is(err, expected) {
		t.Fatalf("expected count error, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_FindError(t *testing.T) {
	expected := errors.New("find failed")
	repo := &mockTaskRepo{findByWorkspaceErr: expected, countByWorkspaceTotal: 5}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 20)
	if !errors.Is(err, expected) {
		t.Fatalf("expected find error, got %v", err)
	}
}

func TestTaskService_ListByWorkspace_Success(t *testing.T) {
	tasks := []model.Task{{ID: uuid.New()}, {ID: uuid.New()}}
	repo := &mockTaskRepo{findByWorkspaceTasks: tasks, countByWorkspaceTotal: 2}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, total, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 20)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(got))
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
}

func TestTaskService_ListByWorkspace_ClampsLargePageSize(t *testing.T) {
	repo := &mockTaskRepo{}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	// Should not error regardless of absurd pageSize input
	_, _, err := svc.ListByWorkspace(uuid.New(), uuid.New(), 1, 9999)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ── UpdateTask ────────────────────────────────────────────────────────────────

func strPtrTask(s string) *string { return &s }

func TestTaskService_UpdateTask_RejectsNilActorID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.Nil, uuid.New(), service.TaskUpdateInput{})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_RejectsNilTaskID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.Nil, service.TaskUpdateInput{})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_TaskNotFound(t *testing.T) {
	repo := &mockTaskRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{})
	if !errors.Is(err, constants.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskService_UpdateTask_Forbidden(t *testing.T) {
	repo := &mockTaskRepo{findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: false})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{})
	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTaskService_UpdateTask_InvalidStatus(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{Status: strPtrTask("INVALID")})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_InvalidPriority(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{Priority: strPtrTask("INVALID")})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_EmptyTitle(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{Title: strPtrTask("")})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_InvalidDueDateFormat(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{DueDate: strPtrTask("not-a-date")})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_UpdateTask_NoChanges_SkipsRepoUpdate(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "same", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	// Sending same title produces no change → Update should not be called
	got, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{Title: strPtrTask("same")})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != task {
		t.Fatal("expected same task pointer")
	}
	if repo.updateErr != nil {
		t.Fatal("Update should not have been called")
	}
}

func TestTaskService_UpdateTask_UpdatesTitle(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "old", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{Title: strPtrTask("new title")})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.Title != "new title" {
		t.Fatalf("expected title 'new title', got %s", got.Title)
	}
}

func TestTaskService_UpdateTask_ClearsDueDate(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{DueDate: strPtrTask("")})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.DueDate != nil {
		t.Fatal("expected DueDate to be nil after clearing")
	}
}

func TestTaskService_UpdateTask_NilUUIDAssigneeRejected(t *testing.T) {
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM"}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	nilUUID := "00000000-0000-0000-0000-000000000000"
	_, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{AssigneeID: &nilUUID})
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput for nil-UUID assignee, got %v", err)
	}
}

func TestTaskService_UpdateTask_UnassignsAssignee(t *testing.T) {
	assignee := uuid.New()
	task := &model.Task{ID: uuid.New(), WorkspaceID: uuid.New(), Title: "t", Status: "TODO", Priority: "MEDIUM", AssigneeID: &assignee}
	repo := &mockTaskRepo{findByIDTask: task}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	got, err := svc.UpdateTask(uuid.New(), uuid.New(), service.TaskUpdateInput{AssigneeID: strPtrTask("")})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.AssigneeID != nil {
		t.Fatal("expected AssigneeID to be nil after unassign")
	}
}

// ── DeleteTask ────────────────────────────────────────────────────────────────

func TestTaskService_DeleteTask_RejectsNilActorID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	err := svc.DeleteTask(uuid.Nil, uuid.New())
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_DeleteTask_RejectsNilTaskID(t *testing.T) {
	svc := service.NewTaskService(&mockTaskRepo{}, &mockWorkspaceMemberRepo{isMember: true})
	err := svc.DeleteTask(uuid.New(), uuid.Nil)
	if !errors.Is(err, constants.ErrTaskInvalidInput) {
		t.Fatalf("expected ErrTaskInvalidInput, got %v", err)
	}
}

func TestTaskService_DeleteTask_NotFound(t *testing.T) {
	repo := &mockTaskRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	err := svc.DeleteTask(uuid.New(), uuid.New())
	if !errors.Is(err, constants.ErrTaskNotFound) {
		t.Fatalf("expected ErrTaskNotFound, got %v", err)
	}
}

func TestTaskService_DeleteTask_Forbidden(t *testing.T) {
	repo := &mockTaskRepo{findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: false})
	err := svc.DeleteTask(uuid.New(), uuid.New())
	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestTaskService_DeleteTask_Success(t *testing.T) {
	repo := &mockTaskRepo{findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()}}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	if err := svc.DeleteTask(uuid.New(), uuid.New()); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestTaskService_DeleteTask_RepoError(t *testing.T) {
	expected := errors.New("delete failed")
	repo := &mockTaskRepo{
		findByIDTask: &model.Task{ID: uuid.New(), WorkspaceID: uuid.New()},
		deleteErr:    expected,
	}
	svc := service.NewTaskService(repo, &mockWorkspaceMemberRepo{isMember: true})
	err := svc.DeleteTask(uuid.New(), uuid.New())
	if !errors.Is(err, expected) {
		t.Fatalf("expected delete error, got %v", err)
	}
}
