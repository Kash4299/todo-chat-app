package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/constants"
	"github.com/Kash4299/todo-chat-app/internal/model"
	emailverificationrepo "github.com/Kash4299/todo-chat-app/internal/repository/emailverification"
	tokenrepo "github.com/Kash4299/todo-chat-app/internal/repository/token"
	userrepo "github.com/Kash4299/todo-chat-app/internal/repository/user"
	useridentityrepo "github.com/Kash4299/todo-chat-app/internal/repository/useridentity"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const LocalIssuer = "kashflow"
const LinkIssuer = "kashflow-link"

const verificationTokenExpiry = 24 * time.Hour
const resendVerificationCooldown = 1 * time.Minute

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ILocalAuthService interface {
	Register(email, password, displayName string) (*model.User, error)
	ResendVerification(email string) error
	Login(email, password string) (*model.User, *TokenPair, error)
	VerifyEmail(rawToken string) (*model.User, *TokenPair, error)
	Refresh(rawRefreshToken string) (*TokenPair, error)
	Logout(rawRefreshToken string) error
	SetPassword(userID uuid.UUID, newPassword string) error
	ConfirmAccountLink(pendingToken, password string) (*model.User, *TokenPair, error)
}

type LocalAuthService struct {
	userRepo              userrepo.IUserRepository
	tokenRepo             tokenrepo.IRefreshTokenRepository
	userIdentityRepo      useridentityrepo.IUserIdentityRepository
	emailVerificationRepo emailverificationrepo.IEmailVerificationRepository
	emailService          IEmailService
	jwtSecret             []byte
	accessExpiry          time.Duration
	refreshExpiry         time.Duration
}

func NewLocalAuthService(
	userRepo userrepo.IUserRepository,
	tokenRepo tokenrepo.IRefreshTokenRepository,
	userIdentityRepo useridentityrepo.IUserIdentityRepository,
	emailVerificationRepo emailverificationrepo.IEmailVerificationRepository,
	emailService IEmailService,
	cfg *config.Config,
) (ILocalAuthService, error) {
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, errors.New("JWT_SECRET is required for local auth")
	}
	return &LocalAuthService{
		userRepo:              userRepo,
		tokenRepo:             tokenRepo,
		userIdentityRepo:      userIdentityRepo,
		emailVerificationRepo: emailVerificationRepo,
		emailService:          emailService,
		jwtSecret:             []byte(cfg.JWTSecret),
		accessExpiry:          time.Duration(cfg.JWTAccessExpiryMin) * time.Minute,
		refreshExpiry:         time.Duration(cfg.JWTRefreshExpiryDay) * 24 * time.Hour,
	}, nil
}

func (s *LocalAuthService) Register(email, password, displayName string) (*model.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, errors.New("email is required")
	}
	if len(password) < 8 {
		return nil, constants.ErrPasswordTooShort
	}
	if displayName == "" {
		displayName = email
	}

	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, constants.ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	hashStr := string(hashBytes)

	user := &model.User{
		Email:         email,
		DisplayName:   displayName,
		PasswordHash:  &hashStr,
		EmailVerified: false,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	if err := s.sendVerificationEmail(user); err != nil {
		if deleteErr := s.userRepo.DeleteByID(user.ID); deleteErr != nil {
			return nil, fmt.Errorf("send verification email: %w; rollback user creation: %v", err, deleteErr)
		}
		return nil, err
	}

	return user, nil
}

