package stores

import (
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/vanti-dev/sc-bos/internal/util/pgxutil"
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
		wg.Go(func() {
			_, _, _, err := s.Postgres()

			if !errors.Is(err, ErrPreviousConnection) {
				t.Error("unexpected error, expected: ErrPreviousConnection, got: ", err)
			}
		})

		wg.Wait()

		// assumes default delay of 100ms between attempts
		time.Sleep(100 * time.Millisecond)
		// next attempt should try to connect again
		_, _, _, err = s.Postgres()

		if errors.Is(err, ErrPreviousConnection) {
			t.Fatalf("expected a new connection attempt, but got ErrPreviousConnection")
		}
	})
}
