package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"orcn/core"
	"orcn/core/providerconn"
	"orcn/models"
	"orcn/pkg/routing"

	"gorm.io/gorm"
)

type createDeploymentRequest struct {
	Name                  string            `json:"name"`
	TemplateID            string            `json:"template_id"`
	ProviderID            string            `json:"provider_id"`
	ProviderConnectionID  string            `json:"provider_connection_id"`
	InstanceTypeID        string            `json:"instance_type_id"`
	InstanceName          string            `json:"instance_name"`
	RuntimeID             string            `json:"runtime_id"`
	Task                  core.ModelTask    `json:"task,omitempty"`
	ModelID               string            `json:"model_id"`
	Replicas              int               `json:"replicas"`
	HFToken               string            `json:"hf_token,omitempty"`
	AdvancedConfig        map[string]string `json:"advanced_config,omitempty"`
	RequestsPerMinute     int               `json:"requests_per_minute,omitempty"`
	BurstRequests         int               `json:"burst_requests,omitempty"`
	MaxConcurrentRequests int               `json:"max_concurrent_requests,omitempty"`
}

type actionRequest struct {
	Action         string `json:"action"`
	TimeoutMinutes int    `json:"timeout_minutes,omitempty"`
}

func validateDeploymentName(name, organizationID string, db *gorm.DB) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("deployment name is required")
	}
	var count int64
	db.Model(&models.Deployment{}).Where("organization_id = ? AND name = ?", organizationID, name).Count(&count)
	if count > 0 {
		return fmt.Errorf("a deployment with the name '%s' already exists in this organization", name)
	}
	return nil
}

func (s *Server) resolveDeploymentRoute(r *http.Request, name string) (string, string, error) {
	organizationID := getOrgID(r)
	var organization models.Organization
	if err := s.DB.Where("id = ?", organizationID).First(&organization).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", "", fmt.Errorf("organization not found")
		}
		return "", "", err
	}
	routeKey, err := routing.DeploymentRouteKey(name, organization.Name)
	if err != nil {
		return "", "", err
	}
	var count int64
	s.DB.Model(&models.Deployment{}).Where("route_key = ?", routeKey).Count(&count)
	if count > 0 {
		return "", "", fmt.Errorf("public route '%s' is already in use", routeKey)
	}
	return organizationID, routeKey, nil
}

func (s *Server) validateIngressPolicy(kind string, rpm, burst, concurrent int) error {
	if rpm < 0 || burst < 0 || concurrent < 0 {
		return fmt.Errorf("ingress policy values cannot be negative")
	}
	maxRPM := s.Cfg.IngressInferenceMaxRPM
	if kind == "workload" {
		maxRPM = s.Cfg.IngressWorkloadMaxRPM
	}
	if rpm > maxRPM {
		return fmt.Errorf("requests_per_minute exceeds the %s maximum of %d", kind, maxRPM)
	}
	return nil
}

