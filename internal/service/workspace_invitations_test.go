package service_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	workspaceInvitationRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspaceinvitation"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── mockUserRepo is defined in user_service_test.go (same package) ──

type mockWorkspaceInvitationRepo struct {
	inviteErr   error
	resendErr   error
	invitations []model.WorkspaceInvitation
	getListErr  error
	createCalls int
	updateCalls int

	acceptErr     error
	markUsedCalls int
	lastMarkedID  uuid.UUID
	lastUsedAt    time.Time

	lastCreated *model.WorkspaceInvitation
	lastUpdated *model.WorkspaceInvitation
}

func (m *mockWorkspaceInvitationRepo) WithTx(tx *gorm.DB) workspaceInvitationRepo.IWorkspaceInvitationRepository {
	return m
}

func (m *mockWorkspaceInvitationRepo) Create(invitation *model.WorkspaceInvitation) error {
	m.createCalls++
	m.lastCreated = invitation
	return m.inviteErr
}

func (m *mockWorkspaceInvitationRepo) FindByToken(token string) (*model.WorkspaceInvitation, error) {
	for _, inv := range m.invitations {
		if inv.Token == token {
			return &inv, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockWorkspaceInvitationRepo) FindByWorkspaceAndEmail(workspaceID uuid.UUID, email string) (*model.WorkspaceInvitation, error) {
	for _, inv := range m.invitations {
		if inv.WorkspaceID == workspaceID && inv.Email == email {
			return &inv, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *mockWorkspaceInvitationRepo) MarkUsed(id uuid.UUID, usedAt time.Time) error {
	m.markUsedCalls++
	m.lastMarkedID = id
	m.lastUsedAt = usedAt
	return m.acceptErr
}

func (m *mockWorkspaceInvitationRepo) Update(invitation *model.WorkspaceInvitation) error {
	m.updateCalls++
	m.lastUpdated = invitation
	return m.resendErr
}

func (m *mockWorkspaceInvitationRepo) ListPendingByWorkspace(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error) {
	return m.invitations, m.getListErr
}

// === Setup for tests ===

// newMockWorkspaceInvitations là helper tạo WorkspaceService với mocks — tránh lặp code trong mỗi test.
func newMockWorkspaceInvitations(tx *mockTxManager, email *mockEmailService, user *mockUserRepo, wpi *mockWorkspaceInvitationRepo, wpm *mockWorkspaceMemberRepo) service.IWorkspaceInvitationService {
	return service.NewWorkspaceInvitationService(tx, email, user, wpi, wpm)
}

func TestWorkspaceInvitation_Invite_RoleNotAdmin(t *testing.T) {
	// Setup mocks
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		inviteErr: nil,
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.GuestRole, // Not an admin
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.Invite(uuid.New(), uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationNotAdmin) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationNotAdmin, got %v", err)
	}
	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for non-admin, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for non-admin, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_Invite_InvitationIsUsed(t *testing.T) {
	// Setup mocks
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Email:       "test@example.com",
				UsedAt:      &time.Time{}, // Marked as used
			},
		},
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.AdminRole,
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.Invite(workspaceID, uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationIsMember) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationIsMember, got %v", err)
	}

	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for existing member, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for existing member, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_Invite_InvitationNotExpired(t *testing.T) {
	// Setup mocks
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Email:       "test@example.com",
				ExpiresAt:   time.Now().Add(1 * time.Hour), // Still valid
			},
		},
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.AdminRole,
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.Invite(workspaceID, uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationStillValid) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationStillValid, got %v", err)
	}

	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for existing member, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for existing member, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_Invite_Success_CreatesAndSendsEmail(t *testing.T) {
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{}
	wpmRepo := &mockWorkspaceMemberRepo{role: constants.AdminRole}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.Invite(workspaceID, uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if wpiRepo.createCalls != 1 {
		t.Fatalf("expected one invitation created, got %d", wpiRepo.createCalls)
	}
	if email.sendCalls != 1 {
		t.Fatalf("expected one invitation email sent, got %d", email.sendCalls)
	}
	if wpiRepo.lastCreated == nil {
		t.Fatal("expected created invitation to be captured")
	}
	if wpiRepo.lastCreated.WorkspaceID != workspaceID {
		t.Fatalf("expected workspaceID %s, got %s", workspaceID, wpiRepo.lastCreated.WorkspaceID)
	}
	if wpiRepo.lastCreated.Email != "test@example.com" {
		t.Fatalf("expected invitee email test@example.com, got %s", wpiRepo.lastCreated.Email)
	}
	if len(wpiRepo.lastCreated.Token) != 64 {
		t.Fatalf("expected token length 64, got %d", len(wpiRepo.lastCreated.Token))
	}
	if wpiRepo.lastCreated.ExpiresAt.Before(time.Now()) {
		t.Fatal("expected future expiration time")
	}
}

func TestWorkspaceInvitation_ResendInvitation_RoleNotAdmin(t *testing.T) {
	// Setup mocks
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		inviteErr: nil,
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.GuestRole, // Not an admin
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.ResendInvitation(uuid.New(), uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationNotAdmin) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationNotAdmin, got %v", err)
	}
	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for non-admin, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for non-admin, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_ResendInvitation_InvitationIsUsed(t *testing.T) {
	// Setup mocks
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Email:       "test@example.com",
				UsedAt:      &time.Time{}, // Marked as used
			},
		},
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.AdminRole,
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.ResendInvitation(workspaceID, uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationIsMember) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationIsMember, got %v", err)
	}

	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for existing member, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for existing member, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_ResendInvitation_InvitationNotExpired(t *testing.T) {
	// Setup mocks
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Email:       "test@example.com",
				ExpiresAt:   time.Now().Add(1 * time.Hour), // Still valid
			},
		},
	}
	wpmRepo := &mockWorkspaceMemberRepo{
		role: constants.AdminRole,
	}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.ResendInvitation(workspaceID, uuid.New(), "test@example.com")
	if !errors.Is(err, constants.ErrWorkspaceInvitationStillValid) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationStillValid, got %v", err)
	}

	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation created for existing member, got %d creates", wpiRepo.createCalls)
	}
	if email.sendCalls != 0 {
		t.Fatalf("expected no email sent for existing member, got %d sends", email.sendCalls)
	}
}

