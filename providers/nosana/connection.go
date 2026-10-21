package nosana

import (
	"fmt"

	"orcn/core"
	"orcn/pkg/util"
)

func (c *Client) ConnectionSchema() []core.ProviderField {
	return []core.ProviderField{
		{
			Key:         "api_key",
			Name:        "API Key",
			Description: "Nosana API key used to authenticate requests.",
			Type:        "password",
			Required:    true,
			Secret:      true,
			Placeholder: "nos_xxxxxxxxxxxxxxxx",
		},
		{
			Key:         "base_url",
			Name:        "Base URL",
			Description: "Override the Nosana API endpoint (optional).",
			Type:        "text",
			Required:    false,
			Secret:      false,
			Default:     "https://api.nosana.com",
		},
	}
}

func (c *Client) ProcessConnection(raw map[string]any) (*core.ProviderConnectionConfig, error) {
	apiKey := util.StringValue(raw["api_key"])
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	config := map[string]any{}
	if base := util.StringValue(raw["base_url"]); base != "" {
		config["base_url"] = base
	}

	return &core.ProviderConnectionConfig{
		Config: config,
		Secret: map[string]any{"api_key": apiKey},
	}, nil
}

func (c *Client) VerifyConnection(cfg *core.ProviderConnectionConfig) error {
	if cfg == nil || cfg.Secret == nil {
		return fmt.Errorf("missing credentials")
	}
	apiKey := util.StringValue(cfg.Secret["api_key"])
	if apiKey == "" {
		return fmt.Errorf("api_key is required")
	}

	baseURL := ""
	if cfg.Config != nil {
		baseURL = util.StringValue(cfg.Config["base_url"])
	}

	client := New(baseURL)
	client.APIKey = apiKey
	if _, err := client.request("GET", "/markets", nil); err != nil {
		return fmt.Errorf("nosana verification failed: %w", err)
	}
	return nil
}

// WithConfig returns a Nosana client bound to the supplied connection credentials.
func (c *Client) WithConfig(cfg *core.ProviderConnectionConfig) (core.Provider, error) {
	if cfg == nil || cfg.Secret == nil {
		return nil, fmt.Errorf("missing credentials")
	}
	apiKey := util.StringValue(cfg.Secret["api_key"])
	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	baseURL := ""
	if cfg.Config != nil {
		baseURL = util.StringValue(cfg.Config["base_url"])
	}
	if baseURL == "" {
		baseURL = c.BaseURL
	}

	client := New(baseURL)
	client.APIKey = apiKey
	return client, nil
}