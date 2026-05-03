package service_test

import (
	"errors"
	"testing"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/Kash4299/todo-chat-app/internal/service"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ── Mocks ────────────────────────────────────────────────────────────────────

type mockUserRepo struct {
	findByEmailUser *model.User
	findByEmailErr  error
	findByEmailFn   func(email string) (*model.User, error)
	findByIDUser    *model.User
	findByIDErr     error
	findByIDFn      func(id uuid.UUID) (*model.User, error)
	createErr       error
	createFn        func(user *model.User) error
	createCalls     int
	createdUser     *model.User
	updateErr       error
	updateFn        func(user *model.User) error
	updateCalls     int
	deleteErr       error
	deleteFn        func(id uuid.UUID) error
	deleteCalls     int
}

func (m *mockUserRepo) Create(user *model.User) error {
	if m.createFn != nil {
		return m.createFn(user)
	}
	m.createCalls++
	m.createdUser = user
	return m.createErr
}
func (m *mockUserRepo) FindByEmail(email string) (*model.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(email)
	}
	return m.findByEmailUser, m.findByEmailErr
}
func (m *mockUserRepo) FindByID(id uuid.UUID) (*model.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(id)
	}
	return m.findByIDUser, m.findByIDErr
}
func (m *mockUserRepo) Update(user *model.User) error {
	if m.updateFn != nil {
		return m.updateFn(user)
	}
	m.updateCalls++
	return m.updateErr
}

func (m *mockUserRepo) DeleteByID(id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	m.deleteCalls++
	return m.deleteErr
}

type mockUserIdentityRepo struct {
	findBySubjectIdentity *model.UserIdentity
	findBySubjectErr      error
	findBySubjectFn       func(providerSubject string) (*model.UserIdentity, error)
	findBySubjectCalls    int

	createErr   error
	createFn    func(identity *model.UserIdentity) error
	createCalls int
	created     []*model.UserIdentity
}

func (m *mockUserIdentityRepo) Create(identity *model.UserIdentity) error {
	if m.createFn != nil {
		return m.createFn(identity)
	}
	m.createCalls++
	m.created = append(m.created, identity)
	return m.createErr
}

func (m *mockUserIdentityRepo) FindByProviderSubject(providerSubject string) (*model.UserIdentity, error) {
	if m.findBySubjectFn != nil {
		return m.findBySubjectFn(providerSubject)
	}
	m.findBySubjectCalls++
	return m.findBySubjectIdentity, m.findBySubjectErr
}

func (m *mockUserIdentityRepo) FindByUserID(userID uuid.UUID) ([]model.UserIdentity, error) {
	return nil, nil
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestUserService_SyncAuth0User_CreatesNewUser(t *testing.T) {
	userRepo := &mockUserRepo{
		findByEmailErr: gorm.ErrRecordNotFound,
	}
	identityRepo := &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, identityRepo)

	user, err := svc.SyncAuth0User("auth0|abc", "test@example.com", "Test User", "https://img", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected created user, got nil")
	}
	if userRepo.createCalls != 1 {
		t.Fatalf("expected one create call, got %d", userRepo.createCalls)
	}
	if userRepo.createdUser == nil || userRepo.createdUser.Email != "test@example.com" {
		t.Fatal("expected created user with normalized email")
	}
	if identityRepo.createCalls != 1 {
		t.Fatalf("expected one identity create call, got %d", identityRepo.createCalls)
	}
	if len(identityRepo.created) != 1 || !identityRepo.created[0].IsPrimary {
		t.Fatal("expected primary identity to be created")
	}
}

func TestUserService_SyncAuth0User_UpdatesExistingUser(t *testing.T) {
	existing := &model.User{
		ID:          uuid.New(),
		Email:       "old@example.com",
		DisplayName: "Old Name",
		AvatarURL:   "",
	}
	userRepo := &mockUserRepo{findByIDUser: existing}
	identityRepo := &mockUserIdentityRepo{
		findBySubjectIdentity: &model.UserIdentity{UserID: existing.ID},
	}
	svc := service.NewUserService(userRepo, identityRepo)

	user, err := svc.SyncAuth0User("auth0|abc", "new@example.com", "New Name", "https://img", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if userRepo.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", userRepo.updateCalls)
	}
	if user.Email != "new@example.com" || user.DisplayName != "New Name" || user.AvatarURL != "https://img" {
		t.Fatal("expected existing user fields to be updated")
	}
}