func (s *Server) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req createDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	organizationID, routeKey, err := s.resolveDeploymentRoute(r, req.Name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateDeploymentName(req.Name, organizationID, s.DB); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.validateIngressPolicy("model_inference", req.RequestsPerMinute, req.BurstRequests, req.MaxConcurrentRequests); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	runtime, err := core.GetRuntime(req.RuntimeID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid runtime_id")
		return
	}

	provider, err := providerconn.Resolve(r.Context(), s.DB, s.Cfg, req.ProviderID, req.ProviderConnectionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid provider or provider_connection_id: "+err.Error())
		return
	}

	req.Task = core.NormalizeModelTask(req.Task)
	spec, err := runtime.BuildJobSpec(req.ModelID, req.Task, req.AdvancedConfig)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to build job spec: "+err.Error())
		return
	}

	if req.HFToken != "" {
		for i := range spec.Containers {
			if spec.Containers[i].Args.Env == nil {
				spec.Containers[i].Args.Env = make(map[string]string)
			}
			spec.Containers[i].Args.Env["HF_TOKEN"] = req.HFToken
		}
	}

	if req.Replicas <= 0 {
		req.Replicas = 1
	}

	specBytes, _ := json.Marshal(spec)
	specJSON := string(specBytes)

	// 1. Create the parent Deployment row
	deploymentID := fmt.Sprintf("dep-%s-%d", req.Name, time.Now().UnixMilli())
	dbDeployment := models.Deployment{
		ID:                    deploymentID,
		OrganizationID:        organizationID,
		Name:                  req.Name,
		RouteKey:              routeKey,
		TemplateID:            req.TemplateID,
		Status:                models.DeploymentDraft,
		ProviderID:            req.ProviderID,
		ProviderConnectionID:  req.ProviderConnectionID,
		InstanceName:          req.InstanceName,
		InstanceTypeID:        req.InstanceTypeID,
		RuntimeID:             req.RuntimeID,
		Task:                  string(req.Task),
		ModelID:               req.ModelID,
		WorkloadType:          "model_inference",
		Replicas:              req.Replicas,
		RequestsPerMinute:     req.RequestsPerMinute,
		BurstRequests:         req.BurstRequests,
		MaxConcurrentRequests: req.MaxConcurrentRequests,
		JobSpecJSON:           specJSON,
	}
	if err := s.DB.Create(&dbDeployment).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save deployment: "+err.Error())
		return
	}

	// 2. Loop to create Nodes (each provider job = 1 Node)
	var createdNodeIDs []string
	var lastErr error

	for i := 0; i < req.Replicas; i++ {
		providerJobID, err := provider.CreateDeployment(req.Name, req.InstanceTypeID, spec)
		if err != nil {
			lastErr = err
			break
		}

		// For local provider, use instanceTypeID (machine ID) as ProviderID
		nodeProviderID := req.ProviderID
		if req.ProviderID == "local" {
			nodeProviderID = req.InstanceTypeID
		}

		node := models.Node{
			ID:           providerJobID,
			DeploymentID: deploymentID,
			ProviderID:   nodeProviderID,
			InfraStatus:  models.InfraPending,
			AppStatus:    models.AppPending,
		}
		if err := s.DB.Create(&node).Error; err != nil {
			lastErr = fmt.Errorf("failed to save node to db: %v", err)
			break
		}
		createdNodeIDs = append(createdNodeIDs, providerJobID)
	}

	if len(createdNodeIDs) == 0 && lastErr != nil {
		s.DB.Delete(&dbDeployment)
		respondError(w, http.StatusInternalServerError, "Failed to create nodes: "+lastErr.Error())
		return
	}

	if lastErr != nil {
		dbDeployment.Status = models.DeploymentPartial
		dbDeployment.Replicas = len(createdNodeIDs)
	}
	s.DB.Save(&dbDeployment)

	response := map[string]interface{}{
		"deployment_id": deploymentID,
		"node_ids":      createdNodeIDs,
		"status":        dbDeployment.Status,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
	}

	if lastErr != nil {
		response["warning"] = fmt.Sprintf(
			"Partial failure: Only created %d of %d nodes. Error: %s",
			len(createdNodeIDs), req.Replicas, lastErr.Error(),
		)
	}

	respondJSON(w, http.StatusCreated, response)
}

func (s *Server) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	idOrName := r.PathValue("id")
	orgID := getOrgID(r)

	var dep models.Deployment
	if err := s.DB.Preload("Nodes").Preload("Endpoints").Where("organization_id = ? AND (id = ? OR name = ?)", orgID, idOrName, idOrName).First(&dep).Error; err != nil {
		respondError(w, http.StatusNotFound, "Deployment not found")
		return
	}

	respondJSON(w, http.StatusOK, dep)
}

func (s *Server) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	var deployments []models.Deployment
	orgID := getOrgID(r)

	query := s.DB.
		Select(
			"id",
			"organization_id",
			"name",
			"route_key",
			"template_id",
			"status",
			"provider_id",
			"provider_connection_id",
			"instance_name",
			"instance_type_id",
			"runtime_id",
			"task",
			"model_id",
			"workload_type",
			"replicas",
			"requests_per_minute",
			"burst_requests",
			"max_concurrent_requests",
			"confidential_mode",
			"created_at",
			"updated_at",
		).
		Preload("Nodes").
		Preload("Endpoints")

	if err := query.Where("organization_id = ?", orgID).Find(&deployments).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch deployments")
		return
	}

	respondJSON(w, http.StatusOK, deployments)
}

