package sim

import (
	"net/http"
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

// A config version with no checksum (a legacy row predating the checksum column) must never be offered
// without one: the check-in fails rather than handing the node an unverifiable payload. Operators repair
// such rows with `cloudsim -cleanup`.
func TestCheckIn_MissingConfigChecksumFailsRequest(t *testing.T) {
	e := setupCheckInEnv(t)

	// Seed a legacy config version (NULL sha256) and a pending deployment for it.
	err := e.store.Write(t.Context(), func(tx *store.Tx) error {
		cv, err := tx.CreateConfigVersion(t.Context(), queries.CreateConfigVersionParams{
			NodeID:  e.node.ID,
			Payload: []byte("legacy-config"), // Sha256 left nil -> NULL
		})
		if err != nil {
			return err
		}
		_, err = tx.CreateConfigDeployment(t.Context(), queries.CreateConfigDeploymentParams{
			ConfigVersionID: cv.ID,
			Status:          "pending",
		})
		return err
	})
	if err != nil {
		t.Fatalf("seed legacy config version: %v", err)
	}

	resp := doCheckIn(t, e.deviceClient, checkInURL(e.testServer.URL), nil, nil)
	assertStatus(t, resp, http.StatusInternalServerError)
}
