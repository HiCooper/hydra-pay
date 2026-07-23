package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Merchant struct {
	ID           uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Name         string    `gorm:"type:varchar(255);not null"                      json:"name"`
	Email        string    `gorm:"type:varchar(255);not null;uniqueIndex"          json:"email"`
	PasswordHash string    `gorm:"type:varchar(255);not null"                      json:"-"`
	ContactName  string    `gorm:"type:varchar(100)"                               json:"contact_name"`
	ContactPhone string    `gorm:"type:varchar(30)"                                json:"contact_phone"`
	Status       string    `gorm:"type:varchar(20);default:active"                 json:"status"`


	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
