package service_test

// ===========================================================================
// HƯỚNG DẪN VIẾT TEST THEO CHUẨN PRODUCTION
// ===========================================================================
//
// CẤU TRÚC MỖI TEST:
//   Arrange → Act → Assert
//   - Arrange: chuẩn bị mock, input
//   - Act:     gọi method cần test
//   - Assert:  verify output và side effects
//
// QUY TẮC:
//   1. Mỗi test chỉ test MỘT behaviour cụ thể
//   2. Tên test: Test<Method>_<Condition>_<ExpectedResult>
//   3. Mock chỉ simulate, không có business logic
//   4. Không test private function trực tiếp (generateSlug)
//      → test gián tiếp qua Create và verify slug format
// ===========================================================================

import (
	"errors"
	"strings"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspace"
	"github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ===========================================================================
// MOCKS
// ===========================================================================

// mockTxManager simulate transaction mà không cần DB thật.
// WithTx trên các repo trả về chính mock đó → closure chạy với mock.
type mockTxManager struct {
	forcedErr error // nếu set, RunInTx trả error ngay mà không chạy fn
}

func (m *mockTxManager) RunInTx(fn func(tx *gorm.DB) error) error {
	if m.forcedErr != nil {
		return m.forcedErr
	}
	return fn(nil) // nil tx — mocked repos bỏ qua tx trong WithTx
}

// mockWorkspaceRepo kiểm soát từng method để simulate các scenario khác nhau.
type mockWorkspaceRepo struct {
	createErr   error
	findByIDws  *model.Workspace
	findByIDErr error
	findByUser  []model.Workspace
	findByUErr  error
	updateErr   error
	deleteErr   error
	// capture args để verify service truyền đúng giá trị
	createdWS  *model.Workspace
	updatedWS  *model.Workspace
}

func (m *mockWorkspaceRepo) WithTx(_ *gorm.DB) workspace.IWorkspaceRepository { return m }
func (m *mockWorkspaceRepo) Create(ws *model.Workspace) error {
	m.createdWS = ws
	return m.createErr
}
func (m *mockWorkspaceRepo) FindByID(_ uuid.UUID) (*model.Workspace, error) {
	return m.findByIDws, m.findByIDErr
}
func (m *mockWorkspaceRepo) FindByUserID(_ uuid.UUID, _, _ int) ([]model.Workspace, int64, error) {
	return m.findByUser, int64(len(m.findByUser)), m.findByUErr
}
func (m *mockWorkspaceRepo) Update(ws *model.Workspace) error {
	m.updatedWS = ws
	return m.updateErr
}
func (m *mockWorkspaceRepo) Delete(_ uuid.UUID) error { return m.deleteErr }

// mockMemberRepo kiểm soát membership check và member creation.
type mockMemberRepo struct {
	isMember    bool
	isMemberErr error
	addErr      error
	role        string
	getRoleErr  error
	// capture args
	addedWorkspaceID uuid.UUID
	addedUserID      uuid.UUID
	addedRole        string
}

func (m *mockMemberRepo) WithTx(_ *gorm.DB) workspacemember.IWorkspaceMemberRepository { return m }
func (m *mockMemberRepo) IsMember(_, _ uuid.UUID) (bool, error) {
	return m.isMember, m.isMemberErr
}
func (m *mockMemberRepo) AddMember(wsID, userID uuid.UUID, role string) error {
	m.addedWorkspaceID = wsID
	m.addedUserID = userID
	m.addedRole = role
	return m.addErr
}
func (m *mockMemberRepo) GetRole(_, _ uuid.UUID) (string, error) {
	return m.role, m.getRoleErr
}

func (m *mockMemberRepo) ListWithUsers(_ uuid.UUID, _, _ int) ([]model.WorkspaceMemberInfo, int64, error) {
	return nil, 0, nil
}

// newMockWorkspaceSerivce là helper tạo WorkspaceService với mocks — tránh lặp code trong mỗi test.
func newMockWorkspaceSerivce(tx *mockTxManager, ws *mockWorkspaceRepo, mb *mockMemberRepo) service.IWorkspaceService {
	return service.NewWorkspaceService(tx, ws, mb)
}

// ===========================================================================
// Create
// ===========================================================================

func TestCreate_RejectsNilOwnerID(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, err := svc.Create(uuid.Nil, "My Team")

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestCreate_RejectsEmptyName(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, err := svc.Create(uuid.New(), "   ") // chỉ whitespace

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestCreate_RejectsNameTooLong(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, err := svc.Create(uuid.New(), strings.Repeat("a", 101))

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestCreate_WorkspaceRepoError_ReturnsError(t *testing.T) {
	// Arrange
	dbErr := errors.New("db connection lost")
	wsRepo := &mockWorkspaceRepo{createErr: dbErr}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	// Act
	got, err := svc.Create(uuid.New(), "My Team")

	// Assert
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
	if got != nil {
		t.Fatal("expected nil workspace on error")
	}
}

func TestCreate_MemberAddError_ReturnsError(t *testing.T) {
	// AddMember fail → transaction rollback, error propagated
	addErr := errors.New("member insert failed")
	memberRepo := &mockMemberRepo{addErr: addErr}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, memberRepo)

	got, err := svc.Create(uuid.New(), "My Team")

	if !errors.Is(err, addErr) {
		t.Fatalf("expected member error, got %v", err)
	}
	if got != nil {
		t.Fatal("expected nil workspace on error")
	}
}

func TestCreate_Success_SetsCorrectFields(t *testing.T) {
	// Arrange
	ownerID := uuid.New()
	wsRepo := &mockWorkspaceRepo{}
	memberRepo := &mockMemberRepo{}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	// Act
	ws, err := svc.Create(ownerID, "  My Team  ") // tên có trailing spaces

	// Assert — không có lỗi
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Workspace fields đúng
	if ws.Name != "My Team" {
		t.Errorf("expected trimmed name 'My Team', got '%s'", ws.Name)
	}
	if ws.OwnerID != ownerID {
		t.Error("expected ownerID to be set")
	}

	// Slug được generate và có format hợp lệ: chỉ [a-z0-9-]
	for _, c := range ws.Slug {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-') {
			t.Errorf("slug contains invalid char '%c': %s", c, ws.Slug)
		}
	}
	if ws.Slug == "" {
		t.Error("expected non-empty slug")
	}

	// AddMember được gọi với đúng role ADMIN
	if memberRepo.addedRole != "ADMIN" {
		t.Errorf("expected ADMIN role, got %s", memberRepo.addedRole)
	}
	if memberRepo.addedUserID != ownerID {
		t.Error("expected owner to be added as member")
	}
}

// ===========================================================================
// GetByID
// ===========================================================================

func TestGetByID_RejectsNilIDs(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	cases := []struct {
		actorID     uuid.UUID
		workspaceID uuid.UUID
	}{
		{uuid.Nil, uuid.New()},
		{uuid.New(), uuid.Nil},
	}

	for _, tc := range cases {
		_, err := svc.GetByID(tc.actorID, tc.workspaceID)
		if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
			t.Errorf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
		}
	}
}

func TestGetByID_WorkspaceNotFound(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	_, err := svc.GetByID(uuid.New(), uuid.New())

	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound, got %v", err)
	}
}

