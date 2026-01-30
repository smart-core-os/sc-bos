package config

import (
	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	system.Config
	Storage *Storage `json:"storage,omitempty"`
}

type StorageType string

const (
	StorageTypePostgres StorageType = "postgres"
	StorageTypeBolt     StorageType = "bolt"
	StorageTypeSqlite   StorageType = "sqlite"
)

type Storage struct {
	Type StorageType `json:"type,omitempty"`
	pgxutil.ConnectConfig
	// TTL is the time-to-live for records. Zero-value (not-specified) means "forever".
	TTL *TTL `json:"ttl,omitempty"`
}

type TTL struct {
	MaxAge   jsontypes.Duration `json:"maxAge,omitempty"`
	MaxCount int64              `json:"maxCount,omitempty"`
}
