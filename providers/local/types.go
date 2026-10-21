package local

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"orcn/core"
	"orcn/core/config"
	"orcn/core/secret"
	"orcn/models"
	"gorm.io/gorm"
)

type Client struct {
	mu            sync.RWMutex
	config        *core.ProviderConnectionConfig
	providerConnID string
	db            *gorm.DB
	cfg           *config.Config
}

type MachineConfig struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Endpoint      string `json:"endpoint"`
	ProxyEndpoint string `json:"proxy_endpoint,omitempty"`
	Token         string `json:"token"`
	Status        string `json:"status,omitempty"`
	LastSeen      string `json:"last_seen,omitempty"`
}

func New() *Client {
	return &Client{}
}

func (c *Client) SetProviderContext(providerConnID string, db *gorm.DB, cfg *config.Config) {
	c.mu.Lock()
	c.providerConnID = providerConnID
	c.db = db
	c.cfg = cfg
	c.mu.Unlock()
}

func (c *Client) getMachineByID(machineID string) (*MachineConfig, error) {
	c.mu.RLock()
	providerConnID := c.providerConnID
	db := c.db
	cfg := c.cfg
	c.mu.RUnlock()

	if providerConnID == "" || db == nil {
		return nil, fmt.Errorf("no provider context")
	}

	var m models.Machine
	if err := db.Where("id = ? AND provider_connection_id = ?", machineID, providerConnID).First(&m).Error; err != nil {
		return nil, fmt.Errorf("machine not found: %s", machineID)
	}

	token := ""
	if m.TokenSecretID != nil {
		var sec models.Secret
		if err := db.Where("id = ?", *m.TokenSecretID).First(&sec).Error; err == nil {
			if cfg != nil {
				plaintext, err := secret.DecryptSecret(context.Background(), cfg, sec.Ciphertext, sec.EncryptedDEK, sec.Nonce)
				if err == nil {
					token = string(plaintext)
				}
			}
		}
	}

	return &MachineConfig{
		ID:            m.ID,
		Name:          m.Name,
		Endpoint:      strings.TrimRight(strings.TrimSpace(m.Endpoint), "/"),
		ProxyEndpoint: strings.TrimRight(strings.TrimSpace(m.ProxyEndpoint), "/"),
		Token:         token,
		Status:        m.Status,
		LastSeen: func() string {
			if m.LastSeen != nil {
				return m.LastSeen.Format(time.RFC3339)
			}
			return ""
		}(),
	}, nil
}