package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppDomain         string
	APIPort           int
	IngressPort       int
	DatabaseDSN       string
	AgentBaseURL      string
	ControlPlaneURL   string

	NosanaURL         string `json:"NOSANA_URL"`
	KMSProvider       string `json:"KMS_PROVIDER"`
	KMSMasterKey      string `json:"KMS_MASTER_KEY"`

	MaxConnsPerHost        int
	MaxActiveRequests      int
	GatewaySyncIntervalSec int
	ControllerSyncInterval int
	ProxyTimeoutSec        int

	IngressWorkloadDefaultRPM         int
	IngressInferenceDefaultRPM        int
	IngressWorkloadMaxRPM             int
	IngressInferenceMaxRPM            int
	IngressWorkloadDefaultConcurrent  int
	IngressInferenceDefaultConcurrent int
	IngressWorkloadGlobalRPM          int
	IngressInferenceGlobalRPM         int
	IngressWorkloadGlobalConcurrent   int
	IngressInferenceGlobalConcurrent  int
	InternalAPIToken                  string
}

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	internalToken := getEnv("INTERNAL_API_TOKEN", "orcn-dev-internal-token")
	if internalToken == "" {
		internalToken = "orcn-dev-internal-token"
	}

	cfg := &Config{
		AppDomain:        getEnv("APP_DOMAIN", "orcn.dev"),
		APIPort:          getEnvAsInt("API_PORT", 8080),
		IngressPort:      getEnvAsInt("INGRESS_PORT", 80),
		DatabaseDSN:      getEnv("DATABASE_DSN", "orcn.db"),
		AgentBaseURL:     getEnv("AGENT_BASE_URL", "http://127.0.0.1:4000"),
		ControlPlaneURL:  getEnv("CONTROL_PLANE_URL", "http://127.0.0.1:8080"),

		KMSProvider: getEnv("KMS_PROVIDER", "local"),
		KMSMasterKey: getEnv("KMS_MASTER_KEY", ""),

		MaxConnsPerHost:        getEnvAsInt("GATEWAY_MAX_CONNS_PER_HOST", 20),
		MaxActiveRequests:      getEnvAsInt("GATEWAY_MAX_ACTIVE_REQUESTS", 100),
		GatewaySyncIntervalSec: getEnvAsInt("GATEWAY_SYNC_INTERVAL_SEC", 5),
		ControllerSyncInterval: getEnvAsInt("CONTROLLER_SYNC_INTERVAL_SEC", 15),
		ProxyTimeoutSec:        getEnvAsInt("GATEWAY_PROXY_TIMEOUT_SEC", 60),

		IngressWorkloadDefaultRPM:         getEnvAsInt("INGRESS_WORKLOAD_DEFAULT_RPM", 6000),
		IngressInferenceDefaultRPM:        getEnvAsInt("INGRESS_INFERENCE_DEFAULT_RPM", 120),
		IngressWorkloadMaxRPM:             getEnvAsInt("INGRESS_WORKLOAD_MAX_RPM", 30000),
		IngressInferenceMaxRPM:            getEnvAsInt("INGRESS_INFERENCE_MAX_RPM", 1200),
		IngressWorkloadDefaultConcurrent:  getEnvAsInt("INGRESS_WORKLOAD_DEFAULT_CONCURRENT", 500),
		IngressInferenceDefaultConcurrent: getEnvAsInt("INGRESS_INFERENCE_DEFAULT_CONCURRENT", 32),
		IngressWorkloadGlobalRPM:          getEnvAsInt("INGRESS_WORKLOAD_GLOBAL_RPM", 0),
		IngressInferenceGlobalRPM:         getEnvAsInt("INGRESS_INFERENCE_GLOBAL_RPM", 0),
		IngressWorkloadGlobalConcurrent:   getEnvAsInt("INGRESS_WORKLOAD_GLOBAL_CONCURRENT", 0),
		IngressInferenceGlobalConcurrent:  getEnvAsInt("INGRESS_INFERENCE_GLOBAL_CONCURRENT", 0),
		InternalAPIToken:                  internalToken,
	}

	return cfg, nil
}

func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return strings.TrimSpace(value)
	}
	return defaultVal
}

func getEnvAsInt(key string, defaultVal int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
			return intVal
		}
	}
	return defaultVal
}