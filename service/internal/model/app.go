package model

import (
	"time"

	"github.com/google/uuid"
)

// App represents an application that can use the payment API.
type App struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"type:varchar(255);not null"`
	APIKey    string    `gorm:"type:varchar(255);not null;uniqueIndex"`
	Status    string    `gorm:"type:varchar(20);default:active"`
	CreatedAt time.Time
}
