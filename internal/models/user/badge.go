package models

import (
	"time"

	"github.com/google/uuid"
)

type Badge struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name      string    `gorm:"size:100;not null;unique" json:"name"`
	Image     string    `gorm:"size:255;not null" json:"image" validate:"required,url"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
