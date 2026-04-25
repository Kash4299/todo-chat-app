package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Kash4299/todo-chat-app/internal/model"
	userrepo "github.com/Kash4299/todo-chat-app/internal/repository/user"
	useridentityrepo "github.com/Kash4299/todo-chat-app/internal/repository/useridentity"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var ErrUserEmailNotVerified = errors.New("email is not verified")
var ErrUserNotFound = errors.New("user not found")

type IUserService interface {
	SyncAuth0User(auth0ID, email, displayName, avatarURL string, emailVerified bool) (*model.User, error)
	GetByAuth0ID(auth0ID string) (*model.User, error)
	GetByID(id uuid.UUID) (*model.User, error)
}

type UserService struct {
	userRepo         userrepo.IUserRepository
	userIdentityRepo useridentityrepo.IUserIdentityRepository
}

func NewUserService(
	userRepo userrepo.IUserRepository,
	userIdentityRepo useridentityrepo.IUserIdentityRepository,
) IUserService {
	return &UserService{
		userRepo:         userRepo,
		userIdentityRepo: userIdentityRepo,
	}
}

func (s *UserService) SyncAuth0User(auth0ID, email, displayName, avatarURL string, emailVerified bool) (*model.User, error) {
	auth0ID = strings.TrimSpace(auth0ID)
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)
	avatarURL = strings.TrimSpace(avatarURL)

	if auth0ID == "" {
		return nil, errors.New("auth0 subject is required")
	}
	provider, err := providerFromSubject(auth0ID)
	if err != nil {
		return nil, err
	}

	identity, err := s.userIdentityRepo.FindByProviderSubject(auth0ID)
	if err == nil {
		existing, err := s.userRepo.FindByID(identity.UserID)
		if err != nil {
			return nil, err
		}
		if err := s.updateUserProfile(existing, email, displayName, avatarURL); err != nil {
			return nil, err
		}
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if email == "" {
		return nil, errors.New("email is required for first login")
	}

	// Identity does not exist yet. Auto-link by verified email if an account exists.
	if existingByEmail, err := s.userRepo.FindByEmail(email); err == nil {
		if !emailVerified {
			return nil, ErrUserEmailNotVerified
		}
		if linkErr := s.userIdentityRepo.Create(&model.UserIdentity{
			UserID:          existingByEmail.ID,
			Provider:        provider,
			ProviderSubject: auth0ID,
			EmailAtLinkTime: email,
			IsPrimary:       false,
		}); linkErr != nil {
			existingIdentity, findErr := s.userIdentityRepo.FindByProviderSubject(auth0ID)
			if findErr != nil {
				return nil, linkErr
			}
			return s.userRepo.FindByID(existingIdentity.UserID)
		}
		return existingByEmail, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if displayName == "" {
		displayName = email
	}

	newUser := &model.User{
		Email:       email,
		DisplayName: displayName,
		AvatarURL:   avatarURL,
	}
	if err := s.userRepo.Create(newUser); err != nil {
		// Concurrent create: fall back to the existing row.
		existingByEmail, findErr := s.userRepo.FindByEmail(email)
		if findErr != nil {
			return nil, err
		}
		newUser = existingByEmail
	}
	if err := s.userIdentityRepo.Create(&model.UserIdentity{
		UserID:          newUser.ID,
		Provider:        provider,
		ProviderSubject: auth0ID,
		EmailAtLinkTime: email,
		IsPrimary:       true,
	}); err != nil {
		existingIdentity, findErr := s.userIdentityRepo.FindByProviderSubject(auth0ID)
		if findErr != nil {
			return nil, err
		}
		existingUser, findErr := s.userRepo.FindByID(existingIdentity.UserID)
		if findErr != nil {
			return nil, findErr
		}
		return existingUser, nil
	}
	return newUser, nil
}

func (s *UserService) GetByAuth0ID(auth0ID string) (*model.User, error) {
	auth0ID = strings.TrimSpace(auth0ID)
	if auth0ID == "" {
		return nil, errors.New("auth0 subject is required")
	}
	identity, err := s.userIdentityRepo.FindByProviderSubject(auth0ID)
	if err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(identity.UserID)
}

func (s *UserService) GetByID(id uuid.UUID) (*model.User, error) {
	if id == uuid.Nil {
		return nil, errors.New("user id is required")
	}
	return s.userRepo.FindByID(id)
}

func (s *UserService) updateUserProfile(user *model.User, email, displayName, avatarURL string) error {
	changed := false
	if email != "" && user.Email != email {
		user.Email = email
		changed = true
	}
	if displayName != "" && user.DisplayName != displayName {
		user.DisplayName = displayName
		changed = true
	}
	if avatarURL != "" && user.AvatarURL != avatarURL {
		user.AvatarURL = avatarURL
		changed = true
	}
	if !changed {
		return nil
	}
	return s.userRepo.Update(user)
}

func providerFromSubject(subject string) (string, error) {
	parts := strings.SplitN(subject, "|", 2)
	if len(parts) == 2 && parts[0] != "" {
		return strings.ToUpper(parts[0]), nil
	}
	return "", fmt.Errorf("invalid auth0 subject format")
}
