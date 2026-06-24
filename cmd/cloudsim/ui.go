package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

//go:embed *.gohtml
var uiTemplateFS embed.FS

var uiTemplates = template.Must(template.ParseFS(uiTemplateFS, "*.gohtml"))

type uiServer struct {
	store *store.Store
}

func (s *uiServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.serveIndex)
	mux.HandleFunc("GET /ui/sites", s.serveSites)
	mux.HandleFunc("POST /ui/sites", s.createSite)
	mux.HandleFunc("POST /ui/sites/{id}/delete", s.deleteSite)
	mux.HandleFunc("GET /ui/nodes", s.serveNodes)
	mux.HandleFunc("POST /ui/nodes", s.createNode)
	mux.HandleFunc("POST /ui/nodes/{id}/delete", s.deleteNode)
	mux.HandleFunc("POST /ui/nodes/{id}/rotate-secret", s.rotateNodeSecret)
	mux.HandleFunc("POST /ui/nodes/{id}/create-enrollment-code", s.createEnrollmentCode)
	mux.HandleFunc("GET /ui/nodes/{id}/check-ins", s.serveCheckIns)
	mux.HandleFunc("GET /ui/config-versions", s.serveConfigVersions)
	mux.HandleFunc("POST /ui/config-versions", s.createConfigVersion)
	mux.HandleFunc("POST /ui/config-versions/{id}/delete", s.deleteConfigVersion)
	mux.HandleFunc("GET /ui/config-deployments", s.serveConfigDeployments)
	mux.HandleFunc("POST /ui/config-deployments", s.createConfigDeployment)
	mux.HandleFunc("POST /ui/config-deployments/{id}/update-status", s.updateConfigDeploymentStatus)
	mux.HandleFunc("POST /ui/config-deployments/{id}/delete", s.deleteConfigDeployment)
	mux.HandleFunc("GET /ui/update-artefacts", s.serveUpdateArtefacts)
	mux.HandleFunc("POST /ui/update-artefacts", s.createUpdateArtefact)
	mux.HandleFunc("POST /ui/update-artefacts/{id}/delete", s.deleteUpdateArtefact)
	mux.HandleFunc("GET /ui/update-deployments", s.serveUpdateDeployments)
	mux.HandleFunc("POST /ui/update-deployments", s.createUpdateDeployment)
	mux.HandleFunc("POST /ui/update-deployments/{id}/update-status", s.updateUpdateDeploymentStatus)
	mux.HandleFunc("POST /ui/update-deployments/{id}/delete", s.deleteUpdateDeployment)
}

func (s *uiServer) render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := uiTemplates.ExecuteTemplate(w, name, data); err != nil {
		// headers already sent; append error to partial response
		_, _ = fmt.Fprintf(w, "\n<!-- template error: %v -->", err)
	}
}

func parseIDPath(r *http.Request, name string) (int64, error) {
	return strconv.ParseInt(r.PathValue(name), 10, 64)
}

func parseIDQuery(r *http.Request, name string) int64 {
	v, _ := strconv.ParseInt(r.URL.Query().Get(name), 10, 64)
	return v
}

// parseBeforeIDQuery parses an ID from the named query parameter for
// descending-order pagination. Returns math.MaxInt64 when absent.
func parseBeforeIDQuery(r *http.Request, name string) int64 {
	if v, err := strconv.ParseInt(r.URL.Query().Get(name), 10, 64); err == nil {
		return v
	}
	return math.MaxInt64
}

func uiGenerateSecret() (secret, hash []byte, err error) {
	secret = make([]byte, 32)
	if _, err = rand.Read(secret); err != nil {
		return nil, nil, err
	}
	h := sha256.Sum256(secret)
	return secret, h[:], nil
}

