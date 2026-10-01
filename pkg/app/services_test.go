package app

import (
	"context"
	"slices"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/smart-core-os/sc-bos/internal/cloud"
	"github.com/smart-core-os/sc-bos/pkg/connect"
	"github.com/smart-core-os/sc-bos/pkg/minibus"
)

const (
	testNodeID      = "test-node" // the CN testRegistration sets on the leaf
	testAPIEndpoint = "https://connect.example.com"
)

// fakeConn stands in for *cloud.Conn, broadcasting each set state the same way.
type fakeConn struct {
	mu    sync.Mutex
	state cloud.ConnState
	bus   minibus.Bus[cloud.ConnState]
}

func (f *fakeConn) State() cloud.ConnState {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

func (f *fakeConn) PullState(ctx context.Context) (cloud.ConnState, <-chan cloud.ConnState) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state, f.bus.Listen(ctx)
}

func (f *fakeConn) set(st cloud.ConnState) {
	f.mu.Lock()
	f.state = st
	f.mu.Unlock()
	f.bus.Send(context.Background(), st)
}

func TestCloudCredential(t *testing.T) {
	enrolled := testRegistration(t)
	enrolled.APIEndpoint = testAPIEndpoint

	tests := []struct {
		name        string
		state       cloud.ConnState
		wantCertErr bool
		want        connect.State
	}{
		{
			// Registration is nil iff Connectivity is Unconfigured, so this is the
			// state of a node that has a cloud connection but has not yet enrolled.
			name:        "unenrolled",
			state:       cloud.ConnState{Connectivity: cloud.Unconfigured},
			wantCertErr: true,
			want:        connect.State{Connectivity: connect.NotEnrolled},
		},
		{
			name:  "enrolled",
			state: cloud.ConnState{Connectivity: cloud.Connected, Registration: enrolled},
			want:  connect.State{Connectivity: connect.Connected, NodeID: testNodeID, APIEndpoint: testAPIEndpoint},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := cloudCredential{conn: &fakeConn{state: tt.state}}

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

			if got := cred.State(); got != tt.want {
				t.Errorf("State() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// TestCloudCredential_readsStatePerCall guards the property the design depends on:
// a credential handed to a driver at start-up must pick up a later enrolment - and
// any subsequent certificate renewal - without being re-fetched.
func TestCloudCredential_readsStatePerCall(t *testing.T) {
	conn := &fakeConn{state: cloud.ConnState{Connectivity: cloud.Unconfigured}}
	cred := cloudCredential{conn: conn}

	if _, err := cred.GetClientCertificate(nil); err == nil {
		t.Error("GetClientCertificate before enrolment: want error, got nil")
	}
	if got := cred.State().NodeID; got != "" {
		t.Errorf("NodeID before enrolment = %q, want empty", got)
	}

	reg := testRegistration(t)
	reg.APIEndpoint = testAPIEndpoint
	conn.set(cloud.ConnState{Connectivity: cloud.Connected, Registration: reg})

	if _, err := cred.GetClientCertificate(nil); err != nil {
		t.Errorf("GetClientCertificate after enrolment: %v", err)
	}
	want := connect.State{Connectivity: connect.Connected, NodeID: testNodeID, APIEndpoint: testAPIEndpoint}
	if got := cred.State(); got != want {
		t.Errorf("State() after enrolment = %+v, want %+v", got, want)
	}
}

// TestCloudCredential_PullState checks that changes reach a prompt reader in
// order, and that the check-in re-broadcasts and renewals which leave
// connect.State unchanged do not reach it at all.
func TestCloudCredential_PullState(t *testing.T) {
	reg := testRegistration(t)
	reg.APIEndpoint = testAPIEndpoint
	renewed := testRegistration(t) // fresh keypair, same node id and endpoint
	renewed.APIEndpoint = testAPIEndpoint

	synctest.Test(t, func(t *testing.T) {
		conn := &fakeConn{state: cloud.ConnState{Connectivity: cloud.Unconfigured}}
		cred := cloudCredential{conn: conn}

		ctx, cancel := context.WithCancel(t.Context())
		initial, changes := cred.PullState(ctx)
		if want := (connect.State{Connectivity: connect.NotEnrolled}); initial != want {
			t.Errorf("initial = %+v, want %+v", initial, want)
		}

		var got []connect.State
		done := make(chan struct{})
		go func() {
			defer close(done)
			for st := range changes {
				got = append(got, st)
			}
		}()

		for _, st := range []cloud.ConnState{
			{Connectivity: cloud.Connecting, Registration: reg},
			{Connectivity: cloud.Connecting, Registration: reg},
			{Connectivity: cloud.Connected, Registration: reg},
			{Connectivity: cloud.Connected, Registration: renewed},
			{Connectivity: cloud.Unconfigured},
		} {
			conn.set(st)
			synctest.Wait() // let each change reach the reader, so none are coalesced
		}
		cancel()
		<-done

		enrolled := connect.State{NodeID: testNodeID, APIEndpoint: testAPIEndpoint}
		connecting, connected := enrolled, enrolled
		connecting.Connectivity = connect.Connecting
		connected.Connectivity = connect.Connected
		want := []connect.State{connecting, connected, {Connectivity: connect.NotEnrolled}}
		if !slices.Equal(got, want) {
			t.Errorf("changes = %+v, want %+v", got, want)
		}
	})
}

// TestCloudCredential_PullStateSlowReader checks a reader that falls behind
// doesn't hold up the cloud connection's broadcasts - each set returns
// without a read - and receives only the latest state once it catches up.
func TestCloudCredential_PullStateSlowReader(t *testing.T) {
	reg := testRegistration(t)
	reg.APIEndpoint = testAPIEndpoint

	synctest.Test(t, func(t *testing.T) {
		conn := &fakeConn{state: cloud.ConnState{Connectivity: cloud.Unconfigured}}
		cred := cloudCredential{conn: conn}

		ctx, cancel := context.WithCancel(t.Context())
		_, changes := cred.PullState(ctx)
		conn.set(cloud.ConnState{Connectivity: cloud.Connecting, Registration: reg})
		conn.set(cloud.ConnState{Connectivity: cloud.Unconfigured})
		conn.set(cloud.ConnState{Connectivity: cloud.Connected, Registration: reg})
		synctest.Wait()

		want := connect.State{Connectivity: connect.Connected, NodeID: testNodeID, APIEndpoint: testAPIEndpoint}
		if got := <-changes; got != want {
			t.Errorf("first value read = %+v, want %+v", got, want)
		}

		cancel()
		synctest.Wait()
		if st, ok := <-changes; ok {
			t.Errorf("changes delivered %+v after ctx was cancelled, want closed", st)
		}
	})
}

// TestCloudCredential_PullStateNetNoChange checks that changes which cancel
// out before the reader catches up deliver nothing: the reader already has
// the state they end at.
func TestCloudCredential_PullStateNetNoChange(t *testing.T) {
	reg := testRegistration(t)

	synctest.Test(t, func(t *testing.T) {
		conn := &fakeConn{state: cloud.ConnState{Connectivity: cloud.Unconfigured}}
		cred := cloudCredential{conn: conn}

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		_, changes := cred.PullState(ctx)
		conn.set(cloud.ConnState{Connectivity: cloud.Connecting, Registration: reg})
		conn.set(cloud.ConnState{Connectivity: cloud.Unconfigured})
		synctest.Wait()

		select {
		case st := <-changes:
			t.Errorf("changes delivered %+v, want nothing", st)
		default:
		}
	})
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

// TestCloudCredentialSource_noCloudConfig checks a node with no cloud block in
// its config still gets a credential: it can be enrolled from the Ops UI at any
// time, and drivers started before then must pick that up.
func TestCloudCredentialSource_noCloudConfig(t *testing.T) {
	c := &Controller{Cloud: &cloud.Conn{}}
	c.SystemConfig.Cloud = nil

	if cred := c.cloudCredentialSource(); cred == nil {
		t.Error("cloudCredentialSource() = nil, want a credential")
	}
}