func (s *Server) handleDeploymentAction(w http.ResponseWriter, r *http.Request) {
	deploymentID := r.PathValue("id")
	orgID := getOrgID(r)

	var dep models.Deployment
	if err := s.DB.Preload("Nodes").Where("id = ? AND organization_id = ?", deploymentID, orgID).First(&dep).Error; err != nil {
		respondError(w, http.StatusNotFound, "Deployment not found in DB")
		return
	}

	provider, err := providerconn.Resolve(r.Context(), s.DB, s.Cfg, dep.ProviderID, dep.ProviderConnectionID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Provider missing: "+err.Error())
		return
	}

	var spec core.JobSpec
	if err := json.Unmarshal([]byte(dep.JobSpecJSON), &spec); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to parse job spec: "+err.Error())
		return
	}

	var req actionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	var lastErr error
	for _, node := range dep.Nodes {
		var actionErr error
		switch req.Action {
		case "start":
			if p, ok := provider.(interface {
				StartDeploymentWithSpec(string, *core.JobSpec) error
			}); ok {
				actionErr = p.StartDeploymentWithSpec(node.ID, &spec)
			} else {
				actionErr = provider.StartDeployment(node.ID)
			}
		case "stop":
			if p, ok := provider.(interface {
				StopDeploymentWithSpec(string, *core.JobSpec) error
			}); ok {
				actionErr = p.StopDeploymentWithSpec(node.ID, &spec)
			} else {
				actionErr = provider.StopDeployment(node.ID)
			}
		case "update_timeout":
			if req.TimeoutMinutes == 0 {
				respondError(w, http.StatusBadRequest, "timeout_minutes required for update_timeout action")
				return
			}
			actionErr = provider.UpdateTimeout(node.ID, req.TimeoutMinutes)
		default:
			respondError(w, http.StatusBadRequest, "Unknown action")
			return
		}

		if actionErr != nil {
			lastErr = actionErr
		}
	}

	if lastErr != nil {
		respondError(w, http.StatusInternalServerError, "Action failed on one or more nodes: "+lastErr.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"deployment_id": deploymentID,
		"message":       "Action successfully processed",
	})
}

func (s *Server) handleGetInternalRoutes(w http.ResponseWriter, r *http.Request) {
	if s.Cfg.InternalAPIToken != "" {
		expected := "Bearer " + s.Cfg.InternalAPIToken
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), []byte(expected)) != 1 {
			respondError(w, http.StatusUnauthorized, "unauthorized internal request")
			return
		}
	}
	var deps []models.Deployment
	if err := s.DB.Preload("Nodes").Where("status IN ?", []string{models.DeploymentReady, models.DeploymentRunning}).Find(&deps).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch internal routes")
		return
	}
	respondJSON(w, http.StatusOK, deps)
}

// Helper Functions for Endpoint Generation
func (s *Server) generateDeploymentEndpoints(deploymentID, name string, ports []int) {
	for i, port := range ports {
		s.DB.Create(&models.RouteEndpoint{
			ID:           fmt.Sprintf("ep-dep-%s-%d", deploymentID, port),
			Subdomain:    fmt.Sprintf("%s-%d", name, port),
			TargetPort:   port,
			Type:         models.EndpointTypeDeployment,
			DeploymentID: deploymentID,
		})
		// First port acts as the primary wildcard route
		if i == 0 {
			s.DB.Create(&models.RouteEndpoint{
				ID:           fmt.Sprintf("ep-dep-%s-primary", deploymentID),
				Subdomain:    name,
				TargetPort:   port,
				Type:         models.EndpointTypeDeployment,
				DeploymentID: deploymentID,
			})
		}
	}
}

func (s *Server) generateNodeEndpoints(deploymentID, nodeID string, ports []int) {
	shortNodeID := nodeID
	if len(shortNodeID) > 8 {
		shortNodeID = shortNodeID[:8]
	}
	for i, port := range ports {
		s.DB.Create(&models.RouteEndpoint{
			ID:           fmt.Sprintf("ep-node-%s-%d", shortNodeID, port),
			Subdomain:    fmt.Sprintf("%s-%d", shortNodeID, port),
			TargetPort:   port,
			Type:         models.EndpointTypeNode,
			DeploymentID: deploymentID,
			NodeID:       nodeID,
		})
		// First port acts as the primary wildcard route for this node
		if i == 0 {
			s.DB.Create(&models.RouteEndpoint{
				ID:           fmt.Sprintf("ep-node-%s-primary", shortNodeID),
				Subdomain:    shortNodeID,
				TargetPort:   port,
				Type:         models.EndpointTypeNode,
				DeploymentID: deploymentID,
				NodeID:       nodeID,
			})
		}
	}
}

type createWorkloadRequest struct {
	Name                  string          `json:"name"`
	TemplateID            string          `json:"template_id"`
	ProviderID            string          `json:"provider_id"`
	ProviderConnectionID  string          `json:"provider_connection_id"`
	InstanceTypeID        string          `json:"instance_type_id"`
	InstanceName          string          `json:"instance_name"`
	Replicas              int             `json:"replicas"`
	RequestsPerMinute     int             `json:"requests_per_minute,omitempty"`
	BurstRequests         int             `json:"burst_requests,omitempty"`
	MaxConcurrentRequests int             `json:"max_concurrent_requests,omitempty"`
	Spec                  json.RawMessage `json:"spec"`
}