func (s *LocalAuthService) ResendVerification(email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return errors.New("email is required")
	}

	user, err := s.userRepo.FindByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if user == nil || user.ID == uuid.Nil {
		return nil
	}
	if user.EmailVerified {
		return nil
	}

	latest, err := s.emailVerificationRepo.FindLatestByUserID(user.ID)
	if err == nil {
		if latest != nil && time.Since(latest.CreatedAt) < resendVerificationCooldown {
			return constants.ErrVerificationEmailRateLimited
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return s.sendVerificationEmail(user)
}

func (s *LocalAuthService) Login(email, password string) (*model.User, *TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.userRepo.FindByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, constants.ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, err
	}

	if user.PasswordHash == nil {
		return nil, nil, constants.ErrNoPasswordSet
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, constants.ErrInvalidCredentials
	}

	if !user.EmailVerified {
		return nil, nil, constants.ErrEmailNotVerified
	}

	pair, err := s.issueTokenPair(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *LocalAuthService) VerifyEmail(rawToken string) (*model.User, *TokenPair, error) {
	hash := hashToken(rawToken)

	userID, err := s.emailVerificationRepo.ConsumeValidByHash(hash, time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, constants.ErrInvalidVerificationToken
		}
		return nil, nil, fmt.Errorf("consume verification token: %w", err)
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, nil, err
	}

	user.EmailVerified = true
	if err := s.userRepo.Update(user); err != nil {
		return nil, nil, err
	}

	pair, err := s.issueTokenPair(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *LocalAuthService) Refresh(rawRefreshToken string) (*TokenPair, error) {
	hash := hashToken(rawRefreshToken)
	stored, err := s.tokenRepo.FindByHash(hash)
	if err != nil {
		return nil, constants.ErrInvalidCredentials
	}
	if time.Now().After(stored.ExpiresAt) {
		_ = s.tokenRepo.DeleteByHash(hash)
		return nil, constants.ErrInvalidCredentials
	}

	if err := s.tokenRepo.DeleteByHash(hash); err != nil {
		return nil, err
	}

	return s.issueTokenPair(stored.UserID)
}

func (s *LocalAuthService) Logout(rawRefreshToken string) error {
	_ = s.tokenRepo.DeleteByHash(hashToken(rawRefreshToken))
	return nil
}

func (s *LocalAuthService) ConfirmAccountLink(pendingToken, password string) (*model.User, *TokenPair, error) {
	token, err := jwt.Parse(pendingToken, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return nil, nil, constants.ErrInvalidPendingToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, nil, constants.ErrInvalidPendingToken
	}
	iss, _ := claims["iss"].(string)
	if iss != LinkIssuer {
		return nil, nil, constants.ErrInvalidPendingToken
	}
	googleSub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	if googleSub == "" || email == "" {
		return nil, nil, constants.ErrInvalidPendingToken
	}

	user, err := s.userRepo.FindByEmail(email)
	if err != nil {
		return nil, nil, constants.ErrInvalidPendingToken
	}
	if user == nil || user.ID == uuid.Nil {
		return nil, nil, constants.ErrInvalidPendingToken
	}

	if user.PasswordHash == nil {
		return nil, nil, constants.ErrNoPasswordSet
	}
	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, constants.ErrInvalidCredentials
	}

	provider, err := providerFromSubject(googleSub)
	if err != nil {
		return nil, nil, constants.ErrInvalidPendingToken
	}

	linkErr := s.userIdentityRepo.Create(&model.UserIdentity{
		UserID:          user.ID,
		Provider:        provider,
		ProviderSubject: googleSub,
		EmailAtLinkTime: email,
		IsPrimary:       false,
	})
	if linkErr != nil {
		existing, findErr := s.userIdentityRepo.FindByProviderSubject(googleSub)
		if findErr != nil {
			return nil, nil, linkErr
		}
		if existing == nil || existing.UserID == uuid.Nil {
			return nil, nil, linkErr
		}
		if existing.UserID != user.ID {
			return nil, nil, constants.ErrLinkConflict
		}
	}

	if !user.EmailVerified {
		user.EmailVerified = true
		if err := s.userRepo.Update(user); err != nil {
			return nil, nil, err
		}
	}

	pair, err := s.issueTokenPair(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *LocalAuthService) SetPassword(userID uuid.UUID, newPassword string) error {
	if len(newPassword) < 8 {
		return constants.ErrPasswordTooShort
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if user.PasswordHash != nil {
		return constants.ErrPasswordAlreadySet
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashStr := string(hashBytes)
	user.PasswordHash = &hashStr
	return s.userRepo.Update(user)
}

func (s *LocalAuthService) sendVerificationEmail(user *model.User) error {
	rawToken, err := generateRandomToken()
	if err != nil {
		return err
	}

	_ = s.emailVerificationRepo.DeleteByUserID(user.ID)

	if err := s.emailVerificationRepo.Create(&model.EmailVerificationToken{
		UserID:    user.ID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: time.Now().Add(verificationTokenExpiry),
	}); err != nil {
		return err
	}

	return s.emailService.SendVerificationEmail(user.Email, rawToken)
}

func (s *LocalAuthService) issueTokenPair(userID uuid.UUID) (*TokenPair, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub": userID.String(),
		"iss": LocalIssuer,
		"iat": now.Unix(),
		"exp": now.Add(s.accessExpiry).Unix(),
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	rawRefresh, err := generateRandomToken()
	if err != nil {
		return nil, err
	}

	rt := &model.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: now.Add(s.refreshExpiry),
	}
	if err := s.tokenRepo.Save(rt); err != nil {
		return nil, err
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: rawRefresh}, nil
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
