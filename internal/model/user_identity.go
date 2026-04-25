package model

import (
	"time"

	"github.com/google/uuid"
)

// UserIdentity maps one external identity provider subject (Auth0 `sub`)
// to exactly one internal user.
type UserIdentity struct {
	ID              uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index"                        json:"user_id"`
	Provider        string    `gorm:"not null;size:50"                                json:"provider"`
	ProviderSubject string    `gorm:"not null;uniqueIndex;size:255"                   json:"provider_subject"`
	EmailAtLinkTime string    `gorm:"not null;size:255"                                json:"email_at_link_time"`
	IsPrimary       bool      `gorm:"not null;default:false"                          json:"is_primary"`
	LinkedAt        time.Time `gorm:"not null;autoCreateTime"                         json:"linked_at"`
	CreatedAt       time.Time `gorm:"not null;autoCreateTime"                         json:"created_at"`
}