func errRedirect(w http.ResponseWriter, r *http.Request, dest, msg string) {
	http.Redirect(w, r, dest+"?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

const uiPageSize = 50

func (s *uiServer) serveIndex(w http.ResponseWriter, r *http.Request) {
	s.render(w, "index", indexViewData{})
}

func (s *uiServer) serveSites(w http.ResponseWriter, r *http.Request) {
	afterID := parseIDQuery(r, "after")
	errMsg := r.URL.Query().Get("error")

	var items []queries.Site
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		items, e = tx.ListSites(r.Context(), queries.ListSitesParams{
			AfterID: afterID,
			Limit:   uiPageSize + 1,
		})
		return e
	})
	if err != nil {
		http.Error(w, "failed to load sites: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "sites", sitesViewData{
		Sites:         items,
		NextPageToken: nextToken,
		Error:         errMsg,
	})
}

func (s *uiServer) createSite(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/sites", "invalid form data")
		return
	}
	name := r.PostForm.Get("name")
	if name == "" {
		errRedirect(w, r, "/ui/sites", "name is required")
		return
	}
	err := s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.CreateSite(r.Context(), name)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/sites", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/sites", http.StatusSeeOther)
}

func (s *uiServer) deleteSite(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/sites", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteSite(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/sites", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/sites", http.StatusSeeOther)
}

func (s *uiServer) serveNodes(w http.ResponseWriter, r *http.Request) {
	afterID := parseIDQuery(r, "after")
	siteID := parseIDQuery(r, "siteId")
	errMsg := r.URL.Query().Get("error")

	var items []queries.Node
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		if siteID != 0 {
			items, e = tx.ListNodesBySite(r.Context(), queries.ListNodesBySiteParams{
				SiteID:  siteID,
				AfterID: afterID,
				Limit:   uiPageSize + 1,
			})
		} else {
			items, e = tx.ListNodes(r.Context(), queries.ListNodesParams{
				AfterID: afterID,
				Limit:   uiPageSize + 1,
			})
		}
		return e
	})
	if err != nil {
		http.Error(w, "failed to load nodes: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "nodes", nodesViewData{
		Nodes:         items,
		SiteID:        siteID,
		NextPageToken: nextToken,
		Error:         errMsg,
	})
}

func (s *uiServer) createNode(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/nodes", "invalid form data")
		return
	}
	hostname := r.PostForm.Get("hostname")
	if hostname == "" {
		errRedirect(w, r, "/ui/nodes", "hostname is required")
		return
	}
	siteID, _ := strconv.ParseInt(r.PostForm.Get("siteId"), 10, 64)

	secret, hash, err := uiGenerateSecret()
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}

	var node queries.Node
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var e error
		node, e = tx.CreateNode(r.Context(), queries.CreateNodeParams{
			Hostname:   hostname,
			SiteID:     siteID,
			SecretHash: hash,
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}

	s.render(w, "node_secret", nodeSecretViewData{
		Node:   node,
		Secret: base64.StdEncoding.EncodeToString(secret),
	})
}

func (s *uiServer) deleteNode(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/nodes", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteNode(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/nodes", http.StatusSeeOther)
}

func (s *uiServer) rotateNodeSecret(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/nodes", "invalid id")
		return
	}

	secret, hash, err := uiGenerateSecret()
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}

	var node queries.Node
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var e error
		node, e = tx.GetNode(r.Context(), id)
		if e != nil {
			return e
		}
		return tx.UpdateNodeSecretHash(r.Context(), queries.UpdateNodeSecretHashParams{
			ID:         id,
			SecretHash: hash,
		})
	})
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}

	s.render(w, "node_secret", nodeSecretViewData{
		Node:   node,
		Secret: base64.StdEncoding.EncodeToString(secret),
	})
}

