package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Machine struct {
	ID                  string         `gorm:"primaryKey"`
	ProviderConnectionID string         `gorm:"not null;index"`
	Name                string         `gorm:"not null;size:100"`
	Endpoint            string         `gorm:"not null;size:255"`
	ProxyEndpoint       string         `gorm:"size:255"`
	TokenSecretID       *string        `gorm:"index"`
	Status              string         `gorm:"default:'unknown';size:20"` // unknown, healthy, unhealthy, offline
	LastSeen            *time.Time
	CPUArch             string         `gorm:"size:50"`
	CPUCores            int
	MemoryGB            int
	GPUModel            string         `gorm:"size:100"`
	GPUVRAMGB           int
	StorageGB           int
	Labels              string         `gorm:"type:jsonb"` // json map of labels
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

func (m *Machine) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}