// Command dbadd-publications-sample connects to a postgres database, creates tables, and seeds them with some sample test data.
// The command is useful when working with the publication api
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"go.uber.org/multierr"
	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/system/publications/config"
	"github.com/smart-core-os/sc-bos/pkg/system/publications/pgxpublications"
)

func main() {
	configFilePath := flag.String("config", "", "Path to config file matching pkg/system/publications/config/root.go format")
	flag.Parse()

	// read config files
	var conf config.Root
	if *configFilePath == "" {
		// default setup used during development
		passFile, err := os.CreateTemp("", "dbadd-publications-sample-*.pass")
		if err != nil {
			panic(fmt.Errorf("failed to create temp password file %w", err))
		}
		defer os.Remove(passFile.Name())
		_, err = passFile.WriteString("postgres")
		if err != nil {
			panic(fmt.Errorf("failed to write password content %w", err))
		}
		conf = config.Root{
			Storage: &config.Storage{
				Type: "postgres",
				ConnectConfig: pgxutil.ConnectConfig{
					URI:          "postgres://postgres@localhost:5432/smart_core",
					PasswordFile: passFile.Name(),
				},
			},
		}
	} else {
		data, err := os.ReadFile(*configFilePath)
		if err != nil {
			panic(fmt.Errorf("unable to read config file %w", err))
		}
		err = json.Unmarshal(data, &conf)
		if err != nil {
			panic(fmt.Errorf("unable to parse config file %w", err))
		}
	}

	// common utils we use
	ctx := context.Background()
	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(fmt.Errorf("setting up logger: %w", err))
	}

	// open connection
	pool, err := pgxutil.Connect(ctx, conf.Storage.ConnectConfig)
	if err != nil {
		panic(fmt.Errorf("connecting to db %w", err))
	}
	defer pool.Close()

	// create tables, etc
	err = pgxpublications.SetupDB(ctx, pool)
	if err != nil {
		panic(fmt.Errorf("setting up db: %w", err))
	}

	// seed with some data
	err = populateDB(ctx, logger, pool)
	if err != nil {
		panic(fmt.Errorf("populating db: %w", err))
	}
}

func populateDB(ctx context.Context, logger *zap.Logger, conn *pgxpool.Pool) error {
	deviceNames := []string{
		"test/area-controller-1",
		"test/area-controller-2",
		"test/area-controller-3",
	}

	baseTime := time.Date(2022, 7, 6, 11, 18, 0, 0, time.UTC)

	err := conn.BeginFunc(ctx, func(tx pgx.Tx) error {
		var errs error
		for _, name := range deviceNames {
			// register a publication
			id := name + ":config"
			errs = multierr.Append(errs, pgxpublications.CreatePublication(ctx, tx, id, name))

			// add some versions to it
			for i := 1; i <= 3; i++ {
				payload := struct {
					Device      string `json:"device"`
					Publication string `json:"publication"`
					Sequence    int    `json:"sequence"`
				}{
					Device:      name,
					Publication: id,
					Sequence:    i,
				}

				encoded, err := json.Marshal(payload)
				if err != nil {
					errs = multierr.Append(errs, err)
					continue
				}

				_, err = pgxpublications.CreatePublicationVersion(ctx, tx, pgxpublications.PublicationVersion{
					PublicationID: id,
					PublishTime:   baseTime.Add(time.Duration(i) * time.Hour),
					Body:          encoded,
					MediaType:     "application/json",
					Changelog:     fmt.Sprintf("auto-populated revision %d", i),
				})
				errs = multierr.Append(errs, err)
			}
		}

		return errs
	})

	if err != nil {
		logger.Error("failed to populate database", zap.Error(err))
	} else {
		logger.Info("database populated")
	}
	return err
}
