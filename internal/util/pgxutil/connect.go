package pgxutil

import (
	"context"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConnectConfig struct {
	URI          string `json:"uri,omitempty"`
	PasswordFile string `json:"passwordFile,omitempty"`
}

func Connect(ctx context.Context, sysConf ConnectConfig) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(sysConf.URI)
	if err != nil {
		return nil, err
	}

	if sysConf.PasswordFile != "" {
		passwordFile, err := os.ReadFile(sysConf.PasswordFile)
		if err != nil {
			return nil, err
		}

		poolConfig.ConnConfig.Password = strings.TrimSpace(string(passwordFile))
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func (cc ConnectConfig) IsZero() bool {
	return cc.URI == ""
}