func (s *uiServer) serveConfigVersions(w http.ResponseWriter, r *http.Request) {
	afterID := parseIDQuery(r, "after")
	nodeID := parseIDQuery(r, "nodeId")
	errMsg := r.URL.Query().Get("error")

	var items []queries.ConfigVersion
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		if nodeID != 0 {
			items, e = tx.ListConfigVersionsByNode(r.Context(), queries.ListConfigVersionsByNodeParams{
				NodeID:  nodeID,
				AfterID: afterID,
				Limit:   uiPageSize + 1,
			})
		} else {
			items, e = tx.ListConfigVersions(r.Context(), queries.ListConfigVersionsParams{
				AfterID: afterID,
				Limit:   uiPageSize + 1,
			})
		}
		return e
	})
	if err != nil {
		http.Error(w, "failed to load config versions: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "config_versions", configVersionsViewData{
		ConfigVersions: items,
		NodeID:         nodeID,
		NextPageToken:  nextToken,
		Error:          errMsg,
	})
}

func (s *uiServer) createConfigVersion(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		errRedirect(w, r, "/ui/config-versions", "invalid form data")
		return
	}
	nodeID, err := strconv.ParseInt(r.FormValue("nodeId"), 10, 64)
	if err != nil || nodeID == 0 {
		errRedirect(w, r, "/ui/config-versions", "valid nodeId is required")
		return
	}
	description := r.FormValue("description")

	var payload []byte
	f, _, ferr := r.FormFile("payload")
	if ferr != nil && ferr != http.ErrMissingFile {
		errRedirect(w, r, "/ui/config-versions", ferr.Error())
		return
	}
	if f != nil {
		defer f.Close()
		payload, err = io.ReadAll(f)
		if err != nil {
			errRedirect(w, r, "/ui/config-versions", err.Error())
			return
		}
	}

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.CreateConfigVersion(r.Context(), queries.CreateConfigVersionParams{
			NodeID:      nodeID,
			Description: sql.NullString{String: description, Valid: description != ""},
			Payload:     payload,
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/config-versions", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/config-versions", http.StatusSeeOther)
}

func (s *uiServer) deleteConfigVersion(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/config-versions", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteConfigVersion(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/config-versions", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/config-versions", http.StatusSeeOther)
}

func (s *uiServer) serveConfigDeployments(w http.ResponseWriter, r *http.Request) {
	beforeID := parseBeforeIDQuery(r, "before")
	nodeID := parseIDQuery(r, "nodeId")
	errMsg := r.URL.Query().Get("error")

	var items []queries.ConfigDeployment
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		if nodeID != 0 {
			items, e = tx.ListConfigDeploymentsByNode(r.Context(), queries.ListConfigDeploymentsByNodeParams{
				NodeID:   nodeID,
				BeforeID: beforeID,
				Limit:    uiPageSize + 1,
			})
		} else {
			items, e = tx.ListConfigDeployments(r.Context(), queries.ListConfigDeploymentsParams{
				BeforeID: beforeID,
				Limit:    uiPageSize + 1,
			})
		}
		return e
	})
	if err != nil {
		http.Error(w, "failed to load deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "config_deployments", configDeploymentsViewData{
		ConfigDeployments: items,
		NodeID:            nodeID,
		NextPageToken:     nextToken,
		Error:             errMsg,
	})
}

func (s *uiServer) createConfigDeployment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/config-deployments", "invalid form data")
		return
	}
	configVersionID, err := strconv.ParseInt(r.PostForm.Get("configVersionId"), 10, 64)
	if err != nil || configVersionID == 0 {
		errRedirect(w, r, "/ui/config-deployments", "valid configVersionId is required")
		return
	}
	status := r.PostForm.Get("status")
	if status == "" {
		status = "pending"
	}

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.CreateConfigDeployment(r.Context(), queries.CreateConfigDeploymentParams{
			ConfigVersionID: configVersionID,
			Status:          status,
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/config-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/config-deployments", http.StatusSeeOther)
}

func (s *uiServer) updateConfigDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/config-deployments", "invalid id")
		return
	}
	if err = r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/config-deployments", "invalid form data")
		return
	}
	status := r.PostForm.Get("status")
	if status == "" {
		errRedirect(w, r, "/ui/config-deployments", "status is required")
		return
	}
	reason := r.PostForm.Get("reason")

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.UpdateConfigDeploymentStatus(r.Context(), queries.UpdateConfigDeploymentStatusParams{
			ID:     id,
			Status: status,
			Reason: sql.NullString{String: reason, Valid: reason != ""},
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/config-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/config-deployments", http.StatusSeeOther)
}

func (s *uiServer) deleteConfigDeployment(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/config-deployments", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteConfigDeployment(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/config-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/config-deployments", http.StatusSeeOther)
}

