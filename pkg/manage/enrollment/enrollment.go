package enrollment

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/multierr"

	"github.com/smart-core-os/sc-bos/internal/util/pki"
)

type Enrollment struct {
	RootDeviceName string `json:"root_device_name"`
	ManagerName    string `json:"manager_name"`
	ManagerAddress string `json:"manager_address"`
	LocalAddress   string `json:"local_address"`

	RootCA *x509.Certificate `json:"-"`
	Cert   tls.Certificate   `json:"-"`
}

func (e Enrollment) Equal(other Enrollment) bool {
	if e.RootDeviceName != other.RootDeviceName ||
		e.ManagerName != other.ManagerName ||
		e.ManagerAddress != other.ManagerAddress ||
		e.LocalAddress != other.LocalAddress {
		return false
	}
	return bytes.Equal(e.Cert.Certificate[0], other.Cert.Certificate[0])
}

func (e Enrollment) IsZero() bool {
	return e.RootDeviceName == "" &&
		e.ManagerName == "" &&
		e.ManagerAddress == "" &&
		e.LocalAddress == "" &&
		e.RootCA == nil &&
		len(e.Cert.Certificate) == 0
}

var ErrNotEnrolled = errors.New("node is not enrolled")

const (
	enrollmentFile = "enrollment.json"
	rootCaCertFile = "root-ca.cert.pem"
	certFile       = "enrollment.cert.pem"
)

// LoadEnrollment will load a previously saved Enrollment from a directory on disk. The directory should have
// the following structure:
//
//	<root>
//	  - enrollment.json - JSON-encoded Enrollment structure
//	  - root-ca.crt - Root CA for the enrollment, PEM-encoded X.509 certificate
//	  - cert.crt - Certificate chain for keyPEM, with the Root CA at the top of the chain
//
// This node's private key must be passed in, in PEM-wrapped PKCS#1 or PKCS#8 format.
func LoadEnrollment(dir string, keyPEM []byte) (Enrollment, error) {
	// read and unmarshal enrollment metadata
	var enrollment Enrollment
	raw, err := os.ReadFile(filepath.Join(dir, enrollmentFile))
	if errors.Is(err, os.ErrNotExist) {
		return Enrollment{}, ErrNotEnrolled
	} else if err != nil {
		return Enrollment{}, err
	}
	err = json.Unmarshal(raw, &enrollment)
	if err != nil {
		return Enrollment{}, err
	}

	// load root CA certificate
	rootCaPem, err := os.ReadFile(filepath.Join(dir, rootCaCertFile))
	if err != nil {
		return enrollment, err
	}
	rootCaBlock, _ := pem.Decode(rootCaPem)
	if rootCaBlock == nil {
		return enrollment, errors.New("CA.pem is not valid PEM data")
	}
	if rootCaBlock.Type != "CERTIFICATE" {
		return enrollment, fmt.Errorf("expected PEM type 'CERTIFICATE' but got %q", rootCaBlock.Type)
	}
	rootCa, err := x509.ParseCertificate(rootCaBlock.Bytes)
	if err != nil {
		return enrollment, err
	}
	enrollment.RootCA = rootCa

	// load this node's certificate
	certPem, err := os.ReadFile(filepath.Join(dir, certFile))
	if err != nil {
		return enrollment, err
	}
	cert, err := tls.X509KeyPair(certPem, keyPEM)
	if err != nil {
		return enrollment, err
	}
	enrollment.Cert = cert

	return enrollment, nil
}

func SaveEnrollment(dir string, enrollment Enrollment) (err error) {
	enrollmentFile := filepath.Join(dir, enrollmentFile)
	rootCaCertFile := filepath.Join(dir, rootCaCertFile)
	certFile := filepath.Join(dir, certFile)

	// check if files already exist
	_, err = os.Stat(enrollmentFile)
	if errors.Is(err, os.ErrNotExist) {
		err = os.MkdirAll(dir, 0750)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// The files already exist, lets treat this as an update.
		// We want to be atomic, so make a backup of the files so we can restore them afterwards if needed
		files := []string{enrollmentFile, rootCaCertFile, certFile}
		for _, f := range files {
			if err := os.Rename(f, f+".bak"); err != nil {
				return fmt.Errorf("failed to backup old files %w", err)
			}
		}
		defer func() {
			if err == nil {
				// remove backup files
				for _, file := range files {
					_ = os.Remove(file + ".bak")
				}
			} else {
				// restore the backup files, removing any existing files in the process
				for _, file := range files {
					if _, e := os.Stat(file + ".bak"); e == nil {
						if _, e := os.Stat(file); e == nil {
							os.Remove(file)
						}
						os.Rename(file+".bak", file)
					} else if errors.Is(e, os.ErrNotExist) {
						continue
					} else {
						err = multierr.Combine(err, e)
					}
				}
			}
		}()
	}

	jsonBytes, err := json.Marshal(enrollment)
	if err != nil {
		return err
	}
	err = os.WriteFile(enrollmentFile, jsonBytes, 0640)
	if err != nil {
		return err
	}
	_, err = pki.SaveCertificateChain(rootCaCertFile, [][]byte{enrollment.RootCA.Raw})
	if err != nil {
		return err
	}
	_, err = pki.SaveCertificateChain(certFile, enrollment.Cert.Certificate)
	return err
}

func DeleteEnrollment(dir string) error {
	return os.RemoveAll(dir)
}
