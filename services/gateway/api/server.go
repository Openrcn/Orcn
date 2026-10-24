package api

import (
	"encoding/json"
	"net/http"

	"orcn/core/config"
	"orcn/core/logger"
	"orcn/services/gateway/health"

	"gorm.io/gorm"
)

var apiLog = logger.New("API")

type Server struct {
	DB     *gorm.DB
	Health *health.Checker
	Cfg    *config.Config
	mux    *http.ServeMux
}

func New(db *gorm.DB, hc *health.Checker, cfg *config.Config) *Server {
	s := &Server{
		DB:     db,
		Health: hc,
		Cfg:    cfg,
		mux:    http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) Handler() http.Handler {
	return withLogging(withCORS(s.mux))
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /api/v1/models/search", s.handleSearchModels)
	s.mux.HandleFunc("GET /api/v1/models/details", s.handleGetModelDetails)

	s.mux.HandleFunc("GET /api/v1/instances", s.handleListInstances)
	s.mux.HandleFunc("GET /api/v1/providers", s.handleListProviders)

	s.mux.HandleFunc("GET /api/v1/runtimes", s.handleListRuntimes)
	s.mux.HandleFunc("GET /api/v1/runtimes/schema", s.handleGetRuntimeSchema)

	s.mux.HandleFunc("POST /api/v1/deployments", s.handleCreateDeployment)
	s.mux.HandleFunc("GET /api/v1/deployments", s.handleListDeployments)
	s.mux.HandleFunc("GET /api/v1/deployments/{id}", s.handleGetDeployment)

	//node logging + metrics
	s.mux.HandleFunc("GET /api/v1/deployments/{id}/nodes/{node_id}/logs", s.handleNodeLogs)
	s.mux.HandleFunc("GET /api/v1/deployments/{id}/nodes/{node_id}/events", s.handleNodeEvents)
	s.mux.HandleFunc("GET /api/v1/deployments/{id}/nodes/{node_id}/metrics", s.handleNodeMetrics)

	s.mux.HandleFunc("POST /api/v1/deployments/{id}/action", s.handleDeploymentAction)

	s.mux.HandleFunc("POST /api/v1/workloads", s.handleCreateWorkload)

	s.mux.HandleFunc("GET /api/v1/secrets", s.handleListSecrets)
	s.mux.HandleFunc("POST /api/v1/secrets", s.handleCreateSecret)
	s.mux.HandleFunc("DELETE /api/v1/secrets/{name}", s.handleDeleteSecret)

	s.mux.HandleFunc("GET /api/v1/registries", s.handleListRegistries)
	s.mux.HandleFunc("POST /api/v1/registries", s.handleCreateRegistry)
	s.mux.HandleFunc("DELETE /api/v1/registries/{id}", s.handleDeleteRegistry)

	s.mux.HandleFunc("GET /api/v1/resource-providers", s.handleListResourceProviders)

	s.mux.HandleFunc("POST /api/v1/provider-connections", s.handleCreateProviderConnection)
	s.mux.HandleFunc("GET /api/v1/provider-connections", s.handleListProviderConnections)
	s.mux.HandleFunc("GET /api/v1/provider-connections/schema", s.handleGetProviderConnectionSchema)
	s.mux.HandleFunc("POST /api/v1/provider-connections/verify", s.handleVerifyProviderConnection)
	s.mux.HandleFunc("GET /api/v1/provider-connections/{id}", s.handleGetProviderConnection)
	s.mux.HandleFunc("GET /api/v1/provider-connections/{id}/config", s.handleGetProviderConnectionConfig)
	s.mux.HandleFunc("PUT /api/v1/provider-connections/{id}", s.handleUpdateProviderConnection)
	s.mux.HandleFunc("POST /api/v1/provider-connections/{id}/verify", s.handleVerifyProviderConnectionByID)
	s.mux.HandleFunc("DELETE /api/v1/provider-connections/{id}", s.handleDeleteProviderConnection)

	// Machine routes
	s.mux.HandleFunc("GET /api/v1/provider-connections/{id}/machines", s.handleListMachines)
	s.mux.HandleFunc("POST /api/v1/provider-connections/{id}/machines", s.handleCreateMachine)
	s.mux.HandleFunc("PUT /api/v1/provider-connections/{id}/machines/{machineId}", s.handleUpdateMachine)
	s.mux.HandleFunc("DELETE /api/v1/provider-connections/{id}/machines/{machineId}", s.handleDeleteMachine)

	s.mux.HandleFunc("GET /api/v1/organizations", s.handleListOrganizations)
	s.mux.HandleFunc("GET /api/v1/organizations/{id}", s.handleGetOrganization)

	s.mux.HandleFunc("GET /api/v1/templates", s.handleListTemplates)
	s.mux.HandleFunc("POST /api/v1/templates", s.handleCreateTemplate)
	s.mux.HandleFunc("GET /api/v1/templates/{id}", s.handleGetTemplate)
	s.mux.HandleFunc("PUT /api/v1/templates/{id}", s.handleUpdateTemplate)
	s.mux.HandleFunc("DELETE /api/v1/templates/{id}", s.handleDeleteTemplate)

	s.mux.HandleFunc("GET /api/v1/endpoints", s.handleListEndpoints)

	// Internal service endpoints
	// TODO Polling -> WebSockets|sse or ?watch=true
	s.mux.HandleFunc("GET /api/v1/internal/routes", s.handleGetInternalRoutes)
}

func respondJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
	// byte , _ := json.Marshal(data)
	// w.Write(byte)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
