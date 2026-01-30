package config

import (
	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/system"
)

type Root struct {
	system.Config
	Storage *Storage `json:"storage,omitempty"`
}

type StorageType string

const (
	StorageTypePostgres StorageType = "postgres"
)

type Storage struct {
	Type StorageType `json:"type,omitempty"`
	pgxutil.ConnectConfig
}
