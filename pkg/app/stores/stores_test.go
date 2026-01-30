package stores

import (
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
)

func TestPostgresStore_Postgres(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := &Stores{
			postgresStore: postgresStore{
				cfg: &PostgresConfig{
					ConnectConfig: pgxutil.ConnectConfig{
						URI: "postgres://user:password@localhost:5432/dbname", // will fail to connect in test
					},
				},
			},
		}

		_, _, _, err := s.Postgres()
		if err == nil {
			t.Fatalf("expected connection error, got nil")
		}

		wg := sync.WaitGroup{}

		for i := 0; i < 10; i++ {
			wg.Go(func() {
				_, _, _, err := s.Postgres()

				if !strings.Contains(err.Error(), "cached") {
					t.Error("unexpected error, expected: cached error, got: ", err)
				}
			})
		}

		wg.Wait()

		time.Sleep(100 * time.Millisecond)
		// next attempt should try to connect again
		_, _, _, err = s.Postgres()

		if strings.Contains(err.Error(), "cached") {
			t.Fatalf("expected a new connection attempt, but got cached error")
		}
	})
}