func TestGetByID_NotMember_ReturnsNotFound(t *testing.T) {
	// Security: không lộ workspace tồn tại hay không với non-member
	wsRepo := &mockWorkspaceRepo{findByIDws: &model.Workspace{ID: uuid.New()}}
	memberRepo := &mockMemberRepo{isMember: false}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	_, err := svc.GetByID(uuid.New(), uuid.New())

	// Phải trả NotFound, không phải Forbidden — để tránh leak thông tin
	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound (not Forbidden), got %v", err)
	}
}

func TestGetByID_MembershipRepoError(t *testing.T) {
	dbErr := errors.New("membership query failed")
	wsRepo := &mockWorkspaceRepo{findByIDws: &model.Workspace{ID: uuid.New()}}
	memberRepo := &mockMemberRepo{isMemberErr: dbErr}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	_, err := svc.GetByID(uuid.New(), uuid.New())

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}
}

func TestGetByID_Success(t *testing.T) {
	expected := &model.Workspace{ID: uuid.New(), Name: "My Team"}
	wsRepo := &mockWorkspaceRepo{findByIDws: expected}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	got, err := svc.GetByID(uuid.New(), uuid.New())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != expected {
		t.Fatal("expected same workspace pointer")
	}
}

// ===========================================================================
// ListByUser
// ===========================================================================

func TestListByUser_RejectsNilUserID(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, _, err := svc.ListByUser(uuid.Nil, 1, 20)

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestListByUser_RepoError(t *testing.T) {
	dbErr := errors.New("query failed")
	wsRepo := &mockWorkspaceRepo{findByUErr: dbErr}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	_, _, err := svc.ListByUser(uuid.New(), 1, 20)

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestListByUser_Success_ReturnsAll(t *testing.T) {
	expected := []model.Workspace{{ID: uuid.New()}, {ID: uuid.New()}}
	wsRepo := &mockWorkspaceRepo{findByUser: expected}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	got, total, err := svc.ListByUser(uuid.New(), 1, 20)

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(got))
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
}

func TestListByUser_EmptyList_ReturnsNilError(t *testing.T) {
	// Không có workspace nào → không phải lỗi, trả slice rỗng
	wsRepo := &mockWorkspaceRepo{findByUser: []model.Workspace{}}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	got, _, err := svc.ListByUser(uuid.New(), 1, 20)

	if err != nil {
		t.Fatalf("expected nil error for empty list, got %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty list, got %d items", len(got))
	}
}

// ===========================================================================
// Delete
// ===========================================================================

func TestDelete_RejectsNilIDs(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	cases := []struct{ actorID, wsID uuid.UUID }{
		{uuid.Nil, uuid.New()},
		{uuid.New(), uuid.Nil},
	}
	for _, tc := range cases {
		if err := svc.Delete(tc.actorID, tc.wsID); !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
			t.Errorf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
		}
	}
}

