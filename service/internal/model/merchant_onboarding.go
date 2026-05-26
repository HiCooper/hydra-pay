package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	OnboardingStatusPending   = "pending"
	OnboardingStatusSubmitted = "submitted"
	OnboardingStatusAuditing  = "auditing"
	OnboardingStatusApproved  = "approved"
	OnboardingStatusRejected  = "rejected"
)

type MerchantOnboarding struct {
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	MerchantID  uuid.UUID      `gorm:"type:uuid;not null;index"`
	Channel     string         `gorm:"type:varchar(32);not null"`
	OutRequestNo string        `gorm:"type:varchar(64);uniqueIndex"`
	ApplymentID  string        `gorm:"type:varchar(64);index"`

	Status        string         `gorm:"type:varchar(32);not null;default:pending"`
	SubMerchantID string         `gorm:"type:varchar(64)"`
	SignURL       string         `gorm:"type:varchar(1000)"`
	QrCodeURL     string         `gorm:"type:varchar(1000)"`

	RequestData  datatypes.JSON `gorm:"type:jsonb"`
	ResponseData datatypes.JSON `gorm:"type:jsonb"`
	CallbackData datatypes.JSON `gorm:"type:jsonb"`
	ErrorMessage string         `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
