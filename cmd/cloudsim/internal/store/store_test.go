package store

import (
	"bytes"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
	"github.com/smart-core-os/sc-bos/internal/sqlite"
)

func TestStore_Sites(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Test creating a site
	var siteID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Test getting a site
	err = store.Read(ctx, func(tx *Tx) error {
		site, err := tx.GetSite(ctx, siteID)
		if err != nil {
			return err
		}
		if site.Name != "Test Site" {
			t.Errorf("expected site name 'Test Site', got '%s'", site.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get site: %v", err)
	}

	// Test listing sites
	err = store.Read(ctx, func(tx *Tx) error {
		sites, err := tx.ListSites(ctx, queries.ListSitesParams{
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(sites) != 1 {
			t.Errorf("expected 1 site, got %d", len(sites))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list sites: %v", err)
	}
}

func TestStore_Nodes(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Create a site first
	var siteID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Test creating a node
	var nodeID int64
	err = store.Write(ctx, func(tx *Tx) error {
		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "TEST-AC-01",
			SiteID:   siteID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	// Test getting a node
	err = store.Read(ctx, func(tx *Tx) error {
		node, err := tx.GetNode(ctx, nodeID)
		if err != nil {
			return err
		}
		if node.Hostname != "TEST-AC-01" {
			t.Errorf("expected hostname 'TEST-AC-01', got '%s'", node.Hostname)
		}
		if node.SiteID != siteID {
			t.Errorf("expected site ID %d, got %d", siteID, node.SiteID)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	// Test listing nodes by site
	err = store.Read(ctx, func(tx *Tx) error {
		nodes, err := tx.ListNodesBySite(ctx, queries.ListNodesBySiteParams{
			SiteID:  siteID,
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(nodes) != 1 {
			t.Errorf("expected 1 node, got %d", len(nodes))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list nodes by site: %v", err)
	}
}

func TestStore_ConfigVersions(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Setup: create site and node
	var nodeID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "TEST-AC-01",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	// Test creating a config version
	var configVersionID int64
	err = store.Write(ctx, func(tx *Tx) error {
		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        nodeID,
			VersionNumber: "v1",
			Payload:       []byte{0xDE, 0xAD, 0xBE, 0xEF},
		})
		if err != nil {
			return err
		}
		configVersionID = config.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create config version: %v", err)
	}

	// Test getting a config version
	err = store.Read(ctx, func(tx *Tx) error {
		config, err := tx.GetConfigVersion(ctx, configVersionID)
		if err != nil {
			return err
		}
		if config.VersionNumber != "v1" {
			t.Errorf("expected version number v1, got %s", config.VersionNumber)
		}
		expectedPayload := []byte{0xDE, 0xAD, 0xBE, 0xEF}
		if !bytes.Equal(config.Payload, expectedPayload) {
			t.Errorf("payload mismatch: expected %#v, got %#v", expectedPayload, config.Payload)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get config version: %v", err)
	}

	// Test listing config versions by node
	err = store.Read(ctx, func(tx *Tx) error {
		configs, err := tx.ListConfigVersionsByNode(ctx, queries.ListConfigVersionsByNodeParams{
			NodeID:  nodeID,
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(configs) != 1 {
			t.Errorf("expected 1 config version, got %d", len(configs))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list config versions: %v", err)
	}
}

func TestStore_Deployments(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Setup: create site, node, and config version
	var configVersionID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "TEST-AC-01",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node.ID,
			VersionNumber: "v1",
			Payload:       []byte{0xCA, 0xFE, 0xBA, 0xBE},
		})
		if err != nil {
			return err
		}
		configVersionID = config.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	// Test creating a deployment
	var deploymentID int64
	err = store.Write(ctx, func(tx *Tx) error {
		deployment, err := tx.CreateDeployment(ctx, queries.CreateDeploymentParams{
			ConfigVersionID: configVersionID,
			Status:          "PENDING",
		})
		if err != nil {
			return err
		}
		deploymentID = deployment.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}

	// Test getting a deployment
	err = store.Read(ctx, func(tx *Tx) error {
		deployment, err := tx.GetDeployment(ctx, deploymentID)
		if err != nil {
			return err
		}
		if deployment.Status != "PENDING" {
			t.Errorf("expected status 'PENDING', got '%s'", deployment.Status)
		}
		if deployment.FinishedTime.Valid {
			t.Error("expected finished_time to be NULL for PENDING deployment")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get deployment: %v", err)
	}

	// Test updating deployment status to completed
	err = store.Write(ctx, func(tx *Tx) error {
		deployment, err := tx.UpdateDeploymentStatus(ctx, queries.UpdateDeploymentStatusParams{
			ID:     deploymentID,
			Status: "COMPLETED",
		})
		if err != nil {
			return err
		}
		if deployment.Status != "COMPLETED" {
			t.Errorf("expected status 'COMPLETED', got '%s'", deployment.Status)
		}
		if !deployment.FinishedTime.Valid {
			t.Error("expected finished_time to be set for COMPLETED deployment")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to update deployment status: %v", err)
	}

	// Test listing deployments by node
	err = store.Read(ctx, func(tx *Tx) error {
		// Get the node ID from the config version
		config, err := tx.GetConfigVersion(ctx, configVersionID)
		if err != nil {
			return err
		}

		deployments, err := tx.ListDeploymentsByNode(ctx, queries.ListDeploymentsByNodeParams{
			NodeID:  config.NodeID,
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(deployments) != 1 {
			t.Errorf("expected 1 deployment, got %d", len(deployments))
		}
		if len(deployments) > 0 && deployments[0].Status != "COMPLETED" {
			t.Errorf("expected status 'COMPLETED', got '%s'", deployments[0].Status)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list deployments by node: %v", err)
	}
}

func TestStore_CascadeDeletes(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Setup: create a full chain
	var siteID, nodeID, configVersionID, deploymentID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "TEST-AC-01",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node.ID,
			VersionNumber: "v1",
			Payload:       []byte{0xFE, 0xED, 0xFA, 0xCE},
		})
		if err != nil {
			return err
		}
		configVersionID = config.ID

		deployment, err := tx.CreateDeployment(ctx, queries.CreateDeploymentParams{
			ConfigVersionID: config.ID,
			Status:          "PENDING",
		})
		if err != nil {
			return err
		}
		deploymentID = deployment.ID

		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	// Delete the site - should cascade delete node, config_version, and deployment
	err = store.Write(ctx, func(tx *Tx) error {
		rows, err := tx.DeleteSite(ctx, siteID)
		if err != nil {
			return err
		}
		if rows != 1 {
			t.Errorf("expected 1 row deleted, got %d", rows)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to delete site: %v", err)
	}

	// Verify all related entities are deleted
	err = store.Read(ctx, func(tx *Tx) error {
		// Check node is deleted
		_, err := tx.GetNode(ctx, nodeID)
		if err == nil {
			t.Error("expected node to be deleted, but it still exists")
		}

		// Check config version is deleted
		_, err = tx.GetConfigVersion(ctx, configVersionID)
		if err == nil {
			t.Error("expected config version to be deleted, but it still exists")
		}

		// Check deployment is deleted
		_, err = tx.GetDeployment(ctx, deploymentID)
		if err == nil {
			t.Error("expected deployment to be deleted, but it still exists")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to verify cascade deletes: %v", err)
	}
}

func TestStore_ConfigVersionUniqueness(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Setup: create site and node
	var nodeID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "TEST-AC-01",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to setup: %v", err)
	}

	// Create first config version
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        nodeID,
			VersionNumber: "v1.0.0",
			Payload:       []byte{0xDE, 0xAD, 0xBE, 0xEF},
		})
		return err
	})
	if err != nil {
		t.Fatalf("failed to create first config version: %v", err)
	}

	// Try to create second config version with same version_number - should fail
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        nodeID,
			VersionNumber: "v1.0.0",
			Payload:       []byte{0xCA, 0xFE, 0xBA, 0xBE},
		})
		return err
	})
	if !sqlite.IsUniqueConstraintError(err) {
		t.Fatalf("expected error when creating duplicate version_number, got %v", err)
	}

	// Create config version with different version_number - should succeed
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        nodeID,
			VersionNumber: "v2.0.0",
			Payload:       []byte{0xFE, 0xED, 0xFA, 0xCE},
		})
		return err
	})
	if err != nil {
		t.Fatalf("failed to create config version with different version_number: %v", err)
	}

	// Verify we have 2 config versions with correct payloads
	err = store.Read(ctx, func(tx *Tx) error {
		configs, err := tx.ListConfigVersionsByNode(ctx, queries.ListConfigVersionsByNodeParams{
			NodeID:  nodeID,
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(configs) != 2 {
			t.Errorf("expected 2 config versions, got %d", len(configs))
		}

		// Check payloads are stored correctly (ordered by id ASC, so first created comes first)
		expect := []queries.ConfigVersion{
			{NodeID: nodeID, VersionNumber: "v1.0.0", Payload: []byte{0xDE, 0xAD, 0xBE, 0xEF}},
			{NodeID: nodeID, VersionNumber: "v2.0.0", Payload: []byte{0xFE, 0xED, 0xFA, 0xCE}},
		}
		if diff := cmp.Diff(expect, configs, cmp.Comparer(configVersionsDataEqual)); diff != "" {
			t.Errorf("config versions data mismatch (-want +got):\n%s", diff)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list config versions: %v", err)
	}
}

func configVersionsDataEqual(a, b queries.ConfigVersion) bool {
	return a.NodeID == b.NodeID && a.VersionNumber == b.VersionNumber && bytes.Equal(a.Payload, b.Payload)
}

func TestOpenStore(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	dbPath := filepath.Join(t.TempDir(), "test.db")

	// Test creating a new store
	store, err := OpenStore(ctx, dbPath, logger)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	// Create some data
	var siteID int64
	err = store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Persistent Site")
		if err != nil {
			return err
		}
		siteID = site.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Close the store
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Re-open the store and verify data persists
	store2, err := OpenStore(ctx, dbPath, logger)
	if err != nil {
		t.Fatalf("failed to re-open store: %v", err)
	}
	defer func() {
		_ = store2.Close()
	}()

	err = store2.Read(ctx, func(tx *Tx) error {
		site, err := tx.GetSite(ctx, siteID)
		if err != nil {
			return err
		}
		if site.Name != "Persistent Site" {
			t.Errorf("expected site name 'Persistent Site', got '%s'", site.Name)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to read persisted data: %v", err)
	}

	// Verify migrations are applied correctly by checking schema version
	err = store2.Read(ctx, func(tx *Tx) error {
		// List all sites to ensure schema is working
		sites, err := tx.ListSites(ctx, queries.ListSitesParams{
			AfterID: 0,
			Limit:   10,
		})
		if err != nil {
			return err
		}
		if len(sites) != 1 {
			t.Errorf("expected 1 site after re-opening, got %d", len(sites))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to verify migrations: %v", err)
	}
}

func TestStore_CountOperations(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Initially, all counts should be 0
	err := store.Read(ctx, func(tx *Tx) error {
		count, err := tx.CountSites(ctx)
		if err != nil {
			return err
		}
		if count != 0 {
			t.Errorf("expected 0 sites initially, got %d", count)
		}

		count, err = tx.CountNodes(ctx)
		if err != nil {
			return err
		}
		if count != 0 {
			t.Errorf("expected 0 nodes initially, got %d", count)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to count empty store: %v", err)
	}

	// Create test data: 2 sites, 3 nodes (2 in first site, 1 in second)
	var site1ID, site2ID, node1ID, node2ID int64
	err = store.Write(ctx, func(tx *Tx) error {
		site1, err := tx.CreateSite(ctx, "Site 1")
		if err != nil {
			return err
		}
		site1ID = site1.ID

		site2, err := tx.CreateSite(ctx, "Site 2")
		if err != nil {
			return err
		}
		site2ID = site2.ID

		node1, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "NODE-01",
			SiteID:   site1ID,
		})
		if err != nil {
			return err
		}
		node1ID = node1.ID

		node2, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "NODE-02",
			SiteID:   site1ID,
		})
		if err != nil {
			return err
		}
		node2ID = node2.ID

		_, err = tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname: "NODE-03",
			SiteID:   site2ID,
		})
		if err != nil {
			return err
		}

		// Create config versions for nodes
		_, err = tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node1ID,
			VersionNumber: "v1",
			Payload:       []byte{0x01},
		})
		if err != nil {
			return err
		}

		cv2, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node2ID,
			VersionNumber: "v1",
			Payload:       []byte{0x02},
		})
		if err != nil {
			return err
		}

		// Create a deployment
		_, err = tx.CreateDeployment(ctx, queries.CreateDeploymentParams{
			ConfigVersionID: cv2.ID,
			Status:          "PENDING",
		})
		return err
	})
	if err != nil {
		t.Fatalf("failed to create test data: %v", err)
	}

	// Test count operations
	err = store.Read(ctx, func(tx *Tx) error {
		// Count sites
		siteCount, err := tx.CountSites(ctx)
		if err != nil {
			return err
		}
		if siteCount != 2 {
			t.Errorf("expected 2 sites, got %d", siteCount)
		}

		// Count all nodes
		nodeCount, err := tx.CountNodes(ctx)
		if err != nil {
			return err
		}
		if nodeCount != 3 {
			t.Errorf("expected 3 nodes, got %d", nodeCount)
		}

		// Count nodes by site 1 (should be 2)
		site1NodeCount, err := tx.CountNodesBySite(ctx, site1ID)
		if err != nil {
			return err
		}
		if site1NodeCount != 2 {
			t.Errorf("expected 2 nodes in site 1, got %d", site1NodeCount)
		}

		// Count nodes by site 2 (should be 1)
		site2NodeCount, err := tx.CountNodesBySite(ctx, site2ID)
		if err != nil {
			return err
		}
		if site2NodeCount != 1 {
			t.Errorf("expected 1 node in site 2, got %d", site2NodeCount)
		}

		// Count nodes by non-existent site (should be 0)
		noSiteNodeCount, err := tx.CountNodesBySite(ctx, 99999)
		if err != nil {
			return err
		}
		if noSiteNodeCount != 0 {
			t.Errorf("expected 0 nodes in non-existent site, got %d", noSiteNodeCount)
		}

		// Count deployments
		deploymentCount, err := tx.CountDeployments(ctx)
		if err != nil {
			return err
		}
		if deploymentCount != 1 {
			t.Errorf("expected 1 deployment, got %d", deploymentCount)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("failed to test count operations: %v", err)
	}
}

func TestStore_UpdateNonExistent(t *testing.T) {
	ctx := t.Context()
	logger := zap.NewNop()
	store := NewMemoryStore(logger)
	defer func() {
		_ = store.Close()
	}()

	// Create a site to use for valid foreign keys
	var siteID int64
	err := store.Write(ctx, func(tx *Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID
		return nil
	})
	if err != nil {
		t.Fatalf("failed to create site: %v", err)
	}

	// Test updating non-existent site
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.UpdateSite(ctx, queries.UpdateSiteParams{
			ID:   99999,
			Name: "Updated Name",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows when updating non-existent site, got %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to test update non-existent site: %v", err)
	}

	// Test updating non-existent node
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.UpdateNode(ctx, queries.UpdateNodeParams{
			ID:       99999,
			Hostname: "Updated Hostname",
			SiteID:   siteID,
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows when updating non-existent node, got %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to test update non-existent node: %v", err)
	}

	// Test updating non-existent deployment status
	err = store.Write(ctx, func(tx *Tx) error {
		_, err := tx.UpdateDeploymentStatus(ctx, queries.UpdateDeploymentStatusParams{
			ID:     99999,
			Status: "COMPLETED",
		})
		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf("expected sql.ErrNoRows when updating non-existent deployment, got %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to test update non-existent deployment: %v", err)
	}
}
