package expire

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

// Manually returns a pki.Expiry that returns true the first time it is called after expire is invoked.
func Manually() (expiry pki.Expiry, expire func()) {
	var expired bool
	expiry = func(_ *tls.Certificate, _ []*x509.Certificate, _ error) bool {
		if expired {
			expired = false
			return true
		}
		return false
	}
	expire = func() {
		expired = true
	}
	return expiry, expire
}