func (s *Server) handleCreateWorkload(w http.ResponseWriter, r *http.Request) {
	var req createWorkloadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	organizationID, routeKey, err := s.resolveDeploymentRoute(r, req.Name)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := validateDeploymentName(req.Name, organizationID, s.DB); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.validateIngressPolicy("workload", req.RequestsPerMinute, req.BurstRequests, req.MaxConcurrentRequests); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	provider, err := providerconn.Resolve(r.Context(), s.DB, s.Cfg, req.ProviderID, req.ProviderConnectionID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "Invalid provider or provider_connection_id: "+err.Error())
		return
	}

	if req.Replicas <= 0 {
		req.Replicas = 1
	}

	// 1. Strict Validation of the incoming Raw Spec!
	valid, validationErrors := core.ValidateJobSpecBytes(req.Spec)
	if !valid {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "Spec validation failed",
			"details": validationErrors,
		})
		return
	}

	var v2Spec core.TemplateSpecV2
	_ = json.Unmarshal(req.Spec, &v2Spec)

	// 2. Convert to Internal Provider Format
	internalSpec := core.ConvertTemplateSpecV2ToJobSpec(&v2Spec)

	// 3. Create the Database Record to archive this deployment
	deploymentID := fmt.Sprintf("workload-%s-%d", req.Name, time.Now().UnixMilli())
	dbDeployment := models.Deployment{
		ID:                    deploymentID,
		OrganizationID:        organizationID,
		Name:                  req.Name,
		RouteKey:              routeKey,
		TemplateID:            req.TemplateID,
		Status:                models.DeploymentDraft,
		ProviderID:            req.ProviderID,
		ProviderConnectionID:  req.ProviderConnectionID,
		InstanceName:          req.InstanceName,
		InstanceTypeID:        req.InstanceTypeID,
		WorkloadType:          "workload",
		Replicas:              req.Replicas,
		RequestsPerMinute:     req.RequestsPerMinute,
		BurstRequests:         req.BurstRequests,
		MaxConcurrentRequests: req.MaxConcurrentRequests,
		JobSpecJSON:           string(req.Spec),
	}

	if err := s.DB.Create(&dbDeployment).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to save deployment to db: "+err.Error())
		return
	}

	// 3.5 Generate Database Endpoints for UI to query
	var ports []int
	for _, c := range internalSpec.Containers {
		for _, p := range c.Args.Expose {
			ports = append(ports, p.Port)
		}
	}
	s.generateDeploymentEndpoints(dbDeployment.ID, routeKey, ports)

	// 4. Launch on the infrastructure
	var createdNodeIDs []string
	var lastErr error

	for i := 0; i < req.Replicas; i++ {
		providerJobID, err := provider.CreateDeployment(req.Name, req.InstanceTypeID, internalSpec)
		if err != nil {
			lastErr = err
			break
		}

		nodeProviderID := req.ProviderID
		if req.ProviderID == "local" {
			nodeProviderID = req.InstanceTypeID
		}

		node := models.Node{
			ID:           providerJobID,
			DeploymentID: deploymentID,
			ProviderID:   nodeProviderID,
			InfraStatus:  models.InfraPending,
			AppStatus:    models.AppPending,
		}
		if err := s.DB.Create(&node).Error; err != nil {
			lastErr = fmt.Errorf("failed to save node to db: %v", err)
			break
		}
		createdNodeIDs = append(createdNodeIDs, providerJobID)

		s.generateNodeEndpoints(dbDeployment.ID, providerJobID, ports)
	}

	if len(createdNodeIDs) == 0 && lastErr != nil {
		s.DB.Delete(&dbDeployment)
		respondError(w, http.StatusInternalServerError, "Failed to deploy on provider: "+lastErr.Error())
		return
	}

	if lastErr != nil {
		dbDeployment.Status = models.DeploymentPartial
		dbDeployment.Replicas = len(createdNodeIDs)
	}
	s.DB.Save(&dbDeployment)

	response := map[string]any{
		"deployment_id": deploymentID,
		"node_ids":      createdNodeIDs,
		"status":        dbDeployment.Status,
		"created_at":    time.Now().UTC().Format(time.RFC3339),
	}

	if lastErr != nil {
		response["warning"] = fmt.Sprintf(
			"Partial failure: Only created %d of %d replicas. Error: %s",
			len(createdNodeIDs), req.Replicas, lastErr.Error(),
		)
	}

	respondJSON(w, http.StatusCreated, response)
}