func TestUserService_SyncAuth0User_RejectsUnverifiedExistingIdentity(t *testing.T) {
	existing := &model.User{ID: uuid.New(), Email: "old@example.com"}
	userRepo := &mockUserRepo{findByIDUser: existing}
	identityRepo := &mockUserIdentityRepo{
		findBySubjectIdentity: &model.UserIdentity{UserID: existing.ID},
	}
	svc := service.NewUserService(userRepo, identityRepo)

	_, err := svc.SyncAuth0User("auth0|abc", "old@example.com", "Old", "", false)
	if !errors.Is(err, constants.ErrUserEmailNotVerified) {
		t.Fatalf("expected constants.ErrUserEmailNotVerified, got %v", err)
	}
	if userRepo.updateCalls != 0 {
		t.Fatalf("expected no profile update, got %d", userRepo.updateCalls)
	}
}

func TestUserService_SyncAuth0User_RequiresConsentForEmailConflict(t *testing.T) {
	existing := &model.User{
		ID:          uuid.New(),
		Email:       "same@example.com",
		DisplayName: "Existing",
	}
	userRepo := &mockUserRepo{findByEmailUser: existing}
	identityRepo := &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, identityRepo)

	_, err := svc.SyncAuth0User("google-oauth2|xyz", "same@example.com", "Existing", "", true)
	var linkErr *service.LinkRequiredError
	if !errors.As(err, &linkErr) {
		t.Fatalf("expected *service.LinkRequiredError, got %v", err)
	}
	if linkErr.GoogleSub != "google-oauth2|xyz" {
		t.Fatalf("expected GoogleSub google-oauth2|xyz, got %s", linkErr.GoogleSub)
	}
	if linkErr.Email != "same@example.com" {
		t.Fatalf("expected Email same@example.com, got %s", linkErr.Email)
	}
	if identityRepo.createCalls != 0 {
		t.Fatal("expected no identity row created — consent not yet given")
	}
}

func TestUserService_SyncAuth0User_RejectsUnverifiedEmailLinking(t *testing.T) {
	existing := &model.User{
		ID:          uuid.New(),
		Email:       "same@example.com",
		DisplayName: "Existing",
	}
	userRepo := &mockUserRepo{findByEmailUser: existing}
	identityRepo := &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, identityRepo)

	_, err := svc.SyncAuth0User("google-oauth2|xyz", "same@example.com", "Existing", "", false)
	if !errors.Is(err, constants.ErrUserEmailNotVerified) {
		t.Fatalf("expected constants.ErrUserEmailNotVerified, got %v", err)
	}
}

func TestUserService_GetByAuth0ID_RequiresAuth0ID(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{})

	_, err := svc.GetByAuth0ID(" ")
	if err == nil {
		t.Fatal("expected error for blank auth0 id")
	}
}

func TestUserService_GetByID_RequiresNonZeroUUID(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{})

	_, err := svc.GetByID(uuid.Nil)
	if err == nil {
		t.Fatal("expected error for zero uuid")
	}
}

func TestUserService_GetByID_RepoError(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{findByIDErr: errors.New("boom")}, &mockUserIdentityRepo{})

	_, err := svc.GetByID(uuid.New())
	if err == nil {
		t.Fatal("expected repository error")
	}
}

func TestUserService_SyncAuth0User_RequiresAuth0ID(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{})
	_, err := svc.SyncAuth0User(" ", "x@example.com", "", "", true)
	if err == nil {
		t.Fatal("expected error for empty auth0 id")
	}
}

func TestUserService_SyncAuth0User_InvalidSubjectFormat(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{})
	_, err := svc.SyncAuth0User("bad-subject", "x@example.com", "", "", true)
	if err == nil {
		t.Fatal("expected invalid subject error")
	}
}

