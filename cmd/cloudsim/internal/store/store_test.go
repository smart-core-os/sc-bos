package store

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

func TestStore_Sites(t *testing.T) {
	ctx := context.Background()
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
			Limit:  10,
			Offset: 0,
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
	ctx := context.Background()
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
			Hostname: "test-node.example.com",
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
		if node.Hostname != "test-node.example.com" {
			t.Errorf("expected hostname 'test-node.example.com', got '%s'", node.Hostname)
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
		nodes, err := tx.ListNodesBySite(ctx, siteID)
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
	ctx := context.Background()
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
			Hostname: "test-node.example.com",
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
			VersionNumber: 1,
			Payload:       []byte(`{"test": "config"}`),
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
		if config.VersionNumber != 1 {
			t.Errorf("expected version number 1, got %d", config.VersionNumber)
		}
		if string(config.Payload) != `{"test": "config"}` {
			t.Errorf("expected payload '{\"test\": \"config\"}', got '%s'", string(config.Payload))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get config version: %v", err)
	}

	// Test listing config versions by node
	err = store.Read(ctx, func(tx *Tx) error {
		configs, err := tx.ListConfigVersionsByNode(ctx, nodeID)
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

	// Test getting latest config version
	err = store.Read(ctx, func(tx *Tx) error {
		config, err := tx.GetLatestConfigVersionByNode(ctx, nodeID)
		if err != nil {
			return err
		}
		if config.VersionNumber != 1 {
			t.Errorf("expected version number 1, got %d", config.VersionNumber)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to get latest config version: %v", err)
	}
}

func TestStore_Deployments(t *testing.T) {
	ctx := context.Background()
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
			Hostname: "test-node.example.com",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node.ID,
			VersionNumber: 1,
			Payload:       []byte(`{"test": "config"}`),
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

	// Test listing deployments by status
	err = store.Read(ctx, func(tx *Tx) error {
		deployments, err := tx.ListDeploymentsByStatus(ctx, queries.ListDeploymentsByStatusParams{
			Status: "COMPLETED",
			Limit:  10,
			Offset: 0,
		})
		if err != nil {
			return err
		}
		if len(deployments) != 1 {
			t.Errorf("expected 1 deployment, got %d", len(deployments))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to list deployments by status: %v", err)
	}
}

func TestStore_CascadeDeletes(t *testing.T) {
	ctx := context.Background()
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
			Hostname: "test-node.example.com",
			SiteID:   site.ID,
		})
		if err != nil {
			return err
		}
		nodeID = node.ID

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:        node.ID,
			VersionNumber: 1,
			Payload:       []byte(`{"test": "config"}`),
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