func TestDelete_WorkspaceNotFound(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	err := svc.Delete(uuid.New(), uuid.New())

	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound, got %v", err)
	}
}

func TestDelete_NonMember_ReturnsNotFound(t *testing.T) {
	// Non-member recieves NotFound — không lộ workspace có tồn tại không
	ownerID := uuid.New()
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: uuid.New(), OwnerID: ownerID},
	}
	memberRepo := &mockMemberRepo{isMember: false}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	err := svc.Delete(uuid.New(), uuid.New())

	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound (not Forbidden), got %v", err)
	}
}

func TestDelete_MemberNotOwner_ReturnsForbidden(t *testing.T) {
	// Là member nhưng không phải owner → Forbidden
	ownerID := uuid.New()
	member := uuid.New()
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: uuid.New(), OwnerID: ownerID},
	}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	err := svc.Delete(member, uuid.New())

	if !errors.Is(err, constants.ErrForbidden) {
		t.Fatalf("expected constants.ErrForbidden, got %v", err)
	}
}

func TestDelete_DeleteRepoError(t *testing.T) {
	ownerID := uuid.New()
	dbErr := errors.New("delete failed")
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: uuid.New(), OwnerID: ownerID},
		deleteErr:  dbErr,
	}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	err := svc.Delete(ownerID, uuid.New())

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected delete error, got %v", err)
	}
}

func TestDelete_Success(t *testing.T) {
	ownerID := uuid.New()
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: uuid.New(), OwnerID: ownerID},
	}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	err := svc.Delete(ownerID, uuid.New())

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ===========================================================================
// Update
// ===========================================================================

func TestUpdate_RejectsNilIDs(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	cases := []struct{ actorID, wsID uuid.UUID }{
		{uuid.Nil, uuid.New()},
		{uuid.New(), uuid.Nil},
	}
	for _, tc := range cases {
		_, err := svc.Update(tc.actorID, tc.wsID, "New Name")
		if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
			t.Errorf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
		}
	}
}

func TestUpdate_RejectsEmptyName(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, err := svc.Update(uuid.New(), uuid.New(), "   ")

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestUpdate_RejectsNameTooLong(t *testing.T) {
	svc := newMockWorkspaceSerivce(&mockTxManager{}, &mockWorkspaceRepo{}, &mockMemberRepo{})

	_, err := svc.Update(uuid.New(), uuid.New(), strings.Repeat("a", 101))

	if !errors.Is(err, constants.ErrWorkspaceInvalidInput) {
		t.Fatalf("expected constants.ErrWorkspaceInvalidInput, got %v", err)
	}
}

func TestUpdate_WorkspaceNotFound(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{findByIDErr: gorm.ErrRecordNotFound}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, &mockMemberRepo{})

	_, err := svc.Update(uuid.New(), uuid.New(), "New Name")

	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound, got %v", err)
	}
}

func TestUpdate_NonMember_ReturnsNotFound(t *testing.T) {
	wsRepo := &mockWorkspaceRepo{findByIDws: &model.Workspace{ID: uuid.New()}}
	memberRepo := &mockMemberRepo{isMember: false}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	_, err := svc.Update(uuid.New(), uuid.New(), "New Name")

	if !errors.Is(err, constants.ErrWorkspaceNotFound) {
		t.Fatalf("expected constants.ErrWorkspaceNotFound (not Forbidden), got %v", err)
	}
}

func TestUpdate_RepoError(t *testing.T) {
	dbErr := errors.New("update failed")
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: uuid.New()},
		updateErr:  dbErr,
	}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	_, err := svc.Update(uuid.New(), uuid.New(), "New Name")

	if !errors.Is(err, dbErr) {
		t.Fatalf("expected update error, got %v", err)
	}
}

func TestUpdate_Success_UpdatesName(t *testing.T) {
	wsID := uuid.New()
	wsRepo := &mockWorkspaceRepo{
		findByIDws: &model.Workspace{ID: wsID, Name: "Old Name"},
	}
	memberRepo := &mockMemberRepo{isMember: true}
	svc := newMockWorkspaceSerivce(&mockTxManager{}, wsRepo, memberRepo)

	got, err := svc.Update(uuid.New(), wsID, "  New Name  ")

	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.Name != "New Name" {
		t.Errorf("expected trimmed name 'New Name', got %q", got.Name)
	}
	if wsRepo.updatedWS == nil {
		t.Fatal("expected Update to be called on repo")
	}
}
