package service

import (
	"crypto/rand"
	"errors"
	"regexp"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	workspaceRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspace"
	workspaceMemberRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	"github.com/Kash4299/todo-chat-app/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	reSlugInvalid = regexp.MustCompile(`[^a-z0-9-]+`)
	reSlugDash    = regexp.MustCompile(`-{2,}`)
)

const slugSuffix = 7                 // "-" + 6 random chars
const maxSlugBase = 100 - slugSuffix // 93 — keeps total within VARCHAR(100)

// Input: "Hello! My  Team@" => Output: "hello-my-team-a3z9k2"
func generateSlug(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	joined := strings.Join(strings.Fields(name), "-")
	clean := reSlugInvalid.ReplaceAllString(joined, "")
	clean = reSlugDash.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")

	if len(clean) > maxSlugBase {
		clean = strings.TrimRight(clean[:maxSlugBase], "-")
	}

	return clean + "-" + strings.ToLower(rand.Text()[:6])
}

type IWorkspaceService interface {
	Create(ownerID uuid.UUID, name string) (*model.Workspace, error)
	GetByID(actorID, workspaceID uuid.UUID) (*model.Workspace, error)
	ListByUser(userID uuid.UUID, page, pageSize int) ([]model.Workspace, int64, error)
	Delete(actorID, workspaceID uuid.UUID) error
}

type WorkspaceService struct {
	txManager           database.ITxManager
	workspaceRepo       workspaceRepo.IWorkspaceRepository
	workspaceMemberRepo workspaceMemberRepo.IWorkspaceMemberRepository
}

func NewWorkspaceService(txManager database.ITxManager, workspaceRepo workspaceRepo.IWorkspaceRepository, workspaceMemberRepo workspaceMemberRepo.IWorkspaceMemberRepository) IWorkspaceService {
	return &WorkspaceService{
		txManager:           txManager,
		workspaceRepo:       workspaceRepo,
		workspaceMemberRepo: workspaceMemberRepo,
	}
}

func (s *WorkspaceService) Create(ownerID uuid.UUID, name string) (*model.Workspace, error) {
	name = strings.TrimSpace(name)
	if ownerID == uuid.Nil || name == "" {
		return nil, constants.ErrWorkspaceInvalidInput
	}
	if len(name) > 100 {
		return nil, constants.ErrWorkspaceInvalidInput
	}

	var ws *model.Workspace
	err := s.txManager.RunInTx(func(tx *gorm.DB) error {
		txWsRepo := s.workspaceRepo.WithTx(tx)
		txMemberRepo := s.workspaceMemberRepo.WithTx(tx)

		ws = &model.Workspace{
			Name:    name,
			Slug:    generateSlug(name),
			OwnerID: ownerID,
		}

		if err := txWsRepo.Create(ws); err != nil {
			return err
		}

		return txMemberRepo.AddMember(ws.ID, ownerID, "ADMIN")
	})

	if err != nil {
		return nil, err
	}

	return ws, nil
}

func (s *WorkspaceService) GetByID(actorID, workspaceID uuid.UUID) (*model.Workspace, error) {
	if actorID == uuid.Nil || workspaceID == uuid.Nil {
		return nil, constants.ErrWorkspaceInvalidInput
	}

	ws, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, constants.ErrWorkspaceNotFound
		}
		return nil, err
	}

	isMember, err := s.workspaceMemberRepo.IsMember(workspaceID, actorID)
	if err != nil {
		return nil, err
	}

	if !isMember {
		return nil, constants.ErrWorkspaceNotFound
	}

	return ws, nil
}

func (s *WorkspaceService) ListByUser(userID uuid.UUID, page, pageSize int) ([]model.Workspace, int64, error) {
	if userID == uuid.Nil {
		return nil, 0, constants.ErrWorkspaceInvalidInput
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return s.workspaceRepo.FindByUserID(userID, page, pageSize)
}

func (s *WorkspaceService) Delete(actorID, workspaceID uuid.UUID) error {
	if actorID == uuid.Nil || workspaceID == uuid.Nil {
		return constants.ErrWorkspaceInvalidInput
	}

	ws, err := s.workspaceRepo.FindByID(workspaceID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return constants.ErrWorkspaceNotFound
		}
		return err
	}

	// Check membership first — non-members get NotFound to prevent workspace enumeration,
	// consistent with GetByID. Members who are not the owner get Forbidden.
	isMember, err := s.workspaceMemberRepo.IsMember(workspaceID, actorID)
	if err != nil {
		return err
	}
	if !isMember {
		return constants.ErrWorkspaceNotFound
	}

	if ws.OwnerID != actorID {
		return constants.ErrForbidden
	}

	return s.workspaceRepo.Delete(workspaceID)
}
