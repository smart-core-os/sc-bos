// Package api provides HTTP JSON REST API handlers for cloudsim database entities.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
)

// Server handles HTTP API requests for cloudsim management.
type Server struct {
	store  *store.Store
	logger *zap.Logger
}

// NewServer creates a new API server.
func NewServer(store *store.Store, logger *zap.Logger) *Server {
	return &Server{
		store:  store,
		logger: logger,
	}
}

// RegisterRoutes registers all API routes on the given mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	// Sites
	mux.HandleFunc("GET /api/v1/management/sites", s.listSites)
	mux.HandleFunc("POST /api/v1/management/sites", s.createSite)
	mux.HandleFunc("GET /api/v1/management/sites/{id}", s.getSite)
	mux.HandleFunc("PUT /api/v1/management/sites/{id}", s.updateSite)
	mux.HandleFunc("DELETE /api/v1/management/sites/{id}", s.deleteSite)

	// Nodes
	mux.HandleFunc("GET /api/v1/management/nodes", s.listNodes)
	mux.HandleFunc("POST /api/v1/management/nodes", s.createNode)
	mux.HandleFunc("GET /api/v1/management/nodes/{id}", s.getNode)
	mux.HandleFunc("PUT /api/v1/management/nodes/{id}", s.updateNode)
	mux.HandleFunc("DELETE /api/v1/management/nodes/{id}", s.deleteNode)
	mux.HandleFunc("POST /api/v1/management/nodes/{id}/rotate-secret", s.rotateNodeSecret)

	// Node Check-Ins
	mux.HandleFunc("GET /api/v1/management/nodes/{nodeId}/check-ins", s.listNodeCheckIns)
	mux.HandleFunc("GET /api/v1/management/nodes/{nodeId}/check-ins/{id}", s.getNodeCheckIn)

	// Config Versions
	mux.HandleFunc("GET /api/v1/management/config-versions", s.listConfigVersions)
	mux.HandleFunc("POST /api/v1/management/config-versions", s.createConfigVersion)
	mux.HandleFunc("GET /api/v1/management/config-versions/{id}", s.getConfigVersion)
	mux.HandleFunc("GET /api/v1/management/config-versions/{id}/payload", s.getConfigVersionPayload)
	mux.HandleFunc("DELETE /api/v1/management/config-versions/{id}", s.deleteConfigVersion)

	// Deployments
	mux.HandleFunc("GET /api/v1/management/deployments", s.listDeployments)
	mux.HandleFunc("POST /api/v1/management/deployments", s.createDeployment)
	mux.HandleFunc("GET /api/v1/management/deployments/{id}", s.getDeployment)
	mux.HandleFunc("PATCH /api/v1/management/deployments/{id}", s.updateDeploymentStatus)
	mux.HandleFunc("DELETE /api/v1/management/deployments/{id}", s.deleteDeployment)
}

func (s *Server) loggerFor(r *http.Request) *zap.Logger {
	return s.logger.With(
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
	)
}

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// parseID decodes an entity ID string from a query parameter into a numeric ID.
// Returns 0 if the string is empty.
// Returns an error if the string is present but invalid (non-numeric or negative).
func parseID(idStr string) (int64, error) {
	if idStr == "" {
		return 0, nil
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, err
	}

	if id < 0 {
		return 0, fmt.Errorf("negative ID not allowed")
	}

	return id, nil
}
