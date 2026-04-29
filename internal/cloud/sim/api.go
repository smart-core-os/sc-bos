// Package sim provides HTTP JSON REST API handlers that simulate the operation of the Cloud Config Service.
//
// The normal way of invoking this simulator is through the cmd/cloudsim program.
package sim

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
)

// Server handles HTTP API requests for cloudsim management.
type Server struct {
	store       *store.Store
	logger      *zap.Logger
	tokenIssuer *tokenIssuer
}

// NewServer creates a new API server.
func NewServer(store *store.Store, logger *zap.Logger) (*Server, error) {
	ti, err := newTokenIssuer()
	if err != nil {
		return nil, fmt.Errorf("create token issuer: %w", err)
	}
	return &Server{
		store:       store,
		logger:      logger,
		tokenIssuer: ti,
	}, nil
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

	// Enrollment codes
	mux.HandleFunc("POST /api/v1/management/nodes/{id}/enrollment-codes", s.createEnrollmentCode)

	// Device API (BOS-facing)
	mux.HandleFunc("POST /v1/device/register", s.deviceRegister)
	mux.HandleFunc("POST /v1/device/token", s.handleToken)
	mux.HandleFunc("POST /v1/device/check-in", s.checkIn)
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

// Returns a complete URL using the same scheme and host as request r, but with a different path.
// The path is absolute - it is not calculated relative to the path of r.
func sameHostURL(r *http.Request, path string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	// Check X-Forwarded-Proto header for connections behind reverse proxies
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	}
	return (&url.URL{
		Scheme: scheme,
		// for a production server, you would want to validate the Host header against a list of known hostnames
		// to prevent DNS rebinding
		Host: r.Host,
		Path: path,
	}).String()
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
