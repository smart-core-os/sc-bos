package dps

import (
	"crypto/hmac"
	"crypto/sha256"

	"github.com/smart-core-os/sc-bos/internal/iothub/auth"
)

// DeriveDeviceKey will derive an SAS Key for a device to use with Group Enrollment.
// groupKey is the Group Enrollment Key.
// regID is the Registration ID that the device will be registered under.
func DeriveDeviceKey(groupKey auth.SASKey, regID string) auth.SASKey {
	if len(groupKey) == 0 {
		panic("groupKey is required")
	}

	h := hmac.New(sha256.New, groupKey)
	_, err := h.Write([]byte(regID))
	if err != nil {
		// There's no condition in which this should fail
		panic("HMAC failed")
	}

	return h.Sum(nil)
}