func TestUserService_SyncAuth0User_IdentityRepoError(t *testing.T) {
	expected := errors.New("identity repo failed")
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{findBySubjectErr: expected})
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "", "", true)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserService_SyncAuth0User_ExistingIdentityUserLookupFails(t *testing.T) {
	expected := errors.New("user lookup failed")
	svc := service.NewUserService(
		&mockUserRepo{findByIDErr: expected},
		&mockUserIdentityRepo{findBySubjectIdentity: &model.UserIdentity{UserID: uuid.New()}},
	)
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "", "", true)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserService_SyncAuth0User_RejectsUnverifiedEmailForNewUser(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	identityRepo := &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, identityRepo)

	_, err := svc.SyncAuth0User("auth0|abc", "test@yopmail.com", "Test", "", false)
	if !errors.Is(err, constants.ErrUserEmailNotVerified) {
		t.Fatalf("expected constants.ErrUserEmailNotVerified for new user with unverified email, got %v", err)
	}
	if userRepo.createCalls != 0 {
		t.Fatal("expected no user created when email is not verified")
	}
}

func TestUserService_SyncAuth0User_RequiresEmailOnFirstLogin(t *testing.T) {
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound})
	_, err := svc.SyncAuth0User("auth0|x", " ", "", "", true)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestUserService_SyncAuth0User_FindByEmailRepoError(t *testing.T) {
	expected := errors.New("find by email failed")
	svc := service.NewUserService(
		&mockUserRepo{findByEmailErr: expected},
		&mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound},
	)
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "", "", true)
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
}

func TestUserService_SyncAuth0User_UsesEmailAsDisplayNameFallback(t *testing.T) {
	userRepo := &mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound})
	user, err := svc.SyncAuth0User("auth0|x", "x@example.com", " ", "", true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.DisplayName != "x@example.com" {
		t.Fatalf("expected display name fallback to email, got %s", user.DisplayName)
	}
}

func TestUserService_SyncAuth0User_CreateConflictFallbackFailsLookup(t *testing.T) {
	createErr := errors.New("create failed")
	svc := service.NewUserService(
		&mockUserRepo{
			findByEmailFn: func(email string) (*model.User, error) {
				return nil, gorm.ErrRecordNotFound
			},
			createErr: createErr,
		},
		&mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound},
	)
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "name", "", true)
	if !errors.Is(err, createErr) {
		t.Fatalf("expected original create error, got %v", err)
	}
}

func TestUserService_SyncAuth0User_CreateConflictFallbackSuccess(t *testing.T) {
	existing := &model.User{ID: uuid.New(), Email: "x@example.com", DisplayName: "name"}
	findCount := 0
	userRepo := &mockUserRepo{
		findByEmailFn: func(email string) (*model.User, error) {
			findCount++
			if findCount == 1 {
				return nil, gorm.ErrRecordNotFound
			}
			return existing, nil
		},
		createErr: errors.New("duplicate"),
	}
	identityRepo := &mockUserIdentityRepo{findBySubjectErr: gorm.ErrRecordNotFound}
	svc := service.NewUserService(userRepo, identityRepo)

	user, err := svc.SyncAuth0User("auth0|x", "x@example.com", "name", "", true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.ID != existing.ID {
		t.Fatal("expected existing user returned after create conflict")
	}
}

func TestUserService_SyncAuth0User_IdentityCreateErrorAndLookupError(t *testing.T) {
	createErr := errors.New("identity create failed")
	svc := service.NewUserService(
		&mockUserRepo{findByEmailErr: gorm.ErrRecordNotFound},
		&mockUserIdentityRepo{
			findBySubjectErr: gorm.ErrRecordNotFound,
			createErr:        createErr,
		},
	)
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "name", "", true)
	if !errors.Is(err, createErr) {
		t.Fatalf("expected createErr, got %v", err)
	}
}

func TestUserService_SyncAuth0User_IdentityCreateErrorThenFindUserByIDError(t *testing.T) {
	findByIDErr := errors.New("find by id failed")
	findCalls := 0
	svc := service.NewUserService(
		&mockUserRepo{
			findByEmailErr: gorm.ErrRecordNotFound,
			findByIDErr:    findByIDErr,
		},
		&mockUserIdentityRepo{
			findBySubjectErr: gorm.ErrRecordNotFound,
			createErr:        errors.New("identity create failed"),
			findBySubjectFn: func(providerSubject string) (*model.UserIdentity, error) {
				findCalls++
				if findCalls == 1 {
					return nil, gorm.ErrRecordNotFound
				}
				return &model.UserIdentity{UserID: uuid.New()}, nil
			},
		},
	)
	_, err := svc.SyncAuth0User("auth0|x", "x@example.com", "name", "", true)
	if !errors.Is(err, findByIDErr) {
		t.Fatalf("expected findByIDErr, got %v", err)
	}
}

