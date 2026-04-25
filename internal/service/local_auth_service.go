package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/config"
	"github.com/Kash4299/todo-chat-app/internal/model"
	tokenrepo "github.com/Kash4299/todo-chat-app/internal/repository/token"
	userrepo "github.com/Kash4299/todo-chat-app/internal/repository/user"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const LocalIssuer = "kashflow"

var ErrEmailTaken = errors.New("email already registered")
var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrNoPasswordSet = errors.New("account has no password; log in with Google")
var ErrPasswordAlreadySet = errors.New("password is already set; use change-password flow")
var ErrPasswordTooShort = errors.New("password must be at least 8 characters")

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ILocalAuthService interface {
	Register(email, password, displayName string) (*model.User, *TokenPair, error)
	Login(email, password string) (*model.User, *TokenPair, error)
	Refresh(rawRefreshToken string) (*TokenPair, error)
	Logout(rawRefreshToken string) error
	SetPassword(userID uuid.UUID, newPassword string) error
}

type LocalAuthService struct {
	userRepo      userrepo.IUserRepository
	tokenRepo     tokenrepo.IRefreshTokenRepository
	jwtSecret     []byte
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewLocalAuthService(
	userRepo userrepo.IUserRepository,
	tokenRepo tokenrepo.IRefreshTokenRepository,
	cfg *config.Config,
) (ILocalAuthService, error) {
	if strings.TrimSpace(cfg.JWTSecret) == "" {
		return nil, errors.New("JWT_SECRET is required for local auth")
	}
	return &LocalAuthService{
		userRepo:      userRepo,
		tokenRepo:     tokenRepo,
		jwtSecret:     []byte(cfg.JWTSecret),
		accessExpiry:  time.Duration(cfg.JWTAccessExpiryMin) * time.Minute,
		refreshExpiry: time.Duration(cfg.JWTRefreshExpiryDay) * 24 * time.Hour,
	}, nil
}

func (s *LocalAuthService) Register(email, password, displayName string) (*model.User, *TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	displayName = strings.TrimSpace(displayName)

	if email == "" {
		return nil, nil, errors.New("email is required")
	}
	if len(password) < 8 {
		return nil, nil, ErrPasswordTooShort
	}
	if displayName == "" {
		displayName = email
	}

	if _, err := s.userRepo.FindByEmail(email); err == nil {
		return nil, nil, ErrEmailTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, err
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, err
	}
	hashStr := string(hashBytes)

	user := &model.User{
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: &hashStr,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, nil, err
	}

	pair, err := s.issueTokenPair(user.ID)
	if err != nil {
		return nil, nil, err
	}
	return user, pair, nil
}

func (s *LocalAuthService) Login(email, password string) (*model.User, *TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.userRepo.FindByEmail(email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, nil, err
	}

	if user.PasswordHash == nil {
		return nil, nil, ErrNoPasswordSet
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrInvalidCredentials
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
		return nil, ErrInvalidCredentials
	}
	if time.Now().After(stored.ExpiresAt) {
		_ = s.tokenRepo.DeleteByHash(hash)
		return nil, ErrInvalidCredentials
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

func (s *LocalAuthService) SetPassword(userID uuid.UUID, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrPasswordTooShort
	}

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}
	if user.PasswordHash != nil {
		return ErrPasswordAlreadySet
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashStr := string(hashBytes)
	user.PasswordHash = &hashStr
	return s.userRepo.Update(user)
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
