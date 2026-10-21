package local

import (
	"context"
	"math"
	"strings"
	"time"

	"orcn/core/secret"
	"orcn/models"
)

func (c *Client) getMachines() ([]MachineConfig, error) {
	c.mu.RLock()
	providerConnID := c.providerConnID
	db := c.db
	c.mu.RUnlock()

	if providerConnID == "" || db == nil {
		return []MachineConfig{}, nil
	}

	var machines []models.Machine
	if err := db.Where("provider_connection_id = ?", providerConnID).Find(&machines).Error; err != nil {
		return nil, err
	}

	result := make([]MachineConfig, 0, len(machines))
	for _, m := range machines {
		token := ""
		if m.TokenSecretID != nil {
			var sec models.Secret
			if err := db.Where("id = ?", *m.TokenSecretID).First(&sec).Error; err == nil {
				c.mu.RLock()
				cfg := c.cfg
				c.mu.RUnlock()
				if cfg != nil {
					plaintext, err := secret.DecryptSecret(context.Background(), cfg, sec.Ciphertext, sec.EncryptedDEK, sec.Nonce)
					if err == nil {
						token = string(plaintext)
					}
				}
			}
		}
		result = append(result, MachineConfig{
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
		})
	}
	return result, nil
}

func (c *Client) fetchMachineCapabilities(m MachineConfig) (map[string]any, string) {
	ctx := context.Background()
	client := NewNodeAgentClient(m.Endpoint, m.Token)
	caps, err := client.GetCapabilities(ctx)
	if err != nil {
		return nil, "unreachable"
	}
	return caps, "healthy"
}

func (c *Client) GetInstanceTypes() (any, error) {
	machines, err := c.getMachines()
	if err != nil {
		return []map[string]any{}, nil
	}

	types := make([]map[string]any, 0, len(machines))
	for _, m := range machines {
		caps, status := c.fetchMachineCapabilities(m)

		instance := map[string]any{
			"id":        m.ID,
			"name":      m.Name,
			"endpoint":  m.Endpoint,
			"status":    status,
			"last_seen": m.LastSeen,
		}

		if caps != nil {
			if cpu, ok := caps["cpu"].(string); ok {
				instance["cpu"] = cpu
			}
			if logicalCPUs, ok := caps["logical_cpus"].(float64); ok {
				instance["cpu_cores"] = logicalCPUs
			}
			if memoryTotal, ok := caps["memory_total_gb"].(float64); ok {
				instance["ram_gb"] = math.Round(memoryTotal*10) / 10
			}
			if diskTotal, ok := caps["disk_total_gb"].(float64); ok {
				instance["storage_gb"] = math.Round(diskTotal*10) / 10
			}
			if arch, ok := caps["architecture"].(string); ok {
				instance["architecture"] = arch
			}
			if os, ok := caps["os"].(string); ok {
				instance["os"] = os
			}
			if docker, ok := caps["docker"].(map[string]any); ok {
				if dockerRunning, ok := docker["running"].(bool); ok {
					instance["docker_running"] = dockerRunning
				}
			}
			if gpuModel, ok := caps["gpu_model"].(string); ok && gpuModel != "" {
				instance["gpu_model"] = gpuModel
			}
			if gpuVRAM, ok := caps["gpu_vram_gb"].(float64); ok {
				instance["vram_gb"] = math.Round(gpuVRAM*10) / 10
			}
		}

		types = append(types, instance)
	}
	return types, nil
}

func (c *Client) MachineStatuses() (map[string]string, error) {
	machines, err := c.getMachines()
	if err != nil {
		return nil, err
	}
	statuses := make(map[string]string, len(machines))
	for _, m := range machines {
		_, status := c.fetchMachineCapabilities(m)
		statuses[m.ID] = status
	}
	return statuses, nil
}