func TestUserService_GetByAuth0IDRepoPaths(t *testing.T) {
	identityErr := errors.New("identity error")
	svc := service.NewUserService(&mockUserRepo{}, &mockUserIdentityRepo{findBySubjectErr: identityErr})
	_, err := svc.GetByAuth0ID("auth0|x")
	if !errors.Is(err, identityErr) {
		t.Fatalf("expected identity error, got %v", err)
	}

	u := &model.User{ID: uuid.New()}
	svc2 := service.NewUserService(
		&mockUserRepo{findByIDUser: u},
		&mockUserIdentityRepo{findBySubjectIdentity: &model.UserIdentity{UserID: u.ID}},
	)
	got, err := svc2.GetByAuth0ID("auth0|x")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.ID != u.ID {
		t.Fatal("expected user from identity mapping")
	}
}

func TestUserService_SyncAuth0User_ExistingIdentityUpdateError(t *testing.T) {
	updateErr := errors.New("update failed")
	u := &model.User{ID: uuid.New(), Email: "old@example.com", DisplayName: "old"}
	svc := service.NewUserService(
		&mockUserRepo{
			findByIDUser: u,
			updateErr:    updateErr,
		},
		&mockUserIdentityRepo{findBySubjectIdentity: &model.UserIdentity{UserID: u.ID}},
	)
	_, err := svc.SyncAuth0User("auth0|x", "new@example.com", "new", "", true)
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected updateErr, got %v", err)
	}
}

func TestUserService_SyncAuth0User_IdentityCreateErrorThenRecoverByLookup(t *testing.T) {
	recoveredUser := &model.User{ID: uuid.New(), Email: "x@example.com"}
	findCalls := 0
	svc := service.NewUserService(
		&mockUserRepo{
			findByEmailErr: gorm.ErrRecordNotFound,
			findByIDUser:   recoveredUser,
		},
		&mockUserIdentityRepo{
			createErr: errors.New("duplicate identity"),
			findBySubjectFn: func(providerSubject string) (*model.UserIdentity, error) {
				findCalls++
				if findCalls == 1 {
					return nil, gorm.ErrRecordNotFound
				}
				return &model.UserIdentity{UserID: recoveredUser.ID}, nil
			},
		},
	)
	user, err := svc.SyncAuth0User("auth0|x", "x@example.com", "name", "", true)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if user.ID != recoveredUser.ID {
		t.Fatal("expected recovered user from identity lookup")
	}
}

// ── UpdateProfile ─────────────────────────────────────────────────────────────

func newTestUserSvc(userRepo *mockUserRepo) service.IUserService {
	return service.NewUserService(userRepo, &mockUserIdentityRepo{})
}

func strPtr(s string) *string { return &s }

func TestUpdateProfile_NilUserID_ReturnsError(t *testing.T) {
	svc := newTestUserSvc(&mockUserRepo{})
	_, err := svc.UpdateProfile(uuid.Nil, "name", nil, nil)
	if err == nil {
		t.Fatal("expected error for nil userID")
	}
}

func TestUpdateProfile_DisplayNameTooLong_ReturnsInvalidInput(t *testing.T) {
	svc := newTestUserSvc(&mockUserRepo{})
	_, err := svc.UpdateProfile(uuid.New(), string(make([]byte, 101)), nil, nil)
	if !errors.Is(err, constants.ErrUserInvalidInput) {
		t.Fatalf("expected ErrUserInvalidInput, got %v", err)
	}
}

func TestUpdateProfile_StatusTextTooLong_ReturnsInvalidInput(t *testing.T) {
	svc := newTestUserSvc(&mockUserRepo{})
	_, err := svc.UpdateProfile(uuid.New(), "", nil, strPtr(string(make([]byte, 151))))
	if !errors.Is(err, constants.ErrUserInvalidInput) {
		t.Fatalf("expected ErrUserInvalidInput, got %v", err)
	}
}

