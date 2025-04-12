package models

import (
	"time"

	"github.com/google/uuid"
)

type SeriesPost struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()" json:"id"`
	SeriesID uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_series" json:"series_id" validate:"required"`
	PostID   uuid.UUID `gorm:"type:uuid;not null;index:idx_series_post_post" json:"post_id" validate:"required"`
	Position int       `gorm:"not null;index:idx_series_position" json:"position" validate:"min=1"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// Relationships
	Post   Posts  `gorm:"foreignKey:PostID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"post" validate:"-"`
	Series Series `gorm:"foreignKey:SeriesID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"series" validate:"-"`
}
