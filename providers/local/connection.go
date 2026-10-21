package local

import (
	"fmt"

	"orcn/core"
	"orcn/pkg/util"
)

func (c *Client) ConnectionSchema() []core.ProviderField {
	return []core.ProviderField{
		{
			Key:         "name",
			Name:        "Connection Name",
			Description: "A friendly name for this local infrastructure connection",
			Type:        "text",
			Required:    true,
			Secret:      false,
			Placeholder: "my-local-cluster",
		},
	}
}

func (c *Client) ProcessConnection(raw map[string]any) (*core.ProviderConnectionConfig, error) {
	name := util.StringValue(raw["name"])
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	return &core.ProviderConnectionConfig{
		Config: map[string]any{"name": name},
		Secret: map[string]any{},
	}, nil
}

func (c *Client) VerifyConnection(cfg *core.ProviderConnectionConfig) error {
	if cfg == nil || cfg.Config == nil {
		return fmt.Errorf("missing config")
	}
	name := util.StringValue(cfg.Config["name"])
	if name == "" {
		return fmt.Errorf("name is required")
	}
	return nil
}

func (c *Client) WithConfig(cfg *core.ProviderConnectionConfig) (core.Provider, error) {
	if cfg == nil || cfg.Config == nil {
		return nil, fmt.Errorf("missing config")
	}
	c.mu.Lock()
	c.config = cfg
	c.mu.Unlock()
	return c, nil
}