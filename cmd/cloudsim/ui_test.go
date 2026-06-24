package main

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// TestTemplates renders every UI page against a seeded store and fails if any
// template errors. render returns a 500 when a template fails to execute, so a
// non-200 status flags a broken template.
func TestTemplates(t *testing.T) {
	dataStore := store.NewMemoryStore(zap.NewNop())
	t.Cleanup(func() { _ = dataStore.Close() })

	nodeID := seedTestbed(t, dataStore)

	srv := &uiServer{store: dataStore}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	nodePath := strconv.FormatInt(nodeID, 10)
	cases := []struct {
		name   string
		method string
		path   string
	}{
		{"index", http.MethodGet, "/"},
		{"sites", http.MethodGet, "/ui/sites"},
		{"nodes", http.MethodGet, "/ui/nodes"},
		{"config_versions", http.MethodGet, "/ui/config-versions"},
		{"config_deployments", http.MethodGet, "/ui/config-deployments"},
		{"check_ins", http.MethodGet, "/ui/nodes/" + nodePath + "/check-ins"},
		{"binary_artefacts", http.MethodGet, "/ui/binary-artefacts"},
		{"binary_deployments", http.MethodGet, "/ui/binary-deployments"},
		{"enrollment_code", http.MethodPost, "/ui/nodes/" + nodePath + "/create-enrollment-code"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body:\n%s", rec.Code, http.StatusOK, rec.Body.String())
			}
		})
	}
}

// seedTestbed populates one row of every entity the UI renders, filling the
// nullable columns so the templates exercise their conditional branches. It
// returns the id of the seeded node, which several routes are keyed on.
func seedTestbed(t *testing.T, s *store.Store) (nodeID int64) {
	t.Helper()
	ctx := t.Context()

	// An artefact's payload is an external file, so it is created through the
	// store method that streams src to disk rather than through a Tx.
	artefact, err := s.CreateBinaryArtefact(ctx, store.CreateBinaryArtefactParams{
		OS:          "linux",
		Arch:        "arm64",
		Version:     "v1.2.3",
		Description: new("nightly build"),
	}, strings.NewReader("artefact-payload"))
	if err != nil {
		t.Fatalf("create binary artefact: %v", err)
	}

	err = s.Write(ctx, func(tx *store.Tx) error {
		site, err := tx.CreateSite(ctx, "London Office")
		if err != nil {
			return err
		}

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Os:       "linux",
			Arch:     "arm64",
			Hostname: "LONDON-AC-01",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID

		configVersion, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:      node.ID,
			Description: sql.NullString{String: "v1.0.0", Valid: true},
			Payload:     []byte{0xDE, 0xAD, 0xBE, 0xEF},
		})
		if err != nil {
			return err
		}

		configDeployment, err := tx.CreateConfigDeployment(ctx, queries.CreateConfigDeploymentParams{
			ConfigVersionID: configVersion.ID,
			Status:          "pending",
		})
		if err != nil {
			return err
		}
		// Move to a terminal state so FinishedTime and Reason are populated.
		if _, err := tx.UpdateConfigDeploymentStatus(ctx, queries.UpdateConfigDeploymentStatusParams{
			ID:     configDeployment.ID,
			Status: "completed",
			Reason: sql.NullString{String: "applied cleanly", Valid: true},
		}); err != nil {
			return err
		}

		binaryDeployment, err := tx.CreateBinaryDeployment(ctx, queries.CreateBinaryDeploymentParams{
			BinaryArtefactID: artefact.ID,
			NodeID:           node.ID,
			Status:           "pending",
		})
		if err != nil {
			return err
		}
		if _, err := tx.SetBinaryDeploymentStatus(ctx, queries.SetBinaryDeploymentStatusParams{
			ID:     binaryDeployment.ID,
			Status: "failed",
			Reason: sql.NullString{String: "checksum mismatch", Valid: true},
		}); err != nil {
			return err
		}

		// A check-in with every nullable install-tracking column set, so the
		// check_ins template renders each field rather than the empty branch.
		_, err = tx.CreateNodeCheckIn(ctx, queries.CreateNodeCheckInParams{
			NodeID:                       node.ID,
			CurrentDeploymentID:          sql.NullInt64{Int64: configDeployment.ID, Valid: true},
			InstallingDeploymentID:       sql.NullInt64{Int64: configDeployment.ID, Valid: true},
			InstallingDeploymentError:    sql.NullString{String: "transient failure", Valid: true},
			InstallingDeploymentAttempts: sql.NullInt64{Int64: 2, Valid: true},
			CurrentBinaryDeploymentID:    sql.NullInt64{Int64: binaryDeployment.ID, Valid: true},
			InstallingBinaryDeploymentID: sql.NullInt64{Int64: binaryDeployment.ID, Valid: true},
			InstallingBinaryError:        sql.NullString{String: "retrying", Valid: true},
			InstallingBinaryAttempts:     sql.NullInt64{Int64: 1, Valid: true},
		})
		return err
	})
	if err != nil {
		t.Fatalf("seed testbed: %v", err)
	}
	return nodeID
}
