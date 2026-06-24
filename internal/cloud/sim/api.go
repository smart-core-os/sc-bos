// Package sim provides HTTP JSON REST API handlers that simulate the operation of the Cloud Config Service.
//
// The normal way of invoking this simulator is through the cmd/cloudsim program.
package sim

import (
	"crypto/tls"
	"crypto/x509"
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
	store  *store.Store
	logger *zap.Logger
	ca     *devCA
}

// NewServer creates a new API server.
func NewServer(store *store.Store, logger *zap.Logger) (*Server, error) {
	ca, err := newDevCA()
	if err != nil {
		return nil, fmt.Errorf("create dev CA: %w", err)
	}
	return &Server{
		store:  store,
		logger: logger,
		ca:     ca,
	}, nil
}

// ServerTLSConfig returns a TLS config presenting a CA-issued server certificate
// and accepting a client certificate when offered (mTLS accept mode). The
// device endpoints require a valid client certificate per-endpoint.
func (s *Server) ServerTLSConfig() (*tls.Config, error) {
	return s.ca.serverTLSConfig()
}

// CACertPool returns the trust anchor clients use to verify the server (and that
// the server uses to verify clients). In production this is the public server CA
// for the server and the Connect CA for clients; the dev CA stands in for both.
func (s *Server) CACertPool() *x509.CertPool {
	return s.ca.pool
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

	// Node Check-Ins
	mux.HandleFunc("GET /api/v1/management/nodes/{nodeId}/check-ins", s.listNodeCheckIns)
	mux.HandleFunc("GET /api/v1/management/nodes/{nodeId}/check-ins/{id}", s.getNodeCheckIn)

	// Config Versions
	mux.HandleFunc("GET /api/v1/management/config-versions", s.listConfigVersions)
	mux.HandleFunc("POST /api/v1/management/config-versions", s.createConfigVersion)
	mux.HandleFunc("GET /api/v1/management/config-versions/{id}", s.getConfigVersion)
	mux.HandleFunc("GET /api/v1/management/config-versions/{id}/payload", s.getConfigVersionPayload)
	mux.HandleFunc("DELETE /api/v1/management/config-versions/{id}", s.deleteConfigVersion)

	// ConfigDeployments
	mux.HandleFunc("GET /api/v1/management/config-deployments", s.listConfigDeployments)
	mux.HandleFunc("POST /api/v1/management/config-deployments", s.createConfigDeployment)
	mux.HandleFunc("GET /api/v1/management/config-deployments/{id}", s.getConfigDeployment)
	mux.HandleFunc("PATCH /api/v1/management/config-deployments/{id}", s.updateConfigDeploymentStatus)
	mux.HandleFunc("DELETE /api/v1/management/config-deployments/{id}", s.deleteConfigDeployment)

	// Binary Artefacts
	mux.HandleFunc("GET /api/v1/management/binary-artefacts", s.listBinaryArtefacts)
	mux.HandleFunc("POST /api/v1/management/binary-artefacts", s.createBinaryArtefact)
	mux.HandleFunc("GET /api/v1/management/binary-artefacts/{id}", s.getBinaryArtefact)
	mux.HandleFunc("GET /api/v1/management/binary-artefacts/{id}/payload", s.getBinaryArtefactPayload)
	mux.HandleFunc("DELETE /api/v1/management/binary-artefacts/{id}", s.deleteBinaryArtefact)

	// Binary Deployments
	mux.HandleFunc("GET /api/v1/management/binary-deployments", s.listBinaryDeployments)
	mux.HandleFunc("POST /api/v1/management/binary-deployments", s.createBinaryDeployment)
	mux.HandleFunc("GET /api/v1/management/binary-deployments/{id}", s.getBinaryDeployment)
	mux.HandleFunc("PATCH /api/v1/management/binary-deployments/{id}", s.updateBinaryDeploymentStatus)
	mux.HandleFunc("DELETE /api/v1/management/binary-deployments/{id}", s.deleteBinaryDeployment)

	// Enrollment codes
	mux.HandleFunc("POST /api/v1/management/nodes/{id}/enrollment-codes", s.createEnrollmentCode)

	// Device API (BOS-facing)
	mux.HandleFunc("POST /v1/device/register", s.deviceRegister)
	mux.HandleFunc("POST /v1/device/certificate/renew", s.deviceRenew)
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
