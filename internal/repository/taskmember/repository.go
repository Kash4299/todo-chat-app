package taskmember

import (
	"github.com/Kash4299/todo-chat-app/internal/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IChannelMemberRepository manages channel membership.
// Package path kept as "taskmember" for import compatibility during transition.
type IChannelMemberRepository interface {
	Add(member *model.ChannelMember) error
	Remove(channelID, userID uuid.UUID) error
	IsMember(channelID, userID uuid.UUID) (bool, error)
	FindByChannel(channelID uuid.UUID) ([]model.ChannelMember, error)
	FindByUser(userID uuid.UUID) ([]model.ChannelMember, error)
}

type ChannelMemberRepository struct {
	db *gorm.DB
}

func NewTaskMemberRepository(db *gorm.DB) IChannelMemberRepository {
	return &ChannelMemberRepository{db: db}
}

func (r *ChannelMemberRepository) Add(member *model.ChannelMember) error {
	return r.db.Create(member).Error
}

func (r *ChannelMemberRepository) Remove(channelID, userID uuid.UUID) error {
	return r.db.
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Delete(&model.ChannelMember{}).Error
}

func (r *ChannelMemberRepository) IsMember(channelID, userID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.Model(&model.ChannelMember{}).
		Where("channel_id = ? AND user_id = ?", channelID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *ChannelMemberRepository) FindByChannel(channelID uuid.UUID) ([]model.ChannelMember, error) {
	var members []model.ChannelMember
	err := r.db.Where("channel_id = ?", channelID).Find(&members).Error
	return members, err
}

func (r *ChannelMemberRepository) FindByUser(userID uuid.UUID) ([]model.ChannelMember, error) {
	var members []model.ChannelMember
	err := r.db.Where("user_id = ?", userID).Find(&members).Error
	return members, err
}
