package store_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store"
	"github.com/smart-core-os/sc-bos/cmd/cloudsim/internal/store/queries"
)

// Example_basic demonstrates basic CRUD operations with the store.
func Example_basic() {
	ctx := context.Background()
	logger := zap.NewNop()

	// Create an in-memory store for this example
	s := store.NewMemoryStore(logger)
	defer s.Close()

	// Create a site
	var siteID int64
	err := s.Write(ctx, func(tx *store.Tx) error {
		site, err := tx.CreateSite(ctx, "London Office")
		if err != nil {
			return err
		}
		siteID = site.ID
		fmt.Printf("Created site: %s (ID: %d)\n", site.Name, site.ID)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// generate a secret for the new node
	secret := make([]byte, 32)
	_, err = rand.Read(secret)
	if err != nil {
		log.Fatal(err)
	}
	hash := sha256.Sum256(secret)

	// Create a node
	var nodeID int64
	err = s.Write(ctx, func(tx *store.Tx) error {
		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname:   "LONDON-AC-01",
			SiteID:     siteID,
			SecretHash: hash[:],
		})
		if err != nil {
			return err
		}
		nodeID = node.ID
		fmt.Printf("Created node: %s (ID: %d)\n", node.Hostname, node.ID)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a config version
	err = s.Write(ctx, func(tx *store.Tx) error {
		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:      nodeID,
			Description: sql.NullString{String: "v1.0.0", Valid: true},
			Payload:     []byte{0xDE, 0xAD, 0xBE, 0xEF}, // Binary config data
		})
		if err != nil {
			return err
		}
		description := ""
		if config.Description.Valid {
			description = config.Description.String
		}
		fmt.Printf("Created config version: %s (ID: %d)\n", description, config.ID)
		// Note: In the REST API, payloads are accessed via URL (e.g., /api/v1/management/config-versions/1/payload)
		// rather than being returned inline in the JSON response.
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Read data back
	err = s.Read(ctx, func(tx *store.Tx) error {
		site, err := tx.GetSite(ctx, siteID)
		if err != nil {
			return err
		}
		fmt.Printf("Retrieved site: %s\n", site.Name)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Created site: London Office (ID: 1)
	// Created node: LONDON-AC-01 (ID: 1)
	// Created config version: v1.0.0 (ID: 1)
	// Retrieved site: London Office
}

// Example_pagination demonstrates page token pagination for list queries.
func Example_pagination() {
	ctx := context.Background()
	logger := zap.NewNop()

	s := store.NewMemoryStore(logger)
	defer s.Close()

	// Create multiple sites
	err := s.Write(ctx, func(tx *store.Tx) error {
		for i := 1; i <= 5; i++ {
			_, err := tx.CreateSite(ctx, fmt.Sprintf("Site %d", i))
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// First page: get first 2 sites
	var lastID int64
	err = s.Read(ctx, func(tx *store.Tx) error {
		sites, err := tx.ListSites(ctx, queries.ListSitesParams{
			AfterID: 0, // Start from beginning
			Limit:   2, // Get 2 results
		})
		if err != nil {
			return err
		}
		fmt.Printf("First page: %d sites\n", len(sites))
		for _, site := range sites {
			fmt.Printf("  - %s (ID: %d)\n", site.Name, site.ID)
		}
		if len(sites) > 0 {
			lastID = sites[len(sites)-1].ID
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Second page: get next 2 sites
	err = s.Read(ctx, func(tx *store.Tx) error {
		sites, err := tx.ListSites(ctx, queries.ListSitesParams{
			AfterID: lastID, // Continue from last ID
			Limit:   2,
		})
		if err != nil {
			return err
		}
		fmt.Printf("Second page: %d sites\n", len(sites))
		for _, site := range sites {
			fmt.Printf("  - %s (ID: %d)\n", site.Name, site.ID)
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// First page: 2 sites
	//   - Site 1 (ID: 1)
	//   - Site 2 (ID: 2)
	// Second page: 2 sites
	//   - Site 3 (ID: 3)
	//   - Site 4 (ID: 4)
}

// Example_deployments demonstrates creating and managing deployments.
func Example_deployments() {
	ctx := context.Background()
	logger := zap.NewNop()

	s := store.NewMemoryStore(logger)
	defer s.Close()

	// Setup: create site, node, and config version
	var configVersionID int64
	err := s.Write(ctx, func(tx *store.Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname:   "TEST-AC-01",
			SiteID:     site.ID,
			SecretHash: []byte("test-hash"),
		})
		if err != nil {
			return err
		}

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:      node.ID,
			Description: sql.NullString{String: "v1.0.0", Valid: true},
			Payload:     []byte{0xCA, 0xFE, 0xBA, 0xBE},
		})
		if err != nil {
			return err
		}
		configVersionID = config.ID
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a deployment
	var deploymentID int64
	err = s.Write(ctx, func(tx *store.Tx) error {
		deployment, err := tx.CreateDeployment(ctx, queries.CreateDeploymentParams{
			ConfigVersionID: configVersionID,
			Status:          "PENDING",
		})
		if err != nil {
			return err
		}
		deploymentID = deployment.ID
		fmt.Printf("Created deployment (ID: %d) with status: %s\n", deployment.ID, deployment.Status)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Update deployment status to completed
	err = s.Write(ctx, func(tx *store.Tx) error {
		deployment, err := tx.UpdateDeploymentStatus(ctx, queries.UpdateDeploymentStatusParams{
			ID:     deploymentID,
			Status: "COMPLETED",
		})
		if err != nil {
			return err
		}
		fmt.Printf("Updated deployment to: %s\n", deployment.Status)
		if deployment.FinishedTime.Valid {
			fmt.Println("Finished time is now set")
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Created deployment (ID: 1) with status: PENDING
	// Updated deployment to: COMPLETED
	// Finished time is now set
}

// Example_cascadeDeletes demonstrates cascade delete behavior.
func Example_cascadeDeletes() {
	ctx := context.Background()
	logger := zap.NewNop()

	s := store.NewMemoryStore(logger)
	defer s.Close()

	// Create a full chain: site -> node -> config version -> deployment
	var siteID, nodeID int64
	err := s.Write(ctx, func(tx *store.Tx) error {
		site, err := tx.CreateSite(ctx, "Test Site")
		if err != nil {
			return err
		}
		siteID = site.ID

		node, err := tx.CreateNode(ctx, queries.CreateNodeParams{
			Hostname:   "TEST-AC-01",
			SiteID:     site.ID,
			SecretHash: []byte("test-hash"),
		})
		if err != nil {
			return err
		}
		nodeID = node.ID

		config, err := tx.CreateConfigVersion(ctx, queries.CreateConfigVersionParams{
			NodeID:      node.ID,
			Description: sql.NullString{String: "v1.0.0", Valid: true},
			Payload:     []byte{0xDE, 0xAD},
		})
		if err != nil {
			return err
		}

		_, err = tx.CreateDeployment(ctx, queries.CreateDeploymentParams{
			ConfigVersionID: config.ID,
			Status:          "PENDING",
		})
		return err
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Created site, node, config version, and deployment")

	// Delete the site - cascades to all related entities
	err = s.Write(ctx, func(tx *store.Tx) error {
		rows, err := tx.DeleteSite(ctx, siteID)
		if err != nil {
			return err
		}
		fmt.Printf("Deleted site (affected %d row)\n", rows)
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Verify node is also deleted
	err = s.Read(ctx, func(tx *store.Tx) error {
		_, err := tx.GetNode(ctx, nodeID)
		if err != nil {
			fmt.Println("Node was cascade deleted")
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// Output:
	// Created site, node, config version, and deployment
	// Deleted site (affected 1 row)
	// Node was cascade deleted
}
