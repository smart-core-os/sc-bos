package config

import (
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/smart-core-os/sc-bos/internal/util/pgxutil"
	"github.com/smart-core-os/sc-bos/pkg/system"
	"github.com/smart-core-os/sc-bos/pkg/util/jsontypes"
)

type Root struct {
	system.Config

	Name    string `json:"name,omitempty"`    // The smart core name of the hub. Defaults to the node name.
	Address string `json:"address,omitempty"` // The network address nodes can reach this hub on. By default attempts to find a non-local net.Iface.

	// Validity controls the expiry of signatures associated with node certificates during enrollment.
	Validity *jsontypes.Duration `json:"validity,omitempty"`

	// Key, Cert, and Roots all represent PEM encoded certs or keys.
	// The values can be either the PEM itself: Key: "-----BEGIN PRIVATE KEY----- ... etc"
	// or a path relative to the data dir for a pem file on the filesystem.
	// IsPEM is used to decide which.

	Key   string `json:"key,omitempty"`   // Key used to sign node certificates as part of enrollment. Defaults to hub.key.pem if present else generates one.
	Cert  string `json:"cert,omitempty"`  // Cert used as the signer for enrolled node certificates. Defaults to hub.cert.pem if present else generates a self signed one.
	Roots string `json:"roots,omitempty"` // Roots pem or path to roots.pem shared as the trust root with enrolled nodes. Defaults to hub.roots.pem if present else cert is used.

	Storage *Storage `json:"storage,omitempty"`
}

type StorageType string

const (
	StorageTypePostgres StorageType = "postgres"
	StorageTypeProxy    StorageType = "proxy"
	StorageTypeBolt     StorageType = "bolt"
)

type Storage struct {
	Type StorageType `json:"type,omitempty"`
	pgxutil.ConnectConfig
}

var (
	ErrEmpty = errors.New("empty value")
)

func IsPEM(val string) bool {
	// A more correct way to do this is to use pem.Decode to see if any blocks exist,
	// however that requires converting val to []byte which is a copy
	return strings.Contains(val, "-----BEGIN ")
}

func ReadPEMOrFile(dir, val string) ([]byte, error) {
	if val == "" {
		return nil, ErrEmpty
	}

	valBytes := []byte(val)
	block, _ := pem.Decode(valBytes)
	if block != nil { // any block, we assume it's a pem
		return valBytes, nil
	}

	return os.ReadFile(filepath.Join(dir, val))
}