func (s *uiServer) createEnrollmentCode(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/nodes", "invalid id")
		return
	}

	code := sim.GenerateEnrollmentCode()
	expiresAt := time.Now().Add(15 * time.Minute)

	var node queries.Node
	var ec queries.EnrollmentCode
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		var e error
		node, e = tx.GetNode(r.Context(), id)
		if e != nil {
			return e
		}
		ec, e = tx.CreateEnrollmentCode(r.Context(), queries.CreateEnrollmentCodeParams{
			NodeID:    id,
			Code:      code,
			ExpiresAt: expiresAt,
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/nodes", err.Error())
		return
	}

	s.render(w, "enrollment_code", enrollmentCodeViewData{
		Node: node,
		Code: ec,
	})
}

func (s *uiServer) serveCheckIns(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}
	beforeID := parseBeforeIDQuery(r, "before")

	var node queries.Node
	var items []queries.NodeCheckIn
	err = s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		node, e = tx.GetNode(r.Context(), id)
		if e != nil {
			return e
		}
		items, e = tx.ListNodeCheckInsByNode(r.Context(), queries.ListNodeCheckInsByNodeParams{
			NodeID:   id,
			BeforeID: beforeID,
			Limit:    uiPageSize + 1,
		})
		return e
	})
	if err != nil {
		http.Error(w, "failed to load check-ins: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "check_ins", checkInsViewData{
		NodeID:        id,
		Hostname:      node.Hostname,
		CheckIns:      items,
		NextPageToken: nextToken,
	})
}

func (s *uiServer) serveUpdateArtefacts(w http.ResponseWriter, r *http.Request) {
	afterID := parseIDQuery(r, "after")
	siteID := parseIDQuery(r, "siteId")
	platform := r.URL.Query().Get("platform")
	errMsg := r.URL.Query().Get("error")

	var items []queries.UpdateArtefact
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		items, e = tx.ListUpdateArtefacts(r.Context(), queries.ListUpdateArtefactsParams{
			AfterID:  afterID,
			Platform: platform,
			SiteID:   siteID,
			Limit:    uiPageSize + 1,
		})
		return e
	})
	if err != nil {
		http.Error(w, "failed to load update artefacts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "update_artefacts", updateArtefactsViewData{
		UpdateArtefacts: items,
		SiteID:          siteID,
		Platform:        platform,
		NextPageToken:   nextToken,
		Error:           errMsg,
	})
}

