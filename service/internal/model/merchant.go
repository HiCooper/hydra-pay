package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Merchant struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name         string    `gorm:"type:varchar(255);not null"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	PasswordHash string    `gorm:"type:varchar(255);not null"`
	ContactName  string    `gorm:"type:varchar(100)"`
	ContactPhone string    `gorm:"type:varchar(30)"`
	Status       string    `gorm:"type:varchar(20);default:active"`

	// Service provider sub-merchant IDs
	AlipayPID      string `gorm:"type:varchar(64)"`
	WechatSubMchid string `gorm:"type:varchar(32)"`
	WechatSubAppid string `gorm:"type:varchar(32)"`

	CreatedAt time.Time
	UpdatedAt time.Time
}

func (m *Merchant) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	m.PasswordHash = string(hash)
	return nil
}

func (m *Merchant) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(m.PasswordHash), []byte(password)) == nil
}
