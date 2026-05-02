package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	workspaceInvitationRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspaceinvitation"
	workspaceMemberRepo "github.com/Kash4299/todo-chat-app/internal/repository/workspacemember"
	userRepo "github.com/Kash4299/todo-chat-app/internal/repository/user"
	"github.com/Kash4299/todo-chat-app/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func generateInvitationToken() (string, error) {
	bytes := make([]byte, 32)

	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil

}

type IWorkspaceInvitationService interface {
	Invite(workspaceID, inviterID uuid.UUID, inviteeEmail string) error
	ResendInvitation(workspaceID, inviterID uuid.UUID, inviteeEmail string) error
	GetListInvitations(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error)
	AcceptInvitation(actorID uuid.UUID, token string) (uuid.UUID, error)
}

type WorkspaceInvitationService struct {
	txManager               database.ITxManager
	emailService            IEmailService
	userRepo                userRepo.IUserRepository
	workspaceInvitationRepo workspaceInvitationRepo.IWorkspaceInvitationRepository
	workspaceMemberRepo     workspaceMemberRepo.IWorkspaceMemberRepository
}

func NewWorkspaceInvitationService(txManager database.ITxManager, emailService IEmailService, userRepo userRepo.IUserRepository, workspaceInvitationRepo workspaceInvitationRepo.IWorkspaceInvitationRepository, workspaceMemberRepo workspaceMemberRepo.IWorkspaceMemberRepository) IWorkspaceInvitationService {
	return &WorkspaceInvitationService{
		txManager:               txManager,
		emailService:            emailService,
		userRepo:                userRepo,
		workspaceInvitationRepo: workspaceInvitationRepo,
		workspaceMemberRepo:     workspaceMemberRepo,
	}
}

func (s *WorkspaceInvitationService) Invite(workspaceID, inviterID uuid.UUID, inviteeEmail string) error {
	if workspaceID == uuid.Nil || inviterID == uuid.Nil || strings.TrimSpace(inviteeEmail) == "" {
		return constants.ErrWorkspaceInvitationInvalidInput
	}
	inviteeEmail = strings.ToLower(strings.TrimSpace(inviteeEmail))

	role, err := s.workspaceMemberRepo.GetRole(workspaceID, inviterID)
	if err != nil {
		return err
	}
	if role != constants.AdminRole {
		return constants.ErrWorkspaceInvitationNotAdmin
	}

	existingInvitation, err := s.workspaceInvitationRepo.FindByWorkspaceAndEmail(workspaceID, inviteeEmail)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if existingInvitation != nil && existingInvitation.UsedAt != nil {
		return constants.ErrWorkspaceInvitationIsMember
	}

	if existingInvitation != nil && existingInvitation.ExpiresAt.After(time.Now()) {
		return constants.ErrWorkspaceInvitationStillValid
	}

	token, err := generateInvitationToken()
	if err != nil {
		return err
	}

	err = s.txManager.RunInTx(func(tx *gorm.DB) error {
		invitationRepo := s.workspaceInvitationRepo.WithTx(tx)
		invitation := &model.WorkspaceInvitation{
			WorkspaceID: workspaceID,
			Email:       inviteeEmail,
			Token:       token,
			InvitedBy:   inviterID,
			ExpiresAt:   time.Now().Add(48 * time.Hour),
		}
		if err := invitationRepo.Create(invitation); err != nil {
			return err
		}

		return s.emailService.SendWorkspaceInvitationEmail(inviteeEmail, invitation.Token)
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *WorkspaceInvitationService) ResendInvitation(workspaceID, inviterID uuid.UUID, inviteeEmail string) error {
	if workspaceID == uuid.Nil || inviterID == uuid.Nil || strings.TrimSpace(inviteeEmail) == "" {
		return constants.ErrWorkspaceInvitationInvalidInput
	}
	inviteeEmail = strings.ToLower(strings.TrimSpace(inviteeEmail))

	role, err := s.workspaceMemberRepo.GetRole(workspaceID, inviterID)
	if err != nil {
		return err
	}
	if role != constants.AdminRole {
		return constants.ErrWorkspaceInvitationNotAdmin
	}

	existingInvitation, err := s.workspaceInvitationRepo.FindByWorkspaceAndEmail(workspaceID, inviteeEmail)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if existingInvitation != nil && existingInvitation.UsedAt != nil {
		return constants.ErrWorkspaceInvitationIsMember
	}

	if existingInvitation != nil && existingInvitation.ExpiresAt.After(time.Now()) {
		return constants.ErrWorkspaceInvitationStillValid
	}

	token, err := generateInvitationToken()
	if err != nil {
		return err
	}

	err = s.txManager.RunInTx(func(tx *gorm.DB) error {
		invitationRepo := s.workspaceInvitationRepo.WithTx(tx)
		if existingInvitation == nil {
			invitation := &model.WorkspaceInvitation{
				WorkspaceID: workspaceID,
				Email:       inviteeEmail,
				Token:       token,
				InvitedBy:   inviterID,
				ExpiresAt:   time.Now().Add(48 * time.Hour),
			}
			if err := invitationRepo.Create(invitation); err != nil {
				return err
			}

			return s.emailService.SendWorkspaceInvitationEmail(inviteeEmail, invitation.Token)
		} else {
			existingInvitation.ExpiresAt = time.Now().Add(48 * time.Hour)
			existingInvitation.Token = token

			if err := invitationRepo.Update(existingInvitation); err != nil {
				return err
			}
		}

		return s.emailService.SendWorkspaceInvitationEmail(inviteeEmail, existingInvitation.Token)
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *WorkspaceInvitationService) GetListInvitations(workspaceID uuid.UUID) ([]model.WorkspaceInvitation, error) {
	return s.workspaceInvitationRepo.ListPendingByWorkspace(workspaceID)
}

func (s *WorkspaceInvitationService) AcceptInvitation(actorID uuid.UUID, token string) (uuid.UUID, error) {
	if actorID == uuid.Nil || token == "" {
		return uuid.Nil, constants.ErrWorkspaceInvitationInvalidInput
	}

	invitation, err := s.workspaceInvitationRepo.FindByToken(token)
	if err != nil {
		return uuid.Nil, err
	}

	if invitation.UsedAt != nil {
		return uuid.Nil, constants.ErrWorkspaceInvitationIsMember
	}

	if invitation.ExpiresAt.Before(time.Now()) {
		return uuid.Nil, constants.ErrWorkspaceInvitationIsExpired
	}

	actor, err := s.userRepo.FindByID(actorID)
	if err != nil {
		return uuid.Nil, err
	}
	if actor == nil {
		return uuid.Nil, constants.ErrUserNotFound
	}
	if !strings.EqualFold(actor.Email, invitation.Email) {
		return uuid.Nil, constants.ErrWorkspaceInvitationEmailMismatch
	}

	isMember, err := s.workspaceMemberRepo.IsMember(invitation.WorkspaceID, actorID)
	if err != nil {
		return uuid.Nil, err
	}
	if isMember {
		return uuid.Nil, constants.ErrWorkspaceInvitationIsMember
	}

	err = s.txManager.RunInTx(func(tx *gorm.DB) error {
		invitationRepo := s.workspaceInvitationRepo.WithTx(tx)
		memberRepo := s.workspaceMemberRepo.WithTx(tx)

		if err := memberRepo.AddMember(invitation.WorkspaceID, actorID, constants.MemberRole); err != nil {
			return err
		}

		return invitationRepo.MarkUsed(invitation.ID, time.Now())
	})
	if err != nil {
		return uuid.Nil, err
	}

	return invitation.WorkspaceID, nil
}
