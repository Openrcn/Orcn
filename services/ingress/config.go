package ingress

import (
	"time"

	appconfig "orcn/core/config"
)

type Config struct {
	AppDomain string

	SyncInterval     time.Duration
	RouteStaleAfter  time.Duration
	InternalAPIToken string
	MaxConnsPerHost  int
}

func ConfigFromAppConfig(cfg *appconfig.Config) Config {
	return Config{
		AppDomain:     cfg.AppDomain,
		SyncInterval:  time.Duration(cfg.GatewaySyncIntervalSec) * time.Second,
		RouteStaleAfter: time.Duration(300) * time.Second,
		InternalAPIToken: "orcn-dev-internal-token",
		MaxConnsPerHost:  cfg.MaxConnsPerHost,
	}
}