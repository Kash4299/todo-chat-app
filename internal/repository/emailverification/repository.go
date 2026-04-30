package emailverification

import (
	"database/sql"
	"errors"
	"time"

	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IEmailVerificationRepository interface {
	Create(token *model.EmailVerificationToken) error
	FindByHash(hash string) (*model.EmailVerificationToken, error)
	FindLatestByUserID(userID uuid.UUID) (*model.EmailVerificationToken, error)
	ConsumeValidByHash(hash string, now time.Time) (uuid.UUID, error)
	DeleteByHash(hash string) error
	DeleteByUserID(userID uuid.UUID) error
}

type emailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(db *gorm.DB) IEmailVerificationRepository {
	return &emailVerificationRepository{db: db}
}

func (r *emailVerificationRepository) Create(token *model.EmailVerificationToken) error {
	return r.db.Create(token).Error
}

func (r *emailVerificationRepository) FindByHash(hash string) (*model.EmailVerificationToken, error) {
	var token model.EmailVerificationToken
	err := r.db.Where("token_hash = ?", hash).First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *emailVerificationRepository) FindLatestByUserID(userID uuid.UUID) (*model.EmailVerificationToken, error) {
	var token model.EmailVerificationToken
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(&token).Error
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *emailVerificationRepository) ConsumeValidByHash(hash string, now time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := r.db.Raw(`
		DELETE FROM email_verification_tokens
		WHERE token_hash = ? AND expires_at > ?
		RETURNING user_id
	`, hash, now).Row().Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func (r *emailVerificationRepository) DeleteByHash(hash string) error {
	return r.db.Where("token_hash = ?", hash).Delete(&model.EmailVerificationToken{}).Error
}

func (r *emailVerificationRepository) DeleteByUserID(userID uuid.UUID) error {
	return r.db.Where("user_id = ?", userID).Delete(&model.EmailVerificationToken{}).Error
}