func TestWorkspaceInvitation_ResendInvitation_NoExistingInvitation_CreatesAndSendsEmail(t *testing.T) {
	workspaceID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{}
	wpmRepo := &mockWorkspaceMemberRepo{role: constants.AdminRole}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.ResendInvitation(workspaceID, uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if wpiRepo.createCalls != 1 {
		t.Fatalf("expected one invitation created, got %d", wpiRepo.createCalls)
	}
	if wpiRepo.updateCalls != 0 {
		t.Fatalf("expected no invitation update, got %d", wpiRepo.updateCalls)
	}
	if email.sendCalls != 1 {
		t.Fatalf("expected one invitation email sent, got %d", email.sendCalls)
	}
}

func TestWorkspaceInvitation_ResendInvitation_ExpiredInvitation_UpdatesAndSendsEmail(t *testing.T) {
	workspaceID := uuid.New()
	invitationID := uuid.New()
	tx := &mockTxManager{}
	email := &mockEmailService{}
	wpiRepo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          invitationID,
				WorkspaceID: workspaceID,
				Email:       "test@example.com",
				Token:       "old-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
		},
	}
	wpmRepo := &mockWorkspaceMemberRepo{role: constants.AdminRole}

	svc := newMockWorkspaceInvitations(tx, email, &mockUserRepo{}, wpiRepo, wpmRepo)

	err := svc.ResendInvitation(workspaceID, uuid.New(), "test@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if wpiRepo.createCalls != 0 {
		t.Fatalf("expected no invitation create, got %d", wpiRepo.createCalls)
	}
	if wpiRepo.updateCalls != 1 {
		t.Fatalf("expected one invitation update, got %d", wpiRepo.updateCalls)
	}
	if email.sendCalls != 1 {
		t.Fatalf("expected one invitation email sent, got %d", email.sendCalls)
	}
	if wpiRepo.lastUpdated == nil {
		t.Fatal("expected updated invitation to be captured")
	}
	if wpiRepo.lastUpdated.ID != invitationID {
		t.Fatalf("expected updated invitation id %s, got %s", invitationID, wpiRepo.lastUpdated.ID)
	}
	if wpiRepo.lastUpdated.Token == "old-token" || wpiRepo.lastUpdated.Token == "" {
		t.Fatalf("expected token to be rotated, got %q", wpiRepo.lastUpdated.Token)
	}
	if !wpiRepo.lastUpdated.ExpiresAt.After(time.Now()) {
		t.Fatal("expected invitation expiration to be refreshed")
	}
}

func TestWorkspaceInvitation_GetListInvitations_Success(t *testing.T) {
	workspaceID := uuid.New()
	invitations := []model.WorkspaceInvitation{
		{ID: uuid.New(), WorkspaceID: workspaceID, Email: "a@example.com"},
		{ID: uuid.New(), WorkspaceID: workspaceID, Email: "b@example.com"},
	}

	svc := newMockWorkspaceInvitations(
		&mockTxManager{},
		&mockEmailService{},
		&mockUserRepo{},
		&mockWorkspaceInvitationRepo{invitations: invitations},
		&mockWorkspaceMemberRepo{},
	)

	got, err := svc.GetListInvitations(workspaceID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 invitations, got %d", len(got))
	}
}

func TestWorkspaceInvitation_GetListInvitations_RepoError(t *testing.T) {
	dbErr := errors.New("db failed")
	svc := newMockWorkspaceInvitations(
		&mockTxManager{},
		&mockEmailService{},
		&mockUserRepo{},
		&mockWorkspaceInvitationRepo{getListErr: dbErr},
		&mockWorkspaceMemberRepo{},
	)

	_, err := svc.GetListInvitations(uuid.New())
	if !errors.Is(err, dbErr) {
		t.Fatalf("expected repo error, got %v", err)
	}
}

