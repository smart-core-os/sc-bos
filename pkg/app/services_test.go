package app

import (
	"testing"

	"github.com/smart-core-os/sc-bos/internal/cloud"
)

const (
	testNodeID      = "test-node" // the CN testRegistration sets on the leaf
	testAPIEndpoint = "https://connect.example.com"
)

func TestCloudCredential(t *testing.T) {
	enrolled := testRegistration(t)
	enrolled.APIEndpoint = testAPIEndpoint

	tests := []struct {
		name        string
		state       cloud.ConnState
		wantCertErr bool
		wantNodeID  string
		wantAPI     string
	}{
		{
			// Registration is nil iff Connectivity is Unconfigured, so this is the
			// state of a node that has a cloud connection but has not yet enrolled.
			name:        "unenrolled",
			state:       cloud.ConnState{Connectivity: cloud.Unconfigured},
			wantCertErr: true,
		},
		{
			name:       "enrolled",
			state:      cloud.ConnState{Connectivity: cloud.Connected, Registration: enrolled},
			wantNodeID: testNodeID,
			wantAPI:    testAPIEndpoint,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := cloudCredential{state: func() cloud.ConnState { return tt.state }}

			cert, err := cred.GetClientCertificate(nil)
			switch {
			case tt.wantCertErr:
				if err == nil {
					t.Error("GetClientCertificate: want error, got nil")
				}
			case err != nil:
				t.Errorf("GetClientCertificate: %v", err)
			default:
				if cert.Leaf != tt.state.Registration.Leaf() {
					t.Error("GetClientCertificate: Leaf is not the registration's leaf")
				}
				if got, want := len(cert.Certificate), len(tt.state.Registration.Chain); got != want {
					t.Errorf("GetClientCertificate: presented %d chain entries, want %d", got, want)
				}
			}

			if got := cred.NodeID(); got != tt.wantNodeID {
				t.Errorf("NodeID() = %q, want %q", got, tt.wantNodeID)
			}
			if got := cred.APIEndpoint(); got != tt.wantAPI {
				t.Errorf("APIEndpoint() = %q, want %q", got, tt.wantAPI)
			}
		})
	}
}

// TestCloudCredential_readsStatePerCall guards the property the design depends on:
// a credential handed to a driver at start-up must pick up a later enrolment - and
// any subsequent certificate renewal - without being re-fetched.
func TestCloudCredential_readsStatePerCall(t *testing.T) {
	state := cloud.ConnState{Connectivity: cloud.Unconfigured}
	cred := cloudCredential{state: func() cloud.ConnState { return state }}

	if _, err := cred.GetClientCertificate(nil); err == nil {
		t.Error("GetClientCertificate before enrolment: want error, got nil")
	}
	if got := cred.NodeID(); got != "" {
		t.Errorf("NodeID() before enrolment = %q, want empty", got)
	}

	reg := testRegistration(t)
	reg.APIEndpoint = testAPIEndpoint
	state = cloud.ConnState{Connectivity: cloud.Connected, Registration: reg}

	if _, err := cred.GetClientCertificate(nil); err != nil {
		t.Errorf("GetClientCertificate after enrolment: %v", err)
	}
	if got := cred.NodeID(); got != testNodeID {
		t.Errorf("NodeID() after enrolment = %q, want %q", got, testNodeID)
	}
	if got := cred.APIEndpoint(); got != testAPIEndpoint {
		t.Errorf("APIEndpoint() after enrolment = %q, want %q", got, testAPIEndpoint)
	}
}

// TestCloudCredentialSource_noCloudConnection checks the accessor hands back an
// untyped nil. A typed nil would be non-nil as an interface and so defeat the
// cred == nil checks consumers rely on, e.g. connecttelemetry's buildTLSConfig.
func TestCloudCredentialSource_noCloudConnection(t *testing.T) {
	c := &Controller{}
	if cred := c.cloudCredentialSource(); cred != nil {
		t.Errorf("cloudCredentialSource() = %#v, want a nil interface", cred)
	}
}
