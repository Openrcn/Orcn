package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"orcn/core/logger"
	"orcn/core/providerconn"
	"orcn/core/secret"
	"orcn/models"
)

var machineLog = logger.New("MACHINES")

func (s *Server) handleListMachines(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("id")
	orgID := getOrgID(r)

	var pc models.ProviderConnection
	if err := s.DB.Where("id = ? AND organization_id = ?", connID, orgID).First(&pc).Error; err != nil {
		respondError(w, http.StatusNotFound, "provider connection not found")
		return
	}

	var machines []models.Machine
	if err := s.DB.Where("provider_connection_id = ?", connID).Find(&machines).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list machines")
		return
	}

	if provider, err := providerconn.Resolve(r.Context(), s.DB, s.Cfg, pc.Provider, connID); err == nil {
		if statuser, ok := provider.(interface{ MachineStatuses() (map[string]string, error) }); ok {
			if statuses, err := statuser.MachineStatuses(); err == nil {
				now := time.Now().UTC()
				for i := range machines {
					if st, ok := statuses[machines[i].ID]; ok {
						machines[i].Status = st
						if st == "healthy" {
							machines[i].LastSeen = &now
							s.DB.Model(&models.Machine{}).Where("id = ?", machines[i].ID).Updates(map[string]any{
								"status":    st,
								"last_seen": now,
							})
						}
					}
				}
			}
		}
	}

	respondJSON(w, http.StatusOK, map[string]any{"machines": machines})
}

func (s *Server) handleCreateMachine(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("id")
	orgID := getOrgID(r)

	var pc models.ProviderConnection
	if err := s.DB.Where("id = ? AND organization_id = ?", connID, orgID).First(&pc).Error; err != nil {
		respondError(w, http.StatusNotFound, "provider connection not found")
		return
	}

	var input struct {
		Name          string `json:"name"`
		Endpoint      string `json:"endpoint"`
		ProxyEndpoint string `json:"proxy_endpoint"`
		Token         string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Endpoint = strings.TrimSpace(input.Endpoint)
	input.ProxyEndpoint = strings.TrimSpace(input.ProxyEndpoint)
	input.Token = strings.TrimSpace(input.Token)

	if input.Name == "" || input.Endpoint == "" || input.Token == "" {
		respondError(w, http.StatusBadRequest, "name, endpoint, and token are required")
		return
	}

	ciphertext, encryptedDEK, nonce, err := secret.EncryptSecret(r.Context(), s.Cfg, []byte(input.Token))
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to encrypt token")
		return
	}
	tokenSecret := models.Secret{
		Name:            "machine_token_" + connID + "_" + input.Name,
		Ciphertext:      ciphertext,
		EncryptedDEK:    encryptedDEK,
		Nonce:           nonce,
		KeyVersion:      "v1",
		OrganizationID:  orgID,
		CreatedBy:       getUserID(r),
	}
	if err := s.DB.Create(&tokenSecret).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create token secret")
		return
	}

	machine := models.Machine{
		ProviderConnectionID: connID,
		Name:                 input.Name,
		Endpoint:             input.Endpoint,
		ProxyEndpoint:        input.ProxyEndpoint,
		TokenSecretID:        &tokenSecret.ID,
		Status:               "unknown",
	}
	if err := s.DB.Create(&machine).Error; err != nil {
		s.DB.Delete(&tokenSecret)
		respondError(w, http.StatusInternalServerError, "failed to create machine")
		return
	}

	respondJSON(w, http.StatusCreated, machine)
}

func (s *Server) handleUpdateMachine(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("id")
	machineID := r.PathValue("machineId")
	orgID := getOrgID(r)

	var pc models.ProviderConnection
	if err := s.DB.Where("id = ? AND organization_id = ?", connID, orgID).First(&pc).Error; err != nil {
		respondError(w, http.StatusNotFound, "provider connection not found")
		return
	}

	var machine models.Machine
	if err := s.DB.Where("id = ? AND provider_connection_id = ?", machineID, connID).First(&machine).Error; err != nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	var input struct {
		Name          *string `json:"name"`
		Endpoint      *string `json:"endpoint"`
		ProxyEndpoint *string `json:"proxy_endpoint"`
		Token         *string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if input.Name != nil {
		machine.Name = strings.TrimSpace(*input.Name)
	}
	if input.Endpoint != nil {
		machine.Endpoint = strings.TrimSpace(*input.Endpoint)
	}
	if input.ProxyEndpoint != nil {
		machine.ProxyEndpoint = strings.TrimSpace(*input.ProxyEndpoint)
	}
	if input.Token != nil && *input.Token != "" {
		token := strings.TrimSpace(*input.Token)
		ciphertext, encryptedDEK, nonce, err := secret.EncryptSecret(r.Context(), s.Cfg, []byte(token))
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to encrypt token")
			return
		}
		if machine.TokenSecretID != nil {
			var sec models.Secret
			if err := s.DB.Where("id = ?", *machine.TokenSecretID).First(&sec).Error; err == nil {
				sec.Ciphertext = ciphertext
				sec.EncryptedDEK = encryptedDEK
				sec.Nonce = nonce
				sec.KeyVersion = "v1"
				sec.CreatedBy = getUserID(r)
				s.DB.Save(&sec)
			}
		} else {
			tokenSecret := models.Secret{
				Name:            "machine_token_" + connID + "_" + machine.Name,
				Ciphertext:      ciphertext,
				EncryptedDEK:    encryptedDEK,
				Nonce:           nonce,
				KeyVersion:      "v1",
				OrganizationID:  orgID,
				CreatedBy:       getUserID(r),
			}
			if err := s.DB.Create(&tokenSecret).Error; err == nil {
				machine.TokenSecretID = &tokenSecret.ID
			}
		}
	}

	if err := s.DB.Save(&machine).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update machine")
		return
	}

	respondJSON(w, http.StatusOK, machine)
}

func (s *Server) handleDeleteMachine(w http.ResponseWriter, r *http.Request) {
	connID := r.PathValue("id")
	machineID := r.PathValue("machineId")
	orgID := getOrgID(r)

	var pc models.ProviderConnection
	if err := s.DB.Where("id = ? AND organization_id = ?", connID, orgID).First(&pc).Error; err != nil {
		respondError(w, http.StatusNotFound, "provider connection not found")
		return
	}

	var machine models.Machine
	if err := s.DB.Where("id = ? AND provider_connection_id = ?", machineID, connID).First(&machine).Error; err != nil {
		respondError(w, http.StatusNotFound, "machine not found")
		return
	}

	if machine.TokenSecretID != nil {
		s.DB.Delete(&models.Secret{}, *machine.TokenSecretID)
	}

	if err := s.DB.Delete(&machine).Error; err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete machine")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}