func (s *uiServer) createUpdateArtefact(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		errRedirect(w, r, "/ui/update-artefacts", "invalid form data")
		return
	}
	version := r.FormValue("version")
	if version == "" {
		errRedirect(w, r, "/ui/update-artefacts", "version is required")
		return
	}
	platform := r.FormValue("platform")
	if platform == "" {
		platform = "podman"
	}

	var siteIDPtr *int64
	if v := r.FormValue("siteId"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n != 0 {
			siteIDPtr = &n
		}
	}
	description := r.FormValue("description")
	var descPtr *string
	if description != "" {
		descPtr = &description
	}

	f, _, ferr := r.FormFile("payload")
	if ferr != nil {
		errRedirect(w, r, "/ui/update-artefacts", "payload file is required: "+ferr.Error())
		return
	}
	defer f.Close()

	_, err := s.store.CreateUpdateArtefact(r.Context(), store.CreateUpdateArtefactParams{
		SiteID:      siteIDPtr,
		Platform:    platform,
		Version:     version,
		Description: descPtr,
	}, f)
	if err != nil {
		errRedirect(w, r, "/ui/update-artefacts", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/update-artefacts", http.StatusSeeOther)
}

func (s *uiServer) deleteUpdateArtefact(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/update-artefacts", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteUpdateArtefact(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/update-artefacts", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/update-artefacts", http.StatusSeeOther)
}

func (s *uiServer) serveUpdateDeployments(w http.ResponseWriter, r *http.Request) {
	beforeID := parseBeforeIDQuery(r, "before")
	nodeID := parseIDQuery(r, "nodeId")
	errMsg := r.URL.Query().Get("error")

	var items []queries.UpdateDeployment
	err := s.store.Read(r.Context(), func(tx *store.Tx) error {
		var e error
		if nodeID != 0 {
			items, e = tx.ListUpdateDeploymentsByNode(r.Context(), queries.ListUpdateDeploymentsByNodeParams{
				NodeID:   nodeID,
				BeforeID: beforeID,
				Limit:    uiPageSize + 1,
			})
		} else {
			items, e = tx.ListUpdateDeployments(r.Context(), queries.ListUpdateDeploymentsParams{
				BeforeID: beforeID,
				Limit:    uiPageSize + 1,
			})
		}
		return e
	})
	if err != nil {
		http.Error(w, "failed to load update deployments: "+err.Error(), http.StatusInternalServerError)
		return
	}

	var nextToken string
	if len(items) > uiPageSize {
		nextToken = strconv.FormatInt(items[uiPageSize-1].ID, 10)
		items = items[:uiPageSize]
	}

	s.render(w, "update_deployments", updateDeploymentsViewData{
		UpdateDeployments: items,
		NodeID:            nodeID,
		NextPageToken:     nextToken,
		Error:             errMsg,
	})
}

func (s *uiServer) createUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/update-deployments", "invalid form data")
		return
	}
	nodeID, err := strconv.ParseInt(r.PostForm.Get("nodeId"), 10, 64)
	if err != nil || nodeID == 0 {
		errRedirect(w, r, "/ui/update-deployments", "valid nodeId is required")
		return
	}
	updateArtefactID, err := strconv.ParseInt(r.PostForm.Get("updateArtefactId"), 10, 64)
	if err != nil || updateArtefactID == 0 {
		errRedirect(w, r, "/ui/update-deployments", "valid updateArtefactId is required")
		return
	}
	status := r.PostForm.Get("status")
	if status == "" {
		status = "pending"
	}

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.CreateUpdateDeployment(r.Context(), queries.CreateUpdateDeploymentParams{
			UpdateArtefactID: updateArtefactID,
			NodeID:           nodeID,
			Status:           status,
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/update-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/update-deployments", http.StatusSeeOther)
}

func (s *uiServer) updateUpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/update-deployments", "invalid id")
		return
	}
	if err = r.ParseForm(); err != nil {
		errRedirect(w, r, "/ui/update-deployments", "invalid form data")
		return
	}
	status := r.PostForm.Get("status")
	if status == "" {
		errRedirect(w, r, "/ui/update-deployments", "status is required")
		return
	}
	reason := r.PostForm.Get("reason")

	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.SetUpdateDeploymentStatus(r.Context(), queries.SetUpdateDeploymentStatusParams{
			ID:     id,
			Status: status,
			Reason: sql.NullString{String: reason, Valid: reason != ""},
		})
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/update-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/update-deployments", http.StatusSeeOther)
}

func (s *uiServer) deleteUpdateDeployment(w http.ResponseWriter, r *http.Request) {
	id, err := parseIDPath(r, "id")
	if err != nil {
		errRedirect(w, r, "/ui/update-deployments", "invalid id")
		return
	}
	err = s.store.Write(r.Context(), func(tx *store.Tx) error {
		_, e := tx.DeleteUpdateDeployment(r.Context(), id)
		return e
	})
	if err != nil {
		errRedirect(w, r, "/ui/update-deployments", err.Error())
		return
	}
	http.Redirect(w, r, "/ui/update-deployments", http.StatusSeeOther)
}