func TestUpdateProfile_InvalidAvatarURL_ReturnsInvalidInput(t *testing.T) {
	svc := newTestUserSvc(&mockUserRepo{})
	_, err := svc.UpdateProfile(uuid.New(), "", strPtr("not-a-url"), nil)
	if !errors.Is(err, constants.ErrUserInvalidInput) {
		t.Fatalf("expected ErrUserInvalidInput, got %v", err)
	}
}

func TestUpdateProfile_UserNotFound_ReturnsError(t *testing.T) {
	userID := uuid.New()
	svc := newTestUserSvc(&mockUserRepo{findByIDErr: gorm.ErrRecordNotFound})
	_, err := svc.UpdateProfile(userID, "name", nil, nil)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("expected gorm.ErrRecordNotFound, got %v", err)
	}
}

func TestUpdateProfile_PartialUpdate_OnlyDisplayName(t *testing.T) {
	userID := uuid.New()
	existing := &model.User{ID: userID, DisplayName: "old", AvatarURL: "https://old.example.com/a.png", StatusText: "hello"}
	repo := &mockUserRepo{findByIDUser: existing}
	svc := newTestUserSvc(repo)

	got, err := svc.UpdateProfile(userID, "new name", nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.DisplayName != "new name" {
		t.Fatalf("expected display name 'new name', got %q", got.DisplayName)
	}
	if got.AvatarURL != "https://old.example.com/a.png" {
		t.Fatal("expected avatarURL to be unchanged when nil")
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
	}
}

func TestUpdateProfile_NoChange_SkipsUpdate(t *testing.T) {
	userID := uuid.New()
	existing := &model.User{ID: userID, DisplayName: "same", AvatarURL: "", StatusText: ""}
	repo := &mockUserRepo{findByIDUser: existing}
	svc := newTestUserSvc(repo)

	_, err := svc.UpdateProfile(userID, "same", nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected 0 update calls when nothing changed, got %d", repo.updateCalls)
	}
}

func TestUpdateProfile_FullUpdate_AllFields(t *testing.T) {
	userID := uuid.New()
	existing := &model.User{ID: userID, DisplayName: "old", AvatarURL: "", StatusText: ""}
	repo := &mockUserRepo{findByIDUser: existing}
	svc := newTestUserSvc(repo)

	got, err := svc.UpdateProfile(userID, "new name", strPtr("https://cdn.example.com/avatar.png"), strPtr("busy"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.DisplayName != "new name" {
		t.Fatalf("expected 'new name', got %q", got.DisplayName)
	}
	if got.AvatarURL != "https://cdn.example.com/avatar.png" {
		t.Fatalf("expected avatar url, got %q", got.AvatarURL)
	}
	if got.StatusText != "busy" {
		t.Fatalf("expected 'busy', got %q", got.StatusText)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
	}
}

func TestUpdateProfile_ClearAvatarURL_UpdatesToEmpty(t *testing.T) {
	userID := uuid.New()
	existing := &model.User{ID: userID, DisplayName: "name", AvatarURL: "https://old.example.com/a.png", StatusText: "hi"}
	repo := &mockUserRepo{findByIDUser: existing}
	svc := newTestUserSvc(repo)

	got, err := svc.UpdateProfile(userID, "", strPtr(""), nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.AvatarURL != "" {
		t.Fatalf("expected avatarURL to be cleared, got %q", got.AvatarURL)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
	}
}

func TestUpdateProfile_ClearStatusText_UpdatesToEmpty(t *testing.T) {
	userID := uuid.New()
	existing := &model.User{ID: userID, DisplayName: "name", AvatarURL: "", StatusText: "busy"}
	repo := &mockUserRepo{findByIDUser: existing}
	svc := newTestUserSvc(repo)

	got, err := svc.UpdateProfile(userID, "", nil, strPtr(""))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got.StatusText != "" {
		t.Fatalf("expected statusText to be cleared, got %q", got.StatusText)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected 1 update call, got %d", repo.updateCalls)
	}
}