func TestWorkspaceInvitation_AcceptInvitation_TokenNotFound(t *testing.T) {
	actorID := uuid.New()
	svc := newMockWorkspaceInvitations(
		&mockTxManager{},
		&mockEmailService{},
		&mockUserRepo{},
		&mockWorkspaceInvitationRepo{},
		&mockWorkspaceMemberRepo{},
	)

	_, err := svc.AcceptInvitation(actorID, "missing-token")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestWorkspaceInvitation_AcceptInvitation_AlreadyUsed(t *testing.T) {
	workspaceID := uuid.New()
	actorID := uuid.New()
	usedAt := time.Now()
	repo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Token:       "used-token",
				UsedAt:      &usedAt,
				ExpiresAt:   time.Now().Add(24 * time.Hour),
			},
		},
	}
	svc := newMockWorkspaceInvitations(&mockTxManager{}, &mockEmailService{}, &mockUserRepo{}, repo, &mockWorkspaceMemberRepo{})

	_, err := svc.AcceptInvitation(actorID, "used-token")
	if !errors.Is(err, constants.ErrWorkspaceInvitationIsMember) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationIsMember, got %v", err)
	}
	if repo.markUsedCalls != 0 {
		t.Fatalf("expected mark used not called, got %d", repo.markUsedCalls)
	}
}

func TestWorkspaceInvitation_AcceptInvitation_ExpiredToken(t *testing.T) {
	workspaceID := uuid.New()
	actorID := uuid.New()
	repo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Token:       "expired-token",
				ExpiresAt:   time.Now().Add(-1 * time.Hour),
			},
		},
	}
	svc := newMockWorkspaceInvitations(&mockTxManager{}, &mockEmailService{}, &mockUserRepo{}, repo, &mockWorkspaceMemberRepo{})

	_, err := svc.AcceptInvitation(actorID, "expired-token")
	if !errors.Is(err, constants.ErrWorkspaceInvitationIsExpired) {
		t.Fatalf("expected constants.ErrWorkspaceInvitationIsExpired, got %v", err)
	}
	if repo.markUsedCalls != 0 {
		t.Fatalf("expected mark used not called for expired token, got %d", repo.markUsedCalls)
	}
}

func TestWorkspaceInvitation_AcceptInvitation_ValidToken_MarksUsed(t *testing.T) {
	workspaceID := uuid.New()
	actorID := uuid.New()
	invitationID := uuid.New()
	repo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          invitationID,
				WorkspaceID: workspaceID,
				Token:       "valid-token",
				Email:       "actor@example.com",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
		},
	}
	userRepo := &mockUserRepo{findByIDUser: &model.User{ID: actorID, Email: "actor@example.com"}}
	memberRepo := &mockWorkspaceMemberRepo{isMember: false}
	svc := newMockWorkspaceInvitations(&mockTxManager{}, &mockEmailService{}, userRepo, repo, memberRepo)

	gotWorkspaceID, err := svc.AcceptInvitation(actorID, "valid-token")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if gotWorkspaceID != workspaceID {
		t.Fatalf("expected returned workspace id %s, got %s", workspaceID, gotWorkspaceID)
	}
	if repo.markUsedCalls != 1 {
		t.Fatalf("expected mark used called once, got %d", repo.markUsedCalls)
	}
	if repo.lastMarkedID != invitationID {
		t.Fatalf("expected marked invitation id %s, got %s", invitationID, repo.lastMarkedID)
	}
	if memberRepo.addedWorkspaceID != workspaceID {
		t.Fatalf("expected added workspace id %s, got %s", workspaceID, memberRepo.addedWorkspaceID)
	}
	if memberRepo.addedUserID != actorID {
		t.Fatalf("expected added user id %s, got %s", actorID, memberRepo.addedUserID)
	}
	if memberRepo.addedRole != constants.MemberRole {
		t.Fatalf("expected added role %s, got %s", constants.MemberRole, memberRepo.addedRole)
	}
}

func TestWorkspaceInvitation_AcceptInvitation_EmailMismatch_ReturnsForbidden(t *testing.T) {
	workspaceID := uuid.New()
	actorID := uuid.New()
	repo := &mockWorkspaceInvitationRepo{
		invitations: []model.WorkspaceInvitation{
			{
				ID:          uuid.New(),
				WorkspaceID: workspaceID,
				Token:       "valid-token",
				Email:       "invited@example.com",
				ExpiresAt:   time.Now().Add(1 * time.Hour),
			},
		},
	}
	userRepo := &mockUserRepo{findByIDUser: &model.User{ID: actorID, Email: "other@example.com"}}
	svc := newMockWorkspaceInvitations(&mockTxManager{}, &mockEmailService{}, userRepo, repo, &mockWorkspaceMemberRepo{})

	_, err := svc.AcceptInvitation(actorID, "valid-token")
	if !errors.Is(err, constants.ErrWorkspaceInvitationEmailMismatch) {
		t.Fatalf("expected ErrWorkspaceInvitationEmailMismatch, got %v", err)
	}
	if repo.markUsedCalls != 0 {
		t.Fatalf("expected mark used not called on email mismatch, got %d", repo.markUsedCalls)
	}
}
