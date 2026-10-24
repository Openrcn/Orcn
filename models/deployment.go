package models

import (
	"time"

	"gorm.io/gorm"
)

type Deployment struct {
	ID             string `gorm:"primaryKey"`
	OrganizationID string `gorm:"not null;default:'00000000-0000-0000-0000-000000000001';uniqueIndex:idx_deployment_org_name;index"`
	Name           string `gorm:"not null;uniqueIndex:idx_deployment_org_name"`
	// RouteKey is the immutable public hostname without AppDomain, for example
	// "llama-chat.default". It is shared by every public route owner.
	RouteKey   string `gorm:"not null;default:'';index"`
	TemplateID string `gorm:"index"`
	Status     string `gorm:"default:'DRAFT'"`

	ProviderID           string `gorm:"not null"`
	ProviderConnectionID string `gorm:"index"`

	InstanceName   string
	InstanceTypeID string

	RuntimeID    string
	Task         string
	ModelID      string
	WorkloadType string `gorm:"not null;default:'model_inference'"`
	Replicas     int    `gorm:"default:1"`

	// Zero means "use the ingress default for this workload type". These are
	// aggregate deployment limits, deliberately independent of replica count.
	RequestsPerMinute     int `gorm:"default:0"`
	BurstRequests         int `gorm:"default:0"`
	MaxConcurrentRequests int `gorm:"default:0"`

	ConfidentialMode bool   `gorm:"default:true"`
	JobSpecJSON      string `gorm:"type:text"`

	Nodes        []Node          `gorm:"foreignKey:DeploymentID"`
	Endpoints    []RouteEndpoint `gorm:"foreignKey:DeploymentID"`
	Organization Organization    `gorm:"foreignKey:OrganizationID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Node struct {
	ID           string `gorm:"primaryKey"` // Provider-assigned ID (Nosana deployment ID, AWS instance ID, etc.)
	DeploymentID string `gorm:"not null;index"`
	ProviderID   string `gorm:"not null"`
	InfraStatus  string `gorm:"default:'PENDING'"`
	AppStatus    string `gorm:"default:'PENDING'"`

	EndpointsJSON string `gorm:"type:text"` // JSON array of core.Endpoint
	NodeURL       string // Legacy/convenience: primary resolved URL

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
