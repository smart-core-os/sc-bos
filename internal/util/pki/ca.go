package pki

import (
	"crypto"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"golang.org/x/exp/slices"

	"github.com/smart-core-os/sc-bos/pkg/util/netutil"
)

// CreateCertificateChain creates a new x509.Certificate using id and pub signed by the leaf and private key of authority.
// The returned bytes contains this new certificate first followed by all intermediates found in authority in PEM encoded format.
// The authority parameter should represent a CA certificate followed any intermediate certificates.
func CreateCertificateChain(authority *tls.Certificate, id *x509.Certificate, pub crypto.PublicKey, opts ...CSROption) (pem []byte, err error) {
	csr := newCSR(opts...)
	leafDer, err := createCertificate(csr, authority, id, pub)
	if err != nil {
		return nil, err
	}
	certs := make([][]byte, 0, len(authority.Certificate)+1)
	certs = append(certs, leafDer)
	certs = append(certs, authority.Certificate...)
	return EncodePEMSequence(certs, "CERTIFICATE"), nil
}

// CreateSelfSignedCertificate creates a new certificate whose issuer is the certificate itself.
func CreateSelfSignedCertificate(template *x509.Certificate, key PrivateKey, opts ...CSROption) (pem []byte, err error) {
	csr := newCSR(opts...)
	cert, err := hydrateTemplate(template, csr)
	if err != nil {
		return nil, err
	}
	return x509.CreateCertificate(csr.rand, cert, cert, key.Public(), key)
}

func createCertificate(csr csr, authority *tls.Certificate, id *x509.Certificate, pub crypto.PublicKey) (der []byte, err error) {
	cert, err := hydrateTemplate(id, csr)
	if err != nil {
		return nil, err
	}
	authorityCert := authority.Leaf
	if authorityCert == nil {
		if len(authority.Certificate) == 0 {
			return nil, errors.New("ca: authority has no leaf or certificate")
		}
		authorityCert, err = x509.ParseCertificate(authority.Certificate[0])
		if err != nil {
			return nil, err
		}
	}
	return x509.CreateCertificate(csr.rand, cert, authorityCert, pub, authority.PrivateKey)
}

// CSROption allows customisation of the certificate creation process.
type CSROption interface {
	apply(csr *csr)
}

// WithExpireAfter sets the created certificates NotAfter to now + expireAfter.
func WithExpireAfter(expireAfter time.Duration) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.expireAfter = expireAfter
	})
}

// WithValidSince sets the created certificates NotBefore to now - validSince.
func WithValidSince(validSince time.Duration) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.validSince = validSince
	})
}

// WithAuthority adds the given host or IP to the created certificate.
// authority can have a port which will be stripped before use
func WithAuthority(authority string) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.authority = netutil.StripPort(authority)
	})
}

// WithSAN adds the given host or IP to the certificates Subject Alternative Names list.
func WithSAN(hostOrIP string) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.sans = append(csr.sans, hostOrIP)
	})
}

// WithNonLoopbackIfaces adds non-loopback net.Interfaces to the created certificate.
func WithNonLoopbackIfaces() CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.discoverInterfaces = func(p net.Interface) bool {
			return p.Flags&net.FlagLoopback == 0
		}
	})
}

// WithIfaces adds all net.Interfaces to the created certificate, including loopback and the DNS "localhost" interface.
func WithIfaces() CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.discoverInterfaces = func(p net.Interface) bool {
			return true
		}
		csr.localhost = true
	})
}

// WithNow sets the source of time used when calculating NotBefore and NotAfter.
func WithNow(now func() time.Time) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.now = now
	})
}

// WithRand set the source of random data used to create certificate signatures.
func WithRand(rand io.Reader) CSROption {
	return csrOptionFunc(func(csr *csr) {
		csr.rand = rand
	})
}

type csrOptionFunc func(csr *csr)

func (c csrOptionFunc) apply(csr *csr) {
	c(csr)
}

type csr struct {
	discoverInterfaces func(p net.Interface) bool
	localhost          bool
	sans               []string

	authority string

	now         func() time.Time
	expireAfter time.Duration
	validSince  time.Duration
	rand        io.Reader

	cacheFile string

	keyUsage    x509.KeyUsage
	extKeyUsage []x509.ExtKeyUsage
}

var defaultCSROptions = []CSROption{
	WithNow(time.Now),
	WithRand(rand.Reader),
	WithExpireAfter(30 * 24 * time.Hour),
}

func newCSR(opts ...CSROption) csr {
	csr := &csr{}
	for _, opt := range defaultCSROptions {
		opt.apply(csr)
	}
	for _, opt := range opts {
		opt.apply(csr)
	}
	return *csr
}

func hydrateTemplate(template *x509.Certificate, csr csr) (*x509.Certificate, error) {
	// clone to avoid editing the original cert
	cert := cloneCert(template)

	if cert.SerialNumber == nil {
		serial, err := GenerateSerialNumber()
		if err != nil {
			return nil, fmt.Errorf("generate serial %w", err)
		}
		cert.SerialNumber = serial
	}

	now := cert.NotBefore
	if now.IsZero() {
		now = csr.now()
		cert.NotBefore = now
	}
	if cert.NotAfter.IsZero() {
		cert.NotAfter = now.Add(csr.expireAfter)
	}
	cert.NotBefore = cert.NotBefore.Add(-csr.validSince)

	if cert.KeyUsage == 0 {
		cert.KeyUsage = csr.keyUsage
	}
	if len(cert.ExtKeyUsage) == 0 {
		cert.ExtKeyUsage = csr.extKeyUsage
	}

	var (
		dnsNames    []string
		ipAddresses []net.IP
	)

	addHostOrIP := func(s string) {
		if s == "" {
			return
		}
		if ip := net.ParseIP(s); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			dnsNames = append(dnsNames, s)
		}
	}

	addHostOrIP(csr.authority)
	for _, san := range csr.sans {
		addHostOrIP(san)
	}

	if csr.discoverInterfaces != nil {
		ips, err := discoverLocalIPs(csr.discoverInterfaces)
		if err != nil {
			return nil, fmt.Errorf("discover interfaces %w", err)
		}
		ipAddresses = append(ipAddresses, ips...)
	}
	if csr.localhost {
		dnsNames = append(dnsNames, "localhost")
	}

	if len(dnsNames) > 0 {
		// remove dupes
		cert.DNSNames = append(dnsNames, cert.DNSNames...)
		slices.Sort(cert.DNSNames)
		cert.DNSNames = slices.Compact(cert.DNSNames)
	}
	if len(ipAddresses) > 0 {
		cert.IPAddresses = append(ipAddresses, cert.IPAddresses...)
		slices.SortFunc(cert.IPAddresses, func(a, b net.IP) int {
			return slices.Compare(a, b)
		})
		cert.IPAddresses = slices.CompactFunc(cert.IPAddresses, func(a, b net.IP) bool {
			return a.Equal(b)
		})
	}

	return cert, nil
}

func discoverLocalIPs(fn func(p net.Interface) bool) ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var ips []net.IP
	for _, iface := range ifaces {
		if !fn(iface) {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			if addr, ok := addr.(*net.IPNet); ok {
				ips = append(ips, addr.IP)
			}
		}
	}
	return ips, nil
}
