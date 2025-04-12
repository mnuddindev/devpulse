package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ReadingListEntry struct {
	ReactionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"reaction_id" validate:"required"`
	Notes      string    `gorm:"type:text" json:"notes" validate:"omitempty,max=500"`
	IsPrivate  bool      `gorm:"default:false;index" json:"is_private"`
	Tags       []string  `gorm:"type:text[];index:idx_reading_tags,gin" json:"tags" validate:"max=5,dive,max=20,alphanumunicode"`

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Reaction Reaction `gorm:"foreignKey:ReactionID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE" json:"reaction" validate:"-"`
}
