package sysconf

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"github.com/smart-core-os/sc-bos/internal/download"
)

// ResolveDownloadHMACKey returns the HMAC key used to sign download URLs,
// derived from c.Download.HMACKeyFile (falling back to the deprecated
// c.Devices.DownloadHMACKeyFile, with a warning). An explicitly configured
// key file that cannot be read or does not contain a valid hex-encoded
// 32-byte key is fatal — the operator asked for stable keying, so silently
// rolling a random key would defeat their intent and invisibly weaken URL
// forgery resistance. With no configuration, a fresh 32-byte random key is
// generated via download.GenerateHMACKey.
func (c Config) ResolveDownloadHMACKey(logger *zap.Logger) ([]byte, error) {
	keyFile := ""
	switch {
	case c.Download != nil && c.Download.HMACKeyFile != "":
		keyFile = c.Download.HMACKeyFile
	case c.Devices != nil && c.Devices.DownloadHMACKeyFile != "":
		keyFile = c.Devices.DownloadHMACKeyFile
		logger.Warn("config.devices.downloadHMACKeyFile is deprecated; use config.download.hmacKeyFile instead")
	}
	if keyFile == "" {
		return download.GenerateHMACKey(), nil
	}
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read download HMAC key file %q: %w", keyFile, err)
	}
	key, err := download.LoadHMACKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse download HMAC key file %q: %w", keyFile, err)
	}
	return key, nil
}
