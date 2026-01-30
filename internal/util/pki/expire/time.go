package expire

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

type Now func() time.Time

// After is like AfterT using time.Now.
func After(d time.Duration) pki.Expiry {
	return AfterT(d, time.Now)
}

// AfterT returns a pki.Expiry that returns true the first time it is called after d multiples of time have passed.
func AfterT(d time.Duration, now Now) pki.Expiry {
	expires := now().Add(d)
	return func(_ *tls.Certificate, _ []*x509.Certificate, _ error) bool {
		n := now()
		if n.Before(expires) {
			return false
		}

		expires = n.Add(d)
		return true
	}
}

// BeforeInvalid is like BeforeInvalidT using time.Now.
func BeforeInvalid(d time.Duration) pki.Expiry {
	return BeforeInvalidT(d, time.Now)
}

// BeforeInvalidT returns a pki.Expiry that returns true if now >= cert.NotAfter - d.
func BeforeInvalidT(d time.Duration, now Now) pki.Expiry {
	return func(cert *tls.Certificate, pool []*x509.Certificate, err error) bool {
		if cert == nil {
			return false
		}

		c, err := pki.TLSLeaf(cert)
		if err != nil {
			return false
		}

		return !now().Before(c.NotAfter.Add(-d))
	}
}

// AfterValid is like AfterValidT using time.Now.
func AfterValid(d time.Duration) pki.Expiry {
	return AfterValidT(d, time.Now)
}

// AfterValidT returns a pki.Expiry that returns true if now >= cert.NotBefore + d.
func AfterValidT(d time.Duration, now Now) pki.Expiry {
	return func(cert *tls.Certificate, pool []*x509.Certificate, err error) bool {
		if cert == nil {
			return false
		}
		c, err := pki.TLSLeaf(cert)
		if err != nil {
			return false
		}

		return !now().Before(c.NotBefore.Add(d))
	}
}

// AfterProgress is like AfterProgressT using time.Now.
func AfterProgress(progress float32) pki.Expiry {
	return AfterProgressT(progress, time.Now)
}

// AfterProgressT returns a pki.Expiry that returns true if age >= cert.NotAfter-cert.NotBefore * progress.
// For example if progress were 0.5 then true is returned when now is half way between NotAfter and NotBefore.
func AfterProgressT(progress float32, now Now) pki.Expiry {
	return func(cert *tls.Certificate, pool []*x509.Certificate, err error) bool {
		if cert == nil {
			return false
		}

		c, err := pki.TLSLeaf(cert)
		if err != nil {
			return false
		}

		age := now().Sub(c.NotBefore)
		return age >= time.Duration(float32(c.NotAfter.Sub(c.NotBefore))*progress)
	}
}
